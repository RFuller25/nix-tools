package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	hueW = 7
	hueH = 5
)

// hue is the shade-sorting puzzle: a smooth colour gradient is cut into tiles
// and shuffled, with a few pinned in place as reference points. Put every
// shade back where it belongs.
type hue struct {
	target [hueH][hueW]rgb  // the finished gradient
	tiles  [hueH][hueW]int  // which target tile currently sits in each slot
	fixed  [hueH][hueW]bool // pinned tiles, which cannot be moved

	cx, cy   int  // cursor
	holding  bool // a tile has been picked up
	hx, hy   int
	moves    int
	solved   bool
	recorded bool

	rng *rand.Rand
}

type rgb struct{ r, g, b float64 }

func (c rgb) hex() string {
	clamp := func(v float64) int {
		i := int(math.Round(v * 255))
		if i < 0 {
			return 0
		}
		if i > 255 {
			return 255
		}
		return i
	}
	return fmt.Sprintf("#%02x%02x%02x", clamp(c.r), clamp(c.g), clamp(c.b))
}

func newHue(seed int64) *hue {
	return &hue{rng: rand.New(rand.NewSource(seed))}
}

func (h *hue) ID() string         { return "hue" }
func (h *hue) Name() string       { return "Hue" }
func (h *hue) Blurb() string      { return "put the shades back in order" }
func (h *hue) ScoreLabel() string { return "moves" }

// hslToRGB converts a hue in degrees plus saturation and lightness in 0..1.
func hslToRGB(hDeg, s, l float64) rgb {
	c := (1 - math.Abs(2*l-1)) * s
	hp := math.Mod(hDeg/60, 6)
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch int(hp) {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return rgb{r + m, g + m, b + m}
}

// buildGradient interpolates a gradient from four corner colours, which is
// what gives the board its smooth two-way blend.
func (h *hue) buildGradient() {
	base := h.rng.Float64() * 360
	spread := 40 + h.rng.Float64()*80 // how far the hue travels across the board

	corners := [4]rgb{
		hslToRGB(base, 0.75, 0.68),
		hslToRGB(math.Mod(base+spread, 360), 0.75, 0.62),
		hslToRGB(math.Mod(base+spread/2, 360), 0.70, 0.34),
		hslToRGB(math.Mod(base+spread*1.5, 360), 0.70, 0.30),
	}

	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			fx := float64(x) / float64(hueW-1)
			fy := float64(y) / float64(hueH-1)
			top := mix(corners[0], corners[1], fx)
			bottom := mix(corners[2], corners[3], fx)
			h.target[y][x] = mix(top, bottom, fy)
		}
	}
}

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func (h *hue) Start() tea.Cmd {
	h.buildGradient()
	h.fixed = [hueH][hueW]bool{}
	h.moves = 0
	h.solved, h.recorded, h.holding = false, false, false
	h.cx, h.cy = 0, 0

	// The four corners are always pinned, plus a scattering of others to
	// give the eye something to work from.
	h.fixed[0][0] = true
	h.fixed[0][hueW-1] = true
	h.fixed[hueH-1][0] = true
	h.fixed[hueH-1][hueW-1] = true
	for i := 0; i < 3; i++ {
		h.fixed[h.rng.Intn(hueH)][h.rng.Intn(hueW)] = true
	}

	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			h.tiles[y][x] = y*hueW + x
		}
	}
	h.shuffle()

	// Start the cursor on a tile the player can actually pick up.
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			if !h.fixed[y][x] {
				h.cx, h.cy = x, y
				return nil
			}
		}
	}
	return nil
}

// shuffle permutes the movable tiles, retrying in the unlikely event that the
// shuffle hands the player a finished board.
func (h *hue) shuffle() {
	var slots [][2]int
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			if !h.fixed[y][x] {
				slots = append(slots, [2]int{x, y})
			}
		}
	}
	if len(slots) < 2 {
		return
	}
	for attempt := 0; attempt < 10; attempt++ {
		perm := h.rng.Perm(len(slots))
		for i, slot := range slots {
			from := slots[perm[i]]
			h.tiles[slot[1]][slot[0]] = from[1]*hueW + from[0]
		}
		if !h.isSolved() {
			return
		}
	}
}

