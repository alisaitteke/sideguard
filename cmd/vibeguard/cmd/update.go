package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/update"
)

var (
	updateCheckJSON    bool
	updateStatusJSON   bool
	updateApplyVersion string
	updateApplyRestart bool
	updateApplyYes     bool
)

// updateCheckerHook overrides checker construction in tests (nil uses default).
var updateCheckerHook func(current string, opts update.Options) (*update.Checker, error)

// updateApplierHook overrides applier construction in tests (nil uses default).
var updateApplierHook func() *update.Applier

// updateStateStoreHook overrides state store in tests (nil uses default).
var updateStateStoreHook func() (update.StateStore, error)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and apply GitHub release updates",
	Long: `Checks GitHub Releases for newer vibeguard binaries and applies verified updates.

See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-3.0-update-cli.md).`,
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Compare the running binary against the latest GitHub release",
	RunE:  runUpdateCheck,
}

var updateApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Download, verify, and install a release update",
	RunE:  runUpdateApply,
}

var updateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show persisted update check state",
	RunE:  runUpdateStatus,
}

func init() {
	updateCheckCmd.Flags().BoolVar(&updateCheckJSON, "json", false, "Output machine-readable JSON")
	updateStatusCmd.Flags().BoolVar(&updateStatusJSON, "json", false, "Output machine-readable JSON")
	updateApplyCmd.Flags().StringVar(&updateApplyVersion, "version", "", "Install a specific release version (default: latest)")
	updateApplyCmd.Flags().BoolVar(&updateApplyRestart, "restart", false, "Restart daemon and tray after apply")
	updateApplyCmd.Flags().BoolVar(&updateApplyYes, "yes", false, "Skip confirmation prompt")

	updateCmd.AddCommand(updateCheckCmd, updateApplyCmd, updateStatusCmd)
	rootCmd.AddCommand(updateCmd)
}

type updateCheckJSONOut struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	AssetName       string `json:"asset_name,omitempty"`
	DownloadURL     string `json:"download_url,omitempty"`
}

type updateStatusJSONOut struct {
	Current          string `json:"current"`
	LastCheckAt      string `json:"last_check_at,omitempty"`
	LatestKnown      string `json:"latest_known,omitempty"`
	DownloadPath     string `json:"download_path,omitempty"`
	DismissedVersion string `json:"dismissed_version,omitempty"`
	UpdateEnabled    bool   `json:"update_enabled"`
	AutoCheckEnabled bool   `json:"auto_check_enabled"`
}

