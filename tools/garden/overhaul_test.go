package main

import (
	"strings"
	"testing"
	"time"
)

// ---- lineage ----------------------------------------------------------------

func TestBreedTraceMatchesBreed(t *testing.T) {
	a, b := SpeciesByID("cosmos").VarietyGenome(0), SpeciesByID("cosmos").VarietyGenome(1)
	for s := int64(0); s < 300; s++ {
		got, _ := breedTrace(7, s, a, b)
		if got != breed(7, s, a, b) {
			t.Fatalf("breedTrace and breed disagree at serial %d", s)
		}
	}
}

func TestMutationsAreReportedForTheGenesThatMutated(t *testing.T) {
	a := SpeciesByID("cosmos").VarietyGenome(0)
	seen := 0
	for s := int64(0); s < 2000 && seen < 5; s++ {
		child, mutated := breedTrace(3, s, a, a)
		shifts := shiftsFor(a, a, child, mutated)
		for i := 0; i < geneCount; i++ {
			if !mutated[i] {
				continue
			}
			seen++
			found := false
			for _, sh := range shifts {
				found = found || (sh.Gene == i && sh.Mutation)
			}
			if !found {
				t.Fatalf("gene %d mutated but the shifts %v do not say so", i, shifts)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no mutation in 2000 seeds")
	}
}

func TestAPlantRemembersItsWholeLine(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	if err := g.Plant(0, sp, 1, now); err != nil {
		t.Fatal(err)
	}
	if h := g.Plots[0].History; len(h) != 1 || h[0].Kind != stepShop || !strings.Contains(h[0].Note, sp.Variety(1).Name) {
		t.Fatalf("a bought plant's record is %+v", h)
	}
	// Three generations of selfing and crossing, each gathered and sown again.
	for gen := 1; gen <= 3; gen++ {
		p := &g.Plots[0]
		p.Growth, p.Matured, p.Pods = 1, true, 3
		if gen == 2 {
			donor := oddCosmos()
			p.Pollen = &donor
		}
		g.Gather(0, now)
		last := len(g.Shed) - 1
		g.Plots[0] = Plot{PH: p.PH, Richness: p.Richness}
		if err := g.SowPacket(0, last, now); err != nil {
			t.Fatal(err)
		}
	}
	h := g.Plots[0].History
	if len(h) != 4 {
		t.Fatalf("%d steps after three generations, want 4", len(h))
	}
	wantKinds := []StepKind{stepShop, stepSelf, stepCross, stepSelf}
	for i, s := range h {
		if s.Kind != wantKinds[i] || s.Gen != i {
			t.Errorf("step %d is %s gen %d, want %s gen %d", i, s.Kind, s.Gen, wantKinds[i], i)
		}
		if i > 0 && (s.Mother == nil || s.Father == nil) {
			t.Errorf("step %d has no parents", i)
		}
	}
	if h[2].Father == nil || *h[2].Father != oddCosmos() {
		t.Error("the crossing did not record the pollen parent")
	}
	if h[3].Genome != g.Plots[0].Genome {
		t.Error("the last step is not the plant's genes")
	}
	rep := reportLineage(h)
	if rep.Depth != 3 || rep.Origin.Kind != stepShop {
		t.Errorf("report: depth %d, origin %s", rep.Depth, rep.Origin.Kind)
	}
}

func TestRecordsDoNotShareMemory(t *testing.T) {
	h := []Step{{Gen: 0, Kind: stepShop}}
	a := extendLineage(h, Step{Gen: 1, Kind: stepSelf})
	b := extendLineage(h, Step{Gen: 1, Kind: stepCross})
	if a[1].Kind != stepSelf || b[1].Kind != stepCross {
		t.Fatal("two records built from one share their last step")
	}
}

func TestLongRecordsKeepTheirFirstGeneration(t *testing.T) {
	var h []Step
	for i := 0; i < lineageMax+25; i++ {
		kind := stepSelf
		if i == 0 {
			kind = stepShop
		}
		h = extendLineage(h, Step{Gen: i, Kind: kind})
	}
	if len(h) != lineageMax || h[0].Kind != stepShop || h[0].Gen != 0 || h[len(h)-1].Gen != lineageMax+24 {
		t.Fatalf("trimmed record: %d steps, first %+v, last gen %d", len(h), h[0], h[len(h)-1].Gen)
	}
}

func TestGnomeMutationAndVolunteersAreRecorded(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	if err := g.Plant(0, sp, 0, now); err != nil {
		t.Fatal(err)
	}
	before := g.Plots[0].Genes()
	after := before.withGene(3, int(before.Height)+30)
	g.mutate(0, after, "height up", now)
	h := g.Plots[0].History
	last := h[len(h)-1]
	if last.Kind != stepGnome || len(last.Shifts) != 1 || last.Shifts[0].Gene != 3 || !last.Shifts[0].Mutation {
		t.Errorf("a gnome's spell is recorded as %+v", last)
	}
}

func TestLegacyPlantsSayTheirRecordIsMissing(t *testing.T) {
	m := demoModel(t, 110, 40)
	m.g.Plots[0].History = nil
	m.g.Plots[0].Gen = 3
	m = press(t, m, "i", "t")
	if m.screen != screenLineage {
		t.Fatalf("t on a card went to screen %d", m.screen)
	}
	v := m.View()
	if !strings.Contains(v, "before lineages were recorded") {
		t.Errorf("a plant without a record is not described honestly:\n%s", v)
	}
	m = press(t, m, "esc")
	if m.screen != screenInfo {
		t.Errorf("esc went back to screen %d, want the card", m.screen)
	}
}

func TestTheGeneticsScreenShowsSourcesMutationsAndDrift(t *testing.T) {
	m := demoModel(t, 120, 60)
	sp := SpeciesByID("cosmos")
	a := sp.VarietyGenome(0)
	hi := a
	hi.Height = 95
	m.g.Plots[0].SpeciesID = "cosmos"
	m.g.Plots[0].Genome = hi
	mother := a
	m.g.Plots[0].History = []Step{
		{Gen: 0, Kind: stepShop, Note: "‘Sensation’", Genome: a},
		{Gen: 1, Kind: stepSelf, Genome: hi, Mother: &mother, Father: &mother,
			Shifts: []Shift{{Gene: 3, From: int(a.Height), To: 95, Mutation: true}, {Gene: 5, From: 40, To: 60}}},
	}
	m.g.Plots[0].Gen = 1
	m = press(t, m, "i", "t")
	v := m.View()
	for _, want := range []string{"‘Sensation’, bought from the shop", "mutation: height", "drift: speed 40 → 60", "HOW THE LINE RAN", "HOW BREEDING WORKS HERE", "GENE BY GENERATION"} {
		if !strings.Contains(v, want) {
			t.Errorf("the genetics screen lacks %q:\n%s", want, v)
		}
	}
}

func TestGeneticsOfAPacketAndACultivar(t *testing.T) {
	m := demoModel(t, 120, 50)
	m = press(t, m, "s") // the shed
	m = press(t, m, "s") // my seeds
	if m.shelf != shelfMine {
		t.Fatal("did not reach my seeds")
	}
	m = press(t, m, "t")
	if m.screen != screenLineage || !strings.Contains(m.View(), "this packet") {
		t.Errorf("a packet's genetics did not render its next generation:\n%s", m.View())
	}
	m = press(t, m, "esc")
	if m.screen != screenShop {
		t.Errorf("back went to screen %d", m.screen)
	}
	m = press(t, m, "esc", "a")
	idx := -1
	for i, r := range m.almanacVisible() {
		if r.Kind == rowCultivar {
			idx = i
		}
	}
	m.almanacCursor = idx
	m = press(t, m, "t")
	if m.screen != screenLineage || !strings.Contains(m.View(), "AGAINST THE DEFINITION") {
		t.Errorf("a cultivar's genetics lacks the definition check:\n%s", m.View())
	}
}

// ---- definitions ------------------------------------------------------------

func newLine(t *testing.T) (*Garden, *Cultivar) {
	t.Helper()
	g := newTestGarden(testStart())
	growToFlower(t, g, 0, SpeciesByID("cosmos"), oddCosmos(), 0, testStart())
	if len(g.Cultivars) != 1 {
		t.Fatalf("%d cultivars", len(g.Cultivars))
	}
	return g, &g.Cultivars[0]
}

func TestADefinitionAdmitsAndRefuses(t *testing.T) {
	_, c := newLine(t)
	if !c.Admits(oddCosmos()) {
		t.Fatal("the ideal is outside its own definition")
	}
	tall := oddCosmos()
	tall.Height = uint8(int(tall.Height) - 40)
	if c.Admits(tall) {
		t.Error("a plant 40 points shorter fits a ±16 band")
	}
	if got := c.Strays(tall); len(got) != 1 || got[0] != 3 {
		t.Errorf("strays = %v, want just height", got)
	}
	if !strings.Contains(c.strayWords(tall), "height") {
		t.Errorf("stray words %q", c.strayWords(tall))
	}
	// Hue is a circle: 350 and 10 are 20 apart.
	near := oddCosmos()
	c.Genome.Hue = 350
	near.Hue = 10
	if len(c.Strays(near)) != 0 {
		t.Error("hue 10 is within 24° of hue 350")
	}
}

func TestColourlessPlantsIgnoreHue(t *testing.T) {
	c := Cultivar{Genome: Genome{Hue: 10, Sat: 3, Light: 90, Height: 50, Shape: 50, Speed: 50, Yield: 50}}
	other := c.Genome
	other.Hue = 200
	if !c.Admits(other) {
		t.Error("two whites with different hues should match")
	}
}

func TestDriftingOutOfADefinitionMakesAnOffType(t *testing.T) {
	g, c := newLine(t)
	now := testStart().Add(48 * time.Hour)
	// A seed of the line that rolls well outside the definition.
	drifted := oddCosmos()
	drifted.Height = 20
	pk := Packet{ID: 500, SpeciesID: "cosmos", A: drifted, B: drifted, Count: 1, Pure: true, Line: c.ID, Label: c.Name}
	g.Shed = append(g.Shed, pk)
	if err := g.SowPacket(5, len(g.Shed)-1, now); err != nil {
		t.Fatal(err)
	}
	p := g.Plots[5]
	if p.Line != 0 || p.LineName != "" {
		t.Errorf("a seedling outside the definition still belongs to the line: %+v", p)
	}
	if !strings.Contains(p.Descent, "off-type") {
		t.Errorf("descent = %q", p.Descent)
	}
	// And one inside it keeps the line.
	pk2 := Packet{ID: 501, SpeciesID: "cosmos", A: oddCosmos(), B: oddCosmos(), Count: 1, Pure: true, Line: c.ID, Label: c.Name}
	g.Shed = append(g.Shed, pk2)
	if err := g.SowPacket(6, len(g.Shed)-1, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[6].Line != c.ID {
		t.Error("a true seedling left its line")
	}
}

func TestAPlantThatFlowersOutsideItsLineLeavesIt(t *testing.T) {
	g, c := newLine(t)
	now := testStart().Add(72 * time.Hour)
	g.Plots[1] = Plot{PH: g.Plots[1].PH, Richness: g.Plots[1].Richness}
	stray := oddCosmos()
	stray.Height = 20
	g.sow(1, SpeciesByID("cosmos"), stray, 1, now)
	g.Plots[1].Line, g.Plots[1].LineName = c.ID, c.Name
	at := now
	for i := 0; i < 24*4*3 && !g.Plots[1].Matured; i++ {
		at = at.Add(15 * time.Minute)
		g.Advance(at)
		g.Water(1, at)
	}
	if g.Plots[1].Line == c.ID {
		t.Error("a plant 70 points off the ideal still counts as the variety")
	}
	found := false
	for _, e := range g.Journal {
		found = found || strings.Contains(e.Text, "drifted out of")
	}
	if !found {
		t.Error("the journal does not say the plant drifted out")
	}
}

func TestEditingADefinition(t *testing.T) {
	g, c := newLine(t)
	now := testStart()
	if c.Tolerances() != defaultTolerance {
		t.Fatal("a new cultivar does not have the default definition")
	}
	// Narrowing height until the growing plant no longer fits releases it.
	g.Plots[0].Line, g.Plots[0].LineName = c.ID, c.Name
	g.Plots[0].Genome.Height = uint8(int(c.Genome.Height) + 10)
	if !c.Admits(g.Plots[0].Genome) {
		t.Fatal("setup: the plant should fit at ±16")
	}
	g.SetTolerance(c.ID, 3, 4, now)
	if c.Tolerances().Height != 4 {
		t.Errorf("height tolerance is %d", c.Tolerances().Height)
	}
	if g.Plots[0].Line != 0 {
		t.Error("narrowing the definition did not release a plant that no longer fits")
	}
	// Widening takes a flowering hybrid in.
	g.SetTolerance(c.ID, 3, 40, now)
	if g.Plots[0].Line != c.ID {
		t.Error("widening the definition did not take in a hybrid that now fits")
	}
	// Moving the ideal moves the cultivar's genes.
	g.SetIdeal(c.ID, 4, 77, now)
	if c.Genome.Shape != 77 {
		t.Errorf("ideal shape = %d", c.Genome.Shape)
	}
	g.ResetDefinition(c.ID, now)
	if c.Tolerances() != defaultTolerance {
		t.Error("reset did not restore the defaults")
	}
	// Bounds hold.
	g.SetTolerance(c.ID, 3, 5000, now)
	if c.Tolerances().Height != 100 {
		t.Errorf("tolerance ran past its range: %d", c.Tolerances().Height)
	}
}

func TestDefinitionsSurviveASave(t *testing.T) {
	g, c := newLine(t)
	c.History = []Step{{Kind: stepShop, Genome: c.Genome, Note: "‘Sensation’"}}
	g.SetTolerance(c.ID, 0, 60, testStart())
	path := t.TempDir() + "/g.json"
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if back.Cultivars[0].Tolerances().Hue != 60 || len(back.Cultivars[0].History) != 1 {
		t.Errorf("definition or history lost: %+v", back.Cultivars[0])
	}
}

func TestOldCultivarsGetTheDefaultDefinition(t *testing.T) {
	c := Cultivar{ID: 1, Genome: oddCosmos(), Species: "cosmos"}
	if c.Tolerances() != defaultTolerance {
		t.Error("a cultivar saved before definitions has no definition")
	}
}

func TestTheAlmanacShowsTheDefinition(t *testing.T) {
	m := demoModel(t, 120, 60)
	m = press(t, m, "a")
	for i, r := range m.almanacVisible() {
		if r.Kind == rowCultivar {
			m.almanacCursor = i
		}
	}
	v := m.View()
	for _, want := range []string{"DEFINITION", "hue", "height", "ideal"} {
		if want == "ideal" {
			want = "IDEAL"
		}
		if !strings.Contains(v, want) {
			t.Errorf("the cultivar page lacks %q", want)
		}
	}
	// D opens the editor; the tolerance moves; esc closes it.
	before := m.g.Cultivars[0].Tolerances().Hue
	m = press(t, m, "D", "l")
	if !m.defining || m.g.Cultivars[0].Tolerances().Hue != before+uint8(tolStep(0)) {
		t.Errorf("defining=%v hue tol=%d, want %d", m.defining, m.g.Cultivars[0].Tolerances().Hue, before+uint8(tolStep(0)))
	}
	m = press(t, m, "j", "h", "h")
	if got := m.g.Cultivars[0].Tolerances().Sat; got != defaultTolerance.Sat-2*uint8(tolStep(1)) {
		t.Errorf("saturation tolerance = %d", got)
	}
	m = press(t, m, "]")
	if m.g.Cultivars[0].Genome.Sat != 80+uint8(idealStep(1)) {
		t.Errorf("ideal saturation = %d", m.g.Cultivars[0].Genome.Sat)
	}
	m = press(t, m, "enter")
	if m.defining {
		t.Error("enter did not finish defining")
	}
}

func TestCultivarsAreVarietiesOnTheSpeciesPage(t *testing.T) {
	m := demoModel(t, 120, 70)
	m = press(t, m, "a")
	for i, r := range m.almanacVisible() {
		if r.Kind == rowPlant && r.Species.ID == "zinnia" {
			m.almanacCursor = i
		}
	}
	v := m.View()
	if !strings.Contains(v, "Zinnia hybrid No. 1") || !strings.Contains(v, "VARIETIES") {
		t.Errorf("the species page does not list the cultivar among its varieties:\n%s", v)
	}
	forms := m.g.varietyForms(SpeciesByID("zinnia"))
	named := len(SpeciesByID("zinnia").Varieties())
	m.almanacVariety = named
	if !strings.Contains(m.View(), "DEFINITION OF") {
		t.Error("flicking to the cultivar does not show its definition")
	}
	m = press(t, m, "o")
	if row, _ := m.almanacCur(); row.Kind != rowCultivar {
		t.Errorf("o on a cultivar variety opened a %v row", row.Kind)
	}
	if len(forms) != named+1 {
		t.Errorf("%d forms, want the named ones and one cultivar", len(forms))
	}
}

// ---- deleting ---------------------------------------------------------------

func TestDeletingACultivarKeepsThePlantsAndSeed(t *testing.T) {
	g, c := newLine(t)
	id := c.ID
	g.Plots[0].Line, g.Plots[0].LineName = id, c.Name
	g.AddPacket(Packet{ID: 77, SpeciesID: "cosmos", A: c.Genome, B: c.Genome, Count: 2, Pure: true, Line: id, Label: c.Name})
	g.Plots[2] = Plot{}
	gone, ok := g.DeleteCultivar(id, testStart())
	if !ok || gone.ID != id || len(g.Cultivars) != 0 {
		t.Fatalf("delete: ok=%v, %d cultivars left", ok, len(g.Cultivars))
	}
	if g.Plots[0].Line != 0 || g.Plots[0].LineName != "" || g.Plots[0].Empty() {
		t.Errorf("the plant should stay, detached: %+v", g.Plots[0])
	}
	if pk := g.Shed[len(g.Shed)-1]; pk.Line != 0 || pk.Label != "" || pk.Count != 2 {
		t.Errorf("the packet should stay, detached: %+v", pk)
	}
	if _, ok := g.DeleteCultivar(id, testStart()); ok {
		t.Error("deleted the same cultivar twice")
	}
}

func TestADeletedLineIsNotWrittenDownAgain(t *testing.T) {
	g, c := newLine(t)
	g.DeleteCultivar(c.ID, testStart())
	growToFlower(t, g, 3, SpeciesByID("cosmos"), oddCosmos(), 0, testStart().Add(48*time.Hour))
	if len(g.Cultivars) != 0 {
		t.Error("a deleted line came back when a plant like it flowered")
	}
	// Naming a seed of it on purpose does write it down.
	g.AddPacket(Packet{ID: 91, SpeciesID: "cosmos", A: oddCosmos(), B: oddCosmos(), Count: 1})
	g.RenamePacket(len(g.Shed)-1, "Back again", testStart())
	if len(g.Cultivars) != 1 {
		t.Error("naming a seed on purpose did not bring the line back")
	}
}

func TestDeletingAnImportedLineDoesNotFreeItsCode(t *testing.T) {
	maker, c := breeder(t)
	code, _, err := maker.ExportCultivar(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	f := newTestGarden(testStart())
	f.Seed = 9
	r, err := f.Redeem(code, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if !r.Cultivar.Gifted || len(r.Cultivar.History) != 1 || r.Cultivar.History[0].Kind != stepGift {
		t.Errorf("an imported line's record is %+v", r.Cultivar.History)
	}
	f.DeleteCultivar(r.Cultivar.ID, testStart())
	if _, err := f.Redeem(code, testStart()); err == nil {
		t.Error("deleting an import let its code be redeemed again")
	}
}

func TestDeleteAsksTwiceInTheAlmanac(t *testing.T) {
	m := demoModel(t, 120, 40)
	m = press(t, m, "a")
	for i, r := range m.almanacVisible() {
		if r.Kind == rowCultivar {
			m.almanacCursor = i
		}
	}
	m = press(t, m, "x")
	if len(m.g.Cultivars) != 1 || !strings.Contains(m.status, "x again") {
		t.Fatalf("one x deleted (or said nothing): %d cultivars, %q", len(m.g.Cultivars), m.status)
	}
	m = press(t, m, "j", "k", "x") // another key cancels
	if len(m.g.Cultivars) != 1 {
		t.Fatal("a key in between did not cancel the delete")
	}
	m = press(t, m, "x")
	if len(m.g.Cultivars) != 0 {
		t.Fatal("the second x did not delete")
	}
	for _, r := range m.almanacVisible() {
		if r.Kind == rowCultivar {
			t.Error("the deleted cultivar is still a row in the almanac")
		}
	}
	_ = m.View()
}

// ---- the shop ---------------------------------------------------------------

func TestACultivarCanBeBoughtAtAHigherPrice(t *testing.T) {
	g, c := newLine(t)
	sp := SpeciesByID("cosmos")
	if c.Price() < sp.seedPrice()*cultivarCost {
		t.Errorf("a cultivar costs %d, only %d for an ordinary seed", c.Price(), sp.seedPrice())
	}
	g.Gold = 1000
	if err := g.BuyCultivar(c.ID, 2, testStart()); err != nil {
		t.Fatal(err)
	}
	if g.Gold != 1000-2*c.Price() {
		t.Errorf("gold = %d, want %d", g.Gold, 1000-2*c.Price())
	}
	pk := g.Shed[len(g.Shed)-1]
	if !pk.Pure || pk.Count != 2 || pk.Line != c.ID || pk.A != c.Genome || pk.Name() != c.Name {
		t.Errorf("bought packet = %+v", pk)
	}
	if len(pk.History) == 0 || pk.History[0].Kind != stepGift && pk.History[0].Kind != stepShop {
		t.Logf("history carries the line's own record: %+v", pk.History)
	}
	g.Gold = c.Price() - 1
	if err := g.BuyCultivar(c.ID, 1, testStart()); err == nil {
		t.Error("bought a cultivar without the gold")
	}
}

func TestBuyingAndSellingACultivarIsNotAProfit(t *testing.T) {
	g, c := newLine(t)
	c.Stable = true
	best := g.cultivarPacket(c, 1)
	best.A.Yield, best.B.Yield = 100, 100
	if seedValue(best) >= c.Price() {
		t.Errorf("the shed pays %d for a seed the shop sells at %d", seedValue(best), c.Price())
	}
}

func TestSowingABoughtCultivarKeepsTheLineAndItsHistory(t *testing.T) {
	g, c := newLine(t)
	g.Gold = 1000
	if err := g.PlantCultivar(7, c.ID, testStart()); err != nil {
		t.Fatal(err)
	}
	p := g.Plots[7]
	if p.Line != c.ID || p.LineName != c.Name || p.Genome != c.Genome {
		t.Errorf("planted cultivar: %+v", p)
	}
	last := p.History[len(p.History)-1]
	if last.Kind != stepCultivar || !strings.Contains(last.Note, c.Name) {
		t.Errorf("the purchase is recorded as %+v", last)
	}
	if len(p.History) != len(c.History)+1 {
		t.Errorf("history has %d steps, the cultivar's own %d", len(p.History), len(c.History))
	}
}

func TestTheShopListsCultivarsWithTheirSpecies(t *testing.T) {
	m := demoModel(t, 130, 60)
	m = press(t, m, "s")
	if m.shelf != shelfShop {
		m = press(t, m, "s")
	}
	for i, sp := range m.shop {
		if sp.ID == "zinnia" {
			m.shopCursor = i
		}
	}
	named := len(SpeciesByID("zinnia").Varieties())
	m.shopVariety = named
	v := m.View()
	for _, want := range []string{"Zinnia hybrid No. 1", "your cultivar", "DEFINITION", "price"} {
		if !strings.Contains(v, want) {
			t.Errorf("the shop card lacks %q:\n%s", want, v)
		}
	}
	gold := m.g.Gold
	m = press(t, m, "b")
	if m.g.Gold >= gold || m.g.Shed[len(m.g.Shed)-1].Line == 0 {
		t.Errorf("b did not buy the cultivar: gold %d -> %d", gold, m.g.Gold)
	}
	// right wraps round the forms.
	m.shopVariety = named
	m = press(t, m, "l")
	if m.shopVariety != 0 {
		t.Errorf("→ from the last form went to %d", m.shopVariety)
	}
}

// ---- folders in the seed shed -------------------------------------------------

func shedWith(t *testing.T, packets ...Packet) model {
	t.Helper()
	m := demoModel(t, 110, 40)
	m.g.Shed = nil
	for _, pk := range packets {
		m.g.AddPacket(pk)
	}
	m.screen, m.shelf = screenShop, shelfMine
	return m
}

func pure(id string, variety, n int) Packet {
	sp := SpeciesByID(id)
	gn := sp.VarietyGenome(variety)
	return Packet{SpeciesID: id, A: gn, B: gn, Count: n, Pure: true, Variety: variety}
}

func titles(rows []shedRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, strings.Repeat(">", r.Depth)+r.Title)
	}
	return out
}

func TestTheShedIsAPlainListUnlessAskedForFolders(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 2), pure("cosmos", 0, 3))
	rows := m.shedRows()
	if len(rows) != 2 || rows[0].Kind != shedPacket || rows[1].Kind != shedPacket {
		t.Fatalf("plain list rows: %v", titles(rows))
	}
	m = press(t, m, "v")
	if !m.g.ShedFolders {
		t.Fatal("v did not switch to folders")
	}
	m = press(t, m, "v")
	if m.g.ShedFolders {
		t.Fatal("v did not switch back to the list")
	}
}

func TestFoldersGoCategorySpeciesVarietyPacket(t *testing.T) {
	m := shedWith(t,
		pure("basil", 0, 2), pure("basil", 1, 2), // two varieties of one herb
		pure("mint", 0, 4),   // another herb
		pure("cosmos", 0, 3), // a flower, one packet
	)
	m.g.ShedFolders = true
	rows := m.shedRows()
	// Only categories show while everything is closed, and only those with seed.
	if len(rows) != 2 || rows[0].Level != 0 || rows[1].Level != 0 {
		t.Fatalf("closed rows: %v", titles(rows))
	}
	cats := map[string]bool{rows[0].Title: true, rows[1].Title: true}
	if !cats["herb"] || !cats["flower"] {
		t.Errorf("categories = %v", cats)
	}
	if rows[0].Seeds+rows[1].Seeds != 11 {
		t.Errorf("category totals = %d and %d seeds", rows[0].Seeds, rows[1].Seeds)
	}
}

func TestAFolderWithOneThingInItIsSkipped(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 2), pure("cosmos", 0, 3), pure("cosmos", 1, 1))
	m.g.ShedFolders = true
	// Open the herb category: it holds one packet, so the packet is right there
	// with no species or variety folder first.
	herb := -1
	for i, r := range m.shedRows() {
		if r.Title == "herb" {
			herb = i
		}
	}
	m.mineCursor = herb
	m = press(t, m, "enter")
	rows := m.shedRows()
	if rows[herb+1].Kind != shedPacket || rows[herb+1].Depth != 1 {
		t.Fatalf("a single packet should sit directly under its category: %v", titles(rows))
	}
	// The flower category has two varieties of one species: the species folder
	// is skipped and the varieties are listed straight away.
	flower := -1
	for i, r := range rows {
		if r.Title == "flower" {
			flower = i
		}
	}
	m.mineCursor = flower
	m = press(t, m, "enter")
	rows = m.shedRows()
	if rows[flower+1].Kind != shedPacket || rows[flower+2].Kind != shedPacket {
		t.Fatalf("two single-packet varieties should be listed directly: %v", titles(rows))
	}
}

