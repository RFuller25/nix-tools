package main

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func TestBuyingAMutationThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 500
	before := m.g.Plots[0].Genes()
	// Random mutation is the second service; then the first planted bed.
	m = press(t, m, "N", "down", "enter")
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

func TestBuyingAChosenMutationAndTheWeatherAndTheSeedThroughTheStall(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 1000
	before := m.g.Plots[0].Genes()
	// Chosen: third service, then "Taller" (first), then the first bed.
	m = press(t, m, "N", "down", "down", "enter", "enter", "enter")
	if m.g.Gold != 1000-gnomeChosenCost {
		t.Fatalf("gold = %d after a chosen mutation", m.g.Gold)
	}
	if m.g.Plots[0].Genes().Height <= before.Height && before.Height < 100 {
		t.Error("the plant did not grow taller")
	}

	// Weather: choose the first sky that is not today's.
	m.screen, m.gnomeStage, m.gnomeCursor = screenGnomes, gnomeMenu, 0
	m = press(t, m, "enter")
	if m.gnomeStage != gnomeChoose {
		t.Fatalf("stage = %d, want the sky list", m.gnomeStage)
	}
	gold := m.g.Gold
	if allWeatherKinds[0] == m.g.Weather(m.now).Kind {
		m = press(t, m, "down")
	}
	want := allWeatherKinds[m.gnomeCursor]
	m = press(t, m, "enter")
	if m.g.Weather(m.now).Kind != want || m.g.Gold != gold-gnomeWeatherCost {
		t.Errorf("weather %v gold %d, want %v and %d", m.g.Weather(m.now).Kind, m.g.Gold, want, gold-gnomeWeatherCost)
	}

	// The seed: fourth service, bought at once.
	gold, packets := m.g.Gold, m.g.SeedsInShed()
	m = press(t, m, "down", "down", "down", "enter")
	if m.g.SeedsInShed() != packets+1 || m.g.Gold >= gold {
		t.Error("the wild seed was not bought")
	}
}

func TestStallRefusesWhenBroke(t *testing.T) {
	m := demoModel(t, 100, 40)
	m.g.Gold = 3
	m = press(t, m, "N", "down", "enter", "enter")
	if m.g.Gold != 3 {
		t.Errorf("gold = %d", m.g.Gold)
	}
	if m.status == "" {
		t.Error("the gnomes said nothing")
	}
}
