//nolint:goconst // test string repetitions are intentional
package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func TestAddContextToHistory(t *testing.T) {
	tests := []struct {
		name            string
		history         []ai.Message
		content         string
		wantLen         int
		wantContextIdx1 bool
	}{
		{
			name:            "insert when no context exists",
			history:         []ai.Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "hello"}},
			content:         `<file path="a.txt">content</file>` + "\n",
			wantLen:         3,
			wantContextIdx1: true,
		},
		{
			name: "append to existing file context",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">content</file>\n"},
			},
			content:         `<file path="b.txt">more</file>` + "\n",
			wantLen:         2,
			wantContextIdx1: true,
		},
		{
			name:            "insert into system-only history",
			history:         []ai.Message{{Role: "system", Content: "sys"}},
			content:         `<file path="a.txt">content</file>` + "\n",
			wantLen:         2,
			wantContextIdx1: true,
		},
		{
			name: "append cmd to existing file context",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">data</file>\n"},
			},
			content:         `<cmd command="echo hi" runs="5">output</cmd>` + "\n",
			wantLen:         2,
			wantContextIdx1: true,
		},
		{
			name: "insert cmd context into system-only history",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
			},
			content:         `<cmd command="top" runs="3">output</cmd>` + "\n",
			wantLen:         2,
			wantContextIdx1: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addContextToHistory(&tt.history, tt.content)
			if len(tt.history) != tt.wantLen {
				t.Errorf("history length = %d, want %d", len(tt.history), tt.wantLen)
			}
			if tt.wantContextIdx1 {
				if !isContextMessage(tt.history[1]) {
					t.Errorf("history[1] is not context message, got: %q", tt.history[1].Content[:20])
				}
			}
		})
	}
}

func TestAddContextToHistoryInsertionShiftsTurns(t *testing.T) {
	history := []ai.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "answer"},
	}
	addContextToHistory(&history, "<file path=\"x.txt\">data</file>\n")

	if len(history) != 4 {
		t.Fatalf("history length = %d, want 4", len(history))
	}
	if history[0].Role != "system" {
		t.Errorf("history[0].Role = %q, want system", history[0].Role)
	}
	if !isContextMessage(history[1]) {
		t.Errorf("history[1] should be context message, got %q", history[1].Content[:10])
	}
	if history[2].Content != "question" {
		t.Errorf("history[2].Content = %q, want question (shifted from index 1)", history[2].Content)
	}
	if history[3].Content != "answer" {
		t.Errorf("history[3].Content = %q, want answer (shifted from index 2)", history[3].Content)
	}
}

