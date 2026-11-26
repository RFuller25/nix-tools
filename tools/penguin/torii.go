package main

import "math"

// toriiFilm: a vermilion torii standing in still water at dusk, Fuji behind
// it, cherry blossom framing the shot and petals on the wind.
type toriiFilm struct {
	w, h, hz int
	fuji     []*layer
	stars    []star
	petals   []flake
	boughs   []bough
	seed     uint32
}

// bough is a cloud of blossom around a point on a branch.
type bough struct{ x, y, rx, ry float64 }

var toriiSky = []stop{
	{0.00, hex("#2a2a6a")}, {0.35, hex("#7a5aa0")}, {0.62, hex("#e08aa4")},
	{0.85, hex("#ffb48a")}, {1.00, hex("#ffe0a8")},
}

var fujiPal = &mtnPal{
	rockLit: col{0.85, 0.58, 0.66}, rockSh: col{0.36, 0.34, 0.62},
	snowLit: col{1.00, 0.92, 0.92}, snowSh: col{0.66, 0.62, 0.90},
	haze: col{0.93, 0.74, 0.84}, fog: col{1.0, 0.80, 0.78}, rim: col{1, 0.96, 0.92}, glowX: 0.74, calm: true,
}

func newToriiFilm(w, h int) *toriiFilm {
	f := &toriiFilm{w: w, h: h, hz: int(math.Round(float64(h) * 0.58)), seed: 77}
	f.fuji = []*layer{
		newLayer(w, f.hz, []peak{{0.10, 0.20, 0.10, 1}, {0.30, 0.25, 0.16, 1}, {0.95, 0.20, 0.14, 1}}, 501, 0.55, 0.0, 0.4, true),
		newLayer(w, f.hz, []peak{{0.76, 0.44, 0.62, 0.95}}, 502, 0.22, 0.50, 0.05, false),
		newLayer(w, f.hz, []peak{{0.55, 0.40, 0.045, 1}, {0.95, 0.3, 0.07, 1}}, 503, 0.45, 0.0, 0.5, true),
	}
	f.stars = newStars(w, f.hz/2)
	n := w * h / 70
	for i := 0; i < n; i++ {
		f.petals = append(f.petals, flake{
			x: rnd(i, 1, 71) * float64(w), y: rnd(i, 2, 71) * float64(h),
			v: 1.2 + 2.6*rnd(i, 3, 71), ph: rnd(i, 4, 71) * 6.28,
		})
	}
	// blossom clouds along the branches (positions as fractions of the frame)
	W, H := float64(w), float64(h)
	branches := [][][2]float64{
		{{0.06, 0.56}, {0.03, 0.40}, {0.0, 0.28}},
		{{0.10, 0.46}, {0.17, 0.30}, {0.25, 0.20}},
		{{0.16, 0.34}, {0.19, 0.19}, {0.24, 0.07}},
		{{1.0, 0.0}, {0.90, 0.09}, {0.82, 0.12}},
		{{0.90, 0.09}, {0.93, 0.23}, {0.98, 0.31}},
	}
	for _, b := range branches {
		for i := 0; i+1 < len(b); i++ {
			for s := 0.0; s <= 1.0; s += 0.2 {
				x := lerp(b[i][0], b[i+1][0], s)
				y := lerp(b[i][1], b[i+1][1], s)
				f.boughs = append(f.boughs, bough{x * W, y * H, 0.032*W + 2, 0.050*H + 1})
			}
		}
	}
	return f
}

type cellmask struct {
	on    []bool
	cells []cell
}

