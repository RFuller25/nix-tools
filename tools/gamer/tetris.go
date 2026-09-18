package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	boardW = 10
	boardH = 20
)

// tetromino describes a piece by the cells it fills inside an n×n box, which
// is all that is needed to rotate it: (x,y) becomes (n-1-y, x).
type tetromino struct {
	name  string
	size  int
	color string
	cells [4][2]int
}

var tetrominoes = [7]tetromino{
	{"I", 4, "51", [4][2]int{{0, 1}, {1, 1}, {2, 1}, {3, 1}}},
	{"O", 2, "226", [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}},
	{"T", 3, "141", [4][2]int{{1, 0}, {0, 1}, {1, 1}, {2, 1}}},
	{"S", 3, "84", [4][2]int{{1, 0}, {2, 0}, {0, 1}, {1, 1}}},
	{"Z", 3, "203", [4][2]int{{0, 0}, {1, 0}, {1, 1}, {2, 1}}},
	{"J", 3, "33", [4][2]int{{0, 0}, {0, 1}, {1, 1}, {2, 1}}},
	{"L", 3, "208", [4][2]int{{2, 0}, {0, 1}, {1, 1}, {2, 1}}},
}

// cellsAt returns a piece's occupied board coordinates for a rotation and
// position.
func (t tetromino) cellsAt(rot, px, py int) [4][2]int {
	var out [4][2]int
	for i, c := range t.cells {
		x, y := c[0], c[1]
		for r := 0; r < rot%4; r++ {
			x, y = t.size-1-y, x
		}
		out[i] = [2]int{px + x, py + y}
	}
	return out
}

type tetrisTickMsg struct{ gen int }

type tetris struct {
	board [boardH][boardW]int // 0 empty, otherwise piece index + 1

	cur      int
	rot      int
	x, y     int
	next     int
	hold     int
	held     bool
	holdUsed bool

	bag []int
	rng *rand.Rand

	score, lines, level int
	over, paused        bool
	gen                 int
	recorded            bool

	snd sounder
}

func newTetris(seed int64) *tetris {
	return &tetris{rng: rand.New(rand.NewSource(seed)), hold: -1, snd: noSound{}}
}

func (t *tetris) SetAudio(s sounder) { t.snd = s }

func (t *tetris) ID() string         { return "tetris" }
func (t *tetris) Name() string       { return "Tetris" }
func (t *tetris) Blurb() string      { return "stack the falling pieces, clear the lines" }
func (t *tetris) ScoreLabel() string { return "points" }

func (t *tetris) Start() tea.Cmd {
	t.board = [boardH][boardW]int{}
	t.score, t.lines, t.level = 0, 0, 1
	t.over, t.paused, t.recorded = false, false, false
	t.hold, t.held, t.holdUsed = -1, false, false
	t.bag = nil
	t.next = t.draw()
	t.spawn()
	t.gen++
	return t.tick()
}

func (t *tetris) tick() tea.Cmd {
	gen := t.gen
	return tea.Tick(t.gravity(), func(time.Time) tea.Msg { return tetrisTickMsg{gen} })
}

// gravity speeds up as the level climbs, but never becomes unplayable.
func (t *tetris) gravity() time.Duration {
	ms := 800 - (t.level-1)*65
	if ms < 90 {
		ms = 90
	}
	return time.Duration(ms) * time.Millisecond
}

// draw takes the next piece from a shuffled bag of all seven, the usual way
// of keeping the sequence fair.
func (t *tetris) draw() int {
	if len(t.bag) == 0 {
		t.bag = t.rng.Perm(7)
	}
	p := t.bag[0]
	t.bag = t.bag[1:]
	return p
}

func (t *tetris) spawn() {
	t.cur = t.next
	t.next = t.draw()
	t.rot = 0
	t.x = boardW/2 - tetrominoes[t.cur].size/2
	t.y = 0
	t.holdUsed = false
	if t.collides(t.rot, t.x, t.y) {
		t.over = true
		t.snd.Play(sfxGameOver()...)
	}
}

