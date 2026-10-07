package main

import (
	"fmt"
	"os"
	"time"
)

// Moving a version 1 garden across. This is a one-time act: version 1 had a
// single kind of seed that was both money and planting stock, and plants that
// were nothing but a species and a form. Version 2 has gold, packets of seed
// in a shed, and plants with genes, so the old garden is turned into the
// equivalent stock and the beds are cleared to begin again.
//
//   - every seed in the old purse becomes one gold;
//   - every plant standing in a bed becomes a packet of its named form in the
//     shed (and every ripe pod it held one more seed in that packet);
//   - the beds are cleared, keeping the ground itself: how many there are,
//     the ponds, the pH and the richness;
//   - the records stay: herbarium, sightings, forms grown, journal, tasks and
//     the lifetime tallies, which also keep what the shop has unlocked.
//
// The version number is the guard, so it can only ever happen once, and the
// old file is copied to <save>.v1.bak first.

// backupSuffix is appended to the save's name for the copy of a v1 garden.
const backupSuffix = ".v1.bak"

// needsMigration reports a garden saved before version 2.
func (g *Garden) needsMigration() bool { return g.Version < 2 }

// migrateV1 turns a version 1 garden into a version 2 one. It reports how
// many packets of seed it made.
func (g *Garden) migrateV1(now time.Time) (packets int) {
	if !g.needsMigration() {
		return 0
	}
	oldSeeds := g.Seeds
	g.Gold += oldSeeds
	g.Seeds = 0

	planted := 0
	for i := range g.Plots {
		p := &g.Plots[i]
		if sp := p.Species(); sp != nil && !p.Empty() {
			planted++
			gn := sp.VarietyGenome(p.Variety)
			before := len(g.Shed)
			g.AddPacket(Packet{
				SpeciesID: sp.ID, A: gn, B: gn, Count: 1 + int(p.Pods),
				Pure: true, Variety: p.Variety, From: "the old garden",
			})
			if len(g.Shed) > before {
				packets++
			}
		}
		moisture := 0.5
		if p.Pond {
			moisture = 1
		}
		*p = Plot{Pond: p.Pond, Moisture: moisture, PH: p.PH, Richness: p.Richness}
	}

	g.Version = gardenVersion
	g.Log(now, "The old garden has been cleared to begin again: %d seed(s) became %s, and what grew here is in the shed as %d packet(s).",
		oldSeeds, goldLabel(oldSeeds), packets)
	return packets
}

// backupV1 copies a version 1 save aside before it is rewritten. An existing
// backup is never overwritten.
func backupV1(path string, data []byte) error {
	dest := path + backupSuffix
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("backing up the old garden to %s: %w", dest, err)
	}
	return nil
}
