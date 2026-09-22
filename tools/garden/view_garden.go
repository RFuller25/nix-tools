package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	cellInner = 13 // drawable width inside a bed's border
	cellGap   = 1
	artHeight = 5
	cellLines = artHeight + 3 // art, soil, name, stats
	cellWidth = cellInner + 2 // plus border
	cellTall  = cellLines + 2 // plus border
	maxCols   = 5
)

// gridCols is the width of the garden itself, which never changes: beds keep
// their places so that neighbours stay neighbours.
func (m model) gridCols() int { return plotCols }

func (m model) gridRows() int { return m.g.Rows() }

// visibleCols is how many columns of beds fit on screen. A narrow terminal
// scrolls across the garden rather than reflowing it.
func (m model) visibleCols() int {
	cols := (m.width + cellGap) / (cellWidth + cellGap)
	if cols < 1 {
		cols = 1
	}
	if cols > plotCols {
		cols = plotCols
	}
	return cols
}

// visibleRows is how many rows of beds fit under the header and footer.
func (m model) visibleRows() int {
	avail := m.height - 5 // header, divider, status, help
	rows := avail / cellTall
	if rows < 1 {
		rows = 1
	}
	if rows > m.gridRows() {
		rows = m.gridRows()
	}
	return rows
}

func (m *model) ensureVisible() {
	row, col := m.cursor/plotCols, m.cursor%plotCols

	vis := m.visibleRows()
	if row < m.scroll {
		m.scroll = row
	}
	if row >= m.scroll+vis {
		m.scroll = row - vis + 1
	}
	if maxScroll := m.gridRows() - vis; m.scroll > maxScroll {
		m.scroll = max(0, maxScroll)
	}

	wide := m.visibleCols()
	if col < m.scrollX {
		m.scrollX = col
	}
	if col >= m.scrollX+wide {
		m.scrollX = col - wide + 1
	}
	if maxScroll := plotCols - wide; m.scrollX > maxScroll {
		m.scrollX = max(0, maxScroll)
	}
}

func (m model) header() string {
	season := m.g.Season(m.now)
	w := m.g.Weather(m.now)

	left := titleStyle.Render("❀ garden")
	ph := m.phase()
	// Most-important facts first; the tail is dropped on a narrow terminal.
	bits := []string{
		labelStyle.Render(ph.Glyph()+" ") + valueStyle.Render(ph.String()),
		labelStyle.Render(season.Glyph()+" ") + valueStyle.Render(season.String()),
		labelStyle.Render(w.Glyph()+" ") + valueStyle.Render(w.Name()),
		seedStyle.Render(fmt.Sprintf("✦ %d seeds", m.g.Seeds)),
		subtleStyle.Render(fmt.Sprintf("%d grown", m.g.Matured)),
	}
	sep := subtleStyle.Render("  ·  ")

	for len(bits) > 0 {
		right := strings.Join(bits, sep)
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap >= 2 {
			return left + strings.Repeat(" ", gap) + right
		}
		bits = bits[:len(bits)-1]
	}
	return fit(left, m.width)
}

// fit trims a rendered line so it never overflows the terminal.
func fit(s string, width int) string {
	if width < 1 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return truncate(s, width)
}

// keyHints joins as many key reminders as the width allows.
func keyHints(width int, hints ...string) string {
	sep := " · "
	out := ""
	for _, h := range hints {
		candidate := h
		if out != "" {
			candidate = out + sep + h
		}
		if lipgloss.Width(candidate) > width {
			break
		}
		out = candidate
	}
	if out == "" && len(hints) > 0 {
		out = fit(hints[0], width)
	}
	return out
}

func (m model) divider() string {
	w := m.width
	if w < 1 {
		w = 1
	}
	return dividerStyle.Render(strings.Repeat("─", w))
}

