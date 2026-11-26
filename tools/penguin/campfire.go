package main

import "math"

// campfireFilm: a fire in a clearing ringed by pines, the Milky Way overhead,
// embers rising from the flames and the occasional shooting star.
type campfireFilm struct {
	w, h, hz int
	fx, fy   int
	stars    []star
	trees    []pine
	embers   []flake
}

func newCampfireFilm(w, h int) *campfireFilm {
	f := &campfireFilm{w: w, h: h, hz: int(math.Round(float64(h) * 0.52))}
	f.fx, f.fy = w/2, f.hz+int(0.50*float64(h-f.hz))
	f.stars = newStars(w, f.hz)
	// a dense treeline along the horizon, taller at the edges
	for i := 0; i < w/3; i++ {
		x := int(rnd(i, 1, 111) * float64(w))
		edge := math.Abs(float64(x)/float64(w)-0.5) * 2
		ht := int(float64(f.hz)*(0.16+0.32*edge*edge) + 3*rnd(i, 2, 111))
		f.trees = append(f.trees, pine{x, f.hz + 1 + int(2*rnd(i, 3, 111)), ht, rnd(i, 4, 111)})
	}
	for i := 0; i < 46; i++ {
		f.embers = append(f.embers, flake{x: rnd(i, 1, 121), v: 0.5 + rnd(i, 2, 121), ph: rnd(i, 3, 121)})
	}
	return f
}

var fireRamp = []col{
	{0.05, 0.0, 0.0}, {0.45, 0.04, 0.0}, {0.85, 0.2, 0.02}, {1.0, 0.45, 0.04},
	{1.0, 0.72, 0.15}, {1.0, 0.90, 0.45}, {1.0, 0.98, 0.80},
}

func fireColour(h float64) col {
	h = clamp01(h) * float64(len(fireRamp)-1)
	i := int(h)
	if i >= len(fireRamp)-1 {
		return fireRamp[len(fireRamp)-1]
	}
	return mix(fireRamp[i], fireRamp[i+1], h-float64(i))
}

// flicker is the fire's brightness at time t.
func flicker(t float64) float64 {
	return 0.86 + 0.10*math.Sin(t*9.1) + 0.06*math.Sin(t*14.3+1) + 0.08*(noise1(t*7, 971)-0.5)
}

// light is how much firelight reaches a cell.
func (f *campfireFilm) light(c, r int, t float64) float64 {
	R := float64(f.w) * 0.34
	dx := float64(c - f.fx)
	dy := float64(r-f.fy) * 2
	return math.Exp(-(dx*dx+dy*dy)/(R*R)) * flicker(t)
}

func (f *campfireFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	hz := f.hz
	cv.fillRows(0, hz, []stop{{0, hex("#02030b")}, {0.6, hex("#0a1030")}, {1, hex("#1b1d44")}}, func(c, r int, base col) col {
		// the Milky Way: a soft diagonal band
		u := (float64(c)*0.45 - float64(r)*1.0 - float64(f.w)*0.1)
		band := math.Exp(-(u * u) / 380)
		n := fbm(float64(c)*0.12, float64(r)*0.25, 4, 981)
		return base.add(col{0.28, 0.30, 0.50}.mul(band * (0.2 + 0.7*n)))
	})
	twinkle(cv, f.stars, t)
	f.shootingStar(cv, t)

	// ground
	g := float64(f.h - hz)
	for r := hz; r < f.h; r++ {
		v := (float64(r) + 0.5 - float64(hz)) / g
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.2/(v+0.1), 1.0/(v+0.06), 3, 983)
			base := mix(col{0.04, 0.07, 0.07}, col{0.09, 0.11, 0.09}, n).mul(0.8 + 0.4*v)
			ch := ' '
			if rnd(c, r, 7) < 0.10 {
				ch = rune("\"',;"[(c+r)%4])
			}
			cv.paint(c, r, ch, base.mul(2).lift(0.02), base)
		}
	}
	// trees behind, tent to the side, then log seats and stones
	for _, p := range f.trees {
		f.tree(cv, p, t)
	}
	f.tent(cv, t)
	f.logs(cv, t)

	// firelight over everything painted so far
	for r := 0; r < f.h; r++ {
		for c := 0; c < f.w; c++ {
			l := f.light(c, r, t)
			if l < 0.01 {
				continue
			}
			p := cv.at(c, r)
			warm := col{1.0, 0.52, 0.16}
			p.bg = p.bg.add(p.bg.mulc(warm).mul(l * 3.6)).add(warm.mul(l * l * 0.30))
			p.fg = p.fg.add(p.fg.mulc(warm).mul(l * 3.2)).add(warm.mul(l * l * 0.30))
		}
	}
	f.flames(cv, t)
	f.drawEmbers(cv, t)
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

func (f *campfireFilm) shootingStar(cv *canvas, t float64) {
	k := int(t / 17)
	start := float64(k)*17 + 4 + 8*rnd(k, 1, 131)
	d := t - start
	if d < 0 || d > 1.1 {
		return
	}
	x0 := float64(f.w) * (0.15 + 0.5*rnd(k, 2, 131))
	y0 := 1.0 + float64(f.hz)*0.25*rnd(k, 3, 131)
	for i := 0; i < 9; i++ {
		u := d*30 - float64(i)*1.1
		if u < 0 {
			continue
		}
		cv.ink(int(x0+u*1.5), int(y0+u*0.35), []rune{'*', '+', '-', '.', '.', '.', '.', '.', '.'}[i], col{0.95, 0.97, 1}.mul(1-float64(i)/10))
	}
}

