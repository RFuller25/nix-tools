package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	tabBoard = iota
	tabWon
)

type screen int

const (
	scBoard screen = iota
	scDetail
	scCreate
	scStake
	scResolve
)

type createStep int

const (
	csTitle createStep = iota
	csOptions
	csSide
	csAmount
)

type stakeStep int

const (
	ssOption stakeStep = iota
	ssAmount
)

// Network results. Nothing here ever fires on a timer: every request is the
// direct result of launching, or of a key the user pressed.
type boardMsg struct {
	resp *boardResp
	err  error
}

type winsMsg struct {
	resp *boardResp
	err  error
}

type detailMsg struct {
	id  int
	d   *Detail
	err error
}

type createMsg struct {
	title  string
	labels []string
	side   int
	amount int
	resp   *actionResp
	err    error
}

type stakeMsg struct {
	id, option, amount int
	resp               *actionResp
	err                error
}

type resolveMsg struct {
	id, option int
	resp       *actionResp
	err        error
}

type model struct {
	cfg        *Config
	client     *APIClient
	setup      setupModel
	needsSetup bool

	screen  screen
	width   int
	height  int
	balance int
	bets    []Bet
	cursor  int

	tab       int // tabBoard or tabWon
	won       []Bet
	wonLoaded bool
	wonCursor int

	busy   bool
	spin   spinner.Model
	flash  string
	failed bool // flash is an error

	detail   *Detail // the open bet; History may lag until the fetch lands
	histWait bool

	input textinput.Model

	cStep   createStep
	cTitle  string
	cLabels []string
	cSide   int

	sStep   stakeStep
	sOption int

	rOption  int
	rConfirm bool
}

func newModel(cfg *Config, needsSetup bool) model {
	in := textinput.New()
	in.CharLimit = 120
	in.Width = 40
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := model{needsSetup: needsSetup, spin: sp, input: in, cfg: cfg, width: 80, height: 24}
	if needsSetup {
		user := ""
		if cfg != nil {
			user = cfg.Username
		}
		m.setup = newSetupModel(user)
		return m
	}
	m.start(cfg)
	return m
}

func (m *model) start(cfg *Config) {
	m.cfg = cfg
	m.client = newAPIClient(cfg.APIKey, cfg.Username)
	if c := loadCache(cfg.Username); c != nil {
		m.balance, m.bets = c.Balance, c.Bets
	}
}

func (m model) Init() tea.Cmd {
	if m.needsSetup {
		return m.setup.Init()
	}
	return m.fetchBoard()
}

// ── commands ───────────────────────────────────────────────────────────────

func (m model) fetchBoard() tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		r, err := c.Board()
		return boardMsg{r, err}
	})
}

func (m model) fetchWins() tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		r, err := c.Wins()
		return winsMsg{r, err}
	})
}

func (m model) fetchDetail(id int) tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		d, err := c.Detail(id)
		return detailMsg{id, d, err}
	})
}

func (m model) doCreate(title string, labels []string, side, amount int) tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		r, err := c.Create(title, labels, side, amount)
		return createMsg{title, labels, side, amount, r, err}
	})
}

func (m model) doStake(id, option, amount int) tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		r, err := c.Stake(id, option, amount)
		return stakeMsg{id, option, amount, r, err}
	})
}

func (m model) doResolve(id, option int) tea.Cmd {
	c := m.client
	return tea.Batch(m.spin.Tick, func() tea.Msg {
		r, err := c.Resolve(id, option)
		return resolveMsg{id, option, r, err}
	})
}

// ── state helpers ──────────────────────────────────────────────────────────

func (m *model) persist() {
	if m.cfg != nil {
		saveCache(boardCache{Username: m.cfg.Username, Balance: m.balance, Bets: m.bets})
	}
}

func (m *model) fail(err error) {
	m.flash, m.failed = err.Error(), true
}

func (m *model) ok(s string) {
	m.flash, m.failed = s, false
}

