// Package run is the orchestrator: it ties signals, optimize, interview, and
// handoff together. Its guiding rule is "helper, never gatekeeper" — every
// failure path degrades toward proceeding with the user's original prompt.
package run

import (
	"context"
	"fmt"
	"io"

	"github.com/ravistakumar/prr/internal/agent"
	"github.com/ravistakumar/prr/internal/config"
	"github.com/ravistakumar/prr/internal/handoff"
	"github.com/ravistakumar/prr/internal/interview"
	"github.com/ravistakumar/prr/internal/optimize"
	"github.com/ravistakumar/prr/internal/signals"
)

// Runner holds the injected dependencies for one prr invocation.
type Runner struct {
	Agent       agent.Agent
	Asker       interview.Asker
	Cfg         config.Config
	Dir         string    // working dir for signal detection ("" → cwd)
	Interactive bool       // false (e.g. piped/CI) downgrades confirm→print
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
}

// Run executes the refine → interview → handoff pipeline for rawPrompt.
func (r Runner) Run(ctx context.Context, rawPrompt string) error {
	sig := signals.Detect(r.dir())

	final, ok := r.refine(ctx, rawPrompt, sig)
	if !ok {
		// Pass-through: optimization failed, proceed with the original prompt.
		fmt.Fprintln(r.Err, "warning: prompt optimization failed; using your original prompt")
		final = rawPrompt
	}

	return r.handoff(ctx, final)
}

// refine runs up to MaxRounds optimization rounds, interviewing between rounds
// while confidence is below threshold. The bool is false if every round failed.
func (r Runner) refine(ctx context.Context, raw string, sig signals.Signals) (string, bool) {
	var answers []optimize.QA
	best := ""
	any := false

	for round := 0; round < r.Cfg.MaxRounds; round++ {
		req := optimize.Request{RawPrompt: raw, Signals: sig, Answers: answers}
		res, err := r.askParse(ctx, optimize.BuildMetaPrompt(req))
		if err != nil {
			break
		}
		any = true
		if res.RefinedPrompt != "" {
			best = res.RefinedPrompt
		}
		if res.Confidence >= r.Cfg.Threshold || len(res.Questions) == 0 {
			return best, true
		}
		if round == r.Cfg.MaxRounds-1 {
			break // bounded: do not interview again, proceed with best
		}
		qa, err := r.Asker.Ask(res.Questions)
		if err != nil {
			break
		}
		answers = append(answers, qa...)
	}
	return best, any
}

// askParse asks the agent and parses the reply, with one bounded retry using a
// stricter "JSON only" instruction when the first reply is not valid JSON.
func (r Runner) askParse(ctx context.Context, meta string) (optimize.Result, error) {
	reply, err := r.Agent.Ask(ctx, meta)
	if err != nil {
		return optimize.Result{}, err
	}
	if res, err := optimize.ParseResult(reply); err == nil {
		return res, nil
	}
	strict := meta + "\n\nIMPORTANT: Your previous reply was not valid JSON. " +
		"Respond with the JSON object ONLY — no prose, no code fences."
	reply, err = r.Agent.Ask(ctx, strict)
	if err != nil {
		return optimize.Result{}, err
	}
	return optimize.ParseResult(reply)
}

// handoff routes the final prompt according to the configured mode.
func (r Runner) handoff(ctx context.Context, final string) error {
	mode := r.Cfg.Mode
	if mode == config.ModeConfirm && !r.Interactive {
		mode = config.ModePrint // cannot prompt without a TTY
	}

	switch mode {
	case config.ModePrint:
		fmt.Fprintln(r.Out, final)
		return nil
	case config.ModeAuto:
		return r.Agent.Launch(ctx, final)
	case config.ModeConfirm:
		ok, err := handoff.Confirm(r.Out, r.In, final)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(r.Out, final) // nothing lost: print so user can reuse it
			return nil
		}
		return r.Agent.Launch(ctx, final)
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

func (r Runner) dir() string {
	if r.Dir != "" {
		return r.Dir
	}
	return "."
}
