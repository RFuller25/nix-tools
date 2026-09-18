package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenGarden screen = iota
	screenShop
	screenInfo
	screenAlmanac
	screenJournal
	screenHelp
)

type tickMsg time.Time
type windTickMsg time.Time
type saveMsg struct{ err error }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// blow drives the wind animation, which runs far faster than the garden's
// once-a-second simulation tick.
func blow() tea.Cmd {
	return tea.Tick(windTick, func(t time.Time) tea.Msg { return windTickMsg(t) })
}

type model struct {
	g    *Garden
	path string
	now  time.Time

	screen   screen
	cursor   int // selected bed
	scroll   int // first visible grid row
	scrollX  int // first visible grid column
	saveErr  error
	lastSave time.Time
	dirty    bool

	shop       []*Species
	shopCursor int
	shopScroll int
	shopSeason bool // limit the shop to species happy in this season

	almanac       []*Species
	almanacCursor int
	almanacScroll int
	almanacStage  int

	journalScroll int

	naming bool
	input  textinput.Model

	audio *Audio
	wind  windState

	status      string
	statusStyle lipgloss.Style

	width  int
	height int
}

func newModel(g *Garden, path string, now time.Time) model {
	ti := textinput.New()
	ti.Placeholder = "a name for this plant"
	ti.CharLimit = 24
	ti.Prompt = "  name › "

	m := model{
		g:       g,
		path:    path,
		now:     now,
		input:   ti,
		almanac: AllSpecies(),
		audio:   NewAudio(sampleRate),
		wind:    newWind(g.Seed ^ now.UnixNano()),
		width:   80,
		height:  30,
	}
	m.refreshShop()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), blow(), tea.ClearScreen, textinput.Blink)
}

// refreshShop rebuilds the seed-shop listing under the current filters.
func (m *model) refreshShop() {
	season := m.g.Season(m.now)
	m.shop = m.shop[:0]
	for _, sp := range AllSpecies() {
		if m.shopSeason && !sp.LikesSeason(season) {
			continue
		}
		m.shop = append(m.shop, sp)
	}
	if m.shopCursor >= len(m.shop) {
		m.shopCursor = max(0, len(m.shop)-1)
	}
}

func (m *model) setStatus(style lipgloss.Style, format string, args ...any) {
	m.status = fmt.Sprintf(format, args...)
	m.statusStyle = style
}

func (m *model) plot() *Plot { return &m.g.Plots[m.cursor] }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		m.g.Advance(m.now)
		if m.dirty && m.now.Sub(m.lastSave) > 10*time.Second {
			return m, tea.Batch(tick(), m.save())
		}
		return m, tick()

	case windTickMsg:
		m.wind.advance(windTick.Seconds(), m.g.Weather(m.now), m.gridCols())
		return m, blow()

	case saveMsg:
		m.saveErr = msg.err
		if msg.err == nil {
			m.dirty = false
			m.lastSave = m.now
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) save() tea.Cmd {
	g := m.g
	path := m.path
	return func() tea.Msg { return saveMsg{Save(path, g)} }
}

func (m model) quit() (tea.Model, tea.Cmd) {
	m.audio.Close()
	if err := Save(m.path, m.g); err != nil {
		m.saveErr = err
	}
	return m, tea.Quit
}

// toggleMusic starts or stops the garden's ambient piece. The choice is kept
// in the save file, so a garden you left humming is humming when you return.
func (m *model) toggleMusic() {
	if m.audio.Playing() {
		m.audio.StopMusic()
		m.g.Music = false
		m.dirty = true
		m.setStatus(subtleStyle, "Music off.")
		return
	}
	if err := m.audio.StartMusic(calmMusic(sampleRate, m.g.Seed)); err != nil {
		m.setStatus(warnStyle, "%s", playerHint())
		m.g.Music = false
		return
	}
	m.g.Music = true
	m.dirty = true
	m.setStatus(okStyle, "Something quiet, in D, through %s.", m.audio.Backend())
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.naming {
		return m.handleNaming(msg)
	}

	key := msg.String()

	// Keys that work from anywhere.
	switch key {
	case "ctrl+c":
		return m.quit()
	case "m":
		m.toggleMusic()
		return m, nil
	case "q":
		if m.screen == screenGarden {
			return m.quit()
		}
		m.screen = screenGarden
		return m, nil
	case "?":
		if m.screen == screenHelp {
			m.screen = screenGarden
		} else {
			m.screen = screenHelp
		}
		return m, nil
	case "tab":
		m.screen = nextScreen(m.screen)
		if m.screen == screenShop {
			m.refreshShop()
		}
		return m, nil
	case "esc":
		if m.screen != screenGarden {
			m.screen = screenGarden
			return m, nil
		}
	}

	switch m.screen {
	case screenGarden:
		return m.handleGardenKey(key)
	case screenShop:
		return m.handleShopKey(key)
	case screenInfo:
		return m.handleInfoKey(key)
	case screenAlmanac:
		return m.handleAlmanacKey(key)
	case screenJournal:
		return m.handleJournalKey(key)
	case screenHelp:
		m.screen = screenGarden
	}
	return m, nil
}

