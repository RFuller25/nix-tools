package main

import (
	"math"
	"strings"
)

// peak describes one summit as fractions of the screen: centre, half width,
// height as a fraction of the sky, and how sharply its flanks fall away.
type peak struct{ c, w, h, p float64 }

// layer is one range of mountains. Every column knows its ridge height and
// which summit owns it, so the faces can be shaded as fans of arêtes running
// down from each peak.
type layer struct {
	peaks  []peak
	seed   uint32
	haze   float64
	snow   float64
	smooth bool // rolling snow hills rather than rock and ice
	top    []float64
	own    []int
}

func newLayer(w, hz int, peaks []peak, seed uint32, haze, snow, rough float64, smooth bool) *layer {
	l := &layer{peaks: peaks, seed: seed, haze: haze, snow: snow, smooth: smooth, top: make([]float64, w), own: make([]int, w)}
	for c := 0; c < w; c++ {
		x := (float64(c) + 0.5) / float64(w)
		best, bi := 0.0, 0
		for i, pk := range peaks {
			dx := (x - pk.c) / pk.w
			if math.Abs(dx) >= 1 {
				continue
			}
			// Each flank falls at its own rate so no summit is a mirror image.
			skew := 0.30 * (rnd(i, 1, seed) - 0.5) * 2
			p := pk.p * (1 + skew*sign(dx))
			v := pk.h * float64(hz) * math.Pow(1-math.Abs(dx), p)
			n1 := noise1(x*24+float64(i)*5, seed)
			n2 := noise1(x*61+float64(i)*9, seed+1)
			n3 := noise1(x*140+float64(i)*3, seed+2)
			r := (n1-0.5)*0.34 + (n2-0.5)*0.22 + (n3-0.5)*0.10
			if smooth {
				r = (n1 - 0.5) * 0.30
			}
			v *= 1 + rough*r*2*(1-math.Abs(dx)*0.15)
			if v > best {
				best, bi = v, i
			}
		}
		l.top[c], l.own[c] = best, bi
	}
	return l
}

func (s *scene) buildLayers() {
	far := []peak{
		{0.06, 0.16, 0.34, 1.15}, {0.29, 0.21, 0.42, 1.2}, {0.53, 0.16, 0.52, 1.25},
		{0.74, 0.21, 0.58, 1.15}, {0.94, 0.17, 0.48, 1.2},
	}
	main := []peak{
		{0.335, 0.150, 0.34, 1.15}, {0.510, 0.190, 0.76, 1.25}, {0.845, 0.215, 0.98, 1.28},
		{0.990, 0.135, 0.64, 1.2}, {0.668, 0.055, 0.08, 1.0},
	}
	foot := []peak{
		{0.16, 0.32, 0.055, 1.0}, {0.42, 0.24, 0.075, 1.0}, {0.60, 0.22, 0.085, 1.0},
		{0.78, 0.28, 0.100, 1.0}, {0.97, 0.22, 0.080, 1.0},
	}
	s.layers = []*layer{
		newLayer(s.w, s.hz, far, 301, 0.58, 0.30, 0.34, false),
		newLayer(s.w, s.hz, main, 302, 0.14, 0.34, 0.46, false),
		newLayer(s.w, s.hz, foot, 303, 0.34, 0.0, 0.5, true),
	}
}

var (
	rockLit  = col{0.70, 0.46, 0.56}
	rockSh   = col{0.16, 0.13, 0.34}
	snowLit  = col{1.00, 0.86, 0.74}
	snowSh   = col{0.36, 0.40, 0.80}
	hazeCol  = col{0.74, 0.58, 0.76}
	baseFog  = col{0.96, 0.72, 0.70}
	rimWhite = col{1.0, 0.95, 0.88}
)

func sigmoid(x float64) float64 { return 1 / (1 + math.Exp(-x)) }

func (s *scene) drawLayer(cv *canvas, l *layer, dawn float64) {
	drawRange(cv, l, s.hz, penguinPal, dawn)
}

// mtnPal colours a range of mountains.
type mtnPal struct {
	rockLit, rockSh, snowLit, snowSh, haze, fog, rim col
	glowX                                            float64
	calm                                             bool // soft slopes: no hatching or arête lines
}

var penguinPal = &mtnPal{rockLit, rockSh, snowLit, snowSh, hazeCol, baseFog, rimWhite, glowX, false}

