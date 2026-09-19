package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Keeping the screen inside the terminal.
//
// A view taller than the window does not simply look wrong: the terminal
// scrolls, and everything at the top — the header, the first row of beds —
// is pushed out of reach where it cannot be scrolled back to. So every screen
// is windowed to the space available, and anything genuinely long scrolls
// under its own key.

// window returns the lines visible at a scroll offset, along with whether
// there is more above or below them.
func window(lines []string, scroll, height int) (visible []string, above, below bool) {
	if height < 1 {
		height = 1
	}
	scroll = clampScroll(scroll, len(lines), height)
	end := min(len(lines), scroll+height)
	return lines[scroll:end], scroll > 0, end < len(lines)
}

// clampScroll keeps a scroll offset inside what there is to look at.
func clampScroll(scroll, total, height int) int {
	if height < 1 {
		height = 1
	}
	maxScroll := max(0, total-height)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}

// clipHeight is the backstop: whatever a view builds, it never leaves the
// terminal taller than it is.
func clipHeight(view string, height int) string {
	if height < 1 {
		return view
	}
	lines := strings.Split(view, "\n")
	if len(lines) <= height {
		return view
	}
	return strings.Join(lines[:height], "\n")
}

// fitScreen is the last word before anything reaches the terminal: never
// wider than the window, never taller. It truncates through the styling
// rather than through the text, so a clipped line cannot leak an unfinished
// escape sequence or leave the colour turned on.
func fitScreen(view string, width, height int) string {
	style := lipgloss.NewStyle()
	if width > 0 {
		style = style.MaxWidth(width)
	}
	if height > 0 {
		style = style.MaxHeight(height)
	}
	return style.Render(view)
}

// scrollHint prefixes a key line when there is more to see.
func scrollHint(above, below bool, keys string) string {
	if !above && !below {
		return keys
	}
	return "↑↓ scroll · " + keys
}