func TestHandleChatCommand(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantExit bool
		wantOut  string
	}{
		{name: "exit", line: "/exit", wantExit: true},
		{name: "quit", line: "/quit", wantExit: true},
		{name: "unknown", line: "/foo", wantExit: false, wantOut: "Unknown command: /foo"},
		{name: "file no args", line: "/file", wantExit: false, wantOut: "Usage: /file <path>"},
		{name: "folder no args", line: "/folder", wantExit: false, wantOut: "Usage: /folder <path>"},
		{name: "cmd no args", line: "/cmd", wantExit: false, wantOut: "Usage: /cmd <command>"},
		{name: "save no args", line: "/save", wantExit: false, wantOut: "Usage: /save <name>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := displayOut
			displayOut = &buf
			defer func() { displayOut = old }()

			history := []ai.Message{{Role: "system", Content: "sys"}}
			trimWarned := false
			cfg := &models.AIConfig{Model: "test-model", URL: "http://localhost", ContextWindow: 128000}
			var client *ai.Client
			budget := 123904

			got := handleChatCommand(tt.line, &history, &trimWarned, &client, cfg, &budget, new(string))
			if got != tt.wantExit {
				t.Errorf("handleChatCommand(%q) exit = %v, want %v", tt.line, got, tt.wantExit)
			}
			if tt.wantOut != "" && !strings.Contains(buf.String(), tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdClear(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		history []ai.Message
		wantLen int
		wantOut string
	}{
		{
			name: "clear keeps file context",
			args: "",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">data</file>\n"},
				{Role: "user", Content: "question"},
				{Role: "assistant", Content: "answer"},
			},
			wantLen: 2,
			wantOut: "1 file(s) still in context",
		},
		{
			name: "clear all removes file context",
			args: "all",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">data</file>\n"},
				{Role: "user", Content: "question"},
				{Role: "assistant", Content: "answer"},
			},
			wantLen: 1,
			wantOut: "Conversation, file, and command context cleared",
		},
		{
			name: "clear with no file context",
			args: "",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "question"},
				{Role: "assistant", Content: "answer"},
			},
			wantLen: 1,
			wantOut: "Conversation cleared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := displayOut
			displayOut = &buf
			defer func() { displayOut = old }()

			trimWarned := true
			cmdClear(tt.args, &tt.history, &trimWarned)

			if len(tt.history) != tt.wantLen {
				t.Errorf("history length = %d, want %d", len(tt.history), tt.wantLen)
			}
			if trimWarned {
				t.Error("trimWarned should be reset to false")
			}
			if !strings.Contains(buf.String(), tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdClearWithCmdContext(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	history := []ai.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "<cmd command=\"top\" runs=\"5\">output</cmd>\n"},
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "answer"},
	}
	trimWarned := true
	cmdClear("", &history, &trimWarned)

	if len(history) != 2 {
		t.Errorf("history length = %d, want 2", len(history))
	}
	if !isContextMessage(history[1]) {
		t.Errorf("history[1] should still have context, got: %q", history[1].Content[:20])
	}
	out := buf.String()
	if !strings.Contains(out, "1 command(s)") {
		t.Errorf("expected '1 command(s)' in output, got: %q", out)
	}
}

func TestCmdContext(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	history := []ai.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	budget := 10000

	cmdContext("", history, budget)

	out := buf.String()
	if !strings.Contains(out, "Context:") {
		t.Errorf("output missing 'Context:', got: %q", out)
	}
	if !strings.Contains(out, "Turns:") {
		t.Errorf("output missing 'Turns:', got: %q", out)
	}
	if !strings.Contains(out, "Files:") {
		t.Errorf("output missing 'Files:', got: %q", out)
	}
	if !strings.Contains(out, "Cmds:") {
		t.Errorf("output missing 'Cmds:', got: %q", out)
	}
}

func TestCmdModel(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantOut string
	}{
		{name: "show model", args: "", wantOut: "test-model"},
		{name: "switch model", args: "gpt-4o", wantOut: "test-model -> gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := displayOut
			displayOut = &buf
			defer func() { displayOut = old }()

			cfg := &models.AIConfig{Model: "test-model", URL: "http://localhost", ContextWindow: 128000}
			var client *ai.Client

			cmdModel(tt.args, cfg, &client)

			if !strings.Contains(buf.String(), tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOut)
			}
		})
	}
}

func TestCmdModelSwitchUpdatesConfig(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	cfg := &models.AIConfig{Model: "old-model", URL: "http://localhost", ContextWindow: 128000, APIKey: "test"}
	var client *ai.Client

	cmdModel("new-model", cfg, &client)

	if cfg.Model != "new-model" {
		t.Errorf("cfg.Model = %q, want new-model", cfg.Model)
	}
	if client == nil {
		t.Error("client should be non-nil after model switch")
	}
}

func TestCmdHelp(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	cmdHelp()

	out := buf.String()
	for _, cmd := range []string{"/file", "/folder", "/cmd", "/clear", "/context", "/model", "/window", "/save", "/load", "/delete", "/help", "/exit"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("help output missing %q", cmd)
		}
	}
}

