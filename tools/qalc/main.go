package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ─────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	answerChipStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	divStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	exprStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	autoTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	errResultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// ── model ──────────────────────────────────────────────────────────────────

type historyEntry struct {
	displayExpr string
	display     string
	isAuto      bool
	isErr       bool
}

type model struct {
	input          textinput.Model
	viewport       viewport.Model
	history        []historyEntry
	lastResult     string
	debounceID     int
	preview        string
	previewLoading bool
	width          int
	height         int
	ready          bool
}

// fixedLines = header + top divider + bottom divider + input + preview + help
const fixedLines = 6

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "2 + 2  •  / 4  •  100 km to miles"
	ti.Focus()
	ti.Prompt = promptStyle.Render("› ")

	return model{input: ti}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// ── update ─────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - fixedLines
		if vpHeight < 1 {
			vpHeight = 1
		}
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
			m.input.SetValue("")
			m.preview = ""
			m.previewLoading = false
			m.debounceID++
			return m, evalExpr(raw, m.lastResult)

		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			m.previewLoading = true
			m.debounceID++
			id := m.debounceID
			debounceCmd := tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return debounceMsg{id: id}
			})
			return m, tea.Batch(inputCmd, debounceCmd)
		}

	case debounceMsg:
		if msg.id != m.debounceID {
			return m, nil
		}
		raw := strings.TrimSpace(m.input.Value())
		if raw == "" {
			m.preview = ""
			m.previewLoading = false
			return m, nil
		}
		return m, evalPreview(raw, m.lastResult, msg.id)

	case previewMsg:
		if msg.id != m.debounceID {
			return m, nil
		}
		m.preview = msg.result
		m.previewLoading = false
		return m, nil

	case resultMsg:
		m.lastResult = msg.answer
		m.history = append(m.history, historyEntry{
			displayExpr: msg.displayExpr,
			display:     msg.display,
			isAuto:      msg.isAuto,
		})
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
		return m, nil

	case calcErrMsg:
		m.history = append(m.history, historyEntry{
			displayExpr: msg.displayExpr,
			display:     msg.err,
			isErr:       true,
		})
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// ── view ───────────────────────────────────────────────────────────────────

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	div := divStyle.Render(strings.Repeat("─", m.width))

	// Header: title + ANSWER chip (when set) + right-aligned help
	header := titleStyle.Render("qalc")
	if m.lastResult != "" {
		header += "  " + answerChipStyle.Render("ANSWER: "+m.lastResult)
	}
	rightHelp := helpStyle.Render("ctrl+c: quit")
	gap := m.width - lipgloss.Width(header) - lipgloss.Width(rightHelp)
	if gap > 0 {
		header += strings.Repeat(" ", gap) + rightHelp
	}

	// Preview line
	var previewLine string
	switch {
	case m.previewLoading:
		previewLine = previewStyle.Render("  …")
	case m.preview != "":
		previewLine = previewStyle.Render("  ≈ " + m.preview)
	}

	help := helpStyle.Render("↑↓: scroll  enter: evaluate  ctrl+c: quit")

	var b strings.Builder
	b.WriteString(header + "\n")
	b.WriteString(div + "\n")
	b.WriteString(m.viewport.View() + "\n")
	b.WriteString(div + "\n")
	b.WriteString(m.input.View() + "\n")
	b.WriteString(previewLine + "\n")
	b.WriteString(help)
	return b.String()
}

func (m model) renderHistory() string {
	if len(m.history) == 0 {
		return helpStyle.Render("No calculations yet. Type an expression and press Enter.")
	}
	var b strings.Builder
	for i, h := range m.history {
		if i > 0 {
			b.WriteString("\n")
		}
		exprLine := exprStyle.Render(h.displayExpr)
		if h.isAuto {
			exprLine += "  " + autoTagStyle.Render("← ANSWER")
		}
		b.WriteString(exprLine + "\n")
		if h.isErr {
			b.WriteString(errResultStyle.Render("  error: "+h.display) + "\n")
		} else {
			b.WriteString("  " + colorizeResult(h.display) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// ── main ───────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
