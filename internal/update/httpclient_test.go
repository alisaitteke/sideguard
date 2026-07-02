package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateHTTPSURLRejectsInsecureScheme(t *testing.T) {
	cases := []string{
		"http://github.com/alisaitteke/vibeguard/releases/download/v1.0.0/asset.tar.gz",
		"ftp://github.com/asset.tar.gz",
	}
	for _, raw := range cases {
		u, err := http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateHTTPSURL(u.URL); err == nil {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

func TestValidateHTTPSURLAllowsGitHubHosts(t *testing.T) {
	cases := []string{
		"https://github.com/alisaitteke/vibeguard/releases/download/v1.0.0/asset.tar.gz",
		"https://api.github.com/repos/alisaitteke/vibeguard/releases/latest",
		"https://release-assets.githubusercontent.com/github-production-release-asset/123/asset.tar.gz",
	}
	for _, raw := range cases {
		u, err := http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateHTTPSURL(u.URL); err != nil {
			t.Fatalf("expected allow for %q: %v", raw, err)
		}
	}
}

func TestValidateHTTPSURLRejectsUnknownHost(t *testing.T) {
	raw := "https://evil.example.com/malware.tar.gz"
	u, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateHTTPSURL(u.URL); err == nil {
		t.Fatal("expected rejection for unknown host")
	}
}

func TestDownloaderRejectsHTTPURL(t *testing.T) {
	dl := NewDownloader(nil)
	_, err := dl.Download(context.Background(), "http://github.com/asset.tar.gz", t.TempDir(), "asset.tar.gz")
	if err == nil {
		t.Fatal("expected insecure scheme rejection")
	}
}

func TestSecureRedirectRejectsUnknownHost(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://evil.example.com/payload.tar.gz", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := secureRedirect(req, []*http.Request{{}}); err == nil {
		t.Fatal("expected redirect host rejection")
	}
}

func TestDownloaderRejectsRedirectToUnknownHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.com/payload.tar.gz", http.StatusFound)
	}))
	defer srv.Close()

	// Use injected client so the test can reach the redirect handler; secureRedirect still runs.
	client := newHTTPClient(0)
	dl := &Downloader{httpClient: client}
	_, err := dl.Download(context.Background(), srv.URL, t.TempDir(), "asset.tar.gz")
	if err == nil {
		t.Fatal("expected redirect host rejection")
	}
}
