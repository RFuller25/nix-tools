package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Echo: four pads light up in a sequence, each with its own note, and you play
// the sequence back. Every round adds one more step, and the playback quickens
// as it grows. One wrong pad ends the game; the score is the longest sequence
// played back whole.

const (
	echoPads      = 4
	echoTickMS    = 30
	echoGapMS     = 120 // dark time between lit pads in playback
	echoLitMS     = 420 // lit time for the first rounds
	echoLitMinMS  = 180 // and the quickest it gets
	echoFlashMS   = 160 // how long a pad you pressed stays lit
	echoPauseMS   = 650 // the breath before each playback
	echoFailPause = 900
)

type echoPhase int

const (
	echoShowing   echoPhase = iota // the game is playing the sequence
	echoListening                  // the player is playing it back
	echoOver
)

type echoTickMsg struct{ gen int }

// The pads: where they sit, the key that presses them, their colour and note.
var echoPadInfo = [echoPads]struct {
	name   string
	keys   []string
	colour string
	note   float64
}{
	{"up", []string{"up", "k", "w", "1"}, "114", 64},
	{"right", []string{"right", "l", "d", "2"}, "203", 69},
	{"down", []string{"down", "j", "s", "3"}, "75", 73},
	{"left", []string{"left", "h", "a", "4"}, "221", 76},
}

type echo struct {
	seq   []int
	phase echoPhase
	pos   int // the step being shown or expected next

	lit     int           // the pad lit now, or -1
	litLeft time.Duration // how much longer it stays lit
	wait    time.Duration // until the next thing happens in playback
	missed  int           // on a loss, the pad that should have been pressed

	best     int // the longest sequence played back whole
	gen      int
	recorded bool

	rng *rand.Rand
	snd sounder
}

func newEcho(seed int64) *echo {
	return &echo{rng: rand.New(rand.NewSource(seed)), snd: noSound{}, lit: -1, missed: -1}
}

func (e *echo) SetAudio(s sounder) { e.snd = s }

func (e *echo) ID() string         { return "echo" }
func (e *echo) Name() string       { return "Echo" }
func (e *echo) Blurb() string      { return "watch the pads light up, then play the sequence back" }
func (e *echo) ScoreLabel() string { return "steps" }

func (e *echo) Start() tea.Cmd {
	e.seq = nil
	e.best = 0
	e.recorded = false
	e.missed = -1
	e.nextRound()
	e.gen++
	return e.tick()
}

func (e *echo) tick() tea.Cmd {
	gen := e.gen
	return tea.Tick(echoTickMS*time.Millisecond, func(time.Time) tea.Msg { return echoTickMsg{gen} })
}

// nextRound adds a step and starts the playback.
func (e *echo) nextRound() {
	e.seq = append(e.seq, e.rng.Intn(echoPads))
	e.phase = echoShowing
	e.pos = 0
	e.lit = -1
	e.wait = echoPauseMS * time.Millisecond
}

// litFor is how long each pad shows in playback: shorter as the sequence grows.
func (e *echo) litFor() time.Duration {
	ms := echoLitMS - 20*(len(e.seq)-1)
	return time.Duration(max(echoLitMinMS, ms)) * time.Millisecond
}

func (e *echo) light(pad int, d time.Duration) {
	e.lit, e.litLeft = pad, d
	e.snd.Play(sfxPad(echoPadInfo[pad].note, d.Seconds())...)
}

// advance runs the clock on by d.
func (e *echo) advance(d time.Duration) {
	if e.lit >= 0 {
		e.litLeft -= d
		if e.litLeft <= 0 {
			e.lit = -1
		}
	}
	if e.phase != echoShowing {
		return
	}
	e.wait -= d
	if e.wait > 0 {
		return
	}
	if e.pos >= len(e.seq) {
		// Playback done: over to the player once the last pad goes dark.
		if e.lit < 0 {
			e.phase = echoListening
			e.pos = 0
		}
		return
	}
	e.light(e.seq[e.pos], e.litFor())
	e.pos++
	e.wait = e.litFor() + echoGapMS*time.Millisecond
}

