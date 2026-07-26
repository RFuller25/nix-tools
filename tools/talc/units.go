package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var superscriptDigits = map[rune]rune{
	'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4',
	'⁵': '5', '⁶': '6', '⁷': '7', '⁸': '8', '⁹': '9',
}

const superscriptMinus = '⁻'

func parseBaseUnit(qalcOutput string) (relation, baseSymbol string, exponent int, err error) {
	display, _ := parseResult(qalcOutput)
	fields := strings.Fields(display)
	if len(fields) != 2 {
		return "", "", 0, fmt.Errorf("expected a single unit term, got %q", display)
	}
	relation = fields[0]
	token := fields[1]
	if strings.ContainsAny(token, "/·") {
		return "", "", 0, fmt.Errorf("compound unit not supported: %q", token)
	}
	runes := []rune(token)
	i := len(runes)
	for i > 0 {
		r := runes[i-1]
		_, isDigit := superscriptDigits[r]
		if !isDigit && r != superscriptMinus {
			break
		}
		i--
	}
	baseSymbol = string(runes[:i])
	if baseSymbol == "" {
		return "", "", 0, fmt.Errorf("empty unit symbol in %q", token)
	}
	expSuffix := runes[i:]
	if len(expSuffix) == 0 {
		return relation, baseSymbol, 1, nil
	}
	var sb strings.Builder
	for _, r := range expSuffix {
		if r == superscriptMinus {
			sb.WriteRune('-')
			continue
		}
		sb.WriteRune(superscriptDigits[r])
	}
	exponent, convErr := strconv.Atoi(sb.String())
	if convErr != nil {
		return "", "", 0, fmt.Errorf("bad exponent suffix %q: %w", string(expSuffix), convErr)
	}
	return relation, baseSymbol, exponent, nil
}

