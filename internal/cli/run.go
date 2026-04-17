package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/execution"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name> [params...]",
		Short: "Resolve and print a stored function",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			params := args[1:]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			fn := config.FindFunction(cfg, name)
			if fn == nil {
				return fmt.Errorf("function %q not found", name)
			}

			resolved, err := execution.Resolve(fn, params)
			if err != nil {
				return err
			}

			execution.PasteCommand(resolved)
			return nil // unreachable — PasteCommand calls os.Exit
		},
	}

	cmd.Hidden = true
	return cmd
}
