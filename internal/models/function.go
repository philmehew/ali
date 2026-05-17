// Package models defines the data structures for ali function configuration.
package models

import (
	"fmt"
	"regexp"
)

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

var validNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// ValidateFunctionName checks that a function name is well-formed.
// Names must start with a letter or underscore and contain only
// letters, digits, underscores, and hyphens.
func ValidateFunctionName(name string) error {
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("function name %q is invalid: must start with a letter or underscore and contain only letters, digits, underscores, and hyphens", name)
	}
	return nil
}
