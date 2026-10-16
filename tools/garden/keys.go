package main

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Keys. Every key the garden answers to is declared once, here, as a binding:
// the keys, what they do, the short hint for the footer, and the function that
// does it. Dispatch, the footer hints, the help screen, the almanac's Keys
// chapter and the README's key table are all generated from this one table, so
// none of them can say a key does something it does not.

// keySet is a group of bindings that are live together: a screen, or a mode
// layered on top of one.
type keySet string

const (
	setGlobal  keySet = "global"
	setGarden  keySet = "garden"
	setShop    keySet = "shed"
	setMine    keySet = "mine"
	setPollen  keySet = "pollinate"
	setPlan    keySet = "plan"
	setStamp   keySet = "stamp"
	setSelect  keySet = "select"
	setTpl     keySet = "layouts"
	setOrders  keySet = "orders"
	setFair    keySet = "fair"
	setShare   keySet = "share"
	setGnomes  keySet = "gnomes"
	setInfo    keySet = "card"
	setAlmanac keySet = "almanac"
	setJournal keySet = "journal"
	setHelp    keySet = "help"
	setLineage keySet = "genetics"
	setDefine  keySet = "define"
)

// anyKey matches whatever was pressed, so a screen can close on any key that
// has no meaning of its own. It is never documented as a key.
const anyKey = "*"

type binding struct {
	Set   keySet
	Keys  []string
	Label string // how the keys read in the docs, e.g. "w / W"
	Desc  string // what they do; empty for bindings that are not documented
	Hint  string // short footer reminder; empty to leave it out
	Do    func(m *model) tea.Cmd
}

// setTitle is how a set is headed in the help screen and the docs.
var setTitles = map[keySet]string{
	setGlobal:  "anywhere",
	setGarden:  "in the garden",
	setShop:    "in the shed: the shop shelf",
	setMine:    "in the shed: my seeds shelf",
	setPollen:  "choosing a pollen donor (after x)",
	setPlan:    "in plan mode (P)",
	setStamp:   "placing a layout (T, then enter)",
	setSelect:  "selecting beds to save as a layout (V)",
	setTpl:     "on the layouts screen (T)",
	setOrders:  "on the order board",
	setFair:    "at the fair",
	setShare:   "on the share screen",
	setGnomes:  "at the gnome shop (N)",
	setInfo:    "on a plant's card",
	setAlmanac: "in the almanac",
	setJournal: "in the journal",
	setHelp:    "on this screen",
	setLineage: "on the genetics screen (t)",
	setDefine:  "defining a cultivar (D, on its almanac page)",
}

// setOrder is the order sets are listed in.
var setOrder = []keySet{setGarden, setPollen, setPlan, setStamp, setSelect, setTpl, setOrders, setFair, setGnomes, setShare, setShop, setMine, setInfo, setAlmanac, setDefine, setLineage, setJournal, setHelp, setGlobal}

var (
	bindingsOnce sync.Once
	bindingList  []binding
)

// allBindings is the whole table, built on first use.
func allBindings() []binding {
	bindingsOnce.Do(func() { bindingList = buildBindings() })
	return bindingList
}

func bindingsIn(set keySet) []binding {
	var out []binding
	for _, b := range allBindings() {
		if b.Set == set {
			out = append(out, b)
		}
	}
	return out
}

// activeSets are the sets live right now, most specific first.
func (m model) activeSets() []keySet {
	if m.defining && m.screen == screenAlmanac {
		return []keySet{setDefine, setGlobal}
	}
	if m.pollinating {
		return []keySet{setPollen, setGlobal}
	}
	switch m.mode {
	case modePlan:
		return []keySet{setPlan, setGlobal}
	case modeStamp:
		return []keySet{setStamp, setGlobal}
	case modeSelect:
		return []keySet{setSelect, setGlobal}
	}
	var sets []keySet
	sets = append(sets, m.screenSet())
	return append(sets, setGlobal)
}

