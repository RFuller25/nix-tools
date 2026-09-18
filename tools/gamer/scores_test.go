package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordKeepsTheBest(t *testing.T) {
	s := newScores()
	now := time.Now()

	if !s.Record("tetris", 100, false, now) {
		t.Error("the first score should always be a best")
	}
	if s.Record("tetris", 50, false, now) {
		t.Error("a lower score beat a higher one where higher is better")
	}
	if !s.Record("tetris", 500, false, now) {
		t.Error("a higher score did not take the record")
	}
	if s.Best["tetris"] != 500 {
		t.Errorf("best = %d, want 500", s.Best["tetris"])
	}
	if s.Played["tetris"] != 3 {
		t.Errorf("played = %d, want 3", s.Played["tetris"])
	}
}

func TestRecordWhereLowerIsBetter(t *testing.T) {
	s := newScores()
	now := time.Now()

	s.Record("hue", 40, true, now)
	if s.Record("hue", 55, true, now) {
		t.Error("more moves should not beat fewer")
	}
	if !s.Record("hue", 22, true, now) {
		t.Error("fewer moves did not take the record")
	}
	if s.Best["hue"] != 22 {
		t.Errorf("best = %d, want 22", s.Best["hue"])
	}
}

func TestScoresRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "scores.json")
	s := newScores()
	now := time.Now().Truncate(time.Second)
	s.Record("snake", 120, false, now)
	s.Record("mines", 93, true, now)

	if err := Save(path, s); err != nil {
		t.Fatalf("saving: %v", err)
	}
	back, err := LoadScores(path)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if back.Best["snake"] != 120 || back.Best["mines"] != 93 {
		t.Errorf("scores came back as %v", back.Best)
	}
	if !back.LastRun["snake"].Equal(now) {
		t.Errorf("last run came back as %v, want %v", back.LastRun["snake"], now)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s, err := LoadScores(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("a missing score file should not be an error: %v", err)
	}
	if len(s.Best) != 0 {
		t.Errorf("a fresh record holds %d scores", len(s.Best))
	}
	// The maps must be usable straight away.
	s.Record("tetris", 1, false, time.Now())
}

func TestLoadGarbageReportsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.json")
	if err := os.WriteFile(path, []byte("{{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScores(path); err == nil {
		t.Error("a corrupt score file should be reported, not silently reset")
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scores.json")
	s := newScores()
	for i := 0; i < 5; i++ {
		s.Record("snake", i*10, false, time.Now())
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "scores.json" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("save directory holds %v, want just scores.json", names)
	}
}

func TestScorePathHonoursXDG(t *testing.T) {
	t.Setenv("GAMER_SCORES", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-gamer")
	got, err := scorePath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/tmp/xdg-gamer", "gamer", "scores.json"); got != want {
		t.Errorf("scorePath() = %q, want %q", got, want)
	}

	t.Setenv("GAMER_SCORES", "/tmp/explicit.json")
	if got, _ := scorePath(); got != "/tmp/explicit.json" {
		t.Errorf("GAMER_SCORES override gave %q", got)
	}
}
