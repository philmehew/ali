// Package main is the entry point for the ali CLI tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/philmehew/ali/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()

	// Build the set of known subcommands (including aliases) for argument interception.
	knownSubcommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		knownSubcommands[cmd.Name()] = true
		for _, alias := range cmd.Aliases {
			knownSubcommands[alias] = true
		}
	}
	// Cobra adds these lazily, so they aren't in Commands() at build time.
	knownSubcommands["help"] = true
	knownSubcommands["completion"] = true

	// Intercept args: if the first non-flag arg is not a known subcommand,
	// insert "run" before it so `ali glog 20` becomes `ali run glog 20`.
	os.Args = cli.InterceptArgs(os.Args, knownSubcommands)

	// Completion commands must output to stdout even when stdout is a pipe
	// (e.g. eval "$(ali completion zsh)"). Force the output writer to stdout
	// for completion subcommands so scripts can capture the output.
	if isCompletionCommand(os.Args) {
		rootCmd.SetOut(os.Stdout)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// isCompletionCommand returns true if the args represent a completion command
// or a Cobra internal completion request.
func isCompletionCommand(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "completion" {
			return true
		}
		if len(arg) > 0 && arg[0] != '-' {
			break
		}
	}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "__complete") {
			return true
		}
	}
	return false
}
