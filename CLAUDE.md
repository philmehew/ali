# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build          # Build binary to ./ali (injects version, strips symbols)
go build ./cmd/ali  # Build without Makefile (version will be "dev")
make test           # Run all tests
go test ./... -v    # Run all tests verbose
go test ./internal/execution/ -v -run TestResolve  # Run single test
go install ./cmd/ali  # Install to $GOPATH/bin
```

Version is set in the Makefile (`VERSION=v1.0.0`) and injected via `-ldflags` at build time. The `internal/version` package holds the default `"dev"`.

Go version is tracked in `go.mod` (currently 1.24.5).

Linting: `golangci-lint run --config=.golangci.yml ./…` (config in `.golangci.yml`). Pre-commit hooks are configured in `.pre-commit-config.yaml` — install with `pre-commit install`.

## Architecture

**Entry point:** `cmd/ali/main.go` calls `cli.NewRootCmd()` then `rootCmd.Execute()`. There is no arg interception — function execution requires the explicit `run` (or `r`) subcommand: `ali run glog 20` or `ali r glog 20`.

**Command wiring:** `internal/cli/root.go` registers subcommands and a `--version` flag on the root command. Each subcommand lives in its own file. Aliases: `r` → run, `ls` → list, `rm` → remove, `mv` → move. `run` is a visible subcommand for executing stored functions — this avoids namespace collisions between subcommand names and function names. `move` reorders functions by position number or name. `init` outputs eval-able shell configuration for PATH and the wrapper function, auto-detecting from `$SHELL`. `install` appends the eval line to your rc file with the full binary path. `init` uses `cmd.SetOut(os.Stdout)` so its output always goes to stdout (not stderr via displayOut) — this is critical because `eval "$(ali init)"` captures stdout. `install` uses `outputf` for user-facing messages. `--version` (or `-v`) prints the version, commit, build date, and author. `history` scans shell history for frequent commands — presents a rotating numbered list, user adds by number or ignores with `i <num>`, `q` to quit.

**Data flow for command resolution (paste mode):**
1. `cli/run.go` calls `config.Load()` to read YAML
2. `splitChainArgs()` splits the args into one segment per command: a standalone `+` argument separates commands, and a `+` at or after `--` is an ordinary argument. `resolveChain()` then resolves each segment independently.
3. `config.FindFunction()` looks up each function by name (or `resolveFunction` by 1-based list index)
4. `execution.Resolve()` merges runtime args with defaults, substitutes `$1`..`$N` in the body (descending order to avoid `$1` matching prefix of `$10`)
5. `execution.PasteCommands()` formats the resolved commands (appending extras with shell escaping), joins them with ` ; ` on one line, writes to stdout, then exits. A shell wrapper function (set up by `ali init`) captures this output and pastes it into the shell's input buffer using `print -z` (zsh), `read -e -i` (bash), or `commandline --replace` (fish). `PasteCommand()` remains as the single-command wrapper used by interactive `ali list`.

**Output routing (isatty):** `internal/cli/output.go` detects whether stdout is a terminal. When stdout is captured (e.g., by the shell wrapper's `$()`), all display output (menus, prompts, errors) is redirected to stderr so the user still sees it. Only the resolved command goes to stdout (where the wrapper captures it). When stdout is a terminal (no wrapper), display output goes to stdout as normal.

**Config layer:** `internal/config/config.go` — `Path()` checks `ALI_CONFIG` env var first, then `os.UserConfigDir()/ali/functions.yaml`. `Save()` uses atomic writes (temp file + rename). `Load()` returns empty config if file doesn't exist.

**`ali edit` flow:** Writes the function YAML to a temp file, launches `$EDITOR` (default `vi`), reads it back, validates, and saves. Supports renaming — checks for name conflicts if the name field changes.

**`ali chat` in-session commands:** All commands use a `/` prefix (e.g., `/file`, `/help`, `/exit`). Bare `exit`/`quit` are treated as regular messages, not commands. `//` at the start escapes to send a literal `/` to the AI. The chat loop dispatches slash commands via `handleChatCommand` in `chat.go`, which returns `true` for `/exit`/`/quit` to break the loop. Context (file and command output) is stored as a single user message at `history[1]` using XML `<file path="...">` and `<cmd command="..." runs="...">` tags, detected by `isContextMessage` (checks for `<file` or `<cmd` prefix). Mid-session file/folder/command additions append to this message or insert a new one at index 1 (shifting conversation turns). `/clear` truncates history to system + context; `/clear all` truncates to system only. Both reset `trimWarned`. `/cmd <command> [--runs <n>] [--wait <s>]` executes a shell command N times (default 5, default 5s wait) via `exec.CommandContext` with process group isolation (`Setpgid: true`) and stdin from `/dev/null`. Each run has a 30s timeout; output is truncated at 100KB per run. `/model <name>` swaps the model by updating `cfg.Model` and recreating the `ai.Client`. `/window <size>` updates `cfg.ContextWindow` and recalculates the budget. `/context` shows token usage, turn count, file paths, command names, and memory token count; `/context all` dumps full message content for debugging. `/memory` shows current memory content and file path; `/memory <text>` appends text to `MEMORY.md` and updates the system prompt immediately. Memory is stored in `~/.config/ali/MEMORY.md` (uppercase, following industry convention like `CLAUDE.md`/`AGENTS.md`), loaded at startup via `buildSystemPrompt()` which appends a `<memory>...</memory>` block to the system prompt, and survives `/clear` and `/load` because it's part of the system message (index 0). `stripMemory()` and `rebuildSystemPrompt()` handle updating the memory block in the system prompt. `--memory <text>` CLI flag appends to memory before starting the session. `/save <name>` persists the full session (messages, model, context window) as JSON to `<config>/ali/chats/<name>.json`; `/load` lists saved chats; `/load <name>` restores a session, updating history, model, context window, client, and budget — `rebuildSystemPrompt` is called after loading to replace any stale memory block with current memory from disk. `/delete <name>` removes a saved chat session; `/delete` with no args lists available chats. The `--load <name>` CLI flag loads a saved session at startup, skipping the default system prompt. The `--delete <name>` CLI flag deletes a saved session and exits. Tab completion for `--load` and `--delete` lists `.json` files in the chats directory via `chatNamesCompletion` in `completion.go`. Ctrl+C during streaming closes a `done` channel before cancelling the context — the `onToken` callback checks `done` via a non-blocking `select` to stop printing immediately. Partial output is tracked in a `printed strings.Builder` and kept in history on interrupt. Error handling captures `wasCanceled := ctx.Err() == context.Canceled` before calling `cancel()` so that API errors (e.g., context window exceeded) are reported with their actual message, not masked as `[interrupted]`.

