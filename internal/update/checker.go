package update

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

// Checker wraps GitHub release discovery, semver comparison, and state updates.
type Checker struct {
	current    string
	disabled   bool
	github     *GitHubClient
	stateStore StateStore
}

// NewChecker builds a Checker for the running binary version.
func NewChecker(currentVersion string, opts Options) (*Checker, error) {
	gh := opts.GitHubClient
	if gh == nil {
		goos := opts.GOOS
		if goos == "" {
			goos = runtime.GOOS
		}
		goarch := opts.GOARCH
		if goarch == "" {
			goarch = runtime.GOARCH
		}
		opts.GOOS = goos
		opts.GOARCH = goarch
		gh = NewGitHubClient(opts)
	}

	store := opts.StateStore
	if store == nil {
		if opts.StatePath != "" {
			store = NewFileStateStoreAt(opts.StatePath)
		} else {
			fs, err := NewFileStateStore()
			if err != nil {
				return nil, err
			}
			store = fs
		}
	}

	return &Checker{
		current:    currentVersion,
		disabled:   opts.Disabled,
		github:     gh,
		stateStore: store,
	}, nil
}

// ShouldAutoCheck reports whether background polling should run.
func (c *Checker) ShouldAutoCheck() bool {
	if c.disabled {
		return false
	}
	return !isDevVersion(c.current)
}

// Check fetches the latest release, compares semver, and updates state.last_check_at.
func (c *Checker) Check(ctx context.Context) (CheckResult, error) {
	result := CheckResult{Current: NormalizeVersion(c.current)}

	release, err := c.github.LatestRelease(ctx)
	if err != nil {
		_ = TouchLastCheck(c.stateStore, "")
		return result, err
	}

	result.Latest = release.Version
	result.UpdateAvailable = IsNewer(result.Current, result.Latest)
	if result.UpdateAvailable {
		rel := *release
		result.Release = &rel
	}

	if saveErr := TouchLastCheck(c.stateStore, release.Version); saveErr != nil {
		return result, fmt.Errorf("save update state: %w", saveErr)
	}
	return result, nil
}

func isDevVersion(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "" || v == "dev" || strings.Contains(v, "snapshot")
}
