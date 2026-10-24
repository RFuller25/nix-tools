package main

// Varieties of the pond margins, the ornamental grasses and the brassicas.

var pondVarieties = map[string][]Variety{
	"watercress": {
		{Name: "Green", Note: "the ordinary peppery cress, deep green and quick", Leaf: "71"},
		{Name: "Brown", Note: "bronze-leaved, hardier, with a hotter flavour", Leaf: "136", Stem: "95"},
		{Name: "Vitamin Green", Note: "a smooth, dark, vigorous leaf with a mild bite", Leaf: "34"},
	},
	"yellowflag": {
		{Name: "the wild plant", Note: "clear yellow flags with brown pencilling", Bloom: "220"},
		{Name: "Variegata", Note: "leaves striped cream in spring, greening by summer", Bloom: "220", Leaf: "150"},
		{Name: "Bastardii", Note: "primrose-pale flowers on the same tall stems", Bloom: "229"},
	},
	"marshmarigold": {
		{Name: "the wild plant", Note: "single golden cups on fleshy stems", Bloom: "220"},
		{Name: "Flore Pleno", Note: "double, packed with petal-like sepals, long-lasting", Bloom: "214"},
		{Name: "Alba", Note: "small, single and white, with a yellow eye", Bloom: "255"},
	},
	"pickerelweed": {
		{Name: "the species", Note: "violet-blue spikes over glossy heart-shaped leaves", Bloom: "105"},
		{Name: "Alba", Note: "white spikes, otherwise the same", Bloom: "255"},
		{Name: "Pink Pons", Note: "soft pink spikes, a rarity among marginals", Bloom: "218"},
	},
	"papyrus": {
		{Name: "the species", Note: "tall triangular stems and a great green mop", Stem: "107"},
		{Name: "Nanus", Note: "a dwarf form, a metre or so, for a small pond", Stem: "114",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ¦  "},
				{" ¦│¦ "},
				{"╲╲│╱╱", " ╲│╱ "},
				{"╲╲│╱╱", " ╲│╱ ", "  │  ", "≈≈│≈≈"},
			}},
		{Name: "King Tut", Note: "a compact, sturdy form with a dense head", Stem: "150", Leaf: "77"},
	},
	"giantwaterlily": {
		{Name: "the species", Note: "white on the first night, pink on the second", Bloom: "255"},
		{Name: "Longwood Hybrid", Note: "a hardier hybrid raised at Longwood Gardens, rose-pink", Bloom: "218", Leaf: "65"},
	},
}

var grassVarieties = map[string][]Variety{
	"bluefescue": {
		{Name: "Elijah Blue", Note: "silvery-blue and the neatest hummock", Leaf: "110"},
		{Name: "Golden Toupee", Note: "a low mound of golden-yellow leaves", Leaf: "185"},
		{Name: "Blaufuchs", Note: "bright steel blue, compact and slow to open at the centre", Leaf: "67"},
	},
	"featherreed": {
		{Name: "Karl Foerster", Note: "upright, bronze plumes fading to straw", Bloom: "180"},
		{Name: "Overdam", Note: "a shorter form with white-striped leaves flushing pink", Leaf: "151", Bloom: "224"},
		{Name: "Avalanche", Note: "a broad white central stripe on every leaf", Leaf: "157"},
		{Name: "Eldorado", Note: "yellow-striped leaves, upright and tall", Leaf: "185", Bloom: "223"},
	},
	"hakone": {
		{Name: "Aureola", Note: "golden leaves striped with green, flushing pink in autumn", Leaf: "185"},
		{Name: "All Gold", Note: "solid sunny gold, brightest in a little shade", Leaf: "227"},
		{Name: "Albostriata", Note: "cream-striped leaves, the older and plainer form", Leaf: "151"},
		{Name: "Nicolas", Note: "green leaves turning orange-red in autumn", Leaf: "130"},
	},
	"switchgrass": {
		{Name: "Shenandoah", Note: "red tips from midsummer, a wine-red autumn", Leaf: "132", Bloom: "175"},
		{Name: "Northwind", Note: "stiffly upright, steel-blue-green, never flops", Leaf: "108"},
		{Name: "Heavy Metal", Note: "blue leaves and a narrow, metallic outline", Leaf: "67"},
	},
	"silvergrass": {
		{Name: "Gracillimus", Note: "narrow, fine-leaved, the classic arching fountain", Leaf: "108"},
		{Name: "Zebrinus", Note: "leaves banded across in gold, like a zebra", Leaf: "185", Stem: "143"},
		{Name: "Morning Light", Note: "white-edged leaves that glow, with copper plumes", Leaf: "151", Bloom: "173"},
		{Name: "Silberfeder", Note: "tall, with silver-pink feathery plumes in good time", Leaf: "107", Bloom: "252"},
	},
	"pampas": {
		{Name: "Sunningdale Silver", Note: "the tallest, with long white plumes", Bloom: "255"},
		{Name: "Pumila", Note: "a dwarf pampas that still flowers freely", Bloom: "223",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ¦  "},
				{" ╲│╱ "},
				{" ╲│╱ ", "╲╲│╱╱"},
				{" ''' ", "'''''", " ╲│╱ ", "╲╲│╱╱"},
			}},
		{Name: "Rosea", Note: "pink-tinted plumes, a fashion of the Victorian lawn", Bloom: "218"},
		{Name: "Gold Band", Note: "leaves edged in gold, plumes silvery", Leaf: "185", Bloom: "254"},
	},
}

