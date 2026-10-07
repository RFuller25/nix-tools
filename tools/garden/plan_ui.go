package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// The planner's keys, screens and drawing.

// planBindings are the bindings for plan mode, stamping, selecting and the
// layouts screen, plus the garden keys that enter them.
func planBindings() []binding {
	var list []binding
	add := func(set keySet, keys []string, label, desc, hint string, do func(m *model) tea.Cmd) {
		list = append(list, binding{Set: set, Keys: keys, Label: label, Desc: desc, Hint: hint, Do: do})
	}
	k := func(keys ...string) []string { return keys }

	// moves is the four directions, for every mode that moves the cursor.
	moves := func(set keySet, desc, hint string) {
		step := func(dc, dr int) func(m *model) tea.Cmd {
			return func(m *model) tea.Cmd {
				col, row := m.cursor%plotCols+dc, m.cursor/plotCols+dr
				if idx := row*plotCols + col; col >= 0 && col < plotCols && row >= 0 && idx < len(m.g.Plots) {
					m.cursor = idx
				}
				m.afterMove()
				return nil
			}
		}
		add(set, k("left", "h"), "←↑↓→ / hjkl", desc, hint, step(-1, 0))
		add(set, k("right", "l"), "", "", "", step(1, 0))
		add(set, k("up", "k"), "", "", "", step(0, -1))
		add(set, k("down", "j"), "", "", "", step(0, 1))
	}

	// ---- from the garden ------------------------------------------------
	add(setGarden, k("P"), "P", "plan mode: lay ghosts of your seed over the beds and see how the layout would get on before sowing anything", "P plan", func(m *model) tcmd {
		m.startPlan()
		return nil
	})
	add(setGarden, k("V"), "V", "select a block of beds, then save what is growing in them as a layout", "V select", func(m *model) tcmd {
		m.startSelect()
		return nil
	})
	add(setGarden, k("T"), "T", "layouts: the built-in ones and your own, to stamp down anywhere", "T layouts", func(m *model) tcmd {
		m.screen, m.tplCursor = screenTemplates, 0
		return nil
	})

	// ---- plan mode ------------------------------------------------------
	moves(setPlan, "move to a bed", "←↑↓→ move")
	add(setPlan, k("["), "[ / ]", "choose which seed to place: yours first, then anything the shop sells", "[ ] pick seed", func(m *model) tcmd { m.planStep(-1); return nil })
	add(setPlan, k("]"), "", "", "", func(m *model) tcmd { m.planStep(1); return nil })
	add(setPlan, k("{"), "{ / }", "jump ten along the seed list", "", func(m *model) tcmd { m.planStep(-10); return nil })
	add(setPlan, k("}"), "", "", "", func(m *model) tcmd { m.planStep(10); return nil })
	add(setPlan, k("/"), "slash", "search the seed list by name, kind or family (empty clears it)", "/ search", func(m *model) tcmd { m.planSearch(); return nil })
	add(setPlan, k("enter", " ", "p"), "enter / space / p", "place the chosen seed in this bed as a ghost", "enter place", func(m *model) tcmd { return m.planPlace() })
	add(setPlan, k("backspace", "x", "delete"), "backspace / x / delete", "take the ghost out of this bed", "x remove", func(m *model) tcmd { return m.planRemove() })
	add(setPlan, k("C"), "C", "sow the whole plan: your own seed where you have it, bought seed where you do not", "C sow it", func(m *model) tcmd { return m.planCommit() })
	add(setPlan, k("esc"), "esc", "leave plan mode without sowing anything", "esc cancel", func(m *model) tcmd { m.endMode("Plan put away."); return nil })

	// ---- stamping -------------------------------------------------------
	moves(setStamp, "move the layout over the garden", "←↑↓→ move")
	add(setStamp, k("enter", " "), "enter / space", "sow the layout here: your own seed where you have it, bought seed where you do not", "enter sow", func(m *model) tcmd { return m.planCommit() })
	add(setStamp, k("esc"), "esc", "put the layout away", "esc cancel", func(m *model) tcmd { m.endMode("Layout put away."); return nil })

	// ---- selecting ------------------------------------------------------
	moves(setSelect, "stretch the selection", "←↑↓→ stretch")
	add(setSelect, k("enter", " "), "enter / space", "save what is growing in the selected beds as a layout, and name it", "enter save", func(m *model) tcmd { return m.selectFinish() })
	add(setSelect, k("esc"), "esc", "cancel", "esc cancel", func(m *model) tcmd { m.endMode("Selection dropped."); return nil })

	// ---- orders ---------------------------------------------------------
	add(setGarden, k("o"), "o", "fill an order with the plant in this bed, if it fits one: the plant is used up and you are paid", "o order", func(m *model) tcmd { return m.actDeliverHere() })
	add(setGarden, k("O"), "O", "the order board", "", func(m *model) tcmd {
		m.screen, m.ordersCursor = screenOrders, 0
		return nil
	})
	add(setOrders, k("up", "k"), "↑↓ / jk", "browse orders", "↑↓ browse", func(m *model) tcmd {
		m.ordersCursor = max(0, m.ordersCursor-1)
		return nil
	})
	add(setOrders, k("down", "j"), "", "", "", func(m *model) tcmd {
		m.ordersCursor = min(len(m.g.Orders)-1, m.ordersCursor+1)
		return nil
	})
	add(setOrders, k("enter", "o", " "), "enter / o / space", "deliver the first of your plants that fits the order; it is used up and you are paid", "enter deliver", func(m *model) tcmd { return m.actDeliverOrder() })

	// ---- the fair -------------------------------------------------------
	add(setGarden, k("e"), "e", "enter the plant in this bed in this week's show", "e show", func(m *model) tcmd { return m.actEnterHere() })
	add(setGarden, k("E"), "E", "the fair: this week's class, your entry and your ribbons", "", func(m *model) tcmd {
		m.screen, m.fairCursor = screenFair, 0
		return nil
	})
	add(setFair, k("up", "k"), "↑↓ / jk", "choose a plant", "↑↓ choose", func(m *model) tcmd {
		m.fairCursor = max(0, m.fairCursor-1)
		return nil
	})
	add(setFair, k("down", "j"), "", "", "", func(m *model) tcmd {
		m.fairCursor = min(len(m.fairCandidates())-1, m.fairCursor+1)
		return nil
	})
	add(setFair, k("enter", "e", " "), "enter / e / space", "enter the chosen plant in this week's show, replacing any earlier entry", "enter show it", func(m *model) tcmd { return m.actEnterFair() })

	// ---- the layouts screen --------------------------------------------
	add(setTpl, k("up", "k"), "↑↓ / jk", "browse layouts", "↑↓ browse", func(m *model) tcmd {
		m.tplCursor = max(0, m.tplCursor-1)
		return nil
	})
	add(setTpl, k("down", "j"), "", "", "", func(m *model) tcmd {
		m.tplCursor = min(len(m.g.AllTemplates())-1, m.tplCursor+1)
		return nil
	})
	add(setTpl, k("enter", "p", " "), "enter / p / space", "place the layout: choose where in the garden it goes, then enter again", "enter place", func(m *model) tcmd { return m.stampBegin() })
	add(setTpl, k("x", "delete", "backspace"), "x / delete / backspace", "delete a layout of your own (the built-in ones stay)", "x delete", func(m *model) tcmd { return m.templateDelete() })
	return list
}

