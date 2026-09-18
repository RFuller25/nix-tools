package main

import (
	"testing"
	"time"
)

func testStart() time.Time { return time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC) }

// newTestGarden builds a garden with a fixed seed so weather is repeatable.
func newTestGarden(now time.Time) *Garden {
	g := NewGarden(now)
	g.Seed = 42
	g.Seeds = 500
	g.Matured = 100 // an experienced gardener: everything is unlocked

	// The soil is derived from the seed, so re-lay it now the seed is fixed.
	for i := range g.Plots {
		g.Plots[i].PH, g.Plots[i].Richness = 0, 0
	}
	g.layOutSoil()
	return g
}

// A watered, weeded plant should reach maturity in roughly the hours its
// species advertises — and always inside a single day.
func TestWellTendedMaturesOnSchedule(t *testing.T) {
	for _, sp := range AllSpecies() {
		now := inSeason(sp)
		g := newTestGarden(now)
		if sp.Kind == KindAquatic {
			g.Plots[0].Pond = true // water plants need water
		}
		if err := g.Plant(0, sp, now); err != nil {
			t.Fatalf("planting %s: %v", sp.ID, err)
		}

		elapsed := 0.0
		for g.Plots[0].Growth < 1 && elapsed < 48 {
			now = now.Add(15 * time.Minute)
			elapsed += 0.25
			g.Advance(now)
			g.Water(0, now)
			g.Weed(0, now)
		}

		if g.Plots[0].Growth < 1 {
			t.Errorf("%s never matured within 48h of perfect care", sp.ID)
			continue
		}
		// The promise the garden makes: tend a seed and it is grown up by
		// the end of the day.
		if elapsed > 24 {
			t.Errorf("%s took %.2fh of perfect care to mature, should be within a day", sp.ID, elapsed)
		}
		if slack := elapsed - sp.Hours; slack < -3 || slack > 5 {
			t.Errorf("%s matured in %.2fh, advertised %.1fh", sp.ID, elapsed, sp.Hours)
		}
	}
}

// inSeason picks a date inside one of a species' growing seasons, so growth
// is measured when the plant is in its element.
func inSeason(sp *Species) time.Time {
	month := map[Season]time.Month{
		Spring: time.April, Summer: time.July,
		Autumn: time.October, Winter: time.January,
	}[sp.Seasons[0]]
	return time.Date(2026, month, 15, 6, 0, 0, 0, time.UTC)
}

// Neglect is slow, not fatal: a forgotten plant keeps its place in the bed and
// keeps creeping forward.
func TestNeglectNeverKills(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("basil")
	if err := g.Plant(0, sp, now); err != nil {
		t.Fatal(err)
	}

	now = now.Add(20 * 24 * time.Hour)
	g.Advance(now)

	p := &g.Plots[0]
	if p.Empty() {
		t.Fatal("a neglected plant vanished from its bed")
	}
	if p.Growth <= 0 {
		t.Error("a neglected plant made no progress at all")
	}
	if !p.Matured {
		t.Error("even neglected, three weeks should be enough to mature")
	}
	if p.Weeds < 0.9 {
		t.Errorf("weeds only reached %.2f after three weeks of neglect", p.Weeds)
	}
}

// Dry soil must slow growth down relative to a watered bed.
func TestThirstSlowsGrowth(t *testing.T) {
	now := testStart()
	sp := SpeciesByID("zinnia")

	wet := newTestGarden(now)
	dry := newTestGarden(now)
	if err := wet.Plant(0, sp, now); err != nil {
		t.Fatal(err)
	}
	if err := dry.Plant(0, sp, now); err != nil {
		t.Fatal(err)
	}

	at := now
	for i := 0; i < 24; i++ {
		at = at.Add(time.Hour)
		wet.Advance(at)
		wet.Water(0, at)
		wet.Weed(0, at)
		dry.Advance(at)
		dry.Plots[0].Moisture = 0 // a bed that never sees the can
	}

	if dry.Plots[0].Growth >= wet.Plots[0].Growth {
		t.Errorf("dry growth %.3f should trail watered growth %.3f",
			dry.Plots[0].Growth, wet.Plots[0].Growth)
	}
	if dry.Plots[0].Growth <= 0 {
		t.Error("a dry plant should still inch along")
	}
}

