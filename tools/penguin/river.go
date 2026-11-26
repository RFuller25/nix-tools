package main

import "math"

// riverFilm: a wide river winding toward the viewer out of blue hills on a
// bright morning. The water moves in perspective, leaves and a canoe drift
// down it, and the far end of the river wraps back to the hills forever.
type riverFilm struct {
	w, h, hz int
	layers   []*layer
	trees    []tree
}

type tree struct {
	v, side, off, size float64 // depth along the river, bank side, distance from bank, scale
	tone               float64
}

var riverPal = &mtnPal{
	rockLit: col{0.55, 0.62, 0.78}, rockSh: col{0.24, 0.32, 0.55},
	snowLit: col{1.0, 0.98, 0.95}, snowSh: col{0.62, 0.72, 0.92},
	haze: col{0.80, 0.88, 0.96}, fog: col{0.90, 0.92, 0.96}, rim: col{1, 1, 0.95}, glowX: 0.7, calm: true,
}

var riverSky = []stop{{0, hex("#5b9ad8")}, {0.55, hex("#a9d0ee")}, {1, hex("#fbe9cf")}}

func newRiverFilm(w, h int) *riverFilm {
	f := &riverFilm{w: w, h: h, hz: int(math.Round(float64(h) * 0.38))}
	f.layers = []*layer{
		newLayer(w, f.hz, []peak{{0.18, 0.22, 0.55, 1.2}, {0.45, 0.26, 0.40, 1.15}, {0.74, 0.26, 0.62, 1.25}, {0.98, 0.18, 0.40, 1.2}}, 801, 0.45, 0.45, 0.30, false),
		newLayer(w, f.hz, []peak{{0.1, 0.3, 0.12, 1}, {0.4, 0.3, 0.16, 1}, {0.7, 0.3, 0.13, 1}, {1.0, 0.25, 0.15, 1}}, 802, 0.30, 0, 0.5, true),
	}
	for i := 0; i < 46; i++ {
		f.trees = append(f.trees, tree{
			v: 0.05 + 0.95*math.Pow(rnd(i, 1, 81), 0.8), side: float64(1 - 2*(i%2)),
			off: 0.05 + 0.45*rnd(i, 2, 81), size: 0.8 + 0.5*rnd(i, 3, 81), tone: rnd(i, 4, 81),
		})
	}
	// far trees first
	for i := 1; i < len(f.trees); i++ {
		for j := i; j > 0 && f.trees[j].v < f.trees[j-1].v; j-- {
			f.trees[j], f.trees[j-1] = f.trees[j-1], f.trees[j]
		}
	}
	return f
}

// centre and half-width of the river at depth v (0 far .. 1 near), in columns.
func (f *riverFilm) bed(v float64) (cx, hw float64) {
	W := float64(f.w)
	cx = W*0.5 + W*0.20*math.Sin(2.3*(1-v)+0.4)*(0.25+0.75*(1-v)) + W*0.03*math.Sin(7*v)
	hw = W * (0.012 + 0.40*math.Pow(v, 1.25))
	return
}

func (f *riverFilm) vOf(r int) float64 { // depth of a ground row
	return (float64(r) + 0.5 - float64(f.hz)) / float64(f.h-f.hz)
}

func (f *riverFilm) draw(t float64) *canvas {
	cv := newCanvas(f.w, f.h)
	hz := f.hz
	sunx := 0.72
	cv.fillRows(0, hz, riverSky, func(c, r int, base col) col {
		dx := (float64(c)/float64(f.w) - sunx) / 0.28
		dy := float64(hz-r) / float64(hz) / 0.5
		return base.add(col{1, 0.85, 0.55}.mul(0.30 * math.Exp(-(dx*dx + dy*dy))))
	})
	for r := 1; r < hz-3; r++ {
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.04-t*0.04, float64(r)*0.30, 4, 821)
			a := smoothstep(0.55, 0.74, n)
			if a > 0.03 {
				p := cv.at(c, r)
				p.bg = mix(p.bg, col{1, 1, 1}, a*0.7)
				if a > 0.4 {
					p.ch, p.fg = '~', mix(p.bg, col{1, 0.98, 0.95}, 0.6)
				}
			}
		}
	}
	// sun disc
	sx, sy := int(sunx*float64(f.w)), hz-int(float64(hz)*0.16)
	for dy := -2; dy <= 2; dy++ {
		for dx := -5; dx <= 5; dx++ {
			if float64(dx*dx)/25+float64(dy*dy)/4 <= 1 {
				cv.paint(sx+dx, sy+dy, ' ', col{}, col{1, 0.97, 0.80})
			}
		}
	}
	for _, l := range f.layers {
		drawRange(cv, l, hz, riverPal, 1)
	}
	f.drawLand(cv, t)
	f.drawWater(cv, t)
	f.drawTrees(cv, t)
	f.drawCanoe(cv, t)
	f.drawLeaves(cv, t)
	cv.fade(smoothstep(0, fadeIn, t))
	return cv
}

