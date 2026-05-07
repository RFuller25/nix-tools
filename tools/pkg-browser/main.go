package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configFlag := flag.String("config", "", "path to NixOS configuration.nix (skips first-run prompt)")
	flag.Parse()

	var configPath string

	if *configFlag != "" {
		configPath = *configFlag
	} else {
		cfg, err := readAppConfig()
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

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = configPath // configPath will be wired into initialModel() in Task 5
}
