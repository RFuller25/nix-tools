package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	accentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

func pad(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + spaces(width-w)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}

func center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return spaces(left) + s + spaces(width-w-left)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
