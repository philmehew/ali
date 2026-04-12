# ali

```
                      -#######-
                    #############
                  ################
                  #################
                  #################
                  ##++++#---+++####
                   ++++-----++++##-
                   ++#+---##++++#--
                   +++++---+---+++
                    ++-+------+++++++++.
              ++###  ++++----++++++++--+++++
           -+++#####+#+++++++++++++#######++++++
          -+#########+#++++-++++++##########+-++++
         -##########--++++++++++#############+--+++
         -########+-+++++++++++++######+++####+--+++
         -#######+++++++++++++++++##############-+++
         -######++++++------+++-+--############.-+++
         ++####+.+-++-------+++++------#######-#-+++
          -#++..+--+++------+++++-------+-####.#-++++
          ++--++----++++----+++++------++#..-+..+++++-
          ++---+---+#+++++-++++++---++++++++#+++++++++
         ++++--++-++#+++++++++++++++++++++++#++-+-+++++
         ++++---++++  +++++++++++++--+++++++##++++++++++
         +++++---++.  +++++++++--------++++++#+++++++++++
          ++++----+   ++++++++++------+++++++#++++++++++
           ++++++++   +++++++++++-----+++++++++++++++++-
           +++++++    ++++++++++++++++++++++++   +++++.
             ++-      ##++++++++++++++-+++++#      ++.
                      ###+---+#++++++++######
                      +##-#-++..##############-
                     #-------++.############-..-
                     ----.--.... .........##....
                    -.---..--... .. ......#+#  .
```

A lightweight CLI tool for managing and executing parametric command-line snippets. A structured alternative to shell aliases and history -- no sourcing, no `.bashrc` edits.

## Install

```bash
go install ./cmd/ali
```

Or build manually:

```bash
make build
sudo make install   # copies to /usr/local/bin/ali
```

Requires Go 1.22+ (go.mod tracks the installed version).

### Add to PATH

After installing, add ali to your PATH by running:

```bash
eval "$(ali init)"
```

This detects your shell from `$SHELL` and outputs the appropriate configuration. Add it to your shell profile (`~/.zshrc`, `~/.bashrc`, etc.) to persist across sessions:

```bash
echo 'eval "$(ali init)"' >> ~/.zshrc
```

You can also specify a shell explicitly:

```bash
ali init bash   # bash
ali init zsh    # zsh
ali init fish   # fish
```

## Quick Start

```bash
# Add a function
ali add glog "git log --oneline -n \$1" -d "Pretty git log" -D "10"

# Execute it (uses default value "10")
ali glog

# Override the default
ali glog 20

# List all functions
ali list

# Edit in $EDITOR
ali edit glog

# Remove a function
ali remove glog
```

## Commands

### `ali <name> [params...]`

Execute a stored function by name. Any parameters override defaults left-to-right.

```bash
ali glog          # uses default for $1
ali glog 20       # overrides $1 with "20"
ali mygrep TODO . # supplies $1=TODO, $2=.
```

This is shorthand for `ali run <name> [params...]`.

### `ali add <name> <body>`

Add a new function.

| Flag | Short | Description |
|------|-------|-------------|
| `--desc` | `-d` | Description of the function |
| `--defaults` | `-D` | Comma-separated default values for `$1`, `$2`, etc. |

```bash
ali add glog "git log --oneline -n \$1" -d "Pretty git log" -D "10"
ali add mygrep "grep -r \"\$1\" \"\$2\"" -d "Recursive search" -D ",."
ali add hello "echo hello"           # no parameters, no defaults
```

**Note:** Use `\$1`, `\$2` in the shell to prevent shell expansion of the `$` sign. Alternatively, use single quotes:

```bash
ali add glog 'git log --oneline -n $1' -d "Pretty git log" -D "10"
```

### `ali list [keywords...]`

List stored functions in a numbered interactive list, optionally filtered by keywords. Keywords perform case-insensitive substring matching against the function name, description, and body. Multiple keywords use AND logic — all keywords must match.

| Flag | Description |
|------|-------------|
| `--ignored` | List ignored commands instead of functions |

```bash
ali list              # show all functions
ali list doc comp     # functions matching both "doc" and "comp"
ali list git          # functions matching "git"
ali list --ignored    # show ignored commands
```

