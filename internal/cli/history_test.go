package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const testCmdGitStatus = "git status"

func TestRemoveFromCandidates_NoMutation(t *testing.T) {
	original := []cmdFreq{
		{command: testCmdGitStatus, count: 5},
		{command: "ls", count: 3},
		{command: "cd ..", count: 2},
	}

	result := removeFromCandidates(original, 1)

	// Result should not contain the removed element.
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[0].command != testCmdGitStatus {
		t.Errorf("result[0] = %q, want %q", result[0].command, testCmdGitStatus)
	}
	if result[1].command != "cd .." {
		t.Errorf("result[1] = %q, want %q", result[1].command, "cd ..")
	}

	// Original should be unmodified.
	if original[1].command != "ls" {
		t.Errorf("original slice mutated: original[1] = %q, want %q", original[1].command, "ls")
	}
}

func TestHistoryPath_ShellDetection(t *testing.T) {
	tests := []struct {
		name    string
		shell   string
		wantDir string
	}{
		{"zsh", "/bin/zsh", ".zsh_history"},
		{"bash", "/bin/bash", ".bash_history"},
		{"fish", "/usr/bin/fish", ".zsh_history"}, // default fallback
		{"zsh with version", "/usr/local/bin/zsh-5.9", ".zsh_history"},
		{"bash in unusual path", "/home/user/.nix-profile/bin/bash", ".bash_history"},
		{"empty shell", "", ".zsh_history"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shell)
			t.Setenv("HISTFILE", "") // Clear any override

			home := t.TempDir()
			t.Setenv("HOME", home)

			path, err := historyPath()
			if err != nil {
				t.Fatalf("historyPath() error: %v", err)
			}
			if filepath.Base(path) != tt.wantDir {
				t.Errorf("historyPath() = %q, want base %q", path, tt.wantDir)
			}
		})
	}
}

func TestHistoryPath_HISTFILE(t *testing.T) {
	t.Setenv("HISTFILE", "/custom/history/file")
	t.Setenv("HOME", t.TempDir())

	path, err := historyPath()
	if err != nil {
		t.Fatalf("historyPath() error: %v", err)
	}
	if path != "/custom/history/file" {
		t.Errorf("historyPath() = %q, want /custom/history/file", path)
	}
}

func TestParseHistory_BasicLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bash_history")
	content := "git status\nls\ncd ..\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cmds, err := parseHistory(path, 10)
	if err != nil {
		t.Fatalf("parseHistory() error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}
	if cmds[0] != testCmdGitStatus {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], testCmdGitStatus)
	}
}

func TestParseHistory_LimitLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bash_history")
	// Write 5 lines, only read last 3.
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cmds, err := parseHistory(path, 3)
	if err != nil {
		t.Fatalf("parseHistory() error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}
	if cmds[0] != "line3" {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], "line3")
	}
}

func TestParseHistory_ZshMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zsh_history")
	content := ": 1234567890:0;git status\n: 1234567891:0;ls -la\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cmds, err := parseHistory(path, 10)
	if err != nil {
		t.Fatalf("parseHistory() error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0] != testCmdGitStatus {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], testCmdGitStatus)
	}
}

func TestParseHistory_ContinuationLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".bash_history")
	// Backslash continuation: "echo hello \" + "world" = "echo hello  world"
	// (the backslash is replaced with a space, joining the two lines)
	content := "echo hello \\\nworld\nls\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cmds, err := parseHistory(path, 10)
	if err != nil {
		t.Fatalf("parseHistory() error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(cmds), cmds)
	}
	// The continuation joins lines with a space where the backslash was.
	if cmds[0] != "echo hello  world" {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], "echo hello  world")
	}
}
