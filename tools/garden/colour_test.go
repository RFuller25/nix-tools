package main

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestHSLRoundTrip(t *testing.T) {
	for n := 0; n < 256; n++ {
		c := xterm(n)
		back := hslToRGB(rgbToHSL(c))
		if math.Abs(back.r-c.r) > 0.005 || math.Abs(back.g-c.g) > 0.005 || math.Abs(back.b-c.b) > 0.005 {
			t.Fatalf("xterm %d: %v came back as %v", n, c, back)
		}
	}
}

func TestHexParsesBackToTheSameColour(t *testing.T) {
	c := rgb{0.2, 0.55, 0.9}
	back, ok := parseColor(c.hex())
	if !ok || math.Abs(back.r-c.r) > 0.003 || math.Abs(back.b-c.b) > 0.003 {
		t.Errorf("%v -> %s -> %v", c, c.hex(), back)
	}
}

func TestCircularMeanTakesTheShortArc(t *testing.T) {
	cases := []struct{ a, b, want float64 }{
		{0, 60, 30},   // red and yellow: orange
		{350, 10, 0},  // across the seam
		{0, 240, 300}, // red and blue: magenta, not green
		{120, 180, 150},
	}
	for _, c := range cases {
		if got := circularMean(c.a, c.b); hueDistance(got, c.want) > 0.5 {
			t.Errorf("mean(%v,%v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestColourNames(t *testing.T) {
	cases := map[string]hsl{
		"red":     {0, 0.8, 0.5},
		"pink":    {350, 0.8, 0.85},
		"yellow":  {58, 0.9, 0.5},
		"blue":    {230, 0.8, 0.5},
		"violet":  {270, 0.7, 0.5},
		"white":   {0, 0, 0.98},
		"black":   {0, 0, 0.03},
		"green":   {130, 0.7, 0.45},
		"magenta": {330, 0.8, 0.5},
	}
	for want, c := range cases {
		if got := colourName(hslToRGB(c)); got != want {
			t.Errorf("%v named %q, want %q", c, got, want)
		}
	}
}

func TestColourModes(t *testing.T) {
	for in, want := range map[string]ColourMode{
		"": colourAuto, "auto": colourAuto, "truecolor": colourTrue, "24bit": colourTrue,
		"256": colour256, "16": colour16, "off": colourNone, "NONE": colourNone,
	} {
		got, ok := parseColourMode(in)
		if !ok || got != want {
			t.Errorf("parseColourMode(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := parseColourMode("sepia"); ok {
		t.Error("a nonsense mode was accepted")
	}
}

// With a true-colour terminal the garden writes real 24-bit colour, and a
// bred shade reaches the screen as itself rather than as the nearest of 256.
func TestTrueColourReachesTheScreen(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	g := newTestGarden(testStart())
	g.Gold = 100
	sp := SpeciesByID("cosmos")
	if err := g.Plant(0, sp, 0, testStart()); err != nil {
		t.Fatal(err)
	}
	gn := sp.VarietyGenome(0)
	gn.Hue, gn.Sat, gn.Light = 171, 83, 41 // a teal no 256-colour entry holds exactly
	g.Plots[0].Genome = gn
	g.Plots[0].Growth = 1
	m := newModel(g, "/tmp/g.json", testStart())
	m.width, m.height = 100, 40
	m.now = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) // noon: no tint over the colour

	want := gn.RGB()
	needle := fmt.Sprintf("38;2;%d;%d;%d", int(math.Round(want.r*255)), int(math.Round(want.g*255)), int(math.Round(want.b*255)))
	if !strings.Contains(m.View(), needle) {
		t.Errorf("the bed does not carry the bred colour %s (%s)", gn.Hex(), needle)
	}

	lipgloss.SetColorProfile(termenv.ANSI256)
	if strings.Contains(m.View(), "38;2;") {
		t.Error("a 256-colour terminal was sent 24-bit colour")
	}
	lipgloss.SetColorProfile(termenv.Ascii)
	if strings.Contains(m.View(), "\x1b[") {
		t.Error("colour was sent with colour turned off")
	}
}
