package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// The gnome shop. A family of gnomes keeps a stall at the bottom of the garden
// and sells magic rather than seed: the sky changed to order, a plant nudged
// into a mutation, and one wild, brewed-up seed a day. Every service costs
// gold, and none of them can be had for free.
//
// Everything the gnomes do is deterministic. A roll is a hash of the garden's
// seed and a running count of spells cast, never a random number, so the same
// garden always gets the same result from the same spell.

const (
	gnomeWeatherCost = 20  // change the day's weather
	gnomeMutateCost  = 10  // a random mutation in one plant
	gnomeChosenCost  = 100 // a mutation of your choosing in one plant
	gnomeSeedBase    = 40  // the wild seed: this, plus the multiple below of the species' shop price
	gnomeSeedMult    = 3
	gnomeNudge       = 35 // how far a chosen mutation moves a gene
	gnomeRolled      = 1  // a random mutation changes this many genes, and sometimes one more
	gnomeSpellDays   = 2  // spells older than this many days are forgotten
)

// GnomeService is one thing the gnomes sell.
type GnomeService int

const (
	ServiceWeather GnomeService = iota
	ServiceMutate
	ServiceChosen
	ServiceSeed
	serviceCount
)

// GnomeInfo is how a service is described in the shop.
type GnomeInfo struct {
	Name  string
	Gnome string // who runs the stall
	Blurb string
}

var gnomeInfos = [serviceCount]GnomeInfo{
	ServiceWeather: {"Change the weather", "Mistress Drizzlewort",
		"Mistress Drizzlewort will have the sky turned over for the rest of today. Rain waters every bed, a heatwave dries them, and a frost slows everything: pick the weather your garden is wanting. Tomorrow the sky goes back to the calendar's."},
	ServiceMutate: {"Random mutation", "Fennick Quickspore",
		"Fennick Quickspore taps a plant with a spoon and mutters. One gene, sometimes two, of the plant changes by a fair amount, up or down: it might come out taller, bluer, faster or all three, or the other way. You do not get to pick, and that is half the fun."},
	ServiceChosen: {"Chosen mutation", "Grandmaster Thistlewhistle",
		"Grandmaster Thistlewhistle does it properly. Name the change you want (taller, bushier, quicker, a particular colour) and the plant is pushed that way, every time. It costs a good deal more than a roll of the dice."},
	ServiceSeed: {"Today's wild seed", "Pip Mushling",
		"Pip Mushling brews one seed a day in a teapot. It is some species or other, with every gene shoved to a wild extreme, and it comes true when sown. There is only one a day, and it is gone when it is bought."},
}

// ---- weather ----------------------------------------------------------------

// allWeatherKinds lists every sky, in the order the shop offers them.
var allWeatherKinds = []WeatherKind{Sunny, Clear, Cloudy, Overcast, Showers, Rain, Storm, Fog, Frost, Snow, Heatwave}

// WeatherAt is the weather at a moment: what the gnomes have set for that day,
// otherwise whatever the calendar and the garden's seed say.
func (g *Garden) WeatherAt(at time.Time) Weather {
	if k, ok := g.GnomeSpells[dayKey(at)]; ok {
		return weatherFor(k)
	}
	return WeatherFor(g.Seed, at)
}

// Spelled reports whether the gnomes have turned the weather on a day.
func (g *Garden) Spelled(at time.Time) bool {
	_, ok := g.GnomeSpells[dayKey(at)]
	return ok
}

// ChangeWeather has the gnomes turn today's sky for the rest of the day.
func (g *Garden) ChangeWeather(kind WeatherKind, now time.Time) error {
	if kind < Sunny || kind > Heatwave {
		return fmt.Errorf("the gnomes do not know that weather")
	}
	if g.WeatherAt(now).Kind == kind {
		return fmt.Errorf("it is already %s", weatherFor(kind).Name())
	}
	if !g.spend(gnomeWeatherCost) {
		return fmt.Errorf("turning the weather costs %s", goldLabel(gnomeWeatherCost))
	}
	// Bring the garden up to now first, so the hours already gone keep the
	// sky they had and only the hours still to come are changed.
	g.Advance(now)
	today, cutoff := dayKey(now), dayKey(now.AddDate(0, 0, -gnomeSpellDays))
	if g.GnomeSpells == nil {
		g.GnomeSpells = map[int]WeatherKind{}
	}
	for day := range g.GnomeSpells {
		if day < cutoff {
			delete(g.GnomeSpells, day)
		}
	}
	g.GnomeSpells[today] = kind
	g.GnomeCasts++
	g.GnomeSpent += gnomeWeatherCost
	g.Log(now, "The gnomes turned the weather: %s for the rest of today.", weatherFor(kind).Name())
	return nil
}