```
$ ali list
   1.  glog       git log --oneline -n $1     Pretty git log
   2.  dcup       docker compose up -d         Docker compose up
   3.  dcdn       docker compose down          Docker compose down

Enter number to execute, 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: 1
...output of git log --oneline -n 10...

$ ali list
   1.  glog       git log --oneline -n $1     Pretty git log
   2.  dcup       docker compose up -d         Docker compose up
   3.  dcdn       docker compose down          Docker compose down

Enter number to execute, 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: r 2
Removed function "dcup"

   1.  glog       git log --oneline -n $1     Pretty git log
   2.  dcdn       docker compose down          Docker compose down

Enter number to execute, 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: q

$ ali list --ignored
   1.  ls
   2.  nslookup

Enter 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: r 2
Removed "nslookup" from ignore list

   1.  ls

Enter 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: q
```

- **Number** — execute that function
- **`e <num>`** — edit that function in `$EDITOR`
- **`r <num>`** — remove that function (or ignored command)
- **`q`** — quit

### `ali history`

Scan your shell history for frequently used commands and interactively add them as ali functions. Commands already in your ali config and commands on the ignore list are excluded.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--lines` | `-l` | 1000 | Number of history lines to scan |
| `--new` | `-n` | 10 | Number of top commands to show |

```bash
ali history              # scan last 1000 lines, show top 10
ali history -l 500 -n 5  # scan last 500 lines, show top 5
```

You're presented with a numbered list that rotates — when you add one, the next most-used command fills its place:

```
  1.  ssh phil@192.168.0.70          (15 times)
  2.  docker compose up -d           (8 times)
  3.  ping 192.168.0.70              (6 times)
  ...

Enter number to add, 'i <num>' to ignore, or 'q' to quit: 2
Add "docker compose up -d" as [docker]? (y/e): y
Added function "docker"

  1.  ssh phil@192.168.0.70          (15 times)
  2.  ping 192.168.0.70              (6 times)
  3.  diskutil list                  (4 times)
  ...

Enter number to add, 'i <num>' to ignore, or 'q' to quit: i 1
Ignored "ssh phil@192.168.0.70"
```

- **Number** — add that command (then confirm alias with `y` or type a custom name with `e`)
- **`i <num>`** — ignore that command (won't appear in future `ali history` runs)
- **`q`** — quit and save changes

Supports zsh (`~/.zsh_history`) and bash (`~/.bash_history`), auto-detected from `$SHELL`. Override with the `HISTFILE` environment variable.

### `ali remove <name>`

Delete a stored function.

### `ali edit <name>`

Open the function in your `$EDITOR` (defaults to `vi`). The function is written to a temporary YAML file with syntax highlighting. You can edit any field — `name`, `description`, `body`, or `defaults`. If you change the `name` field, the function will be renamed.

| Flag | Description |
|------|-------------|
| `--ignored` | Edit the ignore list instead of a function |

```bash
ali edit glog        # edit the glog function
ali edit --ignored   # edit the ignore list
```

### `ali init [shell]`

Output shell configuration to add ali to your PATH. Auto-detects your shell from `$SHELL` if no argument is given.

```bash
ali init        # auto-detect from $SHELL
ali init bash   # explicit shell
ali init zsh
ali init fish
```

The output includes step-by-step instructions:

```
# ali v1.0.0: add to your shell profile (~/.zshrc):
export PATH="/Users/you:$PATH"

# Step 1: Add to your profile:
echo 'export PATH="/Users/you:$PATH"' >> ~/.zshrc

# Step 2: Reload your profile:
source ~/.zshrc
```

**Step 1** — Add to your shell profile (persists across sessions):
```bash
echo 'export PATH="/path/to/ali:$PATH"' >> ~/.zshrc    # zsh
echo 'export PATH="/path/to/ali:$PATH"' >> ~/.bashrc   # bash
```

**Step 2** — Reload your profile for it to take effect:
```bash
source ~/.zshrc    # zsh
source ~/.bashrc   # bash
```

Or simply restart your terminal.

### `ali version`

Print the current version.

```bash
ali version
v1.0.0
```

## Parameters

Functions use positional placeholders `$1` through `$N`:

- **`$1`** — first parameter
- **`$2`** — second parameter
- **`$10`** — tenth parameter (handled correctly, `$10` is not confused with `$1`)

### Resolution order

1. Runtime arguments override defaults left-to-right.
2. If a parameter has a default, it is used when no runtime argument is supplied.
3. If a parameter has no default and no runtime argument is supplied, an error is returned.

```bash
ali add deploy "ssh \$1 'cd \$2 && git pull'"
ali deploy prod /var/www   # $1=prod, $2=/var/www

