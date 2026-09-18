package main

import (
	"math"
	"math/rand"
	"time"
)

// Wind blows through the garden in gusts: a gust starts off to the left, runs
// across the beds and fades out, so plants bend one after another rather than
// all together. Nothing about it is stored — it is weather you watch, not
// state you keep.

// windTick is how often the garden repaints for animation. Nine frames a
// second is plenty for a plant leaning over and back.
const windTick = 110 * time.Millisecond

type gust struct {
	pos      float64 // where it has got to, in bed columns
	speed    float64 // beds per second
	strength float64 // 0..1
	phase    float64 // where it is in its sway
	rate     float64 // sway cycles per second
	width    float64 // how many beds it covers at once
	age      float64 // seconds since it started
	life     float64 // seconds before it dies out
}

type windState struct {
	gusts []gust
	rng   *rand.Rand
}

func newWind(seed int64) windState {
	return windState{rng: rand.New(rand.NewSource(seed))}
}

// gustChance is how likely a new gust is on any given frame, by weather.
// Storms blow constantly; fog barely stirs.
func gustChance(w Weather) float64 {
	switch w.Kind {
	case Storm:
		return 0.14
	case Rain, Showers:
		return 0.07
	case Snow:
		return 0.05
	case Cloudy, Overcast:
		return 0.045
	case Sunny, Clear:
		return 0.03
	case Heatwave:
		return 0.015
	case Frost:
		return 0.02
	default: // Fog
		return 0.01
	}
}

// gustStrength is how hard the weather can blow.
func gustStrength(w Weather) (lo, hi float64) {
	switch w.Kind {
	case Storm:
		return 0.7, 1.0
	case Rain, Showers, Snow:
		return 0.4, 0.85
	case Fog, Heatwave:
		return 0.15, 0.4
	default:
		return 0.25, 0.7
	}
}

// advance moves the wind on by dt seconds across a garden cols beds wide.
func (w *windState) advance(dt float64, weather Weather, cols int) {
	if w.rng == nil {
		w.rng = rand.New(rand.NewSource(1))
	}

	live := w.gusts[:0]
	for _, g := range w.gusts {
		g.pos += g.speed * dt
		g.phase += g.rate * dt
		g.age += dt
		if g.age < g.life && g.pos < float64(cols)+g.width {
			live = append(live, g)
		}
	}
	w.gusts = live

	// A busy sky can have two gusts crossing the garden at once, but not ten.
	if len(w.gusts) < 3 && w.rng.Float64() < gustChance(weather) {
		lo, hi := gustStrength(weather)
		w.gusts = append(w.gusts, gust{
			pos:      -2,
			speed:    2.5 + w.rng.Float64()*3.5,
			strength: lo + w.rng.Float64()*(hi-lo),
			rate:     0.7 + w.rng.Float64()*0.6,
			width:    1.6 + w.rng.Float64()*2.4,
			life:     3 + w.rng.Float64()*4,
		})
	}
}

// blowing reports whether anything is stirring, for the status line.
func (w *windState) blowing() bool { return len(w.gusts) > 0 }

// strongest is the force of the liveliest gust, 0 when the air is still.
func (w *windState) strongest() float64 {
	best := 0.0
	for _, g := range w.gusts {
		best = math.Max(best, g.strength*g.fade())
	}
	return best
}

// fade eases a gust in and out so it never appears or vanishes abruptly.
func (g gust) fade() float64 {
	if g.life <= 0 {
		return 0
	}
	p := g.age / g.life
	switch {
	case p < 0.2:
		return p / 0.2
	case p > 0.75:
		return (1 - p) / 0.25
	default:
		return 1
	}
}

// swayAt is how far the plant in a given bed column is leaning right now,
// from -1 (bent left) to 1 (bent right).
func (w *windState) swayAt(col int) float64 {
	total := 0.0
	for _, g := range w.gusts {
		d := (float64(col) - g.pos) / math.Max(0.5, g.width)
		reach := math.Exp(-d * d) // gusts fall off smoothly either side
		total += g.strength * g.fade() * reach * math.Sin(2*math.Pi*g.phase)
	}
	if total > 1 {
		return 1
	}
	if total < -1 {
		return -1
	}
	return total
}
