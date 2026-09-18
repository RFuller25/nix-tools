package main

// How a plant looks right now, which depends on more than how far it has
// grown: the season it is sleeping through, the soil it is standing in and
// whether it has finished its year.

// appearance picks the frame and the colours a bed's plant is drawn with.
func appearance(sp *Species, p *Plot, season Season) (int, Palette) {
	switch {
	case p.Spent:
		// Gone to seed: the same shape, bleached to straw.
		return StageMature, driedPalette

	case p.Growth >= 1 && dormant(sp, season):
		// Asleep for the winter. Herbaceous perennials die back to a tuft;
		// deciduous trees and shrubs stand bare.
		if sp.Life() == Woody {
			return StageBud, sleepingPalette
		}
		return StageSeedling, sleepingPalette

	default:
		return p.Stage(), sp.PaletteIn(p)
	}
}

// state describes what the plant is doing, for the info card.
func state(sp *Species, p *Plot, season Season) string {
	switch {
	case p.Spent:
		return "gone to seed, standing dry"
	case p.Growth >= 1 && dormant(sp, season):
		if sp.Life() == Woody {
			return "bare for the winter"
		}
		return "died back, resting until spring"
	case p.Growth >= 1:
		return "in full growth"
	default:
		return "growing on"
	}
}
