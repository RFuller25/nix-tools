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
		chPlanner(),
		chYear(),
		chGenetics(),
		chBreeding(),
		chCultivars(),
		chLineage(),
		chVisitors(),
		chGold(),
		chOrders(),
		chFair(),
		chGnomes(),
		chSharing(),
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
						gk("seed shed", "where seed is bought, kept and sown (s).", w),
						gk("orders", "what the village wants, for gold (O, or o on a bed).", w),
						gk("fair", "the weekly show (E, or e on a bed).", w),
						gk("gnome shop", "magic for gold: weather, seasons, growth, mutations, cross-species matches, soil and a wild seed (N).", w),
						gk("layouts", "saved arrangements to stamp down (T).", w),
						gk("almanac", "this book: the guide, every species and every visitor (a).", w),
						gk("journal", "the log, the tally and what is worth doing next.", w),
						gk("info card", "one plant in full (i, or enter on a planted bed).", w),
						gk("help", "the key list (?).", w),
					)...),
				gp("In this almanac every category (the guide, your cultivars, each kind of plant, the visitors) has a heading you can stand on: enter folds it away and opens it again, and C folds or opens them all. Folds are remembered in your garden's save.", w),
				gp("tab cycles garden → shed → orders → fair → almanac → journal. esc or q returns to the garden; q in the garden quits, and the garden saves itself.", w),
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
					gp("Each day's weather comes from the calendar and your garden's own seed, so it is the same every time the missing hours are replayed. Spring, summer, autumn and winter each draw from their own table: sun, clear, cloud, showers, rain, storms, fog, frost, snow and heatwaves. Rain waters the beds for you; heat dries them faster; a storm or a frost slows growth a little. The gnomes will turn the sky for the rest of a day, for gold (see The gnome shop).", w)...),
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
			out = append(out, "", gh("Rules from the genes"))
			out = append(out, gp("These work on how the actual plants have been bred, so the same species can help or hinder depending on its genes:", w)...)
			for _, e := range geneRuleDocs() {
				out = append(out, gk(e.Label, e.Text, w)...)
			}
			out = append(out, "")
			out = append(out, gp("Press g in the garden for the overlay: every bed is outlined green or red by the net of what its neighbours do for it and what it does for them (▲ and ▼ show the figure), so a good arrangement can be seen at a glance.", w)...)
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
				gp("The creatures are drawn over the beds and recorded in the journal. Pollination follows the same weather and hours they do (bees by day, moths at night, few in the rain) but is worked out in the background, so it is the same whether or not one happens to be on screen. See Breeding.", w),
			)
		},
	}
}

