package main

import (
	"fmt"
	"strings"
)

// The almanac's guide. Every feature of the game has a chapter, and wherever a
// chapter quotes a number, a key or a rule it reads it from the same constant,
// table or binding the game itself runs on, so the guide cannot drift from the
// behaviour. Tests walk the key table and fail if a documented key is missing.

type chapter struct {
	Title string
	Lede  string // one line shown under the title
	Build func(m model, width int) []string
}

// guideChapters is the guide in reading order. Features add their chapters
// where they belong.
func guideChapters() []chapter {
	return []chapter{
		chGettingStarted(),
		chTending(),
		chGrowing(),
		chSoil(),
		chNeighbours(),
		chYear(),
		chVisitors(),
		chSeeds(),
		chWorthDoing(),
		chKeys(),
		chCommandLine(),
	}
}

// ---- helpers --------------------------------------------------------------

func gp(text string, width int) []string {
	var out []string
	for _, line := range wrapText(text, max(16, width-2)) {
		out = append(out, valueStyle.Render(line))
	}
	return out
}

func gh(text string) string { return labelStyle.Render(strings.ToUpper(text)) }

// gk is one row of a key or definition table: a label, then wrapped text.
func gk(label, text string, width int) []string {
	const labelW = 16
	textW := max(16, width-labelW-4)
	var out []string
	for i, line := range wrapText(text, textW) {
		if i == 0 {
			out = append(out, "  "+okStyle.Render(pad(label, labelW))+valueStyle.Render(line))
			continue
		}
		out = append(out, spaces(labelW+2)+valueStyle.Render(line))
	}
	if len(out) == 0 {
		out = append(out, "  "+okStyle.Render(label))
	}
	return out
}

func join(parts ...[]string) []string {
	var out []string
	for i, p := range parts {
		if i > 0 && len(p) > 0 {
			out = append(out, "")
		}
		out = append(out, p...)
	}
	return out
}

func lines(ss ...string) []string { return ss }

// keyDoc is one documented binding.
type keyDoc struct {
	Set   keySet
	Label string
	Desc  string
}

// documentedKeys lists every binding that carries a description, in the order
// the sets are presented.
func documentedKeys() []keyDoc {
	var out []keyDoc
	for _, set := range setOrder {
		for _, b := range bindingsIn(set) {
			if b.Desc != "" {
				out = append(out, keyDoc{Set: set, Label: b.Label, Desc: b.Desc})
			}
		}
	}
	return out
}

// keyRows renders the documented keys of the given sets as definition rows.
func keyRows(width int, sets ...keySet) []string {
	var out []string
	for _, d := range documentedKeys() {
		for _, s := range sets {
			if d.Set == s {
				out = append(out, gk(d.Label, d.Desc, width)...)
			}
		}
	}
	return out
}

// ---- chapters -------------------------------------------------------------

func chGettingStarted() chapter {
	return chapter{
		Title: "Getting started",
		Lede:  "Five beds across, growing in real time.",
		Build: func(m model, w int) []string {
			return join(
				gp("A garden that grows in real time. Sow a seed, water it, pull the weeds, and come back tomorrow to something taller. Time passes while the program is closed: on startup the garden replays the hours you were away, weather and all (up to "+fmt.Sprintf("%.0f", maxCatchUpDays)+" days).", w),
				gp("Nothing here dies. Dry soil and weeds slow a plant and make it sulk; they never kill it. Come back after a fortnight and your garden is overgrown, not gone.", w),
				append([]string{gh("The screens")},
					join(
						gk("garden", "the beds. It is five beds wide and grows downwards, so every bed keeps the same neighbours however the terminal is sized.", w),
						gk("seed shed", "where seed is chosen and sown (s).", w),
						gk("almanac", "this book: the guide, every species and every visitor (a).", w),
						gk("journal", "the log, the tally and what is worth doing next.", w),
						gk("info card", "one plant in full (i, or enter on a planted bed).", w),
						gk("help", "the key list (?).", w),
					)...),
				gp("tab cycles garden → shed → almanac → journal. esc or q returns to the garden; q in the garden quits, and the garden saves itself.", w),
				gp("Every screen is built to the window it is given and never drawn larger than the terminal. The garden scrolls both ways rather than reflowing; the card, help and almanac scroll with the arrows or pgup/pgdn.", w),
			)
		},
	}
}

func chTending() chapter {
	return chapter{
		Title: "Tending",
		Lede:  "Water, weeds, names, ground and ponds.",
		Build: func(m model, w int) []string {
			return join(
				gp("Plants dry out over time and weeds creep into every bed, planted or not. Weeds rest through winter. A thirsty bed (moisture under 35%) or a weedy one (weeds over 45%) is flagged on the garden screen; the status line under it nags about the worst.", w),
				append([]string{gh("What you can do")}, keyRows(w, setGarden)...),
				gp(fmt.Sprintf("Clearing a properly overgrown bed (weeds over 45%%) turns up %d %s in the tangle. Breaking new ground costs %d for the first extra bed and %d more for each after it, to a limit of %d beds. A pond costs %d.", weedingReward, currencyWord(weedingReward), bedBaseCost, bedStepCost, maxPlots, pondCost), w),
				gp("Ponds never dry out, never weed over, and are the only place the water lily and the sacred lotus will grow; those two will grow nowhere else. Filling a pond back in returns the bed to ordinary soil.", w),
				gp("Naming a plant (n) is only for you: the name shows on the bed and the card, and the journal notes it.", w),
			)
		},
	}
}

