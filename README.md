<div align="center">
  <img height="100" src="https://avatars.githubusercontent.com/u/226575780?s=200&v=4"  />
</div>

## gessage CLI — AI commit messages that follow Conventional Commits

**🔥 Free usage:** Use OpenRouter free models — [Get your free API key](https://openrouter.ai/settings/keys).

### Quick links

- **Free usage (OpenRouter)**: [How to use free models](#openrouter-free-models) · [Get API key](https://openrouter.ai/settings/keys)
- **Quick start**: [Install and configure](#quick-start)
- **Usage**: [Commands and flags](#usage)
- **Build**: [Build from source](#build-from-source)
- **Troubleshooting**: [Common issues](#troubleshooting)

Generate clear, Conventional Commit–compliant messages from your staged git diff using OpenRouter (free models), OpenAI GPT‑4o, or local Ollama. Fast, safe (secret‑redaction), and developer‑friendly.

### Key features

- Multiple AI backends: `openrouter`, `gpt4-o`, `ollama`
- Free option via OpenRouter (pick a `:free` model like `qwen/qwen3-coder:free`)
- Auto model selection by diff size (override with `--model`)
- Interactive approve / edit / regenerate / cancel flow
- Enforces Conventional Commits (title ≤ 72, body ≤ 100 cols, allowed types)
- Secrets are sanitized before being sent to any AI
- Sensible fallback when an AI call fails

### Quick start

1) Install
```bash
git clone https://github.com/gessage/gessage
cd gessage
go build -o gessage ./cmd/gessage
```

2) Configure a model
- OpenRouter (free):
```bash
./gessage setup --model openrouter
# Follow the prompt to paste your OpenRouter API key and select a free model
# Create a free API key here: https://openrouter.ai/settings/keys
```
- OpenAI GPT‑4o:
```bash
./gessage setup --model gpt4-o
```
- Ollama (local):
```bash
./gessage setup --model ollama
```

3) Use it in a repo with staged changes
```bash
git add -A
./gessage
```

### Usage

```bash
gessage [flags]
gessage setup [--model <name>]
gessage default [--model <name>] [--version <id>]
gessage help [setup|default]
```
#### Usage (Only for local providers - ollama)
```bash
gessage down [--model <name>]
gessage help down
```

Flags:
- `--model string`: AI model to use (e.g., `gpt4-o`, `openrouter`, `ollama`)
- `--auto`: Auto‑select model based on diff size (default true)
- `--type string`: Conventional commit type override (`feat`, `fix`, `refactor`, `docs`, `chore`, `style`, `test`, `perf`)
- `--no-commit`: Do not run `git commit`; just print the message
- `--max-tokens int`: Max tokens for AI generation (default 512)
- `--dry-run`: Print sanitized diff and prompt; do not call AI
- `--max-bytes int`: Max diff bytes to send after sanitization (default 100000)

Examples:
```bash
gessage setup --model openrouter
gessage setup --model ollama
gessage down --model ollama
gessage default --model openrouter --version qwen/qwen3-coder:free
gessage default --model ollama --version qwen2.5-coder:3b
gessage --model openrouter
gessage --dry-run
```

### OpenRouter: free models

- Get a free API key: [OpenRouter API keys](https://openrouter.ai/settings/keys)
- During `setup`, choose a `:free` model (cost‑free tier). Recommended options:
  - `qwen/qwen3-coder:free`
  - `qwen/qwen3-235b-a22b:free`
  - `deepseek/deepseek-r1:free`

Quick help (OpenRouter):
```bash
gessage setup --model openrouter
gessage default --model openrouter --version qwen/qwen3-coder:free
gessage --model openrouter
```

### How it works (high level)

- Reads your staged diff only
- Sanitizes secrets
- Builds a strict prompt to request a Conventional Commit message
- Normalizes and validates the AI output to ensure it meets the spec
- Lets you approve, edit, regenerate, or cancel before committing

### Build from source

```bash
go build -o gessage ./cmd/gessage
```

Cross‑compile examples:
```bash
GOOS=darwin GOARCH=arm64 go build -o bin/gessage-darwin-arm64 ./cmd/gessage
GOOS=darwin GOARCH=amd64 go build -o bin/gessage-darwin-amd64 ./cmd/gessage
GOOS=linux  GOARCH=arm64 go build -o bin/gessage-linux-arm64  ./cmd/gessage
GOOS=linux  GOARCH=amd64 go build -o bin/gessage-linux-amd64  ./cmd/gessage
```

### Troubleshooting

- No staged changes: run `git add` first
- Model not configured: run `gessage setup`
- Ollama errors: ensure the daemon is running and the model is pulled
- OpenAI/OpenRouter errors: check your API key and network connectivity

### License

MIT

### 👨‍💻 About the Author

<div align="center">

### Hi, I’m [**Pooya Karimi** 🚀](https://github.com/ispooya)

💡 Software Engineer with a passion for **clean code**, **scalable architecture**, and **developer-friendly tools**.  
🔧 Expert in **PHP/Laravel**, **Go**, and **JavaScript** — with a side interest in **trading bots** & **automation**.  
❤️ I love turning ideas into high-quality products that people *enjoy* using.


[![LinkedIn](https://img.shields.io/badge/LinkedIn-0A66C2?logo=linkedin&logoColor=white&style=for-the-badge)](https://www.linkedin.com/in/ispooya)
</div>