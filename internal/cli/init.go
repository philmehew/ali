package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/version"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Output shell configuration to add ali to PATH",
		Long: `Output shell configuration to add ali to your PATH.

If no shell is specified, it is auto-detected from the $SHELL environment variable.

Step 1: Add to your shell profile (pick one):
  echo 'export PATH="/path/to/ali:$PATH"' >> ~/.zshrc    # zsh
  echo 'export PATH="/path/to/ali:$PATH"' >> ~/.bashrc   # bash

Step 2: Reload your profile:
  source ~/.zshrc                          # zsh
  source ~/.bashrc                         # bash

Or simply restart your terminal.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell()
			}

			binDir, err := executableDir()
			if err != nil {
				return fmt.Errorf("could not determine ali binary location: %w", err)
			}

			switch shell {
			case "bash", "zsh":
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# ali %s: add to your shell profile (~/.%src):\n", version.Version, shell)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "export PATH=\"%s:$PATH\"\n\n", binDir)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 1: Add to your profile:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "echo 'export PATH=\"%s:$PATH\"' >> ~/.%src\n\n", binDir, shell)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 2: Reload your profile:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "source ~/.%src\n", shell)
			case "fish":
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# ali %s: add to your fish config (~/.config/fish/config.fish):\n", version.Version)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "set -gx PATH %s $PATH\n\n", binDir)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 1: Add to your config:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "echo 'set -gx PATH %s $PATH' >> ~/.config/fish/config.fish\n\n", binDir)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 2: Reload your config:")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "source ~/.config/fish/config.fish")
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# ali %s: unsupported shell %q, using POSIX syntax\n", version.Version, shell)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "export PATH=\"%s:$PATH\"\n\n", binDir)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 1: Add to your profile:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "echo 'export PATH=\"%s:$PATH\"' >> ~/.profile\n\n", binDir)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# Step 2: Reload your profile:")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "source ~/.profile")
			}

			return nil
		},
	}

	return cmd
}

// detectShell returns the shell name from the $SHELL environment variable.
func detectShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return "sh"
	}
	return strings.TrimPrefix(filepath.Base(shellPath), "-")
}

// executableDir returns the directory containing the running ali binary.
func executableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(resolved), nil
}
