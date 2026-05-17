// Package cli implements the ali command-line interface using Cobra.
package cli

import (
	"fmt"
	"strings"

	"github.com/philmehew/ali/internal/config"
	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/version"
)

const aliLogo = `
                      -#######-
                    #############
                  ################
                  #################
                  #################
                  ##++++#---+++####
                   ++++-----++++##-
                   ++#+---##++++#--
                   +++++---+---+++
                    ++-+------+++++++++.
              ++###  ++++----++++++++--+++++
           -+++#####+#+++++++++++++#######++++++
          -+#########+#++++-++++++##########+-++++
         -##########--++++++++++#############+--+++
         -########+-+++++++++++++######+++####+--+++
         -#######+++++++++++++++++##############-+++
         -######++++++------+++-+--############.-+++
         ++####+.+-++-------+++++------#######-#-+++
          -#++..+--+++------+++++-------+-####.#-++++
          ++--++----++++----+++++------++#..-+..+++++-
          ++---+---+#+++++-++++++---++++++++#+++++++++
         ++++--++-++#+++++++++++++++++++++++#++-+-+++++
         ++++---++++  +++++++++++++--+++++++##++++++++++
         +++++---++.  +++++++++--------++++++#+++++++++++
          ++++----+   ++++++++++------+++++++#++++++++++
           ++++++++   +++++++++++-----+++++++++++++++++-
           +++++++    ++++++++++++++++++++++++   +++++.
             ++-      ##++++++++++++++-+++++#      ++.
                      ###+---+#++++++++######
                      +##-#-++..##############-
                     #-------++.############-..-
                     ----.--.... .........##....
                    -.---..--... .. ......#+#  .

Manage and execute parametric command-line snippets.
A structured alternative to shell aliases and history -- no sourcing, no .bashrc edits.`

// NewRootCmd creates and returns the root cobra command for ali.
func NewRootCmd() *cobra.Command {
	configDir, _ := config.Dir()

	rootCmd := &cobra.Command{
		Use:     "ali",
		Short:   "Manage and execute parametric command-line snippets",
		Long:    aliLogo + fmt.Sprintf("\n\nConfig:\n  %s", configDir),
		Version: fmt.Sprintf("ali %s\ncommit: %s\nbuilt:  %s\nauthor: %s", version.Version, version.Commit, version.BuildDate, version.Author),
	}

	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Route Cobra's help/usage output through displayOut so it goes to
	// stderr when stdout is captured by the shell wrapper.
	rootCmd.SetOut(displayOut)

	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newAiCmd())
	rootCmd.AddCommand(newChatCmd())
	rootCmd.AddCommand(newHistoryCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newMoveCmd())
	rootCmd.AddCommand(newRemoveCmd())
	rootCmd.AddCommand(newEditCmd())
	rootCmd.AddCommand(newRunCmd())

	// Hide the auto-generated completion command from help output.
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.ValidArgsFunction = functionNamesCompletion

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
		if len(args[i]) == 0 {
			continue
		}
		if args[i][0] == '-' {
			// Skip flags. Note: this does not skip the value of a flag
			// that takes one (e.g., `ali --config path glog` would see
			// "path" as a non-flag arg). Currently no root-level flags
			// take values, so this is not a problem. If such flags are
			// added, this loop must be updated to skip their values too.
			continue
		}
		if strings.HasPrefix(args[i], "__") {
			// Cobra internal commands (e.g. __complete, __completeNoDesc).
			// Don't rewrite args for completion requests.
			return args
		}
		if knownSubcommands[args[i]] {
			return args
		}
		// Not a known subcommand -- insert "run" before this arg.
		newArgs := make([]string, 0, len(args)+1)
		newArgs = append(newArgs, args[:i]...)
		newArgs = append(newArgs, "run")
		newArgs = append(newArgs, args[i:]...)
		return newArgs
	}
	return args
}
