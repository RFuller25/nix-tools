package main

// Varieties of the bulbs and the climbers.

var bulbVarieties = map[string][]Variety{
	"tulip": {
		{Name: "Queen of Night", Note: "so deep a maroon it reads as black", Bloom: "52"},
		{Name: "Apeldoorn", Note: "the big scarlet Darwin hybrid, reliable for years", Bloom: "196"},
		{Name: "Spring Green", Note: "ivory feathered with green up each petal", Bloom: "194", Accent: "149"},
		{Name: "Ballerina", Note: "tangerine, lily-flowered, and scented", Bloom: "208"},
	},
	"daffodil": {
		{Name: "the wild Lent lily", Note: "pale ruff, deeper trumpet, the woodland plant", Bloom: "228", Accent: "220"},
		{Name: "King Alfred", Note: "all gold, the trumpet daffodil everyone pictures", Bloom: "220"},
		{Name: "Thalia", Note: "white, two or three nodding flowers to a stem", Bloom: "255"},
	},
	"crocus": {
		{Name: "Pickwick", Note: "white, feathered with lilac stripes", Bloom: "189", Accent: "141"},
		{Name: "Remembrance", Note: "deep violet with a silver sheen", Bloom: "99"},
		{Name: "Jeanne d'Arc", Note: "pure white with a gold throat", Bloom: "255", Accent: "220"},
	},
	"snowdrop": {
		{Name: "the common snowdrop", Note: "one green mark on each inner petal", Bloom: "255", Accent: "150"},
		{Name: "Flore Pleno", Note: "double, a ruffle of green-tipped petticoats", Bloom: "254"},
		{Name: "Viridapice", Note: "green tips to the outer petals too", Bloom: "255", Accent: "114"},
	},
	"iris": {
		{Name: "Jane Phillips", Note: "pale sky blue, scented, an old favourite", Bloom: "111"},
		{Name: "Superstition", Note: "near-black violet with a black beard", Bloom: "54", Accent: "233"},
		{Name: "Immortality", Note: "white, and it flowers a second time in autumn", Bloom: "255"},
	},
	"lilyvalley": {
		{Name: "the species", Note: "white bells, and the scent that carries", Bloom: "255"},
		{Name: "Rosea", Note: "soft pink bells, slower to spread", Bloom: "218"},
		{Name: "Albostriata", Note: "leaves striped lengthways in cream", Bloom: "255", Leaf: "150"},
	},
	"hyacinth": {
		{Name: "Delft Blue", Note: "soft porcelain blue, the classic forcing bulb", Bloom: "110"},
		{Name: "City of Haarlem", Note: "primrose yellow, later than the rest", Bloom: "228"},
		{Name: "Woodstock", Note: "deep beetroot purple", Bloom: "89"},
	},
	"allium": {
		{Name: "Globemaster", Note: "the largest heads of all, held for weeks", Bloom: "134"},
		{Name: "Mount Everest", Note: "white spheres on tall bare stems", Bloom: "255"},
		{Name: "Purple Sensation", Note: "smaller, darker, earlier", Bloom: "97"},
	},
	"tigerlily": {
		{Name: "the species", Note: "orange, heavily spotted, petals rolled right back", Bloom: "208", Accent: "88"},
		{Name: "Flore Pleno", Note: "double, a tangle of spotted petals", Bloom: "202", Accent: "52"},
		{Name: "Splendens", Note: "larger, redder, later to flower", Bloom: "160", Accent: "52"},
	},
	"morningglory": {
		{Name: "Heavenly Blue", Note: "sky blue with a white throat, the famous one", Bloom: "69", Accent: "255"},
		{Name: "Grandpa Ott's", Note: "deep purple with a red star in the throat", Bloom: "91", Accent: "160"},
		{Name: "Flying Saucers", Note: "white, streaked irregularly with blue", Bloom: "254", Accent: "75"},
	},
	"sweetpea": {
		{Name: "Cupani", Note: "the original Sicilian plant: small, bicoloured, fiercely scented", Bloom: "91", Accent: "161"},
		{Name: "Painted Lady", Note: "rose over white, grown since the 1730s", Bloom: "211", Accent: "255"},
		{Name: "Spencer Mixed", Note: "the big frilled Edwardian exhibition sort", Bloom: "176"},
	},
	"wisteria": {
		{Name: "Prolific", Note: "lilac-blue, and it flowers young rather than in ten years", Bloom: "140"},
		{Name: "Alba", Note: "long white trails against dark leaves", Bloom: "255"},
		{Name: "Rosea", Note: "pale pink, tipped deeper at the keel", Bloom: "218"},
	},
	"clematis": {
		{Name: "Étoile Violette", Note: "velvet purple, hundreds of small flowers", Bloom: "91"},
		{Name: "Alba Luxurians", Note: "white with odd green petal tips", Bloom: "255", Accent: "149"},
		{Name: "Madame Julia Correvon", Note: "wine red, petals twisting as they open", Bloom: "124"},
	},
	"nasturtium": {
		{Name: "Empress of India", Note: "crimson flowers over dark blue-green leaves", Bloom: "160", Leaf: "65"},
		{Name: "Alaska", Note: "leaves marbled white, flowers in mixed warm shades", Bloom: "214", Leaf: "151"},
		{Name: "Tip Top Mahogany", Note: "deep mahogany, held above the leaves", Bloom: "88"},
	},
	"honeysuckle": {
		{Name: "the wild woodbine", Note: "cream ageing to gold, hedgerow scented", Bloom: "223"},
		{Name: "Serotina", Note: "purple-red outside, cream within, and late", Bloom: "132", Accent: "223"},
		{Name: "Graham Thomas", Note: "long copper-white trumpets, a heavy crop", Bloom: "230"},
	},
}
