package main

// Varieties of the kitchen garden, where cultivars matter most: a tomato is
// never just a tomato.

var edibleVarieties = map[string][]Variety{
	"tomato": {
		{Name: "Gardener's Delight", Note: "small, sweet, reliable trusses of red", Accent: "196"},
		{Name: "Black Krim", Note: "dusky purple-brown beefsteak, rich and salty", Accent: "95"},
		{Name: "Sungold", Note: "orange cherries, the sweetest of the lot", Accent: "214"},
		{Name: "Tumbling Tom", Note: "a trailing bush for a basket, not a cordon", Accent: "203",
			Art: &[StageCount][]string{
				{"  ·  "},
				{"  ϑ  "},
				{" εзε "},
				{" ε❁з ", " ╲│╱ "},
				{" ε●зε ", "●зε●з ", " ╲╲│╱ "},
			}},
	},
	"chilli": {
		{Name: "Jalapeño", Note: "thick-walled, moderate heat, picked green", Accent: "77"},
		{Name: "Cayenne", Note: "long, thin, dried and ground for the spice", Accent: "196"},
		{Name: "Purple Tiger", Note: "purple fruit and mottled purple leaves", Accent: "97", Leaf: "97"},
		{Name: "Lemon Drop", Note: "yellow, citrus-flavoured, seriously hot", Accent: "227"},
	},
	"carrot": {
		{Name: "Nantes", Note: "blunt-ended, sweet, the everyday carrot", Accent: "208"},
		{Name: "Purple Haze", Note: "purple skin, orange core, the older colour", Accent: "97"},
		{Name: "Paris Market", Note: "round and squat, for shallow or stony soil", Accent: "214"},
	},
	"radish": {
		{Name: "French Breakfast", Note: "long, red with a white tip, mild", Accent: "203"},
		{Name: "Cherry Belle", Note: "round, scarlet, ready in three weeks", Accent: "196"},
		{Name: "Black Spanish Round", Note: "black-skinned, white-fleshed, fierce, stores all winter", Accent: "236"},
	},
	"lettuce": {
		{Name: "Little Gem", Note: "small, crisp, sweet-hearted cos", Leaf: "114"},
		{Name: "Lollo Rosso", Note: "frilled and bronze-red, cut-and-come-again", Leaf: "131"},
		{Name: "Merveille des Quatre Saisons", Note: "butterhead, red-tinged, good in any season", Leaf: "138"},
	},
	"pea": {
		{Name: "Kelvedon Wonder", Note: "early, compact, heavy cropping", Accent: "119"},
		{Name: "Sugar Snap", Note: "eaten pod and all, tall and climbing", Accent: "120"},
		{Name: "Purple Podded", Note: "purple pods, green peas, a Victorian sort", Accent: "97"},
	},
	"broadbean": {
		{Name: "Aquadulce Claudia", Note: "the one to sow in autumn and overwinter", Accent: "119"},
		{Name: "Crimson Flowered", Note: "deep red flowers instead of black-and-white", Bloom: "160"},
		{Name: "The Sutton", Note: "dwarf, thirty centimetres, no staking", Accent: "114"},
	},
	"pumpkin": {
		{Name: "Jack Be Little", Note: "flattened orange fruit the size of a fist", Accent: "208"},
		{Name: "Crown Prince", Note: "steel-blue skin, dense orange flesh, keeps for months", Accent: "109"},
		{Name: "Rouge Vif d'Étampes", Note: "the flat scarlet Cinderella pumpkin", Accent: "160"},
	},
	"sweetcorn": {
		{Name: "Golden Bantam", Note: "the old open-pollinated yellow sweetcorn", Accent: "220"},
		{Name: "Glass Gem", Note: "translucent kernels in every colour, for drying", Accent: "141"},
		{Name: "Painted Mountain", Note: "short-season flint corn, red and orange", Accent: "166"},
	},
	"strawberry": {
		{Name: "Cambridge Favourite", Note: "heavy mid-season crop, good in any soil", Accent: "197"},
		{Name: "Mara des Bois", Note: "everbearing, with the scent of a wild strawberry", Accent: "160"},
		{Name: "Pineapple Crush", Note: "white-fruited, and the birds leave it alone", Accent: "230"},
	},
	"garlic": {
		{Name: "Solent Wight", Note: "softneck, stores until the following spring", Accent: "253"},
		{Name: "Chesnok Wight", Note: "hardneck, purple-striped, sweet when roasted", Accent: "182"},
		{Name: "Elephant garlic", Note: "enormous mild cloves; a leek, strictly speaking", Accent: "255"},
	},
	"beetroot": {
		{Name: "Boltardy", Note: "deep red, slow to bolt, sow it early", Accent: "125"},
		{Name: "Chioggia", Note: "cut it open for concentric pink and white rings", Accent: "211"},
		{Name: "Burpee's Golden", Note: "orange skin, yellow flesh, does not bleed", Accent: "214"},
	},
}
