package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTokyoNightTrueColorPalette(t *testing.T) {
	colors := TokyoNightForProfile(ColorProfileTrueColor)
	assertColor(t, colors.BG, "#1a1b26")
	assertColor(t, colors.FG, "#c0caf5")
	assertColor(t, colors.Accent, "#7aa2f7")
	assertColor(t, colors.Success, "#9ece6a")
	assertColor(t, colors.Warning, "#e0af68")
	assertColor(t, colors.Error, "#f7768e")
	assertColor(t, colors.Info, "#7dcfff")
	assertColor(t, colors.Subtle, "#565f89")
	assertColor(t, colors.Muted, "#414868")
}

func TestTokyoNight256FallbackPalette(t *testing.T) {
	colors := TokyoNightForProfile(ColorProfile256)
	assertColor(t, colors.BG, "234")
	assertColor(t, colors.FG, "189")
	assertColor(t, colors.Accent, "111")
	assertColor(t, colors.Success, "114")
	assertColor(t, colors.Warning, "179")
	assertColor(t, colors.Error, "210")
	assertColor(t, colors.Info, "117")
	assertColor(t, colors.Subtle, "60")
	assertColor(t, colors.Muted, "238")
}

func TestDetectColorProfileFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want ColorProfile
	}{
		{name: "truecolor", env: map[string]string{"COLORTERM": "truecolor"}, want: ColorProfileTrueColor},
		{name: "24bit", env: map[string]string{"COLORTERM": "24bit"}, want: ColorProfileTrueColor},
		{name: "forced", env: map[string]string{"NEOCODE_TUI_TRUECOLOR": "1"}, want: ColorProfileTrueColor},
		{name: "fallback", env: map[string]string{"TERM": "xterm-256color"}, want: ColorProfile256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectColorProfileFromEnv(func(key string) string { return tt.env[key] })
			if got != tt.want {
				t.Fatalf("DetectColorProfileFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSymbolsAndASCIIInputs(t *testing.T) {
	if StatusSymbol(PhaseRunning) == "" || AccentBar() == "" || StreamPrefix("tool_finished") == "" {
		t.Fatal("symbol helpers returned empty string")
	}

	if !DetectASCIISymbolsFromEnv(func(key string) string {
		if key == "NEOCODE_TUI_ASCII" {
			return "1"
		}
		return ""
	}) {
		t.Fatal("NEOCODE_TUI_ASCII=1 should force ASCII symbols")
	}

	if !DetectASCIISymbolsFromEnv(func(key string) string {
		if key == "TERM" {
			return "dumb"
		}
		return ""
	}) {
		t.Fatal("TERM=dumb should force ASCII symbols")
	}
}

func TestTruncateUsesDisplayWidth(t *testing.T) {
	got := Truncate("你好abcdef", 6)
	if DisplayWidth(got) > 6 {
		t.Fatalf("DisplayWidth(%q) = %d, want <= 6", got, DisplayWidth(got))
	}
}

func assertColor(t *testing.T, got lipgloss.Color, want string) {
	t.Helper()
	if string(got) != want {
		t.Fatalf("color = %q, want %q", string(got), want)
	}
}
