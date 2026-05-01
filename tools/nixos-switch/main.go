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
		if m.state != stateConfig && m.state != statePassword {
			logW := m.width - statsW - dividerW
			vpH := m.height - 6
			if logW < 1 {
				logW = 1
			}
			if vpH < 1 {
				vpH = 1
			}
			if m.viewport.Width == 0 {
				m.viewport = viewport.New(logW, vpH)
				m.viewport.SetContent(strings.Join(m.lines, "\n"))
				m.viewport.GotoBottom()
			} else {
				m.viewport.Width = logW
				m.viewport.Height = vpH
			}
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
				return m, checkSudo()
			}

		case statePassword:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.state = stateConfig
				m.passwordInput.SetValue("")
				m.sudoErr = ""
				m.pathInput.Focus()
				return m, textinput.Blink
			case "enter":
				pass := m.passwordInput.Value()
				if pass == "" {
					return m, nil
				}
				return m, authenticateSudo(pass)
			}

		case stateDone, stateError:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case stateRunning:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}

	case sudoOKMsg:
		logW := m.width - statsW - dividerW
		vpH := m.height - 6
		if logW < 1 {
			logW = 1
		}
		if vpH < 1 {
			vpH = 1
		}
		m.viewport = viewport.New(logW, vpH)
		m.state = stateRunning
		m.passwordInput.SetValue("")
		m.sudoErr = ""
		return m, startBuild(m.flakePath(), m.hostInput.Value())

	case sudoNeededMsg:
		m.state = statePassword
		m.passwordInput.Focus()
		return m, textinput.Blink

	case sudoFailedMsg:
		m.sudoErr = "wrong password, try again"
		m.passwordInput.SetValue("")
		return m, textinput.Blink

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

	// Forward remaining messages to focused component.
	switch m.state {
	case stateConfig:
		var cmd tea.Cmd
		if m.focusedInput == 0 {
			m.pathInput, cmd = m.pathInput.Update(msg)
		} else {
			m.hostInput, cmd = m.hostInput.Update(msg)
		}
		return m, cmd
	case statePassword:
		var cmd tea.Cmd
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
	case stateRunning, stateDone, stateError:
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