func (f *riverFilm) drawLand(cv *canvas, t float64) {
	for r := f.hz; r < f.h; r++ {
		v := f.vOf(r)
		for c := 0; c < f.w; c++ {
			n := fbm(float64(c)*0.18/(v+0.12), 0.9/(v+0.05), 3, 831)
			g := mix(col{0.60, 0.72, 0.52}, col{0.20, 0.42, 0.20}, smoothstep(0.0, 0.9, v)*0.9)
			g = mix(g, col{0.74, 0.70, 0.42}, smoothstep(0.55, 0.8, n)*0.5)
			g = g.mul(0.85 + 0.3*n)
			ch := ' '
			if rnd(c, r, 41) < 0.08+0.18*v {
				ch = rune("\"'`,;"[(c*7+r)%5])
			}
			cv.paint(c, r, ch, g.mul(1.35).lift(0.03).clamp(), g.clamp())
		}
	}
}

func (f *riverFilm) drawWater(cv *canvas, t float64) {
	sunx := 0.72 * float64(f.w)
	for r := f.hz; r < f.h; r++ {
		v := f.vOf(r)
		cx, hw := f.bed(v)
		z := 1 / (v + 0.04)
		for c := 0; c < f.w; c++ {
			u := (float64(c) + 0.5 - cx) / hw
			if math.Abs(u) > 1.0+0.12*math.Sin(float64(r)*0.7+float64(c)*0.3) {
				continue
			}
			au := math.Abs(u)
			// streaks run along the flow, so they are stretched in depth
			n := fbm(u*5+float64(r)*0.0, z*0.55-t*0.55, 3, 851)
			n2 := vnoise(u*9, z*1.4-t*1.1, 853)
			fres := 0.30 + 0.55*(1-v)
			deep := mix(col{0.10, 0.36, 0.46}, col{0.05, 0.26, 0.38}, au)
			sky := mix(col{0.95, 0.88, 0.78}, col{0.55, 0.75, 0.92}, v)
			bg := mix(deep, sky, fres*(0.7+0.5*n))
			ch, fg := ' ', col{}
			switch {
			case n > 0.60:
				ch = rune("~=~-"[(c+r)%4])
				fg = bg.mul(1.25).lift(0.10)
			case n2 > 0.78:
				ch = '-'
				fg = bg.mul(1.35).lift(0.08)
			}
			// the sun's long glitter on the water
			gl := math.Exp(-math.Pow((float64(c)-sunx)/(3+14*v), 2)) * (1 - v*0.4)
			if gl > 0.25 && n2 > 0.55 && rnd(c+int(t*4), r, 87) < gl {
				ch = []rune("*+'.")[int(rnd(c, r+int(t*3), 88)*4)]
				fg = col{1, 0.97, 0.82}
				bg = bg.add(col{0.2, 0.16, 0.08})
			}
			// foam and wet stones along the edge
			if au > 0.86 {
				e := (au - 0.86) / 0.3
				bg = mix(bg, col{0.85, 0.93, 0.95}, clamp01(e*0.8))
				if rnd(c, r, 95) < 0.5 {
					ch, fg = rune("o.:~"[int(rnd(c, r+int(t*1.5), 96)*4)]), col{1, 1, 1}
				}
			}
			cv.paint(c, r, ch, fg.clamp(), bg.clamp())
		}
	}
}