// Rain should water the garden without the gardener lifting a finger.
func TestRainWatersTheGarden(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("mint"), now); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Moisture = 0.1

	// Find a day this garden gets rain and run it through that day.
	day := now
	for i := 0; i < 40; i++ {
		if WeatherFor(g.Seed, day).Rainfall > 0.15 {
			break
		}
		day = day.AddDate(0, 0, 1)
	}
	if WeatherFor(g.Seed, day).Rainfall <= 0.15 {
		t.Skip("no rainy day found in the next 40 days for this seed")
	}

	g.LastTick = day
	before := g.Plots[0].Moisture
	g.Advance(day.Add(6 * time.Hour))
	if g.Plots[0].Moisture <= before {
		t.Errorf("rain left the soil at %.2f, was %.2f", g.Plots[0].Moisture, before)
	}
}

// Growth accrued while the program was closed, in one big catch-up step,
// should land close to the same place as growth simulated live.
func TestOfflineCatchUpMatchesLiveGrowth(t *testing.T) {
	now := testStart()
	sp := SpeciesByID("marigold")

	live := newTestGarden(now)
	offline := newTestGarden(now)
	if err := live.Plant(0, sp, now); err != nil {
		t.Fatal(err)
	}
	if err := offline.Plant(0, sp, now); err != nil {
		t.Fatal(err)
	}

	at := now
	for i := 0; i < 12*4; i++ { // twelve hours, quarter-hour ticks
		at = at.Add(15 * time.Minute)
		live.Advance(at)
	}
	offline.Advance(at)

	diff := live.Plots[0].Growth - offline.Plots[0].Growth
	if diff < -1e-9 || diff > 1e-9 {
		t.Errorf("offline growth %.6f differs from live %.6f",
			offline.Plots[0].Growth, live.Plots[0].Growth)
	}
}

// A very long absence is capped so the catch-up loop stays bounded.
func TestCatchUpIsCapped(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("thyme"), now); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		g.Advance(now.AddDate(3, 0, 0)) // three years away
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Advance did not finish for a three-year absence")
	}
	if g.Plots[0].Pods > maxPods {
		t.Errorf("pods ran away to %.1f", g.Plots[0].Pods)
	}
}

func TestPlantingCostsAndLocks(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Seeds = 5
	g.Matured = 0

	cheap := SpeciesByID("basil") // cost 2, unlock 0
	if err := g.Plant(0, cheap, now); err != nil {
		t.Fatalf("planting basil: %v", err)
	}
	if g.Seeds != 3 {
		t.Errorf("seeds after planting basil = %d, want 3", g.Seeds)
	}
	if g.Planted != 1 {
		t.Errorf("lifetime planted = %d, want 1", g.Planted)
	}

	if err := g.Plant(0, cheap, now); err == nil {
		t.Error("planting into an occupied bed should fail")
	}

	locked := SpeciesByID("lotus")
	g.Plots[1].Pond = true
	if err := g.Plant(1, locked, now); err == nil {
		t.Error("planting a locked species should fail")
	}
	g.Matured = locked.Unlock
	if err := g.Plant(1, locked, now); err == nil {
		t.Error("planting without enough seeds should fail")
	}
}

