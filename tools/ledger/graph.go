package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var palette = []lipgloss.Color{"39", "208", "42", "205", "220", "135", "81", "203"}

func optColor(i int) lipgloss.Color { return palette[i%len(palette)] }

func sum(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

// pct is an option's share of the pool, 0-100.
func pct(total, pool int) float64 {
	if pool <= 0 {
		return 0
	}
	return float64(total) * 100 / float64(pool)
}

// renderGraph draws each option's share of the pool over time. The line holds
// its value between stakes, so a quiet bet is a flat line.
func renderGraph(points []Point, n, width, height int) string {
	const axis = 5
	if width < axis+8 || height < 3 || n == 0 {
		return ""
	}
	plotW := width - axis

	var valid []Point
	for _, p := range points {
		if len(p.Totals) == n && sum(p.Totals) > 0 {
			valid = append(valid, p)
		}
	}
	if len(valid) == 0 {
		return dim.Render("no stakes yet")
	}
	t0, t1 := valid[0].TS, valid[len(valid)-1].TS

	type cell struct {
		r rune
		c int
	}
	grid := make([][]cell, height)
	for y := range grid {
		grid[y] = make([]cell, plotW)
	}
	rowOf := func(p float64) int {
		y := int((100-p)/100*float64(height-1) + 0.5)
		return min(max(y, 0), height-1)
	}
	for o := 0; o < n; o++ {
		prev := -1
		j := 0
		for x := 0; x < plotW; x++ {
			ts := t0
			if t1 > t0 && plotW > 1 {
				ts = t0 + (t1-t0)*int64(x)/int64(plotW-1)
			}
			for j+1 < len(valid) && valid[j+1].TS <= ts {
				j++
			}
			y := rowOf(pct(valid[j].Totals[o], sum(valid[j].Totals)))
			if prev >= 0 && prev != y {
				lo, hi := min(prev, y), max(prev, y)
				for yy := lo; yy <= hi; yy++ {
					grid[yy][x] = cell{'┃', o + 1}
				}
			} else {
				grid[y][x] = cell{'━', o + 1}
			}
			prev = y
		}
	}

	var b strings.Builder
	for y := 0; y < height; y++ {
		label := "     "
		switch y {
		case 0:
			label = "100% "
		case height / 2:
			label = " 50% "
		case height - 1:
			label = "  0% "
		}
		b.WriteString(dim.Render(label))
		for x := 0; x < plotW; x++ {
			c := grid[y][x]
			if c.c == 0 {
				b.WriteString(dim.Render("·"))
				continue
			}
			b.WriteString(lipgloss.NewStyle().Foreground(optColor(c.c - 1)).Render(string(c.r)))
		}
		b.WriteString("\n")
	}
	left := time.Unix(t0, 0).Format("Jan 2 15:04")
	right := time.Unix(t1, 0).Format("Jan 2 15:04")
	gap := plotW - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(dim.Render(strings.Repeat(" ", axis) + left + strings.Repeat(" ", gap) + right))
	return b.String()
}

// oddsLine is "yes 62% (x1.6)": share of the pool and what a BB on it returns.
func oddsLine(total, pool int) string {
	if total == 0 {
		return fmt.Sprintf("%3.0f%%  (no money)", pct(total, pool))
	}
	return fmt.Sprintf("%3.0f%%  x%.2f", pct(total, pool), float64(pool)/float64(total))
}
