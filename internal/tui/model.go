package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/alisaitteke/vibeguard/internal/api"
)

const autoRefreshInterval = 2 * time.Second

const autoApproveBanner = "*** AUTO-APPROVE MODE — all pending items are approved automatically ***"

type refreshDoneMsg struct {
	items []api.PendingApproval
	err   error
}

type decideDoneMsg struct {
	decision string
	id       string
	err      error
}

type flashClearMsg struct{}

// model drives the interactive approval UI.
type model struct {
	client      *api.Client
	home        string
	items       []api.PendingApproval
	cursor      int
	err         string
	flash       string
	quitting    bool
	width       int
	height      int
	autoApprove bool
	deciding    map[string]bool
}

func newModel(client *api.Client, opts Options) model {
	return model{
		client:      client,
		home:        HomeDir(),
		autoApprove: opts.AutoApprove,
		deciding:    make(map[string]bool),
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
		return refreshDoneMsg{items: items, err: err}
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

// pendingIDsForAutoApprove returns pending ids in FIFO order that are not already being decided.
func pendingIDsForAutoApprove(items []api.PendingApproval, deciding map[string]bool) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if deciding[item.ID] {
			continue
		}
		out = append(out, item.ID)
	}
	return out
}

func (m model) autoApproveCmds() tea.Cmd {
	if !m.autoApprove {
		return nil
	}
	ids := pendingIDsForAutoApprove(m.items, m.deciding)
	if len(ids) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(ids))
	for _, id := range ids {
		m.deciding[id] = true
		cmds = append(cmds, decideCmd(m.client, id, "allow"))
	}
	return tea.Batch(cmds...)
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
				m.flash = "Nothing to approve"
				return m, flashClearAfter()
			}
			id := m.items[m.cursor].ID
			m.flash = "Approving " + ShortApprovalID(id) + "..."
			return m, decideCmd(m.client, id, "allow")
		case "d":
			if len(m.items) == 0 {
				m.flash = "Nothing to deny"
				return m, flashClearAfter()
			}
			id := m.items[m.cursor].ID
			m.flash = "Denying " + ShortApprovalID(id) + "..."
			return m, decideCmd(m.client, id, "deny")
		case "g":
			m.autoApprove = !m.autoApprove
			if m.autoApprove {
				m.flash = "Auto-approve ON"
				if cmd := m.autoApproveCmds(); cmd != nil {
					return m, tea.Batch(flashClearAfter(), cmd)
				}
			} else {
				m.flash = "Auto-approve OFF"
			}
			return m, flashClearAfter()
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
		if prevID != "" {
			for i, item := range m.items {
				if item.ID == prevID {
					m.cursor = i
					if cmd := m.autoApproveCmds(); cmd != nil {
						return m, cmd
					}
					return m, nil
				}
			}
		}
		m.cursor = ClampSelection(m.cursor, len(m.items))
		if cmd := m.autoApproveCmds(); cmd != nil {
			return m, cmd
		}
		return m, nil

	case decideDoneMsg:
		delete(m.deciding, msg.id)
		if msg.err != nil {
			m.flash = fmt.Sprintf("Failed to %s: %v", msg.decision, msg.err)
			return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))
		}
		if m.autoApprove && msg.decision == "allow" {
			m.flash = formatAutoApproveFlash(msg.id, m.items)
		} else {
			label := "approved"
			if msg.decision == "deny" {
				label = "denied"
			}
			m.flash = ShortApprovalID(msg.id) + " " + label
		}
		return m, tea.Batch(flashClearAfter(), refreshCmd(m.client))

	case flashClearMsg:
		m.flash = ""
		return m, nil
	}

	return m, nil
}

func formatAutoApproveFlash(id string, items []api.PendingApproval) string {
	summary := ""
	for _, item := range items {
		if item.ID == id {
			summary = FormatSummary(item)
			break
		}
	}
	if summary != "" {
		return "Auto-approved " + ShortApprovalID(id) + " · " + summary
	}
	return "Auto-approved " + ShortApprovalID(id)
}

func flashClearAfter() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

func (m model) View() string {
	var b strings.Builder

	if m.autoApprove {
		b.WriteString(autoApproveBanner)
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
			fmt.Fprintf(&b, "%s%s\n", prefix, FormatListLine(item, m.home))
		}
		b.WriteString("\n")
		if m.cursor < len(m.items) {
			cwd := FormatCWD(m.items[m.cursor].CWD, m.home)
			fmt.Fprintf(&b, "  cwd: %s\n\n", cwd)
		}
	}

	b.WriteString("─────────────────────────────────────────\n")
	if m.autoApprove {
		b.WriteString("[a] Approve  [d] Deny  [r] Refresh  [g] Auto-approve OFF\n")
	} else {
		b.WriteString("[a] Approve  [d] Deny  [r] Refresh  [g] Auto-approve ON\n")
	}
	b.WriteString("[q] Quit    ↑/↓ navigate\n")
	if m.flash != "" {
		fmt.Fprintf(&b, "\n%s\n", m.flash)
	}
	if m.quitting {
		b.WriteString("\n")
	}
	return b.String()
}
