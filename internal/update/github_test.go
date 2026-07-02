package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubLatestRelease(t *testing.T) {
	const version = "1.2.3"
	assetName := ResolveAssetName("darwin", "arm64", version)

	var assetURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/alisaitteke/sideguard/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v" + version,
			"published_at": time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC),
			"assets": []map[string]string{
				{"name": assetName, "browser_download_url": assetURL},
			},
		})
	}))
	defer srv.Close()
	assetURL = srv.URL + "/asset.tar.gz"

	client := NewGitHubClient(Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
	})

	rel, err := client.LatestRelease(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != version || rel.AssetName != assetName {
		t.Fatalf("unexpected release: %+v", rel)
	}
	if rel.DownloadURL == "" {
		t.Fatal("expected download URL")
	}
}

func TestGitHubUnsupportedPlatform(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v1.0.0",
			"published_at": time.Now().UTC(),
			"assets":       []any{},
		})
	}))
	defer srv.Close()

	client := NewGitHubClient(Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
	})

	_, err := client.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected unsupported platform error")
	}
}

func TestGitHubAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewGitHubClient(Options{
		APIBaseURL: srv.URL,
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: srv.Client(),
	})

	_, err := client.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
