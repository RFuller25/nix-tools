package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Two ways of looking at the garden without opening it: a postcard you can
// paste into a message or redirect to a file, and a one-line summary for a
// shell prompt or a status bar.

// renderPostcard draws the garden as it stands, with no cursor and no chrome.
func renderPostcard(g *Garden, now time.Time, width int) string {
	m := newModel(g, "", now)
	m.width, m.height = width, 200
	m.cursor = -1 // nothing is selected on a postcard

	season, weather, ph := g.Season(now), g.Weather(now), phaseAt(now)

	title := "❀ the garden"
	if g.Gardener != "" {
		title = "❀ " + g.Gardener + "'s garden"
	}
	head := fit(titleStyle.Render(title)+subtleStyle.Render(fmt.Sprintf("  ·  %s, %s  ·  %s  ·  %s",
		now.Format("2 January 2006"), ph, season, weather.Name())), width)

	// The garden is five beds across, which wants 79 columns. Asked for
	// less, the postcard wraps the beds into narrower blocks rather than
	// cropping the garden.
	perBlock := (width + cellGap) / (cellWidth + cellGap)
	if perBlock < 1 {
		perBlock = 1
	}
	if perBlock > plotCols {
		perBlock = plotCols
	}

	var rows []string
	for row := 0; row < g.Rows(); row++ {
		for start := 0; start < plotCols; start += perBlock {
			var cells []string
			for col := start; col < start+perBlock && col < plotCols; col++ {
				idx := row*plotCols + col
				if idx >= len(g.Plots) {
					break
				}
				cells = append(cells, m.renderCell(idx))
			}
			if len(cells) == 0 {
				continue
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(cells)...))
		}
	}

	growing, flowering := 0, 0
	for i := range g.Plots {
		p := &g.Plots[i]
		if p.Empty() {
			continue
		}
		growing++
		if p.Growth >= 1 && !p.Spent {
			flowering++
		}
	}
	foot := fit(subtleStyle.Render(fmt.Sprintf(
		"%d growing  ·  %d in flower  ·  %d species pressed  ·  tending since %s",
		growing, flowering, len(g.Herbarium), g.Created.Format("2 January 2006"))), width)

	return strings.Join(append([]string{head, ""}, append(rows, "", foot)...), "\n")
}

// renderStatus is the one-liner: what the garden would tell you in passing.
func renderStatus(g *Garden, now time.Time) string {
	growing, thirsty, weedy, ripe, flowering := 0, 0, 0, 0, 0
	for i := range g.Plots {
		p := &g.Plots[i]
		if p.Empty() {
			continue
		}
		growing++
		if p.Thirsty() {
			thirsty++
		}
		if p.Weedy() {
			weedy++
		}
		if p.Pods >= 1 {
			ripe++
		}
		if p.Growth >= 1 && !p.Spent {
			flowering++
		}
	}

	parts := []string{fmt.Sprintf("%d/%d growing", growing, len(g.Plots))}
	if flowering > 0 {
		parts = append(parts, fmt.Sprintf("%d in flower", flowering))
	}
	if thirsty > 0 {
		parts = append(parts, fmt.Sprintf("%d thirsty", thirsty))
	}
	if weedy > 0 {
		parts = append(parts, fmt.Sprintf("%d weedy", weedy))
	}
	if ripe > 0 {
		parts = append(parts, fmt.Sprintf("%d ripe", ripe))
	}
	parts = append(parts, fmt.Sprintf("%d seeds", g.Seeds))

	return "❀ " + strings.Join(parts, " · ")
}
