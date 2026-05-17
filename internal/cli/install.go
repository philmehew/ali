package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [shell]",
		Short: "Add shell integration to your rc file",
		Long: `Add the ali eval line to your shell's rc file.
It auto-detects your shell from $SHELL if no argument is given.

  ali install        # auto-detect shell and add to rc file
  ali install zsh    # add to ~/.zshrc
  ali install bash   # add to ~/.bashrc
  ali install fish   # add to config.fish`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell()
			}

			return installShellConfig(shell)
		},
	}

	// Install output must go to stdout — user-facing messages like
	// "Added ali to ~/.zshrc" should be visible when run interactively.
	cmd.SetOut(os.Stdout)

	return cmd
}

// installShellConfig appends `eval "$(/path/to/ali init <shell>)"` to the shell's rc file.
// It uses the full path to the ali binary so the eval line works even if
// ali is not yet on PATH.
func installShellConfig(shell string) error {
	rcPath, err := rcFilePath(shell)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(rcPath) //nolint:gosec // G304: rcPath is from trusted home directory detection
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not read %s: %w", rcPath, err)
	}

	// Back up the existing rc file before modifying it.
	if len(data) > 0 {
		if err := backupFile(rcPath, data); err != nil {
			outputf("Warning: could not back up %s: %v\n", rcPath, err)
		}
	}

	// Resolve the full path to the ali binary so the eval line works
	// without ali being on PATH.
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("could not determine ali binary path: %w", err)
	}

	evalLine := fmt.Sprintf("eval \"$(%s init %s)\"", exePath, shell)
	compLine := completionInstallLine(exePath, shell)
	newBlock := fmt.Sprintf("\n# >>> ali >>>\n%s\n%s\n# <<< ali <<<\n", evalLine, compLine)

	// If the rc file already has an ali block, replace it.
	if len(data) > 0 && (strings.Contains(string(data), "# >>> ali >>>") || strings.Contains(string(data), "# ali")) {
		updated, replaced := replaceAliBlock(string(data), newBlock)
		if replaced {
			if err := os.WriteFile(rcPath, []byte(updated), 0600); err != nil { //nolint:gosec // G304: rcPath is from trusted home directory detection.
				return fmt.Errorf("could not write to %s: %w", rcPath, err)
			}
			outputf("Updated ali in %s. Run `source %s` or restart your terminal.\n", rcPath, rcPath)
			return nil
		}
	}

	// Append a new ali block.
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) //nolint:gosec // G304: rcPath is from trusted home directory detection
	if err != nil {
		return fmt.Errorf("could not write to %s: %w", rcPath, err)
	}
	defer f.Close() //nolint:errcheck // file will be closed on defer

	if _, err := f.WriteString(newBlock); err != nil {
		return fmt.Errorf("could not write to %s: %w", rcPath, err)
	}

	outputf("Added ali to %s. Run `source %s` or restart your terminal.\n", rcPath, rcPath)
	return nil
}

// replaceAliBlock finds the existing ali block in rcContent and replaces
// it with newBlock. It detects both the new-style markers (# >>> ali >>>)
// and the legacy "# ali" marker for backward compatibility.
// Returns the updated content and whether a replacement was made.
func replaceAliBlock(rcContent, newBlock string) (string, bool) {
	lines := strings.Split(rcContent, "\n")
	start := -1
	end := -1

	for i, line := range lines {
		// Prefer new-style marker
		if strings.HasPrefix(line, "# >>> ali >>>") {
			start = i
			// Find the closing marker
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "# <<< ali <<<") {
					end = j + 1
					break
				}
			}
			if end <= start {
				// No closing marker found — include until next non-ali line
				end = i + 1
				for j := i + 1; j < len(lines); j++ {
					trimmed := strings.TrimSpace(lines[j])
					if trimmed == "" || strings.Contains(lines[j], "init ") || strings.Contains(lines[j], "completion ") || strings.HasPrefix(lines[j], "# <<<") {
						end = j + 1
					} else {
						break
					}
				}
			}
			break
		}
		// Legacy marker
		if strings.HasPrefix(line, "# ali") && !strings.HasPrefix(line, "# alias") {
			start = i
			end = i + 1
			for j := i + 1; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "" || strings.Contains(lines[j], "init ") || strings.Contains(lines[j], "completion ") {
					end = j + 1
				} else {
					break
				}
			}
			break
		}
	}

	if start == -1 {
		return rcContent, false
	}

	replacementLines := strings.Split(strings.Trim(newBlock, "\n"), "\n")
	result := make([]string, 0, len(lines)-end+start+len(replacementLines))
	result = append(result, lines[:start]...)
	result = append(result, replacementLines...)
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n"), true
}

// completionInstallLine returns a shell-specific line to set up ali completions.
// Uses the full binary path so it works without ali being on PATH.
func completionInstallLine(exePath, shell string) string {
	switch shell {
	case shellZsh:
		// compinit must be called before compdef is available.
		return fmt.Sprintf("autoload -Uz compinit && compinit; eval \"$(%s completion zsh)\"", exePath)
	case shellBash:
		return fmt.Sprintf("source <(%s completion bash)", exePath)
	case shellFish:
		return fmt.Sprintf("%s completion fish | source", exePath)
	default:
		return ""
	}
}

// rcFilePath returns the path to the shell's rc file.
func rcFilePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	switch shell {
	case shellZsh:
		return filepath.Join(home, ".zshrc"), nil
	case shellBash:
		return filepath.Join(home, ".bashrc"), nil
	case shellFish:
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		return filepath.Join(home, ".profile"), nil
	}
}

// backupFile creates a backup of data at path + ".ali.bak".
func backupFile(path string, data []byte) error {
	backupPath := path + ".ali.bak"
	return os.WriteFile(backupPath, data, 0600) //nolint:gosec // G304: backupPath is derived from trusted rcPath
}
