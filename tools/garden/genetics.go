package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Genetics. Every plant carries seven genes, all continuous and all blended:
// a child is the average of its parents plus a little noise, so a gardener who
// keeps the tallest, the bluest or the quickest can move a line in that
// direction generation by generation.
//
// Colour is three genes (hue, saturation, lightness) so a bloom can be any
// shade a terminal can show. Everything is a small integer, so a genome
// survives JSON exactly, and every roll is a hash of the garden's seed, never a
// random number, so a garden replayed after a fortnight closed grows the same
// plants as one left running.

// Genome is one plant's genes.
type Genome struct {
	Hue    uint16 `json:"h"`  // 0..359, circular
	Sat    uint8  `json:"s"`  // 0..100
	Light  uint8  `json:"l"`  // 0..100
	Height uint8  `json:"ht"` // 0 dwarf .. 100 giant, within the species' own range
	Shape  uint8  `json:"sh"` // 0 slim and upright .. 100 full and bushy
	Speed  uint8  `json:"sp"` // 0 slow .. 100 quick
	Yield  uint8  `json:"y"`  // 0 sparse .. 100 abundant
}

// Blank reports an unset genome, as found in a save from before genes existed.
func (g Genome) Blank() bool { return g == Genome{} }

func (g Genome) hsl() hsl {
	return hsl{h: float64(g.Hue), s: float64(g.Sat) / 100, l: float64(g.Light) / 100}
}

// RGB is the colour the genes express.
func (g Genome) RGB() rgb { return hslToRGB(g.hsl()) }

// Hex is the colour as #rrggbb.
func (g Genome) Hex() string { return g.RGB().hex() }

// ColourName is what a gardener would call the shade.
func (g Genome) ColourName() string { return colourName(g.RGB()) }

func genomeWithColour(g Genome, c rgb) Genome {
	v := rgbToHSL(c)
	g.Hue = uint16(math.Round(v.h)) % 360
	g.Sat = uint8(math.Round(v.s * 100))
	g.Light = uint8(math.Round(v.l * 100))
	return g
}

// The tuning for everything genes do, in one place.
const (
	speedLow, speedHigh = 0.80, 1.25 // growth multiplier at Speed 0 and 100
	heightSlowdown      = 0.20       // a giant grows this much slower than a dwarf, each way from 50
	yieldThirst         = 0.30       // a heavy cropper dries its bed this much faster
	yieldPodLow         = 0.60       // pod rate multiplier at Yield 0
	yieldPodHigh        = 1.40       // and at Yield 100
	podCapLow           = 3.0        // most pods a plant holds at Yield 0
	podCapHigh          = 7.0        // and at Yield 100
	seedSpanLong        = 1.15       // finishes its year slower or faster
	seedSpanShort       = 0.85

	noiseBase     = 3.0  // spread of a cross between identical parents
	noiseFromGap  = 0.35 // extra spread per point the parents differ
	mutationOdds  = 0.04 // per gene, per seed
	mutationLow   = 15.0
	mutationHigh  = 30.0
	hueMutateLow  = 40.0
	hueMutateHigh = 90.0
	hybridGap     = 0.12 // how far from every named variety before a plant counts as a hybrid
	stableGap     = 0.06 // parents this alike make a stable line
	stableRuns    = 3    // generations of that before a line is called stable
)

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// SpeedFactor is the growth multiplier from the Speed gene.
func (g Genome) SpeedFactor() float64 { return lerp(speedLow, speedHigh, float64(g.Speed)/100) }

// HeightFactor is the growth multiplier from being tall: giants take longer.
func (g Genome) HeightFactor() float64 {
	return 1 - heightSlowdown*(float64(g.Height)-50)/50
}

// GrowthFactor is both together. A plant whose genes sit exactly mid-range
// grows at exactly the species' advertised rate.
func (g Genome) GrowthFactor() float64 {
	return g.SpeedFactor() * g.HeightFactor() / (lerp(speedLow, speedHigh, 0.5))
}

// ThirstFactor is how much faster than average the plant dries its bed.
func (g Genome) ThirstFactor() float64 { return 1 + yieldThirst*(float64(g.Yield)-50)/50 }

// PodFactor is the multiplier on how fast pods ripen.
func (g Genome) PodFactor() float64 { return lerp(yieldPodLow, yieldPodHigh, float64(g.Yield)/100) }

// PodCap is the most ripe pods the plant will hold.
func (g Genome) PodCap() float64 { return lerp(podCapLow, podCapHigh, float64(g.Yield)/100) }

// SeedSpanFactor scales how long an annual stays in flower: quick plants finish sooner.
func (g Genome) SeedSpanFactor() float64 {
	return lerp(seedSpanLong, seedSpanShort, float64(g.Speed)/100)
}

