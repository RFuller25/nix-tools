package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPostcardShowsTheWholeGarden(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gardener = "Rhys"
	if err := g.Plant(0, SpeciesByID("sunflower"), now); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Growth = 1
	g.Plots[0].Matured = true
	g.collect(SpeciesByID("sunflower"), now)

	card := renderPostcard(g, now, 90)

	for _, want := range []string{"Rhys's garden", "1 growing", "1 in flower", "1 species pressed"} {
		if !strings.Contains(card, want) {
			t.Errorf("the postcard does not mention %q:\n%s", want, card)
		}
	}
	// Every row of beds should be there.
	rows := strings.Count(card, "╭─")
	if rows != len(g.Plots) {
		t.Errorf("the postcard drew %d beds, want %d", rows, len(g.Plots))
	}
	for _, line := range strings.Split(card, "\n") {
		if w := lipgloss.Width(line); w > 90 {
			t.Errorf("a postcard line came out %d wide: %q", w, line)
		}
	}
}

func TestPostcardGrowsWithTheGarden(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Seeds = 100000
	for i := 0; i < 5; i++ {
		if err := g.BuyBed(now); err != nil {
			t.Fatal(err)
		}
	}
	card := renderPostcard(g, now, 90)
	if got := strings.Count(card, "╭─"); got != len(g.Plots) {
		t.Errorf("the postcard drew %d beds, want %d", got, len(g.Plots))
	}
}

func TestStatusLineSaysWhatNeedsDoing(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)

	if got := renderStatus(g, now); !strings.Contains(got, "0/15 growing") {
		t.Errorf("an empty garden reports %q", got)
	}

	if err := g.Plant(0, SpeciesByID("basil"), now); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Moisture = 0.1
	g.Plots[0].Weeds = 0.9
	if err := g.Plant(1, SpeciesByID("cosmos"), now); err != nil {
		t.Fatal(err)
	}
	g.Plots[1].Growth = 1
	g.Plots[1].Pods = 2

	got := renderStatus(g, now)
	for _, want := range []string{"2/15 growing", "1 in flower", "1 thirsty", "1 weedy", "1 ripe", "seeds"} {
		if !strings.Contains(got, want) {
			t.Errorf("the status line %q does not mention %q", got, want)
		}
	}
	if strings.Contains(got, "\n") {
		t.Error("the status line should be one line")
	}
}

func TestPostcardNeedsNoCursor(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	// renderPostcard sets the cursor outside the garden on purpose; make sure
	// that cannot panic or highlight anything.
	card := renderPostcard(g, now, 80)
	if strings.TrimSpace(card) == "" {
		t.Fatal("the postcard came out blank")
	}
}
