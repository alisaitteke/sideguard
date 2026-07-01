package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/alisaitteke/vibeguard/internal/llm"
	"github.com/alisaitteke/vibeguard/internal/policy"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "LLM auto-triage diagnostics",
	Long: `Dry-run YAML + optional LLM classification without the daemon.
See docs/plans/2026-07-01-0318-llm-auto-triage/ (lat-phase-5.0-cli-config.md).`,
}

var (
	llmTestCommand string
	llmTestTool    string
	llmTestCWD     string
)

// llmTestClassifierHook is set by tests to inject a mock classifier.
var llmTestClassifierHook func(cwd string) (policy.Classifier, error)

var llmTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Dry-run YAML and optional LLM classification",
	Long:  "Evaluate a command or MCP tool against YAML policy and optional LLM triage.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runLLMTest(os.Stdout, llmTestOptions{
			Command: llmTestCommand,
			Tool:    llmTestTool,
			CWD:     llmTestCWD,
		})
	},
}

type llmTestOptions struct {
	Command string
	Tool    string
	CWD     string
}

type llmTestResult struct {
	YAMLAction string
	Action     string
	Reason     string
	LLMEnabled bool
	LLMInvoked bool
	Provider   string
	LatencyMS  int64
	LoadError  error
}

func runLLMTest(w io.Writer, opts llmTestOptions) error {
	if opts.Command == "" && opts.Tool == "" {
		return fmt.Errorf("provide --command and/or --tool")
	}

	cwd := opts.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		return err
	}

	input := policy.Input{
		Command:  opts.Command,
		ToolName: opts.Tool,
		CWD:      cwd,
	}

	result := evaluateLLMTest(context.Background(), cwd, input, cfg)
	if result.LoadError != nil {
		fmt.Fprintf(os.Stderr, "policy load error: %v\n", result.LoadError)
	}

	writeLLMTestOutput(w, result)
	if result.LoadError != nil {
		return result.LoadError
	}
	return nil
}

func evaluateLLMTest(ctx context.Context, cwd string, input policy.Input, cfg config.LLMConfig) llmTestResult {
	yamlResult := policy.Evaluate(cwd, input)
	out := llmTestResult{
		YAMLAction: string(yamlResult.Action),
		LLMEnabled: cfg.Enabled,
		Provider:   cfg.Provider,
		LoadError:  yamlResult.LoadError,
	}

	if yamlResult.LoadError != nil {
		out.Action = string(yamlResult.Action)
		out.Reason = yamlResult.Reason
		return out
	}

	if yamlResult.Action == policy.ActionAllow || yamlResult.Action == policy.ActionDeny {
		out.Action = string(yamlResult.Action)
		out.Reason = yamlResult.Reason
		return out
	}

	if !cfg.Enabled {
		out.Action = string(yamlResult.Action)
		out.Reason = yamlResult.Reason
		return out
	}

	var clf policy.Classifier
	var clfErr error
	if llmTestClassifierHook != nil {
		clf, clfErr = llmTestClassifierHook(cwd)
	} else {
		clf, clfErr = llm.ClassifierFor(cwd)
	}
	if clfErr != nil {
		fmt.Fprintf(os.Stderr, "classifier init error: %v\n", clfErr)
		out.Action = string(yamlResult.Action)
		out.Reason = yamlResult.Reason
		return out
	}

	start := time.Now()
	final := policy.EvaluateWithLLM(ctx, cwd, input, clf, true)
	out.LLMInvoked = true
	out.LatencyMS = time.Since(start).Milliseconds()
	out.Action = string(final.Action)
	out.Reason = final.Reason
	return out
}

func writeLLMTestOutput(w io.Writer, result llmTestResult) {
	fmt.Fprintf(w, "yaml_action: %s\n", result.YAMLAction)
	fmt.Fprintf(w, "action: %s\n", result.Action)
	if result.Reason != "" {
		fmt.Fprintf(w, "reason: %s\n", result.Reason)
	}
	fmt.Fprintf(w, "llm_enabled: %t\n", result.LLMEnabled)
	if result.LLMEnabled {
		fmt.Fprintf(w, "provider: %s\n", result.Provider)
	}
	if result.LLMInvoked {
		fmt.Fprintf(w, "latency_ms: %d\n", result.LatencyMS)
	}
}

func init() {
	llmTestCmd.Flags().StringVar(&llmTestCommand, "command", "", "Shell command to evaluate")
	llmTestCmd.Flags().StringVar(&llmTestTool, "tool", "", "MCP tool name to evaluate")
	llmTestCmd.Flags().StringVar(&llmTestCWD, "cwd", "", "Workspace directory for policy override lookup")

	llmCmd.AddCommand(llmTestCmd)
	rootCmd.AddCommand(llmCmd)
}