func (m *model) betIndex(id int) int {
	for i := range m.bets {
		if m.bets[i].ID == id {
			return i
		}
	}
	return -1
}

func (m model) selected() *Bet {
	list, cur := m.bets, m.cursor
	if m.tab == tabWon {
		list, cur = m.won, m.wonCursor
	}
	if cur < 0 || cur >= len(list) {
		return nil
	}
	return &list[cur]
}

func (m model) isMine(b *Bet) bool { return b != nil && m.cfg != nil && b.Creator == m.cfg.Username }

// applyStake folds a confirmed stake into the local copies, so the screens
// stay current without asking the server again.
func (m *model) applyStake(id, option, amount int) {
	touch := func(b *Bet) {
		if option < 0 || option >= len(b.Options) {
			return
		}
		b.Options[option].Total += amount
		b.Pool += amount
		for len(b.Mine) < len(b.Options) {
			b.Mine = append(b.Mine, 0)
		}
		b.Mine[option] += amount
	}
	if i := m.betIndex(id); i >= 0 {
		touch(&m.bets[i])
	}
	if m.detail != nil && m.detail.ID == id {
		touch(&m.detail.Bet)
		totals := make([]int, len(m.detail.Options))
		for i, o := range m.detail.Options {
			totals[i] = o.Total
		}
		m.detail.History = append(m.detail.History, Point{TS: time.Now().Unix(), Totals: totals})
	}
}

func (m *model) applyResolve(id, option, status, paid int) {
	win := -1
	if status == statusResolved {
		win = option
	}
	set := func(b *Bet) { b.Status, b.Winner, b.Paid, b.Resolved = status, win, paid, time.Now().Unix() }
	if i := m.betIndex(id); i >= 0 {
		set(&m.bets[i])
	}
	if m.detail != nil && m.detail.ID == id {
		set(&m.detail.Bet)
	}
}

func parseAmount(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("enter a whole number of BBs")
	}
	return n, nil
}

// ── update ─────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}

	if m.needsSetup {
		if k, ok := msg.(tea.KeyMsg); ok && k.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.setup, cmd = m.setup.Update(msg)
		if m.setup.done {
			m.needsSetup = false
			m.start(m.setup.config)
			m.busy = true
			return m, m.fetchBoard()
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case boardMsg:
		m.busy = false
		if msg.err != nil {
			m.fail(msg.err)
			return m, nil
		}
		m.balance, m.bets = msg.resp.Balance, msg.resp.Bets
		m.cursor = min(m.cursor, max(len(m.bets)-1, 0))
		m.flash = ""
		m.persist()
		return m, nil

	case winsMsg:
		m.busy = false
		if msg.err != nil {
			m.fail(msg.err)
			return m, nil
		}
		m.balance, m.won, m.wonLoaded = msg.resp.Balance, msg.resp.Bets, true
		m.wonCursor = min(m.wonCursor, max(len(m.won)-1, 0))
		m.flash = ""
		return m, nil

	case detailMsg:
		m.busy, m.histWait = false, false
		if msg.err != nil {
			m.fail(msg.err)
			return m, nil
		}
		m.balance = msg.d.Balance
		if i := m.betIndex(msg.id); i >= 0 {
			m.bets[i] = msg.d.Bet.clone()
		}
		if m.detail != nil && m.detail.ID == msg.id {
			m.detail = msg.d
		}
		m.persist()
		return m, nil

	case createMsg:
		m.busy = false
		if msg.err != nil {
			m.fail(msg.err)
			return m, nil
		}
		opts := make([]Option, len(msg.labels))
		mine := make([]int, len(msg.labels))
		for i, l := range msg.labels {
			opts[i] = Option{Label: l}
		}
		opts[msg.side].Total, mine[msg.side] = msg.amount, msg.amount
		b := Bet{ID: msg.resp.ID, Title: msg.title, Creator: m.cfg.Username, Pool: msg.amount,
			Options: opts, Mine: mine, Winner: -1, Created: time.Now().Unix()}
		m.bets = append([]Bet{b}, m.bets...)
		m.balance, m.cursor, m.screen, m.tab = msg.resp.Balance, 0, scBoard, tabBoard
		m.ok("bet created")
		m.persist()
		return m, nil

	case stakeMsg:
		m.busy = false
		if msg.err != nil {
			m.fail(msg.err)
			return m, nil
		}
		m.balance = msg.resp.Balance
		m.applyStake(msg.id, msg.option, msg.amount)
		m.screen = scDetail
		m.ok(fmt.Sprintf("staked %d BBs", msg.amount))
		m.persist()
		return m, nil

	case resolveMsg:
		m.busy = false
		if msg.err != nil {
			m.fail(msg.err)
			m.rConfirm = false
			return m, nil
		}
		m.balance = msg.resp.Balance
		m.applyResolve(msg.id, msg.option, msg.resp.Status, msg.resp.Paid)
		m.wonLoaded = false // a win may have just been added
		m.screen = scDetail
		if msg.resp.Status == statusVoid {
			m.ok("bet voided, stakes refunded")
		} else {
			m.ok("bet resolved, payouts sent")
		}
		m.persist()
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.busy {
			return m, nil
		}
		switch m.screen {
		case scBoard:
			return m.keyBoard(msg)
		case scDetail:
			return m.keyDetail(msg)
		case scCreate:
			return m.keyCreate(msg)
		case scStake:
			return m.keyStake(msg)
		case scResolve:
			return m.keyResolve(msg)
		}
	}
	return m, nil
}

