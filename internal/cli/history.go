package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

// cmdFreq holds a command string and its frequency count.
type cmdFreq struct {
	command string
	count   int
}

func newHistoryCmd() *cobra.Command {
	var lines int
	var newCount int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Import frequent commands from shell history",
		Long: `Scan your shell history for frequently used commands and interactively
add them as ali functions.

Commands already in your ali config and commands on the ignore list are
excluded. The list rotates — when you add one, the next candidate fills
its place. Type a number to add, "i" to ignore, or "q" to quit.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			histPath, err := historyPath()
			if err != nil {
				return err
			}

			commands, err := parseHistory(histPath, lines)
			if err != nil {
				return fmt.Errorf("could not parse history: %w", err)
			}

			if len(commands) == 0 {
				fmt.Println("No commands found in history.")
				return nil
			}

			// Count frequency of each exact command string.
			freq := make(map[string]int)
			for _, cmd := range commands {
				freq[cmd]++
			}

			// Load ali config.
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			existingBodies := make(map[string]bool)
			existingNames := make(map[string]bool)
			for _, fn := range cfg.Functions {
				existingBodies[fn.Body] = true
				existingNames[fn.Name] = true
			}

			ignoreSet := make(map[string]bool)
			for _, cmd := range cfg.Ignore {
				ignoreSet[cmd] = true
			}

			// Build sorted candidate list: exclude already-added and ignored.
			var candidates []cmdFreq
			for cmd, count := range freq {
				if existingBodies[cmd] || ignoreSet[cmd] {
					continue
				}
				candidates = append(candidates, cmdFreq{cmd, count})
			}
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].count > candidates[j].count
			})

			if len(candidates) == 0 {
				fmt.Println("All frequent commands are already in ali or ignored.")
				return nil
			}

			fmt.Printf("Scanning last %d lines of %s...\n\n", lines, histPath)

			reader := bufio.NewReader(os.Stdin)
			added := 0
			ignored := 0
			cursor := 0 // index into candidates

			for {
				// Display the current window of up to newCount candidates.
				fmt.Println("Top commands not yet in ali:")
				fmt.Println()

				end := cursor + newCount
				if end > len(candidates) {
					end = len(candidates)
				}

				if cursor >= len(candidates) {
					fmt.Println("No more commands in history.")
					break
				}

				window := candidates[cursor:end]
				for i, cf := range window {
					fmt.Printf("  %2d.  %-40s (%d times)\n", i+1, cf.command, cf.count)
				}

				fmt.Println()
				fmt.Print("Enter number to add, 'i <num>' to ignore, or 'q' to quit: ")

				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if input == "" {
					continue
				}

				if strings.ToLower(input) == "q" {
					break
				}

				// Parse "i <num>" to ignore.
				if strings.HasPrefix(strings.ToLower(input), "i ") {
					numStr := strings.TrimSpace(input[2:])
					num := 0
					if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil || num < 1 || num > len(window) {
						fmt.Println("Invalid number.")
						continue
					}
					cmd := window[num-1].command
					cfg.Ignore = append(cfg.Ignore, cmd)
					ignoreSet[cmd] = true
					// Remove from candidates.
					candidates = removeFromCandidates(candidates, cursor+num-1)
					ignored++
					fmt.Printf("Ignored %q\n\n", cmd)
					continue
				}

				// Parse number to add.
				num := 0
				if _, err := fmt.Sscanf(input, "%d", &num); err != nil || num < 1 || num > len(window) {
					fmt.Println("Enter a number, 'i <num>', or 'q'.")
					continue
				}

				cf := window[num-1]
				suggested := suggestAlias(cf.command, existingNames)

				fmt.Printf("Add %q as [%s]? (y/e): ", cf.command, suggested)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))

				var alias string
				switch confirm {
				case "y", "yes", "":
					alias = suggested
				case "e", "edit":
					fmt.Print("Alias name: ")
					nameInput, _ := reader.ReadString('\n')
					alias = strings.TrimSpace(nameInput)
					if alias == "" {
						fmt.Println("Skipped (no name provided).")
						continue
					}
					if existingNames[alias] {
						fmt.Printf("Alias %q already exists. Skipped.\n\n", alias)
						continue
					}
				default:
					fmt.Println("Skipped.")
					continue
				}

				cfg.Functions = append(cfg.Functions, models.AliFunction{
					Name: alias,
					Body: cf.command,
				})
				existingNames[alias] = true
				existingBodies[cf.command] = true
				// Remove from candidates so the next one fills the gap.
				candidates = removeFromCandidates(candidates, cursor+num-1)
				added++
				fmt.Printf("Added function %q\n\n", alias)
			}

			// Save if anything changed.
			if added > 0 || ignored > 0 {
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("could not save config: %w", err)
				}
				if added > 0 {
					fmt.Printf("Added %d function(s).\n", added)
				}
				if ignored > 0 {
					fmt.Printf("Ignored %d command(s).\n", ignored)
				}
			} else {
				fmt.Println("No changes made.")
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&lines, "lines", "l", 1000, "number of history lines to scan")
	cmd.Flags().IntVarP(&newCount, "new", "n", 10, "number of top commands to present")

	return cmd
}

// removeFromCandidates removes the item at index i from the slice.
func removeFromCandidates(candidates []cmdFreq, i int) []cmdFreq {
	return append(candidates[:i], candidates[i+1:]...)
}

// historyPath resolves the shell history file path.
// Checks HISTFILE env var first, then auto-detects from $SHELL.
func historyPath() (string, error) {
	if p := os.Getenv("HISTFILE"); p != "" {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	shellPath := os.Getenv("SHELL")
	switch {
	case strings.Contains(shellPath, "zsh"):
		return filepath.Join(home, ".zsh_history"), nil
	case strings.Contains(shellPath, "bash"):
		return filepath.Join(home, ".bash_history"), nil
	default:
		return filepath.Join(home, ".zsh_history"), nil
	}
}

// zshMetaRe matches the zsh extended history prefix: `: timestamp:duration;`
var zshMetaRe = regexp.MustCompile(`^: \d+:\d+;`)

// parseHistory reads the last N lines of a history file and returns
// cleaned command strings.
func parseHistory(path string, lines int) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: history path is from trusted env/shell detection.
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	allLines := strings.Split(string(data), "\n")

	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}
	tail := allLines[start:]

	var commands []string
	var current strings.Builder

	for _, line := range tail {
		cleaned := zshMetaRe.ReplaceAllString(line, "")

		if strings.HasSuffix(cleaned, "\\") {
			current.WriteString(strings.TrimSuffix(cleaned, "\\"))
			current.WriteString(" ")
			continue
		}

		current.WriteString(cleaned)
		cmd := strings.TrimSpace(current.String())
		current.Reset()

		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	if current.Len() > 0 {
		cmd := strings.TrimSpace(current.String())
		if cmd != "" {
			commands = append(commands, cmd)
		}
	}

	return commands, nil
}

// suggestAlias generates an alias from the first word of a command.
// If the alias is taken, appends incrementing numbers until unique.
func suggestAlias(body string, existing map[string]bool) string {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return "cmd"
	}

	base := strings.ToLower(fields[0])

	if strings.Contains(base, "/") {
		base = filepath.Base(base)
	}

	name := base
	suffix := 1
	for existing[name] {
		suffix++
		name = fmt.Sprintf("%s%d", base, suffix)
	}

	return name
}
