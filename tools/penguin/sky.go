package main

import "math"

var skyStops = []stop{
	{0.00, hex("#070a24")},
	{0.35, hex("#1d1f5c")},
	{0.60, hex("#5a3a86")},
	{0.78, hex("#b3507f")},
	{0.90, hex("#ee8a74")},
	{1.00, hex("#ffc98c")},
}

var (
	glowCol = col{1.00, 0.62, 0.36}
	glowX   = 0.665 // the glow sits in the mountain pass
)

type star struct {
	x, y   int
	bright float64
	phase  float64
	rate   float64
}

func newStars(w, hz int) []star {
	var out []star
	n := w * hz / 38
	for i := 0; i < n; i++ {
		x := int(rnd(i, 1, 101) * float64(w))
		y := int(math.Pow(rnd(i, 2, 101), 1.35) * float64(hz) * 0.82)
		out = append(out, star{
			x: x, y: y,
			bright: 0.35 + 0.65*rnd(i, 3, 101),
			phase:  rnd(i, 4, 101) * 6.28,
			rate:   0.6 + 2.2*rnd(i, 5, 101),
		})
	}
	return out
}

// skyAt is the sky colour at a cell, before stars and clouds.
func skyAt(c, r, w, hz int, dawn float64) col {
	v := (float64(r) + 0.5) / float64(hz)
	u := (float64(c) + 0.5) / float64(w)
	base := gradient(skyStops, pow(v, 1.0+1.4*(1-dawn)))
	dx := (u - glowX) / 0.30
	dy := (1 - v) / 0.55
	g := math.Exp(-(dx*dx + dy*dy))
	core := math.Exp(-(dx*dx*5 + dy*dy*7))
	k := 0.35 + 0.65*dawn
	base = base.add(glowCol.mul((0.50*g + 0.55*core) * k))
	return base.mul(0.75 + 0.25*dawn)
}

func (s *scene) drawSky(cv *canvas, t, dawn float64) {
	for r := 0; r < s.hz; r++ {
		v := (float64(r) + 0.5) / float64(s.hz)
		for c := 0; c < s.w; c++ {
			bg := skyAt(c, r, s.w, s.hz, dawn)
			ch, fg := ' ', col{}
			// Long, slow wisps of cloud catching the first light.
			if v > 0.22 && v < 0.92 {
				n := fbm(float64(c)*0.030+t*0.010, float64(r)*0.30, 4, 211)
				band := smoothstep(0.18, 0.40, v) * (1 - smoothstep(0.72, 0.92, v))
				a := smoothstep(0.56, 0.74, n) * band
				if a > 0.02 {
					lit := col{1.0, 0.66, 0.62}
					bg = mix(bg, mix(bg, lit, 0.55), a*(0.35+0.65*dawn))
					if a > 0.30 {
						fg = mix(bg, lit.mul(1.1), 0.55)
						ch = rune("~-=~"[int(n*40)%4])
						if a > 0.7 {
							ch = '='
						}
					}
				}
			}
			cv.paint(c, r, ch, fg, bg)
		}
	}
	// Stars fade as the sky warms.
	for _, st := range s.stars {
		if !cv.in(st.x, st.y) {
			continue
		}
		cl := cv.at(st.x, st.y)
		if cl.ch != ' ' {
			continue
		}
		night := 1 - smoothstep(0.02, 0.30, cl.bg.lum())
		tw := 0.55 + 0.45*math.Sin(t*st.rate+st.phase)
		a := st.bright * tw * night
		if a < 0.08 {
			continue
		}
		ch := '.'
		switch {
		case a > 0.80:
			ch = '*'
		case a > 0.55:
			ch = '+'
		case a > 0.30:
			ch = '\''
		}
		cl.ch = ch
		cl.fg = mix(cl.bg, col{0.92, 0.93, 1.0}, clamp01(a*1.2))
	}
}
