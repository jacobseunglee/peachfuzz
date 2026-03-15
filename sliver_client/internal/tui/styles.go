package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen  = lipgloss.Color("#00FF00")
	colorBlue   = lipgloss.Color("#5599FF")
	colorRed    = lipgloss.Color("#FF3333")
	colorYellow = lipgloss.Color("#FFCC00")
	colorCyan   = lipgloss.Color("#00CCCC")
	colorDim    = lipgloss.Color("#666666")
	colorWhite  = lipgloss.Color("#FFFFFF")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(colorWhite).
			Padding(0, 1)

	sessionStyle = lipgloss.NewStyle().Foreground(colorGreen)
	beaconStyle  = lipgloss.NewStyle().Foreground(colorBlue)
	errorStyle   = lipgloss.NewStyle().Foreground(colorRed)
	pendingStyle = lipgloss.NewStyle().Foreground(colorYellow)
	dimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	boldStyle    = lipgloss.NewStyle().Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorDim)

	tableRowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			Background(lipgloss.Color("#333333")).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(1, 2)

	successBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(colorGreen).
			Padding(0, 1)

	errorBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(colorRed).
			Padding(0, 1)

	pendingBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(colorYellow).
			Padding(0, 1)
)