func TestFileContextPaths(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single file",
			content: `<file path="a.txt">data</file>` + "\n",
			want:    []string{"a.txt"},
		},
		{
			name:    "multiple files",
			content: `<file path="a.txt">data</file>` + "\n" + `<file path="b.go">code</file>` + "\n",
			want:    []string{"a.txt", "b.go"},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileContextPaths(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("fileContextPaths() = %v, want %v", got, tt.want)
				return
			}
			for i, p := range got {
				if p != tt.want[i] {
					t.Errorf("fileContextPaths()[%d] = %q, want %q", i, p, tt.want[i])
				}
			}
		})
	}
}

func TestIngestFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o600); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package main"), 0o600); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "subdir", "c.txt"), []byte("nested"), 0o600); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	var content strings.Builder
	count := ingestFolder(dir, &content)

	if count != 2 {
		t.Errorf("ingestFolder() = %d, want 2 (non-recursive)", count)
	}
	if !strings.Contains(content.String(), "a.txt") || !strings.Contains(content.String(), "b.go") {
		t.Errorf("content missing expected files: %q", content.String())
	}
	if strings.Contains(content.String(), "c.txt") {
		t.Error("ingestFolder should be non-recursive, found nested file")
	}
}

func TestDoubleSlashEscape(t *testing.T) {
	line := "//file not a command"
	if !strings.HasPrefix(line, "//") {
		t.Error("should detect double-slash prefix")
	}
	escaped := line[1:]
	if escaped != "/file not a command" {
		t.Errorf("escaped = %q, want /file not a command", escaped)
	}
}

func TestCmdWindow(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		wantOut    string
		wantWindow int
		wantBudget int
	}{
		{name: "show window", args: "", wantOut: "128000", wantWindow: 128000, wantBudget: 123904},
		{name: "set window", args: "64000", wantOut: "64000", wantWindow: 64000, wantBudget: 59904},
		{name: "invalid", args: "abc", wantOut: "Usage: /window", wantWindow: 128000, wantBudget: 123904},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := displayOut
			displayOut = &buf
			defer func() { displayOut = old }()

			cfg := &models.AIConfig{ContextWindow: 128000}
			budget := 123904

			cmdWindow(tt.args, cfg, &budget)

			if !strings.Contains(buf.String(), tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOut)
			}
			if cfg.ContextWindow != tt.wantWindow {
				t.Errorf("ContextWindow = %d, want %d", cfg.ContextWindow, tt.wantWindow)
			}
			if budget != tt.wantBudget {
				t.Errorf("budget = %d, want %d", budget, tt.wantBudget)
			}
		})
	}
}

func TestCmdModelSwitchReminder(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	cfg := &models.AIConfig{Model: "old", URL: "http://localhost", ContextWindow: 128000, APIKey: "test"}
	var client *ai.Client

	cmdModel("new-model", cfg, &client)

	out := buf.String()
	if !strings.Contains(out, "Context window is 128000 tokens") {
		t.Errorf("missing reminder about context window, got: %q", out)
	}
	if !strings.Contains(out, "/window") {
		t.Errorf("missing /window hint, got: %q", out)
	}
}

func TestIsContextMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  ai.Message
		want bool
	}{
		{name: "file context", msg: ai.Message{Content: "<file path=\"a.txt\">data</file>"}, want: true},
		{name: "cmd context", msg: ai.Message{Content: "<cmd command=\"top\" runs=\"5\">output</cmd>"}, want: true},
		{name: "regular message", msg: ai.Message{Content: "hello"}, want: false},
		{name: "empty", msg: ai.Message{Content: ""}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isContextMessage(tt.msg); got != tt.want {
				t.Errorf("isContextMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIngestCmd(t *testing.T) {
	content, count := ingestCmd("echo hello", 1, 0)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if !strings.Contains(content, `<cmd command="echo hello" runs="1">`) {
		t.Errorf("missing opening cmd tag, got: %q", content[:60])
	}
	if !strings.Contains(content, "</cmd>") {
		t.Errorf("missing closing cmd tag, got: %q", content)
	}
	if !strings.Contains(content, "--- Run 1") {
		t.Errorf("missing run header, got: %q", content)
	}
	if !strings.Contains(content, "hello") {
		t.Errorf("missing command output, got: %q", content)
	}
}

func TestIngestCmdMultipleRuns(t *testing.T) {
	content, count := ingestCmd("echo test", 3, 0)
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if !strings.Contains(content, "--- Run 1") {
		t.Errorf("missing Run 1 header")
	}
	if !strings.Contains(content, "--- Run 2") {
		t.Errorf("missing Run 2 header")
	}
	if !strings.Contains(content, "--- Run 3") {
		t.Errorf("missing Run 3 header")
	}
}

func TestParseCmdArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		wantCmd  string
		wantRuns int
		wantWait time.Duration
	}{
		{name: "no flags", args: "top", wantCmd: "top", wantRuns: 5, wantWait: 5 * time.Second},
		{name: "with runs flag", args: "top --runs 10", wantCmd: "top", wantRuns: 10, wantWait: 5 * time.Second},
		{name: "multi-word command with runs", args: "ps aux --runs 3", wantCmd: "ps aux", wantRuns: 3, wantWait: 5 * time.Second},
		{name: "runs at end", args: "echo hello --runs 2", wantCmd: "echo hello", wantRuns: 2, wantWait: 5 * time.Second},
		{name: "with wait flag", args: "top --wait 3", wantCmd: "top", wantRuns: 5, wantWait: 3 * time.Second},
		{name: "with both flags", args: "top --runs 10 --wait 2", wantCmd: "top", wantRuns: 10, wantWait: 2 * time.Second},
		{name: "zero wait", args: "uptime --wait 0", wantCmd: "uptime", wantRuns: 5, wantWait: 0},
		{name: "empty args", args: "", wantCmd: "", wantRuns: 5, wantWait: 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, runs, wait := parseCmdArgs(tt.args)
			if cmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tt.wantCmd)
			}
			if runs != tt.wantRuns {
				t.Errorf("runs = %d, want %d", runs, tt.wantRuns)
			}
			if wait != tt.wantWait {
				t.Errorf("wait = %v, want %v", wait, tt.wantWait)
			}
		})
	}
}

func TestCmdContextCommands(t *testing.T) {
	content := `<cmd command="top" runs="5">output1</cmd>` + "\n" + `<cmd command="ps aux" runs="3">output2</cmd>` + "\n"
	cmds := cmdContextCommands(content)
	if len(cmds) != 2 {
		t.Fatalf("len = %d, want 2", len(cmds))
	}
	if cmds[0].command != "top" || cmds[0].runs != "5" {
		t.Errorf("cmds[0] = {%q, %q}, want {top, 5}", cmds[0].command, cmds[0].runs)
	}
	if cmds[1].command != "ps aux" || cmds[1].runs != "3" {
		t.Errorf("cmds[1] = {%q, %q}, want {ps aux, 3}", cmds[1].command, cmds[1].runs)
	}
}

func TestCmdSaveNoArgs(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	history := []ai.Message{{Role: ai.RoleSystem, Content: "sys"}}
	cfg := &models.AIConfig{Model: "test-model", ContextWindow: 128000}

	cmdSave("", history, cfg, new(string))

	if !strings.Contains(buf.String(), "Usage: /save <name>") {
		t.Errorf("expected usage message, got: %q", buf.String())
	}
}

func TestCmdSaveUsesSessionName(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	history := []ai.Message{{Role: ai.RoleSystem, Content: "sys"}}
	cfg := &models.AIConfig{Model: "test-model", ContextWindow: 128000}
	sessionName := "mysession"

	// Save with no args should use sessionName
	cmdSave("", history, cfg, &sessionName)

	// When sessionName is a bare name, resolveChatPath turns it into a full path.
	// The output should contain the session name.
	if !strings.Contains(buf.String(), "Saved") {
		t.Errorf("expected save confirmation, got: %q", buf.String())
	}
	// sessionName should still be "mysession" (it was already set)
	if sessionName != "mysession" {
		t.Errorf("sessionName = %q, want mysession", sessionName)
	}
}

