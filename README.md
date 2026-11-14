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

A lightweight CLI tool for managing parametric command-line snippets with AI-powered command generation and chat. A structured alternative to shell aliases and history — resolved commands are pasted into your shell for editing before execution.

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

Requires Go 1.26+ (go.mod tracks the installed version).

### Shell integration

Run `ali install` to add shell integration to your rc file. It auto-detects your shell from `$SHELL`, or you can specify one explicitly:

```bash
ali install        # auto-detect shell and add to rc file
ali install zsh   # add zsh setup to ~/.zshrc
ali install bash  # add bash setup to ~/.bashrc
ali install fish  # add fish setup to config.fish
```

Then reload your shell:

```bash
source ~/.zshrc
```

This adds an `eval` line to your rc file that sets up PATH and a shell wrapper function on every startup. The wrapper pastes resolved commands into your input line for editing before execution — so `ali run glog 20` puts `git log --oneline -n 20` on your command line, ready to press Enter. If the paste mechanism fails (e.g. in a non-interactive shell), the resolved command is printed to stdout as a fallback.

Tab completion for function names, subcommands, and default parameter values is set up automatically by `ali install`. After installing, press Tab to complete:

```bash
ali run gl<Tab>       # completes to: ali run glog
ali r gl<Tab>         # same thing, using the 'r' alias
ali <Tab><Tab>         # lists all subcommands
ali run <Tab><Tab>     # lists all function names
ali run glog <Tab>     # completes default value for $1 (e.g. "10")
```

## Quick Start

```bash
# Add a function
ali add glog "git log --oneline -n \$1" -d "Pretty git log" -D "10"

# Run it (uses default value "10")
ali run glog

# Override the default
ali run glog 20

# Use the short alias
ali r glog 20

# Chain multiple functions with a standalone "+"
ali r dockup + cellog

# List all functions
ali list

# Run, edit, or remove by number (from ali list)
ali run 3      # run function #3
ali edit 3     # edit function #3
ali rm 3       # remove function #3

# Edit in $EDITOR
ali edit glog

# Remove a function
ali remove glog

# Ask the AI to generate a command
ali ai find all go files larger than 1MB    # pastes: find . -name "*.go" -size +1M

# Chat with the AI (interactive)
ali chat

# Chat one-shot — response goes to stdout
ali chat "what is 2+2"

# Pipe content as context
cat main.go | ali chat "explain this" > response.txt
```

## Commands

### `ali run <name> [params...]` (alias: `r`)

Resolve a stored function and paste it into your command line. You can also use the number shown in `ali list` instead of a name. Any parameters override defaults left-to-right.

```bash
ali run glog          # uses default for $1
ali run glog 20       # overrides $1 with "20"
ali r glog 20         # same thing, using the 'r' alias
ali run 3             # resolve function #3 from ali list
ali run mygrep TODO . # supplies $1=TODO, $2=.
```

#### Chaining commands

Separate commands with a standalone `+` to paste them as one editable line:

```bash
ali r dockup + cellog
# pastes: docker compose up -d ; docker compose logs -f cell

ali r glog 20 + dockup    # parameters stay with their own command
ali r 3 + 5               # mix list numbers and names
```

- `+` must be its own argument (spaces on both sides). `+5`, `5+`, and `a+b` are treated as ordinary parameters, not separators.
- Each command is resolved independently, using its own parameters and defaults.
- Chained commands are joined with ` ; `, so every command runs even if an earlier one fails.
- `--` disables chaining for every argument after it, which also lets you pass a literal `+` or dash-leading extras:

```bash
ali r myfn -- +            # passes a literal "+" as an argument
ali r find "*.go" -- -type f
```

### `ali ai <query>`

Generate a shell command from a natural language query using an AI service. The response is pasted into your command line for editing before execution — just like `ali run`.

```bash
ali ai how do I check disk size              # pastes: df -h
ali ai find all go files larger than 1MB      # pastes: find . -name "*.go" -size +1M
ali ai kill process on port 8080              # pastes: lsof -ti:8080 | xargs kill -9
```

On first run, an interactive setup wizard will prompt for your API key (with hidden input on terminals), host URL (validated), and model. These are saved to `~/.config/ali/ai.yaml` (or `~/Library/Application Support/ali/ai.yaml` on macOS).

### `ali chat`

Start an interactive chat session with an AI assistant. Responses are streamed in real time.

