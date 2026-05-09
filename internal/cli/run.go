package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/execution"
	"github.com/philmehew/ali/internal/models"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name> [params...]",
		Short: "Resolve and print a stored function",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := args[0]
			params := args[1:]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			fn := resolveFunction(cfg, arg)
			if fn == nil {
				return fmt.Errorf("function %q not found", arg)
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

// resolveFunction looks up a function by name or by 1-based index number.
func resolveFunction(cfg *models.AliConfig, arg string) *models.AliFunction {
	// Try as a number first.
	if num, err := strconv.Atoi(arg); err == nil {
		if num >= 1 && num <= len(cfg.Functions) {
			return &cfg.Functions[num-1]
		}
		return nil
	}
	return config.FindFunction(cfg, arg)
}