// drawRange paints one layer of mountains above the horizon row hzi.
func drawRange(cv *canvas, l *layer, hzi int, pal *mtnPal, dawn float64) {
	s := struct{ w, hz int }{cv.w, hzi}
	W, hz := float64(s.w), float64(s.hz)
	k := 0.50 + 0.50*dawn
	for c := 0; c < s.w; c++ {
		top := l.top[c]
		if top < 0.4 {
			continue
		}
		pk := l.peaks[l.own[c]]
		topRow := hz - top
		r0 := int(math.Floor(topRow))
		cx := float64(c) + 0.5
		summit := hz - pk.h*hz
		slope := (l.top[min(c+1, s.w-1)] - l.top[max(c-1, 0)]) / 2
		for r := r0; r < s.hz; r++ {
			if r < 0 {
				continue
			}
			cov := 1.0
			if r == r0 {
				cov = float64(r0+1) - topRow
				if cov < 0.18 {
					continue
				}
			}
			alt := hz - (float64(r) + 0.5)
			rel := clamp01(alt / (pk.h*hz + 1))
			dxp := cx - pk.c*W
			p := clampf(dxp/(pk.w*W*0.6), -1, 1)

			var lit, sn, edge float64
			if l.smooth {
				lit = 0.5 + 0.5*math.Tanh(-slope*1.6)
				lit = clamp01(lit + (vnoise(cx*0.09, float64(r)*0.5, l.seed)-0.5)*0.35)
				sn = 1
			} else {
				ds := (float64(r) + 0.5) - summit + 2.0
				// Two fans of arêtes: one from the summit, one from lower down.
				q1 := dxp / ds
				q2 := dxp / (0.45*ds + 3)
				f1 := facet(q1*0.50+pk.c*91, l.seed)
				f2 := facet(q2*0.62+pk.c*53, l.seed+9)
				side := 0.5 + 0.5*math.Tanh(p*1.8)
				lit = smoothstep(0.22, 0.78, 0.62*f1.lit+0.38*f2.lit+0.30*(side-0.5))
				grain := vnoise(q1*1.1+pk.c*17, ds*0.30, l.seed+5)
				lit = clamp01(lit + (grain-0.5)*0.22)
				edge = math.Max(f1.edge, 0.7*f2.edge)
				snowLine := l.snow * pk.h * hz
				sn = smoothstep(snowLine-1.0, snowLine+1.8, alt+(grain-0.5)*4.0+(f1.n-0.5)*3.5)
				// facets steeper than the snow can hold stay bare
				sn *= 1 - 0.55*smoothstep(0.55, 0.9, f2.n)*(1-rel)
			}

			pool := math.Exp(-math.Pow((cx/W-pal.glowX)/0.22, 2))
			lit = clamp01(lit + 0.22*pool*(1-pow(rel, 1.3)))

			rock := mix(pal.rockSh, pal.rockLit, lit)
			snow := mix(pal.snowSh, pal.snowLit, lit)
			c0 := mix(rock, snow, sn).mul(k)
			fog := l.haze + 0.40*pow(1-rel, 2.6)
			c0 = mix(c0, pal.haze.mul(0.5+0.5*dawn), clamp01(fog*0.85))
			c0 = mix(c0, pal.fog.mul(0.6+0.4*dawn), 0.60*pow(1-clamp01(alt/5), 2))

			if r == r0 {
				// The ridge line itself is inked against the sky.
				ch := '-'
				switch {
				case slope > 0.5:
					ch = '/'
				case slope < -0.5:
					ch = '\\'
				case cov < 0.5:
					ch = '_'
				}
				if l.top[c] >= l.top[max(c-1, 0)] && l.top[c] >= l.top[min(c+1, s.w-1)] && math.Abs(slope) < 1.0 {
					ch = '^'
				}
				rim := mix(c0, pal.rim.mul(k), 0.60*(0.35+0.65*lit))
				cv.ink(c, r, ch, rim)
				continue
			}

			jit := (rnd(c, r, 7) - 0.5) * 0.08
			var ch rune
			var fg, bg col
			if l.smooth {
				ch, fg, bg = shadeCell(c0, 0.18, jit-0.05)
				if strings.IndexRune("#%@*=+", ch) >= 0 {
					ch = '-'
				}
			} else {
				ch, fg, bg = shadeCell(c0, 0.80, jit)
				if pal.calm {
					ch, fg, bg = shadeCell(c0, 0.16, jit-0.04)
					if strings.IndexRune("#%@*=+", ch) >= 0 {
						ch = ':'
					}
				}
				switch {
				case pal.calm:
				case edge > 0.55 && lit > 0.50 && pk.h > 0.2:
					// an arête catching the light
					ch = '|'
					if p < -0.22 {
						ch = '/'
					} else if p > 0.22 {
						ch = '\\'
					}
					fg = mix(c0, pal.rim.mul(k), 0.60).clamp()
				case lit < 0.40 && pk.h > 0.2:
					// engraved hatching in the shade
					dir := 1
					h := (c + r) % 3
					ch = '/'
					if p > 0 {
						ch = '\\'
						dir = -1
						h = (c - r + 300) % 3
					}
					_ = dir
					if h != 0 {
						ch = rune(":;'"[(c+r)%3])
					}
					fg = mix(bg, c0.mul(1.55).lift(0.05), 0.55).clamp()
				}
			}
			cv.paint(c, r, ch, fg, bg)
		}
	}
}

type facetInfo struct{ lit, edge, n float64 }

// facet turns a noise profile into alternating lit and shaded planes. The
// profile is a zig-zag across the face; where it climbs the plane looks away
// from the sun, where it falls the plane catches it, and its turning points
// are the arêtes.
func facet(x float64, seed uint32) facetInfo {
	const e = 0.10
	n := noise1(x, seed)
	d := (noise1(x+e, seed) - noise1(x-e, seed)) / (2 * e)
	lit := sigmoid(-d * 5.0)
	m1 := noise1(x, seed+3)
	edge := 1 - smoothstep(0.0, 0.07, math.Abs(n-0.5))
	_ = m1
	return facetInfo{lit: lit, edge: edge, n: n}
}
