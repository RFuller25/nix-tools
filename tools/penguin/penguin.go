package main

import "math"

// pengu is everything needed to pose and place one emperor penguin (or chick).
//
// yaw turns the whole body about the vertical axis: 0 faces right in profile,
// pi/2 faces away from the viewer, pi faces left, -pi/2 faces the viewer.
type pengu struct {
	x, d      float64 // foot column and depth in rows below the horizon
	size      float64 // 1 for an adult, <1 for a chick
	yaw       float64
	headYaw   float64 // head turn relative to the body
	headPitch float64
	phase     float64 // walk cycle in radians
	walk      float64 // 0 standing .. 1 full waddle
	sway      float64 // idle sway phase
	flap      float64 // extra flipper swing
	chick     bool
}

const (
	camPitch = 0.10 // the camera looks very slightly down on the ice
	minRows  = 1.25 // the walker never shrinks below a visible speck
	rowsPerD = 0.56 // penguin height in rows per row of depth below the horizon
	subX     = 2    // samples across a cell
	subY     = 4    // samples down a cell
)

var sampleY = [subY]float64{0.125, 0.375, 0.625, 0.875}

// rows is the penguin's drawn height in terminal rows.
func (p *pengu) rows() float64 {
	return math.Max(rowsPerD*p.d*p.size, minRows*math.Max(p.size, 0.7))
}

// parts poses the model. All positions are in units of the penguin's height,
// with the feet at the origin.
func (p *pengu) parts() []part {
	Rg := rotX(camPitch).mul(rotY(p.yaw))
	w, ph := p.walk, p.phase
	roll := -w*0.10*math.Cos(ph) + (1-w)*0.02*math.Sin(p.sway)
	lean := 0.05 + 0.05*w
	Rb := Rg.mul(rotZ(-lean)).mul(rotX(roll))
	bob := vec3{0, w * 0.013 * (1 - math.Abs(math.Cos(ph))), 0}
	at := func(R mat3, o vec3) vec3 { return R.vec(o).add(bob) }

	var ps []part
	add := func(o vec3, R mat3, r vec3, m matID) {
		ps = append(ps, newPart(at(Rb, o), Rb.mul(R), r, m, o))
	}

	if p.chick {
		add(vec3{0, 0.30, 0}, ident(), vec3{0.20, 0.29, 0.18}, matChickBody)
		neck := vec3{0.05, 0.50, 0}
		Rh := rotY(p.headYaw).mul(rotZ(p.headPitch))
		hc := neck.add(Rh.vec(vec3{0.02, 0.10, 0}))
		ps = append(ps, newPart(at(Rb, hc), Rb.mul(Rh), vec3{0.145, 0.13, 0.13}, matChickHead, hc))
		bc := hc.add(Rh.vec(vec3{0.15, -0.015, 0}))
		ps = append(ps, newPart(at(Rb, bc), Rb.mul(Rh), vec3{0.045, 0.022, 0.022}, matChickBeak, bc))
		for _, k := range [2]float64{-1, 1} {
			sp := ph + (k+1)*math.Pi/2
			ft := vec3{0.03 + w*0.09*math.Sin(sp), 0.02 + w*0.03*math.Max(0, math.Cos(sp)), k * 0.075}
			ps = append(ps, newPart(at(Rg, ft), Rg, vec3{0.085, 0.022, 0.05}, matFoot, ft))
		}
		return ps
	}

	// Torso, neck, head and beak.
	add(vec3{0.005, 0.315, 0}, ident(), vec3{0.178, 0.285, 0.146}, matBody)
	add(vec3{0.012, 0.545, 0}, rotZ(-0.10), vec3{0.142, 0.245, 0.112}, matBody)
	add(vec3{0.03, 0.775, 0}, rotZ(-0.35), vec3{0.062, 0.092, 0.052}, matNeck)

	neck := vec3{0.035, 0.80, 0}
	Rh := rotY(p.headYaw).mul(rotZ(p.headPitch))
	hc := neck.add(Rh.vec(vec3{0.012, 0.072, 0}))
	ps = append(ps, newPart(at(Rb, hc), Rb.mul(Rh), vec3{0.092, 0.084, 0.078}, matHead, hc))
	Rk := Rh.mul(rotZ(-0.20))
	bc := hc.add(Rh.vec(vec3{0.118, -0.016, 0}))
	ps = append(ps, newPart(at(Rb, bc), Rb.mul(Rk), vec3{0.088, 0.0145, 0.0145}, matBeak, bc))

	add(vec3{-0.15, 0.07, 0}, rotZ(0.32), vec3{0.075, 0.018, 0.042}, matTail)

	// Flippers hang from the shoulders and counter-swing with the stride.
	for _, k := range [2]float64{-1, 1} {
		swing := -0.20 + p.flap*0.25*math.Sin(p.sway*1.3+k) + w*0.38*math.Sin(ph+(k+1)*math.Pi/2+math.Pi)
		R := rotZ(swing).mul(rotX(-k * 0.55))
		sh := vec3{0.0, 0.625, k * 0.118}
		c := sh.add(R.vec(vec3{0, -0.138, 0}))
		add(c, R, vec3{0.034, 0.148, 0.0115}, matFlipper)
	}

	// Feet stay on the ground and take turns stepping.
	for _, k := range [2]float64{-1, 1} {
		sp := ph + (k+1)*math.Pi/2
		ft := vec3{0.035 + w*0.11*math.Sin(sp), 0.02 + w*0.04*math.Max(0, math.Cos(sp)), k * 0.068}
		ps = append(ps, newPart(at(Rg, ft), Rg, vec3{0.078, 0.021, 0.042}, matFoot, ft))
	}
	return ps
}

