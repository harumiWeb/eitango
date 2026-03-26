package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/yourname/eitango/internal/i18n"
	"github.com/yourname/eitango/internal/session"
)

const appDirName = "eitango-cli"

type Paths struct {
	DataDir    string
	DBPath     string
	ConfigPath string
	LogsDir    string
}

type Settings struct {
	SessionSize      int
	ReviewRatio      float64
	FocusModeDefault bool
	Language         string
}

type fileSettings struct {
	SessionSize      *int     `toml:"session_size"`
	ReviewRatio      *float64 `toml:"review_ratio"`
	FocusModeDefault *bool    `toml:"focus_mode_default"`
	Language         *string  `toml:"language"`
}

func DefaultSettings() Settings {
	return Settings{
		SessionSize: session.DefaultQuestionCount,
		ReviewRatio: session.DefaultReviewRatio,
		Language:    i18n.DefaultLang,
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
		if *raw.SessionSize <= 0 {
			return Settings{}, fmt.Errorf("session_size must be greater than 0")
		}
		settings.SessionSize = *raw.SessionSize
	}
	if raw.ReviewRatio != nil {
		if math.IsNaN(*raw.ReviewRatio) || *raw.ReviewRatio < 0 || *raw.ReviewRatio > 1 {
			return Settings{}, fmt.Errorf("review_ratio must be between 0 and 1")
		}
		settings.ReviewRatio = *raw.ReviewRatio
	}
	if raw.FocusModeDefault != nil {
		settings.FocusModeDefault = *raw.FocusModeDefault
	}
	if raw.Language != nil {
		if !i18n.ValidLang(*raw.Language) {
			return Settings{}, fmt.Errorf("unsupported language: %q (use %q or %q)", *raw.Language, i18n.LangJA, i18n.LangEN)
		}
		settings.Language = *raw.Language
	}

	return settings, nil
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
