package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewAlmanac() string {
	listWidth := 30
	if m.width < 66 {
		listWidth = max(20, m.width-4)
	}
	rows := max(3, m.height-8)

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
		lines = append(lines, marker+style.Render(truncate(sp.Common, listWidth-4)))
	}

	detail := ""
	if len(m.almanac) > 0 {
		detail = m.almanacDetail(m.almanac[m.almanacCursor], listWidth)
	}

	head := titleStyle.Render("❦ almanac") + subtleStyle.Render(fmt.Sprintf("  ·  %d species  ·  %d unlocked", len(AllSpecies()), m.unlockedCount()))
	body := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(lines, "\n"), "  ", detail)
	keys := "↑↓ species · ←→ life stage · space cycle stages · esc back"
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}

// almanacDetail shows every drawn stage of a species side by side, with the
// selected stage highlighted, above its botany.
func (m model) almanacDetail(sp *Species, listWidth int) string {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}

	stageW := 11
	fit := max(1, (width-2)/(stageW+1))
	var frames []string
	for i := 0; i < StageCount && i < fit; i++ {
		art := renderArt(sp, sp.Palette, i, stageW, 6, 0)
		label := stageNames[i]
		style := subtleStyle
		if i == m.almanacStage {
			style = okStyle
		}
		block := strings.Join(art, "\n") + "\n" + soilLine(stageW, 0, true) + "\n" + pad(style.Render(center(label, stageW)), stageW)
		frames = append(frames, block)
	}
	strip := lipgloss.JoinHorizontal(lipgloss.Top, joinWithGap(frames)...)

	rows := []string{
		titleStyle.Render(sp.Common),
		latinStyle.Render(sp.Latin),
		subtleStyle.Render(sp.Family + " · " + rarityStyle(sp.Rarity).Render(sp.Rarity.String())),
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
		field("matures", "about "+hours(sp.Hours), width),
		field("cost", fmt.Sprintf("%d seeds", sp.SeedCost), width),
		"",
		lipgloss.NewStyle().Width(width).Render(subtleStyle.Render("✎ " + sp.Note)),
	}
	if !m.g.Unlocked(sp) {
		rows = append(rows, warnStyle.Render(fmt.Sprintf("Unlocks at %d matured plants.", sp.Unlock)))
	}
	return cardBorder.Width(width).Render(strings.Join(rows, "\n"))
}

func center(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", width-w-left)
}
