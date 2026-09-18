package main

import "math/rand"

// The arcade's music: a four-bar chiptune loop of bass and arpeggio over
// A minor, kept low in the mix so it can run all session without wearing thin.

// chord is one bar: a bass root and the notes the arpeggio walks through.
type chord struct {
	root  float64
	tones []float64
}

// progression is Am – F – C – G, the backbone of more or less everything.
var progression = []chord{
	{45, []float64{57, 60, 64, 69}}, // Am
	{41, []float64{53, 57, 60, 65}}, // F
	{48, []float64{60, 64, 67, 72}}, // C
	{43, []float64{55, 59, 62, 67}}, // G
}

// chiptune builds the looping theme. bpm sets the pace: the menu idles, the
// games push along a little faster.
func chiptune(rate int, seed int64, bpm float64) *sequencer {
	rng := rand.New(rand.NewSource(seed))
	eighth := int64(float64(rate) * (60 / bpm) / 2)
	step := 0

	return &sequencer{
		step: func(at int64) ([]voice, int64) {
			bar := (step / 8) % len(progression)
			beat := step % 8
			c := progression[bar]
			var out []voice

			// Bass on the first and third beats of the bar.
			if beat == 0 || beat == 4 {
				out = append(out, voice{
					wave: waveSquare, freq: note(c.root), gain: 0.2,
					env: envelope{attack: 0.005, decay: 0.08, sustain: 0.6, hold: 0.1, release: 0.12},
				})
			}

			// Arpeggio through the chord, turning around at the top.
			idx := beat % len(c.tones)
			if beat >= len(c.tones) {
				idx = len(c.tones) - 1 - idx
			}
			out = append(out, voice{
				wave: waveTriangle, freq: note(c.tones[idx] + 12), gain: 0.10,
				env: envelope{attack: 0.005, decay: 0.05, sustain: 0.5, hold: 0.05, release: 0.1},
			})

			// Now and then a higher note drops in, so the loop does not sit
			// perfectly still for an hour.
			if beat == 6 && rng.Float64() < 0.4 {
				out = append(out, voice{
					wave: waveTriangle, freq: note(c.tones[len(c.tones)-1] + 24), gain: 0.07,
					env: envelope{attack: 0.01, decay: 0.08, sustain: 0.4, hold: 0.05, release: 0.15},
				})
			}

			step++
			return out, eighth
		},
	}
}

// Tempos: unhurried in the menu, brisker once a game starts.
const (
	menuBPM = 104
	gameBPM = 132
)
