package cli

import (
	"fmt"
	"io"
	"os"
)

// displayOut is where menus, prompts, and informational output go.
// When stdout is captured by a shell wrapper (not a terminal), this is
// stderr so the user still sees interactive output. Otherwise, it's stdout.
var displayOut io.Writer = os.Stdout

func init() {
	if !isTerminal(os.Stdout) {
		displayOut = os.Stderr
	}
}

// isTerminal reports whether f is a character device (i.e., a terminal).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// outputf formats and writes to displayOut.
func outputf(format string, args ...any) {
	_, _ = fmt.Fprintf(displayOut, format, args...)
}

// outputln writes a line to displayOut.
func outputln(args ...any) {
	_, _ = fmt.Fprintln(displayOut, args...)
}

// output writes to displayOut.
func output(args ...any) {
	_, _ = fmt.Fprint(displayOut, args...)
}
