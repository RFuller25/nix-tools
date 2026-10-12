package main

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The gnomes themselves. A handful of them wander along the front of the
// stall, stop to chat, throw sparks about and hop for joy whenever you buy
// something. It is scenery: it reads and changes nothing in the garden, and
// runs on the same fast tick as the wind.

const (
	gnomeWidth     = 7 // cells a gnome sprite takes
	gnomeMinCrowd  = 2
	gnomeMaxCrowd  = 6
	gnomePerCells  = 20  // one gnome for this many columns of window
	gnomeWalkLow   = 3.0 // cells a second
	gnomeWalkHigh  = 7.0
	gnomeStopOdds  = 0.30 // chance a second that a walking gnome stops
	gnomeSparkLife = 0.9  // seconds a spark lasts
)

// gnomeHats are the 256-colour codes the gnomes' hats come in.
var gnomeHats = []string{"196", "33", "34", "208", "129", "220"}

type wanderer struct {
	x, dir float64 // left edge, and -1 or +1
	speed  float64
	phase  float64 // walking cycle, in steps
	pause  float64 // seconds left standing about
	cast   float64 // seconds left throwing sparks
	hop    float64 // seconds left in the air
	hat    string
}

type spark struct {
	x, dx float64
	life  float64
	glyph rune
	color string
}

type gnomeCrowd struct {
	rng     *rand.Rand
	gnomes  []wanderer
	sparks  []spark
	clipped int // width the crowd was last laid out for
}

var sparkGlyphs = []rune{'✧', '✦', '·', '*', '✺'}
var sparkColors = []string{"220", "213", "51", "156", "159"}

func (c *gnomeCrowd) ensure(width int, seed int64) {
	if c.rng == nil {
		c.rng = rand.New(rand.NewSource(seed))
	}
	want := max(gnomeMinCrowd, min(gnomeMaxCrowd, width/gnomePerCells))
	room := float64(max(1, width-gnomeWidth))
	for len(c.gnomes) < want {
		i := len(c.gnomes)
		dir := 1.0
		if c.rng.Intn(2) == 0 {
			dir = -1
		}
		c.gnomes = append(c.gnomes, wanderer{
			x: c.rng.Float64() * room, dir: dir,
			speed: gnomeWalkLow + c.rng.Float64()*(gnomeWalkHigh-gnomeWalkLow),
			hat:   gnomeHats[i%len(gnomeHats)],
		})
	}
	if len(c.gnomes) > want {
		c.gnomes = c.gnomes[:want]
	}
	for i := range c.gnomes {
		c.gnomes[i].x = math.Min(c.gnomes[i].x, room)
	}
	c.clipped = width
}

// advance moves everyone on by dt seconds in a window of the given width.
func (c *gnomeCrowd) advance(dt float64, width int, seed int64) {
	c.ensure(width, seed)
	room := float64(max(1, width-gnomeWidth))
	for i := range c.gnomes {
		g := &c.gnomes[i]
		g.hop = math.Max(0, g.hop-dt)
		g.cast = math.Max(0, g.cast-dt)
		if g.cast > 0 && c.rng.Float64() < dt*9 {
			c.throw(g)
		}
		if g.pause > 0 {
			g.pause -= dt
			continue
		}
		// Gnomes are polite: one about to walk into another turns round.
		for j := range c.gnomes {
			if o := &c.gnomes[j]; j != i && (o.x-g.x)*g.dir > 0 && math.Abs(o.x-g.x) < gnomeWidth+1 {
				g.dir = -g.dir
				break
			}
		}
		g.x += g.dir * g.speed * dt
		g.phase += g.speed * dt * 0.7
		switch {
		case g.x <= 0:
			g.x, g.dir = 0, 1
		case g.x >= room:
			g.x, g.dir = room, -1
		}
		if c.rng.Float64() < dt*gnomeStopOdds {
			g.pause = 0.8 + c.rng.Float64()*1.8
			if c.rng.Float64() < 0.5 {
				g.cast = g.pause
			} else if c.rng.Float64() < 0.4 {
				g.dir = -g.dir // turns to look the other way
			}
		}
	}
	live := c.sparks[:0]
	for _, s := range c.sparks {
		s.life -= dt
		s.x += s.dx * dt
		if s.life > 0 {
			live = append(live, s)
		}
	}
	c.sparks = live
}

