package models

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestYAMLRoundTrip(t *testing.T) {
	original := AliConfig{
		Functions: []AliFunction{
			{
				Name:        "glog",
				Description: "Pretty git log",
				Body:        "git log --oneline -n $1",
				Defaults:    []string{"10"},
			},
			{
				Name:        "mygrep",
				Description: "Recursive search",
				Body:        "grep -r \"$1\" \"$2\"",
				Defaults:    []string{"", "."},
			},
		},
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded AliConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Functions) != len(original.Functions) {
		t.Fatalf("expected %d functions, got %d", len(original.Functions), len(loaded.Functions))
	}

	for i, fn := range loaded.Functions {
		orig := original.Functions[i]
		if fn.Name != orig.Name {
			t.Errorf("function %d: expected name %q, got %q", i, orig.Name, fn.Name)
		}
		if fn.Body != orig.Body {
			t.Errorf("function %d: expected body %q, got %q", i, orig.Body, fn.Body)
		}
		if fn.Description != orig.Description {
			t.Errorf("function %d: expected description %q, got %q", i, orig.Description, fn.Description)
		}
		if len(fn.Defaults) != len(orig.Defaults) {
			t.Errorf("function %d: expected %d defaults, got %d", i, len(orig.Defaults), len(fn.Defaults))
		}
	}
}

func TestEmptyDefaultRoundTrip(t *testing.T) {
	cfg := AliConfig{
		Functions: []AliFunction{
			{
				Name:     "test",
				Body:     "echo $1",
				Defaults: []string{""},
			},
		},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded AliConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Functions[0].Defaults) != 1 {
		t.Fatalf("expected 1 default, got %d", len(loaded.Functions[0].Defaults))
	}
	if loaded.Functions[0].Defaults[0] != "" {
		t.Errorf("expected empty string default, got %q", loaded.Functions[0].Defaults[0])
	}
}

func TestNilDefaultsRoundTrip(t *testing.T) {
	cfg := AliConfig{
		Functions: []AliFunction{
			{
				Name: "test",
				Body: "echo $1",
			},
		},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded AliConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Functions[0].Defaults) != 0 {
		t.Errorf("expected 0 defaults, got %d", len(loaded.Functions[0].Defaults))
	}
}

func TestValidateFunctionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "glog", false},
		{"valid underscore", "my_func", false},
		{"valid hyphen", "a-b", false},
		{"valid starts underscore", "_private", false},
		{"valid numbers after first", "glog2", false},
		{"invalid spaces", "hello world", true},
		{"invalid semicolon", "a;b", true},
		{"invalid pipe", "a|b", true},
		{"invalid exclamation", "name!", true},
		{"invalid number only", "123", true},
		{"invalid empty", "", true},
		{"invalid starts with number", "1abc", true},
		{"invalid shell injection", "rm -rf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFunctionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFunctionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