func (m model) footer(keys string) string {
	line := ""
	switch {
	case m.saveErr != nil:
		line = errStyle.Render("save failed: " + m.saveErr.Error())
	case m.status != "":
		line = m.statusStyle.Render(m.status)
	default:
		line = subtleStyle.Render(m.ambient())
	}
	return fit(line, m.width) + "\n" + fit(helpStyle.Render(keys), m.width)
}

// ambient is the idle status line: a quiet nudge about the garden's state.
func (m model) ambient() string {
	thirsty, weedy, ripe, empty := 0, 0, 0, 0
	for i := range m.g.Plots {
		p := &m.g.Plots[i]
		if p.Empty() {
			empty++
			continue
		}
		if p.Thirsty() {
			thirsty++
		}
		if p.Weedy() {
			weedy++
		}
		if p.Pods >= 1 {
			ripe++
		}
	}
	switch {
	case thirsty > 0:
		return fmt.Sprintf("%d bed(s) are thirsty.", thirsty)
	case ripe > 0:
		return fmt.Sprintf("%d plant(s) have seed ready to gather (f).", ripe)
	case weedy > 0:
		return fmt.Sprintf("%d bed(s) could use weeding.", weedy)
	case empty == len(m.g.Plots):
		return "Bare soil, waiting. Press p to sow something."
	case m.visitorLine() != "":
		return m.visitorLine()
	case m.nightScent() != "":
		return m.nightScent()
	case m.wind.strongest() > 0.65:
		return "A gust runs through the beds."
	case m.g.Weather(m.now).Wet():
		return "Rain is doing the watering today."
	default:
		return "The garden is content."
	}
}

func (m model) viewGarden() string {
	vis := m.visibleRows()
	wide := m.visibleCols()
	rows := m.gridRows()

	var body []string
	for row := m.scroll; row < rows && row < m.scroll+vis; row++ {
		cells := make([]string, 0, wide)
		for col := m.scrollX; col < plotCols && col < m.scrollX+wide; col++ {
			idx := row*plotCols + col
			if idx >= len(m.g.Plots) {
				break
			}
			cells = append(cells, m.renderCell(idx))
		}
		if len(cells) == 0 {
			continue
		}
		body = append(body, lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(cells)...))
	}

	head := m.header()
	if rows > vis || wide < plotCols {
		hint := subtleStyle.Render(fmt.Sprintf("  rows %d-%d of %d", m.scroll+1, min(rows, m.scroll+vis), rows))
		if wide < plotCols {
			hint = subtleStyle.Render(fmt.Sprintf("  beds %d-%d of %d across", m.scrollX+1, m.scrollX+wide, plotCols))
		}
		if lipgloss.Width(head)+lipgloss.Width(hint) <= m.width {
			head += hint
		}
	}

	keys := keyHints(m.width, "←↑↓→ move", "p plant", "w water", "c weed", "f gather",
		"n name", "i info", "a almanac", "m music", "b new bed", "d pond",
		"W water all", "C weed all", "tab screens", "? help", "q quit")
	if m.naming {
		return strings.Join([]string{
			head,
			m.divider(),
			strings.Join(body, "\n"),
			m.divider(),
			m.input.View(),
			helpStyle.Render("enter to confirm · esc to cancel"),
		}, "\n")
	}

	// On a window too short even for one row of beds, the beds give way
	// rather than the header and the status line, which are what a gardener
	// in a small terminal most needs to read.
	grid := strings.Split(strings.Join(body, "\n"), "\n")
	if room := m.height - 5; room > 0 && len(grid) > room {
		grid = grid[:room]
	}

	return strings.Join([]string{
		head,
		m.divider(),
		strings.Join(grid, "\n"),
		m.divider(),
		m.footer(keys),
	}, "\n")
}

func joinWithGap(cells []string) []string {
	if len(cells) == 0 {
		return cells
	}
	out := make([]string, 0, len(cells)*2-1)
	for i, c := range cells {
		if i > 0 {
			out = append(out, strings.Repeat(" ", cellGap))
		}
		out = append(out, c)
	}
	return out
}