func nextScreen(s screen) screen {
	switch s {
	case screenGarden:
		return screenShop
	case screenShop:
		return screenAlmanac
	case screenAlmanac:
		return screenJournal
	default:
		return screenGarden
	}
}

func (m model) handleGardenKey(key string) (tea.Model, tea.Cmd) {
	cols := plotCols
	switch key {
	case "left", "h":
		if m.cursor%cols > 0 {
			m.cursor--
		}
	case "right", "l":
		if m.cursor%cols < cols-1 && m.cursor+1 < len(m.g.Plots) {
			m.cursor++
		}
	case "up", "k":
		if m.cursor-cols >= 0 {
			m.cursor -= cols
		}
	case "down", "j":
		if m.cursor+cols < len(m.g.Plots) {
			m.cursor += cols
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.g.Plots) - 1
	case "p", "enter":
		if m.plot().Empty() {
			m.screen = screenShop
			m.refreshShop()
		} else {
			m.screen = screenInfo
		}
	case "i", " ":
		if !m.plot().Empty() {
			m.screen = screenInfo
		}
	case "w":
		if m.g.Water(m.cursor, m.now) {
			m.dirty = true
			m.setStatus(waterStyle, "Watered %s.", m.plot().DisplayName())
		} else if m.plot().Empty() {
			m.setStatus(subtleStyle, "Nothing planted in bed %d.", m.cursor+1)
		} else {
			m.setStatus(subtleStyle, "%s has plenty to drink.", m.plot().DisplayName())
		}
	case "W":
		if n := m.g.WaterAll(m.now); n > 0 {
			m.dirty = true
			m.setStatus(waterStyle, "Watered %d bed(s).", n)
		} else {
			m.setStatus(subtleStyle, "Every bed is already watered.")
		}
	case "c":
		if ok, reward := m.g.Weed(m.cursor, m.now); ok {
			m.dirty = true
			if reward > 0 {
				m.setStatus(okStyle, "Cleared the weeds — found %d seed in the tangle.", reward)
			} else {
				m.setStatus(okStyle, "Tidied bed %d.", m.cursor+1)
			}
		} else {
			m.setStatus(subtleStyle, "Bed %d is already clear.", m.cursor+1)
		}
	case "C":
		total, reward := 0, 0
		for i := range m.g.Plots {
			if ok, r := m.g.Weed(i, m.now); ok {
				total++
				reward += r
			}
		}
		if total > 0 {
			m.dirty = true
			m.g.Log(m.now, "Weeded the whole garden (%d beds).", total)
			m.setStatus(okStyle, "Weeded %d bed(s), earning %d seed(s).", total, reward)
		} else {
			m.setStatus(subtleStyle, "Not a weed in sight.")
		}
	case "f":
		if got := m.g.Gather(m.cursor, m.now); got > 0 {
			m.dirty = true
			m.setStatus(seedStyle, "Gathered %d seed(s) from %s.", got, m.plot().DisplayName())
		} else {
			m.setStatus(subtleStyle, "No ripe seed here yet.")
		}
	case "F":
		got := 0
		for i := range m.g.Plots {
			got += m.g.Gather(i, m.now)
		}
		if got > 0 {
			m.dirty = true
			m.setStatus(seedStyle, "Gathered %d seed(s) from the garden.", got)
		} else {
			m.setStatus(subtleStyle, "Nothing ripe to gather.")
		}
	case "n", "r":
		if !m.plot().Empty() {
			m.startNaming()
		} else {
			m.setStatus(subtleStyle, "Plant something first.")
		}
	case "u":
		if m.g.Uproot(m.cursor, m.now) {
			m.dirty = true
			m.setStatus(warnStyle, "Lifted the plant in bed %d.", m.cursor+1)
		} else {
			m.setStatus(subtleStyle, "Bed %d is already empty.", m.cursor+1)
		}
	case "b":
		if err := m.g.BuyBed(m.now); err != nil {
			m.setStatus(warnStyle, "%s", err.Error())
		} else {
			m.dirty = true
			m.cursor = len(m.g.Plots) - 1
			m.setStatus(okStyle, "New ground broken: bed %d. The next costs %d seeds.", len(m.g.Plots), m.g.BedCost())
		}
	case "d":
		wasPond := m.plot().Pond
		if err := m.g.DigPond(m.cursor, m.now); err != nil {
			m.setStatus(warnStyle, "%s", err.Error())
		} else {
			m.dirty = true
			if wasPond {
				m.setStatus(okStyle, "Filled the pond back in.")
			} else {
				m.setStatus(waterStyle, "Dug a pond. Water lilies and lotus will grow here.")
			}
		}
	case "a":
		m.screen = screenAlmanac
	case "s":
		m.screen = screenShop
		m.refreshShop()
	}
	m.ensureVisible()
	return m, nil
}

