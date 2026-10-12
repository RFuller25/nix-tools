package main

import (
	"fmt"
	"math"
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
	total := max(3, m.height-5)
	scene := m.crowd.scene(m.width, sceneRows(total, m.width))
	avail := total - len(scene)

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
		lines = append(lines, wrapped(subtleStyle, m.gnomeNothing(), listWidth)...)
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
		switch m.gnomeService {
		case ServiceWeather:
			where = "which sky?"
		case ServiceSeason:
			where = "which season?"
		case ServiceSoil:
			where = "which recipe?"
		default:
			where = "which change?"
		}
	case gnomeBed:
		where = "which plant?"
		if m.gnomeService == ServiceSoil {
			where = "which bed?"
		} else if m.gnomeService == ServiceMatch {
			where = "which flower first?"
		}
	case gnomeBed2:
		where = "matched with?"
	}
	head := fit(titleStyle.Render("✧ gnome shop")+subtleStyle.Render("  ·  "+where+"  ·  ")+goldStyle.Render(goldLabel(m.g.Gold)), m.width)
	out := append([]string{head, m.divider()}, scene...)
	return strings.Join(append(out, body, m.divider(), m.footer(m.hintLine(setGnomes))), "\n")
}

// gnomeSummary is one line about the highlighted row, for a narrow window.
func (m model) gnomeSummary(cursor int) string {
	switch m.gnomeStage {
	case gnomeMenu:
		return gnomeInfos[GnomeService(cursor)].Gnome
	case gnomeChoose:
		switch m.gnomeService {
		case ServiceWeather, ServiceSeason:
			return fmt.Sprintf("%d gold", map[GnomeService]int{ServiceWeather: gnomeWeatherCost, ServiceSeason: gnomeSeasonCost}[m.gnomeService])
		case ServiceSoil:
			return gnomeTreatments[cursor].Blurb
		}
		return gnomeMutations[cursor].Blurb
	}
	return "bed " + fmt.Sprint(m.gnomeBeds()[cursor]+1)
}

// gnomeDetail is the card beside the list.
func (m model) gnomeDetail(cursor, width, avail int) string {
	width = max(width, 30)
	var rows []string
	switch m.gnomeStage {
	case gnomeMenu:
		rows = m.gnomeServiceCard(GnomeService(cursor), width, avail)
	case gnomeChoose:
		rows = m.gnomeChoiceCard(cursor, width)
	default:
		rows = m.gnomeBedCard(cursor, width)
	}
	visible, _, _ := window(rows, 0, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n"))
}

func costLine(n int) string { return valueStyle.Render("costs ") + goldStyle.Render(goldLabel(n)) }

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
		rows = append(rows, costLine(gnomeWeatherCost), subtleStyle.Render("today: "+cur.Glyph()+" "+cur.Name()+note))
	case ServiceSeason:
		cur := m.g.Season(m.now)
		note := ""
		if m.g.SeasonSwapped(m.now) {
			note = " (borrowed)"
		}
		rows = append(rows, costLine(gnomeSeasonCost), subtleStyle.Render("today: "+cur.Glyph()+" "+cur.String()+note))
	case ServiceGrowth:
		rows = append(rows, costLine(gnomeGrowthCost))
	case ServiceMutate:
		rows = append(rows, costLine(gnomeMutateCost))
	case ServiceChosen:
		rows = append(rows, costLine(gnomeChosenCost))
	case ServiceSoil:
		rows = append(rows, costLine(gnomeSoilCost))
	case ServiceMatch:
		rows = append(rows, labelStyle.Render("PRICES"),
			field("one kind", goldLabel(gnomeMatchSame), width),
			field("cousins", fmt.Sprintf("%s · same family", goldLabel(gnomeMatchFamily)), width),
			field("strangers", fmt.Sprintf("%s · half the seed", goldLabel(gnomeMatchWide)), width))
	default:
		w := m.g.WildSeedFor(dayKey(m.now))
		rows = append(rows, labelStyle.Render("TODAY'S SEED"),
			"  "+swatch(w.Genome.Hex(), 2)+" "+valueStyle.Render("Wild "+w.Species.Common)+subtleStyle.Render("  "+w.Species.Latin))
		rows = append(rows, geneRows(w.Species, w.Genome, width-2)...)
		if m.g.WildSeedAvailable(m.now) {
			rows = append(rows, "", costLine(w.Price)+subtleStyle.Render("  ·  one a day"))
		} else {
			rows = append(rows, "", warnStyle.Render("Sold. Pip brews another tomorrow."))
		}
	}
	if avail >= 26 && s != ServiceSeed && s != ServiceMatch {
		rows = append(rows, "")
		for _, l := range gnomeSprite {
			rows = append(rows, subtleStyle.Render(l))
		}
	}
	return rows
}