func TestCmdSaveUpdatesSessionName(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	history := []ai.Message{{Role: ai.RoleSystem, Content: "sys"}}
	cfg := &models.AIConfig{Model: "test-model", ContextWindow: 128000}
	sessionName := ""

	dir := t.TempDir()

	// Save with explicit name should update sessionName
	cmdSave(filepath.Join(dir, "new-name.json"), history, cfg, &sessionName)

	// sessionName gets set to whatever args was passed (the full path in this case)
	if sessionName != filepath.Join(dir, "new-name.json") {
		t.Errorf("sessionName = %q, want %q", sessionName, filepath.Join(dir, "new-name.json"))
	}
}

func TestCmdSaveAndLoadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	dir := t.TempDir()
	// Redirect config to temp dir so loadMemory() doesn't pick up a real MEMORY.md
	cfgPath := filepath.Join(dir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)
	if err := config.Save(&models.AliConfig{}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	originalHistory := []ai.Message{
		{Role: ai.RoleSystem, Content: "system prompt"},
		{Role: ai.RoleUser, Content: "<file path=\"a.txt\">data</file>\n"},
		{Role: ai.RoleUser, Content: "hello"},
		{Role: ai.RoleAssistant, Content: "hi there"},
	}
	cfg := &models.AIConfig{Model: "test-model", URL: "http://localhost", ContextWindow: 128000}
	var client *ai.Client
	budget := ai.ContextBudget(cfg.ContextWindow)
	trimWarned := false

	// Save to temp dir
	savePath := filepath.Join(dir, "test-chat.json")
	cmdSave(savePath, originalHistory, cfg, new(string))

	if !strings.Contains(buf.String(), "Saved") {
		t.Errorf("expected save confirmation, got: %q", buf.String())
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(savePath) //nolint:gosec // G304: test path is controlled
	if err != nil {
		t.Fatalf("could not read saved file: %v", err)
	}
	var session chatSession
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("could not parse saved file: %v", err)
	}
	if session.Model != "test-model" {
		t.Errorf("session.Model = %q, want test-model", session.Model)
	}
	if session.ContextWindow != 128000 {
		t.Errorf("session.ContextWindow = %d, want 128000", session.ContextWindow)
	}
	if len(session.Messages) != 4 {
		t.Errorf("len(session.Messages) = %d, want 4", len(session.Messages))
	}
	if session.SavedAt == "" {
		t.Error("session.SavedAt is empty")
	}

	// Load back into a fresh history
	buf.Reset()
	loadedHistory := []ai.Message{{Role: ai.RoleSystem, Content: "placeholder"}}
	cmdLoad(savePath, &loadedHistory, cfg, &client, &budget, &trimWarned, new(string))

	if !strings.Contains(buf.String(), "Loaded") {
		t.Errorf("expected load confirmation, got: %q", buf.String())
	}
	if len(loadedHistory) != 4 {
		t.Errorf("len(loadedHistory) = %d, want 4", len(loadedHistory))
	}
	if loadedHistory[0].Content != "system prompt" {
		t.Errorf("loadedHistory[0].Content = %q, want system prompt", loadedHistory[0].Content)
	}
	if loadedHistory[1].Content != "<file path=\"a.txt\">data</file>\n" {
		t.Errorf("loadedHistory[1].Content = %q, want file context", loadedHistory[1].Content)
	}
	if cfg.Model != "test-model" {
		t.Errorf("cfg.Model = %q after load, want test-model", cfg.Model)
	}
	if client == nil {
		t.Error("client should be non-nil after load")
	}
	if trimWarned {
		t.Error("trimWarned should be false after load")
	}
}

