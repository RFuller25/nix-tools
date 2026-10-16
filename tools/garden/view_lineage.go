package main

import (
	"fmt"
	"strings"
)

// The genetics screen. It reads the record a plant, a packet of seed or a
// cultivar carries and tells the story of the line: where its first generation
// came from, what each generation was bred from and came out as, which genes
// mutated or drifted along the way, and how that stacks up against the
// definition of the cultivar, if there is one. It is meant to show how the
// breeding worked, with the numbers it used, and not only what the genes are.

// lineageSubject is whatever the genetics screen is reading.
type lineageSubject struct {
	Title    string
	Species  *Species
	Steps    []Step
	Now      Genome    // the genes of the thing itself, which ends the trail
	Gen      int       // its generation from a named form
	Descent  string    // in words, for a record from before steps were kept
	Cultivar *Cultivar // the definition to hold each generation up to, if any
	Pending  *Step     // for a packet: the generation its seed will make
	Pack     *Packet   // for a packet: the packet
	What     string    // "plant", "packet" or "cultivar"
}

// plantLineage is the subject for a plant in a bed.
func (g *Garden) plantLineage(p *Plot) (lineageSubject, bool) {
	sp := p.Species()
	if sp == nil {
		return lineageSubject{}, false
	}
	sub := lineageSubject{
		Title: p.FullName(), Species: sp, Steps: p.History, Now: p.Genes(), Gen: p.Gen, Descent: p.Descent, What: "plant",
	}
	if c := g.CultivarByID(p.Line); c != nil {
		sub.Cultivar = c
	}
	return sub, true
}

// packetLineage is the subject for a packet in the shed.
func (g *Garden) packetLineage(pk Packet) (lineageSubject, bool) {
	sp := pk.Species()
	if sp == nil {
		return lineageSubject{}, false
	}
	next := pk.pendingStep(sp, pk.Mean())
	sub := lineageSubject{
		Title: pk.Name(), Species: sp, Steps: pk.History, Now: pk.Mean(), Gen: pk.Gen, Descent: pk.Descent, What: "packet",
		Pending: &next, Pack: &pk,
	}
	if c := g.CultivarByID(pk.Line); c != nil {
		sub.Cultivar = c
	}
	return sub, true
}

// cultivarLineage is the subject for a cultivar in the almanac.
func (g *Garden) cultivarLineage(c *Cultivar) (lineageSubject, bool) {
	sp := c.SpeciesRef()
	if sp == nil {
		return lineageSubject{}, false
	}
	return lineageSubject{
		Title: c.Name, Species: sp, Steps: c.History, Now: c.Genome, Gen: c.Gen, Descent: c.Descent, Cultivar: c, What: "cultivar",
	}, true
}

// openLineage shows the genetics screen for a subject, remembering where to
// return to.
func (m *model) openLineage(sub lineageSubject, ok bool) {
	if !ok {
		m.setStatus(subtleStyle, "Nothing to trace here.")
		return
	}
	m.lineage, m.lineageBack = sub, m.screen
	m.screen, m.cardScroll = screenLineage, 0
}

// stepSource says where a first generation came from.
func stepSource(s Step, sp *Species) string {
	switch s.Kind {
	case stepShop, stepCultivar:
		return s.Note + ", bought from the shop"
	case stepWild:
		return "a wild seed brewed by the gnomes"
	case stepGift:
		return s.Note
	}
	return "an earlier generation, whose own beginnings were not written down"
}

// plainGenes is a genome as a short line of figures.
func plainGenes(g Genome) string {
	return fmt.Sprintf("hue %d° sat %d light %d · height %d shape %d speed %d yield %d",
		g.Hue, g.Sat, g.Light, g.Height, g.Shape, g.Speed, g.Yield)
}

func percent(d float64) string { return fmt.Sprintf("%.0f%%", d*100) }

