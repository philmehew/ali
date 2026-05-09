package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/execution"
	"github.com/philmehew/ali/internal/models"
)

func newListCmd() *cobra.Command {
	var ignored bool

	cmd := &cobra.Command{
		Use:     "list [keywords...]",
		Aliases: []string{"ls"},
		Short: "List stored functions, optionally filtered by keywords",
		Long: `List all stored functions, or filter by keywords.

Keywords perform case-insensitive substring matching against the function name,
description, and body. Multiple keywords are combined with AND logic — all
keywords must match somewhere in the function's fields.

Use --ignored to list commands on the ignore list instead.

Examples:
  ali list              # show all functions
  ali list doc comp     # functions matching both "doc" and "comp"
  ali list git          # functions matching "git"
  ali list --ignored    # show ignored commands`,
		Args: cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			if ignored {
				return listIgnoredInteractive(cfg)
			}

			if len(cfg.Functions) == 0 {
				outputln("No functions defined. Use 'ali add' to create one.")
				return nil
			}

			keywords := args
			return listFunctionsInteractive(cfg, keywords)
		},
	}

	cmd.Flags().BoolVar(&ignored, "ignored", false, "list ignored commands instead of functions")

	return cmd
}

// listFunctionsInteractive displays a numbered list of functions and prompts
// for edit/remove actions in a loop.
func listFunctionsInteractive(cfg *models.AliConfig, keywords []string) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		functions := cfg.Functions
		if len(keywords) > 0 {
			functions = filterFunctions(cfg.Functions, keywords)
		}

		if len(functions) == 0 {
			if len(keywords) > 0 {
				outputf("No functions matching: %s\n", strings.Join(keywords, " "))
			} else {
				outputln("No functions defined. Use 'ali add' to create one.")
			}
			return nil
		}

		printFunctionList(functions)
		outputln()
		output("Enter number to execute, 'e <num>' to edit, 'm <from> <to>' to move, 'r <num>' to remove, or 'q' to quit: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if strings.ToLower(input) == "q" {
			break
		}

		// Parse "e <num>" to edit.
		if strings.HasPrefix(strings.ToLower(input), "e ") {
			numStr := strings.TrimSpace(input[2:])
			num := 0
			if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil || num < 1 || num > len(functions) {
				outputln("Invalid number.")
				continue
			}
			fn := functions[num-1]
			idx := config.FindFunctionIndex(cfg, fn.Name)
			if err := editFunction(cfg, idx); err != nil {
				return err
			}
			outputln()
			continue
		}

		// Parse "m <from> <to>" to move.
		if strings.HasPrefix(strings.ToLower(input), "m ") {
			parts := strings.Fields(input[2:])
			if len(parts) != 2 {
				outputln("Usage: m <from> <to>")
				continue
			}
			fromNum, toNum := 0, 0
			if _, err := fmt.Sscanf(parts[0], "%d", &fromNum); err != nil || fromNum < 1 || fromNum > len(functions) {
				outputln("Invalid 'from' number.")
				continue
			}
			if _, err := fmt.Sscanf(parts[1], "%d", &toNum); err != nil || toNum < 1 || toNum > len(functions) {
				outputln("Invalid 'to' number.")
				continue
			}
			fromIdx := config.FindFunctionIndex(cfg, functions[fromNum-1].Name)
			toIdx := toNum - 1
			name := functions[fromNum-1].Name
			config.MoveFunction(cfg, fromIdx, toIdx)
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}
			outputf("Moved %q to position %d\n\n", name, toNum)
			continue
		}

		// Parse "r <num>" to remove.
		if strings.HasPrefix(strings.ToLower(input), "r ") {
			numStr := strings.TrimSpace(input[2:])
			num := 0
			if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil || num < 1 || num > len(functions) {
				outputln("Invalid number.")
				continue
			}
			fn := functions[num-1]
			idx := config.FindFunctionIndex(cfg, fn.Name)
			if err := removeFunction(cfg, idx); err != nil {
				return err
			}
			outputln()
			continue
		}

		// Parse number to execute.
		num := 0
		if _, err := fmt.Sscanf(input, "%d", &num); err != nil || num < 1 || num > len(functions) {
			outputln("Enter a number, 'e <num>', 'm <from> <to>', 'r <num>', or 'q'.")
			continue
		}

		fn := functions[num-1]
		idx := config.FindFunctionIndex(cfg, fn.Name)
		resolved, err := execution.Resolve(&cfg.Functions[idx], nil)
		if err != nil {
			outputf("Error: %v\n", err)
			continue
		}
		execution.PasteCommand(resolved)
	}

	return nil
}

