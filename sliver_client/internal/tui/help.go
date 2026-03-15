package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpModel struct {
	prevView  viewID
	wantsBack bool
}

func newHelpModel() helpModel {
	return helpModel{}
}

func (m helpModel) update(msg tea.Msg) (helpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?", "enter":
			m.wantsBack = true
		}
	}
	return m, nil
}

func (m helpModel) view(width int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Help — Sliver Mass Dispatch Client") + "\n\n")

	// Global keys
	b.WriteString(boldStyle.Render("Global Keys") + "\n")
	globalKeys := []struct{ key, desc string }{
		{"1", "Dashboard"},
		{"2 / s", "Sessions list"},
		{"3 / b", "Beacons list"},
		{"4 / x", "Target selector"},
		{"5 / d", "Dispatch command"},
		{"6 / t", "Beacon task tracker"},
		{"?", "This help screen"},
		{"q / Ctrl+C", "Quit"},
		{"ESC", "Back to previous view"},
	}
	for _, k := range globalKeys {
		b.WriteString("  " + keyStyle.Render(padRight(k.key, 12)) + descStyle.Render(k.desc) + "\n")
	}

	// Selector keys
	b.WriteString("\n" + boldStyle.Render("Target Selector") + "\n")
	selectKeys := []struct{ key, desc string }{
		{"SPACE", "Toggle selection on current target"},
		{"a", "Select all targets"},
		{"n", "Deselect all targets"},
		{"w", "Select all Windows targets"},
		{"l", "Select all Linux targets"},
		{"ENTER", "Confirm selection and go back"},
	}
	for _, k := range selectKeys {
		b.WriteString("  " + keyStyle.Render(padRight(k.key, 12)) + descStyle.Render(k.desc) + "\n")
	}

	// Task tracker keys
	b.WriteString("\n" + boldStyle.Render("Task Tracker") + "\n")
	taskKeys := []struct{ key, desc string }{
		{"ENTER", "View task output"},
		{"r", "Manually refresh task status"},
		{"c", "Clear completed tasks"},
	}
	for _, k := range taskKeys {
		b.WriteString("  " + keyStyle.Render(padRight(k.key, 12)) + descStyle.Render(k.desc) + "\n")
	}

	// Modules
	b.WriteString("\n" + boldStyle.Render("Dispatch Modules") + "\n")
	b.WriteString("\n  " + sessionStyle.Render("execute") + "\n")
	b.WriteString("  Run an executable on targets. Args: path, arguments (optional)\n")
	b.WriteString("  Sessions: returns output immediately\n")
	b.WriteString("  Beacons: queues task, check results in task tracker\n")

	b.WriteString("\n  " + sessionStyle.Render("upload") + "\n")
	b.WriteString("  Upload a local file to remote targets. Args: local path, remote path\n")
	b.WriteString("  File is read once locally, then sent to each target\n")

	b.WriteString("\n  " + sessionStyle.Render("script") + "\n")
	b.WriteString("  Execute a script on targets. Args: script path, PS assembly (optional)\n")
	b.WriteString("  Windows: runs via ExecuteAssembly with PowerShell loader\n")
	b.WriteString("  Linux: uploads to /tmp, executes via sh, auto-cleans\n")

	// Server safety
	b.WriteString("\n" + boldStyle.Render("Server Safety") + "\n")
	b.WriteString("  • Worker pool limits concurrent RPC calls (default: 5)\n")
	b.WriteString("  • Dead sessions/beacons are automatically filtered\n")
	b.WriteString("  • Each dispatch failure is isolated — never cancels other targets\n")
	b.WriteString("  • No destructive RPCs (kill, remove) are ever called\n")
	b.WriteString("  • Graceful shutdown: Ctrl+C cancels in-flight operations\n")

	b.WriteString(dimStyle.Render("\n  Press ESC or ? to close help"))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
