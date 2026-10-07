package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewOrders is the order board: what the village wants, and which of your
// plants would fill it.
func (m model) viewOrders() string {
	orders := m.g.Orders
	cursor := min(m.ordersCursor, max(0, len(orders)-1))
	listWidth := 38
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(24, m.width-2)
	}
	avail := max(3, m.height-5)

	var lines []string
	for i, o := range orders {
		sp := o.SpeciesRef()
		if sp == nil {
			continue
		}
		marker, style := "  ", valueStyle
		if i == cursor {
			marker, style = titleStyle.Render("› "), titleStyle
		}
		stars := strings.Repeat("★", o.Tier+1)
		ready := "  "
		if len(m.g.QualifyingBeds(o)) > 0 {
			ready = okStyle.Render("✓ ")
		}
		lines = append(lines, marker+ready+style.Render(truncate(sp.Common, listWidth-18))+" "+
			seedStyle.Render(pad(stars, 3))+goldStyle.Render(fmt.Sprintf(" %dg", o.Reward)))
	}
	if len(orders) == 0 {
		lines = append(lines, subtleStyle.Render("No orders on the board."), subtleStyle.Render("New ones are posted each day."))
	}
	if len(lines) > avail {
		lines = lines[:avail]
	}

	detail := ""
	if len(orders) > 0 && !narrow {
		detail = m.orderDetail(orders[cursor], m.width-listWidth-6, avail)
	} else if len(orders) > 0 {
		o := orders[cursor]
		lines = append(lines, "", fit(subtleStyle.Render(m.orderSummary(o)), listWidth))
	}
	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	head := fit(titleStyle.Render("✉ orders")+subtleStyle.Render(fmt.Sprintf("  ·  %d on the board  ·  %d filled  ·  ", len(orders), m.g.OrdersDone))+
		goldStyle.Render(goldLabel(m.g.Gold)), m.width)
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(m.hintLine(setOrders))}, "\n")
}

// orderSummary is one line of what an order wants.
func (m model) orderSummary(o Order) string {
	sp := o.SpeciesRef()
	var parts []string
	for _, c := range o.Wants {
		parts = append(parts, c.Text(sp))
	}
	return sp.Common + ": " + strings.Join(parts, ", ")
}

func (m model) orderDetail(o Order, width, avail int) string {
	width = max(width, 30)
	sp := o.SpeciesRef()
	rows := []string{
		titleStyle.Render(sp.Common) + subtleStyle.Render("  ·  "+o.TierName()),
		latinStyle.Render(sp.Latin),
		"",
		labelStyle.Render("WANTED"),
	}
	for _, c := range o.Wants {
		line := "  " + valueStyle.Render(c.Text(sp))
		if c.Trait == TraitColour {
			if target, ok := parseColor(c.Colour); ok {
				line = "  " + swatch(rgb(target).hex(), 4) + " " + valueStyle.Render(c.Text(sp))
			}
		}
		rows = append(rows, line)
	}
	rows = append(rows, "", valueStyle.Render("pays ")+goldStyle.Render(goldLabel(o.Reward)))
	if b := SpeciesByID(o.Bonus); b != nil {
		rows = append(rows, valueStyle.Render("and two ")+okStyle.Render(b.Common)+valueStyle.Render(" seeds"))
	}
	rows = append(rows, subtleStyle.Render("lapses "+o.Expires.Format("Mon 2 Jan")+", with no penalty"), "")

	beds := m.g.QualifyingBeds(o)
	if len(beds) > 0 {
		rows = append(rows, labelStyle.Render("READY TO DELIVER"))
		for _, b := range beds {
			p := &m.g.Plots[b]
			rows = append(rows, "  "+swatch(p.Genes().Hex(), 2)+" "+valueStyle.Render(fmt.Sprintf("bed %d, %s", b+1, p.FullName())))
		}
		rows = append(rows, "", subtleStyle.Render("enter takes the first one. The plant is used up by the order."))
	} else {
		rows = append(rows, labelStyle.Render("NONE OF YOUR PLANTS FIT YET"))
		held := 0
		for i := range m.g.Plots {
			if m.g.Plots[i].SpeciesID == o.Species {
				held++
			}
		}
		rows = append(rows, wrapped(subtleStyle, fmt.Sprintf("You have %d %s growing. Plants must be in flower, and fit every condition. Breed towards it: cross seed that already leans the right way.", held, sp.Common), width-2)...)
	}
	visible, _, _ := window(rows, 0, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n"))
}
