package main

import (
	"fmt"
	"math"
)

// The economy. Gold buys seed, ground and ponds; it is earned by selling the
// seed your own plants make, by weeding, by the daily stipend, by orders and by
// the fair. Nothing in the garden ever takes gold away: it can only be spent.
//
// Every price and rate is here, and the almanac's guide quotes these
// constants, so retuning the economy is one edit.
const (
	startGold    = 20 // a new gardener's purse
	dailyStipend = 3  // collected on the first visit each day

	weedingReward = 1 // for clearing a properly overgrown bed
	bedBaseCost   = 18
	bedStepCost   = 6
	pondCost      = 12

	sellBase      = 0.45 // fraction of a seed's shop price a plain home-grown seed fetches
	sellNovelty   = 2.0  // extra multiples for a seed unlike any named form
	sellStable    = 0.6  // extra for a stable, named line
	sellYield     = 0.5  // extra at Yield 100, less at 0
	plannerPlot   = 0    // planting fee per bed when stamping a template (stamping is free)
	maxShedLabels = 24
)

// goldLabel formats an amount.
func goldLabel(n int) string { return fmt.Sprintf("%d gold", n) }

// currencyWord is the unit, for sentences that count it.
func currencyWord(n int) string { return "gold" }

// CanAfford reports whether the purse covers a price.
func (g *Garden) CanAfford(price int) bool { return g.Gold >= price }

// spend takes gold, returning false (and taking nothing) if there is not enough.
func (g *Garden) spend(price int) bool {
	if price < 0 || g.Gold < price {
		return false
	}
	g.Gold -= price
	return true
}

// earn adds gold. It never goes negative or wraps.
func (g *Garden) earn(n int) {
	if n <= 0 {
		return
	}
	g.Gold += n
}

// seedPrice is what the shop charges for one seed of a species.
func (s *Species) seedPrice() int { return s.SeedCost }

// seedValue is what the shed pays for one seed of a packet: more for a seed
// that is unlike anything on the shop shelf, for one from a stable line and
// for a high-yielding strain, and never less than 1.
func seedValue(pk Packet) int {
	sp := pk.Species()
	if sp == nil {
		return 0
	}
	mean := pk.Mean()
	_, gap := sp.NearestVariety(mean)
	novelty := math.Min(1, gap/0.35)
	v := float64(sp.seedPrice()) * sellBase * (1 + sellNovelty*novelty)
	if pk.Stable {
		v *= 1 + sellStable
	}
	v *= 1 - sellYield/2 + sellYield*float64(mean.Yield)/100
	if v < 1 {
		return 1
	}
	return int(math.Round(v))
}
