package cli

import (
	"bytes"
	"io"
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
	// ali init (without --install) outputs shell code to stdout.
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

func TestInitCmd_InstallFlag(t *testing.T) {
	// ali init --install appends eval line with full binary path to rc file.
	t.Setenv("HOME", t.TempDir())

	cmd := newInitCmd()
	cmd.SetArgs([]string{"--install", "zsh"})
	cmd.SetOut(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init --install cmd: %v", err)
	}

	home, _ := os.UserHomeDir()
	rcFile := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(rcFile) //nolint:gosec // G304: test file path
	if err != nil {
		t.Fatalf("could not read rc file: %v", err)
	}

	content := string(data)
	// Should contain the full path to the ali binary, not just "ali".
	exePath, _ := executablePath()
	expectedEval := `eval "$(` + exePath + ` init zsh)"`
	if !strings.Contains(content, expectedEval) {
		t.Errorf("expected eval line %q in rc file, got: %q", expectedEval, content)
	}
}

func TestInitCmd_InstallZsh(t *testing.T) {
	// Test installShellConfig directly — it uses the full binary path.
	t.Setenv("HOME", t.TempDir())

	if err := installShellConfig(shellZsh); err != nil {
		t.Fatalf("installShellConfig: %v", err)
	}

	home, _ := os.UserHomeDir()
	rcFile := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(rcFile) //nolint:gosec // G304: test file path
	if err != nil {
		t.Fatalf("could not read rc file: %v", err)
	}

	content := string(data)
	exePath, _ := executablePath()
	expectedEval := `eval "$(` + exePath + ` init zsh)"`
	if !strings.Contains(content, expectedEval) {
		t.Errorf("expected eval line %q in rc file, got: %q", expectedEval, content)
	}
	if !strings.Contains(content, "# ali") {
		t.Errorf("expected # ali marker in rc file, got: %q", content)
	}
}

func TestInitCmd_InstallDuplicate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// First install
	if err := installShellConfig(shellZsh); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Second install should detect duplicate and not add again
	if err := installShellConfig(shellZsh); err != nil {
		t.Fatalf("second install: %v", err)
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
