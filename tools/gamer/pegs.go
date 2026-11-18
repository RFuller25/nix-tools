package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pegs: the traditional solitaire on the 33-hole cross-shaped board. A peg
// jumps straight over a neighbour into the empty hole beyond, and the peg it
// jumped is taken off. Play until no jump is left; the fewer pegs remain the
// better, and a single peg in the centre hole is a perfect game.

const pegsSize = 7

type pegsHole int8

const (
	pegsOff   pegsHole = iota // not part of the board
	pegsEmpty                 // a hole with nothing in it
	pegsPeg
)

type pegsMove struct{ from, over, to point }

type pegs struct {
	board  [pegsSize][pegsSize]pegsHole
	cx, cy int
	held   bool // the peg under the cursor is picked up, waiting for a direction
	undo   []pegsMove

	over     bool
	recorded bool
	snd      sounder
}

func newPegs() *pegs { return &pegs{snd: noSound{}} }

func (p *pegs) SetAudio(s sounder) { p.snd = s }

func (p *pegs) ID() string         { return "pegs" }
func (p *pegs) Name() string       { return "Pegs" }
func (p *pegs) Blurb() string      { return "jump pegs over each other until only one is left" }
func (p *pegs) ScoreLabel() string { return "pegs left" }

func pegsOnBoard(x, y int) bool {
	if x < 0 || y < 0 || x >= pegsSize || y >= pegsSize {
		return false
	}
	return (x >= 2 && x <= 4) || (y >= 2 && y <= 4)
}

func (p *pegs) Start() tea.Cmd {
	for y := 0; y < pegsSize; y++ {
		for x := 0; x < pegsSize; x++ {
			p.board[y][x] = pegsOff
			if pegsOnBoard(x, y) {
				p.board[y][x] = pegsPeg
			}
		}
	}
	p.board[3][3] = pegsEmpty
	p.cx, p.cy = 3, 5
	p.held, p.undo = false, nil
	p.over, p.recorded = false, false
	return nil
}

func (p *pegs) at(q point) pegsHole {
	if !pegsOnBoard(q.x, q.y) {
		return pegsOff
	}
	return p.board[q.y][q.x]
}

// jumpFrom is the move from a hole in a direction, if it is legal.
func (p *pegs) jumpFrom(from, d point) (pegsMove, bool) {
	m := pegsMove{from: from, over: point{from.x + d.x, from.y + d.y}, to: point{from.x + 2*d.x, from.y + 2*d.y}}
	ok := p.at(m.from) == pegsPeg && p.at(m.over) == pegsPeg && p.at(m.to) == pegsEmpty
	return m, ok
}

var pegsDirs = []point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// movesFrom lists every jump the peg in a hole can make.
func (p *pegs) movesFrom(from point) []pegsMove {
	var out []pegsMove
	for _, d := range pegsDirs {
		if m, ok := p.jumpFrom(from, d); ok {
			out = append(out, m)
		}
	}
	return out
}

func (p *pegs) anyMove() bool {
	for y := 0; y < pegsSize; y++ {
		for x := 0; x < pegsSize; x++ {
			if len(p.movesFrom(point{x, y})) > 0 {
				return true
			}
		}
	}
	return false
}

func (p *pegs) count() int {
	n := 0
	for _, row := range p.board {
		for _, h := range row {
			if h == pegsPeg {
				n++
			}
		}
	}
	return n
}

func (p *pegs) apply(m pegsMove) {
	p.board[m.from.y][m.from.x] = pegsEmpty
	p.board[m.over.y][m.over.x] = pegsEmpty
	p.board[m.to.y][m.to.x] = pegsPeg
	p.undo = append(p.undo, m)
	p.cx, p.cy = m.to.x, m.to.y
	p.held = false
	if !p.anyMove() {
		p.over = true
		if p.perfect() {
			p.snd.Play(sfxWin()...)
		} else {
			p.snd.Play(sfxGameOver()...)
		}
		return
	}
	p.snd.Play(sfxJump()...)
}

func (p *pegs) takeBack() {
	if len(p.undo) == 0 {
		return
	}
	m := p.undo[len(p.undo)-1]
	p.undo = p.undo[:len(p.undo)-1]
	p.board[m.from.y][m.from.x] = pegsPeg
	p.board[m.over.y][m.over.x] = pegsPeg
	p.board[m.to.y][m.to.x] = pegsEmpty
	p.cx, p.cy = m.from.x, m.from.y
	p.held, p.over = false, false
	p.snd.Play(sfxDrop()...)
}

func (p *pegs) perfect() bool { return p.count() == 1 && p.board[3][3] == pegsPeg }

