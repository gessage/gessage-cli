package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/gessage/gessage/internal/ai"
	"github.com/gessage/gessage/internal/format"
	"github.com/gessage/gessage/internal/git"
	"github.com/gessage/gessage/internal/sanitize"
	"github.com/gessage/gessage/internal/ui"
	"github.com/gessage/gessage/internal/util"
)

// App encapsulates the CLI surface; keeps logic thin and delegates via DI.
type App struct{}

func NewApp() *App { return &App{} }

// Run parses flags, wires dependencies, and executes the main flow.
// Extend CLI here safely: add subcommands or extra flags without touching deeper layers.
func (a *App) Run(ctx context.Context, argv []string) error {
	// Flags for the root command `gessage`
	fs := flag.NewFlagSet("gessage", flag.ContinueOnError)

	var (
		flagModel      = fs.String("model", "", "AI model to use (gpt4-o, ollama, or custom)")
		flagAuto       = fs.Bool("auto", true, "Auto-select model based on diff size (overrides --model if needed)")
		flagType       = fs.String("type", "", "Conventional commit type override (feat, fix, refactor, docs, chore, style, test, perf)")
		flagNoCommit   = fs.Bool("no-commit", false, "Do not run `git commit`; just print the message")
		flagMaxTokens  = fs.Int("max-tokens", 512, "Max tokens for AI generation")
		flagOllamaName = fs.String("ollama-model", "qwen2.5-coder:3b", "Ollama model name (if --model=ollama)")
		flagDryRun     = fs.Bool("dry-run", false, "Print sanitized diff and prompt; do not call AI")
		flagMaxBytes   = fs.Int("max-bytes", 100_000, "Max diff bytes to send to AI (after sanitization)")
	)
	if err := fs.Parse(argv); err != nil {
		return err
	}

	// Step 1: Get staged diff
	diff, err := git.GetStagedDiff(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return errors.New("no staged changes. Use `git add` first")
	}

	// Step 2: Sanitize secrets before we ever hand this to an AI provider
	safe, _ := sanitize.Redact(diff)
	if len(safe) > *flagMaxBytes {
		safe = safe[:*flagMaxBytes] + "\n... [TRUNCATED]\n"
	}

	// Step 3: Choose model strategy (user choice or auto)
	modelName := *flagModel
	if *flagAuto {
		modelName = ai.AutoSelectModelName(modelName, len(safe)) // selector may override based on diff size
	}
	color.Cyan("Using model: %s", modelName)

	// Step 4: Build AI client using the Factory
	client, err := ai.Create(modelName, ai.Options{
		OpenAIKey:   util.Getenv("OPENAI_API_KEY", ""),
		OllamaHost:  util.Getenv("OLLAMA_HOST", "http://localhost:11434"),
		OllamaModel: *flagOllamaName,
	})
	if err != nil {
		return fmt.Errorf("create model: %w", err)
	}

	// Step 5: Build a Conventional Commit prompt
	prompt := format.BuildPrompt(format.PromptInput{
		Diff:         safe,
		Types:        format.AllowedTypes,
		MaxTitle:     72,
		MaxBody:      100,
		UserTypeHint: *flagType,
	})
	if *flagDryRun {
		fmt.Println("=== [SANITIZED DIFF] ===")
		fmt.Println(safe)
		fmt.Println("\n=== [PROMPT] ===")
		fmt.Println(prompt)
		return nil
	}

	// Step 6: Generate message via Strategy client
	msg, genErr := client.Generate(ctx, prompt, *flagMaxTokens)
	if genErr != nil || strings.TrimSpace(msg) == "" {
		color.Yellow("AI failed or returned empty message. Falling back. err=%v", genErr)
		msg = format.FallbackFromDiff(diff)
	}

	// Step 7: Normalize/validate to Conventional Commits constraints
	msg = format.NormalizeMessage(msg, format.NormalizeOptions{
		MaxTitle: 72,
		MaxBody:  100,
		Types:    format.AllowedTypes,
		DefaultType: func() string {
			if *flagType != "" {
				return *flagType
			}
			return "chore"
		}(),
	})

	// Step 8: Interactive approval loop
	for {
		color.White("\n--- Proposed commit message ---\n")
		fmt.Println(msg)
		fmt.Print("\n[a]pprove  [e]dit  [r]egenerate  [c]ancel > ")

		choice, err := ui.ReadChoice()
		if err != nil {
			return err
		}

		switch choice {
		case "a", "approve":
			if *flagNoCommit {
				fmt.Println("\n[NO-COMMIT] Final message:\n" + msg)
				return nil
			}
			return git.CommitWithMessage(ctx, msg)
		case "e", "edit":
			edited, err := ui.EditInEditor(msg) // opens $EDITOR or inline edit fallback
			if err != nil {
				return err
			}
			msg = format.NormalizeMessage(edited, format.NormalizeOptions{
				MaxTitle: 72, MaxBody: 100, Types: format.AllowedTypes, DefaultType: "chore",
			})
		case "r", "regenerate":
			newMsg, err := client.Generate(ctx, prompt, *flagMaxTokens)
			if err != nil || strings.TrimSpace(newMsg) == "" {
				color.Yellow("Regenerate failed; keeping existing proposal.")
				continue
			}
			msg = format.NormalizeMessage(newMsg, format.NormalizeOptions{
				MaxTitle: 72, MaxBody: 100, Types: format.AllowedTypes, DefaultType: "chore",
			})
		case "c", "cancel":
			return errors.New("cancelled by user")
		default:
			color.Yellow("Unknown option: %s", choice)
		}
	}
}
