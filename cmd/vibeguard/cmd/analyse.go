package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/spf13/cobra"
)

var (
	analyseCommand string
	analyseEventID string
	analyseCWD     string
	analyseJSON    bool
)

// analyseClientHook overrides the API client in tests (nil uses default).
var analyseClientHook func() *api.Client

var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "On-demand LLM safety analysis of a shell command",
	Long: `Calls the daemon POST /v1/analyze for human-readable command safety analysis.

See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-4.0-cli.md).`,
	RunE: runAnalyse,
}

func init() {
	analyseCmd.Flags().StringVar(&analyseCommand, "command", "", "Shell command to analyze")
	analyseCmd.Flags().StringVar(&analyseEventID, "event-id", "", "Analyze a persisted command event by id")
	analyseCmd.Flags().StringVar(&analyseCWD, "cwd", "", "Working directory context for the command")
	analyseCmd.Flags().BoolVar(&analyseJSON, "json", false, "Output machine-readable JSON")
	rootCmd.AddCommand(analyseCmd)
}

func runAnalyse(_ *cobra.Command, _ []string) error {
	command := strings.TrimSpace(analyseCommand)
	eventID := strings.TrimSpace(analyseEventID)
	if command == "" && eventID == "" {
		return fmt.Errorf("provide --command or --event-id")
	}

	req := api.AnalyzeRequest{
		Command: command,
		CWD:     analyseCWD,
		EventID: eventID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := api.NewClient()
	if analyseClientHook != nil {
		client = analyseClientHook()
	}

	resp, err := client.Analyze(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "daemon unreachable") {
			return daemonNotRunningError("analyse")
		}
		return err
	}

	if analyseJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	printAnalyseHuman(resp)
	return nil
}

func printAnalyseHuman(resp *api.AnalyzeResponse) {
	fmt.Fprintf(os.Stdout, "verdict: %s\n", resp.Verdict)
	fmt.Fprintf(os.Stdout, "summary: %s\n", resp.Summary)
	fmt.Fprintf(os.Stdout, "explanation: %s\n", resp.Explanation)
	fmt.Fprintf(os.Stdout, "provider: %s\n", resp.Provider)
	if resp.DetectAction != "" {
		fmt.Fprintf(os.Stdout, "detect_action: %s\n", resp.DetectAction)
	}
	if len(resp.DetectRules) > 0 {
		fmt.Fprintf(os.Stdout, "detect_rules: %s\n", strings.Join(resp.DetectRules, ", "))
	}
}
