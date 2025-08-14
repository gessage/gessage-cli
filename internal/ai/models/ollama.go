package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gessage/gessage/internal/ai"
)

// NewOllama creates a local LLM client via Ollama's REST API.
func NewOllama(opts ai.Options) (ai.Client, error) {
	host := opts.OllamaHost
	if host == "" {
		host = "http://localhost:11434"
	}
	model := opts.OllamaModel
	if model == "" {
		model = "qwen2.5-coder:3b"
	}
	return &ollamaClient{host: host, model: model}, nil
}

type ollamaClient struct {
	host  string
	model string
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
	// Ollama's /api/generate endpoint
	body := ollamaReq{
		Model:  c.model,
		Prompt: "Write a Conventional Commit message ONLY.\n" + prompt,
		Stream: false,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/generate", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 40 * time.Second}
	res, err := httpClient.Do(req)
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
