package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const appDirName = "eitango-cli"

type Paths struct {
	DataDir    string
	DBPath     string
	ConfigPath string
	LogsDir    string
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
