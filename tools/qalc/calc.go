package main

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func substituteAnswer(expr, lastResult string) string {
	if lastResult == "" {
		return expr
	}
	return strings.ReplaceAll(expr, "ANSWER", lastResult)
}

func parseQalcOutput(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.LastIndex(raw, "="); idx >= 0 {
		return strings.TrimSpace(raw[idx+1:])
	}
	return raw
}

type resultMsg struct {
	expr   string
	result string
}

type calcErrMsg struct {
	expr string
	err  string
}

func evalExpr(expr string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("qalc", "--terse", expr).Output()
		if err != nil {
			if len(out) > 0 {
				return resultMsg{expr: expr, result: parseQalcOutput(string(out))}
			}
			return calcErrMsg{expr: expr, err: err.Error()}
		}
		return resultMsg{expr: expr, result: parseQalcOutput(string(out))}
	}
}
