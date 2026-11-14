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
		Use:     "run <name> [params...] [+ <name> [params...]]...",
		Aliases: []string{"r"},
		Short:   "Execute one or more stored functions",
		Long: `Execute a stored function, or chain several with a standalone "+".

Each chain segment is resolved independently: the first token is a function
name (or the number shown in "ali list"), and the rest are its parameters.
Only a standalone "+" argument separates commands, so "+5", "5+", and
"a+b" are treated as ordinary parameters.`,
		Example: `  ali run glog
  ali run glog 20
  ali run dockup + cellog
  ali run 3 + 5
  ali run deploy prod /path + cellog
  ali run find "*.go" -- -type f   # "--" disables chaining for later args`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			resolved, err := resolveChain(cfg, args, cmd.Flags().ArgsLenAtDash())
			if err != nil {
				return err
			}

			execution.PasteCommands(resolved)
			return nil // unreachable — PasteCommands calls os.Exit
		},
	}

	cmd.ValidArgsFunction = functionParamsCompletion
	return cmd
}

// resolveChain splits args into chained commands and resolves each one
// independently against cfg.
func resolveChain(cfg *models.AliConfig, args []string, dash int) ([]*execution.ResolvedCommand, error) {
	segments, err := splitChainArgs(args, dash)
	if err != nil {
		return nil, err
	}

	resolved := make([]*execution.ResolvedCommand, 0, len(segments))
	for _, segment := range segments {
		fn := resolveFunction(cfg, segment[0])
		if fn == nil {
			return nil, fmt.Errorf("function %q not found", segment[0])
		}

		resolvedCmd, err := execution.Resolve(fn, segment[1:])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fn.Name, err)
		}
		resolved = append(resolved, resolvedCmd)
	}

	return resolved, nil
}

// splitChainArgs splits args into one segment per chained command. Commands
// are separated by a standalone "+" argument. dash is the index returned by
// cmd.Flags().ArgsLenAtDash() (-1 when no "--" was used); "+" tokens at or
// after dash are treated as ordinary arguments, which lets users pass a
// literal "+" as a parameter after "--".
func splitChainArgs(args []string, dash int) ([][]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	segments := make([][]string, 0, 1)
	current := make([]string, 0, len(args))

	for i, arg := range args {
		chainingAllowed := dash < 0 || i < dash
		if arg == "+" && chainingAllowed {
			if len(current) == 0 {
				return nil, fmt.Errorf(`empty command before "+": "+" must separate two commands`)
			}
			segments = append(segments, current)
			current = make([]string, 0, len(args)-i)
			continue
		}
		current = append(current, arg)
	}

	if len(current) == 0 {
		return nil, fmt.Errorf(`empty command after "+": "+" must separate two commands`)
	}
	segments = append(segments, current)

	return segments, nil
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
