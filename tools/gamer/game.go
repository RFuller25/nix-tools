package main

import tea "github.com/charmbracelet/bubbletea"

// game is one playable thing in the arcade. Games keep their own state and
// their own clocks; the shell only routes messages and draws the chrome.
type game interface {
	// ID is the stable key used in the score file.
	ID() string
	// Name is what the menu shows.
	Name() string
	// Blurb is the one-line description under the menu entry.
	Blurb() string
	// Start begins a fresh round and returns any command it needs running,
	// such as a gravity tick.
	Start() tea.Cmd
	// Update handles a message while the game is on screen.
	Update(msg tea.Msg) tea.Cmd
	// View draws the board into the space available.
	View(width, height int) string
	// Help is the key reminder shown along the bottom.
	Help() string
	// Over reports that the round has finished.
	Over() bool
	// Result reports the round's score, whether a lower score is a better
	// one, and whether the round should be recorded at all.
	Result() (score int, lowerIsBetter bool, record bool)
	// ScoreLabel names the score in the menu, e.g. "points" or "moves".
	ScoreLabel() string
}
