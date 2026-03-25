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
