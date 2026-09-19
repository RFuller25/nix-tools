package main

import (
	"fmt"
	"strings"
)

func (m model) viewJournal() string {
	var tasks []string
	if active := m.g.ActiveTasks(); len(active) > 0 {
		tasks = append(tasks, labelStyle.Render("WORTH DOING"))
		for _, t := range active {
			tasks = append(tasks, fit("  "+okStyle.Render("○ ")+valueStyle.Render(t.String())+
				subtleStyle.Render(fmt.Sprintf("  · %d seeds", t.Reward)), m.width))
		}
		tasks = append(tasks, "")
	}

	rows := max(3, m.height-6-len(tasks))
	entries := m.g.Journal

	if maxScroll := max(0, len(entries)-rows); m.journalScroll > maxScroll {
		m.journalScroll = maxScroll
	}

	var lines []string
	// Newest first, scrolled back through history.
	for i := len(entries) - 1 - m.journalScroll; i >= 0 && len(lines) < rows; i-- {
		e := entries[i]
		stamp := subtleStyle.Render(e.At.Format("2 Jan 15:04"))
		lines = append(lines, fit(stamp+"  "+valueStyle.Render(e.Text), m.width))
	}
	if len(lines) == 0 {
		lines = append(lines, subtleStyle.Render("Nothing written down yet."))
	}

	stats := subtleStyle.Render(fmt.Sprintf(
		"sown %d  ·  matured %d  ·  gathered %d  ·  volunteers %d  ·  pressed %d  ·  done %d  ·  since %s",
		m.g.Planted, m.g.Matured, m.g.Gathered, m.g.Volunteers, len(m.g.Herbarium), m.g.TasksDone(),
		m.g.Created.Format("2 Jan 2006")))

	head := fit(titleStyle.Render("✎ journal")+"  "+stats, m.width)
	keys := "↑↓ scroll · esc back · q garden"

	body := strings.Join(lines, "\n")
	if len(tasks) > 0 {
		body = strings.Join(tasks, "\n") + "\n" + body
	}
	return strings.Join([]string{head, m.divider(), body, m.divider(), m.footer(keys)}, "\n")
}
