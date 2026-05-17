package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/philmehew/ali/internal/models"
	"go.yaml.in/yaml/v3"
)

const (
	defaultAIURL           = "https://api.openai.com/v1"
	defaultAIModel         = "gpt-4o-mini"
	defaultAICmdPrompt     = "You are a command-line expert. The user will describe what they want to do. Respond with ONLY the shell command or commands needed — nothing else. No explanations, no markdown, no comments. Chain multiple commands with && only. If there is no suitable command, respond with: no commands found"
	defaultAIChatPrompt    = "You are a helpful assistant."
	defaultAIContextWindow = 128000
)

// AIConfigPath returns the path to the AI config file.
// It checks the ALI_AI_CONFIG env var first, then falls back to
// os.UserConfigDir()/ali/ai.yaml.
func AIConfigPath() (string, error) {
	if p := os.Getenv("ALI_AI_CONFIG"); p != "" {
		return p, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config directory: %w", err)
	}

	return filepath.Join(dir, "ali", "ai.yaml"), nil
}

// LoadAIConfig reads the AI config file. If it doesn't exist, it returns
// a config with defaults. Environment variables take priority over file values.
func LoadAIConfig() (*models.AIConfig, error) {
	path, err := AIConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &models.AIConfig{}

	data, err := os.ReadFile(path) //nolint:gosec // G304: path is from trusted config resolution.
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("could not read AI config: %w", err)
		}
		// File doesn't exist — start with empty config, defaults applied below.
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("could not parse AI config: %w", err)
		}
	}

	applyAIDefaults(cfg)
	overrideAIFromEnv(cfg)

	return cfg, nil
}

// SaveAIConfig writes the AI config to disk atomically.
// The file is set to 0600 permissions to protect the API key.
func SaveAIConfig(cfg *models.AIConfig) error {
	path, err := AIConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg) //nolint:gosec // G117: api_key is a config credential, not a hardcoded secret
	if err != nil {
		return fmt.Errorf("could not marshal AI config: %w", err)
	}

	return atomicWrite(path, data, 0600)
}

// DefaultAIPrompt returns the default system prompt used by ali ai.
func DefaultAIPrompt() string {
	return defaultAICmdPrompt
}

// DefaultAIChatPrompt returns the default system prompt used by ali chat.
func DefaultAIChatPrompt() string {
	return defaultAIChatPrompt
}

// DefaultAIURL returns the default AI service URL.
func DefaultAIURL() string {
	return defaultAIURL
}

// DefaultAIModel returns the default AI model name.
func DefaultAIModel() string {
	return defaultAIModel
}

// DefaultAIContextWindow returns the default context window size in tokens.
func DefaultAIContextWindow() int {
	return defaultAIContextWindow
}

// applyAIDefaults fills in default values for empty fields.
func applyAIDefaults(cfg *models.AIConfig) {
	if cfg.URL == "" {
		cfg.URL = defaultAIURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultAIModel
	}
	if cfg.AiPrompt == "" {
		cfg.AiPrompt = defaultAICmdPrompt
	}
	if cfg.ChatPrompt == "" {
		cfg.ChatPrompt = defaultAIChatPrompt
	}
	if cfg.ContextWindow == 0 {
		cfg.ContextWindow = defaultAIContextWindow
	}
}

// overrideAIFromEnv replaces config values with environment variables when set.
func overrideAIFromEnv(cfg *models.AIConfig) {
	if v := os.Getenv("ALI_AI_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("ALI_AI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("ALI_AI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("ALI_AI_CONTEXT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ContextWindow = n
		}
	}
}