func (f *riverFilm) drawTrees(cv *canvas, t float64) {
	for _, tr := range f.trees {
		r := f.hz + int(tr.v*float64(f.h-f.hz))
		if r >= f.h {
			continue
		}
		cx, hw := f.bed(tr.v)
		x := cx + tr.side*(hw*(1.18+tr.off*2.2)+2)
		ht := (2 + 13*tr.v) * tr.size
		rad := (1.5 + 5.6*tr.v) * tr.size
		// trunk
		for k := 0; k < int(ht*0.35); k++ {
			cv.paint(int(x), r-k, '|', col{0.35, 0.22, 0.14}, col{0.20, 0.12, 0.08})
		}
		// canopy
		top := float64(r) - ht
		for dy := -ht * 0.55; dy <= ht*0.15; dy += 0.5 {
			for dx := -rad; dx <= rad; dx += 0.5 {
				e := dx*dx/(rad*rad) + (dy*dy)/(ht*0.38*ht*0.38)
				if e > 1 {
					continue
				}
				c, rr := int(math.Floor(x+dx)), int(math.Floor(top+ht*0.5+dy))
				if !cv.in(c, rr) {
					continue
				}
				lit := clamp01(0.55 + (dx/rad)*0.4*sign(sunDir) - dy/ht*0.5)
				base := mix(col{0.08, 0.26, 0.14}, col{0.45, 0.65, 0.24}, lit)
				if tr.tone > 0.86 {
					base = mix(base, col{0.88, 0.60, 0.20}, 0.55) // a few turning
				}
				n := fbm(float64(c)*0.6, float64(rr)*1.2, 2, 861)
				base = base.mul(0.8 + 0.4*n)
				ch := rune("%#&@o"[int(n*5)%5])
				cv.paint(c, rr, ch, base.mul(1.4).lift(0.03).clamp(), base)
			}
		}
	}
}

const sunDir = 1.0

// the canoe drifts down the middle of the river and starts again at the far
// end whenever it leaves the bottom of the screen
func (f *riverFilm) drawCanoe(cv *canvas, t float64) {
	period := 70.0
	ph := fract(t/period + 0.18)
	v := 0.06 + 0.94*math.Pow(ph, 1.9)
	if v > 1.06 {
		return
	}
	cx, hw := f.bed(v)
	x := cx + hw*0.18*math.Sin(t*0.3)
	r := f.hz + int(v*float64(f.h-f.hz))
	L := 2.5 + 20*v
	bob := math.Sin(t*1.6) * 0.3
	hull := col{0.45, 0.24, 0.12}
	for dx := -L / 2; dx <= L/2; dx++ {
		u := dx / (L / 2)
		rise := int(math.Round(u * u * 1.4 * math.Min(1, L/8)))
		c := int(math.Round(x + dx))
		y := r - rise + int(math.Round(bob*v))
		ch := '_'
		if math.Abs(u) > 0.8 {
			ch = '/'
			if u > 0 {
				ch = '\\'
			}
		}
		cv.paint(c, y, ch, hull.mul(1.9), hull)
		if L > 8 {
			cv.paint(c, y-1, '=', hull.mul(1.5), hull.mul(0.8))
		}
		// reflection
		cv.stamp(c, y+2, '-', hull.mul(0.9), cv.at(max(0, min(f.w-1, c)), min(f.h-1, y+2)).bg.mul(0.85), 0.5)
	}
	if L > 7 { // a paddler in a wide hat, and the paddle
		px := int(math.Round(x - L*0.1))
		cv.paint(px, r-2, 'A', col{0.9, 0.78, 0.35}, col{0.2, 0.2, 0.3})
		cv.paint(px, r-1, 'H', col{0.9, 0.3, 0.25}, col{0.25, 0.2, 0.3})
		sw := int(math.Round(math.Sin(t*1.1) * 2))
		cv.ink(px+2+sw, r, '/', col{0.6, 0.4, 0.2})
	} else {
		cv.ink(int(x), r-1, 'o', col{0.9, 0.3, 0.2})
	}
	// wake
	for k := 1; k < 5; k++ {
		cv.ink(int(x-L/2)-k*2, r+1, '~', mix(cv.at(max(0, int(x-L/2)-k*2), min(f.h-1, r+1)).bg, col{1, 1, 1}, 0.5/float64(k)))
	}
}

func (f *riverFilm) drawLeaves(cv *canvas, t float64) {
	for i := 0; i < 22; i++ {
		ph := fract(t/55 + hashf(i, 91))
		v := 0.04 + 0.96*math.Pow(ph, 1.8)
		cx, hw := f.bed(v)
		u := (hashf(i, 92)*2 - 1) * 0.8
		x := cx + u*hw + math.Sin(t*0.7+float64(i))*hw*0.05
		r := f.hz + int(v*float64(f.h-f.hz))
		c := col{0.95, 0.55, 0.15}
		if i%3 == 0 {
			c = col{0.9, 0.3, 0.2}
		}
		if v > 0.2 {
			cv.ink(int(x), r, '*', c)
		} else {
			cv.ink(int(x), r, '.', c)
		}
	}
}
