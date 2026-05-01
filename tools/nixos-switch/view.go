package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	statsW   = 33
	dividerW = 1
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	divStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	errorBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	doneBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	modalErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	statsTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	statsLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	progressStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	progressDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func (m model) View() string {
	switch m.state {
	case stateConfig:
		return m.viewConfig()
	case statePassword:
		return m.viewPassword()
	case stateRunning, stateDone, stateError:
		return m.viewRunning()
	}
	return ""
}

func (m model) viewConfig() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("nixos-switch") + "\n\n")
	b.WriteString(labelStyle.Render("Flake path:") + "\n")
	b.WriteString(m.pathInput.View() + "\n\n")
	b.WriteString(labelStyle.Render("Hostname:") + "\n")
	b.WriteString(m.hostInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("tab: switch field • enter: run • ctrl+c: quit"))
	return b.String()
}

func (m model) viewPassword() string {
	var inner strings.Builder
	inner.WriteString(titleStyle.Render("nixos-switch") + "\n\n")
	inner.WriteString(labelStyle.Render("sudo password") + "\n")
	inner.WriteString(m.passwordInput.View())
	if m.sudoErr != "" {
		inner.WriteString("\n\n" + modalErrStyle.Render(m.sudoErr))
	}
	inner.WriteString("\n\n" + helpStyle.Render("enter: authenticate  esc: back  ctrl+c: quit"))

	modal := modalBoxStyle.Render(inner.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m model) viewRunning() string {
	var b strings.Builder

	header := titleStyle.Render("nixos-switch") + "  "
	header += labelStyle.Render(m.pathInput.Value() + "#" + m.hostInput.Value())
	b.WriteString(header + "\n\n")

	if m.viewport.Width > 0 {
		logPane := m.viewport.View()
		div := renderDivider(m.viewport.Height)
		statPane := renderStats(m.stats, statsW, m.viewport.Height)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, logPane, div, statPane))
	}
	b.WriteString("\n")

	switch m.state {
	case stateDone:
		b.WriteString("\n" + doneBanner.Render("Done") + "\n")
		b.WriteString(helpStyle.Render("q / ctrl+c: quit"))
	case stateError:
		b.WriteString("\n" + errorBanner.Render("Failed — see errors above") + "\n")
		b.WriteString(helpStyle.Render("q / ctrl+c: quit"))
	default:
		b.WriteString(helpStyle.Render("↑↓: scroll  ctrl+c: abort"))
	}
	return b.String()
}

func renderDivider(height int) string {
	lines := make([]string, height)
	sep := divStyle.Render("│")
	for i := range lines {
		lines[i] = sep
	}
	return strings.Join(lines, "\n")
}

func renderStats(s stats, width, height int) string {
	var lines []string

	add := func(line string) {
		lines = append(lines, line)
	}

	phase := s.phase
	if phase == "" {
		phase = "evaluating"
	}

	add(statsTitleStyle.Render("Stats"))
	add("")
	add(statsLabelStyle.Render("Phase") + "  " + phase)
	add("")

	if s.totalPaths > 0 {
		pct := float64(s.fetchedPaths) / float64(s.totalPaths)
		barW := width - 2
		if barW < 4 {
			barW = 4
		}
		estMiB := pct * s.totalMiB
		add(statsLabelStyle.Render("Download"))
		add(" " + renderBar(pct, barW))
		add(fmt.Sprintf(" %.1f / %.1f MiB", estMiB, s.totalMiB))
		add(fmt.Sprintf(" %d / %d paths", s.fetchedPaths, s.totalPaths))
		add("")
	}

	if s.totalDrvs > 0 {
		add(statsLabelStyle.Render("Build"))
		add(fmt.Sprintf(" %d / %d dervs", s.builtDrvs, s.totalDrvs))
		add("")
	}

	if s.pkgsAdded+s.pkgsRemoved+s.pkgsChanged > 0 {
		add(statsLabelStyle.Render("Packages"))
		add(fmt.Sprintf(" ↑ %-4d + %-4d − %-4d",
			s.pkgsChanged, s.pkgsAdded, s.pkgsRemoved))
		add("")
	}

	if s.diskDelta != "" {
		add(statsLabelStyle.Render("Disk"))
		add(" " + s.diskDelta)
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

func renderBar(pct float64, width int) string {
	if pct > 1.0 {
		pct = 1.0
	}
	filled := int(math.Round(pct * float64(width)))
	empty := width - filled
	if empty < 0 {
		empty = 0
	}
	return progressStyle.Render(strings.Repeat("█", filled)) +
		progressDimStyle.Render(strings.Repeat("░", empty))
}
