//nolint:goconst // test string repetitions are intentional
package cli

import (
	"strings"
	"testing"

	"github.com/philmehew/ali/internal/execution"
	"github.com/philmehew/ali/internal/models"
)

func TestSplitChainArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		dash    int
		want    [][]string
		wantErr string
	}{
		{
			name: "single command",
			args: []string{"dockup"},
			dash: -1,
			want: [][]string{{"dockup"}},
		},
		{
			name: "two commands",
			args: []string{"dockup", "+", "cellog"},
			dash: -1,
			want: [][]string{{"dockup"}, {"cellog"}},
		},
		{
			name: "three commands",
			args: []string{"dockup", "+", "cellog", "+", "other"},
			dash: -1,
			want: [][]string{{"dockup"}, {"cellog"}, {"other"}},
		},
		{
			name: "params stay with their command",
			args: []string{"deploy", "prod", "/path", "+", "cellog"},
			dash: -1,
			want: [][]string{{"deploy", "prod", "/path"}, {"cellog"}},
		},
		{
			name: "numeric indexes",
			args: []string{"3", "+", "5"},
			dash: -1,
			want: [][]string{{"3"}, {"5"}},
		},
		{
			name: "plus suffix is not a separator",
			args: []string{"glog", "+5"},
			dash: -1,
			want: [][]string{{"glog", "+5"}},
		},
		{
			name: "plus prefix is not a separator",
			args: []string{"glog", "5+"},
			dash: -1,
			want: [][]string{{"glog", "5+"}},
		},
		{
			name: "embedded plus is not a separator",
			args: []string{"glog", "a+b"},
			dash: -1,
			want: [][]string{{"glog", "a+b"}},
		},
		{
			name:    "leading separator is an error",
			args:    []string{"+", "cellog"},
			dash:    -1,
			wantErr: "empty command before",
		},
		{
			name:    "trailing separator is an error",
			args:    []string{"dockup", "+"},
			dash:    -1,
			wantErr: "empty command after",
		},
		{
			name:    "double separator is an error",
			args:    []string{"dockup", "+", "+", "cellog"},
			dash:    -1,
			wantErr: "empty command before",
		},
		{
			name:    "empty args is an error",
			args:    []string{},
			dash:    -1,
			wantErr: "no command specified",
		},
		{
			name: "plus after dash is a literal argument",
			args: []string{"myfn", "+"},
			dash: 1,
			want: [][]string{{"myfn", "+"}},
		},
		{
			name: "chaining stops after dash",
			args: []string{"dockup", "+", "cellog", "extra"},
			dash: 3,
			want: [][]string{{"dockup"}, {"cellog", "extra"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitChainArgs(tt.args, tt.dash)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d segments, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if len(got[i]) != len(tt.want[i]) {
					t.Errorf("segment %d = %v, want %v", i, got[i], tt.want[i])
					continue
				}
				for j := range tt.want[i] {
					if got[i][j] != tt.want[i][j] {
						t.Errorf("segment %d token %d = %q, want %q", i, j, got[i][j], tt.want[i][j])
					}
				}
			}
		})
	}
}

func testChainConfig() *models.AliConfig {
	return &models.AliConfig{
		Functions: []models.AliFunction{
			{Name: "dockup", Body: "docker compose up -d"},
			{Name: "cellog", Body: "docker compose logs -f cell"},
			{Name: "glog", Body: "git log --oneline -n $1", Defaults: []string{"10"}},
			{Name: "deploy", Body: "ssh $1 'cd $2 && git pull'"},
			{Name: "find", Body: "find . -name $1", Defaults: []string{"*.go"}},
		},
	}
}

func TestResolveChain(t *testing.T) {
	cfg := testChainConfig()

	tests := []struct {
		name    string
		args    []string
		dash    int
		want    string
		wantErr string
	}{
		{
			name: "single command unchanged",
			args: []string{"dockup"},
			dash: -1,
			want: "docker compose up -d",
		},
		{
			name: "two commands",
			args: []string{"dockup", "+", "cellog"},
			dash: -1,
			want: "docker compose up -d ; docker compose logs -f cell",
		},
		{
			name: "params only apply to their segment",
			args: []string{"glog", "20", "+", "dockup"},
			dash: -1,
			want: "git log --oneline -n '20' ; docker compose up -d",
		},
		{
			name: "defaults still apply per segment",
			args: []string{"glog", "+", "find"},
			dash: -1,
			want: "git log --oneline -n '10' ; find . -name '*.go'",
		},
		{
			name: "numeric indexes resolve",
			args: []string{"2", "+", "1"},
			dash: -1,
			want: "docker compose logs -f cell ; docker compose up -d",
		},
		{
			name: "dash keeps plus as a literal argument",
			args: []string{"dockup", "+"},
			dash: 1,
			want: "docker compose up -d '+'",
		},
		{
			name:    "unknown function",
			args:    []string{"dockup", "+", "nope"},
			dash:    -1,
			wantErr: `function "nope" not found`,
		},
		{
			name:    "missing required parameter names the function",
			args:    []string{"deploy", "+", "dockup"},
			dash:    -1,
			wantErr: "deploy: parameter $1 is required and has no default",
		},
		{
			name:    "empty segment is rejected",
			args:    []string{"dockup", "+"},
			dash:    -1,
			wantErr: "empty command after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveChain(cfg, tt.args, tt.dash)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if joined := execution.JoinCommands(got); joined != tt.want {
				t.Errorf("JoinCommands() = %q, want %q", joined, tt.want)
			}
		})
	}
}
