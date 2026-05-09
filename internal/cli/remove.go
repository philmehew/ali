package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short: "Remove a stored function",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			arg := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			// If the arg is a number, resolve it to the function at that index.
			if num, err := strconv.Atoi(arg); err == nil {
				if num < 1 || num > len(cfg.Functions) {
					return fmt.Errorf("number %d out of range (1-%d)", num, len(cfg.Functions))
				}
				return removeFunction(cfg, num-1)
			}

			idx := config.FindFunctionIndex(cfg, arg)
			if idx == -1 {
				return fmt.Errorf("function %q not found", arg)
			}

			return removeFunction(cfg, idx)
		},
	}

	return cmd
}

// removeFunction removes the function at idx from the config and saves.
func removeFunction(cfg *models.AliConfig, idx int) error {
	name := cfg.Functions[idx].Name

	cfg.Functions = append(cfg.Functions[:idx], cfg.Functions[idx+1:]...)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	outputf("Removed function %q\n", name)
	return nil
}