// ---- mutations --------------------------------------------------------------

// Mutation is a change a gnome will make to a plant when asked.
type Mutation struct {
	ID    string
	Name  string
	Blurb string
	Apply func(Genome) Genome
}

func nudgeGene(v uint8, by int) uint8 { return clampGene(float64(int(v) + by)) }

func geneMutation(id, name, blurb string, trait Trait, by int) Mutation {
	return Mutation{ID: id, Name: name, Blurb: blurb, Apply: func(g Genome) Genome {
		switch trait {
		case TraitHeight:
			g.Height = nudgeGene(g.Height, by)
		case TraitShape:
			g.Shape = nudgeGene(g.Shape, by)
		case TraitSpeed:
			g.Speed = nudgeGene(g.Speed, by)
		case TraitYield:
			g.Yield = nudgeGene(g.Yield, by)
		}
		return g
	}}
}

// colourMutation sets the hue outright and keeps the colour rich enough to
// show, so "blue" comes out blue and not a grey with a blue name.
func colourMutation(id, name string, hue uint16) Mutation {
	return Mutation{ID: id, Name: name, Blurb: "turns the bloom " + strings.ToLower(name) + ", whatever it was", Apply: func(g Genome) Genome {
		g.Hue = hue
		g.Sat = uint8(math.Max(float64(g.Sat), 65))
		g.Light = uint8(math.Min(65, math.Max(35, float64(g.Light))))
		return g
	}}
}

// gnomeMutations is the menu of the chosen mutations.
var gnomeMutations = []Mutation{
	geneMutation("taller", "Taller", fmt.Sprintf("pushes height up by %d", gnomeNudge), TraitHeight, +gnomeNudge),
	geneMutation("shorter", "Shorter", fmt.Sprintf("pushes height down by %d", gnomeNudge), TraitHeight, -gnomeNudge),
	geneMutation("bushier", "Bushier", fmt.Sprintf("pushes shape towards full and bushy by %d", gnomeNudge), TraitShape, +gnomeNudge),
	geneMutation("slimmer", "Slimmer", fmt.Sprintf("pushes shape towards slim and upright by %d", gnomeNudge), TraitShape, -gnomeNudge),
	geneMutation("quicker", "Quicker", fmt.Sprintf("pushes speed up by %d", gnomeNudge), TraitSpeed, +gnomeNudge),
	geneMutation("slower", "Slower", fmt.Sprintf("pushes speed down by %d", gnomeNudge), TraitSpeed, -gnomeNudge),
	geneMutation("generous", "More generous", fmt.Sprintf("pushes yield up by %d", gnomeNudge), TraitYield, +gnomeNudge),
	geneMutation("sparing", "Sparer", fmt.Sprintf("pushes yield down by %d", gnomeNudge), TraitYield, -gnomeNudge),
	colourMutation("red", "Red", 0),
	colourMutation("orange", "Orange", 25),
	colourMutation("yellow", "Yellow", 55),
	colourMutation("green", "Green", 120),
	colourMutation("teal", "Teal", 175),
	colourMutation("blue", "Blue", 220),
	colourMutation("violet", "Violet", 270),
	colourMutation("magenta", "Magenta", 305),
	colourMutation("pink", "Pink", 335),
	{ID: "white", Name: "White", Blurb: "bleaches the bloom to a clean white", Apply: func(g Genome) Genome {
		g.Sat, g.Light = 6, 94
		return g
	}},
	{ID: "midnight", Name: "Midnight", Blurb: "darkens the bloom nearly to black, keeping its hue", Apply: func(g Genome) Genome {
		g.Light = 9
		g.Sat = uint8(math.Max(float64(g.Sat), 40))
		return g
	}},
}

// MutationByID finds a chosen mutation.
func MutationByID(id string) *Mutation {
	for i := range gnomeMutations {
		if gnomeMutations[i].ID == id {
			return &gnomeMutations[i]
		}
	}
	return nil
}

