# gessage CLI

Generate high-quality Conventional Commit messages from staged diffs using GPT‑4o (OpenAI) or local Ollama. Cross-platform, clean architecture, safe by default (secrets redaction), and fully extensible.

## Features

- Linux/macOS (amd64 & arm64)
- Multiple AI models via Strategy + Factory
- Auto model selection by diff size (overridable with `--model`)
- Interactive: approve / edit / regenerate / cancel
- Conventional Commits enforcement (title ≤ 72, body ≤ 100 cols, valid types)
- Sanitizes diffs to remove secrets before sending to AI
- Fallback message when AI fails

## Install

```bash
git clone https://github.com/gessage/gessage
cd gessage
go build ./cmd/gessage
```
## Cross-compile
### macOS arm64
```bash
GOOS=darwin GOARCH=arm64 go build -o bin/gessage-darwin-arm64 ./cmd/gessage
```
### macOS amd64
```bash
GOOS=darwin GOARCH=amd64 go build -o bin/gessage-darwin-amd64 ./cmd/gessage
```
### Linux arm64
```bash
GOOS=linux GOARCH=arm64 go build -o bin/gessage-linux-arm64 ./cmd/gessage
```
### Linux amd64
```bash
GOOS=linux GOARCH=amd64 go build -o bin/gessage-linux-amd64 ./cmd/gessage
```