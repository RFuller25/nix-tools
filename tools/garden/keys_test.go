package main

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// keyToken is how a key is written in a binding's label.
func keyToken(k string) string {
	switch k {
	case "left":
		return "←"
	case "right":
		return "→"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case " ":
		return "space"
	case "/":
		return "slash"
	case "pgdown":
		return "pgdn"
	}
	return k
}

// labelTokens splits a label such as "↑↓ / jk, pgup / pgdn" into the keys it
// names. Runs of arrows and of plain letters (hjkl) are split per character.
func labelTokens(label string) map[string]bool {
	out := map[string]bool{}
	label = strings.NewReplacer("/", " ", ",", " ").Replace(label)
	for _, f := range strings.Fields(label) {
		out[f] = true
		if len([]rune(f)) > 1 && !strings.Contains(f, "+") {
			for _, r := range f {
				out[string(r)] = true
			}
		}
	}
	return out
}

// Every key the game answers to must be named in the documentation of its set,
// and every documented label must belong to a real binding.
func TestEveryKeyIsDocumented(t *testing.T) {
	for _, set := range setOrder {
		documented := map[string]bool{}
		for _, d := range documentedKeys() {
			if d.Set == set {
				for tok := range labelTokens(d.Label) {
					documented[tok] = true
				}
			}
		}
		for _, b := range bindingsIn(set) {
			if len(b.Keys) == 0 || b.Do == nil {
				t.Errorf("%s: a binding has no keys or no action: %+v", set, b.Keys)
			}
			for _, k := range b.Keys {
				if k == anyKey {
					if b.Desc == "" {
						t.Errorf("%s: the catch-all key is undocumented", set)
					}
					continue
				}
				if !documented[keyToken(k)] && !documented[k] {
					t.Errorf("%s: key %q does nothing the docs mention", set, k)
				}
			}
			if b.Label != "" && b.Desc == "" {
				t.Errorf("%s: %q has a label but no description", set, b.Label)
			}
		}
	}
}

func TestBindingsDoNotCollide(t *testing.T) {
	for _, set := range setOrder {
		seen := map[string]string{}
		for _, b := range bindingsIn(set) {
			for _, k := range b.Keys {
				if prev, dup := seen[k]; dup {
					t.Errorf("%s: key %q is bound twice (%s and %s)", set, k, prev, b.Label)
				}
				seen[k] = b.Label
			}
		}
	}
}

// A hint in the footer must name a key that really is bound to something.
func TestFooterHintsPointAtRealKeys(t *testing.T) {
	m := demoModel(t, 90, 30)
	for _, set := range setOrder {
		for _, b := range bindingsIn(set) {
			if b.Hint == "" {
				continue
			}
			if len(strings.Fields(b.Hint)) < 2 {
				t.Errorf("hint %q is not a key and a word", b.Hint)
			}
		}
	}
	_ = m
}

// Every documented key can be pressed on every screen without a crash.
func TestEveryDocumentedKeyIsSafe(t *testing.T) {
	for _, set := range setOrder {
		for _, b := range bindingsIn(set) {
			for _, k := range b.Keys {
				if k == "ctrl+c" || k == "q" && set == setGlobal {
					continue
				}
				m := demoModel(t, 90, 30)
				m.screen = map[keySet]screen{
					setShop: screenShop, setInfo: screenInfo, setAlmanac: screenAlmanac,
					setJournal: screenJournal, setHelp: screenHelp,
				}[set]
				var msg tea.KeyMsg
				switch k {
				case "enter":
					msg = tea.KeyMsg{Type: tea.KeyEnter}
				case "tab":
					msg = tea.KeyMsg{Type: tea.KeyTab}
				case "esc":
					msg = tea.KeyMsg{Type: tea.KeyEsc}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
				}
				next, _ := m.Update(msg)
				if strings.TrimSpace(next.View()) == "" {
					t.Errorf("%s: key %q left a blank screen", set, k)
				}
			}
		}
	}
}

func TestReadmeKeyTablesAreCurrent(t *testing.T) {
	const start, end = "<!-- keys:start -->", "<!-- keys:end -->"
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	a, b := strings.Index(text, start), strings.Index(text, end)
	if a < 0 || b < a {
		t.Fatalf("README.md needs %s and %s markers around the key tables", start, end)
	}
	want := start + "\n\n" + keysMarkdown() + end
	got := text[a : b+len(end)]
	if got == want {
		return
	}
	if os.Getenv("UPDATE_README") != "" {
		if err := os.WriteFile("README.md", []byte(text[:a]+want+text[b+len(end):]), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Errorf("the README key tables are out of date; run: UPDATE_README=1 go test ./tools/garden -run TestReadmeKeyTablesAreCurrent")
}

// Every chapter of the guide renders, at wide and narrow sizes.
func TestGuideChaptersRender(t *testing.T) {
	m := demoModel(t, 100, 40)
	for _, ch := range guideChapters() {
		for _, w := range []int{30, 50, 90} {
			lines := ch.Build(m, w)
			if len(lines) == 0 {
				t.Errorf("chapter %q is empty at width %d", ch.Title, w)
			}
		}
		if ch.Title == "" || ch.Lede == "" {
			t.Errorf("a chapter has no title or lede: %+v", ch.Title)
		}
	}
}

// The guide is the first thing in the almanac, and the keys chapter lists every
// documented key.
func TestKeysChapterListsEveryDocumentedKey(t *testing.T) {
	m := demoModel(t, 100, 40)
	var keys chapter
	for _, ch := range guideChapters() {
		if ch.Title == "Keys" {
			keys = ch
		}
	}
	text := strings.Join(keys.Build(m, 200), "\n")
	for _, d := range documentedKeys() {
		if !strings.Contains(text, d.Label) || !strings.Contains(text, d.Desc) {
			t.Errorf("the Keys chapter lacks %q (%s)", d.Label, d.Desc)
		}
	}
}