// geneNames are the seven genes a roll can land on.
var geneNames = [...]string{"hue", "saturation", "lightness", "height", "shape", "speed", "yield"}

func (g Genome) geneAt(i int) int {
	switch i {
	case 0:
		return int(g.Hue)
	case 1:
		return int(g.Sat)
	case 2:
		return int(g.Light)
	case 3:
		return int(g.Height)
	case 4:
		return int(g.Shape)
	case 5:
		return int(g.Speed)
	}
	return int(g.Yield)
}

func (g Genome) withGene(i, v int) Genome {
	switch i {
	case 0:
		g.Hue = uint16(((v % 360) + 360) % 360)
	case 1:
		g.Sat = clampGene(float64(v))
	case 2:
		g.Light = uint8(math.Max(3, math.Min(97, float64(v))))
	case 3:
		g.Height = clampGene(float64(v))
	case 4:
		g.Shape = clampGene(float64(v))
	case 5:
		g.Speed = clampGene(float64(v))
	default:
		g.Yield = clampGene(float64(v))
	}
	return g
}

// rollMutation is what a random mutation does to a genome. It returns the new
// genome and a few words on what moved. It is a pure function of the garden's
// seed and the spell's number.
func rollMutation(seed, serial int64, gn Genome) (Genome, string) {
	roll := func(salt int64) float64 { return hashUnit(seed, serial, 0x6A0E0000+salt) }
	n := gnomeRolled
	if roll(1) < 0.4 {
		n++
	}
	var said []string
	used := map[int]bool{}
	for k := 0; k < n; k++ {
		gene := int(roll(int64(10+k)) * float64(len(geneNames)))
		for used[gene] {
			gene = (gene + 1) % len(geneNames)
		}
		used[gene] = true
		low, high := mutationLow+5, mutationHigh+10
		if gene == 0 {
			low, high = hueMutateLow, hueMutateHigh
		}
		size := int(math.Round(lerp(low, high, roll(int64(20+k)))))
		if roll(int64(30+k)) < 0.5 {
			size = -size
		}
		before := gn.geneAt(gene)
		// A gene already near one end of its range goes the other way rather
		// than being squashed against the limit and not changing at all.
		if after := before + size; gene != 0 && (after < 3 || after > 97) {
			size = -size
		}
		gn = gn.withGene(gene, before+size)
		said = append(said, fmt.Sprintf("%s %d → %d", geneNames[gene], before, gn.geneAt(gene)))
	}
	return gn, strings.Join(said, ", ")
}

// mutate puts a new genome into a planted bed and tidies up after it: the
// plant no longer belongs to the line it was bred in, and if it is already in
// flower it may now be a hybrid the almanac has not seen.
func (g *Garden) mutate(idx int, gn Genome, how string, now time.Time) {
	p := &g.Plots[idx]
	sp := p.Species()
	p.Genome = gn
	p.Variety, _ = sp.NearestVariety(gn)
	p.Streak, p.Line, p.LineName = 0, 0, ""
	p.Descent = "mutated by the gnomes (" + how + ")"
	if p.Matured && p.Growth >= 1 {
		g.discover(p, sp, idx, now, now)
	}
	g.GnomeCasts++
}

func (g *Garden) mutable(idx int) (*Plot, error) {
	if idx < 0 || idx >= len(g.Plots) {
		return nil, fmt.Errorf("no such bed")
	}
	p := &g.Plots[idx]
	if p.Empty() || p.Species() == nil {
		return nil, fmt.Errorf("bed %d is empty: the gnomes need a plant to work on", idx+1)
	}
	return p, nil
}

// MutateRandom has the gnomes work a random mutation on the plant in a bed.
// It returns what moved, in words.
func (g *Garden) MutateRandom(idx int, now time.Time) (string, error) {
	p, err := g.mutable(idx)
	if err != nil {
		return "", err
	}
	if !g.spend(gnomeMutateCost) {
		return "", fmt.Errorf("a random mutation costs %s", goldLabel(gnomeMutateCost))
	}
	name := p.DisplayName()
	gn, said := rollMutation(g.Seed, int64(g.GnomeCasts), p.Genes())
	g.GnomeSpent += gnomeMutateCost
	g.mutate(idx, gn, said, now)
	g.Log(now, "The gnomes worked a random mutation on %s in bed %d: %s.", name, idx+1, said)
	return said, nil
}

