package main

import (
	"testing"
)

func TestSubstituteAnswer(t *testing.T) {
	got := substituteAnswer("ANSWER * 2", "5")
	if got != "5 * 2" {
		t.Fatalf("got %q, want %q", got, "5 * 2")
	}
}

func TestSubstituteAnswerNoToken(t *testing.T) {
	got := substituteAnswer("2 + 2", "5")
	if got != "2 + 2" {
		t.Fatalf("got %q, want %q", got, "2 + 2")
	}
}

func TestSubstituteAnswerEmptyLast(t *testing.T) {
	got := substituteAnswer("ANSWER + 1", "")
	if got != "ANSWER + 1" {
		t.Fatalf("got %q, want %q", got, "ANSWER + 1")
	}
}

func TestSubstituteAnswerMultiple(t *testing.T) {
	got := substituteAnswer("ANSWER + ANSWER", "3")
	if got != "3 + 3" {
		t.Fatalf("got %q, want %q", got, "3 + 3")
	}
}

func TestParseQalcOutput(t *testing.T) {
	got := parseQalcOutput("2 + 2 = 4")
	if got != "4" {
		t.Fatalf("got %q, want %q", got, "4")
	}
}

func TestParseQalcOutputNoEquals(t *testing.T) {
	got := parseQalcOutput("4")
	if got != "4" {
		t.Fatalf("got %q, want %q", got, "4")
	}
}

func TestParseQalcOutputTrimmed(t *testing.T) {
	got := parseQalcOutput("  42 \n")
	if got != "42" {
		t.Fatalf("got %q, want %q", got, "42")
	}
}