// WindCatch is how much more or less wind a bushy or slim plant takes.
func (g Genome) WindCatch() float64 { return 0.8 + 0.4*float64(g.Shape)/100 }

// ---- height in real units -------------------------------------------------

var heightRE = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[–-]\s*(\d+(?:\.\d+)?)\s*(cm|m)\b`)
var heightOneRE = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(cm|m)\b`)

// HeightRange is a species' height from the text on its card, in centimetres.
func (s *Species) HeightRange() (lo, hi float64) {
	if m := heightRE.FindStringSubmatch(s.Height); m != nil {
		lo, _ = strconv.ParseFloat(m[1], 64)
		hi, _ = strconv.ParseFloat(m[2], 64)
		if m[3] == "m" {
			lo, hi = lo*100, hi*100
		}
		return lo, hi
	}
	if m := heightOneRE.FindStringSubmatch(s.Height); m != nil {
		v, _ := strconv.ParseFloat(m[1], 64)
		if m[2] == "m" {
			v *= 100
		}
		return v * 0.8, v * 1.25
	}
	return 10, 20
}

// HeightCM is how tall a plant with these genes grows, spread across the
// species' own range (geometrically, so a tree's range is not all giants).
func (s *Species) HeightCM(g Genome) float64 {
	lo, hi := s.HeightRange()
	if lo <= 0 {
		lo = 1
	}
	return lo * math.Pow(hi/lo, float64(g.Height)/100)
}

// HeightText is that height as a gardener would say it.
func (s *Species) HeightText(g Genome) string {
	cm := s.HeightCM(g)
	if cm >= 150 {
		return fmt.Sprintf("%.1f m", cm/100)
	}
	return fmt.Sprintf("%.0f cm", cm)
}

// ---- the colour slot ------------------------------------------------------

// slot is the part of a plant the colour genes paint.
type slot int

const (
	slotBloom slot = iota
	slotAccent
	slotLeaf
	slotStem
)

var (
	slotOnce  sync.Once
	slotTable map[string]slot
)

// colourSlot is whichever part of a species its varieties most often recolour:
// the bloom for flowers, the accent for fruit, the leaf for a lettuce.
func (s *Species) colourSlot() slot {
	slotOnce.Do(func() {
		slotTable = map[string]slot{}
		for _, sp := range AllSpecies() {
			var count [4]int
			for _, v := range sp.Varieties() {
				if v.Bloom != "" {
					count[slotBloom]++
				}
				if v.Accent != "" {
					count[slotAccent]++
				}
				if v.Leaf != "" {
					count[slotLeaf]++
				}
				if v.Stem != "" {
					count[slotStem]++
				}
			}
			best := slotBloom
			for _, c := range []slot{slotAccent, slotLeaf, slotStem} {
				if count[c] > count[best] {
					best = c
				}
			}
			slotTable[sp.ID] = best
		}
	})
	return slotTable[s.ID]
}

func slotColour(p Palette, which slot) string {
	switch which {
	case slotAccent:
		return p.Accent
	case slotLeaf:
		return p.Leaf
	case slotStem:
		return p.Stem
	}
	return p.Bloom
}

func withSlot(p Palette, which slot, colour string) Palette {
	switch which {
	case slotAccent:
		p.Accent = colour
	case slotLeaf:
		p.Leaf = colour
	case slotStem:
		p.Stem = colour
	default:
		p.Bloom = colour
	}
	return p
}

// SlotName is what the colour genes paint, for the cards.
func (s *Species) SlotName() string {
	switch s.colourSlot() {
	case slotAccent:
		return "fruit and accents"
	case slotLeaf:
		return "foliage"
	case slotStem:
		return "stems"
	}
	return "flowers"
}

// PaletteForGenome is a plant's colouring: the nearest named form for
// everything the genes do not paint, and the genes for the one part they do.
// Hydrangea keeps its real chemistry — the soil sets the hue — and the genes
// set how deep and how light it is.
func (s *Species) PaletteForGenome(variety int, g Genome, p *Plot) Palette {
	pal := s.PaletteFor(variety, p)
	// An unmodified named form is drawn exactly as its table says.
	if g.Blank() || g == s.VarietyGenome(variety) {
		if s.ID == "hydrangea" && p != nil && !p.Pond && !g.Blank() {
			c := g.hsl()
			c.h = hydrangeaHue(p.PH)
			return withSlot(pal, s.colourSlot(), hslToRGB(c).hex())
		}
		return pal
	}
	c := g.hsl()
	if s.ID == "hydrangea" && p != nil && !p.Pond {
		c.h = hydrangeaHue(p.PH)
	}
	return withSlot(pal, s.colourSlot(), hslToRGB(c).hex())
}

