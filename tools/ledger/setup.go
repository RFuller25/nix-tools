package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The setup flow is wordguesser-term's: key, then username, then a test call
// before anything is saved.

type setupStep int

const (
	stepAPIKey setupStep = iota
	stepUsername
	stepValidating
	stepError
)

type setupModel struct {
	step        setupStep
	apiKeyInput textinput.Model
	userInput   textinput.Model
	spinner     spinner.Model
	errMsg      string
	done        bool
	config      *Config
}

type setupDoneMsg struct{ config *Config }
type setupErrMsg struct{ err error }

func newSetupModel(existingUsername string) setupModel {
	apiKey := textinput.New()
	apiKey.Placeholder = "Enter your API key"
	apiKey.EchoMode = textinput.EchoPassword
	apiKey.EchoCharacter = '*'
	apiKey.Focus()
	apiKey.Width = 40

	user := textinput.New()
	user.Placeholder = "Enter your username"
	user.Width = 40
	if existingUsername != "" {
		user.SetValue(existingUsername)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1"))

	return setupModel{step: stepAPIKey, apiKeyInput: apiKey, userInput: user, spinner: s}
}

func (m setupModel) Init() tea.Cmd { return textinput.Blink }

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			switch m.step {
			case stepAPIKey:
				if m.apiKeyInput.Value() == "" {
					return m, nil
				}
				m.step = stepUsername
				m.apiKeyInput.Blur()
				m.userInput.Focus()
				return m, textinput.Blink
			case stepUsername:
				if m.userInput.Value() == "" {
					return m, nil
				}
				m.step = stepValidating
				apiKey, username := m.apiKeyInput.Value(), m.userInput.Value()
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					client := newAPIClient(apiKey, username)
					if _, err := client.Board(); err != nil {
						return setupErrMsg{err: err}
					}
					cfg := &Config{APIKey: apiKey, Username: username}
					if err := saveConfig(cfg); err != nil {
						return setupErrMsg{err: fmt.Errorf("failed to save config: %w", err)}
					}
					return setupDoneMsg{config: cfg}
				})
			case stepError:
				m.step = stepAPIKey
				m.apiKeyInput.Focus()
				m.errMsg = ""
				return m, textinput.Blink
			}
		}

	case setupDoneMsg:
		m.done = true
		m.config = msg.config
		return m, nil

	case setupErrMsg:
		m.step = stepError
		m.errMsg = msg.err.Error()
		return m, nil

	case spinner.TickMsg:
		if m.step == stepValidating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	switch m.step {
	case stepAPIKey:
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	case stepUsername:
		m.userInput, cmd = m.userInput.Update(msg)
	}
	return m, cmd
}

func (m setupModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	sub := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	bad := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true)

	s := title.Render("ledger") + "\n"
	s += sub.Render("Set up your account.") + "\n\n"
	switch m.step {
	case stepAPIKey:
		s += "API Key:\n" + m.apiKeyInput.View() + "\n\n" + hint.Render("Press Enter to continue")
	case stepUsername:
		s += "API Key: ****\n\nUsername:\n" + m.userInput.View() + "\n\n" + hint.Render("Press Enter to continue")
	case stepValidating:
		s += "API Key: ****\nUsername: " + m.userInput.Value() + "\n\n" + m.spinner.View() + " Validating..."
	case stepError:
		s += bad.Render("Error: "+m.errMsg) + "\n\n" + hint.Render("Press Enter to try again")
	}
	return s
}
