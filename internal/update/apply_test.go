package update

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type recordingPlatform struct {
	stopped, started bool
}

func (p *recordingPlatform) Stop(context.Context) error {
	p.stopped = true
	return nil
}

func (p *recordingPlatform) SwapBinary(_ context.Context, stagingPath, targetPath string) error {
	return atomicSwapBinary(stagingPath, targetPath)
}

func (p *recordingPlatform) Start(context.Context) error {
	p.started = true
	return nil
}

func TestApplyTarGzHappyPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	binary := []byte("#!/bin/sh\necho updated\n")
	assetName := ResolveAssetName("darwin", "arm64", "9.9.9")
	archiveBytes := buildTarGzBytes(t, "vibeguard", binary)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case assetName:
			_, _ = w.Write(archiveBytes)
		case "checksums.txt":
			_, _ = w.Write([]byte(sha256Hex(archiveBytes) + "  " + assetName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	work := t.TempDir()
	target := filepath.Join(work, "vibeguard")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	rel := ReleaseInfo{
		Version:     "9.9.9",
		AssetName:   assetName,
		DownloadURL: srv.URL + "/" + assetName,
	}

	platform := &recordingPlatform{}
	applier := NewApplier(srv.Client(), "darwin")
	err := applier.Apply(context.Background(), rel, ApplyOptions{
		TargetPath:   target,
		Platform:     platform,
		ChecksumsURL: srv.URL + "/checksums.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !platform.stopped || !platform.started {
		t.Fatal("expected platform stop/start")
	}
	if string(readFile(t, target)) != string(binary) {
		t.Fatalf("binary not replaced: %q", readFile(t, target))
	}
}

func TestApplyChecksumMismatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	binary := []byte("payload")
	assetName := ResolveAssetName("darwin", "arm64", "1.0.1")
	archiveBytes := buildTarGzBytes(t, "vibeguard", binary)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Base(r.URL.Path) == assetName {
			_, _ = w.Write(archiveBytes)
			return
		}
		_, _ = w.Write([]byte(sha256Hex([]byte("wrong")) + "  " + assetName + "\n"))
	}))
	defer srv.Close()

	work := t.TempDir()
	target := filepath.Join(work, "vibeguard")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	applier := NewApplier(srv.Client(), "darwin")
	err := applier.Apply(context.Background(), ReleaseInfo{
		Version:     "1.0.1",
		AssetName:   assetName,
		DownloadURL: srv.URL + "/" + assetName,
	}, ApplyOptions{
		TargetPath:   target,
		ChecksumsURL: srv.URL + "/checksums.txt",
	})
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "test.zip")
	payload := []byte("zip-binary")
	writeZip(t, archive, "vibeguard.exe", payload)

	out, err := extractArchive(archive, filepath.Join(dir, "out"), "vibeguard.exe")
	if err != nil {
		t.Fatal(err)
	}
	if string(readFile(t, out)) != string(payload) {
		t.Fatal("zip extract mismatch")
	}
}

func buildTarGzBytes(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a.tar.gz")
	writeTarGz(t, path, name, content)
	return readFile(t, path)
}

func writeZip(t *testing.T, path, name string, content []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.tar.gz")
	writeTarGz(t, archive, "foo/../../vibeguard", []byte("nope"))

	_, err := extractArchive(archive, filepath.Join(dir, "out"), "vibeguard")
	if err == nil {
		t.Fatal("expected path traversal rejection")
	}
}