func (f *campfireFilm) tree(cv *canvas, p pine, t float64) {
	dark := col{0.015, 0.03, 0.04}
	tiers := max(3, p.ht/3)
	for r := 0; r < p.ht; r++ {
		y := p.base - p.ht + r + 1
		u := float64(r) / float64(p.ht)
		tu := fract(u * float64(tiers))
		half := (0.6 + u*float64(p.ht)*0.36) * (0.55 + 0.45*tu)
		for c := p.x - int(half); c <= p.x+int(half); c++ {
			ch := rune("%#&@"[(c+r)%4])
			if c == p.x-int(half) {
				ch = '/'
			} else if c == p.x+int(half) {
				ch = '\\'
			}
			b := dark.add(col{0.01, 0.02, 0.02}.mul(p.tone))
			cv.paint(c, y, ch, b.mul(2.2), b)
		}
	}
}

func (f *campfireFilm) tent(cv *canvas, t float64) {
	cx := int(0.20 * float64(f.w))
	base := f.hz + int(0.45*float64(f.h-f.hz))
	hw := max(7, f.w/11)
	ht := hw * 2 / 3
	for r := 0; r < ht; r++ {
		half := (hw * (r + 1)) / ht
		y := base - ht + r + 1
		for c := cx - half; c <= cx+half; c++ {
			// the side facing the fire is warmer
			rel := float64(c-cx) / float64(hw)
			cloth := mix(col{0.05, 0.08, 0.12}, col{0.13, 0.15, 0.20}, clamp01(0.4+rel*0.7))
			ch := ':'
			if c == cx-half {
				ch = '/'
			} else if c == cx+half {
				ch = '\\'
			}
			cv.paint(c, y, ch, cloth.mul(1.6), cloth)
		}
	}
	// the open flap, dark inside
	for r := 0; r < ht-1; r++ {
		half := max(0, (2*(r+1))/ht)
		for c := cx - half; c <= cx+half; c++ {
			cv.paint(c, base-ht+r+2, ' ', col{}, col{0.01, 0.01, 0.02})
		}
	}
}

func (f *campfireFilm) logs(cv *canvas, t float64) {
	wood := col{0.22, 0.12, 0.07}
	// crossed logs under the flames
	for k := -5; k <= 5; k++ {
		cv.paint(f.fx+k, f.fy+1, '=', wood.mul(2), wood)
		y := f.fy + 1 - (1 - int(math.Abs(float64(k))+0.5)/4)
		if k > -4 && k < 4 {
			cv.paint(f.fx+k, y, rune("\\/"[(k+10)%2]), wood.mul(2.2), wood.mul(1.1))
		}
	}
	// ring of stones
	for i := 0; i < 15; i++ {
		a := float64(i) / 15 * 2 * math.Pi
		x := float64(f.fx) + math.Cos(a)*9.5
		y := float64(f.fy+1) + math.Sin(a)*2.5
		g := col{0.18, 0.18, 0.22}
		cv.paint(int(math.Round(x)), int(math.Round(y)), 'o', g.mul(2), g)
	}
	// two log seats
	for _, sx := range []float64{-0.24, 0.26} {
		x0 := f.fx + int(sx*float64(f.w))
		for k := -4; k <= 4; k++ {
			cv.paint(x0+k, f.fy+3, '=', wood.mul(1.8), wood.mul(0.9))
			cv.paint(x0+k, f.fy+4, '_', wood.mul(1.2), wood.mul(0.5))
		}
	}
}

func (f *campfireFilm) flames(cv *canvas, t float64) {
	H := float64(f.h) * 0.30
	W := float64(f.w) * 0.085
	for r := f.fy; r > f.fy-int(H*1.15); r-- {
		y := float64(f.fy-r) / H // 0 at the logs .. 1 at the tips
		for c := f.fx - int(W*2); c <= f.fx+int(W*2); c++ {
			sway := math.Sin(t*3.1+y*4.5)*0.9*y + math.Sin(t*6.7+y*9)*0.4*y
			dx := (float64(c) - float64(f.fx) - sway) / (W * (1 - y*0.85))
			if math.Abs(dx) >= 1 {
				continue
			}
			n := fbm(float64(c)*0.5, float64(r)*0.32-t*4.2, 3, 991)
			tongue := 0.55 + 0.65*noise1(float64(c)*0.55+t*2.2, 995)
			heat := (1 - math.Abs(dx)*math.Abs(dx)) * (1 - y*0.95) * (0.45 + 1.0*n) * (0.6 + tongue*0.8)
			heat *= 1.15
			if heat < 0.10 {
				continue
			}
			col0 := fireColour(heat)
			ch := rune(" .:^*sS#$@"[min(9, int(heat*9))])
			p := cv.at(c, r)
			if heat > 0.45 {
				cv.paint(c, r, ch, col0.mul(1.1).clamp(), col0.mul(0.75))
			} else {
				p.ch, p.fg = ch, col0
				p.bg = p.bg.add(col0.mul(heat * 1.1))
			}
		}
	}
}

func (f *campfireFilm) drawEmbers(cv *canvas, t float64) {
	rise := float64(f.h) * 0.62
	for i, e := range f.embers {
		cyc := 3.5 + 3*e.v
		ph := fract(t/cyc + e.ph)
		y := float64(f.fy) - 2 - ph*rise
		x := float64(f.fx) + (e.x-0.5)*float64(f.w)*0.10 + math.Sin(t*0.9+float64(i)*1.3)*ph*6 + ph*ph*5*(e.v-0.9)
		a := 1 - ph
		ch := '.'
		if a > 0.7 {
			ch = '*'
		} else if a > 0.4 {
			ch = '\''
		}
		c := fireColour(0.45 + 0.5*a)
		cv.ink(int(x), int(y), ch, c)
		cv.glow(int(x), int(y), c, 0.10*a)
	}
}
