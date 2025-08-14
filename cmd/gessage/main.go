package main

import (
	"context"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/gessage/gessage/internal/ai"
	"github.com/gessage/gessage/internal/ai/models"
	"github.com/gessage/gessage/internal/cli"
)

func main() {
	// CLI parsing & app wiring (Dependency Injection here)
	app := cli.NewApp()

	// Register built-in models with the Factory (Open/Closed principle).
	ai.Register("gpt4-o", models.NewOpenAIGPT4o) // remote API
	ai.Register("ollama", models.NewOllama)      // local LLM

	// Execute with a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := app.Run(ctx, os.Args[1:]); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
	color.Green("Done.")
}
