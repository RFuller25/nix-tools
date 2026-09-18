package main

import (
	"math"
	"testing"
)

// render plays a set of notes through a mixer and returns the samples, so the
// tests can listen to what the arcade actually produces.
func render(seconds float64, vs ...voice) []float64 {
	m := newMixer(sampleRate)
	m.play(vs...)
	out := make([]float64, int(float64(sampleRate)*seconds))
	m.fill(out)
	return out
}

func peak(samples []float64) float64 {
	best := 0.0
	for _, v := range samples {
		best = math.Max(best, math.Abs(v))
	}
	return best
}

// Every effect must make a sound, and none may be loud enough to clip.
func TestEveryEffectSounds(t *testing.T) {
	effects := map[string][]voice{
		"move": sfxMove(), "select": sfxSelect(), "shift": sfxShift(),
		"rotate": sfxRotate(), "lock": sfxLock(), "hard drop": sfxHardDrop(),
		"one line": sfxLines(1), "four lines": sfxLines(4), "level up": sfxLevelUp(),
		"hold": sfxHold(), "game over": sfxGameOver(), "win": sfxWin(),
		"slide": sfxSlide(), "merge": sfxMerge(64), "eat": sfxEat(),
		"pick": sfxPick(), "drop": sfxDrop(), "swap": sfxSwap(),
		"reveal": sfxReveal(), "flag": sfxFlag(), "boom": sfxBoom(),
	}
	for name, vs := range effects {
		if len(vs) == 0 {
			t.Errorf("%s has no notes", name)
			continue
		}
		samples := render(2, vs...)
		p := peak(samples)
		if p < 0.005 {
			t.Errorf("%s is inaudible (peak %.4f)", name, p)
		}
		if p > 1 {
			t.Errorf("%s clips (peak %.4f)", name, p)
		}
	}
}

// Effects have to be short: they play under the music, not over it.
func TestEffectsAreShort(t *testing.T) {
	longest := map[string]float64{"game over": 0.9, "win": 1.2, "boom": 1.2}
	effects := map[string][]voice{
		"move": sfxMove(), "lock": sfxLock(), "hard drop": sfxHardDrop(),
		"four lines": sfxLines(4), "level up": sfxLevelUp(), "game over": sfxGameOver(),
		"win": sfxWin(), "merge": sfxMerge(2048), "boom": sfxBoom(),
		"reveal": sfxReveal(), "eat": sfxEat(),
	}
	for name, vs := range effects {
		limit, ok := longest[name]
		if !ok {
			limit = 0.6
		}
		for _, v := range vs {
			end := float64(v.start)/sampleRate + v.env.total()
			if end > limit {
				t.Errorf("%s runs for %.2fs, longer than its %.2fs budget", name, end, limit)
			}
		}
	}
}

// A bigger clear and a bigger tile should sound higher than a small one.
func TestBiggerEventsSoundBigger(t *testing.T) {
	if len(sfxLines(4)) <= len(sfxLines(1)) {
		t.Error("a four-line clear should be a longer phrase than a single")
	}

	low := sfxMerge(4)[0].freq
	high := sfxMerge(1024)[0].freq
	if high <= low {
		t.Errorf("merging to 1024 sounds at %.1fHz, merging to 4 at %.1fHz", high, low)
	}
}

func TestMergePitchIsBounded(t *testing.T) {
	// Absurd tile values must not run off the top of hearing.
	for _, v := range []int{2, 4, 2048, 65536, 1 << 30} {
		if f := sfxMerge(v)[0].freq; f < 100 || f > 4000 {
			t.Errorf("merging to %d sounds at %.1fHz, outside a sensible range", v, f)
		}
	}
}

func TestMusicIsRepeatableAndDoesNotClip(t *testing.T) {
	play := func(seed int64) []float64 {
		m := newMixer(sampleRate)
		m.setSequencer(chiptune(sampleRate, seed, gameBPM))
		out := make([]float64, sampleRate*4) // four seconds
		m.fill(out)
		return out
	}
	a, b := play(1), play(1)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the same seed played differently at sample %d", i)
		}
	}
	if p := peak(a); p > 1 {
		t.Errorf("the music clips (peak %.3f)", p)
	}
	if p := peak(a); p < 0.02 {
		t.Errorf("the music is inaudible (peak %.3f)", p)
	}
}

func TestMusicTempoChangesWithTheBPM(t *testing.T) {
	count := func(bpm float64) int {
		notes := 0
		seq := chiptune(sampleRate, 2, bpm)
		var pos int64
		for pos < int64(sampleRate)*8 { // eight seconds
			vs, delay := seq.step(pos)
			notes += len(vs)
			pos += delay
		}
		return notes
	}
	slow, fast := count(menuBPM), count(gameBPM)
	if fast <= slow {
		t.Errorf("the game theme played %d notes and the menu theme %d; the game should be busier", fast, slow)
	}
}

func TestNoiseIsNoisy(t *testing.T) {
	samples := render(0.2, voice{wave: waveNoise, freq: 200, gain: 1,
		env: envelope{sustain: 1, hold: 0.15}})
	// Noise should cross zero constantly, unlike a low tone.
	crossings := 0
	for i := 1; i < len(samples); i++ {
		if (samples[i-1] < 0) != (samples[i] < 0) {
			crossings++
		}
	}
	if crossings < 500 {
		t.Errorf("noise crossed zero only %d times; that is a tone, not a hiss", crossings)
	}
}
