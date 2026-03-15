package tui

import (
	"fmt"
	"strings"
	"time"

	"sliver-client/internal/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type beaconsModel struct {
	targets   []state.Target
	cursor    int
	wantsBack bool
}

func newBeaconsModel() beaconsModel {
	return beaconsModel{}
}

func (m *beaconsModel) refresh(s *state.State) {
	m.targets = s.Beacons()
	if m.cursor >= len(m.targets) {
		m.cursor = max(0, len(m.targets)-1)
	}
}

func (m beaconsModel) update(msg tea.Msg, s *state.State) (beaconsModel, tea.Cmd) {
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

func (m beaconsModel) view(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Beacons") + "\n\n")

	if len(m.targets) == 0 {
		b.WriteString(dimStyle.Render("  No active beacons\n"))
		b.WriteString(dimStyle.Render("\n  Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Header
	header := fmt.Sprintf("  %-10s %-16s %-16s %-8s %-12s %-12s %-12s %-6s",
		"ID", "HOSTNAME", "ADDRESS", "OS", "USERNAME", "LAST CHECK", "NEXT CHECK", "TASKS")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Rows
	for i, t := range m.targets {
		lastCheckin := formatAgo(t.LastCheckin)
		nextCheckin := formatAgo(t.NextCheckin)

		row := fmt.Sprintf("  %-10s %-16s %-16s %-8s %-12s %-12s %-12s %-6d",
			t.ShortID, truncate(t.Hostname, 16), t.Address, t.OS,
			truncate(t.Username, 12), lastCheckin, nextCheckin, t.TasksCount)

		style := beaconStyle
		if i == m.cursor {
			style = style.Reverse(true)
		}
		b.WriteString(style.Render(row) + "\n")
	}

	b.WriteString(fmt.Sprintf("\n  %s %d beacons", dimStyle.Render("Total:"), len(m.targets)))
	b.WriteString(dimStyle.Render("\n  ↑/↓ navigate • ESC back"))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func formatAgo(t time.Time) string {
	if t.IsZero() || t.Unix() <= 0 {
		return "n/a"
	}
	d := time.Since(t)
	if d < 0 {
		d = -d
		// Future time (next checkin)
		if d < time.Minute {
			return fmt.Sprintf("in %ds", int(d.Seconds()))
		}
		if d < time.Hour {
			return fmt.Sprintf("in %dm", int(d.Minutes()))
		}
		return fmt.Sprintf("in %dh", int(d.Hours()))
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(d.Hours()))
}