type tcmd = tea.Cmd

// ---- state changes ----------------------------------------------------------

func (m *model) startPlan() {
	m.mode, m.ghosts, m.planPick = modePlan, map[int]ghost{}, 0
	m.rescore()
	m.setStatus(goldStyle, "Plan mode: pick a seed with [ ] (or / to search), move to a bed, press enter to place it. Nothing is sown or paid for until C.")
}

func (m *model) startSelect() {
	m.mode, m.selAnchor = modeSelect, m.cursor
	m.setStatus(goldStyle, "Selecting: stretch the block with the arrow keys over what you want to keep, then enter.")
}

func (m *model) endMode(msg string) {
	m.mode, m.ghosts, m.plan = modeNone, nil, nil
	m.stampSkipped = nil
	m.setStatus(subtleStyle, "%s", msg)
}

// afterMove keeps a stamped layout under the cursor.
func (m *model) afterMove() {
	if m.mode == modeStamp {
		m.restamp()
	}
}

// rescore rebuilds the garden-with-ghosts the planner scores against.
func (m *model) rescore() { m.plan = m.g.withGhosts(m.ghosts) }

// paletteEntry is one seed plan mode can place.
type paletteEntry struct {
	Label  string
	Ghost  ghost
	Shed   bool
	Remain int
	Locked int // matured plants still needed before the shop sells it, or 0
	Price  int
}

