package ai

// EstimateTokens returns a rough token count for text.
// Uses the standard heuristic of ~1 token per 4 characters,
// which works reasonably well for English text and code.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// ContextBudget returns the number of tokens available for the
// conversation, reserving space for the model's response.
func ContextBudget(contextWindow int) int {
	const responseReserve = 4096
	budget := contextWindow - responseReserve
	if budget < 1024 {
		budget = 1024
	}
	return budget
}
