// Package cli implements the ali command-line interface using Cobra.
package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates and returns the root cobra command for ali.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ali",
		Short: "Manage and execute parametric command-line snippets",
		Long:  "ali is a CLI tool for managing and executing command-line snippets with parameter support.",
	}

	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newHistoryCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newRemoveCmd())
	rootCmd.AddCommand(newEditCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newVersionCmd())

	// Hide the auto-generated completion command from help output.
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	return rootCmd
}

// InterceptArgs rewrites os.Args so that if the first non-flag argument
// is not a known subcommand, "run" is inserted before it.
// This allows `ali glog 20` to work as shorthand for `ali run glog 20`.
func InterceptArgs(args []string, knownSubcommands map[string]bool) []string {
	if len(args) <= 1 {
		return args
	}

	// Skip the program name.
	for i := 1; i < len(args); i++ {
		if args[i][0] == '-' {
			// Skip flag and its value if it takes one.
			continue
		}
		if knownSubcommands[args[i]] {
			return args
		}
		// Not a known subcommand — insert "run" before this arg.
		newArgs := make([]string, 0, len(args)+1)
		newArgs = append(newArgs, args[:i]...)
		newArgs = append(newArgs, "run")
		newArgs = append(newArgs, args[i:]...)
		return newArgs
	}
	return args
}
