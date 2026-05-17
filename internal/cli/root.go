// Package cli implements the ali command-line interface using Cobra.
//
//nolint:goconst // UI strings (command names, help text) are not beneficial as constants
package cli

import (
	"fmt"

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

	return rootCmd
}
