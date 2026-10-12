package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
)

// Hybrid species. Match two flowers of different species at the gnome shop and
// the seed is neither parent's: it is a species of its own, made on the spot
// from the two. Nothing about it is stored. A hybrid is named by its parents
// (x:sunflower+foxglove, mother first) and built from them whenever it is
// needed, so the same two parents always give the same hybrid, every plant,
// packet and cultivar of it saves as an ordinary species ID, and a save from
// another machine rebuilds it exactly.
//
// A hybrid takes after its mother in how it lives (when it sleeps, whether it
// seeds itself, what soil it likes) and shares both parents' looks: the head
// of the drawing from one, the stem from the other, the two palettes mixed. It
// is not in the seed shop or the almanac's list of species; it turns up in
// your seed shed, your beds and under YOUR CULTIVARS. Hybrids breed true with
// their own kind like any species, but the gnomes will not cross a hybrid with
// anything else.

const hybridPrefix = "x:"

var (
	hybridMu    sync.Mutex
	hybridCache = map[string]*Species{}
)

// HybridID names the cross of a mother species and a father species.
func HybridID(mother, father *Species) string {
	return hybridPrefix + mother.ID + "+" + father.ID
}

// IsCross reports whether a species is a hybrid made by the gnomes.
func (s *Species) IsCross() bool { return s != nil && s.Parents[0] != nil }

// Lead is the species a hybrid takes after: its mother. Any other species is
// its own lead.
func (s *Species) Lead() *Species {
	if s.IsCross() {
		return s.Parents[0]
	}
	return s
}

// hybridOf builds (or fetches) the cross of two species. It is nil for a pair
// that cannot be crossed: a hybrid with anything but itself, or a species with
// itself.
func hybridOf(mother, father *Species) *Species {
	if mother == nil || father == nil || mother.IsCross() || father.IsCross() || mother.ID == father.ID {
		return nil
	}
	id := HybridID(mother, father)
	hybridMu.Lock()
	defer hybridMu.Unlock()
	if sp, ok := hybridCache[id]; ok {
		return sp
	}
	sp := buildHybrid(id, mother, father)
	hybridCache[id] = sp
	return sp
}

// resolveHybrid rebuilds a hybrid from its ID, or returns nil if the ID is not one.
func resolveHybrid(id string) *Species {
	rest, ok := strings.CutPrefix(id, hybridPrefix)
	if !ok {
		return nil
	}
	a, b, ok := strings.Cut(rest, "+")
	if !ok {
		return nil
	}
	return hybridOf(SpeciesByID(a), SpeciesByID(b))
}

// portmanteau runs two plant names together: Sunflower and Foxglove make Sunflove.
func portmanteau(a, b string) string {
	last := func(s string) []rune {
		f := strings.Fields(s)
		return []rune(f[len(f)-1])
	}
	ra, rb := last(a), last(b)
	head := max(3, len(ra)/2)
	tail := len(rb) / 2
	name := string(ra[:min(head, len(ra))]) + strings.ToLower(string(rb[tail:]))
	r := []rune(strings.ToLower(name))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// shortLatin abbreviates a binomial's genus: Helianthus annuus becomes H. annuus.
func shortLatin(latin string) string {
	f := strings.Fields(latin)
	if len(f) < 2 {
		return latin
	}
	return f[0][:1] + ". " + strings.Join(f[1:], " ")
}

func buildHybrid(id string, a, b *Species) *Species {
	lo := func(s *Species) (float64, float64) { return s.HeightRange() }
	alo, ahi := lo(a)
	blo, bhi := lo(b)
	rarity := a.Rarity
	if b.Rarity > rarity {
		rarity = b.Rarity
	}
	if rarity < Legendary {
		rarity++
	}
	name := portmanteau(a.Common, b.Common)
	sp := &Species{
		ID:      id,
		Common:  name,
		Latin:   shortLatin(a.Latin) + " × " + shortLatin(b.Latin),
		Family:  a.Family,
		Kind:    a.Kind,
		Rarity:  rarity,
		Origin:  "the gnome shop, from " + a.Common + " and " + b.Common,
		Bloom:   a.Bloom,
		Sun:     a.Sun,
		Water:   a.Water,
		Height:  fmt.Sprintf("%.0f–%.0f cm", (alo+blo)/2, (ahi+bhi)/2),
		Note:    fmt.Sprintf("A gnome-made cross: it keeps %s's habits and some of %s's looks.", a.Common, b.Common),
		Desc:    fmt.Sprintf("%s does not exist in the wild. It is what comes of a %s flower taking pollen from a %s, matched by the gnomes: the drawing is a patchwork of the two, the colouring a mixture, and it lives by its mother's rules while its genes are a blend of both. It is not sold in the seed shop. Cross it only with itself.", name, a.Common, b.Common),
		Seasons: unionSeasons(a.Seasons, b.Seasons),
		// Both parents' prices, so a hybrid seed is worth more than either's.
		SeedCost: a.SeedCost + b.SeedCost,
		Hours:    (a.Hours + b.Hours) / 2,
		Palette:  Palette{Stem: a.Palette.Stem, Leaf: b.Palette.Leaf, Bloom: a.Palette.Bloom, Accent: b.Palette.Accent},
		Parents:  [2]*Species{a, b},
	}
	sp.Art = blendArt(a, b)
	sp.Glyphs = map[rune]glyphClass{}
	for r, c := range b.Glyphs {
		sp.Glyphs[r] = c
	}
	for r, c := range a.Glyphs {
		sp.Glyphs[r] = c
	}
	return sp
}

// unionSeasons is every season either parent likes; a parent happy all year
// makes a hybrid happy all year.
func unionSeasons(a, b []Season) []Season {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	var out []Season
	for _, s := range allSeasons {
		for _, have := range append(append([]Season{}, a...), b...) {
			if have == s {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// blendArt draws a hybrid from its parents' drawings: the upper half of each
// stage from the mother (the head, the flower), the lower half from the
// father (the stem and leaves), each kept in its own proportions and the two
// centred on one another. Where one drawing is shorter than the other, the
// other fills the gap.
func blendArt(a, b *Species) [StageCount][]string {
	var out [StageCount][]string
	for st := 0; st < StageCount; st++ {
		fa, fb := a.Art[st], b.Art[st]
		h := max(len(fa), len(fb))
		wa, wb := frameWidth(fa), frameWidth(fb)
		w := max(wa, wb)
		row := func(f []string, fw, r int) string {
			i := r - (h - len(f)) // frames are bottom-aligned
			if i < 0 || i >= len(f) {
				return ""
			}
			return strings.Repeat(" ", (w-fw)/2) + f[i]
		}
		for r := 0; r < h; r++ {
			first, second := row(fa, wa, r), row(fb, wb, r)
			line := first
			if r >= h/2 || first == "" {
				line = second
			}
			if strings.TrimSpace(line) == "" {
				line = first + second
			}
			out[st] = append(out[st], line)
		}
	}
	return out
}

func frameWidth(f []string) int {
	w := 0
	for _, l := range f {
		w = max(w, len([]rune(l)))
	}
	return w
}