// tag is the short note on where the seed would come from.
func (e paletteEntry) tag() string {
	switch {
	case e.Shed:
		return fmt.Sprintf("yours ×%d", e.Remain)
	case e.Locked > 0:
		return fmt.Sprintf("locked: %d plants to grow first", e.Locked)
	}
	return fmt.Sprintf("shop %dg", e.Price)
}

// palette is every seed plan mode can place: your own packets first, then every
// species in the almanac. Species the shop does not stock yet can still be
// planned with; they are left out when the plan is sown.
func (m *model) palette() []paletteEntry {
	used := map[int64]int{}
	for _, gh := range m.ghosts {
		if gh.Packet != 0 {
			used[gh.Packet]++
		}
	}
	filter := strings.ToLower(strings.TrimSpace(m.planFilter))
	match := func(parts ...string) bool {
		if filter == "" {
			return true
		}
		for _, p := range parts {
			if strings.Contains(strings.ToLower(p), filter) {
				return true
			}
		}
		return false
	}
	var out []paletteEntry
	for _, i := range m.g.ShedOrder() {
		pk := m.g.Shed[i]
		sp := pk.Species()
		if sp == nil || !match(pk.Name(), sp.Common, sp.Latin) {
			continue
		}
		out = append(out, paletteEntry{
			Label: pk.Name(), Shed: true, Remain: pk.Count - used[pk.ID],
			Ghost: ghost{Species: pk.SpeciesID, Variety: pk.Variety, Packet: pk.ID, Genome: pk.Mean()},
		})
	}
	// What the shop sells now comes before what it does not stock yet.
	var later []paletteEntry
	for _, sp := range AllSpecies() {
		if !match(sp.Common, sp.Latin, sp.Kind.String(), sp.Family) {
			continue
		}
		e := paletteEntry{
			Label: sp.Common, Price: sp.seedPrice(),
			Ghost: ghost{Species: sp.ID, Genome: sp.VarietyGenome(0)},
		}
		if !m.g.Unlocked(sp) {
			e.Locked = sp.Unlock - m.g.Matured
			later = append(later, e)
			continue
		}
		out = append(out, e)
	}
	return append(out, later...)
}

func (m *model) planStep(delta int) {
	n := len(m.palette())
	if n == 0 {
		return
	}
	m.planPick = ((m.planPick+delta)%n + n) % n
}

// planSearch asks for a name to narrow the seed list to.
func (m *model) planSearch() {
	m.naming, m.namingFor = true, namePlanFilter
	m.input.Placeholder = "part of a name, e.g. rose, herb, Lamiaceae"
	m.input.SetValue(m.planFilter)
	m.input.CursorEnd()
	m.input.Focus()
}

// trayRows is how many extra rows the garden screen carries for the seed tray.
func (m model) trayRows() int {
	if m.screen == screenGarden && (m.mode == modePlan || m.mode == modeStamp || m.mode == modeSelect) {
		return 1
	}
	return 0
}

// tray is the row above the footer while a mode is active: for plan mode, the
// seeds you can place, with the chosen one in the middle.
func (m model) tray() string {
	if m.trayRows() == 0 {
		return ""
	}
	switch m.mode {
	case modePlan:
		pal := m.palette()
		if len(pal) == 0 {
			return goldStyle.Render("no seed matches “" + m.planFilter + "” — / to search again")
		}
		pick := min(m.planPick, len(pal)-1)
		one := func(i int, chosen bool) string {
			e := pal[((i%len(pal))+len(pal))%len(pal)]
			sw := swatch(e.Ghost.Genome.Hex(), 1)
			text := e.Label
			style := subtleStyle
			if e.Locked > 0 {
				style = lockedStyle
			}
			if chosen {
				return sw + " " + titleStyle.Render("‹ "+text+" · "+e.tag()+" ›")
			}
			return sw + " " + style.Render(truncate(text, 14))
		}
		var parts []string
		if len(pal) > 1 {
			parts = append(parts, one(pick-2, false), one(pick-1, false))
		}
		parts = append(parts, one(pick, true))
		if len(pal) > 2 {
			parts = append(parts, one(pick+1, false), one(pick+2, false))
		}
		filter := ""
		if m.planFilter != "" {
			filter = subtleStyle.Render("  [" + m.planFilter + "]")
		}
		return fit(strings.Join(parts, subtleStyle.Render(" · "))+filter+subtleStyle.Render(fmt.Sprintf("  %d/%d", pick+1, len(pal))), m.width)
	case modeStamp:
		return fit(goldStyle.Render("placing ‘"+m.stampTpl.Name+"’")+subtleStyle.Render(" — move it with the arrows; it is scored as you go"), m.width)
	case modeSelect:
		r := rectOf(m.selAnchor, m.cursor)
		return fit(goldStyle.Render(fmt.Sprintf("selecting %d × %d beds", r.c1-r.c0+1, r.r1-r.r0+1))+subtleStyle.Render(" — stretch with the arrows over what to keep"), m.width)
	}
	return ""
}

