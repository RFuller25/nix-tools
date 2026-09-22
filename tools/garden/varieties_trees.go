package main

// Varieties of the trees and shrubs.

var treeVarieties = map[string][]Variety{
	"japanesemaple": {
		{Name: "Bloodgood", Note: "deep purple-red all summer, scarlet in autumn", Leaf: "125"},
		{Name: "Sango-kaku", Note: "coral-red bark, at its best in winter", Leaf: "185", Stem: "203"},
		{Name: "Dissectum", Note: "a weeping mound of thread-fine leaves", Leaf: "167",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ψ  "},
				{" ψ│ψ ", "  │  "},
				{" ψψψ ", " ╲│╱ "},
				{"╭ψψψ╮ ", "ψψψψψ ", "ζ ╲│╱ζ", "   │  "},
			}},
	},
	"silverbirch": {
		{Name: "the wild birch", Note: "white bark, black diamonds, fine twigs", Stem: "252"},
		{Name: "Youngii", Note: "a weeping dome rather than a spire", Stem: "251", Leaf: "107"},
		{Name: "Purpurea", Note: "purple leaves against the white bark", Leaf: "133"},
	},
	"ginkgo": {
		{Name: "the species", Note: "fan leaves, upright when young, vast when old", Leaf: "185"},
		{Name: "Autumn Gold", Note: "male, so no smelly fruit, and better colour", Leaf: "220"},
		{Name: "Mariken", Note: "a dwarf, a metre or so, good in a pot", Leaf: "149"},
	},
	"olive": {
		{Name: "Frantoio", Note: "the Tuscan oil olive, small fruit, heavy crops", Accent: "60"},
		{Name: "Kalamata", Note: "large almond-shaped table olives", Accent: "54"},
		{Name: "Arbequina", Note: "compact, early-cropping, good in a container", Accent: "65"},
	},
	"cherryblossom": {
		{Name: "Kanzan", Note: "double, deep pink, the avenue cherry", Bloom: "211"},
		{Name: "Shirotae", Note: "white, semi-double, scented, spreading wide", Bloom: "255"},
		{Name: "Shōgetsu", Note: "pale pink fading to white, hanging in long clusters", Bloom: "224"},
	},
	"blackpine": {
		{Name: "the species", Note: "craggy, salt-proof, the coastal pine", Leaf: "65"},
		{Name: "Thunderhead", Note: "dwarf, dense, with white candles in spring", Leaf: "29", Accent: "255"},
		{Name: "Kotobuki", Note: "slow and upright, a bonsai favourite", Leaf: "71"},
	},
	"apple": {
		{Name: "Bramley's Seedling", Note: "the cooking apple, cooks to a froth", Accent: "112"},
		{Name: "Cox's Orange Pippin", Note: "aromatic, difficult, worth it", Accent: "173"},
		{Name: "Discovery", Note: "early, crisp, red-flushed, does not keep", Accent: "196"},
		{Name: "Egremont Russet", Note: "rough golden skin, a nutty flavour", Accent: "179"},
	},
	"lemon": {
		{Name: "Eureka", Note: "nearly thornless, fruits almost all year", Accent: "226"},
		{Name: "Meyer", Note: "a lemon-mandarin cross: sweeter, rounder, hardier", Accent: "220"},
		{Name: "Variegata", Note: "striped leaves and striped green-and-yellow fruit", Leaf: "151", Accent: "228"},
	},
	// Hydrangea takes its colour from the soil rather than from its variety,
	// so these differ in the shape of the head instead.
	"hydrangea": {
		{Name: "Endless Summer", Note: "a solid mophead dome, flowering on new wood as well as old",
			Art: &[StageCount][]string{
				{}, {}, {},
				{" ◦◦◦◦ ", " ε╲│╱з", "   │  "},
				{" ❋❋❋❋ ", "❋❋❋❋❋❋", " ε╲│╱з", "   │  "},
			}},
		{Name: "Lacecap", Note: "a flat ring of showy bracts around the tiny true flowers",
			Art: &[StageCount][]string{
				{}, {}, {},
				{" ◦···◦ ", " ε╲│╱з ", "   │   "},
				{"❋❋···❋❋", " ε╲│╱з ", "   │   "},
			}},
		{Name: "Ayesha", Note: "florets cupped like a lilac's, unusual in the genus",
			Art: &[StageCount][]string{
				{}, {}, {},
				{" ◍◍ ◍◍", " ε╲│╱з", "   │  "},
				{" ◍◍ ◍◍", " ◍◍ ◍◍", " ε╲│╱з", "   │  "},
			}},
	},
	"magnolia": {
		{Name: "the hybrid", Note: "pink-flushed goblets on bare branches", Bloom: "218"},
		{Name: "Alba Superba", Note: "pure white, earlier, and scented", Bloom: "255"},
		{Name: "Rustica Rubra", Note: "deep rose-purple, larger flowers", Bloom: "132"},
	},
	"willow": {
		{Name: "the weeping willow", Note: "long trails to the water", Leaf: "107"},
		{Name: "Tortuosa", Note: "corkscrew twigs, prized for cutting", Stem: "137", Leaf: "113"},
		{Name: "Britzensis", Note: "cut it hard each spring for scarlet winter stems", Stem: "203"},
	},
}