// Palette for the animals.
var (
	cBlack    = col{0.115, 0.125, 0.185}
	cBelly    = col{0.96, 0.95, 0.92}
	cBreast   = col{0.98, 0.84, 0.52}
	cEar      = col{1.00, 0.64, 0.14}
	cNeckGold = col{0.98, 0.76, 0.32}
	cBeakPink = col{0.92, 0.50, 0.44}
	cFlipper  = col{0.150, 0.165, 0.235}
	cFoot     = col{0.17, 0.15, 0.19}
	cDown     = col{0.60, 0.59, 0.63}
	cDownLite = col{0.80, 0.79, 0.82}
)

func albedo(pt *part, nl vec3, py float64) col {
	switch pt.mat {
	case matBody:
		b := smoothstep(0.10, 0.46, nl.x)
		wy := smoothstep(0.44, 0.74, py) * smoothstep(0.05, 0.36, nl.x)
		return mix(cBlack, mix(cBelly, cBreast, wy), b)
	case matNeck:
		side := smoothstep(0.25, 0.75, math.Abs(nl.z)) * smoothstep(-0.55, 0.05, nl.x)
		low := smoothstep(0.3, -0.5, nl.y)
		front := smoothstep(0.30, 0.85, nl.x) * (1 - smoothstep(0.25, 0.55, math.Abs(nl.z)))
		a := mix(cBlack, cNeckGold, clamp01(side*0.85+low*0.25))
		return mix(a, cBlack, front*0.8)
	case matHead:
		e := (nl.x+0.18)*(nl.x+0.18)/0.20 + (nl.y+0.06)*(nl.y+0.06)/0.36
		patch := (1 - smoothstep(0.55, 1.0, e)) * smoothstep(0.30, 0.62, math.Abs(nl.z))
		return mix(cBlack, cEar, patch)
	case matBeak:
		return mix(cBeakPink, cBlack, smoothstep(-0.55, 0.05, nl.y))
	case matFlipper:
		return cFlipper
	case matFoot:
		return cFoot
	case matTail:
		return cBlack
	case matChickBody:
		return mix(cDown, cDownLite, smoothstep(0.1, 0.6, nl.x)*0.7+smoothstep(-0.2, 0.7, nl.y)*0.15)
	case matChickHead:
		m := smoothstep(0.05, 0.40, nl.x) * smoothstep(-0.55, 0.15, nl.y)
		ring := smoothstep(0.25, 0.55, nl.x) * (1 - smoothstep(0.25, 0.5, math.Abs(nl.y-0.1)))
		return mix(mix(cDown, cBlack, m*0.92), cDownLite, ring*0.0)
	case matChickBeak:
		return cBlack
	}
	return cBlack
}

