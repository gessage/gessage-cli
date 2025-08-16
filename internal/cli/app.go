package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"

	"github.com/ispooya/gessage-cli/internal/ai"
	"github.com/ispooya/gessage-cli/internal/config"
	"github.com/ispooya/gessage-cli/internal/format"
	"github.com/ispooya/gessage-cli/internal/git"
	"github.com/ispooya/gessage-cli/internal/sanitize"
	"github.com/ispooya/gessage-cli/internal/ui"
)

// App encapsulates the CLI surface; keeps logic thin and delegates via DI.
type App struct{}

func NewApp() *App { return &App{} }

// Run parses flags, wires dependencies, and executes the main flow.
// Extend CLI here safely: add subcommands or extra flags without touching deeper layers.
func (a *App) Run(ctx context.Context, argv []string) error {
	// Print version early if requested
	for _, arg := range argv {
		if arg == "--version" || arg == "-v" {
			fmt.Println("gessage CLI", Version)
			return nil
		}
	}
	if len(argv) > 0 && argv[0] == "help" {
		if len(argv) > 1 && argv[1] == "setup" {
			printSetupUsage()
			return nil
		}
		if len(argv) > 1 && argv[1] == "down" {
			printDownUsage()
			return nil
		}
		if len(argv) > 1 && argv[1] == "default" {
			printDefaultUsage()
			return nil
		}
		printRootUsage()
		return nil
	}
	if len(argv) > 0 && argv[0] == "setup" {
		return a.runSetup(ctx, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "down" {
		return a.runDown(ctx, argv[1:])
	}
	if len(argv) > 0 && argv[0] == "default" {
		return a.runDefault(ctx, argv[1:])
	}

	// Flags for the root command `gessage`
	fs := flag.NewFlagSet("gessage", flag.ContinueOnError)
	fs.Usage = printRootUsage

	var (
		flagModel     = fs.String("model", "", "AI model to use (e.g., gpt4-o, openrouter, ollama)")
		flagAuto      = fs.Bool("auto", true, "Auto-select model based on diff size (overrides --model if needed)")
		flagType      = fs.String("type", "", "Conventional commit type override (feat, fix, refactor, docs, chore, style, test, perf)")
		flagNoCommit  = fs.Bool("no-commit", false, "Do not run `git commit`; just print the message")
		flagMaxTokens = fs.Int("max-tokens", 512, "Max tokens for AI generation")
		flagDryRun    = fs.Bool("dry-run", false, "Print sanitized diff and prompt; do not call AI")
		flagMaxBytes  = fs.Int("max-bytes", 100_000, "Max diff bytes to send to AI (after sanitization)")
	)
	if err := fs.Parse(argv); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
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

	// Step 3: Load persisted config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Step 4: Choose model strategy (user choice or auto)
	modelName := *flagModel
	if modelName == "" {
		modelName = cfg.SelectedModel
	}
	if *flagAuto {
		modelName = ai.AutoSelectModelName(modelName, len(safe)) // selector may override based on diff size
	}
	if modelName == "" {
		color.Yellow("No model configured. Run: gessage setup")
		return errors.New("no model selected")
	}
	color.Cyan("Using model: %s", modelName)

	// Step 5: Build AI client using the Factory
	client, err := ai.Create(modelName, cfg.Models[modelName])
	if err != nil {
		return fmt.Errorf("create model: %w", err)
	}

	// Step 6: Build a Conventional Commit prompt
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

	// Step 7: Generate message via Strategy client
	spin := ui.NewSpinner("Generating commit message...")
	spin.Start()
	msg, genErr := client.Generate(ctx, prompt, *flagMaxTokens)
	spin.Stop()
	fmt.Println()
	if genErr != nil || strings.TrimSpace(msg) == "" {
		color.Yellow("AI failed or returned empty message. Falling back. err=%v", genErr)
		msg = format.FallbackFromDiff(diff)
	}

	// Step 8: Normalize/validate to Conventional Commits constraints
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

	// Step 9: Interactive approval loop
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
			spin := ui.NewSpinner("Regenerating commit message...")
			spin.Start()
			newMsg, err := client.Generate(ctx, prompt, *flagMaxTokens)
			spin.Stop()
			fmt.Println()
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

func (a *App) runSetup(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("gessage setup", flag.ContinueOnError)
	fs.Usage = printSetupUsage
	var flagModel = fs.String("model", "", "Model to configure (one of: "+strings.Join(ai.Known(), ", ")+")")
	if err := fs.Parse(argv); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var modelName string
	if *flagModel != "" {
		modelName = *flagModel
	} else {
		// Interactive selector: show available models and which are already configured
		known := ai.Known()
		if len(known) == 0 {
			return errors.New("no models registered")
		}
		var opts []string
		for _, m := range known {
			_, configured := cfg.Models[m]
			mark := ""
			if configured {
				mark = " (configured)"
			}
			opts = append(opts, fmt.Sprintf("%s%s", m, mark))
		}
		idx, selErr := ui.Select("Select a model to setup:", opts, 0)
		if selErr != nil {
			return selErr
		}
		modelName = known[idx]
	}

	prov, ok := ai.ProviderFor(modelName)
	if !ok {
		return fmt.Errorf("unknown model %q; known: %v", modelName, ai.Known())
	}

	color.Cyan("Configuring model: %s", modelName)
	mcfg, err := prov.Setup(ctx)
	if err != nil {
		return err
	}
	if cfg.Models == nil {
		cfg.Models = map[string]map[string]string{}
	}
	cfg.Models[modelName] = mcfg
	cfg.SelectedModel = modelName
	if err := config.Save(cfg); err != nil {
		return err
	}
	color.Green("\nSaved configuration for %s", modelName)
	return nil
}

func (a *App) runDown(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("gessage down", flag.ContinueOnError)
	fs.Usage = printDownUsage
	var flagModel = fs.String("model", "", "Model to stop (one of: "+strings.Join(ai.Known(), ", ")+")")
	if err := fs.Parse(argv); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var modelName string
	if *flagModel != "" {
		modelName = *flagModel
	} else {
		known := ai.Known()
		if len(known) == 0 {
			return errors.New("no models registered")
		}
		idx, selErr := ui.Select("Select a model to stop:", known, 0)
		if selErr != nil {
			return selErr
		}
		modelName = known[idx]
	}

	prov, ok := ai.ProviderFor(modelName)
	if !ok {
		return fmt.Errorf("unknown model %q; known: %v", modelName, ai.Known())
	}
	if prov.Stop == nil {
		color.Yellow("Model %s does not support a stop action. Nothing to do.", modelName)
		return nil
	}

	color.Cyan("Stopping model: %s", modelName)
	mCfg := cfg.Models[modelName]
	if err := prov.Stop(ctx, mCfg); err != nil {
		return fmt.Errorf("stop %s: %w", modelName, err)
	}
	color.Green("Stopped %s", modelName)
	return nil
}

func (a *App) runDefault(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("gessage default", flag.ContinueOnError)
	fs.Usage = printDefaultUsage
	var flagModel = fs.String("model", "", "Default model to use (one of: "+strings.Join(ai.Known(), ", ")+")")
	var flagVersion = fs.String("version", "", "Model identifier/version to set as default for this provider")
	if err := fs.Parse(argv); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Choose provider
	modelName := strings.TrimSpace(*flagModel)
	if modelName == "" {
		known := ai.Known()
		if len(known) == 0 {
			return errors.New("no models registered")
		}
		var opts []string
		for _, m := range known {
			_, configured := cfg.Models[m]
			mark := ""
			if configured {
				mark = " (configured)"
			}
			opts = append(opts, fmt.Sprintf("%s%s", m, mark))
		}
		idx, selErr := ui.Select("Select default provider:", opts, 0)
		if selErr != nil {
			return selErr
		}
		modelName = known[idx]
	}

	// Choose variant if provider exposes variants; else prompt free-form or use existing
	prov, ok := ai.ProviderFor(modelName)
	if !ok {
		return fmt.Errorf("unknown model %q; known: %v", modelName, ai.Known())
	}
	mcfg := cfg.Models[modelName]
	if mcfg == nil {
		mcfg = map[string]string{}
	}
	version := strings.TrimSpace(*flagVersion)
	if version == "" {
		if prov.Variants != nil {
			variants := prov.Variants()
			if len(variants) > 0 {
				idx, selErr := ui.Select("Select default model for "+modelName+":", variants, 0)
				if selErr != nil {
					return selErr
				}
				version = variants[idx]
			}
		}
		if version == "" { // fallback to prompt
			reader := bufio.NewReader(os.Stdin)
			current := strings.TrimSpace(mcfg["model"])
			prompt := "Model identifier"
			if current != "" {
				prompt += " [" + current + "]: "
			} else {
				prompt += ": "
			}
			fmt.Print(prompt)
			line, _ := reader.ReadString('\n')
			version = strings.TrimSpace(line)
			if version == "" {
				version = current
			}
		}
	}

	// Persist selection
	cfg.SelectedModel = modelName
	if version != "" {
		mcfg["model"] = version
	}
	if cfg.Models == nil {
		cfg.Models = map[string]map[string]string{}
	}
	cfg.Models[modelName] = mcfg
	if err := config.Save(cfg); err != nil {
		return err
	}

	color.Green("Default set: %s (%s)", modelName, mcfg["model"])
	return nil
}

func printRootUsage() {
	cfgPath, _ := config.Path()

	title := color.New(color.FgCyan, color.Bold)
	section := color.New(color.FgWhite, color.Bold)
	cmd := color.New(color.FgGreen)
	flagC := color.New(color.FgYellow)
	dim := color.New(color.FgHiBlack)

	title.Println("gessage - generate Conventional Commit messages from your staged git diff with AI")
	fmt.Println()

	section.Println("Usage:")
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" [flags]"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" setup [--model <name>]"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" down [--model <name>]"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" default [--model <name>] [--version <id>]"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" help [setup|down|default]"))
	fmt.Println()

	section.Println("Subcommands:")
	fmt.Println("  ", cmd.Sprint("setup"), dim.Sprint("    Interactive model selection, installation, and configuration"))
	fmt.Println("  ", cmd.Sprint("down"), dim.Sprint("     Stop or unload local model resources (e.g., Ollama service/model)"))
	fmt.Println("  ", cmd.Sprint("default"), dim.Sprint("  Set default model and its version/identifier"))
	fmt.Println("  ", cmd.Sprint("help"), dim.Sprint("     Show this help, or help for a subcommand"))
	fmt.Println()

	section.Println("Flags:")
	fmt.Println("  ", flagC.Sprint("--model string"), dim.Sprint("     AI model to use (e.g., gpt4-o, openrouter, ollama)"))
	fmt.Println("  ", flagC.Sprint("--auto"), dim.Sprint("             Auto-select model based on diff size (default true)"))
	fmt.Println("  ", flagC.Sprint("--type string"), dim.Sprint("      Conventional commit type override (feat, fix, refactor, docs, chore, style, test, perf)"))
	fmt.Println("  ", flagC.Sprint("--no-commit"), dim.Sprint("        Do not run 'git commit'; just print the message"))
	fmt.Println("  ", flagC.Sprint("--max-tokens int"), dim.Sprint("   Max tokens for AI generation (default 512)"))
	fmt.Println("  ", flagC.Sprint("--dry-run"), dim.Sprint("          Print sanitized diff and prompt; do not call AI"))
	fmt.Println("  ", flagC.Sprint("--max-bytes int"), dim.Sprint("    Max diff bytes to send to AI after sanitization (default 100000)"))
	fmt.Println()

	section.Println("Models (installed/available):")
	fmt.Println("  ", strings.Join(ai.Known(), ", "))
	fmt.Println()

	section.Println("Config file:")
	fmt.Println("  ", dim.Sprint(cfgPath))
	fmt.Println()

	section.Println("Examples:")
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" setup --model openrouter"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" setup --model ollama"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" down --model ollama"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" default --model openrouter --version qwen/qwen3-coder:free"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" default --model ollama --version qwen2.5-coder:3b"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" --model openrouter"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" --model ollama"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" --dry-run"))
	fmt.Println("  ", cmd.Sprint("gessage"), dim.Sprint(" --version | -v"))
}

