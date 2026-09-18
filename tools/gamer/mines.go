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
	minesW     = 16
	minesH     = 14
	mineCount  = 40
	mineTickMS = 250
)

type minesTickMsg struct{ gen int }

type cell struct {
	mine     bool
	revealed bool
	flagged  bool
	near     int // adjacent mines
}

// mines is Minesweeper: the odd one out in this arcade, because it is the
// only game here you win by thinking rather than reacting.
type mines struct {
	grid   [minesH][minesW]cell
	cx, cy int

	seeded   bool // mines are laid after the first reveal, never under it
	lost     bool
	won      bool
	recorded bool

	started time.Time
	ended   time.Time
	gen     int

	rng *rand.Rand
	snd sounder
}

func newMines(seed int64) *mines {
	return &mines{rng: rand.New(rand.NewSource(seed)), snd: noSound{}}
}

func (m *mines) SetAudio(s sounder) { m.snd = s }

func (m *mines) ID() string         { return "mines" }
func (m *mines) Name() string       { return "Minesweeper" }
func (m *mines) Blurb() string      { return "clear the field without standing on a mine" }
func (m *mines) ScoreLabel() string { return "seconds" }

func (m *mines) Start() tea.Cmd {
	m.grid = [minesH][minesW]cell{}
	m.cx, m.cy = minesW/2, minesH/2
	m.seeded, m.lost, m.won, m.recorded = false, false, false, false
	m.started, m.ended = time.Time{}, time.Time{}
	m.gen++
	return m.tick()
}

func (m *mines) tick() tea.Cmd {
	gen := m.gen
	return tea.Tick(mineTickMS*time.Millisecond, func(time.Time) tea.Msg { return minesTickMsg{gen} })
}

// layMines scatters the mines, keeping the first square the player opens and
// everything touching it clear, so no game is lost on move one.
func (m *mines) layMines(safeX, safeY int) {
	var spots [][2]int
	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			if abs(x-safeX) <= 1 && abs(y-safeY) <= 1 {
				continue
			}
			spots = append(spots, [2]int{x, y})
		}
	}
	for _, i := range m.rng.Perm(len(spots))[:min(mineCount, len(spots))] {
		m.grid[spots[i][1]][spots[i][0]].mine = true
	}
	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			m.grid[y][x].near = m.countNear(x, y)
		}
	}
	m.seeded = true
	m.started = time.Now()
}

func (m *mines) countNear(x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= minesW || ny < 0 || ny >= minesH {
				continue
			}
			if m.grid[ny][nx].mine {
				n++
			}
		}
	}
	return n
}

// reveal opens a square, spreading outwards through the empty ones.
func (m *mines) reveal(x, y int) {
	if m.lost || m.won {
		return
	}
	if !m.seeded {
		m.layMines(x, y)
	}
	c := &m.grid[y][x]
	if c.revealed || c.flagged {
		return
	}
	c.revealed = true
	if c.mine {
		m.lost = true
		m.ended = time.Now()
		return
	}
	if c.near == 0 {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || nx >= minesW || ny < 0 || ny >= minesH {
					continue
				}
				if !m.grid[ny][nx].revealed {
					m.reveal(nx, ny)
				}
			}
		}
	}
	m.checkWin()
}

// chord opens every unflagged neighbour of a number whose flags already add
// up — the standard shortcut, and the standard way to blow yourself up.
func (m *mines) chord(x, y int) {
	c := m.grid[y][x]
	if !c.revealed || c.near == 0 {
		return
	}
	flags := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= minesW || ny < 0 || ny >= minesH {
				continue
			}
			if m.grid[ny][nx].flagged {
				flags++
			}
		}
	}
	if flags != c.near {
		return
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= minesW || ny < 0 || ny >= minesH {
				continue
			}
			if !m.grid[ny][nx].flagged && !m.grid[ny][nx].revealed {
				m.reveal(nx, ny)
			}
		}
	}
}

func (m *mines) toggleFlag(x, y int) {
	if m.lost || m.won || m.grid[y][x].revealed {
		return
	}
	m.grid[y][x].flagged = !m.grid[y][x].flagged
}

