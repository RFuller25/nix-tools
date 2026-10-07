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
	screenTemplates
	screenOrders
	screenFair
	screenShare
)

// mode is a state layered over the garden screen that changes what the keys mean.
type mode int

const (
	modeNone   mode = iota
	modePlan        // placing ghosts of seed to see how a layout would do
	modeStamp       // placing a saved layout
	modeSelect      // choosing a block of beds to save as a layout
)

// nameTarget is what the name being typed is for.
type nameTarget int

const (
	namePlant nameTarget = iota
	namePacket
	nameCultivar
	nameTemplate
	namePlanFilter
	nameImport
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

	shelf       shelf // which half of the shed is open
	mineCursor  int   // the highlighted packet, in ShedOrder
	mineScroll  int
	shop        []*Species
	shopCursor  int
	shopScroll  int
	shopVariety int  // the form selected for the highlighted species
	shopSeason  bool // limit the shop to species happy in this season

	almanac          []almanacRow
	almanacCultivars int // how many cultivars the rows were built with
	almanacCursor    int
	almanacScroll    int
	almanacStage     int
	almanacVariety   int

	journalScroll int
	cardScroll    int // scrolling inside the info card and the help screen

	overlay      bool // colour the beds by how well they get on with their neighbours
	mode         mode // plan, stamp or select, layered over the garden
	ghosts       map[int]ghost
	plan         *Garden // the garden as it would be if the ghosts were sown
	planPick     int     // the highlighted seed in the plan palette
	planFilter   string  // what the plan palette is narrowed to
	selAnchor    int     // where a layout selection began
	stampTpl     Template
	stampSkipped []string
	tplCursor    int
	ordersCursor int
	fairCursor   int
	shareCode    string // the code on the share screen
	shareName    string
	pollinating  bool // choosing a donor for pollenTarget
	pollenTarget int
	naming       bool
	namingFor    nameTarget
	input        textinput.Model

	audio   *Audio
	wind    windState
	life    wildlife
	compost map[int]compostFX // beds with a plant on its way into the soil

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
		g:                g,
		path:             path,
		now:              now,
		input:            ti,
		almanac:          almanacRows(g),
		almanacCultivars: len(g.Cultivars) + len(g.Fair.Ribbons),
		audio:            NewAudio(sampleRate),
		wind:             newWind(g.Seed ^ now.UnixNano()),
		life:             newWildlife(g.Seed ^ now.UnixNano() ^ 0x1F0C),
		compost:          map[int]compostFX{},
		width:            80,
		height:           30,
	}
	m.refreshShop()
	m.almanacCursor = 1 // the first chapter of the guide, not its heading
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

// phase is where the real clock has got to in the day.
func (m model) phase() phase { return phaseAt(m.now) }

// lift takes a plant out of its bed and starts it collapsing into the soil.
func (m *model) lift(idx int) (float64, bool) {
	p := &m.g.Plots[idx]
	sp := p.Species()
	if sp == nil {
		return 0, false
	}
	stage, _ := appearance(sp, p, m.g.Season(m.now), m.phase())

	gain, ok := m.g.Uproot(idx, m.now)
	if !ok {
		return 0, false
	}
	m.dirty = true
	if m.compost != nil {
		m.compost[idx] = compostFX{Species: sp, Stage: stage, Gain: gain, Started: m.now}
	}
	return gain, true
}

