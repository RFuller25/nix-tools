package main

import (
	"fmt"
	"strings"
)

func (m model) viewJournal() string {
	rows := max(3, m.height-6)
	entries := m.g.Journal

	if maxScroll := max(0, len(entries)-rows); m.journalScroll > maxScroll {
		m.journalScroll = maxScroll
	}

	var lines []string
	// Newest first, scrolled back through history.
	for i := len(entries) - 1 - m.journalScroll; i >= 0 && len(lines) < rows; i-- {
		e := entries[i]
		stamp := subtleStyle.Render(e.At.Format("2 Jan 15:04"))
		lines = append(lines, stamp+"  "+valueStyle.Render(e.Text))
	}
	if len(lines) == 0 {
		lines = append(lines, subtleStyle.Render("Nothing written down yet."))
	}

	stats := subtleStyle.Render(fmt.Sprintf("sown %d  ·  matured %d  ·  seeds gathered %d  ·  tending since %s",
		m.g.Planted, m.g.Matured, m.g.Gathered, m.g.Created.Format("2 Jan 2006")))

	head := titleStyle.Render("✎ journal") + "  " + stats
	keys := "↑↓ scroll · esc back · q garden"
	return strings.Join([]string{head, m.divider(), strings.Join(lines, "\n"), m.divider(), m.footer(keys)}, "\n")
}
