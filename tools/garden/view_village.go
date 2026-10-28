package main

import (
	"fmt"
	"strings"
)

// The village screen: five tabs of things to spend gold on.

type villageTab int

const (
	tabHelp villageTab = iota
	tabScenery
	tabLab
	tabCommission
	tabStyle
	villageTabs
)

var villageTabNames = [villageTabs]string{"hired help", "scenery", "gene lab", "commissions", "colours & borders"}

// villageStage is how far into a purchase the gardener has got.
type villageStage int

const (
	villageList  villageStage = iota // choosing what to buy
	villagePick                      // choosing the plant or packet it is for
	villagePick2                     // choosing the second flower of a forecast
)

// Rows of the gene lab, in order.
const (
	labForecast = iota
	labClone
	labPurify
)

// villageRow is one line of a tab's list.
type villageRow struct {
	Label string
	Note  string // after the label: owned, working, a price
	Blurb string // what it is, for the panel under the list
	OK    bool   // false when it cannot be had right now
}

func priceNote(g *Garden, price int) string {
	if price == 0 {
		return "free"
	}
	return goldLabel(price)
}

// villageRows lists the highlighted tab.
func (m model) villageRows() []villageRow {
	g := m.g
	var rows []villageRow
	switch m.villageTab {
	case tabHelp:
		for _, d := range helperDefs {
			note := fmt.Sprintf("%s to hire, %s a week", goldLabel(d.Hire), goldLabel(d.Wage))
			ok := g.Gold >= d.Hire
			if g.Hired(d.ID) {
				ok = true
				if g.HelperWorking(d.ID, m.now) {
					note = fmt.Sprintf("working · %s a week", goldLabel(d.Wage))
				} else {
					note = "at home this week, unpaid"
				}
			}
			rows = append(rows, villageRow{Label: d.Name, Note: note, Blurb: d.Blurb, OK: ok})
		}
	case tabScenery:
		for _, d := range sceneryDefs {
			note, ok := goldLabel(d.Price), g.Gold >= d.Price
			if g.HasScenery(d.ID) {
				note, ok = "standing in the garden", true
			}
			rows = append(rows, villageRow{Label: d.Glyph + " " + d.Name, Note: note, Blurb: d.Blurb, OK: ok})
		}
	case tabLab:
		rows = []villageRow{
			{Label: "Forecast a cross", Note: goldLabel(labForecastCost), OK: g.Gold >= labForecastCost,
				Blurb: fmt.Sprintf("Pick two flowers of one species in bloom and the lab shows %d seedlings you could expect from crossing them: a sample of the real spread, not a promise.", labForecastKids)},
			{Label: "Copy a plant", Note: fmt.Sprintf("from %s", goldLabel(labCloneBase)), OK: g.Gold >= labCloneBase,
				Blurb: fmt.Sprintf("Pick a flower in bloom and the lab seeds %d copies of it, gene for gene. They come exactly true. The dearer the plant is to the shop's idea of it, the more it costs.", labCloneSeeds)},
			{Label: "Purify a packet", Note: fmt.Sprintf("from %s", goldLabel(labPurifyBase)), OK: g.Gold >= labPurifyBase,
				Blurb: "Pick a packet of your own mixed seed and the lab settles it on its middle gene, so everything sown from it comes true. The more seed in the packet, the more it costs."},
		}
	case tabCommission:
		for _, d := range commissionDefs {
			rows = append(rows, villageRow{Label: d.Name, Note: fmt.Sprintf("%s plus the seed's price × %d", goldLabel(d.Base), d.Mult), OK: g.Gold >= d.Base,
				Blurb: d.Maker + ": " + d.Blurb + " The plant is used up. You keep the keepsake."})
		}
	case tabStyle:
		for _, d := range themeDefs {
			note, ok := priceNote(g, d.Price), g.Gold >= d.Price
			switch {
			case g.CurrentTheme() == d.ID:
				note, ok = "in use", true
			case g.OwnsTheme(d.ID):
				note, ok = "owned", true
			}
			rows = append(rows, villageRow{Label: "colours: " + d.Name, Note: note, OK: ok,
				Blurb: "A palette for titles, headings, species names and the selected bed. It repaints at once."})
		}
		for _, d := range borderDefs {
			note, ok := priceNote(g, d.Price), g.Gold >= d.Price
			switch {
			case g.CurrentBorder() == d.ID:
				note, ok = "in use", true
			case g.OwnsBorder(d.ID):
				note, ok = "owned", true
			}
			rows = append(rows, villageRow{Label: "border: " + d.Name, Note: note, OK: ok,
				Blurb: "The frame drawn around every bed in the garden.\n" + borderPreview(d)})
		}
	}
	return rows
}

// borderPreview draws a tiny box in a border, for the shop.
func borderPreview(d borderDef) string {
	return plainBorder.Border(d.Shape).Render(" bed ")
}

// villageTarget is a bed or packet a purchase can be worked on.
type villageTarget struct {
	Idx   int
	Label string
	Note  string
}

