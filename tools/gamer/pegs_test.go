package main

import "testing"

func TestPegsStartingBoard(t *testing.T) {
	p := newPegs()
	p.Start()
	if p.count() != 32 || p.board[3][3] != pegsEmpty {
		t.Fatalf("%d pegs, centre %d", p.count(), p.board[3][3])
	}
	holes := 0
	for y := 0; y < pegsSize; y++ {
		for x := 0; x < pegsSize; x++ {
			if pegsOnBoard(x, y) {
				holes++
			}
		}
	}
	if holes != 33 {
		t.Errorf("the board has %d holes, want 33", holes)
	}
	// Four jumps open the game, all into the centre.
	n := 0
	for y := 0; y < pegsSize; y++ {
		for x := 0; x < pegsSize; x++ {
			for _, m := range p.movesFrom(point{x, y}) {
				n++
				if m.to != (point{3, 3}) {
					t.Errorf("an opening jump lands at %v", m.to)
				}
			}
		}
	}
	if n != 4 {
		t.Errorf("%d opening jumps", n)
	}
}

func TestPegsJumpAndUndo(t *testing.T) {
	p := newPegs()
	p.Start() // cursor starts on (3,5), which can jump up into the centre
	p.Update(key(" "))
	if p.count() != 31 || p.board[3][3] != pegsPeg || p.board[4][3] != pegsEmpty || p.board[5][3] != pegsEmpty {
		t.Fatal("space on a peg with one jump did not make it")
	}
	p.Update(key("u"))
	if p.count() != 32 || p.board[3][3] != pegsEmpty {
		t.Error("undo did not put the pegs back")
	}
	// Filling a hole: on the centre with four pegs able to land, nothing happens.
	p.cx, p.cy = 3, 3
	p.Update(key(" "))
	if p.count() != 32 {
		t.Error("an ambiguous fill made a move")
	}
}

func TestPegsHoldAndJumpInADirection(t *testing.T) {
	p := newPegs()
	p.Start()
	clearPegs(p)
	// A peg in the centre that can jump down or right; a spare peg far off.
	p.board[3][3], p.board[4][3], p.board[3][4], p.board[0][2] = pegsPeg, pegsPeg, pegsPeg, pegsPeg
	p.cx, p.cy = 3, 3
	p.Update(key(" "))
	if !p.held {
		t.Fatal("a peg with a choice of jumps was not picked up")
	}
	p.Update(key("up"))
	if !p.held || p.board[3][3] != pegsPeg {
		t.Fatal("an impossible direction moved the peg or dropped it")
	}
	p.Update(key("down"))
	if p.board[5][3] != pegsPeg || p.board[3][3] != pegsEmpty || p.board[4][3] != pegsEmpty || p.held {
		t.Error("jumping down did not move the held peg")
	}
}

func clearPegs(p *pegs) {
	for y := range p.board {
		for x := range p.board[y] {
			if p.board[y][x] != pegsOff {
				p.board[y][x] = pegsEmpty
			}
		}
	}
}

func TestPegsEndsWhenStuck(t *testing.T) {
	p := newPegs()
	p.Start()
	clearPegs(p)
	// Two pegs side by side next to the centre: one jump, then stuck with one peg in the middle.
	p.board[3][1], p.board[3][2] = pegsPeg, pegsPeg
	p.apply(pegsMove{point{1, 3}, point{2, 3}, point{3, 3}})
	if !p.over || !p.perfect() {
		t.Fatalf("over %v perfect %v", p.over, p.perfect())
	}
	score, lower, rec := p.Result()
	if score != 1 || !lower || !rec {
		t.Errorf("result %d lower=%v rec=%v", score, lower, rec)
	}
	p.takeBack()
	if p.over || p.count() != 2 {
		t.Error("undo after the end did not reopen the board")
	}
}