func (f *toriiFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	hz := f.hz
	glowx := 0.74
	// sky
	cv.fillRows(0, hz, toriiSky, func(c, r int, base col) col {
		dx := (float64(c)/float64(f.w) - glowx) / 0.35
		dy := (float64(hz-r) / float64(hz)) / 0.5
		return base.add(col{1, 0.65, 0.4}.mul(0.30 * math.Exp(-(dx*dx + dy*dy))))
	})
	twinkle(cv, f.stars, t)
	// soft clouds
	for r := 2; r < hz-4; r++ {
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.028+t*0.006, float64(r)*0.26, 4, 811)
			a := smoothstep(0.58, 0.76, n) * smoothstep(0, 0.4, float64(r)/float64(hz)) * (1 - smoothstep(0.6, 0.9, float64(r)/float64(hz)))
			if a > 0.03 {
				p := cv.at(c, r)
				p.bg = mix(p.bg, col{1, 0.78, 0.80}, a*0.6)
				if a > 0.35 {
					p.ch, p.fg = '~', mix(p.bg, col{1, 0.92, 0.9}, 0.5)
				}
			}
		}
	}
	for _, l := range f.fuji {
		drawRange(cv, l, hz, fujiPal, 1)
	}
	// water
	g := float64(f.h - hz)
	cv.fillRows(hz, f.h, []stop{{0, hex("#f6c2a8")}, {0.2, hex("#cf9fbd")}, {0.6, hex("#7e72b4")}, {1, hex("#4a4688")}}, nil)
	reflect(cv, hz, t, 0.5, col{0.6, 0.5, 0.8}, 0.9, false)
	for r := hz; r < f.h; r++ {
		v := (float64(r) + 0.5 - float64(hz)) / g
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.09/(v+0.1), 1.2/(v+0.06)-t*0.35, 3, 833)
			if n > 0.62 {
				p := cv.at(c, r)
				p.ch = rune("-~=~"[(c+r)%4])
				p.fg = p.bg.mul(1.0 + 0.5*(n-0.6)*4).lift(0.04)
			}
		}
	}

	f.drawTorii(cv, t)
	f.drawBlossom(cv, t)
	f.drawPetals(cv, t)
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