// lineageLines builds the whole page.
func (m model) lineageLines(sub lineageSubject, width int) []string {
	sp := sub.Species
	steps := sub.Steps
	rep := reportLineage(steps)
	var out []string
	add := func(l ...string) { out = append(out, l...) }
	wrap := func(style func(string) string, text string, indent string) {
		for _, l := range wrapText(text, max(16, width-len(indent)-2)) {
			out = append(out, indent+style(l))
		}
	}
	valueS := func(s string) string { return valueStyle.Render(s) }
	subtleS := func(s string) string { return subtleStyle.Render(s) }

	add(titleStyle.Render("❦ genetics — "+sub.Title), latinStyle.Render(sp.Latin)+subtleStyle.Render("  ·  "+sub.What))

	// ---- the short version
	add("", labelStyle.Render("THE SHORT VERSION"))
	switch {
	case rep.Known:
		depth := fmt.Sprintf("%d generation(s) deep", rep.Depth)
		if rep.Depth == 0 {
			depth = "the first generation"
		}
		wrap(valueS, fmt.Sprintf("%s: generation %d back to generation %d. It began as %s.",
			depth, steps[len(steps)-1].Gen, steps[0].Gen, stepSource(rep.Origin, sp)), "  ")
		add("  " + labelStyle.Render(pad("mutations", 11)) + valueStyle.Render(fmt.Sprintf("%d", len(rep.Mutations))) +
			subtleStyle.Render(fmt.Sprintf("  (about %s of seeds mutate a gene)", percent(mutationOdds*float64(geneCount)))))
		add("  " + labelStyle.Render(pad("drift", 11)) + valueStyle.Render(fmt.Sprintf("%d noted", len(rep.Drifts))) +
			subtleStyle.Render("  (a gene landing well off its parents' middle)"))
		add("  " + labelStyle.Render(pad("wandered", 11)) + valueStyle.Render(percent(rep.Wander)+" from the first generation") +
			subtleStyle.Render("  · furthest "+percent(rep.Peak)))
	default:
		wrap(valueS, fmt.Sprintf("This %s is generation %d from a named form, but it was grown before lineages were recorded, so the generations between are not written down.", sub.What, sub.Gen), "  ")
		if sub.Descent != "" {
			wrap(subtleS, "What the garden remembers: "+sub.Descent+".", "  ")
		}
		wrap(subtleS, "From here on it is recorded: seed gathered from it carries its lineage forward.", "  ")
	}

	// ---- the definition
	if c := sub.Cultivar; c != nil {
		add("", labelStyle.Render("AGAINST THE DEFINITION OF ‘"+strings.ToUpper(c.Name)+"’"))
		if len(steps) > 0 {
			fits := make([]bool, len(steps))
			inN, run := 0, 0
			for i, s := range steps {
				fits[i] = c.Admits(s.Genome)
				if fits[i] {
					inN++
				}
			}
			for i := len(steps) - 1; i >= 0 && fits[i]; i-- {
				run++
			}
			line := fmt.Sprintf("%d of %d recorded generations fit it.", inN, len(steps))
			switch {
			case inN == len(steps):
				line = fmt.Sprintf("Every one of the %d recorded generations fits it.", len(steps))
			case run > 0:
				line += fmt.Sprintf(" The line has been inside it since generation %d.", steps[len(steps)-run].Gen)
			case inN > 0:
				for i := len(steps) - 1; i >= 0; i-- {
					if fits[i] {
						line += fmt.Sprintf(" The last generation to fit was %d; the line has strayed since.", steps[i].Gen)
						break
					}
				}
			default:
				line = fmt.Sprintf("None of the %d recorded generations fits it.", len(steps))
			}
			wrap(valueS, line, "  ")
		}
		if strays := c.Strays(sub.Now); len(strays) > 0 && sub.What != "cultivar" {
			wrap(func(s string) string { return warnStyle.Render(s) }, "As it stands now it is outside the definition: "+c.strayWords(sub.Now)+".", "  ")
		} else if sub.What != "cultivar" {
			add("  " + okStyle.Render("As it stands now it fits the definition."))
		}
	}

	// ---- the timeline
	add("", labelStyle.Render("HOW THE LINE RAN")+subtleStyle.Render("  oldest first"))
	if !rep.Known {
		add("  " + subtleStyle.Render("· earlier generations — not recorded"))
	}
	var prev *Step
	for i := range steps {
		s := steps[i]
		out = append(out, m.stepLines(sub, s, i, prev, width)...)
		prev = &steps[i]
	}
	// The thing itself, where the record has not caught up with it (a plant
	// changed since, or a packet whose seed is not sown).
	if sub.Pending != nil {
		out = append(out, m.pendingLines(sub, width)...)
	} else if len(steps) > 0 && steps[len(steps)-1].Genome != sub.Now {
		add("  " + warnStyle.Render("◇ now") + subtleStyle.Render("  "+describeGenome(sub.Now)+" — the genes have moved since the last record"))
	}

	// ---- gene traces
	if len(steps) > 1 {
		add("", labelStyle.Render("GENE BY GENERATION")+subtleStyle.Render("  one mark a generation, oldest on the left"))
		var hues strings.Builder
		for _, s := range steps {
			hues.WriteString(swatch(s.Genome.Hex(), 1))
		}
		first, last := steps[0].Genome, steps[len(steps)-1].Genome
		add("  " + labelStyle.Render(pad("colour", 11)) + hues.String() + "  " +
			valueStyle.Render(first.ColourName()+" → "+last.ColourName()))
		for gi := 1; gi < geneCount; gi++ {
			tr := geneTrace(steps, gi)
			if gi == 1 || gi == 2 {
				continue // saturation and lightness are shown in the colour row
			}
			delta := tr[len(tr)-1] - tr[0]
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			add("  " + labelStyle.Render(pad(geneNames[gi], 11)) + okStyle.Render(sparkline(tr)) + "  " +
				valueStyle.Render(fmt.Sprintf("%d → %d", tr[0], tr[len(tr)-1])) + subtleStyle.Render(fmt.Sprintf("  (%s%d)", sign, delta)))
		}
	}

	// ---- how it works
	add("", labelStyle.Render("HOW BREEDING WORKS HERE"))
	for _, l := range []string{
		fmt.Sprintf("Each gene of a seedling is the average of its parents' genes, plus a random nudge. The nudge is about ±%.0f when the parents are identical, and widens by %.2f for every point they differ, so crossing unlike plants scatters the seedlings and crossing near-identical ones holds a line steady.", noiseBase, noiseFromGap),
		fmt.Sprintf("On top of that each gene has a %s chance of mutating outright: a jump of %.0f to %.0f points (%.0f to %.0f degrees of hue).", percent(mutationOdds), mutationLow, mutationHigh, hueMutateLow, hueMutateHigh),
		fmt.Sprintf("A line has bred true when a seed comes out within %s of its parents' middle; %d generations running makes it stable.", percent(stableGap), stableRuns),
	} {
		wrap(subtleS, l, "  ")
	}
	return out
}

