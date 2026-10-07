package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// breeder is a garden that has bred and named a cultivar.
func breeder(t *testing.T) (*Garden, *Cultivar) {
	t.Helper()
	now := time.Date(2026, 5, 10, 8, 0, 0, 0, time.UTC)
	g := newTestGarden(now)
	g.Seed = 1001
	g.Gardener = "Ann"
	growToFlower(t, g, 0, SpeciesByID("cosmos"), oddCosmos(), stableRuns, now)
	if len(g.Cultivars) != 1 {
		t.Fatal("no cultivar bred")
	}
	g.RenameCultivar(g.Cultivars[0].ID, "Moss Giant", now)
	return g, &g.Cultivars[0]
}

func friend() *Garden {
	g := newTestGarden(testStart())
	g.Seed = 2002
	return g
}

func TestACodeRoundTrips(t *testing.T) {
	g, c := breeder(t)
	code, sc, err := g.ExportCultivar(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "GD1-") || strings.Contains(code, " ") {
		t.Errorf("odd-looking code %q", code)
	}
	got, err := DecodeShare(code)
	if err != nil {
		t.Fatal(err)
	}
	if got != sc || got.Genome != oddCosmos() || got.Name != "Moss Giant" || got.From != "Ann" || !got.Stable || got.Species != "cosmos" {
		t.Errorf("decoded %+v", got)
	}
	// It survives being pasted lowercase, wrapped over lines, with stray spaces.
	messy := "  " + strings.ToLower(strings.ReplaceAll(code, "-", "-\n")) + " \n"
	if again, err := DecodeShare(messy); err != nil || again != sc {
		t.Errorf("a wrapped, lowercased code failed: %v", err)
	}
}

func TestEachExportIsItsOwnCode(t *testing.T) {
	g, c := breeder(t)
	a, _, _ := g.ExportCultivar(c.ID)
	b, _, _ := g.ExportCultivar(c.ID)
	if a == b {
		t.Fatal("two exports made the same code")
	}
	f := friend()
	if _, err := f.Redeem(a, testStart()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Redeem(b, testStart()); err != nil {
		t.Errorf("a second, separate code was refused: %v", err)
	}
}

func TestBadCodesAreRefusedWithAReason(t *testing.T) {
	g, c := breeder(t)
	code, sc, _ := g.ExportCultivar(c.ID)
	flip := func(s string, at int) string {
		b := []byte(s)
		if b[at] == 'A' {
			b[at] = 'B'
		} else {
			b[at] = 'A'
		}
		return string(b)
	}
	cases := map[string]string{
		"nonsense":      "hello there",
		"empty":         "",
		"wrong version": "GD9-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA-AAAAA",
		"cut off":       code[:len(code)-12],
		"one letter":    flip(code, 20),
		"not base32":    "GD1-!!!!!-?????-11111-00000-99999-88888",
	}
	for name, bad := range cases {
		if _, err := DecodeShare(bad); err == nil {
			t.Errorf("%s: accepted %q", name, bad)
		}
	}
	f := friend()
	for _, bad := range []string{flip(code, 20), code[:len(code)-12]} {
		if _, err := f.Redeem(bad, testStart()); err == nil || len(f.Shed) != 0 || len(f.Cultivars) != 0 {
			t.Errorf("a bad code changed the garden: %v, %d packets", err, len(f.Shed))
		}
	}

	// A well-formed code that names things that cannot be.
	for name, mutate := range map[string]func(*ShareCode){
		"unknown species":   func(c *ShareCode) { c.Species = "triffid" },
		"hue off the wheel": func(c *ShareCode) { c.Genome.Hue = 400 },
		"too light":         func(c *ShareCode) { c.Genome.Light = 100 },
		"height 200":        func(c *ShareCode) { c.Genome.Height = 200 },
		"no name":           func(c *ShareCode) { c.Name = "" },
	} {
		bad := sc
		mutate(&bad)
		if _, err := DecodeShare(bad.Encode()); err == nil {
			t.Errorf("%s: a code with a valid checksum but impossible contents was accepted", name)
		}
	}
}

func TestNamesInCodesAreCleaned(t *testing.T) {
	sc := ShareCode{Species: "cosmos", Genome: oddCosmos(), Name: "Fine\x1b[31m red\nname" + strings.Repeat("x", 80), From: "Bob\x07"}
	sc.Name = cleanText(sc.Name, shareNameMax)
	got, err := DecodeShare(sc.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got.Name+got.From, "\x1b\x07\n") || len([]rune(got.Name)) > shareNameMax {
		t.Errorf("name came back as %q / %q", got.Name, got.From)
	}
	// And anything hostile that reaches the decoder is cleaned there too.
	raw := ShareCode{Species: "cosmos", Genome: oddCosmos(), Name: "ok\x1b[2Jbad", From: "x"}
	if got, err := DecodeShare(raw.Encode()); err != nil || strings.Contains(got.Name, "\x1b") {
		t.Errorf("an escape sequence got through: %q (%v)", got.Name, err)
	}
}

func TestRedeemingAGift(t *testing.T) {
	g, c := breeder(t)
	code, _, _ := g.ExportCultivar(c.ID)
	f := friend()
	now := testStart()

	r, err := f.Redeem(code, now)
	if err != nil {
		t.Fatal(err)
	}
	if !r.NewLine || r.Packet.Count != giftSeeds {
		t.Errorf("redemption %+v", r)
	}
	// Three seeds, in the shed, labelled with the name.
	if f.SeedsInShed() != 3 || len(f.Shed) != 1 {
		t.Fatalf("%d seeds in %d packets", f.SeedsInShed(), len(f.Shed))
	}
	pk := f.Shed[0]
	if pk.Label != "Moss Giant" || pk.SpeciesID != "cosmos" || pk.A != oddCosmos() || pk.Stable() != true {
		t.Errorf("packet %+v", pk)
	}
	// And the line is in the almanac, credited.
	if len(f.Cultivars) != 1 {
		t.Fatalf("%d cultivars", len(f.Cultivars))
	}
	cv := f.Cultivars[0]
	if cv.Name != "Moss Giant" || !cv.Gifted || cv.From != "Ann" || !cv.Named || cv.Genome != oddCosmos() || pk.Line != cv.ID {
		t.Errorf("cultivar %+v", cv)
	}
	rows := almanacRows(f)
	found := false
	for _, row := range rows {
		found = found || (row.Kind == rowCultivar && row.Cultivar == cv.ID)
	}
	if !found {
		t.Error("the line is not a row in the almanac")
	}

	// Once only.
	if _, err := f.Redeem(code, now); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Errorf("a second redemption: %v", err)
	}
	if f.SeedsInShed() != 3 || len(f.Cultivars) != 1 {
		t.Error("a refused redemption still changed the garden")
	}
	// Not in the garden that made it.
	if _, err := g.Redeem(code, now); err == nil || !strings.Contains(err.Error(), "your own") {
		t.Errorf("redeeming in the garden that made it: %v", err)
	}
}

