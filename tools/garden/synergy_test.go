package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func plantedWith(t *testing.T, g *Garden, bed int, id string, gn Genome, growth float64) {
	t.Helper()
	sp := SpeciesByID(id)
	g.Gold = 1000
	if err := g.Plant(bed, sp, 0, testStart()); err != nil {
		t.Fatal(err)
	}
	g.Plots[bed].Genome = gn
	g.Plots[bed].Growth = growth
}

func TestATallPlantShadesASunLover(t *testing.T) {
	g := newTestGarden(testStart())
	giant := SpeciesByID("sunflower").VarietyGenome(0)
	giant.Height = 100
	short := SpeciesByID("zinnia").VarietyGenome(0)
	short.Height = 10
	plantedWith(t, g, 0, "sunflower", giant, 1)
	plantedWith(t, g, 1, "zinnia", short, 0.3)

	effects := g.companionEffects(1, SpeciesByID("zinnia"))
	found := false
	for _, e := range effects {
		if e.Delta == shadeDelta && e.Bed == 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("a giant sunflower does not shade the zinnia beside it: %+v", effects)
	}
	if f := g.companionFactor(1, SpeciesByID("zinnia")); f >= 1 {
		t.Errorf("the shaded zinnia grows at %.3f", f)
	}

	// Breed the sunflower short and the zinnia tall and the shade goes.
	g.Plots[0].Genome.Height = 0
	g.Plots[1].Genome.Height = 100
	for _, e := range g.companionEffects(1, SpeciesByID("zinnia")) {
		if e.Delta == shadeDelta {
			t.Error("a giant zinnia is still shaded by the sunflower")
		}
	}
}

func TestShadeLoversLikeATallNeighbour(t *testing.T) {
	g := newTestGarden(testStart())
	giant := SpeciesByID("sunflower").VarietyGenome(0)
	giant.Height = 100
	plantedWith(t, g, 0, "sunflower", giant, 1)
	plantedWith(t, g, 1, "hosta", SpeciesByID("hosta").VarietyGenome(0), 0.3)
	if !shadeLover(SpeciesByID("hosta")) {
		t.Fatal("hosta should be a shade-lover")
	}
	good := false
	for _, e := range g.companionEffects(1, SpeciesByID("hosta")) {
		if e.Delta == shelterDelta {
			good = true
		}
	}
	if !good {
		t.Error("a hosta is not sheltered by a giant sunflower")
	}
}

func TestBushyAndGreedyNeighboursCost(t *testing.T) {
	g := newTestGarden(testStart())
	bush := SpeciesByID("cosmos").VarietyGenome(0)
	bush.Shape, bush.Yield, bush.Height = 95, 95, 60
	plain := SpeciesByID("zinnia").VarietyGenome(0)
	plain.Height = 50
	plantedWith(t, g, 0, "cosmos", bush, 1)
	plantedWith(t, g, 1, "zinnia", plain, 0.6)
	var crowd, thirst bool
	for _, e := range g.companionEffects(1, SpeciesByID("zinnia")) {
		crowd = crowd || e.Delta == crowdDelta
		thirst = thirst || e.Delta == thirstyDelta
	}
	if !crowd || !thirst {
		t.Errorf("crowding %v, thirst %v from a bushy heavy cropper", crowd, thirst)
	}
}

func TestSynergyAddsWhatIsGivenAndTaken(t *testing.T) {
	g := newTestGarden(testStart())
	plantedWith(t, g, 0, "marigold", SpeciesByID("marigold").VarietyGenome(0), 1)
	plantedWith(t, g, 1, "tomato", SpeciesByID("tomato").VarietyGenome(0), 0.5)

	tomato := g.Synergy(1)
	if tomato.Take < 0.11 {
		t.Errorf("the tomato takes only %+.2f from a marigold", tomato.Take)
	}
	marigold := g.Synergy(0)
	if marigold.Give < 0.11 {
		t.Errorf("the marigold gives only %+.2f to the tomato", marigold.Give)
	}
	if got := marigold.Net(); got != marigold.Take+marigold.Give {
		t.Error("net is not the sum")
	}
	if g.Synergy(14) != (Synergy{}) {
		t.Error("an empty bed has a synergy")
	}
	if SynergyWord(0.2) == SynergyWord(-0.2) {
		t.Error("good and bad get the same word")
	}
}

func TestOverlayToggleAndColours(t *testing.T) {
	g := newTestGarden(testStart())
	plantedWith(t, g, 0, "marigold", SpeciesByID("marigold").VarietyGenome(0), 1)
	plantedWith(t, g, 1, "tomato", SpeciesByID("tomato").VarietyGenome(0), 0.5)
	m := newModel(g, "/tmp/g.json", testStart())
	m.width, m.height = 100, 40

	plain := m.View()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	on := next.(model)
	if !on.overlay {
		t.Fatal("g did not turn the overlay on")
	}
	if v := on.View(); v == plain || !strings.Contains(v, "growth") {
		t.Error("the overlay changed nothing on screen")
	}
	next, _ = on.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if next.(model).overlay {
		t.Error("g again did not turn it off")
	}
	// Home still goes home, now that g is the overlay.
	m.cursor = 7
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("home")})
	if next.(model).cursor != 0 {
		t.Error("home no longer goes to the first bed")
	}
}
