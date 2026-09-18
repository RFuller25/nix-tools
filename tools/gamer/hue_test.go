package main

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestTilesLockWhenTheyReachTheirPlace(t *testing.T) {
	h := newHue(21)
	h.Start()

	// Find a loose tile and the slot it belongs in.
	var from, to [2]int
	found := false
	for y := 0; y < hueH && !found; y++ {
		for x := 0; x < hueW && !found; x++ {
			if !h.movable(x, y) {
				continue
			}
			want := h.tiles[y][x]
			hx, hy := want%hueW, want/hueW
			if h.movable(hx, hy) {
				from, to, found = [2]int{x, y}, [2]int{hx, hy}, true
			}
		}
	}
	if !found {
		t.Skip("this shuffle left no straightforward swap to test")
	}

	h.cx, h.cy = from[0], from[1]
	h.pick()
	h.cx, h.cy = to[0], to[1]
	h.pick()

	if !h.locked[to[1]][to[0]] {
		t.Error("a tile that reached its own slot was not locked")
	}
	if h.fixed[to[1]][to[0]] {
		t.Error("settling a tile should not mark it as one of the pinned anchors")
	}

	// And it stays put from now on.
	before := h.tiles
	h.cx, h.cy = to[0], to[1]
	h.pick()
	if h.holding {
		t.Error("a settled tile was picked up again")
	}
	if h.tiles != before {
		t.Error("the board changed when trying to move a settled tile")
	}
}

func TestSettledTilesCannotBeSwappedInto(t *testing.T) {
	h := newHue(22)
	h.Start()

	// Settle whatever the shuffle placed, then try to swap a loose tile onto
	// one of those slots.
	var locked, loose [2]int
	haveLocked, haveLoose := false, false
	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			switch {
			case h.locked[y][x] && !haveLocked:
				locked, haveLocked = [2]int{x, y}, true
			case h.movable(x, y) && !haveLoose:
				loose, haveLoose = [2]int{x, y}, true
			}
		}
	}
	if !haveLocked || !haveLoose {
		t.Skip("this shuffle produced nothing settled to test against")
	}

	h.cx, h.cy = loose[0], loose[1]
	h.pick()
	if !h.holding {
		t.Fatal("a loose tile could not be picked up")
	}
	before, moves := h.tiles, h.moves
	h.cx, h.cy = locked[0], locked[1]
	h.pick()

	if h.tiles != before {
		t.Error("a held tile was swapped onto a settled one")
	}
	if h.moves != moves {
		t.Error("a refused swap was counted as a move")
	}
	if !h.holding {
		t.Error("the held tile was dropped by a refused swap")
	}
}

func TestStartSettlesWhateverTheShuffleGotRight(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		h := newHue(seed)
		h.Start()

		for y := 0; y < hueH; y++ {
			for x := 0; x < hueW; x++ {
				correct := h.tiles[y][x] == y*hueW+x
				if correct && !h.fixed[y][x] && !h.locked[y][x] {
					t.Fatalf("seed %d: the tile at %d,%d starts in its own slot but is not locked", seed, x, y)
				}
				if !correct && h.locked[y][x] {
					t.Fatalf("seed %d: the tile at %d,%d is locked in the wrong slot", seed, x, y)
				}
			}
		}
		if !h.movable(h.cx, h.cy) {
			t.Fatalf("seed %d: the cursor starts on a tile that cannot be moved", seed)
		}
	}
}

