package cli

import (
	"bytes"
	"os"
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

func TestInitCmd_Default(t *testing.T) {
	// ali init outputs shell code to stdout.
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
	if !strings.Contains(output, "print -z") {
		t.Errorf("expected zsh wrapper (print -z) in output, got: %q", output)
	}
}

func TestPrintShellConfig(t *testing.T) {
	binDir := "/test/bin"

	tests := []struct {
		shell       string
		wantContain string
	}{
		{shellZsh, "print -z"},
		{shellBash, "read -e -i"},
		{shellFish, "commandline --replace"},
		{"sh", "command ali"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			cmd := newInitCmd()
			var buf strings.Builder
			cmd.SetOut(&buf)

			if err := printShellConfig(cmd, tt.shell, binDir); err != nil {
				t.Fatalf("printShellConfig: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.wantContain) {
				t.Errorf("expected %q in output for %s, got: %q", tt.wantContain, tt.shell, output)
			}
			if !strings.Contains(output, binDir) {
				t.Errorf("expected binDir %q in output, got: %q", binDir, output)
			}
		})
	}
}

func TestRcFilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("could not determine home directory: %v", err)
	}

	tests := []struct {
		shell string
		want  string
	}{
		{shellZsh, filepath.Join(home, ".zshrc")},
		{shellBash, filepath.Join(home, ".bashrc")},
		{shellFish, filepath.Join(home, ".config", "fish", "config.fish")},
		{"sh", filepath.Join(home, ".profile")},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got, err := rcFilePath(tt.shell)
			if err != nil {
				t.Fatalf("rcFilePath(%q) error: %v", tt.shell, err)
			}
			if got != tt.want {
				t.Errorf("rcFilePath(%q) = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}
