package main

import (
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles (used by colorizeResult) ───────────────────────────────────────

var (
	numberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	unitStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimPlusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// ── expression resolution ──────────────────────────────────────────────────

// autoPrefixRe matches inputs that should be prepended with ANSWER.
// "-" requires trailing whitespace to distinguish operator from negative number.
var autoPrefixRe = regexp.MustCompile(`^[+*/^%]|^-\s|^to\s|^in\s`)

// substituteAnswer replaces all occurrences of "ANSWER" in expr with lastResult.
// Returns expr unchanged if lastResult is empty.
func substituteAnswer(expr, lastResult string) string {
	if lastResult == "" {
		return expr
	}
	return strings.ReplaceAll(expr, "ANSWER", lastResult)
}

// resolveExpr determines the actual qalc expression to evaluate and the display
// expression to show in history.
func resolveExpr(input, lastResult string) (expr, displayExpr string, isAuto bool) {
	trimmed := strings.TrimSpace(input)
	if lastResult != "" && autoPrefixRe.MatchString(trimmed) {
		return lastResult + " " + trimmed, "ANSWER " + trimmed, true
	}
	expr = substituteAnswer(trimmed, lastResult)
	return expr, trimmed, false
}

// ── output parsing ─────────────────────────────────────────────────────────

// parseResult parses a full qalc output line such as
// "100 kilometers ≈ 62 mi + 241 yd + 0.9895013123 ft"
// Returns display (everything after = or ≈) and answer (primary unit).
func parseResult(raw string) (display, answer string) {
	raw = strings.TrimSpace(raw)
	for _, sep := range []string{"≈", "="} {
		if idx := strings.LastIndex(raw, sep); idx >= 0 {
			display = strings.TrimSpace(raw[idx+len(sep):])
			answer = primaryUnit(display)
			return
		}
	}
	display = raw
	answer = raw
	return
}

// primaryUnit returns the first term of a compound result split on " + ".
func primaryUnit(result string) string {
	parts := strings.SplitN(result, " + ", 2)
	return strings.TrimSpace(parts[0])
}

// ── result colorization ────────────────────────────────────────────────────

var numUnitRe = regexp.MustCompile(`^([\-\d.,]+(?:\s*[eE][+\-]?\d+)?)\s*(.*)$`)

// colorizeResult applies ANSI colors: numbers bright cyan, units orange, "+" dark grey.
func colorizeResult(result string) string {
	parts := strings.Split(result, " + ")
	colored := make([]string, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		m := numUnitRe.FindStringSubmatch(part)
		if m != nil && strings.TrimSpace(m[2]) != "" {
			colored[i] = numberStyle.Render(m[1]) + " " + unitStyle.Render(strings.TrimSpace(m[2]))
		} else if m != nil {
			colored[i] = numberStyle.Render(m[1])
		} else {
			colored[i] = part
		}
	}
	return strings.Join(colored, dimPlusStyle.Render(" + "))
}

// ── messages ───────────────────────────────────────────────────────────────

type resultMsg struct {
	displayExpr string
	display     string
	answer      string
	isAuto      bool
}

type calcErrMsg struct {
	displayExpr string
	err         string
}

type previewMsg struct {
	id     int
	result string
}

type debounceMsg struct{ id int }

// ── qalc invocation ────────────────────────────────────────────────────────

// evalExpr resolves and evaluates the expression, returning full qalc output.
func evalExpr(input, lastResult string) tea.Cmd {
	return func() tea.Msg {
		expr, displayExpr, isAuto := resolveExpr(input, lastResult)
		out, err := exec.Command("qalc", expr).Output()
		if err != nil && len(out) == 0 {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}
		display, answer := parseResult(string(out))
		return resultMsg{
			displayExpr: displayExpr,
			display:     display,
			answer:      answer,
			isAuto:      isAuto,
		}
	}
}

// evalPreview calls qalc for a live preview. Stale results are discarded by id.
func evalPreview(input, lastResult string, id int) tea.Cmd {
	return func() tea.Msg {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return previewMsg{id: id}
		}
		expr, _, _ := resolveExpr(trimmed, lastResult)
		out, err := exec.Command("qalc", expr).Output()
		if err != nil && len(out) == 0 {
			return previewMsg{id: id}
		}
		display, _ := parseResult(string(out))
		return previewMsg{id: id, result: display}
	}
}
