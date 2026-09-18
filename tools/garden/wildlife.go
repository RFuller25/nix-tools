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
}

var creatures = map[creature]creatureKind{
	bee:       {"bee", "✽", "220", 1.1, 9, "A bee found the garden."},
	butterfly: {"butterfly", "❖", "213", 0.7, 12, "A butterfly is working the flowers."},
	finch:     {"finch", "ᐤ", "179", 1.6, 7, "A finch came down for the seed heads."},
	moth:      {"moth", "✺", "189", 0.9, 10, "A moth is out among the night flowers."},
	dragonfly: {"dragonfly", "⋈", "80", 2.0, 8, "A dragonfly is patrolling the pond."},
	fox:       {"fox", "ᗢ", "173", 0.5, 14, "A fox crossed the garden."},
	hedgehog:  {"hedgehog", "ᴥ", "138", 0.35, 16, "A hedgehog is snuffling through the beds."},
}

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
