package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/alisaitteke/vibeguard/internal/api"
)

// Options configures the interactive approval UI session.
type Options struct {
	// AutoApprove automatically allows every pending item on each refresh (session-only).
	AutoApprove bool
}

// Run starts the interactive approval UI against the given API client.
// Requires a TTY on stdin (macOS Terminal, iTerm, tmux pane, etc.).
func Run(client *api.Client, opts Options) error {
	if client == nil {
		client = api.NewClient()
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return fmt.Errorf("vibeguard ui requires an interactive terminal (TTY); stdin is not a terminal")
	}

	p := tea.NewProgram(newModel(client, opts), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("ui: %w", err)
	}
	return nil
}