// MutateChosen has the gnomes make a particular change to the plant in a bed.
func (g *Garden) MutateChosen(idx int, id string, now time.Time) (string, error) {
	mu := MutationByID(id)
	if mu == nil {
		return "", fmt.Errorf("the gnomes do not know that trick")
	}
	p, err := g.mutable(idx)
	if err != nil {
		return "", err
	}
	before := p.Genes()
	after := mu.Apply(before)
	if after == before {
		return "", fmt.Errorf("%s cannot go any further that way", p.DisplayName())
	}
	if !g.spend(gnomeChosenCost) {
		return "", fmt.Errorf("a chosen mutation costs %s", goldLabel(gnomeChosenCost))
	}
	name := p.DisplayName()
	g.GnomeSpent += gnomeChosenCost
	g.mutate(idx, after, strings.ToLower(mu.Name), now)
	g.Log(now, "The gnomes made %s in bed %d %s.", name, idx+1, strings.ToLower(mu.Name))
	return mu.Name, nil
}

// ---- the wild seed ----------------------------------------------------------

// WildSeed is the day's seed from the gnome teapot.
type WildSeed struct {
	Species *Species
	Genome  Genome
	Price   int
}

// wildPool is every species the teapot may brew: anything but a water plant,
// which would have nowhere to go without a pond.
func wildPool() []*Species {
	var out []*Species
	for _, sp := range AllSpecies() {
		if sp.Kind != KindAquatic {
			out = append(out, sp)
		}
	}
	return out
}

// wildGene is a gene thrown to one end of its range or the other.
func wildGene(seed, day, salt int64) uint8 {
	end := hashUnit(seed, day, salt)
	span := hashUnit(seed, day, salt+1)
	if end < 0.5 {
		return uint8(span * 12)
	}
	return uint8(100 - span*12)
}

// WildSeedFor is the seed on offer on a day. Like the weather and the orders,
// it is a pure function of the garden's seed and the date.
func (g *Garden) WildSeedFor(day int) WildSeed {
	pool := wildPool()
	sp := pool[int(hashUnit(g.Seed, int64(day), 0x91D0)*float64(len(pool)))%len(pool)]
	d := int64(day)
	gn := Genome{
		Hue:    uint16(hashUnit(g.Seed, d, 0x91D1) * 360),
		Sat:    uint8(60 + hashUnit(g.Seed, d, 0x91D2)*40),
		Light:  uint8(22 + hashUnit(g.Seed, d, 0x91D3)*52),
		Height: wildGene(g.Seed, d, 0x91D4),
		Shape:  wildGene(g.Seed, d, 0x91D6),
		Speed:  wildGene(g.Seed, d, 0x91D8),
		Yield:  wildGene(g.Seed, d, 0x91DA),
	}
	return WildSeed{Species: sp, Genome: gn, Price: gnomeSeedBase + gnomeSeedMult*sp.seedPrice()}
}

// WildSeedAvailable reports whether today's seed is still in the teapot.
func (g *Garden) WildSeedAvailable(now time.Time) bool { return g.GnomeSeedDay != dayKey(now) }

// BuyWildSeed buys today's seed into the shed.
func (g *Garden) BuyWildSeed(now time.Time) (WildSeed, error) {
	today := dayKey(now)
	if !g.WildSeedAvailable(now) {
		return WildSeed{}, fmt.Errorf("today's seed is gone: Pip brews another tomorrow")
	}
	w := g.WildSeedFor(today)
	if !g.spend(w.Price) {
		return WildSeed{}, fmt.Errorf("today's seed costs %s", goldLabel(w.Price))
	}
	variety, _ := w.Species.NearestVariety(w.Genome)
	g.AddPacket(Packet{
		SpeciesID: w.Species.ID, A: w.Genome, B: w.Genome, Count: 1, Pure: true, Variety: variety,
		Label: "Wild " + w.Species.Common, From: "the gnomes", Descent: "brewed wild by the gnomes",
	})
	g.GnomeSeedDay = today
	g.GnomeSpent += w.Price
	g.Log(now, "Bought a wild %s seed from the gnomes: %s, %s.", w.Species.Common, describeGenome(w.Genome), w.Genome.SpeedWord())
	return w, nil
}
