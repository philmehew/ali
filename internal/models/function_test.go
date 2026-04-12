package models

import (
	"testing"

	"gopkg.in/yaml.v3"
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
