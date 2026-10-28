package main

import (
	"strings"
	"testing"
	"time"
)

// villageGarden is a garden with money and a flowering foxglove in bed 0 and
// another in bed 1.
func villageGarden(t *testing.T) (*Garden, time.Time) {
	t.Helper()
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC) // a Monday
	g := newTestGarden(now)
	g.Gold = 2000
	sp := SpeciesByID("foxglove")
	for _, bed := range []int{0, 1} {
		g.sow(bed, sp, sp.VarietyGenome(0), 0, now)
		g.Plots[bed].Growth = 1
	}
	return g, now
}

func TestHelpersDoTheChoresAndAreOnlyPaidWeekly(t *testing.T) {
	g, now := villageGarden(t)
	for _, id := range []string{"waterer", "weeder"} {
		if err := g.HireHelper(id, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.HireHelper("waterer", now); err == nil {
		t.Error("hired the same person twice")
	}
	paid := g.Gold
	g.Plots[0].Moisture, g.Plots[0].Weeds = 0.1, 0.9
	g.helpersAct(now)
	if g.Plots[0].Moisture < 0.8 || g.Plots[0].Weeds != 0 {
		t.Errorf("helpers did not water or weed: %+v", g.Plots[0])
	}
	if g.Gold != paid {
		t.Error("the helpers' work cost gold mid-week")
	}
	// Next Monday they are paid again.
	next := now.Add(7 * 24 * time.Hour)
	g.settleWages(next)
	want := paid - helperDefs[0].Wage - helperDefs[1].Wage
	if g.Gold != want {
		t.Errorf("gold %d after the weekly wage, want %d", g.Gold, want)
	}
	if !g.HelperWorking("waterer", next) {
		t.Error("a paid helper is not working")
	}
}

func TestHelpersWhoCannotBePaidStayHome(t *testing.T) {
	g, now := villageGarden(t)
	g.Gold = helperDefs[0].Hire
	if err := g.HireHelper("waterer", now); err != nil {
		t.Fatal(err)
	}
	next := now.Add(7 * 24 * time.Hour)
	g.settleWages(next)
	if g.HelperWorking("waterer", next) {
		t.Error("an unpaid helper is still working")
	}
	g.Plots[0].Moisture = 0.1
	g.helpersAct(next)
	if g.Plots[0].Moisture != 0.1 {
		t.Error("an unpaid helper watered the bed")
	}
	if g.Gold != 0 {
		t.Errorf("gold %d: a wage that could not be paid took something", g.Gold)
	}
	g.Gold = 100
	g.settleWages(next.Add(7 * 24 * time.Hour))
	if !g.HelperWorking("waterer", next.Add(7*24*time.Hour)) {
		t.Error("the helper did not come back when there was gold")
	}
}

func TestWagesAreCharged_WhileAwayToo(t *testing.T) {
	g, now := villageGarden(t)
	if err := g.HireHelper("weeder", now); err != nil {
		t.Fatal(err)
	}
	g.LastTick = now
	before := g.Gold
	g.Advance(now.Add(15 * 24 * time.Hour)) // away for two Mondays and a bit
	if got := before - g.Gold; got != 2*helperDefs[1].Wage {
		t.Errorf("paid %d in wages over two weeks away, want %d", got, 2*helperDefs[1].Wage)
	}
}

func TestSceneryIsBoughtOnceAndChangesTheGarden(t *testing.T) {
	g, now := villageGarden(t)
	if g.boost(boostWater) != 1 {
		t.Fatal("scenery boosts a bare garden")
	}
	gold := g.Gold
	if err := g.BuyScenery("waterbutt", now); err != nil {
		t.Fatal(err)
	}
	if g.Gold != gold-sceneryDefByID("waterbutt").Price || g.boost(boostWater) >= 1 {
		t.Error("a water butt neither cost gold nor slowed the drying")
	}
	if err := g.BuyScenery("waterbutt", now); err == nil {
		t.Error("bought a second water butt")
	}
	g.Gold = 0
	if err := g.BuyScenery("fountain", now); err == nil {
		t.Error("bought a fountain with no gold")
	}
	if !strings.Contains(g.SceneryStrip(), sceneryDefByID("waterbutt").Glyph) {
		t.Error("the water butt is not on the strip")
	}
}

func TestScenerySlowsTheDrying(t *testing.T) {
	a, now := villageGarden(t)
	b, _ := villageGarden(t)
	if err := b.BuyScenery("waterbutt", now); err != nil {
		t.Fatal(err)
	}
	for _, g := range []*Garden{a, b} {
		g.GnomeSpells = map[int]WeatherKind{dayKey(now): Heatwave}
		g.Plots[0].Moisture = 1
		g.stepAll(now, 4, now)
	}
	if b.Plots[0].Moisture <= a.Plots[0].Moisture {
		t.Errorf("moisture with a butt %.2f, without %.2f", b.Plots[0].Moisture, a.Plots[0].Moisture)
	}
}

func TestGeneLabForecastCopyAndPurify(t *testing.T) {
	g, now := villageGarden(t)
	g.Plots[1].Genome.Hue = (g.Plots[1].Genome.Hue + 40) % 360
	gold := g.Gold
	kids, err := g.Forecast(0, 1, now)
	if err != nil || len(kids) != labForecastKids || g.Gold != gold-labForecastCost {
		t.Fatalf("forecast: %d kids, %v, gold %d→%d", len(kids), err, gold, g.Gold)
	}
	if _, err := g.Forecast(0, 0, now); err == nil {
		t.Error("forecast a plant crossed with itself")
	}
	g.sow(2, SpeciesByID("lavender"), SpeciesByID("lavender").VarietyGenome(0), 0, now)
	g.Plots[2].Growth = 1
	if _, err := g.Forecast(0, 2, now); err == nil {
		t.Error("forecast a cross of two species")
	}

	shed := len(g.Shed)
	cost := g.CloneCost(0)
	gold = g.Gold
	if err := g.Clone(0, now); err != nil || g.Gold != gold-cost || cost < labCloneBase {
		t.Fatalf("clone: %v, cost %d, gold %d→%d", err, cost, gold, g.Gold)
	}
	if len(g.Shed) != shed+1 || !g.Shed[shed].Pure || g.Shed[shed].Count != labCloneSeeds || g.Shed[shed].A != g.Plots[0].Genes() {
		t.Errorf("the copy is not an exact, pure packet: %+v", g.Shed[len(g.Shed)-1])
	}
	if g.Plots[0].Empty() {
		t.Error("copying a plant took it away")
	}
	// A plant that is unlike anything on the shelf costs more to copy.
	odd := g.Plots[1].Genome
	odd.Hue, odd.Sat, odd.Height = (odd.Hue+150)%360, 95, 95
	g.Plots[1].Genome = odd
	if g.CloneCost(1) <= g.CloneCost(0) {
		t.Errorf("an odd plant costs %d to copy, a plain one %d", g.CloneCost(1), g.CloneCost(0))
	}

	// A mixed packet becomes pure on its middle gene.
	sp := SpeciesByID("foxglove")
	a, b := sp.VarietyGenome(0), sp.VarietyGenome(1)
	g.AddPacket(Packet{SpeciesID: sp.ID, A: a, B: b, Count: 3, From: "a cross"})
	idx := len(g.Shed) - 1
	mean := g.Shed[idx].Mean()
	if err := g.Purify(idx, now); err != nil {
		t.Fatal(err)
	}
	if pk := g.Shed[idx]; !pk.Pure || pk.A != mean || pk.B != mean {
		t.Errorf("purified packet: %+v", pk)
	}
	if err := g.Purify(idx, now); err == nil {
		t.Error("purified a packet twice")
	}
}

func TestACommissionUsesThePlantAndKeepsASouvenir(t *testing.T) {
	g, now := villageGarden(t)
	g.Plots[0].Richness = 0.37
	cost := g.CommissionCost("frame", 0)
	if cost <= commissionDefByID("frame").Base {
		t.Errorf("a frame costs only %d", cost)
	}
	gold := g.Gold
	if err := g.Commission("frame", 0, now); err != nil {
		t.Fatal(err)
	}
	if g.Gold != gold-cost || len(g.Keepsakes) != 1 || !g.Plots[0].Empty() || g.Plots[0].Richness != 0.37 {
		t.Errorf("commission: gold %d→%d, %d keepsakes, bed %q richness %.2f", gold, g.Gold, len(g.Keepsakes), g.Plots[0].SpeciesID, g.Plots[0].Richness)
	}
	if err := g.Commission("frame", 0, now); err == nil {
		t.Error("commissioned an empty bed")
	}
	g.Plots[1].Growth = 0.5
	if err := g.Commission("wreath", 1, now); err == nil {
		t.Error("commissioned a plant not yet in bloom")
	}
}

func TestThemesAndBordersAreBoughtOnceAndKept(t *testing.T) {
	defer applyTheme("meadow")
	g, now := villageGarden(t)
	if !g.OwnsTheme("meadow") || !g.OwnsBorder("rounded") || g.OwnsTheme("dusk") || g.OwnsBorder("double") {
		t.Fatal("the free ones are owned and the rest are not")
	}
	gold := g.Gold
	bought, err := g.UseTheme("dusk", now)
	if err != nil || !bought || g.Gold != gold-themeDefByID("dusk").Price || g.CurrentTheme() != "dusk" {
		t.Fatalf("theme: %v %v gold %d→%d now %s", bought, err, gold, g.Gold, g.CurrentTheme())
	}
	gold = g.Gold
	if _, err := g.UseTheme("meadow", now); err != nil || g.Gold != gold {
		t.Error("going back to a free theme cost gold")
	}
	if bought, err := g.UseTheme("dusk", now); err != nil || bought || g.Gold != gold {
		t.Error("putting a theme back on charged for it twice")
	}
	if _, err := g.UseTheme("nonsense", now); err == nil {
		t.Error("used a theme that does not exist")
	}
	g.Gold = 0
	if _, err := g.UseBorder("hedge", now); err == nil || g.CurrentBorder() != "rounded" {
		t.Error("a border was put on with no gold")
	}
	g.Gold = 500
	if _, err := g.UseBorder("double", now); err != nil || g.bedShape().TopLeft != "╔" {
		t.Errorf("double border: %v, shape %+v", err, g.bedShape())
	}
	applyTheme("dusk")
	if titleStyle.GetForeground() == nil {
		t.Error("titles lost their colour")
	}
}

func TestEveryBorderIsOneCellThick(t *testing.T) {
	for _, d := range borderDefs {
		for _, s := range []string{d.Shape.Top, d.Shape.Bottom, d.Shape.Left, d.Shape.Right, d.Shape.TopLeft, d.Shape.TopRight, d.Shape.BottomLeft, d.Shape.BottomRight} {
			if len([]rune(s)) != 1 {
				t.Errorf("border %s has a part %q that is not one cell", d.ID, s)
			}
		}
		// A bed drawn in it is as wide as ever.
		cell := plainBorder.Border(d.Shape).Render(strings.Repeat("x", cellInner))
		if w := len([]rune(strings.Split(cell, "\n")[0])); w != cellWidth {
			t.Errorf("border %s makes a bed %d wide, want %d", d.ID, w, cellWidth)
		}
	}
}

func TestVillageSaveRoundTrip(t *testing.T) {
	defer applyTheme("meadow")
	g, now := villageGarden(t)
	_ = g.HireHelper("gatherer", now)
	_ = g.BuyScenery("beehive", now)
	_, _ = g.UseTheme("ember", now)
	_, _ = g.UseBorder("stitched", now)
	_ = g.Commission("potpourri", 0, now)
	path := t.TempDir() + "/g.json"
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Hired("gatherer") || !back.HasScenery("beehive") || back.CurrentTheme() != "ember" || back.CurrentBorder() != "stitched" || len(back.Keepsakes) != 1 {
		t.Errorf("the village did not survive a save: %+v", back.Helpers)
	}
}

func TestVillageScreenFlow(t *testing.T) {
	defer applyTheme("meadow")
	g, now := villageGarden(t)
	m := newModel(g, t.TempDir()+"/g.json", now)
	m.width, m.height = 110, 36
	var cur = keyPress(m, "H")
	if cur.(model).screen != screenVillage {
		t.Fatal("H does not open the village")
	}
	for _, tab := range []string{"", "l", "l", "l", "l"} {
		if tab != "" {
			cur = keyPress(cur, tab)
		}
		if v := cur.View(); !strings.Contains(v, "the village") {
			t.Fatalf("blank village on tab %d", cur.(model).villageTab)
		}
	}
	cur = keyPress(cur, "l") // wraps round to the first tab
	if cur.(model).villageTab != tabHelp {
		t.Fatalf("on tab %d, want the hired help tab", cur.(model).villageTab)
	}
	gold := g.Gold
	cur = keyPress(cur, "enter")
	if !g.Hired("waterer") || g.Gold >= gold {
		t.Error("enter did not hire the first helper")
	}
	cur = keyPress(cur, "x")
	if g.Hired("waterer") {
		t.Error("x did not let them go")
	}
	// The lab: a forecast needs two picks.
	cur = keyPress(cur, "l")
	cur = keyPress(cur, "l")
	cur = keyPress(cur, "l")
	if cur.(model).villageTab != tabCommission {
		t.Fatalf("on tab %d", cur.(model).villageTab)
	}
	cur = keyPress(cur, "h")
	if cur.(model).villageTab != tabLab {
		t.Fatalf("on tab %d", cur.(model).villageTab)
	}
	cur = keyPress(cur, "enter") // forecast: pick the first
	if cur.(model).villageStage != villagePick {
		t.Fatal("the lab did not ask for a flower")
	}
	cur = keyPress(cur, "enter")
	if cur.(model).villageStage != villagePick2 {
		t.Fatal("the lab did not ask for a second flower")
	}
	cur = keyPress(cur, "enter")
	if cur.(model).villageStage != villageList || len(cur.(model).villageResult) == 0 {
		t.Error("no forecast shown")
	}
	if v := cur.View(); !strings.Contains(v, "FORECAST") {
		t.Error("the forecast is not on the screen")
	}
	// esc backs out of a choice, then out of the village.
	cur = keyPress(cur, "j")
	cur = keyPress(cur, "enter")
	cur = keyPress(cur, "esc")
	if s := cur.(model); s.villageStage != villageList || s.screen != screenVillage {
		t.Error("esc did not back out of a choice")
	}
	cur = keyPress(cur, "esc")
	if cur.(model).screen != screenGarden {
		t.Error("esc did not leave the village")
	}
}
