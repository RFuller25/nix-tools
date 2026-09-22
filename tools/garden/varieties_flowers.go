package main

// Varieties of the flowers. Named cultivars where the plant has famous ones,
// honest colour forms where it does not.

var flowerVarieties = map[string][]Variety{
	"sunflower": {
		{Name: "Russian Giant", Note: "one stem, one enormous head, four metres up", Bloom: "220"},
		{Name: "Velvet Queen", Note: "deep mahogany red with a dark centre", Bloom: "88", Accent: "52"},
		{Name: "Moonwalker", Note: "pale lemon, branching, many smaller heads", Bloom: "229"},
		{Name: "Teddy Bear", Note: "knee-high and fully double, like a shaggy pompom", Bloom: "214",
			Art: &[StageCount][]string{
				{},
				{},
				{},
				{"  ◍  ", " ε│з "},
				{" ❋❋❋ ", " ❋❋❋ ", " ε│з ", "  │  "},
			}},
	},
	"foxglove": {
		{Name: "Alba", Note: "pure white, no speckling at all", Bloom: "255", Accent: "194"},
		{Name: "Sutton's Apricot", Note: "soft apricot, unusual among foxgloves", Bloom: "216"},
		{Name: "Pam's Choice", Note: "white with a heavily blotched maroon throat", Bloom: "253", Accent: "89"},
		{Name: "the wild plant", Note: "the purple foxglove of hedge banks", Bloom: "176"},
	},
	"marigold": {
		{Name: "Naughty Marietta", Note: "single gold with a mahogany blotch", Bloom: "220", Accent: "94"},
		{Name: "Queen Sophia", Note: "russet orange, each petal edged in gold", Bloom: "202", Accent: "214"},
		{Name: "Bonanza Bolero", Note: "double, gold splashed with red", Bloom: "208", Accent: "160"},
	},
	"zinnia": {
		{Name: "Benary's Giant Scarlet", Note: "tall, fully double, florist's scarlet", Bloom: "196"},
		{Name: "Envy", Note: "soft lime green, which almost nothing else is", Bloom: "149"},
		{Name: "Queen Red Lime", Note: "dusky rose fading to lime at the centre", Bloom: "175", Accent: "149"},
	},
	"cosmos": {
		{Name: "Purity", Note: "white, the tallest and airiest of them", Bloom: "255"},
		{Name: "Sensation Pinkie", Note: "clear pink with a yellow eye", Bloom: "218", Accent: "220"},
		{Name: "Rubenza", Note: "deep ruby, fading to dusty rose as it ages", Bloom: "125"},
	},
	"snapdragon": {
		{Name: "Black Prince", Note: "crimson flowers over bronze foliage", Bloom: "124", Leaf: "65"},
		{Name: "Madame Butterfly", Note: "double, open-faced, azalea-flowered", Bloom: "217"},
		{Name: "Night and Day", Note: "deep crimson with a white throat", Bloom: "160", Accent: "255"},
	},
	"poppy": {
		{Name: "the wild plant", Note: "the scarlet poppy of ploughed fields", Bloom: "196"},
		{Name: "Shirley Single", Note: "pastels, white-based rather than black", Bloom: "218", Accent: "255"},
		{Name: "Amazing Grey", Note: "smoky grey-lilac, no two quite alike", Bloom: "146"},
	},
	"coneflower": {
		{Name: "Magnus", Note: "deep purple, petals held flat rather than drooping", Bloom: "133"},
		{Name: "White Swan", Note: "white rays around an orange cone", Bloom: "255", Accent: "208"},
		{Name: "Green Jewel", Note: "green rays, dwarf and scented", Bloom: "150"},
	},
	"rudbeckia": {
		{Name: "Prairie Sun", Note: "gold petals, light tips, a green eye", Bloom: "220", Accent: "149"},
		{Name: "Cherry Brandy", Note: "deep cherry red, the only red rudbeckia", Bloom: "125"},
		{Name: "Cherokee Sunset", Note: "double, in bronze, rust and gold", Bloom: "208", Accent: "94"},
	},
	"lupine": {
		{Name: "The Governor", Note: "blue standards over a white keel", Bloom: "62", Accent: "255"},
		{Name: "Chandelier", Note: "clear butter yellow", Bloom: "227"},
		{Name: "My Castle", Note: "brick red, shorter and sturdier", Bloom: "160"},
	},
	"delphinium": {
		{Name: "Black Knight", Note: "darkest violet, black-eyed", Bloom: "57", Accent: "233"},
		{Name: "Galahad", Note: "pure white with a white eye", Bloom: "255"},
		{Name: "Summer Skies", Note: "pale sky blue, the classic border spire", Bloom: "111"},
	},
	"peony": {
		{Name: "Sarah Bernhardt", Note: "vast double blush-pink, scented", Bloom: "218"},
		{Name: "Festiva Maxima", Note: "white, flecked crimson at the heart", Bloom: "255", Accent: "160"},
		{Name: "Karl Rosenfield", Note: "deep ruby red, upright stems", Bloom: "124"},
	},
	"rose": {
		{Name: "Officinalis", Note: "the apothecary's rose itself, semi-double crimson", Bloom: "161"},
		{Name: "Versicolor", Note: "Rosa Mundi — every petal striped pink on white", Bloom: "211", Accent: "255"},
		{Name: "Complicata", Note: "single, clear pink, a white eye, enormous hips", Bloom: "218", Accent: "196"},
	},
	"dahlia": {
		{Name: "Café au Lait", Note: "dinner-plate blooms in creamy blush", Bloom: "223"},
		{Name: "Bishop of Llandaff", Note: "single scarlet over near-black foliage", Bloom: "196", Leaf: "52"},
		{Name: "Thomas Edison", Note: "deep velvet purple, formal decorative", Bloom: "91"},
	},
	"chrysanthemum": {
		{Name: "Clara Curtis", Note: "a simple pink daisy, hardy and late", Bloom: "211"},
		{Name: "Bronze Elegance", Note: "small bronze buttons, very double", Bloom: "173"},
		{Name: "Emperor of China", Note: "quilled silver-pink, foliage turning red", Bloom: "182", Leaf: "131"},
	},
	"bleedingheart": {
		{Name: "the species", Note: "rose-pink lockets with a white drop", Bloom: "211", Accent: "255"},
		{Name: "Alba", Note: "pure white throughout, brighter in shade", Bloom: "255"},
		{Name: "Gold Heart", Note: "pink flowers over startling gold foliage", Bloom: "211", Leaf: "185"},
	},
	"mothorchid": {
		{Name: "the white species", Note: "plain white, the wild Phalaenopsis amabilis", Bloom: "255"},
		{Name: "a pink-lipped form", Note: "white petals, a deep pink lip", Bloom: "254", Accent: "205"},
		{Name: "a striped form", Note: "candy-striped through every petal", Bloom: "218", Accent: "161"},
	},
	"eveningprimrose": {
		{Name: "the wild plant", Note: "lemon yellow, opening at dusk", Bloom: "227"},
		{Name: "a large-flowered form", Note: "fewer, wider flowers, deeper gold", Bloom: "220"},
		{Name: "a compact form", Note: "half the height, better in a small bed", Bloom: "228"},
	},
	"nightstock": {
		{Name: "the wild plant", Note: "dowdy lilac by day, extraordinary at dusk", Bloom: "183"},
		{Name: "Starlight Scentsation", Note: "a seed mix from white through rose to purple", Bloom: "218"},
	},
	"moonflower": {
		{Name: "the species", Note: "hand-sized white trumpets, opening in minutes", Bloom: "255"},
		{Name: "a pink-throated form", Note: "white with a blush of pink down the throat", Bloom: "254", Accent: "218"},
	},
}
