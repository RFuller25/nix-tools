package main

import "fmt"

// What a plant actually does in the garden, as opposed to what it looks like.
// Both the almanac and the info card show this, so a gardener can tell before
// sowing whether a thing will spread, sulk in the wrong soil, feed its
// neighbours or shut its flowers the moment the sun goes down.

// effect is one line of the in-the-garden block.
type effect struct {
	Glyph string
	Label string
	Text  string
	Good  bool // worth colouring as a benefit
	Warn  bool // worth colouring as a caution
}

// gardenEffects lists everything a species does, in the order a gardener
// would want to hear it.
func gardenEffects(sp *Species) []effect {
	var out []effect
	add := func(glyph, label, text string) {
		out = append(out, effect{Glyph: glyph, Label: label, Text: text})
	}
	good := func(glyph, label, text string) {
		out = append(out, effect{Glyph: glyph, Label: label, Text: text, Good: true})
	}
	warn := func(glyph, label, text string) {
		out = append(out, effect{Glyph: glyph, Label: label, Text: text, Warn: true})
	}

	// Where it will and will not grow.
	if sp.Kind == KindAquatic {
		warn("≈", "needs", "a pond — it will not grow on dry land")
	}
	add("☀", "season", fmt.Sprintf("%s · grows about a third slower out of season", sp.SeasonNames()))
	add("⚗", "soil", soilWant(sp))

	// How it spends its year.
	switch sp.Life() {
	case Annual:
		add("❧", "lifespan", "annual · flowers, sets seed and stands dry until lifted")
	case Biennial:
		add("❧", "lifespan", "biennial · leaves first, then flowers, then seeds")
	case Woody:
		if sp.Evergreen() {
			add("❧", "lifespan", "evergreen shrub or tree · keeps its leaves all winter")
		} else {
			add("❧", "lifespan", "shrub or tree · stands bare through the winter")
		}
	default:
		if sp.Evergreen() {
			add("❧", "lifespan", "evergreen perennial · carries on year to year")
		} else {
			add("❧", "lifespan", "perennial · dies back in winter, returns in spring")
		}
	}

	// Whether it stays where it is put.
	if sp.SelfSeeds() {
		good("✦", "spreads", "sows itself into bare ground in the beds alongside")
	}

	// What it does to the neighbours, and what they do to it.
	for _, c := range companions {
		if !c.from(sp) {
			continue
		}
		line := fmt.Sprintf("%+.0f%% to %s — %s", c.delta*100, companionTargets(c), c.note)
		if c.delta >= 0 {
			good("⚘", "gives", line)
		} else {
			warn("⚘", "costs", line)
		}
	}
	for _, c := range companions {
		if !c.who(sp) || c.from(sp) {
			continue
		}
		line := fmt.Sprintf("%+.0f%% beside %s — %s", c.delta*100, companionTargets2(c), c.note)
		if c.delta >= 0 {
			good("⚘", "likes", line)
		} else {
			warn("⚘", "dislikes", line)
		}
	}

	// Its hours.
	switch {
	case sp.OpensAtNight():
		add("☾", "hours", "shut all day, open from dusk, scented for the moths")
	case sp.ClosesAtNight():
		add("☾", "hours", "folds its flowers shut for the night")
	case sp.NightScented():
		add("☾", "hours", "carries its scent after dark")
	}

	// Who comes to it.
	if visitors := visitorsFor(sp); visitors != "" {
		good("✽", "visitors", visitors)
	}

	// The plain numbers.
	add("⏱", "matures", fmt.Sprintf("about %s of well-tended growth", hours(sp.Hours)))
	add("✦", "seeds", fmt.Sprintf("costs %d · a mature plant ripens a pod every six hours", sp.SeedCost))
	if sp.Unlock > 0 {
		add("⚑", "unlocks", fmt.Sprintf("after %d plants have reached maturity", sp.Unlock))
	}
	return out
}

// soilWant puts a species' soil preference into words a gardener can act on.
func soilWant(sp *Species) string {
	switch sp.PrefersSoil() {
	case soilAcid:
		return "acid ground, pH 6.2 and below · slower in lime"
	case soilChalk:
		return "chalk and lime, pH 6.9 and above · slower in acid"
	case soilNeutral:
		return "the middle ground, pH 6.0 to 7.3 · slower at either extreme"
	default:
		return "easy-going — any bed will do"
	}
}

// companionTargets names what a rule affects, for the species causing it.
func companionTargets(c companion) string {
	return describeMatch(c.who)
}

// companionTargets2 names what causes a rule, for the species receiving it.
func companionTargets2(c companion) string {
	return describeMatch(c.from)
}

// describeMatch turns a rule's predicate back into something readable by
// seeing which species it actually picks out.
func describeMatch(match func(*Species) bool) string {
	var hits []*Species
	for _, sp := range AllSpecies() {
		if match(sp) {
			hits = append(hits, sp)
		}
	}
	switch {
	case len(hits) == 0:
		return "nothing"
	case len(hits) > 12:
		return "anything beside it"
	case len(hits) > 3:
		// A whole family, most likely.
		family := hits[0].Family
		for _, sp := range hits {
			if sp.Family != family {
				return fmt.Sprintf("%d kinds of plant", len(hits))
			}
		}
		return "the " + family
	}
	names := make([]string, 0, len(hits))
	for _, sp := range hits {
		names = append(names, sp.Common)
	}
	return joinWords(names)
}

func joinWords(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	case 2:
		return words[0] + " and " + words[1]
	}
	out := ""
	for i, w := range words[:len(words)-1] {
		if i > 0 {
			out += ", "
		}
		out += w
	}
	return out + " and " + words[len(words)-1]
}

// visitorsFor names the creatures a species draws in.
func visitorsFor(sp *Species) string {
	var names []string
	if attractsBees(sp) {
		names = append(names, "bees")
	}
	if attractsButterflies(sp) {
		names = append(names, "butterflies")
	}
	if sp.Family == "Asteraceae" {
		names = append(names, "finches, once the seed heads dry")
	}
	if sp.NightScented() || sp.OpensAtNight() {
		names = append(names, "moths after dark")
	}
	if sp.Kind == KindAquatic {
		names = append(names, "dragonflies over the water")
	}
	return joinWords(names)
}

// effectLines renders the in-the-garden block to fit a card's width.
func effectLines(sp *Species, width int) []string {
	const labelW = 13
	textW := max(16, width-labelW-2)

	out := []string{labelStyle.Render("IN THE GARDEN")}
	for _, e := range gardenEffects(sp) {
		style := valueStyle
		switch {
		case e.Good:
			style = okStyle
		case e.Warn:
			style = warnStyle
		}

		head := subtleStyle.Render(e.Glyph+" ") + labelStyle.Render(pad(e.Label, labelW-2))
		for i, line := range wrapText(e.Text, textW) {
			if i == 0 {
				out = append(out, head+style.Render(line))
				continue
			}
			out = append(out, spaces(labelW)+style.Render(line))
		}
	}
	return out
}

// wrapText breaks a sentence onto lines no wider than the space given.
func wrapText(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	line := ""
	for _, word := range splitWords(s) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= width:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

func splitWords(s string) []string {
	var out []string
	word := ""
	for _, r := range s {
		if r == ' ' {
			if word != "" {
				out = append(out, word)
				word = ""
			}
			continue
		}
		word += string(r)
	}
	if word != "" {
		out = append(out, word)
	}
	return out
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}
