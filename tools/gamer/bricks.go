package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Bricks: knock a wall of bricks down with a ball, keeping it off the floor
// with a paddle. Where the ball meets the paddle sets the angle it leaves at,
// so the paddle's edges aim and its middle sends the ball straight up.

const (
	bricksW     = 48 // the field, in terminal columns
	bricksH     = 18 // and rows
	brickCols   = 12
	brickRows   = 6
	brickWidth  = bricksW / brickCols
	brickTop    = 2 // empty rows above the wall
	paddleWidth = 8
	paddleStep  = 3
	paddleRow   = bricksH - 1
	bricksLives = 3
	bricksTick  = 33 * time.Millisecond

	ballSpeed     = 11.0 // rows a second, on the first wall
	ballSpeedUp   = 1.12 // each new wall is this much quicker
	ballMaxAngle  = 62.0 // degrees from straight up, off the paddle's very edge
	bricksMaxStep = 0.4  // the most a ball moves per collision check, in rows
)

type bricksTickMsg struct{ gen int }

var brickRowColors = [brickRows]string{"203", "209", "221", "114", "75", "141"}

type bricks struct {
	wall [brickRows][brickCols]bool

	px       int     // paddle's left column
	bx, by   float64 // the ball, in columns and rows
	vx, vy   float64 // its velocity, columns and rows a second
	held     bool    // sitting on the paddle, waiting to be served
	speed    float64 // rows a second
	lives    int
	score    int
	level    int
	over     bool
	paused   bool
	recorded bool
	gen      int

	rng *rand.Rand
	snd sounder
}

func newBricks(seed int64) *bricks {
	return &bricks{rng: rand.New(rand.NewSource(seed)), snd: noSound{}}
}

func (b *bricks) SetAudio(s sounder) { b.snd = s }

func (b *bricks) ID() string         { return "bricks" }
func (b *bricks) Name() string       { return "Bricks" }
func (b *bricks) Blurb() string      { return "bounce a ball off a paddle and knock the wall down" }
func (b *bricks) ScoreLabel() string { return "points" }

func (b *bricks) Start() tea.Cmd {
	b.lives, b.score, b.level = bricksLives, 0, 1
	b.over, b.paused, b.recorded = false, false, false
	b.speed = ballSpeed
	b.px = (bricksW - paddleWidth) / 2
	b.buildWall()
	b.serve()
	b.gen++
	return b.tick()
}

func (b *bricks) tick() tea.Cmd {
	gen := b.gen
	return tea.Tick(bricksTick, func(time.Time) tea.Msg { return bricksTickMsg{gen} })
}

func (b *bricks) buildWall() {
	for r := range b.wall {
		for c := range b.wall[r] {
			b.wall[r][c] = true
		}
	}
}

// serve puts the ball back on the paddle.
func (b *bricks) serve() {
	b.held = true
	b.placeOnPaddle()
}

func (b *bricks) placeOnPaddle() {
	b.bx = float64(b.px) + float64(paddleWidth)/2
	b.by = paddleRow - 1
}

// launch sends a held ball up, leaning a little to one side.
func (b *bricks) launch() {
	if !b.held {
		return
	}
	b.held = false
	lean := 12 + 20*b.rng.Float64()
	if b.rng.Intn(2) == 0 {
		lean = -lean
	}
	b.aim(lean)
	b.snd.Play(sfxBounce()...)
}

// aim sets the ball moving upwards at an angle from vertical. A row is about
// twice as tall as a column is wide, so sideways speed counts double in
// columns to keep the angle true on screen.
func (b *bricks) aim(degrees float64) {
	rad := degrees * math.Pi / 180
	b.vx = 2 * b.speed * math.Sin(rad)
	b.vy = -b.speed * math.Cos(rad)
}

func (b *bricks) bricksLeft() int {
	n := 0
	for _, row := range b.wall {
		for _, on := range row {
			if on {
				n++
			}
		}
	}
	return n
}

// brickAt is the brick covering a point of the field, if any.
func (b *bricks) brickAt(x, y float64) (r, c int, ok bool) {
	if x < 0 || y < 0 {
		return 0, 0, false
	}
	r, c = int(y)-brickTop, int(x)/brickWidth
	if r < 0 || r >= brickRows || c < 0 || c >= brickCols || !b.wall[r][c] {
		return 0, 0, false
	}
	return r, c, true
}

// hit knocks out a brick; the higher rows are worth more.
func (b *bricks) hit(r, c int) {
	b.wall[r][c] = false
	b.score += (brickRows - r) * 10 * b.level
	b.snd.Play(sfxBrick(r)...)
	if b.bricksLeft() == 0 {
		b.level++
		b.speed *= ballSpeedUp
		b.buildWall()
		b.serve()
		b.snd.Play(sfxLevelUp()...)
	}
}

// advance moves the ball on by dt seconds, in steps small enough that it
// cannot pass through a brick or the paddle.
func (b *bricks) advance(dt float64) {
	if b.over || b.paused {
		return
	}
	if b.held {
		b.placeOnPaddle()
		return
	}
	dist := math.Max(math.Abs(b.vx)/2, math.Abs(b.vy)) * dt
	steps := int(math.Ceil(dist / bricksMaxStep))
	for i := 0; i < max(1, steps) && !b.held && !b.over; i++ {
		b.step(dt / float64(max(1, steps)))
	}
}

