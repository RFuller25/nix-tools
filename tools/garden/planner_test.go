package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBuiltInTemplatesAreSound(t *testing.T) {
	for _, tpl := range builtinTemplates() {
		if tpl.Name == "" || tpl.Note == "" || len(tpl.Cells) == 0 {
			t.Errorf("%q is incomplete", tpl.ID)
		}
		seen := map[[2]int]bool{}
		for _, c := range tpl.Cells {
			sp := SpeciesByID(c.Species)
			if sp == nil {
				t.Errorf("%s names an unknown species %q", tpl.ID, c.Species)
				continue
			}
			if c.DX < 0 || c.DX >= tpl.W || c.DY < 0 || c.DY >= tpl.H || c.DX >= plotCols {
				t.Errorf("%s: %s is outside the layout", tpl.ID, c.Species)
			}
			if c.Variety < 0 || c.Variety >= len(sp.Varieties()) {
				t.Errorf("%s: %s has no form %d", tpl.ID, c.Species, c.Variety)
			}
			if c.Pond != (sp.Kind == KindAquatic) {
				t.Errorf("%s: %s pond flag is wrong", tpl.ID, c.Species)
			}
			if seen[[2]int{c.DX, c.DY}] {
				t.Errorf("%s puts two plants in one bed", tpl.ID)
			}
			seen[[2]int{c.DX, c.DY}] = true
		}
	}
}

func TestStampingPrefersYourSeedAndSkipsWhatDoesNotFit(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 500
	three := builtinTemplates()[0]
	g.AddPacket(Packet{SpeciesID: "pea", A: SpeciesByID("pea").VarietyGenome(0), B: SpeciesByID("pea").VarietyGenome(0), Count: 1, Pure: true})

	ghosts, skipped := g.StampPlan(three, 0)
	if len(ghosts) != 3 || len(skipped) != 0 {
		t.Fatalf("%d ghosts, skipped %v", len(ghosts), skipped)
	}
	if ghosts[1].Packet == 0 {
		t.Error("the pea should come from your shed")
	}
	if ghosts[0].Packet != 0 || ghosts[2].Packet != 0 {
		t.Error("corn and squash have no packet and should come from the shop")
	}
	cost, fromShed, locked := g.PlanCost(ghosts)
	if fromShed != 1 || locked != 0 || cost != SpeciesByID("sweetcorn").SeedCost+SpeciesByID("pumpkin").SeedCost {
		t.Errorf("cost %d, from shed %d, locked %d", cost, fromShed, locked)
	}

	// Off the right edge, and onto an occupied bed.
	if _, skipped := g.StampPlan(three, 3); len(skipped) == 0 {
		t.Error("a layout hanging off the edge was not trimmed")
	}
	if err := g.Plant(1, SpeciesByID("basil"), 0, now); err != nil {
		t.Fatal(err)
	}
	ghosts, skipped = g.StampPlan(three, 0)
	if _, ok := ghosts[1]; ok || len(skipped) != 1 || !strings.Contains(skipped[0], "already holds") {
		t.Errorf("an occupied bed was planted over: %v %v", ghosts, skipped)
	}
	// A pond plant needs a pond.
	pond := builtinTemplates()[4]
	if ghosts, skipped := g.StampPlan(pond, 5); len(ghosts) != 0 || len(skipped) != 3 {
		t.Errorf("water plants were stamped on dry land: %d ghosts, %d skipped", len(ghosts), len(skipped))
	}
}

func TestACraftedWantPicksTheTallestSeed(t *testing.T) {
	g := newTestGarden(testStart())
	sp := SpeciesByID("foxglove")
	low, high := sp.VarietyGenome(0), sp.VarietyGenome(0)
	low.Height, high.Height = 10, 90
	g.AddPacket(Packet{SpeciesID: sp.ID, A: low, B: low, Count: 1})
	g.AddPacket(Packet{SpeciesID: sp.ID, A: high, B: high, Count: 1})
	got := g.seedSource(sp, 0, &TraitWant{TraitHeight, true}, map[int64]int{})
	if got.Genome.Height != 90 {
		t.Errorf("asked for the tallest and got height %d", got.Genome.Height)
	}
	got = g.seedSource(sp, 0, &TraitWant{TraitHeight, false}, map[int64]int{})
	if got.Genome.Height != 10 {
		t.Errorf("asked for the shortest and got height %d", got.Genome.Height)
	}
}

