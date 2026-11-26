package main

import "math"

// lighthouseFilm: a stormy night. A lighthouse on a cliff turns its beam over
// a heaving sea, rain blows across, lightning flickers and spray bursts at the
// foot of the rock.
type lighthouseFilm struct {
	w, h, hz int
	cliff    []float64 // top row of the rock per column (0 = none)
	lx, ly   int       // lantern position
	stars    []star
	rain     []flake
	spray    []flake
}

func newLighthouseFilm(w, h int) *lighthouseFilm {
	f := &lighthouseFilm{w: w, h: h, hz: int(math.Round(float64(h) * 0.56))}
	f.cliff = make([]float64, w)
	base := float64(f.hz) + float64(h-f.hz)*0.42
	for c := 0; c < w; c++ {
		x := float64(c) / float64(w)
		// a rocky headland on the right, broken by noise
		prof := smoothstep(0.50, 0.88, x)
		top := base - prof*float64(h)*0.20
		top -= (noise1(float64(c)*0.35, 931) - 0.5) * 3.5 * prof
		top -= (noise1(float64(c)*1.3, 932) - 0.5) * 1.5 * prof
		top += (1 - smoothstep(0, 0.35, prof)) * float64(h) * 0.5
		f.cliff[c] = top
	}
	f.lx = int(0.80 * float64(w))
	f.stars = newStars(w, f.hz)
	for i := 0; i < w*h/9; i++ {
		f.rain = append(f.rain, flake{x: rnd(i, 1, 99) * float64(w+40), y: rnd(i, 2, 99) * float64(h), v: 18 + 14*rnd(i, 3, 99), ph: rnd(i, 4, 99)})
	}
	for i := 0; i < 70; i++ {
		f.spray = append(f.spray, flake{x: float64(i), v: 0.5 + rnd(i, 5, 99), ph: rnd(i, 6, 99)})
	}
	return f
}

// flash is the lightning's brightness at time t: bursts every ten seconds or
// so, each a quick double flicker.
func flash(t float64) (float64, int) {
	k := int(math.Floor(t / 11))
	for _, kk := range []int{k, k - 1} {
		start := float64(kk)*11 + 3 + 6*rnd(kk, 1, 141)
		d := t - start
		if d < 0 || d > 1.2 {
			continue
		}
		v := math.Exp(-d*7) + 0.7*math.Exp(-math.Abs(d-0.28)*12)
		return clamp01(v), kk
	}
	return 0, k
}

func (f *lighthouseFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	hz := f.hz
	fl, fk := flash(t)

	// sky and a moon behind the cloud
	cv.fillRows(0, hz, []stop{{0, hex("#05070f")}, {0.6, hex("#10182a")}, {1, hex("#243048")}}, func(c, r int, base col) col {
		dx := (float64(c) - float64(f.w)*0.22) / 22
		dy := (float64(r) - float64(hz)*0.28) / 8
		return base.add(col{0.45, 0.55, 0.75}.mul(0.22 * math.Exp(-(dx*dx + dy*dy))))
	})
	twinkle(cv, f.stars, t)
	for r := 0; r < hz; r++ {
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.03-t*0.10, float64(r)*0.17+t*0.01, 5, 911)
			a := smoothstep(0.46, 0.70, n) * (0.5 + 0.5*smoothstep(0, 0.5, float64(r)/float64(hz)))
			if a > 0.04 {
				p := cv.at(c, r)
				p.bg = mix(p.bg, col{0.07, 0.09, 0.14}.add(col{0.5, 0.58, 0.75}.mul(fl*0.8)), a*0.85)
				if a > 0.30 {
					p.ch = rune("=~-~"[(c/2+r)%4])
					p.fg = p.bg.mul(1.4).lift(0.02 + 0.2*fl)
				}
			}
		}
	}
	if fl > 0.1 {
		f.bolt(cv, fk, fl)
	}

	// sea
	f.sea(cv, t, fl)

	// rock and tower
	f.drawCliff(cv, t, fl)
	f.drawTower(cv, t, fl)
	f.beam(cv, t)

	f.drawSpray(cv, t)
	f.drawRain(cv, t, fl)
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

func (f *lighthouseFilm) bolt(cv *canvas, k int, fl float64) {
	x := float64(f.w) * (0.15 + 0.5*rnd(k, 2, 141))
	for r := 0; r < f.hz+2; r++ {
		x += (rnd(k, r+10, 142) - 0.5) * 3.2
		for _, dx := range []int{0, 1} {
			ch := '|'
			if rnd(k, r+50, 143) < 0.3 {
				ch = '/'
			} else if rnd(k, r+50, 143) > 0.7 {
				ch = '\\'
			}
			cv.stamp(int(x)+dx, r, ch, col{0.95, 0.97, 1}, col{0.7, 0.8, 1}, 0.6*fl)
		}
		if rnd(k, r+90, 144) < 0.07 { // a short fork
			fx := x
			for q := 0; q < 6; q++ {
				fx += 1 + rnd(k, q, 145)
				cv.stamp(int(fx), r+q/2, '-', col{0.8, 0.88, 1}, col{}, 0)
			}
		}
	}
}

