# pkg-browser NixOS Config Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a split-panel layout to pkg-browser showing user profile packages (left) and NixOS config packages (right), with in-TUI add/remove editing of `environment.systemPackages`.

**Architecture:** New pure-function `config.go` handles nix config parsing and editing on `[]byte`. A `persist.go` layer stores the config file path in `~/.config/pkg-browser/config.json`. A `setup.go` first-run model runs before the main TUI when no config is saved. The main model is restructured from a tab layout to a split-panel layout with a bottom search panel.

**Tech Stack:** Go, Bubbletea (`github.com/charmbracelet/bubbletea`), Bubbles (`textinput`, `spinner`), Lipgloss

---

## File Map

| File | Status | Responsibility |
|------|--------|----------------|
| `main.go` | Modify | Entry point only: first-run gate, launch setup or main model |
| `model.go` | Create (from main.go) | Main model struct, Init/Update/View, keybindings |
| `layout.go` | Create | Split-panel and search panel rendering helpers |
| `nix.go` | Unchanged | `loadInstalled()`, `searchNixpkgs()` |
| `config.go` | Create | Pure functions: `parseConfigPackages`, `addPackage`, `removePackage` |
| `config_test.go` | Create | Unit tests for all config.go functions |
| `setup.go` | Create | First-run Bubbletea model: prompt for config path, validate, save |
| `persist.go` | Create | `readAppConfig()`, `writeAppConfig()` for `~/.config/pkg-browser/config.json` |
| `nix_test.go` | Unchanged | Existing profile/search parsing tests |

---

## Task 1: Persist layer

**Files:**
- Create: `tools/pkg-browser/persist.go`

- [ ] **Step 1: Create persist.go**

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	ConfigPath string `json:"configPath"`
}

func configFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pkg-browser", "config.json"), nil
}

func readAppConfig() (AppConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AppConfig{}, nil
		}
		return AppConfig{}, err
	}
	var cfg AppConfig
	return cfg, json.Unmarshal(data, &cfg)
}

func writeAppConfig(cfg AppConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./tools/pkg-browser/
```

Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add tools/pkg-browser/persist.go
git commit -m "feat(pkg-browser): add persist layer for config path storage"
```

---

## Task 2: Config parser (TDD)

**Files:**
- Create: `tools/pkg-browser/config.go`
- Create: `tools/pkg-browser/config_test.go`

- [ ] **Step 1: Write failing tests**

```go
// config_test.go
package main

import (
	"testing"
)

var sampleConfig = []byte(`
{ config, pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    vim
    wget
    # a comment
    (python3.withPackages (python-pkgs: with python-pkgs; [
      flask
    ]))
    git
  ];
}
`)

func TestParseConfigPackages_SimpleEntries(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	for _, want := range []string{"vim", "wget", "git"} {
		if !names[want] {
			t.Errorf("want %q in result, got %v", want, pkgs)
		}
	}
}

func TestParseConfigPackages_ComplexReadOnly(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	for _, p := range pkgs {
		if p.Name == "python3" {
			if !p.ReadOnly {
				t.Errorf("python3 entry should be ReadOnly")
			}
			return
		}
	}
	t.Error("python3 complex entry not found")
}

func TestParseConfigPackages_CommentNotIncluded(t *testing.T) {
	pkgs := parseConfigPackages(sampleConfig)
	for _, p := range pkgs {
		if p.Name == "a comment" || p.Name == "#" {
			t.Errorf("comment line should not appear as package: %q", p.Name)
		}
	}
}

func TestAddPackage_AppendsBeforeClosingBracket(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
  ];`)
	result := addPackage(src, "git")
	pkgs := parseConfigPackages(result)
	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	if !names["git"] {
		t.Errorf("git not found after add; packages: %v", pkgs)
	}
	if !names["vim"] {
		t.Errorf("vim disappeared after add; packages: %v", pkgs)
	}
}

