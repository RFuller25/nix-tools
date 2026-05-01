package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type pkg struct {
	Name        string
	Version     string
	Description string
}

func parseProfileBlock(block string) *pkg {
	var attrLine, storeLine string
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "Flake attribute:") {
			attrLine = strings.TrimSpace(strings.TrimPrefix(line, "Flake attribute:"))
		}
		if strings.HasPrefix(line, "Store paths:") {
			storeLine = strings.TrimSpace(strings.TrimPrefix(line, "Store paths:"))
		}
	}
	if attrLine == "" {
		return nil
	}

	var name, version string
	if storeLine != "" {
		name, version = extractNameVersion(storeLine)
	}
	if name == "" {
		parts := strings.Split(attrLine, ".")
		name = parts[len(parts)-1]
	}

	return &pkg{Name: name, Version: version}
}

var storeRe = regexp.MustCompile(`/nix/store/[^-]+-(.+)`)

func extractNameVersion(storePath string) (name, version string) {
	m := storeRe.FindStringSubmatch(storePath)
	if m == nil {
		return "", ""
	}
	rest := m[1]
	idx := strings.LastIndex(rest, "-")
	if idx > 0 {
		tail := rest[idx+1:]
		if len(tail) > 0 && tail[0] >= '0' && tail[0] <= '9' {
			return rest[:idx], tail
		}
	}
	return rest, ""
}

func parseSearchJSON(data []byte) ([]pkg, error) {
	var raw map[string]struct {
		Pname       string `json:"pname"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse nix search output: %w", err)
	}
	pkgs := make([]pkg, 0, len(raw))
	for key, v := range raw {
		name := v.Pname
		if name == "" {
			parts := strings.Split(key, ".")
			name = parts[len(parts)-1]
		}
		pkgs = append(pkgs, pkg{
			Name:        name,
			Version:     v.Version,
			Description: v.Description,
		})
	}
	return pkgs, nil
}

type installedLoadedMsg struct{ pkgs []pkg }
type searchResultMsg struct{ pkgs []pkg }
type nixErrMsg struct{ err error }

func loadInstalled() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("nix", "profile", "list")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nixErrMsg{fmt.Errorf("%s", msg)}
		}
		blocks := strings.Split(strings.TrimSpace(string(out)), "\n\n")
		var pkgs []pkg
		for _, b := range blocks {
			if p := parseProfileBlock(b); p != nil {
				pkgs = append(pkgs, *p)
			}
		}
		return installedLoadedMsg{pkgs}
	}
}

func searchNixpkgs(query string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("nix", "search", "nixpkgs", query, "--json")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nixErrMsg{fmt.Errorf("%s", msg)}
		}
		pkgs, err := parseSearchJSON(out)
		if err != nil {
			return nixErrMsg{err}
		}
		return searchResultMsg{pkgs}
	}
}