// stepLines is one generation on the timeline.
func (m model) stepLines(sub lineageSubject, s Step, i int, prev *Step, width int) []string {
	sp := sub.Species
	var out []string
	glyph := "○"
	style := valueStyle
	switch {
	case s.Kind.Origin():
		glyph = "●"
		style = goldStyle
	case s.Mutations() > 0 || s.Kind == stepGnome:
		glyph = "◆"
		style = warnStyle
	}
	head := fmt.Sprintf("gen %d", s.Gen)
	title := kindWords(s.Kind)
	if s.Kind.Origin() {
		title = stepSource(s, sp)
	} else if s.Kind == stepGnome && s.Note != "" {
		title += ": " + s.Note
	} else if s.Note != "" && s.Kind != stepSelf && s.Kind != stepCross {
		title += " — " + s.Note
	}
	when := ""
	if !s.At.IsZero() {
		when = subtleStyle.Render("  " + s.At.Format("2 Jan"))
	}
	mark := ""
	if c := sub.Cultivar; c != nil {
		if c.Admits(s.Genome) {
			mark = "  " + okStyle.Render("✓ fits")
		} else {
			mark = "  " + warnStyle.Render("✗ outside")
		}
	}
	out = append(out, fit("  "+style.Render(glyph+" "+pad(head, 6))+style.Render(title)+when+mark, width))

	indent := "      "
	// The rule that applied is said once, and again only when it changes.
	explain := func(s Step) []string {
		how := howItWorks(s)
		if how == "" || (prev != nil && howItWorks(*prev) == how) {
			return nil
		}
		var lines []string
		for _, l := range wrapText(how, max(16, width-len(indent)-2)) {
			lines = append(lines, indent+subtleStyle.Render(l))
		}
		return lines
	}
	if s.Mother != nil && s.Father != nil {
		out = append(out, fit(indent+subtleStyle.Render("mother ")+swatch(s.Mother.Hex(), 2)+subtleStyle.Render(" × father ")+
			swatch(s.Father.Hex(), 2)+subtleStyle.Render(" → seedling ")+swatch(s.Genome.Hex(), 2)+" "+
			valueStyle.Render(describeGenome(s.Genome)), width))
		out = append(out, explain(s)...)
	} else {
		out = append(out, fit(indent+swatch(s.Genome.Hex(), 2)+" "+valueStyle.Render(describeGenome(s.Genome)), width))
		out = append(out, explain(s)...)
	}
	for _, sh := range s.Shifts {
		spread := s.Spread(sh.Gene)
		kind, st := "drift", valueStyle
		extra := ""
		if sh.Mutation {
			kind, st = "mutation", warnStyle
			if spread > 0 {
				extra = fmt.Sprintf("  (ordinary spread here is only ±%.0f)", spread)
			}
		} else if spread > 0 {
			extra = fmt.Sprintf("  (%.1f× the usual spread of ±%.0f)", float64(iabs(sh.Delta()))/spread, spread)
		}
		out = append(out, fit(indent+st.Render("⚡ "+kind+": "+shiftWords(sh))+subtleStyle.Render(extra), width))
	}
	if sub.Cultivar != nil && !sub.Cultivar.Admits(s.Genome) {
		for _, l := range wrapText("outside ‘"+sub.Cultivar.Name+"’: "+sub.Cultivar.strayWords(s.Genome), max(16, width-len(indent)-2)) {
			out = append(out, indent+warnStyle.Render(l))
		}
	}
	if prev != nil {
		d := s.Genome.Distance(prev.Genome)
		if d >= 0.02 {
			out = append(out, fit(indent+subtleStyle.Render(fmt.Sprintf("moved %s from the generation before", percent(d))), width))
		}
	}
	return out
}

