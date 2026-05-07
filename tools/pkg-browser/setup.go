package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type setupModel struct {
	input      textinput.Model
	errMsg     string
	configPath string // set on successful confirm
}

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Placeholder = "/etc/nixos/configuration.nix"
	ti.SetValue("/etc/nixos/configuration.nix")
	ti.Focus()
	ti.Width = 60
	return setupModel{input: ti}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			path := strings.TrimSpace(m.input.Value())
			if path == "" {
				m.errMsg = "path cannot be empty"
				return m, nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				m.errMsg = fmt.Sprintf("cannot read file: %v", err)
				return m, nil
			}
			if !strings.Contains(string(data), "environment.systemPackages") {
				m.errMsg = "file does not contain environment.systemPackages"
				return m, nil
			}
			m.configPath = path
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m setupModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("pkg-browser — first run setup"))
	b.WriteString("\n\n")
	b.WriteString("NixOS config path:\n")
	b.WriteString(m.input.View() + "\n")
	if m.errMsg != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.errMsg) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("enter: confirm  ctrl+c: quit"))
	return b.String()
}
