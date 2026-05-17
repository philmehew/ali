package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	apiKey, err := reader.ReadString('\n')
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
