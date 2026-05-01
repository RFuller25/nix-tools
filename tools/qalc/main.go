package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type historyEntry struct {
	expr   string
	result string
	isErr  bool
}

type model struct {
	input      textinput.Model
	viewport   viewport.Model
	history    []historyEntry
	lastResult string
	width      int
	height     int
	ready      bool
}

var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	resultStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	exprStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "2 + 2  (type ANSWER to use last result)"
	ti.Focus()
	ti.Prompt = promptStyle.Render("› ")

	return model{input: ti}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputHeight := 3
		vpHeight := msg.Height - inputHeight - 3
		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.SetContent(m.renderHistory())
			m.viewport.GotoBottom()
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			raw := strings.TrimSpace(m.input.Value())
			if raw == "" {
				return m, nil
			}
			expr := substituteAnswer(raw, m.lastResult)
			m.input.SetValue("")
			return m, evalExpr(expr)
		}

	case resultMsg:
		m.lastResult = msg.result
		m.history = append(m.history, historyEntry{
			expr:   msg.expr,
			result: msg.result,
		})
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
		return m, nil

	case calcErrMsg:
		m.history = append(m.history, historyEntry{
			expr:   msg.expr,
			result: msg.err,
			isErr:  true,
		})
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) renderHistory() string {
	if len(m.history) == 0 {
		return helpStyle.Render("No calculations yet.")
	}
	var b strings.Builder
	for _, h := range m.history {
		b.WriteString(exprStyle.Render(h.expr) + "\n")
		if h.isErr {
			b.WriteString(errStyle.Render("  error: "+h.result) + "\n")
		} else {
			b.WriteString(resultStyle.Render("  = "+h.result) + "\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("qalc") + "  " +
		helpStyle.Render("ctrl+c: quit") + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(m.viewport.View() + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(m.input.View() + "\n")
	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
