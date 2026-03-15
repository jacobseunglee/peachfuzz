package tui

import (
	"context"
	"fmt"
	"strings"

	"sliver-client/internal/dispatch"
	"sliver-client/internal/state"
	"sliver-client/internal/tracker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dispatch workflow steps
type dispatchStep int

const (
	stepPickModule dispatchStep = iota
	stepInputArgs
	stepPickScope
	stepRunning
	stepResults
)

type moduleID int

const (
	modExecute moduleID = iota
	modUpload
	modScript
)

type dispatchModel struct {
	step         dispatchStep
	module       moduleID
	moduleCursor int
	scopeCursor  int

	// Argument inputs
	inputs      [3]string // up to 3 args depending on module
	inputIdx    int       // which input field is active
	inputActive bool

	// Scope
	scope string // "all", "windows", "linux", "selected"

	// Running/results
	running      bool
	results      []dispatch.DispatchResult
	resultScroll int

	wantsBack bool
}

var moduleNames = []string{"execute", "upload", "script"}
var moduleDescs = []string{
	"Run an executable on targets",
	"Upload a local file to targets",
	"Execute a script (PS on Windows, sh on Linux)",
}

var scopeNames = []string{"selected", "all", "windows only", "linux only"}

func newDispatchModel() dispatchModel {
	return dispatchModel{step: stepPickModule}
}

func (m *dispatchModel) reset() {
	*m = dispatchModel{step: stepPickModule}
}

func (m *dispatchModel) onComplete(results []dispatch.DispatchResult, t *tracker.Tracker) {
	m.running = false
	m.step = stepResults
	m.results = results

	// Register beacon tasks with the tracker
	for _, r := range results {
		if r.BeaconTaskID != "" {
			desc := moduleNames[m.module]
			t.Register(r.BeaconTaskID, r.Target.ID, desc)
		}
	}
}

func (m dispatchModel) update(msg tea.Msg, s *state.State, pool *dispatch.Pool, t *tracker.Tracker) (dispatchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case stepPickModule:
			return m.updatePickModule(msg)
		case stepInputArgs:
			return m.updateInputArgs(msg)
		case stepPickScope:
			return m.updatePickScope(msg, s, pool)
		case stepResults:
			return m.updateResults(msg)
		}
	}
	return m, nil
}

func (m dispatchModel) updatePickModule(msg tea.KeyMsg) (dispatchModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.moduleCursor > 0 {
			m.moduleCursor--
		}
	case "down", "j":
		if m.moduleCursor < len(moduleNames)-1 {
			m.moduleCursor++
		}
	case "enter":
		m.module = moduleID(m.moduleCursor)
		m.step = stepInputArgs
		m.inputActive = true
		m.inputIdx = 0
		m.inputs = [3]string{}
	case "esc":
		m.wantsBack = true
	}
	return m, nil
}

func (m dispatchModel) updateInputArgs(msg tea.KeyMsg) (dispatchModel, tea.Cmd) {
	maxInputs := m.argCount()

	switch msg.String() {
	case "enter":
		// Validate required args
		if m.inputIdx < maxInputs-1 || (m.module == modExecute && m.inputIdx == 0) {
			// For execute: first arg is required, second is optional
			if m.module == modExecute && m.inputIdx == 0 && m.inputs[0] == "" {
				return m, nil // Can't proceed without executable path
			}
			m.inputIdx++
			if m.inputIdx >= maxInputs {
				m.inputActive = false
				m.step = stepPickScope
			}
			return m, nil
		}
		// All inputs collected
		m.inputActive = false
		m.step = stepPickScope
	case "tab":
		m.inputIdx = (m.inputIdx + 1) % maxInputs
	case "esc":
		if m.inputIdx > 0 {
			m.inputIdx--
		} else {
			m.inputActive = false
			m.step = stepPickModule
		}
	case "backspace":
		if len(m.inputs[m.inputIdx]) > 0 {
			m.inputs[m.inputIdx] = m.inputs[m.inputIdx][:len(m.inputs[m.inputIdx])-1]
		}
	case "ctrl+u":
		m.inputs[m.inputIdx] = ""
	case "up", "down", "left", "right", "home", "end",
		"pgup", "pgdown", "delete", "insert",
		"ctrl+a", "ctrl+e", "ctrl+c", "ctrl+d",
		"shift+tab", "f1", "f2", "f3", "f4",
		"f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
		// Ignore navigation/control keys
	default:
		// Accept single keystrokes and multi-character paste events
		if len(msg.Runes) > 0 {
			m.inputs[m.inputIdx] += string(msg.Runes)
		}
	}
	return m, nil
}