func TestAGiftOfALineYouHaveJoinsIt(t *testing.T) {
	g, c := breeder(t)
	code, _, _ := g.ExportCultivar(c.ID)
	f := friend()
	f.CultivarSeq = 4
	f.Cultivars = []Cultivar{{ID: 4, Name: "My green one", Species: "cosmos", Genome: oddCosmos(), Found: testStart()}}
	r, err := f.Redeem(code, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if r.NewLine || len(f.Cultivars) != 1 || f.Shed[0].Line != 4 {
		t.Errorf("a duplicate line was added: %d cultivars, new=%v", len(f.Cultivars), r.NewLine)
	}
	if f.Cultivars[0].Name != "My green one" {
		t.Error("a gift renamed a line the gardener already had")
	}
}

func TestGiftSeedSellsAtTheOrdinaryPrice(t *testing.T) {
	g, c := breeder(t)
	code, _, _ := g.ExportCultivar(c.ID)
	f := friend()
	f.Redeem(code, testStart())
	pk := f.Shed[0]
	// The same genes in a packet of your own breeding sell for the same.
	own := Packet{ID: 77, SpeciesID: pk.SpeciesID, A: pk.A, B: pk.B, Count: 3, Streak: pk.Streak}
	if seedValue(pk) != seedValue(own) || seedValue(pk) < 1 {
		t.Errorf("a gift sells for %d, home-grown seed of the same genes for %d", seedValue(pk), seedValue(own))
	}
	before := f.Gold
	got, err := f.SellSeeds(0, 3, testStart())
	if err != nil || got != 3*seedValue(own) || f.Gold != before+got {
		t.Errorf("sold for %d, err %v", got, err)
	}
	// And the sale is sensible against the shop: a gift is not a money machine.
	sp := SpeciesByID("cosmos")
	if seedValue(own)*giftSeeds > 12*sp.SeedCost {
		t.Errorf("three gift seeds sell for %d against a shop price of %d", seedValue(own)*giftSeeds, sp.SeedCost)
	}
}

func TestRedeemedCodesAreRemembered(t *testing.T) {
	g, c := breeder(t)
	code, _, _ := g.ExportCultivar(c.ID)
	f := friend()
	f.Redeem(code, testStart())
	path := filepath.Join(t.TempDir(), "g.json")
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, testStart())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := back.Redeem(code, testStart()); err == nil {
		t.Error("a reloaded garden forgot a code had been used")
	}
	if len(back.Cultivars) != 1 || !back.Cultivars[0].Gifted || back.Cultivars[0].From != "Ann" {
		t.Errorf("gifted cultivar did not survive: %+v", back.Cultivars)
	}
	// Exports keep counting across sessions, so codes stay distinct.
	if g2 := g; g2.Exports < 1 {
		t.Error("exports were not counted")
	}
}

