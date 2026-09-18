package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.1.0"

func main() {
	var (
		savePathFlag = flag.String("save", "", "path to the garden save file (default: $XDG_DATA_HOME/garden/garden.json)")
		showVersion  = flag.Bool("version", false, "print version and exit")
		listSpecies  = flag.Bool("species", false, "list every species in the almanac and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("garden %s\n", version)
		return
	}

	if *listSpecies {
		for _, sp := range AllSpecies() {
			fmt.Printf("%-24s %-32s %-14s %s\n", sp.Common, sp.Latin, sp.Kind, sp.Rarity)
		}
		fmt.Printf("\n%d species\n", len(AllSpecies()))
		return
	}

	path := *savePathFlag
	if path == "" {
		p, err := savePath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "garden:", err)
			os.Exit(1)
		}
		path = p
	}

	now := time.Now()
	g, err := Load(path, now)
	if err != nil {
		fmt.Fprintln(os.Stderr, "garden:", err)
		os.Exit(1)
	}
	g.Advance(now)
	bonus := g.Visit(now)

	m := newModel(g, path, now)
	if g.Music {
		m.toggleMusic() // the garden was left with the music on
	}
	if bonus > 0 {
		m.setStatus(seedStyle, "A new day: %d seeds from the shed.", bonus)
	}
	m.dirty = true

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "garden:", err)
		os.Exit(1)
	}
	if err := Save(path, g); err != nil {
		fmt.Fprintln(os.Stderr, "garden: saving:", err)
		os.Exit(1)
	}
}
