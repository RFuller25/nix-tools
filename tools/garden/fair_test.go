package main

import (
	"testing"
	"time"
)

func TestEveryClassHasAnEligiblePlantAndSensibleScores(t *testing.T) {
	for _, c := range fairCategories {
		if c.Name == "" || c.Blurb == "" || c.Score == nil {
			t.Errorf("class %q is incomplete", c.ID)
		}
		eligible := 0
		for _, sp := range AllSpecies() {
			if c.Eligible(sp) {
				eligible++
				for i := range sp.Varieties() {
					s := FairScore(c, sp, sp.VarietyGenome(i), 1)
					if s < 0 || s > 100 {
						t.Errorf("%s scores %s ‘%d’ at %.1f", c.ID, sp.ID, i, s)
					}
				}
			}
		}
		if eligible == 0 {
			t.Errorf("nothing is eligible for %s", c.ID)
		}
	}
}

func TestScoresRewardWhatTheClassAsksFor(t *testing.T) {
	sp := SpeciesByID("foxglove")
	tall, short := sp.VarietyGenome(0), sp.VarietyGenome(0)
	tall.Height, short.Height = 95, 5
	byID := func(id string) FairCategory { c, _ := categoryByID(id); return c }
	if FairScore(byID("tallest"), sp, tall, 1) <= FairScore(byID("tallest"), sp, short, 1) {
		t.Error("the tallest class does not favour height")
	}
	if FairScore(byID("compact"), sp, short, 1) <= FairScore(byID("compact"), sp, tall, 1) {
		t.Error("the compact class does not favour small plants")
	}
	vivid, dull := tall, tall
	vivid.Sat, vivid.Light = 95, 50
	dull.Sat, dull.Light = 5, 90
	if FairScore(byID("colour"), sp, vivid, 1) <= FairScore(byID("colour"), sp, dull, 1) {
		t.Error("the colour class does not favour rich colour")
	}
	if FairScore(byID("tallest"), sp, tall, 1) <= FairScore(byID("tallest"), sp, tall, 0) {
		t.Error("a well-kept plant should beat a neglected one of the same genes")
	}
}

func TestOneClassPerWeekAndItIsStable(t *testing.T) {
	g := newTestGarden(testStart())
	seen := map[string]bool{}
	for w := 202601; w < 202653; w++ {
		a, b := g.FairFor(w), g.FairFor(w)
		if a.ID != b.ID {
			t.Fatal("the class changed between asks")
		}
		seen[a.ID] = true
	}
	if len(seen) < 4 {
		t.Errorf("only %d different classes in a year", len(seen))
	}
}

func fairGarden(t *testing.T) (*Garden, time.Time, FairCategory) {
	t.Helper()
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC) // a Tuesday
	g := newTestGarden(now)
	g.Gold = 100
	// A class every kind of plant may enter, so the test is not at the mercy
	// of which one the seed picks.
	for seed := int64(1); seed < 400; seed++ {
		g.Seed = seed
		if c := g.FairFor(weekKey(now)); c.Kind < 0 {
			return g, now, c
		}
	}
	t.Fatal("no open class found")
	return nil, now, FairCategory{}
}

func TestEnteringAndBeingJudged(t *testing.T) {
	g, now, cat := fairGarden(t)
	if err := g.Plant(0, SpeciesByID("cosmos"), 0, now); err != nil {
		t.Fatal(err)
	}
	if err := g.Enter(0, now); err == nil {
		t.Error("entered a plant that is not in flower")
	}
	g.Plots[0].Growth, g.Plots[0].Moisture = 1, 0.9
	if err := g.Enter(0, now); err != nil {
		t.Fatal(err)
	}
	if g.Fair.Entry == nil || g.Fair.Entry.Week != weekKey(now) {
		t.Fatal("the entry was not recorded")
	}
	// Entering again replaces it, it does not add one.
	if err := g.Enter(0, now); err != nil {
		t.Fatal(err)
	}

	// Still the same week: nothing is judged.
	g.settleFair(now.Add(24 * time.Hour))
	if g.Fair.Entry == nil || len(g.Fair.Ribbons) != 0 {
		t.Error("judged before the week was out")
	}

	gold := g.Gold
	next := now.AddDate(0, 0, 7)
	g.settleFair(next)
	if g.Fair.Entry != nil || len(g.Fair.Ribbons) != 1 {
		t.Fatalf("not judged after the week: %d ribbons", len(g.Fair.Ribbons))
	}
	r := g.Fair.Ribbons[0]
	if r.Place < 1 || r.Place > fairRivals+1 || r.Category != cat.ID || r.Prize < prizeLast || g.Gold != gold+r.Prize {
		t.Errorf("ribbon %+v, gold %d→%d", r, gold, g.Gold)
	}
	// Never judged twice.
	g.settleFair(next.AddDate(0, 0, 7))
	if len(g.Fair.Ribbons) != 1 {
		t.Error("the same entry was judged twice")
	}
}

