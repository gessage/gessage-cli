package ai

import (
	"fmt"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Constructor{}
	defaults = []string{"gpt4-o", "ollama"}
)

// Register a model constructor under a name (e.g., "gpt4-o", "ollama").
// Call this in main() to wire built-ins, or from plugins.
func Register(name string, c Constructor) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = c
}

// Create builds a Client by name.
func Create(name string, opts Options) (Client, error) {
	mu.RLock()
	c, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown model %q; known: %v", name, Known())
	}
	return c(opts)
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
