package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Definitions. A cultivar is a named form of your own, and a form needs a
// definition or it drifts: seed of a line wanders a little every generation,
// and with nothing to hold it the "variety" ends up being whatever the last
// plant happened to be. So a cultivar carries a definition: an ideal value for
// each of the seven genes and how far either side of it a plant may stray and
// still be counted as the variety. The almanac shows the definition on the
// cultivar's page, and it is the one test of membership: a seedling or a plant
// that falls outside it is an off-type, not that cultivar, and is filed as
// something else.

// Tolerance is how far each gene may stray from the cultivar's ideal: degrees
// of the colour wheel for hue, points for the others.
type Tolerance struct {
	Hue    uint8 `json:"h"`
	Sat    uint8 `json:"s"`
	Light  uint8 `json:"l"`
	Height uint8 `json:"ht"`
	Shape  uint8 `json:"sh"`
	Speed  uint8 `json:"sp"`
	Yield  uint8 `json:"y"`
}

// defaultTolerance is the definition a cultivar starts with: a shade of colour
// and the four other genes held to a band about a sixth of their range wide.
var defaultTolerance = Tolerance{Hue: 24, Sat: 20, Light: 18, Height: 16, Shape: 16, Speed: 16, Yield: 16}

// tolStep is how much one key press widens or narrows a tolerance, and
// idealStep how far it moves an ideal.
func tolStep(gene int) int {
	if gene == 0 {
		return 4
	}
	return 2
}

func idealStep(gene int) int {
	if gene == 0 {
		return 5
	}
	return 2
}

// maxTolerance is the widest a tolerance can be set: the whole wheel, or the
// whole of a 0 to 100 gene.
func maxTolerance(gene int) int {
	if gene == 0 {
		return 180
	}
	return 100
}

// At is the tolerance on one gene.
func (t Tolerance) At(gene int) int {
	switch gene {
	case 0:
		return int(t.Hue)
	case 1:
		return int(t.Sat)
	case 2:
		return int(t.Light)
	case 3:
		return int(t.Height)
	case 4:
		return int(t.Shape)
	case 5:
		return int(t.Speed)
	}
	return int(t.Yield)
}

// With sets the tolerance on one gene, within range.
func (t Tolerance) With(gene, v int) Tolerance {
	v = max(0, min(maxTolerance(gene), v))
	switch gene {
	case 0:
		t.Hue = uint8(v)
	case 1:
		t.Sat = uint8(v)
	case 2:
		t.Light = uint8(v)
	case 3:
		t.Height = uint8(v)
	case 4:
		t.Shape = uint8(v)
	case 5:
		t.Speed = uint8(v)
	default:
		t.Yield = uint8(v)
	}
	return t
}

// Tolerances is the definition's tolerances, falling back to the default for a
// cultivar saved before definitions existed.
func (c Cultivar) Tolerances() Tolerance {
	if c.Tol == (Tolerance{}) {
		return defaultTolerance
	}
	return c.Tol
}

// colourless reports a genome so grey, white or black that its hue means nothing.
func colourless(g Genome) bool { return g.Sat < 12 }

// Strays lists the genes of a genome that fall outside the definition.
func (c Cultivar) Strays(gn Genome) []int {
	tol := c.Tolerances()
	var out []int
	for i := 0; i < geneCount; i++ {
		var off float64
		if i == 0 {
			if colourless(gn) && colourless(c.Genome) {
				continue // hue means nothing in a colourless plant
			}
			off = math.Abs(float64(hueDelta(int(c.Genome.Hue), int(gn.Hue))))
		} else {
			off = math.Abs(float64(gn.geneAt(i) - c.Genome.geneAt(i)))
		}
		if off > float64(tol.At(i)) {
			out = append(out, i)
		}
	}
	return out
}

// Admits reports whether a genome is within the definition: whether a plant
// with these genes counts as this cultivar.
func (c Cultivar) Admits(gn Genome) bool { return len(c.Strays(gn)) == 0 }

// Band is the span of values the definition allows on a gene, clipped to the
// gene's range. For hue it is the arc, which may wrap past 0.
func (c Cultivar) Band(gene int) (lo, hi int) {
	ideal, tol := c.Genome.geneAt(gene), c.Tolerances().At(gene)
	if gene == 0 {
		return ((ideal-tol)%360 + 360) % 360, (ideal + tol) % 360
	}
	return max(0, ideal-tol), min(100, ideal+tol)
}

// strayWords says which genes took a plant out of a definition, and by how
// much: "height 81 (allowed 40 to 72)".
func (c Cultivar) strayWords(gn Genome) string {
	var parts []string
	for _, i := range c.Strays(gn) {
		lo, hi := c.Band(i)
		unit := ""
		if i == 0 {
			unit = "°"
		}
		parts = append(parts, fmt.Sprintf("%s %d%s (allowed %d to %d%s)", geneNames[i], gn.geneAt(i), unit, lo, hi, unit))
	}
	return listWords(parts)
}

