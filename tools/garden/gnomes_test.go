package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func sunflowerGarden(t *testing.T) (*Garden, time.Time) {
	t.Helper()
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("sunflower"), 0, now); err != nil {
		t.Fatal(err)
	}
	g.Gold = 500
	return g, now
}

func TestGnomesTurnTheWeatherForTheRestOfToday(t *testing.T) {
	g, now := sunflowerGarden(t)
	calendar := WeatherFor(g.Seed, now)
	want := Rain
	if calendar.Kind == Rain {
		want = Heatwave
	}
	if err := g.ChangeWeather(want, now); err != nil {
		t.Fatal(err)
	}
	if g.Gold != 500-gnomeWeatherCost {
		t.Errorf("gold = %d, want %d", g.Gold, 500-gnomeWeatherCost)
	}
	if got := g.Weather(now.Add(3 * time.Hour)); got.Kind != want {
		t.Errorf("weather later today is %v, want %v", got.Kind, want)
	}
	if !g.Spelled(now) {
		t.Error("the sky should be marked as spelled")
	}
	// The calendar takes over again tomorrow.
	tomorrow := now.AddDate(0, 0, 1)
	if got := g.Weather(tomorrow); got != WeatherFor(g.Seed, tomorrow) {
		t.Error("the spell leaked into tomorrow")
	}
	if g.Spelled(tomorrow) {
		t.Error("tomorrow should not be marked as spelled")
	}
}

func TestSpelledRainWatersTheBeds(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Moisture = 0.1
	if g.WeatherAt(now).Kind == Storm {
		t.Skip("calendar already stormy")
	}
	if err := g.ChangeWeather(Storm, now); err != nil {
		t.Fatal(err)
	}
	g.Advance(now.Add(2 * time.Hour))
	if g.Plots[0].Moisture < 0.5 {
		t.Errorf("two hours of storm left the bed at %.2f", g.Plots[0].Moisture)
	}
}

func TestChangingWeatherNeedsGoldAndAChange(t *testing.T) {
	g, now := sunflowerGarden(t)
	if err := g.ChangeWeather(g.WeatherAt(now).Kind, now); err == nil {
		t.Error("changing to the weather we already have should be refused")
	}
	if g.Gold != 500 {
		t.Errorf("a refused spell took gold: %d", g.Gold)
	}
	g.Gold = gnomeWeatherCost - 1
	other := Rain
	if g.WeatherAt(now).Kind == Rain {
		other = Sunny
	}
	if err := g.ChangeWeather(other, now); err == nil {
		t.Error("a spell the purse cannot cover went through")
	}
	if g.Gold != gnomeWeatherCost-1 || g.Spelled(now) {
		t.Error("a refused spell changed something")
	}
	if err := g.ChangeWeather(WeatherKind(99), now); err == nil {
		t.Error("an unknown sky was accepted")
	}
}

func TestOldSpellsAreForgotten(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.GnomeSpells = map[int]WeatherKind{dayKey(now.AddDate(0, 0, -30)): Fog}
	other := Rain
	if g.WeatherAt(now).Kind == Rain {
		other = Sunny
	}
	if err := g.ChangeWeather(other, now); err != nil {
		t.Fatal(err)
	}
	if len(g.GnomeSpells) != 1 {
		t.Errorf("spells = %v, want only today's", g.GnomeSpells)
	}
}

func TestRandomMutationChangesTheGenesAndIsRepeatable(t *testing.T) {
	a, now := sunflowerGarden(t)
	b, _ := sunflowerGarden(t)
	before := a.Plots[0].Genes()
	if _, err := a.MutateRandom(0, now); err != nil {
		t.Fatal(err)
	}
	if _, err := b.MutateRandom(0, now); err != nil {
		t.Fatal(err)
	}
	if a.Gold != 500-gnomeMutateCost {
		t.Errorf("gold = %d", a.Gold)
	}
	after := a.Plots[0].Genes()
	if after == before {
		t.Error("a random mutation changed nothing")
	}
	if after != b.Plots[0].Genes() {
		t.Error("the same garden got two different mutations")
	}
	// A second cast on the same plant rolls something else.
	if _, err := a.MutateRandom(0, now); err != nil {
		t.Fatal(err)
	}
	if a.Plots[0].Genes() == after {
		t.Error("two casts gave the same result")
	}
	if a.Plots[0].Descent == "" {
		t.Error("a mutated plant should say where it came from")
	}
}

