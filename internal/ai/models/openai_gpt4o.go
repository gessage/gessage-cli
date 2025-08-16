package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ispooya/gessage-cli/internal/ai"
)

func init() {
	ai.Register("gpt4-o", ai.Provider{
		Constructor: newOpenAIFromConfig,
		Setup:       setupOpenAI,
	})
}

// openaiClient implements ai.Client for OpenAI chat completions
// using a minimal subset required by this app.
//
// It does not stream; it requests a single completion.
// Endpoint and model are configurable via per-model config.

type openaiClient struct {
	apiKey   string
	endpoint string
	model    string
}

type openAIReq struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float32         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResp struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (c *openaiClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	body := openAIReq{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: "You are an assistant that writes Conventional Commit messages. Output only the commit message; no code fences."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.2,
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 40 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("openai error: status %s", res.Status)
	}

	var resp openAIResp
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices from openai")
	}
	return resp.Choices[0].Message.Content, nil
}

func newOpenAIFromConfig(config map[string]string) (ai.Client, error) {
	key := strings.TrimSpace(config["api_key"])
	if key == "" {
		return nil, fmt.Errorf("missing api_key for gpt4-o; run 'gessage setup'")
	}
	endpoint := strings.TrimSpace(config["endpoint"])
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	model := strings.TrimSpace(config["model"])
	if model == "" {
		model = "gpt-4o"
	}
	return &openaiClient{apiKey: key, endpoint: endpoint, model: model}, nil
}

func setupOpenAI(ctx context.Context) (map[string]string, error) {
	in := bufio.NewReader(os.Stdin)
	fmt.Print("Enter OpenAI API Key (sk-...): ")
	key, _ := in.ReadString('\n')
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("API key required")
	}
	fmt.Print("Model name [gpt-4o]: ")
	model, _ := in.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = "gpt-4o"
	}
	return map[string]string{
		"api_key": key,
		"model":   model,
	}, nil
}
