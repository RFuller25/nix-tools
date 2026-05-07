package main

import (
	"strings"
)

type configPkg struct {
	Name     string
	ReadOnly bool
	lineIdx  int
}

// parseConfigPackages extracts packages from environment.systemPackages in a NixOS config.
// Simple bare-name entries are editable; entries starting with '(' are read-only.
func parseConfigPackages(src []byte) []configPkg {
	lines := strings.Split(string(src), "\n")

	blockStart := -1
	for i, line := range lines {
		if strings.Contains(line, "environment.systemPackages") && strings.Contains(line, "[") {
			blockStart = i
			break
		}
	}
	if blockStart < 0 {
		return nil
	}

	var result []configPkg
	bracketDepth := 1
	parenDepth := 0

	for i := blockStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Strip inline comment BEFORE depth counting
		scanLine := trimmed
		if idx := strings.Index(scanLine, " #"); idx >= 0 {
			scanLine = scanLine[:idx]
		}

		// Snapshot depth before processing this line's characters:
		// we want to know if this entry started at top level, not where it ended.
		atTopLevel := bracketDepth == 1 && parenDepth == 0

		for _, ch := range scanLine {
			switch ch {
			case '[':
				bracketDepth++
			case ']':
				bracketDepth--
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			}
		}

		if bracketDepth <= 0 {
			break
		}
		if !atTopLevel {
			continue
		}

		// Strip inline comment for name extraction
		nameLine := trimmed
		if idx := strings.Index(nameLine, " #"); idx >= 0 {
			nameLine = strings.TrimSpace(nameLine[:idx])
		}

		if strings.HasPrefix(nameLine, "(") {
			// Complex: extract first identifier as display name
			inner := strings.TrimPrefix(nameLine, "(")
			name := inner
			for _, sep := range []string{".", " ", "("} {
				if idx := strings.Index(name, sep); idx >= 0 {
					name = name[:idx]
				}
			}
			result = append(result, configPkg{Name: strings.TrimSpace(name), ReadOnly: true, lineIdx: i})
		} else {
			result = append(result, configPkg{Name: nameLine, ReadOnly: false, lineIdx: i})
		}
	}
	return result
}

// addPackage inserts name as a new line before the closing ] of environment.systemPackages.
func addPackage(src []byte, name string) []byte {
	for _, p := range parseConfigPackages(src) {
		if p.Name == name {
			return src
		}
	}

	lines := strings.Split(string(src), "\n")

	blockStart := -1
	for i, line := range lines {
		if strings.Contains(line, "environment.systemPackages") && strings.Contains(line, "[") {
			blockStart = i
			break
		}
	}
	if blockStart < 0 {
		return src
	}

	depth := 1
	for i := blockStart + 1; i < len(lines); i++ {
		scanLine := lines[i]
		if idx := strings.Index(scanLine, " #"); idx >= 0 {
			scanLine = scanLine[:idx]
		}
		for _, ch := range scanLine {
			switch ch {
			case '[':
				depth++
			case ']':
				depth--
			}
		}
		if depth <= 0 {
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:i]...)
			result = append(result, "    "+name)
			result = append(result, lines[i:]...)
			return []byte(strings.Join(result, "\n"))
		}
	}
	return src
}

// removePackage removes the line for name from environment.systemPackages.
// No-op if name is not found or is a complex (ReadOnly) entry.
func removePackage(src []byte, name string) []byte {
	pkgs := parseConfigPackages(src)
	lineIdx := -1
	for _, p := range pkgs {
		if p.Name == name && !p.ReadOnly {
			lineIdx = p.lineIdx
			break
		}
	}
	if lineIdx < 0 {
		return src
	}
	lines := strings.Split(string(src), "\n")
	result := make([]string, 0, len(lines)-1)
	for i, line := range lines {
		if i != lineIdx {
			result = append(result, line)
		}
	}
	return []byte(strings.Join(result, "\n"))
}
