package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	username := flag.String("username", "", "username to use for this session")
	flag.StringVar(username, "u", "", "shorthand for -username")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	needsSetup := cfg == nil || cfg.APIKey == ""
	if !needsSetup && *username != "" {
		cfg.Username = *username
	}

	p := tea.NewProgram(newModel(cfg, needsSetup), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
