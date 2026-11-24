// penguin loops the walk into the mountains: a lone emperor penguin turns away
// from the colony and waddles off toward the peaks, drawn in coloured ASCII.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const version = "0.1.0"

func main() {
	var (
		showVersion = flag.Bool("version", false, "print version and exit")
		frame       = flag.Float64("frame", -1, "print one frame at this many seconds into the loop and exit")
		width       = flag.Int("w", 120, "width for -frame")
		height      = flag.Int("h", 40, "height for -frame")
		plain       = flag.Bool("plain", false, "with -frame: print glyphs without colour")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("penguin %s\n", version)
		return
	}
	if *frame >= 0 {
		p := termenv.TrueColor
		if *plain {
			p = termenv.Ascii
		}
		sc := newScene(*width, *height)
		if os.Getenv("PENGUIN_DEBUG") != "" {
			w := sc.walker(*frame)
			fmt.Fprintf(os.Stderr, "walker x=%.1f d=%.2f rows=%.2f hz=%d yaw=%.0f\n", w.x, w.d, w.rows(), sc.hz, w.yaw*180/3.14159)
		}
		fmt.Println(sc.draw(*frame).render(p))
		return
	}

	m := newModel(lipgloss.ColorProfile())
	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "penguin:", err)
		os.Exit(1)
	}
}
