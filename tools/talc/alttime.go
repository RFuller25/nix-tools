package main

import (
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// altTimeRe matches "to <format>" at the end of an expression.
var altTimeRe = regexp.MustCompile(`(?i)\bto\s+(kak(?:tovik)?|chr(?:on)?|duod?)\s*$`)

// secNumRe extracts a numeric value followed by "s" (seconds) from qalc output.
var secNumRe = regexp.MustCompile(`^([\d,]+(?:\.\d+)?(?:[eE][+-]?\d+)?)\s*s\b`)

// ── numeral systems ────────────────────────────────────────────────────────

func kakDigit(n int) string {
	// Kaktovik Inupiaq numerals: U+1D2C0 (0) … U+1D2D3 (19)
	if n < 0 || n > 19 {
		return "?"
	}
	return string(rune(0x1D2C0 + n))
}

func duodDigit(n int) string {
	const dek = '↊' // ↊
	const el = '↋'  // ↋
	switch {
	case n >= 0 && n <= 9:
		return string(rune('0' + n))
	case n == 10:
		return string(dek)
	case n == 11:
		return string(el)
	}
	return "?"
}

// ── format functions ───────────────────────────────────────────────────────

func formatKaktovik(totalSecs float64) string {
	secs := math.Mod(totalSecs, 86400)
	if secs < 0 {
		secs += 86400
	}
	// 20 h/day, 20 min/h, 20 s/min → 8000 ticks/day
	ticks := int(secs * 8000 / 86400)
	h := ticks / 400
	m := (ticks % 400) / 20
	s := ticks % 20
	return kakDigit(h) + " : " + kakDigit(m) + " : " + kakDigit(s)
}

func formatChron(totalSecs float64) string {
	secs := math.Mod(totalSecs, 86400)
	if secs < 0 {
		secs += 86400
	}
	// 10 h/day, 100 min/h, 100 s/min → 100,000 ticks/day
	ticks := int(secs * 100000 / 86400)
	h := ticks / 10000
	m := (ticks % 10000) / 100
	s := ticks % 100
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

func formatDuod(totalSecs float64) string {
	secs := math.Mod(totalSecs, 86400)
	if secs < 0 {
		secs += 86400
	}
	// 12 h/day, 12 min/h, 12 s/min → 1728 ticks/day
	ticks := int(secs * 1728 / 86400)
	h := ticks / 144
	m := (ticks % 144) / 12
	s := ticks % 12
	return duodDigit(h) + ":" + duodDigit(m) + ":" + duodDigit(s)
}

// ── helpers ────────────────────────────────────────────────────────────────

func parseSecondsFromDisplay(display string) (float64, error) {
	m := secNumRe.FindStringSubmatch(strings.TrimSpace(display))
	if m == nil {
		return 0, fmt.Errorf("result not in seconds: %q", display)
	}
	numStr := strings.ReplaceAll(m[1], ",", "")
	return strconv.ParseFloat(numStr, 64)
}

func resolveAltTime(expr string) (base, target string, ok bool) {
	loc := altTimeRe.FindStringSubmatchIndex(expr)
	if loc == nil {
		return "", "", false
	}
	return strings.TrimSpace(expr[:loc[0]]), strings.ToLower(expr[loc[2]:loc[3]]), true
}

func applyAltTimeFormat(target string, secs float64) (string, error) {
	switch {
	case target == "kak" || target == "kaktovik":
		return formatKaktovik(secs), nil
	case target == "chr" || target == "chron":
		return formatChron(secs), nil
	case target == "duo" || target == "duod":
		return formatDuod(secs), nil
	}
	return "", fmt.Errorf("unknown time format: %s", target)
}

func toSecondsDisplay(base string) (display, secStr string, err error) {
	out, cmdErr := exec.Command("qalc", base+" to s").Output()
	if cmdErr != nil && len(out) == 0 {
		return "", "", cmdErr
	}
	display, secStr = parseResult(string(out))
	return display, secStr, nil
}

// ── tea commands ───────────────────────────────────────────────────────────

// evalAltTimeCmd returns a tea.Cmd if expr ends with a known alt-time target,
// otherwise returns nil (caller should proceed with normal qalc evaluation).
func evalAltTimeCmd(expr, displayExpr string, isAuto bool) tea.Cmd {
	base, target, ok := resolveAltTime(expr)
	if !ok {
		return nil
	}
	return func() tea.Msg {
		secDisplay, secStr, err := toSecondsDisplay(base)
		if err != nil {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}
		secs, err := parseSecondsFromDisplay(secDisplay)
		if err != nil {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}
		formatted, err := applyAltTimeFormat(target, secs)
		if err != nil {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}
		return resultMsg{
			displayExpr: displayExpr,
			display:     formatted,
			answer:      secStr, // ANS holds seconds for further math
			isAuto:      isAuto,
		}
	}
}

// previewAltTime returns the formatted alt-time string for live preview,
// or "" if not applicable or on any error.
func previewAltTime(expr string) string {
	base, target, ok := resolveAltTime(expr)
	if !ok {
		return ""
	}
	secDisplay, _, err := toSecondsDisplay(base)
	if err != nil {
		return ""
	}
	secs, err := parseSecondsFromDisplay(secDisplay)
	if err != nil {
		return ""
	}
	formatted, err := applyAltTimeFormat(target, secs)
	if err != nil {
		return ""
	}
	return formatted
}
