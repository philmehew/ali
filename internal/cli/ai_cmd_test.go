package cli

import (
	"testing"
)

func TestCleanCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain command",
			input: "git status",
			want:  "git status",
		},
		{
			name:  "command with backtick fences",
			input: "```sh\ngit status\n```",
			want:  "git status",
		},
		{
			name:  "command with plain backtick fences",
			input: "```\ngit status\n```",
			want:  "git status",
		},
		{
			name:  "command with leading/trailing whitespace",
			input: "  git status  \n",
			want:  "git status",
		},
		{
			name:  "backtick fence with language tag",
			input: "```bash\nls -la\n```",
			want:  "ls -la",
		},
		{
			name:  "multiple backtick fences",
			input: "```sh\nls\n```\n```sh\npwd\n```",
			want:  "ls\n\n\npwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanCommand(tt.input)
			if got != tt.want {
				t.Errorf("cleanCommand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
