package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func requireQalc(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("qalc"); err != nil {
		t.Skip("qalc not installed, skipping integration test")
	}
}

func TestResolveUnitCmdDefine(t *testing.T) {
	cmd, ok := resolveUnitCmd("unit smoot = 1.702 meter")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if cmd.kind != unitCmdDefine {
		t.Fatalf("kind = %v, want unitCmdDefine", cmd.kind)
	}
	if cmd.name != "smoot" {
		t.Fatalf("name = %q, want %q", cmd.name, "smoot")
	}
	if cmd.expr != "1.702 meter" {
		t.Fatalf("expr = %q, want %q", cmd.expr, "1.702 meter")
	}
}

func TestResolveUnitCmdDefineExtraSpace(t *testing.T) {
	cmd, ok := resolveUnitCmd("unit   smoot   =   1.702 meter  ")
	if !ok || cmd.kind != unitCmdDefine || cmd.name != "smoot" || cmd.expr != "1.702 meter" {
		t.Fatalf("got %+v, ok=%v", cmd, ok)
	}
}

func TestResolveUnitCmdList(t *testing.T) {
	cmd, ok := resolveUnitCmd("unit list")
	if !ok || cmd.kind != unitCmdList {
		t.Fatalf("got %+v, ok=%v", cmd, ok)
	}
}

func TestResolveUnitCmdListCaseInsensitive(t *testing.T) {
	cmd, ok := resolveUnitCmd("UNIT LIST")
	if !ok || cmd.kind != unitCmdList {
		t.Fatalf("got %+v, ok=%v", cmd, ok)
	}
}

func TestResolveUnitCmdDelete(t *testing.T) {
	cmd, ok := resolveUnitCmd("unit delete smoot")
	if !ok || cmd.kind != unitCmdDelete || cmd.name != "smoot" {
		t.Fatalf("got %+v, ok=%v", cmd, ok)
	}
}

func TestResolveUnitCmdInvalid(t *testing.T) {
	cmd, ok := resolveUnitCmd("unit ???")
	if !ok || cmd.kind != unitCmdInvalid {
		t.Fatalf("got %+v, ok=%v", cmd, ok)
	}
}

func TestResolveUnitCmdNotAUnitCommand(t *testing.T) {
	if _, ok := resolveUnitCmd("2 + 2"); ok {
		t.Fatal("expected ok=false for plain expression")
	}
	if _, ok := resolveUnitCmd("united states"); ok {
		t.Fatal("expected ok=false, 'unit' must be a standalone keyword")
	}
	if _, ok := resolveUnitCmd("unit"); ok {
		t.Fatal("expected ok=false for bare 'unit' with no argument")
	}
}

func TestParseBaseUnitSimple(t *testing.T) {
	relation, symbol, exp, err := parseBaseUnit("1.702 meters = 1.702 m")
	if err != nil {
		t.Fatal(err)
	}
	if relation != "1.702" || symbol != "m" || exp != 1 {
		t.Fatalf("got (%q, %q, %d)", relation, symbol, exp)
	}
}

func TestParseBaseUnitDifferentDimension(t *testing.T) {
	relation, symbol, exp, err := parseBaseUnit("2000 pounds = 907.18474 kg")
	if err != nil {
		t.Fatal(err)
	}
	if relation != "907.18474" || symbol != "kg" || exp != 1 {
		t.Fatalf("got (%q, %q, %d)", relation, symbol, exp)
	}
}

func TestParseBaseUnitSuperscriptExponent(t *testing.T) {
	relation, symbol, exp, err := parseBaseUnit("34.0691 liters = 0.0340691 m³")
	if err != nil {
		t.Fatal(err)
	}
	if relation != "0.0340691" || symbol != "m" || exp != 3 {
		t.Fatalf("got (%q, %q, %d)", relation, symbol, exp)
	}
}

func TestParseBaseUnitRejectsSlashCompound(t *testing.T) {
	_, _, _, err := parseBaseUnit("5 meters/second = 5 m/s")
	if err == nil {
		t.Fatal("expected error for compound unit m/s")
	}
}

func TestParseBaseUnitRejectsMultiTokenCompound(t *testing.T) {
	_, _, _, err := parseBaseUnit("1 x = 2 kg m")
	if err == nil {
		t.Fatal("expected error for multi-token compound result")
	}
}

func TestUnitDefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "units.xml")

	doc, err := loadUnitDefs(path)
	if err != nil {
		t.Fatalf("load missing file: %v", err)
	}
	if len(doc.Units) != 0 {
		t.Fatalf("expected empty doc, got %+v", doc)
	}

	doc, err = addUnitEntry(doc, "smoot", "m", "1.702", 1)
	if err != nil {
		t.Fatal(err)
	}
	doc, err = addUnitEntry(doc, "firkin", "m", "0.0340691", 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveUnitDefs(path, doc); err != nil {
		t.Fatal(err)
	}

	reloaded, err := loadUnitDefs(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Units) != 2 {
		t.Fatalf("expected 2 units after reload, got %d", len(reloaded.Units))
	}

	reloaded, found := deleteUnitEntry(reloaded, "smoot")
	if !found {
		t.Fatal("expected to find smoot")
	}
	if err := saveUnitDefs(path, reloaded); err != nil {
		t.Fatal(err)
	}

	final, err := loadUnitDefs(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Units) != 1 || final.Units[0].Names != "firkin" {
		t.Fatalf("expected only firkin left, got %+v", final.Units)
	}
}

func TestAddUnitEntryDuplicateRejected(t *testing.T) {
	doc, err := addUnitEntry(unitDefsDoc{}, "smoot", "m", "1.702", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := addUnitEntry(doc, "smoot", "m", "2", 1); err == nil {
		t.Fatal("expected error defining smoot twice")
	}
}

func TestDeleteUnitEntryNotFound(t *testing.T) {
	_, found := deleteUnitEntry(unitDefsDoc{}, "nope")
	if found {
		t.Fatal("expected found=false for empty doc")
	}
}

func TestFormatUnitEntriesEmpty(t *testing.T) {
	if got := formatUnitEntries(unitDefsDoc{}); got != "no custom units defined" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatUnitEntriesNonEmpty(t *testing.T) {
	doc, _ := addUnitEntry(unitDefsDoc{}, "smoot", "m", "1.702", 1)
	got := formatUnitEntries(doc)
	if got != "smoot = 1.702 m^1" {
		t.Fatalf("got %q", got)
	}
}

func TestQalcNameExistsKnownUnit(t *testing.T) {
	requireQalc(t)
	if !qalcNameExists("meter") {
		t.Fatal("expected meter to be recognized by qalc")
	}
}

func TestQalcNameExistsUnknownName(t *testing.T) {
	requireQalc(t)
	if qalcNameExists("totallynotarealunitxyz123") {
		t.Fatal("expected unknown name to not exist")
	}
}

func TestQalcNameExistsIgnoresFuzzyImplicitMultiplication(t *testing.T) {
	requireQalc(t)
	// "qalc smoot" (as a bare one-shot expression) succeeds with exit 0 by
	// fuzzy-parsing it as second*milli*byte*tonne, even though "smoot" is
	// not a real defined name. qalcNameExists must not be fooled by that.
	if qalcNameExists("smoot") {
		t.Fatal("expected smoot to not exist as a real unit/variable/function name")
	}
}

func TestQalcToBaseSimple(t *testing.T) {
	requireQalc(t)
	out, err := qalcToBase("1.702 meter")
	if err != nil {
		t.Fatal(err)
	}
	relation, symbol, exp, err := parseBaseUnit(out)
	if err != nil {
		t.Fatalf("parseBaseUnit(%q): %v", out, err)
	}
	if relation != "1.702" || symbol != "m" || exp != 1 {
		t.Fatalf("got (%q, %q, %d)", relation, symbol, exp)
	}
}

func TestQalcToBaseDifferentDimension(t *testing.T) {
	requireQalc(t)
	out, err := qalcToBase("2000 lb")
	if err != nil {
		t.Fatal(err)
	}
	_, symbol, exp, err := parseBaseUnit(out)
	if err != nil {
		t.Fatalf("parseBaseUnit(%q): %v", out, err)
	}
	if symbol != "kg" || exp != 1 {
		t.Fatalf("got symbol=%q exp=%d", symbol, exp)
	}
}

func TestQalcToBaseTimeNotMixed(t *testing.T) {
	requireQalc(t)
	// Regression: qalc renders "864 seconds" as "14 min + 24 s" by default
	// (mixed-unit display), which previously made this look like a
	// compound unit and fail. "-set conv none" in qalcToBase must suppress
	// that so this stays a single-term result.
	out, err := qalcToBase("864 seconds")
	if err != nil {
		t.Fatal(err)
	}
	relation, symbol, exp, err := parseBaseUnit(out)
	if err != nil {
		t.Fatalf("parseBaseUnit(%q): %v", out, err)
	}
	if relation != "864" || symbol != "s" || exp != 1 {
		t.Fatalf("got (%q, %q, %d)", relation, symbol, exp)
	}
}

func TestEvalUnitDefCmdNotAUnitCommand(t *testing.T) {
	if evalUnitDefCmd("2 + 2") != nil {
		t.Fatal("expected nil for a plain expression")
	}
}

func TestEvalUnitDefCmdFullLifecycle(t *testing.T) {
	requireQalc(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	// "glorp" (not "smoot") deliberately: qalc's fuzzy implicit-multiplication
	// parser resolves "smoot" as gram*liter*byte with exit 0, which makes
	// qalcNameExists falsely report it as already defined. "glorp" reliably
	// exits nonzero as unrecognized.
	cmd := evalUnitDefCmd("unit glorp = 1.702 meter")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	res, ok := msg.(resultMsg)
	if !ok {
		t.Fatalf("expected resultMsg, got %#v", msg)
	}
	if res.display != "glorp = 1.702 m (saved)" {
		t.Fatalf("display = %q", res.display)
	}

	dupMsg := evalUnitDefCmd("unit glorp = 1.702 meter")()
	if _, ok := dupMsg.(calcErrMsg); !ok {
		t.Fatalf("expected calcErrMsg defining glorp twice, got %#v", dupMsg)
	}

	listMsg := evalUnitDefCmd("unit list")()
	listRes, ok := listMsg.(resultMsg)
	if !ok {
		t.Fatalf("expected resultMsg, got %#v", listMsg)
	}
	if listRes.display != "glorp = 1.702 m^1" {
		t.Fatalf("list display = %q", listRes.display)
	}

	delMsg := evalUnitDefCmd("unit delete glorp")()
	delRes, ok := delMsg.(resultMsg)
	if !ok {
		t.Fatalf("expected resultMsg, got %#v", delMsg)
	}
	if delRes.display != "glorp deleted" {
		t.Fatalf("delete display = %q", delRes.display)
	}

	missingMsg := evalUnitDefCmd("unit delete glorp")()
	if _, ok := missingMsg.(calcErrMsg); !ok {
		t.Fatalf("expected calcErrMsg deleting already-deleted unit, got %#v", missingMsg)
	}

	emptyListMsg := evalUnitDefCmd("unit list")()
	emptyListRes, ok := emptyListMsg.(resultMsg)
	if !ok || emptyListRes.display != "no custom units defined" {
		t.Fatalf("got %#v", emptyListMsg)
	}
}

func TestEvalUnitDefCmdRejectsCompound(t *testing.T) {
	requireQalc(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	msg := evalUnitDefCmd("unit speed = 5 m/s")()
	if _, ok := msg.(calcErrMsg); !ok {
		t.Fatalf("expected calcErrMsg for compound unit, got %#v", msg)
	}
}

func TestEvalUnitDefCmdSmootDefinesSuccessfully(t *testing.T) {
	requireQalc(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	// Regression test: this exact input previously failed with
	// "smoot: already defined" because the old collision check
	// (exec.Command("qalc", name).Run() == nil) treated qalc's fuzzy
	// implicit-multiplication parse of "smoot" as a real name collision.
	msg := evalUnitDefCmd("unit smoot = 1.702 meter")()
	res, ok := msg.(resultMsg)
	if !ok {
		t.Fatalf("expected resultMsg, got %#v", msg)
	}
	if res.display != "smoot = 1.702 m (saved)" {
		t.Fatalf("display = %q", res.display)
	}
}

func TestEvalUnitDefCmdChronDefinesSuccessfully(t *testing.T) {
	requireQalc(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	// Regression test: "unit chron = 864 seconds" previously failed with
	// `expected a single unit term, got "14 min + 24 s"` because qalc's
	// default mixed-unit display split the duration even under "to base".
	msg := evalUnitDefCmd("unit chron = 864 seconds")()
	res, ok := msg.(resultMsg)
	if !ok {
		t.Fatalf("expected resultMsg, got %#v", msg)
	}
	if res.display != "chron = 864 s (saved)" {
		t.Fatalf("display = %q", res.display)
	}
}
