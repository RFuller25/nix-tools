package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// almanacRow is one line of the almanac: a plant, or one of the creatures
// that comes to visit them.
type almanacRow struct {
	Species  *Species
	Creature creature
	IsPlant  bool
}

// almanacRows is the whole book: every species, then the visitors.
func almanacRows() []almanacRow {
	rows := make([]almanacRow, 0, len(AllSpecies())+len(creatureOrder))
	for _, sp := range AllSpecies() {
		rows = append(rows, almanacRow{Species: sp, IsPlant: true})
	}
	for _, c := range creatureOrder {
		rows = append(rows, almanacRow{Creature: c})
	}
	return rows
}

// group is the heading a row sits under.
func (r almanacRow) group() string {
	if r.IsPlant {
		return r.Species.Kind.String()
	}
	return "visitors"
}

// title is how the row reads in the list.
func (r almanacRow) title() string {
	if r.IsPlant {
		return r.Species.Common
	}
	return upperFirst(r.Creature.kind().name)
}

func (m model) viewAlmanac() string {
	formsGrown, formsTotal := m.g.FormsGrown()
	listWidth := 30
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(20, m.width-2)
	}
	// Everything between the header and the footer.
	avail := max(3, m.height-5)
	rows := avail

	if m.almanacCursor < m.almanacScroll {
		m.almanacScroll = m.almanacCursor
	}
	if m.almanacCursor >= m.almanacScroll+rows {
		m.almanacScroll = m.almanacCursor - rows + 1
	}

	var lines []string
	lastGroup := ""
	for i := m.almanacScroll; i < len(m.almanac) && i < m.almanacScroll+rows; i++ {
		row := m.almanac[i]
		if g := row.group(); g != lastGroup {
			lastGroup = g
			lines = append(lines, labelStyle.Render(strings.ToUpper(g)))
		}
		marker := "  "
		style := valueStyle
		if i == m.almanacCursor {
			marker = titleStyle.Render("› ")
			style = titleStyle
		}

		seen := "  "
		if row.IsPlant {
			if !m.g.Unlocked(row.Species) {
				style = lockedStyle
			}
			if _, ok := m.g.Collected(row.Species); ok {
				seen = okStyle.Render("✓ ")
			}
		} else {
			if _, ok := m.g.Sightings[row.Creature.kind().name]; ok {
				seen = okStyle.Render("✓ ")
			} else {
				style = lockedStyle
			}
		}
		lines = append(lines, marker+seen+style.Render(truncate(row.title(), listWidth-6)))
	}

	detail := ""
	clipped := false
	if len(m.almanac) > 0 {
		row := m.almanac[m.almanacCursor]
		switch {
		case narrow && row.IsPlant:
			lines = append(lines, "", fit(latinStyle.Render(row.Species.Latin), listWidth))
		case narrow:
			lines = append(lines, "", fit(subtleStyle.Render(row.Creature.kind().when), listWidth))
		case row.IsPlant:
			detail, clipped = m.almanacDetail(row.Species, listWidth, avail)
		default:
			detail, clipped = m.creatureDetail(row.Creature, listWidth, avail)
		}
	}

	head := fit(titleStyle.Render("❦ almanac")+subtleStyle.Render(fmt.Sprintf(
		"  ·  %d species  ·  ", len(AllSpecies())))+
		okStyle.Render(fmt.Sprintf("%d pressed", len(m.g.Herbarium)))+
		subtleStyle.Render(fmt.Sprintf(" (%d of %d forms)", formsGrown, formsTotal))+
		subtleStyle.Render("  ·  ")+
		okStyle.Render(fmt.Sprintf("%d of %d visitors seen", len(m.g.Sightings), len(creatureOrder))), m.width)
	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	keys := "↑↓ species · ←→ stage · v variety · ✓ grown here · esc back"
	if len(m.almanac) > 0 && !m.almanac[m.almanacCursor].IsPlant {
		keys = "↑↓ browse · ✓ seen in this garden · esc back"
	}
	if clipped {
		keys += " · pgup/pgdn read the card"
	}
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}

