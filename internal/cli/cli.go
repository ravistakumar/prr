// Package cli builds the prr command: it resolves config (defaults → file →
// env → flags) and drives the run.Runner.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/ravistakumar/prr/internal/agent"
	"github.com/ravistakumar/prr/internal/config"
	"github.com/ravistakumar/prr/internal/interview"
	"github.com/ravistakumar/prr/internal/run"
	"github.com/ravistakumar/prr/internal/signals"
)

var version = "dev" // overridden at build time via -ldflags

// flagSet holds the raw command-line flags before resolution.
type flagSet struct {
	auto         bool
	print        bool
	confirm      bool
	agent        string
	threshold    float64
	maxRounds    int
	thresholdSet bool
	maxRoundsSet bool
}

// resolveMode applies the mutually-exclusive handoff flags over the config.
func resolveMode(cfg config.Config, f flagSet) config.Mode {
	switch {
	case f.auto:
		return config.ModeAuto
	case f.print:
		return config.ModePrint
	case f.confirm:
		return config.ModeConfirm
	default:
		return cfg.Mode
	}
}

// Execute is the entrypoint called from main.
func Execute() error {
	var f flagSet

	root := &cobra.Command{
		Use:           "prr [flags] \"your prompt\"",
		Short:         "Refine your prompt, then run your AI coding agent",
		Version:       version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			f.thresholdSet = cmd.Flags().Changed("threshold")
			f.maxRoundsSet = cmd.Flags().Changed("max-rounds")
			return runRoot(f, args)
		},
	}

	root.Flags().BoolVar(&f.auto, "auto", false, "refine and launch immediately")
	root.Flags().BoolVar(&f.print, "print", false, "print refined prompt to stdout, do not launch")
	root.Flags().BoolVar(&f.confirm, "confirm", false, "show refined prompt and confirm before launching (default)")
	root.Flags().StringVar(&f.agent, "agent", "", "force agent: claude | codex (default: auto-detect)")
	root.Flags().Float64Var(&f.threshold, "threshold", 0.7, "confidence gate 0.0-1.0")
	root.Flags().IntVar(&f.maxRounds, "max-rounds", 2, "max interview rounds")

	root.MarkFlagsMutuallyExclusive("auto", "print", "confirm")

	root.AddCommand(configCmd())
	return root.Execute()
}

func runRoot(f flagSet, args []string) error {
	cfg := load()
	cfg.Mode = resolveMode(cfg, f)
	if f.agent != "" {
		cfg.Agent = f.agent
	}
	if f.thresholdSet {
		cfg.Threshold = f.threshold
	}
	if f.maxRoundsSet {
		cfg.MaxRounds = f.maxRounds
	}

	prompt, err := readPrompt(args)
	if err != nil {
		return err
	}

	ag, err := agent.Detect(cfg)
	if err != nil {
		return err
	}

	r := run.Runner{
		Agent:       ag,
		Asker:       interview.HuhAsker{},
		Cfg:         cfg,
		Interactive: term.IsTerminal(int(os.Stdin.Fd())),
		In:          os.Stdin,
		Out:         os.Stdout,
		Err:         os.Stderr,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return r.Run(ctx, prompt)
}

// readPrompt joins args, or reads from stdin when no args are given.
func readPrompt(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("no prompt given; pass a quoted prompt or pipe one via stdin")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("no prompt given and could not read stdin: %w", err)
	}
	return string(data), nil
}

func load() config.Config {
	cfg, err := config.Load(configPath())
	if err != nil {
		cfg = config.Defaults()
	}
	return cfg.ApplyEnv(os.Getenv)
}

func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "prr", "config.toml")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "prr", "config.toml")
	}
	return ""
}

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print resolved configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := load()
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "agent:      %s\n", cfg.Agent)
			fmt.Fprintf(out, "mode:       %s\n", cfg.Mode)
			fmt.Fprintf(out, "threshold:  %.2f\n", cfg.Threshold)
			fmt.Fprintf(out, "max_rounds: %d\n", cfg.MaxRounds)
			if ag, err := agent.Detect(cfg); err == nil {
				fmt.Fprintf(out, "detected:   %s\n", ag.Name())
			} else {
				fmt.Fprintf(out, "detected:   none\n")
			}
			sig := signals.Detect(".")
			fmt.Fprintf(out, "language:   %s\n", sig.Language)
			if sig.IsGitRepo {
				fmt.Fprintf(out, "git branch: %s\n", sig.GitBranch)
			}
			return nil
		},
	}
}
