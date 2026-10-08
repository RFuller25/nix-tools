package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansi.ReplaceAllString(s, "") }

func almanacModel(t *testing.T, w, h int) model {
	t.Helper()
	m := demoModel(t, w, h)
	m.screen = screenAlmanac
	m.refreshAlmanac()
	m.fixAlmanac()
	return m
}

// onScreen reports whether the highlighted row is in the view, and that the
// view fits the terminal.
func checkSelectedVisible(t *testing.T, m model, label string) {
	t.Helper()
	view := plain(m.View())
	lines := strings.Split(view, "\n")
	if len(lines) > m.height {
		t.Fatalf("%s: view is %d lines in a window of %d", label, len(lines), m.height)
	}
	cur, ok := m.almanacCur()
	if !ok {
		t.Fatalf("%s: no row under the cursor", label)
	}
	title := cur.title(m.g)
	stem := string([]rune(title)[:min(6, len([]rune(title)))])
	found := false
	for _, l := range lines {
		if strings.Contains(l, "› ") && strings.Contains(l, stem) {
			found = true
		}
	}
	if !found {
		t.Fatalf("%s: the selected row %q (cursor %d of %d, scroll %d) is not on screen:\n%s",
			label, title, m.almanacCursor, len(m.almanacVisible()), m.almanacScroll, view)
	}
}

// The reported bug: moving down the list let the selection run past the bottom
// of the screen. Walk the whole almanac down and back up at several sizes.
func TestSelectedRowIsAlwaysOnScreen(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {80, 24}, {120, 20}, {60, 18}, {50, 14}, {40, 12}, {100, 9}} {
		m := almanacModel(t, size[0], size[1])
		var cur tea.Model = m
		n := len(m.almanacVisible())
		for i := 0; i < n+2; i++ {
			checkSelectedVisible(t, cur.(model), fmt.Sprintf("%dx%d down step %d", size[0], size[1], i))
			cur = keyPress(cur, "j")
		}
		for i := 0; i < n+2; i++ {
			checkSelectedVisible(t, cur.(model), fmt.Sprintf("%dx%d up step %d", size[0], size[1], i))
			cur = keyPress(cur, "k")
		}
	}
}

// Scrolling is steady: going up from the bottom does not drag the selection
// to the foot of the screen, and the window only moves when it has to.
func TestScrollingIsStable(t *testing.T) {
	m := almanacModel(t, 100, 30)
	var cur tea.Model = m
	for i := 0; i < 40; i++ {
		cur = keyPress(cur, "j")
	}
	down := cur.(model)
	if down.almanacScroll == 0 {
		t.Fatal("never scrolled")
	}
	before := down.almanacScroll
	cur = keyPress(cur, "k")
	up := cur.(model)
	if up.almanacScroll != before {
		t.Errorf("moving up one row moved the window from %d to %d", before, up.almanacScroll)
	}
	_, _, rows := up.almanacMetrics()
	if up.almanacCursor >= up.almanacScroll+rows || up.almanacCursor < up.almanacScroll {
		t.Error("the cursor left the window")
	}
}

func TestResizingKeepsTheSelectionOnScreen(t *testing.T) {
	m := almanacModel(t, 100, 40)
	var cur tea.Model = m
	for i := 0; i < 60; i++ {
		cur = keyPress(cur, "j")
	}
	for _, size := range [][2]int{{100, 14}, {50, 10}, {120, 50}, {40, 12}} {
		next, _ := cur.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		cur = next
		checkSelectedVisible(t, cur.(model), fmt.Sprintf("resized to %dx%d", size[0], size[1]))
	}
}

func headerIndex(m model, group string) int {
	for i, r := range m.almanacVisible() {
		if r.Kind == rowHeader && r.Group == group {
			return i
		}
	}
	return -1
}

func TestCategoriesFold(t *testing.T) {
	m := almanacModel(t, 100, 40)
	total := len(m.almanacVisible())
	flowers := m.groupCount("flower")
	if flowers < 5 {
		t.Fatalf("only %d flowers", flowers)
	}

	// Headings are rows the cursor can stand on.
	var cur tea.Model = m
	m.almanacCursor = headerIndex(m, "flower")
	cur = m
	cur = keyPress(cur, "enter")
	folded := cur.(model)
	if !folded.g.Folded["flower"] {
		t.Fatal("enter on a heading did not fold it")
	}
	if got := len(folded.almanacVisible()); got != total-flowers {
		t.Errorf("%d rows after folding, want %d", got, total-flowers)
	}
	v := plain(folded.View())
	if !strings.Contains(v, "▸ FLOWER") || strings.Contains(v, "Sunflower") {
		t.Errorf("the folded category should show only its heading:\n%s", v)
	}
	if !strings.Contains(v, "▾ GUIDE") {
		t.Error("open categories lose their arrow")
	}

	// The cursor skips the hidden rows: down from the folded heading is the next heading.
	cur = keyPress(cur, "j")
	next, _ := cur.(model).almanacCur()
	if next.Kind != rowHeader || next.Group != "bulb" {
		t.Errorf("down from a folded heading landed on %v %q", next.Kind, next.Group)
	}
	checkSelectedVisible(t, cur.(model), "after folding")

	// Opening it again brings the rows back.
	cur = keyPress(cur, "k")
	cur = keyPress(cur, "enter")
	if len(cur.(model).almanacVisible()) != total || cur.(model).g.Folded["flower"] {
		t.Error("enter did not open the folded category")
	}
}

