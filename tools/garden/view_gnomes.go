package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// gnomeSprite is the stallholder, for when there is room to show one.
var gnomeSprite = []string{
	"    ▲    ",
	"   ╱ ╲   ",
	"  ╱ ✧ ╲  ",
	" ▔▔▔▔▔▔▔ ",
	"  (• •)  ",
	"  ╰▾▾▾╯  ",
	"   ╱ ╲   ",
}

// viewGnomes is the gnome stall: what the gnomes sell, then the choices
// under it, then the plant to work it on.
func (m model) viewGnomes() string {
	rows := m.gnomeRows()
	cursor := min(m.gnomeCursor, max(0, len(rows)-1))
	listWidth := 36
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(20, m.width-2)
	}
	avail := max(3, m.height-5)

	scroll := max(0, cursor-avail+1)
	var lines []string
	for i := scroll; i < len(rows) && i < scroll+avail; i++ {
		r := rows[i]
		marker, style := "  ", valueStyle
		switch {
		case i == cursor:
			marker, style = titleStyle.Render("› "), titleStyle
		case !r.OK:
			style = subtleStyle
		}
		note := goldStyle.Render(r.Note)
		if !r.OK {
			note = lockedStyle.Render(r.Note)
		}
		lines = append(lines, marker+style.Render(pad(truncate(r.Label, listWidth-12), listWidth-12))+" "+note)
	}
	if len(rows) == 0 {
		lines = append(lines, subtleStyle.Render("You have nothing planted."), subtleStyle.Render("The gnomes need a plant to work on."))
	}

	body := strings.Join(lines, "\n")
	if narrow {
		if len(rows) > 0 && len(lines) < avail {
			body += "\n\n" + fit(subtleStyle.Render(m.gnomeSummary(cursor)), listWidth)
		}
	} else {
		detail := m.gnomeDetail(cursor, m.width-listWidth-6, avail)
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}

	where := "the stall"
	switch m.gnomeStage {
	case gnomeChoose:
		if m.gnomeService == ServiceWeather {
			where = "which sky?"
		} else {
			where = "which change?"
		}
	case gnomeBed:
		where = "which plant?"
	}
	head := fit(titleStyle.Render("✧ gnome shop")+subtleStyle.Render("  ·  "+where+"  ·  ")+goldStyle.Render(goldLabel(m.g.Gold)), m.width)
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(m.hintLine(setGnomes))}, "\n")
}

// gnomeSummary is one line about the highlighted row, for a narrow window.
func (m model) gnomeSummary(cursor int) string {
	switch m.gnomeStage {
	case gnomeMenu:
		return gnomeInfos[GnomeService(cursor)].Gnome
	case gnomeChoose:
		if m.gnomeService == ServiceWeather {
			return fmt.Sprintf("%d gold", gnomeWeatherCost)
		}
		return gnomeMutations[cursor].Blurb
	}
	if m.gnomeService == ServiceMutate {
		return "a random change; costs " + goldLabel(gnomeMutateCost)
	}
	return "costs " + goldLabel(gnomeChosenCost)
}

// gnomeDetail is the card beside the list.
func (m model) gnomeDetail(cursor, width, avail int) string {
	width = max(width, 30)
	var rows []string
	switch m.gnomeStage {
	case gnomeMenu:
		rows = m.gnomeServiceCard(GnomeService(cursor), width, avail)
	case gnomeChoose:
		if m.gnomeService == ServiceWeather {
			rows = m.gnomeWeatherCard(allWeatherKinds[cursor], width)
		} else {
			mu := gnomeMutations[cursor]
			rows = append([]string{titleStyle.Render(mu.Name), ""}, wrapped(valueStyle, "The gnome "+mu.Blurb+".", width-2)...)
			rows = append(rows, "", valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeChosenCost)),
				subtleStyle.Render("next: pick the plant"))
		}
	default:
		rows = m.gnomeBedCard(cursor, width)
	}
	visible, _, _ := window(rows, 0, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n"))
}

