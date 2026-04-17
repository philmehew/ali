package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	shellZsh  = "zsh"
	shellBash = "bash"
	shellFish = "fish"
)

func newInitCmd() *cobra.Command {
	var install bool

	cmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Output shell integration code for ali",
		Long: `Output shell integration code so resolved commands are pasted into
your command line for editing before execution.

Add to your shell's rc file:
  eval "$(ali init zsh)"    # for zsh
  eval "$(ali init bash)"   # for bash
  ali init fish | source    # for fish

To automatically add this line to your rc file:
  ali init --install        # auto-detects shell
  ali init --install zsh    # specify shell`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell()
			}

			if install {
				return installShellConfig(shell)
			}

			binDir, err := executableDir()
			if err != nil {
				return fmt.Errorf("could not determine ali binary location: %w", err)
			}
			return printShellConfig(cmd, shell, binDir)
		},
	}

	cmd.Flags().BoolVar(&install, "install", false, "append eval line to shell rc file")

	return cmd
}

// printShellConfig outputs the eval-able shell configuration.
// This is called when ali init is evaluated from the rc file.
func printShellConfig(cmd *cobra.Command, shell string, binDir string) error {
	out := cmd.OutOrStdout()

	wrapper := posixWrapper()
	switch shell {
	case shellZsh:
		wrapper = zshWrapper()
	case shellBash:
		wrapper = bashWrapper()
	case shellFish:
		wrapper = fishWrapper()
	}

	switch shell {
	case shellFish:
		_, _ = fmt.Fprintf(out, "set -gx PATH %s $PATH\n\n%s\n", binDir, wrapper)
	default:
		_, _ = fmt.Fprintf(out, "export PATH=\"%s:$PATH\"\n\n%s\n", binDir, wrapper)
	}

	return nil
}

// installShellConfig appends `eval "$(/path/to/ali init <shell>)"` to the shell's rc file.
// It uses the full path to the ali binary so the eval line works even if
// ali is not yet on PATH.
func installShellConfig(shell string) error {
	rcPath, err := rcFilePath(shell)
	if err != nil {
		return err
	}

	// Check if ali init is already in the rc file.
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
	exe, err := executablePath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// executablePath returns the resolved absolute path to the running ali binary.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// Shell wrapper functions.
// These are output by `ali init <shell>` and eval'd at shell startup.
// They enable pasting resolved commands into the shell's input buffer.

func zshWrapper() string {
	return `ali() { print -z -- "$(command ali "$@")"; }`
}

func bashWrapper() string {
	return `ali() { local _out; _out=$(command ali "$@"); if [[ -n "$_out" ]]; then if [[ -t 0 ]] && [[ ${BASH_VERSINFO[0]} -ge 4 ]]; then read -e -i "$_out" -p ""; eval -- "$REPLY"; else printf '%s\n' "$_out"; fi; fi; }`
}

func fishWrapper() string {
	return `function ali; set -l _out (command ali $argv); test -n "$_out"; and commandline --replace -- $_out; end`
}

func posixWrapper() string {
	// POSIX shells don't have print -z or commandline, so just alias to the binary.
	// The resolved command will be printed to stdout for the user to copy.
	return `ali() { command ali "$@"; }`
}
