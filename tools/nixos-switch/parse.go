package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	_styleBuilding   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	_styleFetching   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	_styleActivating = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	_styleError      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	_styleDefault    = lipgloss.NewStyle()

	styleBuilding   = &_styleBuilding
	styleFetching   = &_styleFetching
	styleActivating = &_styleActivating
	styleError      = &_styleError
	styleDefault    = &_styleDefault
)

func categorizeLine(line string) (*lipgloss.Style, string) {
	switch {
	case strings.HasPrefix(line, "building "):
		return styleBuilding, line
	case strings.HasPrefix(line, "fetching "):
		return styleFetching, line
	case strings.HasPrefix(line, "activating "):
		return styleActivating, line
	case strings.HasPrefix(line, "error:"):
		return styleError, line
	default:
		return styleDefault, line
	}
}
