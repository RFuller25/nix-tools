package main

import (
	"fmt"
	"math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gridN = 4

type g2048 struct {
	grid  [gridN][gridN]int
	score int
	moves int

	won      bool // reached 2048 at least once
	over     bool
	recorded bool

	rng *rand.Rand
	snd sounder
}

func new2048(seed int64) *g2048 {
	return &g2048{rng: rand.New(rand.NewSource(seed)), snd: noSound{}}
}

func (g *g2048) SetAudio(s sounder) { g.snd = s }

func (g *g2048) ID() string         { return "2048" }
func (g *g2048) Name() string       { return "2048" }
func (g *g2048) Blurb() string      { return "slide the tiles, double the numbers" }
func (g *g2048) ScoreLabel() string { return "points" }

func (g *g2048) Start() tea.Cmd {
	g.grid = [gridN][gridN]int{}
	g.score, g.moves = 0, 0
	g.won, g.over, g.recorded = false, false, false
	g.addTile()
	g.addTile()
	return nil
}

// addTile drops a 2 (or occasionally a 4) into a free square.
func (g *g2048) addTile() bool {
	var free [][2]int
	for y := 0; y < gridN; y++ {
		for x := 0; x < gridN; x++ {
			if g.grid[y][x] == 0 {
				free = append(free, [2]int{x, y})
			}
		}
	}
	if len(free) == 0 {
		return false
	}
	cell := free[g.rng.Intn(len(free))]
	value := 2
	if g.rng.Float64() < 0.1 {
		value = 4
	}
	g.grid[cell[1]][cell[0]] = value
	return true
}

// slide is what one row's worth of sliding produced.
type slide struct {
	score   int  // points won
	biggest int  // the largest tile the move created, 0 if nothing merged
	changed bool // whether the row actually moved
}

// slideLine compacts one row towards index 0, merging equal neighbours once
// each.
func slideLine(line [gridN]int) ([gridN]int, slide) {
	var packed []int
	for _, v := range line {
		if v != 0 {
			packed = append(packed, v)
		}
	}

	var merged []int
	var res slide
	for i := 0; i < len(packed); i++ {
		if i+1 < len(packed) && packed[i] == packed[i+1] {
			value := packed[i] * 2
			merged = append(merged, value)
			res.score += value
			if value > res.biggest {
				res.biggest = value
			}
			i++ // the pair is spent
			continue
		}
		merged = append(merged, packed[i])
	}

	var out [gridN]int
	copy(out[:], merged)
	res.changed = out != line
	return out, res
}

type direction int

const (
	dirLeft direction = iota
	dirRight
	dirUp
	dirDown
)

// line reads one row or column in the order a move would process it.
func (g *g2048) line(dir direction, i int) [gridN]int {
	var out [gridN]int
	for j := 0; j < gridN; j++ {
		switch dir {
		case dirLeft:
			out[j] = g.grid[i][j]
		case dirRight:
			out[j] = g.grid[i][gridN-1-j]
		case dirUp:
			out[j] = g.grid[j][i]
		case dirDown:
			out[j] = g.grid[gridN-1-j][i]
		}
	}
	return out
}

func (g *g2048) setLine(dir direction, i int, line [gridN]int) {
	for j := 0; j < gridN; j++ {
		switch dir {
		case dirLeft:
			g.grid[i][j] = line[j]
		case dirRight:
			g.grid[i][gridN-1-j] = line[j]
		case dirUp:
			g.grid[j][i] = line[j]
		case dirDown:
			g.grid[gridN-1-j][i] = line[j]
		}
	}
}

// move slides the whole grid one way, returning whether anything shifted.
func (g *g2048) move(dir direction) bool {
	moved, biggest := false, 0
	for i := 0; i < gridN; i++ {
		line, res := slideLine(g.line(dir, i))
		if res.changed {
			g.setLine(dir, i, line)
			moved = true
		}
		g.score += res.score
		if res.biggest > biggest {
			biggest = res.biggest
		}
	}
	wasWon := g.won
	if g.highest() >= 2048 {
		g.won = true
	}
	if moved {
		g.moves++
		g.addTile()
		g.over = !g.canMove()
	}

	switch {
	case biggest > 0:
		g.snd.Play(sfxMerge(biggest)...)
	case moved:
		g.snd.Play(sfxSlide()...)
	}
	if g.won && !wasWon {
		g.snd.Play(sfxWin()...)
	}
	if g.over {
		g.snd.Play(sfxGameOver()...)
	}
	return moved
}

// canMove reports whether any direction would still change the grid.
func (g *g2048) canMove() bool {
	for y := 0; y < gridN; y++ {
		for x := 0; x < gridN; x++ {
			if g.grid[y][x] == 0 {
				return true
			}
			if x+1 < gridN && g.grid[y][x] == g.grid[y][x+1] {
				return true
			}
			if y+1 < gridN && g.grid[y][x] == g.grid[y+1][x] {
				return true
			}
		}
	}
	return false
}

func (g *g2048) highest() int {
	best := 0
	for _, row := range g.grid {
		for _, v := range row {
			best = max(best, v)
		}
	}
	return best
}

func (g *g2048) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "r":
		return g.Start()
	}
	if g.over {
		return nil
	}
	switch key.String() {
	case "left", "h", "a":
		g.move(dirLeft)
	case "right", "l", "d":
		g.move(dirRight)
	case "up", "k", "w":
		g.move(dirUp)
	case "down", "j", "s":
		g.move(dirDown)
	}
	return nil
}