func TestMutationRefusesAnEmptyBedWithoutCharging(t *testing.T) {
	g, now := sunflowerGarden(t)
	if _, err := g.MutateRandom(3, now); err == nil {
		t.Error("an empty bed was mutated")
	}
	if _, err := g.MutateChosen(3, "taller", now); err == nil {
		t.Error("an empty bed was mutated by choice")
	}
	if g.Gold != 500 {
		t.Errorf("gold = %d after refusals", g.Gold)
	}
	g.Gold = gnomeMutateCost - 1
	if _, err := g.MutateRandom(0, now); err == nil {
		t.Error("a mutation the purse cannot cover went through")
	}
}

func TestChosenMutationsDoWhatTheySay(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Genome = Genome{Hue: 40, Sat: 80, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
	for _, tc := range []struct {
		id   string
		want func(Genome) bool
	}{
		{"taller", func(x Genome) bool { return x.Height == 50+gnomeNudge }},
		{"slower", func(x Genome) bool { return x.Speed == 50-gnomeNudge }},
		{"generous", func(x Genome) bool { return x.Yield == 50+gnomeNudge }},
		{"blue", func(x Genome) bool { return x.Hue == 220 }},
	} {
		g.Plots[0].Genome = Genome{Hue: 40, Sat: 80, Light: 50, Height: 50, Shape: 50, Speed: 50, Yield: 50}
		gold := g.Gold
		if _, err := g.MutateChosen(0, tc.id, now); err != nil {
			t.Fatalf("%s: %v", tc.id, err)
		}
		if !tc.want(g.Plots[0].Genes()) {
			t.Errorf("%s gave %+v", tc.id, g.Plots[0].Genes())
		}
		if g.Gold != gold-gnomeChosenCost {
			t.Errorf("%s cost %d", tc.id, gold-g.Gold)
		}
	}
}

func TestChosenMutationAtTheLimitIsFreeAndRefused(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Genome = Genome{Hue: 40, Sat: 80, Light: 50, Height: 100, Shape: 50, Speed: 50, Yield: 50}
	if _, err := g.MutateChosen(0, "taller", now); err == nil {
		t.Error("pushing the tallest plant taller should be refused")
	}
	if g.Gold != 500 {
		t.Error("a refused mutation took gold")
	}
	if _, err := g.MutateChosen(0, "no-such-thing", now); err == nil {
		t.Error("an unknown mutation was accepted")
	}
}

func TestMutationBreaksTheLine(t *testing.T) {
	g, now := sunflowerGarden(t)
	p := &g.Plots[0]
	p.Line, p.LineName, p.Streak = 7, "Old Faithful", 2
	if _, err := g.MutateChosen(0, "taller", now); err != nil {
		t.Fatal(err)
	}
	if p.Line != 0 || p.LineName != "" || p.Streak != 0 {
		t.Errorf("mutated plant kept its line: %d %q %d", p.Line, p.LineName, p.Streak)
	}
}

func TestAMutatedFlowerIsFiledAsAHybrid(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Growth, g.Plots[0].Matured = 1, true
	if _, err := g.MutateChosen(0, "violet", now); err != nil {
		t.Fatal(err)
	}
	if !g.Plots[0].IsHybrid() {
		t.Skip("violet is still a named form of this plant")
	}
	if len(g.Cultivars) != 1 {
		t.Errorf("cultivars = %d, want the mutant filed", len(g.Cultivars))
	}
}

func TestWildSeedIsStrangeRepeatableAndOnceADay(t *testing.T) {
	a, now := sunflowerGarden(t)
	b, _ := sunflowerGarden(t)
	day := dayKey(now)
	wa, wb := a.WildSeedFor(day), b.WildSeedFor(day)
	if wa.Species.ID != wb.Species.ID || wa.Genome != wb.Genome {
		t.Error("the same garden was offered two different seeds")
	}
	if wa.Species.Kind == KindAquatic {
		t.Error("the teapot brewed a water plant")
	}
	extremes := 0
	for _, v := range []uint8{wa.Genome.Height, wa.Genome.Shape, wa.Genome.Speed, wa.Genome.Yield} {
		if v <= 12 || v >= 88 {
			extremes++
		}
	}
	if extremes != 4 {
		t.Errorf("only %d of 4 genes are extreme: %+v", extremes, wa.Genome)
	}
	// Different days brew different seeds.
	distinct := map[string]bool{}
	for d := 0; d < 20; d++ {
		w := a.WildSeedFor(dayKey(now.AddDate(0, 0, d)))
		distinct[w.Species.ID+w.Genome.Hex()] = true
	}
	if len(distinct) < 15 {
		t.Errorf("20 days gave only %d different seeds", len(distinct))
	}

	before := a.SeedsInShed()
	w, err := a.BuyWildSeed(now)
	if err != nil {
		t.Fatal(err)
	}
	if a.Gold != 500-w.Price {
		t.Errorf("gold = %d, want %d", a.Gold, 500-w.Price)
	}
	if a.SeedsInShed() != before+1 {
		t.Error("the seed is not in the shed")
	}
	if _, err := a.BuyWildSeed(now.Add(time.Hour)); err == nil {
		t.Error("a second wild seed was sold on the same day")
	}
	if a.WildSeedAvailable(now) {
		t.Error("the seed should be gone")
	}
	if !a.WildSeedAvailable(now.AddDate(0, 0, 1)) {
		t.Error("tomorrow's seed should be on the shelf")
	}
}

func TestWildSeedComesExactlyTrueWhenSown(t *testing.T) {
	g, now := sunflowerGarden(t)
	w, err := g.BuyWildSeed(now)
	if err != nil {
		t.Fatal(err)
	}
	pk := len(g.Shed) - 1
	if err := g.SowPacket(1, pk, now); err != nil && w.Species.Kind != KindAquatic {
		t.Fatal(err)
	}
	if got := g.Plots[1].Genes(); got != w.Genome {
		t.Errorf("sown genes %+v, want %+v", got, w.Genome)
	}
	if g.Plots[1].SpeciesID != w.Species.ID {
		t.Errorf("sown species %s, want %s", g.Plots[1].SpeciesID, w.Species.ID)
	}
}

func TestWildSeedCostsMoreThanItCanSellFor(t *testing.T) {
	g, now := sunflowerGarden(t)
	for d := 0; d < 60; d++ {
		w := g.WildSeedFor(dayKey(now.AddDate(0, 0, d)))
		pk := Packet{SpeciesID: w.Species.ID, A: w.Genome, B: w.Genome, Count: 1, Pure: true}
		if seedValue(pk) >= w.Price {
			t.Fatalf("day %d: %s sells for %d but costs %d", d, w.Species.ID, seedValue(pk), w.Price)
		}
	}
}

func TestGnomeStateSurvivesTheSaveFile(t *testing.T) {
	g, now := sunflowerGarden(t)
	other := Rain
	if g.WeatherAt(now).Kind == Rain {
		other = Sunny
	}
	if err := g.ChangeWeather(other, now); err != nil {
		t.Fatal(err)
	}
	if _, err := g.BuyWildSeed(now); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "garden.json")
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if back.Weather(now).Kind != other || back.GnomeSeedDay != dayKey(now) || back.GnomeSpent != g.GnomeSpent {
		t.Errorf("gnome state lost: %+v", back.GnomeSpells)
	}
}

