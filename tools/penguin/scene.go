package main

import (
	"math"
	"strings"
)

// The film, in seconds.
const (
	loopLen  = 48.0
	fadeIn   = 2.4
	fadeOut  = 2.8
	tTurn0   = 5.0  // he has stood looking back at the colony; now he turns
	tTurn1   = 8.6  // and faces the mountains
	tWalk0   = 8.2  // first step
	tWalkEnd = 41.0 // last step
	rampTime = 2.6  // time to reach walking pace
	cadence  = 1.30 // full waddle cycles per second at pace
	caption  = "and into the mountains."
)

type colonist struct {
	xf, df    float64 // position as a fraction of width / of the ground depth
	yaw, size float64
	chick     bool
}

type speck struct {
	xf, df float64
	yaw    float64
}

type flake struct {
	x, y, v, ph float64
}

type scene struct {
	w, h, hz int
	g        float64
	layers   []*layer
	stars    []star
	colony   []colonist
	specks   []speck
	flakes   []flake
}

func deg(d float64) float64 { return d * math.Pi / 180 }

func newScene(w, h int) *scene {
	s := &scene{w: w, h: h}
	s.hz = int(math.Round(float64(h) * 0.585))
	s.g = float64(h - s.hz)
	s.buildLayers()
	s.stars = newStars(w, s.hz)

	for _, c := range []colonist{
		{0.070, 0.99, 20, 1, false}, {0.172, 0.94, 160, 1, false}, {0.262, 0.99, -20, 1, false},
		{0.120, 0.77, 195, 1, false}, {0.212, 0.72, 15, 1, false}, {0.300, 0.80, 165, 1, false},
		{0.035, 0.66, 150, 1, false}, {0.152, 0.58, -35, 1, false}, {0.248, 0.52, 190, 1, false},
		{0.322, 0.60, 8, 1, false}, {0.092, 0.44, 160, 1, false}, {0.192, 0.40, 20, 1, false},
		{0.282, 0.37, -150, 1, false}, {0.055, 0.31, 12, 1, false},
		{0.215, 0.875, 175, 0.52, true}, {0.125, 0.84, 10, 0.46, true}, {0.265, 0.70, 185, 0.5, true},
	} {
		s.colony = append(s.colony, c)
	}
	for i := 0; i < 34; i++ {
		s.specks = append(s.specks, speck{
			xf:  0.015 + 0.40*math.Pow(rnd(i, 1, 51), 0.85),
			df:  0.02 + 0.20*rnd(i, 2, 51),
			yaw: (rnd(i, 3, 51) - 0.5) * 6.2,
		})
	}
	n := w * h / 110
	for i := 0; i < n; i++ {
		s.flakes = append(s.flakes, flake{
			x:  rnd(i, 1, 61) * float64(w+20),
			y:  float64(s.hz-2) + rnd(i, 2, 61)*(s.g+2),
			v:  2.5 + 5.5*rnd(i, 3, 61),
			ph: rnd(i, 4, 61) * 6.28,
		})
	}
	return s
}

func dawnAt(t float64) float64 { return 0.35 + 0.65*smoothstep(0, 40, t) }

func fadeAt(t float64) float64 {
	return smoothstep(0, fadeIn, t) * smoothstep(0, fadeOut, loopLen-t)
}

// walker is the penguin who leaves.
type walkPath struct {
	d0, dEnd, k float64
	x0, xP      float64
	wTotal      float64
}

func (s *scene) path() walkPath {
	p := walkPath{
		d0:   0.66 * s.g,
		dEnd: 1.25 + 0.05*s.g,
		x0:   0.405 * float64(s.w),
		xP:   0.700 * float64(s.w),
	}
	tw := tWalkEnd - tWalk0
	p.wTotal = rampTime/2 + (tw - rampTime)
	p.k = (p.d0/p.dEnd - 1) / p.wTotal
	return p
}

// progress is the distance walked by time t, with a smooth start.
func progress(t float64) float64 {
	tau := t - tWalk0
	if tau <= 0 {
		return 0
	}
	if tau > tWalkEnd-tWalk0 {
		tau = tWalkEnd - tWalk0
	}
	if tau < rampTime {
		u := tau / rampTime
		return rampTime * (u*u*u - u*u*u*u/2)
	}
	return rampTime/2 + (tau - rampTime)
}

func (p walkPath) at(w float64) (x, d float64) {
	d = p.d0 / (1 + p.k*w)
	x = p.xP + (p.x0-p.xP)*d/p.d0
	return
}

const strideW = 0.78 // distance per full waddle cycle

func easeIO(u float64) float64 { u = clamp01(u); return u * u * (3 - 2*u) }

