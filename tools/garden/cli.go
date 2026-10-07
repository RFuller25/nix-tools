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
		"%d growing  ·  %d in flower  ·  %d species pressed  ·  %d cultivars  ·  tending since %s",
		growing, flowering, len(g.Herbarium), len(g.Cultivars), g.Created.Format("2 January 2006"))), width)

	return strings.Join(append([]string{head, ""}, append(rows, "", foot)...), "\n")
}

// renderCultivars lists the hybrid lines the garden has found, for a terminal.
func renderCultivars(g *Garden) string {
	if len(g.Cultivars) == 0 {
		return "No cultivars yet: cross two plants of a species and flower something unlike either.\n"
	}
	var b strings.Builder
	for _, c := range g.Cultivars {
		sp := c.SpeciesRef()
		if sp == nil {
			continue
		}
		mark := "◇"
		if c.Stable {
			mark = "◆"
		}
		fmt.Fprintf(&b, "%s %2d %-28s %-18s %-12s %s, %s, %s, %s\n", mark, c.ID, c.Name, sp.Common, c.Genome.Hex(),
			c.Genome.ColourName(), sp.HeightText(c.Genome), c.Genome.SpeedWord(), c.Genome.YieldWord())
	}
	fmt.Fprintf(&b, "\n%d cultivars, %d stable\n", len(g.Cultivars), g.StableLines())
	return b.String()
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
	parts = append(parts, fmt.Sprintf("%d gold", g.Gold))
	if n := g.SeedsInShed(); n > 0 {
		parts = append(parts, fmt.Sprintf("%d seeds", n))
	}

	return "❀ " + strings.Join(parts, " · ")
}

// findCultivar picks one of the garden's cultivars by number or by part of
// its name. A name that fits more than one is refused rather than guessed.
func findCultivar(g *Garden, query string) (Cultivar, error) {
	query = strings.TrimSpace(query)
	if len(g.Cultivars) == 0 {
		return Cultivar{}, fmt.Errorf("you have no cultivars yet: cross two plants and flower something new")
	}
	var hits []Cultivar
	for _, c := range g.Cultivars {
		if fmt.Sprint(c.ID) == query || strings.EqualFold(c.Name, query) {
			return c, nil
		}
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(query)) {
			hits = append(hits, c)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return Cultivar{}, fmt.Errorf("none of your cultivars is called %q (garden --cultivars lists them)", query)
	}
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = fmt.Sprintf("%d: %s", h.ID, h.Name)
	}
	return Cultivar{}, fmt.Errorf("%q fits several cultivars (%s); use its number", query, strings.Join(names, ", "))
}