func (f *lighthouseFilm) sea(cv *canvas, t, fl float64) {
	hz := f.hz
	g := float64(f.h - hz)
	for r := hz; r < f.h; r++ {
		v := (float64(r) + 0.5 - float64(hz)) / g
		z := 1 / (v + 0.06)
		for c := 0; c < f.w; c++ {
			ph := z*2.6 - t*1.35 + math.Sin(float64(c)*0.07+z*0.9+t*0.2)*1.1 + vnoise(float64(c)*0.05, z*0.5, 951)*2.2
			w := math.Sin(ph)*0.5 + 0.5
			base := mix(col{0.10, 0.16, 0.26}, col{0.02, 0.05, 0.10}, smoothstep(0, 1, v))
			bg := mix(base, col{0.20, 0.27, 0.40}, w*0.35).add(col{0.35, 0.42, 0.60}.mul(fl * 0.5 * (0.4 + w)))
			// the moon's path
			mx := float64(f.w) * 0.22
			bg = bg.add(col{0.25, 0.30, 0.42}.mul(math.Exp(-math.Pow((float64(c)-mx)/(3+10*v), 2)) * 0.5 * (1 - v*0.5) * w))
			ch, fg := ' ', col{}
			if w > 0.90 && rnd(c, r, 5) < 0.75 {
				ch = rune("~-=~"[(c+r)%4])
				fg = bg.mul(1.8).lift(0.08)
				if w > 0.985 {
					ch = '*'
					fg = col{0.8, 0.88, 0.95}
				}
			}
			cv.paint(c, r, ch, fg.clamp(), bg.clamp())
		}
	}
}

func (f *lighthouseFilm) drawCliff(cv *canvas, t, fl float64) {
	for c := 0; c < f.w; c++ {
		top := f.cliff[c]
		if top > float64(f.h) {
			continue
		}
		for r := int(top); r < f.h; r++ {
			depth := float64(r) - top
			n := fbm(float64(c)*0.30, float64(r)*0.6, 3, 961)
			lit := clamp01(0.25 + 0.6*(n-0.4) + fl*0.6)
			base := mix(col{0.05, 0.06, 0.09}, col{0.28, 0.32, 0.40}, lit*0.7)
			ch := ' '
			if n > 0.55 {
				ch = rune(":;/'"[(c*3+r)%4])
			}
			if depth < 1.2 {
				base = base.mul(1.4).add(col{0.04, 0.05, 0.08})
				ch = '_'
			}
			cv.paint(c, r, ch, base.mul(1.7).lift(0.03).clamp(), base)
		}
	}
}

func (f *lighthouseFilm) tower() (cx, base, top, halfBase, halfTop int) {
	cx = f.lx
	base = int(f.cliff[min(f.w-1, cx)]) + 1
	ht := int(float64(f.h) * 0.33)
	top = base - ht
	halfBase = max(4, f.w/26)
	halfTop = max(2, halfBase*5/9)
	f.ly = top - 2
	return
}

func (f *lighthouseFilm) drawTower(cv *canvas, t, fl float64) {
	cx, base, top, hb, ht := f.tower()
	for r := top; r <= base; r++ {
		u := float64(r-top) / float64(max(1, base-top))
		half := float64(ht) + (float64(hb)-float64(ht))*u
		band := ((r - top) / 3) % 2
		for c := cx - int(half); c <= cx+int(half); c++ {
			rel := (float64(c-cx)/half + 1) / 2 // 0 left .. 1 right
			body := col{0.90, 0.88, 0.84}
			if band == 1 {
				body = col{0.72, 0.14, 0.12}
			}
			// lit from the left by the moon, dark on the right
			shade := 0.30 + 0.75*(1-rel)
			body = body.mul(shade * (0.75 + 0.5*fl)).add(col{0.04, 0.05, 0.08})
			ch := '|'
			if c == cx-int(half) {
				ch = '('
			} else if c == cx+int(half) {
				ch = ')'
			}
			cv.paint(c, r, ch, body.mul(1.35).lift(0.03).clamp(), body.clamp())
		}
	}
	// gallery and lantern room
	for c := cx - ht - 2; c <= cx+ht+2; c++ {
		cv.paint(c, top-1, '=', col{0.1, 0.1, 0.12}, col{0.03, 0.03, 0.05})
	}
	pulse := 0.85 + 0.15*math.Sin(t*3)
	for r := top - 3; r < top-1; r++ {
		for c := cx - ht + 1; c <= cx+ht-1; c++ {
			cv.paint(c, r, '#', col{1, 1, 0.85}.mul(pulse), col{1, 0.86, 0.45}.mul(pulse))
		}
	}
	for k := 0; k <= ht; k++ { // dome
		half := ht - k
		for c := cx - half; c <= cx+half; c++ {
			ch := '#'
			if c == cx-half {
				ch = '/'
			} else if c == cx+half {
				ch = '\\'
			}
			cv.paint(c, top-4-k, ch, col{0.5, 0.15, 0.12}, col{0.12, 0.04, 0.04})
		}
	}
	cv.paint(cx, top-5-ht, '|', col{0.8, 0.8, 0.8}, col{0.1, 0.1, 0.1})
	// door and a window
	cv.paint(cx, base-1, 'n', col{1, 0.8, 0.4}, col{0.4, 0.25, 0.1})
	cv.paint(cx, base-(base-top)/2, 'o', col{1, 0.9, 0.55}, col{0.5, 0.35, 0.15})
	f.ly = top - 2
}

