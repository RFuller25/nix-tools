package main

import (
	"math"
	"sort"
	"strings"
)

// film is anything that can paint a frame at a given time. Every film runs
// forever: time only ever moves forward and nothing in them repeats visibly.
type film interface {
	draw(t float64) *canvas
}

type filmInfo struct {
	name, blurb string
	make        func(w, h int) film
}

var filmList = []filmInfo{
	{"penguin", "an emperor penguin walks away from the colony, into the mountains", func(w, h int) film { return newScene(w, h) }},
	{"torii", "a vermilion torii in still water, cherry blossom and Fuji at dusk", func(w, h int) film { return newToriiFilm(w, h) }},
	{"aurora", "a winter range under the northern lights, a cabin and its chimney smoke", func(w, h int) film { return newAuroraFilm(w, h) }},
	{"river", "a wide river winding down from the hills, a canoe drifting with it", func(w, h int) film { return newRiverFilm(w, h) }},
	{"lighthouse", "a storm at night: rain, lightning, spray and a turning light", func(w, h int) film { return newLighthouseFilm(w, h) }},
	{"campfire", "a fire in a pine forest, embers rising to the stars", func(w, h int) film { return newCampfireFilm(w, h) }},
	{"deepsea", "jellyfish drifting through light shafts above the kelp", func(w, h int) film { return newDeepSeaFilm(w, h) }},
}

func filmNames() []string {
	var n []string
	for _, f := range filmList {
		n = append(n, f.name)
	}
	return n
}

func filmIndex(name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	for i, f := range filmList {
		if f.name == name {
			return i
		}
	}
	return -1
}

// ---- shared drawing helpers -------------------------------------------------

// glow adds light to a cell's background and glyph.
func (cv *canvas) glow(x, y int, c col, k float64) {
	if cv.in(x, y) {
		p := cv.at(x, y)
		p.bg = p.bg.add(c.mul(k))
		p.fg = p.fg.add(c.mul(k * 0.8))
	}
}

// stamp sets a glyph with the given colour and blends the cell background
// toward bg by a (0 keeps what is there).
func (cv *canvas) stamp(x, y int, ch rune, fg, bg col, a float64) {
	if cv.in(x, y) {
		p := cv.at(x, y)
		p.ch, p.fg = ch, fg
		if a > 0 {
			p.bg = mix(p.bg, bg, a)
		}
	}
}

func (cv *canvas) text(x, y int, s string, fg col, a float64) {
	for i, ch := range s {
		if ch != ' ' && cv.in(x+i, y) {
			p := cv.at(x+i, y)
			p.ch, p.fg = ch, mix(p.bg, fg, a)
		}
	}
}

// fillRows paints a vertical gradient over rows [r0, r1).
func (cv *canvas) fillRows(r0, r1 int, stops []stop, f func(c, r int, base col) col) {
	for r := r0; r < r1 && r < cv.h; r++ {
		v := (float64(r) + 0.5 - float64(r0)) / float64(max(1, r1-r0))
		base := gradient(stops, v)
		for c := 0; c < cv.w; c++ {
			bg := base
			if f != nil {
				bg = f(c, r, base)
			}
			cv.paint(c, r, ' ', col{}, bg.clamp())
		}
	}
}

// twinkle draws a star field that fades where the sky is bright.
func twinkle(cv *canvas, stars []star, t float64) {
	for _, st := range stars {
		if !cv.in(st.x, st.y) {
			continue
		}
		cl := cv.at(st.x, st.y)
		if cl.ch != ' ' {
			continue
		}
		night := 1 - smoothstep(0.03, 0.30, cl.bg.lum())
		a := st.bright * (0.55 + 0.45*math.Sin(t*st.rate+st.phase)) * night
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

// reflect mirrors the picture above row hz into the water below it, dimmed and
// rippled. depth limits how far the reflection reaches in rows.
func reflect(cv *canvas, hz int, t, strength float64, tint col, ripple float64, keepGlyphs bool) {
	flip := map[rune]rune{'/': '\\', '\\': '/', '^': 'v', '`': '.', '\'': '.', '"': '_'}
	src := make([]cell, len(cv.cells))
	copy(src, cv.cells)
	for r := hz; r < cv.h; r++ {
		d := float64(r-hz) + 0.5
		sr := hz - 1 - int(d*1.1)
		if sr < 0 {
			break
		}
		for c := 0; c < cv.w; c++ {
			w := math.Sin(d*1.3+t*1.4+float64(c)*0.05)*ripple + math.Sin(d*0.55-t*0.9)*ripple*0.6
			sc := c + int(math.Round(w))
			if sc < 0 || sc >= cv.w {
				continue
			}
			s := src[sr*cv.w+sc]
			dst := cv.at(c, r)
			k := strength * (1 - 0.65*clamp01(d/float64(cv.h-hz)))
			dst.bg = mix(dst.bg, mix(s.bg, tint, 0.25), k)
			if keepGlyphs && s.ch != ' ' {
				ch := s.ch
				if f, ok := flip[ch]; ok {
					ch = f
				}
				dst.ch, dst.fg = ch, mix(dst.bg, s.fg, 0.7*k/strength)
			}
		}
	}
}

func hashf(i int, salt uint32) float64 { return rnd(i, 7, salt) }

func fract(x float64) float64 { return x - math.Floor(x) }

// wrapf wraps x into [0, n).
func wrapf(x, n float64) float64 { return x - n*math.Floor(x/n) }

var _ = sort.Ints
