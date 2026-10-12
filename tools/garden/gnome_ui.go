package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// gnomeStage is how far into a purchase the gardener has got at the stall.
type gnomeStage int

const (
	gnomeMenu   gnomeStage = iota // choosing what to buy
	gnomeChoose                   // choosing a sky, a season, a mutation or a recipe
	gnomeBed                      // choosing the plant or bed to work on (the first flower, for a match)
	gnomeBed2                     // choosing the second flower of a match
)

// gnomeRow is one line of the stall's current list.
type gnomeRow struct {
	Label string
	Note  string
	OK    bool // false when it cannot be had right now
}

var allSeasons = []Season{Spring, Summer, Autumn, Winter}

// needsChoice reports whether a service asks for a choice before a bed.
func needsChoice(s GnomeService) bool {
	return s == ServiceWeather || s == ServiceSeason || s == ServiceChosen || s == ServiceSoil
}

// choosesABed reports whether a service ends at a bed rather than at the choice.
func choosesABed(s GnomeService) bool {
	return s != ServiceWeather && s != ServiceSeason && s != ServiceSeed
}

// openGnomes walks to the stall.
func (m *model) openGnomes() {
	m.screen, m.gnomeStage, m.gnomeCursor = screenGnomes, gnomeMenu, 0
}

// gnomeBeds lists the beds the current service could be worked on.
func (m model) gnomeBeds() []int {
	var out []int
	for i := range m.g.Plots {
		ok := false
		switch m.gnomeService {
		case ServiceGrowth:
			ok = m.g.Growable(i)
		case ServiceSoil:
			ok = m.g.Treatable(i)
		case ServiceMatch:
			ok = m.g.Matchable(i)
			if ok && m.gnomeStage == gnomeBed2 {
				_, kind := MatchCost(m.g.Plots[m.gnomeMother].Species(), m.g.Plots[i].Species())
				ok = i != m.gnomeMother && kind != MatchNever
			}
		default:
			ok = !m.g.Plots[i].Empty() && m.g.Plots[i].Species() != nil
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

// gnomeNothing is what the stall says when there is nothing to work on.
func (m model) gnomeNothing() string {
	switch m.gnomeService {
	case ServiceGrowth:
		return "Nothing planted has a stage left to climb."
	case ServiceMatch:
		if m.gnomeStage == gnomeBed2 {
			return "No other flower in bloom will do."
		}
		return "The gnomes need two flowers in bloom."
	case ServiceSoil:
		return "There is no soil to stir."
	}
	return "You have nothing planted for the gnomes to work on."
}

// gnomeChoices is the list of a choosing stage, as rows.
func (m model) gnomeChoices() []gnomeRow {
	var rows []gnomeRow
	switch m.gnomeService {
	case ServiceWeather:
		now := m.g.Weather(m.now)
		for _, k := range allWeatherKinds {
			w := weatherFor(k)
			note := ""
			if k == now.Kind {
				note = "now"
			}
			rows = append(rows, gnomeRow{Label: w.Glyph() + " " + w.Name(), Note: note, OK: k != now.Kind})
		}
	case ServiceSeason:
		now := m.g.Season(m.now)
		for _, s := range allSeasons {
			note := ""
			if s == now {
				note = "now"
			}
			rows = append(rows, gnomeRow{Label: s.Glyph() + " " + s.String(), Note: note, OK: s != now})
		}
	case ServiceSoil:
		for _, tr := range gnomeTreatments {
			rows = append(rows, gnomeRow{Label: tr.Name, OK: true})
		}
	default:
		for _, mu := range gnomeMutations {
			rows = append(rows, gnomeRow{Label: mu.Name, OK: true})
		}
	}
	return rows
}

// gnomeRows is the list on the left of the stall: services, then the options
// of one, then the beds to work them on.
func (m model) gnomeRows() []gnomeRow {
	var rows []gnomeRow
	switch m.gnomeStage {
	case gnomeMenu:
		for s := GnomeService(0); s < serviceCount; s++ {
			cost, ok := m.gnomeCost(s)
			rows = append(rows, gnomeRow{Label: gnomeInfos[s].Name, Note: cost, OK: ok})
		}
	case gnomeChoose:
		rows = m.gnomeChoices()
	default:
		for _, i := range m.gnomeBeds() {
			p := &m.g.Plots[i]
			label, note := truncate(p.DisplayName(), 20), "bed "+strconv.Itoa(i+1)
			if p.Empty() {
				label = "bare soil"
			}
			if m.gnomeStage == gnomeBed2 {
				cost, _ := MatchCost(m.g.Plots[m.gnomeMother].Species(), p.Species())
				note = strconv.Itoa(cost) + "g"
			}
			rows = append(rows, gnomeRow{Label: label, Note: note, OK: true})
		}
	}
	return rows
}

// gnomeCost is a service's price as the menu shows it, and whether it can be had.
func (m model) gnomeCost(s GnomeService) (string, bool) {
	price := map[GnomeService]int{
		ServiceWeather: gnomeWeatherCost, ServiceSeason: gnomeSeasonCost, ServiceGrowth: gnomeGrowthCost,
		ServiceMutate: gnomeMutateCost, ServiceChosen: gnomeChosenCost, ServiceMatch: gnomeMatchSame,
		ServiceSoil: gnomeSoilCost,
	}
	if s == ServiceSeed {
		if !m.g.WildSeedAvailable(m.now) {
			return "sold", false
		}
		p := m.g.WildSeedFor(dayKey(m.now)).Price
		return strconv.Itoa(p) + "g", m.g.Gold >= p
	}
	label := strconv.Itoa(price[s]) + "g"
	if s == ServiceMatch {
		label += "+"
	}
	return label, m.g.Gold >= price[s]
}

func (m *model) gnomeMove(delta int) {
	m.gnomeCursor = max(0, min(len(m.gnomeRows())-1, m.gnomeCursor+delta))
}

// gnomeBack steps back one stage, and from the front of the stall, out of it.
func (m *model) gnomeBack() {
	switch m.gnomeStage {
	case gnomeBed2:
		m.gnomeStage = gnomeBed
		m.gnomeCursor = 0
		for i, b := range m.gnomeBeds() {
			if b == m.gnomeMother {
				m.gnomeCursor = i
			}
		}
	case gnomeBed:
		if needsChoice(m.gnomeService) {
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
		m.setStatus(warnStyle, "%s", m.gnomeNothing())
		return nil
	}
	m.gnomeCursor = min(m.gnomeCursor, len(rows)-1)
	switch m.gnomeStage {
	case gnomeMenu:
		m.gnomeService = GnomeService(m.gnomeCursor)
		switch {
		case m.gnomeService == ServiceSeed:
			return m.gnomeBuySeed()
		case needsChoice(m.gnomeService):
			m.gnomeStage, m.gnomeCursor = gnomeChoose, 0
		default:
			return m.gnomeToBeds()
		}
	case gnomeChoose:
		m.gnomeOption = m.gnomeCursor
		switch m.gnomeService {
		case ServiceWeather:
			return m.gnomeTurnWeather(allWeatherKinds[m.gnomeOption])
		case ServiceSeason:
			return m.gnomeTurnSeason(allSeasons[m.gnomeOption])
		}
		return m.gnomeToBeds()
	case gnomeBed:
		bed := m.gnomeBeds()[m.gnomeCursor]
		if m.gnomeService == ServiceMatch {
			m.gnomeMother, m.gnomeStage, m.gnomeCursor = bed, gnomeBed2, 0
			if len(m.gnomeBeds()) == 0 {
				m.setStatus(warnStyle, "%s", m.gnomeNothing())
				m.gnomeStage = gnomeBed
			}
			return nil
		}
		return m.gnomeWork(bed)
	default:
		return m.gnomeMatch(m.gnomeBeds()[m.gnomeCursor])
	}
	return nil
}

// gnomeToBeds moves on to choosing a bed, if there is one to choose.
func (m *model) gnomeToBeds() tea.Cmd {
	m.gnomeStage = gnomeBed
	if len(m.gnomeBeds()) == 0 {
		m.setStatus(warnStyle, "%s", m.gnomeNothing())
		m.gnomeBack()
		return nil
	}
	m.gnomeCursor = 0
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

func (m *model) gnomeTurnSeason(season Season) tea.Cmd {
	if err := m.g.ChangeSeason(season, m.now); err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.refreshShop()
	m.gnomeStage, m.gnomeCursor = gnomeMenu, int(ServiceSeason)
	m.setStatus(goldStyle, "The gnomes unroll a borrowed %s over the garden for the rest of today.", season)
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

func (m *model) gnomeMatch(father int) tea.Cmd {
	mother := m.gnomeMother
	mn, fn := m.g.Plots[mother].DisplayName(), m.g.Plots[father].DisplayName()
	cost, err := m.g.Matchmake(mother, father, m.now)
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.gnomeStage, m.gnomeCursor = gnomeBed, 0
	m.setStatus(goldStyle, "Madame Bumblewick whispers to %s about %s (%s). Gather its seed (f) to see the cross.", mn, fn, goldLabel(cost))
	return m.save()
}

func (m *model) gnomeWork(bed int) tea.Cmd {
	name := m.g.Plots[bed].DisplayName()
	var said string
	var err error
	switch m.gnomeService {
	case ServiceGrowth:
		said, err = m.g.GrowthSpurt(bed, m.now)
	case ServiceMutate:
		said, err = m.g.MutateRandom(bed, m.now)
	case ServiceSoil:
		said, err = m.g.TreatSoil(bed, gnomeTreatments[m.gnomeOption].ID, m.now)
	default:
		said, err = m.g.MutateChosen(bed, gnomeMutations[m.gnomeOption].ID, m.now)
	}
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	}
	m.dirty = true
	m.refreshAlmanac()
	switch m.gnomeService {
	case ServiceGrowth:
		m.setStatus(goldStyle, "The gnomes sing, and %s goes from %s.", name, said)
	case ServiceMutate:
		m.setStatus(goldStyle, "Sparks! %s changed: %s.", name, said)
	case ServiceSoil:
		m.setStatus(goldStyle, "The gnomes stir bed %d: %s.", bed+1, said)
	default:
		m.setStatus(goldStyle, "The gnomes lean on %s, and it comes out %s.", name, lower(said))
	}
	m.gnomeMove(0)
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

	add(setGarden, k("N"), "N", "the gnome shop: weather, seasons, growth, mutations, matches, soil and a wild seed (tab reaches it too)", "N gnomes", func(m *model) tea.Cmd {
		m.openGnomes()
		return nil
	})
	add(setGnomes, k("up", "k"), "↑↓ / jk", "browse the stall", "↑↓ browse", func(m *model) tea.Cmd { m.gnomeMove(-1); return nil })
	add(setGnomes, k("down", "j"), "", "", "", func(m *model) tea.Cmd { m.gnomeMove(1); return nil })
	add(setGnomes, k("enter", " "), "enter / space", "buy the highlighted service, or choose it and go on: a sky, season, mutation or recipe, then the plant or bed to work it on (a match asks for two flowers)", "enter choose", func(m *model) tea.Cmd { return m.gnomeSelect() })
	add(setGnomes, k("esc"), "esc", "step back; from the front of the stall, back to the garden", "esc back", func(m *model) tea.Cmd {
		m.gnomeBack()
		return nil
	})
	return list
}