**Piping and one-shot mode:** Piped stdin is automatically added as context (wrapped in `<stdin>` tags). Positional arguments form the initial prompt. When a prompt is provided, `ali chat` sends it, streams the response to stdout, and exits — making it easy to pipe or redirect. Without a prompt, `ali chat` enters interactive mode (using `/dev/tty` if stdin was piped).

| Flag | Description |
|------|-------------|
| `--file` | Include a file's contents in the conversation context (repeatable) |
| `--folder` | Include all files in a folder in the conversation context (repeatable, non-recursive) |
| `--cmd` | Execute a command and include output in context (repeatable) |
| `--runs` | Number of times to run each `--cmd` (default 5) |
| `--wait` | Seconds to wait between `--cmd` runs (default 5) |
| `--load` | Load a previously saved chat session by name |
| `--delete` | Delete a saved chat session by name |
| *args* | Positional arguments form the prompt (one-shot mode) |

```bash
ali chat
ali chat — type /help for commands, /exit to quit

[5%] > what does the -exec flag do in find?
The -exec flag allows you to run a command on each file found...
[8%] > how do I use xargs instead?
xargs reads items from stdin and runs a command for each one...
[12%] > /file main.go
Added /home/user/project/main.go. Context: 2500/123904 tokens (2%)
[14%] > explain the ServeHTTP method in this file
The ServeHTTP method on line 42 handles incoming requests...
[30%] > /cmd "ps aux" --runs 3
Running ps aux (run 1/3)...
Run 1 completed in 12ms
...
Added output from "ps aux" (3/3 runs). Context: 18000/123904 tokens (14%)
[18%] > what's using the most memory?
Based on the ps aux output, the process using the most memory is...
[22%] > /clear
Conversation cleared. 1 file(s) and 1 command(s) still in context.
[5%] > /exit
```

Use `--file` and `--folder` to include code, configs, or logs in the context so the AI can answer questions about them:

```bash
ali chat --file main.go --file config.yaml
ali chat --folder ./src
ali chat --folder ./src --file README.md
ali chat --cmd "top" --runs 5 --wait 3
ali chat --cmd "ps aux" --file server.log
ali chat --load mysession
ali chat --delete mysession

# One-shot mode — response goes to stdout
ali chat "what is 2+2"

# Piped stdin as context + one-shot + redirect
cat main.go | ali chat "explain this function" > response.txt
curl -s https://example.com/api/status | ali chat "is the service healthy?"
```

Files larger than 1MB and binary files are skipped with a warning. Command output is truncated at 100KB per run with a `[truncated]` marker. When the conversation exceeds the context window budget, the oldest messages are trimmed automatically.

**In-session commands:**

| Command | Description |
|---------|-------------|
| `/file <path>` | Add a file to the conversation context |
| `/folder <path>` | Add all files in a folder (non-recursive) |
| `/cmd <command> [--runs <n>] [--wait <s>]` | Execute a command N times and add output to context (default 5 runs, 5s wait) |
| `/clear` | Clear conversation history (keeps file and command context) |
| `/clear all` | Clear conversation, file, and command context |
| `/context` | Show token usage, budget, turn count, and context details |
| `/context all` | Dump full content of all messages in context (debug) |
| `/model` | Show current model, URL, and context window |
| `/model <name>` | Switch to a different model for this session |
| `/window` | Show context window size |
| `/window <size>` | Set context window size (e.g., `/window 128000`) |
| `/save <name>` | Save conversation context to a file |
| `/load` | List saved chats |
| `/load <name>` | Load a previously saved conversation |
| `/delete <name>` | Delete a saved chat session |
| `/help` | Show all commands |
| `/exit`, `/quit` | End the session |

Use `//` to send a message that starts with `/` (e.g., `//file not a command`).

Press Ctrl+C during a response to interrupt the AI and keep the partial output in the conversation. Chat history is kept in memory for the session but not persisted — use `/save` to persist a conversation and `/load` to restore it later.

### `ali add <name> <body>`

Add a new function. Function names must start with a letter or underscore and contain only letters, digits, hyphens, and underscores. Names must not be purely numeric (to avoid ambiguity with the numbered references used by `ali list`). Function bodies must not be empty or whitespace-only.

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

Supports zsh (`~/.zsh_history`) and bash (`~/.bash_history`), auto-detected from `$SHELL`. Override with the `HISTFILE` environment variable. History is read efficiently using a ring buffer, so large history files don't require loading the entire file into memory.

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

