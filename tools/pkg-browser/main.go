package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func validateConfigPath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}
	if !strings.Contains(string(data), "environment.systemPackages") {
		return fmt.Errorf("file does not contain environment.systemPackages")
	}
	return nil
}

func main() {
	configFlag := flag.String("config", "", "path to NixOS configuration.nix (skips first-run prompt)")
	flag.Parse()

	var configPath string

	if *configFlag != "" {
		if err := validateConfigPath(*configFlag); err != nil {
			fmt.Fprintf(os.Stderr, "error: --config: %v\n", err)
			os.Exit(1)
		}
		configPath = *configFlag
	} else {
		cfg, err := readAppConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read saved config: %v\n", err)
		}
		if err != nil || cfg.ConfigPath == "" {
			// First-run setup
			p := tea.NewProgram(newSetupModel())
			m, err := p.Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			result := m.(setupModel)
			if result.configPath == "" {
				os.Exit(0)
			}
			if err := writeAppConfig(AppConfig{ConfigPath: result.configPath}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save config: %v\n", err)
			}
			configPath = result.configPath
		} else {
			configPath = cfg.ConfigPath
		}
	}

	p := tea.NewProgram(initialModel(configPath), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
