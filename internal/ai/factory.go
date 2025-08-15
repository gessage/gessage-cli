package ai

import (
	"fmt"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Provider{}
)

// Register a model provider under a name (e.g., "gpt4-o", "ollama").
// Call this in main() to wire built-ins, or from plugins.
func Register(name string, c Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = c
}

// Create builds a Client by name using the provided model-specific configuration map.
func Create(name string, config map[string]string) (Client, error) {
	mu.RLock()
	c, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown model %q; known: %v", name, Known())
	}
	return c.Constructor(config)
}

// Known returns the registered model names.
func Known() []string {
	mu.RLock()
	defer mu.RUnlock()
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}

// ProviderFor returns the registered provider by name.
func ProviderFor(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}