func (m *mines) checkWin() {
	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			c := m.grid[y][x]
			if !c.mine && !c.revealed {
				return
			}
		}
	}
	m.won = true
	m.ended = time.Now()
}

func (m *mines) flagsUsed() int {
	n := 0
	for _, row := range m.grid {
		for _, c := range row {
			if c.flagged {
				n++
			}
		}
	}
	return n
}

func (m *mines) elapsed() time.Duration {
	if m.started.IsZero() {
		return 0
	}
	if !m.ended.IsZero() {
		return m.ended.Sub(m.started)
	}
	return time.Since(m.started)
}

func (m *mines) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case minesTickMsg:
		if msg.gen != m.gen || m.lost || m.won {
			return nil
		}
		return m.tick() // keeps the clock on screen ticking

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			return m.Start()
		}
		if m.lost || m.won {
			return nil
		}
		switch msg.String() {
		case "left", "h":
			m.cx = max(0, m.cx-1)
		case "right", "l":
			m.cx = min(minesW-1, m.cx+1)
		case "up", "k":
			m.cy = max(0, m.cy-1)
		case "down", "j":
			m.cy = min(minesH-1, m.cy+1)
		case "enter", " ":
			// One sound per key press, whatever the flood fill opened.
			if m.grid[m.cy][m.cx].revealed {
				m.chord(m.cx, m.cy)
			} else {
				m.reveal(m.cx, m.cy)
			}
			switch {
			case m.lost:
				m.snd.Play(sfxBoom()...)
			case m.won:
				m.snd.Play(sfxWin()...)
			default:
				m.snd.Play(sfxReveal()...)
			}
		case "f", "x":
			before := m.grid[m.cy][m.cx].flagged
			m.toggleFlag(m.cx, m.cy)
			if m.grid[m.cy][m.cx].flagged != before {
				m.snd.Play(sfxFlag()...)
			}
		}
	}
	return nil
}

func (m *mines) Over() bool { return m.lost || m.won }

func (m *mines) Result() (int, bool, bool) {
	seconds := int(m.elapsed().Seconds())
	if !m.won || m.recorded {
		return seconds, true, false // only a win is worth recording
	}
	m.recorded = true
	return seconds, true, true
}

func (m *mines) Help() string {
	return helpStyle.Render("←↑↓→ move · space open (or clear around a number) · f flag · r restart")
}

var nearColors = [9]string{"240", "75", "114", "203", "141", "215", "80", "252", "244"}

func (m *mines) View(width, height int) string {
	var rows []string
	for y := 0; y < minesH; y++ {
		var row strings.Builder
		for x := 0; x < minesW; x++ {
			c := m.grid[y][x]
			text, style := "▓▓", dividerStyle
			switch {
			case c.flagged && !(m.lost && c.mine):
				text, style = "⚑ ", warnStyle
			case (m.lost || m.won) && c.mine:
				text, style = "✸ ", warnStyle
			case !c.revealed:
				text, style = "▓▓", subtleStyle
			case c.near == 0:
				text, style = "  ", dividerStyle
			default:
				text = fmt.Sprintf("%d ", c.near)
				style = lipgloss.NewStyle().Foreground(lipgloss.Color(nearColors[c.near]))
			}
			if x == m.cx && y == m.cy {
				style = style.Reverse(true)
			}
			row.WriteString(style.Render(text))
		}
		rows = append(rows, row.String())
	}
	board := boxStyle.Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("mines"), accentStyle.Render(fmt.Sprintf("%d", mineCount-m.flagsUsed())), "",
		labelStyle.Render("time"), valueStyle.Render(fmt.Sprintf("%.0fs", m.elapsed().Seconds())), "",
		labelStyle.Render("flags"), valueStyle.Render(fmt.Sprintf("%d", m.flagsUsed())),
	}
	switch {
	case m.won:
		side = append(side, "", okStyle.Render("field cleared"),
			subtleStyle.Render(fmt.Sprintf("%.0f seconds", m.elapsed().Seconds())), subtleStyle.Render("r for another"))
	case m.lost:
		side = append(side, "", warnStyle.Render("boom"), subtleStyle.Render("r to try again"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
