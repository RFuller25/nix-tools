package main

import "fmt"

// Varieties. One species is never one plant: a gardener choosing a sunflower
// is really choosing between a four-metre single-stemmed giant and a knee-high
// shaggy one, and between lemon, gold and near-black.
//
// Each variety is a real cultivar or a genuine colour form. Most differ in
// colour alone; where the plant genuinely grows to a different shape, the
// variety carries its own drawings.

// Variety is one named form of a species.
type Variety struct {
	Name string // the cultivar, without the species name
	Note string // what is different about it

	// Colour overrides. An empty field keeps the species' own colour.
	Bloom  string
	Leaf   string
	Stem   string
	Accent string

	// Art optionally replaces the species' drawings, for the forms that
	// really do grow to another shape.
	Art *[StageCount][]string
}

// Varieties are the forms of a species, always at least two.
func (s *Species) Varieties() []Variety {
	if vs, ok := varieties[s.ID]; ok && len(vs) > 0 {
		return vs
	}
	// A species with no table entry is still itself.
	return []Variety{{Name: "the species", Note: "as it grows in the wild"}}
}

// Variety picks one form, clamped into range so an old save or a hand-edited
// one cannot ask for a plant that does not exist.
func (s *Species) Variety(i int) Variety {
	vs := s.Varieties()
	if i < 0 || i >= len(vs) {
		i = 0
	}
	return vs[i]
}

// VarietyName is how a plant of this form is written: Sunflower 'Velvet Queen'.
func (s *Species) VarietyName(i int) string {
	v := s.Variety(i)
	if len(s.Varieties()) == 1 {
		return s.Common
	}
	return fmt.Sprintf("%s ‘%s’", s.Common, v.Name)
}

// PaletteFor is the colouring of one form in one bed: the species' palette,
// adjusted for the soil where that matters, then for the variety.
func (s *Species) PaletteFor(i int, p *Plot) Palette {
	pal := s.PaletteIn(p)
	v := s.Variety(i)
	if v.Bloom != "" {
		pal.Bloom = v.Bloom
	}
	if v.Leaf != "" {
		pal.Leaf = v.Leaf
	}
	if v.Stem != "" {
		pal.Stem = v.Stem
	}
	if v.Accent != "" {
		pal.Accent = v.Accent
	}
	return pal
}

// StageFor is the drawing of one form at one stage.
func (s *Species) StageFor(variety, stage int) []string {
	if v := s.Variety(variety); v.Art != nil {
		if stage < 0 {
			stage = 0
		}
		if stage >= StageCount {
			stage = StageCount - 1
		}
		if frame := v.Art[stage]; len(frame) > 0 {
			return frame
		}
	}
	return s.Stage(stage)
}

// varietyKey identifies one form of one species in the herbarium.
func varietyKey(id string, i int) string { return fmt.Sprintf("%s#%d", id, i) }