// printFunctionList displays a numbered list of functions with name, body, and description.
func printFunctionList(functions []models.AliFunction) {
	w := tabwriter.NewWriter(displayOut, 0, 0, 2, ' ', 0)
	for i, fn := range functions {
		if fn.Description != "" {
			_, _ = fmt.Fprintf(w, "  %2d.\t%s\t%s\t%s\n", i+1, fn.Name, fn.Body, fn.Description)
		} else {
			_, _ = fmt.Fprintf(w, "  %2d.\t%s\t%s\n", i+1, fn.Name, fn.Body)
		}
	}
	_ = w.Flush()
}

// listIgnoredInteractive displays a numbered list of ignored commands and prompts
// for edit/remove actions in a loop.
func listIgnoredInteractive(cfg *models.AliConfig) error {
	if len(cfg.Ignore) == 0 {
		outputln("No ignored commands.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		if len(cfg.Ignore) == 0 {
			outputln("No ignored commands.")
			return nil
		}

		printIgnoredList(cfg.Ignore)
		outputln()
		output("Enter 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if strings.ToLower(input) == "q" {
			break
		}

		// Parse "e <num>" to edit — opens the entire ignore list in $EDITOR.
		if strings.HasPrefix(strings.ToLower(input), "e ") {
			numStr := strings.TrimSpace(input[2:])
			num := 0
			if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil || num < 1 || num > len(cfg.Ignore) {
				outputln("Invalid number.")
				continue
			}
			if err := editIgnored(); err != nil {
				return err
			}
			// Reload config after editing.
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("could not reload config: %w", err)
			}
			outputln()
			continue
		}

		// Parse "r <num>" to remove.
		if strings.HasPrefix(strings.ToLower(input), "r ") {
			numStr := strings.TrimSpace(input[2:])
			num := 0
			if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil || num < 1 || num > len(cfg.Ignore) {
				outputln("Invalid number.")
				continue
			}
			cmd := cfg.Ignore[num-1]
			cfg.Ignore = append(cfg.Ignore[:num-1], cfg.Ignore[num:]...)
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}
			outputf("Removed %q from ignore list\n\n", cmd)
			continue
		}

		outputln("Enter 'e <num>', 'r <num>', or 'q'.")
		// Reload config after editing.
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("could not reload config: %w", err)
		}
		outputln()
	}

	return nil
}

// printIgnoredList displays a numbered list of ignored commands.
func printIgnoredList(ignore []string) {
	w := tabwriter.NewWriter(displayOut, 0, 0, 2, ' ', 0)
	for i, cmd := range ignore {
		_, _ = fmt.Fprintf(w, "  %2d.\t%s\n", i+1, cmd)
	}
	_ = w.Flush()
}

// filterFunctions returns functions where every keyword is a case-insensitive
// substring match against the function's name, description, or body.
func filterFunctions(functions []models.AliFunction, keywords []string) []models.AliFunction {
	var matched []models.AliFunction

	for _, fn := range functions {
		combined := strings.ToLower(fn.Name + " " + fn.Description + " " + fn.Body)
		allMatch := true
		for _, kw := range keywords {
			if !strings.Contains(combined, strings.ToLower(kw)) {
				allMatch = false
				break
			}
		}
		if allMatch {
			matched = append(matched, fn)
		}
	}

	return matched
}