## Key Design Decisions

- Commands are subcommands (`ali add`, `ali list`, `ali run`), not flags (`ali --add`). This avoids ambiguity with function names. Function execution uses the explicit `run` (or `r`) subcommand — `ali run glog 20` or `ali r glog 20` — so function names can never collide with subcommand names.
- `ali list` accepts optional keywords for filtering — case-insensitive substring match across name/description/body, AND logic for multiple keywords.
- `$1`..`$N` placeholders are replaced before the shell sees the string, so they don't collide with shell positional parameters.
- `Defaults: []string{""}` means "default is empty string"; `Defaults: nil` (omitted) means "parameter is required".
- `AliConfig.Ignore` is a list of command strings excluded from `ali history`. Users add entries interactively with `i <num>` or by editing the config directly.
- Resolved commands are **printed to stdout**, not executed. The shell wrapper (set up by `ali init`) captures the output and pastes it into the shell's input buffer for the user to verify/edit before pressing Enter.
- The shell wrapper is a one-liner per shell (e.g., `ali() { print -z -- "$(command ali "$@")"; }` for zsh). `ali init` generates the appropriate wrapper for each shell.
- `PasteCommand()` calls `os.Exit(0)` because the resolved command must go to stdout without further Cobra processing.
- Display output uses `outputf`/`outputln`/`output` helpers from `output.go` that route to stderr when stdout is captured (isatty check). This ensures interactive menus and prompts remain visible when the shell wrapper captures stdout.
