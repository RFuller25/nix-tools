package main

import (
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

func TestFramesRenderAtManySizes(t *testing.T) {
	for _, sz := range [][2]int{{60, 20}, {80, 24}, {120, 40}, {200, 60}} {
		sc := newScene(sz[0], sz[1])
		for _, ft := range []float64{0, 4, 9, 20, 40, 47} {
			cv := sc.draw(ft)
			out := cv.render(termenv.TrueColor)
			if n := strings.Count(out, "\n") + 1; n != sz[1] {
				t.Fatalf("%v t=%v: %d rows", sz, ft, n)
			}
		}
	}
}

func TestWalkerShrinksTowardHorizon(t *testing.T) {
	sc := newScene(120, 40)
	prev := 1e9
	for ft := tWalk0; ft <= tWalkEnd; ft += 2 {
		w := sc.walker(ft)
		if w.d > prev+1e-9 {
			t.Fatalf("walker moved toward the camera at t=%v", ft)
		}
		prev = w.d
	}
	if sc.walker(tWalkEnd).rows() > 2 {
		t.Fatal("walker should end as a speck")
	}
}

func TestFadeInOnly(t *testing.T) {
	if fadeAt(0) != 0 || fadeAt(1e6) < 0.99 {
		t.Fatal("fade should start black and stay up")
	}
}

func TestTo256(t *testing.T) {
	if to256(0, 0, 0) != 16 && to256(0, 0, 0) != 232 {
		t.Fatal("black")
	}
	if got := to256(255, 0, 0); got != 196 {
		t.Fatalf("red = %d", got)
	}
}

func TestWalkerFlitsForever(t *testing.T) {
	sc := newScene(120, 40)
	a := sc.draw(3600).render(termenv.Ascii)
	b := sc.draw(3600.4).render(termenv.Ascii)
	if !strings.ContainsAny(a, "ij") || a == b {
		t.Fatal("the distant walker should keep flitting between two glyphs")
	}
}

func TestThemeMakesSound(t *testing.T) {
	m := newMixer(sampleRate)
	m.setSequencer(theme(sampleRate))
	buf := make([]float64, 4096)
	loud := false
	for i := 0; i < 200 && !loud; i++ {
		m.fill(buf)
		for _, v := range buf {
			if v > 0.01 || v < -0.01 {
				loud = true
			}
		}
	}
	if !loud {
		t.Fatal("silent theme")
	}
}

func TestScoreLinesAgree(t *testing.T) {
	bs, ls := score()
	_, nb := parseLine(bs)
	_, nl := parseLine(ls)
	if nb != nl || nb != 16*eighthsPB {
		t.Fatalf("bass %d lead %d eighths, want %d each", nb, nl, 16*eighthsPB)
	}
	if noteNumber("C5") != 72 || noteNumber("F#4") != 66 || noteNumber("Bb3") != 58 {
		t.Fatal("note names")
	}
}
