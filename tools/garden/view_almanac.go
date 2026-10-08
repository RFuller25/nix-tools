package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// rowKind says what an almanac row is.
type rowKind int

const (
	rowGuide rowKind = iota
	rowCultivar
	rowRibbon
	rowPlant
	rowCreature
	rowHeader // the heading of a category, which can be folded away
)

// almanacRow is one line of the almanac: a chapter of the guide, a plant, or
// one of the creatures that comes to visit them.
type almanacRow struct {
	Kind     rowKind
	Chapter  int // index into guideChapters, for guide rows
	Cultivar int // the cultivar's ID, for cultivar rows
	Ribbon   int // index into the garden's ribbons, for ribbon rows
	Species  *Species
	Creature creature
	Group    string // for header rows: the category it heads
}

func (r almanacRow) IsPlant() bool { return r.Kind == rowPlant }

// almanacRows is the whole book: the guide, every species, then the visitors.
func almanacRows(g *Garden) []almanacRow {
	return withHeaders(almanacItems(g))
}

// withHeaders puts a heading row in front of each run of rows that share a
// category.
func withHeaders(items []almanacRow) []almanacRow {
	out := make([]almanacRow, 0, len(items)+8)
	last := ""
	for _, r := range items {
		if gname := r.group(); gname != last {
			last = gname
			out = append(out, almanacRow{Kind: rowHeader, Group: gname})
		}
		out = append(out, r)
	}
	return out
}

func almanacItems(g *Garden) []almanacRow {
	chapters := guideChapters()
	rows := make([]almanacRow, 0, len(chapters)+len(AllSpecies())+len(creatureOrder))
	for i := range chapters {
		rows = append(rows, almanacRow{Kind: rowGuide, Chapter: i})
	}
	if g != nil {
		for _, c := range g.Cultivars {
			rows = append(rows, almanacRow{Kind: rowCultivar, Cultivar: c.ID})
		}
		for i := len(g.Fair.Ribbons) - 1; i >= 0; i-- {
			rows = append(rows, almanacRow{Kind: rowRibbon, Ribbon: i})
		}
	}
	for _, sp := range AllSpecies() {
		rows = append(rows, almanacRow{Kind: rowPlant, Species: sp})
	}
	for _, c := range creatureOrder {
		rows = append(rows, almanacRow{Kind: rowCreature, Creature: c})
	}
	return rows
}

// group is the heading a row sits under.
func (r almanacRow) group() string {
	switch r.Kind {
	case rowHeader:
		return r.Group
	case rowGuide:
		return "guide"
	case rowCultivar:
		return "your cultivars"
	case rowRibbon:
		return "ribbons"
	case rowPlant:
		return r.Species.Kind.String()
	}
	return "visitors"
}

// title is how the row reads in the list.
func (r almanacRow) title(g *Garden) string {
	switch r.Kind {
	case rowHeader:
		return strings.ToUpper(r.Group)
	case rowGuide:
		return guideChapters()[r.Chapter].Title
	case rowCultivar:
		if c := g.CultivarByID(r.Cultivar); c != nil {
			return c.Name
		}
		return "cultivar"
	case rowRibbon:
		if r.Ribbon < len(g.Fair.Ribbons) {
			rb := g.Fair.Ribbons[r.Ribbon]
			c, _ := categoryByID(rb.Category)
			return fmt.Sprintf("%s %s", ribbonGlyph(rb.Place), c.Name)
		}
		return "ribbon"
	case rowPlant:
		return r.Species.Common
	}
	return upperFirst(r.Creature.kind().name)
}

