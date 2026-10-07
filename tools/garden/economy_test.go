package main

import (
	"os"
	"path/filepath"
	"testing"
)

// copyFixture puts the v1 garden somewhere a migration may write beside it.
func copyFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(v1Fixture())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "garden.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMigrationConvertsTheOldGardenOnce(t *testing.T) {
	path := copyFixture(t)
	now := testStart()

	g, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if g.Version != gardenVersion {
		t.Errorf("version is %d after migration", g.Version)
	}
	if g.Gold != 37 || g.Seeds != 0 {
		t.Errorf("37 old seeds became %d gold with %d left over", g.Gold, g.Seeds)
	}
	for i, p := range g.Plots {
		if !p.Empty() {
			t.Errorf("bed %d still holds %s after the wipe", i+1, p.SpeciesID)
		}
	}
	// Five plants stood in the old garden: sunflower (3 pods), tomato, waterlily,
	// lavender (1 pod). Each becomes seed of its own form.
	want := map[string]int{"sunflower": 4, "tomato": 1, "waterlily": 1, "lavender": 2}
	got := map[string]int{}
	for _, pk := range g.Shed {
		got[pk.SpeciesID] += pk.Count
		if !pk.Pure {
			t.Errorf("%s packet is not pure", pk.SpeciesID)
		}
	}
	for id, n := range want {
		if got[id] != n {
			t.Errorf("%s: %d seeds in the shed, want %d", id, got[id], n)
		}
	}
	// The ground and the records survive.
	if !g.Plots[2].Pond || g.Plots[2].Moisture != 1 {
		t.Error("the pond was lost in the migration")
	}
	if g.Plots[0].PH != 6.4 || g.Plots[0].Richness != 0.5 {
		t.Errorf("soil was altered: pH %.1f richness %.2f", g.Plots[0].PH, g.Plots[0].Richness)
	}
	if len(g.Herbarium) != 2 || len(g.Sightings) != 1 || len(g.Forms) != 2 || g.Matured != 6 || g.Planted != 9 {
		t.Errorf("records changed: %d herbarium, %d sightings, %d forms, %d matured", len(g.Herbarium), len(g.Sightings), len(g.Forms), g.Matured)
	}
	if len(g.Journal) < 2 {
		t.Error("the migration was not written in the journal")
	}
	if _, err := os.Stat(path + backupSuffix); err != nil {
		t.Errorf("no backup of the old garden: %v", err)
	}

	// Saving and loading again must change nothing: it happens exactly once.
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path, now.Add(24*3600e9))
	if err != nil {
		t.Fatal(err)
	}
	if again.Gold != g.Gold || again.SeedsInShed() != g.SeedsInShed() || len(again.Journal) != len(g.Journal) {
		t.Errorf("a second load migrated again: gold %d→%d, seeds %d→%d", g.Gold, again.Gold, g.SeedsInShed(), again.SeedsInShed())
	}
}

func TestBackupIsNeverOverwritten(t *testing.T) {
	path := copyFixture(t)
	if err := os.WriteFile(path+backupSuffix, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, testStart()); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path + backupSuffix); string(b) != "keep me" {
		t.Error("an existing backup was overwritten")
	}
}

func TestBuyingAndSowing(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 10
	sp := SpeciesByID("sunflower")

	if err := g.Buy(sp, 1, 2, now); err != nil {
		t.Fatal(err)
	}
	if g.Gold != 10-2*sp.SeedCost || g.SeedsInShed() != 2 {
		t.Errorf("gold %d, seeds %d after buying two", g.Gold, g.SeedsInShed())
	}
	if err := g.SowPacket(0, 0, now); err != nil {
		t.Fatal(err)
	}
	if g.SeedsInShed() != 1 || g.Plots[0].Variety != 1 || g.Plots[0].Genome != sp.VarietyGenome(1) {
		t.Errorf("sowing a pure packet gave %+v", g.Plots[0].Genome)
	}
	if err := g.SowPacket(0, 0, now); err == nil {
		t.Error("sowed into an occupied bed")
	}
	if err := g.SowPacket(1, 0, now); err != nil {
		t.Fatal(err)
	}
	if len(g.Shed) != 0 {
		t.Error("an emptied packet should leave the shed")
	}
}

func TestGoldCannotGoNegative(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Gold = 2
	if err := g.Buy(SpeciesByID("tomato"), 0, 5, now); err == nil {
		t.Error("bought seed without the gold")
	}
	if g.Gold != 2 || g.SeedsInShed() != 0 {
		t.Errorf("a refused purchase changed things: %d gold, %d seeds", g.Gold, g.SeedsInShed())
	}
	g.earn(-5)
	if g.Gold != 2 {
		t.Error("a negative earning changed the purse")
	}
	if g.spend(-3) || g.spend(100) {
		t.Error("spent an impossible amount")
	}
}

