package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alisaitteke/vibeguard/internal/config"
	"github.com/spf13/cobra"
)

var (
	providerListJSON    bool
	providerAddID       string
	providerAddDriver   string
	providerAddModel    string
	providerAddBaseURL  string
	providerAddDefault  bool
	providerRemoveID    string
	providerSetKeyID    string
	providerSetKeyValue string
	providerDefaultID   string
)

var llmProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage LLM provider instances in ~/.vibeguard",
	Long: `Read and write provider settings via internal/config (never HTTP).

See docs/plans/2026-07-02-1521-llm-settings-analyse/ (lsa-phase-4.0-cli.md).`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured LLM providers",
	RunE:  runProviderList,
}

var providerAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new LLM provider instance",
	RunE:  runProviderAdd,
}

var providerRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an LLM provider instance",
	RunE:  runProviderRemove,
}

var providerSetKeyCmd = &cobra.Command{
	Use:   "set-key",
	Short: "Set API key for a provider instance",
	Long:  "Writes credentials.yaml (mode 0600). The key is never printed after set.",
	RunE:  runProviderSetKey,
}

var providerSetDefaultCmd = &cobra.Command{
	Use:   "set-default",
	Short: "Set the default LLM provider",
	RunE:  runProviderSetDefault,
}

type providerListRow struct {
	ID            string `json:"id"`
	Driver        string `json:"driver"`
	Model         string `json:"model"`
	BaseURL       string `json:"base_url,omitempty"`
	AuthMode      string `json:"auth_mode"`
	Default       bool   `json:"default"`
	KeyConfigured bool   `json:"key_configured"`
	KeyMasked     string `json:"key_masked,omitempty"`
}

func init() {
	providerListCmd.Flags().BoolVar(&providerListJSON, "json", false, "Output machine-readable JSON")

	providerAddCmd.Flags().StringVar(&providerAddID, "id", "", "Unique provider instance id")
	providerAddCmd.Flags().StringVar(&providerAddDriver, "driver", "", "Driver: openai, anthropic, ollama, openai-compatible")
	providerAddCmd.Flags().StringVar(&providerAddModel, "model", "", "Model name")
	providerAddCmd.Flags().StringVar(&providerAddBaseURL, "base-url", "", "Optional API base URL")
	providerAddCmd.Flags().BoolVar(&providerAddDefault, "default", false, "Set as default provider")
	_ = providerAddCmd.MarkFlagRequired("id")
	_ = providerAddCmd.MarkFlagRequired("driver")
	_ = providerAddCmd.MarkFlagRequired("model")

	providerRemoveCmd.Flags().StringVar(&providerRemoveID, "id", "", "Provider instance id to remove")
	_ = providerRemoveCmd.MarkFlagRequired("id")

	providerSetKeyCmd.Flags().StringVar(&providerSetKeyID, "id", "", "Provider instance id")
	providerSetKeyCmd.Flags().StringVar(&providerSetKeyValue, "key", "", "API key value")
	_ = providerSetKeyCmd.MarkFlagRequired("id")
	_ = providerSetKeyCmd.MarkFlagRequired("key")

	providerSetDefaultCmd.Flags().StringVar(&providerDefaultID, "id", "", "Provider instance id")
	_ = providerSetDefaultCmd.MarkFlagRequired("id")

	llmProviderCmd.AddCommand(providerListCmd, providerAddCmd, providerRemoveCmd, providerSetKeyCmd, providerSetDefaultCmd)
	llmCmd.AddCommand(llmProviderCmd)
}

func runProviderList(_ *cobra.Command, _ []string) error {
	settings, err := config.LoadLLMSettings("")
	if err != nil {
		return err
	}

	rows, err := buildProviderListRows(settings)
	if err != nil {
		return err
	}

	if providerListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	if len(rows) == 0 {
		fmt.Println("no providers configured")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDRIVER\tMODEL\tDEFAULT\tKEY")
	for _, row := range rows {
		defaultMark := ""
		if row.Default {
			defaultMark = "*"
		}
		keyCol := "not set"
		if row.KeyConfigured {
			keyCol = row.KeyMasked
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.ID, row.Driver, row.Model, defaultMark, keyCol)
	}
	return w.Flush()
}

func buildProviderListRows(settings config.LLMSettings) ([]providerListRow, error) {
	rows := make([]providerListRow, 0, len(settings.Providers))
	for _, p := range settings.Providers {
		masked, configured, err := config.ProviderStatus(p.ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, providerListRow{
			ID:            p.ID,
			Driver:        p.Driver,
			Model:         p.Model,
			BaseURL:       p.BaseURL,
			AuthMode:      p.AuthMode,
			Default:       p.ID == settings.DefaultProvider,
			KeyConfigured: configured,
			KeyMasked:     masked,
		})
	}
	return rows, nil
}

func runProviderAdd(_ *cobra.Command, _ []string) error {
	settings, err := config.LoadLLMSettings("")
	if err != nil {
		return err
	}

	instance := config.ProviderInstance{
		ID:       strings.TrimSpace(providerAddID),
		Driver:   strings.TrimSpace(providerAddDriver),
		Model:    strings.TrimSpace(providerAddModel),
		BaseURL:  strings.TrimSpace(providerAddBaseURL),
		AuthMode: "api_key",
	}

	if _, err := config.AddProvider(settings, instance); err != nil {
		return err
	}

	if providerAddDefault {
		if err := config.SetDefaultProvider(instance.ID); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stdout, "provider %q added\n", instance.ID)
	return nil
}

func runProviderRemove(_ *cobra.Command, _ []string) error {
	id := strings.TrimSpace(providerRemoveID)
	settings, err := config.LoadLLMSettings("")
	if err != nil {
		return err
	}

	found := false
	for _, p := range settings.Providers {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("provider %q not found", id)
	}
	if settings.DefaultProvider == id {
		return fmt.Errorf("cannot remove default provider %q; set another default first with `vibeguard llm provider set-default`", id)
	}

	if err := config.RemoveProvider(id); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "provider %q removed\n", id)
	return nil
}

func runProviderSetKey(_ *cobra.Command, _ []string) error {
	id := strings.TrimSpace(providerSetKeyID)
	key := strings.TrimSpace(providerSetKeyValue)
	if key == "" {
		return fmt.Errorf("--key must not be empty")
	}

	settings, err := config.LoadLLMSettings("")
	if err != nil {
		return err
	}

	found := false
	for _, p := range settings.Providers {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("provider %q not found", id)
	}

	if err := config.SetProviderKey(id, key); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "api key set for provider %q\n", id)
	return nil
}

func runProviderSetDefault(_ *cobra.Command, _ []string) error {
	id := strings.TrimSpace(providerDefaultID)
	if err := config.SetDefaultProvider(id); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "default provider set to %q\n", id)
	return nil
}
