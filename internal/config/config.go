package config

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/session"
)

const appDirName = "eitango-cli"

const (
	WriteModeDifficultyBasic = "basic"
	WriteModeDifficultyHard  = "hard"
	ThemeModeDefault         = "default"
	ThemeModeNoColor         = "no_color"
	ThemeModeNeon            = "neon"
	ThemeModeCustom          = "custom"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)

type Paths struct {
	DataDir    string
	DBPath     string
	ConfigPath string
	LogsDir    string
}

type Settings struct {
	SessionSize         int
	ReviewRatio         float64
	FocusModeDefault    bool
	WriteModeDifficulty string
	AudioEnabled        bool
	AudioAutoplay       bool
	Language            string
	ThemeMode           string
	ThemePalette        ThemePalette
}

type ThemePalette struct {
	Accent  string
	Success string
	Danger  string
	Muted   string
	Border  string
}

type fileSettings struct {
	SessionSize         *int             `toml:"session_size"`
	ReviewRatio         *float64         `toml:"review_ratio"`
	FocusModeDefault    *bool            `toml:"focus_mode_default"`
	WriteModeDifficulty *string          `toml:"write_mode_difficulty"`
	AudioEnabled        *bool            `toml:"audio_enabled"`
	AudioAutoplay       *bool            `toml:"audio_autoplay"`
	Language            *string          `toml:"language"`
	ThemeMode           *string          `toml:"theme_mode"`
	ThemePalette        fileThemePalette `toml:"theme_palette"`
}

type fileThemePalette struct {
	Accent  *string `toml:"accent"`
	Success *string `toml:"success"`
	Danger  *string `toml:"danger"`
	Muted   *string `toml:"muted"`
	Border  *string `toml:"border"`
}

func DefaultSettings() Settings {
	return Settings{
		SessionSize:         session.DefaultQuestionCount,
		ReviewRatio:         session.DefaultReviewRatio,
		WriteModeDifficulty: WriteModeDifficultyBasic,
		AudioEnabled:        true,
		AudioAutoplay:       false,
		Language:            i18n.DefaultLang,
		ThemeMode:           ThemeModeDefault,
	}
}

