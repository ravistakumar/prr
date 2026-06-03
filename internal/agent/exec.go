package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// cmdAgent drives a CLI that takes a prompt as its final argument. askArgs are
// the fixed args for the headless refine call (e.g. ["-p"] for claude,
// ["run"] for opencode); launchArgs are the fixed args for the interactive
// handoff (e.g. nil for claude, ["--prompt"] for opencode). The prompt is
// always appended after these args.
type cmdAgent struct {
	name       string
	command    string
	askArgs    []string
	launchArgs []string
}

func newCmdAgent(name, command string, askArgs, launchArgs []string) *cmdAgent {
	return &cmdAgent{name: name, command: command, askArgs: askArgs, launchArgs: launchArgs}
}

func (a *cmdAgent) Name() string { return a.name }

func (a *cmdAgent) Available() bool {
	_, err := exec.LookPath(a.command)
	return err == nil
}

// argv builds the full argument list for the command: the fixed args followed
// by the prompt. A fresh slice is returned so the stored args are never mutated.
func argv(fixed []string, prompt string) []string {
	out := make([]string, 0, len(fixed)+1)
	out = append(out, fixed...)
	return append(out, prompt)
}

func (a *cmdAgent) Ask(ctx context.Context, metaPrompt string) (string, error) {
	cmd := exec.CommandContext(ctx, a.command, argv(a.askArgs, metaPrompt)...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		if errOut.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(errOut.String()))
		}
		return "", err
	}
	return out.String(), nil
}

func (a *cmdAgent) Launch(ctx context.Context, finalPrompt string) error {
	cmd := exec.CommandContext(ctx, a.command, argv(a.launchArgs, finalPrompt)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
