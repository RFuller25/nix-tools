package main

// Varieties of the succulents, ferns, grasses, carnivores and water plants.

var wildVarieties = map[string][]Variety{
	"aloe": {
		{Name: "the species", Note: "plain grey-green, the medicine plant", Leaf: "108"},
		{Name: "a variegated form", Note: "leaves banded and striped in cream", Leaf: "151"},
		{Name: "a short-leaved form", Note: "squat, fat-leaved, slower to offset", Leaf: "72"},
	},
	"jade": {
		{Name: "the species", Note: "thick green paddles on a stout trunk", Leaf: "71"},
		{Name: "Hummel's Sunset", Note: "leaves edged gold and red in good light", Leaf: "179", Accent: "203"},
		{Name: "Gollum", Note: "tubular leaves with suckered tips", Leaf: "72"},
	},
	"sempervivum": {
		{Name: "the common houseleek", Note: "grey-green rosettes, red-tipped", Leaf: "108", Accent: "131"},
		{Name: "a cobweb form", Note: "strung across with white hairs like a web", Leaf: "151"},
		{Name: "a dark form", Note: "deep wine red through the summer", Leaf: "95"},
	},
	"barrelcactus": {
		{Name: "the species", Note: "golden spines in perfect ranks", Accent: "228"},
		{Name: "an ivory-spined form", Note: "white spines rather than gold", Accent: "255"},
	},
	"pricklypear": {
		{Name: "the species", Note: "yellow flowers, purple-red fruit", Bloom: "214", Accent: "125"},
		{Name: "a spineless form", Note: "Burbank's pads, safe to handle, still glochid-fringed", Leaf: "78"},
		{Name: "a purple-padded form", Note: "pads flushing violet in drought and cold", Leaf: "97"},
	},
	"stringofpearls": {
		{Name: "the species", Note: "round beads on long trailing strands", Leaf: "114"},
		{Name: "a variegated form", Note: "beads marbled cream and green", Leaf: "151"},
	},
	"echeveria": {
		{Name: "the species", Note: "pale blue-green rosettes dusted with farina", Leaf: "151"},
		{Name: "a pink-edged form", Note: "leaf margins blushing pink in sun", Leaf: "152", Accent: "218"},
		{Name: "a ruffled form", Note: "wavy-edged leaves, almost frilled", Leaf: "109"},
	},
	"maidenhair": {
		{Name: "the species", Note: "black wiry stalks, fan-shaped leaflets", Leaf: "114"},
		{Name: "a fine-leaved form", Note: "smaller leaflets, denser fronds", Leaf: "108"},
	},
	"ostrichfern": {
		{Name: "the species", Note: "a shuttlecock of arching fronds", Leaf: "71"},
		{Name: "a dwarf form", Note: "half the height, better in a small bed", Leaf: "65"},
	},
	"haircapmoss": {
		{Name: "the species", Note: "upright, the tallest of the common mosses", Leaf: "65"},
		{Name: "a bog form", Note: "looser, paler, from wetter ground", Leaf: "108"},
	},
	"hosta": {
		{Name: "Elegans", Note: "huge puckered blue-grey leaves", Leaf: "109"},
		{Name: "Frances Williams", Note: "blue leaves with wide gold margins", Leaf: "150", Accent: "185"},
		{Name: "a gold-leaved form", Note: "chartreuse, brightest in light shade", Leaf: "185"},
	},
	"bamboo": {
		{Name: "the species", Note: "gold canes, crowded joints at the base", Stem: "185"},
		{Name: "Koi", Note: "yellow canes striped green in the grooves", Stem: "227", Leaf: "113"},
		{Name: "a green-caned form", Note: "plain green culms, slightly taller", Stem: "108"},
	},
	"fountaingrass": {
		{Name: "Hameln", Note: "compact, buff bottlebrushes, early", Bloom: "180"},
		{Name: "Black Beauty", Note: "near-black flower spikes", Bloom: "238"},
		{Name: "a tall form", Note: "shoulder-high plumes, later to flower", Bloom: "223"},
	},
	"venusflytrap": {
		{Name: "the species", Note: "green traps, red-lined inside", Accent: "161"},
		{Name: "Akai Ryu", Note: "entirely deep red, leaf and trap alike", Leaf: "124", Accent: "88"},
		{Name: "a long-toothed form", Note: "exaggerated teeth round each trap", Accent: "167"},
	},
	"pitcherplant": {
		{Name: "the species", Note: "squat purple-veined pitchers, held upright", Accent: "125"},
		{Name: "Heterophylla", Note: "no red pigment at all: pure green pitchers", Accent: "114"},
		{Name: "a dark-lidded form", Note: "deep maroon lids over green pitchers", Accent: "88"},
	},
	"sundew": {
		{Name: "the species", Note: "round leaves, red tentacles, clear glue", Leaf: "167", Accent: "224"},
		{Name: "a green form", Note: "green tentacles, from shadier bog", Leaf: "114"},
	},
	"waterlily": {
		{Name: "the species", Note: "white, scented, opening in the morning", Bloom: "255"},
		{Name: "a pink form", Note: "soft pink, the commonest garden sort", Bloom: "218"},
		{Name: "Chromatella", Note: "canary yellow, leaves mottled bronze", Bloom: "227", Leaf: "137"},
	},
	"lotus": {
		{Name: "the sacred lotus", Note: "pink, enormous, held high above the water", Bloom: "218"},
		{Name: "Alba Grandiflora", Note: "pure white, the temple lotus", Bloom: "255"},
		{Name: "a dwarf bowl form", Note: "small enough for a half-barrel", Bloom: "211"},
	},
}