// Lighting: a low warm key from behind the mountains, cool sky fill and a
// bounce from the ice, plus a warm rim so dark bodies keep their outline.
var (
	keyDir  = vec3{0.72, 0.30, -0.62}.norm()
	fillDir = vec3{-0.25, 0.35, 0.90}.norm()
	keyCol  = col{1.00, 0.70, 0.46}
	fillCol = col{0.95, 0.88, 0.90}
	ambTop  = col{0.60, 0.62, 0.80}
	ambBot  = col{0.62, 0.58, 0.66}
	rimCol  = col{1.00, 0.68, 0.44}
)

func shade(alb col, n vec3, lit float64) col {
	ndl := math.Max(0, n.dot(keyDir))
	ndf := math.Max(0, n.dot(fillDir))
	amb := mix(ambBot, ambTop, n.y*0.5+0.5)
	c := alb.mulc(amb.mul(0.95).add(keyCol.mul(ndl * 0.95 * lit)).add(fillCol.mul(ndf * 0.80)))
	fres := pow(1-math.Max(0, n.z), 3)
	rim := fres * smoothstep(-0.25, 0.65, n.dot(keyDir)) * 0.95 * lit
	return c.add(rimCol.mul(rim))
}

// stage carries the per-frame lighting and fog shared by every penguin.
type stage struct {
	hz   int
	g    float64 // ground rows
	lit  float64 // strength of the dawn light
	haze col
}

// drawShadow darkens the ice under a penguin, stretched away from the light.
func (cv *canvas) drawShadow(p *pengu, st *stage) {
	hIso := p.rows() * 2
	fy := (float64(st.hz) + p.d) * 2
	cx, cy := p.x-0.20*hIso, fy+0.015*hIso
	rx, ry := 0.46*hIso, 0.075*hIso
	for r := int((cy-ry)/2) - 1; r <= int((cy+ry)/2)+1; r++ {
		for c := int(cx-rx) - 1; c <= int(cx+rx)+1; c++ {
			dx := (float64(c) + 0.5 - cx) / rx
			dy := (float64(r)*2 + 1 - cy) / ry
			q := dx*dx + dy*dy
			if q >= 1 {
				continue
			}
			dens := 1 - smoothstep(0.35, 1, q)
			cv.tint(c, r, col{0.20, 0.22, 0.42}, 0.52*dens)
		}
	}
}

