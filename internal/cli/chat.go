package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"

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
Type /exit to end the session.

In-session commands:
  /file <path>    Add a file to the conversation context
  /folder <path>  Add all files in a folder to the context (non-recursive)
  /files          List files currently in context
  /clear          Clear conversation history (keeps file context)
  /clear all      Clear conversation and file context
  /context        Show context window usage
  /model          Show current AI model information
  /model <name>   Switch to a different model for this session
  /window         Show context window size
  /window <size>  Set context window size (e.g., /window 128000)
  /help           Show all commands
  /exit, /quit    End the session

Use // to send a message that starts with /.
Use --file and --folder to include file contents at startup.

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
				{Role: ai.RoleSystem, Content: cfg.ChatPrompt},
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
					fileCount += ingestFolder(dir, &content)
				}

				if content.Len() > 0 {
					history = append(history, ai.Message{Role: ai.RoleUser, Content: content.String()})
					tokens := ai.EstimateTokens(content.String())
					outputf("Ingested %d file(s) (~%d tokens). Context budget: %d tokens.\n\n", fileCount, tokens, budget)
				}
			}

			// Warn if initial context exceeds budget.
			if historyTokens(history) > budget {
				outputf("Warning: initial context (%d tokens) exceeds budget (%d tokens). Responses may be truncated.\n\n", historyTokens(history), budget)
			}

			outputln("ali chat — type /help for commands, /exit to quit")
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
				// Double-slash escape: send as normal message
				if strings.HasPrefix(line, "//") {
					line = line[1:]
				} else if strings.HasPrefix(line, "/") {
					if handleChatCommand(line, &history, &trimWarned, &client, cfg, &budget) {
						break
					}
					continue
				}

				history = append(history, ai.Message{Role: ai.RoleUser, Content: line})

				// Trim oldest conversation turns if over budget.
				// Keep system message (index 0) and file context (index 1 if present).
				history, trimWarned = trimHistory(history, budget, trimWarned)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

				// done is closed on interrupt so the onToken callback stops
				// printing immediately, rather than draining buffered data.
				done := make(chan struct{})
				var printed strings.Builder

				// Catch Ctrl+C during streaming to cancel the request.
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, os.Interrupt)
				go func() {
					select {
					case <-sigCh:
						close(done)
						cancel()
					case <-ctx.Done():
					}
					signal.Stop(sigCh)
				}()

				response, err := client.CompleteStream(ctx, history, func(token string) {
					select {
					case <-done:
						return
					default:
					}
					output(token)
					printed.WriteString(token)
				})
				signal.Stop(sigCh)
				cancel()

				if err != nil {
					if ctx.Err() == context.Canceled {
						// User interrupted — keep what was printed before interrupt.
						outputln()
						outputln("[interrupted]")
						if printed.Len() > 0 {
							history = append(history, ai.Message{Role: ai.RoleAssistant, Content: printed.String()})
						} else {
							history = history[:len(history)-1]
						}
					} else {
						outputf("\nError: %v\n", err)
						history = history[:len(history)-1]
					}
					continue
				}

				outputln()
				outputln()

				history = append(history, ai.Message{Role: ai.RoleAssistant, Content: response})
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&files, "file", nil, "include a file in the conversation context (repeatable)")
	cmd.Flags().StringArrayVar(&folders, "folder", nil, "include all files in a folder in the conversation context (repeatable, non-recursive)")

	return cmd
}

// ingestFolder reads all regular files in a directory (non-recursive) and appends
// their contents to the builder via ingestFile. Returns the number of files ingested.
func ingestFolder(dir string, content *strings.Builder) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		outputf("Warning: could not read folder %s: %v\n", dir, err)
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !e.Type().IsRegular() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if ingestFile(path, path, content) {
			count++
		}
	}
	return count
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

// handleChatCommand parses and dispatches an in-session slash command.
// Returns true if the session should end (/exit or /quit).
func handleChatCommand(line string, history *[]ai.Message, trimWarned *bool, client **ai.Client, cfg *models.AIConfig, budget *int) bool {
	parts := strings.SplitN(line, " ", 2)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "/exit", "/quit":
		return true
	case "/file":
		cmdFile(args, history, *budget)
	case "/folder":
		cmdFolder(args, history, *budget)
	case "/clear":
		cmdClear(args, history, trimWarned)
	case "/files":
		cmdFiles(*history)
	case "/context":
		cmdContext(*history, *budget)
	case "/model":
		cmdModel(args, cfg, client)
	case "/window":
		cmdWindow(args, cfg, budget)
	default:
		outputf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}
	return false
}

// cmdFile adds a file to the conversation context.
func cmdFile(args string, history *[]ai.Message, budget int) {
	if args == "" {
		outputln("Usage: /file <path>")
		return
	}
	path, err := filepath.Abs(args)
	if err != nil {
		outputf("Error: could not resolve path %s: %v\n", args, err)
		return
	}
	var content strings.Builder
	if ingestFile(path, path, &content) {
		addFileContentToHistory(history, content.String())
		tokens := historyTokens(*history)
		pct := tokens * 100 / budget
		outputf("Added %s. Context: %d/%d tokens (%d%%)\n", path, tokens, budget, pct)
	}
}

