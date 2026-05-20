package theme

import "github.com/charmbracelet/lipgloss"

func tokyoNight256() ThemeColors {
	return ThemeColors{
		BG:        lipgloss.Color("234"),
		FG:        lipgloss.Color("189"),
		Subtle:    lipgloss.Color("60"),
		Muted:     lipgloss.Color("238"),
		Border:    lipgloss.Color("60"),
		Accent:    lipgloss.Color("111"),
		Success:   lipgloss.Color("114"),
		Warning:   lipgloss.Color("179"),
		Error:     lipgloss.Color("210"),
		Info:      lipgloss.Color("117"),
		ToolName:  lipgloss.Color("141"),
		FilePath:  lipgloss.Color("117"),
		CodeBlock: lipgloss.Color("189"),
		DiffAdd:   lipgloss.Color("114"),
		DiffDel:   lipgloss.Color("210"),
	}
}