// almanacDetail shows every drawn stage of a species side by side, with the
// selected stage highlighted, above its botany. On a short terminal the
// drawings shrink first, so the botany — which is the point of an almanac —
// stays on screen. It reports whether anything had to be left off.
func (m model) almanacDetail(sp *Species, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}

	artRows := 6
	switch {
	case avail < 22:
		artRows = 3
	case avail < 30:
		artRows = 4
	}

	stageW := 11
	fit := max(1, (width-2)/(stageW+1))
	var frames []string
	for i := 0; i < StageCount && i < fit; i++ {
		art := renderVariety(sp, m.almanacVariety, i, sp.PaletteFor(m.almanacVariety, nil), stageW, artRows, 0, nil)
		label := stageNames[i]
		style := subtleStyle
		if i == m.almanacStage {
			style = okStyle
		}
		block := strings.Join(art, "\n") + "\n" + soilLine(stageW, 0, true, phaseNoon, 0.5) + "\n" + pad(style.Render(center(label, stageW)), stageW)
		frames = append(frames, block)
	}
	strip := lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(frames)...)

	grown := subtleStyle.Render("not yet grown here")
	if at, ok := m.g.Collected(sp); ok {
		grown = okStyle.Render("✓ first flowered " + at.Format("2 January 2006"))
	}

	rows := []string{
		titleStyle.Render(sp.Common),
		latinStyle.Render(sp.Latin),
		subtleStyle.Render(sp.Family + " · " + rarityStyle(sp.Rarity).Render(sp.Rarity.String())),
		grown,
		"",
		strip,
		"",
		lipgloss.NewStyle().Width(max(20, width-2)).Render(valueStyle.Render(sp.Desc)),
		"",
		field("origin", sp.Origin, width),
		field("blooms", sp.Bloom, width),
		field("sun", sp.Sun, width),
		field("water", sp.Water, width),
		field("height", sp.Height, width),
		"",
	}
	// The forms this species comes in, ticked once grown.
	rows = append(rows, labelStyle.Render("VARIETIES")+subtleStyle.Render("  v to flick through"))
	for i, v := range sp.Varieties() {
		mark := subtleStyle.Render("· ")
		if m.g.GrownForm(sp, i) {
			mark = okStyle.Render("✓ ")
		}
		name := valueStyle.Render("‘" + v.Name + "’")
		if i == m.almanacVariety {
			name = titleStyle.Render("‘" + v.Name + "’")
		}
		rows = append(rows, mark+name)
		for _, line := range wrapText(v.Note, max(16, width-4)) {
			rows = append(rows, subtleStyle.Render("  "+line))
		}
	}
	rows = append(rows, "")
	rows = append(rows, effectLines(sp, width)...)
	rows = append(rows, "",
		lipgloss.NewStyle().Width(max(20, width-2)).Render(subtleStyle.Render("✎ "+sp.Note)),
	)
	if !m.g.Unlocked(sp) {
		rows = append(rows, warnStyle.Render(fmt.Sprintf("Unlocks at %d matured plants.", sp.Unlock)))
	}

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2)) // less the card's border
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}

func center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}

// creatureDetail is the almanac page for one of the garden's visitors: what
// brings it, when it comes, and whether it has been here yet.
func (m model) creatureDetail(c creature, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	k := c.kind()
	glyph := lipgloss.NewStyle().Foreground(lipgloss.Color(k.color))

	seen := subtleStyle.Render("not yet seen in this garden")
	if at, ok := m.g.Sightings[k.name]; ok {
		seen = okStyle.Render("✓ first seen " + at.Format("2 January 2006, 15:04"))
	}

	// A little motif rather than a plant drawing.
	motif := []string{
		"   " + glyph.Render(k.glyph) + "      " + glyph.Render(k.glyph),
		"      " + glyph.Render(k.glyph) + "   ",
		"   " + glyph.Render(k.glyph) + "      " + glyph.Render(k.glyph),
	}

	rows := []string{
		titleStyle.Render(upperFirst(k.name)),
		subtleStyle.Render("a visitor, not a plant"),
		seen,
		"",
	}
	rows = append(rows, motif...)
	rows = append(rows, "")
	rows = append(rows, labelStyle.Render("COMES FOR"))
	for _, line := range wrapText(k.comes, max(16, width-2)) {
		rows = append(rows, valueStyle.Render(line))
	}
	rows = append(rows, "", labelStyle.Render("WHEN"))
	for _, line := range wrapText(k.when, max(16, width-2)) {
		rows = append(rows, valueStyle.Render(line))
	}
	rows = append(rows, "", labelStyle.Render("STAYS"))
	rows = append(rows, valueStyle.Render(fmt.Sprintf("about %.0f seconds, crossing a bed in %.1fs", k.life, 1/k.speed)))
	rows = append(rows, "")
	for _, line := range wrapText(k.fact, max(16, width-2)) {
		rows = append(rows, subtleStyle.Render(line))
	}

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}
