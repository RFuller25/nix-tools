package main

import (
	"math"
	"testing"
)

func TestBreedingIsDeterministic(t *testing.T) {
	a := SpeciesByID("sunflower").VarietyGenome(0)
	b := SpeciesByID("sunflower").VarietyGenome(1)
	for serial := int64(0); serial < 50; serial++ {
		if breed(7, serial, a, b) != breed(7, serial, a, b) {
			t.Fatalf("serial %d bred two different children from the same parents", serial)
		}
	}
	if breed(7, 1, a, b) == breed(8, 1, a, b) && breed(7, 2, a, b) == breed(8, 2, a, b) {
		t.Error("the garden's seed has no effect on breeding")
	}
}

func TestChildrenBlendTheirParents(t *testing.T) {
	a := Genome{Hue: 10, Sat: 80, Light: 50, Height: 20, Shape: 20, Speed: 20, Yield: 20}
	b := Genome{Hue: 10, Sat: 80, Light: 50, Height: 80, Shape: 80, Speed: 80, Yield: 80}
	var sum [4]float64
	const n = 600
	for i := 0; i < n; i++ {
		c := breed(3, int64(i), a, b)
		sum[0] += float64(c.Height)
		sum[1] += float64(c.Shape)
		sum[2] += float64(c.Speed)
		sum[3] += float64(c.Yield)
	}
	for i, s := range sum {
		if mean := s / n; math.Abs(mean-50) > 3 {
			t.Errorf("trait %d averaged %.1f over %d children, want about 50", i, mean, n)
		}
	}
}

