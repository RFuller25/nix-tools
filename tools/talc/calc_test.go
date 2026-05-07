package main

import (
	"testing"
)

// ── substituteAnswer ───────────────────────────────────────────────────────

func TestSubstituteAnswer(t *testing.T) {
	got, err := substituteAnswer("ANSWER * 2", []string{"5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "5 * 2" {
		t.Fatalf("got %q, want %q", got, "5 * 2")
	}
}

func TestSubstituteAnswerNoToken(t *testing.T) {
	got, err := substituteAnswer("2 + 2", []string{"5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "2 + 2" {
		t.Fatalf("got %q, want %q", got, "2 + 2")
	}
}

func TestSubstituteAnswerEmptyLast(t *testing.T) {
	got, err := substituteAnswer("ANSWER + 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ANSWER + 1" {
		t.Fatalf("got %q, want %q", got, "ANSWER + 1")
	}
}

func TestSubstituteAnswerMultiple(t *testing.T) {
	got, err := substituteAnswer("ANSWER + ANSWER", []string{"3"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "3 + 3" {
		t.Fatalf("got %q, want %q", got, "3 + 3")
	}
}

func TestSubstituteAnswerIndex1(t *testing.T) {
	got, err := substituteAnswer("ANS(1) + 1", []string{"10", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "10 + 1" {
		t.Fatalf("got %q, want %q", got, "10 + 1")
	}
}

func TestSubstituteAnswerIndex2(t *testing.T) {
	got, err := substituteAnswer("ANS(2) * 2", []string{"10", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "5 * 2" {
		t.Fatalf("got %q, want %q", got, "5 * 2")
	}
}

func TestSubstituteAnswerIndexCaseInsensitive(t *testing.T) {
	got, err := substituteAnswer("answer(1) + ANS(2)", []string{"10", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "10 + 5" {
		t.Fatalf("got %q, want %q", got, "10 + 5")
	}
}

func TestSubstituteAnswerIndexZeroInvalid(t *testing.T) {
	_, err := substituteAnswer("ANS(0) + 1", []string{"10"})
	if err == nil {
		t.Fatal("expected error for ANS(0)")
	}
}

func TestSubstituteAnswerIndexOutOfRange(t *testing.T) {
	_, err := substituteAnswer("ANS(3) + 1", []string{"10", "5"})
	if err == nil {
		t.Fatal("expected error for ANS(3) with only 2 answers")
	}
}

// ── parseResult ────────────────────────────────────────────────────────────

func TestParseResultExact(t *testing.T) {
	display, answer := parseResult("2 + 2 = 4")
	if display != "4" {
		t.Errorf("display = %q, want %q", display, "4")
	}
	if answer != "4" {
		t.Errorf("answer = %q, want %q", answer, "4")
	}
}

func TestParseResultApprox(t *testing.T) {
	display, answer := parseResult("100 kilometers ≈ 62 mi + 241 yd + 0.9895013123 ft")
	if display != "62 mi + 241 yd + 0.9895013123 ft" {
		t.Errorf("display = %q", display)
	}
	if answer != "62 mi" {
		t.Errorf("answer = %q, want %q", answer, "62 mi")
	}
}

func TestParseResultSingleUnit(t *testing.T) {
	display, answer := parseResult("5 celsius = 41 °F")
	if display != "41 °F" {
		t.Errorf("display = %q", display)
	}
	if answer != "41 °F" {
		t.Errorf("answer = %q", answer)
	}
}

func TestParseResultNoSeparator(t *testing.T) {
	display, answer := parseResult("42")
	if display != "42" {
		t.Errorf("display = %q", display)
	}
	if answer != "42" {
		t.Errorf("answer = %q", answer)
	}
}

func TestParseResultTrimsWhitespace(t *testing.T) {
	display, _ := parseResult("  2 + 2 = 4  \n")
	if display != "4" {
		t.Errorf("display = %q", display)
	}
}

// ── primaryUnit ────────────────────────────────────────────────────────────

func TestPrimaryUnitCompound(t *testing.T) {
	got := primaryUnit("62 mi + 241 yd + 0.9895 ft")
	if got != "62 mi" {
		t.Errorf("got %q, want %q", got, "62 mi")
	}
}

func TestPrimaryUnitSingle(t *testing.T) {
	got := primaryUnit("41 °F")
	if got != "41 °F" {
		t.Errorf("got %q, want %q", got, "41 °F")
	}
}

func TestPrimaryUnitPlain(t *testing.T) {
	got := primaryUnit("4")
	if got != "4" {
		t.Errorf("got %q, want %q", got, "4")
	}
}

// ── resolveExpr ────────────────────────────────────────────────────────────

func TestResolveExprAutoPrefix(t *testing.T) {
	expr, displayExpr, isAuto, err := resolveExpr("/ 4", []string{"20"})
	if err != nil {
		t.Fatal(err)
	}
	if expr != "20 / 4" {
		t.Errorf("expr = %q, want %q", expr, "20 / 4")
	}
	if displayExpr != "ANSWER / 4" {
		t.Errorf("displayExpr = %q, want %q", displayExpr, "ANSWER / 4")
	}
	if !isAuto {
		t.Error("isAuto should be true")
	}
}

func TestResolveExprAutoPrefixTo(t *testing.T) {
	expr, displayExpr, isAuto, err := resolveExpr("to miles", []string{"100 km"})
	if err != nil {
		t.Fatal(err)
	}
	if expr != "100 km to miles" {
		t.Errorf("expr = %q, want %q", expr, "100 km to miles")
	}
	if displayExpr != "ANSWER to miles" {
		t.Errorf("displayExpr = %q", displayExpr)
	}
	if !isAuto {
		t.Error("isAuto should be true")
	}
}

func TestResolveExprAutoPrefixMinus(t *testing.T) {
	_, _, isAuto, _ := resolveExpr("- 3", []string{"10"})
	if !isAuto {
		t.Error("'- 3' should trigger auto-prefix")
	}
	_, _, isAutoNeg, _ := resolveExpr("-3", []string{"10"})
	if isAutoNeg {
		t.Error("'-3' should NOT trigger auto-prefix (negative number)")
	}
}

func TestResolveExprNoAutoPrefixWhenEmpty(t *testing.T) {
	expr, displayExpr, isAuto, err := resolveExpr("/ 4", nil)
	if err != nil {
		t.Fatal(err)
	}
	if isAuto {
		t.Error("isAuto should be false when lastResult is empty")
	}
	if expr != "/ 4" {
		t.Errorf("expr = %q", expr)
	}
	if displayExpr != "/ 4" {
		t.Errorf("displayExpr = %q", displayExpr)
	}
}

func TestResolveExprExplicitAnswer(t *testing.T) {
	expr, displayExpr, isAuto, err := resolveExpr("ANSWER * 2", []string{"5"})
	if err != nil {
		t.Fatal(err)
	}
	if expr != "5 * 2" {
		t.Errorf("expr = %q, want %q", expr, "5 * 2")
	}
	if displayExpr != "ANSWER * 2" {
		t.Errorf("displayExpr = %q", displayExpr)
	}
	if isAuto {
		t.Error("isAuto should be false for explicit ANSWER substitution")
	}
}

func TestResolveExprPlain(t *testing.T) {
	expr, displayExpr, isAuto, err := resolveExpr("2 + 2", []string{"5"})
	if err != nil {
		t.Fatal(err)
	}
	if expr != "2 + 2" {
		t.Errorf("expr = %q", expr)
	}
	if displayExpr != "2 + 2" {
		t.Errorf("displayExpr = %q", displayExpr)
	}
	if isAuto {
		t.Error("isAuto should be false for plain expression")
	}
}

func TestResolveExprAnsIndex(t *testing.T) {
	expr, _, _, err := resolveExpr("ANS(2) + 1", []string{"10", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if expr != "5 + 1" {
		t.Errorf("expr = %q, want %q", expr, "5 + 1")
	}
}
