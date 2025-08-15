package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gessage/gessage/internal/ai"
)

func init() {
	ai.Register("ollama", ai.Provider{
		Constructor: newOllamaFromConfig,
		Setup:       setupOllama,
		Stop:        stopOllama,
	})
}

type ollamaClient struct {
	host           string
	model          string
	httpClient     *http.Client
	maxPromptBytes int
}

type ollamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResp struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *ollamaClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	// Build final prompt and clamp size to avoid server-side truncation noise
	finalPrompt := "Write a Conventional Commit message ONLY.\n" + prompt
	if c.maxPromptBytes > 0 && len(finalPrompt) > c.maxPromptBytes {
		finalPrompt = truncateUTF8Bytes(finalPrompt, c.maxPromptBytes)
	}

	// Ollama's /api/generate endpoint
	body := ollamaReq{
		Model:  c.model,
		Prompt: finalPrompt,
		Stream: false,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/generate", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("ollama error: status %s", res.Status)
	}

	var resp ollamaResp
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}
	return resp.Response, nil
}

func newOllamaFromConfig(config map[string]string) (ai.Client, error) {
	host := strings.TrimSpace(config["host"])
	if host == "" {
		host = "http://localhost:11434"
	}
	model := strings.TrimSpace(config["model"])
	if model == "" {
		model = "qwen2.5-coder:3b"
	}

	// Allow configurable timeout via config["timeout_seconds"], default 300s
	timeoutSeconds := 300
	if ts := strings.TrimSpace(config["timeout_seconds"]); ts != "" {
		if v, err := strconv.Atoi(ts); err == nil && v > 0 {
			timeoutSeconds = v
		}
	}

	// Max prompt size (bytes). Default slightly under typical server limit to avoid warnings
	maxPromptBytes := 3800
	if mp := strings.TrimSpace(config["max_prompt_bytes"]); mp != "" {
		if v, err := strconv.Atoi(mp); err == nil && v > 0 {
			maxPromptBytes = v
		}
	}

	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	return &ollamaClient{host: host, model: model, httpClient: client, maxPromptBytes: maxPromptBytes}, nil
}

func setupOllama(ctx context.Context) (map[string]string, error) {
	in := bufio.NewReader(os.Stdin)

	// Ensure ollama CLI is installed
	if _, err := exec.LookPath("ollama"); err != nil {
		fmt.Println("Ollama CLI not found on your system.")
		ok, _ := confirm(in, "Install Ollama now? This will run system commands [y/N]: ")
		if !ok {
			return nil, fmt.Errorf("ollama is required; aborting setup")
		}

		installed := false
		// Prefer Homebrew on macOS if available
		if runtime.GOOS == "darwin" {
			if _, berr := exec.LookPath("brew"); berr == nil {
				fmt.Println("Installing via Homebrew: brew install ollama")
				cmd := exec.CommandContext(ctx, "brew", "install", "ollama")
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err == nil {
					installed = true
				} else {
					fmt.Printf("brew install failed: %v\n", err)
				}
			}
		}
		if !installed {
			fmt.Println("Installing via official script: curl -fsSL https://ollama.com/install.sh | sh")
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "curl -fsSL https://ollama.com/install.sh | sh")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("ollama install failed: %w", err)
			}
		}
	}

	// Gather host and model
	fmt.Print("Ollama host [http://localhost:11434]: ")
	host, _ := in.ReadString('\n')
	host = strings.TrimSpace(host)
	if host == "" {
		host = "http://localhost:11434"
	}
	fmt.Print("Model name [qwen2.5-coder:3b]: ")
	model, _ := in.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = "qwen2.5-coder:3b"
	}

	// If talking to local server, ensure it's running
	if isLocalhost(host) {
		if !pingOllama(ctx, host, 2*time.Second) {
			ok, _ := confirm(in, "Ollama server not detected. Start it now? (Download 1.9GB) [y/N]: ")
			if ok {
				started := false
				if runtime.GOOS == "darwin" {
					if _, berr := exec.LookPath("brew"); berr == nil {
						fmt.Println("Starting via Homebrew services: brew services start ollama")
						cmd := exec.CommandContext(ctx, "brew", "services", "start", "ollama")
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						if err := cmd.Run(); err == nil {
							started = true
						}
					}
				}
				if !started {
					fmt.Println("Starting 'ollama serve' in background...")
					cmd := exec.CommandContext(ctx, "ollama", "serve")
					cmd.Stdin = nil
					cmd.Stdout = io.Discard
					cmd.Stderr = io.Discard
					_ = cmd.Start()
					time.Sleep(2 * time.Second)
				}
			}
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if pingOllama(ctx, host, 2*time.Second) {
					break
				}
				time.Sleep(1 * time.Second)
			}
			if !pingOllama(ctx, host, 2*time.Second) {
				return nil, fmt.Errorf("ollama server is not reachable at %s", host)
			}
		}
	}

	// Try to pull model via `ollama pull <model>` if CLI is available and host is local
	if isLocalhost(host) {
		if _, err := exec.LookPath("ollama"); err == nil {
			cmd := exec.CommandContext(ctx, "ollama", "pull", model)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("ollama pull %q failed: %w", model, err)
			}
			// Verify the model is available after pull
			verify := exec.CommandContext(ctx, "ollama", "show", model)
			verify.Stdin = os.Stdin
			verify.Stdout = os.Stdout
			verify.Stderr = os.Stderr
			if err := verify.Run(); err != nil {
				return nil, fmt.Errorf("ollama model %q not available after pull: %w", model, err)
			}
		}
	}

	return map[string]string{
		"host":  host,
		"model": model,
	}, nil
}

func confirm(in *bufio.Reader, prompt string) (bool, error) {
	fmt.Print(prompt)
	s, err := in.ReadString('\n')
	if err != nil {
		return false, err
	}
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes", nil
}

func isLocalhost(host string) bool {
	h := strings.ToLower(host)
	return strings.Contains(h, "localhost") || strings.Contains(h, "127.0.0.1")
}

func pingOllama(ctx context.Context, host string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(host, "/")+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

// stopOllama attempts to stop the local ollama service and unload the selected model.
// Best-effort: it will try brew services stop (macOS), then `ollama stop <model>`.
func stopOllama(ctx context.Context, cfg map[string]string) error {
	host := strings.TrimSpace(cfg["host"])
	model := strings.TrimSpace(cfg["model"])

	// Only attempt local actions when pointing to localhost
	if !isLocalhost(host) {
		return nil
	}

	// Try to stop running model session (does not remove model)
	if _, err := exec.LookPath("ollama"); err == nil && model != "" {
		cmd := exec.CommandContext(ctx, "ollama", "stop", model)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run() // best-effort
	}

	// On macOS, attempt to stop the background service
	if runtime.GOOS == "darwin" {
		if _, berr := exec.LookPath("brew"); berr == nil {
			cmd := exec.CommandContext(ctx, "brew", "services", "stop", "ollama")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run() // best-effort
		}
	}

	return nil
}

// truncateUTF8Bytes trims s to at most max bytes, preserving valid UTF-8.
func truncateUTF8Bytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	b := []byte(s)
	if len(b) <= max {
		return s
	}
	b = b[:max]
	// Ensure we end on a rune boundary
	for !utf8.Valid(b) && len(b) > 0 {
		b = b[:len(b)-1]
	}
	return string(b) + "\n... [TRUNCATED]\n"
}
