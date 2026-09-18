package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Art frames are centred as a block, so every line in a frame must be the
// same width or the drawing leans.
func TestFrameLinesShareAWidth(t *testing.T) {
	for _, sp := range AllSpecies() {
		for stage := 0; stage < StageCount; stage++ {
			frame := sp.Stage(stage)
			want := lipgloss.Width(frame[0])
			for i, line := range frame {
				if got := lipgloss.Width(line); got != want {
					t.Errorf("%s stage %s line %d is %d wide, first line is %d: %q",
						sp.ID, stageNames[stage], i, got, want, line)
				}
			}
		}
	}
}

func TestRenderArtFillsItsBox(t *testing.T) {
	sp := SpeciesByID("sunflower")
	for _, size := range [][2]int{{13, 5}, {11, 6}, {24, 7}, {5, 2}} {
		lines := renderArt(sp, sp.Palette, StageMature, size[0], size[1], 0)
		if len(lines) != size[1] {
			t.Errorf("%dx%d box got %d lines", size[0], size[1], len(lines))
		}
		for _, l := range lines {
			if got := lipgloss.Width(l); got != size[0] {
				t.Errorf("%dx%d box produced a %d-wide line %q", size[0], size[1], got, l)
			}
		}
	}
}

// Artwork sits on the soil: the frame is bottom-aligned in its box.
func TestArtIsBottomAligned(t *testing.T) {
	sp := SpeciesByID("crocus")
	lines := renderArt(sp, sp.Palette, StageSeed, 13, 5, 0)
	for i := 0; i < 4; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			t.Errorf("line %d should be empty sky, got %q", i, lines[i])
		}
	}
	if strings.TrimSpace(lines[4]) == "" {
		t.Error("the seed should rest on the bottom line")
	}
}

func TestColorizeKeepsTheText(t *testing.T) {
	sp := SpeciesByID("rose")
	for _, line := range sp.Stage(StageMature) {
		painted := colorizeLine(sp, sp.Palette, line)
		if lipgloss.Width(painted) != lipgloss.Width(line) {
			t.Errorf("colouring changed the width of %q", line)
		}
	}
}

// Weeds should show in the soil, and bare beds should read as bare.
func TestSoilLineReflectsWeeds(t *testing.T) {
	clean := soilLine(13, 0, true)
	weedy := soilLine(13, 0.9, true)
	if lipgloss.Width(clean) != 13 || lipgloss.Width(weedy) != 13 {
		t.Fatal("soil lines must fill the bed's width")
	}
	if !strings.Contains(weedy, "⌄") {
		t.Error("an overgrown bed should show weeds in the soil")
	}
	if strings.Contains(clean, "⌄") {
		t.Error("a clean bed should show none")
	}
}