func (m model) keyBoard(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q":
		return m, tea.Quit
	case "tab", "shift+tab", "1", "2":
		next := tabBoard
		switch k.String() {
		case "tab", "shift+tab":
			next = 1 - m.tab
		case "2":
			next = tabWon
		}
		if next == m.tab {
			return m, nil
		}
		m.tab, m.flash = next, ""
		if m.tab == tabWon && !m.wonLoaded {
			m.busy = true
			return m, m.fetchWins()
		}
	case "up", "k":
		if m.tab == tabWon {
			m.wonCursor = max(m.wonCursor-1, 0)
		} else {
			m.cursor = max(m.cursor-1, 0)
		}
	case "down", "j":
		if m.tab == tabWon {
			m.wonCursor = min(m.wonCursor+1, max(len(m.won)-1, 0))
		} else {
			m.cursor = min(m.cursor+1, max(len(m.bets)-1, 0))
		}
	case "r":
		m.busy, m.flash = true, ""
		if m.tab == tabWon {
			return m, m.fetchWins()
		}
		return m, m.fetchBoard()
	case "n":
		m.screen, m.cStep, m.cTitle, m.cLabels, m.cSide, m.flash = scCreate, csTitle, "", nil, 0, ""
		m.input.SetValue("")
		m.input.Placeholder = "What are we betting on?"
		m.input.CharLimit = 120
		return m, m.input.Focus()
	case "enter":
		b := m.selected()
		if b == nil {
			return m, nil
		}
		m.detail = &Detail{Bet: b.clone(), Balance: m.balance}
		m.screen, m.busy, m.histWait, m.flash = scDetail, true, true, ""
		return m, m.fetchDetail(b.ID)
	}
	return m, nil
}

func (m model) keyDetail(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := m.detail
	switch k.String() {
	case "esc", "backspace", "q":
		m.screen, m.flash = scBoard, ""
	case "b":
		if d != nil && d.Status == statusOpen {
			m.screen, m.sStep, m.sOption, m.flash = scStake, ssOption, 0, ""
		}
	case "r":
		if d != nil && d.Status == statusOpen && m.isMine(&d.Bet) {
			m.screen, m.rOption, m.rConfirm, m.flash = scResolve, 0, false, ""
		}
	}
	return m, nil
}

