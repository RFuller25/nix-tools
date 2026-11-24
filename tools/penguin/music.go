package main

import (
	"strconv"
	"strings"
)

// The score, written as note names. Each token is a note with an optional
// length in eighth-notes after a colon ("C5", "G4:2"); "r" is a rest. Bass and
// lead are separate lines that must come out the same length, and the whole
// piece loops forever.
//
// Left hand: C C, E E, A A (and a G G to turn round), then four-on-the-floor
// eighths of C, E, A, F. Right hand: part 1 is a C pedal over a falling
// G-F-E (and the same over B); part 2 climbs E-E-E-C and settles on A. The
// last four bars are a riff on part 1.
const (
	bpm       = 140
	eighthsPB = 8 // eighth-notes per bar
)

func rep(s string, n int) string { return strings.TrimSpace(strings.Repeat(s+" ", n)) }

var (
	bassIntro = "C2:2 C2:2 E2:2 E2:2 A1:2 A1:2 G1:2 G1:2"
	bassMain  = rep("C2", 4) + " " + rep("E2", 4) + " " + rep("A1", 4) + " " + rep("F2", 4)

	leadPart1 = "C5 G4 C5 F4 C5 E4 C5:2 " + // bar 1
		"B4 G4 B4 F4 B4 E4 B4:2 " + // bar 2
		"C5 G4 C5 F4 C5 E4 C5:2" // bar 3
	leadPart2 = "E5 E5 E5 C5 B4:4 " + // bar 1
		"B4 B4 C5 A4:5 " + // bar 2
		"A4 A4 G4 A4 A4 G4 A4 G4 " + // bar 3
		"A4 G4 r:6" // bar 4
	leadRiff = "C5 G4 C5 A4 C5 G4 E5 C5 " +
		"B4 G4 B4 A4 B4 G4 D5 B4 " +
		"E5 E5 E5 C5 B4 B4 C5 A4 " +
		"A4 G4 A4 G4 E4 G4 C5:2"
)

// score is the full loop: 16 bars, bass and lead in step.
func score() (bass, lead string) {
	bass = bassIntro + " " + rep(bassMain, 3) + " " + rep(bassMain, 2) + " " + rep(bassMain, 2)
	lead = "r:16 " + leadPart1 + " " + leadPart1 + " " + leadPart2 + " " + leadRiff
	return
}

type event struct {
	at, dur int
	midi    float64
}

// parseLine reads a line of the score into events and its length in eighths.
func parseLine(s string) ([]event, int) {
	var out []event
	at := 0
	for _, tok := range strings.Fields(s) {
		dur := 1
		if i := strings.IndexByte(tok, ':'); i >= 0 {
			dur, _ = strconv.Atoi(tok[i+1:])
			tok = tok[:i]
		}
		if tok != "r" {
			out = append(out, event{at, dur, noteNumber(tok)})
		}
		at += dur
	}
	return out, at
}

// noteNumber turns "F#4" into a MIDI note number (C5 is 72).
func noteNumber(n string) float64 {
	semis := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
	v := semis[n[0]]
	i := 1
	for ; i < len(n) && (n[i] == '#' || n[i] == 'b'); i++ {
		if n[i] == '#' {
			v++
		} else {
			v--
		}
	}
	oct, _ := strconv.Atoi(n[i:])
	return float64(12*(oct+1) + v)
}

// theme builds the endlessly repeating score.
func theme(rate int) *sequencer {
	bs, ls := score()
	bass, n1 := parseLine(bs)
	lead, n2 := parseLine(ls)
	total := max(n1, n2)
	eighth := int64(float64(rate) * (60 / float64(bpm)) / 2)
	sec := 60.0 / float64(bpm) / 2

	bassAt := map[int]event{}
	for _, e := range bass {
		bassAt[e.at] = e
	}
	leadAt := map[int]event{}
	for _, e := range lead {
		leadAt[e.at] = e
	}
	hold := func(d int, f float64) float64 { return max(0.02, float64(d)*sec*f-0.06) }

	step := 0
	return &sequencer{
		step: func(at int64) ([]voice, int64) {
			i := step % total
			var out []voice
			if e, ok := bassAt[i]; ok {
				out = append(out, voice{
					wave: waveTriangle, freq: note(e.midi), gain: 0.34,
					env: envelope{attack: 0.004, decay: 0.05, sustain: 0.75, hold: hold(e.dur, 0.9), release: 0.05},
				})
			}
			if e, ok := leadAt[i]; ok {
				v := voice{
					wave: waveSquare, freq: note(e.midi), gain: 0.09,
					env: envelope{attack: 0.006, decay: 0.06, sustain: 0.6, hold: hold(e.dur, 0.85), release: 0.08},
				}
				out = append(out, v)
				// a faint echo three sixteenths behind
				echo := v
				echo.gain = 0.03
				echo.start = int64(float64(eighth) * 0.75)
				out = append(out, echo)
			}
			// a hat on the off-beats once the tune is under way
			if i >= 2*eighthsPB && i%2 == 1 {
				out = append(out, voice{
					wave: waveNoise, freq: 1, gain: 0.05,
					env: envelope{attack: 0.001, decay: 0.03, sustain: 0, hold: 0, release: 0.02},
				})
			}
			step++
			return out, eighth
		},
	}
}
