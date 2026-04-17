// Package execution handles parameter substitution and command formatting
// for ali function bodies.
package execution

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/philmehew/ali/internal/models"
)

// ResolvedCommand holds the result of parameter substitution.
type ResolvedCommand struct {
	Command string   // body with $1, $2, ... replaced
	Extras  []string // runtime args beyond highest placeholder
}

var placeholderRe = regexp.MustCompile(`\$([1-9][0-9]*)`)

// maxPlaceholderIndex scans the body for $1..$N and returns the highest N found.
// Returns 0 if there are no placeholders.
func maxPlaceholderIndex(body string) int {
	matches := placeholderRe.FindAllStringSubmatch(body, -1)
	highest := 0
	for _, m := range matches {
		n, _ := strconv.Atoi(m[1])
		if n > highest {
			highest = n
		}
	}
	return highest
}

// Resolve merges runtime args with defaults and substitutes placeholders in the body.
func Resolve(fn *models.AliFunction, args []string) (*ResolvedCommand, error) {
	maxIdx := maxPlaceholderIndex(fn.Body)
	if maxIdx == 0 {
		// No placeholders — just use the body as-is, all args are extras.
		return &ResolvedCommand{Command: fn.Body, Extras: args}, nil
	}

	params := make([]string, maxIdx)
	for i := 1; i <= maxIdx; i++ {
		if i-1 < len(args) {
			// Runtime arg takes precedence.
			params[i-1] = args[i-1]
		} else if i-1 < len(fn.Defaults) {
			// Fall back to default.
			params[i-1] = fn.Defaults[i-1]
		} else {
			return nil, fmt.Errorf("parameter $%d is required and has no default", i)
		}
	}

	var extras []string
	if maxIdx < len(args) {
		extras = args[maxIdx:]
	}
	command := substitute(fn.Body, params)

	return &ResolvedCommand{Command: command, Extras: extras}, nil
}

// substitute replaces $1, $2, ... in the body with the given params.
// It replaces from the highest index down to avoid $1 matching the prefix of $10.
func substitute(body string, params []string) string {
	result := body
	for i := len(params); i >= 1; i-- {
		result = strings.ReplaceAll(result, "$"+strconv.Itoa(i), params[i-1])
	}
	return result
}

// shellArgEscape wraps a string in single quotes for safe shell interpolation.
func shellArgEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// CommandString returns the fully resolved command string, with extra
// arguments appended and shell-escaped, ready for display or pasting.
func CommandString(resolved *ResolvedCommand) string {
	cmdStr := resolved.Command
	if len(resolved.Extras) > 0 {
		parts := []string{cmdStr}
		for _, e := range resolved.Extras {
			parts = append(parts, shellArgEscape(e))
		}
		cmdStr = strings.Join(parts, " ")
	}
	return cmdStr
}

// PasteCommand writes the resolved command to stdout and exits.
// In shell integration mode (stdout is captured by the wrapper),
// this outputs the command for the wrapper to paste into the
// shell's input buffer. Without the wrapper, the command is
// simply displayed for the user.
func PasteCommand(resolved *ResolvedCommand) {
	_, _ = fmt.Fprintln(os.Stdout, CommandString(resolved))
	os.Exit(0)
}
