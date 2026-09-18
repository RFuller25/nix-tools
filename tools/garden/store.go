package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// savePath returns the garden's save file, following the XDG base directory
// spec: $XDG_DATA_HOME/garden/garden.json, else ~/.local/share/garden.
func savePath() (string, error) {
	if p := os.Getenv("GARDEN_SAVE"); p != "" {
		return p, nil
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locating home directory: %w", err)
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "garden", "garden.json"), nil
}

// Load reads the garden from disk, returning a fresh one when no save exists.
func Load(path string, now time.Time) (*Garden, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return NewGarden(now), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var g Garden
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	// Old saves, or a hand-edited one, may hold the wrong number of beds.
	switch {
	case len(g.Plots) < PlotCount:
		plots := make([]Plot, PlotCount)
		copy(plots, g.Plots)
		g.Plots = plots
	case len(g.Plots) > maxPlots:
		g.Plots = g.Plots[:maxPlots]
	}
	if g.Seed == 0 {
		g.Seed = now.UnixNano()
	}
	if g.Created.IsZero() {
		g.Created = now
	}
	g.layOutSoil() // fills in soil for beds saved before the ground was modelled
	// Drop plantings whose species no longer exists in the catalogue rather
	// than crashing on an unknown ID.
	for i := range g.Plots {
		if !g.Plots[i].Empty() && SpeciesByID(g.Plots[i].SpeciesID) == nil {
			g.Plots[i] = Plot{}
		}
	}
	g.Version = gardenVersion
	return &g, nil
}

// Save writes the garden atomically: a temp file in the same directory, then
// a rename, so an interrupted write can never shred an existing garden.
func Save(path string, g *Garden) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding garden: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".garden-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}
