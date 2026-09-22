package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewShop() string {
	season := m.g.Season(m.now)

	// The detail card only appears when there is room for it beside the
	// list; otherwise the list has the window to itself.
	listWidth := 34
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(20, m.width-2)
	}
	avail := max(3, m.height-5)
	rows := avail

	if m.shopCursor < m.shopScroll {
		m.shopScroll = m.shopCursor
	}
	if m.shopCursor >= m.shopScroll+rows {
		m.shopScroll = m.shopCursor - rows + 1
	}

	var lines []string
	for i := m.shopScroll; i < len(m.shop) && i < m.shopScroll+rows; i++ {
		sp := m.shop[i]
		unlocked := m.g.Unlocked(sp)
		marker := "  "
		if i == m.shopCursor {
			marker = titleStyle.Render("› ")
		}

		name := truncate(sp.Common, listWidth-10)
		cost := fmt.Sprintf("%3d✦", sp.SeedCost)
		switch {
		case !unlocked:
			lines = append(lines, marker+lockedStyle.Render(pad(name, listWidth-10)+"  locked"))
		case m.g.Seeds < sp.SeedCost:
			lines = append(lines, marker+subtleStyle.Render(pad(name, listWidth-10))+" "+lockedStyle.Render(cost))
		default:
			style := valueStyle
			if !sp.LikesSeason(season) {
				style = subtleStyle
			}
			lines = append(lines, marker+style.Render(pad(name, listWidth-10))+" "+seedStyle.Render(cost))
		}
	}
	list := strings.Join(lines, "\n")

	var detail string
	clipped := false
	switch {
	case len(m.shop) == 0:
		detail = subtleStyle.Render("No seeds match this filter.")
	case narrow:
		// Too narrow for the card: a line about the selection instead.
		sp := m.shop[m.shopCursor]
		rows--
		lines = append(lines, "", fit(latinStyle.Render(sp.Latin)+subtleStyle.Render("  "+sp.SeasonNames()), listWidth))
	default:
		detail, clipped = m.shopDetail(m.shop[m.shopCursor], listWidth, avail)
	}

	filter := "whole rack"
	if m.shopSeason {
		filter = season.String() + " only"
	}
	head := fit(titleStyle.Render("✦ seed shed")+subtleStyle.Render(fmt.Sprintf(
		"  ·  %s  ·  %d seeds  ·  %d/%d species", filter, m.g.Seeds, m.unlockedCount(), len(AllSpecies()))), m.width)

	body := list
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail)
	}
	keys := "↑↓ browse · ←→ variety · enter sow · t season · esc back"
	if clipped {
		keys = "↑↓ browse · ←→ variety · enter sow · pgup/pgdn read the card · esc"
	}
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}

func (m model) unlockedCount() int {
	n := 0
	for _, sp := range AllSpecies() {
		if m.g.Unlocked(sp) {
			n++
		}
	}
	return n
}

func (m model) shopDetail(sp *Species, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 24 {
		width = 24
	}
	artRows := 6
	if avail < 26 {
		artRows = 4
	}
	art := strings.Join(renderVariety(sp, m.shopVariety, StageMature, sp.PaletteFor(m.shopVariety, nil),
		min(width, 24), artRows, 0, nil), "\n")

	forms := sp.Varieties()
	v := sp.Variety(m.shopVariety)
	grown := "  "
	if m.g.GrownForm(sp, m.shopVariety) {
		grown = okStyle.Render("✓ ")
	}

	rows := []string{
		titleStyle.Render(sp.Common),
		latinStyle.Render(sp.Latin),
		"",
		art,
		"",
		grown + valueStyle.Render("‘"+v.Name+"’") +
			subtleStyle.Render(fmt.Sprintf("  %d of %d  ←→", m.shopVariety+1, len(forms))),
	}
	for _, line := range wrapText(v.Note, max(16, width-2)) {
		rows = append(rows, subtleStyle.Render(line))
	}
	rows = append(rows, []string{
		"",
		field("family", sp.Family, width),
		field("blooms", sp.Bloom, width),
		field("sun", sp.Sun, width),
		field("water", sp.Water, width),
		field("height", sp.Height, width),
		field("rarity", rarityStyle(sp.Rarity).Render(sp.Rarity.String()), width),
		"",
	}...)
	rows = append(rows, effectLines(sp, width)...)
	if !m.g.Unlocked(sp) {
		rows = append(rows, "", warnStyle.Render(fmt.Sprintf("Unlocks after %d plants reach maturity (you have %d).", sp.Unlock, m.g.Matured)))
	}

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2)) // less the card's border
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}

func field(label, value string, width int) string {
	l := labelStyle.Render(pad(label, 9))
	return l + lipgloss.NewStyle().Width(max(10, width-11)).Render(valueStyle.Render(value))
}

func hours(h float64) string {
	if h < 1 {
		return fmt.Sprintf("%d minutes", int(h*60))
	}
	whole := int(h)
	mins := int((h - float64(whole)) * 60)
	if mins == 0 {
		return fmt.Sprintf("%d hours", whole)
	}
	return fmt.Sprintf("%dh %dm", whole, mins)
}
