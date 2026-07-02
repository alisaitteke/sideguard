package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/approvalfmt"
)

var pendingJSON bool

var pendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List pending approval requests",
	Long:  "Shows the daemon approval queue. Use approve/deny with the listed id.",
	RunE:  runPending,
}

func init() {
	pendingCmd.Flags().BoolVar(&pendingJSON, "json", false, "output machine-readable JSON")
}

func runPending(_ *cobra.Command, _ []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, err := api.NewClient().ListPending(ctx)
	if err != nil {
		return daemonNotRunningError("pending")
	}

	if pendingJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}

	printPendingTable(items)
	return nil
}

func printPendingTable(items []api.PendingApproval) {
	count := len(items)
	fmt.Printf("Pending approvals (%d):\n\n", count)
	if count == 0 {
		fmt.Println("  (none)")
		fmt.Println()
		fmt.Println("When an agent is blocked, new items appear here.")
		return
	}

	home := approvalfmt.HomeDir()
	for _, item := range items {
		fmt.Printf(" %s  %-7s  %s  %s\n",
			approvalfmt.ShortApprovalID(item.ID),
			approvalfmt.FormatClientLabel(item.Client),
			approvalfmt.FormatAgeLong(item.AgeSeconds),
			approvalfmt.FormatCWD(item.CWD, home),
		)
		fmt.Printf("     %s\n\n", approvalfmt.FormatSummary(item))
	}

	fmt.Println("Interactive: sideguard ui")
	fmt.Println("Scripting:   sideguard approve <id>  |  sideguard deny <id> [--reason \"...\"]")
	fmt.Println()
	fmt.Println("Tip: open a second terminal tab (tmux pane) while the agent waits.")
}