func (m *model) planPlace() tcmd {
	pal := m.palette()
	if len(pal) == 0 {
		m.setStatus(warnStyle, "No seed to place. Press / to search again.")
		return nil
	}
	m.planPick = min(m.planPick, len(pal)-1)
	e := pal[m.planPick]
	sp := e.Ghost.species()
	switch err := m.g.checkSowable(m.cursor, sp); {
	case err != nil:
		m.setStatus(warnStyle, "%s", err.Error())
		return nil
	case e.Shed && e.Remain < 1:
		m.setStatus(warnStyle, "You have no more of that seed to place; pick another with [ ].")
		return nil
	}
	m.ghosts[m.cursor] = e.Ghost
	m.rescore()
	net := m.plan.Synergy(m.cursor).Net()
	cost, _, locked := m.g.PlanCost(m.ghosts)
	note := fmt.Sprintf("%d placed, %s to buy", len(m.ghosts), goldLabel(cost))
	if locked > 0 {
		note += fmt.Sprintf(", %d not in the shop yet", locked)
	}
	m.setStatus(subtleStyle, "%s in bed %d: %s (%+.0f%% growth). %s. C sows.", e.Label, m.cursor+1, SynergyWord(net), net*100, note)
	return nil
}

func (m *model) planRemove() tcmd {
	if _, ok := m.ghosts[m.cursor]; ok {
		delete(m.ghosts, m.cursor)
		m.rescore()
	}
	return nil
}

// planCommit sows the ghosts. Whatever cannot be sown (seed the shop does not
// stock yet, or that you cannot afford) is left out and reported.
func (m *model) planCommit() tcmd {
	if len(m.ghosts) == 0 {
		m.setStatus(subtleStyle, "Nothing is planned yet: pick a seed with [ ] and press enter on a bed.")
		return nil
	}
	sown, bought, spent, problems := m.g.CommitGhosts(m.ghosts, m.now)
	if sown == 0 {
		m.setStatus(warnStyle, "Nothing could be sown: %s", firstOr(problems, "no seed to hand"))
		return nil
	}
	m.dirty = true
	msg := fmt.Sprintf("Sowed %d bed(s)", sown)
	if bought > 0 {
		msg += fmt.Sprintf(", buying %d seed(s) for %s", bought, goldLabel(spent))
	}
	if len(problems) > 0 {
		msg += fmt.Sprintf(". Left out %d: %s", len(problems), problems[0])
	}
	m.endMode(msg + ".")
	m.screen = screenGarden
	return m.save()
}

func firstOr(list []string, def string) string {
	if len(list) == 0 {
		return def
	}
	return list[0]
}

// ---- stamping -----------------------------------------------------------------

func (m *model) stampBegin() tcmd {
	all := m.g.AllTemplates()
	if m.tplCursor < 0 || m.tplCursor >= len(all) {
		return nil
	}
	m.stampTpl = all[m.tplCursor]
	m.mode, m.screen = modeStamp, screenGarden
	m.restamp()
	m.setStatus(goldStyle, "Placing ‘%s’: move it over the garden and press enter.", m.stampTpl.Name)
	return nil
}

func (m *model) restamp() {
	m.ghosts, m.stampSkipped = m.g.StampPlan(m.stampTpl, m.cursor)
	m.rescore()
}

