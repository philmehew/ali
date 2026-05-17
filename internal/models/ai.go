// Package models defines the data structures for ali configuration.
package models

// AIConfig holds the AI service configuration loaded from ai.yaml.
type AIConfig struct {
	URL           string `yaml:"url"`
	APIKey        string `yaml:"api_key"`
	Model         string `yaml:"model"`
	AiPrompt      string `yaml:"ai_prompt"`
	ChatPrompt    string `yaml:"chat_prompt"`
	ContextWindow int    `yaml:"context_window_tokens"`
}