// collides reports whether a piece placed here would overlap the walls, the
// floor or a settled block.
func (t *tetris) collides(rot, px, py int) bool {
	for _, c := range tetrominoes[t.cur].cellsAt(rot, px, py) {
		x, y := c[0], c[1]
		if x < 0 || x >= boardW || y >= boardH {
			return true
		}
		if y >= 0 && t.board[y][x] != 0 {
			return true
		}
	}
	return false
}

func (t *tetris) move(dx, dy int) bool {
	if t.collides(t.rot, t.x+dx, t.y+dy) {
		return false
	}
	t.x += dx
	t.y += dy
	return true
}

// rotate turns the piece, nudging it sideways or up if the plain rotation is
// blocked — a simplified wall kick.
func (t *tetris) rotate(dir int) bool {
	rot := ((t.rot+dir)%4 + 4) % 4
	for _, kick := range [][2]int{{0, 0}, {-1, 0}, {1, 0}, {-2, 0}, {2, 0}, {0, -1}} {
		if !t.collides(rot, t.x+kick[0], t.y+kick[1]) {
			t.rot = rot
			t.x += kick[0]
			t.y += kick[1]
			return true
		}
	}
	return false
}

// lock settles the current piece, clears any full lines and spawns the next.
func (t *tetris) lock() {
	for _, c := range tetrominoes[t.cur].cellsAt(t.rot, t.x, t.y) {
		if c[1] >= 0 && c[1] < boardH && c[0] >= 0 && c[0] < boardW {
			t.board[c[1]][c[0]] = t.cur + 1
		}
	}
	t.snd.Play(sfxLock()...)
	if cleared := t.clearLines(); cleared > 0 {
		t.lines += cleared
		t.score += [5]int{0, 100, 300, 500, 800}[cleared] * t.level
		level := 1 + t.lines/10
		t.snd.Play(sfxLines(cleared)...)
		if level > t.level {
			t.snd.Play(sfxLevelUp()...)
		}
		t.level = level
	}
	t.spawn()
}

func (t *tetris) clearLines() int {
	cleared := 0
	for y := boardH - 1; y >= 0; y-- {
		full := true
		for x := 0; x < boardW; x++ {
			if t.board[y][x] == 0 {
				full = false
				break
			}
		}
		if !full {
			continue
		}
		cleared++
		for row := y; row > 0; row-- {
			t.board[row] = t.board[row-1]
		}
		t.board[0] = [boardW]int{}
		y++ // re-check this row, it now holds what was above it
	}
	return cleared
}

// ghostY is where the piece would land if dropped from here.
func (t *tetris) ghostY() int {
	y := t.y
	for !t.collides(t.rot, t.x, y+1) {
		y++
	}
	return y
}

func (t *tetris) hardDrop() {
	t.snd.Play(sfxHardDrop()...)
	target := t.ghostY()
	t.score += 2 * (target - t.y)
	t.y = target
	t.lock()
}

// swapHold puts the current piece aside, bringing back whatever was there.
// Allowed once per piece, as usual.
func (t *tetris) swapHold() {
	if t.holdUsed {
		return
	}
	cur := t.cur
	if t.held {
		t.cur = t.hold
	} else {
		t.cur = t.next
		t.next = t.draw()
	}
	t.hold, t.held = cur, true
	t.snd.Play(sfxHold()...)
	t.rot = 0
	t.x = boardW/2 - tetrominoes[t.cur].size/2
	t.y = 0
	t.holdUsed = true
	if t.collides(t.rot, t.x, t.y) {
		t.over = true
		t.snd.Play(sfxGameOver()...)
	}
}

