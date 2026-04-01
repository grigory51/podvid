package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			PaddingLeft(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			PaddingLeft(1)

	normalStyle = lipgloss.NewStyle().
			PaddingLeft(3)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1).
			PaddingTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			PaddingLeft(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("76")).
			PaddingLeft(1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			PaddingLeft(1)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			PaddingLeft(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1)
)
