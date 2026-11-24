package main

import "math"

// col is a linear-ish RGB colour with components nominally in 0..1. Values may
// overshoot while lighting is being accumulated; they are clamped on output.
type col struct{ r, g, b float64 }

func rgb(r, g, b float64) col { return col{r, g, b} }

// hex parses "#rrggbb".
func hex(s string) col {
	v := func(i int) float64 {
		var n int
		for _, ch := range s[i : i+2] {
			n <<= 4
			switch {
			case ch >= '0' && ch <= '9':
				n |= int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				n |= int(ch-'a') + 10
			case ch >= 'A' && ch <= 'F':
				n |= int(ch-'A') + 10
			}
		}
		return float64(n) / 255
	}
	return col{v(1), v(3), v(5)}
}

func (a col) add(b col) col      { return col{a.r + b.r, a.g + b.g, a.b + b.b} }
func (a col) mul(k float64) col  { return col{a.r * k, a.g * k, a.b * k} }
func (a col) mulc(b col) col     { return col{a.r * b.r, a.g * b.g, a.b * b.b} }
func (a col) lum() float64       { return 0.2126*a.r + 0.7152*a.g + 0.0722*a.b }
func (a col) lift(k float64) col { return col{a.r + k, a.g + k, a.b + k} }

func (a col) clamp() col {
	return col{clamp01(a.r), clamp01(a.g), clamp01(a.b)}
}

func mix(a, b col, t float64) col {
	return col{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func smoothstep(e0, e1, x float64) float64 {
	if e0 == e1 {
		if x < e0 {
			return 0
		}
		return 1
	}
	t := clamp01((x - e0) / (e1 - e0))
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// stop is one colour stop of a vertical gradient.
type stop struct {
	at float64
	c  col
}

func gradient(stops []stop, v float64) col {
	if v <= stops[0].at {
		return stops[0].c
	}
	for i := 1; i < len(stops); i++ {
		if v <= stops[i].at {
			a, b := stops[i-1], stops[i]
			return mix(a.c, b.c, (v-a.at)/(b.at-a.at))
		}
	}
	return stops[len(stops)-1].c
}

func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

var _ = math.Pi

func pow(x, y float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Pow(x, y)
}
