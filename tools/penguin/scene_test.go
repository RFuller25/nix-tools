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

func TestLoopWraps(t *testing.T) {
	sc := newScene(80, 24)
	a := sc.draw(3).render(termenv.Ascii)
	b := sc.draw(3 + loopLen).render(termenv.Ascii)
	if a != b {
		t.Fatal("loop does not wrap")
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

func TestFadeBookends(t *testing.T) {
	if fadeAt(0) != 0 || fadeAt(loopLen) != 0 || fadeAt(loopLen/2) < 0.99 {
		t.Fatal("fade should start and end black")
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