func runUpdateCheck(_ *cobra.Command, _ []string) error {
	updateCfg, err := config.LoadUpdate()
	if err != nil {
		return err
	}

	checker, err := newUpdateChecker(updateCfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := checker.Check(ctx)
	if err != nil {
		return fmt.Errorf("update check: %w", err)
	}

	if updateCheckJSON {
		out := updateCheckJSONOut{
			Current:         result.Current,
			Latest:          result.Latest,
			UpdateAvailable: result.UpdateAvailable,
		}
		if result.Release != nil {
			out.AssetName = result.Release.AssetName
			out.DownloadURL = result.Release.DownloadURL
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("current: %s\n", result.Current)
	fmt.Printf("latest:  %s\n", result.Latest)
	if result.UpdateAvailable {
		fmt.Println("update available: yes")
		if result.Release != nil {
			fmt.Printf("asset: %s\n", result.Release.AssetName)
		}
	} else {
		fmt.Println("update available: no")
	}
	return nil
}

func runUpdateApply(_ *cobra.Command, _ []string) error {
	updateCfg, err := config.LoadUpdate()
	if err != nil {
		return err
	}

	checker, err := newUpdateChecker(updateCfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var rel update.ReleaseInfo
	if updateApplyVersion != "" {
		rel = update.ReleaseForVersion(updateApplyVersion, runtime.GOOS, runtime.GOARCH)
		if !update.IsNewer(update.NormalizeVersion(Version), rel.Version) {
			fmt.Println("already up to date")
			return nil
		}
	} else {
		result, err := checker.Check(ctx)
		if err != nil {
			return fmt.Errorf("update check: %w", err)
		}
		if !result.UpdateAvailable || result.Release == nil {
			fmt.Println("already up to date")
			return nil
		}
		rel = *result.Release
	}

	if err := confirmUpdateApply(rel.Version, updateApplyYes); err != nil {
		return err
	}

	applier := newUpdateApplier()
	var platform update.PlatformApplier = update.NoopPlatformApplier{}
	if updateApplyRestart {
		platform = update.NewPlatformApplier()
	}

	err = applier.Apply(ctx, rel, update.ApplyOptions{
		Platform:     platform,
		ChecksumsURL: checksumsURLForApply(rel),
	})
	if err != nil {
		return fmt.Errorf("update apply: %w", err)
	}

	fmt.Printf("updated to %s\n", rel.Version)
	return nil
}

func runUpdateStatus(_ *cobra.Command, _ []string) error {
	updateCfg, err := config.LoadUpdate()
	if err != nil {
		return err
	}

	checker, err := newUpdateChecker(updateCfg)
	if err != nil {
		return err
	}

	store, err := newUpdateStateStore()
	if err != nil {
		return err
	}

	st, err := store.Load()
	if err != nil {
		return fmt.Errorf("read update state: %w", err)
	}

	current := update.NormalizeVersion(Version)
	autoCheck := checker.ShouldAutoCheck()

	if updateStatusJSON {
		out := updateStatusJSONOut{
			Current:          current,
			UpdateEnabled:    updateCfg.Enabled,
			AutoCheckEnabled: autoCheck,
		}
		if !st.LastCheckAt.IsZero() {
			out.LastCheckAt = st.LastCheckAt.UTC().Format(time.RFC3339)
		}
		if st.LatestKnown != "" {
			out.LatestKnown = st.LatestKnown
		}
		if st.DownloadPath != "" {
			out.DownloadPath = st.DownloadPath
		}
		if st.DismissedVersion != "" {
			out.DismissedVersion = st.DismissedVersion
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("current: %s\n", current)
	if !st.LastCheckAt.IsZero() {
		fmt.Printf("last check: %s\n", st.LastCheckAt.Local().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("last check: never")
	}
	if st.LatestKnown != "" {
		fmt.Printf("latest known: %s\n", st.LatestKnown)
	} else {
		fmt.Println("latest known: (none)")
	}
	if !updateCfg.Enabled {
		fmt.Println("background check: disabled (update.enabled: false)")
	} else if !autoCheck {
		fmt.Println("background check: disabled (dev or snapshot build)")
	} else {
		fmt.Printf("background check: enabled (every %s)\n", updateCfg.CheckInterval)
	}
	return nil
}

func newUpdateChecker(updateCfg config.UpdateConfig) (*update.Checker, error) {
	opts := update.Options{Disabled: !updateCfg.Enabled}
	if updateCheckerHook != nil {
		return updateCheckerHook(Version, opts)
	}
	return update.NewChecker(Version, opts)
}

func newUpdateApplier() *update.Applier {
	if updateApplierHook != nil {
		return updateApplierHook()
	}
	return update.NewApplier(nil, runtime.GOOS)
}

func newUpdateStateStore() (update.StateStore, error) {
	if updateStateStoreHook != nil {
		return updateStateStoreHook()
	}
	return update.NewFileStateStore()
}

func checksumsURLForApply(rel update.ReleaseInfo) string {
	if rel.ChecksumsURL != "" {
		return rel.ChecksumsURL
	}
	return update.ChecksumsURLForRelease(rel.Tag)
}

func confirmUpdateApply(version string, yes bool) error {
	if yes {
		return nil
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("non-interactive stdin: use --yes to apply without confirmation")
	}
	fmt.Printf("Apply v%s? [y/N] ", version)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer == "y" || answer == "yes" {
		return nil
	}
	return fmt.Errorf("update apply cancelled")
}