// throw sends a spark up from a gnome's wand.
func (c *gnomeCrowd) throw(g *wanderer) {
	if len(c.sparks) > 40 {
		return
	}
	c.sparks = append(c.sparks, spark{
		x: g.x + float64(gnomeWidth)/2 + (c.rng.Float64()-0.5)*5, dx: (c.rng.Float64() - 0.5) * 3,
		life:  gnomeSparkLife * (0.5 + c.rng.Float64()/2),
		glyph: sparkGlyphs[c.rng.Intn(len(sparkGlyphs))], color: sparkColors[c.rng.Intn(len(sparkColors))],
	})
}

// celebrate has every gnome stop, hop and throw sparks: something was bought.
func (c *gnomeCrowd) celebrate() {
	if c.rng == nil {
		return
	}
	for i := range c.gnomes {
		g := &c.gnomes[i]
		g.pause, g.cast, g.hop = 1.4, 1.4, 0.5+c.rng.Float64()*0.5
		for k := 0; k < 3; k++ {
			c.throw(g)
		}
	}
}

// cell is one character of the scene and its colour.
type cell struct {
	r   rune
	col string
}

// sceneRows is how many lines the gnomes get, for the room there is: none in
// a cramped window, a compact two-line gnome with a line of sparks above in a
// middling one, and full figures standing on grass in a roomy one.
func sceneRows(avail, width int) int {
	switch {
	case width < 2*gnomeWidth || avail < 13:
		return 0
	case avail < 24:
		return 3
	}
	return 6
}

// scene draws the gnomes into rows lines of width cells.
func (c *gnomeCrowd) scene(width, rows int) []string {
	if rows == 0 || width < 1 {
		return nil
	}
	grid := make([][]cell, rows)
	for r := range grid {
		grid[r] = make([]cell, width)
	}
	put := func(row, col int, r rune, color string) {
		if row >= 0 && row < rows && col >= 0 && col < width && r != ' ' {
			grid[row][col] = cell{r, color}
		}
	}
	text := func(row, col int, s, color string) {
		for i, r := range []rune(s) {
			put(row, col+i, r, color)
		}
	}
	full := rows >= 6
	if full {
		for x := 0; x < width; x++ {
			switch {
			case x%13 == 6:
				put(rows-1, x, '♣', "160")
			case x%5 == 2:
				put(rows-1, x, '"', "28")
			default:
				put(rows-1, x, '.', "22")
			}
		}
	}
	for _, s := range c.sparks {
		put(0, int(math.Round(s.x)), s.glyph, s.color)
	}
	feet := [...]string{"/ \\", "| |", "\\ /", "| |"}
	for _, g := range c.gnomes {
		x := int(math.Round(g.x))
		lift := 0
		if g.hop > 0 {
			lift = 1
		}
		step := int(g.phase) % len(feet)
		arms := (g.pause <= 0 && int(g.phase)%2 == 1) || g.cast > 0
		mouth := "(•‿•)"
		if g.cast > 0 {
			mouth = "(•o•)"
		}
		arm1, arm2 := '/', '\\'
		if arms {
			arm1, arm2 = '\\', '/'
		}
		hat, face := 1-lift, 2-lift
		put(hat, x+3, '▲', g.hat)
		put(face, x, arm1, g.hat)
		text(face, x+1, mouth, "223")
		put(face, x+6, arm2, g.hat)
		if full {
			put(face+1, x+3, '▼', "255") // the beard
			text(face+2, x+2, feet[step], "94")
		}
	}
	out := make([]string, rows)
	for r := range grid {
		var b strings.Builder
		for x := 0; x < width; {
			cur := grid[r][x]
			run := []rune{}
			for x < width && grid[r][x].col == cur.col {
				ch := grid[r][x].r
				if ch == 0 {
					ch = ' '
				}
				run = append(run, ch)
				x++
			}
			if cur.col == "" {
				b.WriteString(string(run))
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(cur.col)).Render(string(run)))
			}
		}
		out[r] = b.String()
	}
	return out
}
