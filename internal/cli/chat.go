//nolint:goconst // UI strings (command names, help text) are not beneficial as constants
package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"

	"github.com/spf13/cobra"
)

const maxFileSize = 1024 * 1024 // 1MB — skip files larger than this

// isContextMessage reports whether a message contains preserved context
// (file or command output) that should survive /clear and trimming.
func isContextMessage(msg ai.Message) bool {
	return strings.HasPrefix(msg.Content, "<file") || strings.HasPrefix(msg.Content, "<cmd")
}

func newChatCmd() *cobra.Command {
	var files []string
	var folders []string
	var cmds []string
	var defaultRuns int
	var defaultWait int
	var loadName string
	var deleteName string
	var memoryText string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat with AI",
		Long: `Start an interactive session for chatting with an AI assistant.
Type your message and press Enter to send.
Type /exit to end the session.

In-session commands:
  /file <path>                   Add a file to the conversation context
  /folder <path>                 Add all files in a folder to the context (non-recursive)
  /cmd <command> [--runs <n>] [--wait <s>]   Execute a command N times and add output to context
  /clear                         Clear conversation history (keeps file and command context)
  /clear all                     Clear conversation, file, and command context
  /context                       Show context window usage and context details
  /context all                   Dump full content of all messages in context
  /model          Show current AI model information
  /model <name>   Switch to a different model for this session
  /window         Show context window size
  /window <size>  Set context window size (e.g., /window 128000)
  /memory                        Show current memory content and file path
  /memory <text>                 Append text to memory (persists across sessions)
  /save <name>    Save conversation context to a file
  /load           List saved chats
  /load <name>    Load a previously saved conversation
  /delete <name>  Delete a saved chat session
  /help           Show all commands
  /exit, /quit    End the session

Use // to send a message that starts with /.
Use --file and --folder to include file contents at startup.
Use --cmd to run a command and include its output (--runs and --wait control repetition).
Use --load <name> to resume a previously saved session.
Use --delete <name> to delete a saved session.
Use --memory <text> to append text to memory before starting the session.

Memory is stored in ~/.config/ali/MEMORY.md and automatically loaded into the
system prompt at the start of each session. Use /memory to view it or /memory <text>
to add a line. Memory persists across sessions and survives /clear and /load.

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
			trimWarned := false

			// Process --memory flag: append text to MEMORY.md before starting.
			if memoryText != "" {
				if err := appendMemory(memoryText); err != nil {
					outputf("Warning: could not append to memory: %v\n", err)
				} else {
					outputf("Memory updated: %s\n", memoryText)
				}
			}

			var history []ai.Message
			sessionName := ""

			// Load a previously saved session if --load was given.
			if loadName != "" {
				cmdLoad(loadName, &history, cfg, &client, &budget, &trimWarned, &sessionName)
				if len(history) == 0 {
					return fmt.Errorf("could not load session %q", loadName)
				}
			} else {
				history = []ai.Message{
					{Role: ai.RoleSystem, Content: buildSystemPrompt(cfg.ChatPrompt)},
				}
			}

			// Delete a saved session if --delete was given, then exit.
			if deleteName != "" {
				cmdDelete(deleteName)
				return nil
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

			// Ingest command outputs into context.
			if len(cmds) > 0 {
				var content strings.Builder
				cmdCount := 0

				for _, c := range cmds {
					output, ok := ingestCmd(c, defaultRuns, time.Duration(defaultWait)*time.Second)
					if ok > 0 {
						content.WriteString(output)
						cmdCount++
					}
				}

				if content.Len() > 0 {
					addContextToHistory(&history, content.String())
					tokens := historyTokens(history)
					outputf("Executed %d command(s) (~%d tokens). Context budget: %d tokens.\n\n", cmdCount, tokens, budget)
				}
			}

			// Warn if initial context exceeds budget.
			if historyTokens(history) > budget {
				outputf("Warning: initial context (%d tokens) exceeds budget (%d tokens). Responses may be truncated.\n\n", historyTokens(history), budget)
			}

			outputln("ali chat — type /help for commands, /exit to quit")
			outputln()

			reader := bufio.NewReader(os.Stdin)
			trimWarned = false

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
					if handleChatCommand(line, &history, &trimWarned, &client, cfg, &budget, &sessionName) {
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

				if err != nil {
					wasCanceled := ctx.Err() == context.Canceled
					cancel()
					if wasCanceled {
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

				cancel()

				outputln()
				outputln()

				history = append(history, ai.Message{Role: ai.RoleAssistant, Content: response})
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVar(&files, "file", nil, "include a file in the conversation context (repeatable)")
	cmd.Flags().StringArrayVar(&folders, "folder", nil, "include all files in a folder in the conversation context (repeatable, non-recursive)")
	cmd.Flags().StringArrayVar(&cmds, "cmd", nil, "execute a command and include output in context (repeatable)")
	cmd.Flags().IntVar(&defaultRuns, "runs", 5, "number of times to run each --cmd (default 5)")
	cmd.Flags().IntVar(&defaultWait, "wait", 5, "seconds to wait between --cmd runs (default 5)")
	cmd.Flags().StringVar(&loadName, "load", "", "load a previously saved chat session by name")
	cmd.Flags().StringVar(&deleteName, "delete", "", "delete a saved chat session by name")
	cmd.Flags().StringVar(&memoryText, "memory", "", "append text to memory before starting the session")
	_ = cmd.RegisterFlagCompletionFunc("load", chatNamesCompletion)
	_ = cmd.RegisterFlagCompletionFunc("delete", chatNamesCompletion)

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

const cmdRunTimeout = 30 * time.Second
const maxCmdOutput = 100 * 1024 // 100KB — truncate per-run output beyond this

// ingestCmd executes a shell command the given number of times and returns
// the XML-tagged output suitable for adding to conversation context.
// Returns the tagged content and the number of successful runs.
func ingestCmd(command string, runs int, wait time.Duration) (string, int) {
	var content strings.Builder
	successCount := 0

	fmt.Fprintf(&content, "<cmd command=%q runs=%q>\n", command, strconv.Itoa(runs))

	for i := 1; i <= runs; i++ {
		if i > 1 && wait > 0 {
			outputf("  Waiting %v before run %d/%d...\n", wait.Round(time.Second), i, runs)
			time.Sleep(wait)
		}
		outputf("  Running %s (run %d/%d)...\n", command, i, runs)

		ctx, cancel := context.WithTimeout(context.Background(), cmdRunTimeout)
		cmd := exec.CommandContext(ctx, "sh", "-c", command)  //nolint:gosec // G204: user-specified command for AI context
		cmd.Stdin, _ = os.Open(os.DevNull)                    // don't inherit the terminal — prevents interactive commands from stealing input
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // isolate child process group

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		start := time.Now()
		err := cmd.Run()
		elapsed := time.Since(start).Round(time.Millisecond)
		cancel()

		now := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(&content, "--- Run %d (%s) ---\n", i, now)

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputf("  Run %d timed out after %v (command: %s)\n", i, elapsed, command)
				writeTruncated(&content, stdout.String())
				content.WriteString("[timed out]\n")
				successCount++
				continue
			}
			exitCode := -1
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}
			outputf("  Run %d exited with code %d in %v\n", i, exitCode, elapsed)
			writeTruncated(&content, stdout.String())
			if stderr.Len() > 0 {
				content.WriteString("--- stderr ---\n")
				writeTruncated(&content, stderr.String())
			}
			fmt.Fprintf(&content, "[exit code %d]\n", exitCode)
			successCount++
			continue
		}

		outputf("  Run %d completed in %v\n", i, elapsed)
		writeTruncated(&content, stdout.String())
		if stderr.Len() > 0 {
			content.WriteString("--- stderr ---\n")
			writeTruncated(&content, stderr.String())
		}
		successCount++
	}

	content.WriteString("</cmd>\n")
	return content.String(), successCount
}

// writeTruncated writes s to the builder, truncating at maxCmdOutput with a marker.
func writeTruncated(b *strings.Builder, s string) {
	if len(s) > maxCmdOutput {
		b.WriteString(s[:maxCmdOutput])
		b.WriteString("\n[truncated]\n")
		return
	}
	b.WriteString(s)
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
	if len(history) > 1 && isContextMessage(history[1]) {
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
func handleChatCommand(line string, history *[]ai.Message, trimWarned *bool, client **ai.Client, cfg *models.AIConfig, budget *int, sessionName *string) bool {
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
	case "/cmd":
		cmdCmd(args, history, *budget)
	case "/clear":
		cmdClear(args, history, trimWarned)
	case "/context":
		cmdContext(args, *history, *budget)
	case "/model":
		cmdModel(args, cfg, client)
	case "/window":
		cmdWindow(args, cfg, budget)
	case "/memory":
		cmdMemory(args, history)
	case "/save":
		cmdSave(args, *history, cfg, sessionName)
	case "/load":
		cmdLoad(args, history, cfg, client, budget, trimWarned, sessionName)
	case "/delete":
		cmdDelete(args)
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
		addContextToHistory(history, content.String())
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
		addContextToHistory(history, content.String())
		tokens := historyTokens(*history)
		pct := tokens * 100 / budget
		outputf("Added %d file(s) from %s. Context: %d/%d tokens (%d%%)\n", count, path, tokens, budget, pct)
	}
}

// cmdCmd executes a shell command multiple times and adds output to context.
// Usage: /cmd <command> [--runs <n>] [--wait <seconds>]  (runs defaults to 5, wait to 5)
func cmdCmd(args string, history *[]ai.Message, budget int) {
	if args == "" {
		outputln("Usage: /cmd <command> [--runs <n>] [--wait <seconds>]")
		outputln("  Execute a command N times (default 5) and add output to context.")
		return
	}

	command, runs, wait := parseCmdArgs(args)

	if runs <= 0 || runs > 100 {
		outputln("Runs must be between 1 and 100.")
		return
	}

	content, successCount := ingestCmd(command, runs, wait)
	if successCount == 0 {
		outputf("Command %q produced no output.\n", command)
		return
	}

	addContextToHistory(history, content)
	tokens := historyTokens(*history)
	pct := tokens * 100 / budget
	outputf("Added output from %q (%d/%d runs). Context: %d/%d tokens (%d%%)\n",
		command, successCount, runs, tokens, budget, pct)
}

// parseCmdArgs splits /cmd arguments into command, runs, and wait duration.
// If --runs <n> is present, it is extracted; otherwise runs defaults to 5.
// If --wait <seconds> is present, it is extracted; otherwise wait defaults to 5s.
func parseCmdArgs(args string) (string, int, time.Duration) {
	parts := strings.Fields(args)
	runs := 5
	wait := 5

	// Extract --runs and --wait flags, then rebuild the command string.
	runIdx := -1
	waitIdx := -1
	for i, p := range parts {
		if p == "--runs" && i+1 < len(parts) {
			if n, err := strconv.Atoi(parts[i+1]); err == nil && n > 0 {
				runs = n
				runIdx = i
			}
		}
		if p == "--wait" && i+1 < len(parts) {
			if w, err := strconv.Atoi(parts[i+1]); err == nil && w >= 0 {
				wait = w
				waitIdx = i
			}
		}
	}

	// Remove the flag tokens from the command string.
	var cmdParts []string
	skip := map[int]bool{}
	if runIdx >= 0 {
		skip[runIdx] = true
		skip[runIdx+1] = true
	}
	if waitIdx >= 0 {
		skip[waitIdx] = true
		skip[waitIdx+1] = true
	}
	for i, p := range parts {
		if !skip[i] {
			cmdParts = append(cmdParts, p)
		}
	}

	command := strings.Join(cmdParts, " ")
	return strings.TrimSpace(command), runs, time.Duration(wait) * time.Second
}

// cmdClear clears conversation history.
// With no args, keeps system prompt and file context.
// With "all", keeps only the system prompt.
func cmdClear(args string, history *[]ai.Message, trimWarned *bool) {
	if args == "all" {
		*history = (*history)[:1]
		*trimWarned = false
		outputln("Conversation, file, and command context cleared.")
		return
	}
	prefixLen := 1
	if len(*history) > 1 && isContextMessage((*history)[1]) {
		prefixLen = 2
		fileCount := len(fileContextPaths((*history)[1].Content))
		cmdCount := len(cmdContextCommands((*history)[1].Content))
		*history = (*history)[:prefixLen]
		*trimWarned = false
		parts := []string{}
		if fileCount > 0 {
			parts = append(parts, fmt.Sprintf("%d file(s)", fileCount))
		}
		if cmdCount > 0 {
			parts = append(parts, fmt.Sprintf("%d command(s)", cmdCount))
		}
		if len(parts) > 0 {
			outputf("Conversation cleared. %s still in context.\n", strings.Join(parts, " and "))
		} else {
			outputln("Conversation cleared.")
		}
		return
	}
	*history = (*history)[:prefixLen]
	*trimWarned = false
	outputln("Conversation cleared.")
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

// cmdContextRe extracts command info from the XML-tagged command context.
var cmdContextRe = regexp.MustCompile(`<cmd command="([^"]+)" runs="([^"]+)"`)

type cmdInfo struct {
	command string
	runs    string
}

func cmdContextCommands(content string) []cmdInfo {
	matches := cmdContextRe.FindAllStringSubmatch(content, -1)
	cmds := make([]cmdInfo, 0, len(matches))
	for _, m := range matches {
		cmds = append(cmds, cmdInfo{command: m[1], runs: m[2]})
	}
	return cmds
}

// cmdContext shows context window usage, turn count, and context details.
// With "all", dumps the full content of every message in history.
func cmdContext(args string, history []ai.Message, budget int) {
	if args == "all" {
		for i, m := range history {
			role := m.Role
			if role == "" {
				role = "unknown"
			}
			outputf("---- [%d] %s ----\n", i, role)
			outputln(m.Content)
		}
		return
	}

	tokens := historyTokens(history)
	pct := tokens * 100 / budget

	prefixLen := 1
	if len(history) > 1 && isContextMessage(history[1]) {
		prefixLen = 2
	}
	turns := (len(history) - prefixLen) / 2

	outputf("Context: %d/%d tokens (%d%%)\n", tokens, budget, pct)
	outputf("Turns: %d\n", turns)

	var filePaths []string
	var cmdInfos []cmdInfo
	if prefixLen == 2 {
		filePaths = fileContextPaths(history[1].Content)
		cmdInfos = cmdContextCommands(history[1].Content)
	}
	outputf("Files: %d\n", len(filePaths))
	for _, p := range filePaths {
		outputf("  %s\n", p)
	}
	outputf("Cmds:  %d\n", len(cmdInfos))
	for _, c := range cmdInfos {
		outputf("  %s (%s runs)\n", c.command, c.runs)
	}
	if mem := loadMemory(); mem != "" {
		outputf("Memory: ~%d tokens\n", ai.EstimateTokens(mem))
	}
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

// appendMemory appends a line of text to the MEMORY.md file, creating it if needed.
func appendMemory(text string) error {
	path, err := memoryPath()
	if err != nil {
		return fmt.Errorf("could not resolve memory path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: path is from trusted config resolution.
	if err != nil {
		return fmt.Errorf("could not open memory file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.WriteString(text + "\n"); err != nil {
		return fmt.Errorf("could not write to memory file: %w", err)
	}

	return nil
}

// cmdMemory handles the /memory slash command.
// With no args, displays current memory content and file path.
// With args, appends the text to MEMORY.md and updates the system prompt.
func cmdMemory(args string, history *[]ai.Message) {
	if args == "" {
		mem := loadMemory()
		path, err := memoryPath()
		if err != nil {
			outputf("Error: could not resolve memory path: %v\n", err)
			return
		}
		if mem == "" {
			outputf("Memory is empty. File: %s\n", path)
		} else {
			tokens := ai.EstimateTokens(mem)
			outputf("Memory (%s, ~%d tokens):\n%s\n", path, tokens, mem)
		}
		return
	}

	if err := appendMemory(args); err != nil {
		outputf("Error: %v\n", err)
		return
	}

	rebuildSystemPrompt(history)
	tokens := ai.EstimateTokens((*history)[0].Content)
	outputln("Memory updated.")
	outputf("System prompt: ~%d tokens\n", tokens)
}

// cmdHelp shows all available in-session commands.
func cmdHelp() {
	outputln("Commands:")
	outputln("  /file <path>                   Add a file to the conversation context")
	outputln("  /folder <path>                 Add all files in a folder to the context (non-recursive)")
	outputln("  /cmd <command> [--runs <n>] [--wait <s>]   Execute a command and add output to context (default 5 runs, 5s wait)")
	outputln("  /clear                         Clear conversation history (keeps file and command context)")
	outputln("  /clear all                     Clear conversation, file, and command context")
	outputln("  /context                       Show context window usage and context details")
	outputln("  /context all                   Dump full content of all messages in context")
	outputln("  /model                         Show current AI model information")
	outputln("  /model <name>                  Switch to a different model for this session")
	outputln("  /window                        Show context window size")
	outputln("  /window <size>                 Set context window size (e.g., /window 128000)")
	outputln("  /memory                        Show current memory content and file path")
	outputln("  /memory <text>                 Append text to memory (persists across sessions)")
	outputln("  /save <name>                   Save conversation context to a file")
	outputln("  /load                          List saved chats")
	outputln("  /load <name>                   Load a previously saved conversation")
	outputln("  /delete <name>                 Delete a saved chat session")
	outputln("  /help                          Show this help message")
	outputln("  /exit, /quit                   End the session")
	outputln()
	outputln("Type a message to chat with the AI.")
	outputln("Use // to send a message that starts with /.")
}

// addContextToHistory appends preserved context content (file or command output)
// to the history, either by extending an existing context message at index 1
// or inserting a new one.
func addContextToHistory(history *[]ai.Message, content string) {
	if len(*history) > 1 && isContextMessage((*history)[1]) {
		(*history)[1].Content += content
	} else {
		ctxMsg := ai.Message{Role: ai.RoleUser, Content: content}
		*history = append([]ai.Message{(*history)[0], ctxMsg}, (*history)[1:]...)
	}
}

// chatSession is the JSON format for saving and loading chat history.
type chatSession struct {
	Model         string       `json:"model"`
	ContextWindow int          `json:"context_window"`
	SavedAt       string       `json:"saved_at"`
	Messages      []ai.Message `json:"messages"`
}

// memoryPath returns the path to the MEMORY.md file.
// It lives in the ali config directory alongside functions.yaml and ai.yaml.
func memoryPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "MEMORY.md"), nil
}

// loadMemory reads the MEMORY.md file and returns its trimmed content.
// Returns empty string if the file does not exist or is empty.
func loadMemory() string {
	path, err := memoryPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is from trusted config resolution.
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// buildSystemPrompt constructs the full system prompt by appending the
// memory block to the base prompt. If there is no memory content, the base
// prompt is returned unchanged.
func buildSystemPrompt(base string) string {
	mem := loadMemory()
	if mem == "" {
		return base
	}
	return base + "\n\n<memory>\n" + mem + "\n</memory>"
}

// memoryBlockRe matches the <memory>...</memory> block in a system prompt,
// including any content (even multiline) between the tags.
var memoryBlockRe = regexp.MustCompile(`(?s)\n\n<memory>.*</memory>`)

// stripMemory removes a <memory>...</memory> block from a system prompt
// string, recovering the base prompt.
func stripMemory(prompt string) string {
	return strings.TrimSpace(memoryBlockRe.ReplaceAllString(prompt, ""))
}

// rebuildSystemPrompt strips any existing memory block from the system
// message and re-appends the current memory from disk. This is idempotent —
// calling it when no memory file exists leaves the system prompt unchanged.
func rebuildSystemPrompt(history *[]ai.Message) {
	if len(*history) == 0 {
		return
	}
	basePrompt := stripMemory((*history)[0].Content)
	(*history)[0].Content = buildSystemPrompt(basePrompt)
}

// chatsDir returns the directory for saved chat files, creating it if needed.
func chatsDir() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	chatDir := filepath.Join(dir, "chats")
	if err := os.MkdirAll(chatDir, 0o750); err != nil {
		return "", fmt.Errorf("could not create chats directory: %w", err)
	}
	return chatDir, nil
}

// resolveChatPath returns the full file path for a chat name or explicit path.
// A bare name (no path separators) is stored in the chats directory with .json appended.
// A path containing separators is used as-is.
func resolveChatPath(args string) (string, error) {
	if strings.Contains(args, string(os.PathSeparator)) || strings.HasPrefix(args, ".") {
		path, err := filepath.Abs(args)
		if err != nil {
			return "", fmt.Errorf("could not resolve path %s: %w", args, err)
		}
		return path, nil
	}
	dir, err := chatsDir()
	if err != nil {
		return "", err
	}
	name := args
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}
	return filepath.Join(dir, name), nil
}

// cmdSave saves the current conversation context to a file.
func cmdSave(args string, history []ai.Message, cfg *models.AIConfig, sessionName *string) {
	if args == "" {
		if *sessionName != "" {
			args = *sessionName
		} else {
			outputln("Usage: /save <name>")
			outputln("  Save conversation context to a file.")
			outputln("  Use a bare name to save in the chats directory, or a path to save elsewhere.")
			outputln("  After /load, /save with no name reuses the loaded session name.")
			return
		}
	}

	path, err := resolveChatPath(args)
	if err != nil {
		outputf("Error: %v\n", err)
		return
	}

	session := chatSession{
		Model:         cfg.Model,
		ContextWindow: cfg.ContextWindow,
		SavedAt:       time.Now().Format(time.RFC3339),
		Messages:      history,
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		outputf("Error: could not marshal session: %v\n", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		outputf("Error: could not create directory: %v\n", err)
		return
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		outputf("Error: could not save: %v\n", err)
		return
	}

	prefixLen := 1
	if len(history) > 1 && isContextMessage(history[1]) {
		prefixLen = 2
	}
	turns := (len(history) - prefixLen) / 2

	outputf("Saved %s (%d turns, ~%d tokens)\n", args, turns, historyTokens(history))
	*sessionName = args
}

// cmdLoad loads a previously saved conversation, or lists available chats if no args given.
func cmdLoad(args string, history *[]ai.Message, cfg *models.AIConfig, client **ai.Client, budget *int, trimWarned *bool, sessionName *string) {
	if args == "" {
		cmdLoadList()
		return
	}

	path, err := resolveChatPath(args)
	if err != nil {
		outputf("Error: %v\n", err)
		return
	}

	data, err := os.ReadFile(path) //nolint:gosec // G304: user-specified path for loading chat
	if err != nil {
		outputf("Error: could not read %s: %v\n", args, err)
		return
	}

	var session chatSession
	if err := json.Unmarshal(data, &session); err != nil {
		outputf("Error: could not parse %s: %v\n", args, err)
		return
	}

	if len(session.Messages) == 0 {
		outputf("Error: %s contains no messages\n", args)
		return
	}

	*history = session.Messages
	rebuildSystemPrompt(history)
	cfg.Model = session.Model
	cfg.ContextWindow = session.ContextWindow
	*client = ai.NewClient(cfg)
	*budget = ai.ContextBudget(cfg.ContextWindow)
	*trimWarned = false

	prefixLen := 1
	if len(*history) > 1 && isContextMessage((*history)[1]) {
		prefixLen = 2
	}
	turns := (len(*history) - prefixLen) / 2

	outputf("Loaded %s (%d turns, ~%d tokens)\n", args, turns, historyTokens(*history))
	outputf("Model: %s | Context window: %d tokens\n", cfg.Model, cfg.ContextWindow)
	*sessionName = args
}

// cmdLoadList lists available saved chats sorted by modification time (most recent first).
func cmdLoadList() {
	dir, err := chatsDir()
	if err != nil {
		outputf("Error: %v\n", err)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			outputln("No saved chats.")
		} else {
			outputf("Error: could not read chats directory: %v\n", err)
		}
		return
	}

	type chatEntry struct {
		name string
		time time.Time
	}
	var chats []chatEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		chats = append(chats, chatEntry{name: name, time: info.ModTime()})
	}
	sort.Slice(chats, func(i, j int) bool { return chats[i].time.After(chats[j].time) })

	if len(chats) == 0 {
		outputln("No saved chats.")
		return
	}

	outputln("Saved chats:")
	for _, c := range chats {
		outputf("  %-20s  %s\n", c.name, c.time.Format("2006-01-02 15:04"))
	}
	outputln()
	outputln("Use /load <name> to load a chat.")
}

// cmdDelete deletes a saved chat session, or lists available chats if no args given.
func cmdDelete(args string) {
	if args == "" {
		cmdLoadList()
		outputln("Use /delete <name> to delete a chat.")
		return
	}

	path, err := resolveChatPath(args)
	if err != nil {
		outputf("Error: %v\n", err)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		outputf("Chat %q not found.\n", args)
		return
	}

	if err := os.Remove(path); err != nil {
		outputf("Error: could not delete %s: %v\n", args, err)
		return
	}

	outputf("Deleted %s\n", args)
}
