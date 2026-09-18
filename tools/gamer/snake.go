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
	snakeW = 26
	snakeH = 15
)

type point struct{ x, y int }

type snakeTickMsg struct{ gen int }

type snake struct {
	body []point // head first
	dir  point
	// queued holds the turns typed since the last step, so a quick
	// double-tap cannot fold the snake back through its own neck.
	queued []point

	food  point
	score int
	grow  int

	over, paused bool
	gen          int
	recorded     bool

	rng *rand.Rand
}

func newSnake(seed int64) *snake {
	return &snake{rng: rand.New(rand.NewSource(seed))}
}

func (s *snake) ID() string         { return "snake" }
func (s *snake) Name() string       { return "Snake" }
func (s *snake) Blurb() string      { return "eat, grow, do not bite yourself" }
func (s *snake) ScoreLabel() string { return "points" }

func (s *snake) Start() tea.Cmd {
	mid := point{snakeW / 2, snakeH / 2}
	s.body = []point{mid, {mid.x - 1, mid.y}, {mid.x - 2, mid.y}}
	s.dir = point{1, 0}
	s.queued = nil
	s.score, s.grow = 0, 0
	s.over, s.paused, s.recorded = false, false, false
	s.placeFood()
	s.gen++
	return s.tick()
}

func (s *snake) tick() tea.Cmd {
	gen := s.gen
	return tea.Tick(s.speed(), func(time.Time) tea.Msg { return snakeTickMsg{gen} })
}

// speed quickens as the snake grows, down to a floor that stays playable.
func (s *snake) speed() time.Duration {
	ms := 140 - len(s.body)
	if ms < 60 {
		ms = 60
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *snake) occupies(p point) bool {
	for _, b := range s.body {
		if b == p {
			return true
		}
	}
	return false
}

// placeFood drops an apple on a free square.
func (s *snake) placeFood() {
	free := make([]point, 0, snakeW*snakeH)
	for y := 0; y < snakeH; y++ {
		for x := 0; x < snakeW; x++ {
			p := point{x, y}
			if !s.occupies(p) {
				free = append(free, p)
			}
		}
	}
	if len(free) == 0 {
		s.over = true // the board is entirely snake: a perfect game
		return
	}
	s.food = free[s.rng.Intn(len(free))]
}

// turn queues a direction change, ignoring a reversal into the neck.
func (s *snake) turn(d point) {
	last := s.dir
	if n := len(s.queued); n > 0 {
		last = s.queued[n-1]
	}
	if d.x == -last.x && d.y == -last.y {
		return
	}
	if d == last {
		return
	}
	s.queued = append(s.queued, d)
}

// step advances the snake one square.
func (s *snake) step() {
	if len(s.queued) > 0 {
		s.dir = s.queued[0]
		s.queued = s.queued[1:]
	}

	head := s.body[0]
	next := point{head.x + s.dir.x, head.y + s.dir.y}

	if next.x < 0 || next.x >= snakeW || next.y < 0 || next.y >= snakeH {
		s.over = true
		return
	}
	// The tail square is about to be vacated, so moving into it is fine
	// unless the snake is still growing into it.
	for i, b := range s.body {
		if b != next {
			continue
		}
		if i == len(s.body)-1 && s.grow == 0 {
			continue
		}
		s.over = true
		return
	}

	s.body = append([]point{next}, s.body...)
	if next == s.food {
		s.score += 10
		s.grow += 2
		s.placeFood()
	}
	if s.grow > 0 {
		s.grow--
	} else {
		s.body = s.body[:len(s.body)-1]
	}
}

func (s *snake) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case snakeTickMsg:
		if msg.gen != s.gen || s.over {
			return nil
		}
		if !s.paused {
			s.step()
		}
		if s.over {
			return nil
		}
		return s.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return s.Start()
		case "p":
			if !s.over {
				s.paused = !s.paused
			}
			return nil
		}
		if s.over || s.paused {
			return nil
		}
		switch msg.String() {
		case "left", "h", "a":
			s.turn(point{-1, 0})
		case "right", "l", "d":
			s.turn(point{1, 0})
		case "up", "k", "w":
			s.turn(point{0, -1})
		case "down", "j", "x":
			s.turn(point{0, 1})
		}
	}
	return nil
}

func (s *snake) Over() bool { return s.over }

func (s *snake) Result() (int, bool, bool) {
	if !s.over || s.recorded {
		return s.score, false, false
	}
	s.recorded = true
	return s.score, false, true
}

func (s *snake) Help() string {
	return helpStyle.Render("←↑↓→ steer · p pause · r restart")
}

var (
	snakeHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
	snakeBodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("71"))
	foodStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

func (s *snake) View(width, height int) string {
	cells := map[point]int{} // 1 head, 2 body
	for i, b := range s.body {
		if i == 0 {
			cells[b] = 1
			continue
		}
		cells[b] = 2
	}

	var rows []string
	for y := 0; y < snakeH; y++ {
		var row strings.Builder
		for x := 0; x < snakeW; x++ {
			p := point{x, y}
			switch {
			case cells[p] == 1:
				row.WriteString(snakeHeadStyle.Render("██"))
			case cells[p] == 2:
				row.WriteString(snakeBodyStyle.Render("▓▓"))
			case p == s.food:
				row.WriteString(foodStyle.Render("●●"))
			default:
				row.WriteString(dividerStyle.Render("  "))
			}
		}
		rows = append(rows, row.String())
	}
	board := boxStyle.Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("score"), accentStyle.Render(fmt.Sprintf("%d", s.score)), "",
		labelStyle.Render("length"), valueStyle.Render(fmt.Sprintf("%d", len(s.body))), "",
		labelStyle.Render("speed"), valueStyle.Render(fmt.Sprintf("%dms", s.speed().Milliseconds())),
	}
	if s.over {
		side = append(side, "", warnStyle.Render("game over"), subtleStyle.Render("r to play again"))
	} else if s.paused {
		side = append(side, "", accentStyle.Render("paused"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}