func (g *g2048) Over() bool { return g.over }

func (g *g2048) Result() (int, bool, bool) {
	if !g.over || g.recorded {
		return g.score, false, false
	}
	g.recorded = true
	return g.score, false, true
}

func (g *g2048) Help() string {
	return helpStyle.Render("←↑↓→ slide · r restart")
}

// tileColors give each value its own shade, warming as the numbers climb.
var tileColors = map[int][2]string{
	2:    {"250", "240"},
	4:    {"230", "94"},
	8:    {"232", "215"},
	16:   {"232", "209"},
	32:   {"232", "203"},
	64:   {"255", "160"},
	128:  {"232", "227"},
	256:  {"232", "226"},
	512:  {"232", "220"},
	1024: {"232", "214"},
	2048: {"232", "213"},
}

func (g *g2048) View(width, height int) string {
	const tileW = 7

	var rows []string
	for y := 0; y < gridN; y++ {
		var top, mid, bot []string
		for x := 0; x < gridN; x++ {
			v := g.grid[y][x]
			fg, bg := "244", "236"
			label := ""
			if v > 0 {
				if c, ok := tileColors[v]; ok {
					fg, bg = c[0], c[1]
				} else {
					fg, bg = "232", "212"
				}
				label = fmt.Sprintf("%d", v)
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Background(lipgloss.Color(bg))
			top = append(top, style.Render(spaces(tileW)))
			mid = append(mid, style.Render(center(label, tileW)))
			bot = append(bot, style.Render(spaces(tileW)))
		}
		rows = append(rows,
			strings.Join(top, " "),
			strings.Join(mid, " "),
			strings.Join(bot, " "),
			"")
	}
	board := boxStyle.Render(strings.TrimRight(strings.Join(rows, "\n"), "\n"))

	side := []string{
		labelStyle.Render("score"), accentStyle.Render(fmt.Sprintf("%d", g.score)), "",
		labelStyle.Render("moves"), valueStyle.Render(fmt.Sprintf("%d", g.moves)), "",
		labelStyle.Render("best tile"), valueStyle.Render(fmt.Sprintf("%d", g.highest())),
	}
	if g.won {
		side = append(side, "", okStyle.Render("2048 reached"))
	}
	if g.over {
		side = append(side, "", warnStyle.Render("no moves left"), subtleStyle.Render("r to play again"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}