// hydrangeaHue is the colour wheel position soil chemistry gives the flower.
func hydrangeaHue(ph float64) float64 {
	switch {
	case ph <= 5.9:
		return 225 // aluminium taken up in acid soil: blue
	case ph >= 6.8:
		return 330 // locked away in lime: pink
	default:
		return 275 // mauve in between
	}
}

// ---- the genes of the named varieties --------------------------------------

func strHash(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64() >> 1)
}

var (
	vgOnce  sync.Once
	vgTable map[string][]Genome
)

// keywords in a variety's name or note that say what its genes are like.
var geneHints = []struct {
	trait string
	delta float64
	words []string
}{
	{"height", +34, []string{"giant", "tallest", "towering", "four metres", "tall", "lofty"}},
	{"height", -30, []string{"dwarf", "knee-high", "compact", "miniature", "low ", "short", "tumbling", "patio"}},
	{"shape", +28, []string{"double", "pompom", "shaggy", "bushy", "branching", "frilled", "ruffled", "full"}},
	{"shape", -26, []string{"single ", "slender", "slim", "airy", "spire", "narrow", "upright", "columnar"}},
	{"speed", +18, []string{"early", "quick", "fast", "rapid"}},
	{"speed", -18, []string{"late", "slow", "long-season"}},
	{"yield", +22, []string{"heavy", "prolific", "abundant", "many", "generous", "cut-and-come-again"}},
	{"yield", -20, []string{"sparse", "few"}},
}

// VarietyGenome is the genes of one named form. Colour comes from the form's
// own colour; the rest sit near mid-range, nudged by what its description
// says (a "Russian Giant" is tall) and by a stable hash so the shop is not
// uniform.
func (s *Species) VarietyGenome(i int) Genome {
	vgOnce.Do(func() { vgTable = map[string][]Genome{} })
	if cached, ok := vgTable[s.ID]; ok && i >= 0 && i < len(cached) {
		return cached[i]
	}
	vs := s.Varieties()
	out := make([]Genome, len(vs))
	for vi, v := range vs {
		pal := s.PaletteFor(vi, nil)
		colour, ok := parseColor(slotColour(pal, s.colourSlot()))
		if !ok {
			colour = rgb{0.8, 0.3, 0.4}
		}
		g := genomeWithColour(Genome{}, colour)
		seed := strHash(s.ID + "#" + v.Name)
		// Mid-range, with a small stable offset, then the hints.
		traits := map[string]float64{
			"height": 50 + 10*(2*hashUnit(seed, 1, 0x61)-1),
			"shape":  50 + 12*(2*hashUnit(seed, 2, 0x62)-1),
			"speed":  50 + 8*(2*hashUnit(seed, 3, 0x63)-1),
			"yield":  50 + 12*(2*hashUnit(seed, 4, 0x64)-1),
		}
		text := strings.ToLower(v.Name + " " + v.Note + " ")
		for _, h := range geneHints {
			for _, w := range h.words {
				if strings.Contains(text, w) {
					traits[h.trait] += h.delta
					break
				}
			}
		}
		g.Height = clampGene(traits["height"])
		g.Shape = clampGene(traits["shape"])
		g.Speed = clampGene(traits["speed"])
		g.Yield = clampGene(traits["yield"])
		out[vi] = g
	}
	vgTable[s.ID] = out
	if i < 0 || i >= len(out) {
		i = 0
	}
	return out[i]
}

func clampGene(v float64) uint8 {
	return uint8(math.Max(0, math.Min(100, math.Round(v))))
}

// Distance is how far apart two genomes are, 0 (identical) to about 1. Colour
// counts for as much as the four other genes together.
func (g Genome) Distance(o Genome) float64 {
	col := colourDistance(g.RGB(), o.RGB())
	d := func(a, b uint8) float64 { return math.Abs(float64(a)-float64(b)) / 100 }
	rest := (d(g.Height, o.Height) + d(g.Shape, o.Shape) + d(g.Speed, o.Speed) + d(g.Yield, o.Yield)) / 4
	return 0.5*math.Min(1, col) + 0.5*rest
}

// NearestVariety is the named form a genome most resembles, which supplies the
// drawings and the accent colours the genes do not paint.
func (s *Species) NearestVariety(g Genome) (idx int, gap float64) {
	best, bestD := 0, math.MaxFloat64
	for i := range s.Varieties() {
		if d := g.Distance(s.VarietyGenome(i)); d < bestD {
			best, bestD = i, d
		}
	}
	return best, bestD
}

// ---- breeding -------------------------------------------------------------

