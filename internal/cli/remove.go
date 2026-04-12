package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a stored function",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			idx := config.FindFunctionIndex(cfg, name)
			if idx == -1 {
				return fmt.Errorf("function %q not found", name)
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

	fmt.Printf("Removed function %q\n", name)
	return nil
}