// Locking must never strand a puzzle: with every settled tile out of play,
// there is always a swap left that settles another one.
func TestLockingNeverStrandsThePuzzle(t *testing.T) {
	for seed := int64(0); seed < 40; seed++ {
		h := newHue(seed)
		h.Start()

		swaps := 0
		for !h.isSolved() {
			if swaps > hueW*hueH*2 {
				t.Fatalf("seed %d: still unsolved after %d swaps", seed, swaps)
			}

			// Take the first tile out of place and send it home.
			var from, to [2]int
			found := false
			for y := 0; y < hueH && !found; y++ {
				for x := 0; x < hueW && !found; x++ {
					if h.tiles[y][x] != y*hueW+x {
						want := h.tiles[y][x]
						from, to, found = [2]int{x, y}, [2]int{want % hueW, want / hueW}, true
					}
				}
			}
			if !found {
				break
			}
			if !h.movable(from[0], from[1]) || !h.movable(to[0], to[1]) {
				t.Fatalf("seed %d: the swap that would settle a tile is blocked", seed)
			}

			h.cx, h.cy = from[0], from[1]
			h.pick()
			h.cx, h.cy = to[0], to[1]
			h.pick()
			swaps++
		}

		if !h.solved {
			t.Fatalf("seed %d: the board is complete but the game did not notice", seed)
		}
		for y := 0; y < hueH; y++ {
			for x := 0; x < hueW; x++ {
				if !h.fixed[y][x] && !h.locked[y][x] {
					t.Fatalf("seed %d: a solved board left %d,%d unlocked", seed, x, y)
				}
			}
		}
	}
}

// Every swap should settle at least the tile it sends home, so a careful
// player never has to undo anything.
func TestEverySwapMakesProgress(t *testing.T) {
	h := newHue(23)
	h.Start()

	for i := 0; i < 50 && !h.isSolved(); i++ {
		before := h.placed()
		var from, to [2]int
		found := false
		for y := 0; y < hueH && !found; y++ {
			for x := 0; x < hueW && !found; x++ {
				if h.tiles[y][x] != y*hueW+x {
					want := h.tiles[y][x]
					from, to, found = [2]int{x, y}, [2]int{want % hueW, want / hueW}, true
				}
			}
		}
		if !found {
			break
		}
		h.cx, h.cy = from[0], from[1]
		h.pick()
		h.cx, h.cy = to[0], to[1]
		h.pick()

		if h.placed() <= before {
			t.Fatalf("a swap sending a tile home left %d in place, was %d", h.placed(), before)
		}
	}
}

// solveHue plays a puzzle out and returns the command the winning swap gave
// back, which is what starts the victory wave.
func solveHue(t *testing.T, h *hue) tea.Cmd {
	t.Helper()
	var last tea.Cmd
	for guard := 0; !h.isSolved() && guard < hueW*hueH*4; guard++ {
		for y := 0; y < hueH; y++ {
			for x := 0; x < hueW; x++ {
				want := y*hueW + x
				if h.tiles[y][x] == want {
					continue
				}
				for sy := 0; sy < hueH; sy++ {
					for sx := 0; sx < hueW; sx++ {
						if h.tiles[sy][sx] != want {
							continue
						}
						h.cx, h.cy = sx, sy
						h.pick()
						h.cx, h.cy = x, y
						last = h.pick()
					}
				}
			}
		}
	}
	if !h.solved {
		t.Fatal("could not solve the puzzle")
	}
	return last
}

func TestWinHidesTheMarksAndTheCursor(t *testing.T) {
	h := newHue(31)
	h.Start()

	before := h.View(80, 30)
	if !strings.ContainsAny(before, "·✓") {
		t.Fatal("an unsolved board shows no pinned or settled marks")
	}

	solveHue(t, h)
	after := h.View(80, 30)
	for _, mark := range []string{"·", "✓", "↕", "▛", "▜", "▙", "▟", "▌", "▐"} {
		if strings.Contains(after, mark) {
			t.Errorf("the winning board still shows %q", mark)
		}
	}
	if !strings.Contains(after, "solved!") {
		t.Error("the winning board does not say it is solved")
	}
}

