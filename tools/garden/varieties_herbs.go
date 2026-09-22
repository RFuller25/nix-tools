package main

// Varieties of the herbs.

var herbVarieties = map[string][]Variety{
	"basil": {
		{Name: "Genovese", Note: "large sweet leaves, the pesto basil", Leaf: "77"},
		{Name: "Dark Opal", Note: "deep purple leaves, milder flavour", Leaf: "97", Stem: "89"},
		{Name: "Thai", Note: "narrow leaves, purple stems, anise scented", Leaf: "71", Stem: "133"},
		{Name: "Greek", Note: "a tight globe of tiny leaves", Leaf: "113",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ϑ  "},
				{" εзε "},
				{" εзε ", " ╲│╱ "},
				{" εзεз ", " εзεз ", " ╲╲│╱ "},
			}},
	},
	"rosemary": {
		{Name: "Miss Jessopp's Upright", Note: "stiffly upright, good for hedging", Bloom: "111"},
		{Name: "Prostratus", Note: "creeping, spilling over a wall rather than standing",
			Bloom: "110",
			Art: &[StageCount][]string{
				{},
				{},
				{" ¦¦¦ "},
				{" ¦¦¦ ", " ╲╱╲ "},
				{"❋¦❋¦❋", "¦¦¦¦¦", "╲╱╲╱╲"},
			}},
		{Name: "Majorca Pink", Note: "pink flowers instead of blue", Bloom: "182"},
	},
	"thyme": {
		{Name: "the common thyme", Note: "grey-green, the strongest for cooking", Bloom: "176"},
		{Name: "Silver Posie", Note: "leaves edged in cream, pale pink flowers", Leaf: "151", Bloom: "218"},
		{Name: "Lemon", Note: "lemon-scented, brighter green", Leaf: "113", Bloom: "182"},
	},
	"mint": {
		{Name: "the black peppermint", Note: "dark stems, the sharpest menthol", Stem: "89"},
		{Name: "Chocolate", Note: "bronze-tinted, and it really does smell of it", Leaf: "101", Stem: "95"},
		{Name: "Variegata", Note: "leaves splashed cream, milder", Leaf: "151"},
	},
	"sage": {
		{Name: "the common sage", Note: "grey-green, the kitchen plant", Leaf: "108"},
		{Name: "Purpurascens", Note: "purple young leaves, ageing grey", Leaf: "139"},
		{Name: "Tricolor", Note: "grey, cream and pink, tender in a hard winter", Leaf: "182"},
		{Name: "Icterina", Note: "gold-variegated, milder and less hardy", Leaf: "185"},
	},
	"lavender": {
		{Name: "Hidcote", Note: "compact, the darkest violet spikes", Bloom: "62"},
		{Name: "Munstead", Note: "earlier, softer blue, better in cold", Bloom: "110"},
		{Name: "Alba", Note: "white flowers over the same grey leaves", Bloom: "255"},
		{Name: "Rosea", Note: "pale pink spikes, unusual and slower", Bloom: "218"},
	},
	"chamomile": {
		{Name: "the wild plant", Note: "single daisies, the tea chamomile", Bloom: "255"},
		{Name: "Bodegold", Note: "larger heads, more oil, easier to pick", Bloom: "254", Accent: "227"},
	},
	"chives": {
		{Name: "the common chive", Note: "mauve pompoms, fine hollow leaves", Bloom: "176"},
		{Name: "Album", Note: "white flowers, otherwise the same", Bloom: "255"},
		{Name: "garlic chives", Note: "flat leaves, white stars, a garlic edge", Bloom: "254", Leaf: "108"},
	},
	"parsley": {
		{Name: "Moss Curled", Note: "tightly curled, the garnish parsley", Leaf: "77"},
		{Name: "Italian flat-leaf", Note: "flat leaves, stronger flavour", Leaf: "71"},
		{Name: "Hamburg", Note: "grown for its parsnip-like root as much as its leaf", Leaf: "108"},
	},
	"dill": {
		{Name: "Bouquet", Note: "big seed heads, the pickling dill", Bloom: "227"},
		{Name: "Fernleaf", Note: "dwarf, slow to bolt, good in a pot", Leaf: "79"},
		{Name: "Mammoth", Note: "tall and fast, for seed rather than leaf", Bloom: "220"},
	},
	"oregano": {
		{Name: "Greek oregano", Note: "white flowers, the pungent culinary form", Bloom: "255"},
		{Name: "Aureum", Note: "gold leaves, milder, scorches in full sun", Leaf: "185"},
		{Name: "Compactum", Note: "a low mat, good at a path edge", Leaf: "71"},
	},
	"lemonbalm": {
		{Name: "the common plant", Note: "plain green, seeds itself everywhere", Leaf: "113"},
		{Name: "Aurea", Note: "leaves splashed gold in spring", Leaf: "185"},
		{Name: "All Gold", Note: "entirely gold-leaved, best in light shade", Leaf: "227"},
	},
}
