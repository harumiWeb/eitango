package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(settings, DefaultSettings()) {
		t.Fatalf("settings = %+v, want %+v", settings, DefaultSettings())
	}
}

func TestLoadOverridesSupportedSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 12
review_ratio = 0.4
focus_mode_default = true
write_mode_difficulty = "hard"
audio_enabled = false
audio_autoplay = true
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.SessionSize != 12 {
		t.Fatalf("SessionSize = %d, want 12", settings.SessionSize)
	}
	if settings.ReviewRatio != 0.4 {
		t.Fatalf("ReviewRatio = %v, want 0.4", settings.ReviewRatio)
	}
	if !settings.FocusModeDefault {
		t.Fatal("FocusModeDefault = false, want true")
	}
	if settings.WriteModeDifficulty != WriteModeDifficultyHard {
		t.Fatalf("WriteModeDifficulty = %q, want %q", settings.WriteModeDifficulty, WriteModeDifficultyHard)
	}
	if settings.AudioEnabled {
		t.Fatal("AudioEnabled = true, want false")
	}
	if !settings.AudioAutoplay {
		t.Fatal("AudioAutoplay = false, want true")
	}
	if settings.ThemeMode != ThemeModeDefault {
		t.Fatalf("ThemeMode = %q, want %q", settings.ThemeMode, ThemeModeDefault)
	}
}

