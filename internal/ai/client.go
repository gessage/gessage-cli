package ai

import "context"

// Client is the Strategy interface for any AI backend.
// Each model translates (prompt, maxTokens) into a message string.
type Client interface {
	Generate(ctx context.Context, prompt string, maxTokens int) (string, error)
}

// Options carries DI config for all clients.
// Add new fields comfortably (e.g., base URLs, API versions).
type Options struct {
	OpenAIKey   string
	OllamaHost  string
	OllamaModel string
}

// Constructor signature for Factory registry.
type Constructor func(opts Options) (Client, error)
