package main

import (
	"testing"
	"time"
)

// runShow lets the playback run until the player is asked to answer.
func runShow(t *testing.T, e *echo) {
	t.Helper()
	for i := 0; i < 2000 && e.phase == echoShowing; i++ {
		e.advance(echoTickMS * time.Millisecond)
	}
	if e.phase != echoListening {
		t.Fatalf("playback never handed over (phase %d)", e.phase)
	}
}

func TestEchoShowsEachStepThenListens(t *testing.T) {
	e := newEcho(1)
	e.Start()
	if len(e.seq) != 1 || e.phase != echoShowing {
		t.Fatalf("a new game starts with %d steps in phase %d", len(e.seq), e.phase)
	}
	var shown []int
	last := -1
	for i := 0; i < 2000 && e.phase == echoShowing; i++ {
		e.advance(echoTickMS * time.Millisecond)
		if e.lit >= 0 && e.lit != last {
			shown = append(shown, e.lit)
		}
		last = e.lit
	}
	if len(shown) != 1 || shown[0] != e.seq[0] {
		t.Errorf("showed %v for sequence %v", shown, e.seq)
	}
}

func TestEchoGrowsWhileYouKeepUp(t *testing.T) {
	e := newEcho(2)
	e.Start()
	for round := 1; round <= 12; round++ {
		runShow(t, e)
		if len(e.seq) != round {
			t.Fatalf("round %d has %d steps", round, len(e.seq))
		}
		// Pressing during playback is ignored, so it cannot be cheated.
		for _, p := range append([]int(nil), e.seq...) {
			e.press(p)
		}
		if e.phase == echoOver {
			t.Fatalf("a correct playback of round %d was called wrong", round)
		}
	}
	if e.best != 12 {
		t.Errorf("best run %d after twelve rounds", e.best)
	}
	if e.litFor() >= echoLitMS*time.Millisecond {
		t.Error("playback did not speed up")
	}
}

func TestEchoEndsOnAWrongPad(t *testing.T) {
	e := newEcho(3)
	e.Start()
	runShow(t, e)
	e.press(e.seq[0])
	runShow(t, e)
	wrong := (e.seq[0] + 1) % echoPads
	e.press(wrong)
	if e.phase != echoOver || e.missed != e.seq[0] {
		t.Fatalf("a wrong pad gave phase %d, missed %d", e.phase, e.missed)
	}
	score, lower, rec := e.Result()
	if score != 1 || lower || !rec {
		t.Errorf("result %d lower=%v rec=%v, want 1 point recorded", score, lower, rec)
	}
	e.press(e.seq[0])
	if e.phase != echoOver {
		t.Error("the game carried on after it ended")
	}
}

func TestEchoPressesDuringPlaybackAreIgnored(t *testing.T) {
	e := newEcho(4)
	e.Start()
	e.press((e.seq[0] + 1) % echoPads)
	if e.phase != echoShowing {
		t.Error("a press while the game was still showing counted")
	}
}
