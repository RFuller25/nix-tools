package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewAlmanac() string {
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
	lastKind := Kind(-1)
	for i := m.almanacScroll; i < len(m.almanac) && i < m.almanacScroll+rows; i++ {
		sp := m.almanac[i]
		if sp.Kind != lastKind {
			lastKind = sp.Kind
			lines = append(lines, labelStyle.Render(strings.ToUpper(sp.Kind.String())))
		}
		marker := "  "
		style := valueStyle
		if i == m.almanacCursor {
			marker = titleStyle.Render("› ")
			style = titleStyle
		}
		if !m.g.Unlocked(sp) {
			style = lockedStyle
		}
		pressed := "  "
		if _, ok := m.g.Collected(sp); ok {
			pressed = okStyle.Render("✓ ")
		}
		lines = append(lines, marker+pressed+style.Render(truncate(sp.Common, listWidth-6)))
	}

	detail := ""
	clipped := false
	if len(m.almanac) > 0 && !narrow {
		detail, clipped = m.almanacDetail(m.almanac[m.almanacCursor], listWidth, avail)
	} else if len(m.almanac) > 0 {
		sp := m.almanac[m.almanacCursor]
		lines = append(lines, "", fit(latinStyle.Render(sp.Latin), listWidth))
	}

	head := fit(titleStyle.Render("❦ almanac")+subtleStyle.Render(fmt.Sprintf(
		"  ·  %d species  ·  %d unlocked  ·  ", len(AllSpecies()), m.unlockedCount()))+
		okStyle.Render(fmt.Sprintf("%d pressed", len(m.g.Herbarium))), m.width)
	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	keys := "↑↓ species · ←→ life stage · space cycle stages · ✓ grown here · esc back"
	if clipped {
		keys = "↑↓ species · ←→ life stage · a taller window shows more · esc back"
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
		art := renderArt(sp, sp.Palette, i, stageW, artRows, 0)
		label := stageNames[i]
		style := subtleStyle
		if i == m.almanacStage {
			style = okStyle
		}
		block := strings.Join(art, "\n") + "\n" + soilLine(stageW, 0, true, phaseNoon) + "\n" + pad(style.Render(center(label, stageW)), stageW)
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
		lipgloss.NewStyle().Width(width).Render(valueStyle.Render(sp.Desc)),
		"",
		field("origin", sp.Origin, width),
		field("blooms", sp.Bloom, width),
		field("sun", sp.Sun, width),
		field("water", sp.Water, width),
		field("height", sp.Height, width),
		field("season", sp.SeasonNames(), width),
		field("life", lifeNote(sp), width),
		field("soil", sp.PrefersSoil().String(), width),
		field("matures", "about "+hours(sp.Hours), width),
		field("cost", fmt.Sprintf("%d seeds", sp.SeedCost), width),
		"",
		lipgloss.NewStyle().Width(width).Render(subtleStyle.Render("✎ " + sp.Note)),
	}
	if !m.g.Unlocked(sp) {
		rows = append(rows, warnStyle.Render(fmt.Sprintf("Unlocks at %d matured plants.", sp.Unlock)))
	}

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, _, below := window(lines, 0, max(1, avail-2)) // less the card's border
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), below
}

func center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}
