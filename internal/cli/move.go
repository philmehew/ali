package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
)

func newMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "move <from> <to>",
		Aliases: []string{"mv"},
		Short:   "Move a function to a new position",
		Long: `Move a function to a new position in the list.
Both <from> and <to> can be a 1-based number (as shown in ali list) or a function name.
<from> identifies which function to move; <to> is the target position number.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			fromArg := args[0]
			toArg := args[1]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			if len(cfg.Functions) == 0 {
				return fmt.Errorf("no functions defined")
			}

			// Resolve <from> — name or number.
			var fromIdx int
			if num, err := strconv.Atoi(fromArg); err == nil {
				if num < 1 || num > len(cfg.Functions) {
					return fmt.Errorf("number %d out of range (1-%d)", num, len(cfg.Functions))
				}
				fromIdx = num - 1
			} else {
				fromIdx = config.FindFunctionIndex(cfg, fromArg)
				if fromIdx == -1 {
					return fmt.Errorf("function %q not found", fromArg)
				}
			}

			// Resolve <to> — must be a position number.
			toIdx, err := strconv.Atoi(toArg)
			if err != nil {
				return fmt.Errorf("<to> must be a position number, got %q", toArg)
			}
			if toIdx < 1 || toIdx > len(cfg.Functions) {
				return fmt.Errorf("position %d out of range (1-%d)", toIdx, len(cfg.Functions))
			}
			toIdx-- // convert to 0-based

			name := cfg.Functions[fromIdx].Name
			config.MoveFunction(cfg, fromIdx, toIdx)

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}

			outputf("Moved %q to position %d\n", name, toIdx+1)
			return nil
		},
	}

	return cmd
}