// drawTorii builds the gate in an overlay, then lays it and its reflection
// onto the picture.
func (f *toriiFilm) drawTorii(cv *canvas, t float64) {
	W, hz := f.w, f.hz
	yb := hz + int(0.30*float64(f.h-hz)) // waterline at the feet
	ht := int(0.78 * float64(hz))
	yt := yb - ht
	cx := int(0.50 * float64(W))
	span := int(0.17 * float64(W))
	pw := max(4, int(0.040*float64(W)))
	if pw%2 == 0 {
		pw++
	}
	ov := make(map[[2]int]cell)
	black := col{0.07, 0.06, 0.08}
	put := func(c, r int, base col, ch rune) {
		ov[[2]int{c, r}] = cell{ch, base.mul(1.35).lift(0.04).clamp(), base}
	}
	shadeRed := func(rel, vy float64) col {
		// rel: -1 (lit left edge) .. 1 (shadowed right edge); vy: 0 top .. 1 foot
		c := mix(col{0.92, 0.30, 0.16}, col{0.50, 0.08, 0.08}, smoothstep(-0.6, 1, rel))
		return c.mul(1 - 0.28*vy)
	}
	tk := max(2, int(float64(ht)*0.085))
	curve := func(x float64) int { // ends of the roof sweep upward
		u := math.Abs(x-float64(cx)) / float64(span+pw+span/3)
		return int(math.Round(2.4 * u * u * u))
	}
	// pillars, slightly leaning inward
	for _, side := range []int{-1, 1} {
		for r := yt; r <= yb; r++ {
			vy := float64(r-yt) / float64(ht)
			lean := int(math.Round(float64(side) * float64(span) * 0.0 * vy))
			base := cx + side*(span-int(math.Round(float64(pw)*0.0))) + lean
			wid := pw - int(math.Round((1-vy)*1.0))
			for k := 0; k < wid; k++ {
				c := base - wid/2 + k
				rel := float64(k)/float64(max(1, wid-1))*2 - 1
				base0 := shadeRed(rel, vy)
				ch := '|'
				if k == wid-1 {
					ch = ':'
				}
				if r > yb-3 {
					base0 = black.mul(1 + 0.4*(1-rel)/2)
					ch = '#'
				}
				put(c, r, base0, ch)
			}
		}
	}
	// nuki (lower tie beam)
	yn := yt + tk + max(2, ht/7)
	for r := yn; r < yn+max(1, tk-1); r++ {
		for c := cx - span - pw; c <= cx+span+pw; c++ {
			rel := float64(r-yn) / float64(max(1, tk-1))
			put(c, r, mix(col{0.88, 0.26, 0.14}, col{0.55, 0.09, 0.09}, rel), '=')
		}
	}
	// gakuzuka (centre strut) with a little plaque
	for r := yt + tk; r < yn; r++ {
		for c := cx - 1; c <= cx+1; c++ {
			put(c, r, mix(col{0.7, 0.14, 0.1}, col{0.4, 0.06, 0.06}, float64(c-cx+1)/2), '|')
		}
	}
	// kasagi (top beam, swept up at the tips) capped by the black shimaki
	for c := cx - span - pw - span/3; c <= cx+span+pw+span/3; c++ {
		lift := curve(float64(c))
		for r := 0; r < tk; r++ {
			rr := yt + r - lift
			put(c, rr, mix(col{0.90, 0.28, 0.14}, col{0.52, 0.09, 0.09}, float64(r)/float64(max(1, tk-1))), '=')
		}
		put(c, yt-1-lift, black.mul(1.2), '=')
	}
	// lay it down, then its reflection
	for k, v := range ov {
		cv.paint(k[0], k[1], v.ch, v.fg, v.bg)
	}
	for k, v := range ov {
		r := 2*yb - k[1] + 1
		if r <= yb || r >= f.h {
			continue
		}
		wob := int(math.Round(math.Sin(float64(r)*1.3+t*1.5) * 0.9))
		c := k[0] + wob
		if !cv.in(c, r) {
			continue
		}
		d := float64(r-yb) / float64(f.h-yb)
		p := cv.at(c, r)
		a := 0.62 * (1 - 0.6*d)
		p.bg = mix(p.bg, v.bg.mul(0.8), a)
		p.ch, p.fg = ':', mix(p.bg, v.fg, 0.5)
	}
	// ripples around the feet
	for _, side := range []int{-1, 1} {
		fx := float64(cx + side*span)
		for i := 0; i < 3; i++ {
			ph := fract(t*0.22 + float64(i)/3 + float64(side)*0.1)
			rx := 2 + ph*float64(pw)*3.2
			for a := 0.0; a < 6.283; a += 0.12 {
				c := int(math.Round(fx + math.Cos(a)*rx))
				r := int(math.Round(float64(yb+1) + math.Sin(a)*rx*0.22))
				if r > yb && cv.in(c, r) {
					p := cv.at(c, r)
					if _, hit := ov[[2]int{c, r}]; !hit {
						p.ch, p.fg = '-', mix(p.bg, col{1, 0.92, 0.9}, 0.55*(1-ph))
					}
				}
			}
		}
	}
}