func (m dispatchModel) updatePickScope(msg tea.KeyMsg, s *state.State, pool *dispatch.Pool) (dispatchModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.scopeCursor > 0 {
			m.scopeCursor--
		}
	case "down", "j":
		if m.scopeCursor < len(scopeNames)-1 {
			m.scopeCursor++
		}
	case "enter":
		m.scope = scopeNames[m.scopeCursor]
		m.step = stepRunning
		m.running = true
		return m, m.startDispatch(s, pool)
	case "esc":
		m.step = stepInputArgs
		m.inputActive = true
	}
	return m, nil
}

func (m dispatchModel) updateResults(msg tea.KeyMsg) (dispatchModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.resultScroll > 0 {
			m.resultScroll--
		}
	case "down", "j":
		m.resultScroll++
	case "esc", "enter":
		m.wantsBack = true
	}
	return m, nil
}

func (m dispatchModel) startDispatch(s *state.State, pool *dispatch.Pool) tea.Cmd {
	targets := m.getTargets(s)
	fn, err := m.buildDispatchFunc()
	if err != nil {
		return func() tea.Msg {
			return dispatchCompleteMsg{results: []dispatch.DispatchResult{{
				Error: err,
			}}}
		}
	}

	return func() tea.Msg {
		ctx := context.Background()
		results := pool.Run(ctx, targets, fn)
		return dispatchCompleteMsg{results: results}
	}
}

func (m dispatchModel) getTargets(s *state.State) []state.Target {
	switch m.scope {
	case "selected":
		return s.Selected()
	case "windows only":
		return filterOS(s.AllTargets(), "windows")
	case "linux only":
		return filterOS(s.AllTargets(), "linux")
	default: // "all"
		return s.AllTargets()
	}
}

func filterOS(targets []state.Target, os string) []state.Target {
	var out []state.Target
	for _, t := range targets {
		if strings.EqualFold(t.OS, os) {
			out = append(out, t)
		}
	}
	return out
}

func (m dispatchModel) buildDispatchFunc() (dispatch.DispatchFunc, error) {
	switch m.module {
	case modExecute:
		path := strings.TrimSpace(m.inputs[0])
		if path == "" {
			return nil, fmt.Errorf("executable path is required")
		}
		var args []string
		if a := strings.TrimSpace(m.inputs[1]); a != "" {
			args = strings.Fields(a)
		}
		return dispatch.ExecuteCmd(path, args), nil

	case modUpload:
		local := strings.TrimSpace(m.inputs[0])
		remote := strings.TrimSpace(m.inputs[1])
		if local == "" || remote == "" {
			return nil, fmt.Errorf("both local and remote paths are required")
		}
		return dispatch.UploadCmd(local, remote)

	case modScript:
		script := strings.TrimSpace(m.inputs[0])
		assembly := strings.TrimSpace(m.inputs[1])
		if script == "" {
			return nil, fmt.Errorf("script path is required")
		}
		return dispatch.ScriptCmd(script, assembly)

	default:
		return nil, fmt.Errorf("unknown module")
	}
}

func (m dispatchModel) argCount() int {
	switch m.module {
	case modExecute:
		return 2 // path, args (optional)
	case modUpload:
		return 2 // local path, remote path
	case modScript:
		return 2 // script path, assembly path (optional for linux)
	}
	return 1
}

