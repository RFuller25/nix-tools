package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The rule every screen has to keep: never taller than the window and never
// wider. A view that overflows does not merely look wrong — the terminal
// scrolls, and the header and the first row of beds are pushed out of reach.
func TestEveryScreenFitsTheTerminal(t *testing.T) {
	sizes := [][2]int{
		{80, 24}, // the classic
		{80, 30},
		{100, 30},
		{120, 40},
		{70, 20},
		{60, 18},
		{50, 15},
		{40, 12},
		{200, 60},
	}
	screens := map[string]screen{
		"garden": screenGarden, "shed": screenShop, "info": screenInfo,
		"almanac": screenAlmanac, "journal": screenJournal, "help": screenHelp,
	}

	for _, size := range sizes {
		for name, s := range screens {
			m := demoModel(t, size[0], size[1])
			m.screen = s
			m.ensureVisible()
			view := m.View()

			lines := strings.Split(view, "\n")
			if len(lines) > size[1] {
				t.Errorf("%s at %dx%d is %d lines tall", name, size[0], size[1], len(lines))
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w > size[0] {
					t.Errorf("%s at %dx%d: line %d is %d wide: %q", name, size[0], size[1], i+1, w, line)
				}
			}
		}
	}
}

// Whatever is on screen, growing the window must never lose the footer and
// shrinking it must never hide the header.
func TestHeaderAndFooterSurviveEverySize(t *testing.T) {
	for _, height := range []int{12, 16, 20, 24, 30, 40} {
		m := demoModel(t, 90, height)
		m.screen = screenGarden
		m.ensureVisible()
		lines := strings.Split(m.View(), "\n")

		if !strings.Contains(lines[0], "garden") {
			t.Errorf("at height %d the header is missing: %q", height, lines[0])
		}
		if last := lines[len(lines)-1]; strings.TrimSpace(last) == "" {
			t.Errorf("at height %d the footer came out blank", height)
		}
	}
}

// The info card is taller than a small window, so it has to scroll rather
// than run off the bottom.
func TestInfoCardScrolls(t *testing.T) {
	m := demoModel(t, 90, 24)
	m.screen = screenInfo
	m.cursor = 0

	top := m.View()
	if !strings.Contains(top, "scroll") {
		t.Error("a clipped info card does not offer to scroll")
	}

	var cur tea.Model = m
	for i := 0; i < 8; i++ {
		next, _ := cur.Update(tea.KeyMsg{Type: tea.KeyDown})
		cur = next
	}
	scrolled := cur.(model)
	if scrolled.cardScroll == 0 {
		t.Fatal("the card did not scroll")
	}
	if scrolled.View() == top {
		t.Error("scrolling the card changed nothing on screen")
	}
	if lines := strings.Split(scrolled.View(), "\n"); len(lines) > 24 {
		t.Errorf("the scrolled card is %d lines tall", len(lines))
	}

	// Left and right still move between beds, and reset the scroll.
	next, _ := cur.Update(tea.KeyMsg{Type: tea.KeyRight})
	moved := next.(model)
	if moved.cursor == scrolled.cursor {
		t.Error("right did not move to the next bed")
	}
	if moved.cardScroll != 0 {
		t.Error("moving to another bed kept the old scroll position")
	}
}

func TestHelpScrollsAndThenCloses(t *testing.T) {
	m := demoModel(t, 90, 20)
	m.screen = screenHelp

	var cur tea.Model = m
	next, _ := cur.Update(tea.KeyMsg{Type: tea.KeyDown})
	scrolled := next.(model)
	if scrolled.cardScroll == 0 {
		t.Fatal("down did not scroll the help")
	}
	if scrolled.screen != screenHelp {
		t.Error("scrolling closed the help screen")
	}

	next, _ = scrolled.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if next.(model).screen != screenGarden {
		t.Error("any other key should close the help screen")
	}
}

// Windowing itself: the arithmetic the screens rely on.
func TestWindow(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	got, above, below := window(lines, 0, 3)
	if strings.Join(got, "") != "abc" || above || !below {
		t.Errorf("top of the list = %v, above=%v below=%v", got, above, below)
	}

	got, above, below = window(lines, 2, 3)
	if strings.Join(got, "") != "cde" || !above || below {
		t.Errorf("bottom of the list = %v, above=%v below=%v", got, above, below)
	}

	// Scrolling past the end settles on the last screenful.
	got, _, below = window(lines, 99, 3)
	if strings.Join(got, "") != "cde" || below {
		t.Errorf("overscrolled to %v", got)
	}

	// A window bigger than the content shows all of it and nothing more.
	got, above, below = window(lines, 0, 50)
	if len(got) != 5 || above || below {
		t.Errorf("a roomy window gave %v", got)
	}
}

func TestClipHeight(t *testing.T) {
	view := "one\ntwo\nthree\nfour"
	if got := clipHeight(view, 2); got != "one\ntwo" {
		t.Errorf("clipHeight = %q", got)
	}
	if got := clipHeight(view, 10); got != view {
		t.Error("a view that already fits should be left alone")
	}
	if got := clipHeight(view, 0); got != view {
		t.Error("an unknown height should be left alone")
	}
}

// A window that changes size mid-session must settle immediately.
func TestResizingIsHandled(t *testing.T) {
	m := demoModel(t, 120, 40)
	var cur tea.Model = m
	for _, size := range [][2]int{{80, 24}, {200, 50}, {45, 14}, {100, 30}} {
		next, _ := cur.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		cur = next
		for _, s := range []screen{screenGarden, screenInfo, screenAlmanac, screenShop, screenJournal, screenHelp} {
			got := cur.(model)
			got.screen = s
			got.ensureVisible()
			lines := strings.Split(got.View(), "\n")
			if len(lines) > size[1] {
				t.Errorf("screen %d is %d lines tall in a %dx%d window", s, len(lines), size[0], size[1])
			}
		}
	}
}

func TestSnapshotFits(t *testing.T) {
	// The postcard has no window to fit, but it must still respect its width.
	g := newTestGarden(time.Now())
	card := renderPostcard(g, time.Now(), 72)
	for _, line := range strings.Split(card, "\n") {
		if w := lipgloss.Width(line); w > 72 {
			t.Errorf("a postcard line is %d wide: %q", w, line)
		}
	}
}

func TestUndatedPlantsReadAsNew(t *testing.T) {
	now := testStart()
	p := &Plot{SpeciesID: "basil", Growth: 0.5} // no planting date at all
	if got := p.Age(now); got != 0 {
		t.Errorf("an undated plant is %v old", got)
	}
	if got := plantedWhen(p); got == "" || strings.Contains(got, "0001") {
		t.Errorf("an undated plant says it was planted %q", got)
	}
}
