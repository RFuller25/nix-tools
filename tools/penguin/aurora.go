package main

import "math"

// auroraFilm: a snowbound range under the northern lights, mirrored in a
// frozen lake, with pines and a cabin whose chimney smoke drifts up.
type auroraFilm struct {
	w, h, hz int
	layers   []*layer
	stars    []star
	pines    []pine
}

type pine struct {
	x, base, ht int
	tone        float64
}

var auroraPal = &mtnPal{
	rockLit: col{0.34, 0.42, 0.62}, rockSh: col{0.05, 0.07, 0.18},
	snowLit: col{0.80, 0.90, 1.00}, snowSh: col{0.14, 0.20, 0.44},
	haze: col{0.10, 0.20, 0.34}, fog: col{0.10, 0.24, 0.30}, rim: col{0.8, 0.98, 1.0}, glowX: 0.5,
}

func newAuroraFilm(w, h int) *auroraFilm {
	f := &auroraFilm{w: w, h: h, hz: int(math.Round(float64(h) * 0.60))}
	f.layers = []*layer{
		newLayer(w, f.hz, []peak{{0.12, 0.2, 0.34, 1.15}, {0.36, 0.22, 0.46, 1.2}, {0.62, 0.2, 0.40, 1.2}, {0.9, 0.2, 0.5, 1.15}}, 601, 0.60, 0.28, 0.34, false),
		newLayer(w, f.hz, []peak{{0.20, 0.2, 0.55, 1.25}, {0.47, 0.24, 0.80, 1.28}, {0.78, 0.2, 0.66, 1.25}, {1.0, 0.16, 0.44, 1.2}, {0.62, 0.05, 0.1, 1}}, 602, 0.16, 0.34, 0.44, false),
		newLayer(w, f.hz, []peak{{0.15, 0.3, 0.06, 1}, {0.5, 0.3, 0.08, 1}, {0.85, 0.3, 0.07, 1}}, 603, 0.35, 0, 0.5, true),
	}
	f.stars = newStars(w, f.hz)
	g := f.h - f.hz
	for i, x := range []float64{0.03, 0.10, 0.17, 0.86, 0.93, 0.98, 0.80, 0.24} {
		ht := int(float64(g) * (0.55 + 0.55*rnd(i, 1, 61)))
		if i > 5 {
			ht = int(float64(g) * 0.35)
		}
		f.pines = append(f.pines, pine{int(x * float64(w)), f.h - 1 - int(float64(g)*0.12*rnd(i, 2, 61)), ht, rnd(i, 3, 61)})
	}
	return f
}

// curtain returns the colour and strength of the aurora at a sky cell.
func (f *auroraFilm) curtain(c, r int, t float64) (col, float64) {
	var sum col
	tot := 0.0
	for k := 0; k < 3; k++ {
		kf := float64(k)
		x := float64(c)
		base := float64(f.hz)*(0.34+0.10*kf) + 3.2*math.Sin(x*0.045+t*0.22+kf*2.1) + 1.8*math.Sin(x*0.11-t*0.37+kf) + 0.9*math.Sin(x*0.23+t*0.6)
		span := float64(f.hz) * 0.36
		d := base - float64(r)
		var prof float64
		if d >= 0 {
			prof = math.Exp(-d / (span * 0.55))
		} else {
			prof = math.Exp(d * 1.3)
		}
		rays := 0.45 + 0.55*vnoise(x*0.55+kf*30+0.8*math.Sin(t*0.3+kf), t*0.35+kf*7, 701)
		rays *= 0.7 + 0.3*noise1(x*1.7+t*0.9+kf*11, 703)
		pulse := 0.65 + 0.35*math.Sin(t*0.45+x*0.02+kf*1.7)
		a := prof * rays * pulse * (0.9 - 0.18*kf)
		hf := clamp01(d / span)
		cc := mix(mix(col{0.20, 1.0, 0.50}, col{0.10, 0.85, 0.80}, smoothstep(0.15, 0.5, hf)), col{0.72, 0.30, 0.95}, smoothstep(0.55, 1.0, hf))
		if k == 1 {
			cc = mix(cc, col{1.0, 0.35, 0.6}, smoothstep(0.7, 1, hf)*0.6)
		}
		sum = sum.add(cc.mul(a))
		tot += a
	}
	if tot <= 0 {
		return col{}, 0
	}
	return sum.mul(1 / tot), clamp01(tot)
}

