package main

import "testing"

func TestMinesFirstMoveIsAlwaysSafe(t *testing.T) {
	for seed := int64(0); seed < 25; seed++ {
		m := newMines(seed)
		m.Start()
		m.cx, m.cy = 7, 7
		m.reveal(7, 7)

		if m.lost {
			t.Fatalf("seed %d: the first square opened onto a mine", seed)
		}
		// The whole neighbourhood is kept clear, so the first click always
		// opens a patch rather than a lone number.
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if m.grid[7+dy][7+dx].mine {
					t.Fatalf("seed %d: a mine sits next to the first square", seed)
				}
			}
		}
	}
}

func TestMinesLaysTheRightNumber(t *testing.T) {
	m := newMines(1)
	m.Start()
	m.reveal(3, 3)

	count := 0
	for _, row := range m.grid {
		for _, c := range row {
			if c.mine {
				count++
			}
		}
	}
	if count != mineCount {
		t.Errorf("laid %d mines, want %d", count, mineCount)
	}
}

func TestMinesNeighbourCountsAreRight(t *testing.T) {
	m := newMines(2)
	m.Start()
	m.reveal(5, 5)

	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			if want := m.countNear(x, y); m.grid[y][x].near != want {
				t.Fatalf("cell %d,%d says %d neighbours, actually %d", x, y, m.grid[y][x].near, want)
			}
		}
	}
}

func TestMinesFloodFillOpensAPatch(t *testing.T) {
	m := newMines(3)
	m.Start()
	m.reveal(8, 7)

	opened := 0
	for _, row := range m.grid {
		for _, c := range row {
			if c.revealed {
				opened++
			}
			if c.revealed && c.mine {
				t.Fatal("the flood fill opened a mine")
			}
		}
	}
	if opened < 9 {
		t.Errorf("the first move opened only %d squares; the safe neighbourhood alone is 9", opened)
	}
}

func TestMinesFlagsBlockReveals(t *testing.T) {
	m := newMines(4)
	m.Start()
	m.reveal(2, 2) // lay the field

	// Find a covered square and flag it.
	var fx, fy int
	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			if !m.grid[y][x].revealed {
				fx, fy = x, y
			}
		}
	}
	m.cx, m.cy = fx, fy
	m.toggleFlag(fx, fy)
	if !m.grid[fy][fx].flagged {
		t.Fatal("flagging a covered square did nothing")
	}
	if m.flagsUsed() != 1 {
		t.Errorf("flags used = %d, want 1", m.flagsUsed())
	}

	m.reveal(fx, fy)
	if m.grid[fy][fx].revealed {
		t.Error("a flagged square was opened")
	}

	m.toggleFlag(fx, fy)
	if m.grid[fy][fx].flagged {
		t.Error("flagging twice did not clear the flag")
	}
}

func TestMinesSteppingOnAMineEndsIt(t *testing.T) {
	m := newMines(5)
	m.Start()
	m.reveal(4, 4)

	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			if m.grid[y][x].mine {
				m.reveal(x, y)
				if !m.lost {
					t.Fatal("opening a mine did not end the game")
				}
				if _, _, record := m.Result(); record {
					t.Error("a lost game should not be recorded")
				}
				return
			}
		}
	}
	t.Fatal("no mines on the board")
}

func TestMinesWinWhenEverySafeSquareIsOpen(t *testing.T) {
	m := newMines(6)
	m.Start()
	m.reveal(6, 6)

	for y := 0; y < minesH; y++ {
		for x := 0; x < minesW; x++ {
			if !m.grid[y][x].mine {
				m.grid[y][x].flagged = false
				m.reveal(x, y)
			}
		}
	}
	if m.lost {
		t.Fatal("opening only safe squares lost the game")
	}
	if !m.won {
		t.Fatal("opening every safe square did not win")
	}
	if !m.Over() {
		t.Error("a won game should report as over")
	}

	_, lower, record := m.Result()
	if !lower {
		t.Error("minesweeper is scored by time, so lower must be better")
	}
	if !record {
		t.Error("a win should be recorded")
	}
	if _, _, again := m.Result(); again {
		t.Error("the same win was recorded twice")
	}
}

func TestMinesChordOpensNeighbours(t *testing.T) {
	m := newMines(7)
	m.Start()
	m.reveal(5, 5)

	// Find a revealed number whose mines are all findable, flag them, then
	// chord and check the rest of the neighbourhood opened.
	for y := 1; y < minesH-1; y++ {
		for x := 1; x < minesW-1; x++ {
			c := m.grid[y][x]
			if !c.revealed || c.near == 0 {
				continue
			}
			covered := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					n := m.grid[y+dy][x+dx]
					if n.mine {
						m.grid[y+dy][x+dx].flagged = true
					}
					if !n.revealed && !n.mine {
						covered++
					}
				}
			}
			if covered == 0 {
				continue
			}
			m.chord(x, y)
			if m.lost {
				t.Fatal("chording with correct flags hit a mine")
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					n := m.grid[y+dy][x+dx]
					if !n.mine && !n.revealed {
						t.Fatalf("chord left %d,%d covered", x+dx, y+dy)
					}
				}
			}
			return
		}
	}
	t.Skip("this board had no chordable number after the first move")
}

func TestMinesChordNeedsMatchingFlags(t *testing.T) {
	m := newMines(8)
	m.Start()
	m.reveal(5, 5)

	for y := 1; y < minesH-1; y++ {
		for x := 1; x < minesW-1; x++ {
			if !m.grid[y][x].revealed || m.grid[y][x].near == 0 {
				continue
			}
			before := m.grid
			m.chord(x, y) // no flags placed at all
			if m.grid != before {
				t.Error("chording without enough flags opened squares anyway")
			}
			return
		}
	}
}

func TestMinesClockRunsOnlyWhilePlaying(t *testing.T) {
	m := newMines(9)
	m.Start()
	if m.elapsed() != 0 {
		t.Error("the clock started before the first move")
	}
	m.reveal(5, 5)
	if m.elapsed() < 0 {
		t.Error("negative elapsed time")
	}

	m.lost = true
	m.ended = m.started.Add(42_000_000_000) // 42s
	if got := int(m.elapsed().Seconds()); got != 42 {
		t.Errorf("a finished game reports %ds, want 42", got)
	}
}