// ---- the stall --------------------------------------------------------------

func press(t *testing.T, m model, keys ...string) model {
	t.Helper()
	var cur tea.Model = m
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		cur, _ = cur.Update(msg)
	}
	return cur.(model)
}

func TestTheStallIsReachableByKeyAndByTab(t *testing.T) {
	m := demoModel(t, 100, 40)
	if got := press(t, m, "N").screen; got != screenGnomes {
		t.Errorf("N went to screen %d", got)
	}
	m.screen = screenFair
	if got := press(t, m, "tab").screen; got != screenGnomes {
		t.Errorf("tab from the fair went to screen %d", got)
	}
	m.screen = screenGnomes
	if got := press(t, m, "tab").screen; got != screenAlmanac {
		t.Errorf("tab from the stall went to screen %d", got)
	}
	if got := press(t, m, "esc").screen; got != screenGarden {
		t.Errorf("esc from the stall went to screen %d", got)
	}
}

// atStall opens the stall with a service highlighted.
func atStall(m model, svc GnomeService) model {
	m.openGnomes()
	m.gnomeCursor = int(svc)
	return m
}

func TestBuyingAMutationThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 500
	before := m.g.Plots[0].Genes()
	m = press(t, atStall(m, ServiceMutate), "enter")
	if m.gnomeStage != gnomeBed {
		t.Fatalf("stage = %d, want the plant list", m.gnomeStage)
	}
	m = press(t, m, "enter")
	if m.g.Plots[0].Genes() == before {
		t.Error("the plant did not change")
	}
	if m.g.Gold != 500-gnomeMutateCost {
		t.Errorf("gold = %d", m.g.Gold)
	}
	// esc steps back to the front, and once more leaves.
	m = press(t, m, "esc")
	if m.screen != screenGnomes || m.gnomeStage != gnomeMenu {
		t.Errorf("esc left the stall early: screen %d stage %d", m.screen, m.gnomeStage)
	}
	if got := press(t, m, "esc").screen; got != screenGarden {
		t.Errorf("second esc went to %d", got)
	}
}

func TestBuyingAChosenMutationThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 1000
	before := m.g.Plots[0].Genes()
	// "Taller" is first on the list, the sunflower the first bed.
	m = press(t, atStall(m, ServiceChosen), "enter", "enter", "enter")
	if m.g.Gold != 1000-gnomeChosenCost {
		t.Fatalf("gold = %d after a chosen mutation", m.g.Gold)
	}
	if m.g.Plots[0].Genes().Height <= before.Height && before.Height < 100 {
		t.Error("the plant did not grow taller")
	}
}

func TestBuyingTheWeatherAndTheSeasonAndTheSeedThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 1000
	m = press(t, atStall(m, ServiceWeather), "enter")
	if m.gnomeStage != gnomeChoose {
		t.Fatalf("stage = %d, want the sky list", m.gnomeStage)
	}
	if allWeatherKinds[0] == m.g.Weather(m.now).Kind {
		m = press(t, m, "down")
	}
	want := allWeatherKinds[m.gnomeCursor]
	m = press(t, m, "enter")
	if m.g.Weather(m.now).Kind != want || m.g.Gold != 1000-gnomeWeatherCost {
		t.Errorf("weather %v gold %d", m.g.Weather(m.now).Kind, m.g.Gold)
	}

	gold := m.g.Gold
	m = press(t, atStall(m, ServiceSeason), "enter")
	if m.g.Season(m.now) == allSeasons[m.gnomeCursor] {
		m = press(t, m, "down")
	}
	season := allSeasons[m.gnomeCursor]
	m = press(t, m, "enter")
	if m.g.Season(m.now) != season || m.g.Gold != gold-gnomeSeasonCost {
		t.Errorf("season %v gold %d", m.g.Season(m.now), m.g.Gold)
	}

	gold, seeds := m.g.Gold, m.g.SeedsInShed()
	m = press(t, atStall(m, ServiceSeed), "enter")
	if m.g.SeedsInShed() != seeds+1 || m.g.Gold >= gold {
		t.Error("the wild seed was not bought")
	}
}

func TestGrowthSpurtAndSoilThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 1000
	// The first plant that can still grow is the sunflower, at growth 0.
	m.g.Plots[0].Growth = 0
	m = press(t, atStall(m, ServiceGrowth), "enter", "enter")
	if m.g.Plots[0].Growth != 0.10 || m.g.Gold != 1000-gnomeGrowthCost {
		t.Errorf("growth %.2f gold %d", m.g.Plots[0].Growth, m.g.Gold)
	}
	// Soil: "More acid" is first, bed 1 is the first bed.
	ph := m.g.Plots[0].PH
	m = press(t, atStall(m, ServiceSoil), "enter", "enter", "enter")
	if m.g.Plots[0].PH >= ph || m.g.Gold != 1000-gnomeGrowthCost-gnomeSoilCost {
		t.Errorf("pH %.1f → %.1f, gold %d", ph, m.g.Plots[0].PH, m.g.Gold)
	}
}

func TestMatchmakerThroughTheStallAsksForTwoFlowers(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 1000
	for _, i := range []int{0, 1} {
		m.g.Plots[i].Growth, m.g.Plots[i].Matured = 1, true
	}
	m = press(t, atStall(m, ServiceMatch), "enter")
	if m.gnomeStage != gnomeBed {
		t.Fatalf("stage = %d, want the first flower", m.gnomeStage)
	}
	m = press(t, m, "enter")
	if m.gnomeStage != gnomeBed2 {
		t.Fatalf("stage = %d, want the second flower", m.gnomeStage)
	}
	for _, b := range m.gnomeBeds() {
		if b == m.gnomeMother {
			t.Error("a flower is offered as its own partner")
		}
	}
	// Both screens render with prices.
	if v := m.View(); v == "" {
		t.Error("blank screen")
	}
	m = press(t, m, "enter")
	if m.g.Plots[m.gnomeMother].Pollen == nil {
		t.Error("the match did nothing")
	}
	if m.g.Gold >= 1000 {
		t.Error("the match was free")
	}
	// esc from the partner list goes back to the first flower.
	m = press(t, m, "enter", "esc")
	if m.gnomeStage != gnomeBed {
		t.Errorf("esc from the partner list went to stage %d", m.gnomeStage)
	}
}

