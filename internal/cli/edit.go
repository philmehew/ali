package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func newEditCmd() *cobra.Command {
	var ignored bool

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a stored function in your $EDITOR",
		Long: `Open a stored function in your $EDITOR (defaults to vi).
Use --ignored to edit the ignore list instead.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if ignored {
				return editIgnored()
			}

			if len(args) == 0 {
				return fmt.Errorf("requires a function name, or use --ignored")
			}

			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			idx := config.FindFunctionIndex(cfg, name)
			if idx == -1 {
				return fmt.Errorf("function %q not found", name)
			}

			return editFunction(cfg, idx)
		},
	}

	cmd.Flags().BoolVar(&ignored, "ignored", false, "edit the ignore list instead of a function")

	return cmd
}

// editFunction opens the function at idx in $EDITOR, reads it back, and saves.
func editFunction(cfg *models.AliConfig, idx int) error {
	name := cfg.Functions[idx].Name

	// Write the function to a temp YAML file.
	tmp, err := os.CreateTemp("", "ali-edit-*.yaml")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // cleanup: best-effort removal of temp file.

	data, err := yaml.Marshal(&cfg.Functions[idx])
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not marshal function: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write temp file: %w", err)
	}
	_ = tmp.Close()

	// Launch editor.
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	editCmd := exec.Command(editor, tmpName) //nolint:gosec // G204: editor from $EDITOR is intentional.
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = displayOut
	editCmd.Stderr = os.Stderr

	if err := editCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Read back the edited function.
	editedData, err := os.ReadFile(tmpName) //nolint:gosec // G304: tmpName is a controlled temp file path.
	if err != nil {
		return fmt.Errorf("could not read edited file: %w", err)
	}

	var editedFn models.AliFunction
	if err := yaml.Unmarshal(editedData, &editedFn); err != nil {
		return fmt.Errorf("could not parse edited function: %w", err)
	}

	if editedFn.Name == "" {
		return fmt.Errorf("edited function has no name")
	}

	// If the name changed, check for conflicts.
	if editedFn.Name != name {
		if config.FindFunction(cfg, editedFn.Name) != nil {
			return fmt.Errorf("function %q already exists", editedFn.Name)
		}
	}

	cfg.Functions[idx] = editedFn

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	outputf("Updated function %q\n", editedFn.Name)
	return nil
}

// editIgnored opens the ignore list in $EDITOR as a YAML list,
// reads it back, and saves the updated config.
func editIgnored() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("could not load config: %w", err)
	}

	if len(cfg.Ignore) == 0 {
		outputln("No ignored commands.")
		return nil
	}

	// Write the ignore list to a temp YAML file.
	tmp, err := os.CreateTemp("", "ali-edit-ignored-*.yaml")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck // cleanup: best-effort removal of temp file.

	data, err := yaml.Marshal(cfg.Ignore)
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not marshal ignore list: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write temp file: %w", err)
	}
	_ = tmp.Close()

	// Launch editor.
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	editCmd := exec.Command(editor, tmpName) //nolint:gosec // G204: editor from $EDITOR is intentional.
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = displayOut
	editCmd.Stderr = os.Stderr

	if err := editCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Read back the edited ignore list.
	editedData, err := os.ReadFile(tmpName) //nolint:gosec // G304: tmpName is a controlled temp file path.
	if err != nil {
		return fmt.Errorf("could not read edited file: %w", err)
	}

	var editedIgnore []string
	if err := yaml.Unmarshal(editedData, &editedIgnore); err != nil {
		return fmt.Errorf("could not parse edited ignore list: %w", err)
	}

	cfg.Ignore = editedIgnore

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}

	outputf("Updated ignore list (%d command(s))\n", len(cfg.Ignore))
	return nil
}
