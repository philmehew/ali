// Package main is the entry point for the ali CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/philmehew/ali/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()

	// Build the set of known subcommands for argument interception.
	knownSubcommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		knownSubcommands[cmd.Name()] = true
	}

	// Intercept args: if the first non-flag arg is not a known subcommand,
	// insert "run" before it so `ali glog 20` becomes `ali run glog 20`.
	os.Args = cli.InterceptArgs(os.Args, knownSubcommands)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
