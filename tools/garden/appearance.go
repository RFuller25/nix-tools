package main

// How a plant looks right now, which depends on more than how far it has
// grown: the season it is sleeping through, the soil it is standing in and
// whether it has finished its year.

// appearance picks the frame and the colours a bed's plant is drawn with,
// under the light of the moment.
func appearance(sp *Species, p *Plot, season Season, ph phase) (int, Palette) {
	stage, pal := bareAppearance(sp, p, season, ph)
	return stage, tintPalette(pal, ph)
}

// bareAppearance is the same choice without the light applied, which is what
// the tests reason about.
func bareAppearance(sp *Species, p *Plot, season Season, ph phase) (int, Palette) {
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

	case p.Growth >= 1 && sp.ClosesAtNight() && ph.Dark():
		// Shut for the night: the flower folds back to a bud.
		return StageBud, sp.PaletteIn(p)

	case p.Growth >= 1 && sp.OpensAtNight() && !ph.Dark() && ph != phaseDusk:
		// The other way round: these wait for the dark to open at all.
		return StageBud, sp.PaletteIn(p)

	default:
		return p.Stage(), sp.PaletteIn(p)
	}
}

// state describes what the plant is doing, for the info card.
func state(sp *Species, p *Plot, season Season, ph phase) string {
	switch {
	case p.Spent:
		return "gone to seed, standing dry"
	case p.Growth >= 1 && sp.ClosesAtNight() && ph.Dark():
		return "closed up for the night"
	case p.Growth >= 1 && sp.OpensAtNight() && (ph.Dark() || ph == phaseDusk):
		return "open for the moths, and scenting the dark"
	case p.Growth >= 1 && sp.OpensAtNight():
		return "shut until dusk"
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