func (b *bricks) step(dt float64) {
	// Across first, then down, so a corner hit bounces the way it looks.
	nx := b.bx + b.vx*dt
	switch {
	case nx < 0:
		nx, b.vx = -nx, math.Abs(b.vx)
		b.snd.Play(sfxBounce()...)
	case nx >= bricksW:
		nx, b.vx = 2*bricksW-nx-0.001, -math.Abs(b.vx)
		b.snd.Play(sfxBounce()...)
	}
	if r, c, ok := b.brickAt(nx, b.by); ok {
		b.vx = -b.vx
		b.hit(r, c)
		return
	}
	b.bx = nx

	ny := b.by + b.vy*dt
	if ny < 0 {
		ny, b.vy = -ny, math.Abs(b.vy)
		b.snd.Play(sfxBounce()...)
	}
	if r, c, ok := b.brickAt(b.bx, ny); ok {
		b.vy = -b.vy
		b.hit(r, c)
		return
	}
	if b.vy > 0 && b.by < paddleRow && ny >= paddleRow {
		if off := b.bx - float64(b.px); off >= -0.5 && off <= float64(paddleWidth)+0.5 {
			// Where it lands on the paddle, -1 at the left edge to 1 at the right.
			t := math.Max(-1, math.Min(1, 2*off/float64(paddleWidth)-1))
			b.aim(t * ballMaxAngle)
			b.by = paddleRow - (ny - paddleRow) - 0.001
			b.snd.Play(sfxBounce()...)
			return
		}
	}
	b.by = ny
	if b.by >= bricksH {
		b.lives--
		if b.lives <= 0 {
			b.over = true
			b.snd.Play(sfxGameOver()...)
			return
		}
		b.snd.Play(sfxLifeLost()...)
		b.serve()
	}
}

func (b *bricks) movePaddle(d int) {
	b.px = max(0, min(bricksW-paddleWidth, b.px+d))
	if b.held {
		b.placeOnPaddle()
	}
}

func (b *bricks) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case bricksTickMsg:
		if msg.gen != b.gen || b.over {
			return nil
		}
		b.advance(bricksTick.Seconds())
		return b.tick()

	case tea.KeyMsg:
		if msg.String() == "r" {
			return b.Start()
		}
		if b.over {
			return nil
		}
		switch msg.String() {
		case "p":
			b.paused = !b.paused
			return nil
		}
		if b.paused {
			return nil
		}
		switch msg.String() {
		case "left", "h", "a":
			b.movePaddle(-paddleStep)
		case "right", "l", "d":
			b.movePaddle(paddleStep)
		case "shift+left", "H", "A", "home":
			b.movePaddle(-bricksW)
		case "shift+right", "L", "D", "end":
			b.movePaddle(bricksW)
		case " ", "up", "k", "w", "enter":
			b.launch()
		}
	}
	return nil
}

func (b *bricks) Over() bool { return b.over }

func (b *bricks) Result() (int, bool, bool) {
	if !b.over || b.recorded {
		return b.score, false, false
	}
	b.recorded = true
	return b.score, false, true
}

func (b *bricks) Help() string {
	return helpStyle.Render("←→ move (shift: jump to the edge) · space serve · p pause · r restart")
}

var (
	ballStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true)
	paddleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
)

func (b *bricks) View(width, height int) string {
	bx, by := int(math.Floor(b.bx)), int(math.Floor(b.by))
	rows := make([]string, bricksH)
	for y := 0; y < bricksH; y++ {
		var row strings.Builder
		for x := 0; x < bricksW; x++ {
			r, c := y-brickTop, x/brickWidth
			switch {
			case x == bx && y == by && !b.over:
				row.WriteString(ballStyle.Render("●"))
			case y == paddleRow && x >= b.px && x < b.px+paddleWidth:
				row.WriteString(paddleStyle.Render("▀"))
			case r >= 0 && r < brickRows && b.wall[r][c]:
				// Each brick is a block with a gap on its right, so the
				// wall reads as bricks and not as stripes.
				glyph := "█"
				if x%brickWidth == brickWidth-1 {
					glyph = "▌"
				}
				row.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(brickRowColors[r])).Render(glyph))
			default:
				row.WriteByte(' ')
			}
		}
		rows[y] = row.String()
	}
	board := boxStyle.Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("score"), accentStyle.Render(fmt.Sprintf("%d", b.score)), "",
		labelStyle.Render("balls"), valueStyle.Render(strings.Repeat("● ", b.lives)), "",
		labelStyle.Render("wall"), valueStyle.Render(fmt.Sprintf("%d", b.level)), "",
		labelStyle.Render("bricks"), valueStyle.Render(fmt.Sprintf("%d", b.bricksLeft())),
	}
	switch {
	case b.over:
		side = append(side, "", warnStyle.Render("game over"), subtleStyle.Render("r to play again"))
	case b.paused:
		side = append(side, "", accentStyle.Render("paused"))
	case b.held:
		side = append(side, "", subtleStyle.Render("space to serve"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}
