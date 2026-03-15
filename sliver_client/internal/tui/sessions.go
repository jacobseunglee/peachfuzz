package tui

import (
	"fmt"
	"strings"

	"sliver-client/internal/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionsModel struct {
	targets   []state.Target
	cursor    int
	wantsBack bool
}

func newSessionsModel() sessionsModel {
	return sessionsModel{}
}

func (m *sessionsModel) refresh(s *state.State) {
	m.targets = s.Sessions()
	if m.cursor >= len(m.targets) {
		m.cursor = max(0, len(m.targets)-1)
	}
}

func (m sessionsModel) update(msg tea.Msg, s *state.State) (sessionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.targets)-1 {
				m.cursor++
			}
		case "esc":
			m.wantsBack = true
		}
	}
	return m, nil
}

func (m sessionsModel) view(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Sessions") + "\n\n")

	if len(m.targets) == 0 {
		b.WriteString(dimStyle.Render("  No active sessions\n"))
		b.WriteString(dimStyle.Render("\n  Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Header
	header := fmt.Sprintf("  %-10s %-18s %-16s %-10s %-15s %-6s %-15s",
		"ID", "HOSTNAME", "ADDRESS", "OS", "USERNAME", "PID", "PROCESS")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Rows
	for i, t := range m.targets {
		row := fmt.Sprintf("  %-10s %-18s %-16s %-10s %-15s %-6d %-15s",
			t.ShortID, truncate(t.Hostname, 18), t.Address, t.OS, truncate(t.Username, 15), t.PID, truncate(t.Process, 15))

		style := sessionStyle
		if i == m.cursor {
			style = style.Reverse(true)
		}
		b.WriteString(style.Render(row) + "\n")
	}

	b.WriteString(fmt.Sprintf("\n  %s %d sessions", dimStyle.Render("Total:"), len(m.targets)))
	b.WriteString(dimStyle.Render("\n  ↑/↓ navigate • ESC back"))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