func chGold() chapter {
	return chapter{
		Title: "Gold and the seed shed",
		Lede:  "What money is for, and where seed is kept.",
		Build: func(m model, w int) []string {
			return join(
				gp("Gold buys seed, ground and ponds. Seed is kept in your own shed in packets, and the shed has two shelves: the shop, where named forms are sold for gold, and your own seeds, which holds everything you have bought but not sown and everything you have gathered. Nothing ever takes gold away from you: it can only be spent.", w),
				append([]string{gh("Where gold comes from")},
					join(
						gk(fmt.Sprintf("+%d", dailyStipend), "a stipend from the shed on your first visit each day", w),
						gk(fmt.Sprintf("+%d", weedingReward), "for clearing a properly overgrown bed (weeds over 45%)", w),
						gk("sell seed", fmt.Sprintf("seed from your own plants, on the my seeds shelf ($ sells one, S the packet). A plain seed fetches about %.0f%% of its shop price; one that is unlike any named form fetches up to %.0fx more, a seed from a stable line %.0f%% more, and a high yield gene up to %.0f%% more.", sellBase*100, 1+sellNovelty, sellStable*100, sellYield*100), w),
						gk("tasks", "the suggestions in the journal pay gold", w),
						gk("orders, fair", "see their chapters", w),
					)...),
				append([]string{gh("Where it goes")},
					join(
						gk("seed", "the shop price on every card: named forms start at a few gold, rarer species cost more", w),
						gk(fmt.Sprintf("%d+", bedBaseCost), fmt.Sprintf("a new bed: %d for the first extra, %d more for each after it, to %d beds", bedBaseCost, bedStepCost, maxPlots), w),
						gk(fmt.Sprintf("%d", pondCost), "a pond", w),
						gk("gnomes", "magic: weather, seasons, growth, mutations, matches, soil and a wild seed; see The gnome shop", w),
					)...),
				append([]string{gh("The shed")}, keyRows(w, setShop, setMine)...),
				append([]string{gh("Folders")}, join(
					gp("Press v on your own seeds to file the packets in folders instead of a plain list. The folders go category (herb, flower, edible...), then species, then variety, then the packets themselves. Only folders that hold seed are shown. A folder with just one thing in it is skipped: if the herb category holds a single packet, opening herb shows the packet itself, with no species or variety folders to open first. enter or space opens a folder, left and right close and open, and C closes or opens them all. The choice and the folders you left open are kept in your save.", w),
				)...),
				append([]string{gh("Buying cultivars")}, join(
					gp(fmt.Sprintf("The shop sells your cultivars too, whether you bred them or took them in from a friend's code: they follow a species' named forms when you flick through its varieties with the arrow keys. A cultivar's seed costs %d times the ordinary seed of its species, and half as much again for a stable line. It comes exactly true to the cultivar's ideal, like seed of any named form. The price is well above what the shed pays for the same seed, so buying and selling it is no way to make gold.", cultivarCost), w),
				)...),
				gp("In the shop, enter buys one seed and sows it at once and b only buys it. In your own seeds, enter sows one seed from the highlighted packet in the selected bed. A packet is a family: each seed's genes are decided when it is sown, so sowing several from one packet gives different plants. The card shows the parents, the likely spread of each gene, and a sample of the colours the family could be. Rarer species unlock in the shop as more of your plants reach maturity.", w),
				gp("If you played the first version of the garden, your old seeds became gold one for one, each plant standing in a bed became a packet of its own form in your shed (with one extra seed for every ripe pod), and the beds were cleared. The ground, ponds, records and herbarium were kept, and the old file was saved beside the new one with .v1.bak on the end.", w),
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
		{"--cultivars", "list the hybrid lines your garden has found, and exit"},
		{"--keys", "print the key tables as markdown, and exit"},
		{"--export <name|number>", "make a code for one of your cultivars to give a friend, and exit"},
		{"--import <code>", "redeem a friend's code: three seeds of their cultivar and a place in your almanac, then exit"},
		{"--gardener <name>", "the name your shared codes are signed with (saved)"},
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

func chGenetics() chapter {
	return chapter{
		Title: "Genetics",
		Lede:  "Seven genes: colour, height, shape, speed and yield.",
		Build: func(m model, w int) []string {
			return join(
				gp("Every plant carries genes. The named forms in the shed (a Russian Giant sunflower, a Black Krim tomato) each have their own set, and a plant grown from a seed bought there comes true to its form. Bred plants are blends of their parents and can end up anywhere between them, or a little past.", w),
				append([]string{gh("The genes")},
					join(
						gk("colour", "three genes — hue, saturation and lightness — so a bloom can be any shade your terminal can show. They paint the flowers on most plants, the fruit on tomatoes, chillies and strawberries, and the leaves on lettuce. The card names the shade and shows it.", w),
						gk("height", fmt.Sprintf("dwarf to giant, spread across the species' own height range. Giants grow up to %.0f%% slower and dwarfs up to %.0f%% quicker. In the garden a tall plant is drawn with a longer stem where the bed has room, and a dwarf with a shorter one; the card and the shed show the real size.", heightSlowdown*100, heightSlowdown*100), w),
						gk("shape", fmt.Sprintf("slim and upright to full and bushy. Full plants are drawn with a wider head and slim ones with a pinched one, and a full plant catches up to %.0f%% more wind than a slim one.", 50.0), w),
						gk("speed", fmt.Sprintf("slow to quick: ×%.2f to ×%.2f growth, and a quick annual finishes its year sooner.", speedLow, speedHigh), w),
						gk("yield", fmt.Sprintf("sparse to abundant: pods ripen ×%.1f to ×%.1f as fast and a plant holds %.0f to %.0f of them, but a heavy cropper dries its bed up to %.0f%% faster.", yieldPodLow, yieldPodHigh, podCapLow, podCapHigh, yieldThirst*100), w),
					)...),
				gp("Nothing is a pure upgrade. Height costs speed, yield costs water, and a bushy plant crowds its neighbours (see Neighbours). A plant whose genes all sit mid-range grows at exactly the rate its card advertises.", w),
				append([]string{gh("How genes are passed on")},
					gp(fmt.Sprintf("A seed's genes are the average of its two parents, plus a little noise. The noise is about %.0f points for identical parents, and grows by %.2f points for every point the parents differ: cross two unlike plants and the seedlings spread wide, so there is something to choose between; cross near-identical siblings and the seedlings come out nearly the same.", noiseBase, noiseFromGap), w)...),
				gp(fmt.Sprintf("Colour is blended on the colour wheel, not on paper: red and yellow make orange, red and blue make magenta, and a white crossed with a blue is a pale blue. About %.0f%% of seeds also mutate a gene by %.0f to %.0f points (hue jumps %.0f to %.0f degrees), which is where rare colours come from.", mutationOdds*100, mutationLow, mutationHigh, hueMutateLow, hueMutateHigh), w),
				gp("Every roll is a hash of your garden's seed, never a random number. A garden left running and one that catches up on a fortnight it spent closed grow exactly the same plants.", w),
				append([]string{gh("Named forms and hybrids")},
					gp(fmt.Sprintf("A plant keeps the drawings and accent colours of the named form it most resembles. Once its genes sit more than %.0f%% of the way from every named form it is a hybrid, and the card says so.", hybridGap*100), w)...),
				append([]string{gh("Colour on a 24-bit terminal")},
					gp("Colours are written as full RGB. If your terminal advertises true colour (COLORTERM=truecolor) you see them exactly; otherwise they are snapped to the nearest of 256 colours, and GARDEN_COLOR or --color can force a mode, for tmux or ssh sessions that under-report.", w)...),
			)
		},
	}
}

func chBreeding() chapter {
	return chapter{
		Title: "Breeding",
		Lede:  "Pollination, crossed seed and making new lines.",
		Build: func(m model, w int) []string {
			return join(
				gp("Every ripe pod holds seed with two parents: the plant it grew on, and whatever pollinated that plant. A flower that is pollinated passes the other parent's genes into every seed it sets until the seed is gathered; a flower that is not sets seed of itself.", w),
				append([]string{gh("By insects")},
					gp(fmt.Sprintf("An open flower with a neighbour of its own kind in flower beside it may take that neighbour's pollen. Each quarter hour a flower has a %.0f%% chance by day (bees and butterflies), %.0f%% after dark if it is one of the night-scented flowers (moths), %.0f%% as much in the rain and %.0f%% as much in winter, and more with several mates beside it. A flower shut for the night, a bud, a plant gone to seed or one asleep for the winter cannot be pollinated. A plant with no mate beside it sets seed of itself. The decision is hashed, so a garden left running and one catching up cross exactly the same plants. Plant a variety next to another one of its species to cross them.", beeOdds*100, mothOdds*100, wetFactor*100, winterOdds*100), w)...),
				append([]string{gh("By hand")},
					gp("Press x on a flower, then move to any other flower of the same species anywhere in the garden (the ones that will do are outlined in gold) and press enter. The next seed gathered from the first flower is certainly that cross. It needs both flowers open, and a dry sky: the rain washes the pollen out of the air. A flower carrying pollen shows a coloured ✿ on its bed, and its card says whose it is.", w)...),
				append([]string{gh("Gathering")},
					gp("f gathers ripe pods into a packet in your own seeds. Crossed seed is labelled with its generation. It uses up the flower's pollen. Seed from a plant of an established line stays in that line if nothing crossed it.", w)...),
				append([]string{gh("Hybrids and cultivars")},
					gp("When a plant of your own breeding comes into flower unlike every named form and every line you have found already, the garden writes it down: it goes into YOUR CULTIVARS in this almanac, and among its species' varieties, with its genes, its descent and the date, and the journal says so. A plant or seedling belongs to a line while it fits the line's definition (see Cultivars). Select the line you like and press n on its page to name it; the name carries to the plants and the seed.", w)...),
				append([]string{gh("Stable lines")},
					gp(fmt.Sprintf("A line breeds true when seed from a plant of it, crossed with nothing unlike it, grows into seedlings that come out within %.0f%% of their packet's centre. Do that %d generations running and the line is stable (◆ in the almanac): its seed is worth %.0f%% more in the shed. A wide cross, or a mutation, breaks the run and it starts again; a mutation is about one seed in four.", stableGap*100, stableRuns, sellStable*100), w)...),
				append([]string{gh("Making a line")},
					gp("Sow a packet, keep the seedlings you like best (the tallest, the bluest, the quickest), and breed from those. Because seed is a blend of two parents plus noise, and the noise shrinks as the parents come to resemble each other, each generation of selection moves the line a little further and steadies it. The card shows the spread to expect, and a sample of the colours the family could be.", w)...),
			)
		},
	}
}

func chPlanner() chapter {
	return chapter{
		Title: "Layout planner and layouts",
		Lede:  "Plan a garden before sowing it, and stamp layouts down.",
		Build: func(m model, w int) []string {
			var names []string
			for _, t := range builtinTemplates() {
				names = append(names, t.Name)
			}
			return join(
				gp("Because neighbours matter, it pays to see an arrangement before committing seed to it. Three tools help, and none of them costs anything until you sow.", w),
				append([]string{gh("The overlay (g)")},
					gp("Outlines every planted bed green or red by the net of what its neighbours do for it and what it does for them, and prints the figure under it (▲ for a gain in growth speed, ▼ for a loss). It counts the planting rules and the gene rules in Neighbours.", w)...),
				append([]string{gh("Plan mode (P)")}, join(
					gp("Lays ghosts of seed over the beds. The seed tray above the footer shows what you are placing, with its neighbours in the list: [ and ] move along it ({ and } ten at a time) and slash searches it by name, kind or family. Your own packets come first, then everything the shop sells, then species it does not stock yet, which you can plan with but which are left out when the plan is sown. enter places the seed in the bed under the cursor as a ghost, x takes it out, and every bed is scored as though the plan were already grown, so you can see what a marigold would do for a tomato before you sow either. The footer counts what the plan would cost: seed from your own shed is free, anything else is bought from the shop at its price when you press C to sow the lot. esc puts the plan away; nothing is sown or spent until C.", w),
					keyRows(w, setPlan),
				)...),
				append([]string{gh("Layouts (V, T)")}, join(
					gp("Press V, stretch a block of beds with the arrow keys over a planting you like, and press enter to name it: it is saved with what is growing there. T lists layouts, with a preview, what you already have the seed for and what the rest would cost. Choose one, move it over the garden (it is re-scored as it goes, and anything that does not fit is left out and listed) and press enter to sow it: your own seed first, shop seed for the rest. Some layouts ask for the tallest seed you have where it matters, such as the back of a border.", w),
					gp("Built in: "+joinWords(names)+". Layouts you save are kept in your garden; x deletes one of your own, never a built-in.", w),
					keyRows(w, setSelect, setStamp, setTpl),
				)...),
			)
		},
	}
}

func chOrders() chapter {
	return chapter{
		Title: "Orders",
		Lede:  "The village wants particular plants.",
		Build: func(m model, w int) []string {
			return join(
				gp(fmt.Sprintf("Once a week, on Monday, %d orders are posted on the order board (tab, or O in the garden), each from a different villager and for a different species. They stay up until the next board goes up. There is no penalty for letting one lapse and no timer to beat. An order names a species and one or two conditions: a height, a bushiness, a speed, a yield, or a colour to match.", ordersPerWeek), w),
				append([]string{gh("The three tiers")},
					join(
						gk("★ easy", fmt.Sprintf("something a named form from the shop already is. Pays %d gold plus %d times the seed's shop price.", orderEasyBase, orderEasyMult), w),
						gk("★★ a stretch", fmt.Sprintf("a gene %d to %d points past the best the shop sells, sometimes with a second easy condition, so it takes a few generations of breeding. Pays %d plus %d times the price.", stretchMin, stretchMax, orderMidBase, orderMidMult), w),
						gk("★★★ hard", fmt.Sprintf("a gene %d to %d points past the shelf, together with a colour the shelf does not have or a second gene past it. Pays %d plus %d times the price, and two seeds of another species thrown in.", hardMin, hardMax, orderHardBase, orderHardMult), w),
					)...),
				gp(fmt.Sprintf("Every order also pays %d gold for each point of difficulty: a gene point past the shelf, or a point of colour distance from the nearest named form. Two orders of the same tier can pay quite differently, and the hardest ones are worth the wait.", orderPointPay), w),
				gp(fmt.Sprintf("A colour order is filled by any shade within about %.0f%% of the one shown, judged on hue mostly, so a coral is fine when coral is asked for and not when scarlet is.", colourMatch*100), w),
				gp("To fill one the plant must be in flower and fit every condition. Press o on that bed, or enter on the order board (which delivers the first of your plants that fits). The plant is removed from its bed and used up by the order: it is not composted, so the bed gets no richness back. You are paid at once.", w),
				append([]string{gh("Keys")}, keyRows(w, setOrders)...),
			)
		},
	}
}

func chFair() chapter {
	return chapter{
		Title: "The fair",
		Lede:  "A show every week, judged on your plant's genes.",
		Build: func(m model, w int) []string {
			var classes []string
			for _, c := range fairCategories {
				classes = append(classes, c.Name+" ("+c.Blurb+")")
			}
			return join(
				gp("Every week (Monday to Sunday) the village holds a show with one class, chosen from the calendar and your garden's seed. Enter one plant that is in flower and suits the class with e on its bed, or from the fair screen (E, or tab). You may change your entry until the week is out. The plant stays in its bed.", w),
				gp(fmt.Sprintf("When the week is over, the judges mark your plant against %d other growers. The class decides what is marked (height for the Tallest, colour richness and novelty for Finest colour, and so on) and the plant's health at the time you entered counts for up to %.0f points of a hundred, so water and weed before you enter.", fairRivals, fairHealthPart), w),
				append([]string{gh("Prizes")}, join(
					gk("first", fmt.Sprintf("%d gold", prizeFirst), w), gk("second", fmt.Sprintf("%d gold", prizeSecond), w),
					gk("third", fmt.Sprintf("%d gold", prizeThird), w), gk("fourth", fmt.Sprintf("%d gold", prizeFourth), w),
					gk("after that", fmt.Sprintf("%d gold for taking part", prizeLast), w),
				)...),
				gp(fmt.Sprintf("The field starts at about %.0f points and rises by %.1f for every first place you have won, to a ceiling of +%.0f: you are always up against a rising bar and never another player. Each result is kept as a ribbon under RIBBONS in this almanac.", fairRivalMean, fairBarPerWin, fairBarMax), w),
				gp("Judging is hashed from the week, so a show missed while the program was closed is judged exactly as if you had been watching, whenever you next open the garden.", w),
				append([]string{gh("The classes")}, gp(joinWords(classes)+".", w)...),
				append([]string{gh("Keys")}, keyRows(w, setFair)...),
			)
		},
	}
}

func chGnomes() chapter {
	return chapter{
		Title: "The gnome shop",
		Lede:  "Magic for gold: sky, season, growth, mutations, matches, soil.",
		Build: func(m model, w int) []string {
			return join(
				gp("A family of gnomes keeps a stall at the bottom of the garden (N, or tab). They sell magic rather than seed, and every spell costs gold. Choose a service with enter, then whatever it needs: a sky, a season, a mutation or recipe, then a plant or bed. esc steps back. The gnomes themselves wander along the front of the stall, stop to chat and throw sparks, and hop for joy when you buy something; in a very short window they go home.", w),
				append([]string{gh("What they sell")}, join(
					gk(fmt.Sprintf("%d gold", gnomeWeatherCost), "change the weather. Pick any of the eleven skies and it holds for the rest of today; tomorrow goes back to the calendar's. Rain waters the beds, a heatwave dries them, a frost slows everything, so choose the one the garden wants. It only changes the hours still to come, never the ones already gone, and a sky you already have is refused without charge.", w),
					gk(fmt.Sprintf("%d gold", gnomeSeasonCost), "swap the season. Pick any season and it is that season for the rest of today: plants grow, rest and set seed by its rules (annuals outside their season slow down, perennials sleep through a borrowed winter) and the weather is drawn from its table, unless you have turned the sky as well. Tomorrow the calendar's season returns.", w),
					gk(fmt.Sprintf("%d gold", gnomeGrowthCost), "a growth spurt. Pick a plant and it climbs exactly one stage: seed to sprout, sprout to seedling, seedling to bud, bud to full bloom. A plant partway through a stage goes to the start of the next. Reaching bloom this way counts as blooming for the herbarium, the almanac and your tally of grown plants.", w),
					gk(fmt.Sprintf("%d gold", gnomeMutateCost), fmt.Sprintf("a random mutation. Pick a plant and one of its seven genes (colour counts as hue, saturation and lightness), sometimes two, moves by %.0f to %.0f points, or the hue by %.0f to %.0f degrees, up or down. The roll is a hash of your garden's seed and how many spells have been cast, so it is not something to re-roll by reloading.", mutationLow+5, mutationHigh+10, hueMutateLow, hueMutateHigh), w),
					gk(fmt.Sprintf("%d gold", gnomeChosenCost), fmt.Sprintf("a chosen mutation. Pick the change first (taller, shorter, bushier, slimmer, quicker, slower, more generous, sparer, or a colour: red, orange, yellow, green, teal, blue, violet, magenta, pink, white or midnight), then the plant, which shows what it will look like before you pay. Height, shape, speed and yield move by %d; a colour sets the hue outright. Nothing is charged for a change that would do nothing.", gnomeNudge), w),
					gk(fmt.Sprintf("%d to %d gold", gnomeMatchSame, gnomeMatchWide), fmt.Sprintf("the matchmaker. Pick a flower in bloom, then the flower whose pollen it should take, wherever they are and whatever the hour or weather: no brush, no waiting for dry air or an open bloom. The next seed gathered from the first (f) is that cross. Two of one species cost %d; two species of one botanical family (cousins) %d; two of different families %d, and set only half as much seed. A cross of two species gives seed of a hybrid species of its own (see Hybrid species below). A water plant cannot be matched with a land plant, and a hybrid can be matched only with its own kind. If the flower takes pollen from a bee or a brush before you gather, that replaces the match.", gnomeMatchSame, gnomeMatchFamily, gnomeMatchWide), w),
					gk(fmt.Sprintf("%d gold", gnomeSoilCost), fmt.Sprintf("soil alchemy. Pick a recipe, then a bed (planted or bare, but not a pond): more acid or more chalky moves pH by %.1f (between %.1f and %.1f), neutralise sets it to 6.5, enrich adds %.0f%% richness. The card shows how the bed's plant will like the ground before and after, and a hydrangea's flowers follow the pH from blue through mauve to pink.", gnomeSoilPH, phMin, phMax, gnomeSoilRich*100), w),
					gk("wild seed", fmt.Sprintf("one seed a day from the teapot: some species, any but a water plant, with every gene thrown to an extreme and a vivid colour. It costs %d gold plus %d times the species' shop price, there is one a day for everyone, and it comes exactly true when sown. It goes in your shed as a packet labelled Wild, and when it flowers it is added to the almanac as a new hybrid.", gnomeSeedBase, gnomeSeedMult), w),
				)...),
				append([]string{gh("Hybrid species")}, join(
					gp("Cross two species at the matchmaker and the seed is a species that has never existed before. It is named by running the parents' names together (Sunflower and Foxglove make Sunflove) and written with both parents' Latin names. Its drawing is made from the two: the upper half of the mother's, the lower half of the father's, with the two palettes mixed. Its genes start as the average of the parents' named forms, so it breeds, varies and is bred on like any species.", w),
					gp("A hybrid lives by its mother's rules: when it sleeps, whether it seeds itself, what soil it likes, whether its flowers shut at night. Sunflove is a sunflower in habit with a foxglove's looks; the reverse cross, from a foxglove mother, is a different hybrid. Its price is both parents' prices added together, so its seed sells for more.", w),
					gp("Hybrids are not in the seed shop or the almanac's species list, and the herbarium does not count them. They live in your seed shed, your beds and, once you have bred something new from one, under YOUR CULTIVARS. The gnomes will cross a hybrid only with another of its own kind, never with a parent or a third species. They are saved by their parents' names, so a garden shared between machines grows the same hybrids.", w),
				)...),
				gp("A mutated plant is no longer part of the line it was bred in: its streak of true-breeding generations starts again, and if it is already in flower and unlike any named form it is entered in the almanac at once. The seed it sets carries the new genes, so a mutation is a way to start a new line as well as to rescue a stalled one.", w),
				append([]string{gh("Keys")}, keyRows(w, setGnomes)...),
			)
		},
	}
}

func chCultivars() chapter {
	return chapter{
		Title: "Cultivars",
		Lede:  "Varieties of your own: defining them, buying them, deleting them.",
		Build: func(m model, w int) []string {
			t := defaultTolerance
			return join(
				gp("A cultivar is a variety of your own: a hybrid line the garden found when it flowered, a seed you named, or a friend's line taken in from a code. Cultivars are listed under YOUR CULTIVARS and also with their species' named forms (press v on a species page to flick through them, and o to open one), with the same ticks, drawings and genes.", w),
				append([]string{gh("Defining one")}, join(
					gp("Seed wanders a little every generation, so a variety with nothing to hold it drifts into whatever the last plant happened to be. A cultivar therefore has a definition: an ideal value for each of the seven genes, and how far either side of it a plant may stray and still count as the variety. It is shown on the cultivar's page (and on its species page and its card in the shop) as a bar per gene with the allowed band filled in and the ideal marked ◆.", w),
					gp(fmt.Sprintf("A new cultivar starts with hue ±%d°, saturation ±%d, lightness ±%d and height, shape, speed and yield each ±%d. Press D on the cultivar's page to change it: up and down choose a gene, left and right narrow or widen its band (by %d points, or %d degrees of hue), [ and ] move the ideal itself, and r puts the bands back to the defaults. A hue means nothing in a colourless plant, so white, grey and black ignore it.", t.Hue, t.Sat, t.Light, t.Height, tolStep(1), tolStep(0)), w),
				)...),
				append([]string{gh("Off-types")}, join(
					gp("The definition is the one test of membership. A seed sown from a packet of the line, a seedling that sows itself, a plant that mutates or one that flowers outside the bands is an off-type: it grows as an ordinary plant, the journal says which gene took it out and by how much, and it is filed as a new hybrid if it is unlike every named form and every line. Seed gathered from a line's plants stays in the line only if it would still fit. Narrowing a definition releases the plants and seed that no longer fit it; widening one takes in flowering hybrids that now do.", w),
					gp("A packet's card says what share of its seedlings would fit a cultivar, from a fixed sample, so you can tell whether a cross will hold the variety before you sow it.", w),
				)...),
				append([]string{gh("Buying")}, join(
					gp(fmt.Sprintf("Any cultivar, bred by you or imported, can be bought back in the shop at %d times the ordinary seed price of its species, and half as much again if it is stable. See Gold and the seed shed.", cultivarCost), w),
				)...),
				append([]string{gh("Deleting")}, join(
					gp("Press x on a cultivar's page, and again to confirm, to delete its record from the almanac, bred or imported. The plants growing and the seed in your shed stay, as ordinary stock no longer attached to a line, and a plant that flowers like the deleted line does not bring it back. A friend's code that made an imported line stays used, so it cannot be redeemed again.", w),
				)...),
				append([]string{gh("Keys")}, keyRows(w, setDefine)...),
			)
		},
	}
}

func chLineage() chapter {
	return chapter{
		Title: "The genetics screen",
		Lede:  "How a line came about: sources, mutations and drift.",
		Build: func(m model, w int) []string {
			return join(
				gp("Press t on a plant's card, on a packet in your own seeds or on a cultivar's page in this almanac to read its genetics: not only what the genes are, but how the line got there. Every plant, packet and cultivar carries a record, one step a generation, made at the moment each seed was sown from the same roll that decided its genes.", w),
				append([]string{gh("What it shows")}, join(
					gk("short version", "how many generations deep the record goes, where the first generation came from, how many mutations and drifts were noted, and how far the line has wandered from the first generation (and furthest it ever got).", w),
					gk("first generation", fmt.Sprintf("the source of the line: a named form bought from the shop, a cultivar bought from the shop, a wild seed from the gnomes, or a friend's gift. Records are kept to %d steps; if a line runs longer, the first generation is always kept.", lineageMax), w),
					gk("the timeline", "each generation oldest first: how it came about (selfed, crossed, sown by itself, changed by the gnomes), the mother and father and the seedling between them, and the arithmetic of the cross: how alike the parents were and so how widely the seedlings could scatter.", w),
					gk("mutations", fmt.Sprintf("a gene that jumped (⚡), against the ordinary spread; about %.0f%% of seeds mutate a gene.", mutationOdds*float64(geneCount)*100), w),
					gk("drift", "a gene that landed well off its parents' middle without mutating, with how many times the usual spread that was.", w),
					gk("gene by generation", "a line of marks for each gene across the generations, and the colour as a row of swatches.", w),
					gk("against the definition", "for a cultivar's line, whether each generation fit the definition, and the first to stray.", w),
				)...),
				gp("A plant or packet from before lineages were recorded says so plainly and starts recording from there; nothing in the history is guessed.", w),
				append([]string{gh("Keys")}, keyRows(w, setLineage)...),
			)
		},
	}
}

func chSharing() chapter {
	return chapter{
		Title: "Sharing cultivars",
		Lede:  "Give a friend a line you have bred, as a short code.",
		Build: func(m model, w int) []string {
			return join(
				gp("A cultivar you have bred can be given to a friend as text. Open its page under YOUR CULTIVARS in this almanac and press e: you get a code beginning GD1- (it is also sent to your clipboard if your terminal allows). Paste it into a message. From the command line, garden --export \"name\" prints the same.", w),
				gp(fmt.Sprintf("Your friend presses i in the seed shed and pastes it (or runs garden --import <code>). They receive %d seeds of the line, which sell at the ordinary price like any seed of that quality, and the line is added to their almanac under YOUR CULTIVARS with the name you gave it and who it came from. If they already had that line, the seed joins it.", giftSeeds), w),
				append([]string{gh("What a code holds")},
					gp("The species, all seven genes, the name you gave the line, its generation, whether it is a stable line, and the name you signed it with (garden --gardener \"name\" sets that; it is saved). Nothing else about your garden is in it.", w)...),
				append([]string{gh("The rules")}, join(
					gk("once", "A code can be redeemed once in any one garden, and never in the garden that made it. There is no server to say it has been used elsewhere, so a friend can pass it on; but every time you share you make a new code, and each is its own.", w),
					gk("checked", "A code is checked when it is read. A mistyped, cut-off or edited code is refused with the reason, and genes outside the possible range are refused, so a bad code can never make a broken plant.", w),
					gk("new line", "Seedlings of a gift vary a little, like any bred line, and keep its name.", w),
				)...),
				append([]string{gh("Keys")}, keyRows(w, setShare)...),
			)
		},
	}
}