func TestFoldingFromARowMovesToItsHeading(t *testing.T) {
	m := almanacModel(t, 100, 40)
	for i, r := range m.almanacVisible() {
		if r.IsPlant() && r.group() == "herb" {
			m.almanacCursor = i
			break
		}
	}
	var cur tea.Model = m
	cur = keyPress(cur, "c")
	got := cur.(model)
	row, _ := got.almanacCur()
	if row.Kind != rowHeader || row.Group != "herb" || !got.g.Folded["herb"] {
		t.Errorf("c on a herb left the cursor on %v %q (folded=%v)", row.Kind, row.Group, got.g.Folded["herb"])
	}
	checkSelectedVisible(t, got, "after c")
}

func TestFoldAllAndOpenAll(t *testing.T) {
	m := almanacModel(t, 100, 40)
	var cur tea.Model = m
	cur = keyPress(cur, "C")
	all := cur.(model)
	for _, r := range all.almanacVisible() {
		if r.Kind != rowHeader {
			t.Fatalf("a %v row survived fold-all", r.Kind)
		}
	}
	if len(all.almanacVisible()) < 5 {
		t.Error("fold-all should leave every heading")
	}
	checkSelectedVisible(t, all, "after fold all")
	cur = keyPress(cur, "C")
	open := cur.(model)
	if len(open.g.Folded) != 0 || len(open.almanacVisible()) != len(open.almanac) {
		t.Error("C a second time did not open everything")
	}
	checkSelectedVisible(t, open, "after open all")

	// If only some are folded, C folds the rest.
	open.g.Folded = map[string]bool{"flower": true}
	before := len(open.almanacVisible())
	cur = keyPress(open, "C")
	if len(cur.(model).almanacVisible()) >= before {
		t.Error("C with some categories open should fold them")
	}
}

func TestFoldedCategoriesAreRemembered(t *testing.T) {
	m := almanacModel(t, 100, 40)
	var cur tea.Model = m
	cur = keyPress(cur, "C")
	g := cur.(model).g
	path := t.TempDir() + "/g.json"
	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if !back.Folded["flower"] || !back.Folded["guide"] {
		t.Errorf("folded categories did not survive: %v", back.Folded)
	}
	m2 := newModel(back, path, testStart())
	m2.width, m2.height = 100, 40
	m2.screen = screenAlmanac
	m2.fixAlmanac()
	if len(m2.almanacVisible()) >= len(m2.almanac) {
		t.Error("a reloaded garden showed folded categories open")
	}
	checkSelectedVisible(t, m2, "reloaded")
}

func TestFoldingASelectionKeepsRenamingWorking(t *testing.T) {
	// A cultivar row under a folded heading must not be reachable or renameable.
	m := almanacModel(t, 100, 40)
	m.g.Folded = map[string]bool{"your cultivars": true}
	m.fixAlmanac()
	for _, r := range m.almanacVisible() {
		if r.Kind == rowCultivar {
			t.Error("a cultivar shows under a folded heading")
		}
	}
	m.almanacCursor = headerIndex(m, "your cultivars")
	var cur tea.Model = m
	cur = keyPress(cur, "n")
	if cur.(model).naming {
		t.Error("n on a heading started naming")
	}
}

func TestHeadingCardsRender(t *testing.T) {
	m := almanacModel(t, 100, 40)
	for _, r := range m.almanacVisible() {
		if r.Kind != rowHeader {
			continue
		}
		m.almanacCursor = headerIndex(m, r.Group)
		v := plain(m.View())
		if !strings.Contains(v, "in this category") {
			t.Errorf("%s heading has no card:\n%s", r.Group, v)
		}
	}
}

func TestUniqueCultivarsOfDedupesByGenome(t *testing.T) {
	sp := AllSpecies()[0]
	gn := sp.VarietyGenome(0)
	other := gn
	other.Hue = (other.Hue + 90) % 360
	g := &Garden{Cultivars: []Cultivar{
		{ID: 1, Species: sp.ID, Genome: gn},
		{ID: 2, Species: sp.ID, Genome: gn},
		{ID: 3, Species: sp.ID, Genome: other},
	}}
	if got := g.UniqueCultivarsOf(sp); len(got) != 2 || got[0].ID != 1 {
		t.Fatalf("want 2 unique lines starting with #1, got %+v", got)
	}
}

func TestNamingAHybridSeedFilesACultivar(t *testing.T) {
	sp := AllSpecies()[0]
	far := sp.VarietyGenome(0)
	far.Hue = (far.Hue + 150) % 360
	far.Height = 100 - far.Height%50
	g := &Garden{Shed: []Packet{{ID: 1, SpeciesID: sp.ID, A: far, B: far, Count: 3}}}
	g.RenamePacket(0, "Ember", time.Now())
	if len(g.Cultivars) != 1 || g.Cultivars[0].Name != "Ember" || !g.Cultivars[0].Named || g.Shed[0].Line != 1 {
		t.Fatalf("want one named cultivar linked to the packet, got %+v / line %d", g.Cultivars, g.Shed[0].Line)
	}
	g.RenamePacket(0, "Blaze", time.Now())
	if len(g.Cultivars) != 1 || g.Cultivars[0].Name != "Blaze" {
		t.Fatalf("renaming should rename the line, got %+v", g.Cultivars)
	}
}
