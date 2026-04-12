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

**Entry point:** `cmd/ali/main.go` calls `cli.NewRootCmd()` then `cli.InterceptArgs()` to rewrite `os.Args` before Cobra sees them. If the first non-flag arg isn't a known subcommand, `"run"` is inserted — so `ali glog 20` becomes `ali run glog 20`.

**Command wiring:** `internal/cli/root.go` registers subcommands (add, history, init, list, remove, edit, run, version). Each subcommand lives in its own file. `run` is hidden — users invoke it implicitly by passing a function name as the first arg. `init` outputs shell config to add ali to PATH, auto-detecting from `$SHELL`. `version` prints the version injected at build time. `history` scans shell history for frequent commands — presents a rotating numbered list, user adds by number or ignores with `i <num>`, `q` to quit.

**Data flow for execution:**
1. `cli/run.go` calls `config.Load()` to read YAML
2. `config.FindFunction()` looks up the function by name
3. `execution.Resolve()` merges runtime args with defaults, substitutes `$1`..`$N` in the body (descending order to avoid `$1` matching prefix of `$10`)
4. `execution.Execute()` appends extra args with single-quote escaping, then runs via `exec.Command("/bin/sh", "-c", cmdStr)` — this is how pipes/redirects work

**Config layer:** `internal/config/config.go` — `Path()` checks `ALI_CONFIG` env var first, then `os.UserConfigDir()/ali/functions.yaml`. `Save()` uses atomic writes (temp file + rename). `Load()` returns empty config if file doesn't exist.

**`ali edit` flow:** Writes the function YAML to a temp file, launches `$EDITOR` (default `vi`), reads it back, validates, and saves. Supports renaming — checks for name conflicts if the name field changes.

## Key Design Decisions

- Commands are subcommands (`ali add`, `ali list`), not flags (`ali --add`). This avoids ambiguity with function names.
- `ali list` accepts optional keywords for filtering — case-insensitive substring match across name/description/body, AND logic for multiple keywords.
- `$1`..`$N` placeholders are replaced before the shell sees the string, so they don't collide with shell positional parameters.
- `Defaults: []string{""}` means "default is empty string"; `Defaults: nil` (omitted) means "parameter is required".
- `AliConfig.Ignore` is a list of command strings excluded from `ali history`. Users add entries interactively with `i <num>` or by editing the config directly.
