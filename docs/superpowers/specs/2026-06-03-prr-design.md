# prr — Prompt Refine & Run

**Status:** Design approved · **Date:** 2026-06-03 · **License:** MIT (open source)

## Summary

`prr` is a terminal-first, agent-agnostic **prompt firewall** for AI coding agents
(Claude Code, Codex). You pipe your raw intent through it; it refines the prompt,
gauges how confident it is that the prompt is buildable, asks targeted questions
*only when it is unsure*, then hands a sharpened prompt to your agent — so the
expensive agent builds the right thing on the first try.

The guiding principle: **`prr` is a helper, never a gatekeeper.** Every failure
path degrades toward letting the user proceed with their original prompt, never
toward blocking their work.

## Core loop

```
$ prr "add dark mode to my settings page"
   → detect lightweight signals (lang/framework/git)
   → ask the user's own agent (headless) to optimize + score confidence
   → confidence below threshold?  → ask 1–3 targeted questions in the terminal
   → assemble refined prompt
   → handoff:  --confirm (default) | --auto | --print
   → launch claude / codex with the sharpened prompt
```

## Locked-in decisions

| Decision          | Choice                                                             |
| ----------------- | ----------------------------------------------------------------- |
| Name              | `prr` (prompt-refine-run)                                         |
| Integration       | Standalone preprocessor (v1) → transparent wrapper layer (later)  |
| Optimizer engine  | Reuse the user's agent session (headless `claude -p` / `codex exec`) |
| Handoff           | Configurable: `--confirm` (default), `--auto`, `--print`          |
| Question logic    | Confidence-gated                                                  |
| Context awareness | Lightweight signals only (lang/framework/git detection)           |
| Stack             | Go (Charm TUI + `os/exec`)                                        |
| License           | MIT, open source                                                  |

## Why this engine choice matters

Because `prr` **reuses the agent's already-authenticated session** (it shells out
to `claude -p` / `codex exec` rather than calling an LLM API directly):

- **Zero config** — no separate API key to manage.
- **Quality scales with the user's own agent** — if they run Opus, their prompt
  optimization is Opus-grade for free.
- The usual reason to pick a JS/Python stack (rich LLM SDKs) evaporates — we call a
  subprocess, not an API. That removes the main argument against Go and tilts the
  decision toward whichever language has the best **distribution + TUI** story: Go.