func listWords(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

// Fits reports how many of the seedlings a packet is likely to give would be
// within the definition, from a fixed sample of rolls (not the real seeds). The
// answer is the same every time it is asked.
func (c Cultivar) Fits(pk Packet) float64 {
	const n = 200
	if pk.Pure {
		if c.Admits(pk.A) {
			return 1
		}
		return 0
	}
	in := 0
	for i := 0; i < n; i++ {
		if c.Admits(breed(pk.ID^0x7E57, int64(i), pk.A, pk.B)) {
			in++
		}
	}
	return float64(in) / n
}

// ---- changing a definition ------------------------------------------------

// SetTolerance sets how far a gene may stray from the ideal.
func (g *Garden) SetTolerance(id, gene, v int, now time.Time) bool {
	c := g.CultivarByID(id)
	if c == nil || gene < 0 || gene >= geneCount {
		return false
	}
	c.Tol = c.Tolerances().With(gene, v)
	g.reclassify(c, now)
	return true
}

// SetIdeal moves a gene of the definition's ideal. The cultivar's genes are
// its ideal, so this is also what is drawn and what a friend would be given.
func (g *Garden) SetIdeal(id, gene, v int, now time.Time) bool {
	c := g.CultivarByID(id)
	sp := (*Species)(nil)
	if c != nil {
		sp = c.SpeciesRef()
	}
	if sp == nil || gene < 0 || gene >= geneCount {
		return false
	}
	c.Tol = c.Tolerances()
	c.Genome = c.Genome.withGene(gene, v)
	c.Variety, _ = sp.NearestVariety(c.Genome)
	g.reclassify(c, now)
	return true
}

// ResetDefinition puts a cultivar's tolerances back to the defaults.
func (g *Garden) ResetDefinition(id int, now time.Time) bool {
	c := g.CultivarByID(id)
	if c == nil {
		return false
	}
	c.Tol = Tolerance{}
	g.reclassify(c, now)
	return true
}

// reclassify applies a changed definition to what is growing and what is in
// the shed: plants and packets of the line that no longer fit it are let go,
// and flowering hybrids that now fit it join. It reports how many left.
func (g *Garden) reclassify(c *Cultivar, now time.Time) (left int) {
	for i := range g.Plots {
		p := &g.Plots[i]
		if p.Empty() || p.SpeciesID != c.Species {
			continue
		}
		switch {
		case p.Line == c.ID && !c.Admits(p.Genes()):
			p.Line, p.LineName, p.Streak = 0, "", 0
			left++
		case p.Line == 0 && p.Matured && p.IsHybrid() && c.Admits(p.Genes()):
			p.Line, p.LineName = c.ID, c.Name
		}
	}
	for i := range g.Shed {
		pk := &g.Shed[i]
		if pk.Line == c.ID && !c.Admits(pk.Mean()) {
			pk.detach(c)
		}
	}
	return left
}

// detach takes a packet out of a cultivar it no longer fits. A label the line
// gave it goes too, or sowing it would file the label as a new line.
func (pk *Packet) detach(c *Cultivar) {
	pk.Line, pk.Streak = 0, 0
	if pk.Label == c.Name {
		pk.Label = ""
	}
}

// ---- deleting ---------------------------------------------------------------

// Forgotten is a line the gardener deleted from the almanac. A plant that
// flowers like it is not written down again.
type Forgotten struct {
	Species string `json:"species"`
	Genome  Genome `json:"genome"`
}

// forgotten reports whether a genome is within reach of a deleted line.
func (g *Garden) forgotten(sp *Species, gn Genome) bool {
	for _, f := range g.Forgot {
		if f.Species == sp.ID && f.Genome.Distance(gn) <= lineMatch {
			return true
		}
	}
	return false
}

// unforget lets a genome be written down again, as when the gardener names a
// seed or takes in a friend's code for it on purpose.
func (g *Garden) unforget(sp *Species, gn Genome) {
	kept := g.Forgot[:0]
	for _, f := range g.Forgot {
		if f.Species == sp.ID && f.Genome.Distance(gn) <= lineMatch {
			continue
		}
		kept = append(kept, f)
	}
	g.Forgot = kept
}

// DeleteCultivar removes a line from the almanac: the record, and the claim it
// had on plants and seed. The plants and packets stay in the garden and the
// shed, as ordinary stock. A code that made an imported line has still been
// used, so it cannot be redeemed again.
func (g *Garden) DeleteCultivar(id int, now time.Time) (Cultivar, bool) {
	idx := -1
	for i := range g.Cultivars {
		if g.Cultivars[i].ID == id {
			idx = i
		}
	}
	if idx < 0 {
		return Cultivar{}, false
	}
	c := g.Cultivars[idx]
	for i := range g.Plots {
		p := &g.Plots[i]
		if p.Line == id {
			p.Line, p.LineName, p.Streak = 0, "", 0
		}
	}
	for i := range g.Shed {
		if g.Shed[i].Line == id {
			g.Shed[i].detach(&c)
		}
	}
	g.Cultivars = append(g.Cultivars[:idx], g.Cultivars[idx+1:]...)
	if sp := c.SpeciesRef(); sp != nil {
		g.Forgot = append(g.Forgot, Forgotten{Species: sp.ID, Genome: c.Genome})
	}
	what := "cultivar"
	if c.Gifted {
		what = "imported cultivar"
	}
	g.Log(now, "Deleted the %s ‘%s’ from the almanac.", what, c.Name)
	return c, true
}