func Load(path string) (Settings, error) {
	settings := DefaultSettings()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw fileSettings
	meta, err := toml.Decode(string(data), &raw)
	if err != nil {
		return Settings{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return Settings{}, fmt.Errorf("unknown config keys: %s", joinUndecoded(undecoded))
	}

	if raw.SessionSize != nil {
		settings.SessionSize = *raw.SessionSize
	}
	if raw.ReviewRatio != nil {
		settings.ReviewRatio = *raw.ReviewRatio
	}
	if raw.FocusModeDefault != nil {
		settings.FocusModeDefault = *raw.FocusModeDefault
	}
	if raw.WriteModeDifficulty != nil {
		writeModeDifficulty, err := parseWriteModeDifficulty(*raw.WriteModeDifficulty)
		if err != nil {
			return Settings{}, err
		}
		settings.WriteModeDifficulty = writeModeDifficulty
	}
	if raw.AudioEnabled != nil {
		settings.AudioEnabled = *raw.AudioEnabled
	}
	if raw.AudioAutoplay != nil {
		settings.AudioAutoplay = *raw.AudioAutoplay
	}
	if raw.Language != nil {
		settings.Language = *raw.Language
	}
	if raw.ThemeMode != nil {
		themeMode, err := parseThemeMode(*raw.ThemeMode)
		if err != nil {
			return Settings{}, err
		}
		settings.ThemeMode = themeMode
	}
	if raw.ThemePalette.hasAny() {
		themePalette, err := parseThemePalette(raw.ThemePalette)
		if err != nil {
			return Settings{}, err
		}
		settings.ThemePalette = themePalette
	}

	if err := validateSettings(settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func Save(path string, settings Settings) error {
	writeModeDifficulty, err := parseWriteModeDifficulty(settings.WriteModeDifficulty)
	if err != nil {
		return err
	}
	settings.WriteModeDifficulty = writeModeDifficulty
	settings.ThemeMode = NormalizeThemeMode(settings.ThemeMode)
	settings.ThemePalette, err = normalizeThemePalette(settings.ThemePalette)
	if err != nil {
		return err
	}

	if err := validateSettings(settings); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(struct {
		SessionSize         int               `toml:"session_size"`
		ReviewRatio         float64           `toml:"review_ratio"`
		FocusModeDefault    bool              `toml:"focus_mode_default"`
		WriteModeDifficulty string            `toml:"write_mode_difficulty"`
		AudioEnabled        bool              `toml:"audio_enabled"`
		AudioAutoplay       bool              `toml:"audio_autoplay"`
		Language            string            `toml:"language"`
		ThemeMode           string            `toml:"theme_mode"`
		ThemePalette        *saveThemePalette `toml:"theme_palette,omitempty"`
	}{
		SessionSize:         settings.SessionSize,
		ReviewRatio:         settings.ReviewRatio,
		FocusModeDefault:    settings.FocusModeDefault,
		WriteModeDifficulty: settings.WriteModeDifficulty,
		AudioEnabled:        settings.AudioEnabled,
		AudioAutoplay:       settings.AudioAutoplay,
		Language:            settings.Language,
		ThemeMode:           settings.ThemeMode,
		ThemePalette:        newSaveThemePalette(settings.ThemePalette),
	}); err != nil {
		return fmt.Errorf("encode config %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "eitango-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config for %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp config for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config for %s: %w", path, err)
	}

	if err := replaceFile(tmpPath, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

func Resolve() (Paths, error) {
	baseDir, err := dataDir()
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		DataDir:    baseDir,
		DBPath:     filepath.Join(baseDir, "user.db"),
		ConfigPath: filepath.Join(baseDir, "config.toml"),
		LogsDir:    filepath.Join(baseDir, "logs"),
	}, nil
}

func Ensure() (Paths, error) {
	paths, err := Resolve()
	if err != nil {
		return Paths{}, err
	}

	for _, dir := range []string{paths.DataDir, paths.LogsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Paths{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}

	return paths, nil
}

func dataDir() (string, error) {
	if override := os.Getenv("EITANGO_DATA_DIR"); override != "" {
		return override, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, appDirName), nil
		}
		return filepath.Join(home, "AppData", "Roaming", appDirName), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", appDirName), nil
	default:
		return filepath.Join(home, ".local", "share", appDirName), nil
	}
}

func joinUndecoded(keys []toml.Key) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key.String())
	}
	return strings.Join(parts, ", ")
}

func NormalizeWriteModeDifficulty(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case WriteModeDifficultyHard:
		return WriteModeDifficultyHard
	default:
		return WriteModeDifficultyBasic
	}
}

func NormalizeThemeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ThemeModeNoColor:
		return ThemeModeNoColor
	case ThemeModeNeon:
		return ThemeModeNeon
	case ThemeModeCustom:
		return ThemeModeCustom
	default:
		return ThemeModeDefault
	}
}

func parseWriteModeDifficulty(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case WriteModeDifficultyBasic:
		return WriteModeDifficultyBasic, nil
	case WriteModeDifficultyHard:
		return WriteModeDifficultyHard, nil
	default:
		return "", fmt.Errorf("write_mode_difficulty must be %q or %q", WriteModeDifficultyBasic, WriteModeDifficultyHard)
	}
}

func parseThemeMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ThemeModeDefault:
		return ThemeModeDefault, nil
	case ThemeModeNoColor:
		return ThemeModeNoColor, nil
	case ThemeModeNeon:
		return ThemeModeNeon, nil
	case ThemeModeCustom:
		return ThemeModeCustom, nil
	default:
		return "", fmt.Errorf("theme_mode must be %q, %q, %q, or %q", ThemeModeDefault, ThemeModeNoColor, ThemeModeNeon, ThemeModeCustom)
	}
}