func TestStallRefusesWhenBroke(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 3
	m = press(t, atStall(m, ServiceMutate), "enter", "enter")
	if m.g.Gold != 3 {
		t.Errorf("gold = %d", m.g.Gold)
	}
	if m.status == "" {
		t.Error("the gnomes said nothing")
	}
}

func TestStallWithNothingToWorkOnSaysSo(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	m := newModel(g, "/tmp/garden.json", now)
	m.width, m.height = 100, 30
	for _, svc := range []GnomeService{ServiceGrowth, ServiceMutate, ServiceMatch} {
		got := press(t, atStall(m, svc), "enter")
		if got.gnomeStage != gnomeMenu || got.status == "" {
			t.Errorf("service %d: stage %d status %q", svc, got.gnomeStage, got.status)
		}
	}
}

// ---- season swap ------------------------------------------------------------

func TestGnomesBorrowASeasonForTheRestOfToday(t *testing.T) {
	g, now := sunflowerGarden(t) // mid June: summer
	if err := g.ChangeSeason(Summer, now); err == nil {
		t.Error("swapping to the season we are in was allowed")
	}
	if err := g.ChangeSeason(Winter, now); err != nil {
		t.Fatal(err)
	}
	if g.Gold != 500-gnomeSeasonCost {
		t.Errorf("gold = %d", g.Gold)
	}
	if g.Season(now.Add(2*time.Hour)) != Winter || !g.SeasonSwapped(now) {
		t.Error("it is not winter today")
	}
	tomorrow := now.AddDate(0, 0, 1)
	if g.Season(tomorrow) != Summer || g.SeasonSwapped(tomorrow) {
		t.Error("the borrowed season leaked into tomorrow")
	}
	// Today's weather is drawn from winter's table.
	if got, want := g.Weather(now), weatherIn(g.Seed, now, Winter); got != want {
		t.Errorf("weather %v, want %v", got.Kind, want.Kind)
	}
	g.Gold = gnomeSeasonCost - 1
	if err := g.ChangeSeason(Autumn, now); err == nil {
		t.Error("a swap the purse cannot cover went through")
	}
	if err := g.ChangeSeason(Season(9), now); err == nil {
		t.Error("an unknown season was accepted")
	}
}

func TestABorrowedSeasonChangesGrowth(t *testing.T) {
	// A summer annual grows more slowly in a borrowed winter.
	a, now := sunflowerGarden(t)
	b, _ := sunflowerGarden(t)
	if err := b.ChangeSeason(Winter, now); err != nil {
		t.Fatal(err)
	}
	a.Plots[0].Moisture, b.Plots[0].Moisture = 1, 1
	a.Advance(now.Add(time.Hour))
	b.Advance(now.Add(time.Hour))
	if b.Plots[0].Growth >= a.Plots[0].Growth {
		t.Errorf("winter growth %.3f is not slower than summer's %.3f", b.Plots[0].Growth, a.Plots[0].Growth)
	}
}

// ---- growth spurt -----------------------------------------------------------

func TestGrowthSpurtClimbsExactlyOneStage(t *testing.T) {
	g, now := sunflowerGarden(t)
	want := []float64{0.10, 0.35, 0.70}
	for i, w := range want {
		stage := g.Plots[0].Stage()
		if _, err := g.GrowthSpurt(0, now); err != nil {
			t.Fatalf("spurt %d: %v", i, err)
		}
		if g.Plots[0].Growth != w {
			t.Errorf("spurt %d: growth %.2f, want %.2f", i, g.Plots[0].Growth, w)
		}
		if g.Plots[0].Stage() != stage+1 {
			t.Errorf("spurt %d moved from stage %d to %d", i, stage, g.Plots[0].Stage())
		}
		if g.Plots[0].Matured {
			t.Errorf("spurt %d reached bloom too soon", i)
		}
	}
	if g.Gold != 500-3*gnomeGrowthCost {
		t.Errorf("gold = %d", g.Gold)
	}
}

