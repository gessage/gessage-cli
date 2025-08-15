package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config stores the selected model and per-model configuration maps.
// Models[modelName] is an arbitrary string map owned by that model implementation.
type Config struct {
	SelectedModel string                       `json:"selected_model"`
	Models        map[string]map[string]string `json:"models"`
}

// Default returns an empty configuration.
func Default() *Config {
	return &Config{Models: map[string]map[string]string{}}
}

// Path returns the config file path, creating the directory if necessary.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(dir, "gessage")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(base, "config.json"), nil
}

// Load reads the config from disk. If missing, returns Default().
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Models == nil {
		cfg.Models = map[string]map[string]string{}
	}
	return &cfg, nil
}

// Save writes the config to disk atomically.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