func parseThemePalette(raw fileThemePalette) (ThemePalette, error) {
	palette := ThemePalette{}
	var err error

	if raw.Accent != nil {
		palette.Accent, err = normalizeThemeColor(*raw.Accent)
		if err != nil {
			return ThemePalette{}, fmt.Errorf("theme_palette.accent: %w", err)
		}
	}
	if raw.Success != nil {
		palette.Success, err = normalizeThemeColor(*raw.Success)
		if err != nil {
			return ThemePalette{}, fmt.Errorf("theme_palette.success: %w", err)
		}
	}
	if raw.Danger != nil {
		palette.Danger, err = normalizeThemeColor(*raw.Danger)
		if err != nil {
			return ThemePalette{}, fmt.Errorf("theme_palette.danger: %w", err)
		}
	}
	if raw.Muted != nil {
		palette.Muted, err = normalizeThemeColor(*raw.Muted)
		if err != nil {
			return ThemePalette{}, fmt.Errorf("theme_palette.muted: %w", err)
		}
	}
	if raw.Border != nil {
		palette.Border, err = normalizeThemeColor(*raw.Border)
		if err != nil {
			return ThemePalette{}, fmt.Errorf("theme_palette.border: %w", err)
		}
	}

	return palette, nil
}

func normalizeThemePalette(palette ThemePalette) (ThemePalette, error) {
	var err error

	palette.Accent, err = normalizeThemeColor(palette.Accent)
	if err != nil {
		return ThemePalette{}, fmt.Errorf("theme_palette.accent: %w", err)
	}
	palette.Success, err = normalizeThemeColor(palette.Success)
	if err != nil {
		return ThemePalette{}, fmt.Errorf("theme_palette.success: %w", err)
	}
	palette.Danger, err = normalizeThemeColor(palette.Danger)
	if err != nil {
		return ThemePalette{}, fmt.Errorf("theme_palette.danger: %w", err)
	}
	palette.Muted, err = normalizeThemeColor(palette.Muted)
	if err != nil {
		return ThemePalette{}, fmt.Errorf("theme_palette.muted: %w", err)
	}
	palette.Border, err = normalizeThemeColor(palette.Border)
	if err != nil {
		return ThemePalette{}, fmt.Errorf("theme_palette.border: %w", err)
	}

	return palette, nil
}

func normalizeThemeColor(value string) (string, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "", nil
	}
	if !hexColorPattern.MatchString(trimmed) {
		return "", fmt.Errorf("must be #RRGGBB")
	}
	return trimmed, nil
}

func validateSettings(settings Settings) error {
	if settings.SessionSize <= 0 {
		return fmt.Errorf("session_size must be greater than 0")
	}
	if math.IsNaN(settings.ReviewRatio) || settings.ReviewRatio < 0 || settings.ReviewRatio > 1 {
		return fmt.Errorf("review_ratio must be between 0 and 1")
	}
	if normalized := NormalizeWriteModeDifficulty(settings.WriteModeDifficulty); normalized != settings.WriteModeDifficulty {
		return fmt.Errorf("write_mode_difficulty must be %q or %q", WriteModeDifficultyBasic, WriteModeDifficultyHard)
	}
	if !i18n.ValidLang(settings.Language) {
		return fmt.Errorf("unsupported language: %q (use %q or %q)", settings.Language, i18n.LangJA, i18n.LangEN)
	}
	if normalized := NormalizeThemeMode(settings.ThemeMode); normalized != settings.ThemeMode {
		return fmt.Errorf("theme_mode must be %q, %q, %q, or %q", ThemeModeDefault, ThemeModeNoColor, ThemeModeNeon, ThemeModeCustom)
	}
	if _, err := normalizeThemePalette(settings.ThemePalette); err != nil {
		return err
	}
	return nil
}

func (p fileThemePalette) hasAny() bool {
	return p.Accent != nil || p.Success != nil || p.Danger != nil || p.Muted != nil || p.Border != nil
}

type saveThemePalette struct {
	Accent  *string `toml:"accent,omitempty"`
	Success *string `toml:"success,omitempty"`
	Danger  *string `toml:"danger,omitempty"`
	Muted   *string `toml:"muted,omitempty"`
	Border  *string `toml:"border,omitempty"`
}

func newSaveThemePalette(palette ThemePalette) *saveThemePalette {
	if palette == (ThemePalette{}) {
		return nil
	}
	saved := &saveThemePalette{
		Accent:  saveThemeColor(palette.Accent),
		Success: saveThemeColor(palette.Success),
		Danger:  saveThemeColor(palette.Danger),
		Muted:   saveThemeColor(palette.Muted),
		Border:  saveThemeColor(palette.Border),
	}
	if saved.Accent == nil && saved.Success == nil && saved.Danger == nil && saved.Muted == nil && saved.Border == nil {
		return nil
	}
	return saved
}

func saveThemeColor(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}