func (m model) viewAlmanac() string {
	formsGrown, formsTotal := m.g.FormsGrown()
	listWidth, narrow, rows := m.almanacMetrics()
	avail := max(3, m.height-5) // everything between the header and the footer
	vis := m.almanacVisible()

	// The scroll position is decided by fixAlmanac after each key; this only
	// keeps a stale one from running off the end.
	cursor := max(0, min(m.almanacCursor, len(vis)-1))
	scroll := max(0, min(m.almanacScroll, max(0, len(vis)-rows)))
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+rows {
		scroll = cursor - rows + 1
	}

	// One row of the list is one line, headings included, so the rows that are
	// drawn are exactly the rows that fit.
	var lines []string
	for i := scroll; i < len(vis) && i < scroll+rows; i++ {
		row := vis[i]
		selected := i == cursor
		if row.Kind == rowHeader {
			lines = append(lines, m.headerLine(row.Group, selected, listWidth))
			continue
		}
		marker := "  "
		style := valueStyle
		if selected {
			marker = titleStyle.Render("› ")
			style = titleStyle
		}
		seen := "  "
		switch row.Kind {
		case rowPlant:
			if !m.g.Unlocked(row.Species) {
				style = lockedStyle
			}
			if _, ok := m.g.Collected(row.Species); ok {
				seen = okStyle.Render("✓ ")
			}
		case rowCultivar:
			if c := m.g.CultivarByID(row.Cultivar); c != nil && c.Stable {
				seen = okStyle.Render("◆ ")
			} else {
				seen = subtleStyle.Render("◇ ")
			}
		case rowCreature:
			if _, ok := m.g.Sightings[row.Creature.kind().name]; ok {
				seen = okStyle.Render("✓ ")
			} else {
				style = lockedStyle
			}
		}
		lines = append(lines, "  "+marker+seen+style.Render(truncate(row.title(m.g), listWidth-8)))
	}

	detail := ""
	clipped := false
	if len(vis) > 0 {
		row := vis[cursor]
		if narrow {
			// No room for a card: one line about the selection under the list.
			if note := m.almanacNote(row, listWidth); note != "" {
				lines = append(lines, "", note)
			}
		} else {
			switch row.Kind {
			case rowHeader:
				detail = m.groupDetail(row.Group, listWidth, avail)
			case rowGuide:
				detail, clipped = m.guideDetail(row.Chapter, listWidth, avail)
			case rowRibbon:
				detail, clipped = m.ribbonDetail(row.Ribbon, listWidth, avail)
			case rowCultivar:
				detail, clipped = m.cultivarDetail(row.Cultivar, listWidth, avail)
			case rowPlant:
				detail, clipped = m.almanacDetail(row.Species, listWidth, avail)
			default:
				detail, clipped = m.creatureDetail(row.Creature, listWidth, avail)
			}
		}
	}

	head := fit(titleStyle.Render("❦ almanac")+subtleStyle.Render(fmt.Sprintf(
		"  ·  %d species  ·  ", len(AllSpecies())))+
		okStyle.Render(fmt.Sprintf("%d pressed", len(m.g.Herbarium)))+
		subtleStyle.Render(fmt.Sprintf(" (%d of %d forms)", formsGrown, formsTotal))+
		subtleStyle.Render("  ·  ")+
		okStyle.Render(fmt.Sprintf("%d of %d visitors seen", len(m.g.Sightings), len(creatureOrder)))+
		subtleStyle.Render(fmt.Sprintf("  ·  %d cultivars  ·  %d ribbons", len(m.g.Cultivars), len(m.g.Fair.Ribbons))), m.width)
	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	keys := "↑↓ browse · enter fold · C fold all · ←→ stage · v variety · esc back"
	if len(vis) > 0 {
		switch vis[cursor].Kind {
		case rowHeader:
			keys = "↑↓ browse · enter fold or open · C fold all · esc back"
		case rowCreature:
			keys = "↑↓ browse · enter fold · C fold all · ✓ seen in this garden · esc back"
		case rowCultivar:
			keys = "↑↓ browse · n rename · e share · enter fold · ◆ stable line · esc back"
		case rowRibbon:
			keys = "↑↓ browse · enter fold · C fold all · esc back"
		case rowGuide:
			keys = "↑↓ browse · enter fold · C fold all · pgup/pgdn read · esc back"
		}
	}
	if clipped {
		keys += " · pgup/pgdn read the page"
	}
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}

// headerLine is a category heading in the list: an arrow that says whether it
// is open, its name, and how many rows it holds.
func (m model) headerLine(group string, selected bool, width int) string {
	arrow := "▾"
	if m.g.Folded[group] {
		arrow = "▸"
	}
	count := subtleStyle.Render(fmt.Sprintf(" %d", m.groupCount(group)))
	if selected {
		return titleStyle.Render("› "+arrow+" "+strings.ToUpper(group)) + count
	}
	return "  " + labelStyle.Render(arrow+" "+strings.ToUpper(group)) + count
}

