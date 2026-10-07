package main

// Moving about the almanac. The list is long, so each category can be folded
// to its heading. Headings are rows of their own that the cursor can stand on,
// which is how a folded category is found and opened again.

// almanacVisible is the rows on show: every heading, and the rows under each
// heading that has not been folded.
func (m model) almanacVisible() []almanacRow {
	out := make([]almanacRow, 0, len(m.almanac))
	for _, r := range m.almanac {
		if r.Kind != rowHeader && m.g.Folded[r.group()] {
			continue
		}
		out = append(out, r)
	}
	return out
}

// almanacCur is the row under the cursor, and whether there is one.
func (m model) almanacCur() (almanacRow, bool) {
	vis := m.almanacVisible()
	if m.almanacCursor < 0 || m.almanacCursor >= len(vis) {
		return almanacRow{}, false
	}
	return vis[m.almanacCursor], true
}

// groupCount is how many rows sit under a heading.
func (m model) groupCount(group string) int {
	n := 0
	for _, r := range m.almanac {
		if r.Kind != rowHeader && r.group() == group {
			n++
		}
	}
	return n
}

// almanacMetrics is the shape of the list: how wide it is, whether the window
// is too narrow for a card beside it, and how many rows of it fit. When there
// is no room for a card the selected row's one-line summary goes under the
// list, and takes two rows of it.
func (m model) almanacMetrics() (listWidth int, narrow bool, rows int) {
	listWidth = 32
	narrow = m.width-listWidth-6 < 30
	avail := max(3, m.height-5) // header, divider, divider, status and keys
	if narrow {
		listWidth = max(20, m.width-2)
		return listWidth, true, max(1, avail-2)
	}
	return listWidth, false, avail
}

// fixAlmanac keeps the cursor on a real row and the list scrolled so the row
// under it is on screen. It is the one place the scroll position is decided,
// and runs after every key and every resize, because drawing the screen works
// on a copy of the model and could not remember it.
func (m *model) fixAlmanac() {
	vis := m.almanacVisible()
	m.almanacCursor = max(0, min(m.almanacCursor, len(vis)-1))
	_, _, rows := m.almanacMetrics()
	if m.almanacCursor < m.almanacScroll {
		m.almanacScroll = m.almanacCursor
	}
	if m.almanacCursor >= m.almanacScroll+rows {
		m.almanacScroll = m.almanacCursor - rows + 1
	}
	m.almanacScroll = max(0, min(m.almanacScroll, max(0, len(vis)-rows)))
}

// toggleGroup folds or opens one category. Folding leaves the cursor on the
// heading, since the row it was on is gone.
func (m *model) toggleGroup(group string) {
	if m.g.Folded == nil {
		m.g.Folded = map[string]bool{}
	}
	if m.g.Folded[group] {
		delete(m.g.Folded, group)
	} else {
		m.g.Folded[group] = true
		for i, r := range m.almanacVisible() {
			if r.Kind == rowHeader && r.Group == group {
				m.almanacCursor = i
			}
		}
	}
	m.dirty = true
	m.cardScroll = 0
	m.fixAlmanac()
}

// toggleAllGroups folds every category, or opens them all if all are folded.
func (m *model) toggleAllGroups() {
	if m.g.Folded == nil {
		m.g.Folded = map[string]bool{}
	}
	open := false
	for _, r := range m.almanac {
		if r.Kind == rowHeader && !m.g.Folded[r.Group] {
			open = true
		}
	}
	for _, r := range m.almanac {
		if r.Kind != rowHeader {
			continue
		}
		if open {
			m.g.Folded[r.Group] = true
		} else {
			delete(m.g.Folded, r.Group)
		}
	}
	// Stand on the heading of whatever the cursor was in.
	cur, _ := m.almanacCur()
	m.dirty = true
	m.cardScroll = 0
	for i, r := range m.almanacVisible() {
		if r.Kind == rowHeader && r.Group == cur.group() {
			m.almanacCursor = i
		}
	}
	m.fixAlmanac()
}
