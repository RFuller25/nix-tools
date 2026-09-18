package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The garden runs on the real clock, so it also runs on the real light. Dawn
// comes up rose, noon is plain daylight, dusk goes amber and night settles
// into blue — and a few flowers close up for the dark, because they do.

type phase int

const (
	phaseNight phase = iota
	phaseDawn
	phaseMorning
	phaseNoon
	phaseAfternoon
	phaseDusk
)

func (p phase) String() string {
	switch p {
	case phaseDawn:
		return "dawn"
	case phaseMorning:
		return "morning"
	case phaseNoon:
		return "midday"
	case phaseAfternoon:
		return "afternoon"
	case phaseDusk:
		return "dusk"
	default:
		return "night"
	}
}

func (p phase) Glyph() string {
	switch p {
	case phaseDawn:
		return "🌅"
	case phaseMorning, phaseNoon:
		return "☀"
	case phaseAfternoon:
		return "🌤"
	case phaseDusk:
		return "🌆"
	default:
		return "☾"
	}
}

// Dark reports whether the sun is down.
func (p phase) Dark() bool { return p == phaseNight }

// daylight is when the sun is up, in hours after midnight. Summer days are
// long, winter days are short, the way they are.
func daylight(season Season) (sunrise, sunset float64) {
	switch season {
	case Spring:
		return 6.5, 19.5
	case Summer:
		return 5.25, 21.0
	case Autumn:
		return 7.0, 18.5
	default:
		return 8.0, 16.5
	}
}

// phaseAt works out where in the day a moment falls.
func phaseAt(t time.Time) phase {
	season := SeasonOf(t)
	sunrise, sunset := daylight(season)
	h := float64(t.Hour()) + float64(t.Minute())/60

	switch {
	case h < sunrise-1 || h > sunset+1:
		return phaseNight
	case h < sunrise+0.75:
		return phaseDawn
	case h > sunset-0.75:
		return phaseDusk
	case h < (sunrise+sunset)/2-1.5:
		return phaseMorning
	case h > (sunrise+sunset)/2+1.5:
		return phaseAfternoon
	default:
		return phaseNoon
	}
}

// tintFor is the colour the light of a phase washes over the garden, and how
// strongly it does it.
func tintFor(p phase) (rgb, float64) {
	switch p {
	case phaseDawn:
		return rgb{1.00, 0.62, 0.45}, 0.30 // rose
	case phaseMorning:
		return rgb{1.00, 0.97, 0.88}, 0.06
	case phaseNoon:
		return rgb{1, 1, 1}, 0
	case phaseAfternoon:
		return rgb{1.00, 0.90, 0.70}, 0.10
	case phaseDusk:
		return rgb{1.00, 0.55, 0.30}, 0.30 // amber
	default:
		return rgb{0.16, 0.22, 0.55}, 0.55 // deep blue night
	}
}

// rgb is a colour with each channel from 0 to 1.
type rgb struct{ r, g, b float64 }

func (c rgb) hex() string {
	to255 := func(v float64) int {
		i := int(math.Round(v * 255))
		if i < 0 {
			return 0
		}
		if i > 255 {
			return 255
		}
		return i
	}
	return fmt.Sprintf("#%02x%02x%02x", to255(c.r), to255(c.g), to255(c.b))
}

// blend mixes two colours, t running from 0 (all a) to 1 (all b).
func blend(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// parseColor reads either a hex colour or one of the 256 terminal palette
// entries, so a palette written either way can be tinted.
func parseColor(s string) (rgb, bool) {
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		v, err := strconv.ParseUint(s[1:], 16, 32)
		if err != nil {
			return rgb{}, false
		}
		return rgb{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return rgb{}, false
	}
	return xterm(n), true
}

// xterm turns one of the 256 terminal colours into real channel values: the
// sixteen system colours, then a 6×6×6 cube, then a grey ramp.
func xterm(n int) rgb {
	switch {
	case n < 16:
		base := [16]rgb{
			{0, 0, 0}, {0.5, 0, 0}, {0, 0.5, 0}, {0.5, 0.5, 0},
			{0, 0, 0.5}, {0.5, 0, 0.5}, {0, 0.5, 0.5}, {0.75, 0.75, 0.75},
			{0.5, 0.5, 0.5}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0},
			{0, 0, 1}, {1, 0, 1}, {0, 1, 1}, {1, 1, 1},
		}
		return base[n]
	case n < 232:
		n -= 16
		steps := [6]float64{0, 95.0 / 255, 135.0 / 255, 175.0 / 255, 215.0 / 255, 1}
		return rgb{steps[(n/36)%6], steps[(n/6)%6], steps[n%6]}
	default:
		v := (float64(n-232)*10 + 8) / 255
		return rgb{v, v, v}
	}
}

// tint washes one colour with the light of the moment.
func tint(color string, p phase) string {
	target, strength := tintFor(p)
	if strength == 0 || color == "" {
		return color
	}
	base, ok := parseColor(color)
	if !ok {
		return color
	}
	lit := blend(base, target, strength)
	if p == phaseNight {
		// Night is dim as well as blue: the garden is still there, you just
		// cannot see much of it.
		lit = rgb{lit.r * 0.72, lit.g * 0.72, lit.b * 0.8}
	}
	return lit.hex()
}

// tintPalette puts a whole plant under the light of the moment.
func tintPalette(p Palette, ph phase) Palette {
	return Palette{
		Stem:   tint(p.Stem, ph),
		Leaf:   tint(p.Leaf, ph),
		Bloom:  tint(p.Bloom, ph),
		Accent: tint(p.Accent, ph),
	}
}

// closers shut their flowers for the night and open again with the light.
var closers = map[string]bool{
	"crocus": true, "tulip": true, "waterlily": true, "lotus": true,
	"morningglory": true, "chamomile": true, "snowdrop": true,
}

// nightFlowers do the opposite: shut through the day, open at dusk, and scent
// the dark for the moths that pollinate them.
var nightFlowers = map[string]bool{
	"moonflower": true, "eveningprimrose": true, "nightstock": true,
}

// nightScented carry their scent after dark, whether or not they close.
var nightScented = map[string]bool{
	"honeysuckle": true, "moonflower": true, "eveningprimrose": true,
	"nightstock": true, "lilyvalley": true,
}

// ClosesAtNight reports whether the flowers shut for the dark.
func (s *Species) ClosesAtNight() bool { return closers[s.ID] }

// OpensAtNight reports whether the flowers wait for the dark.
func (s *Species) OpensAtNight() bool { return nightFlowers[s.ID] }

// NightScented reports whether the plant perfumes the evening.
func (s *Species) NightScented() bool { return nightScented[s.ID] }