func (m *model) startNaming() {
	m.naming = true
	m.input.SetValue(m.plot().Name)
	m.input.CursorEnd()
	m.input.Focus()
}

func (m model) handleNaming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.naming = false
		m.input.Blur()
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		m.g.Rename(m.cursor, name, m.now)
		m.dirty = true
		m.naming = false
		m.input.Blur()
		if name == "" {
			m.setStatus(subtleStyle, "Name cleared.")
		} else {
			m.setStatus(okStyle, "Say hello to %s.", name)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) handleShopKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.shopCursor > 0 {
			m.shopCursor--
		}
	case "down", "j":
		if m.shopCursor < len(m.shop)-1 {
			m.shopCursor++
		}
	case "pgup":
		m.shopCursor = max(0, m.shopCursor-10)
	case "pgdown":
		m.shopCursor = min(len(m.shop)-1, m.shopCursor+10)
	case "home", "g":
		m.shopCursor = 0
	case "end", "G":
		m.shopCursor = max(0, len(m.shop)-1)
	case "t":
		m.shopSeason = !m.shopSeason
		m.refreshShop()
		if m.shopSeason {
			m.setStatus(subtleStyle, "Showing seeds for %s only.", m.g.Season(m.now))
		} else {
			m.setStatus(subtleStyle, "Showing the whole seed rack.")
		}
	case "enter", "p", " ":
		if len(m.shop) == 0 {
			return m, nil
		}
		sp := m.shop[m.shopCursor]
		idx := m.firstEmptyFrom(m.cursor)
		if idx < 0 {
			m.setStatus(warnStyle, "Every bed is full — lift something first (u).")
			return m, nil
		}
		if err := m.g.Plant(idx, sp, m.now); err != nil {
			m.setStatus(errStyle, "%s", err.Error())
			return m, nil
		}
		m.cursor = idx
		m.dirty = true
		m.screen = screenGarden
		m.setStatus(okStyle, "Sowed %s in bed %d.", sp.Common, idx+1)
		return m, m.save()
	}
	return m, nil
}

// firstEmptyFrom prefers the selected bed, then scans forward for a free one.
func (m model) firstEmptyFrom(start int) int {
	if m.g.Plots[start].Empty() {
		return start
	}
	for i := 0; i < len(m.g.Plots); i++ {
		if n := (start + i) % len(m.g.Plots); m.g.Plots[n].Empty() {
			return n
		}
	}
	return -1
}

func (m model) handleInfoKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "w":
		if m.g.Water(m.cursor, m.now) {
			m.dirty = true
			m.setStatus(waterStyle, "Watered %s.", m.plot().DisplayName())
		}
	case "c":
		if ok, _ := m.g.Weed(m.cursor, m.now); ok {
			m.dirty = true
			m.setStatus(okStyle, "Tidied the bed.")
		}
	case "f":
		if got := m.g.Gather(m.cursor, m.now); got > 0 {
			m.dirty = true
			m.setStatus(seedStyle, "Gathered %d seed(s).", got)
		}
	case "n", "r":
		m.startNaming()
	case "left", "h", "up", "k":
		m.cursor = (m.cursor + len(m.g.Plots) - 1) % len(m.g.Plots)
	case "right", "l", "down", "j":
		m.cursor = (m.cursor + 1) % len(m.g.Plots)
	case "u":
		if m.g.Uproot(m.cursor, m.now) {
			m.dirty = true
			m.screen = screenGarden
			m.setStatus(warnStyle, "Lifted the plant in bed %d.", m.cursor+1)
		}
	}
	return m, nil
}

func (m model) handleAlmanacKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.almanacCursor > 0 {
			m.almanacCursor--
		}
	case "down", "j":
		if m.almanacCursor < len(m.almanac)-1 {
			m.almanacCursor++
		}
	case "pgup":
		m.almanacCursor = max(0, m.almanacCursor-10)
	case "pgdown":
		m.almanacCursor = min(len(m.almanac)-1, m.almanacCursor+10)
	case "home", "g":
		m.almanacCursor = 0
	case "end", "G":
		m.almanacCursor = len(m.almanac) - 1
	case "left", "h":
		m.almanacStage = max(0, m.almanacStage-1)
	case "right", "l":
		m.almanacStage = min(StageCount-1, m.almanacStage+1)
	case " ":
		m.almanacStage = (m.almanacStage + 1) % StageCount
	}
	return m, nil
}

func (m model) handleJournalKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.journalScroll = max(0, m.journalScroll-1)
	case "down", "j":
		m.journalScroll++
	case "pgup":
		m.journalScroll = max(0, m.journalScroll-10)
	case "pgdown":
		m.journalScroll += 10
	case "home", "g":
		m.journalScroll = 0
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenShop:
		return m.viewShop()
	case screenInfo:
		return m.viewInfo()
	case screenAlmanac:
		return m.viewAlmanac()
	case screenJournal:
		return m.viewJournal()
	case screenHelp:
		return m.viewHelp()
	default:
		return m.viewGarden()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