func chGrowing() chapter {
	return chapter{
		Title: "How things grow",
		Lede:  "Stages, time, weather, light and wind.",
		Build: func(m model, w int) []string {
			return join(
				gp("Every plant has five drawn stages: seed, sprout, seedling, budding, mature, reached at 10%, 35%, 70% and 100% of its growth. A watered, weeded plant in its own season reaches maturity in the hours its card advertises, always inside a day.", w),
				gp("Growth speed is moisture, weeds, season, sky, soil and neighbours multiplied together. It never reaches zero. A plant out of its season grows about 30% slower. Moisture under 60% slows it, and weeds take up to 40% off.", w),
				append([]string{gh("Weather")},
					gp("Each day's weather comes from the calendar and your garden's own seed, so it is the same every time the missing hours are replayed. Spring, summer, autumn and winter each draw from their own table: sun, clear, cloud, showers, rain, storms, fog, frost, snow and heatwaves. Rain waters the beds for you; heat dries them faster; a storm or a frost slows growth a little.", w)...),
				append([]string{gh("Light")},
					gp("The garden runs on the real sun. Dawn comes up rose, dusk goes amber, night settles blue and dim, and day length follows the season. Crocus, tulip, water lily, lotus, morning glory, snowdrop and chamomile fold shut for the dark. Moonflower, evening primrose and night-scented stock do the opposite: shut all day, open at dusk.", w)...),
				append([]string{gh("Wind")},
					gp("Gusts blow through at random, more often in a storm than in fog. They start at the left and cross the beds one after another, plants bending with the tip moving furthest. A tall, mature plant catches more wind than a seedling. It is weather to watch, not state to keep, so it never changes how anything grows.", w)...),
			)
		},
	}
}

func chSoil() chapter {
	return chapter{
		Title: "Soil and compost",
		Lede:  "pH, richness, and what lifting a plant does.",
		Build: func(m model, w int) []string {
			return join(
				gp("Every bed has its own pH, from about 4.5 (peat bog) to 8 (chalk), and its own richness, both derived from the garden's seed. Mediterranean herbs want chalk (pH 6.9 and above), bog and woodland plants want acid ground (6.2 and below), and some want the middle (6.0 to 7.3). Most plants are easy-going. A plant in ground it dislikes grows about 15% slower, never badly.", w),
				gp("Richness is how much compost the bed has had worked into it. Rich ground grows a plant up to a quarter faster than poor ground. Growing plants draw it down slowly, and the soil line under every bed shows it, from thin and dotted to dark and solid.", w),
				gp("Lifting a plant (u) collapses it into the bed over a second or so. A tree returns about twice what a seedling does, and a plant gone to seed most of all. The bed's pH drifts towards neutral as the compost buffers it.", w),
				gp("Bigleaf hydrangea reads its bed and flowers blue in acid soil, mauve in the middle and pink in lime: the one plant here doing its own chemistry.", w),
			)
		},
	}
}

func chNeighbours() chapter {
	return chapter{
		Title: "Neighbours",
		Lede:  "Companion planting, from the rules the garden runs on.",
		Build: func(m model, w int) []string {
			out := gp("What you plant alongside matters, using relationships gardeners have really used. A bed's four neighbours (not diagonals) each add or take a percentage of growth speed, capped at +30% and -20% in total. The info card lists what the beds beside a plant are doing and why.", w)
			out = append(out, "")
			out = append(out, gh("The rules"))
			for _, c := range companions {
				sign := "+"
				if c.delta < 0 {
					sign = ""
				}
				out = append(out, gk(fmt.Sprintf("%s%.0f%%", sign, c.delta*100),
					fmt.Sprintf("%s beside %s — %s", describeMatch(c.who), describeMatch(c.from), c.note), w)...)
			}
			return out
		},
	}
}

func chYear() chapter {
	return chapter{
		Title: "The year",
		Lede:  "Annuals, perennials, and plants that sow themselves.",
		Build: func(m model, w int) []string {
			var seeders []string
			for _, sp := range AllSpecies() {
				if sp.SelfSeeds() {
					seeders = append(seeders, sp.Common)
				}
			}
			return join(
				gp("Every species leads the life it really leads. Annuals and biennials flower, set seed and finish: a spent plant is not dead, it stands bleached to straw with a last handful of seed until you lift it. Perennials die back over winter, deciduous trees and shrubs stand bare, evergreens carry on, and all of them wake in spring.", w),
				gp("An annual stays in flower for about two and a half times its growing time, a biennial about three and a half.", w),
				gp("Self-seeders drop volunteers into bare ground beside them, free. The decision is hashed from the garden's seed rather than rolled, so a garden left running and one catching up on a fortnight it spent closed grow exactly the same plants. Self-seeders: "+joinWords(seeders)+".", w),
			)
		},
	}
}