func (t *tetris) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tetrisTickMsg:
		if msg.gen != t.gen {
			return nil // a tick left over from a previous round
		}
		if t.over {
			return nil
		}
		if !t.paused {
			if !t.move(0, 1) {
				t.lock()
			}
		}
		return t.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return t.Start()
		case "p":
			if !t.over {
				t.paused = !t.paused
			}
			return nil
		}
		if t.over || t.paused {
			return nil
		}
		switch msg.String() {
		case "left", "h":
			if t.move(-1, 0) {
				t.snd.Play(sfxShift()...)
			}
		case "right", "l":
			if t.move(1, 0) {
				t.snd.Play(sfxShift()...)
			}
		case "down", "j":
			if t.move(0, 1) {
				t.score++
			}
		case "up", "k", "x":
			if t.rotate(1) {
				t.snd.Play(sfxRotate()...)
			}
		case "z":
			if t.rotate(-1) {
				t.snd.Play(sfxRotate()...)
			}
		case " ":
			t.hardDrop()
		case "c":
			t.swapHold()
		}
	}
	return nil
}

func (t *tetris) Over() bool { return t.over }

func (t *tetris) Result() (int, bool, bool) {
	if !t.over || t.recorded {
		return t.score, false, false
	}
	t.recorded = true
	return t.score, false, true
}

func (t *tetris) Help() string {
	return helpStyle.Render("←→ move · ↓ soft drop · ↑/z rotate · space hard drop · c hold · p pause · r restart")
}

func (t *tetris) View(width, height int) string {
	ghost := t.ghostY()
	live := map[[2]int]int{}
	if !t.over {
		for _, c := range tetrominoes[t.cur].cellsAt(t.rot, t.x, ghost) {
			live[[2]int{c[0], c[1]}] = -1 // ghost marker
		}
		for _, c := range tetrominoes[t.cur].cellsAt(t.rot, t.x, t.y) {
			live[[2]int{c[0], c[1]}] = t.cur + 1
		}
	}

	var rows []string
	for y := 0; y < boardH; y++ {
		var row strings.Builder
		for x := 0; x < boardW; x++ {
			switch cell := t.board[y][x]; {
			case cell != 0:
				row.WriteString(blockStyle(tetrominoes[cell-1].color).Render("██"))
			case live[[2]int{x, y}] > 0:
				row.WriteString(blockStyle(tetrominoes[live[[2]int{x, y}]-1].color).Render("██"))
			case live[[2]int{x, y}] == -1:
				row.WriteString(dividerStyle.Render("▒▒"))
			default:
				row.WriteString(dividerStyle.Render("· "))
			}
		}
		rows = append(rows, row.String())
	}
	board := boxStyle.Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("score"), accentStyle.Render(fmt.Sprintf("%d", t.score)), "",
		labelStyle.Render("lines"), valueStyle.Render(fmt.Sprintf("%d", t.lines)), "",
		labelStyle.Render("level"), valueStyle.Render(fmt.Sprintf("%d", t.level)), "",
		labelStyle.Render("next"),
	}
	side = append(side, miniPiece(t.next)...)
	side = append(side, "", labelStyle.Render("hold"))
	if t.held {
		side = append(side, miniPiece(t.hold)...)
	} else {
		side = append(side, subtleStyle.Render("  —"))
	}
	if t.over {
		side = append(side, "", warnStyle.Render("game over"), subtleStyle.Render("r to play again"))
	} else if t.paused {
		side = append(side, "", accentStyle.Render("paused"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}

// miniPiece draws a piece in a small 4x2 preview.
func miniPiece(idx int) []string {
	if idx < 0 {
		return []string{subtleStyle.Render("  —")}
	}
	p := tetrominoes[idx]
	grid := map[[2]int]bool{}
	minY, maxY, minX, maxX := 9, -1, 9, -1
	for _, c := range p.cells {
		grid[[2]int{c[0], c[1]}] = true
		minX, maxX = min(minX, c[0]), max(maxX, c[0])
		minY, maxY = min(minY, c[1]), max(maxY, c[1])
	}
	var out []string
	for y := minY; y <= maxY; y++ {
		var row strings.Builder
		for x := minX; x <= maxX; x++ {
			if grid[[2]int{x, y}] {
				row.WriteString(blockStyle(p.color).Render("██"))
			} else {
				row.WriteString("  ")
			}
		}
		out = append(out, row.String())
	}
	return out
}

func blockStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}
