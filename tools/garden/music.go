package main

import "math/rand"

// The garden's music: slow, unmetred and deliberately uneventful. A low drone
// holds the key while single notes from a major pentatonic scale drift over
// it, so there is never a wrong note and nothing ever demands attention.

// calmScale is D major pentatonic (D E F# A B) over two and a bit octaves,
// as MIDI note numbers.
var calmScale = []float64{62, 64, 66, 69, 71, 74, 76, 78, 81, 83}

func seconds(rate int, s float64) int64 { return int64(float64(rate) * s) }

// calmMusic builds the garden's ambient piece. The same seed always gives the
// same drift, which keeps it testable.
func calmMusic(rate int, seed int64) *sequencer {
	rng := rand.New(rand.NewSource(seed))
	sinceDrone := int64(1 << 40) // force a drone on the first step
	last := -1

	return &sequencer{
		step: func(at int64) ([]voice, int64) {
			var out []voice

			// Refresh the drone every twenty seconds or so, overlapping the
			// one before it so the floor never drops out.
			if sinceDrone > seconds(rate, 19) {
				sinceDrone = 0
				out = append(out, drone(62-24, 0.16, rate), drone(69-24, 0.11, rate))
			}

			// Pick a note, avoiding an immediate repeat.
			i := rng.Intn(len(calmScale))
			if i == last {
				i = (i + 1 + rng.Intn(len(calmScale)-1)) % len(calmScale)
			}
			last = i
			n := calmScale[i]

			hold := 0.6 + rng.Float64()*1.6
			out = append(out, voice{
				wave: waveSine,
				freq: note(n),
				gain: 0.15 + rng.Float64()*0.05,
				env: envelope{
					attack: 0.55, decay: 0.5, sustain: 0.75,
					hold: hold, release: 2.8 + rng.Float64(),
				},
			})

			// A quiet harmonic a fifth up, a beat behind, like a distant echo
			// of the note just played.
			if rng.Float64() < 0.45 {
				out = append(out, voice{
					wave:  waveTriangle,
					freq:  note(n + 7),
					gain:  0.05,
					start: seconds(rate, 0.35+rng.Float64()*0.4),
					env: envelope{
						attack: 0.7, decay: 0.4, sustain: 0.6,
						hold: hold * 0.6, release: 2.2,
					},
				})
			}

			delay := seconds(rate, 1.7+rng.Float64()*2.8)
			sinceDrone += delay
			return out, delay
		},
	}
}

// drone is the long, low note the piece rests on.
func drone(n, gain float64, rate int) voice {
	return voice{
		wave: waveSine,
		freq: note(n),
		gain: gain,
		env: envelope{
			attack: 4.5, decay: 2, sustain: 0.85,
			hold: 14, release: 6,
		},
	}
}
