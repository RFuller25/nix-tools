package main

import (
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Colour. Flowers in this garden can be any shade a terminal can show, so
// colours are carried as real channel values and written out as #rrggbb.
// lipgloss and termenv emit 24-bit colour when the terminal advertises it
// (COLORTERM=truecolor) and quietly snap to the nearest 256- or 16-colour
// entry when it does not.

// hsl is a colour as hue (degrees), saturation and lightness (0 to 1).
type hsl struct{ h, s, l float64 }

func rgbToHSL(c rgb) hsl {
	maxC := math.Max(c.r, math.Max(c.g, c.b))
	minC := math.Min(c.r, math.Min(c.g, c.b))
	l := (maxC + minC) / 2
	d := maxC - minC
	if d < 1e-9 {
		return hsl{0, 0, l}
	}
	s := d / (1 - math.Abs(2*l-1))
	var h float64
	switch maxC {
	case c.r:
		h = math.Mod((c.g-c.b)/d, 6)
	case c.g:
		h = (c.b-c.r)/d + 2
	default:
		h = (c.r-c.g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return hsl{h, clamp01(s), clamp01(l)}
}

func hslToRGB(c hsl) rgb {
	h := math.Mod(c.h, 360)
	if h < 0 {
		h += 360
	}
	chroma := (1 - math.Abs(2*c.l-1)) * c.s
	x := chroma * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := c.l - chroma/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = chroma, x, 0
	case h < 120:
		r, g, b = x, chroma, 0
	case h < 180:
		r, g, b = 0, chroma, x
	case h < 240:
		r, g, b = 0, x, chroma
	case h < 300:
		r, g, b = x, 0, chroma
	default:
		r, g, b = chroma, 0, x
	}
	return rgb{clamp01(r + m), clamp01(g + m), clamp01(b + m)}
}

// circularMean averages two hues along the shorter arc between them, so red
// and blue make magenta rather than the green on the far side of the wheel.
func circularMean(a, b float64) float64 {
	ra, rb := a*math.Pi/180, b*math.Pi/180
	x := math.Cos(ra) + math.Cos(rb)
	y := math.Sin(ra) + math.Sin(rb)
	if math.Abs(x) < 1e-9 && math.Abs(y) < 1e-9 {
		return a // exact opposites: no shorter arc, keep the first
	}
	h := math.Atan2(y, x) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}

// hueDistance is the shorter way round the wheel between two hues, 0 to 180.
func hueDistance(a, b float64) float64 {
	d := math.Abs(math.Mod(a-b, 360))
	if d > 180 {
		d = 360 - d
	}
	return d
}

// colourName puts a name to a shade, so a gardener (and an order) can talk
// about it: coral, violet, cream.
func colourName(c rgb) string {
	v := rgbToHSL(c)
	switch {
	case v.l >= 0.93:
		return "white"
	case v.l <= 0.10:
		return "black"
	case v.s < 0.12:
		if v.l > 0.7 {
			return "silver"
		}
		if v.l < 0.3 {
			return "charcoal"
		}
		return "grey"
	}
	light, dark := v.l > 0.72, v.l < 0.30
	h := v.h
	switch {
	case h >= 345 || h < 12:
		switch {
		case light:
			return "pink"
		case dark:
			return "maroon"
		}
		return "red"
	case h < 22:
		if light {
			return "coral"
		}
		return "scarlet"
	case h < 40:
		switch {
		case light:
			return "peach"
		case dark:
			return "brown"
		}
		return "orange"
	case h < 52:
		if light {
			return "cream"
		}
		return "gold"
	case h < 68:
		if light {
			return "pale yellow"
		}
		return "yellow"
	case h < 95:
		return "lime"
	case h < 165:
		if dark {
			return "forest green"
		}
		return "green"
	case h < 195:
		return "teal"
	case h < 215:
		return "sky blue"
	case h < 255:
		if dark {
			return "navy"
		}
		return "blue"
	case h < 285:
		if light {
			return "lavender"
		}
		return "violet"
	case h < 315:
		if light {
			return "lilac"
		}
		return "purple"
	default:
		if light {
			return "rose"
		}
		return "magenta"
	}
}

// colourDistance is a cheap perceptual-ish distance between two shades, 0 to
// about 1, weighting hue most for saturated colours.
func colourDistance(a, b rgb) float64 {
	ha, hb := rgbToHSL(a), rgbToHSL(b)
	sat := math.Min(ha.s, hb.s)
	dh := hueDistance(ha.h, hb.h) / 180 * sat
	ds := math.Abs(ha.s - hb.s)
	dl := math.Abs(ha.l - hb.l)
	return math.Sqrt(dh*dh*2 + ds*ds*0.6 + dl*dl*1.2)
}

// swatch draws a block of a colour, for the shed, the almanac and orders.
func swatch(hex string, width int) string {
	if width < 1 {
		width = 2
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(strings.Repeat("█", width))
}

// ColourMode is how the garden was asked to talk to the terminal.
type ColourMode string

const (
	colourAuto  ColourMode = ""
	colourTrue  ColourMode = "truecolor"
	colour256   ColourMode = "256"
	colour16    ColourMode = "16"
	colourNone  ColourMode = "off"
	envColour              = "GARDEN_COLOR"
	envNoColour            = "NO_COLOR"
)

// parseColourMode reads a --color flag or GARDEN_COLOR value.
func parseColourMode(s string) (ColourMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return colourAuto, true
	case "truecolor", "true", "24bit", "24-bit", "rgb":
		return colourTrue, true
	case "256", "ansi256":
		return colour256, true
	case "16", "ansi":
		return colour16, true
	case "off", "none", "mono", "0":
		return colourNone, true
	}
	return colourAuto, false
}

// applyColourMode pins lipgloss to a colour profile. Auto leaves termenv's own
// detection alone, which reads COLORTERM and TERM. NO_COLOR is honoured unless
// a mode was asked for explicitly.
func applyColourMode(flag string) (ColourMode, error) {
	raw := flag
	if raw == "" {
		raw = os.Getenv(envColour)
	}
	mode, ok := parseColourMode(raw)
	if !ok {
		return colourAuto, errBadColour(raw)
	}
	if mode == colourAuto && os.Getenv(envNoColour) != "" {
		mode = colourNone
	}
	switch mode {
	case colourTrue:
		lipgloss.SetColorProfile(termenv.TrueColor)
	case colour256:
		lipgloss.SetColorProfile(termenv.ANSI256)
	case colour16:
		lipgloss.SetColorProfile(termenv.ANSI)
	case colourNone:
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	return mode, nil
}

type errBadColour string

func (e errBadColour) Error() string {
	return "unknown colour mode \"" + string(e) + "\" (want truecolor, 256, 16 or off)"
}

// circularWeighted is the mean of two hues on the wheel, each weighted, taking
// the shorter arc. A weight of zero for one side returns the other's hue.
func circularWeighted(ha, wa, hb, wb float64) float64 {
	x := wa*math.Cos(ha*math.Pi/180) + wb*math.Cos(hb*math.Pi/180)
	y := wa*math.Sin(ha*math.Pi/180) + wb*math.Sin(hb*math.Pi/180)
	if math.Abs(x) < 1e-9 && math.Abs(y) < 1e-9 {
		return ha
	}
	h := math.Atan2(y, x) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}
