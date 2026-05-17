package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/config"

	"github.com/spf13/cobra"
)

const maxFileSize = 1024 * 1024 // 1MB — skip files larger than this

func newChatCmd() *cobra.Command {
	var files []string
	var folders []string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat with AI",
		Long: `Start an interactive session for chatting with an AI assistant.
Type your message and press Enter to send.
Type "exit" or "quit" to end the session.

Use --file and --folder to include file contents in the conversation
context so the AI can answer questions about your code, configs, or logs.

Configuration is stored in ~/.config/ali/ai.yaml or set via
environment variables: ALI_AI_URL, ALI_AI_API_KEY, ALI_AI_MODEL.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
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
			budget := ai.ContextBudget(cfg.ContextWindow)

			history := []ai.Message{
				{Role: "system", Content: cfg.ChatPrompt},
			}

			// Ingest files and folders into context.
			if len(files) > 0 || len(folders) > 0 {
				var content strings.Builder
				fileCount := 0

				for _, f := range files {
					if ingestFile(f, f, &content) {
						fileCount++
					}
				}
				for _, dir := range folders {
					entries, err := os.ReadDir(dir)
					if err != nil {
						outputf("Warning: could not read folder %s: %v\n", dir, err)
						continue
					}
					for _, e := range entries {
						if e.IsDir() || !e.Type().IsRegular() {
							continue
						}
						path := filepath.Join(dir, e.Name())
						if ingestFile(path, path, &content) {
							fileCount++
						}
					}
				}

				if content.Len() > 0 {
					history = append(history, ai.Message{Role: "user", Content: content.String()})
					tokens := ai.EstimateTokens(content.String())
					outputf("Ingested %d file(s) (~%d tokens). Context budget: %d tokens.\n\n", fileCount, tokens, budget)
				}
			}

			// Warn if initial context exceeds budget.
			if historyTokens(history) > budget {
				outputf("Warning: initial context (%d tokens) exceeds budget (%d tokens). Responses may be truncated.\n\n", historyTokens(history), budget)
			}

			outputln("ali chat — type 'exit' or 'quit' to end, Ctrl+C to interrupt")
			outputln()

			reader := bufio.NewReader(os.Stdin)
			trimWarned := false

			for {
				pct := historyTokens(history) * 100 / budget
				outputf("[%d%%] > ", pct)
				line, err := reader.ReadString('\n')
				if err != nil {
					outputln()
					break
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if line == "exit" || line == "quit" {
					break
				}

				history = append(history, ai.Message{Role: "user", Content: line})

				// Trim oldest conversation turns if over budget.
				// Keep system message (index 0) and file context (index 1 if present).
				history, trimWarned = trimHistory(history, budget, trimWarned)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				response, err := client.CompleteStream(ctx, history, func(token string) {
					output(token)
				})
				cancel()
				if err != nil {
					outputf("\nError: %v\n", err)
					history = history[:len(history)-1]
					continue
				}

				outputln()
				outputln()

				history = append(history, ai.Message{Role: "assistant", Content: response})
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&files, "file", nil, "include a file in the conversation context (repeatable)")
	cmd.Flags().StringArrayVar(&folders, "folder", nil, "include all files in a folder in the conversation context (repeatable, non-recursive)")

	return cmd
}

// ingestFile reads a file and appends its contents to the builder in XML-tagged format.
// Returns true if the file was successfully ingested.
func ingestFile(displayPath, filePath string, content *strings.Builder) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		outputf("Warning: could not stat %s: %v\n", displayPath, err)
		return false
	}
	if info.Size() > maxFileSize {
		outputf("Warning: skipping %s (exceeds 1MB)\n", displayPath)
		return false
	}

	data, err := os.ReadFile(filePath) //nolint:gosec // G304: user-specified file for AI context
	if err != nil {
		outputf("Warning: could not read %s: %v\n", displayPath, err)
		return false
	}

	if isBinary(data) {
		outputf("Warning: skipping binary file %s\n", displayPath)
		return false
	}

	fmt.Fprintf(content, "<file path=%q>\n", displayPath)
	content.Write(data)
	if !bytes.HasSuffix(data, []byte("\n")) {
		content.WriteByte('\n')
	}
	content.WriteString("</file>\n")
	return true
}

// isBinary checks whether data appears to be binary by looking for null bytes
// in the first 512 bytes.
func isBinary(data []byte) bool {
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	return bytes.Contains(check, []byte{0})
}

// historyTokens estimates the total token count of all messages in history.
func historyTokens(history []ai.Message) int {
	total := 0
	for _, m := range history {
		total += ai.EstimateTokens(m.Content)
	}
	return total
}

// trimHistory removes the oldest conversation turns (user/assistant pairs)
// when the history exceeds the token budget. It always preserves the system
// message and file context. Returns the trimmed history and whether a
// warning was issued.
func trimHistory(history []ai.Message, budget int, warned bool) ([]ai.Message, bool) {
	if historyTokens(history) <= budget {
		return history, warned
	}

	// Find the first conversation turn after system message and file context.
	// System message is always index 0. File context (if present) is index 1.
	start := 1
	if len(history) > 1 && strings.HasPrefix(history[1].Content, "<file") {
		start = 2
	}

	// Remove one user/assistant pair at a time until under budget.
	for historyTokens(history) > budget && len(history) > start+2 {
		// Remove the oldest turn after the preserved prefix.
		// A turn is a user + assistant pair, so remove 2 messages.
		end := start + 2
		if end > len(history) {
			break
		}
		history = append(history[:start], history[end:]...)
	}

	if !warned {
		outputf("Warning: context window approaching limit, trimmed oldest messages.\n")
		warned = true
	}

	return history, warned
}
