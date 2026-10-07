package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// Whatever the genes, a drawing stays rectangular and inside its box.
func TestGeneFramesStayInsideTheirBox(t *testing.T) {
	genomes := []Genome{
		{Hue: 10, Sat: 80, Light: 50, Height: 0, Shape: 0, Speed: 50, Yield: 50},
		{Hue: 10, Sat: 80, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50},
		{Hue: 10, Sat: 80, Light: 50, Height: 100, Shape: 100, Speed: 50, Yield: 50},
		{Hue: 10, Sat: 80, Light: 50, Height: 100, Shape: 0, Speed: 50, Yield: 50},
		{Hue: 10, Sat: 80, Light: 50, Height: 0, Shape: 100, Speed: 50, Yield: 50},
	}
	boxes := [][2]int{{cellInner, artHeight}, {24, 7}, {11, 6}}
	for _, sp := range AllSpecies() {
		for vi := range sp.Varieties() {
			for stage := 0; stage < StageCount; stage++ {
				base := sp.StageFor(vi, stage)
				for _, gn := range genomes {
					for _, box := range boxes {
						frame := geneFrame(sp, base, gn, box[0], box[1])
						if len(frame) == 0 {
							t.Fatalf("%s form %d %s lost its drawing", sp.ID, vi, stageNames[stage])
						}
						w := lipgloss.Width(frame[0])
						for _, row := range frame {
							if lipgloss.Width(row) != w {
								t.Fatalf("%s form %d %s: ragged rows %q", sp.ID, vi, stageNames[stage], frame)
							}
						}
						if len(base) <= box[1] && len(frame) > box[1] {
							t.Errorf("%s form %d %s grew to %d rows in a box of %d", sp.ID, vi, stageNames[stage], len(frame), box[1])
						}
						if lipgloss.Width(frame[0]) > max(box[0], lipgloss.Width(base[0])) {
							t.Errorf("%s form %d %s widened to %d in a box of %d", sp.ID, vi, stageNames[stage], w, box[0])
						}
					}
				}
			}
		}
	}
}

func TestTallPlantsGetLongerStemsWhereThereIsRoom(t *testing.T) {
	sp := SpeciesByID("sunflower")
	base := sp.StageFor(0, StageMature)
	tall := geneFrame(sp, base, Genome{Height: 100, Shape: 50}, 24, 9)
	if len(tall) <= len(base) {
		t.Errorf("a giant sunflower is %d rows, the plain one %d", len(tall), len(base))
	}

	// Dwarfs lose a stem row wherever there is more than one to lose.
	shorter := 0
	for _, sp := range AllSpecies() {
		base := sp.StageFor(0, StageMature)
		if len(geneFrame(sp, base, Genome{Height: 0, Shape: 50}, 24, 9)) < len(base) {
			shorter++
		}
	}
	if shorter < 8 {
		t.Errorf("only %d species draw a dwarf shorter than the plain form", shorter)
	}
}

func TestFullPlantsAreWiderThanSlimOnes(t *testing.T) {
	sp := SpeciesByID("sunflower")
	base := sp.StageFor(0, StageMature)
	count := func(frame []string) int {
		n := 0
		for _, row := range frame {
			for _, r := range row {
				if classOf(sp, r) == clBloom {
					n++
				}
			}
		}
		return n
	}
	full := count(geneFrame(sp, base, Genome{Height: 50, Shape: 100}, 24, 9))
	slim := count(geneFrame(sp, base, Genome{Height: 50, Shape: 0}, 24, 9))
	if full <= slim {
		t.Errorf("a full head has %d bloom glyphs, a slim one %d", full, slim)
	}
}
