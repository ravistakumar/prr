// Package agent isolates all interaction with external coding-agent CLIs
// (Claude Code, Codex, OpenCode, Aider) behind a single interface. Adding a new
// agent is one new case in New; a CLI flag change touches only that case.
package agent

import (
	"context"
	"fmt"

	"github.com/ravistakumar/prr/internal/config"
)

// Agent is a coding-agent CLI that prr can ask to refine a prompt (headless)
// and then launch with the final prompt (interactive).
type Agent interface {
	Name() string
	Available() bool
	Ask(ctx context.Context, metaPrompt string) (string, error)
	Launch(ctx context.Context, finalPrompt string) error
}

// supported lists the agents prr knows how to drive, in auto-detection order.
var supported = []string{"claude", "codex", "opencode", "aider"}

// New builds a supported agent using cfg for command overrides. The argument
// lists encode each CLI's documented non-interactive (Ask) and handoff
// (Launch) invocations; the prompt is appended after them.
func New(name string, cfg config.Config) (Agent, error) {
	command := name
	if ac, ok := cfg.Agents[name]; ok && ac.Command != "" {
		command = ac.Command
	}
	switch name {
	case "claude":
		// claude -p "<prompt>"  /  claude "<prompt>"
		return newCmdAgent("claude", command, []string{"-p"}, nil), nil
	case "codex":
		// codex exec "<prompt>"  /  codex "<prompt>"
		return newCmdAgent("codex", command, []string{"exec"}, nil), nil
	case "opencode":
		// opencode run "<prompt>"  /  opencode --prompt "<prompt>"
		return newCmdAgent("opencode", command, []string{"run"}, []string{"--prompt"}), nil
	case "aider":
		// aider is edit-oriented: --message runs one-shot and exits, so the
		// same non-interactive invocation is used for both Ask and Launch.
		return newCmdAgent("aider", command,
			[]string{"--yes", "--no-auto-commits", "--message"},
			[]string{"--yes", "--message"}), nil
	default:
		return nil, fmt.Errorf("unknown agent %q", name)
	}
}

// Detect returns the configured agent, or the first available one when the
// config says "auto". It errors if none is available.
func Detect(cfg config.Config) (Agent, error) {
	if cfg.Agent != "auto" {
		a, err := New(cfg.Agent, cfg)
		if err != nil {
			return nil, err
		}
		if !a.Available() {
			return nil, fmt.Errorf("agent %q is not installed or on PATH", cfg.Agent)
		}
		return a, nil
	}
	for _, name := range supported {
		a, err := New(name, cfg)
		if err != nil {
			continue
		}
		if a.Available() {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no supported agent found; install Claude Code, Codex, OpenCode, or Aider, or set --agent")
}
