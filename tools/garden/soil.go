package main

import "math"

// Soil. Every bed has its own pH and its own richness, and plants notice.
// Nothing here is harsh: ground a plant dislikes slows it down, it never
// kills it.

type soilPref int

const (
	soilAny soilPref = iota
	soilAcid
	soilNeutral
	soilChalk
)

func (s soilPref) String() string {
	switch s {
	case soilAcid:
		return "acid soil"
	case soilChalk:
		return "chalky, alkaline soil"
	case soilNeutral:
		return "neutral soil"
	default:
		return "almost any soil"
	}
}

// soilPrefs lists the species that genuinely care. Everything absent from
// this table is easy-going and grows anywhere, which is true of most garden
// plants.
var soilPrefs = map[string]soilPref{
	// Acid lovers: bog plants, woodlanders and the ericaceous crowd.
	"hydrangea":     soilAcid,
	"japanesemaple": soilAcid,
	"magnolia":      soilAcid,
	"maidenhair":    soilAcid,
	"ostrichfern":   soilAcid,
	"haircapmoss":   soilAcid,
	"venusflytrap":  soilAcid,
	"pitcherplant":  soilAcid,
	"sundew":        soilAcid,
	"blackpine":     soilAcid,
	"potato":        soilAcid,

	// Chalk and lime: the Mediterranean herbs and a few border plants.
	"lavender":   soilChalk,
	"rosemary":   soilChalk,
	"thyme":      soilChalk,
	"sage":       soilChalk,
	"oregano":    soilChalk,
	"olive":      soilChalk,
	"clematis":   soilChalk,
	"delphinium": soilChalk,
	"iris":       soilChalk,

	// Fussy about neither extreme, but they do want the middle ground.
	"peony":      soilNeutral,
	"rose":       soilNeutral,
	"tomato":     soilNeutral,
	"beetroot":   soilNeutral,
	"sweetcorn":  soilNeutral,
	"lettuce":    soilNeutral,
	"pea":        soilNeutral,
	"broadbean":  soilNeutral,
	"apple":      soilNeutral,
	"lilyvalley": soilNeutral,
}

// PrefersSoil is what a species wants underfoot.
func (s *Species) PrefersSoil() soilPref { return soilPrefs[s.ID] }

// suitsPH reports whether a pH reading suits a preference.
func (p soilPref) suitsPH(ph float64) bool {
	switch p {
	case soilAcid:
		return ph <= 6.2
	case soilChalk:
		return ph >= 6.9
	case soilNeutral:
		return ph >= 6.0 && ph <= 7.3
	default:
		return true
	}
}

// soilFactor is how much the bed's ground helps or hinders. Rich, suitable
// soil grows a plant about a quarter faster than poor, wrong soil.
func soilFactor(sp *Species, p *Plot) float64 {
	if p.Pond {
		return 1 // a pond is what the water plants want and all they get
	}
	fit := 1.0
	if !sp.PrefersSoil().suitsPH(p.PH) {
		fit = 0.85
	}
	return fit * (0.95 + 0.2*clamp01(p.Richness))
}

// soilNote explains a bed's ground to the gardener in one line.
func soilNote(sp *Species, p *Plot) string {
	if p.Pond {
		return "a pond, which is exactly right"
	}
	pref := sp.PrefersSoil()
	switch {
	case pref == soilAny:
		return "happy in this bed"
	case pref.suitsPH(p.PH):
		return "likes this ground (" + pref.String() + ")"
	case pref == soilAcid:
		return "would prefer more acid ground"
	case pref == soilChalk:
		return "would prefer chalkier ground"
	default:
		return "would prefer the middle ground, nearer pH 6.5"
	}
}

// hydrangeaBlue and friends: bigleaf hydrangea reads the soil and flowers to
// match it, the one plant in the garden that does its own chemistry.
var hydrangeaBloom = struct{ acid, neutral, chalk string }{
	acid:    "69",  // aluminium taken up in acid soil: blue
	neutral: "140", // mauve in between
	chalk:   "168", // locked away in lime: pink
}

// PaletteIn is the species' colouring in a particular bed. Only hydrangea
// changes, and it changes for the real reason.
func (s *Species) PaletteIn(p *Plot) Palette {
	pal := s.Palette
	if s.ID != "hydrangea" || p == nil || p.Pond {
		return pal
	}
	switch {
	case p.PH <= 5.9:
		pal.Bloom = hydrangeaBloom.acid
	case p.PH >= 6.8:
		pal.Bloom = hydrangeaBloom.chalk
	default:
		pal.Bloom = hydrangeaBloom.neutral
	}
	return pal
}

// HydrangeaColour names the flower colour a bed's pH will produce.
func HydrangeaColour(ph float64) string {
	switch {
	case ph <= 5.9:
		return "blue"
	case ph >= 6.8:
		return "pink"
	default:
		return "mauve"
	}
}

// feed is the slow draw a growing plant makes on a bed's richness.
func feed(p *Plot, dt float64) {
	p.Richness = math.Max(0, p.Richness-0.005*dt)
}
