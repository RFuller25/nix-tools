package main

import (
	"testing"
)

// ── formatKaktovik ─────────────────────────────────────────────────────────

func TestFormatKaktovikMidnight(t *testing.T) {
	got := formatKaktovik(0)
	// 0 ticks → 𝋀:𝋀:𝋀
	want := kakDigit(0) + ":" + kakDigit(0) + ":" + kakDigit(0)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatKaktovikHalfDay(t *testing.T) {
	// 43200 s = half day = 4000 kak-ticks
	// h=10 (4000/400), m=0, s=0
	got := formatKaktovik(43200)
	want := kakDigit(10) + ":" + kakDigit(0) + ":" + kakDigit(0)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatKaktovikRoundtrip(t *testing.T) {
	// 14:30:00 SI = 52200 s
	// ticks = floor(52200*8000/86400) = floor(4833.33) = 4833
	// h=floor(4833/400)=12, rem=33, m=floor(33/20)=1, s=13
	got := formatKaktovik(52200)
	want := kakDigit(12) + ":" + kakDigit(1) + ":" + kakDigit(13)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatKaktovikWrapAround(t *testing.T) {
	// 86400 s = full day → wraps to midnight
	got := formatKaktovik(86400)
	want := kakDigit(0) + ":" + kakDigit(0) + ":" + kakDigit(0)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── formatChron ────────────────────────────────────────────────────────────

func TestFormatChronMidnight(t *testing.T) {
	got := formatChron(0)
	if got != "0:00:00" {
		t.Errorf("got %q, want %q", got, "0:00:00")
	}
}

func TestFormatChronHalfDay(t *testing.T) {
	// 43200 s = half day = 50000 chron-ticks → 5:00:00
	got := formatChron(43200)
	if got != "5:00:00" {
		t.Errorf("got %q, want %q", got, "5:00:00")
	}
}

func TestFormatChronEndOfDay(t *testing.T) {
	// 86399 s → ticks = floor(86399*100000/86400) = 99998
	// h=9, m=99, s=98
	got := formatChron(86399)
	if got != "9:99:98" {
		t.Errorf("got %q, want %q", got, "9:99:98")
	}
}

// ── formatDuod ─────────────────────────────────────────────────────────────

func TestFormatDuodMidnight(t *testing.T) {
	got := formatDuod(0)
	want := "0:0:0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatDuodHalfDay(t *testing.T) {
	// 43200 s = half day = 864 duod-ticks
	// h=floor(864/144)=6, rem=0, m=0, s=0 → 6:0:0
	got := formatDuod(43200)
	if got != "6:0:0" {
		t.Errorf("got %q, want %q", got, "6:0:0")
	}
}

func TestFormatDuodDecElph(t *testing.T) {
	// 14:30:00 SI = 52200 s
	// ticks = floor(52200*1728/86400) = floor(1044) = 1044
	// h=floor(1044/144)=7, rem=36, m=floor(36/12)=3, s=0
	got := formatDuod(52200)
	if got != "7:3:0" {
		t.Errorf("got %q, want %q", got, "7:3:0")
	}
}

func TestFormatDuodMaxTime(t *testing.T) {
	// ticks=1727 → h=11, m=11, s=11 → ↋:↋:↋
	secs := float64(1727) * 86400 / 1728
	got := formatDuod(secs)
	want := duodDigit(11) + ":" + duodDigit(11) + ":" + duodDigit(11)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── parseSecondsFromDisplay ────────────────────────────────────────────────

func TestParseSecondsBasic(t *testing.T) {
	v, err := parseSecondsFromDisplay("52200 s")
	if err != nil {
		t.Fatal(err)
	}
	if v != 52200 {
		t.Errorf("got %v, want 52200", v)
	}
}

func TestParseSecondsWithComma(t *testing.T) {
	v, err := parseSecondsFromDisplay("52,200 s")
	if err != nil {
		t.Fatal(err)
	}
	if v != 52200 {
		t.Errorf("got %v, want 52200", v)
	}
}

func TestParseSecondsNotSeconds(t *testing.T) {
	_, err := parseSecondsFromDisplay("62 mi")
	if err == nil {
		t.Fatal("expected error for non-seconds display")
	}
}

// ── resolveAltTime ─────────────────────────────────────────────────────────

func TestResolveAltTimeKak(t *testing.T) {
	base, target, ok := resolveAltTime("14:30:00 to kak")
	if !ok {
		t.Fatal("expected match")
	}
	if base != "14:30:00" {
		t.Errorf("base = %q", base)
	}
	if target != "kak" {
		t.Errorf("target = %q", target)
	}
}

func TestResolveAltTimeKaktovik(t *testing.T) {
	_, target, ok := resolveAltTime("1 hour to Kaktovik")
	if !ok {
		t.Fatal("expected match")
	}
	if target != "kaktovik" {
		t.Errorf("target = %q", target)
	}
}

func TestResolveAltTimeChron(t *testing.T) {
	_, target, ok := resolveAltTime("3600 s to Chron")
	if !ok {
		t.Fatal("expected match")
	}
	if target != "chron" {
		t.Errorf("target = %q", target)
	}
}

func TestResolveAltTimeDuo(t *testing.T) {
	_, target, ok := resolveAltTime("1 day to Duo")
	if !ok {
		t.Fatal("expected match")
	}
	if target != "duo" {
		t.Errorf("target = %q", target)
	}
}

func TestResolveAltTimeNoMatch(t *testing.T) {
	_, _, ok := resolveAltTime("100 km to miles")
	if ok {
		t.Fatal("expected no match for normal qalc conversion")
	}
}

func TestResolveAltTimeChr(t *testing.T) {
	base, target, ok := resolveAltTime("52200 s to chr")
	if !ok {
		t.Fatal("expected match")
	}
	if base != "52200 s" {
		t.Errorf("base = %q", base)
	}
	if target != "chr" {
		t.Errorf("target = %q", target)
	}
}
