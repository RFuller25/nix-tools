package main

import (
	"math"
	"testing"
)

func TestHueStartsScrambledWithAnchors(t *testing.T) {
	h := newHue(1)
	h.Start()

	if h.isSolved() {
		t.Error("a new puzzle started already solved")
	}
	corners := [][2]int{{0, 0}, {hueW - 1, 0}, {0, hueH - 1}, {hueW - 1, hueH - 1}}
	for _, c := range corners {
		if !h.fixed[c[1]][c[0]] {
			t.Errorf("corner %v is not pinned", c)
		}
		if h.tiles[c[1]][c[0]] != c[1]*hueW+c[0] {
			t.Errorf("pinned corner %v was shuffled away", c)
		}
	}
	if h.fixed[h.cy][h.cx] {
		t.Error("the cursor starts on a tile that cannot be moved")
	}
	if h.moves != 0 {
		t.Errorf("a new puzzle starts on %d moves", h.moves)
	}
}

// The gradient must actually be a gradient: neighbouring target tiles should
// be close in colour, and opposite corners far apart.
func TestGradientIsSmooth(t *testing.T) {
	h := newHue(2)
	h.Start()

	dist := func(a, b rgb) float64 {
		return math.Sqrt((a.r-b.r)*(a.r-b.r) + (a.g-b.g)*(a.g-b.g) + (a.b-b.b)*(a.b-b.b))
	}

	var worstNeighbour float64
	for y := 0; y < hueH; y++ {
		for x := 0; x+1 < hueW; x++ {
			worstNeighbour = math.Max(worstNeighbour, dist(h.target[y][x], h.target[y][x+1]))
		}
	}
	corners := dist(h.target[0][0], h.target[hueH-1][hueW-1])
	if worstNeighbour > corners {
		t.Errorf("neighbouring tiles (%.3f apart) are further apart than opposite corners (%.3f)", worstNeighbour, corners)
	}
}

func TestHueSwapCountsAndSolves(t *testing.T) {
	h := newHue(3)
	h.Start()

	// Solve it by hand: walk the board putting every tile in its place.
	guard := 0
	for !h.isSolved() && guard < hueW*hueH*4 {
		guard++
		for y := 0; y < hueH; y++ {
			for x := 0; x < hueW; x++ {
				want := y*hueW + x
				if h.tiles[y][x] == want {
					continue
				}
				// Find where the wanted tile currently sits and swap.
				for sy := 0; sy < hueH; sy++ {
					for sx := 0; sx < hueW; sx++ {
						if h.tiles[sy][sx] != want {
							continue
						}
						h.cx, h.cy = sx, sy
						h.pick()
						h.cx, h.cy = x, y
						h.pick()
					}
				}
			}
		}
	}

	if !h.solved {
		t.Fatal("solving the board by hand did not register as solved")
	}
	if h.moves == 0 {
		t.Error("solving took no moves")
	}
	if h.placed() != hueW*hueH {
		t.Errorf("%d of %d tiles are in place on a solved board", h.placed(), hueW*hueH)
	}

	moves, lower, record := h.Result()
	if !lower {
		t.Error("Hue is scored by fewest moves, so lower must be better")
	}
	if !record || moves != h.moves {
		t.Errorf("Result() = %d, record=%v", moves, record)
	}
	if _, _, again := h.Result(); again {
		t.Error("a solved puzzle was recorded twice")
	}
}

func TestHuePinnedTilesCannotMove(t *testing.T) {
	h := newHue(4)
	h.Start()

	// Park the cursor on a pinned tile and try to pick it up.
	var px, py int
	found := false
	for y := 0; y < hueH && !found; y++ {
		for x := 0; x < hueW && !found; x++ {
			if h.fixed[y][x] {
				px, py, found = x, y, true
			}
		}
	}
	h.cx, h.cy = px, py
	h.pick()
	if h.holding {
		t.Error("a pinned tile was picked up")
	}
	if h.moves != 0 {
		t.Error("trying to move a pinned tile counted as a move")
	}
}

func TestHuePutDownWithoutSwapping(t *testing.T) {
	h := newHue(5)
	h.Start()
	h.pick()
	if !h.holding {
		t.Fatal("a movable tile was not picked up")
	}
	h.pick() // the same tile again
	if h.holding {
		t.Error("picking the held tile again should put it back down")
	}
	if h.moves != 0 {
		t.Error("putting a tile back counted as a move")
	}
}

func TestHueColorAtFollowsTheTiles(t *testing.T) {
	h := newHue(6)
	h.Start()
	// Find two movable tiles and swap them; the colours must follow.
	var a, b [2]int
	var found []int
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			if !h.fixed[y][x] && len(found) < 2 {
				if len(found) == 0 {
					a = [2]int{x, y}
				} else {
					b = [2]int{x, y}
				}
				found = append(found, 1)
			}
		}
	}
	colorA := h.colorAt(a[0], a[1])
	colorB := h.colorAt(b[0], b[1])

	h.cx, h.cy = a[0], a[1]
	h.pick()
	h.cx, h.cy = b[0], b[1]
	h.pick()

	if h.colorAt(a[0], a[1]) != colorB || h.colorAt(b[0], b[1]) != colorA {
		t.Error("swapping tiles did not swap their colours")
	}
	if h.moves != 1 {
		t.Errorf("a swap counted as %d moves", h.moves)
	}
}

func TestHueIsRepeatableFromASeed(t *testing.T) {
	a, b := newHue(99), newHue(99)
	a.Start()
	b.Start()
	if a.tiles != b.tiles || a.fixed != b.fixed || a.target != b.target {
		t.Error("the same seed produced a different puzzle")
	}
}

func TestRGBHex(t *testing.T) {
	cases := []struct {
		in   rgb
		want string
	}{
		{rgb{0, 0, 0}, "#000000"},
		{rgb{1, 1, 1}, "#ffffff"},
		{rgb{2, -1, 0.5}, "#ff0080"}, // out-of-range values are clamped
	}
	for _, c := range cases {
		if got := c.in.hex(); got != c.want {
			t.Errorf("%v.hex() = %s, want %s", c.in, got, c.want)
		}
	}
}
