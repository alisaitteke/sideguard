// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/daemon"
)

// ErrNoPendingApprovals is returned when the queue is empty and no id was given.
var ErrNoPendingApprovals = errors.New("no pending approvals")

// MultiplePendingError is returned when more than one pending item exists and no id was given.
type MultiplePendingError struct {
	Items []api.PendingApproval
}

func (e *MultiplePendingError) Error() string {
	return fmt.Sprintf("multiple pending approvals (%d)", len(e.Items))
}

// resolveApprovalID picks the approval id to act on from an optional user id and the pending list.
// When id is empty: auto-selects the sole pending item, or returns a typed error for zero/many.
// When id is set: resolves prefix matches against pending; otherwise returns id unchanged.
func resolveApprovalID(id string, pending []api.PendingApproval) (string, error) {
	id = strings.TrimSpace(id)
	if id != "" {
		for i := range pending {
			if pending[i].ID == id || matchIDPrefix(pending[i].ID, id) {
				return pending[i].ID, nil
			}
		}
		return id, nil
	}

	switch len(pending) {
	case 0:
		return "", ErrNoPendingApprovals
	case 1:
		return pending[0].ID, nil
	default:
		return "", &MultiplePendingError{Items: pending}
	}
}

func resolvePendingTarget(ctx context.Context, id string) (string, error) {
	pending, err := api.NewClient().ListPending(ctx)
	if err != nil {
		return "", err
	}
	return resolveApprovalID(id, pending)
}

func resolvePendingTargetFromArgs(args []string) (string, error) {
	var id string
	if len(args) > 0 {
		id = args[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return resolvePendingTarget(ctx, id)
}

func handlePendingResolveError(err error, verb string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNoPendingApprovals) {
		fmt.Println("No pending approvals.")
		fmt.Println()
		fmt.Println("Hint: run `sideguard ui` to watch for new requests.")
		return nil
	}
	var multi *MultiplePendingError
	if errors.As(err, &multi) {
		fmt.Fprintf(os.Stderr, "Multiple pending approvals — run `sideguard ui` or `sideguard %s <id>`\n\n", verb)
		printPendingTable(multi.Items)
		return fmt.Errorf("multiple pending approvals")
	}
	if strings.Contains(err.Error(), "daemon unreachable") {
		return daemonNotRunningError(verb)
	}
	return err
}

func runRootDefault() error {
	line, err := daemon.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon is not running\n")
		fmt.Fprintf(os.Stderr, "hint: start the daemon with `sideguard daemon start`\n")
		fmt.Fprintf(os.Stderr, "health endpoint: %s\n", api.HealthURL())
	} else {
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Println("Interactive approvals: sideguard ui")
	fmt.Println("List queue:          sideguard pending")
	fmt.Println("Quick approve/deny:  sideguard approve  |  sideguard deny  (when one pending)")
	return nil
}
