package update

import (
	"net/http"
	"time"
)

const (
	// DefaultOwner is the GitHub organization/user hosting release assets.
	DefaultOwner = "alisaitteke"
	// DefaultRepo is the GitHub repository name for release assets.
	DefaultRepo = "vibeguard"
)

// ReleaseInfo describes a GitHub release asset matching the current platform.
type ReleaseInfo struct {
	Tag         string    `json:"tag"`
	Version     string    `json:"version"`
	AssetName   string    `json:"asset_name"`
	DownloadURL string    `json:"download_url"`
	PublishedAt time.Time `json:"published_at"`
}

// CheckResult is returned by Checker.Check after comparing semver versions.
type CheckResult struct {
	Current         string       `json:"current"`
	Latest          string       `json:"latest"`
	UpdateAvailable bool         `json:"update_available"`
	Release         *ReleaseInfo `json:"release,omitempty"`
}

// State is persisted at ~/.vibeguard/update-state.json between background checks.
type State struct {
	LastCheckAt      time.Time `json:"last_check_at"`
	LatestKnown      string    `json:"latest_known"`
	DismissedVersion string    `json:"dismissed_version"`
	DownloadPath     string    `json:"download_path"`
}

// Options configures Checker construction and GitHub asset resolution.
type Options struct {
	Disabled     bool
	GitHubOwner  string
	GitHubRepo   string
	APIBaseURL   string
	HTTPClient   *http.Client
	GOOS         string
	GOARCH       string
	StatePath    string
	StateStore   StateStore
	GitHubClient *GitHubClient
}

// ApplyOptions configures binary replacement during Apply.
type ApplyOptions struct {
	TargetPath    string
	Platform      PlatformApplier
	ChecksumsPath string
	SkipDownload  bool
	ArchivePath   string
	ChecksumsURL  string
}

// StateStore abstracts update state persistence for tests.
type StateStore interface {
	Load() (State, error)
	Save(State) error
}