func TestUnlikeParentsSpreadMoreThanLikeOnes(t *testing.T) {
	spread := func(a, b Genome) float64 {
		var vals []float64
		for i := 0; i < 400; i++ {
			vals = append(vals, float64(breed(5, int64(i), a, b).Height))
		}
		var mean float64
		for _, v := range vals {
			mean += v
		}
		mean /= float64(len(vals))
		var ss float64
		for _, v := range vals {
			ss += (v - mean) * (v - mean)
		}
		return math.Sqrt(ss / float64(len(vals)))
	}
	same := Genome{Hue: 100, Sat: 50, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	tall, short := same, same
	tall.Height, short.Height = 90, 10
	if like, unlike := spread(same, same), spread(tall, short); unlike < like*3 {
		t.Errorf("unlike parents spread %.1f, like parents %.1f: crossing should be far wider", unlike, like)
	}
	if like := spread(same, same); like > 8 {
		t.Errorf("a cross of identical parents spread %.1f; a stable line needs to be steady", like)
	}
}

func TestSelectionMovesALine(t *testing.T) {
	// Keep the tallest of 12 children each generation and breed from them.
	parent := Genome{Hue: 200, Sat: 60, Light: 50, Height: 40, Shape: 50, Speed: 50, Yield: 50}
	first := parent.Height
	for gen := 0; gen < 12; gen++ {
		best := parent
		for i := 0; i < 12; i++ {
			c := breed(11, hashSerial(int64(gen), int64(i)), parent, parent)
			if c.Height > best.Height {
				best = c
			}
		}
		parent = best
	}
	if parent.Height < first+12 {
		t.Errorf("twelve generations of keeping the tallest moved height from %d to only %d", first, parent.Height)
	}
}

func TestWhiteBlendsToTheOtherParentsHue(t *testing.T) {
	white := Genome{Hue: 0, Sat: 0, Light: 97, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	blue := Genome{Hue: 235, Sat: 80, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	var off int
	for i := 0; i < 200; i++ {
		c := breed(2, int64(i), white, blue)
		if hueDistance(float64(c.Hue), 235) > 40 && c.Sat > 20 {
			off++
		}
	}
	if off > 20 {
		t.Errorf("%d of 200 white x blue children were some other colour", off)
	}
}

func TestRedAndBlueMakeMagentaNotGreen(t *testing.T) {
	red := Genome{Hue: 0, Sat: 90, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	blue := red
	blue.Hue = 240
	green := 0
	for i := 0; i < 400; i++ {
		c := breed(2, int64(i), red, blue)
		if hueDistance(float64(c.Hue), 300) > 100 {
			green++
		}
	}
	if green > 400/20 {
		t.Errorf("%d of 400 red x blue children landed on the far side of the wheel", green)
	}
}

func TestMutationIsRareButHappens(t *testing.T) {
	g := Genome{Hue: 100, Sat: 50, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	big := 0
	const n = 3000
	for i := 0; i < n; i++ {
		c := breed(9, int64(i), g, g)
		if math.Abs(float64(c.Height)-50) >= 12 {
			big++
		}
	}
	// Per gene odds are 4%; a jump of 15 or more, plus the odd large roll.
	if big < n/100 || big > n/10 {
		t.Errorf("%d of %d seeds mutated in height, want between 1%% and 10%%", big, n)
	}
}

func TestGenesStayInRange(t *testing.T) {
	a := Genome{Hue: 359, Sat: 100, Light: 97, Height: 100, Shape: 0, Speed: 100, Yield: 0}
	b := Genome{Hue: 1, Sat: 0, Light: 3, Height: 0, Shape: 100, Speed: 0, Yield: 100}
	for i := 0; i < 2000; i++ {
		c := breed(1, int64(i), a, b)
		if c.Hue >= 360 || c.Sat > 100 || c.Light > 97 || c.Light < 3 || c.Height > 100 || c.Shape > 100 || c.Speed > 100 || c.Yield > 100 {
			t.Fatalf("a child left the legal range: %+v", c)
		}
	}
}

func TestNeutralGenomeGrowsAtTheAdvertisedRate(t *testing.T) {
	g := Genome{Height: 50, Speed: 50, Yield: 50}
	if f := g.GrowthFactor(); math.Abs(f-1) > 1e-9 {
		t.Errorf("a mid-range plant grows at %.3f of the species rate, want exactly 1", f)
	}
	fast, slow := Genome{Speed: 100, Height: 50}, Genome{Speed: 0, Height: 50}
	if fast.GrowthFactor() <= slow.GrowthFactor() {
		t.Error("the speed gene does not speed anything up")
	}
	tall, short := Genome{Speed: 50, Height: 100}, Genome{Speed: 50, Height: 0}
	if tall.GrowthFactor() >= short.GrowthFactor() {
		t.Error("giants should grow slower than dwarfs")
	}
	if (Genome{Yield: 100}).ThirstFactor() <= (Genome{Yield: 0}).ThirstFactor() {
		t.Error("a heavy cropper should be thirstier")
	}
}

func TestEverySpeciesHasAReadableHeight(t *testing.T) {
	for _, sp := range AllSpecies() {
		lo, hi := sp.HeightRange()
		if lo <= 0 || hi < lo {
			t.Errorf("%s: height %q parsed as %v-%v", sp.ID, sp.Height, lo, hi)
		}
		short := sp.HeightCM(Genome{Height: 0})
		giant := sp.HeightCM(Genome{Height: 100})
		if giant < short {
			t.Errorf("%s: a giant (%.0f) is shorter than a dwarf (%.0f)", sp.ID, giant, short)
		}
	}
}

func TestVarietyGenomesCarryTheirColours(t *testing.T) {
	for _, sp := range AllSpecies() {
		for i, v := range sp.Varieties() {
			g := sp.VarietyGenome(i)
			if g.Blank() {
				t.Errorf("%s ‘%s’ has no genome", sp.ID, v.Name)
			}
			pal := sp.PaletteForGenome(i, g, nil)
			want, _ := parseColor(slotColour(sp.PaletteFor(i, nil), sp.colourSlot()))
			got, ok := parseColor(slotColour(pal, sp.colourSlot()))
			if !ok || colourDistance(got, want) > 0.03 {
				t.Errorf("%s ‘%s’: genes paint %s, the form is %s", sp.ID, v.Name, g.Hex(), rgb(want).hex())
			}
			if idx, gap := sp.NearestVariety(g); idx != i || gap > 1e-9 {
				t.Errorf("%s ‘%s’ is nearest to form %d (gap %.3f)", sp.ID, v.Name, idx, gap)
			}
		}
	}
}

func TestGiantsAndDwarfsAreSpottedFromTheirNotes(t *testing.T) {
	sf := SpeciesByID("sunflower")
	giant, dwarf := sf.VarietyGenome(0), sf.VarietyGenome(3) // Russian Giant, Teddy Bear
	if giant.Height < 75 || dwarf.Height > 35 {
		t.Errorf("Russian Giant is height %d and Teddy Bear %d", giant.Height, dwarf.Height)
	}
	if sf.HeightCM(giant) < sf.HeightCM(dwarf)*1.4 {
		t.Error("a giant sunflower should be at least 40% taller than a dwarf")
	}
}

func TestEverySpeciesResolvesAColourSlot(t *testing.T) {
	for _, sp := range AllSpecies() {
		pal := sp.PaletteFor(0, nil)
		if _, ok := parseColor(slotColour(pal, sp.colourSlot())); !ok {
			t.Errorf("%s: its colour slot (%s) has no usable colour %q", sp.ID, sp.SlotName(), slotColour(pal, sp.colourSlot()))
		}
	}
	if SpeciesByID("tomato").colourSlot() != slotAccent {
		t.Error("a tomato's colour is its fruit")
	}
}

func TestHydrangeaKeepsItsSoilChemistry(t *testing.T) {
	sp := SpeciesByID("hydrangea")
	g := sp.VarietyGenome(0)
	acid, lime := &Plot{PH: 5.2}, &Plot{PH: 7.4}
	a := sp.PaletteForGenome(0, g, acid)
	l := sp.PaletteForGenome(0, g, lime)
	ca, _ := parseColor(a.Bloom)
	cl, _ := parseColor(l.Bloom)
	if colourName(ca) == colourName(cl) {
		t.Errorf("hydrangea is %s in both acid and lime soil", colourName(ca))
	}
}

// Every form on the shop shelf is ready within about a day of perfect care,
// whatever its genes; the schedule moves a little with the form, not wildly.
func TestEveryNamedFormMaturesWithinReason(t *testing.T) {
	for _, sp := range AllSpecies() {
		for vi := range sp.Varieties() {
			f := sp.VarietyGenome(vi).GrowthFactor()
			if hours := sp.Hours / f; hours > 30 || f < 0.7 || f > 1.4 {
				t.Errorf("%s form %d: growth factor %.2f makes %.1fh of a %.0fh plant", sp.ID, vi, f, hours, sp.Hours)
			}
		}
	}
}

func TestAnUnmodifiedFormKeepsItsDesignedPalette(t *testing.T) {
	for _, sp := range AllSpecies() {
		if sp.ID == "hydrangea" {
			continue
		}
		for vi := range sp.Varieties() {
			if got, want := sp.PaletteForGenome(vi, sp.VarietyGenome(vi), nil), sp.PaletteFor(vi, nil); got != want {
				t.Errorf("%s form %d was repainted: %+v vs %+v", sp.ID, vi, got, want)
			}
		}
	}
}