func TestAddPackage_EmptyBlock(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
  ];`)
	result := addPackage(src, "htop")
	pkgs := parseConfigPackages(result)
	if len(pkgs) != 1 || pkgs[0].Name != "htop" {
		t.Errorf("want [htop], got %v", pkgs)
	}
}

func TestRemovePackage_RemovesEntry(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
    wget
    git
  ];`)
	result := removePackage(src, "wget")
	pkgs := parseConfigPackages(result)
	for _, p := range pkgs {
		if p.Name == "wget" {
			t.Error("wget still present after remove")
		}
	}
}

func TestRemovePackage_NoopOnMissing(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
  ];`)
	result := removePackage(src, "nonexistent")
	if string(result) != string(src) {
		t.Error("src should be unchanged when package not found")
	}
}

func TestRemovePackage_NoopOnComplex(t *testing.T) {
	result := removePackage(sampleConfig, "python3")
	pkgs := parseConfigPackages(result)
	found := false
	for _, p := range pkgs {
		if p.Name == "python3" {
			found = true
		}
	}
	if !found {
		t.Error("complex entry should not be removable")
	}
}

func TestRoundTrip_RemoveThenParse(t *testing.T) {
	src := []byte(`  environment.systemPackages = with pkgs; [
    vim
    wget
    git
  ];`)
	after := removePackage(src, "wget")
	pkgs := parseConfigPackages(after)
	if len(pkgs) != 2 {
		t.Errorf("want 2 packages after remove, got %d: %v", len(pkgs), pkgs)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test ./tools/pkg-browser/ -run "TestParseConfig|TestAddPackage|TestRemovePackage|TestRoundTrip" -v
```

Expected: compile error `undefined: parseConfigPackages`

- [ ] **Step 3: Implement config.go**

```go
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

		atTopLevel := bracketDepth == 1 && parenDepth == 0

		for _, ch := range trimmed {
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

		// Strip inline comment
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
		for _, ch := range lines[i] {
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
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test ./tools/pkg-browser/ -run "TestParseConfig|TestAddPackage|TestRemovePackage|TestRoundTrip" -v
```

Expected: all PASS

- [ ] **Step 5: Run full test suite**

```bash
go test ./tools/pkg-browser/ -v
```

Expected: all existing tests still PASS

- [ ] **Step 6: Commit**

```bash
git add tools/pkg-browser/config.go tools/pkg-browser/config_test.go
git commit -m "feat(pkg-browser): add NixOS config parser with pure add/remove functions"
```

---

## Task 3: Refactor main.go → model.go + main.go

Move the existing model code out of `main.go` into `model.go`. No behavior change — this just cleans the slate for subsequent tasks.

**Files:**
- Modify: `tools/pkg-browser/main.go`
- Create: `tools/pkg-browser/model.go`

- [ ] **Step 1: Create model.go with the existing model code**

Move everything except `func main()` out of `main.go` into `model.go`. The result:

`model.go` contains: all `var` style declarations, `type activeTab`, `type model`, `initialModel()`, `(m model) Init()`, `(m model) Update()`, `(m model) View()`, `(m model) currentList()`, all `lipgloss` style vars.

`main.go` contains only:

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles and tests pass**

```bash
go build ./tools/pkg-browser/ && go test ./tools/pkg-browser/ -v
```

Expected: builds cleanly, all tests PASS

- [ ] **Step 3: Commit**

```bash
git add tools/pkg-browser/main.go tools/pkg-browser/model.go
git commit -m "refactor(pkg-browser): extract model into model.go"
```

---

## Task 4: First-run setup model

**Files:**
- Create: `tools/pkg-browser/setup.go`
- Modify: `tools/pkg-browser/main.go`

- [ ] **Step 1: Create setup.go**

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type setupModel struct {
	input      textinput.Model
	errMsg     string
	configPath string // set on successful confirm
}