func printSetupUsage() {
	fmt.Println("gessage setup - configure your preferred AI model (and install local dependencies if needed)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gessage setup [--model <name>]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --model string     Model to configure (one of:", strings.Join(ai.Known(), ", "), ")")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - 'ollama' setup can install the Ollama CLI (with confirmation), start the local service, and pull the selected model.")
	fmt.Println("  - 'gpt4-o' setup asks for your OpenAI API key and preferred model name.")
	fmt.Println("  - 'openrouter' setup asks for your OpenRouter API key and lets you pick a free model (e.g., qwen/qwen3-coder:free).")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gessage setup --model ollama")
	fmt.Println("  gessage setup --model gpt4-o")
	fmt.Println("  gessage setup --model openrouter")
}

func printDownUsage() {
	fmt.Println("gessage down - stop or unload local model resources")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gessage down [--model <name>]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --model string     Model to stop (one of:", strings.Join(ai.Known(), ", "), ")")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - For 'ollama', this attempts to stop any running model session and stop the background service on macOS.")
	fmt.Println("  - Remote hosts are not affected.")
}

func printDefaultUsage() {
	fmt.Println("gessage default - set default model and version")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gessage default [--model <name>] [--version <id>]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --model string     Default model name (one of:", strings.Join(ai.Known(), ", "), ")")
	fmt.Println("  --version string   Model identifier/version to set for the chosen provider (e.g., qwen2.5-coder:3b, gpt-4o)")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Updates the config's selected model and the provider's model identifier.")
	fmt.Println("  - If the provider isn't configured yet, only the selection is saved; run 'gessage setup' to configure it.")
}
