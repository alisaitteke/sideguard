package update

import (
	"os"
	"testing"
)

func TestParseChecksumsAndVerify(t *testing.T) {
	content := []byte("hello release")
	archiveName := "sideguard_1.0.0_darwin_arm64.tar.gz"
	line := sha256Hex(content) + "  " + archiveName + "\n"

	checksums, err := ParseChecksums([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	if checksums[archiveName] == "" {
		t.Fatal("missing checksum entry")
	}

	dir := t.TempDir()
	path := dir + "/" + archiveName
	if err := osWrite(path, content); err != nil {
		t.Fatal(err)
	}

	if err := VerifyArchive(path, []byte(line)); err != nil {
		t.Fatalf("verify: %v", err)
	}

	bad := sha256Hex([]byte("tampered")) + "  " + archiveName + "\n"
	if err := VerifyArchive(path, []byte(bad)); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func osWrite(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