func (m model) keyCreate(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyEsc {
		m.screen, m.flash = scBoard, ""
		return m, nil
	}
	switch m.cStep {
	case csTitle:
		if k.Type == tea.KeyEnter {
			t := strings.TrimSpace(m.input.Value())
			if t == "" {
				return m, nil
			}
			m.cTitle, m.cStep = t, csOptions
			m.input.SetValue("")
			m.input.Placeholder = "Outcome (enter on empty to finish)"
			m.input.CharLimit = 40
			m.flash = ""
			return m, nil
		}
	case csOptions:
		if k.Type == tea.KeyEnter {
			l := strings.TrimSpace(m.input.Value())
			if l == "" {
				if len(m.cLabels) < 2 {
					m.fail(fmt.Errorf("need at least 2 outcomes"))
					return m, nil
				}
				m.cStep, m.cSide, m.flash = csSide, 0, ""
				m.input.Blur()
				return m, nil
			}
			for _, e := range m.cLabels {
				if strings.EqualFold(e, l) {
					m.fail(fmt.Errorf("duplicate outcome"))
					return m, nil
				}
			}
			if len(m.cLabels) >= maxOpts {
				m.fail(fmt.Errorf("at most %d outcomes", maxOpts))
				return m, nil
			}
			m.cLabels = append(m.cLabels, l)
			m.input.SetValue("")
			m.flash = ""
			return m, nil
		}
	case csSide:
		switch k.String() {
		case "up", "k":
			m.cSide = max(m.cSide-1, 0)
		case "down", "j":
			m.cSide = min(m.cSide+1, len(m.cLabels)-1)
		case "enter":
			m.cStep, m.flash = csAmount, ""
			m.input.SetValue(strconv.Itoa(minCreate))
			m.input.CharLimit = 9
			return m, m.input.Focus()
		}
		return m, nil
	case csAmount:
		if k.Type == tea.KeyEnter {
			n, err := parseAmount(m.input.Value())
			if err == nil && n < minCreate {
				err = fmt.Errorf("minimum to start a bet is %d BBs", minCreate)
			}
			if err == nil && n > m.balance {
				err = fmt.Errorf("you only have %d BBs", m.balance)
			}
			if err != nil {
				m.fail(err)
				return m, nil
			}
			m.busy, m.flash = true, ""
			return m, m.doCreate(m.cTitle, m.cLabels, m.cSide, n)
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m model) keyStake(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := m.detail
	if k.Type == tea.KeyEsc {
		m.screen, m.flash = scDetail, ""
		return m, nil
	}
	if m.sStep == ssOption {
		switch k.String() {
		case "up", "k":
			m.sOption = max(m.sOption-1, 0)
		case "down", "j":
			m.sOption = min(m.sOption+1, len(d.Options)-1)
		case "enter":
			m.sStep, m.flash = ssAmount, ""
			m.input.SetValue("")
			m.input.Placeholder = fmt.Sprintf("BBs (you have %d)", m.balance)
			m.input.CharLimit = 9
			return m, m.input.Focus()
		}
		return m, nil
	}
	if k.Type == tea.KeyEnter {
		n, err := parseAmount(m.input.Value())
		if err == nil && n > m.balance {
			err = fmt.Errorf("you only have %d BBs", m.balance)
		}
		if err != nil {
			m.fail(err)
			return m, nil
		}
		m.busy, m.flash = true, ""
		return m, m.doStake(d.ID, m.sOption, n)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(k)
	return m, cmd
}

func (m model) keyResolve(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	d := m.detail
	last := len(d.Options) // the extra row is VOID
	if m.rConfirm {
		switch k.String() {
		case "y", "Y":
			opt := m.rOption
			if opt == last {
				opt = -1
			}
			m.busy, m.flash = true, ""
			return m, m.doResolve(d.ID, opt)
		default:
			m.rConfirm = false
		}
		return m, nil
	}
	switch k.String() {
	case "esc":
		m.screen, m.flash = scDetail, ""
	case "up", "k":
		m.rOption = max(m.rOption-1, 0)
	case "down", "j":
		m.rOption = min(m.rOption+1, last)
	case "enter":
		m.rConfirm, m.flash = true, ""
	}
	return m, nil
}
