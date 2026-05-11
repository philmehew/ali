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

	// Check if ali is already in the rc file.
	data, err := os.ReadFile(rcPath) //nolint:gosec // G304: rcPath is from trusted home directory detection
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not read %s: %w", rcPath, err)
	}

	marker := "ali init"
	if strings.Contains(string(data), marker) {
		outputf("ali is already configured in %s\n", rcPath)
		return nil
	}

	// Resolve the full path to the ali binary so the eval line works
	// without ali being on PATH.
	exePath, err := executablePath()
	if err != nil {
		return fmt.Errorf("could not determine ali binary path: %w", err)
	}

	// Append the eval line with the full binary path.
	evalLine := fmt.Sprintf("\n# ali\neval \"$(%s init %s)\"\n", exePath, shell)
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) //nolint:gosec // G304: rcPath is from trusted home directory detection
	if err != nil {
		return fmt.Errorf("could not write to %s: %w", rcPath, err)
	}
	defer f.Close() //nolint:errcheck // file will be closed on defer

	if _, err := f.WriteString(evalLine); err != nil {
		return fmt.Errorf("could not write to %s: %w", rcPath, err)
	}

	outputf("Added ali to %s. Run `source %s` or restart your terminal.\n", rcPath, rcPath)
	return nil
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
