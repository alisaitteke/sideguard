package update

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultHTTPTimeout = 60 * time.Second

// newHTTPClient returns an HTTP client that only follows HTTPS redirects to GitHub hosts.
// When GITHUB_TOKEN or GH_TOKEN is set, requests to GitHub hosts include Authorization.
// See https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api
func newHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	transport := http.RoundTripper(http.DefaultTransport)
	if token := githubTokenFromEnv(); token != "" {
		transport = &authTransport{base: transport, token: token}
	}
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: secureRedirect,
		Transport:     transport,
	}
}

func githubTokenFromEnv() string {
	for _, key := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.token == "" || !isGitHubAPIHost(strings.ToLower(req.URL.Hostname())) {
		return base.RoundTrip(req)
	}
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	cloned.Header.Set("Authorization", "Bearer "+t.token)
	return base.RoundTrip(cloned)
}

func isGitHubAPIHost(host string) bool {
	switch host {
	case "github.com", "api.github.com":
		return true
	default:
		return strings.HasSuffix(host, ".githubusercontent.com")
	}
}

func secureRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}
	return validateHTTPSURL(req.URL)
}

func validateHTTPSURL(u *url.URL) error {
	if u == nil {
		return errors.New("missing URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("insecure URL scheme: %s", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	if !isAllowedUpdateHost(host) {
		return fmt.Errorf("download host not allowed: %s", host)
	}
	return nil
}

func isAllowedUpdateHost(host string) bool {
	switch host {
	case "github.com", "api.github.com":
		return true
	default:
		return strings.HasSuffix(host, ".githubusercontent.com")
	}
}