func (m dispatchModel) view(s *state.State, width, height int) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Dispatch Command") + "\n\n")

	switch m.step {
	case stepPickModule:
		b.WriteString(boldStyle.Render("  Select module:") + "\n\n")
		for i, name := range moduleNames {
			cursor := "  "
			if i == m.moduleCursor {
				cursor = "▸ "
			}
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, boldStyle.Render(name)))
			b.WriteString(fmt.Sprintf("      %s\n", dimStyle.Render(moduleDescs[i])))
		}
		b.WriteString(dimStyle.Render("\n  ↑/↓ navigate • ENTER select • ESC back"))

	case stepInputArgs:
		b.WriteString(boldStyle.Render("  Module: "+moduleNames[m.module]) + "\n\n")
		labels := m.argLabels()
		for i, label := range labels {
			prefix := "  "
			if i == m.inputIdx {
				prefix = "▸ "
			}
			val := m.inputs[i]
			if i == m.inputIdx {
				val += "█"
			}
			b.WriteString(fmt.Sprintf("  %s%s: %s\n", prefix, dimStyle.Render(label), val))
		}
		b.WriteString(dimStyle.Render("\n  ENTER next/confirm • TAB switch field • ESC back"))

	case stepPickScope:
		targets := m.getTargets(s)
		_ = targets
		b.WriteString(boldStyle.Render("  Select scope:") + "\n\n")
		for i, name := range scopeNames {
			cursor := "  "
			if i == m.scopeCursor {
				cursor = "▸ "
			}
			count := len(m.getScopeTargets(s, name))
			b.WriteString(fmt.Sprintf("  %s%-20s %s\n", cursor,
				boldStyle.Render(name),
				dimStyle.Render(fmt.Sprintf("(%d targets)", count))))
		}
		b.WriteString(dimStyle.Render("\n  ↑/↓ navigate • ENTER dispatch • ESC back"))

	case stepRunning:
		b.WriteString(pendingStyle.Render("  ⟳ Dispatching...") + "\n")
		b.WriteString(dimStyle.Render("  Please wait"))

	case stepResults:
		b.WriteString(boldStyle.Render("  Results:") + "\n\n")

		if len(m.results) == 0 {
			b.WriteString(dimStyle.Render("  No results\n"))
		} else {
			successes := 0
			failures := 0
			taskQueued := 0
			for _, r := range m.results {
				if r.Error != nil {
					failures++
				} else if r.BeaconTaskID != "" {
					taskQueued++
				} else {
					successes++
				}
			}
			b.WriteString(fmt.Sprintf("  %s %d  %s %d  %s %d\n\n",
				selectedStyle.Render("Success:"), successes,
				errorStyle.Render("Failed:"), failures,
				pendingStyle.Render("Queued:"), taskQueued))

			// Show individual results (scrollable by line, not by result)
			var allLines []string
			for _, r := range m.results {
				allLines = append(allLines, strings.Split(dispatch.FormatResult(r), "\n")...)
				allLines = append(allLines, "") // blank separator between results
			}
			// Use terminal height minus overhead (title, summary, footer, status bar)
			maxVisible := height - 10
			if maxVisible < 5 {
				maxVisible = 5
			}
			start := m.resultScroll
			if start >= len(allLines) {
				start = max(0, len(allLines)-1)
			}
			end := start + maxVisible
			if end > len(allLines) {
				end = len(allLines)
			}
			for _, line := range allLines[start:end] {
				b.WriteString(line + "\n")
			}
			if end < len(allLines) {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more lines (↓ to scroll)", len(allLines)-end)))
			}
		}
		b.WriteString(dimStyle.Render("\n  ↑/↓ scroll • ESC/ENTER back to dashboard"))
	}

	// Only pad non-results views; results need full width for command output
	if m.step == stepResults {
		return b.String()
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m dispatchModel) argLabels() []string {
	switch m.module {
	case modExecute:
		return []string{"Executable path", "Arguments (optional)"}
	case modUpload:
		return []string{"Local file path", "Remote destination path"}
	case modScript:
		return []string{"Script file path", "PS assembly path (optional, Windows)"}
	}
	return []string{"Argument"}
}

func (m dispatchModel) getScopeTargets(s *state.State, scope string) []state.Target {
	switch scope {
	case "selected":
		return s.Selected()
	case "windows only":
		return filterOS(s.AllTargets(), "windows")
	case "linux only":
		return filterOS(s.AllTargets(), "linux")
	default:
		return s.AllTargets()
	}
}
