package tui

import (
	"fmt"
	"strings"

	"sliver-client/internal/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectorModel struct {
	targets   []state.Target
	cursor    int
	wantsBack bool
}

func newSelectorModel() selectorModel {
	return selectorModel{}
}

func (m *selectorModel) refresh(s *state.State) {
	m.targets = s.AllTargets()
	if m.cursor >= len(m.targets) {
		m.cursor = max(0, len(m.targets)-1)
	}
}

func (m selectorModel) update(msg tea.Msg, s *state.State) (selectorModel, tea.Cmd) {
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
		case " ": // Space to toggle
			if m.cursor < len(m.targets) {
				s.ToggleSelected(m.targets[m.cursor].ID)
			}
		case "enter": // Confirm
			m.wantsBack = true
		case "a": // Select all
			s.SelectAll()
		case "n": // Select none
			s.SelectNone()
		case "w": // Select Windows only
			s.SelectByOS("windows")
		case "l": // Select Linux only
			s.SelectByOS("linux")
		case "esc":
			m.wantsBack = true
		}
	}
	return m, nil
}

func (m selectorModel) view(s *state.State, width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Target Selector") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d selected", s.SelectedCount())) + "\n\n")

	if len(m.targets) == 0 {
		b.WriteString(dimStyle.Render("  No targets available\n"))
		b.WriteString(dimStyle.Render("\n  Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	for i, t := range m.targets {
		kind := sessionStyle.Render("[S]")
		if t.Kind == state.KindBeacon {
			kind = beaconStyle.Render("[B]")
		}

		check := "  "
		nameStyle := unselectedStyle
		if s.IsSelected(t.ID) {
			check = selectedStyle.Render("✓ ")
			nameStyle = selectedStyle
		}

		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		line := fmt.Sprintf("%s%s%s %-10s %-16s %-16s %-8s %s",
			cursor, check, kind, t.ShortID,
			truncate(t.Hostname, 16), t.Address, t.OS,
			truncate(t.Username, 12))

		b.WriteString(nameStyle.Render(line) + "\n")
	}

	b.WriteString("\n")
	controls := []struct{ key, desc string }{
		{"SPACE", "toggle"},
		{"a", "select all"},
		{"n", "select none"},
		{"w", "windows only"},
		{"l", "linux only"},
		{"ENTER/ESC", "done"},
	}
	var parts []string
	for _, c := range controls {
		parts = append(parts, keyStyle.Render(c.key)+dimStyle.Render(" "+c.desc))
	}
	b.WriteString("  " + strings.Join(parts, "  "))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