// visitorLine names whatever has come to call, for the status line.
func (m model) visitorLine() string {
	c, ok := m.life.present()
	if !ok {
		return ""
	}
	return upperFirst(c.kind().name) + " in the garden."
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

// nightScent is the line shown when something is perfuming the dark.
func (m model) nightScent() string {
	ph := m.phase()
	if ph != phaseDusk && !ph.Dark() {
		return ""
	}
	for i := range m.g.Plots {
		p := &m.g.Plots[i]
		if p.Empty() || p.Growth < 1 || p.Spent {
			continue
		}
		if sp := p.Species(); sp != nil && sp.NightScented() {
			return sp.Common + " is scenting the dark."
		}
	}
	return ""
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.fixAlmanac()
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		m.g.Advance(m.now)
		if m.almanacKey() != m.almanacCultivars {
			m.refreshAlmanac()
		}
		if m.dirty && m.now.Sub(m.lastSave) > 10*time.Second {
			return m, tea.Batch(tick(), m.save())
		}
		return m, tick()

	case windTickMsg:
		m.wind.advance(windTick.Seconds(), m.g.Weather(m.now), m.gridCols())
		for bed, fx := range m.compost {
			if fx.done(m.now) {
				delete(m.compost, bed)
			}
		}
		m.life.advance(windTick.Seconds(), m.g, m.now, m.g.Weather(m.now), func(c creature) {
			k := c.kind()
			if m.g.sight(k.name, k.note, m.now) {
				m.dirty = true
			}
		})
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

func nextScreen(s screen) screen {
	switch s {
	case screenGarden:
		return screenShop
	case screenShop:
		return screenOrders
	case screenOrders:
		return screenFair
	case screenFair:
		return screenAlmanac
	case screenAlmanac:
		return screenJournal
	default:
		return screenGarden
	}
}

// refreshAlmanac rebuilds the almanac's rows after the garden has found a
// new cultivar, keeping the cursor on the same entry where it can.
func (m *model) refreshAlmanac() {
	keep, _ := m.almanacCur()
	m.almanac = almanacRows(m.g)
	m.almanacCultivars = m.almanacKey()
	for i, r := range m.almanacVisible() {
		if r.Kind == keep.Kind && r.Chapter == keep.Chapter && r.Species == keep.Species && r.Creature == keep.Creature &&
			r.Cultivar == keep.Cultivar && r.Ribbon == keep.Ribbon && r.Group == keep.Group {
			m.almanacCursor = i
			m.fixAlmanac()
			return
		}
	}
	m.fixAlmanac()
}

// startNamingCultivar renames the cultivar highlighted in the almanac.
func (m *model) startNamingCultivar() {
	row, _ := m.almanacCur()
	c := m.g.CultivarByID(row.Cultivar)
	if row.Kind != rowCultivar || c == nil {
		return
	}
	m.naming, m.namingFor = true, nameCultivar
	m.input.Placeholder = "a name for this line"
	m.input.SetValue(c.Name)
	m.input.CursorEnd()
	m.input.Focus()
}

func (m *model) startNaming() {
	m.naming, m.namingFor = true, namePlant
	m.input.Placeholder = "a name for this plant"
	m.input.SetValue(m.plot().Name)
	m.input.CursorEnd()
	m.input.Focus()
}

// startNamingPacket names the highlighted packet in the shed.
func (m *model) startNamingPacket() {
	order := m.g.ShedOrder()
	if len(order) == 0 {
		return
	}
	pk := m.g.Shed[order[min(m.mineCursor, len(order)-1)]]
	m.naming, m.namingFor = true, namePacket
	m.input.Placeholder = "a name for this seed"
	m.input.SetValue(pk.Label)
	m.input.CursorEnd()
	m.input.Focus()
}

func (m model) handleNaming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.naming = false
		m.input.Blur()
		m.input.CharLimit = 24
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		m.naming = false
		m.input.Blur()
		m.dirty = true
		if m.namingFor == nameImport {
			m.importCode(name)
			return m, nil
		}
		if m.namingFor == namePlanFilter {
			m.planFilter, m.planPick = name, 0
			if n := len(m.palette()); n == 0 {
				m.setStatus(warnStyle, "No seed matches “%s”. / to search again, or clear it to see everything.", name)
			} else if name == "" {
				m.setStatus(subtleStyle, "Showing every seed.")
			} else {
				m.setStatus(okStyle, "%d seed(s) match “%s”.", n, name)
			}
			return m, nil
		}
		if m.namingFor == nameTemplate {
			m.finishSaveTemplate(name)
			return m, nil
		}
		if m.namingFor == nameCultivar {
			if row, _ := m.almanacCur(); row.Kind == rowCultivar && m.g.RenameCultivar(row.Cultivar, name, m.now) {
				m.setStatus(okStyle, "The line is now ‘%s’.", name)
				m.refreshAlmanac()
			} else {
				m.setStatus(subtleStyle, "A line needs a name.")
			}
			return m, nil
		}
		if m.namingFor == namePacket {
			if order := m.g.ShedOrder(); len(order) > 0 {
				m.g.RenamePacket(order[min(m.mineCursor, len(order)-1)], name)
			}
			if name == "" {
				m.setStatus(subtleStyle, "Label cleared.")
			} else {
				m.setStatus(okStyle, "Labelled the packet %s.", name)
			}
			return m, nil
		}
		m.g.Rename(m.cursor, name, m.now)
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

func (m model) View() string {
	var view string
	switch m.screen {
	case screenShop:
		view = m.viewShed()
	case screenInfo:
		view = m.viewInfo()
	case screenAlmanac:
		view = m.viewAlmanac()
	case screenJournal:
		view = m.viewJournal()
	case screenHelp:
		view = m.viewHelp()
	case screenTemplates:
		view = m.viewTemplates()
	case screenOrders:
		view = m.viewOrders()
	case screenFair:
		view = m.viewFair()
	case screenShare:
		view = m.viewShare()
	default:
		view = m.viewGarden()
	}
	return fitScreen(view, m.width, m.height)
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

// almanacKey changes whenever the almanac gains a row that is not fixed.
func (m model) almanacKey() int { return len(m.g.Cultivars) + len(m.g.Fair.Ribbons) }
