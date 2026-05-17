package ai

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"a", 0},
		{"ab", 0},
		{"abcd", 1},
		{"hello world", 2},
		{"a longer sentence with many words", 8},
	}

	for _, tt := range tests {
		got := EstimateTokens(tt.text)
		if got != tt.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestContextBudget(t *testing.T) {
	tests := []struct {
		window int
		want   int
	}{
		{128000, 123904},
		{4096, 1024},
		{0, 1024},
		{100, 1024},
		{5000, 1024},
	}

	for _, tt := range tests {
		got := ContextBudget(tt.window)
		if got != tt.want {
			t.Errorf("ContextBudget(%d) = %d, want %d", tt.window, got, tt.want)
		}
	}
}