Output shell integration code for ali. This is called by the `eval` line in your rc file (set up by `ali install`). It auto-detects your shell from `$SHELL` if no argument is given.

```bash
ali init        # auto-detect from $SHELL
ali init bash   # explicit shell
ali init zsh
ali init fish
```

To add the eval line to your rc file automatically, use `ali install` instead.

### `ali install [shell]`

Add the ali eval line to your shell's rc file. It auto-detects your shell from `$SHELL` if no argument is given. The eval line is marked with `# >>> ali >>>` / `# <<< ali <<<` delimiters for reliable replacement on subsequent runs. Previously installed lines with the older `# ali` marker are also detected and upgraded.

```bash
ali install        # auto-detect and add to rc file
ali install zsh   # add to ~/.zshrc
ali install bash  # add to ~/.bashrc
```

### `ali --version`

Print the current version.

```bash
ali --version
ali v1.2.0
commit: abc1234
built:  2025-01-01T00:00:00Z
author: Your Name
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
ali run deploy prod /var/www   # $1=prod, $2=/var/www

ali add glog "git log --oneline -n \$1" -D "10"
ali run glog       # $1=10 (default)
ali run glog 20    # $1=20 (override)
```

### Extra arguments

Arguments beyond the highest `$N` placeholder are appended to the command with proper shell escaping. Use `--` before dash-leading extras so they are not parsed as flags:

```bash
ali add find "find . -name \$1" -D "*.go"
ali run find "*.go" -- -type f   # resolves: find . -name '*.go' '-type' 'f'
```

A standalone `+` starts a chained command (see above); put it after `--` to pass it as a literal argument.

### Shell escaping

All parameter values and extra arguments are shell-escaped before substitution, preventing injection through special characters. Values containing spaces, quotes, or other shell metacharacters are safely quoted:

```bash
ali add greet "echo \$1"
ali run greet "hello world"   # resolves: echo 'hello world'

ali add deploy "ssh \$1 'cd \$2 && git pull'"
ali run deploy "user; rm -rf /" /safe/path   # resolves: ssh 'user; rm -rf /' 'cd /safe/path && git pull'
```

### Shell features

The function body is executed through `/bin/sh -c`, so pipes, redirects, subshells, and other shell features all work:

```bash
ali add count "git log --oneline | wc -l"
ali run count
ali add status "echo \$1 && echo \$2"
ali run status hello world
```

Resolved commands are pasted into your shell's input line for editing before execution — not executed directly. This requires shell integration (set up via `ali install`).

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
| `ALI_AI_CONFIG` | Override the AI config file path |
| `ALI_AI_URL` | Override the AI service URL |
| `ALI_AI_API_KEY` | Override the AI API key (takes priority over config file) |
| `ALI_AI_MODEL` | Override the AI model name |
| `ALI_AI_CONTEXT_WINDOW` | Override the context window size (tokens) |
| `EDITOR` | Editor used by `ali edit` (defaults to `vi`) |

### AI configuration

AI settings are stored in a separate config file (`ai.yaml`) in the same directory as `functions.yaml`. If the file doesn't exist, defaults are used and `ali ai`/`ali chat` will launch a setup wizard on first run.

```yaml
url: https://api.openai.com/v1
api_key: sk-...
model: gpt-4o-mini
ai_prompt: "The user is asking for one or more specific shell commands..."
chat_prompt: "You are a helpful assistant."
context_window_tokens: 128000
```

- **`ai_prompt`** — system prompt for `ali ai`. Defaults to instructing the AI to respond with shell commands only.
- **`chat_prompt`** — system prompt for `ali chat`. Defaults to "You are a helpful assistant."
- **`context_window_tokens`** — context window size in tokens. Defaults to 128000 (gpt-4o-mini). Used by `ali chat` to manage conversation history and trim old messages when the budget is exceeded.

Transient errors (rate limits and server errors) are retried automatically with exponential backoff — up to 2 retries with 1s and 2s delays. Authentication and other non-retryable errors fail immediately.

Environment variables take priority over the config file, which is useful for keeping API keys out of dotfiles:

```bash
export ALI_AI_API_KEY=sk-...
export ALI_AI_URL=https://my-server.com/v1
```

## Security

### Command execution model

