package tui

import (
	"fmt"
	"strings"

	"sliver-client/internal/state"
	"sliver-client/internal/tracker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dashboardModel struct{}

func newDashboardModel() dashboardModel {
	return dashboardModel{}
}

func (d dashboardModel) update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	return d, nil
}

func (d dashboardModel) view(s *state.State, t *tracker.Tracker, width int) string {
	var b strings.Builder

	title := titleStyle.Render("🍑 Sliver Mass Dispatch Client")
	b.WriteString(title + "\n\n")

	// Summary box
	sessions := s.Sessions()
	beacons := s.Beacons()
	selected := s.SelectedCount()
	pendingTasks := t.TotalCount() - t.CompletedCount()

	winSess, linSess := countByOS(sessions)
	winBeac, linBeac := countByOS(beacons)

	summary := boxStyle.Width(min(60, width-4)).Render(
		boldStyle.Render("Connected to Sliver Server") + "\n\n" +
			sessionStyle.Render("Sessions") + "\n" +
			fmt.Sprintf("  Total: %d  (Windows: %d, Linux: %d)\n\n", len(sessions), winSess, linSess) +
			beaconStyle.Render("Beacons") + "\n" +
			fmt.Sprintf("  Total: %d  (Windows: %d, Linux: %d)\n\n", len(beacons), winBeac, linBeac) +
			selectedStyle.Render(fmt.Sprintf("Selected targets: %d", selected)) + "\n" +
			pendingStyle.Render(fmt.Sprintf("Pending beacon tasks: %d", pendingTasks)),
	)
	b.WriteString(summary + "\n\n")

	// Quick actions
	actions := []struct{ key, desc string }{
		{"s / 2", "View sessions"},
		{"b / 3", "View beacons"},
		{"x / 4", "Select targets"},
		{"d / 5", "Dispatch command"},
		{"t / 6", "View beacon tasks"},
		{"?", "Help"},
		{"q", "Quit"},
	}
	b.WriteString(boldStyle.Render("Quick Actions") + "\n")
	for _, a := range actions {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keyStyle.Render(fmt.Sprintf("%-5s", a.key)),
			descStyle.Render(a.desc)))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func countByOS(targets []state.Target) (win, lin int) {
	for _, t := range targets {
		switch strings.ToLower(t.OS) {
		case "windows":
			win++
		case "linux":
			lin++
		}
	}
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