// renderCell draws one bed: its plant, the soil, its name and a stat strip.
func (m model) renderCell(idx int) string {
	p := &m.g.Plots[idx]
	selected := idx == m.cursor

	var lines []string
	if fx, ok := m.compost[idx]; ok && !fx.done(m.now) {
		// A plant on its way into the soil.
		lines = append(lines, fx.frame(cellInner, artHeight, m.now)...)
		lines = append(lines, soilLine(cellInner, p.Weeds, true, m.phase(), p.Richness))
		lines = append(lines, pad(compostStyle.Render(truncate("composting…", cellInner)), cellInner))
		lines = append(lines, pad(okStyle.Render(fmt.Sprintf("+%.0f%% soil", fx.Gain*100)), cellInner))

		border := plainBorder
		if idx == m.cursor {
			border = selBorder
		}
		return border.Render(strings.Join(lines, "\n"))
	}
	if sp := p.Species(); sp != nil {
		// Taller plants catch more of the gust than a seedling does.
		sway := m.wind.swayAt(idx%plotCols) * (0.45 + 0.55*p.Growth)
		stage, pal := appearance(sp, p, m.g.Season(m.now), m.phase())
		visitors := m.life.overlayFor(idx, cellInner, artHeight)
		lines = append(lines, renderArtWith(sp, pal, stage, cellInner, artHeight, sway, visitors)...)
	} else {
		// Bare ground still gets visitors passing over it.
		visitors := m.life.overlayFor(idx, cellInner, artHeight)
		for row := 0; row < artHeight; row++ {
			line := []rune(strings.Repeat(" ", cellInner))
			out := ""
			for col := range line {
				if glyph, ok := visitors[[2]int{row, col}]; ok {
					out += glyph
					continue
				}
				out += " "
			}
			lines = append(lines, out)
		}
	}
	if p.Pond {
		lines = append(lines, litStyle("74", m.phase()).Render(strings.Repeat("≈", cellInner)))
	} else {
		lines = append(lines, soilLine(cellInner, p.Weeds, !p.Empty(), m.phase(), p.Richness))
	}

	name := p.DisplayName()
	if p.Empty() {
		name = fmt.Sprintf("bed %d", idx+1)
		lines = append(lines, pad(bareSoilStyle.Render(truncate(name, cellInner)), cellInner))
		state := "empty"
		if p.Pond {
			state = "pond"
		}
		lines = append(lines, pad(subtleStyle.Render(truncate(state, cellInner)), cellInner))
	} else {
		nameStyle := valueStyle
		if selected {
			nameStyle = titleStyle
		}
		lines = append(lines, pad(nameStyle.Render(truncate(name, cellInner)), cellInner))
		lines = append(lines, pad(m.statStrip(p), cellInner))
	}

	border := plainBorder
	if selected {
		border = selBorder
	}
	return border.Render(strings.Join(lines, "\n"))
}

// statStrip packs growth, thirst, weeds and ripe pods into 13 columns.
func (m model) statStrip(p *Plot) string {
	growth := meter(p.Growth, 4, okStyle)
	water := meter(p.Moisture, 3, waterStyle)
	extra := ""
	switch {
	case p.Spent:
		extra = subtleStyle.Render("seed")
	case p.Pods >= 1:
		extra = seedStyle.Render(fmt.Sprintf("✦%d", int(p.Pods)))
	case p.Weeds > 0.45:
		extra = weedStyle.Render("⌄⌄")
	case p.Growth >= 1:
		extra = okStyle.Render("❀")
	default:
		extra = subtleStyle.Render(shortStage(p.Stage()))
	}
	return growth + " " + water + " " + extra
}

func shortStage(stage int) string {
	switch stage {
	case StageSeed:
		return "sd"
	case StageSprout:
		return "sp"
	case StageSeedling:
		return "sl"
	case StageBud:
		return "bd"
	default:
		return "mt"
	}
}

func pad(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 1 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
