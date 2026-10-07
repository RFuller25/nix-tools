package main

import (
	"path/filepath"
	"testing"
)

// v1Fixture is a garden as the first version of the game saved it: seeds as
// currency, plants as a species and a variety index, nothing else.
func v1Fixture() string { return filepath.Join("testdata", "v1_garden.json") }

func TestV1FixtureLoads(t *testing.T) {
	g, err := Load(v1Fixture(), testStart())
	if err != nil {
		t.Fatalf("loading the v1 fixture: %v", err)
	}
	if len(g.Plots) != PlotCount {
		t.Errorf("fixture has %d beds", len(g.Plots))
	}
}
