package tui

import "github.com/charmbracelet/lipgloss"

var (
	accent    = lipgloss.Color("#7C3AED") // violet
	subtle    = lipgloss.Color("#6B7280")
	text      = lipgloss.Color("#E5E7EB")
	highlight = lipgloss.Color("#A78BFA")
	danger    = lipgloss.Color("#EF4444")
	success   = lipgloss.Color("#10B981")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(subtle).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(text)

	dimStyle = lipgloss.NewStyle().
			Foreground(subtle)

	errorStyle = lipgloss.NewStyle().
			Foreground(danger)

	successStyle = lipgloss.NewStyle().
			Foreground(success)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			MarginTop(1)

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(subtle)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accent)

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(highlight).
				Bold(true)
)