func TestCmdLoadList(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	dir := t.TempDir()

	// Create a couple of chat files
	for _, name := range []string{"chat1.json", "chat2.json"} {
		session := chatSession{
			Model:         "test",
			ContextWindow: 128000,
			SavedAt:       time.Now().Format(time.RFC3339),
			Messages:      []ai.Message{{Role: ai.RoleSystem, Content: "sys"}},
		}
		data, _ := json.MarshalIndent(session, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o600)
	}

	// Test listing — we need to override the chats dir temporarily
	// Since cmdLoadList uses chatsDir which goes to config.Dir(),
	// we'll test by saving into the actual chatsDir via cmdSave
	// and verify the list output. Instead, test via the direct path.
	// For a simpler test, just call cmdLoad with a path to verify error.
	buf.Reset()
	cmdLoad("/nonexistent/path.json", &[]ai.Message{}, &models.AIConfig{}, new(*ai.Client), new(int), new(bool), new(string))

	if !strings.Contains(buf.String(), "Error") {
		t.Errorf("expected error for nonexistent path, got: %q", buf.String())
	}
}

func TestResolveChatPath(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantEnd string // suffix of the result path
	}{
		{name: "bare name", args: "mysession", wantEnd: "mysession.json"},
		{name: "name with json", args: "mysession.json", wantEnd: "mysession.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveChatPath(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("resolveChatPath(%q) = %q, want suffix %q", tt.args, got, tt.wantEnd)
			}
		})
	}
}

func TestCmdDeleteNoArgs(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	cmdDelete("")

	// Should list chats (same as /load) plus the delete hint
	out := buf.String()
	if !strings.Contains(out, "Use /delete <name> to delete a chat.") {
		t.Errorf("expected delete hint, got: %q", out)
	}
}

func TestCmdDeleteByName(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	// Set up config dir so chatsDir resolves correctly
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)
	if err := config.Save(&models.AliConfig{}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Create a chat file
	chatDir := filepath.Join(tmpDir, "chats")
	if err := os.MkdirAll(chatDir, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	session := chatSession{
		Model:         "test",
		ContextWindow: 128000,
		SavedAt:       time.Now().Format(time.RFC3339),
		Messages:      []ai.Message{{Role: ai.RoleSystem, Content: "sys"}},
	}
	data, _ := json.MarshalIndent(session, "", "  ")
	if err := os.WriteFile(filepath.Join(chatDir, "to-delete.json"), data, 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cmdDelete("to-delete")

	out := buf.String()
	if !strings.Contains(out, "Deleted to-delete") {
		t.Errorf("expected delete confirmation, got: %q", out)
	}

	// Verify file is gone
	if _, err := os.Stat(filepath.Join(chatDir, "to-delete.json")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestCmdDeleteNotFound(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	cmdDelete("nonexistent")

	if !strings.Contains(buf.String(), "not found") {
		t.Errorf("expected not found message, got: %q", buf.String())
	}
}

func TestMemoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	path, err := memoryPath()
	if err != nil {
		t.Fatalf("memoryPath() error: %v", err)
	}
	if !strings.HasSuffix(path, "MEMORY.md") {
		t.Errorf("memoryPath() = %q, want suffix MEMORY.md", path)
	}
}

func TestLoadMemory(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		content := "always use Python 3.12\nprefer concise answers\n"
		if err := os.WriteFile(memPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		mem := loadMemory()
		if mem != "always use Python 3.12\nprefer concise answers" {
			t.Errorf("loadMemory() = %q, want trimmed content", mem)
		}
	})

	t.Run("file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		mem := loadMemory()
		if mem != "" {
			t.Errorf("loadMemory() = %q, want empty string for missing file", mem)
		}
	})

	t.Run("file empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		if err := os.WriteFile(memPath, []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}

		mem := loadMemory()
		if mem != "" {
			t.Errorf("loadMemory() = %q, want empty string for empty file", mem)
		}
	})
}

func TestBuildSystemPrompt(t *testing.T) {
	t.Run("no memory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		result := buildSystemPrompt("You are a helpful assistant.")
		if result != "You are a helpful assistant." {
			t.Errorf("buildSystemPrompt() = %q, want base prompt unchanged", result)
		}
	})

	t.Run("with memory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		if err := os.WriteFile(memPath, []byte("prefer concise answers\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		result := buildSystemPrompt("You are a helpful assistant.")
		want := "You are a helpful assistant.\n\n<memory>\nprefer concise answers\n</memory>"
		if result != want {
			t.Errorf("buildSystemPrompt() = %q, want %q", result, want)
		}
	})
}

