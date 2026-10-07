package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// twoCosmos is a garden with two flowering cosmos side by side, one of each
// colour form, on a warm spring day with the weather fixed in the bees' favour.
func twoCosmos(t *testing.T) (*Garden, time.Time) {
	t.Helper()
	now := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	// Give the garden a seed whose first week is dry and bright.
	for seed := int64(1); seed < 500; seed++ {
		dry := true
		for d := 0; d < 4; d++ {
			w := WeatherFor(seed, now.Add(time.Duration(d)*24*time.Hour))
			if w.Kind != Sunny && w.Kind != Clear {
				dry = false
			}
		}
		if dry {
			g.Seed = seed
			break
		}
	}
	sp := SpeciesByID("cosmos")
	for i, v := range []int{0, 1} {
		if err := g.Plant(i, sp, v, now); err != nil {
			t.Fatal(err)
		}
		g.Plots[i].Growth, g.Plots[i].Matured = 1, true
		g.Plots[i].MaturedAt = now
		g.Plots[i].Moisture = 1
	}
	return g, now
}

func TestOnlyFlowersWithMatesGetPollen(t *testing.T) {
	g, now := twoCosmos(t)
	g.Gold = 100
	if err := g.Plant(14, SpeciesByID("cosmos"), 2, now); err != nil { // far from the others
		t.Fatal(err)
	}
	g.Plots[14].Growth = 1

	at := now
	for i := 0; i < 24*4*3; i++ {
		at = at.Add(15 * time.Minute)
		g.Advance(at)
		for j := range g.Plots {
			g.Plots[j].Moisture = 1
		}
	}
	if g.Plots[0].Pollen == nil && g.Plots[1].Pollen == nil {
		t.Fatal("two flowering neighbours never cross-pollinated in three bright days")
	}
	if g.Plots[14].Pollen != nil {
		t.Error("a flower with no neighbour of its kind got pollen")
	}
	if g.Plots[0].Pollen != nil && *g.Plots[0].Pollen != g.Plots[1].Genes() {
		t.Error("bed 1's pollen is not bed 2's genome")
	}
	if g.Crossed == 0 {
		t.Error("crossings were not counted")
	}
}

func TestPollinationIsTheSameLiveOrCaughtUp(t *testing.T) {
	live, now := twoCosmos(t)
	caught, _ := twoCosmos(t)
	end := now.Add(72 * time.Hour)

	for at := now; at.Before(end); {
		at = at.Add(time.Hour)
		live.Advance(at)
	}
	caught.Advance(end)

	for i := 0; i < 2; i++ {
		a, b := live.Plots[i].Pollen, caught.Plots[i].Pollen
		if (a == nil) != (b == nil) || (a != nil && *a != *b) {
			t.Errorf("bed %d: live pollen %v, caught-up pollen %v", i+1, a, b)
		}
	}
	if live.Crossed != caught.Crossed {
		t.Errorf("live garden crossed %d times, caught-up garden %d", live.Crossed, caught.Crossed)
	}
}

func TestPollinatorsFollowTheWeatherAndTheClock(t *testing.T) {
	cosmos, moon := SpeciesByID("cosmos"), SpeciesByID("moonflower")
	dry, wet := weatherFor(Sunny), weatherFor(Rain)

	if o := pollinatorOdds(cosmos, dry, Summer, phaseNoon); o <= 0 {
		t.Error("no bees on a sunny day")
	}
	if pollinatorOdds(cosmos, wet, Summer, phaseNoon) >= pollinatorOdds(cosmos, dry, Summer, phaseNoon)/4 {
		t.Error("rain should keep most of the bees in")
	}
	if pollinatorOdds(cosmos, dry, Winter, phaseNoon) >= pollinatorOdds(cosmos, dry, Summer, phaseNoon) {
		t.Error("winter should have fewer insects")
	}
	if pollinatorOdds(cosmos, dry, Summer, phaseNight) != 0 {
		t.Error("an ordinary flower is not pollinated at night")
	}
	if pollinatorOdds(moon, dry, Summer, phaseNight) <= 0 {
		t.Error("the moths should work the night flowers")
	}
}

