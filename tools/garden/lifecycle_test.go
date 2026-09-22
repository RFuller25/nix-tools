package main

import (
	"testing"
	"time"
)

func TestEverySpeciesHasALife(t *testing.T) {
	for _, sp := range AllSpecies() {
		if _, ok := lives[sp.ID]; !ok {
			t.Errorf("%s (%s) is not in the lifecycle table", sp.ID, sp.Common)
		}
		if sp.Kind == KindTree && sp.Life() != Woody {
			t.Errorf("%s is a tree or shrub but lives as a %s", sp.ID, sp.Life())
		}
		if sp.Evergreen() && sp.Life() == Annual {
			t.Errorf("%s is an evergreen annual, which cannot be right", sp.ID)
		}
	}
}

func TestAnnualsGoToSeedAndPerennialsDoNot(t *testing.T) {
	cases := []struct {
		id    string
		seeds bool
	}{
		{"sunflower", true},      // annual
		{"foxglove", true},       // biennial
		{"coneflower", false},    // perennial
		{"japanesemaple", false}, // tree
	}

	for _, c := range cases {
		now := inSeason(SpeciesByID(c.id))
		g := newTestGarden(now)
		if err := g.Plant(0, SpeciesByID(c.id), now); err != nil {
			t.Fatalf("planting %s: %v", c.id, err)
		}

		// Run a fortnight of well-watered growth.
		at := now
		for i := 0; i < 14*24; i++ {
			at = at.Add(time.Hour)
			g.Advance(at)
			g.Water(0, at)
			g.Weed(0, at)
		}

		p := g.Plots[0]
		if p.Spent != c.seeds {
			t.Errorf("%s: spent = %v after a fortnight, want %v", c.id, p.Spent, c.seeds)
		}
		if p.Empty() {
			t.Errorf("%s vanished from its bed", c.id)
		}
		if !p.Matured {
			t.Errorf("%s never flowered", c.id)
		}
	}
}

func TestSpentPlantsStandStillAndStopRipening(t *testing.T) {
	now := inSeason(SpeciesByID("radish"))
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("radish"), now); err != nil {
		t.Fatal(err)
	}

	at := now
	for i := 0; i < 24*10 && !g.Plots[0].Spent; i++ {
		at = at.Add(time.Hour)
		g.Advance(at)
		g.Water(0, at)
	}
	if !g.Plots[0].Spent {
		t.Fatal("the radish never finished its year")
	}

	// Going to seed hands over a last few seeds.
	if g.Plots[0].Pods < 1 {
		t.Error("a plant gone to seed left nothing to gather")
	}

	g.Gather(0, at)
	pods := g.Plots[0].Pods
	g.Advance(at.Add(48 * time.Hour))
	if g.Plots[0].Pods != pods {
		t.Errorf("a spent plant went on ripening: %.2f then %.2f", pods, g.Plots[0].Pods)
	}
	if g.Plots[0].Empty() {
		t.Error("a spent plant disappeared instead of standing")
	}

	// And it can still be lifted and composted.
	if _, ok := g.Uproot(0, at); !ok {
		t.Error("a spent plant could not be lifted")
	}
}

func TestWinterSendsPerennialsToSleep(t *testing.T) {
	winter := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	summer := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)

	perennial := SpeciesByID("hosta")    // dies back
	tree := SpeciesByID("japanesemaple") // stands bare
	evergreen := SpeciesByID("rosemary") // carries on
	annual := SpeciesByID("basil")       // never gets the chance

	if !dormant(perennial, SeasonOf(winter)) {
		t.Error("a hosta should die back in winter")
	}
	if !dormant(tree, SeasonOf(winter)) {
		t.Error("a maple should be bare in winter")
	}
	if dormant(evergreen, SeasonOf(winter)) {
		t.Error("rosemary is evergreen and should not go dormant")
	}
	if dormant(annual, SeasonOf(winter)) {
		t.Error("an annual does not go dormant; it goes to seed")
	}
	if dormant(perennial, SeasonOf(summer)) {
		t.Error("nothing should be dormant in July")
	}
}

func TestAppearanceFollowsTheLifecycle(t *testing.T) {
	grown := &Plot{SpeciesID: "hosta", Growth: 1, Matured: true, PH: 6.5}
	hosta := SpeciesByID("hosta")

	stage, pal := bareAppearance(hosta, grown, Summer, phaseNoon)
	if stage != StageMature || pal != hosta.Palette {
		t.Error("a growing hosta in summer should be drawn in full")
	}

	stage, pal = bareAppearance(hosta, grown, Winter, phaseNoon)
	if stage != StageSeedling {
		t.Errorf("a dormant perennial drew stage %s, want a tuft", stageNames[stage])
	}
	if pal != sleepingPalette {
		t.Error("a dormant plant should be drawn in resting colours")
	}

	maple := SpeciesByID("japanesemaple")
	bare := &Plot{SpeciesID: "japanesemaple", Growth: 1, Matured: true, PH: 6.5}
	if stage, _ := bareAppearance(maple, bare, Winter, phaseNoon); stage != StageBud {
		t.Errorf("a bare tree drew stage %s", stageNames[stage])
	}

	spent := &Plot{SpeciesID: "sunflower", Growth: 1, Matured: true, Spent: true, PH: 6.5}
	stage, pal = bareAppearance(SpeciesByID("sunflower"), spent, Summer, phaseNoon)
	if stage != StageMature || pal != driedPalette {
		t.Error("a plant gone to seed should keep its shape and lose its colour")
	}
}

func TestHerbariumRecordsFirstFlowering(t *testing.T) {
	now := inSeason(SpeciesByID("radish"))
	g := newTestGarden(now)
	if _, ok := g.Collected(SpeciesByID("radish")); ok {
		t.Fatal("a new garden already has a herbarium entry")
	}

	if err := g.Plant(0, SpeciesByID("radish"), now); err != nil {
		t.Fatal(err)
	}
	at := now
	for i := 0; i < 24*3 && !g.Plots[0].Matured; i++ {
		at = at.Add(time.Hour)
		g.Advance(at)
		g.Water(0, at)
	}

	when, ok := g.Collected(SpeciesByID("radish"))
	if !ok {
		t.Fatal("flowering did not press the radish into the herbarium")
	}
	if when.Before(now) || when.After(at.Add(time.Hour)) {
		t.Errorf("the herbarium recorded %v, outside the run from %v to %v", when, now, at)
	}
	if len(g.Herbarium) != 1 {
		t.Errorf("the herbarium holds %d entries, want 1", len(g.Herbarium))
	}

	// Growing a second one does not file it twice.
	g.Uproot(0, at) //nolint:errcheck // the yield is tested elsewhere
	if err := g.Plant(0, SpeciesByID("radish"), at); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Growth = 0.99
	g.Advance(at.Add(6 * time.Hour))
	if len(g.Herbarium) != 1 {
		t.Errorf("the herbarium holds %d entries after a second radish", len(g.Herbarium))
	}
	if g.Herbarium["radish"] != when {
		t.Error("the first-flowered date was overwritten")
	}
}

func TestHerbariumSurvivesASave(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	g := newTestGarden(now)
	g.collect(SpeciesByID("peony"), now)

	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := back.Collected(SpeciesByID("peony")); !ok {
		t.Error("the herbarium did not survive a save and load")
	}
}
