package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

var approveAlways bool

var approveCmd = &cobra.Command{
	Use:   "approve [id]",
	Short: "Approve a pending request (id optional when only one pending)",
	Long:  "Approves a queued request. When exactly one item is pending, the id may be omitted. With multiple pending items, use `vibeguard ui` or pass the id.",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(_ *cobra.Command, args []string) error {
		id, err := resolvePendingTargetFromArgs(args)
		if err != nil {
			return handlePendingResolveError(err, "approve")
		}
		if approveAlways {
			if err := appendAlwaysAllowRule(id); err != nil {
				return err
			}
		}
		return decideApproval(id, "allow", "")
	},
}

func init() {
	approveCmd.Flags().BoolVar(&approveAlways, "always", false, "Append an allow rule for this command/tool and approve")
}

func appendAlwaysAllowRule(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := api.NewClient()
	pending, err := client.ListPending(ctx)
	if err != nil {
		return err
	}

	var item *api.PendingApproval
	for i := range pending {
		if pending[i].ID == id || matchIDPrefix(pending[i].ID, id) {
			item = &pending[i]
			break
		}
	}
	if item == nil {
		return fmt.Errorf("approval not found in pending queue: %s", id)
	}

	var match policy.Match
	reason := fmt.Sprintf("always allow from approve %s", item.ID)
	switch item.Source {
	case "mcp":
		match.MCPTool = "^" + regexp.QuoteMeta(item.ToolName) + "$"
	default:
		match.Command = "^" + regexp.QuoteMeta(item.Command) + "$"
	}

	return policy.AppendAllowRule(match, reason)
}

func matchIDPrefix(full, prefix string) bool {
	if len(prefix) >= len(full) {
		return full == prefix
	}
	return full[:len(prefix)] == prefix
}