// breed makes a child of two genomes. It is a pure function of its inputs: the
// garden's seed and a serial number decide every roll.
//
// Each gene is the average of the parents' plus noise, and the noise grows
// with how unlike the parents are. Cross two different plants and the children
// spread widely; cross near-identical siblings and they come out nearly
// identical, which is how a stable line is made. A few seeds in a hundred
// mutate.
func breed(seed, serial int64, a, b Genome) Genome {
	roll := func(trait int, k int64) float64 { return hashUnit(seed, serial, 0xB8EE0000+int64(trait)*8+k) }
	// A triangular roll, which stands in for a normal one: -1 to 1, mostly near 0.
	noise := func(trait int) float64 { return roll(trait, 0) + roll(trait, 1) - 1 }
	const sdScale = 2.45 // turns the triangular roll into one with sd of 1

	mutate := func(trait int, low, high float64) float64 {
		if roll(trait, 2) >= mutationOdds {
			return 0
		}
		size := lerp(low, high, roll(trait, 3))
		if roll(trait, 4) < 0.5 {
			size = -size
		}
		return size
	}

	linear := func(trait int, x, y uint8) uint8 {
		mean := (float64(x) + float64(y)) / 2
		sd := noiseBase + noiseFromGap*math.Abs(float64(x)-float64(y))
		return clampGene(mean + noise(trait)*sd*sdScale + mutate(trait, mutationLow, mutationHigh))
	}

	var c Genome
	// Hue: averaged on the colour wheel, weighted by saturation, so crossing a
	// white with a blue gives a pale blue and not whatever hue the white
	// happened to carry.
	wa, wb := float64(a.Sat)+1, float64(b.Sat)+1
	ha, hb := float64(a.Hue), float64(b.Hue)
	x := wa*math.Cos(ha*math.Pi/180) + wb*math.Cos(hb*math.Pi/180)
	y := wa*math.Sin(ha*math.Pi/180) + wb*math.Sin(hb*math.Pi/180)
	hue := ha
	if math.Abs(x) > 1e-9 || math.Abs(y) > 1e-9 {
		hue = math.Atan2(y, x) * 180 / math.Pi
	}
	gap := hueDistance(ha, hb)
	satWeight := math.Max(0.25, (wa+wb)/200) // colourless parents say little about hue
	sdHue := (noiseBase*2 + noiseFromGap*gap) * satWeight
	hue += noise(0) * sdHue * sdScale
	hue += mutate(0, hueMutateLow, hueMutateHigh)
	hue = math.Mod(hue+720, 360)
	c.Hue = uint16(math.Round(hue)) % 360

	c.Sat = linear(1, a.Sat, b.Sat)
	l := linear(2, a.Light, b.Light)
	if l < 3 {
		l = 3
	}
	if l > 97 {
		l = 97
	}
	c.Light = l
	c.Height = linear(3, a.Height, b.Height)
	c.Shape = linear(4, a.Shape, b.Shape)
	c.Speed = linear(5, a.Speed, b.Speed)
	c.Yield = linear(6, a.Yield, b.Yield)
	return c
}

// ---- words for the cards --------------------------------------------------

func band(v uint8, words ...string) string {
	i := int(v) * len(words) / 101
	if i >= len(words) {
		i = len(words) - 1
	}
	return words[i]
}

func (g Genome) HeightWord() string {
	return band(g.Height, "dwarf", "short", "medium", "tall", "towering")
}
func (g Genome) ShapeWord() string {
	return band(g.Shape, "slim", "upright", "balanced", "full", "bushy")
}
func (g Genome) SpeedWord() string {
	return band(g.Speed, "sluggish", "slow", "steady", "quick", "swift")
}
func (g Genome) YieldWord() string {
	return band(g.Yield, "sparse", "modest", "fair", "generous", "abundant")
}

// Traits are the five things a gardener breeds for, as shown on the cards.
type Trait int

const (
	TraitColour Trait = iota
	TraitHeight
	TraitShape
	TraitSpeed
	TraitYield
	traitCount
)

func (t Trait) String() string {
	return [...]string{"colour", "height", "shape", "speed", "yield"}[t]
}

// value is a trait on a 0 to 100 scale. Colour has no scale of its own; it is
// compared by distance instead.
func (g Genome) value(t Trait) uint8 {
	switch t {
	case TraitHeight:
		return g.Height
	case TraitShape:
		return g.Shape
	case TraitSpeed:
		return g.Speed
	case TraitYield:
		return g.Yield
	}
	return g.Sat
}

// hashSerial combines the parts of a packet or plant identity into one number.
func hashSerial(parts ...int64) int64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, p := range parts {
		putInt(buf[:], p)
		_, _ = h.Write(buf[:])
	}
	return int64(h.Sum64() >> 1)
}