func TestGrowthSpurtFromMidStageGoesToTheNextOne(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Growth = 0.5 // seedling
	if _, err := g.GrowthSpurt(0, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[0].Growth != 0.70 {
		t.Errorf("growth = %.2f, want the bud stage at 0.70", g.Plots[0].Growth)
	}
}

func TestGrowthSpurtToBloomCountsAsBlooming(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Plots[0].Growth = 0.8
	matured := g.Matured
	if _, err := g.GrowthSpurt(0, now); err != nil {
		t.Fatal(err)
	}
	p := &g.Plots[0]
	if p.Growth != 1 || !p.Matured || g.Matured != matured+1 {
		t.Errorf("growth %.2f matured %v count %d→%d", p.Growth, p.Matured, matured, g.Matured)
	}
	if _, ok := g.Collected(SpeciesByID("sunflower")); !ok {
		t.Error("the bloom was not pressed into the herbarium")
	}
	// Nothing further to give.
	gold := g.Gold
	if _, err := g.GrowthSpurt(0, now); err == nil {
		t.Error("a plant in bloom was given a spurt")
	}
	if g.Gold != gold {
		t.Error("a refused spurt took gold")
	}
	if _, err := g.GrowthSpurt(5, now); err == nil {
		t.Error("an empty bed was given a spurt")
	}
}

// ---- soil alchemy -----------------------------------------------------------

func TestSoilAlchemyMovesPHAndRichness(t *testing.T) {
	g, now := sunflowerGarden(t)
	p := &g.Plots[2]
	p.PH, p.Richness = 6.5, 0.3
	if _, err := g.TreatSoil(2, "acid", now); err != nil {
		t.Fatal(err)
	}
	if got := p.PH; got < 5.29 || got > 5.31 {
		t.Errorf("pH = %.2f, want 5.3", got)
	}
	if _, err := g.TreatSoil(2, "chalk", now); err != nil {
		t.Fatal(err)
	}
	if got := p.PH; got < 6.49 || got > 6.51 {
		t.Errorf("pH = %.2f, want 6.5", got)
	}
	if _, err := g.TreatSoil(2, "enrich", now); err != nil {
		t.Fatal(err)
	}
	if got := p.Richness; got < 0.69 || got > 0.71 {
		t.Errorf("richness = %.2f, want 0.7", got)
	}
	p.PH = 5.0
	if _, err := g.TreatSoil(2, "neutral", now); err != nil || p.PH != 6.5 {
		t.Errorf("neutralise: %v, pH %.1f", err, p.PH)
	}
	if g.Gold != 500-4*gnomeSoilCost {
		t.Errorf("gold = %d", g.Gold)
	}
}

func TestSoilAlchemyHonoursTheLimits(t *testing.T) {
	g, now := sunflowerGarden(t)
	p := &g.Plots[2]
	p.PH, p.Richness = phMin, 1
	for _, id := range []string{"acid", "enrich"} {
		if _, err := g.TreatSoil(2, id, now); err == nil {
			t.Errorf("%s at the limit was accepted", id)
		}
	}
	p.PH = 7.8
	if _, err := g.TreatSoil(2, "chalk", now); err != nil || p.PH != phMax {
		t.Errorf("pH = %.1f, want it capped at %.1f (err %v)", p.PH, phMax, err)
	}
	if g.Gold != 500-gnomeSoilCost {
		t.Errorf("refusals took gold: %d", g.Gold)
	}
	g.Plots[3].Pond = true
	if _, err := g.TreatSoil(3, "acid", now); err == nil {
		t.Error("a pond was treated")
	}
	if _, err := g.TreatSoil(2, "nonsense", now); err == nil {
		t.Error("an unknown recipe was accepted")
	}
}

func TestSoilAlchemyCanFixAnAcidLover(t *testing.T) {
	g, now := sunflowerGarden(t)
	maple := SpeciesByID("japanesemaple")
	if err := g.Plant(4, maple, 0, now); err != nil {
		t.Fatal(err)
	}
	g.Plots[4].PH = 7.5
	before := soilFactor(maple, &g.Plots[4])
	for i := 0; i < 3; i++ {
		if _, err := g.TreatSoil(4, "acid", now); err != nil {
			t.Fatal(err)
		}
	}
	if after := soilFactor(maple, &g.Plots[4]); after <= before {
		t.Errorf("soil factor %.2f did not improve on %.2f", after, before)
	}
}

// ---- matchmaker -------------------------------------------------------------

func flowering(g *Garden, idx int, id string, now time.Time) {
	sp := SpeciesByID(id)
	old := g.Plots[idx]
	g.Plots[idx] = Plot{PH: old.PH, Richness: old.Richness, Pond: old.Pond}
	if sp.Kind == KindAquatic {
		g.Plots[idx].Pond = true
	}
	if err := g.Plant(idx, sp, 0, now); err != nil {
		panic(err)
	}
	g.Plots[idx].Growth, g.Plots[idx].Matured, g.Plots[idx].Pods = 1, true, 4
}

func TestMatchmakerPricesByHowRelatedTheFlowersAre(t *testing.T) {
	sun, zin, fox := SpeciesByID("sunflower"), SpeciesByID("zinnia"), SpeciesByID("foxglove")
	if sun.Family != zin.Family || sun.Family == fox.Family {
		t.Skip("catalogue families changed")
	}
	for _, tc := range []struct {
		a, b *Species
		cost int
		kind MatchKind
	}{
		{sun, sun, gnomeMatchSame, MatchSame},
		{sun, zin, gnomeMatchFamily, MatchFamily},
		{sun, fox, gnomeMatchWide, MatchWide},
	} {
		if cost, kind := MatchCost(tc.a, tc.b); cost != tc.cost || kind != tc.kind {
			t.Errorf("%s × %s = %d %v, want %d %v", tc.a.ID, tc.b.ID, cost, kind, tc.cost, tc.kind)
		}
	}
	for _, sp := range AllSpecies() {
		if sp.Kind == KindAquatic {
			if _, kind := MatchCost(sp, sun); kind != MatchNever {
				t.Errorf("%s was matched with a land plant", sp.ID)
			}
			break
		}
	}
}

func TestMatchmakerSetsPollenAnywhereAnytime(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	flowering(g, 14, "sunflower", now)
	g.Plots[14].Genome = Genome{Hue: 200, Sat: 90, Light: 50, Height: 90, Shape: 50, Speed: 50, Yield: 50}
	// Dead of night, in a storm: the brush would refuse, magic does not.
	night := time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC)
	if err := g.ChangeWeather(Storm, night); err != nil {
		t.Fatal(err)
	}
	gold := g.Gold
	cost, err := g.Matchmake(0, 14, night)
	if err != nil {
		t.Fatal(err)
	}
	if cost != gnomeMatchSame || g.Gold != gold-cost {
		t.Errorf("cost %d gold %d→%d", cost, gold, g.Gold)
	}
	if g.Plots[0].Pollen == nil || *g.Plots[0].Pollen != g.Plots[14].Genes() || g.Plots[0].PollenFrom != "" {
		t.Error("the donor's pollen is not on the flower")
	}
	if g.Gather(0, night) != 4 {
		t.Error("gather did not return the seed")
	}
	pk := g.Shed[len(g.Shed)-1]
	if pk.SpeciesID != "sunflower" || pk.B != g.Plots[14].Genes() {
		t.Errorf("packet %+v is not the cross", pk)
	}
}

