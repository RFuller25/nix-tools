package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.3.3"

func main() {
	var (
		savePathFlag = flag.String("save", "", "path to the garden save file (default: $XDG_DATA_HOME/garden/garden.json)")
		showVersion  = flag.Bool("version", false, "print version and exit")
		listSpecies  = flag.Bool("species", false, "list every species in the almanac and exit")
		postcard     = flag.Bool("postcard", false, "print the garden as it stands and exit")
		status       = flag.Bool("status", false, "print a one-line summary and exit, for a prompt or status bar")
		width        = flag.Int("width", 80, "how wide to draw the postcard")
		colorMode    = flag.String("color", "", "colour mode: truecolor, 256, 16 or off (default: detect; also GARDEN_COLOR)")
		keysMD       = flag.Bool("keys", false, "print the key table as markdown and exit")
		cultivars    = flag.Bool("cultivars", false, "list the hybrid lines your garden has found and exit")
		exportName   = flag.String("export", "", "make a code for one of your cultivars (by name or number) to give a friend, and exit")
		importCode   = flag.String("import", "", "redeem a friend's code: three seeds of their cultivar, and it goes in your almanac, then exit")
		gardener     = flag.String("gardener", "", "the name codes you share are signed with (saved)")
	)
	flag.Parse()

	if *keysMD {
		fmt.Print(keysMarkdown())
		return
	}
	if _, err := applyColourMode(*colorMode); err != nil {
		fmt.Fprintln(os.Stderr, "garden:", err)
		os.Exit(2)
	}

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

	if *gardener != "" {
		g.Gardener = cleanText(*gardener, shareNameMax)
		fmt.Printf("Your codes will be signed %q.\n", g.Gardener)
		if *exportName == "" && *importCode == "" {
			saveOrWarn(path, g)
			return
		}
	}
	if *exportName != "" {
		c, err := findCultivar(g, *exportName)
		if err == nil {
			var code string
			if code, _, err = g.ExportCultivar(c.ID); err == nil {
				fmt.Println(code)
				fmt.Fprintf(os.Stderr, "That is a code for ‘%s’. A friend redeems it with: garden --import <code>, or i in the seed shed.\n", c.Name)
				saveOrWarn(path, g)
				return
			}
		}
		fmt.Fprintln(os.Stderr, "garden:", err)
		os.Exit(1)
	}
	if *importCode != "" {
		r, err := g.Redeem(*importCode, now)
		if err != nil {
			fmt.Fprintln(os.Stderr, "garden:", err)
			os.Exit(1)
		}
		fmt.Printf("Received %d seeds of ‘%s’%s, a %s. It is in your almanac under your cultivars.\n",
			giftSeeds, r.Cultivar.Name, fromText(r.Cultivar.From), r.Cultivar.SpeciesRef().Common)
		saveOrWarn(path, g)
		return
	}
	if *cultivars {
		fmt.Print(renderCultivars(g))
		return
	}

	if *postcard {
		fmt.Println(renderPostcard(g, now, *width))
		if err := Save(path, g); err != nil {
			fmt.Fprintln(os.Stderr, "garden: saving:", err)
		}
		return
	}
	if *status {
		fmt.Println(renderStatus(g, now))
		if err := Save(path, g); err != nil {
			fmt.Fprintln(os.Stderr, "garden: saving:", err)
		}
		return
	}

	m := newModel(g, path, now)
	if g.Music {
		m.toggleMusic() // the garden was left with the music on
	}
	if bonus > 0 {
		m.setStatus(goldStyle, "A new day: %s from the shed.", goldLabel(bonus))
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

// saveOrWarn writes the garden for the commands that change it and exit.
func saveOrWarn(path string, g *Garden) {
	if err := Save(path, g); err != nil {
		fmt.Fprintln(os.Stderr, "garden: saving:", err)
		os.Exit(1)
	}
}