func TestLoadNormalizesWriteModeDifficulty(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 12
review_ratio = 0.4
write_mode_difficulty = " Hard "
language = "ja"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.WriteModeDifficulty != WriteModeDifficultyHard {
		t.Fatalf("WriteModeDifficulty = %q, want %q", settings.WriteModeDifficulty, WriteModeDifficultyHard)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("session_size = 0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "session_size") {
		t.Fatalf("Load() error = %v, want session_size validation", err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("unexpected_key = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown config keys") {
		t.Fatalf("Load() error = %v, want unknown key error", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("Load() error = %v, want config path %q", err, path)
	}
}

func TestSaveRoundTripsSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	want := Settings{
		SessionSize:         15,
		ReviewRatio:         0.6,
		FocusModeDefault:    true,
		WriteModeDifficulty: WriteModeDifficultyHard,
		AudioEnabled:        false,
		AudioAutoplay:       true,
		Language:            "en",
		ThemeMode:           ThemeModeDefault,
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

func TestLoadMissingWriteModeDifficultyDefaultsToBasic(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("session_size = 10\nreview_ratio = 0.4\nlanguage = \"ja\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.WriteModeDifficulty != WriteModeDifficultyBasic {
		t.Fatalf("WriteModeDifficulty = %q, want %q", settings.WriteModeDifficulty, WriteModeDifficultyBasic)
	}
}

func TestLoadKeymapOverrides(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 10
review_ratio = 0.4
language = "ja"

[keymap.home]
toggle_answer_mode = ["x"]
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := settings.Keymap.Home["toggle_answer_mode"]; !reflect.DeepEqual(got, []string{"x"}) {
		t.Fatalf("Keymap.Home[toggle_answer_mode] = %v, want [x]", got)
	}
}

func TestSaveOmitsEmptyKeymap(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := Save(path, DefaultSettings()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(data), "[keymap]") {
		t.Fatalf("config = %q, must omit empty keymap", string(data))
	}
}

func TestSaveRoundTripsKeymapOverride(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	settings := DefaultSettings()
	settings.Keymap.Home = map[string][]string{
		"toggle_answer_mode": {"x"},
	}
	settings.Keymap.Version = 1

	if err := Save(path, settings); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if override := got.Keymap.Home["toggle_answer_mode"]; !reflect.DeepEqual(override, []string{"x"}) {
		t.Fatalf("got.Keymap.Home[toggle_answer_mode] = %v, want [x]", override)
	}
}

func TestSaveRejectsInvalidSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	err := Save(path, Settings{
		SessionSize:         0,
		ReviewRatio:         0.4,
		WriteModeDifficulty: WriteModeDifficultyBasic,
		AudioEnabled:        true,
		AudioAutoplay:       false,
		Language:            "ja",
	})
	if err == nil {
		t.Fatal("Save() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "session_size") {
		t.Fatalf("Save() error = %v, want session_size validation", err)
	}
}

func TestSaveRejectsInvalidWriteModeDifficulty(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	err := Save(path, Settings{
		SessionSize:         10,
		ReviewRatio:         0.4,
		WriteModeDifficulty: "invalid",
		AudioEnabled:        true,
		AudioAutoplay:       false,
		Language:            "ja",
	})
	if err == nil {
		t.Fatal("Save() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "write_mode_difficulty") {
		t.Fatalf("Save() error = %v, want write_mode_difficulty validation", err)
	}
}

func TestSaveNormalizesWriteModeDifficulty(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := Save(path, Settings{
		SessionSize:         10,
		ReviewRatio:         0.4,
		WriteModeDifficulty: " Hard ",
		AudioEnabled:        true,
		AudioAutoplay:       true,
		Language:            "ja",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.WriteModeDifficulty != WriteModeDifficultyHard {
		t.Fatalf("WriteModeDifficulty = %q, want %q", settings.WriteModeDifficulty, WriteModeDifficultyHard)
	}
}

func TestLoadRejectsInvalidWriteModeDifficulty(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 10
review_ratio = 0.4
write_mode_difficulty = "invalid"
language = "ja"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "write_mode_difficulty") {
		t.Fatalf("Load() error = %v, want write_mode_difficulty validation", err)
	}
}

func TestLoadMissingAudioSettingsDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("session_size = 10\nreview_ratio = 0.4\nlanguage = \"ja\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !settings.AudioEnabled {
		t.Fatal("AudioEnabled = false, want true")
	}
	if settings.AudioAutoplay {
		t.Fatal("AudioAutoplay = true, want false")
	}
	if settings.ThemeMode != ThemeModeDefault {
		t.Fatalf("ThemeMode = %q, want %q", settings.ThemeMode, ThemeModeDefault)
	}
}

func TestLoadThemeSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 12
review_ratio = 0.4
theme_mode = "custom"

[theme_palette]
accent = " #a6ff00 "
danger = "#ff6b6b"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.ThemeMode != ThemeModeCustom {
		t.Fatalf("ThemeMode = %q, want %q", settings.ThemeMode, ThemeModeCustom)
	}
	if settings.ThemePalette.Accent != "#A6FF00" {
		t.Fatalf("ThemePalette.Accent = %q, want %q", settings.ThemePalette.Accent, "#A6FF00")
	}
	if settings.ThemePalette.Danger != "#FF6B6B" {
		t.Fatalf("ThemePalette.Danger = %q, want %q", settings.ThemePalette.Danger, "#FF6B6B")
	}
}

func TestSaveRoundTripsThemeSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	want := Settings{
		SessionSize:         15,
		ReviewRatio:         0.6,
		FocusModeDefault:    true,
		WriteModeDifficulty: WriteModeDifficultyHard,
		AudioEnabled:        false,
		AudioAutoplay:       true,
		Language:            "en",
		ThemeMode:           ThemeModeCustom,
		ThemePalette: ThemePalette{
			Accent:  "#A6FF00",
			Success: "#7DFF7A",
			Danger:  "#FF6B6B",
			Muted:   "#8FB38F",
			Border:  "#D7FFAF",
		},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

func TestSaveOmitsEmptyThemePaletteSlots(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	settings := Settings{
		SessionSize:         10,
		ReviewRatio:         0.4,
		WriteModeDifficulty: WriteModeDifficultyBasic,
		AudioEnabled:        true,
		AudioAutoplay:       false,
		Language:            "ja",
		ThemeMode:           ThemeModeCustom,
		ThemePalette: ThemePalette{
			Accent: "#A6FF00",
		},
	}

	if err := Save(path, settings); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "[theme_palette]") {
		t.Fatalf("config = %q, want [theme_palette] table", text)
	}
	if !strings.Contains(text, "accent = \"#A6FF00\"") {
		t.Fatalf("config = %q, want accent entry", text)
	}
	for _, key := range []string{"success = \"\"", "danger = \"\"", "muted = \"\"", "border = \"\""} {
		if strings.Contains(text, key) {
			t.Fatalf("config = %q, must omit %s", text, key)
		}
	}
}

func TestSaveRejectsInvalidThemeMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	err := Save(path, Settings{
		SessionSize:         10,
		ReviewRatio:         0.4,
		WriteModeDifficulty: WriteModeDifficultyBasic,
		AudioEnabled:        true,
		AudioAutoplay:       false,
		Language:            "ja",
		ThemeMode:           "bad",
	})
	if err == nil {
		t.Fatal("Save() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "theme_mode") {
		t.Fatalf("Save() error = %v, want theme_mode validation", err)
	}
}

func TestLoadRejectsInvalidThemeMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 10
review_ratio = 0.4
theme_mode = "bad"
language = "ja"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "theme_mode") {
		t.Fatalf("Load() error = %v, want theme_mode validation", err)
	}
}

func TestLoadRejectsInvalidThemePaletteColor(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
session_size = 10
review_ratio = 0.4
theme_mode = "custom"

[theme_palette]
accent = "#12"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "theme_palette.accent") {
		t.Fatalf("Load() error = %v, want theme_palette validation", err)
	}
}