var brassicaVarieties = map[string][]Variety{
	"cabbage": {
		{Name: "Hispi", Note: "pointed, sweet and early, tender enough to eat raw", Accent: "114"},
		{Name: "January King", Note: "hardy, flushed purple on the outer leaves, for winter", Accent: "139", Leaf: "97"},
		{Name: "Red Drumhead", Note: "dense red-purple heads that keep well", Accent: "125", Leaf: "96"},
		{Name: "Savoy", Note: "crinkled blistered leaves, the hardy winter cabbage", Accent: "65", Leaf: "65"},
	},
	"kale": {
		{Name: "Cavolo Nero", Note: "dark blue-green strap leaves, the Tuscan kale", Leaf: "23"},
		{Name: "Redbor", Note: "ruffled, deep red-purple, dramatic in a winter bed", Leaf: "89"},
		{Name: "Red Russian", Note: "flat grey-green leaves with purple veins, tender", Leaf: "108", Stem: "97"},
		{Name: "Dwarf Green Curled", Note: "a short, tight, frilled kale for a small bed", Leaf: "71",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ϑ  "},
				{" εзε "},
				{" ζ~ξ ", "  │  "},
				{" ζ~ξ ", " ξ│ζ ", "  │  "},
			}},
	},
	"broccoli": {
		{Name: "Calabrese", Note: "the classic green head, followed by side shoots", Accent: "65"},
		{Name: "Purple Sprouting", Note: "many purple spears through late winter into spring", Accent: "97"},
		{Name: "Sicilian Purple", Note: "an old heirloom head, purple-green and sweet", Accent: "133"},
	},
	"brusselssprout": {
		{Name: "Long Island Improved", Note: "the old reliable green sprout, heavy cropper", Accent: "77"},
		{Name: "Red Bull", Note: "dark red sprouts that keep a flush of colour", Accent: "89"},
		{Name: "Doric", Note: "firm mild green buttons, a generous crop", Accent: "114"},
	},
	"kohlrabi": {
		{Name: "Superschmelz", Note: "a giant pale-green kohlrabi that stays tender", Accent: "114"},
		{Name: "Purple Vienna", Note: "a purple-skinned bulb, crisp and white inside", Accent: "97"},
		{Name: "White Vienna", Note: "early, small and pale, the old standard", Accent: "151"},
	},
	"pakchoi": {
		{Name: "Joi Choi", Note: "white spoon stems and dark leaves, bolt-resistant", Stem: "255", Leaf: "71"},
		{Name: "Rubi", Note: "red-purple leaves with pale stems", Leaf: "89", Stem: "175"},
		{Name: "Shanghai Green", Note: "pale green stems, mild and quick", Stem: "150", Leaf: "77"},
	},
	"turnip": {
		{Name: "Purple Top Milan", Note: "flat, purple-topped and early, the market turnip", Accent: "133"},
		{Name: "Golden Ball", Note: "yellow flesh and a sweet nutty flavour", Accent: "220"},
		{Name: "Tokyo Cross", Note: "a small, white and very quick salad turnip", Accent: "255"},
	},
	"rocket": {
		{Name: "Salad Rocket", Note: "broad lobed leaves, mild when young", Leaf: "77"},
		{Name: "Wild Rocket", Note: "narrow serrated leaves with a stronger flavour", Leaf: "71"},
		{Name: "Olive Leaf", Note: "smooth rounded leaves, slow to bolt", Leaf: "114"},
	},
	"cauliflower": {
		{Name: "All the Year Round", Note: "a dependable white curd, sown in any season", Accent: "255"},
		{Name: "Romanesco", Note: "lime-green curd in spiralling fractal cones", Accent: "114",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ϑ  "},
				{" εзε "},
				{" ε▲з ", " ╲│╱ "},
				{" ε▲▲з ", "ε▲▲▲▲з", " ╲││╱ "},
			}},
		{Name: "Graffiti", Note: "a vivid purple curd that fades to green when cooked", Accent: "97"},
		{Name: "Cheddar", Note: "an orange curd, rich in carotene", Accent: "208"},
	},
	"horseradish": {
		{Name: "Maliner Kren", Note: "the standard hot-rooted European horseradish", Leaf: "71"},
		{Name: "Bohemian", Note: "thicker roots with a sharper bite, wide leaves", Leaf: "65"},
		{Name: "Variegata", Note: "leaves splashed with cream, mildly less hot", Leaf: "151"},
	},
}
