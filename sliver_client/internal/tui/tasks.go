package tui

import (
	"fmt"
	"strings"

	"sliver-client/internal/tracker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tasksModel struct {
	tasks      []tracker.TaskInfo
	cursor     int
	scroll     int
	showDetail bool
	wantsBack  bool
}

func newTasksModel() tasksModel {
	return tasksModel{}
}

func (m *tasksModel) refresh(t *tracker.Tracker) {
	m.tasks = t.AllTasks()
	if m.cursor >= len(m.tasks) {
		m.cursor = max(0, len(m.tasks)-1)
	}
}

func (m tasksModel) update(msg tea.Msg, t *tracker.Tracker) (tasksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDetail {
			switch msg.String() {
			case "esc", "enter":
				m.showDetail = false
			case "up", "k":
				if m.scroll > 0 {
					m.scroll--
				}
			case "down", "j":
				m.scroll++
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.tasks) {
				m.showDetail = true
				m.scroll = 0
			}
		case "r":
			t.CheckAll()
			m.refresh(t)
		case "c":
			t.ClearCompleted()
			m.refresh(t)
		case "esc":
			m.wantsBack = true
		}
	}
	return m, nil
}

func (m tasksModel) view(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Beacon Tasks") + "\n\n")

	if m.showDetail && m.cursor < len(m.tasks) {
		return m.viewDetail(width)
	}

	if len(m.tasks) == 0 {
		b.WriteString(dimStyle.Render("  No tracked tasks\n"))
		b.WriteString(dimStyle.Render("\n  Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Header
	header := fmt.Sprintf("  %-12s %-10s %-12s %-12s %-20s",
		"TASK ID", "STATE", "BEACON", "SUBMITTED", "DESCRIPTION")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Rows
	for i, task := range m.tasks {
		stateStr := string(task.State)
		var stateStyle lipgloss.Style
		switch task.State {
		case tracker.TaskPending:
			stateStyle = pendingStyle
		case tracker.TaskSent:
			stateStyle = beaconStyle
		case tracker.TaskCompleted:
			stateStyle = sessionStyle
		case tracker.TaskFailed:
			stateStyle = errorStyle
		}

		submitted := task.SubmittedAt.Format("15:04:05")

		row := fmt.Sprintf("  %-12s %s %-10s %-12s %-20s",
			truncate(task.TaskID, 12),
			stateStyle.Render(fmt.Sprintf("%-10s", stateStr)),
			truncate(task.BeaconID, 10),
			submitted,
			truncate(task.Description, 20))

		style := lipgloss.NewStyle()
		if i == m.cursor {
			style = style.Reverse(true)
		}
		b.WriteString(style.Render(row) + "\n")
	}

	pending := 0
	completed := 0
	for _, t := range m.tasks {
		if t.State == tracker.TaskCompleted || t.State == tracker.TaskFailed {
			completed++
		} else {
			pending++
		}
	}

	b.WriteString(fmt.Sprintf("\n  %s %d pending, %d completed",
		dimStyle.Render("Total:"), pending, completed))

	controls := "\n  " +
		keyStyle.Render("ENTER") + dimStyle.Render(" view output") + "  " +
		keyStyle.Render("r") + dimStyle.Render(" refresh") + "  " +
		keyStyle.Render("c") + dimStyle.Render(" clear done") + "  " +
		keyStyle.Render("ESC") + dimStyle.Render(" back")
	b.WriteString(controls)

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m tasksModel) viewDetail(width int) string {
	var b strings.Builder

	task := m.tasks[m.cursor]
	b.WriteString(titleStyle.Render("Task Detail") + "\n\n")
	b.WriteString(fmt.Sprintf("  Task ID:     %s\n", task.TaskID))
	b.WriteString(fmt.Sprintf("  Beacon ID:   %s\n", task.BeaconID))
	b.WriteString(fmt.Sprintf("  Description: %s\n", task.Description))
	b.WriteString(fmt.Sprintf("  State:       %s\n", task.State))
	b.WriteString(fmt.Sprintf("  Submitted:   %s\n", task.SubmittedAt.Format("2006-01-02 15:04:05")))
	if !task.CompletedAt.IsZero() {
		b.WriteString(fmt.Sprintf("  Completed:   %s\n", task.CompletedAt.Format("2006-01-02 15:04:05")))
	}

	b.WriteString("\n" + boldStyle.Render("  Output:") + "\n")
	if task.Output == "" {
		b.WriteString(dimStyle.Render("  (no output yet)") + "\n")
	} else {
		lines := strings.Split(task.Output, "\n")
		start := m.scroll
		if start >= len(lines) {
			start = max(0, len(lines)-1)
		}
		end := start + 20
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[start:end] {
			b.WriteString("  " + line + "\n")
		}
		if end < len(lines) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\n  ... %d more lines (↓ to scroll)", len(lines)-end)))
		}
	}

	b.WriteString(dimStyle.Render("\n  ↑/↓ scroll • ESC/ENTER back to list"))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
