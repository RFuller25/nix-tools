package main

import (
	"fmt"
	"strings"
)

// traitLines is the GENES block of a plant's card: its colour as a swatch and a
// name, and the four other genes as bars with a word and what they do.
func traitLines(sp *Species, p *Plot, width int) []string {
	gn := p.Genes()
	out := []string{labelStyle.Render("GENES") + subtleStyle.Render("  "+geneLineage(p))}
	out = append(out, geneRows(sp, gn, width)...)
	return out
}

// geneRows renders one genome as rows. It is shared by the plant card, the
// seed shed and the almanac so every place reads a genome the same way.
func geneRows(sp *Species, gn Genome, width int) []string {
	bar := func(v uint8) string { return meter(float64(v)/100, 10, okStyle) }
	row := func(label, bars, text string) string {
		return fit("  "+labelStyle.Render(pad(label, 8))+bars+"  "+valueStyle.Render(text), width)
	}
	pods := fmt.Sprintf("holds %.0f pods, ripens ×%.2f", gn.PodCap(), gn.PodFactor())
	return []string{
		fit("  "+labelStyle.Render(pad("colour", 8))+swatch(gn.Hex(), 10)+"  "+
			valueStyle.Render(gn.ColourName())+subtleStyle.Render(" · "+gn.Hex()+" · "+sp.SlotName()), width),
		row("height", bar(gn.Height), gn.HeightWord()+" · "+sp.HeightText(gn)),
		row("shape", bar(gn.Shape), gn.ShapeWord()+fmt.Sprintf(" · catches ×%.2f wind", gn.WindCatch())),
		row("speed", bar(gn.Speed), gn.SpeedWord()+fmt.Sprintf(" · grows ×%.2f", gn.GrowthFactor())),
		row("yield", bar(gn.Yield), gn.YieldWord()+" · "+pods),
	}
}

// geneLineage says where the plant came from.
func geneLineage(p *Plot) string {
	switch {
	case p.Gen == 0 && !p.IsHybrid():
		return "a named form, true to type"
	case p.Gen == 0:
		return "a named form, drifted"
	case p.IsHybrid():
		return fmt.Sprintf("a hybrid, %s generation from a named form", ordinal(p.Gen))
	}
	return fmt.Sprintf("%s generation, still close to its ‘%s’ parents", ordinal(p.Gen), p.VarietyName())
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

var _ = strings.Join