// gnomeChoiceCard describes the highlighted sky, season, mutation or recipe.
func (m model) gnomeChoiceCard(cursor, width int) []string {
	switch m.gnomeService {
	case ServiceWeather:
		w := weatherFor(allWeatherKinds[cursor])
		rain := "no rain"
		if w.Rainfall > 0 {
			rain = fmt.Sprintf("waters the beds %.0f%% an hour", w.Rainfall*100)
		}
		return []string{titleStyle.Render(w.Glyph() + " " + upperFirst(w.Name())), "",
			field("rain", rain, width),
			field("soil", fmt.Sprintf("dries ×%.2f", w.Dryness), width),
			field("growth", fmt.Sprintf("×%.2f", w.Growth), width),
			"", costLine(gnomeWeatherCost), subtleStyle.Render("lasts for the rest of today")}
	case ServiceSeason:
		season := allSeasons[cursor]
		happy, planted := 0, 0
		for i := range m.g.Plots {
			if sp := m.g.Plots[i].Species(); sp != nil {
				planted++
				if sp.LikesSeason(season) {
					happy++
				}
			}
		}
		rows := []string{titleStyle.Render(season.Glyph() + " " + upperFirst(season.String())), ""}
		rows = append(rows, wrapped(valueStyle, fmt.Sprintf("%d of your %d plants are happy in %s.", happy, planted, season), width-2)...)
		rows = append(rows, "")
		rows = append(rows, wrapped(subtleStyle, "Plants outside their season grow slowly; perennials sleep through a borrowed winter, annuals set seed on a borrowed summer's schedule, and the weather is drawn from that season's table.", width-2)...)
		return append(rows, "", costLine(gnomeSeasonCost), subtleStyle.Render("lasts for the rest of today"))
	case ServiceSoil:
		tr := gnomeTreatments[cursor]
		rows := []string{titleStyle.Render(tr.Name), ""}
		rows = append(rows, wrapped(valueStyle, "The gnome "+tr.Blurb+".", width-2)...)
		return append(rows, "", costLine(gnomeSoilCost), subtleStyle.Render("next: pick the bed"))
	}
	mu := gnomeMutations[cursor]
	rows := append([]string{titleStyle.Render(mu.Name), ""}, wrapped(valueStyle, "The gnome "+mu.Blurb+".", width-2)...)
	return append(rows, "", costLine(gnomeChosenCost), subtleStyle.Render("next: pick the plant"))
}

func (m model) gnomeBedCard(cursor, width int) []string {
	beds := m.gnomeBeds()
	if cursor >= len(beds) {
		return []string{subtleStyle.Render(m.gnomeNothing())}
	}
	idx := beds[cursor]
	p := &m.g.Plots[idx]
	if m.gnomeService == ServiceSoil {
		return m.gnomeSoilCard(idx, width)
	}
	sp := p.Species()
	rows := []string{
		titleStyle.Render(p.DisplayName()) + subtleStyle.Render(fmt.Sprintf("  ·  bed %d", idx+1)),
		latinStyle.Render(sp.Latin),
		"",
	}
	switch m.gnomeService {
	case ServiceGrowth:
		next := *p
		next.Growth = nextStage(p.Growth)
		rows = append(rows, field("now", p.StageName(), width), field("after", next.StageName(), width), "")
		if next.Growth >= 1 {
			rows = append(rows, wrapped(subtleStyle, "This one reaches full bloom.", width-2)...)
		}
		return append(rows, "", costLine(gnomeGrowthCost))
	case ServiceMutate:
		rows = append(rows, labelStyle.Render("NOW"))
		rows = append(rows, geneRows(sp, p.Genes(), width-2)...)
		rows = append(rows, "")
		rows = append(rows, wrapped(subtleStyle, "One gene, sometimes two, will change. Nobody knows which until it happens.", width-2)...)
		return append(rows, "", costLine(gnomeMutateCost))
	case ServiceMatch:
		if m.gnomeStage == gnomeBed2 {
			return m.gnomeMatchCard(idx, width)
		}
		rows = append(rows, labelStyle.Render("GENES"))
		rows = append(rows, geneRows(sp, p.Genes(), width-2)...)
		return append(rows, "", subtleStyle.Render("this flower will take the other's pollen"), subtleStyle.Render("next: pick its partner"))
	}
	mu := gnomeMutations[m.gnomeOption]
	after := mu.Apply(p.Genes())
	rows = append(rows, labelStyle.Render("NOW"))
	rows = append(rows, geneRows(sp, p.Genes(), width-2)...)
	rows = append(rows, "", labelStyle.Render("AFTER ")+subtleStyle.Render(strings.ToUpper(mu.Name)))
	if after == p.Genes() {
		rows = append(rows, wrapped(warnStyle, "It cannot go any further that way.", width-2)...)
	} else {
		rows = append(rows, geneRows(sp, after, width-2)...)
	}
	return append(rows, "", costLine(gnomeChosenCost))
}

