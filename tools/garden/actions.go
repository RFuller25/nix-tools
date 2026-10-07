package main

import tea "github.com/charmbracelet/bubbletea"

// The things a key can do to the garden. Each returns a command only when it
// has something to run, such as a save.

func (m *model) actWater() tea.Cmd {
	switch {
	case m.g.Water(m.cursor, m.now):
		m.dirty = true
		m.setStatus(waterStyle, "Watered %s.", m.plot().DisplayName())
	case m.plot().Empty():
		m.setStatus(subtleStyle, "Nothing planted in bed %d.", m.cursor+1)
	default:
		m.setStatus(subtleStyle, "%s has plenty to drink.", m.plot().DisplayName())
	}
	return nil
}

func (m *model) actWaterAll() tea.Cmd {
	if n := m.g.WaterAll(m.now); n > 0 {
		m.dirty = true
		m.setStatus(waterStyle, "Watered %d bed(s).", n)
	} else {
		m.setStatus(subtleStyle, "Every bed is already watered.")
	}
	return nil
}

func (m *model) actWeed() tea.Cmd {
	if ok, reward := m.g.Weed(m.cursor, m.now); ok {
		m.dirty = true
		if reward > 0 {
			m.setStatus(okStyle, "Cleared the weeds — found %d %s in the tangle.", reward, currencyWord(reward))
		} else {
			m.setStatus(okStyle, "Tidied bed %d.", m.cursor+1)
		}
	} else {
		m.setStatus(subtleStyle, "Bed %d is already clear.", m.cursor+1)
	}
	return nil
}

func (m *model) actWeedAll() tea.Cmd {
	total, reward := 0, 0
	for i := range m.g.Plots {
		if ok, r := m.g.Weed(i, m.now); ok {
			total++
			reward += r
		}
	}
	if total > 0 {
		m.dirty = true
		m.g.Log(m.now, "Weeded the whole garden (%d beds).", total)
		m.setStatus(okStyle, "Weeded %d bed(s), earning %d %s.", total, reward, currencyWord(reward))
	} else {
		m.setStatus(subtleStyle, "Not a weed in sight.")
	}
	return nil
}

func (m *model) actGather() tea.Cmd {
	if got := m.g.Gather(m.cursor, m.now); got > 0 {
		m.dirty = true
		m.setStatus(seedStyle, "Gathered %d seed(s) from %s.", got, m.plot().DisplayName())
	} else {
		m.setStatus(subtleStyle, "No ripe seed here yet.")
	}
	return nil
}

func (m *model) actGatherAll() tea.Cmd {
	got := 0
	for i := range m.g.Plots {
		got += m.g.Gather(i, m.now)
	}
	if got > 0 {
		m.dirty = true
		m.setStatus(seedStyle, "Gathered %d seed(s) from the garden.", got)
	} else {
		m.setStatus(subtleStyle, "Nothing ripe to gather.")
	}
	return nil
}

// actLift composts the plant under the cursor. From the card it also returns
// to the garden so the collapse can be watched.
func (m *model) actLift(fromCard bool) tea.Cmd {
	gain, ok := m.lift(m.cursor)
	if !ok {
		if !fromCard {
			m.setStatus(subtleStyle, "Bed %d is already empty.", m.cursor+1)
		}
		return nil
	}
	if fromCard {
		m.screen = screenGarden
	}
	m.setStatus(okStyle, "Composted %s into bed %d — the soil is %s now (+%.0f%% richness).",
		m.compost[m.cursor].Species.Common, m.cursor+1,
		richnessWord(m.g.Plots[m.cursor].Richness), gain*100)
	return nil
}