func TestEmptyFoldersNeverAppear(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 1))
	m.g.ShedFolders = true
	for _, r := range m.shedRows() {
		if r.Kind == shedFolder && r.Packets == 0 {
			t.Errorf("folder %q holds no seed", r.Title)
		}
	}
	if len(m.shedRows()) != 1 {
		t.Errorf("rows: %v", titles(m.shedRows()))
	}
	// Selling the last packet empties the shed, and so the folders.
	m.g.ShedOpen = map[string]bool{}
	m.g.Shed = nil
	if len(m.shedRows()) != 0 {
		t.Error("an empty shed has folders")
	}
	_ = m.View()
}

func TestFolderKeysOpenCloseAndStepOut(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 1), pure("basil", 1, 2), pure("mint", 0, 1))
	m.g.ShedFolders = true
	m.mineCursor = 0
	m = press(t, m, "l") // open the herb folder
	rows := m.shedRows()
	if len(rows) < 3 {
		t.Fatalf("opening a category showed %v", titles(rows))
	}
	m = press(t, m, "j") // onto the first child
	m = press(t, m, "h") // step out to the parent
	if m.mineCursor != 0 {
		t.Errorf("h from a child went to row %d, want the category", m.mineCursor)
	}
	m = press(t, m, "h") // close it
	if len(m.shedRows()) != 1 {
		t.Errorf("h on an open folder left %v", titles(m.shedRows()))
	}
	m = press(t, m, "C") // opens every folder
	if len(m.shedRows()) < 4 {
		t.Errorf("C opened %v", titles(m.shedRows()))
	}
	m = press(t, m, "C")
	if len(m.shedRows()) != 1 {
		t.Errorf("C again left %v", titles(m.shedRows()))
	}
}

