package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func newAddCmd() *cobra.Command {
	var desc string
	var defaults string

	cmd := &cobra.Command{
		Use:   "add <name> <body>",
		Short: "Add a new function",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			body := args[1]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			if config.FindFunction(cfg, name) != nil {
				return fmt.Errorf("function %q already exists", name)
			}

			var defaultSlice []string
			if defaults != "" {
				defaultSlice = strings.Split(defaults, ",")
			}

			fn := models.AliFunction{
				Name:        name,
				Description: desc,
				Body:        body,
				Defaults:    defaultSlice,
			}

			cfg.Functions = append(cfg.Functions, fn)

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}

			outputf("Added function %q\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&desc, "desc", "d", "", "description of the function")
	cmd.Flags().StringVarP(&defaults, "defaults", "D", "", "comma-separated default values for parameters")

	return cmd
}