func TestTendingRewards(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("sunflower"), now); err != nil {
		t.Fatal(err)
	}

	g.Plots[0].Moisture = 0.2
	if !g.Water(0, now) {
		t.Error("a dry bed should accept water")
	}
	if g.Water(0, now) {
		t.Error("a soaked bed should not need watering again")
	}

	seeds := g.Seeds
	g.Plots[0].Weeds = 0.8
	ok, reward := g.Weed(0, now)
	if !ok || reward != weedingReward {
		t.Errorf("weeding an overgrown bed gave (%v, %d)", ok, reward)
	}
	if g.Seeds != seeds+weedingReward {
		t.Errorf("seeds = %d, want %d", g.Seeds, seeds+weedingReward)
	}
	if ok, _ := g.Weed(0, now); ok {
		t.Error("a clean bed should not report weeding work")
	}

	g.Plots[0].Growth = 1
	g.Plots[0].Pods = 3.4
	seeds = g.Seeds
	if got := g.Gather(0, now); got != 3 {
		t.Errorf("gathered %d seeds, want 3", got)
	}
	if g.Seeds != seeds+3 || g.Gathered != 3 {
		t.Errorf("seeds = %d (want %d), lifetime gathered = %d", g.Seeds, seeds+3, g.Gathered)
	}
	if got := g.Gather(0, now); got != 0 {
		t.Errorf("gathering the leftovers gave %d", got)
	}
}

func TestUprootAndRename(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("tulip"), now); err != nil {
		t.Fatal(err)
	}

	if !g.Rename(0, "Mabel", now) {
		t.Fatal("rename failed")
	}
	if got := g.Plots[0].DisplayName(); got != "Mabel" {
		t.Errorf("display name = %q", got)
	}
	g.Rename(0, "", now)
	if got := g.Plots[0].DisplayName(); got != "Garden Tulip" {
		t.Errorf("cleared name fell back to %q", got)
	}

	g.Plots[0].Weeds = 0.5
	if !g.Uproot(0, now) {
		t.Fatal("uproot failed")
	}
	if !g.Plots[0].Empty() {
		t.Error("bed still holds a plant after uprooting")
	}
	if g.Plots[0].Weeds != 0.5 {
		t.Error("uprooting should leave the soil's weeds alone")
	}
	if g.Uproot(0, now) {
		t.Error("uprooting an empty bed should report nothing done")
	}
	if g.Rename(0, "Ghost", now) {
		t.Error("renaming an empty bed should fail")
	}
}

func TestDailyVisitBonus(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Seeds = 0

	if got := g.Visit(now); got != dailyBonus {
		t.Errorf("first visit gave %d seeds, want %d", got, dailyBonus)
	}
	if got := g.Visit(now.Add(2 * time.Hour)); got != 0 {
		t.Errorf("second visit the same day gave %d seeds", got)
	}
	if got := g.Visit(now.AddDate(0, 0, 1)); got != dailyBonus {
		t.Errorf("next day's visit gave %d seeds", got)
	}
	if g.Seeds != 2*dailyBonus {
		t.Errorf("seeds = %d, want %d", g.Seeds, 2*dailyBonus)
	}
}

func TestStageThresholds(t *testing.T) {
	cases := []struct {
		growth float64
		stage  int
	}{
		{0, StageSeed}, {0.09, StageSeed},
		{0.10, StageSprout}, {0.34, StageSprout},
		{0.35, StageSeedling}, {0.69, StageSeedling},
		{0.70, StageBud}, {0.99, StageBud},
		{1.0, StageMature},
	}
	p := Plot{SpeciesID: "basil"}
	for _, c := range cases {
		p.Growth = c.growth
		if got := p.Stage(); got != c.stage {
			t.Errorf("growth %.2f gave stage %s, want %s", c.growth, stageNames[got], stageNames[c.stage])
		}
	}
}

func TestPodsAccumulateAndCap(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("cosmos"), now); err != nil {
		t.Fatal(err)
	}
	g.Plots[0].Growth = 1
	g.Plots[0].Matured = true

	g.Advance(now.Add(6 * time.Hour))
	if pods := g.Plots[0].Pods; pods < 0.9 || pods > 1.1 {
		t.Errorf("six hours of ripening gave %.2f pods, want about 1", pods)
	}

	g.Advance(now.Add(20 * 24 * time.Hour))
	if g.Plots[0].Pods > maxPods {
		t.Errorf("pods reached %.2f, cap is %.0f", g.Plots[0].Pods, maxPods)
	}
}