// villageTargets lists what the current service can be used on.
func (m model) villageTargets() []villageTarget {
	var out []villageTarget
	g := m.g
	service := m.villageCursor
	switch {
	case m.villageTab == tabLab && service == labPurify:
		for _, i := range g.ShedOrder() {
			pk := g.Shed[i]
			if pk.Pure || pk.Species() == nil || pk.Count <= 0 {
				continue
			}
			name := pk.Species().Common
			if pk.Label != "" {
				name = pk.Label
			}
			out = append(out, villageTarget{Idx: i, Label: fmt.Sprintf("%s × %d", name, pk.Count), Note: goldLabel(g.PurifyCost(i))})
		}
	case m.villageTab == tabLab || m.villageTab == tabCommission:
		for i := range g.Plots {
			if !g.labReady(i) {
				continue
			}
			p := &g.Plots[i]
			t := villageTarget{Idx: i, Label: fmt.Sprintf("bed %d, %s", i+1, p.FullName())}
			switch {
			case m.villageTab == tabCommission:
				t.Note = goldLabel(g.CommissionCost(commissionDefs[service].ID, i))
			case service == labClone:
				t.Note = goldLabel(g.CloneCost(i))
			case m.villageStage == villagePick2:
				if i == m.villageFirst || p.SpeciesID != g.Plots[m.villageFirst].SpeciesID {
					continue
				}
			}
			out = append(out, t)
		}
	}
	return out
}

func (m model) viewVillage() string {
	avail := max(5, m.height-5)

	var tabs []string
	for t := villageTab(0); t < villageTabs; t++ {
		name := villageTabNames[t]
		if t == m.villageTab {
			tabs = append(tabs, titleStyle.Render("["+name+"]"))
		} else {
			tabs = append(tabs, subtleStyle.Render(" "+name+" "))
		}
	}
	lines := []string{strings.Join(tabs, " "), ""}

	if m.villageStage == villageList {
		rows := m.villageRows()
		cursor := min(m.villageCursor, max(0, len(rows)-1))
		for i, r := range rows {
			marker, style := "  ", valueStyle
			if i == cursor {
				marker, style = titleStyle.Render("› "), titleStyle
			}
			if !r.OK {
				style = lockedStyle
			}
			lines = append(lines, marker+style.Render(truncate(r.Label, 34))+"  "+subtleStyle.Render(r.Note))
		}
		if len(rows) > 0 {
			lines = append(lines, "")
			parts := strings.SplitN(rows[cursor].Blurb, "\n", 2)
			lines = append(lines, wrapped(subtleStyle, parts[0], max(20, m.width-4))...)
			if len(parts) > 1 {
				lines = append(lines, strings.Split(parts[1], "\n")...)
			}
		}
		lines = append(lines, m.villageFooter()...)
	} else {
		lines = append(lines, labelStyle.Render(m.villagePrompt()), "")
		targets := m.villageTargets()
		cursor := min(m.villagePickCursor, max(0, len(targets)-1))
		if len(targets) == 0 {
			lines = append(lines, subtleStyle.Render("  Nothing to choose from."))
		}
		for i, t := range targets {
			marker, style := "  ", valueStyle
			if i == cursor {
				marker, style = titleStyle.Render("› "), titleStyle
			}
			lines = append(lines, marker+style.Render(truncate(t.Label, 44))+"  "+goldStyle.Render(t.Note))
		}
	}
	if len(m.villageResult) > 0 && m.villageStage == villageList {
		lines = append(lines, "")
		lines = append(lines, m.villageResult...)
	}

	visible, _, _ := window(lines, 0, avail)
	for i := range visible {
		visible[i] = fit(visible[i], m.width)
	}
	head := fit(titleStyle.Render("⌂ the village")+subtleStyle.Render("  ·  "+m.g.sceneryHeadline()+"  ·  ")+goldStyle.Render(goldLabel(m.g.Gold)), m.width)
	return strings.Join([]string{head, m.divider(), strings.Join(visible, "\n"), m.divider(), m.footer(m.hintLine(setVillage))}, "\n")
}

// villagePrompt says what is being chosen.
func (m model) villagePrompt() string {
	switch {
	case m.villageTab == tabCommission:
		return "WHICH PLANT? It will be used up."
	case m.villageStage == villagePick2:
		return "WHICH FLOWER TO CROSS IT WITH?"
	case m.villageCursor == labForecast:
		return "WHICH TWO FLOWERS? Pick the first."
	case m.villageCursor == labClone:
		return "WHICH PLANT SHALL THE LAB COPY?"
	}
	return "WHICH PACKET SHALL THE LAB PURIFY?"
}

// villageFooter is the extra lines under a tab's list.
func (m model) villageFooter() []string {
	g := m.g
	switch m.villageTab {
	case tabHelp:
		bill := g.wagesLabel()
		return []string{"", subtleStyle.Render(fmt.Sprintf("  Wage bill %s a week, paid Mondays. Unpaid help stays home. x lets one go.", bill))}
	case tabCommission:
		if len(g.Keepsakes) == 0 {
			return []string{"", subtleStyle.Render("  Your keepsake shelf is empty.")}
		}
		out := []string{"", labelStyle.Render("YOUR KEEPSAKES")}
		for i := len(g.Keepsakes) - 1; i >= 0 && i >= len(g.Keepsakes)-6; i-- {
			k := g.Keepsakes[i]
			name := k.Kind
			if d := commissionDefByID(k.Kind); d != nil {
				name = d.Name
			}
			out = append(out, "  "+swatch(k.Hex, 2)+" "+valueStyle.Render(truncate(name+": "+k.Name, 56)))
		}
		if len(g.Keepsakes) > 6 {
			out = append(out, subtleStyle.Render(fmt.Sprintf("  and %d more", len(g.Keepsakes)-6)))
		}
		return out
	}
	return nil
}

// sceneryHeadline is a short line for the header.
func (g *Garden) sceneryHeadline() string {
	if s := g.SceneryStrip(); s != "" {
		return s
	}
	return "nothing set up yet"
}
