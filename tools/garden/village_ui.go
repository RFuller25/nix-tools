package main

import (
	"fmt"
	"strings"
)

// openVillage walks down to the village.
func (m *model) openVillage() {
	m.screen, m.villageStage, m.villageCursor, m.villagePickCursor = screenVillage, villageList, 0, 0
}

// villageMove changes tab.
func (m *model) villageMove(delta int) {
	if m.villageStage != villageList {
		return
	}
	m.villageTab = villageTab((int(m.villageTab) + delta + int(villageTabs)) % int(villageTabs))
	m.villageCursor, m.villageResult = 0, nil
}

// villageStep moves the highlight in whichever list is showing.
func (m *model) villageStep(delta int) {
	if m.villageStage == villageList {
		n := len(m.villageRows())
		m.villageCursor = max(0, min(n-1, m.villageCursor+delta))
		m.villageResult = nil
		return
	}
	n := len(m.villageTargets())
	m.villagePickCursor = max(0, min(n-1, m.villagePickCursor+delta))
}

// villageBack steps out of a choice, or leaves the village.
func (m *model) villageBack() {
	if m.villageStage != villageList {
		m.villageStage = villageList
		return
	}
	m.screen = screenGarden
}

// actVillageEnter does the highlighted thing.
func (m *model) actVillageEnter() tcmd {
	if m.villageStage != villageList {
		return m.actVillagePick()
	}
	rows := m.villageRows()
	if m.villageCursor >= len(rows) {
		return nil
	}
	m.villageResult = nil
	g := m.g
	switch m.villageTab {
	case tabHelp:
		d := helperDefs[m.villageCursor]
		if g.Hired(d.ID) {
			m.setStatus(subtleStyle, "%s is on the payroll. x lets them go.", d.Name)
			return nil
		}
		if err := g.HireHelper(d.ID, m.now); err != nil {
			m.setStatus(warnStyle, "%s", err.Error())
			return nil
		}
		m.dirty = true
		m.setStatus(goldStyle, "%s starts today.", d.Name)
	case tabScenery:
		d := sceneryDefs[m.villageCursor]
		if err := g.BuyScenery(d.ID, m.now); err != nil {
			m.setStatus(warnStyle, "%s", err.Error())
			return nil
		}
		m.dirty = true
		m.setStatus(goldStyle, "A %s now stands in the garden.", d.Name)
	case tabLab, tabCommission:
		m.villageStage, m.villagePickCursor = villagePick, 0
		if len(m.villageTargets()) == 0 {
			m.villageStage = villageList
			m.setStatus(warnStyle, "%s", m.villageNothing())
		}
	case tabStyle:
		var bought bool
		var err error
		if m.villageCursor < len(themeDefs) {
			d := themeDefs[m.villageCursor]
			if bought, err = g.UseTheme(d.ID, m.now); err == nil {
				applyTheme(d.ID)
				m.setStatus(okStyle, "The garden is dressed in %s.", d.Name)
			}
		} else {
			d := borderDefs[m.villageCursor-len(themeDefs)]
			if bought, err = g.UseBorder(d.ID, m.now); err == nil {
				m.setStatus(okStyle, "Every bed now has a %s border.", strings.ToLower(d.Name))
			}
		}
		if err != nil {
			m.setStatus(warnStyle, "%s", err.Error())
			return nil
		}
		_ = bought
		m.dirty = true
	}
	return nil
}

func (m model) villageNothing() string {
	if m.villageTab == tabLab && m.villageCursor == labPurify {
		return "You have no mixed seed of your own to purify."
	}
	if m.villageTab == tabLab && m.villageCursor == labForecast && m.villageStage == villagePick2 {
		return "No other flower of that species is in bloom."
	}
	return "Nothing of yours is in bloom."
}

// actVillagePick works on the highlighted bed or packet.
func (m *model) actVillagePick() tcmd {
	targets := m.villageTargets()
	if len(targets) == 0 {
		return nil
	}
	t := targets[min(m.villagePickCursor, len(targets)-1)]
	g := m.g
	var err error
	switch {
	case m.villageTab == tabCommission:
		d := commissionDefs[m.villageCursor]
		if err = g.Commission(d.ID, t.Idx, m.now); err == nil {
			m.setStatus(goldStyle, "%s will have it ready: a %s for your shelf.", d.Maker, strings.ToLower(d.Name))
		}
	case m.villageCursor == labClone:
		if err = g.Clone(t.Idx, m.now); err == nil {
			m.setStatus(goldStyle, "The lab put %d seeds of it in your shed, exactly true.", labCloneSeeds)
		}
	case m.villageCursor == labPurify:
		if err = g.Purify(t.Idx, m.now); err == nil {
			m.setStatus(goldStyle, "That packet now comes exactly true.")
		}
	case m.villageStage == villagePick:
		m.villageFirst, m.villageStage, m.villagePickCursor = t.Idx, villagePick2, 0
		if len(m.villageTargets()) == 0 {
			m.villageStage = villageList
			m.setStatus(warnStyle, "%s", "No other flower of that species is in bloom.")
		}
		return nil
	default:
		var kids []Genome
		if kids, err = g.Forecast(m.villageFirst, t.Idx, m.now); err == nil {
			m.villageResult = forecastLines(g, m.villageFirst, t.Idx, kids)
			m.setStatus(goldStyle, "The lab made its forecast.")
		}
	}
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		m.villageStage = villageList
		return nil
	}
	m.villageStage = villageList
	m.dirty = true
	return nil
}

// forecastLines draws a forecast for the village screen.
func forecastLines(g *Garden, a, b int, kids []Genome) []string {
	pa, pb := &g.Plots[a], &g.Plots[b]
	lines := []string{labelStyle.Render("FORECAST"),
		subtleStyle.Render(fmt.Sprintf("  bed %d × bed %d, %s; %d seedlings you might expect", a+1, b+1, pa.Species().Common, len(kids)))}
	_ = pb
	for _, k := range kids {
		lines = append(lines, "  "+swatch(k.Hex(), 2)+" "+valueStyle.Render(describeGenome(k))+
			subtleStyle.Render(fmt.Sprintf("  shape %d · speed %d · yield %d", k.Shape, k.Speed, k.Yield)))
	}
	return lines
}

// actVillageDismiss lets the highlighted helper go.
func (m *model) actVillageDismiss() tcmd {
	if m.villageTab != tabHelp || m.villageStage != villageList || m.villageCursor >= len(helperDefs) {
		return nil
	}
	d := helperDefs[m.villageCursor]
	if err := m.g.DismissHelper(d.ID, m.now); err != nil {
		m.setStatus(subtleStyle, "%s is not on the payroll.", d.Name)
		return nil
	}
	m.dirty = true
	m.setStatus(subtleStyle, "You thanked %s and let them go.", d.Name)
	return nil
}
