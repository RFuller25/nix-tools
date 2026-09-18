package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestTetrominoRotation(t *testing.T) {
	// The T piece points up at rest and right after one turn clockwise.
	piece := tetrominoes[2]
	if piece.name != "T" {
		t.Fatalf("expected T at index 2, got %s", piece.name)
	}
	got := piece.cellsAt(1, 0, 0)
	want := map[[2]int]bool{{2, 1}: true, {1, 0}: true, {1, 1}: true, {1, 2}: true}
	for _, c := range got {
		if !want[c] {
			t.Errorf("rotated T occupies unexpected cell %v", c)
		}
	}

	// The O piece is the same in every rotation.
	o := tetrominoes[1]
	for rot := 1; rot < 4; rot++ {
		base := map[[2]int]bool{}
		for _, c := range o.cellsAt(0, 0, 0) {
			base[c] = true
		}
		for _, c := range o.cellsAt(rot, 0, 0) {
			if !base[c] {
				t.Errorf("O piece moved when rotated to %d", rot)
			}
		}
	}

	// Four turns is where you started.
	for i, p := range tetrominoes {
		start, full := p.cellsAt(0, 0, 0), p.cellsAt(4, 0, 0)
		if start != full {
			t.Errorf("piece %d (%s) did not return to its starting shape after four turns", i, p.name)
		}
	}
}

func TestTetrisWallsAndFloor(t *testing.T) {
	g := newTetris(1)
	g.Start()
	g.cur, g.rot = 1, 0 // O piece, 2x2

	g.x, g.y = 0, 0
	if g.move(-1, 0) {
		t.Error("piece moved through the left wall")
	}
	g.x = boardW - 2
	if g.move(1, 0) {
		t.Error("piece moved through the right wall")
	}
	g.x, g.y = 4, boardH-2
	if g.move(0, 1) {
		t.Error("piece moved through the floor")
	}
}

func TestTetrisLineClearAndScore(t *testing.T) {
	g := newTetris(2)
	g.Start()
	g.score, g.lines, g.level = 0, 0, 1

	// Fill the bottom two rows apart from one column, then drop an I piece
	// standing on end into the gap.
	for y := boardH - 2; y < boardH; y++ {
		for x := 0; x < boardW; x++ {
			if x == boardW-1 {
				continue
			}
			g.board[y][x] = 1
		}
	}
	for y := boardH - 4; y < boardH-2; y++ {
		g.board[y][boardW-1] = 1
	}
	g.cur, g.rot = 0, 1 // I piece, vertical
	g.x, g.y = boardW-3, boardH-4
	g.lock()

	if g.lines != 2 {
		t.Errorf("cleared %d lines, want 2", g.lines)
	}
	if g.score != 300 {
		t.Errorf("scored %d for a double, want 300", g.score)
	}
	for x := 0; x < boardW; x++ {
		if g.board[boardH-1][x] != 0 && x != boardW-1 {
			t.Errorf("bottom row was not cleared at column %d", x)
		}
	}
}

func TestTetrisClearedRowsFallTogether(t *testing.T) {
	g := newTetris(3)
	g.Start()

	// A full row with a marker row above it: after the clear the marker
	// should have dropped by exactly one.
	g.board[boardH-3][2] = 5
	for x := 0; x < boardW; x++ {
		g.board[boardH-2][x] = 1
	}
	if cleared := g.clearLines(); cleared != 1 {
		t.Fatalf("cleared %d rows, want 1", cleared)
	}
	if g.board[boardH-2][2] != 5 {
		t.Error("the row above a cleared line did not fall into it")
	}
	if g.board[boardH-3][2] != 0 {
		t.Error("the row above a cleared line was left behind as well")
	}
}

func TestTetrisHardDrop(t *testing.T) {
	g := newTetris(4)
	g.Start()
	g.cur, g.rot, g.x, g.y = 1, 0, 4, 0 // O piece at the top
	g.score = 0

	before := g.y
	g.hardDrop()
	if g.score != 2*(boardH-2-before) {
		t.Errorf("hard drop scored %d, want %d", g.score, 2*(boardH-2-before))
	}
	if g.board[boardH-1][4] == 0 || g.board[boardH-1][5] == 0 {
		t.Error("hard-dropped piece did not settle on the floor")
	}
}

func TestTetrisHoldOncePerPiece(t *testing.T) {
	g := newTetris(5)
	g.Start()

	first := g.cur
	g.swapHold()
	if !g.held || g.hold != first {
		t.Fatalf("hold did not take the current piece")
	}
	swapped := g.cur
	g.swapHold()
	if g.cur != swapped {
		t.Error("hold was allowed twice for the same piece")
	}

	g.lock() // locking spawns a new piece and re-arms the hold
	if g.holdUsed {
		t.Error("hold was not re-armed after a new piece spawned")
	}
}

func TestTetrisBagGivesEveryPiece(t *testing.T) {
	g := newTetris(6)
	seen := map[int]int{}
	for i := 0; i < 7; i++ {
		seen[g.draw()]++
	}
	if len(seen) != 7 {
		t.Errorf("a bag of seven produced %d distinct pieces", len(seen))
	}
}

func TestTetrisGameOverWhenStackReachesTheTop(t *testing.T) {
	g := newTetris(7)
	g.Start()
	for y := 0; y < 4; y++ {
		for x := 0; x < boardW; x++ {
			g.board[y][x] = 1
		}
	}
	g.spawn()
	if !g.over {
		t.Error("spawning into a full board should end the game")
	}

	score, lower, record := g.Result()
	if !record || lower {
		t.Errorf("a finished game should be recorded as a high score, got record=%v lower=%v", record, lower)
	}
	if _, _, again := g.Result(); again {
		t.Error("the same game was offered for recording twice")
	}
	if score != g.score {
		t.Errorf("reported score %d, want %d", score, g.score)
	}
}

func TestTetrisIgnoresStaleTicks(t *testing.T) {
	g := newTetris(8)
	g.Start()
	stale := g.gen - 1
	g.Start() // a new round bumps the generation again

	y := g.y
	g.Update(tetrisTickMsg{stale})
	if g.y != y {
		t.Error("a tick from an earlier round moved the piece")
	}
	g.Update(tetrisTickMsg{g.gen})
	if g.y == y && !g.over {
		t.Error("the current round's tick did not apply gravity")
	}
}

func TestTetrisPauseStopsGravity(t *testing.T) {
	g := newTetris(9)
	g.Start()
	g.Update(key("p"))
	y := g.y
	g.Update(tetrisTickMsg{g.gen})
	if g.y != y {
		t.Error("gravity kept pulling while paused")
	}
	g.Update(key("p"))
	g.Update(tetrisTickMsg{g.gen})
	if g.y == y {
		t.Error("gravity did not resume after unpausing")
	}
}
