package main

import "math"

// deepSeaFilm: sunlight falling through the water onto drifting jellyfish,
// kelp swaying on the seabed, a school of fish crossing and marine snow
// sinking past, all forever.
type deepSeaFilm struct {
	w, h   int
	jellys []jelly
	kelp   []strand
	snow   []flake
	fish   []flake
	floor  []float64
}

type jelly struct {
	x, y, size float64
	ph, sp     float64
	c          col
}

type strand struct {
	x   float64
	ht  float64
	ph  float64
	col col
}

func newDeepSeaFilm(w, h int) *deepSeaFilm {
	f := &deepSeaFilm{w: w, h: h}
	W, H := float64(w), float64(h)
	cols := []col{{0.35, 0.95, 1.0}, {1.0, 0.45, 0.90}, {1.0, 0.78, 0.40}, {0.62, 0.55, 1.0}, {0.45, 1.0, 0.70}}
	for i, c := range cols {
		f.jellys = append(f.jellys, jelly{
			x: W * (0.10 + 0.20*float64(i) + 0.05*rnd(i, 1, 201)), y: H * (0.18 + 0.12*float64((i*3)%5)),
			size: H * (0.09 + 0.09*rnd(i, 3, 201)), ph: rnd(i, 4, 201) * 6.28, sp: 0.7 + 0.5*rnd(i, 5, 201), c: c,
		})
	}
	f.floor = make([]float64, w)
	for c := 0; c < w; c++ {
		f.floor[c] = H*0.90 - 3.5*noise1(float64(c)*0.08, 211) - 1.5*noise1(float64(c)*0.3, 212)
	}
	for i := 0; i < w/6; i++ {
		f.kelp = append(f.kelp, strand{
			x: rnd(i, 1, 221) * W, ht: H * (0.18 + 0.4*rnd(i, 2, 221)), ph: rnd(i, 3, 221) * 6.28,
			col: mix(col{0.05, 0.35, 0.25}, col{0.30, 0.65, 0.30}, rnd(i, 4, 221)),
		})
	}
	for i := 0; i < w*h/60; i++ {
		f.snow = append(f.snow, flake{x: rnd(i, 1, 231) * W, y: rnd(i, 2, 231) * H, v: 0.6 + 1.6*rnd(i, 3, 231), ph: rnd(i, 4, 231) * 6.28})
	}
	for i := 0; i < 14; i++ {
		f.fish = append(f.fish, flake{x: rnd(i, 1, 241) * 18, y: rnd(i, 2, 241) * 6, v: 0, ph: rnd(i, 3, 241)})
	}
	return f
}

func (f *deepSeaFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	W, H := float64(f.w), float64(f.h)
	// water, brightest near the surface, with slanting shafts of light
	stops := []stop{{0, hex("#1c7ea0")}, {0.25, hex("#0c4f74")}, {0.6, hex("#06274a")}, {1, hex("#020b1c")}}
	cv.fillRows(0, f.h, stops, func(c, r int, base col) col {
		v := float64(r) / H
		sh := 0.0
		for k := 0; k < 5; k++ {
			x0 := W*(0.1+0.2*float64(k)) + 6*math.Sin(t*0.13+float64(k)*1.9)
			xc := x0 + float64(r)*0.55
			wd := 2.5 + float64(r)*0.14
			d := (float64(c) - xc) / wd
			sh += math.Exp(-d*d) * (0.5 + 0.5*math.Sin(t*0.4+float64(k)*2.3))
		}
		return base.add(col{0.30, 0.55, 0.60}.mul(sh * 0.22 * math.Pow(1-v, 1.5)))
	})
	for r := 0; r < f.h; r++ { // faint slanting rays as glyphs
		for c := 0; c < f.w; c++ {
			p := cv.at(c, r)
			if p.bg.g > 0.38 && rnd(c, r, 7) < 0.25 {
				p.ch, p.fg = '/', p.bg.mul(1.25).lift(0.03)
			}
		}
	}
	// surface caustics
	for r := 0; r < 4; r++ {
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.18+t*0.4, float64(r)*0.9-t*0.3, 3, 251)
			if n > 0.58 {
				cv.stamp(c, r, rune("~-=~"[(c+r)%4]), col{0.7, 0.95, 1}.mul(0.7), col{0.35, 0.75, 0.85}, 0.4*(1-float64(r)/4))
			}
		}
	}

	// seabed
	for c := 0; c < f.w; c++ {
		for r := int(f.floor[c]); r < f.h; r++ {
			d := float64(r) - f.floor[c]
			n := fbm(float64(c)*0.3, float64(r)*0.8, 2, 261)
			b := mix(col{0.10, 0.12, 0.14}, col{0.03, 0.04, 0.07}, clamp01(d/4)).mul(0.7 + 0.6*n)
			ch := ' '
			if n > 0.55 {
				ch = rune(".:,;"[(c+r)%4])
			}
			cv.paint(c, r, ch, b.mul(2.2).lift(0.02), b)
		}
	}
	f.drawKelp(cv, t)
	f.drawFish(cv, t)
	for _, j := range f.jellys {
		f.drawJelly(cv, j, t)
	}
	// marine snow
	for _, s := range f.snow {
		x := wrapf(s.x+math.Sin(t*0.4+s.ph)*2+t*0.3, W)
		y := wrapf(s.y+t*s.v*0.6, H)
		cv.ink(int(x), int(y), '.', mix(cv.at(int(x), int(y)).bg, col{0.8, 0.95, 1}, 0.25+0.15*s.v))
	}
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