func (m model) screenSet() keySet {
	switch m.screen {
	case screenShop:
		if m.shelf == shelfMine {
			return setMine
		}
		return setShop
	case screenInfo:
		return setInfo
	case screenAlmanac:
		return setAlmanac
	case screenJournal:
		return setJournal
	case screenHelp:
		return setHelp
	case screenTemplates:
		return setTpl
	case screenOrders:
		return setOrders
	case screenFair:
		return setFair
	case screenShare:
		return setShare
	case screenGnomes:
		return setGnomes
	case screenLineage:
		return setLineage
	}
	return setGarden
}

// lookup finds the binding for a key among the live sets. Exact keys win over
// the catch-all, whichever set the catch-all belongs to.
func (m model) lookup(key string) *binding {
	sets := m.activeSets()
	for _, set := range sets {
		list := bindingsIn(set)
		for i := range list {
			for _, k := range list[i].Keys {
				if k == key {
					return &list[i]
				}
			}
		}
	}
	for _, set := range sets {
		list := bindingsIn(set)
		for i := range list {
			if len(list[i].Keys) == 1 && list[i].Keys[0] == anyKey {
				return &list[i]
			}
		}
	}
	return nil
}

// hintParts are the footer reminders for a screen: its own hints, then (in
// the garden) the global ones, or Esc everywhere else.
func (m model) hintParts(set keySet, extra ...string) []string {
	var hints []string
	for _, b := range bindingsIn(set) {
		if b.Hint != "" {
			hints = append(hints, b.Hint)
		}
	}
	hints = append(hints, extra...)
	for _, h := range hints {
		if strings.HasPrefix(h, "esc ") {
			return hints // the set says what esc does
		}
	}
	if set == setGarden {
		for _, b := range bindingsIn(setGlobal) {
			if b.Hint != "" {
				hints = append(hints, b.Hint)
			}
		}
		return hints
	}
	return append(hints, "esc back")
}

// hintSet is the set whose keys the garden screen's footer should remind you of:
// the mode you are in, if any, otherwise the garden's own.
func (m model) hintSet() keySet {
	if m.pollinating {
		return setPollen
	}
	switch m.mode {
	case modePlan:
		return setPlan
	case modeStamp:
		return setStamp
	case modeSelect:
		return setSelect
	}
	return setGarden
}

// hintLine joins them for a footer.
func (m model) hintLine(set keySet, extra ...string) string {
	return strings.Join(m.hintParts(set, extra...), " · ")
}

func keyOf(msg tea.KeyMsg) string { return msg.String() }

// handleKey dispatches a key through the table.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.naming {
		return m.handleNaming(msg)
	}
	var cmd tea.Cmd
	if m.confirmDel != 0 && keyOf(msg) != "x" {
		m.confirmDel, m.status = 0, ""
	}
	if b := m.lookup(keyOf(msg)); b != nil && b.Do != nil {
		cmd = b.Do(&m)
	}
	m.ensureVisible()
	m.fixAlmanac()
	if m.screen == screenShop && m.shelf == shelfMine {
		m.fixMine()
	}
	return m, cmd
}

// quitCmd saves and ends the program.
func (m *model) quitCmd() tea.Cmd {
	m.audio.Close()
	if err := Save(m.path, m.g); err != nil {
		m.saveErr = err
	}
	return tea.Quit
}

func scrollBy(m *model, delta int) {
	m.cardScroll = max(0, m.cardScroll+delta)
}

