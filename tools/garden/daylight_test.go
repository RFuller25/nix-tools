package main

import (
	"testing"
	"time"
)

func at(month time.Month, hour, min int) time.Time {
	return time.Date(2026, month, 15, hour, min, 0, 0, time.UTC)
}

func TestPhasesFollowTheSun(t *testing.T) {
	cases := []struct {
		when time.Time
		want phase
	}{
		{at(time.July, 2, 0), phaseNight},
		{at(time.July, 5, 30), phaseDawn},
		{at(time.July, 9, 0), phaseMorning},
		{at(time.July, 13, 0), phaseNoon},
		{at(time.July, 17, 0), phaseAfternoon},
		{at(time.July, 20, 45), phaseDusk},
		{at(time.July, 23, 30), phaseNight},

		// Winter days are short: the same clock time is a different world.
		{at(time.January, 18, 30), phaseNight},
		{at(time.January, 17, 0), phaseDusk},
		{at(time.January, 12, 0), phaseNoon},
	}
	for _, c := range cases {
		if got := phaseAt(c.when); got != c.want {
			t.Errorf("%s in %s was %s, want %s", c.when.Format("15:04"), c.when.Month(), got, c.want)
		}
	}
}

func TestSummerDaysAreLongerThanWinterOnes(t *testing.T) {
	for _, s := range []Season{Spring, Summer, Autumn, Winter} {
		rise, set := daylight(s)
		if rise >= set {
			t.Errorf("%s: the sun sets at %.1f before rising at %.1f", s, set, rise)
		}
	}
	_, summerSet := daylight(Summer)
	summerRise, _ := daylight(Summer)
	winterRise, winterSet := daylight(Winter)
	if (summerSet - summerRise) <= (winterSet - winterRise) {
		t.Error("summer days should be longer than winter ones")
	}
}

func TestEveryPhaseHasAFace(t *testing.T) {
	for p := phaseNight; p <= phaseDusk; p++ {
		if p.String() == "" || p.Glyph() == "" {
			t.Errorf("phase %d has no name or glyph", p)
		}
	}
	if !phaseNight.Dark() {
		t.Error("night should be dark")
	}
	if phaseNoon.Dark() {
		t.Error("midday should not be dark")
	}
}

func TestColoursAreParsedBothWays(t *testing.T) {
	if _, ok := parseColor("#ff8000"); !ok {
		t.Error("a hex colour should parse")
	}
	if _, ok := parseColor("114"); !ok {
		t.Error("a terminal palette number should parse")
	}
	if _, ok := parseColor("chartreuse"); ok {
		t.Error("a colour name should not parse")
	}
	if _, ok := parseColor("999"); ok {
		t.Error("a number outside the palette should not parse")
	}

	// The corners of the 6x6x6 cube and the grey ramp.
	if c := xterm(16); c != (rgb{0, 0, 0}) {
		t.Errorf("colour 16 should be black, got %+v", c)
	}
	if c := xterm(231); c != (rgb{1, 1, 1}) {
		t.Errorf("colour 231 should be white, got %+v", c)
	}
	if c := xterm(244); c.r != c.g || c.g != c.b {
		t.Errorf("the grey ramp produced a colour: %+v", c)
	}
}

func TestLightChangesWithTheHour(t *testing.T) {
	green := "114"

	if got := tint(green, phaseNoon); got != green {
		t.Errorf("midday changed the colour to %q; it should leave it alone", got)
	}

	night, _ := parseColor(tint(green, phaseNight))
	base, _ := parseColor(green)
	if night.r+night.g+night.b >= base.r+base.g+base.b {
		t.Error("night should be darker than day")
	}
	if night.b <= night.r {
		t.Error("night should be blue")
	}

	dawn, _ := parseColor(tint(green, phaseDawn))
	if dawn.r <= base.r {
		t.Error("dawn should be warmer than plain daylight")
	}

	// A colour it cannot read is passed through rather than mangled.
	if got := tint("rebeccapurple", phaseNight); got != "rebeccapurple" {
		t.Errorf("an unreadable colour came back as %q", got)
	}
}

func TestTintPaletteLightsEveryPart(t *testing.T) {
	pal := Palette{Stem: "71", Leaf: "77", Bloom: "218", Accent: "223"}
	lit := tintPalette(pal, phaseDusk)
	if lit.Stem == pal.Stem || lit.Leaf == pal.Leaf || lit.Bloom == pal.Bloom || lit.Accent == pal.Accent {
		t.Errorf("dusk left part of the palette unlit: %+v", lit)
	}
	if same := tintPalette(pal, phaseNoon); same != pal {
		t.Errorf("midday should change nothing, got %+v", same)
	}
}

func TestFlowersThatCloseForTheNight(t *testing.T) {
	crocus := SpeciesByID("crocus")
	open := &Plot{SpeciesID: "crocus", Growth: 1, Matured: true, PH: 6.5}

	if stage, _ := bareAppearance(crocus, open, Spring, phaseNoon); stage != StageMature {
		t.Error("a crocus should be open at midday")
	}
	if stage, _ := bareAppearance(crocus, open, Spring, phaseNight); stage != StageBud {
		t.Error("a crocus should be closed at night")
	}

	// A plant that does not close stays as it is.
	rose := SpeciesByID("rose")
	steady := &Plot{SpeciesID: "rose", Growth: 1, Matured: true, PH: 6.5}
	if stage, _ := bareAppearance(rose, steady, Summer, phaseNight); stage != StageMature {
		t.Error("a rose does not shut at night")
	}
}

func TestFlowersThatWaitForTheDark(t *testing.T) {
	moon := SpeciesByID("moonflower")
	p := &Plot{SpeciesID: "moonflower", Growth: 1, Matured: true, PH: 6.5}

	if stage, _ := bareAppearance(moon, p, Summer, phaseNoon); stage != StageBud {
		t.Error("a moonflower should be shut at midday")
	}
	if stage, _ := bareAppearance(moon, p, Summer, phaseDusk); stage != StageMature {
		t.Error("a moonflower should be open at dusk")
	}
	if stage, _ := bareAppearance(moon, p, Summer, phaseNight); stage != StageMature {
		t.Error("a moonflower should be open at night")
	}

	if got := state(moon, p, Summer, phaseNoon); got == "" {
		t.Error("a shut moonflower should say what it is waiting for")
	}
}

// The night plants must actually be in the catalogue, or the tables quietly
// do nothing.
func TestNightTablesNameRealPlants(t *testing.T) {
	for _, table := range []map[string]bool{closers, nightFlowers, nightScented} {
		for id := range table {
			if SpeciesByID(id) == nil {
				t.Errorf("%q is listed as a night plant but is not in the catalogue", id)
			}
		}
	}
	// A plant cannot both shut for the night and wait for it.
	for id := range closers {
		if nightFlowers[id] {
			t.Errorf("%s both closes at night and opens at night", id)
		}
	}
}

func TestNightScentIsAnnouncedAfterDark(t *testing.T) {
	g := newTestGarden(at(time.July, 22, 0))
	m := newModel(g, "", at(time.July, 22, 0))
	m.g.Plots[0] = Plot{SpeciesID: "honeysuckle", Growth: 1, Matured: true, PH: 6.5}

	if got := m.nightScent(); got == "" {
		t.Error("a honeysuckle in flower said nothing at ten at night")
	}

	m.now = at(time.July, 12, 0)
	if got := m.nightScent(); got != "" {
		t.Errorf("midday reported night scent: %q", got)
	}
}
