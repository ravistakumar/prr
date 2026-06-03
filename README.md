<div align="center">

# prr — Prompt Refine & Run

**Refine your prompt _before_ your AI coding agent runs it.**
A terminal prompt firewall for Claude Code & Codex that sharpens vague prompts,
asks the right clarifying questions, and hands your agent a prompt it can actually
build — the first time.

[![CI](https://github.com/ravistakumar/prr/actions/workflows/ci.yml/badge.svg)](https://github.com/ravistakumar/prr/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ravistakumar/prr)](https://github.com/ravistakumar/prr/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravistakumar/prr)](https://goreportcard.com/report/github.com/ravistakumar/prr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

![prr demo](docs/demo.gif)

</div>

## Why prr?

You fire a vague prompt at Claude Code or Codex. It guesses wrong, burns tokens
building the wrong thing, and you iterate forever. `prr` sits in front of your
agent and fixes the prompt first:

- **Confidence-gated questions** — only interrupts you when the prompt is actually
  ambiguous; crisp prompts pass straight through.
- **Reuses your agent** — no API keys. It rides on your already-authenticated
  `claude` / `codex` CLI, so refinement quality matches your own model.
- **Agent-agnostic** — works with **Claude Code, Codex, OpenCode, and Aider**, auto-detected (more to come).
- **One static binary** — installs anywhere, no runtime.

## Install

```bash
# Homebrew (macOS/Linux)
brew install ravistakumar/tap/prr

# Go
go install github.com/ravistakumar/prr/cmd/prr@latest

# Script
curl -fsSL https://raw.githubusercontent.com/ravistakumar/prr/main/install.sh | sh
```

## Usage

```bash
prr "refactor the auth module"                    # interactive, confirm, launch
prr --auto "fix the failing test in user_test.go" # one-shot, hands off to agent
prr --print "build a REST API" | pbcopy           # just give me the refined prompt
echo "vague idea" | prr --print                   # reads from stdin
```

## How it works

```
prr "add dark mode"
  → detect lightweight signals (language / framework / git)
  → ask your agent (headless) to optimize the prompt + score its confidence
  → confidence low?  → ask 1–3 targeted questions in the terminal
  → hand the sharpened prompt to claude / codex
```

`prr` is a helper, never a gatekeeper: if anything fails, it falls back to your
original prompt so it never blocks your work.

## Configuration

`~/.config/prr/config.toml` (overridden by `PRR_*` env vars, then flags):

```toml
agent      = "auto"      # auto | claude | codex | opencode | aider
mode       = "confirm"   # confirm | auto | print
threshold  = 0.7
max_rounds = 2

# Override an agent's command if it isn't on PATH under the default name:
[agents.opencode]
command = "opencode"
```

## Supported agents

`prr` auto-detects the first installed agent (in this order) or you can force one
with `--agent`:

| Agent | Refine call (headless) | Handoff |
| --- | --- | --- |
| Claude Code | `claude -p` | `claude "<prompt>"` |
| Codex | `codex exec` | `codex "<prompt>"` |
| OpenCode | `opencode run` | `opencode --prompt "<prompt>"` |
| Aider | `aider --yes --no-auto-commits --message` | `aider --yes --message "<prompt>"` (one-shot) |

Aider is edit-oriented and has no interactive "chat-only" mode, so its handoff
runs the refined prompt **one-shot** (`--message`) rather than opening a
persistent session. To add another agent, implement one case in
`internal/agent` — the prompt is always passed as the final argument.

## Contributing

Contributions welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). prr is written in
Go with a functional-core / imperative-shell design that keeps logic easy to test.

## License

[MIT](LICENSE)
