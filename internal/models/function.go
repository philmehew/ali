// Package models defines the data structures for ali function configuration.
package models

// AliFunction represents a single parametric command snippet.
type AliFunction struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Body        string   `yaml:"body"`
	Defaults    []string `yaml:"defaults,omitempty"`
}

// AliConfig is the root YAML document containing all functions.
type AliConfig struct {
	Functions []AliFunction `yaml:"functions"`
	Ignore    []string      `yaml:"ignore,omitempty"`
}
