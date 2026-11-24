package main

// The score is an original piece for the penguin: a slow, wistful E minor
// figure, square-wave lead over a triangle bass and arpeggio. Each lead bar is
// eight eighth-notes of MIDI note numbers, 0 being a rest. Swap the tables to
// change the tune.

// chord is one bar: a bass root and the notes the arpeggio walks through.
type chord struct {
	root  float64
	tones []float64
}

var progression = []chord{
	{40, []float64{52, 55, 59, 64}}, // Em
	{36, []float64{52, 55, 60, 64}}, // C
	{43, []float64{50, 55, 59, 62}}, // G
	{38, []float64{50, 54, 57, 62}}, // D
}

// lead is eight bars: the four chords twice, the second pass a variation.
var lead = [][8]float64{
	{71, 0, 74, 0, 76, 0, 74, 71},
	{72, 0, 76, 0, 79, 0, 76, 72},
	{71, 0, 74, 0, 79, 0, 74, 71},
	{69, 0, 74, 0, 78, 0, 74, 69},
	{76, 0, 74, 71, 69, 0, 71, 0},
	{72, 0, 74, 76, 72, 0, 67, 0},
	{74, 0, 71, 67, 71, 0, 74, 0},
	{78, 76, 74, 0, 69, 0, 0, 0},
}

const bpm = 108

// theme builds the endlessly repeating score.
func theme(rate int) *sequencer {
	eighth := int64(float64(rate) * (60 / bpm) / 2)
	step := 0
	return &sequencer{
		step: func(at int64) ([]voice, int64) {
			bar := (step / 8) % len(lead)
			beat := step % 8
			c := progression[bar%len(progression)]
			var out []voice

			if beat == 0 || beat == 4 {
				out = append(out, voice{
					wave: waveTriangle, freq: note(c.root), gain: 0.30,
					env: envelope{attack: 0.005, decay: 0.1, sustain: 0.7, hold: 0.25, release: 0.15},
				})
			}
			idx := beat % len(c.tones)
			if beat >= len(c.tones) {
				idx = len(c.tones) - 1 - idx%len(c.tones)
			}
			out = append(out, voice{
				wave: waveTriangle, freq: note(c.tones[idx] + 12), gain: 0.09,
				env: envelope{attack: 0.005, decay: 0.05, sustain: 0.5, hold: 0.05, release: 0.1},
			})
			if n := lead[bar][beat]; n > 0 {
				out = append(out, voice{
					wave: waveSquare, freq: note(n), gain: 0.10,
					env: envelope{attack: 0.008, decay: 0.07, sustain: 0.55, hold: 0.12, release: 0.14},
				})
			}
			step++
			return out, eighth
		},
	}
}