// moveCursor steps the cursor to the next hole in a direction, skipping the
// corners that are not part of the board.
func (p *pegs) moveCursor(d point) {
	for x, y := p.cx+d.x, p.cy+d.y; x >= 0 && y >= 0 && x < pegsSize && y < pegsSize; x, y = x+d.x, y+d.y {
		if pegsOnBoard(x, y) {
			p.cx, p.cy = x, y
			return
		}
	}
}

func (p *pegs) Update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "r":
		return p.Start()
	case "u", "z", "backspace":
		p.takeBack()
		return nil
	}
	if p.over {
		return nil
	}
	var d point
	switch k.String() {
	case "up", "k", "w":
		d = point{0, -1}
	case "right", "l", "d":
		d = point{1, 0}
	case "down", "j", "s":
		d = point{0, 1}
	case "left", "h", "a":
		d = point{-1, 0}
	case " ", "enter":
		cur := point{p.cx, p.cy}
		switch {
		case p.held:
			p.held = false
			p.snd.Play(sfxDrop()...)
		case p.at(cur) == pegsPeg:
			moves := p.movesFrom(cur)
			if len(moves) == 1 {
				p.apply(moves[0]) // only one way to go: just go
			} else if len(moves) > 1 {
				p.held = true
				p.snd.Play(sfxPick()...)
			}
		case p.at(cur) == pegsEmpty:
			// On an empty hole, jump the one peg that can land here, if only one can.
			var into []pegsMove
			for _, dd := range pegsDirs {
				if m, ok := p.jumpFrom(point{cur.x - 2*dd.x, cur.y - 2*dd.y}, dd); ok {
					into = append(into, m)
				}
			}
			if len(into) == 1 {
				p.apply(into[0])
			}
		}
		return nil
	default:
		return nil
	}
	if p.held {
		if m, ok := p.jumpFrom(point{p.cx, p.cy}, d); ok {
			p.apply(m)
		}
		return nil
	}
	p.moveCursor(d)
	return nil
}

func (p *pegs) Over() bool { return p.over }

func (p *pegs) Result() (int, bool, bool) {
	if !p.over || p.recorded {
		return p.count(), true, false
	}
	p.recorded = true
	return p.count(), true, true
}

func (p *pegs) Help() string {
	if p.held {
		return helpStyle.Render("←↑↓→ jump that way · space put it down · u undo · r restart")
	}
	return helpStyle.Render("←↑↓→ move · space pick up a peg (or fill this hole) · u undo · r restart")
}

var (
	pegStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	pegLiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	holeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (p *pegs) View(width, height int) string {
	// Holes a held peg could land in, to show where it may go.
	targets := map[point]bool{}
	if p.held {
		for _, m := range p.movesFrom(point{p.cx, p.cy}) {
			targets[m.to] = true
		}
	}
	var rows []string
	for y := 0; y < pegsSize; y++ {
		var row strings.Builder
		for x := 0; x < pegsSize; x++ {
			q := point{x, y}
			cell := "   "
			switch p.at(q) {
			case pegsPeg:
				style := pegStyle
				if !p.over && len(p.movesFrom(q)) == 0 {
					style = style.Faint(true)
				}
				if p.held && x == p.cx && y == p.cy {
					style = pegLiveStyle.Bold(true)
				}
				cell = style.Render(" ● ")
			case pegsEmpty:
				if targets[q] {
					cell = pegLiveStyle.Render(" ◌ ")
				} else {
					cell = holeStyle.Render(" · ")
				}
			}
			if x == p.cx && y == p.cy {
				cell = lipgloss.NewStyle().Reverse(true).Render(stripANSI(cell))
			}
			row.WriteString(cell)
		}
		rows = append(rows, row.String())
		if y < pegsSize-1 {
			rows = append(rows, "")
		}
	}
	board := boxStyle.Padding(0, 1).Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("pegs left"), accentStyle.Render(fmt.Sprintf("%d", p.count())), "",
		labelStyle.Render("jumps"), valueStyle.Render(fmt.Sprintf("%d", len(p.undo))),
	}
	if p.over {
		verdict := subtleStyle.Render("no jumps left")
		switch {
		case p.perfect():
			verdict = okStyle.Render("perfect: one peg, dead centre")
		case p.count() == 1:
			verdict = okStyle.Render("one peg left")
		}
		side = append(side, "", verdict, subtleStyle.Render("u to take back · r for a fresh board"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}

// stripANSI removes colour codes, so the cursor can be drawn as plain reverse video.
func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			esc = true
		case esc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			esc = false
		case !esc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