func TestShutFlowersAreNotPollinated(t *testing.T) {
	tulip := SpeciesByID("tulip")
	p := &Plot{SpeciesID: "tulip", Growth: 1}
	if flowerOpen(tulip, p, Spring, phaseNight) {
		t.Error("a tulip is shut at night")
	}
	if !flowerOpen(tulip, p, Spring, phaseNoon) {
		t.Error("a tulip is open at noon")
	}
	moon := SpeciesByID("moonflower")
	mp := &Plot{SpeciesID: "moonflower", Growth: 1}
	if flowerOpen(moon, mp, Summer, phaseNoon) {
		t.Error("a moonflower is shut by day")
	}
	if flowerOpen(tulip, &Plot{SpeciesID: "tulip", Growth: 0.5}, Spring, phaseNoon) {
		t.Error("a bud is not open")
	}
	if flowerOpen(tulip, &Plot{SpeciesID: "tulip", Growth: 1, Spent: true}, Spring, phaseNoon) {
		t.Error("a spent plant is not open")
	}
}

func TestHandPollination(t *testing.T) {
	g, now := twoCosmos(t)
	// A day with no rain, in daylight.
	for d := 0; d < 40; d++ {
		if w := g.Weather(now); w.Kind != Rain && w.Kind != Storm && w.Kind != Snow {
			break
		}
		now = now.Add(24 * time.Hour)
	}
	if err := g.HandPollinate(0, 0, now); err == nil {
		t.Error("a flower pollinated itself")
	}
	if err := g.HandPollinate(0, 1, now); err != nil {
		t.Fatalf("hand pollination: %v", err)
	}
	if g.Plots[0].Pollen == nil || *g.Plots[0].Pollen != g.Plots[1].Genes() {
		t.Error("the donor's pollen was not carried across")
	}
	if g.HandCrossed != 1 {
		t.Errorf("%d hand crossings counted", g.HandCrossed)
	}

	g.Gold = 100
	if err := g.Plant(2, SpeciesByID("zinnia"), 0, now); err != nil {
		t.Fatal(err)
	}
	g.Plots[2].Growth = 1
	if err := g.HandPollinate(0, 2, now); err == nil {
		t.Error("pollen took between two different species")
	}
}

func TestHandPollinationNeedsAnOpenFlower(t *testing.T) {
	g, now := twoCosmos(t)
	g.Gold = 100
	for i := 2; i < 4; i++ {
		if err := g.Plant(i, SpeciesByID("tulip"), 0, now); err != nil {
			t.Fatal(err)
		}
		g.Plots[i].Growth = 1
	}
	day := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	night := time.Date(2026, 5, 10, 2, 0, 0, 0, time.UTC)
	if w := g.Weather(day); w.Kind == Rain || w.Kind == Storm || w.Kind == Snow {
		t.Skip("the test day is wet in this garden")
	}
	if err := g.HandPollinate(2, 3, day); err != nil {
		t.Errorf("two open tulips at noon: %v", err)
	}
	if err := g.HandPollinate(2, 3, night); err == nil {
		t.Error("hand pollination worked on tulips shut for the night")
	}
}

func TestHandPollinationKeyFlow(t *testing.T) {
	g, now := twoCosmos(t)
	for d := 0; d < 40; d++ {
		if w := g.Weather(now); w.Kind != Rain && w.Kind != Storm && w.Kind != Snow {
			break
		}
		now = now.Add(24 * time.Hour)
	}
	g.LastTick = now
	m := newModel(g, "/tmp/garden.json", now)
	m.width, m.height = 100, 40
	var cur tea.Model = m
	press := func(k string) {
		var msg tea.KeyMsg
		if k == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		cur, _ = cur.Update(msg)
	}
	press("x")
	if !cur.(model).pollinating {
		t.Fatalf("x did not start pollinating: %q", cur.(model).status)
	}
	press("l") // to bed 2
	press("enter")
	got := cur.(model)
	if got.pollinating || got.g.Plots[0].Pollen == nil {
		t.Errorf("the cross was not made: pollinating=%v status=%q", got.pollinating, got.status)
	}
	if got.cursor != 0 {
		t.Errorf("the cursor should return to the pollinated flower, is on %d", got.cursor)
	}

	// Escape puts the brush down without crossing anything.
	press("x")
	press("esc")
	if cur.(model).pollinating {
		t.Error("esc did not cancel pollinating")
	}
}
