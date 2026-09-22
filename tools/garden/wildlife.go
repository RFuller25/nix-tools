package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// A garden is only half plants. Bees work the flowers on warm afternoons,
// butterflies follow the nectar, finches take the seed heads in autumn, moths
// come to the night-scented things after dark, and a fox crosses the beds when
// nobody is watching.
//
// What turns up depends entirely on what is growing: plant nothing and nothing
// visits.

type creature int

const (
	bee creature = iota
	butterfly
	finch
	moth
	dragonfly
	fox
	hedgehog
)

type creatureKind struct {
	name  string
	glyph string
	color string
	speed float64 // beds crossed per second
	life  float64 // seconds before it moves on
	note  string  // what the journal says the first time

	// For the almanac: what brings it, when it comes, and something true
	// about it worth knowing.
	comes string
	when  string
	fact  string
}

var creatures = map[creature]creatureKind{
	bee: {
		name: "bee", glyph: "✽", color: "220", speed: 1.1, life: 9,
		note:  "A bee found the garden.",
		comes: "any plant in open flower, except the ferns, grasses and carnivores",
		when:  "daylight, and not while it is raining",
		fact:  "A honeybee visits a few hundred flowers in one trip and tells the hive where they were by dancing the direction and distance on the comb.",
	},
	butterfly: {
		name: "butterfly", glyph: "❖", color: "213", speed: 0.7, life: 12,
		note:  "A butterfly is working the flowers.",
		comes: "daisies, mints, carrot-family umbels, honeysuckles and the climbers",
		when:  "bright weather only — sun, clear skies or a heatwave",
		fact:  "Butterflies taste with their feet: standing on a leaf is how a female knows whether it is the right plant to lay on.",
	},
	finch: {
		name: "finch", glyph: "ᐤ", color: "179", speed: 1.6, life: 7,
		note:  "A finch came down for the seed heads.",
		comes: "daisy-family seed heads left standing — sunflower, coneflower, cosmos and their kin, once a pod has ripened",
		when:  "daylight, any weather",
		fact:  "This is why the almanac keeps telling you to leave the dead heads up: a standing sunflower feeds goldfinches all autumn.",
	},
	moth: {
		name: "moth", glyph: "✺", color: "189", speed: 0.9, life: 10,
		note:  "A moth is out among the night flowers.",
		comes: "night-scented and night-opening flowers — moonflower, evening primrose, night-scented stock, honeysuckle, lily of the valley",
		when:  "after dark",
		fact:  "Night-flowering plants are pale and heavily scented because a moth finds them by smell and by what little light there is, not by colour.",
	},
	dragonfly: {
		name: "dragonfly", glyph: "⋈", color: "80", speed: 2.0, life: 8,
		comes: "a pond — dig one with d",
		note:  "A dragonfly is patrolling the pond.",
		when:  "daylight, and not in the wet",
		fact:  "A dragonfly spends most of its life underwater as a nymph; the flying adult you see may last only a few weeks.",
	},
	fox: {
		name: "fox", glyph: "ᗢ", color: "173", speed: 0.5, life: 14,
		note:  "A fox crossed the garden.",
		comes: "nothing in particular — it is passing through",
		when:  "after dark",
		fact:  "Urban foxes hunt by sound more than sight, and can hear a worm moving under the soil.",
	},
	hedgehog: {
		name: "hedgehog", glyph: "ᴥ", color: "138", speed: 0.35, life: 16,
		note:  "A hedgehog is snuffling through the beds.",
		comes: "an untidy garden — it wants a bed or two left weedy",
		when:  "after dark",
		fact:  "A hedgehog covers a mile or two a night eating slugs and beetles, which makes a weedy corner the most useful thing in a vegetable garden.",
	},
}

// creatureOrder is the order the almanac lists them in: the ones you are
// likeliest to see first.
var creatureOrder = []creature{bee, butterfly, finch, dragonfly, moth, hedgehog, fox}

func (c creature) kind() creatureKind { return creatures[c] }
func (c creature) String() string     { return creatures[c].name }

// visitor is one creature making its way across the garden.
type visitor struct {
	what    creature
	x, y    float64 // where it is, in beds
	dx      float64 // how fast it crosses
	bob     float64 // where it is in its up-and-down
	bobRate float64
	age     float64
	life    float64
}

type wildlife struct {
	visitors []visitor
	rng      *rand.Rand
}

func newWildlife(seed int64) wildlife {
	return wildlife{rng: rand.New(rand.NewSource(seed))}
}

// advance moves everything on by dt seconds and decides what turns up next.
func (w *wildlife) advance(dt float64, g *Garden, now time.Time, weather Weather, log func(creature)) {
	if w.rng == nil {
		w.rng = rand.New(rand.NewSource(now.UnixNano()))
	}

	live := w.visitors[:0]
	for _, v := range w.visitors {
		v.x += v.dx * dt
		v.bob += v.bobRate * dt
		v.age += dt
		if v.age < v.life && v.x > -2 && v.x < plotCols+2 {
			live = append(live, v)
		}
	}
	w.visitors = live

	if len(w.visitors) >= 3 {
		return
	}
	for _, c := range w.candidates(g, now, weather) {
		if w.rng.Float64() > c.chance {
			continue
		}
		w.arrive(c.what, c.row)
		if log != nil {
			log(c.what)
		}
		return
	}
}

