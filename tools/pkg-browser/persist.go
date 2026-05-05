package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	ConfigPath string `json:"configPath"`
}

func configFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pkg-browser", "config.json"), nil
}

func readAppConfig() (AppConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AppConfig{}, nil
		}
		return AppConfig{}, err
	}
	var cfg AppConfig
	return cfg, json.Unmarshal(data, &cfg)
}

func writeAppConfig(cfg AppConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
