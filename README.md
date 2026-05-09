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

A lightweight CLI tool for managing parametric command-line snippets. A structured alternative to shell aliases and history — resolved commands are pasted into your shell for editing before execution.

**macOS and Linux only.** ali relies on `/bin/sh` for command execution and Unix shell conventions — it does not run on Windows.

## Install

```bash
go install ./cmd/ali
```

Or build manually:

```bash
make build
sudo make install   # copies to /usr/local/bin/ali
```

Requires Go 1.24+ (go.mod tracks the installed version).

### Shell integration

Run `ali init --install` to add shell integration to your rc file. It auto-detects your shell from `$SHELL`, or you can specify one explicitly:

```bash
ali init --install        # auto-detect shell and add to rc file
ali init --install zsh   # add zsh setup to ~/.zshrc
ali init --install bash  # add bash setup to ~/.bashrc
ali init --install fish  # add fish setup to config.fish
```

Then reload your shell:

```bash
source ~/.zshrc
```

This adds an `eval` line to your rc file that sets up PATH and a shell wrapper function on every startup. The wrapper pastes resolved commands into your input line for editing before execution — so `ali glog 20` puts `git log --oneline -n 20` on your command line, ready to press Enter.

## Quick Start

```bash
# Add a function
ali add glog "git log --oneline -n \$1" -d "Pretty git log" -D "10"

# Run it (uses default value "10")
ali glog

# Override the default
ali glog 20

# List all functions
ali list

# Run, edit, or remove by number (from ali list)
ali 3          # run function #3
ali edit 3     # edit function #3
ali rm 3       # remove function #3

# Edit in $EDITOR
ali edit glog

# Remove a function
ali remove glog
```

## Commands

### `ali <name> [params...]`

Resolve a stored function and paste it into your command line. You can also use the number shown in `ali list` instead of a name. Any parameters override defaults left-to-right.

```bash
ali glog          # uses default for $1
ali glog 20       # overrides $1 with "20"
ali 3             # resolve function #3 from ali list
ali mygrep TODO . # supplies $1=TODO, $2=.
```

This is shorthand for `ali run <name> [params...]`.

### `ali add <name> <body>`

Add a new function. Function names must not be purely numeric (to avoid ambiguity with the numbered references used by `ali list`).

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

### `ali list [keywords...]` (alias: `ls`)

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

Enter number to resolve, 'e <num>' to edit, 'm <from> <to>' to move, 'r <num>' to remove, or 'q' to quit: 1

$ ali list
   1.  glog       git log --oneline -n $1     Pretty git log
   2.  dcup       docker compose up -d         Docker compose up
   3.  dcdn       docker compose down          Docker compose down

Enter number to resolve, 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: r 2
Removed function "dcup"

   1.  glog       git log --oneline -n $1     Pretty git log
   2.  dcdn       docker compose down          Docker compose down

Enter number to execute, 'e <num>' to edit, 'm <from> <to>' to move, 'r <num>' to remove, or 'q' to quit: q

$ ali list --ignored
   1.  ls
   2.  nslookup

Enter 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: r 2
Removed "nslookup" from ignore list

   1.  ls

