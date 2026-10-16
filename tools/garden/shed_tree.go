package main

import (
	"fmt"
	"sort"
	"strings"
)

// The seed shed as folders. "My seeds" is a plain list of packets by default;
// with folders on, the packets are filed category (the kind of plant), then
// species, then variety, then the packets themselves. Only folders that hold
// seed are shown, and a folder that has just one thing in it is stepped over:
// open a category with a single packet in it and the packet is right there,
// with no species or variety folder to open first.

type shedRowKind int

const (
	shedPacket shedRowKind = iota
	shedFolder
)

// shedLevel names how deep a folder is.
var shedLevels = [...]string{"category", "species", "variety"}

// shedRow is one line of the shed as it is shown.
type shedRow struct {
	Kind    shedRowKind
	Key     string // the folder's key, for remembering that it is open
	Depth   int
	Level   int // 0 category, 1 species, 2 variety
	Title   string
	Seeds   int // seeds under a folder
	Packets int
	Open    bool
	Packet  int      // index in the garden's shed, for a packet row
	Tagged  bool     // a packet inside a variety folder, told apart by where it came from
	Kids    []string // for a folder card: what is inside
}

type shedNode struct {
	key, title string
	level      int
	kids       []*shedNode
	packet     int // -1 for a folder
	seeds      int
	packets    int
}

// VarietyTitle is the variety a packet is filed under.
func (pk Packet) VarietyTitle() string {
	sp := pk.Species()
	if sp == nil {
		return "unknown seed"
	}
	if pk.Label != "" {
		return pk.Label
	}
	if pk.Pure {
		if len(sp.Varieties()) == 1 {
			return "the species"
		}
		return "‘" + sp.Variety(pk.Variety).Name + "’"
	}
	idx, gap := sp.NearestVariety(pk.Mean())
	name := "‘" + sp.Variety(idx).Name + "’"
	if len(sp.Varieties()) == 1 {
		name = "the species"
	}
	if gap > hybridGap {
		return "hybrid (" + name + " type)"
	}
	return name
}

// packetTag tells packets of one variety apart inside its folder.
func packetTag(pk Packet) string {
	parts := []string{"gen " + fmt.Sprint(pk.Gen)}
	switch {
	case pk.Pure && pk.From == "":
		parts = append(parts, "from the shop")
	case pk.From != "":
		parts = append(parts, "from "+pk.From)
	}
	if pk.Descent != "" && pk.From == "" {
		parts = append(parts, pk.Descent)
	}
	return strings.Join(parts, " · ")
}

// shedTree builds the folders for the packets in the shed.
func (g *Garden) shedTree() []*shedNode {
	var cats []*shedNode
	find := func(list *[]*shedNode, key, title string, level int) *shedNode {
		for _, n := range *list {
			if n.key == key {
				return n
			}
		}
		n := &shedNode{key: key, title: title, level: level, packet: -1}
		*list = append(*list, n)
		return n
	}
	for _, pi := range g.ShedOrder() {
		pk := g.Shed[pi]
		catName, catKey := "unknown", "unknown"
		spKey, spName := pk.SpeciesID, pk.SpeciesID
		if sp := pk.Species(); sp != nil {
			catName, catKey = sp.Kind.String(), fmt.Sprintf("k%02d", int(sp.Kind))
			spName = sp.Common
		}
		cat := find(&cats, catKey, catName, 0)
		spn := find(&cat.kids, catKey+"/"+spKey, spName, 1)
		vt := pk.VarietyTitle()
		vn := find(&spn.kids, catKey+"/"+spKey+"/"+vt, vt, 2)
		vn.kids = append(vn.kids, &shedNode{key: fmt.Sprintf("%s#%d", vn.key, pk.ID), packet: pi, level: 3})
		for _, n := range []*shedNode{cat, spn, vn} {
			n.seeds += pk.Count
			n.packets++
		}
	}
	sort.SliceStable(cats, func(i, j int) bool { return cats[i].key < cats[j].key })
	return cats
}

