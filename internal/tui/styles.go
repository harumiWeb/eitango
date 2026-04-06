package tui

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

const (
	ThemeDefault = "default"
	ThemeNoColor = "no_color"
	ThemeNeon    = "neon"
	ThemeCustom  = "custom"
)

type Theme struct {
	Mode    string
	Palette ThemePalette
}

type ThemePalette struct {
	Accent  string
	Success string
	Danger  string
	Muted   string
	Border  string
}

type Styles struct {
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Panel          lipgloss.Style
	ModalPanel     lipgloss.Style
	CorrectPanel   lipgloss.Style
	WrongPanel     lipgloss.Style
	Choice         lipgloss.Style
	ChoiceSelected lipgloss.Style
	Correct        lipgloss.Style
	Wrong          lipgloss.Style
	Status         lipgloss.Style
	Error          lipgloss.Style
	Muted          lipgloss.Style
	QuizMeta       lipgloss.Style
	Accent         lipgloss.Style
}

func NewStyles(theme Theme) Styles {
	mode := NormalizeThemeMode(theme.Mode)
	switch mode {
	case ThemeNoColor:
		return buildNoColorStyles()
	case ThemeNeon:
		return applyPalette(buildDefaultStyles(), neonPalette())
	case ThemeCustom:
		return applyPalette(buildDefaultStyles(), theme.Palette)
	default:
		return buildDefaultStyles()
	}
}

// ResolvePalette returns the effective palette for config-facing uses.
// For ThemeNoColor it intentionally returns the default palette, while
// NewStyles ignores palettes entirely in no-color rendering.
func ResolvePalette(theme Theme) ThemePalette {
	switch NormalizeThemeMode(theme.Mode) {
	case ThemeNeon:
		return neonPalette()
	case ThemeCustom:
		return mergePalette(defaultPalette(), theme.Palette)
	default:
		return defaultPalette()
	}
}

func defaultPalette() ThemePalette {
	return ThemePalette{
		Accent:  "81",
		Success: "42",
		Danger:  "203",
		Muted:   "245",
		Border:  "",
	}
}

func neonPalette() ThemePalette {
	return ThemePalette{
		Accent:  "#A6FF00",
		Success: "#7DFF7A",
		Danger:  "#FF6B6B",
		Muted:   "#8FB38F",
		Border:  "#D7FFAF",
	}
}

func mergePalette(base, override ThemePalette) ThemePalette {
	if override.Accent != "" {
		base.Accent = override.Accent
	}
	if override.Success != "" {
		base.Success = override.Success
	}
	if override.Danger != "" {
		base.Danger = override.Danger
	}
	if override.Muted != "" {
		base.Muted = override.Muted
	}
	if override.Border != "" {
		base.Border = override.Border
	}
	return base
}

func buildNoColorStyles() Styles {
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	return Styles{
		Title:          lipgloss.NewStyle().Bold(true),
		Subtitle:       lipgloss.NewStyle().Bold(true),
		Panel:          panel,
		ModalPanel:     lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1, 2),
		CorrectPanel:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
		WrongPanel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
		Choice:         lipgloss.NewStyle().PaddingLeft(1),
		ChoiceSelected: lipgloss.NewStyle().PaddingLeft(1).Bold(true),
		Correct:        lipgloss.NewStyle().Bold(true),
		Wrong:          lipgloss.NewStyle().Bold(true),
		Status:         lipgloss.NewStyle(),
		Error:          lipgloss.NewStyle().Bold(true),
		Muted:          lipgloss.NewStyle(),
		QuizMeta:       lipgloss.NewStyle(),
		Accent:         lipgloss.NewStyle().Bold(true),
	}
}

func buildDefaultStyles() Styles {
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	return Styles{
		Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		Subtitle:       lipgloss.NewStyle().Bold(true),
		Panel:          panel,
		ModalPanel:     lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("81")),
		CorrectPanel:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("42")),
		WrongPanel:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("203")),
		Choice:         lipgloss.NewStyle().PaddingLeft(1),
		ChoiceSelected: lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(lipgloss.Color("86")),
		Correct:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		Wrong:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")),
		Status:         lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		Error:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")),
		Muted:          lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		QuizMeta:       lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		Accent:         lipgloss.NewStyle().Foreground(lipgloss.Color("81")),
	}
}

func applyPalette(styles Styles, palette ThemePalette) Styles {
	if palette.Accent != "" {
		styles.Title = foregroundStyle(styles.Title, palette.Accent)
		styles.ModalPanel = borderStyle(styles.ModalPanel, palette.Accent)
		styles.ChoiceSelected = foregroundStyle(styles.ChoiceSelected, palette.Accent)
		styles.Accent = foregroundStyle(styles.Accent, palette.Accent)
	}
	if palette.Success != "" {
		styles.CorrectPanel = borderStyle(styles.CorrectPanel, palette.Success)
		styles.Correct = foregroundStyle(styles.Correct, palette.Success)
	}
	if palette.Danger != "" {
		styles.WrongPanel = borderStyle(styles.WrongPanel, palette.Danger)
		styles.Wrong = foregroundStyle(styles.Wrong, palette.Danger)
		styles.Error = foregroundStyle(styles.Error, palette.Danger)
	}
	if palette.Muted != "" {
		styles.Subtitle = foregroundStyle(styles.Subtitle, palette.Muted)
		styles.Status = foregroundStyle(styles.Status, palette.Muted)
		styles.Muted = foregroundStyle(styles.Muted, palette.Muted)
		styles.QuizMeta = foregroundStyle(styles.QuizMeta, palette.Muted)
	}
	if palette.Border != "" {
		styles.Panel = borderStyle(styles.Panel, palette.Border)
	}
	return styles
}

func NormalizeThemeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ThemeNoColor:
		return ThemeNoColor
	case ThemeNeon:
		return ThemeNeon
	case ThemeCustom:
		return ThemeCustom
	default:
		return ThemeDefault
	}
}

func foregroundStyle(style lipgloss.Style, colorSpec string) lipgloss.Style {
	if colorSpec == "" {
		return style
	}
	return style.Foreground(lipgloss.Color(colorSpec))
}

func borderStyle(style lipgloss.Style, colorSpec string) lipgloss.Style {
	if colorSpec == "" {
		return style
	}
	return style.BorderForeground(lipgloss.Color(colorSpec))
}
