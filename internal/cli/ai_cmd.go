package cli

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/config"

	"github.com/spf13/cobra"
)

// backtickRe matches markdown code fences like ```sh or ``` that AI models
// sometimes wrap around shell commands. Only matches standalone fence
// lines (opening or closing), not backticks embedded in commands.
var backtickRe = regexp.MustCompile("(?m)^```\\w*\\s*$")

func newAiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai <query>",
		Short: "Generate a shell command from a natural language query",
		Long: `Send a natural language query to an AI service and paste the
resulting shell command into your terminal input buffer.

The AI is instructed to respond with ONLY a shell command.
The result is pasted using the same mechanism as 'ali run',
so you can edit it before executing.

Configuration is stored in ~/.config/ali/ai.yaml or set via
environment variables: ALI_AI_URL, ALI_AI_API_KEY, ALI_AI_MODEL.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.LoadAIConfig()
			if err != nil {
				return fmt.Errorf("could not load AI config: %w", err)
			}

			if cfg.APIKey == "" {
				cfg, err = setupAIConfig()
				if err != nil {
					return err
				}
			}

			client := ai.NewClient(cfg)
			query := strings.Join(args, " ")

			messages := []ai.Message{
				{Role: ai.RoleSystem, Content: cfg.AiPrompt},
				{Role: ai.RoleUser, Content: query},
			}

			outputf("Thinking...\n")

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := client.Complete(ctx, messages)
			if err != nil {
				return err
			}

			result = cleanCommand(result)
			if result == "" {
				return fmt.Errorf("AI returned an empty response")
			}

			// Write command to stdout for the shell wrapper to paste.
			// This mirrors execution.PasteCommand's behavior.
			_, _ = fmt.Fprintln(os.Stdout, result)
			os.Exit(0)
			return nil
		},
	}

	return cmd
}

// cleanCommand strips markdown code fences and trims whitespace from AI output.
func cleanCommand(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```sh ... ``` or ``` ... ``` wrappers.
	s = backtickRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}

// aiConfigPath returns a user-friendly path to the AI config file.
func aiConfigPath() string {
	path, err := config.AIConfigPath()
	if err != nil {
		return "ai.yaml"
	}
	return path
}
