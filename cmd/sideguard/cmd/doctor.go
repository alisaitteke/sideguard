package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/doctor"
)

var (
	doctorCursor bool
	doctorClaude bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose SideGuard install health and bypass vectors",
	Long: `Checks daemon health, hook presence, MCP wrap status, and policy validity.
Reports HIGH when hooks are missing (bypass risk) or the daemon is down.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		report, err := doctor.Run(doctor.Options{
			Cursor: doctorCursor,
			Claude: doctorClaude,
		})
		if err != nil {
			return err
		}

		for _, f := range report.Findings {
			fmt.Printf("[%s] %s: %s\n", f.Severity, f.Check, f.Message)
		}

		if report.HasHigh() {
			fmt.Fprintln(os.Stderr, "\nSideGuard doctor: HIGH severity findings — run `sideguard install` to repair")
		} else {
			fmt.Println("\nSideGuard doctor: no HIGH severity findings")
		}

		if code := report.ExitCode(); code != 0 {
			os.Exit(code)
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorCursor, "cursor", false, "check Cursor configs only")
	doctorCmd.Flags().BoolVar(&doctorClaude, "claude", false, "check Claude Code configs only")
	rootCmd.AddCommand(doctorCmd)
}