// walker poses the penguin who leaves at time t.
func (s *scene) walker(t float64) *pengu {
	p := s.path()
	w := progress(t)
	x, d := p.at(w)
	pg := &pengu{x: x, d: d, size: 1.3, sway: t * 0.9}
	yawBack, yawWalk := deg(176), deg(48)
	switch {
	case t < tTurn0:
		pg.yaw = yawBack + 0.03*math.Sin(t*0.8)
		pg.headYaw = 0.10 * math.Sin(t*0.55)
		pg.headPitch = 0.05 * math.Sin(t*0.7)
	case t < tTurn1:
		u := easeIO((t - tTurn0) / (tTurn1 - tTurn0))
		pg.yaw = lerp(yawBack, yawWalk, u)
		// the head leads the turn and settles a beat later
		pg.headYaw = -0.55 * math.Sin(math.Pi*clamp01(u*1.15))
	default:
		pg.yaw = yawWalk
	}
	// walking animation, with a gentle fade to a stand at the very end
	end := 1 - smoothstep(tWalkEnd-0.6, tWalkEnd, t)
	if t >= tWalk0 {
		pg.walk = smoothstep(tWalk0, tWalk0+1.4, t) * end
		pg.phase = 2 * math.Pi * w / strideW
		pg.headYaw += 0.05 * math.Sin(pg.phase)
		pg.yaw += 0.03 * math.Sin(pg.phase*0.5)
	}
	return pg
}

// draw composes one frame of the film.
func (s *scene) draw(t float64) *canvas {
	t = math.Mod(t, loopLen)
	cv := newCanvas(s.w, s.h)
	dawn := dawnAt(t)
	s.drawSky(cv, t, dawn)
	for _, l := range s.layers {
		s.drawLayer(cv, l, dawn)
	}
	s.drawGround(cv, t, dawn)

	st := &stage{hz: s.hz, g: s.g, lit: 0.55 + 0.55*dawn, haze: hazeCol.mul(0.55 + 0.45*dawn)}
	lighten := 0.65 + 0.35*dawn
	_ = lighten

	// footprints behind the walker
	pth := s.path()
	w := progress(t)
	for n := 1; ; n++ {
		wn := float64(n) * strideW / 2
		if wn > w-0.05 {
			break
		}
		x, d := pth.at(wn)
		h := rowsPerD * d
		side := 1.0
		if n%2 == 0 {
			side = -1
		}
		x += side * 0.09 * h * 2
		row := int(float64(s.hz) + d - 0.2)
		ch := '.'
		switch {
		case h >= 4.5:
			ch = 'o'
		case h >= 2.4:
			ch = ':'
		}
		col0 := col{0.22, 0.25, 0.48}
		if cv.in(int(x), row) {
			cv.ink(int(x), row, rune(ch), mix(cv.at(int(x), row).bg, col0, 0.6))
		}
	}

	// everyone, far to near
	type item struct {
		p  *pengu
		by float64
	}
	var items []item
	for i, sp := range s.specks {
		p := &pengu{x: sp.xf * float64(s.w), d: sp.df * s.g, size: 1, yaw: sp.yaw, sway: float64(i)}
		items = append(items, item{p, p.d})
	}
	for i, c := range s.colony {
		p := &pengu{
			x: c.xf * float64(s.w), d: c.df * s.g, size: c.size, chick: c.chick,
			yaw: deg(c.yaw), sway: t*0.8 + float64(i)*2.1, flap: 1,
		}
		p.headYaw = 0.30 * math.Sin(t*0.33+float64(i)*1.7)
		p.headPitch = 0.07 * math.Sin(t*0.5+float64(i))
		items = append(items, item{p, p.d})
	}
	wk := s.walker(t)
	items = append(items, item{wk, wk.d})
	// insertion sort by depth: smaller d is further away
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].by < items[j-1].by; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	for _, it := range items {
		if it.p.rows() >= 1.8 {
			cv.drawShadow(it.p, st)
		}
	}
	for _, it := range items {
		cv.drawPenguin(it.p, st, t)
	}

	s.drawFlakes(cv, t)
	s.drawCaption(cv, t)
	cv.fade(fadeAt(t))
	return cv
}

func (s *scene) drawFlakes(cv *canvas, t float64) {
	W := float64(s.w + 20)
	for _, f := range s.flakes {
		x := math.Mod(f.x-f.v*t+W*8, W) - 10
		y := f.y + 0.8*math.Sin(t*0.9+f.ph)
		c, r := int(x), int(y)
		if !cv.in(c, r) {
			continue
		}
		cl := cv.at(c, r)
		ch := '.'
		switch {
		case f.v > 6.5:
			ch = '-'
		case f.v > 4.5:
			ch = ','
		}
		cl.ch = ch
		cl.fg = mix(cl.bg, rimWhite, 0.45)
	}
}

func (s *scene) drawCaption(cv *canvas, t float64) {
	a := smoothstep(36.5, 40.5, t) * smoothstep(0, 2.0, loopLen-fadeOut-t)
	if a < 0.02 {
		return
	}
	txt := strings.Join(strings.Split(caption, ""), "")
	x0 := (s.w - len(txt)) / 2
	y := s.h - 3
	for i, ch := range txt {
		if ch == ' ' {
			continue
		}
		if cv.in(x0+i, y) {
			cl := cv.at(x0+i, y)
			cl.ch = ch
			cl.fg = mix(cl.bg, col{0.93, 0.90, 1.0}, a*0.85)
		}
	}
}
