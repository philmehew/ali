package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCmd_Default(t *testing.T) {
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
	exePath, _ := executablePath()
	expectedEval := `eval "$(` + exePath + ` init zsh)"`
	if !strings.Contains(content, expectedEval) {
		t.Errorf("expected eval line %q in rc file, got: %q", expectedEval, content)
	}
	expectedComp := `eval "$(` + exePath + ` completion zsh)"`
	if !strings.Contains(content, expectedComp) {
		t.Errorf("expected completion line %q in rc file, got: %q", expectedComp, content)
	}
}

func TestInstallCmd_ExplicitShell(t *testing.T) {
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

	if err := installShellConfig(shellZsh); err != nil {
		t.Fatalf("first install: %v", err)
	}

	if err := installShellConfig(shellZsh); err != nil {
		t.Fatalf("second install: %v", err)
	}

	home, _ := os.UserHomeDir()
	rcFile := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(rcFile) //nolint:gosec // G304: test file path
	if err != nil {
		t.Fatalf("could not read rc file: %v", err)
	}

	content := string(data)
	count := strings.Count(content, "# ali")
	if count != 1 {
		t.Errorf("expected exactly 1 ali block, found %d in: %q", count, content)
	}
}

func TestCompletionInstallLine(t *testing.T) {
	tests := []struct {
		shell   string
		wantSub string
	}{
		{shellZsh, "completion zsh)"},
		{shellBash, "completion bash)"},
		{shellFish, "completion fish | source"},
		{"sh", ""},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got := completionInstallLine("/usr/local/bin/ali", tt.shell)
			if tt.wantSub == "" {
				if got != "" {
					t.Errorf("completionInstallLine(%q) = %q, want empty", tt.shell, got)
				}
			} else if !strings.Contains(got, tt.wantSub) {
				t.Errorf("completionInstallLine(%q) = %q, want containing %q", tt.shell, got, tt.wantSub)
			}
			if tt.wantSub != "" && !strings.Contains(got, "/usr/local/bin/ali") {
				t.Errorf("completionInstallLine should use full binary path, got %q", got)
			}
		})
	}
}

func TestReplaceAliBlock(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		newBlock string
		want     string
		replaced bool
	}{
		{
			name:     "replaces two-line block",
			existing: "some rc content\n# ali\neval \"$(/old/path init zsh)\"\nmore content\n",
			newBlock: "\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\n",
			want:     "some rc content\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\nmore content\n",
			replaced: true,
		},
		{
			name:     "replaces three-line block with completion",
			existing: "some rc content\n# ali\neval \"$(/old/path init zsh)\"\neval \"$(/old/path completion zsh)\"\nmore content\n",
			newBlock: "\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\n",
			want:     "some rc content\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\nmore content\n",
			replaced: true,
		},
		{
			name:     "no ali block found",
			existing: "some rc content\nno ali here\n",
			newBlock: "\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\n",
			want:     "some rc content\nno ali here\n",
			replaced: false,
		},
		{
			name:     "replaces block at end of file",
			existing: "some rc content\n# ali\neval \"$(/old/path init zsh)\"\n",
			newBlock: "\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"\n",
			want:     "some rc content\n# ali\neval \"$(/new/path init zsh)\"\neval \"$(/new/path completion zsh)\"",
			replaced: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, replaced := replaceAliBlock(tt.existing, tt.newBlock)
			if replaced != tt.replaced {
				t.Errorf("replaceAliBlock replaced = %v, want %v", replaced, tt.replaced)
			}
			if got != tt.want {
				t.Errorf("replaceAliBlock got:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
