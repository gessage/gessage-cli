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

	"github.com/fatih/color"
	"github.com/ispooya/gessage-cli/internal/ai"
	"github.com/ispooya/gessage-cli/internal/ui"
)

func init() {
	ai.Register("openrouter", ai.Provider{
		Constructor: newOpenRouterFromConfig,
		Setup:       setupOpenRouter,
		Variants:    openRouterVariants,
	})
}

func openRouterVariants() []string {
	return []string{
		"qwen/qwen3-coder:free",
		"qwen/qwen3-235b-a22b:free",
		"deepseek/deepseek-r1:free",
	}
}

type openRouterClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type orMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type orReq struct {
	Model       string      `json:"model"`
	Messages    []orMessage `json:"messages"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Temperature float32     `json:"temperature,omitempty"`
}

type orResp struct {
	Choices []struct {
		Message orMessage `json:"message"`
	} `json:"choices"`
}

func (c *openRouterClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	body := orReq{
		Model: c.model,
		Messages: []orMessage{
			{Role: "system", Content: "You are an assistant that writes Conventional Commit messages. Output only the commit message; no code fences."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.2,
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("openrouter error: status %s", res.Status)
	}

	var resp orResp
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices from openrouter")
	}
	return resp.Choices[0].Message.Content, nil
}

func newOpenRouterFromConfig(config map[string]string) (ai.Client, error) {
	// API key is provided by user during setup and stored in config
	key := strings.TrimSpace(config["api_key"])
	if key == "" {
		return nil, fmt.Errorf("missing OpenRouter API key; run 'gessage setup --model openrouter' and paste your key from https://openrouter.ai/settings/keys")
	}

	model := strings.TrimSpace(config["model"])
	if model == "" {
		model = "qwen/qwen3-coder:free"
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	return &openRouterClient{apiKey: key, model: model, httpClient: httpClient}, nil
}

// setupOpenRouter prompts for API key and preferred model (from variants)
func setupOpenRouter(ctx context.Context) (map[string]string, error) {
	in := bufio.NewReader(os.Stdin)
	color.Cyan("OpenRouter setup")
	color.Yellow("1) Visit %s and create a free API key.", "https://openrouter.ai/settings/keys")
	color.Yellow("2) Paste your key below. Your key will be stored locally in gessage's config file.")
	fmt.Print(color.HiWhiteString("OpenRouter API key: "))
	key, _ := in.ReadString('\n')
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Pick a default model from variants using arrow-key selector (fallbacks inside ui.Select)
	variants := openRouterVariants()
	model := "qwen/qwen3-coder:free"
	if len(variants) > 0 {
		idx, err := ui.Select("Select a default OpenRouter model:", variants, 0)
		if err == nil && idx >= 0 && idx < len(variants) {
			model = variants[idx]
		}
	}

	return map[string]string{
		"api_key": key,
		"model":   model,
	}, nil
}