// beam sweeps both ways from the lantern as the light turns; it flares
// toward the viewer when it points this way.
func (f *lighthouseFilm) beam(cv *canvas, t float64) {
	phi := t * 2 * math.Pi / 9.0
	lx, ly := float64(f.lx), float64(f.ly)
	for _, dir := range []float64{-1, 1} {
		cosp := math.Cos(phi) * dir
		if cosp <= 0.05 {
			continue
		}
		length := float64(f.w) * 0.85 * cosp
		for dx := 2.0; dx < length; dx += 0.5 {
			cx := lx + dir*dx
			spread := 0.8 + dx*0.07
			cy := ly + dx*0.05
			for dy := -spread; dy <= spread; dy += 0.5 {
				a := (1 - dx/length) * (1 - math.Abs(dy)/spread) * cosp * 0.55
				cv.glow(int(cx), int(math.Round(cy+dy)), col{1, 0.92, 0.65}, a)
			}
		}
	}
	// flare when it points at us
	toward := math.Max(0, math.Sin(phi))
	for dy := -7; dy <= 7; dy++ {
		for dx := -14; dx <= 14; dx++ {
			d := float64(dx*dx)/196 + float64(dy*dy)/49
			if d < 1 {
				cv.glow(f.lx+dx, int(ly)+dy, col{1, 0.9, 0.6}, (1-d)*(1-d)*(0.18+0.45*toward))
			}
		}
	}
}

func (f *lighthouseFilm) drawSpray(cv *canvas, t float64) {
	base := float64(int(f.cliff[int(0.7*float64(f.w))]))
	_ = base
	for i, s := range f.spray {
		// bursts break against the rock face
		cyc := 3.2 + 2.2*s.v
		ph := fract(t/cyc + s.ph)
		if ph > 0.55 {
			continue
		}
		u := ph / 0.55
		cx := int(0.62*float64(f.w)) + int(s.ph*float64(f.w)*0.2)
		cy := int(f.cliff[min(f.w-1, cx)]) + 1
		x := float64(cx) - u*(4+8*s.v) - math.Sin(float64(i))*2
		y := float64(cy) - (4*u-3.6*u*u)*(5+6*s.v)
		ch := '.'
		if u < 0.5 && s.v > 0.9 {
			ch = '*'
		}
		cv.ink(int(x), int(y), ch, mix(cv.at(max(0, min(f.w-1, int(x))), max(0, min(f.h-1, int(y)))).bg, col{0.9, 0.95, 1}, 0.8*(1-u)))
	}
	// foam running along the waterline at the base of the cliff
	for c := int(0.55 * float64(f.w)); c < f.w; c++ {
		top := int(f.cliff[c])
		if top >= f.h {
			continue
		}
		r := min(f.h-1, top+int(2*(math.Sin(t*1.7+float64(c)*0.3)+1)))
		if rnd(c, int(t*5), 77) < 0.5 {
			cv.ink(c, r, rune("~*-"[(c+int(t*3))%3]), col{0.85, 0.92, 1})
		}
	}
}

func (f *lighthouseFilm) drawRain(cv *canvas, t, fl float64) {
	H := float64(f.h)
	for _, p := range f.rain {
		y := wrapf(p.y+t*p.v, H)
		x := wrapf(p.x-(y-p.y)*0+(-t*p.v*0.35), float64(f.w+40)) - 20
		c, r := int(x), int(y)
		if !cv.in(c, r) {
			continue
		}
		cl := cv.at(c, r)
		cl.ch = '/'
		if p.v > 28 {
			cl.ch = '|'
		}
		cl.fg = mix(cl.bg, col{0.65, 0.75, 0.95}, 0.35+0.4*fl)
	}
}