ali add glog "git log --oneline -n \$1" -D "10"
ali glog       # $1=10 (default)
ali glog 20    # $1=20 (override)
```

### Extra arguments

Arguments beyond the highest `$N` placeholder are appended to the command with proper shell escaping:

```bash
ali add find "find . -name \$1" -D "*.go"
ali find "*.go" -type f   # runs: find . -name "*.go" '-type' 'f'
```

### Shell features

The function body is executed through `/bin/sh -c`, so pipes, redirects, subshells, and other shell features all work:

```bash
ali add count "git log --oneline | wc -l"
ali add status "echo \$1 && echo \$2"
```

## Configuration

### File location

`ali` reads its config from a YAML file. The location is resolved in order:

1. **`ALI_CONFIG` environment variable** — if set, used directly
2. **Platform default** — `os.UserConfigDir()` + `ali/functions.yaml`

| Platform | Default path |
|----------|-------------|
| macOS | `~/Library/Application Support/ali/functions.yaml` |
| Linux | `~/.config/ali/functions.yaml` |

### Config format

```yaml
functions:
  - name: glog
    description: "Pretty git log"
    body: "git log --oneline -n $1"
    defaults:
      - "10"
  - name: mygrep
    description: "Recursive search"
    body: "grep -r \"$1\" \"$2\""
    defaults:
      - ""
      - "."
  - name: hello
    description: "Say hello"
    body: "echo hello"
ignore:
  - "ls"
  - "cd"
```

- **`defaults`** is optional. If omitted, all parameters are required at runtime.
- An **empty string** default (`""`) means the parameter defaults to an empty value, not that it is required.
- **`ignore`** is optional. Commands listed here are excluded from `ali history`. Add entries interactively with `i <num>` or edit the file directly.
- You can edit this file directly or use `ali edit`.

### Environment variables

| Variable | Description |
|----------|-------------|
| `ALI_CONFIG` | Override the config file path |
| `EDITOR` | Editor used by `ali edit` (defaults to `vi`) |

## Project structure

```
ali/
├── cmd/ali/main.go              # Entry point
├── internal/
│   ├── models/
│   │   ├── function.go          # AliFunction, AliConfig structs
│   │   └── function_test.go
│   ├── config/
│   │   ├── config.go            # Load, Save, Path, FindFunction
│   │   └── config_test.go
│   ├── execution/
│   │   ├── execute.go           # Parameter substitution + shell execution
│   │   └── execute_test.go
│   ├── version/
│   │   └── version.go           # Version constant (injected via -ldflags)
│   └── cli/
│       ├── root.go              # Root command + arg interception
│       ├── add.go               # ali add
│       ├── history.go           # ali history
│       ├── init.go              # ali init
│       ├── version.go           # ali version
│       ├── list.go              # ali list
│       ├── remove.go            # ali remove
│       ├── edit.go              # ali edit
│       └── run.go               # ali run (hidden)
├── go.mod
├── go.sum
└── Makefile
```

## GitHub Security

The `github-secure.sh` script configures repository security settings and branch protection using the [GitHub CLI](https://cli.github.com/) (`gh`). It requires `gh` to be authenticated with repo admin access.

**What it configures:**

- **Secret scanning** — detects secrets committed to the repo
- **Push protection** — blocks pushes containing detected secrets
- **Branch protection** — requires 1 approval, linear history, conversation resolution; enforces admin restrictions; disables force pushes and deletions

**Usage:**

```bash
./github-secure.sh
```

The script checks current settings before making changes, only enabling what isn't already active. It verifies all settings at the end and exits with an error if anything didn't apply correctly.

## Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
go test ./... -v
```
