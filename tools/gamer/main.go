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
		scoreFile   = flag.String("scores", "", "path to the score file (default: $XDG_DATA_HOME/gamer/scores.json)")
		showVersion = flag.Bool("version", false, "print version and exit")
		startWith   = flag.String("play", "", "jump straight into a game: tetris, 2048, snake, hue, mines")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("gamer %s\n", version)
		return
	}

	path := *scoreFile
	if path == "" {
		p, err := scorePath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "gamer:", err)
			os.Exit(1)
		}
		path = p
	}

	scores, err := LoadScores(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gamer:", err)
		os.Exit(1)
	}

	m := newModel(scores, path, time.Now())
	if *startWith != "" {
		if !m.startByID(*startWith) {
			fmt.Fprintf(os.Stderr, "gamer: no game called %q\n", *startWith)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gamer:", err)
		os.Exit(1)
	}
	if err := Save(path, scores); err != nil {
		fmt.Fprintln(os.Stderr, "gamer: saving scores:", err)
		os.Exit(1)
	}
}