func TestCommittingAPlanSowsAndBuys(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 50
	sp := SpeciesByID("pea")
	g.AddPacket(Packet{SpeciesID: sp.ID, A: sp.VarietyGenome(0), B: sp.VarietyGenome(0), Count: 1, Pure: true})
	ghosts, _ := g.StampPlan(builtinTemplates()[0], 0)
	before := g.Gold
	sown, bought, spent, problems := g.CommitGhosts(ghosts, now)
	if sown != 3 || bought != 2 || len(problems) != 0 {
		t.Fatalf("sown %d bought %d problems %v", sown, bought, problems)
	}
	if before-g.Gold != spent || spent != SpeciesByID("sweetcorn").SeedCost+SpeciesByID("pumpkin").SeedCost {
		t.Errorf("spent %d, gold %d→%d", spent, before, g.Gold)
	}
	if g.SeedsInShed() != 0 {
		t.Error("the shed pea was not used")
	}
	for i, id := range []string{"sweetcorn", "pea", "pumpkin"} {
		if g.Plots[i].SpeciesID != id {
			t.Errorf("bed %d holds %q, want %s", i+1, g.Plots[i].SpeciesID, id)
		}
	}

	// With no gold to buy with, the shop seeds are refused, not forced.
	h := newTestGarden(now)
	h.Gold = 0
	ghosts, _ = h.StampPlan(builtinTemplates()[0], 0)
	sown, _, _, problems = h.CommitGhosts(ghosts, now)
	if sown != 0 || len(problems) != 3 {
		t.Errorf("broke gardener sowed %d with %d problems", sown, len(problems))
	}
}

func TestGhostsDoNotTouchTheRealGarden(t *testing.T) {
	g := newTestGarden(testStart())
	ghosts := map[int]ghost{0: {Species: "marigold", Genome: SpeciesByID("marigold").VarietyGenome(0)}, 1: {Species: "tomato", Genome: SpeciesByID("tomato").VarietyGenome(0)}}
	c := g.withGhosts(ghosts)
	if !g.Plots[0].Empty() || !g.Plots[1].Empty() {
		t.Fatal("the plan changed the real beds")
	}
	if c.Synergy(1).Take < 0.11 {
		t.Errorf("a planned marigold gives the tomato %+.2f", c.Synergy(1).Take)
	}
}

func TestSavingAndReloadingALayout(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 500
	_ = g.Plant(6, SpeciesByID("tomato"), 2, now)
	_ = g.Plant(7, SpeciesByID("basil"), 0, now)
	tpl, err := g.SaveTemplate("Pair", rectOf(6, 12), now)
	if err != nil || len(tpl.Cells) != 2 || tpl.W != 2 || tpl.H != 2 {
		t.Fatalf("saved %+v, %v", tpl, err)
	}
	if _, err := g.SaveTemplate("", rectOf(6, 7), now); err == nil {
		t.Error("an unnamed layout was saved")
	}
	if _, err := g.SaveTemplate("Empty", rectOf(0, 1), now); err == nil {
		t.Error("an empty layout was saved")
	}

	path := filepath.Join(t.TempDir(), "g.json")
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Templates) != 1 || back.Templates[0].Name != "Pair" || back.Templates[0].Cells[0].Variety != 2 || back.Templates[0].W != 2 {
		t.Errorf("layout did not survive: %+v", back.Templates)
	}
	if len(back.AllTemplates()) != len(builtinTemplates())+1 {
		t.Error("built-in layouts and your own are not listed together")
	}
	if !back.DeleteTemplate(back.Templates[0].ID) || len(back.Templates) != 0 {
		t.Error("delete failed")
	}
	if back.DeleteTemplate("three-sisters") {
		t.Error("a built-in layout was deleted")
	}
}

func keyPress(cur tea.Model, k string) tea.Model {
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	next, _ := cur.Update(msg)
	return next
}