// drawPenguin ray-traces the model into the canvas.
func (cv *canvas) drawPenguin(p *pengu, st *stage, t float64) {
	ps := p.parts()
	rows := p.rows()
	hIso := rows * 2
	footY := (float64(st.hz) + p.d) * 2
	// Cell bounds of the model.
	c0, c1 := int(math.Floor(p.x-0.6*hIso))-1, int(math.Ceil(p.x+0.6*hIso))+1
	r0, r1 := int(math.Floor((footY-1.1*hIso)/2))-1, int(math.Ceil((footY+0.08*hIso)/2))+1
	fog := 0.38 * pow(1-clamp01(p.d/st.g), 1.3)
	minCov := 0.20
	if rows < 2.4 {
		minCov = 0.11
	}
	drawn := 0
	for r := r0; r <= r1; r++ {
		for c := c0; c <= c1; c++ {
			if !cv.in(c, r) {
				continue
			}
			var sum col
			hits := 0
			var quad [4]int // UL UR LL LR
			for sy := 0; sy < subY; sy++ {
				Y := (float64(r) + sampleY[sy]) * 2
				my := (footY - Y) / hIso
				for sx := 0; sx < subX; sx++ {
					X := float64(c) + (float64(sx)+0.5)/subX
					mx := (X - p.x) / hIso
					bestT, bi := math.Inf(1), -1
					var bu vec3
					for i := range ps {
						pt := &ps[i]
						// cheap reject
						if math.Abs(mx-pt.c.x) > pt.r.x+0.2 || math.Abs(my-pt.c.y) > pt.r.y+0.2 {
							if math.Abs(mx-pt.c.x) > 0.5 || math.Abs(my-pt.c.y) > 0.5 {
								continue
							}
						}
						if t, u, ok := pt.hit(mx, my); ok && t < bestT {
							bestT, bi, bu = t, i, u
						}
					}
					if bi < 0 {
						continue
					}
					pt := &ps[bi]
					inv := vec3{bu.x / pt.r.x, bu.y / pt.r.y, bu.z / pt.r.z}
					nl := inv.norm()
					nw := pt.rot.vec(inv).norm()
					py := pt.cb.y + pt.r.y*bu.y
					col := shade(albedo(pt, nl, py), nw, st.lit)
					col = mix(col, st.haze, fog)
					sum = sum.add(col)
					hits++
					q := sx
					if sy >= subY/2 {
						q += 2
					}
					quad[q]++
				}
			}
			if hits == 0 {
				continue
			}
			cov := float64(hits) / float64(subX*subY)
			if cov < minCov {
				continue
			}
			avg := sum.mul(1 / float64(hits)).clamp()
			cur := cv.at(c, r)
			drawn++
			if rows >= 3.2 {
				if cov < 0.26 {
					drawn--
					continue
				}
				a := smoothstep(0.26, 0.90, cov)
				ch, fg, bg := shadeCell(avg, 0.20+0.55*(1-avg.lum()), (rnd(c, r, 9)-0.5)*0.10-0.12)
				if cov < 0.86 {
					if e := edgeGlyph(quad, cov); e != 0 {
						ch = e
						fg = avg.mul(1.25).lift(0.06).clamp()
						a = smoothstep(0.1, 0.8, cov)
					} else if cov < 0.55 {
						ch = ' '
					}
				}
				cur.ch = ch
				cur.fg = mix(avg, fg, a)
				cur.bg = mix(cur.bg, bg, a)
			} else if cov >= 0.62 {
				a := smoothstep(0.55, 0.95, cov)
				ch, fg, bg := shadeCell(avg, 1.0, (rnd(c, r, 9)-0.5)*0.10)
				cur.ch = ch
				cur.fg = mix(avg, fg, a)
				cur.bg = mix(cur.bg, bg, a)
			} else {
				glyphs := ".:i!"
				gi := int((cov - minCov) / (0.62 - minCov) * float64(len(glyphs)))
				if gi >= len(glyphs) {
					gi = len(glyphs) - 1
				}
				cur.ch = rune(glyphs[gi])
				cur.fg = avg
			}
		}
	}
	if drawn == 0 {
		cx, cy := int(p.x), int(float64(st.hz)+p.d-0.5)
		cv.ink(cx, cy, 'i', hex("#15121f"))
	}
}

// edgeGlyph draws the outline of a partly covered cell. quad holds how many of
// the two samples in each quarter (upper-left, upper-right, lower-left,
// lower-right) hit the animal; 0 means "not an edge, fill it".
func edgeGlyph(quad [4]int, cov float64) rune {
	var q [4]bool
	n := 0
	for i, v := range quad {
		q[i] = v >= 1
		if q[i] {
			n++
		}
	}
	ul, ur, ll, lr := q[0], q[1], q[2], q[3]
	switch {
	case n == 1:
		switch {
		case ul:
			return '`'
		case ur:
			return '\''
		}
		return '.'
	case n == 2:
		switch {
		case ul && ll:
			return ')'
		case ur && lr:
			return '('
		case ul && ur:
			if cov < 0.42 {
				return '\''
			}
			return '-'
		case ll && lr:
			if cov < 0.42 {
				return '_'
			}
			return '-'
		case ul && lr:
			return '\\'
		}
		return '/'
	}
	return 0
}