func TestMatchmakerRefusals(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	if err := g.Plant(1, SpeciesByID("zinnia"), 0, now); err != nil {
		t.Fatal(err)
	}
	gold := g.Gold
	if _, err := g.Matchmake(0, 0, now); err == nil {
		t.Error("a flower was matched with itself")
	}
	if _, err := g.Matchmake(0, 1, now); err == nil {
		t.Error("a seedling was used as a partner")
	}
	if _, err := g.Matchmake(0, 7, now); err == nil {
		t.Error("an empty bed was used as a partner")
	}
	var pond *Species
	for _, sp := range AllSpecies() {
		if sp.Kind == KindAquatic {
			pond = sp
			break
		}
	}
	if pond != nil {
		flowering(g, 2, pond.ID, now)
		gold = g.Gold
		if _, err := g.Matchmake(0, 2, now); err == nil {
			t.Error("a land plant was matched with a water plant")
		}
	}
	if g.Gold != gold {
		t.Error("a refusal took gold")
	}
	flowering(g, 3, "foxglove", now)
	g.Gold = gnomeMatchWide - 1
	if _, err := g.Matchmake(0, 3, now); err == nil {
		t.Error("a match the purse cannot cover went through")
	}
}

func TestCrossSpeciesSeedComesUpAsEitherParent(t *testing.T) {
	g, now := sunflowerGarden(t)
	g.Gold = 100000
	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		flowering(g, 0, "sunflower", now)
		flowering(g, 1, "zinnia", now)
		if _, err := g.Matchmake(0, 1, now); err != nil {
			t.Fatal(err)
		}
		if g.Plots[0].PollenFrom != "zinnia" {
			t.Fatalf("PollenFrom = %q", g.Plots[0].PollenFrom)
		}
		g.Gather(0, now)
		pk := g.Shed[len(g.Shed)-1]
		seen[pk.SpeciesID] = true
		if pk.Descent == "" || pk.Count != 4 {
			t.Fatalf("family cross packet %+v", pk)
		}
		if g.Plots[0].Pollen != nil || g.Plots[0].PollenFrom != "" {
			t.Fatal("gathering left the pollen on the plant")
		}
	}
	if !seen["sunflower"] || !seen["zinnia"] {
		t.Errorf("40 gatherings only ever gave %v", seen)
	}
}

