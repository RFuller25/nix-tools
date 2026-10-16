package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// definitionRows draws a cultivar's definition: for each gene a track from one
// end of its range to the other, with the band the definition allows filled in
// and the ideal marked, and the figures beside it. It is shown on the cultivar's
// page, on its card in the shop and while the definition is being edited.
func definitionRows(c *Cultivar, width int) []string {
	return definitionRowsAt(c, width, -1)
}

// definitionRowsAt is definitionRows with one gene marked as the one being edited.
func definitionRowsAt(c *Cultivar, width, editing int) []string {
	out := make([]string, 0, geneCount)
	tol := c.Tolerances()
	for i := 0; i < geneCount; i++ {
		lo, hi := c.Band(i)
		mark := "  "
		label := labelStyle.Render(pad(geneNames[i], 11))
		if i == editing {
			mark = titleStyle.Render("▸ ")
			label = titleStyle.Render(pad(geneNames[i], 11))
		}
		ideal := c.Genome.geneAt(i)
		var figures string
		if i == 0 {
			figures = fmt.Sprintf("%d° ±%d°  (%d° to %d°)", ideal, tol.At(i), lo, hi)
		} else {
			figures = fmt.Sprintf("%d ±%d  (%d to %d)", ideal, tol.At(i), lo, hi)
		}
		out = append(out, fit(mark+label+definitionTrack(c, i)+"  "+valueStyle.Render(figures), width))
	}
	return out
}

// definitionTrack is the bar for one gene.
func definitionTrack(c *Cultivar, gene int) string {
	const cells = 20
	ideal, tol := c.Genome.geneAt(gene), c.Tolerances().At(gene)
	var b strings.Builder
	for k := 0; k < cells; k++ {
		var inBand, isIdeal bool
		var colour string
		if gene == 0 {
			centre := (k*360 + 180) / cells
			inBand = iabs(hueDelta(ideal, centre)) <= tol+360/cells/2
			isIdeal = iabs(hueDelta(ideal, centre)) <= 360/cells/2
			colour = Genome{Hue: uint16(centre), Sat: uint8(max(int(c.Genome.Sat), 40)), Light: c.Genome.Light}.Hex()
		} else {
			cellLo, cellHi := k*100/cells, (k+1)*100/cells
			inBand = cellHi >= ideal-tol && cellLo <= ideal+tol
			isIdeal = ideal >= cellLo && ideal < cellHi || (ideal == 100 && k == cells-1)
		}
		switch {
		case isIdeal:
			b.WriteString(titleStyle.Render("◆"))
		case inBand && gene == 0:
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colour)).Render("█"))
		case inBand:
			b.WriteString(okStyle.Render("█"))
		default:
			b.WriteString(dividerStyle.Render("·"))
		}
	}
	return b.String()
}

// cultivarDetail is the almanac page for a line of your own: what it is, its
// definition, how it came to be, and what it can do for you.
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

	kind := "a " + sp.Common + " of your own"
	if c.Gifted {
		kind = "a " + sp.Common + " taken in from a friend" + fromText(c.From)
	}
	found := "found " + c.Found.Format("2 January 2006")
	if c.Gifted {
		found = "imported " + c.Found.Format("2 January 2006")
	}
	rows := []string{
		titleStyle.Render(c.Name),
		latinStyle.Render(sp.Latin) + subtleStyle.Render("  ·  "+kind),
		status,
		subtleStyle.Render(found + " · " + ordinal(max(1, c.Gen)) + " generation from a named form"),
		"",
		art,
		"",
	}

	defHead := labelStyle.Render("DEFINITION") + subtleStyle.Render("  a plant must fit every band to be this variety")
	if m.defining {
		defHead = titleStyle.Render("DEFINING") + subtleStyle.Render("  ↑↓ gene · ←→ tolerance · [ ] ideal · r reset · enter done")
	}
	rows = append(rows, defHead)
	editing := -1
	if m.defining {
		editing = m.defGene
	}
	rows = append(rows, definitionRowsAt(c, width, editing)...)
	rows = append(rows, "")
	rows = append(rows, wrapped(subtleStyle,
		"A seedling or a plant that strays outside any band is an off-type: it grows as an ordinary plant and is no longer counted as this variety. Press D to change the definition.", width-2)...)
	rows = append(rows, "", labelStyle.Render("IDEAL"))
	rows = append(rows, geneRows(sp, c.Genome, width)...)
	rows = append(rows, "")

	rep := reportLineage(c.History)
	rows = append(rows, labelStyle.Render("LINEAGE")+subtleStyle.Render("  t to read how it came about"))
	switch {
	case !rep.Known:
		rows = append(rows, wrapped(subtleStyle, "The record of this line begins here; earlier generations were not written down.", width-2)...)
		if c.Descent != "" {
			rows = append(rows, wrapped(valueStyle, "From "+c.Descent+".", width-2)...)
		}
	default:
		rows = append(rows, wrapped(valueStyle, fmt.Sprintf("%d generation(s) recorded, %d mutation(s), %d drift(s) noted. It began as %s.",
			len(c.History), len(rep.Mutations), len(rep.Drifts), stepSource(rep.Origin, sp)), width-2)...)
	}
	rows = append(rows, "", labelStyle.Render("IN YOUR GARDEN"),
		valueStyle.Render(fmt.Sprintf("%d growing now · %d seed(s) in the shed", growing, packets)),
		valueStyle.Render(fmt.Sprintf("the shop sells it for %s a seed (s, then the shop shelf)", goldLabel(c.Price()))), "")
	rows = append(rows, wrapped(subtleStyle, "Press n to name the line, e to make a code to share it, t for its lineage, D to define it and x to delete the record. Seed gathered from a plant of the line, and not crossed with anything unlike it, stays in the line while it fits the definition.", width-2)...)

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}
