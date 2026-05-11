package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCmd_Default(t *testing.T) {
	// ali install (without shell arg) auto-detects shell and appends to rc file.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/zsh")

	cmd := newInstallCmd()
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("install cmd: %v", err)
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

func TestInstallCmd_ExplicitShell(t *testing.T) {
	// ali install zsh appends eval line with full binary path to rc file.
	t.Setenv("HOME", t.TempDir())

	cmd := newInstallCmd()
	cmd.SetArgs([]string{"zsh"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("install cmd: %v", err)
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

func TestInstallCmd_Duplicate(t *testing.T) {
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
