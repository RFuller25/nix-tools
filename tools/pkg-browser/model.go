package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusedPanel int

const (
	panelProfile focusedPanel = iota
	panelConfig
)

type model struct {
	focus  focusedPanel
	width  int
	height int
	ready  bool

	profilePkgs   []pkg
	profileCursor int
	profileErr    error

	configPkgs   []configPkg
	configCursor int
	configPath   string
	configErr    error

	searchOpen   bool
	searchInput  textinput.Model
	searchResult []pkg
	searchCursor int
	searchErr    error

	spinner   spinner.Model
	loading   bool
	statusMsg string
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	activeTabS   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Underline(true)
	inactiveTabS = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedRow  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	normalRow    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	versionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	descStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	divStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func initialModel(configPath string) model {
	ti := textinput.New()
	ti.Placeholder = "search nixpkgs..."

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		focus:       panelProfile,
		configPath:  configPath,
		searchInput: ti,
		spinner:     sp,
		loading:     true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadInstalled(), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.searchInput.Width = msg.Width - 4
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		m.statusMsg = ""

		if m.searchOpen {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.searchOpen = false
				m.focus = panelConfig
				m.searchResult = nil
				m.searchCursor = 0
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				return m, nil
			case "enter":
				q := strings.TrimSpace(m.searchInput.Value())
				if q != "" {
					m.loading = true
					m.searchResult = nil
					m.searchCursor = 0
					return m, tea.Batch(searchNixpkgs(q), m.spinner.Tick)
				}
				return m, nil
			case "up", "k":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
				return m, nil
			case "down", "j":
				if m.searchCursor < len(m.searchResult)-1 {
					m.searchCursor++
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == panelProfile {
				m.focus = panelConfig
			} else {
				m.focus = panelProfile
			}
			return m, nil
		case "/":
			m.searchOpen = true
			m.searchInput.Focus()
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		}

	case installedLoadedMsg:
		m.profilePkgs = msg.pkgs
		m.loading = false
		return m, nil

	case searchResultMsg:
		m.searchResult = msg.pkgs
		m.loading = false
		return m, nil

	case nixErrMsg:
		if m.searchOpen {
			m.searchErr = msg.err
		} else {
			m.profileErr = msg.err
		}
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

func (m *model) moveCursor(delta int) {
	switch m.focus {
	case panelProfile:
		m.profileCursor = clamp(m.profileCursor+delta, 0, len(m.profilePkgs)-1)
	case panelConfig:
		m.configCursor = clamp(m.configCursor+delta, 0, len(m.configPkgs)-1)
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}
	var b strings.Builder
	b.WriteString(renderHeader(m))
	b.WriteString(renderPanels(m))
	b.WriteString(divStyle.Render(strings.Repeat("─", m.width)) + "\n")
	if m.searchOpen {
		b.WriteString(m.searchInput.View() + "\n")
		b.WriteString(divStyle.Render(strings.Repeat("─", m.width)) + "\n")
		b.WriteString(renderSearchPanel(m))
	}
	b.WriteString(renderHelp(m))
	return b.String()
}

func renderHeader(m model) string {
	profileLabel := "Profile"
	configLabel := "Config"
	if m.focus == panelProfile {
		profileLabel = activeTabS.Render(profileLabel)
		configLabel = inactiveTabS.Render(configLabel)
	} else {
		profileLabel = inactiveTabS.Render(profileLabel)
		configLabel = activeTabS.Render(configLabel)
	}
	return titleStyle.Render("pkg-browser") + "  " + profileLabel + "  " + configLabel + "\n" +
		divStyle.Render(strings.Repeat("─", m.width)) + "\n"
}

func renderHelp(m model) string {
	if m.statusMsg != "" {
		return helpStyle.Render(m.statusMsg)
	}
	if m.searchOpen {
		return helpStyle.Render("↑/↓: navigate results  a: add to config  esc: close  ctrl+c: quit")
	}
	return helpStyle.Render(fmt.Sprintf(
		"tab: switch panel  ↑/↓ j/k: navigate  /: search  d: remove  q: quit",
	))
}
