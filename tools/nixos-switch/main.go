package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state != stateConfig {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateConfig:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "tab":
				if m.focusedInput == 0 {
					m.pathInput.Blur()
					m.hostInput.Focus()
					m.focusedInput = 1
				} else {
					m.hostInput.Blur()
					m.pathInput.Focus()
					m.focusedInput = 0
				}
				return m, textinput.Blink
			case "enter":
				m.state = stateRunning
				m.viewport = viewport.New(m.width, m.height-6)
				return m, startRebuild(m.flakePath(), m.hostInput.Value())
			}
		case stateDone, stateError:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateRunning:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}

	case cmdStartedMsg:
		m.reader = msg.reader
		m.pr = msg.pr
		m.exitCh = msg.exitCh
		return m, readNextLine(m.reader, m.pr, m.exitCh)

	case lineMsg:
		m.appendLine(string(msg))
		return m, readNextLine(m.reader, m.pr, m.exitCh)

	case cmdDoneMsg:
		if msg.lastLine != "" {
			m.appendLine(msg.lastLine)
		}
		if msg.exitCode == 0 {
			m.state = stateDone
		} else {
			m.state = stateError
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.state = stateError
		m.appendLine("error: " + msg.err.Error())
		return m, nil
	}

	if m.state == stateConfig {
		var cmd tea.Cmd
		if m.focusedInput == 0 {
			m.pathInput, cmd = m.pathInput.Update(msg)
		} else {
			m.hostInput, cmd = m.hostInput.Update(msg)
		}
		return m, cmd
	}

	if m.state == stateRunning || m.state == stateDone || m.state == stateError {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