func (f *deepSeaFilm) drawKelp(cv *canvas, t float64) {
	for _, s := range f.kelp {
		base := int(f.floor[min(f.w-1, int(s.x))])
		n := int(s.ht)
		px := s.x
		for k := 0; k < n; k++ {
			u := float64(k) / s.ht
			sway := math.Sin(t*0.8+s.ph+u*4.5)*2.6*u*u + math.Sin(t*0.45+s.ph*2)*1.2*u
			x := s.x + sway
			dx := x - px
			px = x
			ch := '|'
			if dx > 0.45 {
				ch = '/'
			} else if dx < -0.45 {
				ch = '\\'
			}
			shade := 0.5 + 0.5*u
			c := s.col.mul(shade)
			if k%5 == 3 { // a leaf
				cv.ink(int(x)+1, base-k, ')', c.mul(1.3))
			}
			cv.ink(int(x), base-k, ch, c.mul(1.7).lift(0.02))
			cv.tint(int(x), base-k, c, 0.35)
		}
	}
}

func (f *deepSeaFilm) drawFish(cv *canvas, t float64) {
	W := float64(f.w)
	for i, s := range f.fish {
		dir := 1.0
		if i%7 == 0 {
			dir = -1
		}
		cx := wrapf(s.x*1.2+t*(4+s.ph)*dir*0.8+float64(f.w)*0.1*math.Sin(float64(i)), W+16) - 8
		y := float64(f.h)*0.5 + s.y*1.5 + math.Sin(t*0.7+float64(i))*1.5 + float64(i%3)*1.2
		body := "><>"
		if dir < 0 {
			body = "<><"
		}
		sh := col{0.75, 0.85, 0.95}.mul(0.55 + 0.35*math.Sin(t*2.5+float64(i)))
		for k, ch := range body {
			cv.ink(int(cx)+k, int(y), ch, sh)
		}
	}
}

func (f *deepSeaFilm) drawJelly(cv *canvas, j jelly, t float64) {
	pulse := math.Sin(t*j.sp*1.6 + j.ph)
	rad := j.size * (1 + 0.18*pulse)
	ht := rad * 0.80
	cx := j.x + 7*math.Sin(t*0.13*j.sp+j.ph) + 2*math.Sin(t*0.37+j.ph)
	cy := j.y + 4*math.Sin(t*0.17*j.sp+j.ph*1.3) - 2.5*pulse
	// the bell: a dome lit from within, brightest at the rim
	for dy := -ht - 1; dy <= 1; dy += 0.5 {
		for dx := -rad - 1; dx <= rad+1; dx += 0.5 {
			e := dx*dx/(rad*rad) + (dy*dy)/(ht*ht)
			if e > 1 || dy > 0.4 {
				continue
			}
			c, r := int(math.Floor(cx+dx)), int(math.Floor(cy+dy))
			if !cv.in(c, r) {
				continue
			}
			rim := smoothstep(0.55, 1, e)
			glow := j.c.mul(0.22 + 0.45*rim + 0.12*math.Sin(t*2+dx*0.8))
			p := cv.at(c, r)
			p.bg = mix(p.bg, glow, 0.55+0.3*rim)
			ch := '.'
			switch {
			case rim > 0.85:
				ch = 'o'
				if dy > -ht*0.25 {
					ch = '_'
				}
			case rim > 0.5:
				ch = ':'
			case math.Abs(dx) < rad*0.3 && dy > -ht*0.6: // gonads
				ch = '8'
			}
			p.ch, p.fg = ch, j.c.mul(1.0).add(col{0.25, 0.25, 0.25}).clamp()
		}
	}
	// frilled skirt
	for dx := -rad; dx <= rad; dx++ {
		cv.ink(int(math.Round(cx+dx)), int(cy+1), rune("~wW~"[int(math.Abs(dx))%4]), j.c.mul(1.1))
	}
	// trailing tentacles, longer when the bell has just pushed
	n := int(rad)
	for k := 0; k < n; k++ {
		off := (float64(k)/float64(max(1, n-1))*2 - 1) * rad * 0.85
		L := int(j.size * (1.6 + 0.4*math.Sin(float64(k)*1.7+j.ph)))
		for s := 1; s < L; s++ {
			u := float64(s) / float64(L)
			x := cx + off + math.Sin(t*1.3+j.ph+float64(s)*0.55+float64(k))*(0.6+1.6*u) - pulse*0.6*u
			ch := '|'
			if math.Sin(t*1.3+j.ph+float64(s)*0.55+float64(k)) > 0.4 {
				ch = ')'
			} else if math.Sin(t*1.3+j.ph+float64(s)*0.55+float64(k)) < -0.4 {
				ch = '('
			}
			cv.ink(int(math.Round(x)), int(cy)+1+s, ch, mix(cv.at(max(0, min(f.w-1, int(x))), max(0, min(f.h-1, int(cy)+1+s))).bg, j.c, 0.75*(1-u*0.8)))
		}
	}
	// two thick oral arms
	for _, o := range []float64{-0.25, 0.25} {
		for s := 1; s < int(j.size*0.9); s++ {
			u := float64(s) / (j.size * 0.9)
			x := cx + o*rad + math.Sin(t*1.1+j.ph+float64(s)*0.4)*1.3*u
			cv.ink(int(math.Round(x)), int(cy)+1+s, ':', j.c.mul(1.1).clamp())
		}
	}
	// a glow around the whole animal
	for dy := -int(ht + 5); dy < int(j.size*2); dy++ {
		for dx := -int(rad + 6); dx <= int(rad+6); dx++ {
			d2 := float64(dx*dx)/((rad+6)*(rad+6)) + float64(dy*dy)/((j.size*1.8)*(j.size*1.8))
			if d2 < 1 {
				cv.glow(int(cx)+dx, int(cy)+dy, j.c, 0.10*(1-d2)*(1-d2))
			}
		}
	}
}
