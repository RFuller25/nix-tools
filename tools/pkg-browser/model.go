package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type activeTab int

const (
	tabInstalled activeTab = iota
	tabSearch
)

type model struct {
	tab          activeTab
	installed    []pkg
	searchResult []pkg
	searchInput  textinput.Model
	spinner      spinner.Model
	loading      bool
	err          error
	cursor       int
	width        int
	height       int
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
)

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "search nixpkgs..."

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return model{
		tab:         tabInstalled,
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
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !(m.tab == tabSearch && m.searchInput.Focused()) {
				return m, tea.Quit
			}
		case "tab":
			if m.tab == tabInstalled {
				m.tab = tabSearch
				m.cursor = 0
				m.searchInput.Focus()
			} else {
				m.tab = tabInstalled
				m.cursor = 0
				m.searchInput.Blur()
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.currentList())-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if m.tab == tabSearch {
				q := strings.TrimSpace(m.searchInput.Value())
				if q != "" {
					m.loading = true
					m.searchResult = nil
					m.cursor = 0
					return m, tea.Batch(searchNixpkgs(q), m.spinner.Tick)
				}
			}
			return m, nil
		}

	case installedLoadedMsg:
		sort.Slice(msg.pkgs, func(i, j int) bool {
			return msg.pkgs[i].Name < msg.pkgs[j].Name
		})
		m.installed = msg.pkgs
		m.loading = false
		return m, nil

	case searchResultMsg:
		sort.Slice(msg.pkgs, func(i, j int) bool {
			return msg.pkgs[i].Name < msg.pkgs[j].Name
		})
		m.searchResult = msg.pkgs
		m.loading = false
		return m, nil

	case nixErrMsg:
		m.err = msg.err
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

	if m.tab == tabSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) currentList() []pkg {
	if m.tab == tabInstalled {
		return m.installed
	}
	return m.searchResult
}

func (m model) View() string {
	var b strings.Builder

	installedLabel := "Installed"
	searchLabel := "Search"
	if m.tab == tabInstalled {
		installedLabel = activeTabS.Render(installedLabel)
		searchLabel = inactiveTabS.Render(searchLabel)
	} else {
		installedLabel = inactiveTabS.Render(installedLabel)
		searchLabel = activeTabS.Render(searchLabel)
	}
	b.WriteString(titleStyle.Render("pkg-browser") + "  ")
	b.WriteString(installedLabel + "  " + searchLabel + "\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	if m.tab == tabSearch {
		b.WriteString(m.searchInput.View() + "\n")
		b.WriteString(strings.Repeat("─", m.width) + "\n")
	}

	if m.err != nil {
		b.WriteString(errStyle.Render("Error: "+m.err.Error()) + "\n")
		return b.String()
	}

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading...\n")
		return b.String()
	}

	list := m.currentList()
	if len(list) == 0 {
		if m.tab == tabInstalled {
			b.WriteString(descStyle.Render("No packages installed.") + "\n")
		} else {
			b.WriteString(descStyle.Render("Type a query and press Enter to search.") + "\n")
		}
		return b.String()
	}

	headerRows := 3
	if m.tab == tabSearch {
		headerRows = 5
	}
	maxRows := m.height - headerRows - 1
	if maxRows < 1 {
		maxRows = 1
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(list) {
		end = len(list)
	}

	for i := start; i < end; i++ {
		p := list[i]
		nameVer := p.Name
		if p.Version != "" {
			nameVer += " " + versionStyle.Render(p.Version)
		}
		line := nameVer
		if p.Description != "" {
			line += "  " + descStyle.Render(p.Description)
		}
		if i == m.cursor {
			b.WriteString(selectedRow.Render("> ") + line + "\n")
		} else {
			b.WriteString("  " + normalRow.Render(line) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf(
		"tab: switch  ↑/↓ j/k: navigate  %d/%d  q: quit",
		m.cursor+1, len(list),
	)))
	return b.String()
}