func (m *model) actBuyBed() tea.Cmd {
	if err := m.g.BuyBed(m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.cursor = len(m.g.Plots) - 1
	m.setStatus(okStyle, "New ground broken: bed %d. The next costs %d %s.", len(m.g.Plots), m.g.BedCost(), currencyWord(m.g.BedCost()))
	return nil
}

func (m *model) actPond() tea.Cmd {
	wasPond := m.plot().Pond
	if err := m.g.DigPond(m.cursor, m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	if wasPond {
		m.setStatus(okStyle, "Filled the pond back in.")
	} else {
		m.setStatus(waterStyle, "Dug a pond. Water lilies and lotus will grow here.")
	}
	return nil
}

// openShed goes to the seed shed. Choosing a bed to sow opens your own seed if
// you have any, since that is what you most likely want to plant.
func (m *model) openShed(sowing bool) {
	m.screen, m.cardScroll = screenShop, 0
	if sowing && len(m.g.Shed) > 0 {
		m.shelf = shelfMine
	}
	m.refreshShop()
}

func (m *model) actShelf(to shelf) tea.Cmd {
	m.shelf, m.cardScroll = to, 0
	if to == shelfShop {
		m.refreshShop()
	}
	return nil
}

// actSow buys the seed chosen in the shop and sows it in the selected bed, or
// the next free one.
func (m *model) actSow() tea.Cmd {
	if len(m.shop) == 0 {
		return nil
	}
	sp := m.shop[m.shopCursor]
	idx := m.firstEmptyFrom(m.cursor)
	if idx < 0 {
		m.setStatus(warnStyle, "Every bed is full — lift something first (u).")
		return nil
	}
	if err := m.g.Plant(idx, sp, m.shopVariety, m.now); err != nil {
		m.setStatus(errStyle, "%s", err.Error())
		return nil
	}
	m.cursor = idx
	m.dirty = true
	m.screen = screenGarden
	m.setStatus(okStyle, "Sowed %s in bed %d.%s", m.g.Plots[idx].FullName(), idx+1, companionAside(m.g, idx, sp))
	return m.save()
}

// actBuySeed puts a seed in the shed without planting it.
func (m *model) actBuySeed() tea.Cmd {
	if len(m.shop) == 0 {
		return nil
	}
	sp := m.shop[m.shopCursor]
	if err := m.g.Buy(sp, m.shopVariety, 1, m.now); err != nil {
		m.setStatus(errStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.setStatus(okStyle, "One %s seed in the shed. You have %s left.", sp.VarietyName(m.shopVariety), goldLabel(m.g.Gold))
	return nil
}

// minePacket is the index in the shed of the highlighted packet, or -1.
func (m *model) minePacket() int {
	order := m.g.ShedOrder()
	if len(order) == 0 {
		return -1
	}
	m.mineCursor = min(m.mineCursor, len(order)-1)
	return order[m.mineCursor]
}

// actSowMine sows a seed from the highlighted packet.
func (m *model) actSowMine() tea.Cmd {
	pi := m.minePacket()
	if pi < 0 {
		return nil
	}
	idx := m.firstEmptyFrom(m.cursor)
	if idx < 0 {
		m.setStatus(warnStyle, "Every bed is full — lift something first (u).")
		return nil
	}
	sp := m.g.Shed[pi].Species()
	if err := m.g.SowPacket(idx, pi, m.now); err != nil {
		m.setStatus(errStyle, "%s", err.Error())
		return nil
	}
	m.cursor = idx
	m.dirty = true
	if len(m.g.Shed) == 0 || m.mineCursor >= len(m.g.Shed) {
		m.mineCursor = max(0, len(m.g.Shed)-1)
	}
	m.screen = screenGarden
	m.setStatus(okStyle, "Sowed %s in bed %d.%s", m.g.Plots[idx].FullName(), idx+1, companionAside(m.g, idx, sp))
	return m.save()
}

// actSell sells one seed, or (n = 0) the whole packet.
func (m *model) actSell(n int) tea.Cmd {
	pi := m.minePacket()
	if pi < 0 {
		return nil
	}
	count := m.g.Shed[pi].Count
	if n == 0 || n > count {
		n = count
	}
	gold, err := m.g.SellSeeds(pi, n, m.now)
	if err != nil {
		m.setStatus(errStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	if m.mineCursor >= len(m.g.Shed) {
		m.mineCursor = max(0, len(m.g.Shed)-1)
	}
	m.setStatus(goldStyle, "Sold %d seed(s) for %s.", n, goldLabel(gold))
	return nil
}

// actStartPollinate begins choosing a donor for the flower under the cursor.
func (m *model) actStartPollinate() tea.Cmd {
	p := m.plot()
	sp := p.Species()
	switch {
	case sp == nil:
		m.setStatus(subtleStyle, "Nothing is growing in bed %d.", m.cursor+1)
	case !flowerOpen(sp, p, m.g.Season(m.now), m.phase()):
		m.setStatus(warnStyle, "%s has no open flower to pollinate right now.", p.DisplayName())
	case m.donorCount(m.cursor) == 0:
		m.setStatus(warnStyle, "There is no other open %s to take pollen from. Grow a second one.", sp.Common)
	default:
		m.pollinating, m.pollenTarget = true, m.cursor
		m.setStatus(goldStyle, "Pollinating %s: move to a %s to take pollen from, then enter.", p.DisplayName(), sp.Common)
	}
	return nil
}

// donorCount is how many other flowers could give this bed pollen.
func (m *model) donorCount(target int) int {
	n := 0
	for i := range m.g.Plots {
		if m.isDonor(target, i) {
			n++
		}
	}
	return n
}

// isDonor reports whether bed d could give bed t pollen right now.
func (m *model) isDonor(t, d int) bool {
	if t == d || t < 0 || d < 0 || t >= len(m.g.Plots) || d >= len(m.g.Plots) {
		return false
	}
	tp, dp := &m.g.Plots[t], &m.g.Plots[d]
	sp := tp.Species()
	return sp != nil && dp.SpeciesID == tp.SpeciesID && flowerOpen(sp, dp, m.g.Season(m.now), m.phase())
}

func (m *model) actFinishPollinate() tea.Cmd {
	target := m.pollenTarget
	if err := m.g.HandPollinate(target, m.cursor, m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.pollinating = false
	m.dirty = true
	m.setStatus(goldStyle, "Brushed %s's pollen onto %s. Its next seed will be that cross.",
		m.plot().DisplayName(), m.g.Plots[target].DisplayName())
	m.cursor = target
	return nil
}

// actDeliverHere fills the best-paying order the plant under the cursor fits.
func (m *model) actDeliverHere() tea.Cmd {
	p := m.plot()
	if p.Empty() {
		m.setStatus(subtleStyle, "Nothing is growing in bed %d.", m.cursor+1)
		return nil
	}
	fits := m.g.OrdersFor(p)
	if len(fits) == 0 {
		m.setStatus(subtleStyle, "No order on the board wants %s, as it is. The order board (O) says what is asked for.", p.DisplayName())
		return nil
	}
	name := p.FullName()
	gold, err := m.g.Deliver(fits[0].ID, m.cursor, m.now)
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.setStatus(goldStyle, "Delivered %s: %s.", name, goldLabel(gold))
	return m.save()
}

// actDeliverOrder fills the highlighted order from the first plant that fits.
func (m *model) actDeliverOrder() tea.Cmd {
	if len(m.g.Orders) == 0 {
		return nil
	}
	m.ordersCursor = min(m.ordersCursor, len(m.g.Orders)-1)
	o := m.g.Orders[m.ordersCursor]
	beds := m.g.QualifyingBeds(o)
	if len(beds) == 0 {
		m.setStatus(warnStyle, "None of your plants fit this order yet: they must be in flower and fit every condition.")
		return nil
	}
	name := m.g.Plots[beds[0]].FullName()
	gold, err := m.g.Deliver(o.ID, beds[0], m.now)
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.ordersCursor = min(m.ordersCursor, max(0, len(m.g.Orders)-1))
	m.setStatus(goldStyle, "Delivered %s: %s.", name, goldLabel(gold))
	return m.save()
}

func (m *model) actEnterHere() tea.Cmd {
	if err := m.g.Enter(m.cursor, m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.setStatus(goldStyle, "Entered %s in the %s class. The judges decide at the end of the week.", m.plot().DisplayName(), m.g.FairFor(weekKey(m.now)).Name)
	return nil
}

func (m *model) actEnterFair() tea.Cmd {
	cands := m.fairCandidates()
	if len(cands) == 0 {
		m.setStatus(warnStyle, "You have nothing in flower that suits this week's class.")
		return nil
	}
	c := cands[min(m.fairCursor, len(cands)-1)]
	m.cursor = c.Bed
	return m.actEnterHere()
}
