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
	style, _ := categorizeLine("fetching path '/nix/store/abc'")
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
