package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alisaitteke/vibeguard/internal/api"
	"github.com/alisaitteke/vibeguard/internal/approvalfmt"
	"github.com/alisaitteke/vibeguard/internal/approvalmode"
)

const autoRefreshInterval = 2 * time.Second

const (
	autoAllowBanner = "*** AUTO-ALLOW MODE ***"
	autoDenyBanner  = "*** AUTO-DENY MODE ***"
)

type refreshDoneMsg struct {
	items []api.PendingApproval
	mode  approvalmode.Mode
	err   error
}

type decideDoneMsg struct {
	decision string
	id       string
	err      error
}

type setModeDoneMsg struct {
	mode  approvalmode.Mode
	err   error
}

type flashClearMsg struct{}

// model drives the interactive approval UI.
// Global approval mode is owned by the daemon; see gam-phase-4.0-tui-sync.md.
type model struct {
	client   *api.Client
	home     string
	items    []api.PendingApproval
	mode     approvalmode.Mode
	cursor   int
	err      string
	flash    string
	quitting bool
	width    int
	height   int
}

func newModel(client *api.Client) model {
	return model{
		client: client,
		home:   approvalfmt.HomeDir(),
		mode:   approvalmode.Ask,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(m.client), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type tickMsg struct{}

func refreshCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := client.ListPending(ctx)
		if err != nil {
			return refreshDoneMsg{err: err}
		}
		mode, modeErr := client.GetApprovalMode(ctx)
		if modeErr != nil {
			return refreshDoneMsg{items: items, err: modeErr}
		}
		return refreshDoneMsg{items: items, mode: mode}
	}
}

func decideCmd(client *api.Client, id, decision string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.Decide(ctx, id, decision, "")
		return decideDoneMsg{decision: decision, id: id, err: err}
	}
}

func setModeCmd(client *api.Client, mode approvalmode.Mode) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := client.SetApprovalMode(ctx, mode)
		return setModeDoneMsg{mode: mode, err: err}
	}
}

// nextMode cycles ask → auto → auto_allow → auto_deny → ask. Auto (smart
// triage) needs no warning banner: uncertain commands still queue for review.
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-11.0-tray-tui.md).
func nextMode(m approvalmode.Mode) approvalmode.Mode {
	switch m {
	case approvalmode.Ask:
		return approvalmode.Auto
	case approvalmode.Auto:
		return approvalmode.AutoAllow
	case approvalmode.AutoAllow:
		return approvalmode.AutoDeny
	default:
		return approvalmode.Ask
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if len(m.items) > 0 {
				m.cursor = ClampSelection(m.cursor-1, len(m.items))
			}
			return m, nil
		case "down", "j":
			if len(m.items) > 0 {
				m.cursor = ClampSelection(m.cursor+1, len(m.items))
			}
			return m, nil
		case "r":
			m.flash = "Refreshing..."
			return m, refreshCmd(m.client)
		case "a":
			if len(m.items) == 0 {
				m.flash = "Nothing to run"
				return m, flashClearAfter()
			}
			id := m.items[m.cursor].ID
			m.flash = "Running " + approvalfmt.ShortApprovalID(id) + "..."
			return m, decideCmd(m.client, id, "allow")
		case "d":
			if len(m.items) == 0 {
				m.flash = "Nothing to decline"
				return m, flashClearAfter()
			}
			id := m.items[m.cursor].ID
			m.flash = "Declining " + approvalfmt.ShortApprovalID(id) + "..."
			return m, decideCmd(m.client, id, "deny")
		case "g":
			next := nextMode(m.mode)
			m.flash = "Setting mode: " + next.Label() + "..."
			return m, setModeCmd(m.client, next)
		}

	case tickMsg:
		return m, tea.Batch(refreshCmd(m.client), tickCmd())

	case refreshDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.items = nil
			return m, nil
		}
		m.err = ""
		prevID := ""
		if len(m.items) > 0 && m.cursor < len(m.items) {
			prevID = m.items[m.cursor].ID
		}
		m.items = msg.items
		m.mode = msg.mode
		if prevID != "" {
			for i, item := range m.items {
				if item.ID == prevID {
					m.cursor = i
					return m, nil
				}
			}
		}
		m.cursor = ClampSelection(m.cursor, len(m.items))
		return m, nil

	case setModeDoneMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Mode change failed: %v", msg.err)
			return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))
		}
		m.mode = msg.mode
		m.flash = "Mode: " + msg.mode.Label()
		return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))

	case decideDoneMsg:
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed to %s: %v", msg.decision, msg.err)
			return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))
		}
		label := "run"
		if msg.decision == "deny" {
			label = "declined"
		}
		m.flash = approvalfmt.ShortApprovalID(msg.id) + " " + label
		return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))

	case flashClearMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func flashClearAfter() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

func modeBanner(mode approvalmode.Mode) string {
	switch mode {
	case approvalmode.AutoAllow:
		return autoAllowBanner
	case approvalmode.AutoDeny:
		return autoDenyBanner
	default:
		return ""
	}
}

func (m model) View() string {
	var b strings.Builder

	if banner := modeBanner(m.mode); banner != "" {
		b.WriteString(banner)
		b.WriteString("\n\n")
	}

	count := len(m.items)
	fmt.Fprintf(&b, "Pending approvals (%d)\n\n", count)

	if m.err != "" {
		fmt.Fprintf(&b, "  Error: %s\n\n", m.err)
	} else if count == 0 {
		b.WriteString("  (none — waiting...)\n\n")
	} else {
		for i, item := range m.items {
			prefix := "  "
			if i == m.cursor {
				prefix = "▶ "
			}
			fmt.Fprintf(&b, "%s%s\n", prefix, approvalfmt.FormatListLine(item, m.home))
		}
		b.WriteString("\n")
		if m.cursor < len(m.items) {
			cwd := approvalfmt.FormatCWD(m.items[m.cursor].CWD, m.home)
			fmt.Fprintf(&b, "  cwd: %s\n\n", cwd)
		}
	}

	b.WriteString("─────────────────────────────────────────\n")
	fmt.Fprintf(&b, "[a] Run  [d] Decline  [r] Refresh  [g] Mode: %s\n", m.mode.Label())
	b.WriteString("[q] Quit    ↑/↓ navigate\n")
	if m.flash != "" {
		fmt.Fprintf(&b, "\n%s\n", m.flash)
	}
	if m.quitting {
		b.WriteString("\n")
	}
	return b.String()
}