// blossom branches and trunks
func (f *toriiFilm) drawBlossom(cv *canvas, t float64) {
	W, H := float64(f.w), float64(f.h)
	bark := col{0.17, 0.10, 0.12}
	stroke := func(pts [][2]float64, th0, th1 float64) {
		total := 0.0
		for i := 0; i+1 < len(pts); i++ {
			total += math.Hypot((pts[i+1][0]-pts[i][0])*W, (pts[i+1][1]-pts[i][1])*H*2)
		}
		done := 0.0
		for i := 0; i+1 < len(pts); i++ {
			ax, ay, bx, by := pts[i][0]*W, pts[i][1]*H, pts[i+1][0]*W, pts[i+1][1]*H
			seg := math.Hypot(bx-ax, (by-ay)*2)
			for s := 0.0; s < seg; s += 0.35 {
				u := s / seg
				x, y := lerp(ax, bx, u), lerp(ay, by, u)
				th := lerp(th0, th1, (done+s)/total)
				for dy := -th / 2; dy <= th/2; dy += 0.5 {
					for dx := -th / 2; dx <= th/2; dx += 0.5 {
						if dx*dx+dy*dy*4 > th*th/4+0.2 {
							continue
						}
						c, r := int(math.Floor(x+dx)), int(math.Floor(y+dy/2))
						if cv.in(c, r) {
							lit := clamp01(0.5 - dx/th)
							cc := mix(bark, col{0.55, 0.30, 0.30}, lit*0.7)
							cv.paint(c, r, rune("#%&@"[(c*3+r)%4]), cc.mul(1.5), cc)
						}
					}
				}
			}
			done += seg
		}
	}
	stroke([][2]float64{{0.015, 1.04}, {0.05, 0.82}, {0.065, 0.64}, {0.10, 0.47}, {0.16, 0.35}, {0.19, 0.19}, {0.24, 0.07}}, 7, 1)
	stroke([][2]float64{{0.06, 0.60}, {0.03, 0.42}, {0.0, 0.28}}, 3, 1)
	stroke([][2]float64{{0.10, 0.47}, {0.17, 0.30}, {0.25, 0.20}}, 3, 1)
	stroke([][2]float64{{1.02, -0.02}, {0.90, 0.09}, {0.82, 0.12}}, 6, 1)
	stroke([][2]float64{{0.90, 0.09}, {0.93, 0.23}, {0.98, 0.31}}, 3, 1)

	pinks := []col{{1.0, 0.90, 0.93}, {0.99, 0.74, 0.84}, {0.94, 0.52, 0.70}, {0.82, 0.34, 0.56}}
	for r := 0; r < f.h; r++ {
		for c := 0; c < f.w; c++ {
			dens := 0.0
			for _, b := range f.boughs {
				dx := (float64(c) - b.x) / b.rx
				dy := (float64(r) - b.y) / b.ry
				if dx*dx+dy*dy < 6 {
					dens += math.Exp(-(dx*dx + dy*dy))
				}
			}
			if dens < 0.05 {
				continue
			}
			sway := math.Sin(t*0.5+float64(c)*0.05) * 0.4
			n := fbm(float64(c)*0.42+sway, float64(r)*0.85, 3, 901)
			v := dens * (0.35 + 1.05*n)
			if v < 0.70 {
				continue
			}
			tone := clamp01((n-0.35)*2.2 + (rnd(c, r, 5)-0.5)*0.5)
			pc := pinks[min(3, int(tone*4))]
			// a low sun from the right warms the right-hand side of each mass
			pc = pc.add(col{0.10, 0.04, 0}.mul(smoothstep(0.4, 1.2, v)))
			ch := rune(".o*@@"[min(4, int((v-0.70)*3.0+tone*1.5))])
			p := cv.at(c, r)
			p.ch = ch
			p.fg = pc.clamp()
			p.bg = mix(p.bg, pc.mul(0.6), clamp01(0.10+(v-0.70)*0.6))
		}
	}
}

func (f *toriiFilm) drawPetals(cv *canvas, t float64) {
	pink := []col{{1, 0.88, 0.92}, {0.98, 0.7, 0.82}, {1, 0.96, 0.96}}
	for i, p := range f.petals {
		x := wrapf(p.x+7*math.Sin(t*0.45+p.ph)-t*p.v*0.9, float64(f.w))
		y := wrapf(p.y+t*p.v*0.75, float64(f.h))
		c, r := int(x), int(y)
		if !cv.in(c, r) {
			continue
		}
		ch := '.'
		if p.v > 2.6 {
			ch = '*'
		} else if p.v > 1.9 {
			ch = ','
		}
		cv.ink(c, r, ch, pink[i%3])
	}
}