// shedRows is the shed as the list shows it: plain packets, or the open folders.
func (m model) shedRows() []shedRow {
	g := m.g
	if !g.ShedFolders {
		order := g.ShedOrder()
		rows := make([]shedRow, 0, len(order))
		for _, pi := range order {
			rows = append(rows, shedRow{Kind: shedPacket, Packet: pi, Seeds: g.Shed[pi].Count, Packets: 1})
		}
		return rows
	}
	var rows []shedRow
	var emit func(n *shedNode, depth int, inVariety bool)
	// open emits a folder row, and the rows under it if it is open.
	emit = func(n *shedNode, depth int, inVariety bool) {
		if n.packet >= 0 {
			rows = append(rows, shedRow{Kind: shedPacket, Depth: depth, Packet: n.packet, Seeds: g.Shed[n.packet].Count, Packets: 1, Tagged: inVariety})
			return
		}
		open := g.ShedOpen[n.key]
		kids := shownKids(n)
		var names []string
		for _, k := range kids {
			if k.packet >= 0 {
				names = append(names, fmt.Sprintf("%s ×%d", g.Shed[k.packet].Name(), g.Shed[k.packet].Count))
			} else {
				names = append(names, fmt.Sprintf("%s ×%d", k.title, k.seeds))
			}
		}
		rows = append(rows, shedRow{
			Kind: shedFolder, Key: n.key, Depth: depth, Level: n.level, Title: n.title,
			Seeds: n.seeds, Packets: n.packets, Open: open, Kids: names,
		})
		if !open {
			return
		}
		for _, k := range kids {
			emit(k, depth+1, n.level == 2)
		}
	}
	// A category is always a folder, so the shed can be seen by kind.
	for _, c := range g.shedTree() {
		emit(c, 0, false)
	}
	return rows
}

// shownKids is what is listed when a folder is opened. A folder with one thing
// in it is stepped over (its contents stand in its place, under a title that
// remembers the way down), and when that leaves a folder as the only thing
// listed, the folder itself is stepped over too, so a category with a single
// species or variety in it opens straight onto what is in that.
func shownKids(n *shedNode) []*shedNode {
	var list []*shedNode
	for _, c := range n.kids {
		list = append(list, skipSingle(c))
	}
	for len(list) == 1 && list[0].packet < 0 {
		var next []*shedNode
		for _, c := range list[0].kids {
			next = append(next, skipSingle(c))
		}
		list = next
	}
	return list
}

// skipSingle replaces a folder that holds exactly one thing with that thing.
// The folder that is shown keeps the names of the folders skipped on the way.
func skipSingle(c *shedNode) *shedNode {
	title := c.title
	for c.packet < 0 && len(c.kids) == 1 {
		c = c.kids[0]
		if c.packet < 0 {
			title += " › " + c.title
		}
	}
	if c.packet >= 0 {
		return c
	}
	cp := *c
	cp.title = title
	return &cp
}

// shedCur is the row under the cursor, and whether there is one.
func (m model) shedCur() (shedRow, bool) {
	rows := m.shedRows()
	if len(rows) == 0 {
		return shedRow{}, false
	}
	return rows[max(0, min(m.mineCursor, len(rows)-1))], true
}

// minePacket is the index in the shed of the highlighted packet, or -1 when the
// cursor is on a folder or the shed is empty.
func (m *model) minePacket() int {
	rows := m.shedRows()
	if len(rows) == 0 {
		return -1
	}
	m.mineCursor = max(0, min(m.mineCursor, len(rows)-1))
	if rows[m.mineCursor].Kind != shedPacket {
		return -1
	}
	return rows[m.mineCursor].Packet
}

