package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/philmehew/ali/internal/config"
	"github.com/philmehew/ali/internal/models"
)

// setupAIConfig runs an interactive wizard to configure the AI service.
// It is called when ali ai or ali chat is run without an API key configured.
// Returns the config on success, or an error if the user cancels.
func setupAIConfig() (*models.AIConfig, error) {
	outputln("AI is not configured. Let's set it up.")
	outputln()

	reader := bufio.NewReader(os.Stdin)

	output("API key: ")
	apiKey, err := readAPIKey(reader)
	if err != nil {
		// EOF (Ctrl+D) or read error.
		outputln()
		return nil, fmt.Errorf("AI API key not configured. Set ALI_AI_API_KEY or create %s with api_key field", aiConfigPath())
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("AI API key is required. Set ALI_AI_API_KEY or create %s with api_key field", aiConfigPath())
	}

	outputf("Host URL [%s]: ", config.DefaultAIURL())
	url, err := reader.ReadString('\n')
	if err != nil {
		outputln()
		url = ""
	} else {
		url = strings.TrimSpace(url)
	}
	if url == "" {
		url = config.DefaultAIURL()
	}
	if err := validateAIURL(url); err != nil {
		return nil, err
	}

	outputf("Model [%s]: ", config.DefaultAIModel())
	model, err := reader.ReadString('\n')
	if err != nil {
		outputln()
		model = ""
	} else {
		model = strings.TrimSpace(model)
	}
	if model == "" {
		model = config.DefaultAIModel()
	}

	outputf("Context window (tokens) [%d]: ", config.DefaultAIContextWindow())
	contextWindowInput, err := reader.ReadString('\n')
	if err != nil {
		outputln()
		contextWindowInput = ""
	} else {
		contextWindowInput = strings.TrimSpace(contextWindowInput)
	}
	contextWindow := config.DefaultAIContextWindow()
	if contextWindowInput != "" {
		if n, err := strconv.Atoi(contextWindowInput); err == nil && n > 0 {
			contextWindow = n
		}
	}

	outputf("Chat prompt [%s]: ", config.DefaultAIChatPrompt())
	chatPrompt, err := reader.ReadString('\n')
	if err != nil {
		outputln()
		chatPrompt = ""
	} else {
		chatPrompt = strings.TrimSpace(chatPrompt)
	}
	if chatPrompt == "" {
		chatPrompt = config.DefaultAIChatPrompt()
	}

	cfg := &models.AIConfig{
		URL:           url,
		APIKey:        apiKey,
		Model:         model,
		AiPrompt:      config.DefaultAIPrompt(),
		ChatPrompt:    chatPrompt,
		ContextWindow: contextWindow,
	}

	if err := config.SaveAIConfig(cfg); err != nil {
		return nil, fmt.Errorf("could not save AI config: %w", err)
	}

	path, _ := config.AIConfigPath()
	outputf("AI configuration saved to %s\n\n", path)

	// Reload to get defaults applied.
	return config.LoadAIConfig()
}

// readAPIKey reads an API key from stdin. When stdin is a terminal,
// it uses term.ReadPassword to disable echo. Otherwise it falls back
// to the bufio.Reader.
func readAPIKey(reader *bufio.Reader) (string, error) {
	fd := os.Stdin.Fd()
	if term.IsTerminal(int(fd)) { //nolint:gosec // G115: fd is always a small positive integer
		b, err := term.ReadPassword(int(fd)) //nolint:gosec // G115: fd is always a small positive integer
		outputln()                           // ReadPassword does not echo a newline
		return string(b), err
	}
	return reader.ReadString('\n')
}

// validateAIURL checks that the AI host URL is a valid http or https URL.
func validateAIURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid AI host URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid AI host URL %q: must use http or https scheme", rawURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid AI host URL %q: missing host", rawURL)
	}
	return nil
}
