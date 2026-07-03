// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/spf13/cobra"
)

const (
	historyDefaultSince = "7d"
	historyMaxDisplay   = 60
)

var (
	historySince  string
	historyDenied bool
	historyCWD    string
	historyJSON   bool
)

// historyClientHook overrides the API client in tests (nil uses default).
var historyClientHook func() *api.Client

var historyCmd = &cobra.Command{
	Use:   "history [search]",
	Short: "Query local command intercept history",
	Long: `Lists persisted command_events from the daemon audit store.

See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-5.0-history-cli.md).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().StringVar(&historySince, "since", historyDefaultSince, "Only events after this duration ago (e.g. 24h, 7d)")
	historyCmd.Flags().BoolVar(&historyDenied, "denied", false, "Show only denied commands (final_action=deny)")
	historyCmd.Flags().StringVar(&historyCWD, "cwd", "", "Filter by working-directory prefix")
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output machine-readable JSON")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(_ *cobra.Command, args []string) error {
	search := ""
	if len(args) > 0 {
		search = args[0]
	}

	dur, err := parseHistorySince(historySince)
	if err != nil {
		return err
	}

	opts := historyQueryOpts{
		Since:  time.Now().UTC().Add(-dur),
		Denied: historyDenied,
		CWD:    historyCWD,
		Search: search,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := api.NewClient()
	if historyClientHook != nil {
		client = historyClientHook()
	}

	events, err := queryHistory(ctx, client, opts)
	if err != nil {
		return daemonNotRunningError("history")
	}

	if historyJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(events)
	}

	printHistoryTable(events)
	return nil
}

type historyQueryOpts struct {
	Since  time.Time
	Denied bool
	CWD    string
	Search string
}

func queryHistory(ctx context.Context, client *api.Client, opts historyQueryOpts) ([]api.CommandEvent, error) {
	q := api.EventQueryParams{
		Since:  opts.Since.UTC().Format(time.RFC3339),
		Denied: opts.Denied,
		CWD:    opts.CWD,
		Search: opts.Search,
		Limit:  1000,
	}
	return client.QueryEvents(ctx, q)
}

func printHistoryTable(events []api.CommandEvent) {
	if len(events) == 0 {
		fmt.Println("no events")
		return
	}

	fmt.Printf("Command history (%d):\n\n", len(events))
	fmt.Printf("%-20s  %-8s  %-10s  %s\n", "CREATED_AT", "ACTION", "DECISION", "COMMAND")
	for _, ev := range events {
		created := ev.CreatedAt
		if t, err := time.Parse(time.RFC3339, ev.CreatedAt); err == nil {
			created = t.Local().Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%-20s  %-8s  %-10s  %s\n",
			created,
			ev.FinalAction,
			ev.DecisionBy,
			truncateHistoryCommand(ev.CommandRedacted),
		)
	}
}

func truncateHistoryCommand(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= historyMaxDisplay {
		return s
	}
	return s[:historyMaxDisplay-3] + "..."
}

// parseHistorySince parses CLI durations like 24h and 7d (days as 24h units).
func parseHistorySince(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if strings.HasSuffix(raw, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid day duration %q", raw)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid --since duration %q: %w", raw, err)
	}
	return d, nil
}
