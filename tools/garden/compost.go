package main

import (
	"math"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Composting. Lifting a plant does not simply empty the bed: the plant goes
// back into the ground it grew in, and how much good it does depends on how
// much of it there was.

// compostDuration is how long the plant takes to collapse into the soil on
// screen. Long enough to see, short enough not to be in the way.
const compostDuration = 1300 * time.Millisecond

// compostYield is the richness a lifted plant returns to its bed. A woody
// shrub is worth far more than a seedling, and a plant that has already gone
// to seed is all dry matter, which is exactly what a heap wants.
func compostYield(sp *Species, p *Plot) float64 {
	base := 0.18
	switch sp.Life() {
	case Woody:
		base = 0.34
	case Perennial:
		base = 0.26
	case Biennial:
		base = 0.21
	}

	// Most of the bulk arrives in the second half of a plant's growth.
	size := 0.25 + 0.75*math.Min(1, p.Growth)
	if p.Spent {
		size = 1.1
	}
	return round2(base * size)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// compostFX is a plant on its way into the soil, kept by the model for as
// long as the animation runs.
type compostFX struct {
	Species *Species
	Stage   int
	Gain    float64
	Started time.Time
}

// progress is how far through the collapse we are, from 0 to 1.
func (c compostFX) progress(now time.Time) float64 {
	if c.Started.IsZero() {
		return 1
	}
	p := float64(now.Sub(c.Started)) / float64(compostDuration)
	return math.Max(0, math.Min(1, p))
}

// done reports that the bed can go back to being drawn as bare earth.
func (c compostFX) done(now time.Time) bool { return c.progress(now) >= 1 }

// frame draws the collapse: the plant slumps, browns, and folds down into a
// heap that settles into the soil.
func (c compostFX) frame(width, height int, now time.Time) []string {
	p := c.progress(now)

	switch {
	case p < 0.30:
		// Still standing, but the colour has gone out of it.
		return renderArt(c.Species, driedPalette, c.Stage, width, height, -0.4)

	case p < 0.55:
		// Slumping: a smaller frame, leaning further.
		return renderArt(c.Species, driedPalette, max(0, c.Stage-1), width, height, 0.8)

	case p < 0.80:
		// A heap of stems and leaves.
		return heap(width, height, []string{"≋≋≋≋≋", " ≋≋≋ "})

	default:
		// Worked into the soil, with the goodness showing.
		return heap(width, height, []string{"· ≡ ·", "≡≡≡≡≡"})
	}
}

// heap centres a few rows of litter on the bottom of the bed.
func heap(width, height int, rows []string) []string {
	out := make([]string, 0, height)
	for i := 0; i < height-len(rows); i++ {
		out = append(out, spaces(width))
	}
	for _, r := range rows {
		// Measured in columns, not bytes: the litter is drawn with wide runes.
		left := max(0, (width-lipgloss.Width(r))/2)
		out = append(out, pad(spaces(left)+compostStyle.Render(r), width))
	}
	return out
}

// richnessGlyph is how a bed's soil line reads: hungry ground is thin and
// dotted, well-composted ground is dark and solid.
func richnessGlyph(richness float64) (glyph, color string) {
	switch {
	case richness >= 0.75:
		return "▃", "94"
	case richness >= 0.5:
		return "▂", "137"
	case richness >= 0.25:
		return "▁", "101"
	default:
		return "╌", "240"
	}
}
