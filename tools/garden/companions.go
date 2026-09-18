package main

// Companion planting. What you put in the bed next door matters, and the
// relationships here are the real ones gardeners have used for centuries:
// marigolds guarding tomatoes, alliums guarding roses, beans feeding whatever
// grows beside them, and mint crowding out the lot.

// companion is one neighbourly effect: when `from` grows beside `who`, `who`
// grows faster or slower by `delta`.
type companion struct {
	who   func(sp *Species) bool
	from  func(sp *Species) bool
	delta float64
	note  string
}

func isID(ids ...string) func(*Species) bool {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(sp *Species) bool { return set[sp.ID] }
}

func isFamily(family string) func(*Species) bool {
	return func(sp *Species) bool { return sp.Family == family }
}

func anyPlant() func(*Species) bool { return func(*Species) bool { return true } }

var companions = []companion{
	{
		who: isFamily("Solanaceae"), from: isID("marigold"), delta: 0.12,
		note: "French marigold roots discourage the nematodes that trouble it",
	},
	{
		who: isID("tomato"), from: isID("basil"), delta: 0.08,
		note: "basil is the tomato's oldest companion in the bed and the kitchen",
	},
	{
		who: anyPlant(), from: isFamily("Fabaceae"), delta: 0.08,
		note: "nitrogen fixed at the legume's roots feeds the ground around it",
	},
	{
		who: anyPlant(), from: isID("nasturtium"), delta: 0.06,
		note: "nasturtium takes the blackfly on itself and spares its neighbours",
	},
	{
		who: isID("rose"), from: isID("chives", "garlic", "allium"), delta: 0.10,
		note: "an onion relative at its feet keeps the aphids off a rose",
	},
	{
		who: isID("carrot"), from: isID("chives"), delta: 0.08,
		note: "the smell of chives muddles the carrot fly",
	},
	{
		who: isID("pea", "broadbean"), from: isID("sweetcorn"), delta: 0.06,
		note: "the corn gives the beans something to climb — two of the Three Sisters",
	},
	{
		who: anyPlant(), from: isID("pumpkin"), delta: 0.05,
		note: "squash leaves shade the soil and hold the moisture in",
	},
	{
		who: anyPlant(), from: isID("lavender", "rosemary", "thyme", "sage", "oregano"), delta: 0.05,
		note: "the aromatic herb draws pollinators in and confuses what would eat it",
	},
	{
		who: anyPlant(), from: isID("mint"), delta: -0.10,
		note: "mint's runners crowd out whatever is next to it",
	},
	{
		who: isFamily("Fabaceae"), from: isID("sunflower"), delta: -0.06,
		note: "sunflowers are allelopathic, and beans sulk beside them",
	},
	{
		who: anyPlant(), from: isID("silverbirch", "willow"), delta: -0.05,
		note: "the tree's shallow roots take the water first",
	},
}

// companionEffect is one neighbour's influence on a bed, ready to show.
type companionEffect struct {
	Bed   int
	Other *Species
	Delta float64
	Note  string
}

// companionEffects gathers what the beds around this one are doing to it.
func (g *Garden) companionEffects(idx int, sp *Species) []companionEffect {
	var out []companionEffect
	for _, n := range g.Neighbours(idx) {
		other := g.Plots[n].Species()
		if other == nil || other.ID == sp.ID {
			continue
		}
		for _, c := range companions {
			if c.who(sp) && c.from(other) {
				out = append(out, companionEffect{Bed: n, Other: other, Delta: c.delta, Note: c.note})
			}
		}
	}
	return out
}

// companionFactor is the growth multiplier from the neighbours. It is capped
// either way: a well-companioned bed grows noticeably better, never twice as
// fast, and a bad neighbour is a nuisance rather than a disaster.
func (g *Garden) companionFactor(idx int, sp *Species) float64 {
	total := 0.0
	for _, e := range g.companionEffects(idx, sp) {
		total += e.Delta
	}
	if total > 0.3 {
		total = 0.3
	}
	if total < -0.2 {
		total = -0.2
	}
	return 1 + total
}

// companionAside is the aside shown when something is sown next to a plant it
// has an opinion about: the strongest effect, good or bad.
func companionAside(g *Garden, idx int, sp *Species) string {
	effects := g.companionEffects(idx, sp)
	if len(effects) == 0 {
		return ""
	}
	best := effects[0]
	for _, e := range effects[1:] {
		if abs(e.Delta) > abs(best.Delta) {
			best = e
		}
	}
	if best.Delta < 0 {
		return " Mind the " + best.Other.Common + " beside it: " + best.Note + "."
	}
	return " Good company: " + best.Note + "."
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
