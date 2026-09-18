package main

// Plants for the dark: flowers that stay shut all day, open at dusk in a
// matter of minutes, and hand the evening over to the moths.
func init() {
	register(
		&Species{
			ID: "moonflower", Common: "Moonflower", Latin: "Ipomoea alba",
			Family: "Convolvulaceae", Kind: KindVine, Rarity: Rare,
			Origin: "tropical America", Bloom: "summer nights",
			Sun: "full sun", Water: "moderate", Height: "3–5 m of climb",
			Note:    "Sit by it at dusk: the buds unfurl fast enough to watch.",
			Desc:    "A night-blooming relative of the morning glory whose hand-sized white trumpets spiral open in a minute or two at dusk, pour out scent for the hawk moths that pollinate them, and are limp by breakfast. Nick and soak the hard seed before sowing.",
			Seasons: []Season{Summer}, SeedCost: 9, Unlock: 8, Hours: 17,
			Palette: Palette{Stem: "65", Leaf: "71", Bloom: "255", Accent: "194"},
			Art: [StageCount][]string{
				{"  ●  "},
				{"  ϑ  "},
				{" ϑ╱  ", " ╱   "},
				{" ◦╱ϑ ", " ╱ϑ  ", " │   "},
				{" ❀╱ϑ ", " ╱❀  ", " ϑ╱  ", " ╱   "},
			},
		},
		&Species{
			ID: "eveningprimrose", Common: "Evening Primrose", Latin: "Oenothera biennis",
			Family: "Onagraceae", Kind: KindFlower, Rarity: Uncommon,
			Origin: "eastern North America", Bloom: "summer evenings",
			Sun: "full sun", Water: "low once settled", Height: "1–1.5 m",
			Note:    "Leave a few seed heads: it sows itself and never needs buying twice.",
			Desc:    "A biennial that spends a year as a flat rosette and the next throwing up a spire of lemon-scented flowers. Each bud snaps open at dusk in seconds, fast enough to see, and is done by the following afternoon. The seed oil is one of the few plant sources of gamma-linolenic acid.",
			Seasons: []Season{Summer}, SeedCost: 5, Unlock: 3, Hours: 16,
			Palette: Palette{Stem: "101", Leaf: "108", Bloom: "227", Accent: "229"},
			Art: [StageCount][]string{
				{"  ·  "},
				{"  ϑ  "},
				{" ε│з ", "  │  "},
				{"  ◦  ", " ◦│◦ ", " ε│з "},
				{"  ✿  ", " ✿│✿ ", " ✿│✿ ", " ε│з ", "  │  "},
			},
		},
		&Species{
			ID: "nightstock", Common: "Night-scented Stock", Latin: "Matthiola longipetala",
			Family: "Brassicaceae", Kind: KindFlower, Rarity: Common,
			Origin: "Greece and southwest Asia", Bloom: "summer, from dusk",
			Sun: "full sun", Water: "moderate", Height: "30–45 cm",
			Note:    "Sow it under a window you leave open on summer evenings.",
			Desc:    "A plain, straggling little annual that looks like nothing at all by day, with its flowers shut and limp. At dusk they open and the scent carries right across a garden — one of the strongest of any hardy plant, and all of it aimed at night-flying moths.",
			Seasons: []Season{Summer}, SeedCost: 3, Unlock: 1, Hours: 13,
			Palette: Palette{Stem: "65", Leaf: "108", Bloom: "183", Accent: "225"},
			Art: [StageCount][]string{
				{"  ·  "},
				{"  ,  "},
				{" ~~~ "},
				{" ~❁~ ", " ╲│╱ "},
				{" ❁~❁ ", " ~❁~ ", " ╲│╱ "},
			},
		},
	)
}
