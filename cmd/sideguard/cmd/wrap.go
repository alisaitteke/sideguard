package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/sideguard/internal/proxy"
)

var wrapCmd = &cobra.Command{
	Use:   "wrap -- <upstream-command> [args...]",
	Short: "Transparent STDIO MCP proxy with tools/call approval",
	Long: `Runs as a transparent MCP proxy between the IDE client and an upstream MCP server.
tools/call requests are held for daemon approval; initialize, ping, and tools/list pass through.

Example:
  sideguard wrap -- npx -y @modelcontextprotocol/server-filesystem /tmp`,
	DisableFlagParsing: true,
	RunE: func(_ *cobra.Command, args []string) error {
		upstream, err := parseWrapArgs(args)
		if err != nil {
			return err
		}
		return proxy.Wrap(upstream)
	},
}

func init() {
	rootCmd.AddCommand(wrapCmd)
}

// parseWrapArgs extracts upstream argv after the "--" separator.
func parseWrapArgs(args []string) ([]string, error) {
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep < 0 {
		return nil, fmt.Errorf("usage: sideguard wrap -- <upstream-command> [args...]")
	}
	upstream := args[sep+1:]
	if len(upstream) == 0 {
		return nil, fmt.Errorf("upstream command required after --")
	}
	return upstream, nil
}
