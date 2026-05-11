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
	cmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Output shell integration code for ali",
		Long: `Output shell integration code so resolved commands are pasted into
your command line for editing before execution.

Add to your shell's rc file:
  eval "$(ali init zsh)"    # for zsh
  eval "$(ali init bash)"   # for bash
  ali init fish | source    # for fish

To add this line to your rc file automatically:
  ali install        # auto-detects shell
  ali install zsh    # specify shell`,
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
			return printShellConfig(cmd, shell, binDir)
		},
	}

	// Init output must always go to stdout — it's captured by eval "$(ali init)"
	// in rc files, where stdout is a pipe (not a tty). The root command routes
	// output through displayOut (which becomes stderr when stdout is a pipe),
	// which is correct for interactive commands but breaks eval capture.
	cmd.SetOut(os.Stdout)

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