var unitCmdRe = regexp.MustCompile(`(?i)^unit\s+(.+)$`)
var unitDeleteRe = regexp.MustCompile(`(?i)^delete\s+(\S+)\s*$`)
var unitDefineRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+)$`)

type unitCmdKind int

const (
	unitCmdDefine unitCmdKind = iota
	unitCmdList
	unitCmdDelete
	unitCmdInvalid
)

type unitCmd struct {
	kind unitCmdKind
	name string
	expr string
}

func resolveUnitCmd(input string) (unitCmd, bool) {
	m := unitCmdRe.FindStringSubmatch(strings.TrimSpace(input))
	if m == nil {
		return unitCmd{}, false
	}
	rest := strings.TrimSpace(m[1])
	if strings.EqualFold(rest, "list") {
		return unitCmd{kind: unitCmdList}, true
	}
	if dm := unitDeleteRe.FindStringSubmatch(rest); dm != nil {
		return unitCmd{kind: unitCmdDelete, name: dm[1]}, true
	}
	if dfm := unitDefineRe.FindStringSubmatch(rest); dfm != nil {
		return unitCmd{kind: unitCmdDefine, name: dfm[1], expr: strings.TrimSpace(dfm[2])}, true
	}
	return unitCmd{kind: unitCmdInvalid}, true
}

type unitDefsDoc struct {
	XMLName xml.Name    `xml:"QALCULATE"`
	Version string      `xml:"version,attr"`
	Units   []unitEntry `xml:"unit"`
}

type unitEntry struct {
	Type  string      `xml:"type,attr"`
	Names string      `xml:"names"`
	Base  unitBaseDef `xml:"base"`
}

type unitBaseDef struct {
	Unit     string `xml:"unit"`
	Relation string `xml:"relation"`
	Exponent int    `xml:"exponent"`
}

func unitDefsPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "qalculate", "definitions", "units.xml"), nil
}

func loadUnitDefs(path string) (unitDefsDoc, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return unitDefsDoc{Version: "5.11.0"}, nil
	}
	if err != nil {
		return unitDefsDoc{}, err
	}
	var doc unitDefsDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return unitDefsDoc{}, err
	}
	return doc, nil
}

func saveUnitDefs(path string, doc unitDefsDoc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if doc.Version == "" {
		doc.Version = "5.11.0"
	}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append([]byte(xml.Header), out...)
	return os.WriteFile(path, out, 0o644)
}

func addUnitEntry(doc unitDefsDoc, name, baseSymbol, relation string, exponent int) (unitDefsDoc, error) {
	for _, u := range doc.Units {
		if strings.EqualFold(u.Names, name) {
			return doc, fmt.Errorf("%s: already defined", name)
		}
	}
	doc.Units = append(doc.Units, unitEntry{
		Type:  "alias",
		Names: name,
		Base: unitBaseDef{
			Unit:     baseSymbol,
			Relation: relation,
			Exponent: exponent,
		},
	})
	return doc, nil
}

func deleteUnitEntry(doc unitDefsDoc, name string) (unitDefsDoc, bool) {
	for i, u := range doc.Units {
		if strings.EqualFold(u.Names, name) {
			doc.Units = append(doc.Units[:i], doc.Units[i+1:]...)
			return doc, true
		}
	}
	return doc, false
}

func formatUnitEntries(doc unitDefsDoc) string {
	if len(doc.Units) == 0 {
		return "no custom units defined"
	}
	lines := make([]string, len(doc.Units))
	for i, u := range doc.Units {
		lines[i] = fmt.Sprintf("%s = %s %s^%d", u.Names, u.Base.Relation, u.Base.Unit, u.Base.Exponent)
	}
	return strings.Join(lines, "\n")
}

var qalcListTitleRe = regexp.MustCompile(`\s*\([^()]*\)\s*$`)

// qalcListContainsExactName checks the output of `qalc --list-<kind> <name>`
// for an exact name match. qalc's search is fuzzy (it matches on title text
// too), so a hit in the raw output doesn't mean name is a real defined
// identifier — e.g. "smoot" as a search term matches nothing, but as a
// one-shot expression qalc parses it via fuzzy implicit multiplication
// (second*milli*byte*tonne) and happily "succeeds". Parsing out just the
// slash-separated name tokens (stripping the trailing "(Title)") avoids
// that false positive.
func qalcListContainsExactName(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		for _, group := range strings.Split(line, "\t") {
			group = qalcListTitleRe.ReplaceAllString(strings.TrimSpace(group), "")
			for _, token := range strings.Split(group, "/") {
				if strings.EqualFold(strings.TrimSpace(token), name) {
					return true
				}
			}
		}
	}
	return false
}

func qalcNameExists(name string) bool {
	for _, flag := range []string{"--list-units", "--list-variables", "--list-functions"} {
		out, err := exec.Command("qalc", flag, name).Output()
		if err != nil {
			continue
		}
		if qalcListContainsExactName(string(out), name) {
			return true
		}
	}
	return false
}

func qalcToBase(expr string) (string, error) {
	// "-set conv none" disables qalc's automatic mixed-unit display (e.g.
	// "864 seconds" rendering as "14 min + 24 s" even under "to base"),
	// which would otherwise make parseBaseUnit see a compound result and
	// reject a perfectly valid single-dimension definition.
	out, err := exec.Command("qalc", "-set", "conv none", expr+" to base").Output()
	if err != nil && len(out) == 0 {
		return "", err
	}
	return string(out), nil
}

func evalUnitDefCmd(rawInput string) tea.Cmd {
	cmd, ok := resolveUnitCmd(rawInput)
	if !ok {
		return nil
	}
	displayExpr := strings.TrimSpace(rawInput)
	return func() tea.Msg {
		path, err := unitDefsPath()
		if err != nil {
			return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
		}

		switch cmd.kind {
		case unitCmdList:
			doc, err := loadUnitDefs(path)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			return resultMsg{displayExpr: displayExpr, display: formatUnitEntries(doc)}

		case unitCmdDelete:
			doc, err := loadUnitDefs(path)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			doc, found := deleteUnitEntry(doc, cmd.name)
			if !found {
				return calcErrMsg{displayExpr: displayExpr, err: fmt.Sprintf("%s: not found", cmd.name)}
			}
			if err := saveUnitDefs(path, doc); err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			return resultMsg{displayExpr: displayExpr, display: fmt.Sprintf("%s deleted", cmd.name)}

		case unitCmdDefine:
			if qalcNameExists(cmd.name) {
				return calcErrMsg{displayExpr: displayExpr, err: fmt.Sprintf("%s: already defined", cmd.name)}
			}
			out, err := qalcToBase(cmd.expr)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			relation, baseSymbol, exponent, err := parseBaseUnit(out)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: fmt.Sprintf("%s: %s", cmd.name, err.Error())}
			}
			doc, err := loadUnitDefs(path)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			doc, err = addUnitEntry(doc, cmd.name, baseSymbol, relation, exponent)
			if err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			if err := saveUnitDefs(path, doc); err != nil {
				return calcErrMsg{displayExpr: displayExpr, err: err.Error()}
			}
			return resultMsg{
				displayExpr: displayExpr,
				display:     fmt.Sprintf("%s = %s %s (saved)", cmd.name, relation, baseSymbol),
			}

		default:
			return calcErrMsg{
				displayExpr: displayExpr,
				err:         "invalid unit command, expected: unit <name> = <expr> | unit list | unit delete <name>",
			}
		}
	}
}
