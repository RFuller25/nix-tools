package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCategorizeLineBuilding(t *testing.T) {
	style, _ := categorizeLine("building '/nix/store/abc-foo.drv'...")
	if style != styleBuilding {
		t.Fatalf("expected styleBuilding, got %v", style)
	}
}

func TestCategorizeLineFetching(t *testing.T) {
	style, _ := categorizeLine("copying path '/nix/store/abc' from 'https://cache.nixos.org'")
	if style != styleFetching {
		t.Fatalf("expected styleFetching, got %v", style)
	}
}

func TestCategorizeLineActivating(t *testing.T) {
	style, _ := categorizeLine("activating the configuration...")
	if style != styleActivating {
		t.Fatalf("expected styleActivating, got %v", style)
	}
}

func TestCategorizeLineError(t *testing.T) {
	style, _ := categorizeLine("error: some build failure")
	if style != styleError {
		t.Fatalf("expected styleError, got %v", style)
	}
}

func TestCategorizeLineDefault(t *testing.T) {
	style, _ := categorizeLine("some random output line")
	if style != styleDefault {
		t.Fatalf("expected styleDefault, got %v", style)
	}
}

// Ensure categorizeLine returns the original text unchanged.
func TestCategorizeLinePreservesText(t *testing.T) {
	_, text := categorizeLine("error: build failed")
	if text != "error: build failed" {
		t.Fatalf("text modified: %q", text)
	}
}

// silence unused import if lipgloss not yet used in tests
var _ = lipgloss.NewStyle

func TestStripANSI(t *testing.T) {
	got := stripANSI("\x1b[32m> Building\x1b[0m")
	if got != "> Building" {
		t.Errorf("got %q, want %q", got, "> Building")
	}
}

func TestStripANSINoCode(t *testing.T) {
	got := stripANSI("plain text")
	if got != "plain text" {
		t.Errorf("got %q, want %q", got, "plain text")
	}
}

func TestUpdateStatsFetchPaths(t *testing.T) {
	s := stats{}
	updateStats("these 15 paths will be fetched (142.53 MiB download, 487.23 MiB unpacked):", &s)
	if s.totalPaths != 15 {
		t.Errorf("totalPaths = %d, want 15", s.totalPaths)
	}
	if s.totalMiB != 142.53 {
		t.Errorf("totalMiB = %f, want 142.53", s.totalMiB)
	}
	if s.phase != "fetching" {
		t.Errorf("phase = %q, want %q", s.phase, "fetching")
	}
}

func TestUpdateStatsFetchPathsGiB(t *testing.T) {
	s := stats{}
	updateStats("these 5 paths will be fetched (1.5 GiB download, 3.2 GiB unpacked):", &s)
	if s.totalPaths != 5 {
		t.Errorf("totalPaths = %d, want 5", s.totalPaths)
	}
	if s.totalMiB != 1536.0 {
		t.Errorf("totalMiB = %f, want 1536.0", s.totalMiB)
	}
}

func TestUpdateStatsCopyingPath(t *testing.T) {
	s := stats{totalPaths: 10}
	updateStats("copying path '/nix/store/hash-pkg' from 'https://cache.nixos.org'...", &s)
	if s.fetchedPaths != 1 {
		t.Errorf("fetchedPaths = %d, want 1", s.fetchedPaths)
	}
}

func TestUpdateStatsTotalDrvs(t *testing.T) {
	s := stats{}
	updateStats("these 3 derivations will be built:", &s)
	if s.totalDrvs != 3 {
		t.Errorf("totalDrvs = %d, want 3", s.totalDrvs)
	}
}

func TestUpdateStatsBuildingDrv(t *testing.T) {
	s := stats{}
	updateStats("building '/nix/store/hash-pkg.drv'...", &s)
	if s.builtDrvs != 1 {
		t.Errorf("builtDrvs = %d, want 1", s.builtDrvs)
	}
	if s.phase != "building" {
		t.Errorf("phase = %q, want building", s.phase)
	}
}

func TestUpdateStatsNVDChanged(t *testing.T) {
	s := stats{}
	updateStats("[U.]  #001  firefox  149.0.2 -> 150.0", &s)
	if s.pkgsChanged != 1 {
		t.Errorf("pkgsChanged = %d, want 1", s.pkgsChanged)
	}
}

func TestUpdateStatsNVDAdded(t *testing.T) {
	s := stats{}
	updateStats("[A.]  #001  new-package  1.0.0", &s)
	if s.pkgsAdded != 1 {
		t.Errorf("pkgsAdded = %d, want 1", s.pkgsAdded)
	}
}

func TestUpdateStatsNVDRemoved(t *testing.T) {
	s := stats{}
	updateStats("[R.]  #001  old-package  2.0.0", &s)
	if s.pkgsRemoved != 1 {
		t.Errorf("pkgsRemoved = %d, want 1", s.pkgsRemoved)
	}
}

func TestUpdateStatsDiskDelta(t *testing.T) {
	s := stats{}
	updateStats("Closure size: 2440 -> 2455 (2426 paths added, 2411 paths removed, delta +15, disk usage +406.6MiB).", &s)
	if s.diskDelta == "" {
		t.Error("diskDelta should be set")
	}
	if s.diskDelta != "+406.6MiB" {
		t.Errorf("diskDelta = %q, want %q", s.diskDelta, "+406.6MiB")
	}
}

func TestUpdateStatsNHBuildHeader(t *testing.T) {
	s := stats{}
	updateStats("> Building NixOS configuration", &s)
	if s.phase != "building" {
		t.Errorf("phase = %q, want building", s.phase)
	}
}

func TestUpdateStatsNHActivateHeader(t *testing.T) {
	s := stats{phase: "building"}
	updateStats("> Activating", &s)
	if s.phase != "activating" {
		t.Errorf("phase = %q, want activating", s.phase)
	}
}

func TestUpdateStatsActivatingLine(t *testing.T) {
	s := stats{}
	updateStats("activating the configuration...", &s)
	if s.phase != "activating" {
		t.Errorf("phase = %q, want activating", s.phase)
	}
}

func TestCategorizeLineNVDChanged(t *testing.T) {
	style, _ := categorizeLine("[U.]  #001  firefox  149.0 -> 150.0")
	if style != styleChanged {
		t.Fatalf("expected styleChanged")
	}
}

func TestCategorizeLineNVDAdded(t *testing.T) {
	style, _ := categorizeLine("[A.]  #001  new-package  1.0.0")
	if style != styleAdded {
		t.Fatalf("expected styleAdded")
	}
}

func TestCategorizeLineNVDRemoved(t *testing.T) {
	style, _ := categorizeLine("[R.]  #001  old-package  2.0.0")
	if style != styleRemoved {
		t.Fatalf("expected styleRemoved")
	}
}

func TestCategorizeLineCopyingPath(t *testing.T) {
	style, _ := categorizeLine("copying path '/nix/store/hash' from 'https://cache.nixos.org'...")
	if style != styleFetching {
		t.Fatalf("expected styleFetching")
	}
}

func TestCategorizeLineSectionHeader(t *testing.T) {
	style, _ := categorizeLine("> Building NixOS configuration")
	if style != styleSection {
		t.Fatalf("expected styleSection")
	}
}
