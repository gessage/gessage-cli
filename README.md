<div align="center">
  <img height="100" src="https://avatars.githubusercontent.com/u/226575780?s=200&v=4" alt="gessage logo" />
</div>

# gessage CLI — AI Commit Messages that Follow Conventional Commits

**🔥 Free usage:** Generate commit messages using OpenRouter free models — [Get your free API key](https://openrouter.ai/settings/keys).

---

## 🚀 Quick Links

- [Quick Start](#-quick-start)
- [Usage](#-usage)
- [🔥 OpenRouter Free Models](#-openrouter-free-models)
- [About the Author](#-about-the-author)
- [License](#-license)

---

## ✨ Key Features

- Multiple AI backends: `openrouter`, `gpt4-o`, `ollama`
- Free option via OpenRouter (`:free` models like `qwen/qwen3-coder:free`)
- Automatic model selection based on diff size (override with `--model`)
- Interactive flow: approve, edit, regenerate, or cancel
- Enforces Conventional Commits:
  - Title ≤ 72 characters
  - Body ≤ 100 columns
  - Allowed commit types only
- Secret redaction for privacy and security
- Sensible fallback when AI fails

---

## ⚡ Quick Start

### 1. Install

#### Linux & macOS

```bash
curl -fsSL https://raw.githubusercontent.com/gessage/gessage-cli/refs/heads/main/install.sh | bash
```

#### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/gessage/gessage-cli/refs/heads/main/install.ps1 | iex
```

#### Verify Installation

```bash
gessage --help
gessage --version
```

### 2. Configure a Model

- **OpenRouter (Free)**

```bash
gessage setup --model openrouter
# Paste your OpenRouter API key when prompted
# Free API key: https://openrouter.ai/settings/keys
```

- **OpenAI GPT‑4o**

```bash
gessage setup --model gpt4-o
```

- **Ollama (Local)**

```bash
gessage setup --model ollama
```

### 3. Use in a Repo

```bash
git add .
gessage
```

---

## 📖 Usage

```bash
gessage [flags]
gessage setup [--model <name>]
gessage default [--model <name>] [--version <id>]
gessage help [setup|default]
```

### Local Providers (Ollama only)

```bash
gessage down [--model <name>]
gessage help down
```

#### Common Flags

- `--model string` — AI model to use (`gpt4-o`, `openrouter`, `ollama`)
- `--auto` — Auto-select model based on diff size (default: `true`)
- `--type string` — Commit type override (`feat`, `fix`, `refactor`, `docs`, `chore`, `style`, `test`, `perf`)
- `--no-commit` — Print message without committing
- `--max-tokens int` — Max tokens for AI generation (default: 512)
- `--dry-run` — Print sanitized diff & prompt; skip AI call
- `--max-bytes int` — Max diff bytes to send (default: 100000)

#### Examples

```bash
gessage setup --model openrouter
gessage setup --model ollama
gessage down --model ollama
gessage default --model openrouter --version qwen/qwen3-coder:free
gessage default --model ollama --version qwen2.5-coder:3b
gessage --model openrouter
gessage --dry-run
```

---

## 🆓 OpenRouter: Free Models

- Get a free API key: [OpenRouter API keys](https://openrouter.ai/settings/keys)
- During `setup`, choose a `:free` model (cost-free tier). Recommended:
  - `qwen/qwen3-coder:free`
  - `qwen/qwen3-235b-a22b:free`
  - `deepseek/deepseek-r1:free`

```bash
gessage setup --model openrouter
gessage default --model openrouter --version qwen/qwen3-coder:free
gessage --model openrouter
```

---

## ⚙️ How It Works

- Reads staged diff only
- Sanitizes secrets
- Builds a strict prompt for Conventional Commit messages
- Normalizes and validates AI output
- Interactive approval, edit, regenerate, or cancel before committing

---

## 👨‍💻 About the Author

<div align="center">

##### Hi, I’m [**Pooya Karimi** 🚀](https://github.com/ispooya)

💡 Software Engineer passionate about **clean code**, **scalable architecture**, and **developer-friendly tools**.  
🔧 Expert in **PHP/Laravel**, **Go**, and **JavaScript** — with side interests in **trading bots** & **automation**.  
❤️ I enjoy turning ideas into high-quality products that people *love* using.

[![LinkedIn](https://img.shields.io/badge/LinkedIn-0A66C2?logo=linkedin&logoColor=white&style=for-the-badge)](https://www.linkedin.com/in/ispooya)

</div>

---

## 📄 License

MIT

