package cli

import "testing"

func TestValidateAIURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://api.openai.com/v1", false},
		{"valid http", "http://localhost:8080", false},
		{"valid with path", "https://api.example.com/v1/chat", false},
		{"invalid ftp", "ftp://bad.example.com", true},
		{"invalid no scheme", "api.example.com", true},
		{"empty string", "", true},
		{"invalid just scheme", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAIURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAIURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
