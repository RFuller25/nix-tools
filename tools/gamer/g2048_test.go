package main

import "testing"

func TestSlideLine(t *testing.T) {
	cases := []struct {
		name    string
		in      [gridN]int
		want    [gridN]int
		score   int
		changed bool
	}{
		{"empty", [4]int{}, [4]int{}, 0, false},
		{"already packed", [4]int{2, 4, 8, 16}, [4]int{2, 4, 8, 16}, 0, false},
		{"gaps close up", [4]int{0, 2, 0, 4}, [4]int{2, 4, 0, 0}, 0, true},
		{"simple merge", [4]int{2, 2, 0, 0}, [4]int{4, 0, 0, 0}, 4, true},
		{"merge across a gap", [4]int{2, 0, 0, 2}, [4]int{4, 0, 0, 0}, 4, true},
		{"two merges", [4]int{2, 2, 4, 4}, [4]int{4, 8, 0, 0}, 12, true},
		{"a tile merges only once", [4]int{2, 2, 2, 2}, [4]int{4, 4, 0, 0}, 8, true},
		{"leading pair wins", [4]int{4, 4, 8, 0}, [4]int{8, 8, 0, 0}, 8, true},
		{"unequal neighbours stay", [4]int{2, 4, 2, 4}, [4]int{2, 4, 2, 4}, 0, false},
	}
	for _, c := range cases {
		got, res := slideLine(c.in)
		if got != c.want || res.score != c.score || res.changed != c.changed {
			t.Errorf("%s: slideLine(%v) = %v, %d, %v; want %v, %d, %v",
				c.name, c.in, got, res.score, res.changed, c.want, c.score, c.changed)
		}
		if res.biggest > 0 && res.score == 0 {
			t.Errorf("%s: reported a merge to %d but scored nothing", c.name, res.biggest)
		}
	}
}

// The merge sound is pitched by the biggest tile a move created, so that
// number has to be right.
func TestSlideLineReportsTheBiggestMerge(t *testing.T) {
	cases := []struct {
		in   [gridN]int
		want int
	}{
		{[4]int{2, 2, 0, 0}, 4},
		{[4]int{2, 2, 4, 4}, 8},
		{[4]int{8, 8, 2, 2}, 16},
		{[4]int{2, 4, 8, 16}, 0}, // nothing merged
	}
	for _, c := range cases {
		if _, res := slideLine(c.in); res.biggest != c.want {
			t.Errorf("slideLine(%v).biggest = %d, want %d", c.in, res.biggest, c.want)
		}
	}
}

func TestMoveDirections(t *testing.T) {
	setup := func() *g2048 {
		g := new2048(1)
		g.grid = [gridN][gridN]int{
			{2, 0, 0, 2},
			{0, 0, 0, 0},
			{0, 0, 0, 0},
			{4, 0, 0, 0},
		}
		return g
	}

	left := setup()
	left.move(dirLeft)
	if left.grid[0][0] != 4 {
		t.Errorf("left move did not merge the top row: %v", left.grid[0])
	}

	right := setup()
	right.move(dirRight)
	if right.grid[0][gridN-1] != 4 {
		t.Errorf("right move did not merge into the right edge: %v", right.grid[0])
	}

	up := setup()
	up.grid = [gridN][gridN]int{{2, 0, 0, 0}, {2, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}
	up.move(dirUp)
	if up.grid[0][0] != 4 {
		t.Errorf("up move did not merge the left column: %v", up.grid)
	}

	down := setup()
	down.grid = [gridN][gridN]int{{2, 0, 0, 0}, {2, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}}
	down.move(dirDown)
	if down.grid[gridN-1][0] != 4 {
		t.Errorf("down move did not merge into the bottom: %v", down.grid)
	}
}

func TestMoveScoresAndSpawns(t *testing.T) {
	g := new2048(2)
	g.grid = [gridN][gridN]int{{2, 2, 0, 0}}
	if !g.move(dirLeft) {
		t.Fatal("a merging move should report that the board changed")
	}
	if g.score != 4 {
		t.Errorf("score = %d, want 4", g.score)
	}
	if g.moves != 1 {
		t.Errorf("moves = %d, want 1", g.moves)
	}

	filled := 0
	for _, row := range g.grid {
		for _, v := range row {
			if v != 0 {
				filled++
			}
		}
	}
	if filled != 2 {
		t.Errorf("after merging two tiles and spawning one, %d tiles are on the board, want 2", filled)
	}
}

func TestBlockedMoveDoesNothing(t *testing.T) {
	g := new2048(3)
	g.grid = [gridN][gridN]int{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}
	before := g.grid
	if g.move(dirLeft) {
		t.Error("a move that changes nothing should report false")
	}
	if g.grid != before {
		t.Error("a blocked move altered the grid")
	}
	if g.moves != 0 {
		t.Error("a blocked move was counted")
	}
}

func TestGameOverDetection(t *testing.T) {
	g := new2048(4)
	g.grid = [gridN][gridN]int{
		{2, 4, 2, 4},
		{4, 2, 4, 2},
		{2, 4, 2, 4},
		{4, 2, 4, 2},
	}
	if g.canMove() {
		t.Error("a fully interlocked board should have no moves")
	}

	g.grid[3][3] = 4 // now the bottom right pair matches horizontally
	if !g.canMove() {
		t.Error("a matching neighbour should count as a move")
	}
}

func TestWinIsFlaggedButPlayContinues(t *testing.T) {
	g := new2048(5)
	g.grid = [gridN][gridN]int{{1024, 1024, 0, 0}}
	g.move(dirLeft)
	if !g.won {
		t.Error("reaching 2048 should be flagged")
	}
	if g.over {
		t.Error("reaching 2048 should not end the game")
	}
}

func TestStartDealsTwoTiles(t *testing.T) {
	g := new2048(6)
	g.Start()
	filled := 0
	for _, row := range g.grid {
		for _, v := range row {
			if v != 0 {
				filled++
			}
			if v != 0 && v != 2 && v != 4 {
				t.Errorf("a starting tile was %d, want 2 or 4", v)
			}
		}
	}
	if filled != 2 {
		t.Errorf("a new game deals %d tiles, want 2", filled)
	}
}

func TestResultRecordedOnce(t *testing.T) {
	g := new2048(7)
	g.Start()
	g.over = true
	g.score = 1234
	score, lower, record := g.Result()
	if !record || lower || score != 1234 {
		t.Errorf("Result() = %d, %v, %v", score, lower, record)
	}
	if _, _, again := g.Result(); again {
		t.Error("the same game was recorded twice")
	}
}
