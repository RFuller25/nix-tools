package main

import (
	"math"
	"testing"
	"time"
)

func TestEnvelopeShape(t *testing.T) {
	e := envelope{attack: 1, decay: 1, sustain: 0.5, hold: 2, release: 2}
	if got := e.total(); got != 6 {
		t.Errorf("total = %v, want 6", got)
	}

	cases := []struct {
		t, want float64
	}{
		{-1, 0},    // before the note
		{0, 0},     // silent at the very start
		{0.5, 0.5}, // halfway up the attack
		{1, 1},     // peak
		{1.5, 0.75},
		{2, 0.5},  // settled at the sustain level
		{3, 0.5},  // still holding
		{4, 0.5},  // release begins
		{5, 0.25}, // halfway through the release
		{6, 0},    // finished
		{99, 0},
	}
	for _, c := range cases {
		if got := e.at(c.t); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("at(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestEnvelopeHandlesZeroStages(t *testing.T) {
	e := envelope{sustain: 1, hold: 1}
	if got := e.at(0); got != 1 {
		t.Errorf("a note with no attack should open at full level, got %v", got)
	}
	if got := e.at(2); got != 0 {
		t.Errorf("past the end should be silent, got %v", got)
	}
}

func TestWaveformsStayInRange(t *testing.T) {
	for _, w := range []wave{waveSine, waveTriangle, waveSquare, waveSaw} {
		for i := 0; i <= 100; i++ {
			phase := float64(i) / 100
			if v := shape(w, phase); v < -1.0001 || v > 1.0001 {
				t.Errorf("wave %d at phase %.2f = %v, outside -1..1", w, phase, v)
			}
		}
	}
	if got := shape(waveSquare, 0.25); got != 1 {
		t.Errorf("square first half = %v, want 1", got)
	}
	if got := shape(waveSquare, 0.75); got != -1 {
		t.Errorf("square second half = %v, want -1", got)
	}
	if got := shape(waveTriangle, 0.5); math.Abs(got-1) > 1e-9 {
		t.Errorf("triangle peak = %v, want 1", got)
	}
}

func TestNoteFrequencies(t *testing.T) {
	cases := map[float64]float64{69: 440, 81: 880, 57: 220, 62: 293.6648}
	for n, want := range cases {
		if got := note(n); math.Abs(got-want) > 0.01 {
			t.Errorf("note(%v) = %.4f, want %.4f", n, got, want)
		}
	}
}

func TestMixerSilenceAndSound(t *testing.T) {
	m := newMixer(sampleRate)
	buf := make([]float64, 512)

	m.fill(buf)
	for i, v := range buf {
		if v != 0 {
			t.Fatalf("an idle mixer produced sound at sample %d: %v", i, v)
		}
	}

	m.play(voice{wave: waveSine, freq: 440, gain: 1, env: envelope{sustain: 1, hold: 1}})
	m.fill(buf)
	loud := false
	for _, v := range buf {
		if math.Abs(v) > 0.01 {
			loud = true
		}
		if v < -1 || v > 1 {
			t.Fatalf("sample %v escaped the -1..1 range", v)
		}
	}
	if !loud {
		t.Error("a playing note produced no sound")
	}
}

func TestMixerDropsFinishedVoices(t *testing.T) {
	m := newMixer(sampleRate)
	m.play(voice{wave: waveSine, freq: 440, gain: 1, env: envelope{sustain: 1, hold: 0.01}})

	buf := make([]float64, 1024)
	for i := 0; i < 20; i++ {
		m.fill(buf)
	}
	m.mu.Lock()
	left := len(m.voices)
	m.mu.Unlock()
	if left != 0 {
		t.Errorf("%d finished voices are still being mixed", left)
	}
}

func TestSequencerSchedulesOverTime(t *testing.T) {
	m := newMixer(sampleRate)
	calls := 0
	m.setSequencer(&sequencer{step: func(at int64) ([]voice, int64) {
		calls++
		return []voice{{wave: waveSine, freq: 440, gain: 0.5, env: envelope{sustain: 1, hold: 0.05}}},
			int64(sampleRate / 2) // a note every half second
	}})

	buf := make([]float64, sampleRate) // one second
	m.fill(buf)
	if calls != 2 {
		t.Errorf("a second of music asked the sequencer %d times, want 2", calls)
	}

	m.setSequencer(nil)
	before := calls
	m.fill(buf)
	if calls != before {
		t.Error("the sequencer kept playing after being stopped")
	}
}

// A sequencer that asks to be called again immediately must not lock the
// mixer up in an endless loop.
func TestSequencerCannotSpin(t *testing.T) {
	m := newMixer(sampleRate)
	calls := 0
	m.setSequencer(&sequencer{step: func(at int64) ([]voice, int64) {
		calls++
		return nil, 0
	}})

	done := make(chan struct{})
	go func() {
		m.fill(make([]float64, 256))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("fill() never returned for a sequencer with no delay")
	}
	if calls > 2 {
		t.Errorf("a zero-delay sequencer was called %d times in one block", calls)
	}
}

func TestSequencerStartsAtTheCurrentMoment(t *testing.T) {
	m := newMixer(sampleRate)
	m.fill(make([]float64, 4096)) // let the clock run on a while

	var firstAt int64 = -1
	m.setSequencer(&sequencer{step: func(at int64) ([]voice, int64) {
		if firstAt < 0 {
			firstAt = at
		}
		return nil, int64(sampleRate)
	}})
	m.fill(make([]float64, 256))
	if firstAt != 4096 {
		t.Errorf("music started at sample %d, want the mixer's current position 4096", firstAt)
	}
}

func TestSoftClip(t *testing.T) {
	cases := map[float64]float64{0: 0, 0.5: 0.5, -0.5: -0.5, 2: 1, -2: -1}
	for in, want := range cases {
		if got := softClip(in); got != want {
			t.Errorf("softClip(%v) = %v, want %v", in, got, want)
		}
	}
}
