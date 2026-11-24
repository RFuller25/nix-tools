package main

import (
	"strconv"
	"strings"

	"github.com/muesli/termenv"
)

// cell is one terminal character with a foreground and background colour.
type cell struct {
	ch     rune
	fg, bg col
}

type canvas struct {
	w, h  int
	cells []cell
}

func newCanvas(w, h int) *canvas {
	cv := &canvas{w: w, h: h, cells: make([]cell, w*h)}
	for i := range cv.cells {
		cv.cells[i].ch = ' '
	}
	return cv
}

func (cv *canvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < cv.w && y < cv.h }

func (cv *canvas) at(x, y int) *cell { return &cv.cells[y*cv.w+x] }

// paint replaces a whole cell.
func (cv *canvas) paint(x, y int, ch rune, fg, bg col) {
	if cv.in(x, y) {
		*cv.at(x, y) = cell{ch, fg, bg}
	}
}

// ink draws a glyph over whatever background is already there.
func (cv *canvas) ink(x, y int, ch rune, fg col) {
	if cv.in(x, y) {
		c := cv.at(x, y)
		c.ch, c.fg = ch, fg
	}
}

// tint blends the background (and the glyph colour) toward c.
func (cv *canvas) tint(x, y int, c col, t float64) {
	if cv.in(x, y) {
		p := cv.at(x, y)
		p.bg = mix(p.bg, c, t)
		p.fg = mix(p.fg, c, t)
	}
}

func (cv *canvas) fade(k float64) {
	if k >= 0.999 {
		return
	}
	for i := range cv.cells {
		cv.cells[i].fg = cv.cells[i].fg.mul(k)
		cv.cells[i].bg = cv.cells[i].bg.mul(k)
	}
}

func b8(x float64) int {
	v := int(clamp01(x)*255 + 0.5)
	return v
}

// to256 maps an RGB triple onto the xterm 256-colour palette.
func to256(r, g, b int) int {
	q := func(v int) int {
		switch {
		case v < 48:
			return 0
		case v < 115:
			return 1
		}
		return (v-35)/40 + 0
	}
	qr, qg, qb := q(r), q(g), q(b)
	step := [...]int{0, 95, 135, 175, 215, 255}
	cr, cg, cb := step[min(qr, 5)], step[min(qg, 5)], step[min(qb, 5)]
	cube := 16 + 36*min(qr, 5) + 6*min(qg, 5) + min(qb, 5)
	avg := (r + g + b) / 3
	gi := (avg - 8 + 5) / 10
	if gi < 0 {
		gi = 0
	}
	if gi > 23 {
		gi = 23
	}
	gv := 8 + 10*gi
	d := func(a, b int) int { return (a - b) * (a - b) }
	dc := d(r, cr) + d(g, cg) + d(b, cb)
	dg := d(r, gv) + d(g, gv) + d(b, gv)
	if dg < dc {
		return 232 + gi
	}
	return cube
}

// render turns the canvas into escape-sequence text. Colour is emitted only
// when it changes so a frame stays a few kilobytes.
func (cv *canvas) render(p termenv.Profile) string {
	var sb strings.Builder
	sb.Grow(cv.w * cv.h * 8)
	for y := 0; y < cv.h; y++ {
		if y > 0 {
			sb.WriteByte('\n')
		}
		var pfr, pfg, pfb, pbr, pbg, pbb = -1, -1, -1, -1, -1, -1
		for x := 0; x < cv.w; x++ {
			c := cv.at(x, y)
			if p == termenv.Ascii {
				sb.WriteRune(c.ch)
				continue
			}
			br, bg, bb := b8(c.bg.r), b8(c.bg.g), b8(c.bg.b)
			if br != pbr || bg != pbg || bb != pbb {
				if p == termenv.TrueColor {
					sb.WriteString("\x1b[48;2;")
					sb.WriteString(strconv.Itoa(br))
					sb.WriteByte(';')
					sb.WriteString(strconv.Itoa(bg))
					sb.WriteByte(';')
					sb.WriteString(strconv.Itoa(bb))
					sb.WriteByte('m')
				} else {
					sb.WriteString("\x1b[48;5;")
					sb.WriteString(strconv.Itoa(to256(br, bg, bb)))
					sb.WriteByte('m')
				}
				pbr, pbg, pbb = br, bg, bb
			}
			if c.ch != ' ' {
				fr, fg, fb := b8(c.fg.r), b8(c.fg.g), b8(c.fg.b)
				if fr != pfr || fg != pfg || fb != pfb {
					if p == termenv.TrueColor {
						sb.WriteString("\x1b[38;2;")
						sb.WriteString(strconv.Itoa(fr))
						sb.WriteByte(';')
						sb.WriteString(strconv.Itoa(fg))
						sb.WriteByte(';')
						sb.WriteString(strconv.Itoa(fb))
						sb.WriteByte('m')
					} else {
						sb.WriteString("\x1b[38;5;")
						sb.WriteString(strconv.Itoa(to256(fr, fg, fb)))
						sb.WriteByte('m')
					}
					pfr, pfg, pfb = fr, fg, fb
				}
			}
			sb.WriteRune(c.ch)
		}
		if p != termenv.Ascii {
			sb.WriteString("\x1b[0m")
		}
	}
	return sb.String()
}

// ramp orders glyphs from empty to dense; with the foreground lighter than the
// background, a denser glyph reads as a brighter surface.
const ramp = " .'`:;-+=*#%@"

// shadeCell picks a glyph and a fg/bg pair for a surface of colour c. contrast
// controls how far apart glyph and background sit; jitter nudges the glyph
// choice so flat areas get a grain instead of a stamped look.
func shadeCell(c col, contrast, jitter float64) (rune, col, col) {
	c = c.clamp()
	l := clamp01(c.lum()*1.15 + jitter)
	i := int(pow(l, 0.8)*float64(len(ramp)-1) + 0.5)
	if i >= len(ramp) {
		i = len(ramp) - 1
	}
	bg := c.mul(1 - 0.22*contrast)
	fg := c.mul(1 + 0.38*contrast).lift(0.035 * contrast)
	return rune(ramp[i]), fg.clamp(), bg
}