// cmdFolder adds all files in a folder to the conversation context.
func cmdFolder(args string, history *[]ai.Message, budget int) {
	if args == "" {
		outputln("Usage: /folder <path>")
		return
	}
	path, err := filepath.Abs(args)
	if err != nil {
		outputf("Error: could not resolve path %s: %v\n", args, err)
		return
	}
	var content strings.Builder
	count := ingestFolder(path, &content)
	if count > 0 {
		addFileContentToHistory(history, content.String())
		tokens := historyTokens(*history)
		pct := tokens * 100 / budget
		outputf("Added %d file(s) from %s. Context: %d/%d tokens (%d%%)\n", count, path, tokens, budget, pct)
	}
}

// cmdClear clears conversation history.
// With no args, keeps system prompt and file context.
// With "all", keeps only the system prompt.
func cmdClear(args string, history *[]ai.Message, trimWarned *bool) {
	if args == "all" {
		*history = (*history)[:1]
		*trimWarned = false
		outputln("Conversation and file context cleared.")
		return
	}
	prefixLen := 1
	if len(*history) > 1 && strings.HasPrefix((*history)[1].Content, "<file") {
		prefixLen = 2
		fileCount := len(fileContextPaths((*history)[1].Content))
		*history = (*history)[:prefixLen]
		*trimWarned = false
		outputf("Conversation cleared. %d file(s) still in context.\n", fileCount)
		return
	}
	*history = (*history)[:prefixLen]
	*trimWarned = false
	outputln("Conversation cleared.")
}

// cmdFiles lists the file paths currently in the conversation context.
func cmdFiles(history []ai.Message) {
	if len(history) < 2 || !strings.HasPrefix(history[1].Content, "<file") {
		outputln("No files in context.")
		return
	}
	paths := fileContextPaths(history[1].Content)
	if len(paths) == 0 {
		outputln("No files in context.")
		return
	}
	outputln("Files in context:")
	for _, p := range paths {
		outputf("  %s\n", p)
	}
}

// fileContextPaths extracts file paths from the XML-tagged file context message.
var filePathRe = regexp.MustCompile(`<file path="([^"]+)"`)

func fileContextPaths(content string) []string {
	matches := filePathRe.FindAllStringSubmatch(content, -1)
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, m[1])
	}
	return paths
}

// cmdContext shows context window usage and turn count.
func cmdContext(history []ai.Message, budget int) {
	tokens := historyTokens(history)
	pct := tokens * 100 / budget

	prefixLen := 1
	if len(history) > 1 && strings.HasPrefix(history[1].Content, "<file") {
		prefixLen = 2
	}
	turns := (len(history) - prefixLen) / 2

	outputf("Context: %d/%d tokens (%d%%)\n", tokens, budget, pct)
	outputf("Turns: %d\n", turns)
	fileCount := 0
	if prefixLen == 2 {
		fileCount = len(fileContextPaths(history[1].Content))
	}
	outputf("Files: %d\n", fileCount)
}

// cmdModel shows or switches the current AI model.
func cmdModel(args string, cfg *models.AIConfig, client **ai.Client) {
	if args == "" {
		outputf("Model:  %s\n", cfg.Model)
		outputf("URL:    %s\n", cfg.URL)
		outputf("Window: %d tokens\n", cfg.ContextWindow)
		return
	}
	oldModel := cfg.Model
	cfg.Model = args
	*client = ai.NewClient(cfg)
	outputf("Model changed: %s -> %s\n", oldModel, args)
	outputf("Context window is %d tokens. Use /window <size> to update.\n", cfg.ContextWindow)
}

// cmdWindow shows or updates the context window size.
func cmdWindow(args string, cfg *models.AIConfig, budget *int) {
	if args == "" {
		outputf("Context window: %d tokens\n", cfg.ContextWindow)
		return
	}
	var size int
	if _, err := fmt.Sscanf(args, "%d", &size); err != nil || size <= 0 {
		outputln("Usage: /window <size>  (e.g., /window 128000)")
		return
	}
	cfg.ContextWindow = size
	*budget = ai.ContextBudget(size)
	outputf("Context window set to %d tokens (budget: %d)\n", size, *budget)
}

// cmdHelp shows all available in-session commands.
func cmdHelp() {
	outputln("Commands:")
	outputln("  /file <path>    Add a file to the conversation context")
	outputln("  /folder <path>  Add all files in a folder to the context (non-recursive)")
	outputln("  /files          List files currently in context")
	outputln("  /clear          Clear conversation history (keeps file context)")
	outputln("  /clear all      Clear conversation and file context")
	outputln("  /context        Show context window usage")
	outputln("  /model          Show current AI model information")
	outputln("  /model <name>   Switch to a different model for this session")
	outputln("  /window         Show context window size")
	outputln("  /window <size>  Set context window size (e.g., /window 128000)")
	outputln("  /help           Show this help message")
	outputln("  /exit, /quit    End the session")
	outputln()
	outputln("Type a message to chat with the AI.")
	outputln("Use // to send a message that starts with /.")
}

// addFileContentToHistory appends file content to the history, either by
// extending an existing file-context message at index 1 or inserting a new one.
func addFileContentToHistory(history *[]ai.Message, content string) {
	if len(*history) > 1 && strings.HasPrefix((*history)[1].Content, "<file") {
		(*history)[1].Content += content
	} else {
		fileMsg := ai.Message{Role: ai.RoleUser, Content: content}
		*history = append([]ai.Message{(*history)[0], fileMsg}, (*history)[1:]...)
	}
}
