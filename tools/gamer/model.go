package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenMenu screen = iota
	screenPlay
)

type model struct {
	games  []game
	cursor int
	active game

	scores *Scores
	path   string
	now    time.Time

	screen  screen
	status  string
	saveErr error

	width  int
	height int
}

func newModel(scores *Scores, path string, now time.Time) model {
	return model{
		games: []game{
			newTetris(now.UnixNano()),
			new2048(now.UnixNano()),
			newSnake(now.UnixNano()),
			newHue(now.UnixNano()),
			newMines(now.UnixNano()),
		},
		scores: scores,
		path:   path,
		now:    now,
		width:  80,
		height: 30,
	}
}

func (m *model) Init() tea.Cmd {
	if m.active != nil {
		return tea.Batch(tea.ClearScreen, m.active.Start())
	}
	return tea.ClearScreen
}

// startByID launches a game by its key, for the --play flag.
func (m *model) startByID(id string) bool {
	for i, g := range m.games {
		if g.ID() == id {
			m.cursor = i
			m.active = g
			m.screen = screenPlay
			return true
		}
	}
	return false
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.screen == screenPlay && m.active != nil {
		cmd := m.active.Update(msg)
		m.harvest()
		return m, cmd
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, m.quit()
	}

	if m.screen == screenMenu {
		switch key {
		case "q", "esc":
			return m, m.quit()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.games)-1 {
				m.cursor++
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.games) - 1
		case "enter", " ":
			m.active = m.games[m.cursor]
			m.screen = screenPlay
			m.status = ""
			return m, m.active.Start()
		}
		return m, nil
	}

	// In a game: escape hands back to the menu, everything else belongs to
	// the game itself.
	switch key {
	case "esc", "Q":
		m.screen = screenMenu
		m.active = nil
		return m, tea.ClearScreen
	}

	cmd := m.active.Update(msg)
	m.harvest()
	return m, cmd
}

// harvest records a finished round exactly once.
func (m *model) harvest() {
	if m.active == nil || !m.active.Over() {
		return
	}
	score, lower, record := m.active.Result()
	if !record {
		return
	}
	if m.scores.Record(m.active.ID(), score, lower, time.Now()) {
		m.status = fmt.Sprintf("New best at %s: %d %s", m.active.Name(), score, m.active.ScoreLabel())
	}
	if err := Save(m.path, m.scores); err != nil {
		m.saveErr = err
	}
}

func (m *model) quit() tea.Cmd {
	if err := Save(m.path, m.scores); err != nil {
		m.saveErr = err
	}
	return tea.Quit
}

func (m *model) View() string {
	if m.screen == screenPlay && m.active != nil {
		return m.playView()
	}
	return m.menuView()
}

func (m *model) menuView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("▰ gamer"))
	b.WriteString(subtleStyle.Render("   five small games for a terminal"))
	b.WriteString("\n" + dividerStyle.Render(strings.Repeat("─", max(1, min(m.width, 72)))) + "\n\n")

	for i, g := range m.games {
		marker := "  "
		name := valueStyle.Render(pad(g.Name(), 12))
		if i == m.cursor {
			marker = selectedStyle.Render("▸ ")
			name = selectedStyle.Render(pad(g.Name(), 12))
		}
		line := marker + name + subtleStyle.Render(g.Blurb())
		b.WriteString(line + "\n")

		best, seen := m.scores.Best[g.ID()]
		detail := "    " + subtleStyle.Render("not played yet")
		if seen {
			detail = "    " + labelStyle.Render("best ") + accentStyle.Render(fmt.Sprintf("%d", best)) +
				subtleStyle.Render(" "+g.ScoreLabel()+fmt.Sprintf("  ·  %d played", m.scores.Played[g.ID()]))
		}
		b.WriteString(detail + "\n\n")
	}

	if m.status != "" {
		b.WriteString(okStyle.Render(m.status) + "\n")
	}
	if m.saveErr != nil {
		b.WriteString(warnStyle.Render("scores not saved: "+m.saveErr.Error()) + "\n")
	}
	b.WriteString(helpStyle.Render("↑↓ choose · enter play · q quit"))
	return b.String()
}

func (m *model) playView() string {
	g := m.active
	head := titleStyle.Render(g.Name())
	if m.status != "" {
		head += subtleStyle.Render("   " + m.status)
	}

	bodyHeight := max(4, m.height-4)
	body := g.View(m.width, bodyHeight)

	help := g.Help() + subtleStyle.Render(" · esc menu")
	return strings.Join([]string{head, body, helpStyle.Render(help)}, "\n")
}