Enter 'e <num>' to edit, 'r <num>' to remove, or 'q' to quit: q
```

- **Number** — resolve and paste that function
- **`e <num>`** — edit that function in `$EDITOR`
- **`m <from> <to>`** — move a function to a new position
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

### `ali remove <name>` (alias: `rm`)

Delete a stored function. Accepts a function name or its number from `ali list`.

```bash
ali remove glog    # remove by name
ali rm glog        # same thing
ali rm 3           # remove function #3
```

### `ali edit [name]`

Open a stored function in your `$EDITOR` (defaults to `vi`). The function is written to a temporary YAML file with syntax highlighting. You can edit any field — `name`, `description`, `body`, or `defaults`. If you change the `name` field, the function will be renamed.

Without a name, opens the entire config file for direct editing. Accepts a number from `ali list` instead of a name.

| Flag | Description |
|------|-------------|
| `--ignored` | Edit the ignore list instead of a function |

```bash
ali edit glog        # edit the glog function
ali edit 3           # edit function #3
ali edit              # edit the entire config file
ali edit --ignored   # edit the ignore list
```

### `ali move <from> <to>` (alias: `mv`)

Move a function to a new position in the list. Both `<from>` and `<to>` can be a number (from `ali list`) or a function name. `<to>` must be a position number.

```bash
ali move 5 1      # move function #5 to position #1
ali mv glog 1     # move glog to position #1
```

### `ali init [shell]`

Output shell integration code for ali. This is called by the `eval` line in your rc file (set up by `ali init --install`). It auto-detects your shell from `$SHELL` if no argument is given.

```bash
ali init        # auto-detect from $SHELL
ali init bash   # explicit shell
ali init zsh
ali init fish
```

To add the eval line to your rc file instead, use `--install`:

```bash
ali init --install        # auto-detect and add to rc file
ali init --install zsh    # add to ~/.zshrc
```

### `ali version`

Print the current version.

```bash
ali version
v1.2.0
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
ali find "*.go" -type f   # resolves: find . -name "*.go" '-type' 'f'
```

### Shell features

The function body is executed through `/bin/sh -c`, so pipes, redirects, subshells, and other shell features all work:

```bash
ali add count "git log --oneline | wc -l"
ali add status "echo \$1 && echo \$2"
```

Resolved commands are pasted into your shell's input line for editing before execution — not executed directly. This requires shell integration (set up via `ali init --install`).

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
- You can edit this file directly, or use `ali edit` (which also validates the file on save).

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
│   │   ├── execute.go           # Parameter substitution + command output
│   │   └── execute_test.go
│   ├── version/
│   │   └── version.go           # Version constant (injected via -ldflags)
│   └── cli/
│       ├── root.go              # Root command + arg interception
│       ├── add.go               # ali add
│       ├── move.go              # ali move (alias: mv)
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
- **Branch protection** — requires PRs, linear history, conversation resolution; enforces admin restrictions; disables force pushes and deletions

**Usage:**

```bash
./github-secure.sh
```

The script checks current settings before making changes, only enabling what isn't already active. It verifies all settings at the end and exits with an error if anything didn't apply correctly.

## Development & Release

### Development workflow

Branch protection is enforced on `main` — all changes must go through a pull request.

```bash
# 1. Create a branch
git checkout -b chore/my-change

# 2. Make changes, then test locally
make test
make build

# 3. Commit and push the branch
git add -A
git commit -m "chore: description of change"
git push -u origin chore/my-change

# 4. Open a pull request
gh pr create

# 5. Merge the PR via GitHub, then pull the merged state
git checkout main
git pull

# 6. Clean up the branch
git branch -d chore/my-change               # local
git push origin --delete chore/my-change     # remote
```

CI runs `go test` and `go vet` automatically on pull requests.

### Releasing

Releases are built automatically by [GoReleaser](https://goreleaser.com/) when a version tag is pushed to `main`. Tags must be created **after** the PR is merged — otherwise the tag points to the branch commit, not main.

```bash
# 1. Make sure main is up to date after a PR merge
git checkout main
git pull

# 2. Tag the release
git tag v1.2.0

# 3. Push the tag — this triggers the release pipeline
git push origin v1.2.0
```

The release pipeline:

1. Builds binaries for linux/darwin on amd64 and arm64
2. Packages them as `.tar.gz`
3. Generates SHA256 checksums
4. Creates a GitHub Release with all artefacts attached

No GitHub secrets configuration is needed — the default `GITHUB_TOKEN` handles release creation.

To have GitHub automatically delete remote branches on merge, enable **Settings > General > Pull Requests > Automatically delete head branches**.

### Release artefacts

| OS | Arch | Format |
|----|------|--------|
| Linux | amd64, arm64 | `.tar.gz` |
| macOS | amd64, arm64 | `.tar.gz` |

## Dependencies

| Package | Purpose |
|---------|---------|
| [github.com/spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML config parsing |

## Running tests

```bash
go test ./... -v
```
