package ai

import "context"

// Client is the Strategy interface for any AI backend.
// Each model translates (prompt, maxTokens) into a message string.
type Client interface {
	Generate(ctx context.Context, prompt string, maxTokens int) (string, error)
}

// Provider describes a model plugin: how to construct a client from
// a model-specific configuration map, and how to interactively setup
// that configuration for the user (e.g., prompt for API key or download a model).
type Provider struct {
	// Constructor builds a client from a model-specific config map.
	// The map is persisted per model by the setup command.
	Constructor func(config map[string]string) (Client, error)

	// Setup interactively gathers the model-specific configuration and
	// returns it for persistence. Implementations should be idempotent
	// and safe to re-run.
	Setup func(ctx context.Context) (map[string]string, error)

	// Stop attempts to shut down or unload the local resources for this model
	// (e.g., stop background services, unload models, or offer to remove assets).
	// Implementations should be best-effort and safe to call when nothing is running.
	// If nil, the CLI treats it as a no-op.
	Stop func(ctx context.Context, config map[string]string) error
}
