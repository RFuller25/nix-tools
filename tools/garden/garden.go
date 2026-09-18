package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// PlotCount is how many beds the garden holds.
const PlotCount = 15

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
}

// Empty reports whether anything is planted here.
func (p *Plot) Empty() bool { return p.SpeciesID == "" }

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
		return "bare soil"
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
	Version   int            `json:"version"`
	Seed      int64          `json:"seed"`
	Gardener  string         `json:"gardener,omitempty"`
	Plots     []Plot         `json:"plots"`
	Seeds     int            `json:"seeds"`
	Matured   int            `json:"matured"`  // lifetime plants brought to maturity
	Planted   int            `json:"planted"`  // lifetime plants sown
	Gathered  int            `json:"gathered"` // lifetime seeds gathered
	Music     bool           `json:"music"`    // was the music playing when we last closed
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
	g.Log(now, "A patch of bare earth. Something could grow here.")
	return g
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
}

func (g *Garden) stepAll(at time.Time, dt float64, now time.Time) {
	w := WeatherFor(g.Seed, at)
	season := SeasonOf(at)
	for i := range g.Plots {
		g.step(&g.Plots[i], i, w, season, dt, at, now)
	}
}

func (g *Garden) step(p *Plot, idx int, w Weather, season Season, dt float64, at, now time.Time) {
	// Weeds creep in everywhere, planted or not, and rest over winter.
	weedRate := baseWeedPerHour
	if season == Winter {
		weedRate *= 0.45
	}
	p.Weeds = clamp01(p.Weeds + weedRate*dt)

	if p.Empty() {
		p.Moisture = clamp01(p.Moisture - baseDryPerHour*w.Dryness*dt + w.Rainfall*dt)
		return
	}

	p.Moisture = clamp01(p.Moisture - baseDryPerHour*w.Dryness*dt + w.Rainfall*dt)

	sp := p.Species()
	if sp == nil {
		return
	}

	if p.Growth >= 1.0 {
		p.Pods = math.Min(maxPods, p.Pods+podsPerHour*dt*seasonPodFactor(season))
		return
	}

	rate := (1.0 / sp.Hours) * growthFactor(p, sp, w, season)
	p.Growth = math.Min(1.0, p.Growth+rate*dt)
	if p.Growth >= 1.0 && !p.Matured {
		p.Matured = true
		g.Matured++
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
	return moisture * weeds * seasonal * w.Growth
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
	}
	g.Log(now, "Sowed %s (%s) in bed %d.", sp.Common, sp.Latin, idx+1)
	return nil
}

// Water fills a bed's soil. Returns false when there was nothing to do.
func (g *Garden) Water(idx int, now time.Time) bool {
	p := &g.Plots[idx]
	if p.Empty() || p.Moisture > 0.92 {
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
	*p = Plot{Moisture: p.Moisture, Weeds: p.Weeds}
	g.Log(now, "Lifted %s and turned the soil in bed %d.", name, idx+1)
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
