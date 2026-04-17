package execution

import (
	"testing"

	"github.com/philmehew/ali/internal/models"
)

func TestResolve_WithDefaults(t *testing.T) {
	fn := &models.AliFunction{
		Name:     "glog",
		Body:     "git log --oneline -n $1",
		Defaults: []string{"10"},
	}

	resolved, err := Resolve(fn, []string{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Command != "git log --oneline -n 10" {
		t.Errorf("expected %q, got %q", "git log --oneline -n 10", resolved.Command)
	}
	if len(resolved.Extras) != 0 {
		t.Errorf("expected no extras, got %v", resolved.Extras)
	}
}

func TestResolve_OverrideDefault(t *testing.T) {
	fn := &models.AliFunction{
		Name:     "glog",
		Body:     "git log --oneline -n $1",
		Defaults: []string{"10"},
	}

	resolved, err := Resolve(fn, []string{"20"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Command != "git log --oneline -n 20" {
		t.Errorf("expected %q, got %q", "git log --oneline -n 20", resolved.Command)
	}
}

func TestResolve_MissingRequired(t *testing.T) {
	fn := &models.AliFunction{
		Name: "glog",
		Body: "git log --oneline -n $1",
	}

	_, err := Resolve(fn, []string{})
	if err == nil {
		t.Fatal("expected error for missing required parameter")
	}
}

func TestResolve_ExtraArgs(t *testing.T) {
	fn := &models.AliFunction{
		Name:     "test",
		Body:     "echo $1",
		Defaults: []string{"hello"},
	}

	resolved, err := Resolve(fn, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Command != "echo a" {
		t.Errorf("expected %q, got %q", "echo a", resolved.Command)
	}
	if len(resolved.Extras) != 2 || resolved.Extras[0] != "b" || resolved.Extras[1] != "c" {
		t.Errorf("expected extras [b c], got %v", resolved.Extras)
	}
}

func TestResolve_MultiplePlaceholders(t *testing.T) {
	fn := &models.AliFunction{
		Name:     "mygrep",
		Body:     "grep -r \"$1\" \"$2\"",
		Defaults: []string{"", "."},
	}

	resolved, err := Resolve(fn, []string{"TODO"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Command != "grep -r \"TODO\" \".\"" {
		t.Errorf("expected %q, got %q", "grep -r \"TODO\" \".\"", resolved.Command)
	}
}

func TestResolve_NoPlaceholders(t *testing.T) {
	fn := &models.AliFunction{
		Name: "hello",
		Body: "echo hello",
	}

	resolved, err := Resolve(fn, []string{"extra1", "extra2"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Command != "echo hello" {
		t.Errorf("expected %q, got %q", "echo hello", resolved.Command)
	}
	if len(resolved.Extras) != 2 {
		t.Errorf("expected 2 extras, got %d", len(resolved.Extras))
	}
}

func TestSubstitute_Ordering(t *testing.T) {
	body := "$10 and $1"
	params := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

	result := substitute(body, params)
	if result != "j and a" {
		t.Errorf("expected %q, got %q", "j and a", result)
	}
}

func TestCommandString(t *testing.T) {
	tests := []struct {
		name     string
		resolved *ResolvedCommand
		want     string
	}{
		{
			name:     "simple command",
			resolved: &ResolvedCommand{Command: "git log --oneline -n 10"},
			want:     "git log --oneline -n 10",
		},
		{
			name:     "with extras",
			resolved: &ResolvedCommand{Command: "echo hello", Extras: []string{"world", "foo bar"}},
			want:     "echo hello 'world' 'foo bar'",
		},
		{
			name:     "with special chars in extras",
			resolved: &ResolvedCommand{Command: "echo", Extras: []string{"it's"}},
			want:     "echo 'it'\\''s'",
		},
		{
			name:     "no extras",
			resolved: &ResolvedCommand{Command: "git status"},
			want:     "git status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommandString(tt.resolved)
			if got != tt.want {
				t.Errorf("CommandString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShellArgEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}

	for _, tt := range tests {
		got := shellArgEscape(tt.input)
		if got != tt.expected {
			t.Errorf("shellArgEscape(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMaxPlaceholderIndex(t *testing.T) {
	tests := []struct {
		body string
		want int
	}{
		{"echo $1", 1},
		{"$1 $2 $3", 3},
		{"no placeholders", 0},
		{"$10 and $1", 10},
		{"$99", 99},
	}

	for _, tt := range tests {
		got := maxPlaceholderIndex(tt.body)
		if got != tt.want {
			t.Errorf("maxPlaceholderIndex(%q) = %d, want %d", tt.body, got, tt.want)
		}
	}
}