func (m model) gnomeServiceCard(s GnomeService, width, avail int) []string {
	info := gnomeInfos[s]
	rows := []string{titleStyle.Render(info.Name), subtleStyle.Render("run by " + info.Gnome), ""}
	rows = append(rows, wrapped(valueStyle, info.Blurb, width-2)...)
	rows = append(rows, "")
	switch s {
	case ServiceWeather:
		cur := m.g.Weather(m.now)
		note := ""
		if m.g.Spelled(m.now) {
			note = " (set by the gnomes)"
		}
		rows = append(rows, valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeWeatherCost)),
			subtleStyle.Render("today: "+cur.Glyph()+" "+cur.Name()+note))
	case ServiceMutate:
		rows = append(rows, valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeMutateCost)))
	case ServiceChosen:
		rows = append(rows, valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeChosenCost)))
	default:
		w := m.g.WildSeedFor(dayKey(m.now))
		rows = append(rows, labelStyle.Render("TODAY'S SEED"),
			"  "+swatch(w.Genome.Hex(), 2)+" "+valueStyle.Render("Wild "+w.Species.Common)+subtleStyle.Render("  "+w.Species.Latin))
		rows = append(rows, geneRows(w.Species, w.Genome, width-2)...)
		if m.g.WildSeedAvailable(m.now) {
			rows = append(rows, "", valueStyle.Render("costs ")+goldStyle.Render(goldLabel(w.Price))+subtleStyle.Render("  ·  one a day"))
		} else {
			rows = append(rows, "", warnStyle.Render("Sold. Pip brews another tomorrow."))
		}
	}
	if avail >= 22 && s != ServiceSeed {
		rows = append(rows, "")
		for _, l := range gnomeSprite {
			rows = append(rows, subtleStyle.Render(l))
		}
	}
	return rows
}

func (m model) gnomeWeatherCard(kind WeatherKind, width int) []string {
	w := weatherFor(kind)
	rows := []string{titleStyle.Render(w.Glyph() + " " + upperFirst(w.Name())), ""}
	rain := "no rain"
	if w.Rainfall > 0 {
		rain = fmt.Sprintf("waters the beds %.0f%% an hour", w.Rainfall*100)
	}
	rows = append(rows,
		field("rain", rain, width),
		field("soil", fmt.Sprintf("dries ×%.2f", w.Dryness), width),
		field("growth", fmt.Sprintf("×%.2f", w.Growth), width),
		"",
		valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeWeatherCost)),
		subtleStyle.Render("lasts for the rest of today"))
	return rows
}

func (m model) gnomeBedCard(cursor, width int) []string {
	beds := m.gnomeBeds()
	if cursor >= len(beds) {
		return []string{subtleStyle.Render("No plants to work on.")}
	}
	p := &m.g.Plots[beds[cursor]]
	sp := p.Species()
	rows := []string{
		titleStyle.Render(p.DisplayName()) + subtleStyle.Render(fmt.Sprintf("  ·  bed %d", beds[cursor]+1)),
		latinStyle.Render(sp.Latin),
		"",
		labelStyle.Render("NOW"),
	}
	rows = append(rows, geneRows(sp, p.Genes(), width-2)...)
	rows = append(rows, "")
	if m.gnomeService == ServiceMutate {
		rows = append(rows, wrapped(subtleStyle, "One gene, sometimes two, will change. Nobody knows which until it happens.", width-2)...)
		rows = append(rows, "", valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeMutateCost)))
		return rows
	}
	mu := gnomeMutations[m.gnomeOption]
	after := mu.Apply(p.Genes())
	rows = append(rows, labelStyle.Render("AFTER ")+subtleStyle.Render(strings.ToUpper(mu.Name)))
	if after == p.Genes() {
		rows = append(rows, wrapped(warnStyle, "It cannot go any further that way.", width-2)...)
	} else {
		rows = append(rows, geneRows(sp, after, width-2)...)
	}
	rows = append(rows, "", valueStyle.Render("costs ")+goldStyle.Render(goldLabel(gnomeChosenCost)))
	return rows
}
