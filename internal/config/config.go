package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const appName = "adb-gui"

// Config holds user preferences and paths persisted to disk.
type Config struct {
	ADBPath    string `json:"adb_path"`
	LastDevice string `json:"last_device,omitempty"`
	ThemeMode  string `json:"theme_mode,omitempty"` // "system" (default), "light", "dark"
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appName), nil
}

// Path returns the full path to the JSON config file.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config file if present, otherwise returns defaults.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No config yet; return defaults
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to disk, creating the directory as needed.
func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	p := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}