// fixMine keeps the cursor on a real row and the list scrolled to show it.
func (m *model) fixMine() {
	rows := m.shedRows()
	m.mineCursor = max(0, min(m.mineCursor, len(rows)-1))
	avail := max(1, m.height-5)
	if m.mineCursor < m.mineScroll {
		m.mineScroll = m.mineCursor
	}
	if m.mineCursor >= m.mineScroll+avail {
		m.mineScroll = m.mineCursor - avail + 1
	}
	m.mineScroll = max(0, min(m.mineScroll, max(0, len(rows)-avail)))
}

// toggleShedFolders switches between the plain list and folders, keeping the
// cursor on the same packet and opening the folders on the way to it.
func (m *model) toggleShedFolders() {
	pi := m.minePacket()
	m.g.ShedFolders = !m.g.ShedFolders
	m.dirty = true
	m.cardScroll = 0
	if pi >= 0 {
		m.revealPacket(pi)
	} else {
		m.mineCursor = 0
	}
	if m.g.ShedFolders {
		m.setStatus(subtleStyle, "Seeds in folders: category, species, variety. v goes back to the plain list.")
	} else {
		m.setStatus(subtleStyle, "Seeds as a plain list. v files them in folders.")
	}
}

// revealPacket opens the folders above a packet and puts the cursor on it.
func (m *model) revealPacket(pi int) {
	if m.g.ShedOpen == nil {
		m.g.ShedOpen = map[string]bool{}
	}
	if m.g.ShedFolders {
		var walk func(n *shedNode, path []string) bool
		walk = func(n *shedNode, path []string) bool {
			if n.packet == pi {
				for _, k := range path {
					m.g.ShedOpen[k] = true
				}
				return true
			}
			for _, k := range n.kids {
				if walk(k, append(path, n.key)) {
					return true
				}
			}
			return false
		}
		for _, c := range m.g.shedTree() {
			if walk(c, nil) {
				break
			}
		}
	}
	for i, r := range m.shedRows() {
		if r.Kind == shedPacket && r.Packet == pi {
			m.mineCursor = i
			return
		}
	}
}

// shedParent moves the cursor to the folder that holds the highlighted row.
func (m *model) shedParent() {
	rows := m.shedRows()
	if m.mineCursor >= len(rows) {
		return
	}
	depth := rows[m.mineCursor].Depth
	for i := m.mineCursor - 1; i >= 0; i-- {
		if rows[i].Kind == shedFolder && rows[i].Depth < depth {
			m.mineCursor = i
			return
		}
	}
}

// toggleShedFolder opens or closes the folder under the cursor.
func (m *model) toggleShedFolder(open *bool) {
	row, ok := m.shedCur()
	if !ok || row.Kind != shedFolder {
		return
	}
	if m.g.ShedOpen == nil {
		m.g.ShedOpen = map[string]bool{}
	}
	want := !row.Open
	if open != nil {
		want = *open
	}
	if want {
		m.g.ShedOpen[row.Key] = true
	} else {
		delete(m.g.ShedOpen, row.Key)
	}
	m.dirty = true
	m.cardScroll = 0
}

// toggleAllShedFolders closes every folder, or opens every one if all are closed.
func (m *model) toggleAllShedFolders() {
	if m.g.ShedOpen == nil {
		m.g.ShedOpen = map[string]bool{}
	}
	if len(m.g.ShedOpen) > 0 {
		m.g.ShedOpen = map[string]bool{}
		m.mineCursor = 0
		for i, r := range m.shedRows() {
			if r.Kind == shedFolder {
				m.mineCursor = i
				break
			}
		}
	} else {
		var walk func(n *shedNode)
		walk = func(n *shedNode) {
			if n.packet >= 0 {
				return
			}
			m.g.ShedOpen[n.key] = true
			for _, k := range n.kids {
				walk(k)
			}
		}
		for _, c := range m.g.shedTree() {
			walk(c)
		}
	}
	m.dirty = true
	m.cardScroll = 0
}
