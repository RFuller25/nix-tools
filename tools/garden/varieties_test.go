package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestEverySpeciesHasVarieties(t *testing.T) {
	for _, sp := range AllSpecies() {
		vs := sp.Varieties()
		if len(vs) < 2 || len(vs) > 5 {
			t.Errorf("%s has %d varieties, want between 2 and 5", sp.ID, len(vs))
		}

		names := map[string]bool{}
		for i, v := range vs {
			if strings.TrimSpace(v.Name) == "" {
				t.Errorf("%s variety %d has no name", sp.ID, i)
			}
			if strings.TrimSpace(v.Note) == "" {
				t.Errorf("%s ‘%s’ says nothing about what is different", sp.ID, v.Name)
			}
			if names[v.Name] {
				t.Errorf("%s lists ‘%s’ twice", sp.ID, v.Name)
			}
			names[v.Name] = true

			// A variety that changes nothing at all is not a variety.
			if v.Bloom == "" && v.Leaf == "" && v.Stem == "" && v.Accent == "" && v.Art == nil {
				t.Errorf("%s ‘%s’ looks exactly like the species", sp.ID, v.Name)
			}
		}
	}
}

// A variety's own drawings have to obey the same rules as a species': the
// same width on every line, inside a bed, and never shrinking as it grows.
func TestVarietyArtIsWellFormed(t *testing.T) {
	overrides := 0
	for _, sp := range AllSpecies() {
		for vi, v := range sp.Varieties() {
			if v.Art == nil {
				continue
			}
			overrides++

			prevLines := 0
			for stage := 0; stage < StageCount; stage++ {
				frame := sp.StageFor(vi, stage)
				if len(frame) == 0 {
					t.Errorf("%s ‘%s’ has no drawing at %s", sp.ID, v.Name, stageNames[stage])
					continue
				}
				want := lipgloss.Width(frame[0])
				for _, line := range frame {
					if got := lipgloss.Width(line); got != want {
						t.Errorf("%s ‘%s’ %s: line %q is %d wide, first line is %d",
							sp.ID, v.Name, stageNames[stage], line, got, want)
					}
					if lipgloss.Width(line) > cellInner {
						t.Errorf("%s ‘%s’ %s is wider than a bed: %q", sp.ID, v.Name, stageNames[stage], line)
					}
				}
				if len(frame) > artHeight {
					t.Errorf("%s ‘%s’ %s is %d lines, taller than a bed", sp.ID, v.Name, stageNames[stage], len(frame))
				}
				if len(frame) < prevLines {
					t.Errorf("%s ‘%s’ shrinks at %s", sp.ID, v.Name, stageNames[stage])
				}
				prevLines = len(frame)
			}
		}
	}
	if overrides == 0 {
		t.Error("no variety brings drawings of its own")
	}
}

func TestVarietyColoursApply(t *testing.T) {
	sun := SpeciesByID("sunflower")
	plain := sun.PaletteFor(0, nil)
	dark := sun.PaletteFor(1, nil) // Velvet Queen
	if plain.Bloom == dark.Bloom {
		t.Error("two varieties of sunflower flower the same colour")
	}
	// Whatever a variety leaves alone stays as the species has it.
	if dark.Stem != sun.Palette.Stem {
		t.Error("a variety changed a colour it did not set")
	}
	// Out-of-range indexes fall back rather than crashing.
	if sun.PaletteFor(99, nil) != plain {
		t.Error("an impossible variety did not fall back to the first")
	}
	if got := sun.Variety(-1).Name; got != sun.Varieties()[0].Name {
		t.Errorf("a negative variety gave %q", got)
	}
}

func TestVarietyNameReads(t *testing.T) {
	sun := SpeciesByID("sunflower")
	if got := sun.VarietyName(1); !strings.Contains(got, "Sunflower") || !strings.Contains(got, "Velvet Queen") {
		t.Errorf("VarietyName = %q", got)
	}
}

func TestPlantingKeepsTheVariety(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("tulip")

	if err := g.Plant(0, sp, 2, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[0].Variety != 2 {
		t.Errorf("planted variety %d, want 2", g.Plots[0].Variety)
	}
	if got := g.Plots[0].FullName(); !strings.Contains(got, sp.Varieties()[2].Name) {
		t.Errorf("the plant reads as %q", got)
	}

	// An impossible variety is corrected rather than stored.
	if err := g.Plant(1, sp, 99, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[1].Variety != 0 {
		t.Errorf("an out-of-range variety was stored as %d", g.Plots[1].Variety)
	}
}

func TestVolunteersComeTrueToTheirParent(t *testing.T) {
	sp := SpeciesByID("poppy")
	now := inSeason(sp)
	g := newTestGarden(now)
	if err := g.Plant(6, sp, 2, now); err != nil {
		t.Fatal(err)
	}

	growTo(g, now, 24*6)
	if g.Volunteers == 0 {
		t.Skip("no volunteer appeared in this run")
	}
	for i, p := range g.Plots {
		if i == 6 || p.Empty() {
			continue
		}
		if p.Variety != 2 {
			t.Errorf("a volunteer in bed %d came up as variety %d, not its parent's", i+1, p.Variety)
		}
	}
}

func TestHerbariumRecordsTheForm(t *testing.T) {
	now := inSeason(SpeciesByID("radish"))
	g := newTestGarden(now)
	sp := SpeciesByID("radish")

	if g.GrownForm(sp, 1) {
		t.Fatal("a new garden has already grown a variety")
	}
	if err := g.Plant(0, sp, 1, now); err != nil {
		t.Fatal(err)
	}

	at := now
	for i := 0; i < 24*3 && !g.Plots[0].Matured; i++ {
		at = at.Add(time.Hour)
		g.Advance(at)
		g.Water(0, at)
	}

	if !g.GrownForm(sp, 1) {
		t.Error("flowering did not record the variety")
	}
	if g.GrownForm(sp, 0) {
		t.Error("a variety that was never grown is ticked")
	}
	grown, total := g.FormsGrown()
	if grown != 1 || total < 2*len(AllSpecies()) {
		t.Errorf("forms grown = %d of %d", grown, total)
	}
}

func TestVarietiesSurviveASave(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("dahlia"), 1, now); err != nil {
		t.Fatal(err)
	}
	g.collect(SpeciesByID("dahlia"), 1, now)

	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if back.Plots[0].Variety != 1 {
		t.Errorf("the variety came back as %d", back.Plots[0].Variety)
	}
	if !back.GrownForm(SpeciesByID("dahlia"), 1) {
		t.Error("the grown form was forgotten")
	}
}

// Every variety of every species has to draw inside a bed, at every stage.
func TestEveryVarietyDraws(t *testing.T) {
	for _, sp := range AllSpecies() {
		for vi := range sp.Varieties() {
			for stage := 0; stage < StageCount; stage++ {
				lines := renderVariety(sp, vi, stage, sp.PaletteFor(vi, nil), cellInner, artHeight, 0, nil)
				if len(lines) != artHeight {
					t.Fatalf("%s ‘%s’ drew %d lines", sp.ID, sp.Variety(vi).Name, len(lines))
				}
				for _, line := range lines {
					if w := lipgloss.Width(line); w != cellInner {
						t.Fatalf("%s ‘%s’ drew a %d-wide line", sp.ID, sp.Variety(vi).Name, w)
					}
				}
			}
		}
	}
}