func newSetupModel() setupModel {
	ti := textinput.New()
	ti.Placeholder = "/etc/nixos/configuration.nix"
	ti.SetValue("/etc/nixos/configuration.nix")
	ti.Focus()
	ti.Width = 60
	return setupModel{input: ti}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			path := strings.TrimSpace(m.input.Value())
			if path == "" {
				m.errMsg = "path cannot be empty"
				return m, nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				m.errMsg = fmt.Sprintf("cannot read file: %v", err)
				return m, nil
			}
			if !strings.Contains(string(data), "environment.systemPackages") {
				m.errMsg = "file does not contain environment.systemPackages"
				return m, nil
			}
			m.configPath = path
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m setupModel) View() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("pkg-browser — first run setup"))
	b.WriteString("\n\n")
	b.WriteString("NixOS config path:\n")
	b.WriteString(m.input.View() + "\n")
	if m.errMsg != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.errMsg) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("enter: confirm  ctrl+c: quit"))
	return b.String()
}
```

- [ ] **Step 2: Update main.go to run setup when no config saved**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configFlag := flag.String("config", "", "path to NixOS configuration.nix (skips first-run prompt)")
	flag.Parse()

	var configPath string

	if *configFlag != "" {
		configPath = *configFlag
	} else {
		cfg, err := readAppConfig()
		if err != nil || cfg.ConfigPath == "" {
			// First-run setup
			p := tea.NewProgram(newSetupModel())
			m, err := p.Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			result := m.(setupModel)
			if result.configPath == "" {
				os.Exit(0)
			}
			if err := writeAppConfig(AppConfig{ConfigPath: result.configPath}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", err)
			}
			configPath = result.configPath
		} else {
			configPath = cfg.ConfigPath
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = configPath // wired into model in next task
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./tools/pkg-browser/
```

Expected: no output

- [ ] **Step 4: Commit**

```bash
git add tools/pkg-browser/setup.go tools/pkg-browser/main.go
git commit -m "feat(pkg-browser): add first-run setup model and config path persistence"
```

---

## Task 5: Split-panel layout

Replace the tab-based layout in `model.go` with a split-panel layout. Profile packages on the left, a placeholder (empty) panel on the right. Config loading is wired in Task 6.

**Files:**
- Modify: `tools/pkg-browser/model.go`
- Create: `tools/pkg-browser/layout.go`
- Modify: `tools/pkg-browser/main.go` (pass configPath to initialModel)

- [ ] **Step 1: Rewrite model.go with the new struct and Update/View**

Replace the entire contents of `model.go`:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedPanel int

const (
	panelProfile focusedPanel = iota
	panelConfig
)

