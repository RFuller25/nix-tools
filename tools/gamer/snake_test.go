package main

import "testing"

func TestSnakeStarts(t *testing.T) {
	s := newSnake(1)
	s.Start()
	if len(s.body) != 3 {
		t.Errorf("a new snake is %d long, want 3", len(s.body))
	}
	if s.dir != (point{1, 0}) {
		t.Errorf("a new snake heads %v, want right", s.dir)
	}
	if s.occupies(s.food) {
		t.Error("the first apple landed on the snake")
	}
}

func TestSnakeEatsAndGrows(t *testing.T) {
	s := newSnake(2)
	s.Start()
	head := s.body[0]
	s.food = point{head.x + 1, head.y}
	before := len(s.body)

	s.step()
	if s.score != 10 {
		t.Errorf("eating scored %d, want 10", s.score)
	}
	// Growth is paid out over the next steps, so the snake keeps getting
	// longer rather than jumping in size.
	s.step()
	s.step()
	if len(s.body) != before+2 {
		t.Errorf("snake is %d long after one apple, want %d", len(s.body), before+2)
	}
}

func TestSnakeHitsTheWall(t *testing.T) {
	s := newSnake(3)
	s.Start()
	s.body = []point{{snakeW - 1, 5}}
	s.dir = point{1, 0}
	s.step()
	if !s.over {
		t.Error("running into the right wall should end the game")
	}
}

func TestSnakeBitesItself(t *testing.T) {
	s := newSnake(4)
	s.Start()
	// A tight coil: moving down from the head runs into its own body.
	s.body = []point{{5, 5}, {5, 6}, {6, 6}, {6, 5}, {7, 5}}
	s.dir = point{0, 1}
	s.queued = nil
	s.step()
	if !s.over {
		t.Error("biting its own body should end the game")
	}
}

func TestSnakeMayFollowItsTail(t *testing.T) {
	s := newSnake(5)
	s.Start()
	// The tail square is vacated on the same step, so entering it is legal.
	s.body = []point{{5, 5}, {6, 5}, {6, 6}, {5, 6}}
	s.dir = point{0, 1}
	s.grow = 0
	s.food = point{1, 1}
	s.step()
	if s.over {
		t.Error("moving into the vacated tail square should be allowed")
	}
}

func TestSnakeWillNotReverse(t *testing.T) {
	s := newSnake(6)
	s.Start() // heading right
	s.turn(point{-1, 0})
	if len(s.queued) != 0 {
		t.Error("a reversal into the neck should be ignored")
	}

	s.turn(point{0, 1})
	s.turn(point{0, -1}) // a reversal of the queued turn, not the current one
	if len(s.queued) != 1 {
		t.Errorf("queued turns = %d, want 1: the second reverses the first", len(s.queued))
	}
}

func TestSnakeQueuesTurnsInOrder(t *testing.T) {
	s := newSnake(7)
	s.Start()
	s.turn(point{0, -1}) // up
	s.turn(point{-1, 0}) // then left
	s.step()
	if s.dir != (point{0, -1}) {
		t.Errorf("first step took direction %v, want up", s.dir)
	}
	s.step()
	if s.dir != (point{-1, 0}) {
		t.Errorf("second step took direction %v, want left", s.dir)
	}
}

func TestSnakeSpeedsUpToAFloor(t *testing.T) {
	s := newSnake(8)
	s.Start()
	quick := s.speed()
	s.body = make([]point, 60)
	if s.speed() >= quick {
		t.Error("a longer snake should move faster")
	}
	s.body = make([]point, 500)
	if s.speed() < 60 {
		t.Errorf("speed floor breached: %v", s.speed())
	}
}

func TestSnakeStaleTickIgnored(t *testing.T) {
	s := newSnake(9)
	s.Start()
	stale := s.gen - 1
	head := s.body[0]
	s.Update(snakeTickMsg{stale})
	if s.body[0] != head {
		t.Error("a tick from a previous round moved the snake")
	}
}
