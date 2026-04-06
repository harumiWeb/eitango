package tui

import (
	"image/color"
	"reflect"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
)

func TestResolvePaletteDefault(t *testing.T) {
	t.Parallel()

	palette := ResolvePalette(Theme{Mode: ThemeDefault})
	want := ThemePalette{
		Accent:  "81",
		Success: "42",
		Danger:  "203",
		Muted:   "245",
		Border:  "",
	}
	if palette != want {
		t.Fatalf("palette = %+v, want %+v", palette, want)
	}
}

func TestResolvePaletteNeon(t *testing.T) {
	t.Parallel()

	palette := ResolvePalette(Theme{Mode: ThemeNeon})
	want := ThemePalette{
		Accent:  "#A6FF00",
		Success: "#7DFF7A",
		Danger:  "#FF6B6B",
		Muted:   "#8FB38F",
		Border:  "#D7FFAF",
	}
	if palette != want {
		t.Fatalf("palette = %+v, want %+v", palette, want)
	}
}

func TestResolvePaletteCustomReturnsOverridesOnly(t *testing.T) {
	t.Parallel()

	palette := ResolvePalette(Theme{
		Mode: ThemeCustom,
		Palette: ThemePalette{
			Accent: "#112233",
			Border: "#445566",
		},
	})
	if palette.Accent != "#112233" {
		t.Fatalf("Accent = %q, want %q", palette.Accent, "#112233")
	}
	if palette.Border != "#445566" {
		t.Fatalf("Border = %q, want %q", palette.Border, "#445566")
	}
	if palette.Success != "" || palette.Danger != "" || palette.Muted != "" {
		t.Fatalf("palette = %+v, want only explicit overrides", palette)
	}
}

func TestNewStylesNoColorUnsetsForegrounds(t *testing.T) {
	t.Parallel()

	styles := NewStyles(Theme{Mode: ThemeNoColor})
	if !isNoColor(styles.Title.GetForeground()) {
		t.Fatalf("Title foreground = %#v, want NoColor", styles.Title.GetForeground())
	}
	if !isNoColor(styles.Panel.GetBorderTopForeground()) {
		t.Fatalf("Panel border foreground = %#v, want NoColor", styles.Panel.GetBorderTopForeground())
	}
}

func TestNewStylesDefaultPreservesLegacyColors(t *testing.T) {
	t.Parallel()

	styles := NewStyles(Theme{Mode: ThemeDefault})
	if !reflect.DeepEqual(styles.Title.GetForeground(), lipgloss.Color("63")) {
		t.Fatalf("Title foreground = %#v, want %#v", styles.Title.GetForeground(), lipgloss.Color("63"))
	}
	if !reflect.DeepEqual(styles.ModalPanel.GetBorderTopForeground(), lipgloss.Color("81")) {
		t.Fatalf("ModalPanel border = %#v, want %#v", styles.ModalPanel.GetBorderTopForeground(), lipgloss.Color("81"))
	}
	if !reflect.DeepEqual(styles.ChoiceSelected.GetForeground(), lipgloss.Color("86")) {
		t.Fatalf("ChoiceSelected foreground = %#v, want %#v", styles.ChoiceSelected.GetForeground(), lipgloss.Color("86"))
	}
	if !reflect.DeepEqual(styles.Status.GetForeground(), lipgloss.Color("243")) {
		t.Fatalf("Status foreground = %#v, want %#v", styles.Status.GetForeground(), lipgloss.Color("243"))
	}
	if !reflect.DeepEqual(styles.Muted.GetForeground(), lipgloss.Color("245")) {
		t.Fatalf("Muted foreground = %#v, want %#v", styles.Muted.GetForeground(), lipgloss.Color("245"))
	}
	if !reflect.DeepEqual(styles.QuizMeta.GetForeground(), lipgloss.Color("241")) {
		t.Fatalf("QuizMeta foreground = %#v, want %#v", styles.QuizMeta.GetForeground(), lipgloss.Color("241"))
	}
	if !isNoColor(styles.Panel.GetBorderTopForeground()) {
		t.Fatalf("Panel border foreground = %#v, want NoColor", styles.Panel.GetBorderTopForeground())
	}
}

func TestNewStylesNeonUsesPresetColors(t *testing.T) {
	t.Parallel()

	styles := NewStyles(Theme{Mode: ThemeNeon})
	if !reflect.DeepEqual(styles.Title.GetForeground(), lipgloss.Color("#A6FF00")) {
		t.Fatalf("Title foreground = %#v, want %#v", styles.Title.GetForeground(), lipgloss.Color("#A6FF00"))
	}
	if !reflect.DeepEqual(styles.Error.GetForeground(), lipgloss.Color("#FF6B6B")) {
		t.Fatalf("Error foreground = %#v, want %#v", styles.Error.GetForeground(), lipgloss.Color("#FF6B6B"))
	}
	if !reflect.DeepEqual(styles.Status.GetForeground(), lipgloss.Color("#8FB38F")) {
		t.Fatalf("Status foreground = %#v, want %#v", styles.Status.GetForeground(), lipgloss.Color("#8FB38F"))
	}
	if !reflect.DeepEqual(styles.Panel.GetBorderTopForeground(), lipgloss.Color("#D7FFAF")) {
		t.Fatalf("Panel border = %#v, want %#v", styles.Panel.GetBorderTopForeground(), lipgloss.Color("#D7FFAF"))
	}
}

func TestNewStylesCustomKeepsLegacyFallbackForUnsetSlots(t *testing.T) {
	t.Parallel()

	styles := NewStyles(Theme{
		Mode: ThemeCustom,
		Palette: ThemePalette{
			Accent: "#112233",
		},
	})
	if !reflect.DeepEqual(styles.Title.GetForeground(), lipgloss.Color("#112233")) {
		t.Fatalf("Title foreground = %#v, want %#v", styles.Title.GetForeground(), lipgloss.Color("#112233"))
	}
	if !reflect.DeepEqual(styles.Status.GetForeground(), lipgloss.Color("243")) {
		t.Fatalf("Status foreground = %#v, want legacy fallback %#v", styles.Status.GetForeground(), lipgloss.Color("243"))
	}
	if !isNoColor(styles.Panel.GetBorderTopForeground()) {
		t.Fatalf("Panel border foreground = %#v, want NoColor", styles.Panel.GetBorderTopForeground())
	}
}

func isNoColor(c color.Color) bool {
	_, ok := c.(lipgloss.NoColor)
	return ok
}
