package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
)

// functionNamesCompletion returns stored function names as completions.
func functionNamesCompletion(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []cobra.Completion
	for _, fn := range cfg.Functions {
		if !strings.HasPrefix(fn.Name, toComplete) {
			continue
		}
		if fn.Description != "" {
			completions = append(completions, cobra.CompletionWithDesc(fn.Name, fn.Description))
		} else {
			completions = append(completions, fn.Name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// functionParamsCompletion completes function names for the first argument
// and default parameter values for subsequent arguments.
func functionParamsCompletion(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return functionNamesCompletion(cmd, args, toComplete)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	fn := resolveFunction(cfg, args[0])
	if fn == nil {
		return nil, cobra.ShellCompDirectiveError
	}

	paramIdx := len(args) - 1
	if paramIdx < len(fn.Defaults) && fn.Defaults[paramIdx] != "" {
		dv := fn.Defaults[paramIdx]
		if strings.HasPrefix(dv, toComplete) {
			return []cobra.Completion{cobra.CompletionWithDesc(dv, fmt.Sprintf("default for $%d", paramIdx+1))}, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return nil, cobra.ShellCompDirectiveDefault
}

// chatNamesCompletion returns saved chat session names as completions for the --load flag.
func chatNamesCompletion(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	dir, err := chatsDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []cobra.Completion
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if !strings.HasPrefix(name, toComplete) {
			continue
		}
		completions = append(completions, name)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
