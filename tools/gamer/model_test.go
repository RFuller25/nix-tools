package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func testModel(t *testing.T) *model {
	t.Helper()
	m := newModel(newScores(), filepath.Join(t.TempDir(), "scores.json"), time.Now())
	m.width, m.height = 100, 40
	return &m
}

func TestMenuListsEveryGame(t *testing.T) {
	m := testModel(t)
	if len(m.games) != 5 {
		t.Fatalf("the arcade holds %d games, want 5", len(m.games))
	}
	ids := map[string]bool{}
	view := m.View()
	for _, g := range m.games {
		if ids[g.ID()] {
			t.Errorf("duplicate game id %q", g.ID())
		}
		ids[g.ID()] = true
		if g.Name() == "" || g.Blurb() == "" || g.ScoreLabel() == "" {
			t.Errorf("%s is missing its menu text", g.ID())
		}
		if !strings.Contains(view, g.Name()) {
			t.Errorf("%s is missing from the menu", g.Name())
		}
	}
	for _, want := range []string{"tetris", "2048", "snake", "hue", "mines"} {
		if !ids[want] {
			t.Errorf("no game with id %q", want)
		}
	}
}

func TestMenuNavigationAndLaunch(t *testing.T) {
	m := testModel(t)

	m.Update(key("up")) // already at the top
	if m.cursor != 0 {
		t.Errorf("cursor went above the first entry: %d", m.cursor)
	}
	for i := 0; i < 20; i++ {
		m.Update(key("down"))
	}
	if m.cursor != len(m.games)-1 {
		t.Errorf("cursor ran past the last entry: %d", m.cursor)
	}

	m.Update(key("enter"))
	if m.screen != screenPlay || m.active == nil {
		t.Fatal("enter did not start a game")
	}
	if m.active != m.games[m.cursor] {
		t.Error("the wrong game started")
	}

	m.Update(key("esc"))
	if m.screen != screenMenu || m.active != nil {
		t.Error("escape did not return to the menu")
	}
}

func TestStartByID(t *testing.T) {
	m := testModel(t)
	if !m.startByID("snake") {
		t.Fatal("--play snake was refused")
	}
	if m.active.ID() != "snake" {
		t.Errorf("started %q", m.active.ID())
	}
	if m.startByID("backgammon") {
		t.Error("an unknown game was accepted")
	}
}

func TestFinishedGameIsRecordedOnce(t *testing.T) {
	m := testModel(t)
	m.startByID("2048")
	g := m.active.(*g2048)
	g.Start()
	g.score = 500
	g.over = true

	m.harvest()
	if got := m.scores.Best["2048"]; got != 500 {
		t.Errorf("best score = %d, want 500", got)
	}
	if got := m.scores.Played["2048"]; got != 1 {
		t.Errorf("played = %d, want 1", got)
	}

	m.harvest() // the same finished game again
	if got := m.scores.Played["2048"]; got != 1 {
		t.Errorf("the same game was counted twice: played = %d", got)
	}
	if m.saveErr != nil {
		t.Errorf("saving the score failed: %v", m.saveErr)
	}

	back, err := LoadScores(m.path)
	if err != nil {
		t.Fatalf("reloading scores: %v", err)
	}
	if back.Best["2048"] != 500 {
		t.Errorf("score did not reach the disk: %v", back.Best)
	}
}

// Every screen must draw without panicking, at generous and cramped sizes.
func TestScreensRender(t *testing.T) {
	for _, size := range [][2]int{{120, 45}, {100, 40}, {80, 24}, {60, 20}, {40, 15}} {
		m := testModel(t)
		m.width, m.height = size[0], size[1]
		if strings.TrimSpace(m.View()) == "" {
			t.Errorf("the menu drew nothing at %dx%d", size[0], size[1])
		}
		for _, g := range m.games {
			m.active = g
			m.screen = screenPlay
			g.Start()
			if strings.TrimSpace(m.View()) == "" {
				t.Errorf("%s drew nothing at %dx%d", g.ID(), size[0], size[1])
			}
			if strings.TrimSpace(g.Help()) == "" {
				t.Errorf("%s has no help line", g.ID())
			}
		}
	}
}

// Boards should not be wider than a standard terminal.
func TestBoardsFitEightyColumns(t *testing.T) {
	m := testModel(t)
	for _, g := range m.games {
		g.Start()
		for _, line := range strings.Split(g.View(80, 30), "\n") {
			if w := lipgloss.Width(line); w > 80 {
				t.Errorf("%s drew a %d-wide line", g.ID(), w)
			}
		}
	}
}

// Hammering every key in every game must never panic or wedge the shell.
func TestKeyPressesAreSafe(t *testing.T) {
	keys := []string{"up", "down", "left", "right", "h", "j", "k", "l", "w", "a", "s", "d",
		"x", "z", "c", "f", "p", "r", "n", " ", "enter", "g", "G", "home", "end", "?"}

	for _, g := range newModel(newScores(), "", time.Now()).games {
		m := testModel(t)
		m.startByID(g.ID())
		m.active.Start()
		for round := 0; round < 3; round++ {
			for _, k := range keys {
				m.Update(key(k))
				if strings.TrimSpace(m.View()) == "" {
					t.Fatalf("%s drew nothing after key %q", g.ID(), k)
				}
			}
		}
	}
}

// A tick that belongs to a game the player has left must not be acted on.
func TestTicksFromOtherRoundsAreIgnored(t *testing.T) {
	m := testModel(t)
	m.startByID("tetris")
	g := m.active.(*tetris)
	g.Start()

	m.Update(key("esc")) // back to the menu
	var msg tea.Msg = tetrisTickMsg{g.gen}
	y := g.y
	m.Update(msg)
	if g.y != y {
		t.Error("a tick moved a piece in a game that is no longer on screen")
	}
}
