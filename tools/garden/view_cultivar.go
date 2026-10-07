package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// cultivarDetail is the almanac page for a hybrid line the garden has found.
func (m model) cultivarDetail(id, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	c := m.g.CultivarByID(id)
	if c == nil || c.SpeciesRef() == nil {
		return cardBorder.Width(width).Render(subtleStyle.Render("This line is lost to the record.")), false
	}
	sp := c.SpeciesRef()
	artRows := 6
	if avail < 26 {
		artRows = 4
	}
	art := strings.Join(renderGene(sp, c.Variety, c.Genome, StageMature, sp.PaletteForGenome(c.Variety, c.Genome, nil),
		min(width, 24), artRows, 0, nil), "\n")

	status := subtleStyle.Render("◇ not yet settled — breed it true " + fmt.Sprint(stableRuns) + " generations running")
	if c.Stable {
		status = okStyle.Render("◆ a stable line — it breeds true")
	}
	growing, packets := 0, 0
	for i := range m.g.Plots {
		if m.g.Plots[i].Line == c.ID {
			growing++
		}
	}
	for _, pk := range m.g.Shed {
		if pk.Line == c.ID {
			packets += pk.Count
		}
	}

	rows := []string{
		titleStyle.Render(c.Name),
		latinStyle.Render(sp.Latin) + subtleStyle.Render("  ·  a "+sp.Common+" of your own"),
		status,
		subtleStyle.Render("found " + c.Found.Format("2 January 2006") + " · " + ordinal(max(1, c.Gen)) + " generation from a named form"),
		"",
		art,
		"",
		labelStyle.Render("GENES"),
	}
	rows = append(rows, geneRows(sp, c.Genome, width)...)
	rows = append(rows, "")
	if c.Descent != "" {
		rows = append(rows, labelStyle.Render("DESCENT"))
		rows = append(rows, wrapped(valueStyle, "From "+c.Descent+".", width-2)...)
		rows = append(rows, "")
	}
	rows = append(rows, labelStyle.Render("IN YOUR GARDEN"),
		valueStyle.Render(fmt.Sprintf("%d growing now · %d seed(s) in the shed", growing, packets)), "")
	rows = append(rows, wrapped(subtleStyle, "Press n to give the line a name of your own, and e to make a code to share it with a friend. Seed gathered from a plant of the line, and not crossed with anything unlike it, stays in the line.", width-2)...)

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}

var _ = lipgloss.Width
