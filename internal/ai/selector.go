package ai

// AutoSelectModelName chooses model based on diff size.
// You can extend this with smarter heuristics (entropy, file kinds, etc).
func AutoSelectModelName(requested string, diffBytes int) string {
	// If user explicitly set a model, prefer it.
	if requested != "" {
		return requested
	}
	// Heuristic: small diffs -> GPT-4o, large diffs -> ollama (local, no token limits)
	if diffBytes <= 20_000 {
		return "gpt4-o"
	}
	return "ollama"
}
