package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	fps      = 24
	maxW     = 200
	maxH     = 60
	minW     = 60
	minH     = 20
	helpText = "space pause  ·  r restart  ·  m mute  ·  q quit"
)

type tickMsg time.Time

type model struct {
	profile    termenv.Profile
	termW      int
	termH      int
	sc         *scene
	t          float64 // film time in seconds
	last       time.Time
	paused     bool
	sceneW     int
	sceneH     int
	showHelp   bool
	helpUntilT float64
	audio      *Audio
}

func newModel(p termenv.Profile) model {
	a := NewAudio(sampleRate)
	return model{profile: p, showHelp: true, helpUntilT: 6, audio: a}
}

func (m model) Init() tea.Cmd {
	m.audio.SetMusic(theme(sampleRate))
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) resize(w, h int) {
	m.termW, m.termH = w, h
	sw, sh := min(w, maxW), min(h-1, maxH)
	if sw < minW || sh < minH {
		m.sc, m.sceneW, m.sceneH = nil, 0, 0
		return
	}
	if m.sc == nil || sw != m.sceneW || sh != m.sceneH {
		m.sc = newScene(sw, sh)
		m.sceneW, m.sceneH = sw, sh
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.audio.Close()
			return m, tea.Quit
		case "m":
			m.audio.SetMuted(!m.audio.Muted())
		case " ", "p":
			m.paused = !m.paused
			m.last = time.Time{}
		case "r":
			m.t = 0
		case "left":
			m.t = max(0, m.t-5)
		case "right":
			m.t += 5
		case "?":
			m.showHelp = !m.showHelp
		}
	case tickMsg:
		now := time.Time(msg)
		if !m.paused {
			if !m.last.IsZero() {
				dt := now.Sub(m.last).Seconds()
				if dt > 0.25 {
					dt = 0.25
				}
				m.t += dt
			}
			m.last = now
		}
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	if m.termW == 0 {
		return ""
	}
	if m.sc == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9a8fc8")).
			Render("the penguin needs a bigger terminal (at least 60x21).")
	}
	body := m.sc.draw(m.t).render(m.profile)
	padL := strings.Repeat(" ", (m.termW-m.sceneW)/2)
	if padL != "" {
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = padL + lines[i]
		}
		body = strings.Join(lines, "\n")
	}
	help := ""
	if m.showHelp && m.t < m.helpUntilT || m.paused {
		help = helpText
		if m.paused {
			help = "paused  ·  " + helpText
		}
	}
	foot := lipgloss.NewStyle().Foreground(lipgloss.Color("#6d6a8f")).Render(help)
	return body + "\n" + padL + foot
}
