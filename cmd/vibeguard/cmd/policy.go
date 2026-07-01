package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Validate and test YAML policy rules",
	Long: `Manage deterministic allow/deny/ask policy for shell commands and MCP tools.
See docs/plans/2026-07-01-0127-vibeguard-foundation/ (vgf-phase-7.0-policy-engine.md).`,
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate global and optional workspace policy files",
	RunE: func(_ *cobra.Command, _ []string) error {
		cwd, _ := os.Getwd()
		path, err := policy.GlobalPath()
		if err != nil {
			return err
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("global policy: not found (%s) — default is ask-all\n", path)
		} else {
			if _, err := policy.LoadFile(path); err != nil {
				return fmt.Errorf("global policy invalid: %w", err)
			}
			fmt.Printf("global policy: ok (%s)\n", path)
		}

		workspacePath := fmt.Sprintf("%s/.vibeguard/policy.yaml", cwd)
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			fmt.Printf("workspace policy: not found (%s)\n", workspacePath)
		} else {
			if _, err := policy.Load(cwd); err != nil {
				return fmt.Errorf("workspace policy invalid: %w", err)
			}
			fmt.Printf("workspace policy: ok (%s)\n", workspacePath)
		}

		if _, err := policy.Load(cwd); err != nil {
			return fmt.Errorf("merged policy invalid: %w", err)
		}
		fmt.Println("merged policy: ok")
		return nil
	},
}

var (
	policyTestCommand  string
	policyTestTool     string
	policyTestCWD      string
	policyTestWithLLM  bool
)

var policyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Evaluate a command or MCP tool against loaded policy",
	RunE: func(_ *cobra.Command, _ []string) error {
		if policyTestCommand == "" && policyTestTool == "" {
			return fmt.Errorf("provide --command and/or --tool")
		}

		cwd := policyTestCWD
		if cwd == "" {
			cwd, _ = os.Getwd()
		}

		input := policy.Input{
			Command:  policyTestCommand,
			ToolName: policyTestTool,
			CWD:      cwd,
		}

		var result policy.Result
		if policyTestWithLLM {
			clf, clfErr := llm.ClassifierFor(cwd)
			if clfErr != nil {
				fmt.Fprintf(os.Stderr, "classifier init error: %v\n", clfErr)
			}
			result = policy.EvaluateWithLLM(context.Background(), cwd, input, clf, llm.Enabled(cwd))
		} else {
			result = policy.Evaluate(cwd, input)
		}

		if result.LoadError != nil {
			fmt.Fprintf(os.Stderr, "policy load error: %v\n", result.LoadError)
		}

		fmt.Printf("action: %s\n", result.Action)
		if result.Reason != "" {
			fmt.Printf("reason: %s\n", result.Reason)
		}
		if result.LoadError != nil {
			return result.LoadError
		}
		return nil
	},
}

var policyInitDevCmd = &cobra.Command{
	Use:   "init-dev",
	Short: "Write repo-scoped workspace dev policy for local development",
	Long: `Creates .vibeguard/policy.yaml in the current repo with allow rules for make, go, and scripts/
only when the shell cwd is under this repo root. Safe for developing VibeGuard inside Cursor
without weakening global policy for other projects. Skips if the file already exists.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path, created, err := policy.EnsureDevWorkspacePolicy(cwd)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("workspace dev policy created: %s\n", path)
		} else {
			fmt.Printf("workspace dev policy already exists: %s\n", path)
		}
		return nil
	},
}

func init() {
	policyTestCmd.Flags().StringVar(&policyTestCommand, "command", "", "Shell command to evaluate")
	policyTestCmd.Flags().StringVar(&policyTestTool, "tool", "", "MCP tool name to evaluate")
	policyTestCmd.Flags().StringVar(&policyTestCWD, "cwd", "", "Workspace directory for policy override lookup")
	policyTestCmd.Flags().BoolVar(&policyTestWithLLM, "with-llm", false, "Run LLM triage after YAML returns ask (requires llm.enabled)")

	policyCmd.AddCommand(policyValidateCmd, policyTestCmd, policyInitDevCmd)
	rootCmd.AddCommand(policyCmd)
}