func (h *hue) isSolved() bool {
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			if h.tiles[y][x] != y*hueW+x {
				return false
			}
		}
	}
	return true
}

// colorAt is the colour currently shown in a slot.
func (h *hue) colorAt(x, y int) rgb {
	idx := h.tiles[y][x]
	return h.target[idx/hueW][idx%hueW]
}

// pick picks up a tile, or swaps it with the one already held.
func (h *hue) pick() {
	if h.solved || h.fixed[h.cy][h.cx] {
		return
	}
	if !h.holding {
		h.holding = true
		h.hx, h.hy = h.cx, h.cy
		return
	}
	if h.hx == h.cx && h.hy == h.cy {
		h.holding = false // put it back down
		return
	}
	h.tiles[h.hy][h.hx], h.tiles[h.cy][h.cx] = h.tiles[h.cy][h.cx], h.tiles[h.hy][h.hx]
	h.holding = false
	h.moves++
	if h.isSolved() {
		h.solved = true
	}
}

// placed counts how many tiles are already where they belong.
func (h *hue) placed() int {
	n := 0
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			if h.tiles[y][x] == y*hueW+x {
				n++
			}
		}
	}
	return n
}

func (h *hue) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "r":
		return h.Start()
	case "esc":
		h.holding = false
	case "left", "h":
		h.cx = max(0, h.cx-1)
	case "right", "l":
		h.cx = min(hueW-1, h.cx+1)
	case "up", "k":
		h.cy = max(0, h.cy-1)
	case "down", "j":
		h.cy = min(hueH-1, h.cy+1)
	case "enter", " ":
		h.pick()
	}
	return nil
}

func (h *hue) Over() bool { return h.solved }

func (h *hue) Result() (int, bool, bool) {
	if !h.solved || h.recorded {
		return h.moves, true, false
	}
	h.recorded = true
	return h.moves, true, true // fewer moves is better
}

func (h *hue) Help() string {
	return helpStyle.Render("←↑↓→ move · space pick up and swap · r new puzzle")
}

func (h *hue) View(width, height int) string {
	const tileW, tileH = 8, 3

	var rows []string
	for y := 0; y < hueH; y++ {
		lines := make([]string, tileH)
		for x := 0; x < hueW; x++ {
			style := lipgloss.NewStyle().Background(lipgloss.Color(h.colorAt(x, y).hex()))
			mark := spaces(tileW)
			switch {
			case h.holding && x == h.hx && y == h.hy:
				mark = center("↕", tileW)
			case h.fixed[y][x]:
				mark = center("·", tileW)
			}
			for i := range lines {
				content := spaces(tileW)
				if i == tileH/2 {
					content = mark
				}
				cell := style.Render(content)
				if x == h.cx && y == h.cy {
					cell = cursorCell(style, content, i, tileW, tileH)
				}
				lines[i] += cell
			}
		}
		rows = append(rows, strings.Join(lines, "\n"))
	}
	board := boxStyle.Render(strings.Join(rows, "\n"))

	side := []string{
		labelStyle.Render("moves"), accentStyle.Render(fmt.Sprintf("%d", h.moves)), "",
		labelStyle.Render("in place"), valueStyle.Render(fmt.Sprintf("%d/%d", h.placed(), hueW*hueH)), "",
		subtleStyle.Render("· pinned tiles"), subtleStyle.Render("↕ held tile"),
	}
	if h.solved {
		side = append(side, "", okStyle.Render("solved!"),
			subtleStyle.Render(fmt.Sprintf("%d moves", h.moves)), subtleStyle.Render("r for a new puzzle"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}

// cursorCell draws a tile with a bracket around it so the selection is
// visible whatever colour the tile happens to be.
func cursorCell(style lipgloss.Style, content string, line, tileW, tileH int) string {
	runes := []rune(content)
	if len(runes) < 2 {
		return style.Render(content)
	}
	switch line {
	case 0:
		runes[0], runes[len(runes)-1] = '▛', '▜'
	case tileH - 1:
		runes[0], runes[len(runes)-1] = '▙', '▟'
	default:
		runes[0], runes[len(runes)-1] = '▌', '▐'
	}
	return style.Foreground(lipgloss.Color("231")).Render(string(runes))
}
