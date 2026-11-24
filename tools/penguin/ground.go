package main

import "math"

var groundStops = []stop{
	{0.00, hex("#f0b59f")},
	{0.10, hex("#d99fae")},
	{0.35, hex("#9d9cc9")},
	{0.70, hex("#6c78b0")},
	{1.00, hex("#4a5690")},
}

func groundAt(c, r, w, hz, h int, dawn float64) col {
	g := float64(h - hz)
	v := (float64(r) + 0.5 - float64(hz)) / g
	base := gradient(groundStops, v)
	// The glow's long reflection down the ice.
	dx := (float64(c)/float64(w) - glowX) / (0.10 + 0.38*v)
	refl := math.Exp(-dx*dx) * pow(1-v, 1.5)
	base = base.add(glowCol.mul(0.40 * refl * (0.4 + 0.6*dawn)))
	return base.mul(0.62 + 0.38*dawn)
}

func (s *scene) drawGround(cv *canvas, t, dawn float64) {
	g := float64(s.h - s.hz)
	for r := s.hz; r < s.h; r++ {
		v := (float64(r) + 0.5 - float64(s.hz)) / g
		for c := 0; c < s.w; c++ {
			bg := groundAt(c, r, s.w, s.hz, s.h, dawn)
			ch, fg := ' ', col{}
			// Wind-sculpted sastrugi, compressed toward the horizon.
			z := v + 0.025
			u := (float64(c) - float64(s.w)*0.5) / z * 0.050
			n := fbm(u, 1.1/z, 3, 401)
			switch {
			case n > 0.60:
				ch = rune("-=-~"[(c+r)%4])
				fg = bg.mul(1.0 + 0.30*(n-0.58)*8).lift(0.03)
				bg = bg.mul(0.97)
			case n < 0.33:
				ch = '_'
				fg = bg.mul(0.84)
			default:
				if rnd(c, r, 17) < 0.020*(1-v*0.5) {
					ch = '.'
					fg = bg.mul(1.18).lift(0.03)
				}
			}
			// Sparkling ice crystals.
			if rnd(c, r, 29) < 0.010 {
				tw := 0.5 + 0.5*math.Sin(t*2.1+rnd(c, r, 31)*6.28)
				if tw > 0.7 {
					ch = '*'
					if tw < 0.85 {
						ch = '+'
					}
					fg = mix(bg, rimWhite, 0.55*tw)
				}
			}
			cv.paint(c, r, ch, fg.clamp(), bg.clamp())
		}
	}
}
