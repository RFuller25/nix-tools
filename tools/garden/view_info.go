package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) viewInfo() string {
	p := m.plot()
	sp := p.Species()
	if sp == nil {
		return strings.Join([]string{
			titleStyle.Render("bed " + fmt.Sprint(m.cursor+1)),
			m.divider(),
			subtleStyle.Render("Nothing is planted here yet. Press p to visit the seed shed."),
			m.divider(),
			m.footer("esc back · p plant"),
		}, "\n")
	}

	width := min(m.width-4, 76)
	if width < 30 {
		width = 30
	}

	season := m.g.Season(m.now)
	stage, pal := appearance(sp, p, season, m.phase())
	art := renderArt(sp, pal, stage, 24, 7, m.wind.swayAt(m.cursor%plotCols))
	artBlock := strings.Join(art, "\n") + "\n" + soilLine(24, p.Weeds, true, m.phase())

	headline := []string{
		titleStyle.Render(p.DisplayName()),
		latinStyle.Render(sp.Latin),
		subtleStyle.Render(sp.Family + " · " + sp.Kind.String() + " · " + rarityStyle(sp.Rarity).Render(sp.Rarity.String())),
		"",
		field("stage", fmt.Sprintf("%s (%d of %d)", p.StageName(), p.Stage()+1, StageCount), width-28),
		field("doing", state(sp, p, season, m.phase()), width-28),
		field("mood", p.Mood(), width-28),
		field("age", humanDuration(p.Age(m.now)), width-28),
		field("next", m.nextStageNote(p, sp), width-28),
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, artBlock, "  ", strings.Join(headline, "\n"))

	meters := strings.Join([]string{
		labelStyle.Render(pad("growth", 9)) + meter(p.Growth, 24, okStyle) + subtleStyle.Render(fmt.Sprintf("  %3.0f%%", p.Growth*100)),
		labelStyle.Render(pad("water", 9)) + meter(p.Moisture, 24, waterStyle) + subtleStyle.Render(fmt.Sprintf("  %3.0f%%", p.Moisture*100)),
		labelStyle.Render(pad("weeds", 9)) + meter(p.Weeds, 24, weedStyle) + subtleStyle.Render(fmt.Sprintf("  %3.0f%%", p.Weeds*100)),
	}, "\n")

	desc := lipgloss.NewStyle().Width(width).Render(valueStyle.Render(sp.Desc))

	ground := soilNote(sp, p)
	if sp.ID == "hydrangea" && !p.Pond {
		ground += ", so it flowers " + HydrangeaColour(p.PH)
	}

	care := strings.Join([]string{
		field("bed", fmt.Sprintf("%d — %s, %s", m.cursor+1, p.Soil(), richnessWord(p.Richness)), width),
		field("ground", ground, width),
		field("origin", sp.Origin, width),
		field("blooms", sp.Bloom, width),
		field("sun", sp.Sun, width),
		field("water", sp.Water, width),
		field("height", sp.Height, width),
		field("season", sp.SeasonNames(), width),
		field("life", lifeNote(sp), width),
		field("planted", plantedWhen(p), width),
	}, "\n")

	tip := lipgloss.NewStyle().Width(width).Render(subtleStyle.Render("✎ " + sp.Note))

	var neighbours []string
	for _, e := range m.g.companionEffects(m.cursor, sp) {
		style, sign := okStyle, "+"
		if e.Delta < 0 {
			style, sign = warnStyle, ""
		}
		line := fmt.Sprintf("bed %d, %s: %s%.0f%% — %s", e.Bed+1, e.Other.Common, sign, e.Delta*100, e.Note)
		neighbours = append(neighbours, lipgloss.NewStyle().Width(width).Render(style.Render(line)))
	}
	if len(neighbours) > 0 {
		neighbours = append([]string{labelStyle.Render("neighbours")}, neighbours...)
	}

	pods := ""
	if p.Pods >= 1 {
		pods = seedStyle.Render(fmt.Sprintf("✦ %d ripe seed pod(s) — press f to gather.", int(p.Pods)))
	}

	parts := []string{top, "", meters, "", desc, "", care, ""}
	parts = append(parts, neighbours...)
	if len(neighbours) > 0 {
		parts = append(parts, "")
	}
	parts = append(parts, tip, pods)
	card := strings.Join(compact(parts), "\n")

	// The card is usually taller than a small terminal, so it is windowed
	// rather than allowed to push the top of the screen out of reach.
	avail := max(3, m.height-4) // the card's border, plus the two footer lines
	visible, above, below := window(strings.Split(card, "\n"), m.cardScroll, avail)
	body := cardBorder.Render(strings.Join(visible, "\n"))

	if m.naming {
		return strings.Join([]string{
			body,
			m.input.View(),
			helpStyle.Render("enter to confirm · esc to cancel"),
		}, "\n")
	}

	keys := scrollHint(above, below, "w water · c weed · f gather · n name · u lift · ←→ other beds · esc back")
	return strings.Join([]string{body, m.footer(keys)}, "\n")
}