// pendingLines shows what a packet's seed will make next: its parents, and the
// spread of what to expect.
func (m model) pendingLines(sub lineageSubject, width int) []string {
	s := *sub.Pending
	var out []string
	out = append(out, fit("  "+goldStyle.Render("◇ "+pad(fmt.Sprintf("gen %d", s.Gen), 6))+goldStyle.Render("this packet: "+kindWords(s.Kind))+
		subtleStyle.Render("  not yet sown"), width))
	indent := "      "
	if s.Mother != nil && s.Father != nil {
		out = append(out, fit(indent+subtleStyle.Render("mother ")+swatch(s.Mother.Hex(), 2)+subtleStyle.Render(" × father ")+
			swatch(s.Father.Hex(), 2)+subtleStyle.Render(" → centre of the family ")+swatch(s.Genome.Hex(), 2), width))
		for _, tr := range []struct {
			name string
			a, b uint8
		}{{"height", s.Mother.Height, s.Father.Height}, {"shape", s.Mother.Shape, s.Father.Shape}, {"speed", s.Mother.Speed, s.Father.Speed}, {"yield", s.Mother.Yield, s.Father.Yield}} {
			lo, hi := familyRange(tr.a, tr.b)
			out = append(out, fit(indent+labelStyle.Render(pad(tr.name, 8))+valueStyle.Render(fmt.Sprintf("%d to %d", lo, hi))+
				subtleStyle.Render(spread(hi-lo)), width))
		}
		if how := howItWorks(s); how != "" {
			for _, l := range wrapText(how, max(16, width-len(indent)-2)) {
				out = append(out, indent+subtleStyle.Render(l))
			}
		}
	}
	if c := sub.Cultivar; c != nil {
		out = append(out, fit(indent+valueStyle.Render(fmt.Sprintf("about %s of its seedlings would fit ‘%s’",
			percent(c.Fits(*sub.Pack)), c.Name)), width))
	}
	return out
}

// viewLineage is the genetics screen.
func (m model) viewLineage() string {
	width := max(30, min(m.width-4, 100))
	lines := m.lineageLines(m.lineage, width)
	avail := max(3, m.height-5)
	visible, above, below := window(lines, m.cardScroll, avail)
	keys := m.hintLine(setLineage)
	if above || below {
		keys += " · pgup/pgdn read on"
	}
	head := fit(titleStyle.Render("❦ genetics")+subtleStyle.Render("  ·  "+m.lineage.Title), m.width)
	for i := range visible {
		visible[i] = fit(visible[i], m.width)
	}
	return strings.Join([]string{head, m.divider(), strings.Join(visible, "\n"), m.divider(), m.footer(keys)}, "\n")
}
