// Package agent isolates all interaction with external coding-agent CLIs
// (Claude Code, Codex) behind a single interface. Adding a new agent is one
// new constructor; a CLI flag change touches only exec.go.
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

// New builds the named agent ("claude" or "codex") using cfg for command
// overrides.
func New(name string, cfg config.Config) (Agent, error) {
	command := name
	if ac, ok := cfg.Agents[name]; ok && ac.Command != "" {
		command = ac.Command
	}
	switch name {
	case "claude":
		return newCmdAgent("claude", command, "-p"), nil
	case "codex":
		return newCmdAgent("codex", command, "exec"), nil
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
	for _, name := range []string{"claude", "codex"} {
		a, err := New(name, cfg)
		if err != nil {
			continue
		}
		if a.Available() {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no supported agent found; install Claude Code or Codex, or set --agent")
}
