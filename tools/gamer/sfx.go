package main

// The arcade's sound effects. Everything here is a handful of short notes:
// square and triangle waves for the bleeps, saw and noise for the thuds, all
// kept brief and fairly quiet so they sit over the music without shouting.

// blip is the workhorse: one note, quick in, quick out.
func blip(w wave, freq, dur, gain float64) voice {
	return voice{
		wave: w, freq: freq, gain: gain,
		env: envelope{attack: 0.004, decay: dur * 0.3, sustain: 0.7, hold: dur * 0.5, release: dur * 0.3},
	}
}

// glide is a note that slides from one pitch to another.
func glide(w wave, from, to, dur, gain float64) voice {
	return voice{
		wave: w, freq: from, glide: to, gain: gain,
		env: envelope{attack: 0.005, decay: dur * 0.2, sustain: 0.8, hold: dur * 0.5, release: dur * 0.3},
	}
}

// at delays a note within an effect, so several can make a little phrase.
func at(v voice, seconds float64) voice {
	v.start = int64(float64(sampleRate) * seconds)
	return v
}

// ── menu ───────────────────────────────────────────────────────────────────

func sfxMove() []voice { return []voice{blip(waveSquare, 620, 0.045, 0.16)} }
func sfxSelect() []voice {
	return []voice{
		blip(waveSquare, note(72), 0.06, 0.2),
		at(blip(waveSquare, note(79), 0.09, 0.2), 0.06),
	}
}

// ── tetris ─────────────────────────────────────────────────────────────────

func sfxShift() []voice  { return []voice{blip(waveSquare, 300, 0.03, 0.10)} }
func sfxRotate() []voice { return []voice{blip(waveTriangle, 540, 0.045, 0.18)} }
func sfxLock() []voice {
	return []voice{
		blip(waveTriangle, 150, 0.09, 0.26),
		blip(waveNoise, 200, 0.05, 0.07),
	}
}
func sfxHardDrop() []voice {
	return []voice{
		glide(waveSaw, 420, 110, 0.13, 0.22),
		blip(waveNoise, 180, 0.07, 0.10),
	}
}

// sfxLines rises further the more lines went at once, so a four-line clear
// sounds like the reward it is.
func sfxLines(n int) []voice {
	ladder := []float64{72, 76, 79, 84, 88} // a major triad climbing two octaves
	out := make([]voice, 0, n+1)
	for i := 0; i <= n && i < len(ladder); i++ {
		out = append(out, at(blip(waveSquare, note(ladder[i]), 0.1, 0.2), float64(i)*0.055))
	}
	return out
}

func sfxLevelUp() []voice {
	return []voice{
		blip(waveSquare, note(67), 0.09, 0.18),
		at(blip(waveSquare, note(71), 0.09, 0.18), 0.07),
		at(blip(waveSquare, note(74), 0.16, 0.2), 0.14),
		at(blip(waveTriangle, note(79), 0.22, 0.16), 0.21),
	}
}

func sfxHold() []voice {
	return []voice{
		blip(waveTriangle, note(69), 0.05, 0.16),
		at(blip(waveTriangle, note(64), 0.07, 0.16), 0.05),
	}
}

// ── endings ────────────────────────────────────────────────────────────────

func sfxGameOver() []voice {
	return []voice{
		glide(waveSquare, note(62), note(50), 0.55, 0.2),
		at(glide(waveSaw, note(50), note(38), 0.6, 0.14), 0.12),
	}
}

func sfxWin() []voice {
	notes := []float64{72, 76, 79, 84}
	out := make([]voice, 0, len(notes)+1)
	for i, n := range notes {
		out = append(out, at(blip(waveSquare, note(n), 0.12, 0.18), float64(i)*0.08))
	}
	out = append(out, at(voice{
		wave: waveTriangle, freq: note(88), gain: 0.16,
		env: envelope{attack: 0.01, decay: 0.18, sustain: 0.5, hold: 0.15, release: 0.4},
	}, 0.32))
	return out
}

// ── 2048 ───────────────────────────────────────────────────────────────────

func sfxSlide() []voice { return []voice{blip(waveTriangle, 220, 0.045, 0.12)} }

// sfxMerge climbs a whole-tone ladder with the value of the tile, so a big
// merge is audibly bigger than a small one.
func sfxMerge(value int) []voice {
	step := 0
	for v := value; v > 2; v /= 2 {
		step++
	}
	if step > 10 {
		step = 10
	}
	return []voice{
		blip(waveSquare, note(60+float64(step)*2), 0.09, 0.2),
		at(blip(waveTriangle, note(67+float64(step)*2), 0.1, 0.12), 0.05),
	}
}

// ── snake ──────────────────────────────────────────────────────────────────

func sfxEat() []voice {
	return []voice{
		blip(waveSquare, note(81), 0.045, 0.18),
		at(blip(waveSquare, note(88), 0.06, 0.18), 0.045),
	}
}

// ── hue ────────────────────────────────────────────────────────────────────

func sfxPick() []voice { return []voice{blip(waveTriangle, note(74), 0.05, 0.16)} }
func sfxDrop() []voice { return []voice{blip(waveTriangle, note(69), 0.05, 0.14)} }

// sfxSettle is the small click of a tile locking into its place: quieter than
// a swap, since several can land in a row.
func sfxSettle() []voice {
	return []voice{
		blip(waveTriangle, note(83), 0.04, 0.12),
		at(blip(waveTriangle, note(90), 0.06, 0.10), 0.04),
	}
}

func sfxSwap() []voice {
	return []voice{
		blip(waveTriangle, note(71), 0.05, 0.16),
		at(blip(waveTriangle, note(78), 0.07, 0.16), 0.05),
	}
}

// ── minesweeper ────────────────────────────────────────────────────────────

func sfxReveal() []voice { return []voice{blip(waveSquare, 900, 0.02, 0.08)} }
func sfxFlag() []voice {
	return []voice{
		blip(waveSquare, note(76), 0.035, 0.15),
		at(blip(waveSquare, note(83), 0.04, 0.12), 0.035),
	}
}
func sfxBoom() []voice {
	return []voice{
		{wave: waveNoise, freq: 120, gain: 0.3,
			env: envelope{attack: 0.005, decay: 0.35, sustain: 0.25, hold: 0.05, release: 0.35}},
		glide(waveSaw, 180, 35, 0.7, 0.2),
	}
}
