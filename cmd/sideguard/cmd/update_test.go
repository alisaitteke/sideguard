package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/update"
)

func TestUpdateCheckJSONShape(t *testing.T) {
	const latest = "2.0.0"
	assetName := update.ResolveAssetName("darwin", "arm64", latest)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v" + latest,
			"published_at": time.Now().UTC(),
			"assets": []map[string]string{
				{"name": assetName, "browser_download_url": "https://example.invalid/asset.tar.gz"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	statePath := filepath.Join(t.TempDir(), "update-state.json")
	prevVersion := Version
	Version = "1.0.0"
	t.Cleanup(func() { Version = prevVersion })

	updateCheckerHook = func(current string, opts update.Options) (*update.Checker, error) {
		return update.NewChecker(current, update.Options{
			APIBaseURL: srv.URL,
			GOOS:       "darwin",
			GOARCH:     "arm64",
			HTTPClient: srv.Client(),
			StatePath:  statePath,
			Disabled:   opts.Disabled,
		})
	}
	t.Cleanup(func() { updateCheckerHook = nil })

	var buf bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prevStdout })

	updateCheckJSON = true
	t.Cleanup(func() { updateCheckJSON = false })

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	if err := runUpdateCheck(updateCheckCmd, nil); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	<-done

	var out updateCheckJSONOut
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if out.Current != "1.0.0" || out.Latest != latest || !out.UpdateAvailable {
		t.Fatalf("unexpected payload: %+v", out)
	}
	if out.AssetName != assetName {
		t.Fatalf("asset_name = %q, want %q", out.AssetName, assetName)
	}
	if out.DownloadURL == "" {
		t.Fatal("expected download_url")
	}
}

func TestUpdateCheckerDevAutoCheckDisabled(t *testing.T) {
	prevVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = prevVersion })

	updateCfg := config.UpdateConfig{Enabled: true, CheckInterval: "6h", Channel: "stable"}
	checker, err := newUpdateChecker(updateCfg)
	if err != nil {
		t.Fatal(err)
	}
	if checker.ShouldAutoCheck() {
		t.Fatal("expected auto check disabled for dev builds")
	}
}

func TestUpdateCheckerDisabledByConfig(t *testing.T) {
	prevVersion := Version
	Version = "1.0.0"
	t.Cleanup(func() { Version = prevVersion })

	updateCfg := config.UpdateConfig{Enabled: false}
	checker, err := newUpdateChecker(updateCfg)
	if err != nil {
		t.Fatal(err)
	}
	if checker.ShouldAutoCheck() {
		t.Fatal("expected auto check disabled when update.enabled is false")
	}
}

func TestConfirmUpdateApplyNonTTYRequiresYes(t *testing.T) {
	if err := confirmUpdateApply("2.0.0", false); err == nil {
		t.Fatal("expected error on non-TTY without --yes")
	}
}

func TestUpdateStatusJSON(t *testing.T) {
	prevVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = prevVersion })

	statePath := filepath.Join(t.TempDir(), "update-state.json")
	store := update.NewFileStateStoreAt(statePath)
	checkAt := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	if err := store.Save(update.State{
		LastCheckAt:  checkAt,
		LatestKnown:  "2.0.0",
		DownloadPath: "/tmp/sideguard",
	}); err != nil {
		t.Fatal(err)
	}

	updateStateStoreHook = func() (update.StateStore, error) {
		return store, nil
	}
	t.Cleanup(func() { updateStateStoreHook = nil })

	prevHook := updateCheckerHook
	updateCheckerHook = func(current string, opts update.Options) (*update.Checker, error) {
		return update.NewChecker(current, update.Options{Disabled: opts.Disabled, StatePath: statePath})
	}
	t.Cleanup(func() { updateCheckerHook = prevHook })

	var buf bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prevStdout })

	updateStatusJSON = true
	t.Cleanup(func() { updateStatusJSON = false })

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	if err := runUpdateStatus(updateStatusCmd, nil); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	<-done

	var out updateStatusJSONOut
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if out.Current != "1.2.3" || out.LatestKnown != "2.0.0" {
		t.Fatalf("unexpected status: %+v", out)
	}
	if out.DownloadPath != "/tmp/sideguard" {
		t.Fatalf("download_path = %q", out.DownloadPath)
	}
}

func TestReleaseForVersionMatchesPlatform(t *testing.T) {
	rel := update.ReleaseForVersion("2.1.0", runtime.GOOS, runtime.GOARCH)
	if rel.Version != "2.1.0" {
		t.Fatalf("version = %q", rel.Version)
	}
	wantAsset := update.ResolveAssetName(runtime.GOOS, runtime.GOARCH, "2.1.0")
	if rel.AssetName != wantAsset {
		t.Fatalf("asset = %q, want %q", rel.AssetName, wantAsset)
	}
	if rel.DownloadURL == "" || rel.Tag != "v2.1.0" {
		t.Fatalf("unexpected release: %+v", rel)
	}
}

func TestUpdateApplyAlreadyUpToDate(t *testing.T) {
	prevVersion := Version
	Version = "2.0.0"
	t.Cleanup(func() { Version = prevVersion })

	var buf bytes.Buffer
	prevStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = prevStdout })

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetName := update.ResolveAssetName(runtime.GOOS, runtime.GOARCH, "2.0.0")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v2.0.0",
			"published_at": time.Now().UTC(),
			"assets": []map[string]string{
				{"name": assetName, "browser_download_url": "http://example.invalid/asset"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	statePath := filepath.Join(t.TempDir(), "state.json")
	updateCheckerHook = func(current string, opts update.Options) (*update.Checker, error) {
		return update.NewChecker(current, update.Options{
			APIBaseURL: srv.URL,
			GOOS:       runtime.GOOS,
			GOARCH:     runtime.GOARCH,
			HTTPClient: srv.Client(),
			StatePath:  statePath,
		})
	}
	t.Cleanup(func() { updateCheckerHook = nil })

	updateApplyVersion = ""
	t.Cleanup(func() { updateApplyVersion = "" })

	if err := runUpdateApply(updateApplyCmd, nil); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	<-done

	if !bytes.Contains(buf.Bytes(), []byte("already up to date")) {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestUpdateCheckContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v1.0.0",
			"published_at": time.Now().UTC(),
			"assets":       []map[string]string{},
		})
	}))
	t.Cleanup(srv.Close)

	checker, err := update.NewChecker("0.9.0", update.Options{
		APIBaseURL: srv.URL,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		HTTPClient: srv.Client(),
		StatePath:  filepath.Join(t.TempDir(), "state.json"),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = checker.Check(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}
