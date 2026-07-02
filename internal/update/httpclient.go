package update

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHTTPTimeout = 60 * time.Second

// newHTTPClient returns an HTTP client that only follows HTTPS redirects to GitHub hosts.
func newHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: secureRedirect,
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
