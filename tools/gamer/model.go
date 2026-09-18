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

	audio     *Audio
	musicSeed int64

	screen  screen
	status  string
	saveErr error

	width  int
	height int
}

func newModel(scores *Scores, path string, now time.Time) model {
	m := model{
		games: []game{
			newTetris(now.UnixNano()),
			new2048(now.UnixNano()),
			newSnake(now.UnixNano()),
			newHue(now.UnixNano()),
			newMines(now.UnixNano()),
		},
		scores:    scores,
		path:      path,
		now:       now,
		audio:     NewAudio(sampleRate),
		musicSeed: now.UnixNano(),
		width:     80,
		height:    30,
	}
	m.audio.SetMuted(scores.Muted)
	for _, g := range m.games {
		g.SetAudio(m.audio)
	}
	return m
}

// setMusic puts the theme on at the given tempo: unhurried in the menu,
// brisker in a game. Muted or with no player about, this does nothing.
func (m *model) setMusic(bpm float64) {
	m.audio.SetMusic(chiptune(sampleRate, m.musicSeed, bpm))
}

// toggleMute switches all sound off or back on and remembers the choice.
func (m *model) toggleMute() {
	muted := !m.audio.Muted()
	m.audio.SetMuted(muted)
	m.scores.Muted = muted

	switch {
	case muted:
		m.status = "Sound off."
	case !m.audio.Available():
		m.status = playerHint()
	default:
		m.status = "Sound on, through " + m.audio.Backend() + "."
		if m.screen == screenPlay {
			m.setMusic(gameBPM)
		} else {
			m.setMusic(menuBPM)
		}
	}
	if err := Save(m.path, m.scores); err != nil {
		m.saveErr = err
	}
}

func (m *model) Init() tea.Cmd {
	if m.active != nil {
		m.setMusic(gameBPM)
		return tea.Batch(tea.ClearScreen, m.active.Start())
	}
	m.setMusic(menuBPM)
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
	if key == "m" {
		m.toggleMute()
		return m, nil
	}

	if m.screen == screenMenu {
		switch key {
		case "q", "esc":
			return m, m.quit()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.audio.Play(sfxMove()...)
			}
		case "down", "j":
			if m.cursor < len(m.games)-1 {
				m.cursor++
				m.audio.Play(sfxMove()...)
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.games) - 1
		case "enter", " ":
			m.active = m.games[m.cursor]
			m.screen = screenPlay
			m.status = ""
			m.audio.Play(sfxSelect()...)
			m.setMusic(gameBPM)
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
		m.setMusic(menuBPM)
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
	m.audio.Close()
	if err := Save(m.path, m.scores); err != nil {
		m.saveErr = err
	}
	return tea.Quit
}

// soundLine describes the state of the sound, for the menu.
func (m *model) soundLine() string {
	switch {
	case m.audio.Muted():
		return subtleStyle.Render("sound muted · m to unmute")
	case !m.audio.Available():
		return subtleStyle.Render("silent · " + playerHint())
	default:
		return subtleStyle.Render("sound on via " + m.audio.Backend() + " · m to mute")
	}
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
	b.WriteString(m.soundLine() + "\n")
	b.WriteString(helpStyle.Render("↑↓ choose · enter play · m mute · q quit"))
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

	sound := " · m mute"
	if m.audio.Muted() {
		sound = " · m unmute"
	}
	help := g.Help() + subtleStyle.Render(sound+" · esc menu")
	return strings.Join([]string{head, body, helpStyle.Render(help)}, "\n")
}
