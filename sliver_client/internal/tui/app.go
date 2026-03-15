package tui

import (
	"sliver-client/internal/dispatch"
	"sliver-client/internal/state"
	"sliver-client/internal/tracker"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View identifiers
type viewID int

const (
	viewDashboard viewID = iota
	viewSessions
	viewBeacons
	viewSelector
	viewDispatch
	viewTasks
	viewHelp
)

// Messages
type refreshTickMsg time.Time
type trackerTickMsg time.Time

type dispatchCompleteMsg struct {
	results []dispatch.DispatchResult
}

// App is the top-level Bubbletea model.
type App struct {
	rpc     rpcpb.SliverRPCClient
	state   *state.State
	tracker *tracker.Tracker
	pool    *dispatch.Pool
	width   int
	height  int

	// Current view
	view viewID

	// Sub-models
	dashboard dashboardModel
	sessions  sessionsModel
	beacons   beaconsModel
	selector  selectorModel
	dispatch  dispatchModel
	tasks     tasksModel
	help      helpModel
}

func NewApp(rpc rpcpb.SliverRPCClient, s *state.State, t *tracker.Tracker, workers int) *App {
	return &App{
		rpc:     rpc,
		state:   s,
		tracker: t,
		pool: &dispatch.Pool{
			MaxWorkers: workers,
			Rpc:        rpc,
		},
		view:      viewDashboard,
		dashboard: newDashboardModel(),
		sessions:  newSessionsModel(),
		beacons:   newBeaconsModel(),
		selector:  newSelectorModel(),
		dispatch:  newDispatchModel(),
		tasks:     newTasksModel(),
		help:      newHelpModel(),
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tickRefresh(),
		tickTracker(),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		// Global keys (unless in text-input mode)
		if !a.isInputActive() {
			switch msg.String() {
			case "q", "ctrl+c":
				return a, tea.Quit
			case "1":
				a.view = viewDashboard
				return a, nil
			case "2", "s":
				if a.view == viewDashboard {
					a.view = viewSessions
					a.sessions.refresh(a.state)
					return a, nil
				}
			case "3", "b":
				if a.view == viewDashboard {
					a.view = viewBeacons
					a.beacons.refresh(a.state)
					return a, nil
				}
			case "4", "x":
				if a.view == viewDashboard {
					a.view = viewSelector
					a.selector.refresh(a.state)
					return a, nil
				}
			case "5", "d":
				if a.view == viewDashboard {
					a.view = viewDispatch
					a.dispatch.reset()
					return a, nil
				}
			case "6", "t":
				if a.view == viewDashboard {
					a.view = viewTasks
					a.tasks.refresh(a.tracker)
					return a, nil
				}
			case "?":
				if a.view != viewHelp {
					a.help.prevView = a.view
					a.view = viewHelp
					return a, nil
				}
			case "esc":
				if a.view != viewDashboard {
					a.view = viewDashboard
					return a, nil
				}
			}
		}

	case refreshTickMsg:
		// Refresh data from state
		switch a.view {
		case viewSessions:
			a.sessions.refresh(a.state)
		case viewBeacons:
			a.beacons.refresh(a.state)
		case viewSelector:
			a.selector.refresh(a.state)
		}
		cmds = append(cmds, tickRefresh())
		return a, tea.Batch(cmds...)

	case trackerTickMsg:
		if a.view == viewTasks {
			a.tasks.refresh(a.tracker)
		}
		cmds = append(cmds, tickTracker())
		return a, tea.Batch(cmds...)

	case dispatchCompleteMsg:
		a.dispatch.onComplete(msg.results, a.tracker)
		return a, nil
	}

	// Delegate to active view
	var cmd tea.Cmd
	switch a.view {
	case viewDashboard:
		a.dashboard, cmd = a.dashboard.update(msg)
	case viewSessions:
		a.sessions, cmd = a.sessions.update(msg, a.state)
		if a.sessions.wantsBack {
			a.sessions.wantsBack = false
			a.view = viewDashboard
		}
	case viewBeacons:
		a.beacons, cmd = a.beacons.update(msg, a.state)
		if a.beacons.wantsBack {
			a.beacons.wantsBack = false
			a.view = viewDashboard
		}
	case viewSelector:
		a.selector, cmd = a.selector.update(msg, a.state)
		if a.selector.wantsBack {
			a.selector.wantsBack = false
			a.view = viewDashboard
		}
	case viewDispatch:
		a.dispatch, cmd = a.dispatch.update(msg, a.state, a.pool, a.tracker)
		if a.dispatch.wantsBack {
			a.dispatch.wantsBack = false
			a.view = viewDashboard
		}
	case viewTasks:
		a.tasks, cmd = a.tasks.update(msg, a.tracker)
		if a.tasks.wantsBack {
			a.tasks.wantsBack = false
			a.view = viewDashboard
		}
	case viewHelp:
		a.help, cmd = a.help.update(msg)
		if a.help.wantsBack {
			a.help.wantsBack = false
			a.view = a.help.prevView
		}
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	statusBar := a.renderStatusBar()

	var content string
	switch a.view {
	case viewDashboard:
		content = a.dashboard.view(a.state, a.tracker, a.width)
	case viewSessions:
		content = a.sessions.view(a.width)
	case viewBeacons:
		content = a.beacons.view(a.width)
	case viewSelector:
		content = a.selector.view(a.state, a.width)
	case viewDispatch:
		content = a.dispatch.view(a.state, a.width, a.height)
	case viewTasks:
		content = a.tasks.view(a.width)
	case viewHelp:
		content = a.help.view(a.width)
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

func (a *App) renderStatusBar() string {
	sessions := len(a.state.Sessions())
	beacons := len(a.state.Beacons())
	selected := a.state.SelectedCount()
	pending := a.tracker.TotalCount() - a.tracker.CompletedCount()

	bar := statusBarStyle.Width(a.width).Render(
		sessionStyle.Render("Sessions: ") + boldStyle.Render(itoa(sessions)) + "  " +
			beaconStyle.Render("Beacons: ") + boldStyle.Render(itoa(beacons)) + "  " +
			selectedStyle.Render("Selected: ") + boldStyle.Render(itoa(selected)) + "  " +
			pendingStyle.Render("Tasks: ") + boldStyle.Render(itoa(pending)),
	)
	return bar
}

func (a *App) isInputActive() bool {
	return a.view == viewDispatch && a.dispatch.inputActive
}

func tickRefresh() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

func tickTracker() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return trackerTickMsg(t)
	})
}

func itoa(n int) string {
	return lipgloss.NewStyle().Render(intToStr(n))
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
