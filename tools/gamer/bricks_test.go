package main

import "testing"

func TestBricksServeFromThePaddle(t *testing.T) {
	b := newBricks(1)
	b.Start()
	if !b.held {
		t.Fatal("the ball is not waiting on the paddle")
	}
	b.movePaddle(paddleStep)
	b.advance(0.5)
	if b.bx != float64(b.px)+paddleWidth/2.0 {
		t.Error("a held ball does not ride along with the paddle")
	}
	b.launch()
	if b.held || b.vy >= 0 {
		t.Errorf("serving did not send the ball up: vy %.2f", b.vy)
	}
}

func TestBricksBallBreaksBricksAndBouncesOffWalls(t *testing.T) {
	b := newBricks(2)
	b.Start()
	b.launch()
	before := b.bricksLeft()
	for i := 0; i < 300 && b.bricksLeft() == before; i++ {
		// Keep the paddle under the ball.
		b.px = max(0, min(bricksW-paddleWidth, int(b.bx)-paddleWidth/2))
		b.advance(bricksTick.Seconds())
		if b.bx < 0 || b.bx >= bricksW || b.by < 0 {
			t.Fatalf("the ball left the field at %.2f,%.2f", b.bx, b.by)
		}
	}
	if b.bricksLeft() >= before || b.score == 0 {
		t.Error("the ball never broke a brick")
	}
}

// A paddle that tracks the ball never drops it, and in time clears the wall.
func TestBricksTrackingPaddleClearsAWall(t *testing.T) {
	b := newBricks(3)
	b.Start()
	b.launch()
	for i := 0; i < 60000 && b.level == 1; i++ {
		b.px = max(0, min(bricksW-paddleWidth, int(b.bx)-paddleWidth/2))
		b.advance(bricksTick.Seconds())
		if b.held && b.level == 1 {
			b.launch()
		}
	}
	if b.level < 2 {
		t.Fatalf("after a long rally the wall still has %d bricks", b.bricksLeft())
	}
	if b.lives != bricksLives {
		t.Errorf("a perfect paddle lost %d balls", bricksLives-b.lives)
	}
	if !b.held {
		t.Error("a fresh wall does not start with a serve")
	}
}

func TestBricksMissedBallsCostLivesAndEndTheGame(t *testing.T) {
	b := newBricks(4)
	b.Start()
	for lost := 0; lost < bricksLives; lost++ {
		b.launch()
		b.bx, b.by, b.vx, b.vy = 1, bricksH-2, 0, 10
		b.px = bricksW - paddleWidth
		for i := 0; i < 100 && !b.held && !b.over; i++ {
			b.advance(bricksTick.Seconds())
		}
	}
	if !b.over || b.lives != 0 {
		t.Fatalf("over %v with %d lives after missing every ball", b.over, b.lives)
	}
	if _, lower, rec := b.Result(); lower || !rec {
		t.Error("a finished game of Bricks is not recorded as points")
	}
	if _, _, rec := b.Result(); rec {
		t.Error("the same game was recorded twice")
	}
}

func TestBricksPaddleStaysOnTheField(t *testing.T) {
	b := newBricks(5)
	b.Start()
	for i := 0; i < 40; i++ {
		b.movePaddle(-paddleStep)
	}
	if b.px != 0 {
		t.Errorf("paddle at %d", b.px)
	}
	b.movePaddle(bricksW * 2)
	if b.px != bricksW-paddleWidth {
		t.Errorf("paddle at %d", b.px)
	}
}