func (m *model) templateDelete() tcmd {
	all := m.g.AllTemplates()
	if m.tplCursor < 0 || m.tplCursor >= len(all) {
		return nil
	}
	t := all[m.tplCursor]
	if t.Builtin {
		m.setStatus(subtleStyle, "‘%s’ is built in; only layouts you saved can be deleted.", t.Name)
		return nil
	}
	m.g.DeleteTemplate(t.ID)
	m.dirty = true
	m.tplCursor = max(0, min(m.tplCursor, len(m.g.AllTemplates())-1))
	m.setStatus(subtleStyle, "Deleted ‘%s’.", t.Name)
	return nil
}

// ---- selecting and saving --------------------------------------------------------

func (m *model) selectFinish() tcmd {
	r := rectOf(m.selAnchor, m.cursor)
	planted := 0
	for idx := range m.g.Plots {
		if r.contains(idx) && !m.g.Plots[idx].Empty() {
			planted++
		}
	}
	if planted == 0 {
		m.setStatus(warnStyle, "There is nothing planted in those beds to save.")
		return nil
	}
	m.naming, m.namingFor = true, nameTemplate
	m.input.Placeholder = "a name for this layout"
	m.input.SetValue("")
	m.input.Focus()
	return nil
}

func (m *model) finishSaveTemplate(name string) {
	r := rectOf(m.selAnchor, m.cursor)
	m.naming = false
	m.input.Blur()
	t, err := m.g.SaveTemplate(name, r, m.now)
	if err != nil {
		m.setStatus(warnStyle, "%s", err.Error())
		return
	}
	m.dirty = true
	m.endMode(fmt.Sprintf("Saved ‘%s’: %d beds. Place it again with T.", t.Name, len(t.Cells)))
}

// ---- drawing --------------------------------------------------------------------

// synergyGarden is the garden synergy is scored against: the real one, or
// the one with the plan sown while planning.
func (m model) synergyGarden() *Garden {
	if m.plan != nil && (m.mode == modePlan || m.mode == modeStamp) {
		return m.plan
	}
	return m.g
}

// showSynergy reports whether beds are drawn with their neighbour scores.
func (m model) showSynergy() bool {
	return m.overlay || m.mode == modePlan || m.mode == modeStamp
}

// fadePalette greys a palette out, for plants that are only planned.
func fadePalette(p Palette, amount float64) Palette {
	grey := rgb{0.42, 0.44, 0.46}
	fade := func(c string) string {
		v, ok := parseColor(c)
		if !ok {
			return c
		}
		return blend(v, grey, amount).hex()
	}
	return Palette{Stem: fade(p.Stem), Leaf: fade(p.Leaf), Bloom: fade(p.Bloom), Accent: fade(p.Accent)}
}

// renderGhost draws a planned plant in a bed.
func (m model) renderGhost(idx int, gh ghost, selected bool) string {
	sp := gh.species()
	pal := fadePalette(tintPalette(sp.PaletteForGenome(gh.Variety, gh.Genome, nil), m.phase()), 0.45)
	lines := renderGene(sp, gh.Variety, gh.Genome, StageMature, pal, cellInner, artHeight, 0, nil)
	lines = append(lines, soilLine(cellInner, 0, true, m.phase(), m.g.Plots[idx].Richness))
	lines = append(lines, pad(subtleStyle.Render(truncate("▫ "+sp.Common, cellInner)), cellInner))
	src := "shop"
	if gh.Packet != 0 {
		src = "your seed"
	} else if !m.g.Unlocked(sp) {
		src = "locked"
	}
	badge := synergyBadge(m.synergyGarden().Synergy(idx).Net())
	lines = append(lines, pad(badge, cellInner))
	_ = src
	border := synergyBorder(m.synergyGarden().Synergy(idx).Net())
	if selected {
		border = selBorder
	}
	return border.Render(strings.Join(lines, "\n"))
}

// selectBorder frames the beds in a selection.
var selectBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6ea8fe"))

