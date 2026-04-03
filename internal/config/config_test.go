package config

import (
	"os"
	"path/filepath"
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
	if settings != DefaultSettings() {
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
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save error = %v", err)
	}
	if got != want {
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
}