func (f *auroraFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	hz := f.hz
	cv.fillRows(0, hz, []stop{{0, hex("#02030c")}, {0.55, hex("#07112c")}, {1, hex("#0f2547")}}, nil)
	twinkle(cv, f.stars, t)
	col0 := make([]float64, f.w) // mean light per column, for tinting the snow
	for r := 0; r < hz; r++ {
		for c := 0; c < f.w; c++ {
			ac, a := f.curtain(c, r, t)
			if a < 0.03 {
				continue
			}
			p := cv.at(c, r)
			p.bg = p.bg.add(ac.mul(a * 1.0))
			if a > 0.22 {
				i := min(5, int(a*6))
				p.ch = rune(".:!|||"[i])
				if rnd(c+int(t*3), r, 9) < 0.5 && a < 0.6 {
					p.ch = ':'
				}
				p.fg = mix(p.bg, ac.mul(1.2).add(col{0.25, 0.25, 0.25}), 0.7).clamp()
			}
			col0[c] += a * ac.g / float64(hz)
		}
	}
	for _, l := range f.layers {
		drawRange(cv, l, hz, auroraPal, 1)
	}
	// a green wash from the sky on the snow of the range
	for c := 0; c < f.w; c++ {
		for r := hz - 16; r < hz; r++ {
			if r >= 0 && cv.at(c, r).ch != ' ' {
				cv.tint(c, r, col{0.15, 0.9, 0.55}, clamp01(col0[c]*2.2)*0.16)
			}
		}
	}
	// frozen lake mirroring the sky
	g := float64(f.h - hz)
	cv.fillRows(hz, f.h, []stop{{0, hex("#143a4c")}, {0.5, hex("#0c2438")}, {1, hex("#08182c")}}, nil)
	for r := hz; r < f.h; r++ {
		d := float64(r-hz) + 0.5
		sr := int(float64(hz) - 1 - d*1.2)
		if sr < 0 {
			break
		}
		for c := 0; c < f.w; c++ {
			sc := c + int(math.Round(math.Sin(d*0.9+t*0.4+float64(c)*0.07)*0.7))
			if sc < 0 || sc >= f.w {
				continue
			}
			ac, a := f.curtain(sc, sr, t)
			k := a * 0.75 * (1 - 0.6*d/g)
			p := cv.at(c, r)
			p.bg = p.bg.add(ac.mul(k))
			if k > 0.18 {
				p.ch, p.fg = rune("-~=-"[(c+r)%4]), p.bg.mul(1.5).lift(0.05)
			}
		}
	}
	// ice sparkle
	for r := hz; r < f.h; r++ {
		for c := 0; c < f.w; c++ {
			if rnd(c, r, 61) < 0.012 {
				tw := 0.5 + 0.5*math.Sin(t*2.3+rnd(c, r, 62)*6.28)
				if tw > 0.7 {
					cv.stamp(c, r, '*', mix(cv.at(c, r).bg, col{0.8, 0.95, 1}, tw*0.7), col{}, 0)
				}
			}
		}
	}

	f.drawCabin(cv, t)
	for _, p := range f.pines {
		f.drawPine(cv, p, t)
	}
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

func (f *auroraFilm) drawPine(cv *canvas, p pine, t float64) {
	dark := col{0.02, 0.07, 0.10}
	tiers := max(3, p.ht/3)
	sway := math.Sin(t*0.5+float64(p.x)) * 0.35
	for r := 0; r < p.ht; r++ {
		y := p.base - p.ht + r + 1
		u := float64(r) / float64(p.ht)
		tier := int(u * float64(tiers))
		tu := fract(u * float64(tiers))
		half := (1.0 + u*float64(p.ht)*0.42) * (0.55 + 0.45*tu)
		cx := float64(p.x) + sway*(1-u)
		for c := int(cx - half); c <= int(cx+half); c++ {
			rel := (float64(c) - cx) / math.Max(half, 0.5)
			body := mix(dark, col{0.05, 0.14, 0.16}, clamp01(0.5-rel*0.3))
			ch := rune("%#&@"[(c+r+tier)%4])
			fg := body.mul(1.7).lift(0.02)
			if tu < 0.28 && math.Abs(rel) < 0.85 { // snow on each tier
				fg = mix(col{0.45, 0.6, 0.8}, col{0.9, 1, 1}, clamp01(0.5-rel*0.5))
				ch = '*'
				if tu < 0.14 {
					ch = '^'
				}
			}
			if c == int(cx-half) {
				ch = '/'
			} else if c == int(cx+half) {
				ch = '\\'
			}
			cv.paint(c, y, ch, fg.clamp(), body)
		}
	}
	for r := 0; r < 2; r++ { // trunk
		cv.paint(p.x, p.base+r-0, '|', col{0.2, 0.12, 0.1}, col{0.08, 0.05, 0.05})
	}
}

func (f *auroraFilm) drawCabin(cv *canvas, t float64) {
	W := f.w
	cw := max(11, W/10)
	cx := int(0.64 * float64(W))
	base := f.hz + (f.h-f.hz)/4
	ch := max(4, cw/3)
	wood := col{0.22, 0.12, 0.09}
	roofc := col{0.80, 0.88, 0.98}
	// warm pool of light on the snow
	for r := base - 1; r < base+5; r++ {
		for c := cx - cw; c < cx+cw*2; c++ {
			dx := float64(c-(cx+cw/2)) / float64(cw)
			dy := float64(r-base) / 3
			cv.glow(c, r, col{1, 0.6, 0.2}, 0.22*math.Exp(-(dx*dx*1.2+dy*dy))*(0.92+0.08*math.Sin(t*5)))
		}
	}
	for r := 0; r < ch; r++ {
		for c := 0; c < cw; c++ {
			y := base - ch + r
			g := wood.mul(0.8 + 0.4*float64(r)/float64(ch))
			cv.paint(cx+c, y, rune("=-"[r%2]), g.mul(1.7), g)
		}
	}
	// window and door
	wx, wy := cx+cw/4, base-ch+1
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 3; dx++ {
			fl := 0.9 + 0.1*math.Sin(t*6+float64(dx))
			cv.paint(wx+dx, wy+dy, '#', col{1, 0.95, 0.6}.mul(fl), col{1, 0.7, 0.25}.mul(fl))
		}
	}
	for dy := 0; dy < 3; dy++ {
		cv.paint(cx+cw*3/4, base-3+dy, '|', col{0.5, 0.3, 0.15}, col{0.14, 0.08, 0.05})
	}
	// snowy roof
	for r := 0; r < ch/2+1; r++ {
		half := cw/2 + 1 - r*(cw/2+1)/(ch/2+1)
		y := base - ch - r
		for c := cx + cw/2 - half; c <= cx+cw/2+half; c++ {
			ch2 := '#'
			if c == cx+cw/2-half {
				ch2 = '/'
			} else if c == cx+cw/2+half {
				ch2 = '\\'
			}
			cv.paint(c, y, ch2, roofc, roofc.mul(0.55))
		}
	}
	// chimney and smoke
	chx := cx + cw*3/4
	cv.paint(chx, base-ch-ch/2, '#', col{0.4, 0.3, 0.3}, col{0.15, 0.1, 0.1})
	for i := 0; i < 14; i++ {
		ph := fract(t*0.18 + float64(i)/14)
		y := float64(base-ch-ch/2-1) - ph*float64(ch+5)
		x := float64(chx) + ph*5 + math.Sin(t*0.7+float64(i))*ph*1.5
		a := (1 - ph) * 0.6
		ch2 := rune("(~)o"[i%4])
		cv.ink(int(x), int(y), ch2, mix(cv.at(max(0, min(f.w-1, int(x))), max(0, min(f.h-1, int(y)))).bg, col{0.8, 0.8, 0.85}, a))
	}
}