// modeLine is the summary shown in the footer while a mode is active and
// there is no message to show instead.
func (m model) modeLine() string {
	switch m.mode {
	case modePlan:
		cost, fromShed, locked := m.g.PlanCost(m.ghosts)
		line := fmt.Sprintf("plan: %d placed · %d from your seed · %s to buy", len(m.ghosts), fromShed, goldLabel(cost))
		if locked > 0 {
			line += fmt.Sprintf(" · %d not in the shop yet", locked)
		}
		return goldStyle.Render(line + " · C sows it")
	case modeStamp:
		cost, fromShed, _ := m.g.PlanCost(m.ghosts)
		line := fmt.Sprintf("‘%s’: %d beds · %d from your seed · %s to buy", m.stampTpl.Name, len(m.ghosts), fromShed, goldLabel(cost))
		if len(m.stampSkipped) > 0 {
			line += fmt.Sprintf(" · %d left out: %s", len(m.stampSkipped), m.stampSkipped[0])
		}
		return goldStyle.Render(line)
	case modeSelect:
		r := rectOf(m.selAnchor, m.cursor)
		return goldStyle.Render(fmt.Sprintf("selecting %d × %d beds · enter saves them as a layout", r.c1-r.c0+1, r.r1-r.r0+1))
	}
	return ""
}

// viewTemplates is the layouts screen: the list, and a preview of the one chosen.
func (m model) viewTemplates() string {
	all := m.g.AllTemplates()
	cursor := min(m.tplCursor, max(0, len(all)-1))
	listWidth := 28
	narrow := m.width-listWidth-6 < 30
	if narrow {
		listWidth = max(20, m.width-2)
	}
	var lines []string
	for i, t := range all {
		marker, style := "  ", valueStyle
		if i == cursor {
			marker, style = titleStyle.Render("› "), titleStyle
		}
		tag := ""
		if t.Builtin {
			tag = subtleStyle.Render(" ·")
		}
		lines = append(lines, marker+style.Render(truncate(t.Name, listWidth-6))+tag)
	}
	detail := ""
	if len(all) > 0 && !narrow {
		detail = m.templateDetail(all[cursor], m.width-listWidth-6, max(3, m.height-5))
	}
	body := strings.Join(lines, "\n")
	if detail != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "  ", detail)
	}
	head := fit(titleStyle.Render("▦ layouts")+subtleStyle.Render(fmt.Sprintf("  ·  %d built in, %d of your own", len(builtinTemplates()), len(m.g.Templates))), m.width)
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(m.hintLine(setTpl))}, "\n")
}

// templateDetail previews a layout as a little grid of colour chips and names.
func (m model) templateDetail(t Template, width, avail int) string {
	width = max(width, 30)
	cellW := max(6, min(12, (width-4)/max(1, t.W)-1))
	grid := make([][]string, t.H)
	for r := range grid {
		grid[r] = make([]string, t.W)
		for c := range grid[r] {
			grid[r][c] = pad(subtleStyle.Render("·"), cellW)
		}
	}
	cost, have := 0, 0
	need := map[string]int{}
	for _, c := range t.Cells {
		sp := SpeciesByID(c.Species)
		if sp == nil || c.DY >= t.H || c.DX >= t.W {
			continue
		}
		gn := sp.VarietyGenome(c.Variety)
		grid[c.DY][c.DX] = pad(swatch(gn.Hex(), 1)+" "+valueStyle.Render(truncate(sp.Common, cellW-2)), cellW)
		need[sp.ID]++
	}
	for id, n := range need {
		sp := SpeciesByID(id)
		held := 0
		for _, pk := range m.g.Shed {
			if pk.SpeciesID == id {
				held += pk.Count
			}
		}
		have += min(n, held)
		if n > held {
			cost += (n - held) * sp.seedPrice()
		}
	}
	rows := []string{titleStyle.Render(t.Name), subtleStyle.Render(fmt.Sprintf("%d × %d beds · %d planted", t.W, t.H, len(t.Cells))), ""}
	for _, r := range grid {
		rows = append(rows, strings.Join(r, " "))
	}
	rows = append(rows, "")
	if t.Note != "" {
		rows = append(rows, wrapped(subtleStyle, t.Note, width-2)...)
		rows = append(rows, "")
	}
	rows = append(rows, valueStyle.Render(fmt.Sprintf("%d of %d seeds already in your shed", have, len(t.Cells))),
		goldStyle.Render(fmt.Sprintf("%s to buy the rest from the shop", goldLabel(cost))))
	for _, c := range t.Cells {
		if c.Want != nil {
			dir := "lowest"
			if c.Want.High {
				dir = "highest"
			}
			rows = append(rows, subtleStyle.Render(fmt.Sprintf("Seed with the %s %s is used where it matters.", dir, c.Want.Trait)))
			break
		}
	}
	visible, _, _ := window(rows, 0, max(1, avail-2))
	return cardBorder.Width(width).Render(strings.Join(visible, "\n"))
}
