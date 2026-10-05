package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleSt = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hl      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	selSt   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	goodSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	badSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

func (m model) View() string {
	if m.needsSetup {
		return m.setup.View()
	}
	var body string
	switch m.screen {
	case scBoard:
		body = m.viewBoard()
	case scDetail:
		body = m.viewDetail()
	case scCreate:
		body = m.viewCreate()
	case scStake:
		body = m.viewStake()
	case scResolve:
		body = m.viewResolve()
	}
	return m.header() + "\n" + m.tabBar() + "\n\n" + body + "\n" + m.footer()
}

func (m model) header() string {
	right := fmt.Sprintf("%s  %d BBs", m.cfg.Username, m.balance)
	left := titleSt.Render("ledger")
	gap := max(m.width-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	return left + strings.Repeat(" ", gap) + hl.Render(right)
}

func (m model) tabBar() string {
	if m.screen != scBoard {
		return ""
	}
	names := []string{"1 Board", "2 Won"}
	out := make([]string, len(names))
	for i, n := range names {
		if i == m.tab {
			out[i] = selSt.Render("[" + n + "]")
		} else {
			out[i] = dim.Render(" " + n + " ")
		}
	}
	return strings.Join(out, " ")
}

func (m model) footer() string {
	var s string
	switch {
	case m.busy:
		s = m.spin.View() + dim.Render(" working...")
	case m.flash != "" && m.failed:
		s = badSt.Render(m.flash)
	case m.flash != "":
		s = goodSt.Render(m.flash)
	}
	return "\n" + s + "\n" + dim.Render(m.help())
}

func (m model) help() string {
	switch m.screen {
	case scBoard:
		return "↑↓ move · enter open · tab switch · n new bet · r refresh · q quit"
	case scDetail:
		h := "esc back"
		if m.detail != nil && m.detail.Status == statusOpen {
			h += " · b bet"
			if m.isMine(&m.detail.Bet) {
				h += " · r resolve"
			}
		}
		return h
	case scResolve:
		if m.rConfirm {
			return "y confirm · any other key cancels"
		}
		return "↑↓ choose · enter select · esc cancel"
	default:
		return "enter next · esc cancel"
	}
}

func stakedTotal(b Bet) int { return sum(b.Mine) }

// outcomeTag says what happened to the bet, the same for everyone.
func outcomeTag(b Bet) string {
	switch b.Status {
	case statusResolved:
		if b.Winner >= 0 && b.Winner < len(b.Options) {
			return "outcome: " + b.Options[b.Winner].Label
		}
		return "resolved"
	case statusVoid:
		return "void"
	}
	return ""
}

// resultTag says how a closed bet went for this user, and nothing for an
// open one. Winning the bet is not the same as winning the outcome: it is
// only a win if the user backed it.
func resultTag(b Bet) string {
	if b.Status == statusOpen {
		return ""
	}
	staked := stakedTotal(b)
	switch {
	case staked == 0:
		return dim.Render("no stake")
	case b.Status == statusVoid:
		return dim.Render(fmt.Sprintf("refunded %d", b.Paid))
	case b.Paid > 0:
		return goodSt.Render(fmt.Sprintf("you won %+d", b.Paid-staked))
	default:
		return badSt.Render(fmt.Sprintf("you lost %d", staked))
	}
}

func (m model) viewBoard() string {
	if m.tab == tabWon {
		return m.viewWon()
	}
	if len(m.bets) == 0 {
		if m.busy {
			return dim.Render("loading the board...")
		}
		return dim.Render("no bets yet. press n to start one.")
	}
	visible := max(m.height-9, 3)
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := min(start+visible, len(m.bets))
	w := max(m.width-4, 20)

	var b strings.Builder
	for i := start; i < end; i++ {
		bet := m.bets[i]
		mark := " "
		if m.isMine(&bet) {
			mark = "★"
		}
		meta := fmt.Sprintf("%d BBs", bet.Pool)
		if tag := resultTag(bet); tag != "" {
			meta += " · " + tag
		} else if bet.Status == statusOpen && stakedTotal(bet) > 0 {
			meta += fmt.Sprintf(" · in for %d", stakedTotal(bet))
		}
		room := max(w-lipgloss.Width(meta)-6, 8)
		title := truncate(bet.Title, room)
		line := fmt.Sprintf("%s %s", mark, title)
		pad := max(w-lipgloss.Width(line)-lipgloss.Width(meta), 2)
		row := line + strings.Repeat(" ", pad) + dim.Render(meta)
		if i == m.cursor {
			row = selSt.Render("▸ ") + hl.Render(line) + strings.Repeat(" ", pad) + dim.Render(meta)
		} else {
			row = "  " + row
		}
		b.WriteString(row + "\n")
	}
	if len(m.bets) > visible {
		b.WriteString(dim.Render(fmt.Sprintf("  %d/%d", m.cursor+1, len(m.bets))) + "\n")
	}
	return b.String()
}

// viewWon lists the bets this user backed to a win, with what each paid.
func (m model) viewWon() string {
	if len(m.won) == 0 {
		if m.busy {
			return dim.Render("loading your wins...")
		}
		return dim.Render("no wins yet.")
	}
	total := 0
	for _, b := range m.won {
		total += b.Paid - stakedTotal(b)
	}
	visible := max(m.height-11, 3)
	start := 0
	if m.wonCursor >= visible {
		start = m.wonCursor - visible + 1
	}
	end := min(start+visible, len(m.won))
	w := max(m.width-4, 20)

	var b strings.Builder
	b.WriteString(dim.Render(fmt.Sprintf("%d wins · %+d BBs net", len(m.won), total)) + "\n\n")
	for i := start; i < end; i++ {
		bet := m.won[i]
		when := ""
		if bet.Resolved > 0 {
			when = time.Unix(bet.Resolved, 0).Format("Jan 2")
		}
		meta := fmt.Sprintf("%s  staked %d → paid %d (%+d)", when, stakedTotal(bet), bet.Paid, bet.Paid-stakedTotal(bet))
		room := max(w-lipgloss.Width(meta)-6, 8)
		line := truncate(bet.Title, room)
		pad := max(w-lipgloss.Width(line)-lipgloss.Width(meta), 2)
		if i == m.wonCursor {
			b.WriteString(selSt.Render("▸ ") + hl.Render(line) + strings.Repeat(" ", pad) + goodSt.Render(meta) + "\n")
		} else {
			b.WriteString("  " + line + strings.Repeat(" ", pad) + dim.Render(meta) + "\n")
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func (m model) viewDetail() string {
	d := m.detail
	if d == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(hl.Render(d.Title) + "\n")
	by := "by " + d.Creator + " · " + time.Unix(d.Created, 0).Format("Jan 2 15:04")
	if t := outcomeTag(d.Bet); t != "" {
		by += " · " + t
	}
	b.WriteString(dim.Render(by) + "\n")
	if r := resultTag(d.Bet); r != "" {
		b.WriteString(r + "\n")
	}
	b.WriteString("\n")

	for i, o := range d.Options {
		mine := ""
		if i < len(d.Mine) && d.Mine[i] > 0 {
			mine = fmt.Sprintf("  you: %d", d.Mine[i])
		}
		mark := "●"
		if d.Status == statusResolved && d.Winner == i {
			mark = "★"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(optColor(i)).Render(mark) + " " +
			fmt.Sprintf("%-16s %5d BBs  %s", truncate(o.Label, 16), o.Total, oddsLine(o.Total, d.Pool)) + dim.Render(mine) + "\n")
	}
	b.WriteString(dim.Render(fmt.Sprintf("pool %d BBs", d.Pool)) + "\n\n")

	h := min(max(m.height-len(d.Options)-14, 5), 12)
	if m.histWait {
		b.WriteString(dim.Render("loading graph...") + "\n")
	} else {
		b.WriteString(renderGraph(d.History, len(d.Options), min(m.width-2, 90), h) + "\n")
	}
	return b.String()
}

func (m model) viewCreate() string {
	var b strings.Builder
	b.WriteString(hl.Render("New bet") + "\n\n")
	b.WriteString("Question: " + m.cTitle)
	if m.cStep == csTitle {
		b.Reset()
		b.WriteString(hl.Render("New bet") + "\n\nQuestion:\n" + m.input.View() + "\n")
		return b.String()
	}
	b.WriteString("\n\nOutcomes:\n")
	for i, l := range m.cLabels {
		b.WriteString(lipgloss.NewStyle().Foreground(optColor(i)).Render("● ") + l + "\n")
	}
	switch m.cStep {
	case csOptions:
		b.WriteString("\n" + m.input.View() + "\n")
	case csSide:
		b.WriteString("\nYour side:\n")
		b.WriteString(m.optionList(m.cLabels, m.cSide))
	case csAmount:
		b.WriteString(fmt.Sprintf("\nYour side: %s\n\nStake (min %d, you have %d):\n%s\n",
			m.cLabels[m.cSide], minCreate, m.balance, m.input.View()))
	}
	return b.String()
}

func (m model) optionList(labels []string, cur int) string {
	var b strings.Builder
	for i, l := range labels {
		dot := lipgloss.NewStyle().Foreground(optColor(i)).Render("●")
		if i == cur {
			b.WriteString(selSt.Render("▸ ") + dot + " " + hl.Render(l) + "\n")
		} else {
			b.WriteString("  " + dot + " " + l + "\n")
		}
	}
	return b.String()
}

func (m model) viewStake() string {
	d := m.detail
	labels := make([]string, len(d.Options))
	for i, o := range d.Options {
		labels[i] = o.Label
	}
	s := hl.Render("Bet on: "+d.Title) + "\n\n"
	if m.sStep == ssOption {
		return s + "Pick a side:\n" + m.optionList(labels, m.sOption)
	}
	return s + fmt.Sprintf("Side: %s\n\nAmount:\n%s\n", labels[m.sOption], m.input.View())
}

func (m model) viewResolve() string {
	d := m.detail
	labels := make([]string, 0, len(d.Options)+1)
	for _, o := range d.Options {
		labels = append(labels, o.Label)
	}
	labels = append(labels, "VOID (refund everyone)")
	s := hl.Render("Resolve: "+d.Title) + "\n\nWhat was the outcome?\n" + m.optionList(labels, m.rOption)
	if m.rConfirm {
		s += "\n" + badSt.Render(fmt.Sprintf("Settle as %q? This can't be undone. (y/n)", labels[m.rOption]))
	}
	return s
}
