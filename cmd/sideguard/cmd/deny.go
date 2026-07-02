package cmd

import (
	"github.com/spf13/cobra"
)

var denyCmd = &cobra.Command{
	Use:   "deny [id]",
	Short: "Deny a pending request (id optional when only one pending)",
	Long:  "Denies a queued request. When exactly one item is pending, the id may be omitted. With multiple pending items, use `sideguard ui` or pass the id.",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolvePendingTargetFromArgs(args)
		if err != nil {
			return handlePendingResolveError(err, "deny")
		}
		reason, _ := cmd.Flags().GetString("reason")
		return decideApproval(id, "deny", reason)
	},
}

func init() {
	denyCmd.Flags().String("reason", "", "optional denial reason shown to the agent")
}