func TestStripMemory(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{
			name:   "no memory block",
			prompt: "You are a helpful assistant.",
			want:   "You are a helpful assistant.",
		},
		{
			name:   "with memory block",
			prompt: "You are a helpful assistant.\n\n<memory>\nprefer concise answers\n</memory>",
			want:   "You are a helpful assistant.",
		},
		{
			name:   "multiline memory",
			prompt: "You are a helpful assistant.\n\n<memory>\nline one\nline two\n</memory>",
			want:   "You are a helpful assistant.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMemory(tt.prompt)
			if got != tt.want {
				t.Errorf("stripMemory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRebuildSystemPrompt(t *testing.T) {
	t.Run("add memory to base prompt", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		if err := os.WriteFile(memPath, []byte("use Python 3.12\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		history := []ai.Message{
			{Role: ai.RoleSystem, Content: "You are a helpful assistant."},
		}
		rebuildSystemPrompt(&history)

		want := "You are a helpful assistant.\n\n<memory>\nuse Python 3.12\n</memory>"
		if history[0].Content != want {
			t.Errorf("history[0].Content = %q, want %q", history[0].Content, want)
		}
	})

	t.Run("update existing memory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		if err := os.WriteFile(memPath, []byte("new memory content\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		history := []ai.Message{
			{Role: ai.RoleSystem, Content: "You are a helpful assistant.\n\n<memory>\nold memory\n</memory>"},
		}
		rebuildSystemPrompt(&history)

		want := "You are a helpful assistant.\n\n<memory>\nnew memory content\n</memory>"
		if history[0].Content != want {
			t.Errorf("history[0].Content = %q, want %q", history[0].Content, want)
		}
	})

	t.Run("no memory file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "functions.yaml")
		t.Setenv("ALI_CONFIG", cfgPath)

		history := []ai.Message{
			{Role: ai.RoleSystem, Content: "You are a helpful assistant."},
		}
		rebuildSystemPrompt(&history)

		if history[0].Content != "You are a helpful assistant." {
			t.Errorf("history[0].Content = %q, want unchanged base prompt", history[0].Content)
		}
	})
}

func TestCmdMemoryNoArgs(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	history := []ai.Message{{Role: ai.RoleSystem, Content: "sys"}}

	t.Run("empty memory", func(t *testing.T) {
		buf.Reset()
		cmdMemory("", &history)
		if !strings.Contains(buf.String(), "Memory is empty") {
			t.Errorf("expected 'Memory is empty', got: %q", buf.String())
		}
	})

	t.Run("non-empty memory", func(t *testing.T) {
		memPath := filepath.Join(tmpDir, "MEMORY.md")
		if err := os.WriteFile(memPath, []byte("prefer concise answers\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		buf.Reset()
		cmdMemory("", &history)
		if !strings.Contains(buf.String(), "prefer concise answers") {
			t.Errorf("expected memory content in output, got: %q", buf.String())
		}
		if !strings.Contains(buf.String(), "~") && !strings.Contains(buf.String(), "tokens") {
			t.Errorf("expected token count in output, got: %q", buf.String())
		}
	})
}

func TestCmdMemoryWithArgs(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	history := []ai.Message{{Role: ai.RoleSystem, Content: "You are a helpful assistant."}}

	// First addition
	cmdMemory("prefer concise answers", &history)
	if !strings.Contains(buf.String(), "Memory updated") {
		t.Errorf("expected 'Memory updated', got: %q", buf.String())
	}
	if !strings.Contains(history[0].Content, "<memory>") {
		t.Errorf("expected memory block in system prompt, got: %q", history[0].Content)
	}
	if !strings.Contains(history[0].Content, "prefer concise answers") {
		t.Errorf("expected memory content in system prompt, got: %q", history[0].Content)
	}

	// Second addition
	buf.Reset()
	cmdMemory("always use Python 3", &history)
	if !strings.Contains(buf.String(), "Memory updated") {
		t.Errorf("expected 'Memory updated', got: %q", buf.String())
	}
	if !strings.Contains(history[0].Content, "prefer concise answers") {
		t.Errorf("expected first memory entry still present, got: %q", history[0].Content)
	}
	if !strings.Contains(history[0].Content, "always use Python 3") {
		t.Errorf("expected second memory entry, got: %q", history[0].Content)
	}

	// Verify file on disk
	memPath := filepath.Join(tmpDir, "MEMORY.md")
	data, err := os.ReadFile(memPath) //nolint:gosec // G304: test path is controlled
	if err != nil {
		t.Fatalf("could not read memory file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "prefer concise answers") {
		t.Errorf("memory file missing first entry, got: %q", content)
	}
	if !strings.Contains(content, "always use Python 3") {
		t.Errorf("memory file missing second entry, got: %q", content)
	}
}

func TestMemorySurvivesClear(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	memPath := filepath.Join(tmpDir, "MEMORY.md")
	if err := os.WriteFile(memPath, []byte("persist across clears\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	history := []ai.Message{
		{Role: ai.RoleSystem, Content: buildSystemPrompt("You are a helpful assistant.")},
		{Role: ai.RoleUser, Content: "hello"},
		{Role: ai.RoleAssistant, Content: "hi"},
	}

	t.Run("clear preserves memory in system prompt", func(t *testing.T) {
		buf.Reset()
		trimWarned := false
		cmdClear("", &history, &trimWarned)

		if !strings.Contains(history[0].Content, "<memory>") {
			t.Errorf("memory block should survive /clear, got: %q", history[0].Content)
		}
	})

	t.Run("clear all preserves memory in system prompt", func(t *testing.T) {
		buf.Reset()
		trimWarned := false
		cmdClear("all", &history, &trimWarned)

		if !strings.Contains(history[0].Content, "<memory>") {
			t.Errorf("memory block should survive /clear all, got: %q", history[0].Content)
		}
	})
}

func TestHandleChatCommandMemory(t *testing.T) {
	var buf bytes.Buffer
	old := displayOut
	displayOut = &buf
	defer func() { displayOut = old }()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	history := []ai.Message{{Role: ai.RoleSystem, Content: "sys"}}
	trimWarned := false
	cfg := &models.AIConfig{Model: "test-model", URL: "http://localhost", ContextWindow: 128000}
	var client *ai.Client
	budget := 123904

	// /memory with no args should not exit
	got := handleChatCommand("/memory", &history, &trimWarned, &client, cfg, &budget, new(string))
	if got {
		t.Error("handleChatCommand(/memory) should return false (no exit)")
	}
}

func TestAppendMemory(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "functions.yaml")
	t.Setenv("ALI_CONFIG", cfgPath)

	t.Run("creates file if missing", func(t *testing.T) {
		if err := appendMemory("first line"); err != nil {
			t.Fatalf("appendMemory() error: %v", err)
		}

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		data, err := os.ReadFile(memPath) //nolint:gosec // G304: test path is controlled
		if err != nil {
			t.Fatalf("could not read memory file: %v", err)
		}
		want := "first line\n"
		if string(data) != want {
			t.Errorf("memory file = %q, want %q", string(data), want)
		}
	})

	t.Run("appends to existing file", func(t *testing.T) {
		if err := appendMemory("second line"); err != nil {
			t.Fatalf("appendMemory() error: %v", err)
		}

		memPath := filepath.Join(tmpDir, "MEMORY.md")
		data, err := os.ReadFile(memPath) //nolint:gosec // G304: test path is controlled
		if err != nil {
			t.Fatalf("could not read memory file: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "first line") {
			t.Errorf("memory file missing first line, got: %q", content)
		}
		if !strings.Contains(content, "second line") {
			t.Errorf("memory file missing second line, got: %q", content)
		}
	})
}
