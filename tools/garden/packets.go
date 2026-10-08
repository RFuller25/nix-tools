package main

import (
	"fmt"
	"sort"
	"time"
)

// Packets. Seed in the shed is kept as a packet: a species, the genes of the
// two parents it came from, and how many seeds are left. A packet is a
// family. The genes of each seed are decided when it is sown, from the packet
// and how many seeds have already come out of it, so sowing three gives three
// different plants and sowing the same packet again on a replayed garden gives
// the same three.
//
// Seed from the shop is pure: both parents are the named form, and it comes
// true. Seed gathered from your own plants has the plant for one parent and
// whatever pollinated it for the other.

// Packet is a family of seeds in the shed.
type Packet struct {
	ID        int64  `json:"id"`
	SpeciesID string `json:"species"`
	A         Genome `json:"a"`
	B         Genome `json:"b"`
	Count     int    `json:"count"`
	Used      int    `json:"used,omitempty"` // seeds already sown from it
	Pure      bool   `json:"pure,omitempty"` // shop stock: comes exactly true
	Gen       int    `json:"gen,omitempty"`  // generation of the plants it grows
	Variety   int    `json:"variety,omitempty"`
	Label     string `json:"label,omitempty"`   // the gardener's own name for it
	From      string `json:"from,omitempty"`    // where it was gathered
	Streak    int    `json:"streak,omitempty"`  // generations in a row this line has bred true
	Descent   string `json:"descent,omitempty"` // how it came about, in words
	Line      int    `json:"line,omitempty"`    // cultivar it belongs to, if any
}

// Stable reports a line that has bred true for long enough to count as settled.
func (pk Packet) Stable() bool { return pk.Streak >= stableRuns }

func (pk Packet) Species() *Species { return SpeciesByID(pk.SpeciesID) }

// Mean is the genome the seedlings centre on.
func (pk Packet) Mean() Genome {
	if pk.Pure || pk.A == pk.B {
		return pk.A
	}
	return meanGenome(pk.A, pk.B)
}

// meanGenome is the noise-free blend of two genomes.
func meanGenome(a, b Genome) Genome {
	m := func(x, y uint8) uint8 { return uint8((int(x) + int(y) + 1) / 2) }
	wa, wb := float64(a.Sat)+1, float64(b.Sat)+1
	ha := circularWeighted(float64(a.Hue), wa, float64(b.Hue), wb)
	return Genome{Hue: uint16(ha) % 360, Sat: m(a.Sat, b.Sat), Light: m(a.Light, b.Light),
		Height: m(a.Height, b.Height), Shape: m(a.Shape, b.Shape), Speed: m(a.Speed, b.Speed), Yield: m(a.Yield, b.Yield)}
}

// Name is how the packet reads in the shed.
func (pk Packet) Name() string {
	sp := pk.Species()
	if sp == nil {
		return "unknown seed"
	}
	if pk.Label != "" {
		return pk.Label
	}
	if pk.Pure {
		return sp.VarietyName(pk.Variety)
	}
	idx, gap := sp.NearestVariety(pk.Mean())
	if gap > hybridGap {
		return fmt.Sprintf("%s hybrid (‘%s’ type)", sp.Common, sp.Variety(idx).Name)
	}
	return sp.VarietyName(idx)
}

// Child rolls the genes of the next seed out of the packet.
func (pk Packet) Child(gardenSeed int64) Genome {
	if pk.Pure {
		return pk.A
	}
	return breed(gardenSeed, hashSerial(pk.ID, int64(pk.Used)), pk.A, pk.B)
}

// nextPacketID hands out packet numbers, which are part of every roll.
func (g *Garden) nextPacketID() int64 {
	g.PacketSeq++
	return g.PacketSeq
}

// AddPacket files a packet in the shed. A pure packet of the same form is
// merged into the one already there.
func (g *Garden) AddPacket(pk Packet) {
	if pk.Count <= 0 {
		return
	}
	if pk.Pure {
		for i := range g.Shed {
			o := &g.Shed[i]
			if o.Pure && o.SpeciesID == pk.SpeciesID && o.A == pk.A && o.Label == pk.Label && o.Gen == pk.Gen {
				o.Count += pk.Count
				return
			}
		}
	}
	if pk.ID == 0 {
		pk.ID = g.nextPacketID()
	}
	g.Shed = append(g.Shed, pk)
}

// ShedOrder is the shed in the order it is shown: by species, then by name.
func (g *Garden) ShedOrder() []int {
	idx := make([]int, len(g.Shed))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		pa, pb := g.Shed[idx[a]], g.Shed[idx[b]]
		if pa.SpeciesID != pb.SpeciesID {
			sa, sb := pa.SpeciesID, pb.SpeciesID
			if A, B := pa.Species(), pb.Species(); A != nil && B != nil {
				sa, sb = A.Common, B.Common
			}
			return sa < sb
		}
		if na, nb := pa.Name(), pb.Name(); na != nb {
			return na < nb
		}
		return pa.ID < pb.ID
	})
	return idx
}

// SeedsInShed counts seeds across every packet.
func (g *Garden) SeedsInShed() int {
	n := 0
	for _, pk := range g.Shed {
		n += pk.Count
	}
	return n
}