func TestSowingFromAFolderRowDoesNotSowAFolder(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 1), pure("basil", 1, 2))
	m.g.ShedFolders = true
	m.mineCursor = 0
	before := m.g.SeedsInShed()
	m = press(t, m, "$")
	if m.g.SeedsInShed() != before {
		t.Error("selling with a folder highlighted sold seed")
	}
	m = press(t, m, "enter")
	if m.screen != screenShop || m.g.SeedsInShed() != before {
		t.Error("enter on a folder sowed or left the shed")
	}
	if !m.g.ShedOpen[m.shedRows()[0].Key] {
		t.Error("enter on a folder did not open it")
	}
	// Into the folder and sow a packet.
	m = press(t, m, "j", "j")
	if row, _ := m.shedCur(); row.Kind != shedPacket {
		t.Fatalf("expected a packet, on %v", row)
	}
	m = press(t, m, "enter")
	if m.screen != screenGarden || m.g.SeedsInShed() != before-1 {
		t.Errorf("sowing from a folder: screen %d, seeds %d", m.screen, m.g.SeedsInShed())
	}
}

func TestSwitchingModeKeepsThePacketSelected(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 1), pure("cosmos", 2, 1), pure("mint", 0, 1))
	m.mineCursor = 2
	pi := m.minePacket()
	m = press(t, m, "v")
	if got := m.minePacket(); got != pi {
		t.Errorf("after v the cursor is on packet %d, want %d", got, pi)
	}
	m = press(t, m, "v")
	if got := m.minePacket(); got != pi {
		t.Errorf("after v again the cursor is on packet %d, want %d", got, pi)
	}
}

