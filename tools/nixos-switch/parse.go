package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── styles ──────────────────────────────────────────────────────────────────

var (
	_styleBuilding   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	_styleFetching   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	_styleActivating = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	_styleError      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	_styleDefault    = lipgloss.NewStyle()
	_styleChanged    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	_styleAdded      = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	_styleRemoved    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	_styleSection    = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)

	styleBuilding   = &_styleBuilding
	styleFetching   = &_styleFetching
	styleActivating = &_styleActivating
	styleError      = &_styleError
	styleDefault    = &_styleDefault
	styleChanged    = &_styleChanged
	styleAdded      = &_styleAdded
	styleRemoved    = &_styleRemoved
	styleSection    = &_styleSection
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func categorizeLine(line string) (*lipgloss.Style, string) {
	switch {
	case strings.HasPrefix(line, "[U."):
		return styleChanged, line
	case strings.HasPrefix(line, "[A."):
		return styleAdded, line
	case strings.HasPrefix(line, "[R."):
		return styleRemoved, line
	case strings.HasPrefix(line, "> "):
		return styleSection, line
	case strings.HasPrefix(line, "building '"):
		return styleBuilding, line
	case strings.HasPrefix(line, "copying path '"):
		return styleFetching, line
	case strings.HasPrefix(line, "activating "):
		return styleActivating, line
	case strings.HasPrefix(line, "error:"), strings.HasPrefix(line, "Error:"):
		return styleError, line
	default:
		return styleDefault, line
	}
}

var (
	reFetchPaths = regexp.MustCompile(`these (\d+) paths? will be fetched \(([\d.]+) (MiB|GiB) download`)
	reFetchDrvs  = regexp.MustCompile(`these (\d+) derivations? will be built`)
	reDiskUsage  = regexp.MustCompile(`disk usage ([-+][\d.]+\s*(?:MiB|GiB))`)
)

func updateStats(line string, s *stats) {
	if strings.HasPrefix(line, "> ") {
		lower := strings.ToLower(line[2:])
		switch {
		case strings.Contains(lower, "activat"):
			s.phase = "activating"
		case strings.Contains(lower, "build"):
			s.phase = "building"
		}
		return
	}

	switch {
	case strings.HasPrefix(line, "[U."):
		s.pkgsChanged++
		return
	case strings.HasPrefix(line, "[A."):
		s.pkgsAdded++
		return
	case strings.HasPrefix(line, "[R."):
		s.pkgsRemoved++
		return
	}

	if m := reDiskUsage.FindStringSubmatch(line); m != nil {
		s.diskDelta = strings.TrimSpace(m[1])
		return
	}

	if m := reFetchPaths.FindStringSubmatch(line); m != nil {
		n, _ := strconv.Atoi(m[1])
		mib, _ := strconv.ParseFloat(m[2], 64)
		if m[3] == "GiB" {
			mib *= 1024
		}
		s.totalPaths = n
		s.totalMiB = mib
		s.phase = "fetching"
		return
	}

	if m := reFetchDrvs.FindStringSubmatch(line); m != nil {
		n, _ := strconv.Atoi(m[1])
		s.totalDrvs = n
		return
	}

	if strings.HasPrefix(line, "copying path '") {
		s.fetchedPaths++
		return
	}

	if strings.HasPrefix(line, "building '") {
		s.builtDrvs++
		if s.phase != "activating" {
			s.phase = "building"
		}
		return
	}

	if strings.HasPrefix(line, "activating ") {
		s.phase = "activating"
		return
	}
}
