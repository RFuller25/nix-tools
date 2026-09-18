package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// demoModel builds a garden mid-season with plants at every life stage.
func demoModel(t *testing.T, w, h int) model {
	t.Helper()
	now := testStart()
	g := newTestGarden(now)

	ids := []string{"sunflower", "foxglove", "lavender", "tomato", "japanesemaple",
		"aloe", "crocus", "venusflytrap", "strawberry", "bamboo"}
	for i, id := range ids {
		sp := SpeciesByID(id)
		if err := g.Plant(i, sp, now.Add(-time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("planting %s: %v", id, err)
		}
		g.Plots[i].Growth = float64(i%5) * 0.25
		if g.Plots[i].Growth >= 1 {
			g.Plots[i].Matured = true
			g.Plots[i].Pods = 2
		}
		g.Plots[i].Moisture = 1 - float64(i)/12
		g.Plots[i].Weeds = float64(i) / 14
	}
	g.Rename(0, "Big Yellow", now)

	m := newModel(g, "/tmp/garden.json", now)
	m.width, m.height = w, h
	return m
}

// Every screen must render without panicking, at a comfortable size and at a
// cramped one.
func TestScreensRender(t *testing.T) {
	sizes := [][2]int{{100, 40}, {80, 24}, {60, 20}, {40, 14}, {24, 10}}
	screens := []screen{screenGarden, screenShop, screenInfo, screenAlmanac, screenJournal, screenHelp}

	for _, size := range sizes {
		for _, s := range screens {
			m := demoModel(t, size[0], size[1])
			m.screen = s
			out := m.View()
			if strings.TrimSpace(out) == "" {
				t.Errorf("screen %d at %dx%d rendered nothing", s, size[0], size[1])
			}
		}
	}
}

// The grid must not spill past the terminal's width, or the layout tears.
func TestGardenFitsTheTerminal(t *testing.T) {
	for _, w := range []int{40, 60, 80, 100, 120} {
		m := demoModel(t, w, 40)
		m.ensureVisible()
		for _, line := range strings.Split(m.viewGarden(), "\n") {
			if got := lipgloss.Width(line); got > w {
				t.Errorf("at width %d a line came out %d wide: %q", w, got, line)
			}
		}
	}
}

// Every species must be drawable in a bed and on its info card.
func TestEverySpeciesRendersInABed(t *testing.T) {
	m := demoModel(t, 100, 40)
	for _, sp := range AllSpecies() {
		m.g.Plots[0] = Plot{SpeciesID: sp.ID, PlantedAt: m.now, Moisture: 0.5}
		for stage := 0; stage < StageCount; stage++ {
			m.g.Plots[0].Growth = []float64{0, 0.2, 0.5, 0.8, 1}[stage]
			cell := m.renderCell(0)
			lines := strings.Split(cell, "\n")
			if len(lines) != cellTall {
				t.Errorf("%s stage %s drew %d lines, want %d", sp.ID, stageNames[stage], len(lines), cellTall)
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != cellWidth {
					t.Errorf("%s stage %s drew a %d-wide line, want %d", sp.ID, stageNames[stage], got, cellWidth)
				}
			}
			m.screen = screenInfo
			if strings.TrimSpace(m.viewInfo()) == "" {
				t.Errorf("%s has an empty info card", sp.ID)
			}
		}
	}
}

// Keys should never take the program down, whatever screen is showing.
func TestKeyPressesAreSafe(t *testing.T) {
	keys := []string{"up", "down", "left", "right", "h", "j", "k", "l", "g", "G",
		"p", "w", "W", "c", "C", "f", "F", "n", "u", "i", "a", "s", "t",
		"enter", " ", "tab", "esc", "?", "pgup", "pgdown", "home", "end"}

	for _, s := range []screen{screenGarden, screenShop, screenInfo, screenAlmanac, screenJournal, screenHelp} {
		m := demoModel(t, 90, 30)
		m.screen = s
		var cur tea.Model = m
		for _, k := range keys {
			next, _ := cur.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			cur = next
			if strings.TrimSpace(cur.View()) == "" {
				t.Fatalf("screen %d rendered nothing after key %q", s, k)
			}
		}
	}
}

// A key that names a bed must never walk off the end of the garden.
func TestCursorStaysInBounds(t *testing.T) {
	m := demoModel(t, 90, 40)
	var cur tea.Model = m
	for i := 0; i < 400; i++ {
		key := []string{"up", "down", "left", "right", "home", "end"}[i%6]
		next, _ := cur.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		cur = next
		got := cur.(model)
		if got.cursor < 0 || got.cursor >= PlotCount {
			t.Fatalf("cursor escaped to %d after %q", got.cursor, key)
		}
	}
}

// Printed with -v, this is the quickest way to look at the garden.
func TestSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("snapshot is for eyeballing")
	}
	m := demoModel(t, 96, 40)
	for _, s := range []screen{screenGarden, screenShop, screenAlmanac} {
		m.screen = s
		m.ensureVisible()
		fmt.Printf("\n=== screen %d ===\n%s\n", s, m.View())
	}
}

// A session's worth of keystrokes should end up on disk: plant, name, water,
// then save and load it back.
func TestKeystrokesPersist(t *testing.T) {
	path := t.TempDir() + "/garden.json"
	m := demoModel(t, 90, 30)
	m.path = path
	m.cursor = 12 // an empty bed

	press := func(cur tea.Model, keys ...string) tea.Model {
		t.Helper()
		for _, k := range keys {
			var msg tea.KeyMsg
			switch k {
			case "enter":
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			next, _ := cur.Update(msg)
			cur = next
		}
		return cur
	}

	var cur tea.Model = m
	cur = press(cur, "p")                                   // seed shed
	cur = press(cur, "enter")                               // sow whatever is selected
	cur = press(cur, "n", "R", "o", "s", "i", "e", "enter") // name it
	cur = press(cur, "w")                                   // water it

	got := cur.(model)
	if err := Save(got.path, got.g); err != nil {
		t.Fatalf("saving: %v", err)
	}

	back, err := Load(path, got.now)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	p := back.Plots[got.cursor]
	if p.Empty() {
		t.Fatalf("bed %d is empty after planting through the keyboard", got.cursor+1)
	}
	if p.Name != "Rosie" {
		t.Errorf("plant came back named %q, want Rosie", p.Name)
	}
	if p.Moisture < 0.99 {
		t.Errorf("watering did not persist: moisture %.2f", p.Moisture)
	}
	if back.Seeds >= got.g.Seeds+1 {
		t.Error("planting did not cost any seeds")
	}
}
