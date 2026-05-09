// Package config handles loading, saving, and querying the ali YAML configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philmehew/ali/internal/models"
	"gopkg.in/yaml.v3"
)

// Path returns the path to the ali config file.
// It checks the ALI_CONFIG env var first, then falls back to
// os.UserConfigDir()/ali/functions.yaml.
func Path() (string, error) {
	if p := os.Getenv("ALI_CONFIG"); p != "" {
		return p, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config directory: %w", err)
	}

	return filepath.Join(dir, "ali", "functions.yaml"), nil
}

// Load reads the config file. If it doesn't exist, it returns an empty config.
func Load() (*models.AliConfig, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) //nolint:gosec // G304: path is from trusted config resolution.
	if err != nil {
		if os.IsNotExist(err) {
			return &models.AliConfig{Functions: []models.AliFunction{}}, nil
		}
		return nil, fmt.Errorf("could not read config: %w", err)
	}

	var cfg models.AliConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk atomically.
func Save(cfg *models.AliConfig) error {
	path, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	// Write to temp file in the same directory, then rename for atomicity.
	tmp, err := os.CreateTemp(dir, "ali-config-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("could not write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("could not close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("could not rename temp file: %w", err)
	}

	return nil
}

// FindFunction looks up a function by name. Returns nil if not found.
func FindFunction(cfg *models.AliConfig, name string) *models.AliFunction {
	for i := range cfg.Functions {
		if cfg.Functions[i].Name == name {
			return &cfg.Functions[i]
		}
	}
	return nil
}

// FindFunctionIndex returns the index of a function by name, or -1 if not found.
func FindFunctionIndex(cfg *models.AliConfig, name string) int {
	for i, fn := range cfg.Functions {
		if fn.Name == name {
			return i
		}
	}
	return -1
}

// MoveFunction moves the function at fromIdx to toIdx (both 0-based) and saves.
func MoveFunction(cfg *models.AliConfig, fromIdx, toIdx int) {
	fn := cfg.Functions[fromIdx]
	cfg.Functions = append(cfg.Functions[:fromIdx], cfg.Functions[fromIdx+1:]...)
	cfg.Functions = append(cfg.Functions[:toIdx], append([]models.AliFunction{fn}, cfg.Functions[toIdx:]...)...)
}
