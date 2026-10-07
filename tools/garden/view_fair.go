package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// fairCandidate is a plant that could be entered, with its predicted score.
type fairCandidate struct {
	Bed   int
	Score float64
}

// fairCandidates lists the plants that may be entered this week, best first.
func (m model) fairCandidates() []fairCandidate {
	cat := m.g.FairFor(weekKey(m.now))
	var out []fairCandidate
	for i := range m.g.Plots {
		if m.g.CanEnter(i, m.now) != nil {
			continue
		}
		p := &m.g.Plots[i]
		out = append(out, fairCandidate{Bed: i, Score: FairScore(cat, p.Species(), p.Genes(), plantHealth(p))})
	}
	for a := 1; a < len(out); a++ {
		for b := a; b > 0 && out[b].Score > out[b-1].Score; b-- {
			out[b], out[b-1] = out[b-1], out[b]
		}
	}
	return out
}

// weekEnds is when this week's show closes: the end of Sunday.
func weekEnds(now time.Time) time.Time {
	days := (7 - int(now.Weekday())) % 7
	y, mo, d := now.AddDate(0, 0, days).Date()
	return time.Date(y, mo, d, 23, 59, 59, 0, now.Location())
}

func (m model) viewFair() string {
	cat := m.g.FairFor(weekKey(m.now))
	cands := m.fairCandidates()
	cursor := min(m.fairCursor, max(0, len(cands)-1))
	avail := max(3, m.height-5)

	var lines []string
	lines = append(lines,
		titleStyle.Render(cat.Name)+subtleStyle.Render("  ·  this week's class"),
		subtleStyle.Render("for "+cat.Blurb),
		subtleStyle.Render("judging closes "+weekEnds(m.now).Format("Mon 2 Jan")+", then you are paid"),
		"")
	if e := m.g.Fair.Entry; e != nil && e.Week == weekKey(m.now) {
		lines = append(lines, labelStyle.Render("ENTERED"),
			"  "+swatch(e.Genome.Hex(), 2)+" "+valueStyle.Render(truncate(e.Name, 40))+
				subtleStyle.Render(fmt.Sprintf("  · %.0f points as it stood", FairScore(cat, SpeciesByID(e.Species), e.Genome, e.Health))), "")
	} else {
		lines = append(lines, subtleStyle.Render("Nothing entered yet."), "")
	}

	lines = append(lines, labelStyle.Render("YOUR PLANTS THAT MAY BE ENTERED")+subtleStyle.Render("  best first, with the score the judges would give today"))
	if len(cands) == 0 {
		lines = append(lines, subtleStyle.Render("  None: plants have to be in flower"+m.classNote(cat)+"."))
	}
	for i, c := range cands {
		p := &m.g.Plots[c.Bed]
		marker, style := "  ", valueStyle
		if i == cursor {
			marker, style = titleStyle.Render("› "), titleStyle
		}
		lines = append(lines, marker+swatch(p.Genes().Hex(), 2)+" "+style.Render(truncate(fmt.Sprintf("bed %d, %s", c.Bed+1, p.FullName()), 46))+
			subtleStyle.Render(fmt.Sprintf("  %.0f", c.Score)))
	}
	lines = append(lines, "", labelStyle.Render("THE FIELD"),
		subtleStyle.Render(fmt.Sprintf("  %d other growers, around %.0f points, rising by %.1f for every first place you win (now %.0f).",
			fairRivals, fairRivalMean, fairBarPerWin, fairRivalMean+math.Min(fairBarMax, float64(m.g.Fair.Wins)*fairBarPerWin))))

	if rs := m.g.Fair.Ribbons; len(rs) > 0 {
		lines = append(lines, "", labelStyle.Render("LAST RESULTS"))
		for i := len(rs) - 1; i >= 0 && i >= len(rs)-3; i-- {
			r := rs[i]
			c, _ := categoryByID(r.Category)
			lines = append(lines, "  "+okStyle.Render(ribbonGlyph(r.Place))+" "+valueStyle.Render(fmt.Sprintf("%s in %s", ordinalPlace(r.Place), c.Name))+
				subtleStyle.Render(" · "+r.Name+fmt.Sprintf(" · %s", goldLabel(r.Prize))))
		}
	}

	visible, _, _ := window(lines, 0, avail)
	for i := range visible {
		visible[i] = fit(visible[i], m.width)
	}
	head := fit(titleStyle.Render("❖ the fair")+subtleStyle.Render(fmt.Sprintf("  ·  %d ribbons  ·  %d firsts  ·  ", len(m.g.Fair.Ribbons), m.g.Fair.Wins))+goldStyle.Render(goldLabel(m.g.Gold)), m.width)
	return strings.Join([]string{head, m.divider(), strings.Join(visible, "\n"), m.divider(), m.footer(m.hintLine(setFair))}, "\n")
}

// classNote says what else a class asks, for the empty list.
func (m model) classNote(c FairCategory) string {
	if c.Kind >= 0 {
		return " and be a " + c.Kind.String()
	}
	return ""
}

// ribbonGlyph is the rosette for a place.
func ribbonGlyph(place int) string {
	switch place {
	case 1:
		return "❂"
	case 2:
		return "✪"
	case 3:
		return "✯"
	}
	return "○"
}

// ribbonDetail is the almanac page for one result.
func (m model) ribbonDetail(idx, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	if idx < 0 || idx >= len(m.g.Fair.Ribbons) {
		return cardBorder.Width(width).Render(subtleStyle.Render("No such ribbon.")), false
	}
	r := m.g.Fair.Ribbons[idx]
	c, _ := categoryByID(r.Category)
	sp := SpeciesByID(r.Species)
	rows := []string{
		titleStyle.Render(ribbonGlyph(r.Place) + " " + upperFirst(ordinalPlace(r.Place)) + " place"),
		valueStyle.Render(c.Name),
		subtleStyle.Render(fmt.Sprintf("week %d of %d · judged %s · %d entrants · %.0f points · %s", r.Week%100, r.Week/100, r.Judged.Format("2 Jan"), r.Entrants, r.Score, goldLabel(r.Prize))),
		"",
		valueStyle.Render(r.Name),
	}
	if sp != nil {
		rows = append(rows, latinStyle.Render(sp.Latin), "")
		rows = append(rows, geneRows(sp, r.Genome, width)...)
	}
	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}

var _ = lipgloss.Width