func TestFolderStateSurvivesASave(t *testing.T) {
	m := shedWith(t, pure("basil", 0, 1), pure("mint", 0, 1))
	m.g.ShedFolders = true
	m.mineCursor = 0
	m = press(t, m, "enter")
	path := t.TempDir() + "/g.json"
	if err := Save(path, m.g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if !back.ShedFolders || len(back.ShedOpen) != 1 {
		t.Errorf("folders=%v open=%v", back.ShedFolders, back.ShedOpen)
	}
}

func TestFolderScreensRender(t *testing.T) {
	for _, size := range [][2]int{{110, 40}, {60, 20}, {40, 12}} {
		m := shedWith(t, pure("basil", 0, 1), pure("basil", 1, 2), pure("cosmos", 0, 1), pure("cosmos", 1, 4), pure("mint", 0, 1))
		m.width, m.height = size[0], size[1]
		m.g.ShedFolders = true
		m = press(t, m, "C")
		for i := 0; i < len(m.shedRows()); i++ {
			if v := m.View(); strings.TrimSpace(v) == "" {
				t.Fatal("blank shed")
			}
			m = press(t, m, "j")
		}
	}
}

func TestCultivarsOfGnomeCrossesAreInTheShop(t *testing.T) {
	m := demoModel(t, 130, 50)
	cross := hybridOf(SpeciesByID("sunflower"), SpeciesByID("foxglove"))
	gn := cross.VarietyGenome(0)
	gn.Hue, gn.Height = 200, 90
	m.g.Cultivars = append(m.g.Cultivars, Cultivar{ID: 9, Name: "Blue Spire", Species: cross.ID, Genome: gn, Found: testStart(), Grown: testStart()})
	m.g.CultivarSeq = 9
	m.refreshShop()
	found := false
	for i, sp := range m.shop {
		if sp.ID == cross.ID {
			m.shopCursor, found = i, true
		}
	}
	if !found {
		t.Fatal("a gnome cross with a cultivar is not in the shop")
	}
	m.screen, m.shelf = screenShop, shelfShop
	m.shopVariety = len(cross.Varieties())
	if v := m.View(); !strings.Contains(v, "Blue Spire") {
		t.Errorf("the cross's cultivar is not on its card:\n%s", v)
	}
	m = press(t, m, "b")
	if last := m.g.Shed[len(m.g.Shed)-1]; last.SpeciesID != cross.ID || last.Line != 9 {
		t.Errorf("bought %+v", last)
	}
}
