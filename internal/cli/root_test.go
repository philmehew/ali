package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRunCommandIsVisible(t *testing.T) {
	cmd := NewRootCmd()

	var runCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "run" {
			runCmd = sub
			break
		}
	}

	if runCmd == nil {
		t.Fatal("expected 'run' subcommand to exist")
	}
	if runCmd.Hidden {
		t.Error("expected 'run' subcommand to be visible, but it is hidden")
	}
}

func TestRunCommandHasRAlias(t *testing.T) {
	cmd := NewRootCmd()

	var runCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "run" {
			runCmd = sub
			break
		}
	}

	if runCmd == nil {
		t.Fatal("expected 'run' subcommand to exist")
	}

	found := false
	for _, alias := range runCmd.Aliases {
		if alias == "r" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'run' to have alias 'r', got aliases: %v", runCmd.Aliases)
	}
}
