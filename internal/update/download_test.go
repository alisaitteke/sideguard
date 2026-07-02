package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloaderResumeAndComplete(t *testing.T) {
	payload := []byte("binary payload for download test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			http.Error(w, "no range", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dl := NewDownloader(srv.Client())

	path, err := dl.Download(context.Background(), srv.URL, dir, "asset.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if readFile(t, path) == nil {
		t.Fatal("empty download")
	}
	if string(readFile(t, path)) != string(payload) {
		t.Fatal("content mismatch")
	}

	// Second call should reuse existing file.
	path2, err := dl.Download(context.Background(), srv.URL, dir, "asset.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if path2 != path {
		t.Fatalf("paths differ: %q vs %q", path, path2)
	}
}

func TestDownloaderCreatesPartFile(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "file.tar.gz.part")
	if err := os.WriteFile(part, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := []byte("full-payload-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dl := NewDownloader(srv.Client())
	path, err := dl.Download(context.Background(), srv.URL, dir, "file.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if string(readFile(t, path)) != string(payload) {
		t.Fatal("unexpected payload")
	}
}
