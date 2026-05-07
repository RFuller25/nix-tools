package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
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

var ansIndexRe = regexp.MustCompile(`(?i)\b(ans|answer)\(([0-9]+)\)`)
var ansPlainRe = regexp.MustCompile(`(?i)\b(ans|answer)\b`)

// substituteAnswer replaces ANS(x)/ANSWER(x) with the x-th previous answer
// (1=most recent) and bare ANS/ANSWER with the most recent answer.
func substituteAnswer(expr string, answers []string) (string, error) {
	var subErr error
	result := ansIndexRe.ReplaceAllStringFunc(expr, func(match string) string {
		sub := ansIndexRe.FindStringSubmatch(match)
		x, _ := strconv.Atoi(sub[2])
		if x <= 0 {
			subErr = fmt.Errorf("ANS index must be ≥ 1")
			return match
		}
		if x > len(answers) {
			subErr = fmt.Errorf("ANS(%d) out of range (only %d answer(s))", x, len(answers))
			return match
		}
		return answers[x-1]
	})
	if subErr != nil {
		return "", subErr
	}
	if len(answers) > 0 {
		result = ansPlainRe.ReplaceAllString(result, answers[0])
	}
	return result, nil
}

// resolveExpr determines the actual qalc expression to evaluate and the display
// expression to show in history.
func resolveExpr(input string, answers []string) (expr, displayExpr string, isAuto bool, err error) {
	trimmed := strings.TrimSpace(input)
	lastResult := ""
	if len(answers) > 0 {
		lastResult = answers[0]
	}
	if lastResult != "" && autoPrefixRe.MatchString(trimmed) {
		return lastResult + " " + trimmed, "ANSWER " + trimmed, true, nil
	}
	expr, err = substituteAnswer(trimmed, answers)
	if err != nil {
		return "", trimmed, false, err
	}
	return expr, trimmed, false, nil
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
func evalExpr(input string, answers []string) tea.Cmd {
	return func() tea.Msg {
		expr, displayExpr, isAuto, err := resolveExpr(input, answers)
		if err != nil {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}
		if cmd := evalAltTimeCmd(expr, displayExpr, isAuto); cmd != nil {
			return cmd()
		}
		out, cmdErr := exec.Command("qalc", expr).Output()
		if cmdErr != nil && len(out) == 0 {
			return calcErrMsg{displayExpr: displayExpr, err: cmdErr.Error()}
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
func evalPreview(input string, answers []string, id int) tea.Cmd {
	return func() tea.Msg {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return previewMsg{id: id}
		}
		expr, _, _, err := resolveExpr(trimmed, answers)
		if err != nil {
			return previewMsg{id: id}
		}
		if preview := previewAltTime(expr); preview != "" {
			return previewMsg{id: id, result: preview}
		}
		out, cmdErr := exec.Command("qalc", expr).Output()
		if cmdErr != nil && len(out) == 0 {
			return previewMsg{id: id}
		}
		display, _ := parseResult(string(out))
		return previewMsg{id: id, result: display}
	}
}