// press is the player hitting a pad.
func (e *echo) press(pad int) {
	if e.phase != echoListening {
		return
	}
	if pad != e.seq[e.pos] {
		e.missed = e.seq[e.pos]
		e.lit, e.litLeft = pad, echoFailPause*time.Millisecond
		e.phase = echoOver
		e.snd.Play(sfxGameOver()...)
		return
	}
	e.light(pad, echoFlashMS*time.Millisecond)
	e.pos++
	if e.pos == len(e.seq) {
		e.best = len(e.seq)
		e.nextRound()
	}
}

func (e *echo) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case echoTickMsg:
		if msg.gen != e.gen {
			return nil
		}
		e.advance(echoTickMS * time.Millisecond)
		if e.phase == echoOver && e.lit < 0 {
			return nil // nothing left to animate
		}
		return e.tick()
	case tea.KeyMsg:
		k := msg.String()
		if k == "r" {
			return e.Start()
		}
		for i, p := range echoPadInfo {
			for _, pk := range p.keys {
				if k == pk {
					e.press(i)
					return nil
				}
			}
		}
	}
	return nil
}

func (e *echo) Over() bool { return e.phase == echoOver }

func (e *echo) Result() (int, bool, bool) {
	if e.phase != echoOver || e.recorded || e.best == 0 {
		return e.best, false, false
	}
	e.recorded = true
	return e.best, false, true
}

func (e *echo) Help() string {
	return helpStyle.Render("←↑↓→ (or 1–4) press a pad · r restart")
}

const echoPadW, echoPadH = 14, 3

func (e *echo) padBox(i int) string {
	p := echoPadInfo[i]
	fill, edge := "░", lipgloss.Color("238")
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(p.colour)).Faint(true)
	switch {
	case e.phase == echoOver && i == e.missed:
		fill, edge = "▒", lipgloss.Color(p.colour)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(p.colour))
	case i == e.lit && e.phase == echoOver:
		fill, edge = "╳", lipgloss.Color("203")
		style = warnStyle
	case i == e.lit:
		fill, edge = "█", lipgloss.Color(p.colour)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(p.colour)).Bold(true)
	}
	lines := make([]string, echoPadH)
	for y := range lines {
		lines[y] = style.Render(strings.Repeat(fill, echoPadW))
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(edge).
		Render(strings.Join(lines, "\n"))
}

func (e *echo) View(width, height int) string {
	blank := spaces(echoPadW + 2)
	gap := "  "
	top := lipgloss.JoinHorizontal(lipgloss.Top, blank, gap, e.padBox(0), gap, blank)
	mid := lipgloss.JoinHorizontal(lipgloss.Top, e.padBox(3), gap, e.status(), gap, e.padBox(1))
	bot := lipgloss.JoinHorizontal(lipgloss.Top, blank, gap, e.padBox(2), gap, blank)
	board := lipgloss.JoinVertical(lipgloss.Left, top, mid, bot)

	side := []string{
		labelStyle.Render("round"), accentStyle.Render(fmt.Sprintf("%d", len(e.seq))), "",
		labelStyle.Render("best run"), valueStyle.Render(fmt.Sprintf("%d", e.best)), "",
	}
	if e.phase == echoListening {
		side = append(side, labelStyle.Render("played"), valueStyle.Render(fmt.Sprintf("%d / %d", e.pos, len(e.seq))))
	}
	if e.phase == echoOver {
		side = append(side, warnStyle.Render("wrong pad"),
			subtleStyle.Render("it was "+echoPadInfo[e.missed].name),
			subtleStyle.Render("r to play again"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, board, "  ", strings.Join(side, "\n"))
}

// status is the word in the middle of the pads.
func (e *echo) status() string {
	lines := []string{"", "", ""}
	switch e.phase {
	case echoShowing:
		lines[1] = subtleStyle.Render("watch")
	case echoListening:
		lines[1] = accentStyle.Render("your turn")
	case echoOver:
		lines[1] = warnStyle.Render("game over")
	}
	for i := range lines {
		lines[i] = center(lines[i], echoPadW+2)
	}
	return strings.Join(lines, "\n")
}
