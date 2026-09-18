package main

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

// The garden is a fixed five-bed-wide world that grows downwards as the
// gardener buys more ground. Beds keep their positions, so neighbours stay
// neighbours however the terminal is sized.
const (
	plotCols    = 5
	PlotCount   = 15 // beds a new garden starts with
	maxPlots    = 30
	bedBaseCost = 18 // seeds for the first extra bed
	bedStepCost = 6  // added for each bed after that
	pondCost    = 12
)

// Tuning constants for the growth simulation. A well-watered, weeded plant
// reaches maturity in Species.Hours, which is under a day for every species.
const (
	baseDryPerHour  = 0.055 // moisture lost per hour in average weather
	baseWeedPerHour = 0.013 // weed pressure gained per hour
	podsPerHour     = 1.0 / 6.0
	maxPods         = 5.0
	simStepHours    = 0.25
	maxCatchUpDays  = 45.0
	dailyBonus      = 3
	weedingReward   = 1
)

// Plot is one bed in the garden: empty, or holding a single plant.
type Plot struct {
	SpeciesID string    `json:"species_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	PlantedAt time.Time `json:"planted_at,omitempty"`
	Growth    float64   `json:"growth"`   // 0..1 toward mature
	Moisture  float64   `json:"moisture"` // 0..1
	Weeds     float64   `json:"weeds"`    // 0..1
	Pods      float64   `json:"pods"`     // ripe seed pods waiting to be gathered
	Matured   bool      `json:"matured"`  // has reached the mature stage at least once

	// MaturedAt is when it first came into flower, and Spent marks an annual
	// or biennial that has finished its year and gone to seed. Nothing dies:
	// a spent plant stands there, dry and full of seed, until it is lifted.
	MaturedAt time.Time `json:"matured_at,omitempty"`
	Spent     bool      `json:"spent,omitempty"`

	// Pond beds hold water instead of soil. Only the aquatic species will
	// grow in one, and nothing else will.
	Pond bool `json:"pond,omitempty"`

	// Soil. pH runs from about 4.5 (peat bog) to 8 (chalk); richness is how
	// much compost the bed has had worked into it.
	PH       float64 `json:"ph"`
	Richness float64 `json:"richness"`
}

// Empty reports whether anything is planted here.
func (p *Plot) Empty() bool { return p.SpeciesID == "" }

// Soil describes a bed's earth in words, for the info card.
func (p *Plot) Soil() string {
	switch {
	case p.Pond:
		return "still water"
	case p.PH < 5.6:
		return fmt.Sprintf("acid, pH %.1f", p.PH)
	case p.PH > 7.3:
		return fmt.Sprintf("chalky, pH %.1f", p.PH)
	default:
		return fmt.Sprintf("neutral, pH %.1f", p.PH)
	}
}

// Species resolves the plot's species, or nil when the bed is empty.
func (p *Plot) Species() *Species {
	if p.Empty() {
		return nil
	}
	return SpeciesByID(p.SpeciesID)
}

// Stage is the drawn life stage the plant currently occupies.
func (p *Plot) Stage() int {
	switch {
	case p.Growth >= 1.0:
		return StageMature
	case p.Growth >= 0.70:
		return StageBud
	case p.Growth >= 0.35:
		return StageSeedling
	case p.Growth >= 0.10:
		return StageSprout
	default:
		return StageSeed
	}
}

func (p *Plot) StageName() string { return stageNames[p.Stage()] }

// DisplayName is the gardener's chosen name, falling back to the common name.
func (p *Plot) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if sp := p.Species(); sp != nil {
		return sp.Common
	}
	return "empty bed"
}

// Thirsty reports whether the plant would appreciate the watering can.
func (p *Plot) Thirsty() bool { return !p.Empty() && p.Moisture < 0.35 }

// Weedy reports whether weeds are crowding the bed.
func (p *Plot) Weedy() bool { return !p.Empty() && p.Weeds > 0.45 }

// Mood is a short, cheerful status line. Nothing here ever dies; a neglected
// plant simply sulks and grows slowly until someone comes back.
func (p *Plot) Mood() string {
	switch {
	case p.Empty():
		if p.Pond {
			return "still water"
		}
		return "bare soil"
	case p.Spent:
		return "gone to seed"
	case p.Moisture < 0.15:
		return "parched, but holding on"
	case p.Thirsty() && p.Weedy():
		return "thirsty and crowded"
	case p.Thirsty():
		return "thirsty"
	case p.Weedy():
		return "crowded by weeds"
	case p.Growth >= 1.0 && p.Pods >= 1:
		return "ripe with seed"
	case p.Growth >= 1.0:
		return "in full glory"
	case p.Moisture > 0.75:
		return "well watered"
	default:
		return "growing nicely"
	}
}

// Age returns how long the plant has been in the ground.
func (p *Plot) Age(now time.Time) time.Duration {
	if p.Empty() {
		return 0
	}
	return now.Sub(p.PlantedAt)
}

// JournalEntry is a dated line in the garden's log.
type JournalEntry struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

// Garden is the whole saved world.
type Garden struct {
	Version    int    `json:"version"`
	Seed       int64  `json:"seed"`
	Gardener   string `json:"gardener,omitempty"`
	Plots      []Plot `json:"plots"`
	Seeds      int    `json:"seeds"`
	Matured    int    `json:"matured"`    // lifetime plants brought to maturity
	Planted    int    `json:"planted"`    // lifetime plants sown
	Gathered   int    `json:"gathered"`   // lifetime seeds gathered
	Volunteers int    `json:"volunteers"` // plants that sowed themselves
	Music      bool   `json:"music"`      // was the music playing when we last closed
	// Herbarium records the first time each species was brought into flower,
	// and Sightings the first time each creature came to visit.
	Herbarium map[string]time.Time `json:"herbarium,omitempty"`
	Sightings map[string]time.Time `json:"sightings,omitempty"`
	// Tasks are the gentle suggestions the journal keeps.
	Tasks     []TaskState    `json:"tasks,omitempty"`
	Journal   []JournalEntry `json:"journal"`
	LastTick  time.Time      `json:"last_tick"`
	LastVisit time.Time      `json:"last_visit"`
	Created   time.Time      `json:"created"`
}

const gardenVersion = 1

// NewGarden makes a fresh, empty garden with a handful of starter seeds.
func NewGarden(now time.Time) *Garden {
	g := &Garden{
		Version:  gardenVersion,
		Seed:     rand.Int63(),
		Plots:    make([]Plot, PlotCount),
		Seeds:    14,
		LastTick: now,
		Created:  now,
	}
	g.layOutSoil()
	g.refreshTasks(now)
	g.Log(now, "A patch of bare earth. Something could grow here.")
	return g
}

// layOutSoil gives every bed its own patch of ground, derived from the
// garden's seed so the same garden always has the same soil.
func (g *Garden) layOutSoil() {
	for i := range g.Plots {
		if g.Plots[i].PH == 0 {
			g.Plots[i].PH = 5.7 + 1.7*hashUnit(g.Seed, int64(i), 0x501)
		}
		if g.Plots[i].Richness == 0 {
			g.Plots[i].Richness = 0.3 + 0.35*hashUnit(g.Seed, int64(i), 0x502)
		}
	}
}

// Rows is how many rows of beds the garden currently has.
func (g *Garden) Rows() int {
	return (len(g.Plots) + plotCols - 1) / plotCols
}

// Neighbours lists the beds sharing an edge with this one. Companion planting
// works on these, and so does a plant sowing itself about.
func (g *Garden) Neighbours(idx int) []int {
	col, row := idx%plotCols, idx/plotCols
	var out []int
	add := func(c, r int) {
		if c < 0 || c >= plotCols || r < 0 {
			return
		}
		if n := r*plotCols + c; n >= 0 && n < len(g.Plots) {
			out = append(out, n)
		}
	}
	add(col-1, row)
	add(col+1, row)
	add(col, row-1)
	add(col, row+1)
	return out
}

// BedCost is what the next new bed costs, rising as the garden spreads.
func (g *Garden) BedCost() int {
	extra := len(g.Plots) - PlotCount
	return bedBaseCost + bedStepCost*extra
}

// BuyBed breaks new ground at the bottom of the garden.
func (g *Garden) BuyBed(now time.Time) error {
	if len(g.Plots) >= maxPlots {
		return fmt.Errorf("there is no more room to dig")
	}
	cost := g.BedCost()
	if g.Seeds < cost {
		return fmt.Errorf("breaking new ground costs %d seeds", cost)
	}
	g.Seeds -= cost
	g.Plots = append(g.Plots, Plot{})
	g.layOutSoil()
	g.Log(now, "Broke new ground: bed %d is ready.", len(g.Plots))
	return nil
}

// DigPond turns an empty bed into water, which is the only place the aquatic
// plants will grow. Filling it back in returns it to ordinary soil.
func (g *Garden) DigPond(idx int, now time.Time) error {
	p := &g.Plots[idx]
	if !p.Empty() {
		return fmt.Errorf("lift %s first", p.DisplayName())
	}
	if p.Pond {
		p.Pond = false
		p.Moisture = 0.6
		g.Log(now, "Filled in the pond in bed %d.", idx+1)
		return nil
	}
	if g.Seeds < pondCost {
		return fmt.Errorf("digging a pond costs %d seeds", pondCost)
	}
	g.Seeds -= pondCost
	p.Pond = true
	p.Moisture = 1
	p.Weeds = 0
	g.Log(now, "Dug a pond in bed %d.", idx+1)
	return nil
}

// collect files a species in the herbarium the first time it flowers.
func (g *Garden) collect(sp *Species, at time.Time) {
	if g.Herbarium == nil {
		g.Herbarium = map[string]time.Time{}
	}
	if _, seen := g.Herbarium[sp.ID]; seen {
		return
	}
	g.Herbarium[sp.ID] = at
	g.Log(at, "Pressed %s (%s) into the herbarium — %d of %d.",
		sp.Common, sp.Latin, len(g.Herbarium), len(AllSpecies()))
}

// sight records a creature's first visit to the garden.
func (g *Garden) sight(name, note string, at time.Time) bool {
	if g.Sightings == nil {
		g.Sightings = map[string]time.Time{}
	}
	if _, seen := g.Sightings[name]; seen {
		return false
	}
	g.Sightings[name] = at
	g.Log(at, "%s", note)
	return true
}

// Collected reports whether a species has ever flowered in this garden.
func (g *Garden) Collected(sp *Species) (time.Time, bool) {
	at, ok := g.Herbarium[sp.ID]
	return at, ok
}

// Log appends a journal entry, keeping the log to a sane length.
func (g *Garden) Log(at time.Time, format string, args ...any) {
	g.Journal = append(g.Journal, JournalEntry{At: at, Text: fmt.Sprintf(format, args...)})
	if len(g.Journal) > 300 {
		g.Journal = g.Journal[len(g.Journal)-300:]
	}
}

// Level is the gardener's standing, derived from plants brought to maturity.
func (g *Garden) Level() int { return g.Matured }

// Weather is today's sky over this particular garden.
func (g *Garden) Weather(now time.Time) Weather { return WeatherFor(g.Seed, now) }

// Season is the season the garden currently sits in.
func (g *Garden) Season(now time.Time) Season { return SeasonOf(now) }

// Unlocked reports whether the seed shop stocks a species yet.
func (g *Garden) Unlocked(sp *Species) bool { return g.Matured >= sp.Unlock }

// Advance brings the garden up to date with the wall clock. Growth continues
// while the program is closed, so the simulation steps through the elapsed
// time in small increments using each day's deterministic weather.
func (g *Garden) Advance(now time.Time) {
	if g.LastTick.IsZero() {
		g.LastTick = now
		return
	}
	elapsed := now.Sub(g.LastTick)
	if elapsed <= 0 {
		return
	}
	if max := time.Duration(maxCatchUpDays * 24 * float64(time.Hour)); elapsed > max {
		elapsed = max
		g.LastTick = now.Add(-max)
	}

	step := time.Duration(simStepHours * float64(time.Hour))
	t := g.LastTick
	for t.Before(now) {
		next := t.Add(step)
		if next.After(now) {
			next = now
		}
		dt := next.Sub(t).Hours()
		g.stepAll(t, dt, now)
		t = next
	}
	g.LastTick = now
	// The suggestions are settled once per catch-up rather than once per
	// step: nobody finishes one inside a quarter of an hour.
	g.checkTasks(now)
}

func (g *Garden) stepAll(at time.Time, dt float64, now time.Time) {
	w := WeatherFor(g.Seed, at)
	season := SeasonOf(at)
	for i := range g.Plots {
		g.step(&g.Plots[i], i, w, season, dt, at, now)
	}
}

func (g *Garden) step(p *Plot, idx int, w Weather, season Season, dt float64, at, now time.Time) {
	if p.Pond {
		p.Moisture = 1
		p.Weeds = 0
	} else {
		// Weeds creep in everywhere, planted or not, and rest over winter.
		weedRate := baseWeedPerHour
		if season == Winter {
			weedRate *= 0.45
		}
		p.Weeds = clamp01(p.Weeds + weedRate*dt)
	}

	if !p.Pond {
		p.Moisture = clamp01(p.Moisture - baseDryPerHour*w.Dryness*dt + w.Rainfall*dt)
	}
	if p.Empty() {
		return
	}

	sp := p.Species()
	if sp == nil {
		return
	}

	if p.Growth >= 1.0 {
		if !p.Spent {
			p.Pods = math.Min(maxPods, p.Pods+podsPerHour*dt*seasonPodFactor(season))
			g.maybeGoToSeed(p, idx, sp, at, now)
		}
		// A plant in seed scatters some of it about, whether it is still
		// flowering or standing dry.
		g.maybeSelfSeed(p, idx, sp, season, at, dt, now)
		return
	}

	feed(p, dt)
	rate := (1.0 / sp.Hours) * growthFactor(p, sp, w, season) * g.companionFactor(idx, sp)
	p.Growth = math.Min(1.0, p.Growth+rate*dt)
	if p.Growth >= 1.0 && !p.Matured {
		p.Matured = true
		p.MaturedAt = at
		g.Matured++
		g.collect(sp, at)
		// Journal the moment, dated when it actually happened.
		when := at
		if when.After(now) {
			when = now
		}
		g.Log(when, "%s (%s) reached full bloom.", p.DisplayName(), sp.Common)
	}
}

func seasonPodFactor(s Season) float64 {
	if s == Winter {
		return 0.6
	}
	return 1.0
}

// growthFactor combines soil, weeds, season and sky into a single multiplier.
// It never reaches zero: a forgotten plant slows down, it does not die.
func growthFactor(p *Plot, sp *Species, w Weather, season Season) float64 {
	moisture := 0.35 + 0.65*math.Min(1, p.Moisture/0.6)
	weeds := 1.0 - 0.40*p.Weeds
	seasonal := 0.70
	if sp.LikesSeason(season) {
		seasonal = 1.0
	}
	return moisture * weeds * seasonal * w.Growth * soilFactor(sp, p)
}

// Plant sows a species into a bed, charging its seed cost.
func (g *Garden) Plant(idx int, sp *Species, now time.Time) error {
	if idx < 0 || idx >= len(g.Plots) {
		return fmt.Errorf("no such bed")
	}
	p := &g.Plots[idx]
	if !p.Empty() {
		return fmt.Errorf("bed %d already holds %s", idx+1, p.DisplayName())
	}
	if !g.Unlocked(sp) {
		return fmt.Errorf("%s needs %d matured plants to unlock", sp.Common, sp.Unlock)
	}
	if sp.Kind == KindAquatic && !p.Pond {
		return fmt.Errorf("%s needs a pond — dig one with d", sp.Common)
	}
	if sp.Kind != KindAquatic && p.Pond {
		return fmt.Errorf("bed %d is a pond; only water plants will grow there", idx+1)
	}
	if g.Seeds < sp.SeedCost {
		return fmt.Errorf("not enough seeds for %s (costs %d)", sp.Common, sp.SeedCost)
	}
	g.Seeds -= sp.SeedCost
	g.Planted++
	*p = Plot{
		SpeciesID: sp.ID,
		PlantedAt: now,
		Moisture:  0.65, // a watering-in, as any gardener would
		Weeds:     0,
		Pond:      p.Pond,
		PH:        p.PH,
		Richness:  p.Richness,
	}
	g.Log(now, "Sowed %s (%s) in bed %d.", sp.Common, sp.Latin, idx+1)
	return nil
}

// Water fills a bed's soil. Returns false when there was nothing to do.
func (g *Garden) Water(idx int, now time.Time) bool {
	p := &g.Plots[idx]
	if p.Empty() || p.Pond || p.Moisture > 0.92 {
		return false
	}
	p.Moisture = 1.0
	return true
}

// WaterAll waters every planted bed that wants it, returning the count.
func (g *Garden) WaterAll(now time.Time) int {
	n := 0
	for i := range g.Plots {
		if g.Water(i, now) {
			n++
		}
	}
	if n > 0 {
		g.Log(now, "Walked the rows with the watering can (%d beds).", n)
	}
	return n
}

// Weed clears a bed. Clearing a properly overgrown bed earns a seed.
func (g *Garden) Weed(idx int, now time.Time) (bool, int) {
	p := &g.Plots[idx]
	if p.Weeds < 0.05 {
		return false, 0
	}
	reward := 0
	if p.Weeds > 0.45 {
		reward = weedingReward
		g.Seeds += reward
	}
	p.Weeds = 0
	return true, reward
}

// Gather collects ripe seed pods from a mature plant.
func (g *Garden) Gather(idx int, now time.Time) int {
	p := &g.Plots[idx]
	if p.Empty() || p.Pods < 1 {
		return 0
	}
	got := int(p.Pods)
	p.Pods -= float64(got)
	g.Seeds += got
	g.Gathered += got
	g.Log(now, "Gathered %d seed(s) from %s.", got, p.DisplayName())
	return got
}

// Uproot clears a bed, composting whatever was growing there.
func (g *Garden) Uproot(idx int, now time.Time) bool {
	p := &g.Plots[idx]
	if p.Empty() {
		return false
	}
	name := p.DisplayName()
	// The lifted plant goes back into the ground it came from: the bed gets
	// richer, and its pH drifts towards neutral as compost buffers it.
	richness := math.Min(1, p.Richness+0.18)
	ph := p.PH + (6.5-p.PH)*0.12
	*p = Plot{Moisture: p.Moisture, Weeds: p.Weeds, Pond: p.Pond, PH: ph, Richness: richness}
	g.Log(now, "Lifted %s and composted it into bed %d.", name, idx+1)
	return true
}

// Rename gives a plant the gardener's own name for it.
func (g *Garden) Rename(idx int, name string, now time.Time) bool {
	p := &g.Plots[idx]
	if p.Empty() {
		return false
	}
	old := p.DisplayName()
	p.Name = name
	if name == "" {
		g.Log(now, "%s goes back to being simply a %s.", old, p.Species().Common)
	} else {
		g.Log(now, "Named the %s in bed %d \"%s\".", p.Species().Common, idx+1, name)
	}
	return true
}

// Visit records a new day's first visit, handing out the daily seed bonus.
func (g *Garden) Visit(now time.Time) int {
	defer func() { g.LastVisit = now }()
	if sameDay(g.LastVisit, now) {
		return 0
	}
	g.Seeds += dailyBonus
	if g.LastVisit.IsZero() {
		return dailyBonus
	}
	g.Log(now, "A new day in the garden: %d seeds from the shed.", dailyBonus)
	return dailyBonus
}

func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// hashUnit turns a few numbers into a repeatable value between 0 and 1. The
// garden uses it wherever it wants variety that survives being reloaded, or
// replayed hour by hour after the program has been closed.
func hashUnit(seed, a, salt int64) float64 {
	h := fnv.New64a()
	var buf [24]byte
	putInt(buf[0:8], seed)
	putInt(buf[8:16], a)
	putInt(buf[16:24], salt)
	_, _ = h.Write(buf[:])
	return float64(h.Sum64()%1_000_003) / 1_000_003
}

// maybeGoToSeed finishes an annual or biennial that has had its season. The
// plant does not die: it dries, hands over a last few seeds, and stands until
// the gardener lifts it.
func (g *Garden) maybeGoToSeed(p *Plot, idx int, sp *Species, at, now time.Time) {
	span := seedSpan(sp)
	if span <= 0 || p.MaturedAt.IsZero() {
		return
	}
	if at.Sub(p.MaturedAt).Hours() < span {
		return
	}
	p.Spent = true
	p.Pods = math.Min(maxPods, p.Pods+2)
	when := at
	if when.After(now) {
		when = now
	}
	g.Log(when, "%s has gone to seed in bed %d.", p.DisplayName(), idx+1)
}

// Self-seeding. A cottage garden fills its own gaps: poppies, cosmos and
// foxgloves drop seed where they stand and come up in whatever bare ground
// they can find.
const (
	quarterSeconds = 900
	volunteerOdds  = 0.02 // per quarter hour, for a seeding plant with a gap beside it
)

// maybeSelfSeed drops a volunteer seedling into a neighbouring bed.
//
// The decision is taken only on quarter-hour boundaries, and from a hash of
// the garden's seed rather than a random number generator. That matters: a
// garden left running and one catching up on a fortnight it spent closed must
// arrive at exactly the same plants.
func (g *Garden) maybeSelfSeed(p *Plot, idx int, sp *Species, season Season, at time.Time, dt float64, now time.Time) {
	if !sp.SelfSeeds() || p.Pods < 1 || !sp.LikesSeason(season) {
		return
	}

	start := at.Unix() / quarterSeconds
	end := at.Add(time.Duration(dt*float64(time.Hour))).Unix() / quarterSeconds
	for q := start + 1; q <= end; q++ {
		if hashUnit(g.Seed, q, int64(idx)+0x5EED) > volunteerOdds {
			continue
		}
		// Bare ground only, and never a pond.
		var gaps []int
		for _, n := range g.Neighbours(idx) {
			if g.Plots[n].Empty() && !g.Plots[n].Pond {
				gaps = append(gaps, n)
			}
		}
		if len(gaps) == 0 {
			return
		}
		target := gaps[int(hashUnit(g.Seed, q, int64(idx)+0xB1AD)*float64(len(gaps)))%len(gaps)]

		when := at
		if when.After(now) {
			when = now
		}
		bed := &g.Plots[target]
		*bed = Plot{
			SpeciesID: sp.ID,
			PlantedAt: when,
			Moisture:  bed.Moisture,
			PH:        bed.PH,
			Richness:  bed.Richness,
		}
		p.Pods--
		g.Planted++
		g.Volunteers++
		g.Log(when, "A %s has sown itself into bed %d.", sp.Common, target+1)
		return
	}
}
