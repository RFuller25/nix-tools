package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))

	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	seedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("180")).Bold(true)
	weedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("101"))
	bareSoilStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	waterStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("74"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	latinStyle    = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("152"))
	lockedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))

	selBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("114"))

	plainBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("237"))

	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("108")).
			Padding(0, 1)
)

func rarityStyle(r Rarity) lipgloss.Style {
	switch r {
	case Common:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	case Uncommon:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	case Rare:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("177")).Bold(true)
	}
}

// meter draws a small proportional bar, used for moisture, weeds and growth.
func meter(v float64, width int, style lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	filled := int(v*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	out := style.Render(repeat("█", filled))
	return out + dividerStyle.Render(repeat("░", width-filled))
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