type model struct {
	focus  focusedPanel
	width  int
	height int
	ready  bool

	profilePkgs   []pkg
	profileCursor int
	profileErr    error

	configPkgs    []configPkg
	configCursor  int
	configPath    string
	configErr     error

	searchOpen   bool
	searchInput  textinput.Model
	searchResult []pkg
	searchCursor int
	searchErr    error

	spinner   spinner.Model
	loading   bool
	statusMsg string
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	activeTabS   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Underline(true)
	inactiveTabS = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedRow  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	normalRow    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	versionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	divStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func initialModel(configPath string) model {
	ti := textinput.New()
	ti.Placeholder = "search nixpkgs..."

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		focus:      panelProfile,
		configPath: configPath,
		searchInput: ti,
		spinner:    sp,
		loading:    true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadInstalled(), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = msg.Width - 4
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Search input captures all keys when open
		if m.searchOpen {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.searchOpen = false
				m.focus = panelConfig
				m.searchResult = nil
				m.searchCursor = 0
				m.searchInput.Blur()
				return m, nil
			case "enter":
				q := strings.TrimSpace(m.searchInput.Value())
				if q != "" {
					m.loading = true
					m.searchResult = nil
					m.searchCursor = 0
					return m, tea.Batch(searchNixpkgs(q), m.spinner.Tick)
				}
				return m, nil
			case "up", "k":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
				return m, nil
			case "down", "j":
				if m.searchCursor < len(m.searchResult)-1 {
					m.searchCursor++
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == panelProfile {
				m.focus = panelConfig
			} else {
				m.focus = panelProfile
			}
			return m, nil
		case "/":
			m.searchOpen = true
			m.searchInput.Focus()
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		}

	case installedLoadedMsg:
		m.profilePkgs = msg.pkgs
		m.loading = false
		return m, nil

	case searchResultMsg:
		m.searchResult = msg.pkgs
		m.loading = false
		return m, nil

	case nixErrMsg:
		if m.focus == panelProfile {
			m.profileErr = msg.err
		} else {
			m.searchErr = msg.err
		}
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

func (m *model) moveCursor(delta int) {
	switch m.focus {
	case panelProfile:
		m.profileCursor = clamp(m.profileCursor+delta, 0, len(m.profilePkgs)-1)
	case panelConfig:
		m.profileCursor = clamp(m.profileCursor+delta, 0, len(m.profilePkgs)-1)
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}
	var b strings.Builder
	b.WriteString(renderHeader(m))
	b.WriteString(renderPanels(m))
	b.WriteString(divStyle.Render(strings.Repeat("─", m.width)) + "\n")
	if m.searchOpen {
		b.WriteString(m.searchInput.View() + "\n")
		b.WriteString(divStyle.Render(strings.Repeat("─", m.width)) + "\n")
		b.WriteString(renderSearchPanel(m))
	}
	b.WriteString(renderHelp(m))
	return b.String()
}

func renderHeader(m model) string {
	profileLabel := "Profile"
	configLabel := "Config"
	if m.focus == panelProfile {
		profileLabel = activeTabS.Render(profileLabel)
		configLabel = inactiveTabS.Render(configLabel)
	} else {
		profileLabel = inactiveTabS.Render(profileLabel)
		configLabel = activeTabS.Render(configLabel)
	}
	return titleStyle.Render("pkg-browser") + "  " + profileLabel + "  " + configLabel + "\n" +
		divStyle.Render(strings.Repeat("─", m.width)) + "\n"
}

func renderHelp(m model) string {
	if m.statusMsg != "" {
		return helpStyle.Render(m.statusMsg)
	}
	if m.searchOpen {
		return helpStyle.Render("↑/↓: navigate results  a: add to config  esc: close search  ctrl+c: quit")
	}
	return helpStyle.Render(fmt.Sprintf(
		"tab: switch panel  ↑/↓ j/k: navigate  /: search  d: remove  q: quit",
	))
}
```

Note: `moveCursor` for `panelConfig` has a copy-paste bug above — fix it:

```go
func (m *model) moveCursor(delta int) {
	switch m.focus {
	case panelProfile:
		m.profileCursor = clamp(m.profileCursor+delta, 0, len(m.profilePkgs)-1)
	case panelConfig:
		m.configCursor = clamp(m.configCursor+delta, 0, len(m.configPkgs)-1)
	}
}
```

- [ ] **Step 2: Create layout.go**

```go
package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// panelContentHeight returns the number of rows available for list content
// given the fixed chrome: header(2) + bottom_divider(1) + help(1) = 4.
// When search is open: also subtract search_input(1) + search_divider(1) + search_rows.
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
	if len(m.configPkgs) == 0 && m.configPath == "" {
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
		name := normalRow.Render(p.Name)
		if p.ReadOnly {
			name = descStyle.Render("[" + p.Name + "…]")
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
```

- [ ] **Step 3: Update main.go to pass configPath to initialModel**

Replace the `p := tea.NewProgram(initialModel(), ...)` line:

```go
p := tea.NewProgram(initialModel(configPath), tea.WithAltScreen())
```

Also remove the `_ = configPath` line.

- [ ] **Step 4: Verify it compiles and runs**

```bash
go build ./tools/pkg-browser/ && go test ./tools/pkg-browser/ -v
```

Expected: builds cleanly, all tests pass. Run manually with `go run ./tools/pkg-browser/ --config /etc/nixos/configuration.nix` to verify split panel renders with profile packages on the left.

- [ ] **Step 5: Commit**

```bash
git add tools/pkg-browser/model.go tools/pkg-browser/layout.go tools/pkg-browser/main.go
git commit -m "feat(pkg-browser): split-panel layout with profile and config panels"
```

---

## Task 6: Load and display config packages

**Files:**
- Modify: `tools/pkg-browser/nix.go` (add config load cmd)
- Modify: `tools/pkg-browser/model.go` (handle configLoadedMsg, wire Init)

- [ ] **Step 1: Add config load cmd and messages to nix.go**

Add to the bottom of `nix.go`:

```go
type configLoadedMsg struct{ pkgs []configPkg }
type configErrMsg struct{ err error }
type configSavedMsg struct{ pkgs []configPkg }

func loadConfig(path string) tea.Cmd {
	return func() tea.Msg {
		src, err := os.ReadFile(path)
		if err != nil {
			return configErrMsg{err}
		}
		return configLoadedMsg{pkgs: parseConfigPackages(src)}
	}
}
```

Add `"os"` to the imports in `nix.go`.

- [ ] **Step 2: Handle configLoadedMsg and configErrMsg in model.go Update**

Add cases to the `switch msg := msg.(type)` in `Update`:

```go
case configLoadedMsg:
    m.configPkgs = msg.pkgs
    return m, nil

case configErrMsg:
    m.configErr = msg.err
    return m, nil
```

- [ ] **Step 3: Add config load to Init**

Replace `Init()` in `model.go`:

```go
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadInstalled(), m.spinner.Tick}
	if m.configPath != "" {
		cmds = append(cmds, loadConfig(m.configPath))
	}
	return tea.Batch(cmds...)
}
```

- [ ] **Step 4: Verify and test manually**

```bash
go build ./tools/pkg-browser/ && go test ./tools/pkg-browser/ -v
```

Run manually with `go run ./tools/pkg-browser/ --config /etc/nixos/configuration.nix`. Expected: config packages appear in the right panel.

- [ ] **Step 5: Commit**

```bash
git add tools/pkg-browser/nix.go tools/pkg-browser/model.go
git commit -m "feat(pkg-browser): load and display NixOS config packages in right panel"
```

---

## Task 7: Edit operations — add and remove

**Files:**
- Modify: `tools/pkg-browser/nix.go` (add applyAdd, applyRemove cmds)
- Modify: `tools/pkg-browser/model.go` (handle keys + configSavedMsg)

- [ ] **Step 1: Add applyAdd and applyRemove to nix.go**

```go
func applyAdd(configPath, name string) tea.Cmd {
	return func() tea.Msg {
		src, err := os.ReadFile(configPath)
		if err != nil {
			return configErrMsg{err}
		}
		updated := addPackage(src, name)
		if err := os.WriteFile(configPath, updated, 0644); err != nil {
			return configErrMsg{err}
		}
		return configSavedMsg{pkgs: parseConfigPackages(updated)}
	}
}

func applyRemove(configPath, name string) tea.Cmd {
	return func() tea.Msg {
		src, err := os.ReadFile(configPath)
		if err != nil {
			return configErrMsg{err}
		}
		updated := removePackage(src, name)
		if err := os.WriteFile(configPath, updated, 0644); err != nil {
			return configErrMsg{err}
		}
		return configSavedMsg{pkgs: parseConfigPackages(updated)}
	}
}
```

- [ ] **Step 2: Handle configSavedMsg in model.go Update**

```go
case configSavedMsg:
    m.configPkgs = msg.pkgs
    m.statusMsg = "Saved. Run nixos-switch to apply."
    return m, nil
```

- [ ] **Step 3: Wire 'd' key in main panel (remove from config)**

Inside the `!m.searchOpen` key handler block in `Update`, add after the `"/"` case:

```go
case "d":
    if m.focus == panelConfig && len(m.configPkgs) > 0 {
        p := m.configPkgs[m.configCursor]
        if !p.ReadOnly {
            return m, applyRemove(m.configPath, p.Name)
        }
    }
    return m, nil
```

- [ ] **Step 4: Wire 'a' key in search panel (add to config)**

Inside the `m.searchOpen` key handler block in `Update`, add before the `"enter"` case:

```go
case "a":
    if len(m.searchResult) > 0 {
        p := m.searchResult[m.searchCursor]
        return m, applyAdd(m.configPath, p.Name)
    }
    return m, nil
```

- [ ] **Step 5: Clear statusMsg on any keypress**

At the top of the `tea.KeyMsg` case, before any other handling:

```go
case tea.KeyMsg:
    m.statusMsg = ""
    // ... rest of key handling
```

- [ ] **Step 6: Verify, test manually**

```bash
go build ./tools/pkg-browser/ && go test ./tools/pkg-browser/ -v
```

Manual test with `go run ./tools/pkg-browser/ --config /etc/nixos/configuration.nix`:
1. Navigate to Config panel, press `d` on a simple package — verify it disappears and status message shows
2. Open search with `/`, search for a package, press `a` — verify it appears in config panel

- [ ] **Step 7: Commit**

```bash
git add tools/pkg-browser/nix.go tools/pkg-browser/model.go
git commit -m "feat(pkg-browser): add/remove packages from NixOS config in-TUI"
```

---

## Task 8: Fix vendorHash in flake.nix

The `go.sum` for `pkg-browser` is unchanged (no new dependencies added), but confirm this and update `flake.nix` if needed.

**Files:**
- Modify: `tools/pkg-browser/flake.nix` (if vendorHash changed)

- [ ] **Step 1: Check if go.sum changed**

```bash
git diff tools/pkg-browser/go.sum
```

Expected: no changes (no new Go dependencies were added). If `go.sum` did change, proceed to step 2; otherwise skip to step 3.

- [ ] **Step 2: (If go.sum changed) Update vendorHash**

```bash
# Set vendorHash to empty string first to get the correct hash from nix
# Edit flake.nix: set vendorHash = "";
# Then run:
nix build .#pkg-browser 2>&1 | grep "got:"
# Use the printed hash as the new vendorHash value
```

- [ ] **Step 3: Run full test suite**

```bash
go test ./... 
```

Expected: all tests pass across all tools.

- [ ] **Step 4: Final commit**

```bash
git add -p  # stage any flake.nix changes if needed
git commit -m "chore(pkg-browser): verify vendorHash after NixOS config integration"
```

---

## Self-Review Notes

- All spec requirements covered: split panel (Task 5), config parse/edit (Tasks 2, 6, 7), first-run (Task 4), persist (Task 1), search panel (wired in Task 5 Update, rendered in layout.go), status message (Task 7), read-only complex entries (Task 2 config.go + Task 5 configPanelLines)
- Types consistent throughout: `configPkg` defined in Task 2, used in Tasks 5/6/7; `AppConfig` in Task 1, used in Task 4; `configLoadedMsg`/`configErrMsg`/`configSavedMsg` defined in Task 6, handled in Tasks 6/7
- `applyAdd`/`applyRemove` in nix.go use `os.ReadFile`/`os.WriteFile` — `"os"` import already added in Task 6 Step 1
- `moveCursor` for `panelConfig` uses `m.configCursor` (corrected in Task 5 Step 1 note)
- Search panel `nixErrMsg` handler assigns to `m.searchErr` only when search is active — current handler uses `m.focus` which may not be `panelProfile` during search; update the `nixErrMsg` case in Task 6:

```go
case nixErrMsg:
    if m.searchOpen {
        m.searchErr = msg.err
    } else {
        m.profileErr = msg.err
    }
    m.loading = false
    return m, nil
```
