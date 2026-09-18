package main

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func stormy() Weather { return weatherFor(Storm) }
func still() Weather  { return weatherFor(Fog) }

func TestStillAirDoesNotMovePlants(t *testing.T) {
	w := newWind(1)
	for col := 0; col < 5; col++ {
		if got := w.swayAt(col); got != 0 {
			t.Errorf("bed %d swayed by %v with no wind about", col, got)
		}
	}
	if w.blowing() {
		t.Error("a fresh garden reports wind before any gust")
	}
}

func TestGustsArriveAndPassThrough(t *testing.T) {
	w := newWind(2)

	frames := 0
	for !w.blowing() && frames < 2000 {
		w.advance(windTick.Seconds(), stormy(), 5)
		frames++
	}
	if !w.blowing() {
		t.Fatal("no gust arrived in a storm after 2000 frames")
	}

	// It must eventually blow itself out rather than hanging about forever.
	for i := 0; i < 2000 && w.blowing(); i++ {
		w.advance(windTick.Seconds(), still(), 5)
	}
	if w.blowing() {
		t.Error("a gust never died down")
	}
}

// A gust travels: the beds it reaches first should stir before the later ones.
func TestGustTravelsAcrossTheBeds(t *testing.T) {
	w := newWind(3)
	w.gusts = []gust{{pos: 0, speed: 3, strength: 1, rate: 1, width: 1.5, life: 5, age: 2.5}}

	peakFor := func(col int) float64 {
		w2 := windState{gusts: []gust{w.gusts[0]}}
		best := 0.0
		for i := 0; i < 400; i++ {
			w2.advance(0.02, still(), 6)
			best = math.Max(best, math.Abs(w2.swayAt(col)))
		}
		return best
	}
	// Both ends of the garden should feel it, since the gust crosses them all.
	if peakFor(0) == 0 || peakFor(5) == 0 {
		t.Error("a gust crossing the garden left some beds untouched")
	}

	// Right now the gust sits over bed 0, so bed 0 must be moving more than
	// the far end of the garden.
	near, far := math.Abs(w.swayAt(0)), math.Abs(w.swayAt(5))
	w.advance(0.05, still(), 6)
	near, far = math.Max(near, math.Abs(w.swayAt(0))), math.Max(far, math.Abs(w.swayAt(5)))
	if near <= far {
		t.Errorf("bed under the gust swayed %.3f, distant bed %.3f", near, far)
	}
}

func TestSwayStaysWithinRange(t *testing.T) {
	w := newWind(4)
	for i := 0; i < 5000; i++ {
		w.advance(windTick.Seconds(), stormy(), 5)
		for col := 0; col < 5; col++ {
			if s := w.swayAt(col); s < -1 || s > 1 {
				t.Fatalf("sway %v at bed %d is outside -1..1", s, col)
			}
		}
		if len(w.gusts) > 3 {
			t.Fatalf("%d gusts at once, want at most 3", len(w.gusts))
		}
	}
}

func TestWeatherSetsHowOftenItBlows(t *testing.T) {
	count := func(w Weather, seed int64) int {
		wind := newWind(seed)
		seen := 0
		had := false
		for i := 0; i < 4000; i++ {
			wind.advance(windTick.Seconds(), w, 5)
			if wind.blowing() && !had {
				seen++
			}
			had = wind.blowing()
		}
		return seen
	}
	storm, fog := count(stormy(), 5), count(still(), 5)
	if storm <= fog {
		t.Errorf("a storm produced %d gusts and fog %d; storms should blow harder", storm, fog)
	}
}

// Leaning must never push a plant out of its bed or lift it off the soil.
func TestSwayKeepsPlantsInsideTheirBed(t *testing.T) {
	for _, sp := range AllSpecies() {
		for stage := 0; stage < StageCount; stage++ {
			var base []string
			for _, sway := range []float64{-1, -0.5, 0, 0.5, 1} {
				lines := renderArt(sp, stage, cellInner, artHeight, sway)
				if len(lines) != artHeight {
					t.Fatalf("%s stage %s drew %d lines", sp.ID, stageNames[stage], len(lines))
				}
				for _, l := range lines {
					if got := lipgloss.Width(l); got != cellInner {
						t.Fatalf("%s leaning at %v drew a %d-wide line, want %d", sp.ID, sway, got, cellInner)
					}
				}
				if sway == 0 {
					base = lines
					continue
				}
				// The base of the plant is rooted: the bottom line never moves.
				if base != nil && lines[artHeight-1] != base[artHeight-1] {
					t.Errorf("%s stage %s slid along the ground when leaning", sp.ID, stageNames[stage])
				}
			}
		}
	}
}

func TestTallPlantsLeanFurtherThanTheirBase(t *testing.T) {
	sp := SpeciesByID("foxglove")
	upright := renderArt(sp, StageMature, cellInner, artHeight, 0)
	leaning := renderArt(sp, StageMature, cellInner, artHeight, 1)

	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	topShift := indent(leaning[0]) - indent(upright[0])
	baseShift := indent(leaning[artHeight-1]) - indent(upright[artHeight-1])

	if topShift <= 0 {
		t.Errorf("the top of the plant did not lean (shift %d)", topShift)
	}
	if baseShift != 0 {
		t.Errorf("the base of the plant moved by %d", baseShift)
	}
	if topShift <= baseShift {
		t.Error("the plant slid rather than bent")
	}
}
