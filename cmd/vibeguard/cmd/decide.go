package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alisaitteke/vibeguard/internal/api"
)

func decideApproval(id, decision, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := api.NewClient().Decide(ctx, id, decision, reason)
	if err != nil {
		if api.IsNotFound(err) {
			fmt.Fprintf(os.Stderr, "%s: approval not found: %s\n", decision, id)
			fmt.Fprintf(os.Stderr, "hint: run `vibeguard ui` or `vibeguard pending` to list current queue ids\n")
			return fmt.Errorf("approval not found")
		}
		if strings.Contains(err.Error(), "daemon unreachable") {
			return daemonNotRunningError(decision)
		}
		if strings.Contains(err.Error(), "already decided") {
			fmt.Fprintf(os.Stderr, "%s: request already decided: %s\n", decision, id)
			return err
		}
		return err
	}
	fmt.Printf("%s: %s\n", resp.Permission, resp.UserMessage)
	return nil
}