func buildBindings() []binding {
	var list []binding
	add := func(set keySet, keys []string, label, desc, hint string, do func(m *model) tea.Cmd) {
		list = append(list, binding{Set: set, Keys: keys, Label: label, Desc: desc, Hint: hint, Do: do})
	}
	k := func(keys ...string) []string { return keys }

	// ---- anywhere -------------------------------------------------------
	add(setGlobal, k("ctrl+c"), "ctrl+c", "quit at once (the garden saves itself)", "", func(m *model) tea.Cmd { return m.quitCmd() })
	add(setGlobal, k("m"), "m", "calming music on or off", "m music", func(m *model) tea.Cmd {
		m.toggleMusic()
		return nil
	})
	add(setGlobal, k("q"), "q", "back to the garden; from the garden, quit (it saves itself)", "q quit", func(m *model) tea.Cmd {
		if m.screen == screenGarden {
			return m.quitCmd()
		}
		m.screen = screenGarden
		return nil
	})
	add(setGlobal, k("?"), "?", "this help, from anywhere; again to close it", "? help", func(m *model) tea.Cmd {
		if m.screen == screenHelp {
			m.screen = screenGarden
		} else {
			m.screen, m.cardScroll = screenHelp, 0
		}
		return nil
	})
	add(setGlobal, k("tab"), "tab", "cycle garden → shed → orders → fair → gnomes → almanac → journal", "tab screens", func(m *model) tea.Cmd {
		m.screen = nextScreen(m.screen)
		if m.screen == screenShop {
			m.refreshShop()
		}
		return nil
	})
	add(setGlobal, k("esc"), "esc", "back to the garden", "", func(m *model) tea.Cmd {
		if m.screen != screenGarden {
			m.screen = screenGarden
		}
		return nil
	})

	// ---- the garden -----------------------------------------------------
	add(setGarden, k("left", "h"), "←↑↓→ / hjkl", "move between beds", "←↑↓→ move", func(m *model) tea.Cmd {
		if m.cursor%plotCols > 0 {
			m.cursor--
		}
		return nil
	})
	add(setGarden, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor%plotCols < plotCols-1 && m.cursor+1 < len(m.g.Plots) {
			m.cursor++
		}
		return nil
	})
	add(setGarden, k("up", "k"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor-plotCols >= 0 {
			m.cursor -= plotCols
		}
		return nil
	})
	add(setGarden, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor+plotCols < len(m.g.Plots) {
			m.cursor += plotCols
		}
		return nil
	})
	add(setGarden, k("home"), "home / end / G", "first bed / last bed", "", func(m *model) tea.Cmd {
		m.cursor = 0
		return nil
	})
	add(setGarden, k("end", "G"), "", "", "", func(m *model) tea.Cmd {
		m.cursor = len(m.g.Plots) - 1
		return nil
	})
	add(setGarden, k("p", "enter"), "p / enter", "sow in the selected bed (opens the seed shed), or open the card of what is growing", "p plant", func(m *model) tea.Cmd {
		if m.plot().Empty() {
			m.openShed(true)
		} else {
			m.screen, m.cardScroll = screenInfo, 0
		}
		return nil
	})
	add(setGarden, k("i", " "), "i / space", "open the plant's info card", "i info", func(m *model) tea.Cmd {
		if !m.plot().Empty() {
			m.screen, m.cardScroll = screenInfo, 0
		}
		return nil
	})
	add(setGarden, k("w"), "w / W", "water this bed / every bed", "w water", func(m *model) tea.Cmd { return m.actWater() })
	add(setGarden, k("W"), "", "", "W water all", func(m *model) tea.Cmd { return m.actWaterAll() })
	add(setGarden, k("c"), "c / C", "clear weeds here / everywhere", "c weed", func(m *model) tea.Cmd { return m.actWeed() })
	add(setGarden, k("C"), "", "", "C weed all", func(m *model) tea.Cmd { return m.actWeedAll() })
	add(setGarden, k("f"), "f / F", "gather ripe seed here / everywhere", "f gather", func(m *model) tea.Cmd { return m.actGather() })
	add(setGarden, k("F"), "", "", "", func(m *model) tea.Cmd { return m.actGatherAll() })
	add(setGarden, k("n", "r"), "n / r", "name the plant in this bed", "n name", func(m *model) tea.Cmd {
		if !m.plot().Empty() {
			m.startNaming()
		} else {
			m.setStatus(subtleStyle, "Plant something first.")
		}
		return nil
	})
	add(setGarden, k("u"), "u", "lift a plant and compost it into the bed", "", func(m *model) tea.Cmd { return m.actLift(false) })
	add(setGarden, k("b"), "b", "break new ground: one more bed", "b new bed", func(m *model) tea.Cmd { return m.actBuyBed() })
	add(setGarden, k("d"), "d", "dig a pond here, or fill it back in", "d pond", func(m *model) tea.Cmd { return m.actPond() })
	add(setGarden, k("g"), "g", "neighbour overlay: colour every bed by how well it gets on with the beds around it, green for good company and red for crowding", "g overlay", func(m *model) tea.Cmd {
		m.overlay = !m.overlay
		if m.overlay {
			m.setStatus(okStyle, "Overlay on: green beds help one another, red beds hinder; the figure is the net effect on growth. g again to leave.")
		} else {
			m.setStatus(subtleStyle, "Overlay off.")
		}
		return nil
	})
	add(setGarden, k("x"), "x", "pollinate by hand: brush pollen from another flower of the same species onto this one, so its next seed is that cross", "x pollinate", func(m *model) tea.Cmd { return m.actStartPollinate() })
	add(setGarden, k("a"), "a", "the almanac", "a almanac", func(m *model) tea.Cmd {
		m.screen, m.cardScroll = screenAlmanac, 0
		m.refreshAlmanac()
		return nil
	})
	add(setGarden, k("s"), "s", "the seed shed (shop and your own seeds)", "s seeds", func(m *model) tea.Cmd {
		m.openShed(false)
		return nil
	})

	// ---- choosing a pollen donor ----------------------------------------
	add(setPollen, k("left", "h"), "←↑↓→ / hjkl", "move to the flower to take pollen from (flowers that will do are outlined in gold)", "←↑↓→ move", func(m *model) tea.Cmd {
		if m.cursor%plotCols > 0 {
			m.cursor--
		}
		return nil
	})
	add(setPollen, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor%plotCols < plotCols-1 && m.cursor+1 < len(m.g.Plots) {
			m.cursor++
		}
		return nil
	})
	add(setPollen, k("up", "k"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor-plotCols >= 0 {
			m.cursor -= plotCols
		}
		return nil
	})
	add(setPollen, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		if m.cursor+plotCols < len(m.g.Plots) {
			m.cursor += plotCols
		}
		return nil
	})
	add(setPollen, k("enter", "x", " "), "enter / x / space", "take pollen from the selected flower", "enter pollinate", func(m *model) tea.Cmd { return m.actFinishPollinate() })
	add(setPollen, k("esc"), "esc", "cancel", "esc cancel", func(m *model) tea.Cmd {
		m.pollinating = false
		m.setStatus(subtleStyle, "Put the brush down.")
		return nil
	})

	// ---- the seed shed --------------------------------------------------
	add(setShop, k("up", "k"), "↑↓ / jk", "browse", "↑↓ browse", func(m *model) tea.Cmd {
		if m.shopCursor > 0 {
			m.shopCursor--
			m.cardScroll, m.shopVariety = 0, 0
		}
		return nil
	})
	add(setShop, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		if m.shopCursor < len(m.shop)-1 {
			m.shopCursor++
			m.cardScroll, m.shopVariety = 0, 0
		}
		return nil
	})
	add(setShop, k("left", "h"), "←→ / hl", "choose a variety: the named forms, then your own cultivars of the species", "←→ variety", func(m *model) tea.Cmd {
		if len(m.shop) > 0 {
			forms := len(m.g.varietyForms(m.shop[m.shopCursor]))
			m.shopVariety = (m.shopVariety + forms - 1) % forms
		}
		return nil
	})
	add(setShop, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		if len(m.shop) > 0 {
			m.shopVariety = (m.shopVariety + 1) % len(m.g.varietyForms(m.shop[m.shopCursor]))
		}
		return nil
	})
	add(setShop, k("pgup"), "pgup / pgdn", "read a long card", "", func(m *model) tea.Cmd { scrollBy(m, -6); return nil })
	add(setShop, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 6); return nil })
	add(setShop, k("home", "g"), "home / g, end / G", "first / last", "", func(m *model) tea.Cmd {
		m.shopCursor, m.cardScroll, m.shopVariety = 0, 0, 0
		return nil
	})
	add(setShop, k("end", "G"), "", "", "", func(m *model) tea.Cmd {
		m.shopCursor, m.cardScroll, m.shopVariety = max(0, len(m.shop)-1), 0, 0
		return nil
	})
	add(setShop, k("t"), "t", "show only what is happy in this season, or the whole rack", "t season", func(m *model) tea.Cmd {
		m.shopSeason = !m.shopSeason
		m.refreshShop()
		if m.shopSeason {
			m.setStatus(subtleStyle, "Showing seeds for %s only.", m.g.Season(m.now))
		} else {
			m.setStatus(subtleStyle, "Showing the whole seed rack.")
		}
		return nil
	})
	add(setShop, k("enter", "p", " "), "enter / p / space", "buy one seed and sow it in the selected bed (your own cultivars cost much more than named forms)", "enter buy & sow", func(m *model) tea.Cmd { return m.actSow() })
	add(setShop, k("b"), "b", "buy one seed into your shed without sowing it", "b buy", func(m *model) tea.Cmd { return m.actBuySeed() })
	add(setShop, k("i"), "i", "import a friend's code: three seeds of their cultivar, and the line goes in your almanac", "i import", func(m *model) tea.Cmd { m.startImport(); return nil })
	add(setShop, k("s"), "s", "switch to your own seeds", "s my seeds", func(m *model) tea.Cmd { return m.actShelf(shelfMine) })

	// ---- the seed shed: my seeds ---------------------------------------
	add(setMine, k("up", "k"), "↑↓ / jk", "browse your packets (or folders)", "↑↓ browse", func(m *model) tea.Cmd {
		if m.mineCursor > 0 {
			m.mineCursor--
			m.cardScroll = 0
		}
		return nil
	})
	add(setMine, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		if m.mineCursor < len(m.shedRows())-1 {
			m.mineCursor++
			m.cardScroll = 0
		}
		return nil
	})
	add(setMine, k("left", "h"), "←→ / hl", "in folders: close the folder, or step out to the one above / open it, or step in", "", func(m *model) tea.Cmd {
		if !m.g.ShedFolders {
			return nil
		}
		if row, ok := m.shedCur(); ok && row.Kind == shedFolder && row.Open {
			no := false
			m.toggleShedFolder(&no)
		} else {
			m.shedParent()
			m.cardScroll = 0
		}
		return nil
	})
	add(setMine, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		if !m.g.ShedFolders {
			return nil
		}
		if row, ok := m.shedCur(); ok && row.Kind == shedFolder {
			if !row.Open {
				yes := true
				m.toggleShedFolder(&yes)
			} else if m.mineCursor < len(m.shedRows())-1 {
				m.mineCursor++
				m.cardScroll = 0
			}
		}
		return nil
	})
	add(setMine, k("v"), "v", "switch between a plain list and folders (category, species, variety, packets); a folder holding one thing is skipped", "v folders", func(m *model) tea.Cmd {
		m.toggleShedFolders()
		return nil
	})
	add(setMine, k("C"), "C", "in folders: close every folder, or open them all if all are closed", "", func(m *model) tea.Cmd {
		if m.g.ShedFolders {
			m.toggleAllShedFolders()
		}
		return nil
	})
	add(setMine, k("t"), "t", "genetics: the packet's lineage, its first generation and every mutation and drift since", "t genetics", func(m *model) tea.Cmd {
		pi := m.minePacket()
		if pi < 0 {
			m.setStatus(subtleStyle, "Open the folder to a packet to read its genetics.")
			return nil
		}
		sub, ok := m.g.packetLineage(m.g.Shed[pi])
		m.openLineage(sub, ok)
		return nil
	})
	add(setMine, k("pgup"), "pgup / pgdn", "read a long card", "", func(m *model) tea.Cmd { scrollBy(m, -6); return nil })
	add(setMine, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 6); return nil })
	add(setMine, k("home", "g"), "home / g, end / G", "first / last packet", "", func(m *model) tea.Cmd {
		m.mineCursor, m.cardScroll = 0, 0
		return nil
	})
	add(setMine, k("end", "G"), "", "", "", func(m *model) tea.Cmd {
		m.mineCursor, m.cardScroll = max(0, len(m.shedRows())-1), 0
		return nil
	})
	add(setMine, k("enter", "p", " "), "enter / p / space", "sow one seed from the packet in the selected bed (on a folder: open or close it)", "enter sow", func(m *model) tea.Cmd { return m.actSowMine() })
	add(setMine, k("$"), "$ / S", "sell one seed from the packet / the whole packet", "$ sell", func(m *model) tea.Cmd { return m.actSell(1) })
	add(setMine, k("S"), "", "", "", func(m *model) tea.Cmd { return m.actSell(0) })
	add(setMine, k("r", "n"), "r / n", "give the packet a name of your own", "r label", func(m *model) tea.Cmd {
		m.startNamingPacket()
		return nil
	})
	add(setMine, k("i"), "i", "import a friend's code: three seeds of their cultivar, and the line goes in your almanac", "i import", func(m *model) tea.Cmd { m.startImport(); return nil })
	add(setMine, k("s"), "s", "switch to the shop", "s shop", func(m *model) tea.Cmd { return m.actShelf(shelfShop) })

	// ---- a plant's card -------------------------------------------------
	add(setInfo, k("w"), "w", "water the bed", "w water", func(m *model) tea.Cmd {
		if m.g.Water(m.cursor, m.now) {
			m.dirty = true
			m.setStatus(waterStyle, "Watered %s.", m.plot().DisplayName())
		}
		return nil
	})
	add(setInfo, k("c"), "c", "clear the weeds", "c weed", func(m *model) tea.Cmd {
		if ok, _ := m.g.Weed(m.cursor, m.now); ok {
			m.dirty = true
			m.setStatus(okStyle, "Tidied the bed.")
		}
		return nil
	})
	add(setInfo, k("f"), "f", "gather ripe seed", "f gather", func(m *model) tea.Cmd {
		if got := m.g.Gather(m.cursor, m.now); got > 0 {
			m.dirty = true
			m.setStatus(seedStyle, "Gathered %d seed(s).", got)
		}
		return nil
	})
	add(setInfo, k("n", "r"), "n / r", "name the plant", "n name", func(m *model) tea.Cmd {
		m.startNaming()
		return nil
	})
	add(setInfo, k("t"), "t", "genetics: the plant's lineage, its first generation and every mutation and drift on the way", "t genetics", func(m *model) tea.Cmd {
		sub, ok := m.g.plantLineage(m.plot())
		m.openLineage(sub, ok)
		return nil
	})
	add(setInfo, k("u"), "u", "lift the plant and compost it", "u lift", func(m *model) tea.Cmd { return m.actLift(true) })
	add(setInfo, k("left", "h"), "←→ / hl", "the previous / next bed", "←→ other beds", func(m *model) tea.Cmd {
		m.cursor = (m.cursor + len(m.g.Plots) - 1) % len(m.g.Plots)
		m.cardScroll = 0
		return nil
	})
	add(setInfo, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		m.cursor = (m.cursor + 1) % len(m.g.Plots)
		m.cardScroll = 0
		return nil
	})
	add(setInfo, k("up", "k"), "↑↓ / jk, pgup / pgdn", "scroll the card", "", func(m *model) tea.Cmd { scrollBy(m, -1); return nil })
	add(setInfo, k("down", "j"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 1); return nil })
	add(setInfo, k("pgup"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, -10); return nil })
	add(setInfo, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 10); return nil })
	add(setInfo, k("home"), "home", "back to the top of the card", "", func(m *model) tea.Cmd { m.cardScroll = 0; return nil })

	// ---- the almanac ----------------------------------------------------
	add(setAlmanac, k("up", "k"), "↑↓ / jk", "browse the guide, species and visitors; category headings are rows too", "↑↓ browse", func(m *model) tea.Cmd {
		if m.almanacCursor > 0 {
			m.almanacCursor--
			m.cardScroll, m.almanacVariety = 0, 0
		}
		return nil
	})
	add(setAlmanac, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		if m.almanacCursor < len(m.almanacVisible())-1 {
			m.almanacCursor++
			m.cardScroll, m.almanacVariety = 0, 0
		}
		return nil
	})
	add(setAlmanac, k("enter", "c"), "enter / c", "fold or open the category the cursor is in", "enter fold", func(m *model) tea.Cmd {
		if row, ok := m.almanacCur(); ok {
			m.toggleGroup(row.group())
		}
		return nil
	})
	add(setAlmanac, k("C"), "C", "fold every category, or open them all if they are all folded", "C fold all", func(m *model) tea.Cmd {
		m.toggleAllGroups()
		return nil
	})
	add(setAlmanac, k("left", "h"), "←→ / hl, space", "step through a plant's five stages", "←→ stage", func(m *model) tea.Cmd {
		m.almanacStage = max(0, m.almanacStage-1)
		return nil
	})
	add(setAlmanac, k("right", "l"), "", "", "", func(m *model) tea.Cmd {
		m.almanacStage = min(StageCount-1, m.almanacStage+1)
		return nil
	})
	add(setAlmanac, k(" "), "", "", "", func(m *model) tea.Cmd {
		m.almanacStage = (m.almanacStage + 1) % StageCount
		return nil
	})
	add(setAlmanac, k("v"), "v", "flick through a species' varieties: the named forms, then your own cultivars of it", "v variety", func(m *model) tea.Cmd {
		if row, ok := m.almanacCur(); ok && row.IsPlant() {
			m.almanacVariety = (m.almanacVariety + 1) % len(m.g.varietyForms(row.Species))
		}
		return nil
	})
	add(setAlmanac, k("o"), "o", "on a species page: open the cultivar page of the variety you have flicked to", "", func(m *model) tea.Cmd {
		m.openOwnVariety()
		return nil
	})
	add(setAlmanac, k("e"), "e", "share a cultivar you have bred: make a code to give a friend (on its page)", "", func(m *model) tea.Cmd { return m.actShare() })
	add(setAlmanac, k("n"), "n", "name a cultivar you have bred (on a cultivar's page)", "", func(m *model) tea.Cmd {
		m.startNamingCultivar()
		return nil
	})
	add(setAlmanac, k("t"), "t", "genetics: how a cultivar's line came about, generation by generation (on its page)", "", func(m *model) tea.Cmd {
		row, ok := m.almanacCur()
		if !ok || row.Kind != rowCultivar {
			m.setStatus(subtleStyle, "Open one of your cultivars (under YOUR CULTIVARS) to read its genetics.")
			return nil
		}
		if c := m.g.CultivarByID(row.Cultivar); c != nil {
			sub, ok := m.g.cultivarLineage(c)
			m.openLineage(sub, ok)
		}
		return nil
	})
	add(setAlmanac, k("D"), "D", "define a cultivar: the ideal for each gene and how far a plant may stray and still be the variety (on its page)", "", func(m *model) tea.Cmd {
		row, ok := m.almanacCur()
		if !ok || row.Kind != rowCultivar || m.g.CultivarByID(row.Cultivar) == nil {
			m.setStatus(subtleStyle, "Open one of your cultivars (under YOUR CULTIVARS) to define it.")
			return nil
		}
		m.defining, m.defGene = true, 0
		m.cardScroll = 0
		m.setStatus(okStyle, "Defining ‘%s’: ←→ tolerance, [ ] ideal, ↑↓ gene, enter when done.", m.g.CultivarByID(row.Cultivar).Name)
		return nil
	})
	add(setAlmanac, k("x"), "x", "delete the record of a cultivar, bred or imported (on its page; asks twice); plants and seed you hold stay", "", func(m *model) tea.Cmd { return m.actDeleteCultivar() })
	add(setAlmanac, k("pgup"), "pgup / pgdn", "read a long page", "", func(m *model) tea.Cmd { scrollBy(m, -6); return nil })
	add(setAlmanac, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 6); return nil })
	add(setAlmanac, k("home", "g"), "home / g, end / G", "first / last entry", "", func(m *model) tea.Cmd {
		m.almanacCursor, m.cardScroll = 0, 0
		return nil
	})
	add(setAlmanac, k("end", "G"), "", "", "", func(m *model) tea.Cmd {
		m.almanacCursor, m.cardScroll = len(m.almanacVisible())-1, 0
		return nil
	})

	// ---- the journal ----------------------------------------------------
	add(setJournal, k("up", "k"), "↑↓ / jk, pgup / pgdn", "scroll back through the log", "↑↓ scroll", func(m *model) tea.Cmd {
		m.journalScroll = max(0, m.journalScroll-1)
		return nil
	})
	add(setJournal, k("down", "j"), "", "", "", func(m *model) tea.Cmd { m.journalScroll++; return nil })
	add(setJournal, k("pgup"), "", "", "", func(m *model) tea.Cmd { m.journalScroll = max(0, m.journalScroll-10); return nil })
	add(setJournal, k("pgdown"), "", "", "", func(m *model) tea.Cmd { m.journalScroll += 10; return nil })
	add(setJournal, k("home", "g"), "home / g", "newest entry", "", func(m *model) tea.Cmd { m.journalScroll = 0; return nil })

	// ---- the share screen -----------------------------------------------
	add(setShare, k(anyKey), "any key", "back to the almanac", "", func(m *model) tea.Cmd {
		m.screen = screenAlmanac
		return nil
	})

	// ---- the genetics screen --------------------------------------------
	add(setLineage, k("up", "k"), "↑↓ / jk, pgup / pgdn", "scroll", "↑↓ scroll", func(m *model) tea.Cmd { scrollBy(m, -1); return nil })
	add(setLineage, k("down", "j"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 1); return nil })
	add(setLineage, k("pgup"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, -10); return nil })
	add(setLineage, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 10); return nil })
	add(setLineage, k("home", "g"), "home / g", "back to the top", "", func(m *model) tea.Cmd { m.cardScroll = 0; return nil })
	add(setLineage, k("esc", "t", "enter", "backspace"), "esc / t / enter / backspace", "go back", "esc back", func(m *model) tea.Cmd {
		m.screen, m.cardScroll = m.lineageBack, 0
		return nil
	})

	// ---- defining a cultivar --------------------------------------------
	add(setDefine, k("up", "k"), "↑↓ / jk", "choose a gene", "↑↓ gene", func(m *model) tea.Cmd {
		m.defGene = (m.defGene + geneCount - 1) % geneCount
		return nil
	})
	add(setDefine, k("down", "j"), "", "", "", func(m *model) tea.Cmd {
		m.defGene = (m.defGene + 1) % geneCount
		return nil
	})
	add(setDefine, k("right", "l"), "←→ / hl", "widen / narrow how far this gene may stray from the ideal", "←→ tolerance", func(m *model) tea.Cmd { return m.actDefine(false, +1) })
	add(setDefine, k("left", "h"), "", "", "", func(m *model) tea.Cmd { return m.actDefine(false, -1) })
	add(setDefine, k("]"), "[ / ]", "move the ideal for this gene down / up", "[ ] ideal", func(m *model) tea.Cmd { return m.actDefine(true, +1) })
	add(setDefine, k("["), "", "", "", func(m *model) tea.Cmd { return m.actDefine(true, -1) })
	add(setDefine, k("r"), "r", "put every tolerance back to the default", "r reset", func(m *model) tea.Cmd { return m.actDefineReset() })
	add(setDefine, k("enter", "esc", "D"), "enter / esc / D", "done", "enter done", func(m *model) tea.Cmd {
		m.defining = false
		m.setStatus(okStyle, "Definition kept.")
		return nil
	})

	// ---- help -----------------------------------------------------------
	add(setHelp, k("up", "k"), "↑↓ / jk, pgup / pgdn", "scroll", "", func(m *model) tea.Cmd { scrollBy(m, -1); return nil })
	add(setHelp, k("down", "j"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 1); return nil })
	add(setHelp, k("pgup"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, -10); return nil })
	add(setHelp, k("pgdown"), "", "", "", func(m *model) tea.Cmd { scrollBy(m, 10); return nil })
	add(setHelp, k("home", "g"), "home / g", "back to the top", "", func(m *model) tea.Cmd { m.cardScroll = 0; return nil })
	add(setHelp, k(anyKey), "any other key", "close help", "", func(m *model) tea.Cmd {
		m.screen = screenGarden
		return nil
	})
	list = append(list, planBindings()...)
	return append(list, gnomeBindings()...)
}

// keysMarkdown renders the documented keys as the markdown tables the README
// carries, one per set.
func keysMarkdown() string {
	var b strings.Builder
	for _, set := range setOrder {
		var rows []keyDoc
		for _, d := range documentedKeys() {
			if d.Set == set {
				rows = append(rows, d)
			}
		}
		if len(rows) == 0 {
			continue
		}
		b.WriteString("**" + upperFirst(setTitles[set]) + "**\n\n| key | action |\n| --- | --- |\n")
		for _, d := range rows {
			b.WriteString("| `" + d.Label + "` | " + d.Desc + " |\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