Resolved commands are **pasted into your shell's input buffer** for editing before execution — not executed directly. This gives you a chance to review and modify commands before pressing Enter. The shell wrapper function (set up by `ali init`) handles the paste mechanism; without it, the command is simply printed to stdout.

### Parameter escaping

All parameter values and extra arguments are shell-escaped using single-quote wrapping before substitution. This prevents injection through special characters like `$()`, backticks, `;`, and `&` in user-supplied arguments.

### API key storage

- API keys are stored in `ai.yaml` with **0600 permissions** (owner read/write only). The config directory is created with **0700 permissions**.
- When saving, the config file is written atomically (temp file + rename) to prevent corruption from crashes mid-write. Permissions are reset to 0600 on every save, even if they were manually loosened.
- The setup wizard hides API key input on terminals (using `term.ReadPassword`).
- API keys are never included in error messages or logs.
- For additional security, store your key in an environment variable instead of the config file: `export ALI_AI_API_KEY=sk-...`

### AI interaction

- `ali ai` responses are cleaned of markdown code fences but are otherwise unfiltered — the user-review model (pasting, not executing) is the safety mechanism.
- `ali chat` skips files larger than 1MB and binary files (detected by null bytes in the first 512 bytes), with warnings for each skipped file.
- Transient API errors (429 rate limit, 5xx server errors) are retried with exponential backoff (up to 2 retries). Authentication errors fail immediately.

### Shell integration

- The shell wrapper code generated by `ali init` is static — it's not user-supplied and cannot be injected through ali configuration.
- `ali install` creates a backup of your rc file (at `<rcpath>.ali.bak`) before modifying it.
- If the paste mechanism (`print -z` in zsh) fails, the resolved command falls back to being printed to stdout.

### Config file integrity

Config files are written atomically (temp file + rename) to prevent corruption from crashes or concurrent access.

```
ali/
├── cmd/ali/main.go              # Entry point
├── internal/
│   ├── models/
│   │   ├── function.go          # AliFunction, AliConfig structs
│   │   ├── ai.go                # AIConfig struct
│   │   └── function_test.go
│   ├── config/
│   │   ├── config.go            # Load, Save, Path, FindFunction
│   │   ├── aiconfig.go          # LoadAIConfig, SaveAIConfig, AIConfigPath
│   │   └── config_test.go
│   ├── ai/
│   │   ├── client.go            # OpenAI-compatible HTTP client (with retry)
│   │   ├── tokens.go            # Token estimation and context budget
│   │   ├── client_test.go
│   │   └── tokens_test.go
│   ├── execution/
│   │   ├── execute.go           # Parameter substitution + command output
│   │   └── execute_test.go
│   ├── version/
│   │   └── version.go           # Version constant (injected via -ldflags)
│   └── cli/
│       ├── root.go              # Root command
│       ├── add.go               # ali add
│       ├── ai_cmd.go            # ali ai
│       ├── ai_setup.go          # AI setup wizard
│       ├── chat.go              # ali chat
│       ├── completion.go        # Tab completion
│       ├── move.go              # ali move (alias: mv)
│       ├── history.go           # ali history
│       ├── init.go              # ali init
│       ├── install.go           # ali install
│       ├── list.go              # ali list
│       ├── remove.go            # ali remove
│       ├── edit.go              # ali edit
│       └── run.go               # ali run (alias: r)
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

CI runs `go test`, `go vet`, and `golangci-lint` automatically on pull requests.

### Releasing

Releases are built automatically by [GoReleaser](https://goreleaser.com/) when a version tag is pushed to `main`. Tags must be created **after** the PR is merged — otherwise the tag points to the branch commit, not main.

Version is derived from the git tag at build time via `-ldflags`. Local builds show `dev` as the version; tagged releases show the tag (e.g. `v1.2.1`). There's no version number to update in the Makefile — just tag and push.

```bash
# 1. Make sure main is up to date after a PR merge
git checkout main
git pull

# 2. Tag the release (this also sets the version string)
git tag v1.2.1

# 3. Push the tag — this triggers the release pipeline
git push origin v1.2.1
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
| [github.com/spf13/pflag](https://github.com/spf13/pflag) | POSIX/GNU flag parsing |
| [go.yaml.in/yaml/v3](https://go.yaml.in/yaml/v3) | YAML config parsing |
| [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) | Terminal mode detection (hidden API key input) |

## Running tests

```bash
go test ./... -v
```