// plantedWhen reads the planting date, or admits it does not know.
func plantedWhen(p *Plot) string {
	if p.PlantedAt.IsZero() {
		return "some time ago"
	}
	return p.PlantedAt.Format("Mon 2 Jan, 15:04")
}

// lifeNote says what kind of life the plant leads, and what that means for
// the gardener watching it.
func lifeNote(sp *Species) string {
	switch sp.Life() {
	case Annual:
		return "annual — flowers, seeds, and is done"
	case Biennial:
		return "biennial — leaves first, then flowers"
	case Woody:
		if sp.Evergreen() {
			return "evergreen shrub or tree"
		}
		return "deciduous shrub or tree — bare in winter"
	default:
		if sp.Evergreen() {
			return "evergreen perennial — back every year"
		}
		return "perennial — dies back, returns in spring"
	}
}

// richnessWord describes how much heart a bed's soil has left in it.
func richnessWord(v float64) string {
	switch {
	case v > 0.75:
		return "deeply composted"
	case v > 0.5:
		return "in good heart"
	case v > 0.25:
		return "workable"
	default:
		return "hungry ground"
	}
}

func compact(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" && len(out) > 0 && out[len(out)-1] == "" {
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// nextStageNote estimates when the plant reaches its next drawn stage at the
// rate it is growing right now.
func (m model) nextStageNote(p *Plot, sp *Species) string {
	if p.Spent {
		return "finished for the year — lift it (u) to compost the bed"
	}
	if dormant(sp, m.g.Season(m.now)) && p.Growth >= 1 {
		return "asleep until spring"
	}
	if p.Growth >= 1 {
		if p.Pods >= maxPods {
			return "fully grown, seed pods full"
		}
		next := (math.Ceil(p.Pods+1e-9) - p.Pods) / podsPerHour
		return fmt.Sprintf("fully grown · next seed pod in %s", humanDuration(time.Duration(next*float64(time.Hour))))
	}
	thresholds := []float64{0.10, 0.35, 0.70, 1.00}
	target := 1.0
	for _, t := range thresholds {
		if p.Growth < t {
			target = t
			break
		}
	}
	factor := growthFactor(p, sp, m.g.Weather(m.now), m.g.Season(m.now))
	rate := (1.0 / sp.Hours) * factor
	if rate <= 0 {
		return "stalled"
	}
	remaining := (target - p.Growth) / rate
	name := stageNames[min(StageCount-1, p.Stage()+1)]
	return fmt.Sprintf("%s in about %s", name, humanDuration(time.Duration(remaining*float64(time.Hour))))
}

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "moments"
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 48*time.Hour:
		h := int(d.Hours())
		mins := int(d.Minutes()) - h*60
		if mins == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, mins)
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