func chVisitors() chapter {
	return chapter{
		Title: "Visitors",
		Lede:  "Seven creatures, each with a page of its own below.",
		Build: func(m model, w int) []string {
			return join(
				gp("Bees work the flowers on a dry day, butterflies follow the nectar, finches drop in on seed heads, dragonflies patrol a pond, and after dark moths come to the night-scented flowers while a fox or a hedgehog crosses the beds. Nothing visits a garden with nothing in it.", w),
				gp("The journal notes each one's first visit, and each has a page under VISITORS in this almanac: what brings it, when it comes, how long it stays, and a tick once you have seen it.", w),
				gp("Visitors are scenery in this version of the garden: they are drawn over the beds and recorded, but do not change growth.", w),
			)
		},
	}
}

func chSeeds() chapter {
	return chapter{
		Title: "Seeds and the shed",
		Lede:  "Currency, planting, and unlocking rarer species.",
		Build: func(m model, w int) []string {
			return join(
				gp(fmt.Sprintf("Seeds are the currency. Collect %d on your first visit each day, earn %d for clearing a properly overgrown bed, gather ripe pods from mature plants (one pod every six hours, up to %d), and finish the suggestions in the journal.", dailyBonus, weedingReward, int(maxPods)), w),
				gp("Rarer species unlock as more of your plants reach maturity. A locked seed is shown with its requirement.", w),
				append([]string{gh("In the seed shed")}, keyRows(w, setShop)...),
				gp("Every species comes in two to five real forms: cultivars where the plant has famous ones, honest colour forms where it does not. Pick one with the left and right arrows before sowing. Self-sown volunteers come true to their parent.", w),
			)
		},
	}
}

func chWorthDoing() chapter {
	return chapter{
		Title: "Worth doing",
		Lede:  "Gentle goals in the journal.",
		Build: func(m model, w int) []string {
			out := gp(fmt.Sprintf("The journal keeps %d suggestions at a time. Never a timer and never a failure: finish one and its reward arrives and another appears. They are picked in a way that is the same every time for a given garden.", activeTasks), w)
			out = append(out, "", gh("Every suggestion"))
			for _, t := range taskList {
				out = append(out, gk(fmt.Sprintf("+%d", t.Reward), t.Text, w)...)
			}
			return out
		},
	}
}

func chKeys() chapter {
	return chapter{
		Title: "Keys",
		Lede:  "Every key, generated from the table the game runs on.",
		Build: func(m model, w int) []string {
			var out []string
			for _, set := range setOrder {
				rows := keyRows(w, set)
				if len(rows) == 0 {
					continue
				}
				if len(out) > 0 {
					out = append(out, "")
				}
				out = append(out, gh(setTitles[set]))
				out = append(out, rows...)
			}
			return out
		},
	}
}

func chCommandLine() chapter {
	return chapter{
		Title: "Command line and files",
		Lede:  "Flags, environment and where the garden lives.",
		Build: func(m model, w int) []string {
			var out []string
			out = append(out, gh("Flags"))
			for _, f := range cliDocs() {
				out = append(out, gk(f.Name, f.Desc, w)...)
			}
			out = append(out, "", gh("Environment"))
			for _, e := range envDocs() {
				out = append(out, gk(e.Name, e.Desc, w)...)
			}
			out = append(out, "")
			out = append(out, gp("The garden is saved to $XDG_DATA_HOME/garden/garden.json, or ~/.local/share/garden/garden.json. Writes are atomic, so an interrupted save cannot shred an existing garden. This session is saved to "+m.path+".", w)...)
			return out
		},
	}
}

// cliDoc is one documented flag or environment variable.
type cliDoc struct{ Name, Desc string }

func cliDocs() []cliDoc {
	return []cliDoc{
		{"--save <path>", "use this save file instead of the default"},
		{"--postcard", "print the garden as it stands, with no cursor or chrome, and exit"},
		{"--width <n>", "how wide to draw the postcard (default 80; the garden wants 79, narrower wraps the beds into blocks)"},
		{"--status", "print a one-line summary for a prompt or status bar, and exit"},
		{"--species", "list the whole catalogue and exit"},
		{"--color <mode>", "truecolor, 256, 16 or off; default is to detect the terminal"},
		{"--version", "print the version and exit"},
	}
}

func envDocs() []cliDoc {
	return []cliDoc{
		{"GARDEN_SAVE", "the save file (same as --save)"},
		{"GARDEN_COLOR", "colour mode (same as --color); NO_COLOR also turns colour off"},
		{"GARDEN_AUDIO", "set to off to disable music entirely"},
	}
}
