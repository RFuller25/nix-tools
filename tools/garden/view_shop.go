package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewShop() string {
	season := m.g.Season(m.now)
	listWidth := 34
	if m.width < 70 {
		listWidth = max(24, m.width-4)
	}
	rows := max(3, m.height-8)

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
	if len(m.shop) > 0 {
		detail = m.shopDetail(m.shop[m.shopCursor], listWidth)
	} else {
		detail = subtleStyle.Render("No seeds match this filter.")
	}

	filter := "whole rack"
	if m.shopSeason {
		filter = season.String() + " only"
	}
	head := titleStyle.Render("✦ seed shed") + subtleStyle.Render(fmt.Sprintf("  ·  %s  ·  %d seeds in the tin  ·  %d/%d species",
		filter, m.g.Seeds, m.unlockedCount(), len(AllSpecies())))

	body := lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail)
	keys := "↑↓ browse · enter sow · t season filter · esc back · q garden"
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

func (m model) shopDetail(sp *Species, listWidth int) string {
	width := m.width - listWidth - 6
	if width < 24 {
		width = 24
	}
	art := strings.Join(renderArt(sp, StageMature, min(width, 24), 6, 0), "\n")

	rows := []string{
		titleStyle.Render(sp.Common),
		latinStyle.Render(sp.Latin),
		"",
		art,
		"",
		field("family", sp.Family, width),
		field("kind", sp.Kind.String(), width),
		field("season", sp.SeasonNames(), width),
		field("blooms", sp.Bloom, width),
		field("sun", sp.Sun, width),
		field("water", sp.Water, width),
		field("height", sp.Height, width),
		field("matures", fmt.Sprintf("about %s of good care", hours(sp.Hours)), width),
		field("cost", fmt.Sprintf("%d seeds", sp.SeedCost), width),
		field("rarity", rarityStyle(sp.Rarity).Render(sp.Rarity.String()), width),
	}
	if !m.g.Unlocked(sp) {
		rows = append(rows, "", warnStyle.Render(fmt.Sprintf("Unlocks after %d plants reach maturity (you have %d).", sp.Unlock, m.g.Matured)))
	}
	return cardBorder.Width(width).Render(strings.Join(rows, "\n"))
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