// Buy puts n seeds of a named form into the shed, charging the shop price.
func (g *Garden) Buy(sp *Species, variety, n int, now time.Time) error {
	if n < 1 {
		return fmt.Errorf("nothing to buy")
	}
	if !g.Unlocked(sp) {
		return fmt.Errorf("%s needs %d matured plants to unlock", sp.Common, sp.Unlock)
	}
	price := sp.seedPrice() * n
	if !g.spend(price) {
		return fmt.Errorf("%d %s costs %s (you have %s)", n, sp.Common, goldLabel(price), goldLabel(g.Gold))
	}
	if variety < 0 || variety >= len(sp.Varieties()) {
		variety = 0
	}
	gn := sp.VarietyGenome(variety)
	g.AddPacket(Packet{SpeciesID: sp.ID, A: gn, B: gn, Count: n, Pure: true, Variety: variety})
	return nil
}

// checkSowable is the rules for putting a species in a bed.
func (g *Garden) checkSowable(idx int, sp *Species) error {
	if idx < 0 || idx >= len(g.Plots) {
		return fmt.Errorf("no such bed")
	}
	p := &g.Plots[idx]
	if !p.Empty() {
		return fmt.Errorf("bed %d already holds %s", idx+1, p.DisplayName())
	}
	if sp.Kind == KindAquatic && !p.Pond {
		return fmt.Errorf("%s needs a pond — dig one with d", sp.Common)
	}
	if sp.Kind != KindAquatic && p.Pond {
		return fmt.Errorf("bed %d is a pond; only water plants will grow there", idx+1)
	}
	return nil
}

// sow puts a plant with the given genes into a bed.
func (g *Garden) sow(idx int, sp *Species, gn Genome, gen int, now time.Time) {
	p := &g.Plots[idx]
	variety, _ := sp.NearestVariety(gn)
	*p = Plot{
		SpeciesID: sp.ID,
		Variety:   variety,
		Genome:    gn,
		Gen:       gen,
		PlantedAt: now,
		Moisture:  0.65, // a watering-in, as any gardener would
		Pond:      p.Pond,
		PH:        p.PH,
		Richness:  p.Richness,
	}
	g.Planted++
}

// SowPacket sows one seed from a packet in the shed.
func (g *Garden) SowPacket(idx, packet int, now time.Time) error {
	if packet < 0 || packet >= len(g.Shed) {
		return fmt.Errorf("no such packet")
	}
	pk := &g.Shed[packet]
	sp := pk.Species()
	if sp == nil {
		return fmt.Errorf("that seed is of no species this garden knows")
	}
	if err := g.checkSowable(idx, sp); err != nil {
		return err
	}
	gn := pk.Child(g.Seed)
	pk.Used++
	pk.Count--
	line, gen, label, descent := pk.Line, pk.Gen, pk.Label, pk.Descent
	// A seed that comes out close to its packet's centre carries the line's
	// run of true-breeding generations forward; one that does not starts again.
	streak := 0
	if gn.Distance(pk.Mean()) <= stableGap {
		streak = pk.Streak
	}
	g.sow(idx, sp, gn, gen, now)
	g.Plots[idx].Line = line
	g.Plots[idx].Streak = streak
	g.Plots[idx].Descent = descent
	if line != 0 {
		g.Plots[idx].LineName = label
	}
	if pk.Count <= 0 {
		g.Shed = append(g.Shed[:packet], g.Shed[packet+1:]...)
	}
	g.Log(now, "Sowed %s (%s) in bed %d.", g.Plots[idx].FullName(), sp.Latin, idx+1)
	return nil
}

// Plant buys one seed of a named form and sows it at once, as the shop's
// enter key does.
func (g *Garden) Plant(idx int, sp *Species, variety int, now time.Time) error {
	if err := g.checkSowable(idx, sp); err != nil {
		return err
	}
	if !g.Unlocked(sp) {
		return fmt.Errorf("%s needs %d matured plants to unlock", sp.Common, sp.Unlock)
	}
	if !g.spend(sp.seedPrice()) {
		return fmt.Errorf("not enough gold for %s (costs %d)", sp.Common, sp.seedPrice())
	}
	if variety < 0 || variety >= len(sp.Varieties()) {
		variety = 0
	}
	g.sow(idx, sp, sp.VarietyGenome(variety), 0, now)
	g.Log(now, "Sowed %s (%s) in bed %d.", sp.VarietyName(variety), sp.Latin, idx+1)
	return nil
}

// SellSeeds sells n seeds from a packet and returns the gold earned.
func (g *Garden) SellSeeds(packet, n int, now time.Time) (int, error) {
	if packet < 0 || packet >= len(g.Shed) {
		return 0, fmt.Errorf("no such packet")
	}
	pk := &g.Shed[packet]
	if n < 1 || n > pk.Count {
		n = pk.Count
	}
	each := seedValue(*pk)
	total := each * n
	name := pk.Name()
	pk.Count -= n
	pk.Used += n
	if pk.Count <= 0 {
		g.Shed = append(g.Shed[:packet], g.Shed[packet+1:]...)
	}
	g.earn(total)
	g.Sold += n
	g.Log(now, "Sold %d × %s for %s.", n, name, goldLabel(total))
	return total, nil
}

// RenamePacket gives a packet the gardener's own name.
func (g *Garden) RenamePacket(packet int, label string, now time.Time) bool {
	if packet < 0 || packet >= len(g.Shed) {
		return false
	}
	g.Shed[packet].Label = label
	g.nameSeedLine(packet, label, now)
	return true
}
