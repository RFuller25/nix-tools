package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// The catalogue is the heart of the tool, so it gets checked hard: every
// species must be complete, unique and drawable inside a bed.
func TestCatalogueIsComplete(t *testing.T) {
	all := AllSpecies()
	if len(all) < 50 {
		t.Fatalf("catalogue holds %d species, want at least 50", len(all))
	}

	ids := map[string]bool{}
	latin := map[string]string{}
	for _, sp := range all {
		if ids[sp.ID] {
			t.Errorf("duplicate species id %q", sp.ID)
		}
		ids[sp.ID] = true

		if prev, ok := latin[sp.Latin]; ok {
			t.Errorf("latin name %q used by both %s and %s", sp.Latin, prev, sp.ID)
		}
		latin[sp.Latin] = sp.ID

		fields := map[string]string{
			"Common": sp.Common, "Latin": sp.Latin, "Family": sp.Family,
			"Origin": sp.Origin, "Bloom": sp.Bloom, "Sun": sp.Sun,
			"Water": sp.Water, "Height": sp.Height, "Note": sp.Note, "Desc": sp.Desc,
		}
		for name, v := range fields {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: %s is empty", sp.ID, name)
			}
		}
		if len(sp.Desc) < 80 {
			t.Errorf("%s: description is only %d characters, wanted something substantial", sp.ID, len(sp.Desc))
		}
		if !strings.Contains(sp.Latin, " ") {
			t.Errorf("%s: latin name %q is not binomial", sp.ID, sp.Latin)
		}
		if len(sp.Seasons) == 0 {
			t.Errorf("%s: no growing seasons", sp.ID)
		}
		if sp.SeedCost < 1 {
			t.Errorf("%s: seed cost %d", sp.ID, sp.SeedCost)
		}
		if sp.Unlock < 0 {
			t.Errorf("%s: negative unlock threshold", sp.ID)
		}
	}
}

// Every species must reach adulthood inside a day of well-tended growth.
func TestMaturesWithinADay(t *testing.T) {
	for _, sp := range AllSpecies() {
		if sp.Hours <= 0 || sp.Hours > 20 {
			t.Errorf("%s: matures in %.1f hours, want (0, 20]", sp.ID, sp.Hours)
		}
	}
}

// Five distinct drawings per plant: no blanks, no repeats, nothing too wide
// for a bed, and each stage no smaller than the one before.
func TestArtwork(t *testing.T) {
	for _, sp := range AllSpecies() {
		seen := map[string]int{}
		prevLines := 0
		for stage := 0; stage < StageCount; stage++ {
			frame := sp.Stage(stage)
			if len(frame) == 0 {
				t.Errorf("%s: stage %s has no artwork", sp.ID, stageNames[stage])
				continue
			}
			joined := strings.Join(frame, "\n")
			if strings.TrimSpace(joined) == "" {
				t.Errorf("%s: stage %s is blank", sp.ID, stageNames[stage])
			}
			if prev, ok := seen[joined]; ok {
				t.Errorf("%s: stages %s and %s are drawn identically", sp.ID, stageNames[prev], stageNames[stage])
			}
			seen[joined] = stage

			for _, line := range frame {
				if w := lipgloss.Width(line); w > cellInner {
					t.Errorf("%s: stage %s line %q is %d wide, max %d", sp.ID, stageNames[stage], line, w, cellInner)
				}
			}
			if len(frame) > artHeight {
				t.Errorf("%s: stage %s is %d lines tall, max %d", sp.ID, stageNames[stage], len(frame), artHeight)
			}
			if len(frame) < prevLines {
				t.Errorf("%s: stage %s shrinks from %d to %d lines", sp.ID, stageNames[stage], prevLines, len(frame))
			}
			prevLines = len(frame)
		}
	}
}

// Rarer plants should cost more and unlock later than common ones.
func TestRarityIsOrdered(t *testing.T) {
	bounds := map[Rarity][2]int{
		Common:    {1, 5},
		Uncommon:  {4, 9},
		Rare:      {7, 16},
		Legendary: {12, 30},
	}
	for _, sp := range AllSpecies() {
		b := bounds[sp.Rarity]
		if sp.SeedCost < b[0] || sp.SeedCost > b[1] {
			t.Errorf("%s: %s costs %d seeds, want %d-%d", sp.ID, sp.Rarity, sp.SeedCost, b[0], b[1])
		}
	}
	// Something must be plantable from the very first seed tin.
	starters := 0
	for _, sp := range AllSpecies() {
		if sp.Unlock == 0 && sp.SeedCost <= 3 {
			starters++
		}
	}
	if starters < 5 {
		t.Errorf("only %d cheap starter species, want at least 5", starters)
	}
}

func TestLookupByID(t *testing.T) {
	for _, sp := range AllSpecies() {
		if got := SpeciesByID(sp.ID); got != sp {
			t.Errorf("SpeciesByID(%q) did not round-trip", sp.ID)
		}
	}
	if SpeciesByID("no-such-plant") != nil {
		t.Error("unknown id should resolve to nil")
	}
}
