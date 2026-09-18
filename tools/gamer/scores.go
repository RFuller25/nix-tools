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

// Scores is the persisted record of how well you have done at each game.
type Scores struct {
	Best    map[string]int       `json:"best"`
	Played  map[string]int       `json:"played"`
	LastRun map[string]time.Time `json:"last_run"`
}

// scorePath follows the XDG base directory spec, with an override for tests
// and for anyone who keeps their dotfiles somewhere unusual.
func scorePath() (string, error) {
	if p := os.Getenv("GAMER_SCORES"); p != "" {
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
	return filepath.Join(dir, "gamer", "scores.json"), nil
}

func newScores() *Scores {
	return &Scores{
		Best:    map[string]int{},
		Played:  map[string]int{},
		LastRun: map[string]time.Time{},
	}
}

// LoadScores reads the score file, returning an empty record when there is
// none. A corrupt file is reported rather than silently wiped.
func LoadScores(path string) (*Scores, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return newScores(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	s := newScores()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if s.Best == nil {
		s.Best = map[string]int{}
	}
	if s.Played == nil {
		s.Played = map[string]int{}
	}
	if s.LastRun == nil {
		s.LastRun = map[string]time.Time{}
	}
	return s, nil
}

// Record files a finished game. It reports whether this run set a new best.
//
// Some games are won by a small number: fewest moves, fewest seconds. Those
// pass lowerIsBetter so the comparison runs the other way.
func (s *Scores) Record(game string, score int, lowerIsBetter bool, now time.Time) bool {
	s.Played[game]++
	s.LastRun[game] = now

	best, seen := s.Best[game]
	switch {
	case !seen:
		s.Best[game] = score
		return true
	case lowerIsBetter && score < best:
		s.Best[game] = score
		return true
	case !lowerIsBetter && score > best:
		s.Best[game] = score
		return true
	}
	return false
}

// Save writes the scores atomically, so a crash mid-write cannot destroy the
// record of a good run.
func Save(path string, s *Scores) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding scores: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".scores-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)

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
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}
