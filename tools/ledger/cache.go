package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// boardCache is the last board seen, so the screen has something to show
// while the one request at launch is in flight.
type boardCache struct {
	Username string `json:"username"`
	Balance  int    `json:"balance"`
	Bets     []Bet  `json:"bets"`
}

func cachePath() (string, error) {
	if p := os.Getenv("LEDGER_CACHE"); p != "" {
		return p, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ledger", "board.json"), nil
}

func loadCache(username string) *boardCache {
	path, err := cachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c boardCache
	if json.Unmarshal(data, &c) != nil || c.Username != username {
		return nil
	}
	return &c
}

func saveCache(c boardCache) {
	path, err := cachePath()
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	if data, err := json.Marshal(c); err == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
}