func TestSiblingsFromOnePacketDiffer(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("zinnia")
	a, b := sp.VarietyGenome(0), sp.VarietyGenome(1)
	g.AddPacket(Packet{SpeciesID: sp.ID, A: a, B: b, Count: 5})

	seen := map[Genome]bool{}
	for i := 0; i < 5; i++ {
		if err := g.SowPacket(i, 0, now); err != nil {
			t.Fatal(err)
		}
		seen[g.Plots[i].Genome] = true
	}
	if len(seen) < 4 {
		t.Errorf("five seeds from a mixed packet gave only %d different plants", len(seen))
	}
}

func TestSowingIsRepeatable(t *testing.T) {
	now := testStart()
	sp := SpeciesByID("zinnia")
	a, b := sp.VarietyGenome(0), sp.VarietyGenome(2)
	grow := func() []Genome {
		g := newTestGarden(now)
		g.AddPacket(Packet{SpeciesID: sp.ID, A: a, B: b, Count: 4})
		var out []Genome
		for i := 0; i < 4; i++ {
			if err := g.SowPacket(i, 0, now); err != nil {
				t.Fatal(err)
			}
			out = append(out, g.Plots[i].Genome)
		}
		return out
	}
	first, second := grow(), grow()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("seed %d grew differently on a second run", i)
		}
	}
}

func TestSellingSeedPaysByQuality(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("tomato")

	// A plain shop seed fetches less than its purchase price.
	plain := Packet{ID: 1, SpeciesID: sp.ID, A: sp.VarietyGenome(0), B: sp.VarietyGenome(0), Count: 1, Pure: true}
	if v := seedValue(plain); v < 1 || v > sp.SeedCost {
		t.Errorf("a plain seed is worth %d against a price of %d", v, sp.SeedCost)
	}
	// A seed of something new, and a stable line of it, is worth more.
	odd := sp.VarietyGenome(0)
	odd.Hue, odd.Sat, odd.Light, odd.Height, odd.Yield = (odd.Hue+150)%360, 90, 60, 95, 90
	novel := Packet{ID: 2, SpeciesID: sp.ID, A: odd, B: odd, Count: 1}
	stable := novel
	stable.Streak = stableRuns
	if seedValue(novel) <= seedValue(plain) {
		t.Errorf("a novel seed (%d) is not worth more than a plain one (%d)", seedValue(novel), seedValue(plain))
	}
	if seedValue(stable) <= seedValue(novel) {
		t.Error("a stable line is not worth more than an unsettled one")
	}

	g.Shed = []Packet{{ID: 3, SpeciesID: sp.ID, A: odd, B: odd, Count: 4}}
	before := g.Gold
	got, err := g.SellSeeds(0, 3, now)
	if err != nil || got != 3*seedValue(g.Shed[0]) || g.Gold != before+got || g.Shed[0].Count != 1 {
		t.Errorf("sold 3: got %d, err %v, gold %d→%d, %d left", got, err, before, g.Gold, g.Shed[0].Count)
	}
	if g.Sold != 3 {
		t.Errorf("lifetime sold is %d", g.Sold)
	}
	if _, err := g.SellSeeds(0, 99, now); err != nil || len(g.Shed) != 0 {
		t.Error("selling more than the packet holds should sell the rest and empty it")
	}
}

func TestPureShopSeedsMerge(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("basil")
	g.Gold = 100
	for i := 0; i < 3; i++ {
		if err := g.Buy(sp, 0, 1, now); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.Shed) != 1 || g.Shed[0].Count != 3 {
		t.Errorf("three purchases made %d packets", len(g.Shed))
	}
	if err := g.Buy(sp, 1, 1, now); err != nil || len(g.Shed) != 2 {
		t.Error("a different form should be its own packet")
	}
}

func TestGatheringCrossedSeedUsesPollen(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("cosmos")
	if err := g.Plant(0, sp, 0, now); err != nil {
		t.Fatal(err)
	}
	p := &g.Plots[0]
	p.Growth, p.Pods = 1, 3
	donor := sp.VarietyGenome(1)
	p.Pollen = &donor

	if got := g.Gather(0, now); got != 3 {
		t.Fatalf("gathered %d", got)
	}
	pk := g.Shed[0]
	if pk.A != sp.VarietyGenome(0) || pk.B != donor || pk.Pure || pk.Gen != 1 {
		t.Errorf("packet %+v is not a cross of the plant and its pollen", pk)
	}
	if g.Plots[0].Pollen != nil {
		t.Error("the pollen was not used up")
	}

	// Without pollen the plant is selfed.
	p.Pods = 2
	g.Gather(0, now)
	last := g.Shed[len(g.Shed)-1]
	if last.A != last.B {
		t.Error("an unpollinated plant should self")
	}
}