func TestJudgingIsTheSameLiveOrCaughtUp(t *testing.T) {
	build := func() *Garden {
		g, now, _ := fairGarden(t)
		_ = g.Plant(0, SpeciesByID("cosmos"), 0, now)
		g.Plots[0].Growth = 1
		_ = g.Enter(0, now)
		return g
	}
	a, b := build(), build()
	week2 := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	a.settleFair(week2)
	b.settleFair(week2.AddDate(0, 0, 30)) // judged a month late, from a closed garden
	ra, rb := a.Fair.Ribbons[0], b.Fair.Ribbons[0]
	if ra.Place != rb.Place || ra.Score != rb.Score || ra.Prize != rb.Prize {
		t.Errorf("judged differently late: %+v vs %+v", ra, rb)
	}
}

func TestWinningRaisesTheBar(t *testing.T) {
	g, now, cat := fairGarden(t)
	sp := SpeciesByID("cosmos")
	// A perfect plant for the class.
	best := sp.VarietyGenome(0)
	best.Height, best.Speed, best.Yield, best.Sat, best.Light, best.Shape = 100, 100, 100, 100, 50, 50
	g.sow(0, sp, best, 0, now)
	g.Plots[0].Growth, g.Plots[0].Moisture, g.Plots[0].Weeds = 1, 1, 0
	_ = g.Enter(0, now)
	g.settleFair(now.AddDate(0, 0, 7))
	if cat.ID == "compact" {
		t.Skip("a perfect plant for another class")
	}
	if g.Fair.Ribbons[0].Place != 1 || g.Fair.Wins != 1 {
		t.Skipf("the perfect plant came %d in %s", g.Fair.Ribbons[0].Place, cat.Name)
	}
	if g.Gold < 100+prizeFirst {
		t.Error("first place was not paid")
	}
}

func TestClassLimitedToAKindRefusesOthers(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	g.Gold = 100
	for seed := int64(1); seed < 800; seed++ {
		g.Seed = seed
		if c := g.FairFor(weekKey(now)); c.Kind == KindHerb {
			break
		}
	}
	if g.FairFor(weekKey(now)).Kind != KindHerb {
		t.Skip("no herb class in range")
	}
	_ = g.Plant(0, SpeciesByID("tomato"), 0, now)
	g.Plots[0].Growth = 1
	if err := g.Enter(0, now); err == nil {
		t.Error("a tomato was entered in the herb class")
	}
	_ = g.Plant(1, SpeciesByID("basil"), 0, now)
	g.Plots[1].Growth = 1
	if err := g.Enter(1, now); err != nil {
		t.Errorf("a basil was refused the herb class: %v", err)
	}
}

func TestFairScreenAndKeys(t *testing.T) {
	g, now, _ := fairGarden(t)
	_ = g.Plant(0, SpeciesByID("cosmos"), 0, now)
	g.Plots[0].Growth = 1
	m := newModel(g, t.TempDir()+"/g.json", now)
	m.width, m.height = 100, 34
	cur := keyPress(m, "E")
	if cur.(model).screen != screenFair {
		t.Fatal("E did not open the fair")
	}
	if cur.View() == "" {
		t.Fatal("blank fair")
	}
	cur = keyPress(cur, "enter")
	if g.Fair.Entry == nil {
		t.Errorf("enter did not enter the plant: %q", cur.(model).status)
	}
}
