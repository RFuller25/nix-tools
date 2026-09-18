package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "garden.json")
	now := testStart()

	g := newTestGarden(now)
	if err := g.Plant(2, SpeciesByID("foxglove"), now); err != nil {
		t.Fatal(err)
	}
	g.Rename(2, "Bertha", now)
	g.Plots[2].Growth = 0.6
	g.Plots[2].Moisture = 0.42

	if err := Save(path, g); err != nil {
		t.Fatalf("saving: %v", err)
	}

	back, err := Load(path, now)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if back.Seed != g.Seed || back.Seeds != g.Seeds || back.Planted != g.Planted {
		t.Errorf("garden fields did not survive the round trip")
	}
	p := back.Plots[2]
	if p.SpeciesID != "foxglove" || p.Name != "Bertha" {
		t.Errorf("plot came back as %+v", p)
	}
	if p.Growth != 0.6 || p.Moisture != 0.42 {
		t.Errorf("growth/moisture came back as %.2f/%.2f", p.Growth, p.Moisture)
	}
	if len(back.Journal) != len(g.Journal) {
		t.Errorf("journal has %d entries, want %d", len(back.Journal), len(g.Journal))
	}
}

func TestLoadMissingFileStartsFresh(t *testing.T) {
	now := testStart()
	g, err := Load(filepath.Join(t.TempDir(), "absent.json"), now)
	if err != nil {
		t.Fatalf("a missing save should not be an error: %v", err)
	}
	if len(g.Plots) != PlotCount {
		t.Errorf("fresh garden has %d plots, want %d", len(g.Plots), PlotCount)
	}
	if g.Seeds <= 0 {
		t.Error("a new gardener should start with some seeds")
	}
}

// A save written by an older version, or hand-edited, must not take the
// program down.
func TestLoadToleratesOldSaves(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "garden.json")
	body := `{
		"version": 1,
		"seed": 0,
		"plots": [
			{"species_id": "sunflower", "growth": 0.5},
			{"species_id": "extinct-plant", "growth": 0.9}
		],
		"seeds": 4
	}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := Load(path, testStart())
	if err != nil {
		t.Fatalf("loading a short save: %v", err)
	}
	if len(g.Plots) != PlotCount {
		t.Errorf("plots were not padded out: %d", len(g.Plots))
	}
	if g.Plots[0].SpeciesID != "sunflower" {
		t.Error("known species was dropped")
	}
	if !g.Plots[1].Empty() {
		t.Error("a species no longer in the catalogue should be cleared, not kept")
	}
	if g.Seed == 0 {
		t.Error("a garden with no seed should be given one")
	}
	if g.Created.IsZero() {
		t.Error("a garden with no creation date should be given one")
	}
}

func TestLoadRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garden.json")
	if err := os.WriteFile(path, []byte("this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, testStart()); err == nil {
		t.Error("garbage should report an error rather than silently resetting the garden")
	}
}

// Saving must never leave a half-written file behind, and must not leak temp
// files into the save directory.
func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "garden.json")
	now := testStart()
	g := newTestGarden(now)

	for i := 0; i < 5; i++ {
		g.Seeds = i
		if err := Save(path, g); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "garden.json" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("save directory holds %v, want just garden.json", names)
	}

	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if back.Seeds != 4 {
		t.Errorf("last save did not win: seeds = %d", back.Seeds)
	}
}

func TestSavePathHonoursXDG(t *testing.T) {
	t.Setenv("GARDEN_SAVE", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-test")
	got, err := savePath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/tmp/xdg-test", "garden", "garden.json"); got != want {
		t.Errorf("savePath() = %q, want %q", got, want)
	}

	t.Setenv("GARDEN_SAVE", "/tmp/explicit.json")
	if got, _ := savePath(); got != "/tmp/explicit.json" {
		t.Errorf("GARDEN_SAVE override gave %q", got)
	}
}

func TestJournalStaysBounded(t *testing.T) {
	g := newTestGarden(testStart())
	for i := 0; i < 500; i++ {
		g.Log(time.Now(), "entry %d", i)
	}
	if len(g.Journal) > 300 {
		t.Errorf("journal grew to %d entries", len(g.Journal))
	}
}