func TestWideCrossesSetHalfTheSeed(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	flowering(g, 1, "foxglove", now)
	if _, err := g.Matchmake(0, 1, now); err != nil {
		t.Fatal(err)
	}
	if got := g.Gather(0, now); got != 2 {
		t.Errorf("a cross between families set %d seeds from 4 pods, want 2", got)
	}
	if pk := g.Shed[len(g.Shed)-1]; pk.Count != 2 || (pk.SpeciesID != "sunflower" && pk.SpeciesID != "foxglove") {
		t.Errorf("packet %+v", pk)
	}
}

func TestCrossSpeciesSeedCanBeSownAndGrows(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	flowering(g, 1, "foxglove", now)
	g.Matchmake(0, 1, now)
	g.Gather(0, now)
	pk := len(g.Shed) - 1
	species := g.Shed[pk].SpeciesID
	if err := g.SowPacket(5, pk, now); err != nil {
		t.Fatal(err)
	}
	if g.Plots[5].SpeciesID != species {
		t.Errorf("sowed %s, packet was %s", g.Plots[5].SpeciesID, species)
	}
	g.Advance(now.Add(36 * time.Hour)) // must simulate without trouble
}

func TestOpenPollinationReplacesAMatchedPollen(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	flowering(g, 1, "foxglove", now)
	flowering(g, 2, "sunflower", now)
	g.Matchmake(0, 1, now)
	noon := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	if err := g.HandPollinate(0, 2, noon); err != nil {
		t.Skipf("brush refused: %v", err)
	}
	if g.Plots[0].PollenFrom != "" {
		t.Error("a brush of own-kind pollen kept the foreign species marker")
	}
}

func TestFamilyMatchStillSavesAndLoads(t *testing.T) {
	g, now := sunflowerGarden(t)
	flowering(g, 0, "sunflower", now)
	flowering(g, 1, "zinnia", now)
	g.Matchmake(0, 1, now)
	g.ChangeSeason(Winter, now)
	path := filepath.Join(t.TempDir(), "garden.json")
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if back.Plots[0].PollenFrom != "zinnia" || back.Season(now) != Winter {
		t.Errorf("lost: pollen from %q, season %v", back.Plots[0].PollenFrom, back.Season(now))
	}
}

// Every stage of every service fits the window, however small.
func TestEveryStallStageFitsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {80, 24}, {60, 18}, {40, 12}, {24, 10}} {
		for svc := GnomeService(0); svc < serviceCount; svc++ {
			m := demoModel(t, size[0], size[1])
			m.g.Gold = 1000
			for _, i := range []int{0, 1} {
				m.g.Plots[i].Growth, m.g.Plots[i].Matured = 1, true
			}
			m.gnomeMother = 0
			for _, stage := range []gnomeStage{gnomeMenu, gnomeChoose, gnomeBed, gnomeBed2} {
				if stage == gnomeBed2 && svc != ServiceMatch {
					continue
				}
				m.screen, m.gnomeService, m.gnomeStage = screenGnomes, svc, stage
				for _, cur := range []int{0, 1, 3, 7} {
					m.gnomeCursor = cur
					lines := strings.Split(m.View(), "\n")
					if len(lines) > size[1] {
						t.Errorf("svc %d stage %d cursor %d at %dx%d: %d lines", svc, stage, cur, size[0], size[1], len(lines))
					}
					for _, l := range lines {
						if w := lipgloss.Width(l); w > size[0] {
							t.Errorf("svc %d stage %d at %dx%d: line %d wide: %q", svc, stage, size[0], size[1], w, l)
						}
					}
				}
			}
		}
	}
}
