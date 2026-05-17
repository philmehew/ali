package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/philmehew/ali/internal/models"
)

func TestAIConfigPath_EnvOverride(t *testing.T) {
	custom := "/tmp/ali-test/ai.yaml"
	t.Setenv("ALI_AI_CONFIG", custom)

	path, err := AIConfigPath()
	if err != nil {
		t.Fatalf("AIConfigPath: %v", err)
	}
	if path != custom {
		t.Errorf("expected %q, got %q", custom, path)
	}
}

func TestAIConfigPath_Default(t *testing.T) {
	t.Setenv("ALI_AI_CONFIG", "")

	dir, _ := os.UserConfigDir()
	expected := filepath.Join(dir, "ali", "ai.yaml")

	path, err := AIConfigPath()
	if err != nil {
		t.Fatalf("AIConfigPath: %v", err)
	}
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLoadAIConfig_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_AI_CONFIG", filepath.Join(tmpDir, "nonexistent.yaml"))
	t.Setenv("ALI_AI_API_KEY", "")
	t.Setenv("ALI_AI_URL", "")
	t.Setenv("ALI_AI_MODEL", "")
	t.Setenv("ALI_AI_CONTEXT_WINDOW", "")

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig: %v", err)
	}
	if cfg.URL != "https://api.openai.com/v1" {
		t.Errorf("expected default URL, got %q", cfg.URL)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("expected default model, got %q", cfg.Model)
	}
	if cfg.APIKey != "" {
		t.Errorf("expected empty API key, got %q", cfg.APIKey)
	}
	if cfg.ContextWindow != 128000 {
		t.Errorf("expected default context window 128000, got %d", cfg.ContextWindow)
	}
	if cfg.ChatPrompt != "You are a helpful assistant." {
		t.Errorf("expected default chat prompt, got %q", cfg.ChatPrompt)
	}
}

func TestLoadAIConfig_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_AI_CONFIG", filepath.Join(tmpDir, "ai.yaml"))
	t.Setenv("ALI_AI_API_KEY", "env-key")
	t.Setenv("ALI_AI_URL", "https://custom.api.com/v1")
	t.Setenv("ALI_AI_MODEL", "custom-model")
	t.Setenv("ALI_AI_CONTEXT_WINDOW", "32000")

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected API key from env, got %q", cfg.APIKey)
	}
	if cfg.URL != "https://custom.api.com/v1" {
		t.Errorf("expected URL from env, got %q", cfg.URL)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("expected model from env, got %q", cfg.Model)
	}
	if cfg.ContextWindow != 32000 {
		t.Errorf("expected context window from env, got %d", cfg.ContextWindow)
	}
}

func TestLoadAIConfig_FileValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ai.yaml")
	t.Setenv("ALI_AI_CONFIG", configPath)
	t.Setenv("ALI_AI_API_KEY", "")
	t.Setenv("ALI_AI_URL", "")
	t.Setenv("ALI_AI_MODEL", "")

	original := &models.AIConfig{
		URL:           "https://my-server.com/v1",
		APIKey:        "file-key",
		Model:         "my-model",
		AiPrompt:      "custom ai prompt",
		ChatPrompt:    "custom chat prompt",
		ContextWindow: 64000,
	}
	if err := SaveAIConfig(original); err != nil {
		t.Fatalf("SaveAIConfig: %v", err)
	}

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig: %v", err)
	}
	if cfg.URL != "https://my-server.com/v1" {
		t.Errorf("expected URL from file, got %q", cfg.URL)
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("expected API key from file, got %q", cfg.APIKey)
	}
	if cfg.Model != "my-model" {
		t.Errorf("expected model from file, got %q", cfg.Model)
	}
	if cfg.AiPrompt != "custom ai prompt" {
		t.Errorf("expected ai prompt from file, got %q", cfg.AiPrompt)
	}
	if cfg.ChatPrompt != "custom chat prompt" {
		t.Errorf("expected chat prompt from file, got %q", cfg.ChatPrompt)
	}
	if cfg.ContextWindow != 64000 {
		t.Errorf("expected context window from file, got %d", cfg.ContextWindow)
	}
}

func TestLoadAIConfig_EnvOverridesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ai.yaml")
	t.Setenv("ALI_AI_CONFIG", configPath)
	t.Setenv("ALI_AI_API_KEY", "env-key-override")
	t.Setenv("ALI_AI_URL", "https://env-override.com/v1")
	t.Setenv("ALI_AI_MODEL", "env-model-override")
	t.Setenv("ALI_AI_CONTEXT_WINDOW", "96000")

	original := &models.AIConfig{
		URL:           "https://file-value.com/v1",
		APIKey:        "file-key",
		Model:         "file-model",
		ContextWindow: 64000,
	}
	if err := SaveAIConfig(original); err != nil {
		t.Fatalf("SaveAIConfig: %v", err)
	}

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig: %v", err)
	}
	if cfg.APIKey != "env-key-override" {
		t.Errorf("expected API key from env, got %q", cfg.APIKey)
	}
	if cfg.URL != "https://env-override.com/v1" {
		t.Errorf("expected URL from env, got %q", cfg.URL)
	}
	if cfg.Model != "env-model-override" {
		t.Errorf("expected model from env, got %q", cfg.Model)
	}
	if cfg.ContextWindow != 96000 {
		t.Errorf("expected context window from env, got %d", cfg.ContextWindow)
	}
}

func TestLoadAIConfig_DefaultsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ai.yaml")
	t.Setenv("ALI_AI_CONFIG", configPath)
	t.Setenv("ALI_AI_API_KEY", "")
	t.Setenv("ALI_AI_URL", "")
	t.Setenv("ALI_AI_MODEL", "")
	t.Setenv("ALI_AI_CONTEXT_WINDOW", "")

	// Save a config with only API key set.
	original := &models.AIConfig{APIKey: "test-key"}
	if err := SaveAIConfig(original); err != nil {
		t.Fatalf("SaveAIConfig: %v", err)
	}

	cfg, err := LoadAIConfig()
	if err != nil {
		t.Fatalf("LoadAIConfig: %v", err)
	}
	if cfg.URL != "https://api.openai.com/v1" {
		t.Errorf("expected default URL, got %q", cfg.URL)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("expected default model, got %q", cfg.Model)
	}
	if cfg.AiPrompt == "" {
		t.Error("expected default ai prompt to be set")
	}
	if cfg.ContextWindow != 128000 {
		t.Errorf("expected default context window 128000, got %d", cfg.ContextWindow)
	}
	if cfg.ChatPrompt != "You are a helpful assistant." {
		t.Errorf("expected default chat prompt, got %q", cfg.ChatPrompt)
	}
}

func TestSaveAIConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("ALI_AI_CONFIG", filepath.Join(tmpDir, "nested", "dir", "ai.yaml"))

	cfg := &models.AIConfig{URL: "https://example.com/v1"}
	if err := SaveAIConfig(cfg); err != nil {
		t.Fatalf("SaveAIConfig: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "nested", "dir", "ai.yaml")); os.IsNotExist(err) {
		t.Error("expected AI config file to be created in nested directory")
	}
}
