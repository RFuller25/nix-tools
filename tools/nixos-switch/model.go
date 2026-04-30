package main

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int

const (
	stateConfig appState = iota
	stateRunning
	stateDone
	stateError
)

type model struct {
	state        appState
	pathInput    textinput.Model
	hostInput    textinput.Model
	focusedInput int
	viewport     viewport.Model
	lines        []string
	reader       *bufio.Reader
	exitCode     int
	err          error
	width        int
	height       int
}

type cmdStartedMsg struct{ reader *bufio.Reader }
type lineMsg string
type cmdDoneMsg struct{ exitCode int }
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func initialModel() model {
	hostname, _ := os.Hostname()

	path := textinput.New()
	path.Placeholder = "~/.config/flakes"
	path.SetValue("~/.config/flakes")
	path.Focus()
	path.Width = 50

	host := textinput.New()
	host.Placeholder = hostname
	host.SetValue(hostname)
	host.Width = 50

	return model{
		state:        stateConfig,
		pathInput:    path,
		hostInput:    host,
		focusedInput: 0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) flakePath() string {
	p := m.pathInput.Value()
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		p = home + p[1:]
	}
	return p
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
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
)

func (m model) View() string {
	switch m.state {
	case stateConfig:
		return m.viewConfig()
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

func (m model) viewRunning() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("nixos-switch") + " ")
	b.WriteString(labelStyle.Render(m.pathInput.Value()+"#"+m.hostInput.Value()) + "\n\n")
	b.WriteString(m.viewport.View() + "\n")

	switch m.state {
	case stateDone:
		b.WriteString("\n" + doneBanner.Render("Done") + "\n")
		b.WriteString(helpStyle.Render("q / ctrl+c: quit"))
	case stateError:
		b.WriteString("\n" + errorBanner.Render("Failed — see errors above") + "\n")
		b.WriteString(helpStyle.Render("q / ctrl+c: quit"))
	default:
		b.WriteString(helpStyle.Render("ctrl+c: abort"))
	}
	return b.String()
}

func (m *model) appendLine(line string) {
	style, _ := categorizeLine(line)
	m.lines = append(m.lines, style.Render(line))
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	m.viewport.GotoBottom()
}

func readNextLine(r *bufio.Reader) tea.Cmd {
	return func() tea.Msg {
		line, err := r.ReadString('\n')
		line = strings.TrimRight(line, "\n\r")
		if err == io.EOF {
			if line != "" {
				return lineMsg(line)
			}
			return cmdDoneMsg{exitCode: 0}
		}
		if err != nil {
			return errMsg{err}
		}
		return lineMsg(line)
	}
}
