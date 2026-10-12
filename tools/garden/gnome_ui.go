package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// gnomeStage is how far into a purchase the gardener has got at the stall.
type gnomeStage int

const (
	gnomeMenu   gnomeStage = iota // choosing what to buy
	gnomeChoose                   // choosing a sky, or a mutation
	gnomeBed                      // choosing the plant to work on
)

// gnomeRow is one line of the stall's current list.
type gnomeRow struct {
	Label string
	Note  string
	OK    bool // false when it cannot be had right now
}

// openGnomes walks to the stall.
func (m *model) openGnomes() {
	m.screen, m.gnomeStage, m.gnomeCursor = screenGnomes, gnomeMenu, 0
}

// gnomeBeds lists the beds with a plant the gnomes could work on.
func (m model) gnomeBeds() []int {
	var out []int
	for i := range m.g.Plots {
		if !m.g.Plots[i].Empty() && m.g.Plots[i].Species() != nil {
			out = append(out, i)
		}
	}
	return out
}

// gnomeRows is the list on the left of the stall: services, then the options
// of one, then the plants to work on.
func (m model) gnomeRows() []gnomeRow {
	var rows []gnomeRow
	switch m.gnomeStage {
	case gnomeMenu:
		for s := GnomeService(0); s < serviceCount; s++ {
			cost, ok := m.gnomeCost(s)
			rows = append(rows, gnomeRow{Label: gnomeInfos[s].Name, Note: cost, OK: ok})
		}
	case gnomeChoose:
		if m.gnomeService == ServiceWeather {
			now := m.g.Weather(m.now)
			for _, k := range allWeatherKinds {
				w := weatherFor(k)
				note := ""
				if k == now.Kind {
					note = "now"
				}
				rows = append(rows, gnomeRow{Label: w.Glyph() + " " + w.Name(), Note: note, OK: k != now.Kind})
			}
			break
		}
		for _, mu := range gnomeMutations {
			rows = append(rows, gnomeRow{Label: mu.Name, OK: true})
		}
	default:
		for _, i := range m.gnomeBeds() {
			p := &m.g.Plots[i]
			rows = append(rows, gnomeRow{Label: truncate(p.DisplayName(), 20), Note: bedLabel(i), OK: true})
		}
	}
	return rows
}

func bedLabel(i int) string { return "bed " + strconv.Itoa(i+1) }

// gnomeCost is a service's price as the menu shows it, and whether it can be had.
func (m model) gnomeCost(s GnomeService) (string, bool) {
	switch s {
	case ServiceWeather:
		return strconv.Itoa(gnomeWeatherCost) + "g", m.g.Gold >= gnomeWeatherCost
	case ServiceMutate:
		return strconv.Itoa(gnomeMutateCost) + "g", m.g.Gold >= gnomeMutateCost
	case ServiceChosen:
		return strconv.Itoa(gnomeChosenCost) + "g", m.g.Gold >= gnomeChosenCost
	}
	w := m.g.WildSeedFor(dayKey(m.now))
	if !m.g.WildSeedAvailable(m.now) {
		return "sold", false
	}
	return strconv.Itoa(w.Price) + "g", m.g.Gold >= w.Price
}

func (m *model) gnomeMove(delta int) {
	m.gnomeCursor = max(0, min(len(m.gnomeRows())-1, m.gnomeCursor+delta))
}

// gnomeBack steps back one stage, and from the front of the stall, out of it.
func (m *model) gnomeBack() {
	switch m.gnomeStage {
	case gnomeBed:
		if m.gnomeService == ServiceChosen {
			m.gnomeStage, m.gnomeCursor = gnomeChoose, m.gnomeOption
		} else {
			m.gnomeStage, m.gnomeCursor = gnomeMenu, int(m.gnomeService)
		}
	case gnomeChoose:
		m.gnomeStage, m.gnomeCursor = gnomeMenu, int(m.gnomeService)
	default:
		m.screen = screenGarden
	}
}

