package main

// How long a plant lives. Nothing in this garden dies: an annual that has
// finished its year goes to seed and stands there dry until the gardener
// lifts it, and a perennial simply sleeps through the winter.

type LifeKind int

const (
	Annual LifeKind = iota
	Biennial
	Perennial
	Woody
)

func (l LifeKind) String() string {
	switch l {
	case Annual:
		return "annual"
	case Biennial:
		return "biennial"
	case Perennial:
		return "perennial"
	default:
		return "shrub or tree"
	}
}

// lives records what each species actually is. Where gardeners and botanists
// disagree — a tomato is a perennial in its homeland and an annual in a
// temperate garden — this follows the way the plant is grown.
var lives = map[string]LifeKind{
	// Annuals: one season, then seed.
	"basil": Annual, "broadbean": Annual, "chamomile": Annual, "chilli": Annual,
	"cosmos": Annual, "dill": Annual, "garlic": Annual, "lettuce": Annual,
	"marigold": Annual, "morningglory": Annual, "nasturtium": Annual,
	"pea": Annual, "poppy": Annual, "pumpkin": Annual, "radish": Annual,
	"snapdragon": Annual, "sunflower": Annual, "sweetcorn": Annual,
	"sweetpea": Annual, "tomato": Annual, "zinnia": Annual,
	"moonflower": Annual, "nightstock": Annual,

	// Biennials: leaves the first year, flowers the second.
	"beetroot": Biennial, "carrot": Biennial, "foxglove": Biennial, "parsley": Biennial,
	"eveningprimrose": Biennial,

	// Herbaceous perennials, including the bulbs and the tender things we
	// keep going year to year.
	"allium": Perennial, "aloe": Perennial, "bamboo": Perennial,
	"barrelcactus": Perennial, "bleedingheart": Perennial, "chives": Perennial,
	"chrysanthemum": Perennial, "coneflower": Perennial, "crocus": Perennial,
	"daffodil": Perennial, "dahlia": Perennial, "delphinium": Perennial,
	"echeveria": Perennial, "fountaingrass": Perennial, "haircapmoss": Perennial,
	"hosta": Perennial, "hyacinth": Perennial, "iris": Perennial,
	"jade": Perennial, "lemonbalm": Perennial, "lilyvalley": Perennial,
	"lotus": Perennial, "lupine": Perennial, "maidenhair": Perennial,
	"mint": Perennial, "mothorchid": Perennial, "oregano": Perennial,
	"ostrichfern": Perennial, "peony": Perennial, "pitcherplant": Perennial,
	"pricklypear": Perennial, "rudbeckia": Perennial, "sempervivum": Perennial,
	"snowdrop": Perennial, "strawberry": Perennial, "stringofpearls": Perennial,
	"sundew": Perennial, "tigerlily": Perennial, "tulip": Perennial,
	"venusflytrap": Perennial, "waterlily": Perennial,

	// Woody: shrubs, trees and the climbers that build a trunk.
	"apple": Woody, "blackpine": Woody, "cherryblossom": Woody, "clematis": Woody,
	"ginkgo": Woody, "honeysuckle": Woody, "hydrangea": Woody, "japanesemaple": Woody,
	"lavender": Woody, "lemon": Woody, "magnolia": Woody, "olive": Woody,
	"rose": Woody, "rosemary": Woody, "sage": Woody, "silverbirch": Woody,
	"thyme": Woody, "willow": Woody, "wisteria": Woody,
}

// evergreens keep their leaves through the winter, so they are drawn the same
// in January as in June.
var evergreens = map[string]bool{
	"aloe": true, "bamboo": true, "barrelcactus": true, "blackpine": true,
	"echeveria": true, "haircapmoss": true, "jade": true, "lavender": true,
	"mothorchid": true, "olive": true, "pricklypear": true, "rosemary": true,
	"sage": true, "sempervivum": true, "stringofpearls": true, "thyme": true,
}

// selfSeeders scatter their own seed about and come up wherever there is a
// gap, which is why these are the plants that fill a cottage garden.
var selfSeeders = map[string]bool{
	"poppy": true, "cosmos": true, "chamomile": true, "nasturtium": true,
	"foxglove": true, "lupine": true, "dill": true, "lemonbalm": true,
	"marigold": true, "snapdragon": true, "sunflower": true, "parsley": true,
	"chives": true, "lettuce": true, "sweetpea": true, "morningglory": true,
	"eveningprimrose": true, "nightstock": true,
}

// Life is what kind of life the species leads.
func (s *Species) Life() LifeKind {
	if l, ok := lives[s.ID]; ok {
		return l
	}
	// Anything that slipped the table is treated by its kind.
	switch s.Kind {
	case KindTree:
		return Woody
	case KindEdible:
		return Annual
	default:
		return Perennial
	}
}

// Evergreen reports whether the plant keeps its leaves all winter.
func (s *Species) Evergreen() bool { return evergreens[s.ID] }

// SelfSeeds reports whether the plant sows itself into nearby ground.
func (s *Species) SelfSeeds() bool { return selfSeeders[s.ID] }

// seedSpan is how long a plant carries on after reaching maturity before it
// finishes and goes to seed. Perennials and woody plants never do.
func seedSpan(s *Species) float64 {
	switch s.Life() {
	case Annual:
		return s.Hours * 2.5
	case Biennial:
		return s.Hours * 3.5
	default:
		return 0
	}
}

// Dormant reports whether the plant is asleep for the winter: herbaceous
// perennials die back to the ground, deciduous trees and shrubs stand bare,
// and evergreens carry on as they are.
func dormant(s *Species, season Season) bool {
	if season != Winter || s.Evergreen() {
		return false
	}
	switch s.Life() {
	case Perennial, Woody:
		return true
	default:
		return false
	}
}

// driedPalette is how a plant that has gone to seed is drawn: stems and seed
// heads bleached to straw.
var driedPalette = Palette{Stem: "101", Leaf: "137", Bloom: "180", Accent: "94"}

// sleepingPalette is a dormant plant: still there, just not doing much.
var sleepingPalette = Palette{Stem: "95", Leaf: "65", Bloom: "101", Accent: "138"}