func TestPlanModeKeyFlow(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 100
	sp := SpeciesByID("marigold")
	g.AddPacket(Packet{SpeciesID: sp.ID, A: sp.VarietyGenome(0), B: sp.VarietyGenome(0), Count: 2, Pure: true})
	m := newModel(g, filepath.Join(t.TempDir(), "g.json"), now)
	m.width, m.height = 110, 40

	var cur tea.Model = m
	cur = keyPress(cur, "P")
	if cur.(model).mode != modePlan {
		t.Fatal("P did not start plan mode")
	}
	cur = keyPress(cur, "enter")
	cur = keyPress(cur, "l")
	cur = keyPress(cur, "enter")
	if n := len(cur.(model).ghosts); n != 2 {
		t.Fatalf("%d ghosts placed", n)
	}
	cur = keyPress(cur, "l")
	cur = keyPress(cur, "enter") // no marigold seed left: refused
	if n := len(cur.(model).ghosts); n != 2 {
		t.Errorf("placed a third ghost from two seeds: %d", n)
	}
	if v := cur.View(); !strings.Contains(v, "plan") {
		t.Error("plan mode does not say it is plan mode")
	}
	if !g.Plots[0].Empty() {
		t.Error("planning planted something")
	}
	cur = keyPress(cur, "C")
	got := cur.(model)
	if got.mode != modeNone || g.Plots[0].SpeciesID != "marigold" || g.Plots[1].SpeciesID != "marigold" {
		t.Errorf("C did not sow the plan: mode %d, beds %q %q", got.mode, g.Plots[0].SpeciesID, g.Plots[1].SpeciesID)
	}

	cur = keyPress(cur, "P")
	cur = keyPress(cur, "esc")
	if cur.(model).mode != modeNone {
		t.Error("esc did not leave plan mode")
	}
}

func TestSelectAndSaveAndStampKeyFlow(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 500
	_ = g.Plant(0, SpeciesByID("tomato"), 0, now)
	_ = g.Plant(1, SpeciesByID("basil"), 0, now)
	m := newModel(g, filepath.Join(t.TempDir(), "g.json"), now)
	m.width, m.height = 110, 40

	var cur tea.Model = m
	cur = keyPress(cur, "V")
	cur = keyPress(cur, "l")
	cur = keyPress(cur, "enter")
	if !cur.(model).naming {
		t.Fatalf("enter did not ask for a name: %q", cur.(model).status)
	}
	cm := cur.(model)
	cm.input.SetValue("Twosome")
	cur = cm
	cur = keyPress(cur, "enter")
	if len(g.Templates) != 1 || cur.(model).mode != modeNone {
		t.Fatalf("layout not saved: %d templates, mode %d", len(g.Templates), cur.(model).mode)
	}

	// Stamp it two rows down.
	cur = keyPress(cur, "T")
	if cur.(model).screen != screenTemplates {
		t.Fatal("T did not open the layouts")
	}
	for i := 0; i < len(builtinTemplates()); i++ {
		cur = keyPress(cur, "j")
	}
	if v := cur.View(); !strings.Contains(v, "Twosome") {
		t.Error("the saved layout is not listed")
	}
	cur = keyPress(cur, "enter")
	if cur.(model).mode != modeStamp {
		t.Fatal("enter did not begin stamping")
	}
	cm = cur.(model)
	cm.cursor = 10
	cur = cm
	cur = keyPress(cur, "l") // move, re-stamp at 11
	cur = keyPress(cur, "enter")
	if g.Plots[11].SpeciesID != "tomato" || g.Plots[12].SpeciesID != "basil" {
		t.Errorf("layout landed as %q %q", g.Plots[11].SpeciesID, g.Plots[12].SpeciesID)
	}
}

func TestPlannerScreensFitEverySize(t *testing.T) {
	for _, size := range [][2]int{{110, 40}, {80, 24}, {60, 20}, {40, 14}, {24, 10}} {
		m := demoModel(t, size[0], size[1])
		m.screen = screenTemplates
		if v := m.View(); strings.TrimSpace(v) == "" || len(strings.Split(v, "\n")) > size[1] {
			t.Errorf("layouts screen at %dx%d is %d lines", size[0], size[1], len(strings.Split(v, "\n")))
		}
		m.screen = screenGarden
		m.startPlan()
		if v := m.View(); len(strings.Split(v, "\n")) > size[1] {
			t.Errorf("plan mode at %dx%d is %d lines", size[0], size[1], len(strings.Split(v, "\n")))
		}
		m.endMode("")
		m.screen = screenShop
		m.shelf = shelfMine
		if v := m.View(); len(strings.Split(v, "\n")) > size[1] {
			t.Errorf("my seeds at %dx%d is %d lines", size[0], size[1], len(strings.Split(v, "\n")))
		}
	}
}
