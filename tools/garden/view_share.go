package main

import (
	"fmt"
	"strings"
)

// viewShare shows the code for a cultivar, to be copied into a message.
func (m model) viewShare() string {
	w := max(20, min(m.width-4, 64))
	lines := []string{
		titleStyle.Render("❦ share ‘" + m.shareName + "’"),
		m.divider(),
		"",
		valueStyle.Render("Give this to a friend. They paste it in their seed shed (i) and get"),
		valueStyle.Render(fmt.Sprintf("%d seeds of the line, and it goes in their almanac under your name.", giftSeeds)),
		"",
	}
	// Break the code on its dashes so no group is split across lines.
	line := ""
	for _, part := range strings.Split(m.shareCode, "-") {
		if line != "" && len(line)+1+len(part) > w {
			lines = append(lines, goldStyle.Render(line+"-"))
			line = part
			continue
		}
		if line != "" {
			line += "-"
		}
		line += part
	}
	lines = append(lines, goldStyle.Render(line), "")
	lines = append(lines, wrapped(subtleStyle, "It has been sent to your clipboard if your terminal allows that; otherwise select it with the mouse. Each time you share you make a new code. A code works once in a garden, and not in this one.", w)...)
	lines = append(lines, "", m.divider(), helpStyle.Render("any key goes back"))
	for i := range lines {
		lines[i] = fit(lines[i], m.width)
	}
	return strings.Join(lines, "\n")
}
