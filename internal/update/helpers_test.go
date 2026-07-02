package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{" v0.1.0 ", "0.1.0"},
	}
	for _, tc := range tests {
		if got := NormalizeVersion(tc.in); got != tc.want {
			t.Fatalf("NormalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCompare(t *testing.T) {
	if Compare("1.0.0", "1.0.0") != 0 {
		t.Fatal("same version should compare equal")
	}
	if Compare("1.0.0", "1.1.0") <= 0 {
		t.Fatal("expected newer latest")
	}
	if Compare("2.0.0", "1.9.9") >= 0 {
		t.Fatal("expected older latest")
	}
}

func TestResolveAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, version, want string
	}{
		{"darwin", "arm64", "1.2.3", "vibeguard_1.2.3_darwin_arm64.tar.gz"},
		{"linux", "amd64", "0.1.0", "vibeguard_0.1.0_linux_amd64.tar.gz"},
		{"windows", "amd64", "2.0.0", "vibeguard_2.0.0_windows_amd64.zip"},
	}
	for _, tc := range tests {
		got := ResolveAssetName(tc.goos, tc.goarch, tc.version)
		if got != tc.want {
			t.Fatalf("ResolveAssetName(%q,%q,%q) = %q, want %q", tc.goos, tc.goarch, tc.version, got, tc.want)
		}
	}
}

func writeTarGz(t *testing.T, path, binaryName string, content []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name: binaryName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeChecksums(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	var buf bytes.Buffer
	for name, data := range files {
		buf.WriteString(sha256Hex(data))
		buf.WriteString("  ")
		buf.WriteString(name)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func copyReaderToFile(t *testing.T, r io.Reader, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		t.Fatal(err)
	}
}
