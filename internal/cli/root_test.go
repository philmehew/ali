package cli

import "testing"

func TestInterceptArgs_CompletionCommands(t *testing.T) {
	known := map[string]bool{
		"add": true, "run": true, "list": true, "ls": true,
		"edit": true, "remove": true, "rm": true, "move": true, "mv": true,
		"history": true, "init": true, "install": true,
		"ai": true, "chat": true,
		"help": true, "completion": true,
	}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "completion command stays unchanged",
			args: []string{"ali", "completion", "zsh"},
			want: []string{"ali", "completion", "zsh"},
		},
		{
			name: "help command stays unchanged",
			args: []string{"ali", "help"},
			want: []string{"ali", "help"},
		},
		{
			name: "__complete command stays unchanged",
			args: []string{"ali", "__complete", "g"},
			want: []string{"ali", "__complete", "g"},
		},
		{
			name: "__completeNoDesc stays unchanged",
			args: []string{"ali", "__completeNoDesc", "g"},
			want: []string{"ali", "__completeNoDesc", "g"},
		},
		{
			name: "unknown arg gets run inserted",
			args: []string{"ali", "glog", "20"},
			want: []string{"ali", "run", "glog", "20"},
		},
		{
			name: "known subcommand stays unchanged",
			args: []string{"ali", "list"},
			want: []string{"ali", "list"},
		},
		{
			name: "empty string arg is skipped",
			args: []string{"ali", ""},
			want: []string{"ali", ""},
		},
		{
			name: "no args stays unchanged",
			args: []string{"ali"},
			want: []string{"ali"},
		},
		{
			name: "ai command stays unchanged",
			args: []string{"ali", "ai", "find large files"},
			want: []string{"ali", "ai", "find large files"},
		},
		{
			name: "chat command stays unchanged",
			args: []string{"ali", "chat"},
			want: []string{"ali", "chat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InterceptArgs(tt.args, known)
			if len(got) != len(tt.want) {
				t.Errorf("InterceptArgs() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("InterceptArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
