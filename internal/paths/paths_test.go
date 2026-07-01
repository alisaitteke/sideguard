package paths

import (
	"path/filepath"
	"testing"
)

func TestHome(t *testing.T) {
	home, err := Home()
	if err != nil {
		t.Fatalf("Home() error: %v", err)
	}
	if filepath.Base(home) != DirName {
		t.Fatalf("expected base %q, got %q", DirName, filepath.Base(home))
	}
}

func TestSocketPath(t *testing.T) {
	sock, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath() error: %v", err)
	}
	if filepath.Base(sock) != SocketFile {
		t.Fatalf("expected socket file %q, got %q", SocketFile, filepath.Base(sock))
	}
}

func TestAuditDBPath(t *testing.T) {
	db, err := AuditDBPath()
	if err != nil {
		t.Fatalf("AuditDBPath() error: %v", err)
	}
	if filepath.Base(db) != AuditDBFile {
		t.Fatalf("expected db file %q, got %q", AuditDBFile, filepath.Base(db))
	}
}
