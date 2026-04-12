package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		shellEnv string
		want     string
	}{
		{"/bin/zsh", "zsh"},
		{"/bin/bash", "bash"},
		{"/usr/bin/fish", "fish"},
		{"/bin/-zsh", "zsh"}, // handles leading dash
		{"", "sh"},           // empty falls back
	}

	for _, tt := range tests {
		t.Run(tt.shellEnv, func(t *testing.T) {
			t.Setenv("SHELL", tt.shellEnv)
			got := detectShell()
			if got != tt.want {
				t.Errorf("detectShell() with SHELL=%q = %q, want %q", tt.shellEnv, got, tt.want)
			}
		})
	}
}

func TestExecutableDir(t *testing.T) {
	dir, err := executableDir()
	if err != nil {
		t.Fatalf("executableDir() error: %v", err)
	}
	if dir == "" {
		t.Error("executableDir() returned empty string")
	}
	// Should be an absolute path.
	if !filepath.IsAbs(dir) {
		t.Errorf("executableDir() returned relative path: %q", dir)
	}
}

func TestInitCmd_ZshOutput(t *testing.T) {
	cmd := newInitCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"zsh"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init cmd: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "export PATH=") {
		t.Errorf("expected export PATH in output, got: %q", output)
	}
	if !strings.Contains(output, ":$PATH") {
		t.Errorf("expected :$PATH suffix in output, got: %q", output)
	}
}

func TestInitCmd_FishOutput(t *testing.T) {
	cmd := newInitCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"fish"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init cmd: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "set -gx PATH") {
		t.Errorf("expected fish syntax in output, got: %q", output)
	}
}

func TestInitCmd_AutoDetect(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	cmd := newInitCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{}) // no shell arg — auto-detect

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init cmd: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "export PATH=") {
		t.Errorf("expected export PATH in auto-detected output, got: %q", output)
	}
}
