package main

import "strings"

// helpPrimer is the short orientation under the keys. The almanac's guide
// goes into everything properly.
var helpPrimer = [][2]string{
	{"time", "plants grow in real time, even while closed"},
	{"a day", "well-tended seeds reach maturity within a day"},
	{"water", "dry soil slows growth — nothing ever dies here"},
	{"weeds", "creep in slowly and hold plants back"},
	{"weather", "rain waters for you; frost and snow slow things"},
	{"light", "the garden follows the real sun; some flowers close at night"},
	{"soil", "each bed has its own pH; lifting a plant composts it"},
	{"neighbours", "what grows alongside helps or hinders — see the info card"},
	{"the year", "annuals go to seed, perennials sleep through winter"},
	{"visitors", "bees, moths and foxes come for what you have planted"},
	{"gold", "earned by selling your own seed, weeding and doing; spent on seed, beds and ponds"},
	{"genes", "every plant has colour, height, shape, speed and yield genes; seed blends its parents"},
}

// viewHelp is the key list, generated from the binding table, plus a primer.
// The almanac's guide chapters hold the long form of everything here.
func (m model) viewHelp() string {
	var lines []string
	lines = append(lines, titleStyle.Render("❀ garden — help"),
		subtleStyle.Render("the almanac (a) has a guide to everything the garden does"), "")

	for _, set := range setOrder {
		var rows []string
		for _, d := range documentedKeys() {
			if d.Set == set {
				rows = append(rows, "  "+okStyle.Render(pad(d.Label, 22))+valueStyle.Render(d.Desc))
			}
		}
		if len(rows) == 0 {
			continue
		}
		lines = append(lines, labelStyle.Render(strings.ToUpper(setTitles[set])))
		lines = append(lines, rows...)
		lines = append(lines, "")
	}

	lines = append(lines, labelStyle.Render("HOW IT GROWS"))
	for _, k := range helpPrimer {
		lines = append(lines, "  "+okStyle.Render(pad(k[0], 22))+valueStyle.Render(k[1]))
	}
	lines = append(lines, "", subtleStyle.Render("saved to "+m.path))

	// Help is long; on a short terminal it scrolls instead of running off it.
	visible, above, below := window(lines, m.cardScroll, max(2, m.height-1))
	keys := scrollHint(above, below, "any other key returns to the garden")
	return strings.Join(append(visible, helpStyle.Render(keys)), "\n")
}