func (m model) gnomeSoilCard(idx, width int) []string {
	p := &m.g.Plots[idx]
	tr := gnomeTreatments[m.gnomeOption]
	ph, rich := tr.Apply(p.PH, p.Richness)
	title := "Bare soil"
	if !p.Empty() {
		title = p.DisplayName()
	}
	rows := []string{titleStyle.Render(title) + subtleStyle.Render(fmt.Sprintf("  ·  bed %d", idx+1)), "",
		field("pH now", fmt.Sprintf("%.1f · %s", p.PH, p.Soil()), width),
		field("richness", richnessWord(p.Richness), width),
		"", labelStyle.Render("AFTER ") + subtleStyle.Render(strings.ToUpper(tr.Name))}
	after := *p
	after.PH, after.Richness = ph, rich
	rows = append(rows, field("pH", fmt.Sprintf("%.1f · %s", after.PH, after.Soil()), width), field("richness", richnessWord(after.Richness), width))
	if sp := p.Species(); sp != nil {
		rows = append(rows, "", field("now", soilNote(sp, p), width), field("after", soilNote(sp, &after), width),
			field("growth", fmt.Sprintf("×%.2f → ×%.2f", soilFactor(sp, p), soilFactor(sp, &after)), width))
		if sp.ID == "hydrangea" {
			rows = append(rows, field("flowers", fmt.Sprintf("%s → %s", colourName(hexRGB(hydrangeaHue(p.PH))), colourName(hexRGB(hydrangeaHue(after.PH)))), width))
		}
	}
	if math.Abs(ph-p.PH) < 0.05 && math.Abs(rich-p.Richness) < 0.01 {
		rows = append(rows, "", warnStyle.Render("It cannot go any further that way."))
	}
	return append(rows, "", costLine(gnomeSoilCost))
}

// hexRGB is the colour of a hue at full strength, for naming it.
func hexRGB(hue float64) rgb { return hslToRGB(hsl{h: hue, s: 0.7, l: 0.5}) }

func (m model) gnomeMatchCard(father, width int) []string {
	mother := &m.g.Plots[m.gnomeMother]
	f := &m.g.Plots[father]
	ms, fs := mother.Species(), f.Species()
	cost, kind := MatchCost(ms, fs)
	mean := meanGenome(mother.Genes(), f.Genes())
	rows := []string{
		titleStyle.Render(mother.DisplayName()) + subtleStyle.Render(fmt.Sprintf("  ·  bed %d", m.gnomeMother+1)),
		subtleStyle.Render("takes the pollen of"),
		titleStyle.Render(f.DisplayName()) + subtleStyle.Render(fmt.Sprintf("  ·  bed %d", father+1)),
		"",
		field("match", kind.Name(), width),
	}
	switch kind {
	case MatchSame:
		rows = append(rows, field("seed", "comes up as "+ms.Common, width))
	default:
		rows = append(rows, field("seed", ms.Common+" or "+fs.Common+", even odds", width))
		if kind == MatchWide {
			rows = append(rows, field("yield", "half as much seed", width))
		}
	}
	rows = append(rows, field("colour", swatch(mean.Hex(), 4)+" "+mean.ColourName(), width),
		field("height", mean.HeightWord()+" on average", width),
		"", subtleStyle.Render("the seeds vary around that blend"), "", costLine(cost))
	return rows
}