The fragility (depending on each CLI's non-interactive flags) is isolated behind a
single `Agent` adapter interface, so a flag change touches one small file.

## Architecture

Functional core, imperative shell: a thin **impure shell** (`agent`, `signals`,
`handoff` — they touch the OS) wraps a **pure core** (`optimize`, `prompt`) where
all interesting logic lives and is trivially unit-testable with no mocks.

```
prr/
├── cmd/prr/main.go          # entrypoint, wires everything
├── internal/
│   ├── cli/                 # arg parsing, flags, config resolution (cobra)
│   ├── config/              # config.toml + env + flags merge
│   ├── signals/             # lightweight env detection (go.mod, package.json, git)
│   ├── agent/               # ◄── THE adapter boundary (only part touching ext CLIs)
│   │   ├── agent.go         #     Agent interface
│   │   ├── claude.go        #     Claude Code adapter (claude -p)
│   │   ├── codex.go         #     Codex adapter (codex exec)
│   │   └── detect.go        #     auto-pick installed agent
│   ├── optimize/            # builds meta-prompt, parses confidence + questions
│   ├── interview/           # Charm TUI Q&A loop (Huh forms)
│   ├── prompt/              # assembles final refined prompt
│   └── handoff/             # confirm / auto / print modes
└── ...
```

### Components

**1. `Agent` interface** — the only part that knows about external CLIs. Adding a
new agent (Gemini CLI, Aider, etc.) = one new file.

```go
type Agent interface {
    Name() string
    Available() bool                          // is the CLI installed + authed?
    Ask(ctx, meta Prompt) (Response, error)   // headless call for optimization
    Launch(ctx, finalPrompt string) error     // start the real dev session
}
```

**2. `optimize`** — builds the meta-prompt sent to the agent ("here is the user's
raw prompt + env signals; return JSON: `{confidence, refined_prompt, questions[]}`")
and parses the structured response. **Pure logic, no IO.**

**3. `interview`** — if `confidence < threshold`, renders questions as an interactive
Charm/Huh form. Skipped entirely when confident or in non-interactive mode.

**4. `signals`** — cheap, bounded filesystem peeks (does `go.mod` / `package.json` /
`Cargo.toml` exist? git repo? branch?) to enrich the meta-prompt.

**5. `handoff`** — routes the final prompt to confirm / auto / print.

## Data flow

```
prr "add dark mode to settings" --confirm
  1. cli/config: resolve flags + config.toml + env → mode=confirm, agent=auto, threshold=0.7
  2. agent/detect: pick installed+authed agent (claude); signals: detect go.mod/package.json/git
  3. optimize: build meta-prompt → agent.Ask() headless → JSON
       { confidence: 0.45, refined_prompt: "...",
         questions: ["Toggle in settings or follow OS theme?",
                     "Persist preference across sessions?"] }
  4. confidence 0.45 < 0.7 → LOW → interview: render questions (Charm/Huh), collect answers
  5. optimize round 2: re-send raw + answers → confidence 0.9 → proceed
       (loops max max_rounds, then proceeds regardless)
  6. handoff (confirm): show refined prompt + diff; [Enter]=accept / [e]=edit / [q]=abort
  7. agent.Launch(finalPrompt) → real claude/codex session
```

### Key behaviors

- **Confident-prompt fast path:** if step 3 returns `confidence ≥ threshold` with no
  questions, steps 4–5 are skipped — straight to handoff. A crisp prompt feels nearly
  instant.
- **Bounded interview:** the question loop caps at `max_rounds` (default 2). If still
  uncertain, proceed with best refinement rather than nagging forever.
- **`--print` / non-TTY:** steps 4–6 skipped; return best refinement to stdout
  (CI/pipe-friendly). Reads from stdin too.
- **`--auto`:** step 6 skipped — accept and launch immediately.

## CLI surface

```
prr [flags] "your prompt"

Handoff mode (mutually exclusive):
  --confirm        Show refined prompt, ask before launching   (default)
  --auto           Refine and launch immediately, no checkpoint
  --print          Output refined prompt to stdout, don't launch  (CI/pipe)

Agent selection:
  --agent <name>   Force claude | codex   (default: auto-detect)

Tuning:
  --threshold <f>  Confidence gate, 0.0–1.0   (default 0.7)
  --max-rounds <n> Max interview rounds        (default 2)
  --yes            Skip questions entirely, just optimize once
  --dry-run        Show the meta-prompt that would be sent, then exit

Utility:
  prr config       Print resolved config + detected agent/signals
  prr --version
```

### Examples

```bash
prr "refactor the auth module"                    # interactive, confirm, launch
prr --auto "fix the failing test in user_test.go" # one-shot, hands off to agent
prr --print "build a REST API" | pbcopy           # just give me the refined prompt
echo "vague idea" | prr --print                   # reads from stdin too
```

## Configuration

Resolution order (later overrides earlier):
**defaults → `~/.config/prr/config.toml` → env vars (`PRR_*`) → flags.**

```toml
# ~/.config/prr/config.toml
agent       = "auto"      # auto | claude | codex
mode        = "confirm"   # confirm | auto | print
threshold   = 0.7
max_rounds  = 2

[agents.claude]
command = "claude"        # override if not on PATH
[agents.codex]
command = "codex"
```

## Error handling

Guiding rule: **`prr` is a helper, never a gatekeeper.** Every failure path degrades
toward letting the user proceed with their original prompt.

| Failure                            | Behavior                                                                                          |
| ---------------------------------- | ------------------------------------------------------------------------------------------------- |
| No agent installed/authed          | Clear message ("Install Claude Code or Codex, or set `--agent`"), non-zero exit.                  |
| Agent headless call fails/times out | **Pass-through**: warn, proceed with the *original* prompt (optionally launch it).                |
| Agent returns unparseable JSON     | One bounded retry with stricter "JSON only" instruction; if still failing → pass-through + warn.  |
| Non-TTY but mode needs interaction | Auto-switch to `--print` behavior, note on stderr.                                                |
| User aborts at confirm (`q`)       | Clean exit code 0; original prompt printed so nothing is lost.                                    |
| Ctrl-C mid-interview               | Context cancellation propagates; child process killed cleanly.                                    |

## Testing strategy

- **Pure core (`optimize`, `prompt`, `config`, `signals`):** table-driven unit tests,
  no mocks. Bulk of coverage — confidence parsing, meta-prompt assembly, config
  precedence, signal detection against temp dirs.
- **`Agent` adapters:** tested against a **fake agent binary** (a tiny script echoing
  canned JSON) so subprocess plumbing is tested without calling real `claude`/`codex`
  or burning tokens.
- **`interview` TUI:** Bubble Tea's `teatest` harness for the form flow.
- **End-to-end:** one golden-path integration test using the fake agent, asserting the
  full pipeline (confident path + low-confidence interview path).

## Distribution (open source)

- **GoReleaser** + GitHub Actions: tag → cross-compiled binaries for
  darwin/linux/windows (amd64/arm64), checksums, GitHub Release.
- Install paths: **Homebrew tap**, **Scoop** (Windows), `go install`, raw binary
  download + `install.sh`.
- **Repo hygiene:** MIT license, `README` with demo GIF, `CONTRIBUTING.md`,
  conventional-commit-friendly, CI running `go test` + `golangci-lint` on PRs.

## Out of scope (v1)

- Transparent wrapper mode (`prr claude` / `prr codex`) — designed for, shipped later.
- Deep codebase scanning / RAG over the repo — only lightweight signals in v1.
- Direct LLM API providers (BYOK) — v1 reuses the agent session only.
- Hook/plugin integration into Claude Code or Codex config.

## Future directions

- Transparent wrapper mode as a thin layer over the standalone core.
- Optional BYOK provider so `prr` works without an agent CLI present.
- Opt-in deeper codebase awareness (README + file tree injection).
- Additional agent adapters (Gemini CLI, Aider).
- Prompt history / replay and shareable refined-prompt snippets.
