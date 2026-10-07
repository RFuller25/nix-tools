package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// shelf is which half of the seed shed is open.
type shelf int

const (
	shelfShop shelf = iota // the shop: named forms, priced
	shelfMine              // your own seed, in packets
)

// viewShed shows whichever shelf is open.
func (m model) viewShed() string {
	if m.shelf == shelfMine {
		return m.viewMine()
	}
	return m.viewShopShelf()
}

// viewMine is the shelf of packets: what you have gathered and bought.
func (m model) viewMine() string {
	order := m.g.ShedOrder()
	listWidth := 36
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(22, m.width-2)
	}
	avail := max(3, m.height-5)

	cursor := min(m.mineCursor, max(0, len(order)-1))
	scroll := m.mineScroll
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+avail {
		scroll = cursor - avail + 1
	}

	var lines []string
	for i := scroll; i < len(order) && i < scroll+avail; i++ {
		pk := m.g.Shed[order[i]]
		marker := "  "
		style := valueStyle
		if i == cursor {
			marker = titleStyle.Render("› ")
			style = titleStyle
		}
		sw := swatch(pk.Mean().Hex(), 2)
		count := subtleStyle.Render(fmt.Sprintf(" ×%d", pk.Count))
		name := truncate(pk.Name(), listWidth-10)
		lines = append(lines, marker+sw+" "+style.Render(name)+count)
	}
	if len(order) == 0 {
		lines = append(lines,
			subtleStyle.Render("No seed of your own yet."),
			subtleStyle.Render("Buy some in the shop (s),"),
			subtleStyle.Render("or gather ripe seed (f)."))
	}

	detail, clipped := "", false
	switch {
	case len(order) == 0 || narrow:
		if len(order) > 0 {
			pk := m.g.Shed[order[cursor]]
			lines = append(lines, "", fit(subtleStyle.Render(pk.Name()+" · "+pk.Mean().ColourName()), listWidth))
		}
	default:
		detail, clipped = m.packetDetail(m.g.Shed[order[cursor]], listWidth, avail)
	}

	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	head := fit(m.shedHeader(), m.width)
	keys := m.hintLine(setMine)
	if clipped {
		keys = m.hintLine(setMine, "pgup/pgdn read the card")
	}
	if m.naming {
		return strings.Join([]string{head, m.divider(), body, m.divider(), m.input.View(), helpStyle.Render("enter to confirm · esc to cancel")}, "\n")
	}
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}

// familyRange is the span a gene is likely to land in across a packet's
// seedlings: the parents' mean, give or take two standard deviations.
func familyRange(a, b uint8) (lo, hi int) {
	mean := (float64(a) + float64(b)) / 2
	sd := noiseBase + noiseFromGap*absF(float64(a)-float64(b))
	lo, hi = int(mean-2*sd+0.5), int(mean+2*sd+0.5)
	if lo < 0 {
		lo = 0
	}
	if hi > 100 {
		hi = 100
	}
	return lo, hi
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// packetDetail is the card for one packet: what it is, where it came from, and
// what its seedlings are likely to be like.
func (m model) packetDetail(pk Packet, listWidth, avail int) (string, bool) {
	width := m.width - listWidth - 6
	if width < 30 {
		width = 30
	}
	sp := pk.Species()
	if sp == nil {
		return cardBorder.Width(width).Render(subtleStyle.Render("A seed of something this garden does not know.")), false
	}
	mean := pk.Mean()
	variety, _ := sp.NearestVariety(mean)
	artRows := 6
	if avail < 26 {
		artRows = 4
	}
	art := strings.Join(renderGene(sp, variety, mean, StageMature, sp.PaletteForGenome(variety, mean, nil),
		min(width, 24), artRows, 0, nil), "\n")

	rows := []string{
		titleStyle.Render(pk.Name()),
		latinStyle.Render(sp.Latin),
		subtleStyle.Render(packetOrigin(pk)),
		"",
		art,
		"",
		valueStyle.Render(fmt.Sprintf("%d seed(s) in this packet", pk.Count)) +
			subtleStyle.Render(fmt.Sprintf(" · sells for %s each", goldLabel(seedValue(pk)))),
		"",
	}
	if pk.Pure {
		rows = append(rows, labelStyle.Render("GENES")+subtleStyle.Render("  every seed comes true"))
		rows = append(rows, geneRows(sp, mean, width)...)
	} else {
		rows = append(rows, labelStyle.Render("PARENTS"))
		rows = append(rows, "  "+labelStyle.Render(pad("mother", 8))+swatch(pk.A.Hex(), 4)+" "+
			valueStyle.Render(pk.A.ColourName()+", "+pk.A.HeightWord()+", "+pk.A.SpeedWord()+", "+pk.A.YieldWord()))
		rows = append(rows, "  "+labelStyle.Render(pad("father", 8))+swatch(pk.B.Hex(), 4)+" "+
			valueStyle.Render(pk.B.ColourName()+", "+pk.B.HeightWord()+", "+pk.B.SpeedWord()+", "+pk.B.YieldWord()))
		rows = append(rows, "", labelStyle.Render("SEEDLINGS")+subtleStyle.Render("  centre of the family, and the spread to expect"))
		rows = append(rows, geneRows(sp, mean, width)...)
		for _, tr := range []struct {
			name string
			a, b uint8
		}{{"height", pk.A.Height, pk.B.Height}, {"shape", pk.A.Shape, pk.B.Shape}, {"speed", pk.A.Speed, pk.B.Speed}, {"yield", pk.A.Yield, pk.B.Yield}} {
			lo, hi := familyRange(tr.a, tr.b)
			rows = append(rows, "  "+labelStyle.Render(pad(tr.name, 8))+valueStyle.Render(fmt.Sprintf("%d to %d", lo, hi))+
				subtleStyle.Render(spread(hi-lo)))
		}
		rows = append(rows, "", labelStyle.Render("A SAMPLE OF THE FAMILY")+subtleStyle.Render("  colours the seedlings might be"))
		rows = append(rows, "  "+familySample(pk, 14))
	}
	rows = append(rows, "")
	rows = append(rows, wrapped(subtleStyle, "Each seed is rolled when it is sown, so sowing several gives different plants: keep the ones you like and sow from those.", width-2)...)

	lines := strings.Split(strings.Join(rows, "\n"), "\n")
	visible, above, below := window(lines, m.cardScroll, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n")), above || below
}

// spread words the width of a gene's range.
func spread(w int) string {
	switch {
	case w <= 10:
		return "  · very steady"
	case w <= 30:
		return "  · some variation"
	}
	return "  · wide open"
}

// packetOrigin says where a packet came from.
func packetOrigin(pk Packet) string {
	switch {
	case pk.From != "" && pk.Pure:
		return "kept from " + pk.From
	case pk.From != "":
		return fmt.Sprintf("%s generation, gathered from %s", ordinal(pk.Gen), pk.From)
	case pk.Pure:
		return "bought from the shop · true to type"
	}
	return fmt.Sprintf("%s generation", ordinal(pk.Gen))
}

// familySample shows the colours of a few seedlings the packet might give, rolled
// from a salt of their own so they are an indication and not the real seeds.
func familySample(pk Packet, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		c := breed(pk.ID^0x5A17, int64(i), pk.A, pk.B)
		b.WriteString(swatch(c.Hex(), 2))
	}
	return b.String()
}

func wrapped(style lipgloss.Style, text string, width int) []string {
	var out []string
	for _, l := range wrapText(text, max(16, width)) {
		out = append(out, style.Render(l))
	}
	return out
}
