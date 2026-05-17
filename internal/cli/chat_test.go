package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/philmehew/ali/internal/ai"
	"github.com/philmehew/ali/internal/models"
)

func TestAddFileContentToHistory(t *testing.T) {
	tests := []struct {
		name           string
		history        []ai.Message
		content        string
		wantLen        int
		wantFileAtIdx1 bool
	}{
		{
			name:           "insert when no file context exists",
			history:        []ai.Message{{Role: "system", Content: "sys"}, {Role: "user", Content: "hello"}},
			content:        `<file path="a.txt">content</file>` + "\n",
			wantLen:        3,
			wantFileAtIdx1: true,
		},
		{
			name: "append to existing file context",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">content</file>\n"},
			},
			content:        `<file path="b.txt">more</file>` + "\n",
			wantLen:        2,
			wantFileAtIdx1: true,
		},
		{
			name:           "insert into system-only history",
			history:        []ai.Message{{Role: "system", Content: "sys"}},
			content:        `<file path="a.txt">content</file>` + "\n",
			wantLen:        2,
			wantFileAtIdx1: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addFileContentToHistory(&tt.history, tt.content)
			if len(tt.history) != tt.wantLen {
				t.Errorf("history length = %d, want %d", len(tt.history), tt.wantLen)
			}
			if tt.wantFileAtIdx1 {
				if !strings.HasPrefix(tt.history[1].Content, "<file") {
					t.Errorf("history[1] does not start with <file, got: %q", tt.history[1].Content[:20])
				}
			}
		})
	}
}

func TestAddFileContentToHistoryInsertionShiftsTurns(t *testing.T) {
	history := []ai.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "question"},
		{Role: "assistant", Content: "answer"},
	}
	addFileContentToHistory(&history, "<file path=\"x.txt\">data</file>\n")

	if len(history) != 4 {
		t.Fatalf("history length = %d, want 4", len(history))
	}
	if history[0].Role != "system" {
		t.Errorf("history[0].Role = %q, want system", history[0].Role)
	}
	if !strings.HasPrefix(history[1].Content, "<file") {
		t.Errorf("history[1] should be file context, got %q", history[1].Content[:10])
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

			got := handleChatCommand(tt.line, &history, &trimWarned, &client, cfg, &budget)
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
			wantOut: "Conversation and file context cleared",
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

func TestCmdFiles(t *testing.T) {
	tests := []struct {
		name    string
		history []ai.Message
		wantOut string
	}{
		{
			name: "lists file paths",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "<file path=\"a.txt\">data</file>\n<file path=\"b.go\">code</file>\n"},
			},
			wantOut: "a.txt",
		},
		{
			name: "no files in context",
			history: []ai.Message{
				{Role: "system", Content: "sys"},
			},
			wantOut: "No files in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			old := displayOut
			displayOut = &buf
			defer func() { displayOut = old }()

			cmdFiles(tt.history)

			if !strings.Contains(buf.String(), tt.wantOut) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOut)
			}
		})
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

	cmdContext(history, budget)

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
	for _, cmd := range []string{"/file", "/folder", "/files", "/clear", "/context", "/model", "/window", "/help", "/exit"} {
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