func TestWinStartsAWaveThatKeepsGoing(t *testing.T) {
	h := newHue(32)
	h.Start()

	if cmd := h.Update(hueTickMsg{gen: h.gen}); cmd != nil || h.frame != 0 {
		t.Error("the wave is running before the puzzle is solved")
	}

	if solveHue(t, h) == nil {
		t.Fatal("the winning swap did not start the wave")
	}

	for i := 1; i <= 3; i++ {
		cmd := h.Update(hueTickMsg{gen: h.gen})
		if cmd == nil {
			t.Fatalf("the wave stopped after %d frames", i)
		}
		if h.frame != i {
			t.Fatalf("frame = %d after %d ticks", h.frame, i)
		}
	}
}

func TestWaveStopsOnRestart(t *testing.T) {
	h := newHue(33)
	h.Start()
	solveHue(t, h)
	stale := h.gen

	h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if h.solved {
		t.Fatal("r did not start a new puzzle")
	}
	if h.frame != 0 {
		t.Errorf("the new puzzle starts on wave frame %d", h.frame)
	}
	if cmd := h.Update(hueTickMsg{gen: stale}); cmd != nil {
		t.Error("a tick from the finished round kept the wave alive")
	}
	if h.frame != 0 {
		t.Error("a stale tick advanced the new puzzle's wave")
	}
}

// The wave moves the colours round the wheel over time and along the board,
// but leaves the gradient's shape alone: every tile keeps its saturation and
// lightness, so the board still reads as the finished picture.
func TestWaveRollsTheHuesWithoutFlatteningThem(t *testing.T) {
	h := newHue(34)
	h.Start()
	solveHue(t, h)

	still := h.waveColorAt(1, 1)
	h.frame = 4
	moved := h.waveColorAt(1, 1)
	if still == moved {
		t.Error("the wave does not move over time")
	}

	a, b := h.waveColorAt(0, 0), h.waveColorAt(hueW-1, hueH-1)
	if a == b {
		t.Error("every tile shifts by the same amount, so there is no wave")
	}

	for y := 0; y < hueH; y++ {
		for x := 0; x < hueW; x++ {
			_, wantS, wantL := h.target[y][x].hsl()
			_, gotS, gotL := h.waveColorAt(x, y).hsl()
			if math.Abs(gotS-wantS) > 0.01 || math.Abs(gotL-wantL) > 0.01 {
				t.Errorf("tile %d,%d: saturation/lightness %.3f/%.3f, want %.3f/%.3f", x, y, gotS, gotL, wantS, wantL)
			}
		}
	}
}

func TestRotateHueGoesRoundTheWheel(t *testing.T) {
	c := hslToRGB(200, 0.6, 0.5)
	full := rotateHue(c, 360)
	if math.Abs(full.r-c.r) > 0.001 || math.Abs(full.g-c.g) > 0.001 || math.Abs(full.b-c.b) > 0.001 {
		t.Errorf("a full turn changed the colour: %v, want %v", full, c)
	}
	back := rotateHue(rotateHue(c, 90), -90)
	if math.Abs(back.r-c.r) > 0.001 || math.Abs(back.g-c.g) > 0.001 || math.Abs(back.b-c.b) > 0.001 {
		t.Errorf("rotating back gave %v, want %v", back, c)
	}
	if got, _, _ := rotateHue(c, 40).hsl(); math.Abs(got-240) > 0.5 {
		t.Errorf("hue after a 40 degree turn = %.2f, want 240", got)
	}
}

func TestHSLRoundTrip(t *testing.T) {
	for _, c := range []rgb{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {0.2, 0.4, 0.8}, {0.5, 0.5, 0.5}, {0, 0, 0}, {1, 1, 1}} {
		hDeg, s, l := c.hsl()
		got := hslToRGB(hDeg, s, l)
		if math.Abs(got.r-c.r) > 0.001 || math.Abs(got.g-c.g) > 0.001 || math.Abs(got.b-c.b) > 0.001 {
			t.Errorf("%v round-tripped to %v", c, got)
		}
	}
}
