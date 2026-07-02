package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckerUpdateAvailable(t *testing.T) {
	const latest = "2.0.0"
	assetName := ResolveAssetName("darwin", "arm64", latest)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/alisaitteke/vibeguard/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name":     "v" + latest,
				"published_at": time.Now().UTC(),
				"assets": []map[string]string{
					{"name": assetName, "browser_download_url": "http://example.invalid/asset"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	statePath := filepath.Join(t.TempDir(), "update-state.json")
	checker, err := NewChecker("1.0.0", Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
		StatePath:  statePath,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !checker.ShouldAutoCheck() {
		t.Fatal("expected auto check enabled")
	}

	result, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.UpdateAvailable || result.Latest != latest {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Release == nil || result.Release.AssetName != assetName {
		t.Fatalf("missing release info: %+v", result.Release)
	}

	st, err := NewFileStateStoreAt(statePath).Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.LatestKnown != latest || st.LastCheckAt.IsZero() {
		t.Fatalf("state not updated: %+v", st)
	}
}

func TestCheckerNoUpdate(t *testing.T) {
	const version = "1.5.0"
	assetName := ResolveAssetName("darwin", "arm64", version)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v" + version,
			"published_at": time.Now().UTC(),
			"assets": []map[string]string{
				{"name": assetName, "browser_download_url": "http://example.invalid/asset"},
			},
		})
	}))
	defer srv.Close()

	checker, err := NewChecker("1.5.0", Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
		StatePath:  filepath.Join(t.TempDir(), "state.json"),
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := checker.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdateAvailable {
		t.Fatalf("expected no update: %+v", result)
	}
}

func TestCheckerShouldAutoCheckDev(t *testing.T) {
	checker, err := NewChecker("dev", Options{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if checker.ShouldAutoCheck() {
		t.Fatal("dev version should disable auto check")
	}
}

func TestCheckerAPIErrorStillTouchesState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	statePath := filepath.Join(t.TempDir(), "state.json")
	checker, err := NewChecker("1.0.0", Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
		StatePath:  statePath,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = checker.Check(context.Background())
	if err == nil {
		t.Fatal("expected API error")
	}

	st, err := NewFileStateStoreAt(statePath).Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.LastCheckAt.IsZero() {
		t.Fatal("expected last_check_at on API failure")
	}
}