// almanacNote is the one line shown under the list in a window too narrow for
// a card.
func (m model) almanacNote(row almanacRow, width int) string {
	switch row.Kind {
	case rowHeader:
		if m.g.Folded[row.Group] {
			return fit(subtleStyle.Render("folded · enter opens it"), width)
		}
		return fit(subtleStyle.Render("enter folds it away"), width)
	case rowGuide:
		return fit(subtleStyle.Render(guideChapters()[row.Chapter].Lede), width)
	case rowPlant:
		return fit(latinStyle.Render(row.Species.Latin), width)
	case rowCultivar:
		if c := m.g.CultivarByID(row.Cultivar); c != nil {
			return fit(swatch(c.Genome.Hex(), 2)+" "+subtleStyle.Render(c.Genome.ColourName()), width)
		}
	case rowRibbon:
		if row.Ribbon < len(m.g.Fair.Ribbons) {
			return fit(subtleStyle.Render(fmt.Sprintf("%s place", ordinalPlace(m.g.Fair.Ribbons[row.Ribbon].Place))), width)
		}
	case rowCreature:
		return fit(subtleStyle.Render(row.Creature.kind().when), width)
	}
	return ""
}

// groupDetail is the card for a category heading: what is under it and how
// to fold it.
func (m model) groupDetail(group string, listWidth, avail int) string {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	n := m.groupCount(group)
	rows := []string{titleStyle.Render(strings.ToUpper(group)), subtleStyle.Render(fmt.Sprintf("%d in this category", n)), ""}
	switch group {
	case "guide":
		rows = append(rows, wrapped(valueStyle, "How the garden works: every feature, with the real numbers and keys.", width-2)...)
	case "your cultivars":
		rows = append(rows, wrapped(valueStyle, "Hybrid lines you have bred that flowered unlike any named form. ◆ marks a stable line.", width-2)...)
	case "ribbons":
		rows = append(rows, wrapped(valueStyle, "Your results at the weekly fair, newest first.", width-2)...)
	case "visitors":
		seen := len(m.g.Sightings)
		rows = append(rows, valueStyle.Render(fmt.Sprintf("%d of %d seen in this garden", seen, n)))
	default:
		pressed := 0
		for _, r := range m.almanac {
			if r.IsPlant() && r.group() == group {
				if _, ok := m.g.Collected(r.Species); ok {
					pressed++
				}
			}
		}
		rows = append(rows, valueStyle.Render(fmt.Sprintf("%d of %d pressed in the herbarium", pressed, n)))
	}
	state := "open. enter folds it away; C folds every category."
	if m.g.Folded[group] {
		state = "folded. enter opens it again; C folds or opens every category."
	}
	rows = append(rows, "")
	rows = append(rows, wrapped(subtleStyle, state, width-2)...)
	visible, _, _ := window(rows, 0, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n"))
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
	if lines := m.g.UniqueCultivarsOf(sp); len(lines) > 0 {
		rows = append(rows, "", labelStyle.Render("YOUR VARIETIES")+subtleStyle.Render("  y to pick, o to open"))
		for i, c := range lines {
			mark := subtleStyle.Render("◇ ")
			if c.Stable {
				mark = okStyle.Render("◆ ")
			}
			name := valueStyle.Render("‘" + c.Name + "’")
			if i == min(m.almanacOwn, len(lines)-1) {
				mark, name = titleStyle.Render("▸ "), titleStyle.Render("‘"+c.Name+"’")
			}
			rows = append(rows, mark+name+" "+swatch(c.Genome.Hex(), 2),
				subtleStyle.Render("  "+describeGenome(c.Genome)))
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

// guideDetail is one chapter of the guide, read in the pane beside the list.
func (m model) guideDetail(idx, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	ch := guideChapters()[idx]
	body := ch.Build(m, width-2)
	rows := append([]string{titleStyle.Render(ch.Title), subtleStyle.Render(ch.Lede), ""}, body...)
	visible, above, below := window(rows, m.cardScroll, max(1, avail-2))
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
