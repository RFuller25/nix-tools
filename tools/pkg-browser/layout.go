package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// panelContentHeight returns list content rows available given fixed chrome.
// Fixed chrome: header(2) + bottom_divider(1) + help(1) = 4.
// When search open: also subtracts search_input(1) + search_divider(1) + min(results,5)(variable).
func panelContentHeight(m model) int {
	fixed := 4
	if m.searchOpen {
		searchRows := len(m.searchResult)
		if searchRows > 5 {
			searchRows = 5
		}
		fixed += 2 + searchRows
	}
	h := m.height - fixed
	if h < 1 {
		h = 1
	}
	return h
}

func renderPanels(m model) string {
	halfW := m.width / 2
	ph := panelContentHeight(m)

	leftLines := profilePanelLines(m, halfW-1, ph)
	rightLines := configPanelLines(m, m.width-halfW-1, ph)

	sep := divStyle.Render("│")
	var b strings.Builder
	for i := 0; i < ph; i++ {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}
		b.WriteString(padRight(left, halfW-1) + sep + right + "\n")
	}
	return b.String()
}

func profilePanelLines(m model, width, height int) []string {
	if m.profileErr != nil {
		return []string{errStyle.Render("error: " + m.profileErr.Error())}
	}
	if m.loading && len(m.profilePkgs) == 0 {
		return []string{m.spinner.View() + " loading..."}
	}
	if len(m.profilePkgs) == 0 {
		return []string{descStyle.Render("no profile packages")}
	}
	return pkgLines(m.profilePkgs, m.profileCursor, m.focus == panelProfile, width, height)
}

func configPanelLines(m model, width, height int) []string {
	if m.configPath == "" {
		return []string{descStyle.Render("no config loaded")}
	}
	if m.configErr != nil {
		return []string{errStyle.Render("error: " + m.configErr.Error())}
	}
	if len(m.configPkgs) == 0 {
		return []string{descStyle.Render("no packages in config")}
	}
	lines := make([]string, 0)
	start, end := visibleWindow(m.configCursor, height, len(m.configPkgs))
	for i := start; i < end; i++ {
		p := m.configPkgs[i]
		var name string
		if p.ReadOnly {
			name = descStyle.Render("[" + p.Name + "…]")
		} else {
			name = normalRow.Render(p.Name)
		}
		if i == m.configCursor && m.focus == panelConfig {
			lines = append(lines, selectedRow.Render("> ")+name)
		} else {
			lines = append(lines, "  "+name)
		}
	}
	return lines
}

func pkgLines(pkgs []pkg, cursor int, focused bool, width, height int) []string {
	lines := make([]string, 0)
	start, end := visibleWindow(cursor, height, len(pkgs))
	for i := start; i < end; i++ {
		p := pkgs[i]
		nameVer := p.Name
		if p.Version != "" {
			nameVer += " " + versionStyle.Render(p.Version)
		}
		line := nameVer
		if p.Description != "" {
			line += "  " + descStyle.Render(p.Description)
		}
		if i == cursor && focused {
			lines = append(lines, selectedRow.Render("> ")+line)
		} else {
			lines = append(lines, "  "+normalRow.Render(line))
		}
	}
	return lines
}

func renderSearchPanel(m model) string {
	if m.searchErr != nil {
		return errStyle.Render("search error: "+m.searchErr.Error()) + "\n"
	}
	if m.loading {
		return m.spinner.View() + " searching...\n"
	}
	if len(m.searchResult) == 0 {
		return descStyle.Render("type a query and press enter") + "\n"
	}
	maxRows := 5
	lines := pkgLines(m.searchResult, m.searchCursor, true, m.width, maxRows)
	return strings.Join(lines, "\n") + "\n"
}

func visibleWindow(cursor, height, total int) (start, end int) {
	if total == 0 {
		return 0, 0
	}
	start = 0
	if cursor >= height {
		start = cursor - height + 1
	}
	end = start + height
	if end > total {
		end = total
	}
	return
}

func padRight(s string, width int) string {
	vis := lipgloss.Width(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}