func TestShareKeysInTheAlmanacAndShed(t *testing.T) {
	g, c := breeder(t)
	m := newModel(g, filepath.Join(t.TempDir(), "g.json"), testStart())
	m.width, m.height = 100, 36

	// e on a cultivar's page shows a code.
	var cur tea.Model = m
	cur = keyPress(cur, "a")
	am := cur.(model)
	for i, r := range am.almanacVisible() {
		if r.Kind == rowCultivar && r.Cultivar == c.ID {
			am.almanacCursor = i
		}
	}
	am.fixAlmanac()
	cur = am
	cur, cmd := cur.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	got := cur.(model)
	if got.screen != screenShare || !strings.HasPrefix(got.shareCode, "GD1-") || cmd == nil {
		t.Fatalf("e did not open the share screen: %v %q", got.screen, got.status)
	}
	v := plain(got.View())
	if !strings.Contains(v, "Moss Giant") || !strings.Contains(v, "GD1-") {
		t.Errorf("share screen:\n%s", v)
	}
	// every group of the code is on the screen
	for _, part := range strings.Split(got.shareCode, "-") {
		if !strings.Contains(v, part) {
			t.Errorf("share screen is missing %q", part)
		}
	}
	cur = keyPress(cur, "x")
	if cur.(model).screen != screenAlmanac {
		t.Error("any key should close the share screen")
	}

	// e off a cultivar says what to do.
	am2 := cur.(model)
	am2.almanacCursor = 1
	cur = keyPress(am2, "e")
	if cur.(model).screen == screenShare {
		t.Error("e on a guide chapter opened the share screen")
	}

	// A friend imports it from the shed.
	code := got.shareCode
	f := friend()
	fm := newModel(f, filepath.Join(t.TempDir(), "f.json"), testStart())
	fm.width, fm.height = 100, 36
	cur = keyPress(fm, "s")
	cur = keyPress(cur, "i")
	if !cur.(model).naming {
		t.Fatal("i did not ask for a code")
	}
	cm := cur.(model)
	cm.input.SetValue(code)
	cur = cm
	cur = keyPress(cur, "enter")
	fin := cur.(model)
	if f.SeedsInShed() != 3 || len(f.Cultivars) != 1 || !strings.Contains(fin.status, "Moss Giant") {
		t.Errorf("import failed: %d seeds, %d cultivars, %q", f.SeedsInShed(), len(f.Cultivars), fin.status)
	}
	if fin.input.CharLimit != 24 {
		t.Errorf("the input limit stayed at %d", fin.input.CharLimit)
	}

	// Pasting it again says so.
	cur = keyPress(cur, "i")
	cm = cur.(model)
	cm.input.SetValue(code)
	cur = keyPress(cm, "enter")
	if !strings.Contains(cur.(model).status, "already used") {
		t.Errorf("second import: %q", cur.(model).status)
	}
}

func TestFindCultivarForTheCommandLine(t *testing.T) {
	g := newTestGarden(testStart())
	if _, err := findCultivar(g, "x"); err == nil {
		t.Error("found a cultivar in an empty garden")
	}
	g.Cultivars = []Cultivar{{ID: 1, Name: "Moss Giant"}, {ID: 2, Name: "Moss Dwarf"}, {ID: 3, Name: "Blue"}}
	for query, want := range map[string]int{"blue": 3, "2": 2, "Moss Giant": 1, "giant": 1} {
		c, err := findCultivar(g, query)
		if err != nil || c.ID != want {
			t.Errorf("%q → %v, %v", query, c.ID, err)
		}
	}
	if _, err := findCultivar(g, "moss"); err == nil || !strings.Contains(err.Error(), "several") {
		t.Errorf("an ambiguous name was guessed: %v", err)
	}
	if _, err := findCultivar(g, "zzz"); err == nil {
		t.Error("a missing name was found")
	}
}

func TestShareScreenFitsSmallWindows(t *testing.T) {
	g, c := breeder(t)
	for _, size := range [][2]int{{100, 36}, {60, 20}, {40, 14}, {30, 12}} {
		m := newModel(g, "/tmp/g.json", testStart())
		m.width, m.height = size[0], size[1]
		m.shareCode, _, _ = func() (string, ShareCode, error) { return g.ExportCultivar(c.ID) }()
		m.shareName = c.Name
		m.screen = screenShare
		for i, line := range strings.Split(plain(m.View()), "\n") {
			if len([]rune(line)) > size[0] {
				t.Errorf("%dx%d: share line %d is %d wide", size[0], size[1], i+1, len([]rune(line)))
			}
		}
	}
}