// arrive sends a creature in from one side, aimed along a row of beds.
func (w *wildlife) arrive(what creature, row int) {
	k := what.kind()
	dx := k.speed * (0.7 + 0.6*w.rng.Float64())
	x := -1.0
	if w.rng.Float64() < 0.5 {
		x, dx = float64(plotCols)+1, -dx
	}
	w.visitors = append(w.visitors, visitor{
		what:    what,
		x:       x,
		y:       float64(row),
		dx:      dx,
		bob:     w.rng.Float64(),
		bobRate: 0.6 + w.rng.Float64(),
		life:    k.life,
	})
}

// chanceOf is one creature that might turn up, and how likely it is per frame.
type chanceOf struct {
	what   creature
	chance float64
	row    int
}

// candidates works out what could plausibly visit right now, given the time of
// day, the weather, and above all what is in flower.
func (w *wildlife) candidates(g *Garden, now time.Time, weather Weather) []chanceOf {
	ph := phaseAt(now)
	season := g.Season(now)
	wet := weather.Kind == Rain || weather.Kind == Storm || weather.Kind == Snow

	var out []chanceOf
	add := func(what creature, chance float64, row int) {
		out = append(out, chanceOf{what, chance, row})
	}

	flowering, nectar, seedHeads, night := -1, -1, -1, -1
	ponds := -1
	for i := range g.Plots {
		p := &g.Plots[i]
		if p.Pond {
			ponds = i / plotCols
		}
		sp := p.Species()
		if sp == nil || p.Growth < 1 || p.Spent || dormant(sp, season) {
			continue
		}
		row := i / plotCols
		if attractsBees(sp) {
			flowering = row
		}
		if attractsButterflies(sp) {
			nectar = row
		}
		if p.Pods >= 1 && sp.Family == "Asteraceae" {
			seedHeads = row
		}
		if sp.NightScented() || sp.OpensAtNight() {
			night = row
		}
	}

	switch {
	case ph.Dark():
		if night >= 0 {
			add(moth, 0.05, night)
		}
		add(fox, 0.004, w.rng.Intn(max(1, g.Rows())))
		if anyWeeds(g) {
			add(hedgehog, 0.006, w.rng.Intn(max(1, g.Rows())))
		}
	default:
		if !wet && flowering >= 0 {
			add(bee, 0.05, flowering)
		}
		if !wet && nectar >= 0 && (weather.Kind == Sunny || weather.Kind == Clear || weather.Kind == Heatwave) {
			add(butterfly, 0.035, nectar)
		}
		if seedHeads >= 0 {
			add(finch, 0.02, seedHeads)
		}
		if ponds >= 0 && !wet {
			add(dragonfly, 0.02, ponds)
		}
	}
	return out
}

// attractsBees is true of anything with an open flower a bee can work.
func attractsBees(sp *Species) bool {
	switch sp.Kind {
	case KindFern, KindGrass, KindCarnivore:
		return false
	default:
		return true
	}
}

// attractsButterflies is narrower: butterflies want nectar they can reach
// from a landing pad, which is the daisies, the mints and the buddleia-like
// shrubs and climbers.
func attractsButterflies(sp *Species) bool {
	switch sp.Family {
	case "Asteraceae", "Lamiaceae", "Caprifoliaceae", "Verbenaceae", "Apiaceae":
		return true
	}
	return sp.Kind == KindVine
}

func anyWeeds(g *Garden) bool {
	for i := range g.Plots {
		if g.Plots[i].Weeds > 0.4 {
			return true
		}
	}
	return false
}

// present reports what is in the garden right now, for the status line.
func (w *wildlife) present() (creature, bool) {
	if len(w.visitors) == 0 {
		return 0, false
	}
	return w.visitors[0].what, true
}

// overlayFor gives the glyphs to draw over one bed's artwork.
func (w *wildlife) overlayFor(bed, width, height int) map[[2]int]string {
	if len(w.visitors) == 0 {
		return nil
	}
	col, row := bed%plotCols, bed/plotCols

	var out map[[2]int]string
	for _, v := range w.visitors {
		if int(math.Floor(v.x)) != col || int(v.y) != row {
			continue
		}
		k := v.what.kind()
		// Where it sits inside the bed: across from its position, and up and
		// down on its own little bob.
		cx := int((v.x - math.Floor(v.x)) * float64(width))
		cy := int((0.5 + 0.45*math.Sin(2*math.Pi*v.bob)) * float64(height-1))
		if cx < 0 || cx >= width || cy < 0 || cy >= height {
			continue
		}
		if out == nil {
			out = map[[2]int]string{}
		}
		out[[2]int{cy, cx}] = lipgloss.NewStyle().
			Foreground(lipgloss.Color(k.color)).
			Render(k.glyph)
	}
	return out
}
