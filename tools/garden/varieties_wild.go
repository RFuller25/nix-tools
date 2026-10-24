package main

// Varieties of the succulents, ferns, grasses, carnivores and water plants.

var wildVarieties = map[string][]Variety{
	"aloe": {
		{Name: "the species", Note: "plain grey-green, the medicine plant", Leaf: "108"},
		{Name: "Delta Lights", Note: "leaves speckled and banded in cream", Leaf: "151"},
		{Name: "Blue Elf", Note: "squat, blue-green and slower to offset, with orange flowers", Leaf: "72"},
		{Name: "Hedgehog", Note: "a small clumping aloe, short and spiny-edged", Leaf: "65"},
	},
	"jade": {
		{Name: "the species", Note: "thick green paddles on a stout trunk", Leaf: "71"},
		{Name: "Hummel's Sunset", Note: "leaves edged gold and red in good light", Leaf: "179", Accent: "203"},
		{Name: "Gollum", Note: "tubular leaves with suckered tips", Leaf: "72"},
	},
	"sempervivum": {
		{Name: "the common houseleek", Note: "grey-green rosettes, red-tipped", Leaf: "108", Accent: "131"},
		{Name: "Cobweb Houseleek", Note: "Sempervivum arachnoideum, strung across with white hairs like a web", Leaf: "151"},
		{Name: "Commander Hay", Note: "large rosettes in deep wine red through the summer", Leaf: "95"},
		{Name: "Oddity", Note: "unusual tubular green leaves, flared and red-tipped", Leaf: "72", Accent: "131"},
	},
	"barrelcactus": {
		{Name: "the species", Note: "golden spines in perfect ranks", Accent: "228"},
		{Name: "Albispinus", Note: "white spines rather than gold", Accent: "255"},
		{Name: "Inermis", Note: "the spineless barrel, golden-ribbed and safe to handle", Accent: "180"},
	},
	"pricklypear": {
		{Name: "the species", Note: "yellow flowers, purple-red fruit", Bloom: "214", Accent: "125"},
		{Name: "Burbank Spineless", Note: "Burbank's pads, safe to handle, still glochid-fringed", Leaf: "78"},
		{Name: "Santa Rita", Note: "pads flushing violet in drought and cold", Leaf: "97"},
	},
	"stringofpearls": {
		{Name: "the species", Note: "round beads on long trailing strands", Leaf: "114"},
		{Name: "Variegata", Note: "beads marbled cream and green", Leaf: "151"},
	},
	"echeveria": {
		{Name: "the species", Note: "pale blue-green rosettes dusted with farina", Leaf: "151"},
		{Name: "Perle von Nürnberg", Note: "lilac-pink rosettes with a pearly bloom, blushing in sun", Leaf: "152", Accent: "218"},
		{Name: "Lola", Note: "pale lavender-pink rosettes with ruffled, wavy edges", Leaf: "183"},
		{Name: "Black Prince", Note: "near-black, deep wine-purple rosettes", Leaf: "53"},
	},
	"maidenhair": {
		{Name: "the species", Note: "black wiry stalks, fan-shaped leaflets", Leaf: "114"},
		{Name: "Fritz-Lüthi", Note: "smaller leaflets, denser and more upright fronds", Leaf: "108"},
		{Name: "Fragrans", Note: "a bushy, fine-textured maidenhair with a faint scent", Leaf: "151"},
	},
	"ostrichfern": {
		{Name: "the species", Note: "a shuttlecock of arching fronds", Leaf: "71"},
		{Name: "The King", Note: "bolder, more vigorous fronds, the largest of the shuttlecocks", Leaf: "65"},
	},
	"haircapmoss": {
		{Name: "the species", Note: "upright, the tallest of the common mosses", Leaf: "65"},
		{Name: "a bog form", Note: "looser, paler, from wetter ground", Leaf: "108"},
	},
	"hosta": {
		{Name: "Elegans", Note: "huge puckered blue-grey leaves", Leaf: "109"},
		{Name: "Frances Williams", Note: "blue leaves with wide gold margins", Leaf: "150", Accent: "185"},
		{Name: "Sum and Substance", Note: "huge chartreuse-gold leaves, the biggest of the hostas", Leaf: "185"},
		{Name: "Blue Angel", Note: "enormous blue-grey leaves and white flowers", Leaf: "109"},
		{Name: "Patriot", Note: "dark green leaves with a broad white margin", Leaf: "65", Accent: "255"},
	},
	"bamboo": {
		{Name: "the species", Note: "gold canes, crowded joints at the base", Stem: "185"},
		{Name: "Koi", Note: "yellow canes striped green in the grooves", Stem: "227", Leaf: "113"},
		{Name: "Holochrysa", Note: "canes turning wholly golden in full sun", Stem: "222"},
		{Name: "Flavescens Inversa", Note: "yellow canes with green striping, joints inverted", Stem: "143", Leaf: "113"},
	},
	"fountaingrass": {
		{Name: "Hameln", Note: "compact, buff bottlebrushes, early", Bloom: "180"},
		{Name: "Black Beauty", Note: "near-black flower spikes", Bloom: "238"},
		{Name: "Cassian's Choice", Note: "shoulder-high plumes, later to flower", Bloom: "223"},
		{Name: "Little Bunny", Note: "a dwarf fountain grass, knee-high and tidy", Bloom: "180"},
	},
	"venusflytrap": {
		{Name: "the species", Note: "green traps, red-lined inside", Accent: "161"},
		{Name: "Akai Ryu", Note: "entirely deep red, leaf and trap alike", Leaf: "124", Accent: "88"},
		{Name: "Dente", Note: "exaggerated teeth round each trap", Accent: "167"},
	},
	"pitcherplant": {
		{Name: "the species", Note: "squat purple-veined pitchers, held upright", Accent: "125"},
		{Name: "Heterophylla", Note: "no red pigment at all: pure green pitchers", Accent: "114"},
		{Name: "Adrian Slack", Note: "deep maroon lids over green pitchers", Accent: "88"},
		{Name: "Judith Hindle", Note: "a vigorous hybrid with big red-veined pitchers", Accent: "124"},
	},
	"sundew": {
		{Name: "the species", Note: "round leaves, red tentacles, clear glue", Leaf: "167", Accent: "224"},
		{Name: "Cape Sundew", Note: "Drosera capensis, narrow leaves with pink-red tentacles", Leaf: "168"},
		{Name: "Spoon-leaved", Note: "Drosera spatulata, a flat rosette of spoon-shaped leaves", Leaf: "114"},
	},
	"waterlily": {
		{Name: "the species", Note: "white, scented, opening in the morning", Bloom: "255"},
		{Name: "Marliacea Rosea", Note: "soft pink, the commonest garden sort", Bloom: "218"},
		{Name: "Chromatella", Note: "canary yellow, leaves mottled bronze", Bloom: "227", Leaf: "137"},
	},
	"lotus": {
		{Name: "the sacred lotus", Note: "pink, enormous, held high above the water", Bloom: "218"},
		{Name: "Alba Grandiflora", Note: "pure white, the temple lotus", Bloom: "255"},
		{Name: "Momo Botan", Note: "a dwarf lotus with deep pink double flowers, small enough for a half-barrel", Bloom: "211"},
	},
}
