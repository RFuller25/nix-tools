package main

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type appState int

const (
	stateConfig appState = iota
	statePassword
	stateRunning
	stateDone
	stateError
)

type stats struct {
	phase        string  // "evaluating" | "fetching" | "building" | "activating"
	totalPaths   int
	fetchedPaths int
	totalMiB     float64
	totalDrvs    int
	builtDrvs    int
	pkgsChanged  int
	pkgsAdded    int
	pkgsRemoved  int
	diskDelta    string  // e.g. "+406.6MiB"
}

type model struct {
	state         appState
	pathInput     textinput.Model
	hostInput     textinput.Model
	passwordInput textinput.Model
	focusedInput  int
	viewport      viewport.Model
	lines         []string
	stats         stats
	reader        *bufio.Reader
	pr            *io.PipeReader
	exitCh        chan int
	exitCode      int
	err           error
	sudoErr       string
	width         int
	height        int
}

type lineMsg string
type cmdDoneMsg struct {
	exitCode int
	lastLine string
}
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

	pw := textinput.New()
	pw.Placeholder = "password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Width = 30

	return model{
		state:         stateConfig,
		pathInput:     path,
		hostInput:     host,
		passwordInput: pw,
		focusedInput:  0,
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

func (m *model) appendLine(line string) {
	clean := stripANSI(line)
	style, _ := categorizeLine(clean)
	updateStats(clean, &m.stats)
	m.lines = append(m.lines, style.Render(clean))
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	m.viewport.GotoBottom()
}

func readNextLine(r *bufio.Reader, pr *io.PipeReader, exitCh chan int) tea.Cmd {
	return func() tea.Msg {
		line, err := r.ReadString('\n')
		line = strings.TrimRight(line, "\n\r")
		if err == io.EOF {
			exitCode := <-exitCh
			pr.Close()
			return cmdDoneMsg{exitCode: exitCode, lastLine: line}
		}
		if err != nil {
			return errMsg{err}
		}
		return lineMsg(line)
	}
}