// gnomeSelect acts on the highlighted row.
func (m *model) gnomeSelect() tea.Cmd {
	rows := m.gnomeRows()
	if len(rows) == 0 {
		m.setStatus(warnStyle, "You have nothing planted for the gnomes to work on.")
		return nil
	}
	m.gnomeCursor = min(m.gnomeCursor, len(rows)-1)
	switch m.gnomeStage {
	case gnomeMenu:
		m.gnomeService = GnomeService(m.gnomeCursor)
		switch m.gnomeService {
		case ServiceWeather, ServiceChosen:
			m.gnomeStage, m.gnomeCursor = gnomeChoose, 0
		case ServiceMutate:
			if len(m.gnomeBeds()) == 0 {
				m.setStatus(warnStyle, "You have nothing planted for the gnomes to work on.")
				return nil
			}
			m.gnomeStage, m.gnomeCursor = gnomeBed, 0
		default:
			return m.gnomeBuySeed()
		}
	case gnomeChoose:
		if m.gnomeService == ServiceWeather {
			return m.gnomeTurnWeather(allWeatherKinds[m.gnomeCursor])
		}
		if len(m.gnomeBeds()) == 0 {
			m.setStatus(warnStyle, "You have nothing planted for the gnomes to work on.")
			return nil
		}
		m.gnomeOption = m.gnomeCursor
		m.gnomeStage, m.gnomeCursor = gnomeBed, 0
	default:
		return m.gnomeWork(m.gnomeBeds()[m.gnomeCursor])
	}
	return nil
}

func (m *model) gnomeTurnWeather(kind WeatherKind) tea.Cmd {
	if err := m.g.ChangeWeather(kind, m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.gnomeStage, m.gnomeCursor = gnomeMenu, int(ServiceWeather)
	m.setStatus(goldStyle, "The gnomes chant, and the sky turns: %s for the rest of today.", weatherFor(kind).Name())
	return m.save()
}

func (m *model) gnomeBuySeed() tea.Cmd {
	w, err := m.g.BuyWildSeed(m.now)
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.setStatus(goldStyle, "A wild %s seed is in your shed. Sow it and see.", w.Species.Common)
	return m.save()
}

func (m *model) gnomeWork(bed int) tea.Cmd {
	name := m.g.Plots[bed].DisplayName()
	var said string
	var err error
	if m.gnomeService == ServiceMutate {
		said, err = m.g.MutateRandom(bed, m.now)
	} else {
		said, err = m.g.MutateChosen(bed, gnomeMutations[m.gnomeOption].ID, m.now)
	}
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.refreshAlmanac()
	if m.gnomeService == ServiceMutate {
		m.setStatus(goldStyle, "Sparks! %s changed: %s.", name, said)
	} else {
		m.setStatus(goldStyle, "The gnomes lean on %s, and it comes out %s.", name, lower(said))
	}
	return m.save()
}

func lower(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 32
	}
	return string(r)
}

func gnomeBindings() []binding {
	var list []binding
	add := func(set keySet, keys []string, label, desc, hint string, do func(m *model) tea.Cmd) {
		list = append(list, binding{Set: set, Keys: keys, Label: label, Desc: desc, Hint: hint, Do: do})
	}
	k := func(keys ...string) []string { return keys }

	add(setGarden, k("N"), "N", "the gnome shop: change the weather, mutate a plant, buy the day's wild seed (tab reaches it too)", "N gnomes", func(m *model) tea.Cmd {
		m.openGnomes()
		return nil
	})
	add(setGnomes, k("up", "k"), "↑↓ / jk", "browse the stall", "↑↓ browse", func(m *model) tea.Cmd { m.gnomeMove(-1); return nil })
	add(setGnomes, k("down", "j"), "", "", "", func(m *model) tea.Cmd { m.gnomeMove(1); return nil })
	add(setGnomes, k("enter", " "), "enter / space", "buy the highlighted service, or choose it and go on: a sky, then a mutation, then the plant to work it on", "enter choose", func(m *model) tea.Cmd { return m.gnomeSelect() })
	add(setGnomes, k("esc"), "esc", "step back; from the front of the stall, back to the garden", "esc back", func(m *model) tea.Cmd {
		m.gnomeBack()
		return nil
	})
	return list
}
