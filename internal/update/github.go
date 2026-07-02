package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.github.com"

// GitHubClient fetches release metadata from the GitHub REST API.
type GitHubClient struct {
	owner      string
	repo       string
	apiBaseURL string
	goos       string
	goarch     string
	httpClient *http.Client
}

// NewGitHubClient builds a client with platform defaults for asset resolution.
func NewGitHubClient(opts Options) *GitHubClient {
	owner := opts.GitHubOwner
	if owner == "" {
		owner = DefaultOwner
	}
	repo := opts.GitHubRepo
	if repo == "" {
		repo = DefaultRepo
	}
	apiBase := opts.APIBaseURL
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	client := opts.HTTPClient
	if client == nil {
		client = newHTTPClient(defaultHTTPTimeout)
	}
	return &GitHubClient{
		owner:      owner,
		repo:       repo,
		apiBaseURL: strings.TrimRight(apiBase, "/"),
		goos:       opts.GOOS,
		goarch:     opts.GOARCH,
		httpClient: client,
	}
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// LatestRelease fetches the newest GitHub release and resolves the platform asset.
func (c *GitHubClient) LatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.apiBaseURL, c.owner, c.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github releases/latest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases/latest: status %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode github release: %w", err)
	}

	version := NormalizeVersion(release.TagName)
	assetName := ResolveAssetName(c.goos, c.goarch, version)
	downloadURL, err := findAssetURL(release.Assets, assetName)
	if err != nil {
		return nil, err
	}
	checksumsURL, _ := findAssetURL(release.Assets, "checksums.txt")

	return &ReleaseInfo{
		Tag:          release.TagName,
		Version:      version,
		AssetName:    assetName,
		DownloadURL:  downloadURL,
		ChecksumsURL: checksumsURL,
		PublishedAt:  release.PublishedAt,
	}, nil
}

func findAssetURL(assets []ghAsset, name string) (string, error) {
	for _, a := range assets {
		if a.Name == name {
			// Prefer API asset URL: browser_download_url ignores Bearer tokens on private repos.
			// https://stackoverflow.com/questions/77593100/github-private-release-asset-browser-download-url-returns-http-404-when-accessed
			if a.URL != "" {
				return a.URL, nil
			}
			if a.BrowserDownloadURL != "" {
				return a.BrowserDownloadURL, nil
			}
			break
		}
	}
	return "", fmt.Errorf("unsupported platform: no release asset %q", name)
}

// ResolveAssetName returns the expected archive filename for a platform build.
func ResolveAssetName(goos, goarch, version string) string {
	if goos == "windows" {
		return fmt.Sprintf("sideguard_%s_%s_%s.zip", version, goos, goarch)
	}
	return fmt.Sprintf("sideguard_%s_%s_%s.tar.gz", version, goos, goarch)
}

// ReleaseForVersion builds ReleaseInfo for a pinned version without a GitHub API call.
func ReleaseForVersion(version, goos, goarch string) ReleaseInfo {
	norm := NormalizeVersion(version)
	tag := "v" + norm
	if strings.HasPrefix(strings.TrimSpace(version), "v") {
		tag = strings.TrimSpace(version)
	}
	assetName := ResolveAssetName(goos, goarch, norm)
	base := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", DefaultOwner, DefaultRepo, tag)
	return ReleaseInfo{
		Tag:         tag,
		Version:     norm,
		AssetName:   assetName,
		DownloadURL: base + "/" + assetName,
	}
}

// ChecksumsURLForRelease returns the checksums.txt download URL for a release tag.
func ChecksumsURLForRelease(tag string) string {
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + NormalizeVersion(tag)
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt", DefaultOwner, DefaultRepo, tag)
}
