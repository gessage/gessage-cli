## Contributing to gessage

Thanks for your interest in contributing! This document explains how to build the project locally, run it, and submit improvements.

### Prerequisites

- Go 1.21+ (module declares 1.24; any recent Go should work)
- Git
- For Ollama-related features: the `ollama` CLI (optional)

### Project layout

- `cmd/gessage`: CLI entrypoint
- `internal/cli`: CLI surface and help/UX
- `internal/ai`: Provider registry and client interfaces
- `internal/ai/models`: Built-in providers (`gpt4-o`, `openrouter`, `ollama`)
- `internal/format`: Prompt building and Conventional Commit normalization
- `internal/git`: Git helpers (staged diff, commit)
- `internal/ui`: Simple terminal UI (spinner, select, editor)
- `internal/config`: Config load/save

### Build

```bash
go build -o gessage ./cmd/gessage
```

### Run from source

```bash
./gessage help
./gessage setup --model openrouter
./gessage --dry-run
```

### Code style

- Prefer explicit, readable names over abbreviations
- Keep error handling meaningful; avoid swallowing errors
- Follow the existing formatting; do not reformat unrelated code

### Adding a new AI provider

1. Create a file under `internal/ai/models/<provider>.go`
2. Implement `ai.Client` and register with `ai.Register("name", ai.Provider{...})` in `init()`
3. Provide a `Setup` function to capture and persist provider config
4. Optionally implement `Stop` and `Variants`

### Tests

Currently the project focuses on integration flows. If you add helpers that are easy to unit‑test, include tests.

### Opening a PR

- Keep edits focused and small
- Update `README.md` and CLI help if user‑facing behaviour changes
- Ensure builds succeed on macOS and Linux where possible

### License

By contributing, you agree that your contributions will be licensed under the MIT License.


