package main

import (
	"context"
	"os"

	"github.com/fatih/color"

	_ "github.com/gessage/gessage/internal/ai/models"
	"github.com/gessage/gessage/internal/cli"
)

func main() {
	// CLI parsing & app wiring (Dependency Injection here)
	app := cli.NewApp()

	// Execute with a cancellable context (no hard timeout to allow long setup/pulls)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx, os.Args[1:]); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}
