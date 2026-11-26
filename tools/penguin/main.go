// penguin loops the walk into the mountains: a lone emperor penguin turns away
// from the colony and waddles off toward the peaks, drawn in coloured ASCII.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const version = "0.3.0"

func main() {
	var (
		showVersion = flag.Bool("version", false, "print version and exit")
		frame       = flag.Float64("frame", -1, "print one frame at this many seconds into the loop and exit")
		width       = flag.Int("w", 120, "width for -frame")
		height      = flag.Int("h", 40, "height for -frame")
		sceneName   = flag.String("scene", "penguin", "which scene to play: "+strings.Join(filmNames(), ", "))
		list        = flag.Bool("list", false, "list the scenes and exit")
		plain       = flag.Bool("plain", false, "with -frame: print glyphs without colour")
	)
	flag.Parse()

	if *list {
		for _, f := range filmList {
			fmt.Printf("%-11s %s\n", f.name, f.blurb)
		}
		return
	}
	fi := filmIndex(*sceneName)
	if fi < 0 {
		fmt.Fprintf(os.Stderr, "penguin: no scene called %q (try: %s)\n", *sceneName, strings.Join(filmNames(), ", "))
		os.Exit(1)
	}
	if *showVersion {
		fmt.Printf("penguin %s\n", version)
		return
	}
	if *frame >= 0 {
		p := termenv.TrueColor
		if *plain {
			p = termenv.Ascii
		}
		f := filmList[fi].make(*width, *height)
		if sc, ok := f.(*scene); ok && os.Getenv("PENGUIN_DEBUG") != "" {
			w := sc.walker(*frame)
			fmt.Fprintf(os.Stderr, "walker x=%.1f d=%.2f rows=%.2f hz=%d yaw=%.0f\n", w.x, w.d, w.rows(), sc.hz, w.yaw*180/3.14159)
		}
		fmt.Println(f.draw(*frame).render(p))
		return
	}

	m := newModel(lipgloss.ColorProfile(), fi)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "penguin:", err)
		os.Exit(1)
	}
}
