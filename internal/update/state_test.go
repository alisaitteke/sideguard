package update

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStateLoadSaveAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update-state.json")
	store := NewFileStateStoreAt(path)

	st, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !st.LastCheckAt.IsZero() {
		t.Fatal("expected zero state")
	}

	want := State{
		LastCheckAt:      time.Date(2026, 7, 2, 11, 2, 0, 0, time.UTC),
		LatestKnown:      "1.2.3",
		DismissedVersion: "",
		DownloadPath:     filepath.Join(dir, "run", "update", "1.2.3", "vibeguard"),
	}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.LatestKnown != want.LatestKnown || got.DownloadPath != want.DownloadPath {
		t.Fatalf("state mismatch: %+v vs %+v", got, want)
	}
}
