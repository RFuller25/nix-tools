package main

import (
	"math"
	"sync"
)

// A very small software synthesiser. Every bleep and every bar of music is
// generated here as samples rather than shipped as audio files: no assets, no
// cgo, and it stays testable because nothing in this file touches a speaker.

const sampleRate = 44100

type wave int

const (
	waveSine wave = iota
	waveTriangle
	waveSquare
	waveSaw
	waveNoise
)

// envelope is the usual attack/decay/sustain/release shape, with hold being
// how long the note sits at its sustain level before letting go.
type envelope struct {
	attack  float64 // seconds
	decay   float64 // seconds
	sustain float64 // level, 0..1
	hold    float64 // seconds at the sustain level
	release float64 // seconds
}

func (e envelope) total() float64 {
	return e.attack + e.decay + e.hold + e.release
}

// at returns the envelope's level at t seconds into the note.
func (e envelope) at(t float64) float64 {
	switch {
	case t < 0:
		return 0
	case t < e.attack:
		if e.attack == 0 {
			return 1
		}
		return t / e.attack
	case t < e.attack+e.decay:
		if e.decay == 0 {
			return e.sustain
		}
		p := (t - e.attack) / e.decay
		return 1 + (e.sustain-1)*p
	case t < e.attack+e.decay+e.hold:
		return e.sustain
	case t < e.total():
		if e.release == 0 {
			return 0
		}
		p := (t - e.attack - e.decay - e.hold) / e.release
		return e.sustain * (1 - p)
	default:
		return 0
	}
}

// voice is a single note: a waveform, an envelope and a place in time.
type voice struct {
	wave  wave
	freq  float64 // starting frequency in Hz
	glide float64 // if non-zero, the frequency it slides to by the end
	gain  float64
	env   envelope

	// start is the absolute sample position the note begins at. Sequencers
	// fill this in relative to their step and the mixer offsets it.
	start int64

	phase float64
	rnd   uint32
}

// render adds the voice into buf, which covers samples [pos, pos+len(buf)).
// It reports whether the note has finished and can be dropped.
func (v *voice) render(buf []float64, pos int64, rate float64) bool {
	total := v.env.total()
	for i := range buf {
		t := float64(pos+int64(i)-v.start) / rate
		if t < 0 {
			continue
		}
		if t >= total {
			return true
		}
		freq := v.freq
		if v.glide > 0 && total > 0 {
			freq = v.freq + (v.glide-v.freq)*(t/total)
		}
		v.phase += freq / rate
		if v.phase > 1 {
			v.phase -= math.Floor(v.phase)
		}
		sample := shape(v.wave, v.phase)
		if v.wave == waveNoise {
			sample = v.noise()
		}
		buf[i] += v.gain * v.env.at(t) * sample
	}
	return float64(pos+int64(len(buf))-v.start)/rate >= total
}

// noise is a cheap pseudo-random source for percussion and explosions. It is
// an xorshift rather than a real random number generator: each voice makes the
// same hiss every time, which keeps the tests honest.
func (v *voice) noise() float64 {
	if v.rnd == 0 {
		v.rnd = 0x2545F491
	}
	v.rnd ^= v.rnd << 13
	v.rnd ^= v.rnd >> 17
	v.rnd ^= v.rnd << 5
	return float64(int32(v.rnd)) / float64(1<<31)
}

// shape turns a phase in 0..1 into a sample for the given waveform.
func shape(w wave, phase float64) float64 {
	switch w {
	case waveTriangle:
		if phase < 0.5 {
			return 4*phase - 1
		}
		return 3 - 4*phase
	case waveSquare:
		if phase < 0.5 {
			return 1
		}
		return -1
	case waveSaw:
		return 2*phase - 1
	default:
		return math.Sin(2 * math.Pi * phase)
	}
}

// note converts a MIDI note number to a frequency: 69 is concert A.
func note(n float64) float64 {
	return 440 * math.Pow(2, (n-69)/12)
}

// sequencer feeds the mixer notes over time. step is called with the sample
// position the notes should start at, and returns those notes plus how many
// samples to wait before being asked again.
type sequencer struct {
	next int64
	step func(at int64) ([]voice, int64)
}

// mixer sums the voices that are currently sounding.
type mixer struct {
	mu     sync.Mutex
	rate   int
	pos    int64
	voices []*voice
	seq    *sequencer
	gain   float64
}

func newMixer(rate int) *mixer {
	return &mixer{rate: rate, gain: 0.28}
}

// play schedules notes to start immediately.
func (m *mixer) play(vs ...voice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range vs {
		v := vs[i]
		v.start += m.pos
		m.voices = append(m.voices, &v)
	}
}

// setSequencer starts a piece of music, or stops it when given nil.
func (m *mixer) setSequencer(s *sequencer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s != nil {
		s.next = m.pos
	}
	m.seq = s
}

func (m *mixer) playing() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seq != nil
}

// fill renders the next len(out) samples, advancing the clock.
func (m *mixer) fill(out []float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range out {
		out[i] = 0
	}
	end := m.pos + int64(len(out))

	// Ask the sequencer for everything that starts inside this block.
	for m.seq != nil && m.seq.next < end {
		at := m.seq.next
		vs, delay := m.seq.step(at)
		for i := range vs {
			v := vs[i]
			v.start += at
			m.voices = append(m.voices, &v)
		}
		if delay <= 0 {
			delay = int64(m.rate) // a broken sequencer must not spin
		}
		m.seq.next = at + delay
	}

	live := m.voices[:0]
	for _, v := range m.voices {
		if done := v.render(out, m.pos, float64(m.rate)); !done {
			live = append(live, v)
		}
	}
	m.voices = live

	for i := range out {
		out[i] = softClip(out[i] * m.gain)
	}
	m.pos = end
}

// softClip keeps a busy mix from tearing into digital distortion.
func softClip(v float64) float64 {
	switch {
	case v > 1:
		return 1
	case v < -1:
		return -1
	}
	return v
}
