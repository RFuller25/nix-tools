package main

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// glyphClass says which part of a plant a rune in an art frame represents, so
// a single palette can colour every frame without hand-painting each cell.
type glyphClass int

const (
	clPlain glyphClass = iota
	clStem
	clLeaf
	clBloom
	clAccent
	clSoil
)

var defaultGlyphs = map[rune]glyphClass{}

func init() {
	classify := func(class glyphClass, runes string) {
		for _, r := range runes {
			defaultGlyphs[r] = class
		}
	}
	classify(clStem, `|/\┃│┆┊╱╲╽¦⌇ǀ┋╿┇`)
	classify(clLeaf, `^vwWmM<>~≈)(}{ϑεζξϟᐯᐱ⌄⌃╰╯╭╮┌┐└┘─━╵╷λγψ`)
	classify(clBloom, `*✿❀❁❃❋✽✼✻❉✾@&ღ♣♠oO`)
	classify(clAccent, `●◍◉○◦•★✦♥▲▼◆◇▰▪▫§¤`)
	classify(clSoil, `.,_˳·˙¸`)
}

func classOf(sp *Species, r rune) glyphClass {
	if sp != nil && sp.Glyphs != nil {
		if c, ok := sp.Glyphs[r]; ok {
			return c
		}
	}
	if c, ok := defaultGlyphs[r]; ok {
		return c
	}
	return clPlain
}

var soilStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("94"))

// litStyle is a foreground colour under the light of the moment.
func litStyle(color string, ph phase) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(tint(color, ph)))
}

func colorFor(pal Palette, class glyphClass) string {
	p := pal
	switch class {
	case clStem:
		return fallback(p.Stem, "71")
	case clLeaf:
		return fallback(p.Leaf, "77")
	case clBloom:
		return fallback(p.Bloom, "218")
	case clAccent:
		return fallback(p.Accent, "223")
	case clSoil:
		return "94"
	default:
		return fallback(p.Leaf, "77")
	}
}

func fallback(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// colorizeLine paints one line of art using a palette, which is usually the
// species' own but may be adjusted by the bed it is growing in.
func colorizeLine(sp *Species, pal Palette, line string) string {
	var b strings.Builder
	var run []rune
	runClass := clPlain
	flush := func() {
		if len(run) == 0 {
			return
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFor(pal, runClass)))
		b.WriteString(style.Render(string(run)))
		run = run[:0]
	}
	for _, r := range line {
		if r == ' ' {
			flush()
			b.WriteRune(' ')
			continue
		}
		c := classOf(sp, r)
		if len(run) > 0 && c != runClass {
			flush()
		}
		runClass = c
		run = append(run, r)
	}
	flush()
	return b.String()
}

// renderArt lays a species' stage frame into a box of the given size: bottom
// aligned so plants stand on the soil, and centred as a whole. The frame is
// offset by a single amount rather than centring each line on its own, so a
// drawing keeps the shape it was drawn with.
//
// sway leans the plant in the wind, from -1 to 1. The lean is strongest at the
// top and nothing at all at the base, so the plant bends rather than slides,
// and it is clamped to the space left inside the bed so nothing is clipped.
func renderArt(sp *Species, pal Palette, stage, width, height int, sway float64) []string {
	return renderArtWith(sp, pal, stage, width, height, sway, nil)
}

// renderArtWith is renderArt with things drawn over the top of the plant: a
// bee working the flowers, a bird on the seed heads. The overlay is keyed by
// row and column within the box, and each entry is already styled.
func renderArtWith(sp *Species, pal Palette, stage, width, height int, sway float64, overlay map[[2]int]string) []string {
	grid := artGrid(sp, stage, width, height, sway)

	out := make([]string, 0, height)
	for row, runes := range grid {
		out = append(out, paintRow(sp, pal, runes, row, overlay))
	}
	return out
}

// artGrid lays the frame out as plain runes, one row per line of the box.
func artGrid(sp *Species, stage, width, height int, sway float64) [][]rune {
	blank := func() []rune {
		row := make([]rune, width)
		for i := range row {
			row[i] = ' '
		}
		return row
	}

	frame := sp.Stage(stage)
	if len(frame) > height {
		frame = frame[len(frame)-height:] // keep the base of an over-tall frame
	}

	grid := make([][]rune, 0, height)
	for i := 0; i < height-len(frame); i++ {
		grid = append(grid, blank())
	}

	frameWidth := 0
	for _, line := range frame {
		if w := lipgloss.Width(line); w > frameWidth {
			frameWidth = w
		}
	}
	left := (width - frameWidth) / 2
	if left < 0 {
		left = 0
	}

	// How far the top of the plant may lean without leaving the bed.
	room := min(left, width-frameWidth-left)
	if room > 2 {
		room = 2
	}

	for i, line := range frame {
		row := blank()
		shift := 0
		if room > 0 && len(frame) > 1 {
			lean := float64(len(frame)-1-i) / float64(len(frame)-1) // 1 at the tip, 0 at the base
			shift = int(math.Round(sway * float64(room) * lean))
		}
		col := max(0, left+shift)
		for _, r := range line {
			if col >= width {
				break
			}
			row[col] = r
			col++
		}
		grid = append(grid, row)
	}
	return grid
}

// paintRow colours one row of the grid, letting any overlay take precedence.
func paintRow(sp *Species, pal Palette, runes []rune, row int, overlay map[[2]int]string) string {
	var b strings.Builder
	var run []rune
	runClass := clPlain

	flush := func() {
		if len(run) == 0 {
			return
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFor(pal, runClass)))
		b.WriteString(style.Render(string(run)))
		run = run[:0]
	}

	for col, r := range runes {
		if glyph, ok := overlay[[2]int{row, col}]; ok {
			flush()
			b.WriteString(glyph)
			continue
		}
		if r == ' ' {
			flush()
			b.WriteRune(' ')
			continue
		}
		c := classOf(sp, r)
		if len(run) > 0 && c != runClass {
			flush()
		}
		runClass = c
		run = append(run, r)
	}
	flush()
	return b.String()
}

func trimToWidth(line string, width int) string {
	runes := []rune(line)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

// soilLine draws the strip of earth a plant stands on, dusted with weeds as
// they take hold, and lit by the time of day.
func soilLine(width int, weeds float64, planted bool, ph phase, richness float64) string {
	base, soilColor := richnessGlyph(richness)
	ground := []rune(strings.Repeat(base, width))
	switch {
	case weeds > 0.75:
		for i := 1; i < width; i += 2 {
			ground[i] = '⌄'
		}
	case weeds > 0.45:
		for i := 2; i < width; i += 4 {
			ground[i] = '⌄'
		}
	case weeds > 0.2:
		if width > 4 {
			ground[width/2] = '⌄'
		}
	}
	line := string(ground)
	switch {
	case weeds > 0.2:
		return litStyle("101", ph).Render(line)
	case planted:
		return litStyle(soilColor, ph).Render(line)
	default:
		return litStyle(soilColor, ph).Render(line)
	}
}
