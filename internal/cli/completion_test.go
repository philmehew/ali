package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

func TestFunctionNamesCompletion(t *testing.T) {
	// Create a temp config file so config.Load() finds it.
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/functions.yaml"
	t.Setenv("ALI_CONFIG", cfgPath)

	cfg := &models.AliConfig{
		Functions: []models.AliFunction{
			{Name: "glog", Body: "git log --oneline"},
			{Name: "gstatus", Description: "show git status", Body: "git status"},
			{Name: "deploy", Body: "kubectl apply -f $1", Defaults: []string{"staging"}},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name       string
		toComplete string
		wantNames  []string
		wantDesc   map[string]string
		wantDir    cobra.ShellCompDirective
	}{
		{
			name:       "empty prefix returns all",
			toComplete: "",
			wantNames:  []string{"glog", "gstatus", "deploy"},
			wantDesc: map[string]string{
				"gstatus": "show git status",
			},
			wantDir: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "prefix g returns glog and gstatus",
			toComplete: "g",
			wantNames:  []string{"glog", "gstatus"},
			wantDesc: map[string]string{
				"gstatus": "show git status",
			},
			wantDir: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "prefix dep returns deploy",
			toComplete: "dep",
			wantNames:  []string{"deploy"},
			wantDesc:   map[string]string{},
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "prefix xyz returns nothing",
			toComplete: "xyz",
			wantNames:  nil,
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, dir := functionNamesCompletion(nil, nil, tt.toComplete)
			if dir != tt.wantDir {
				t.Errorf("directive = %v, want %v", dir, tt.wantDir)
			}
			if len(completions) != len(tt.wantNames) {
				t.Errorf("got %d completions, want %d: %v", len(completions), len(tt.wantNames), completions)
			}
			for _, want := range tt.wantNames {
				found := false
				for _, c := range completions {
					name := strings.SplitN(string(c), "\t", 2)[0]
					if name == want {
						found = true
						if desc, ok := tt.wantDesc[want]; ok {
							parts := strings.SplitN(string(c), "\t", 2)
							if len(parts) < 2 || parts[1] != desc {
								t.Errorf("completion %q description = %q, want %q", want, parts, desc)
							}
						}
						break
					}
				}
				if !found {
					t.Errorf("missing completion %q in %v", want, completions)
				}
			}
		})
	}
}

func TestFunctionParamsCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/functions.yaml"
	t.Setenv("ALI_CONFIG", cfgPath)

	cfg := &models.AliConfig{
		Functions: []models.AliFunction{
			{Name: "deploy", Body: "kubectl apply -f $1 --env $2", Defaults: []string{"staging", "prod"}},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("no args delegates to function name completion", func(t *testing.T) {
		completions, dir := functionParamsCompletion(nil, []string{}, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		if len(completions) != 1 {
			t.Errorf("expected 1 completion, got %v", completions)
		}
		name := strings.SplitN(string(completions[0]), "\t", 2)[0]
		if name != "deploy" {
			t.Errorf("expected deploy completion, got %q", name)
		}
	})

	t.Run("first param offers default for $1", func(t *testing.T) {
		completions, dir := functionParamsCompletion(nil, []string{"deploy"}, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		if len(completions) != 1 {
			t.Fatalf("expected 1 completion, got %v", completions)
		}
		name := strings.SplitN(string(completions[0]), "\t", 2)[0]
		if name != "staging" {
			t.Errorf("expected staging default, got %q", name)
		}
	})

	t.Run("second param offers default for $2", func(t *testing.T) {
		completions, dir := functionParamsCompletion(nil, []string{"deploy", "staging"}, "")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		if len(completions) != 1 {
			t.Fatalf("expected 1 completion, got %v", completions)
		}
		name := strings.SplitN(string(completions[0]), "\t", 2)[0]
		if name != "prod" {
			t.Errorf("expected prod default, got %q", name)
		}
	})

	t.Run("beyond defaults falls back to file comp", func(t *testing.T) {
		_, dir := functionParamsCompletion(nil, []string{"deploy", "staging", "prod"}, "")
		if dir != cobra.ShellCompDirectiveDefault {
			t.Errorf("directive = %v, want Default", dir)
		}
	})

	t.Run("unknown function returns error directive", func(t *testing.T) {
		_, dir := functionParamsCompletion(nil, []string{"nonexistent"}, "")
		if dir != cobra.ShellCompDirectiveError {
			t.Errorf("directive = %v, want Error", dir)
		}
	})
}

func TestFunctionNamesCompletion_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := tmpDir + "/functions.yaml"
	t.Setenv("ALI_CONFIG", cfgPath)

	// No config file — Load returns empty config.
	completions, dir := functionNamesCompletion(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", dir)
	}
	if len(completions) != 0 {
		t.Errorf("expected no completions for empty config, got %v", completions)
	}
}

func TestInitCmd_WrapperContainsCompletePassthrough(t *testing.T) {
	tests := []struct {
		shell string
		want  string
	}{
		{shellZsh, "__complete"},
		{shellBash, "__complete"},
		{shellFish, "__complete"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			var wrapper string
			switch tt.shell {
			case shellZsh:
				wrapper = zshWrapper()
			case shellBash:
				wrapper = bashWrapper()
			case shellFish:
				wrapper = fishWrapper()
			}
			if !strings.Contains(wrapper, tt.want) {
				t.Errorf("expected %q wrapper to contain %q, got: %s", tt.shell, tt.want, wrapper)
			}
		})
	}
}

func TestPrintShellConfig_NoCompletionSnippet(t *testing.T) {
	// Verify ali init no longer includes completion setup (it's in ali install now).
	tests := []struct {
		shell string
	}{
		{shellZsh},
		{shellBash},
		{shellFish},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			cmd := newInitCmd()
			var buf strings.Builder
			cmd.SetOut(&buf)

			if err := printShellConfig(cmd, tt.shell, "/test/bin"); err != nil {
				t.Fatalf("printShellConfig: %v", err)
			}

			output := buf.String()
			if strings.Contains(output, "completion zsh") || strings.Contains(output, "completion bash") || strings.Contains(output, "completion fish") {
				t.Errorf("init output should not contain completion setup, got: %q", output)
			}
		})
	}
}
