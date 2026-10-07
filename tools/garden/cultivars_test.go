package main

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// growToFlower plants a genome in a bed and advances until it flowers.
func growToFlower(t *testing.T, g *Garden, bed int, sp *Species, gn Genome, streak int, now time.Time) time.Time {
	t.Helper()
	g.Plots[bed] = Plot{PH: g.Plots[bed].PH, Richness: g.Plots[bed].Richness}
	g.sow(bed, sp, gn, 1, now)
	g.Plots[bed].Streak = streak
	at := now
	for i := 0; i < 24*4*3 && !g.Plots[bed].Matured; i++ {
		at = at.Add(15 * time.Minute)
		g.Advance(at)
		g.Water(bed, at)
	}
	if !g.Plots[bed].Matured {
		t.Fatal("the plant never flowered")
	}
	return at
}

func oddCosmos() Genome {
	g := SpeciesByID("cosmos").VarietyGenome(0)
	g.Hue, g.Sat, g.Light = 150, 85, 45 // a green cosmos, which no named form is
	g.Height, g.Yield = 90, 80
	return g
}

func TestANewHybridIsDiscovered(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")

	at := growToFlower(t, g, 0, sp, oddCosmos(), 0, now)
	if len(g.Cultivars) != 1 {
		t.Fatalf("%d cultivars after a hybrid flowered", len(g.Cultivars))
	}
	c := g.Cultivars[0]
	if c.Species != "cosmos" || c.Genome != oddCosmos() || c.Stable {
		t.Errorf("cultivar recorded as %+v", c)
	}
	if g.Plots[0].Line != c.ID || g.Plots[0].LineName != c.Name {
		t.Error("the plant was not filed into its line")
	}
	found := false
	for _, e := range g.Journal {
		if e.Text != "" && len(e.Text) > 10 && e.Text[:5] == "A new" {
			found = true
		}
	}
	if !found {
		t.Error("the discovery was not written in the journal")
	}

	// A second plant of the same genes joins the line rather than founding one.
	growToFlower(t, g, 1, sp, oddCosmos(), 0, at)
	if len(g.Cultivars) != 1 || g.Plots[1].Line != c.ID {
		t.Errorf("a second plant of the line made %d cultivars, line %d", len(g.Cultivars), g.Plots[1].Line)
	}
}

func TestNamedFormsAreNotDiscovered(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	growToFlower(t, g, 0, sp, sp.VarietyGenome(1), 0, now)
	if len(g.Cultivars) != 0 {
		t.Error("a plain named form was recorded as a new hybrid")
	}
}

func TestAStableLineIsRecognised(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	growToFlower(t, g, 0, sp, oddCosmos(), stableRuns, now)
	if !g.Cultivars[0].Stable || g.StableLines() != 1 {
		t.Error("a line that has bred true three generations running is not stable")
	}
}

func TestStreaksBuildAndBreak(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	if err := g.Plant(0, sp, 0, now); err != nil {
		t.Fatal(err)
	}
	p := &g.Plots[0]
	p.Growth, p.Pods, p.Streak = 1, 2, 1

	g.Gather(0, now) // selfed: the parents are identical
	if pk := g.Shed[0]; pk.Streak != 2 {
		t.Errorf("a selfed packet has streak %d, want 2", pk.Streak)
	}

	// A wide cross breaks the run.
	p.Pods = 2
	donor := oddCosmos()
	p.Pollen = &donor
	g.Gather(0, now)
	if pk := g.Shed[len(g.Shed)-1]; pk.Streak != 0 || pk.Descent == "" {
		t.Errorf("a wide cross has streak %d and descent %q", pk.Streak, pk.Descent)
	}

	// Sowing a seed that comes out close to the packet carries the streak on.
	pure := Packet{ID: 99, SpeciesID: sp.ID, A: sp.VarietyGenome(0), B: sp.VarietyGenome(0), Count: 1, Streak: 2, Pure: true}
	g.Shed = append(g.Shed, pure)
	if err := g.SowPacket(3, len(g.Shed)-1, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[3].Streak != 2 {
		t.Errorf("a true seedling has streak %d", g.Plots[3].Streak)
	}
}

func TestRenamingACultivarReachesPlantsAndSeed(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	growToFlower(t, g, 0, sp, oddCosmos(), 0, now)
	id := g.Cultivars[0].ID
	g.Plots[0].Pods = 3
	g.Gather(0, now)

	if !g.RenameCultivar(id, "Green Giant", now) {
		t.Fatal("rename failed")
	}
	if g.Cultivars[0].Name != "Green Giant" || !g.Cultivars[0].Named {
		t.Error("cultivar not renamed")
	}
	if g.Plots[0].LineName != "Green Giant" {
		t.Error("the plant did not take the new name")
	}
	if g.Plots[0].FullName() != "Common Cosmos ‘Green Giant’" && g.Plots[0].FullName() != sp.Common+" ‘Green Giant’" {
		t.Errorf("the plant reads %q", g.Plots[0].FullName())
	}
	for _, pk := range g.Shed {
		if pk.Line == id && pk.Label != "Green Giant" {
			t.Errorf("a packet of the line is still labelled %q", pk.Label)
		}
	}
	if g.RenameCultivar(id, "   ", now) || g.RenameCultivar(999, "x", now) {
		t.Error("renaming to nothing or a missing line should fail")
	}
}

func TestEverythingBredSurvivesASave(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	growToFlower(t, g, 0, sp, oddCosmos(), 0, now)
	g.Plots[0].Pods = 2
	donor := sp.VarietyGenome(2)
	g.Plots[0].Pollen = &donor
	path := filepath.Join(t.TempDir(), "g.json")
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	p := back.Plots[0]
	if p.Genome != oddCosmos() || p.Pollen == nil || *p.Pollen != donor || p.Line != g.Plots[0].Line || p.LineName != g.Plots[0].LineName {
		t.Errorf("a plant's genes did not survive: %+v", p)
	}
	if len(back.Cultivars) != 1 || back.Cultivars[0] != g.Cultivars[0] {
		t.Error("cultivars did not survive")
	}
	g.Gather(0, now)
	back2, _ := Load(path, now)
	_ = back2
}

func TestCultivarsAppearInTheAlmanacAndCanBeNamed(t *testing.T) {
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	growToFlower(t, g, 0, SpeciesByID("cosmos"), oddCosmos(), 0, now)

	m := newModel(g, filepath.Join(t.TempDir(), "g.json"), now)
	m.width, m.height = 110, 36
	var cur tea.Model = m
	send := func(msg tea.Msg) { cur, _ = cur.Update(msg) }
	key := func(k string) {
		if k == "enter" {
			send(tea.KeyMsg{Type: tea.KeyEnter})
			return
		}
		send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}

	key("a")
	got := cur.(model)
	idx := -1
	for i, r := range got.almanac {
		if r.Kind == rowCultivar {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("the almanac has no cultivar row")
	}
	got.almanacCursor = idx
	cur = got
	if v := cur.View(); !contains(v, "YOUR CULTIVARS") && !contains(v, "your cultivars") {
		t.Error("the almanac does not head the cultivars section")
	}
	key("n")
	if !cur.(model).naming {
		t.Fatal("n did not begin naming the cultivar")
	}
	for _, r := range "Moss" {
		key(string(r))
	}
	// Clear what was there first: the input starts with the old name.
	cm := cur.(model)
	cm.input.SetValue("Moss")
	cur = cm
	key("enter")
	if name := cur.(model).g.Cultivars[0].Name; name != "Moss" {
		t.Errorf("the cultivar is called %q", name)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
