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

// actSow plants the seed chosen in the shed into the selected bed, or the
// next free one.
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
	m.setStatus(okStyle, "Sowed %s in bed %d.%s", sp.VarietyName(m.shopVariety), idx+1, companionAside(m.g, idx, sp))
	return m.save()
}

// currencyWord is the unit, singular or plural.
func currencyWord(n int) string {
	if n == 1 {
		return "seed"
	}
	return "seeds"
}
