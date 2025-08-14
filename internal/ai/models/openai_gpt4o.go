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

// NewOpenAIGPT4o implements the Factory constructor for OpenAI GPT-4o.
func NewOpenAIGPT4o(opts ai.Options) (ai.Client, error) {
	if opts.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for gpt4-o")
	}
	return &openaiClient{
		apiKey:   opts.OpenAIKey,
		endpoint: "https://api.openai.com/v1/chat/completions",
		model:    "gpt-4o",
	}, nil
}

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
