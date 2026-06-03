package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// cmdAgent drives a CLI that takes a prompt as a single argument. askSub is the
// subcommand/flag for headless mode (e.g. "-p" for claude, "exec" for codex);
// Launch runs the bare command with the prompt for an interactive session.
type cmdAgent struct {
	name    string
	command string
	askSub  string
}

func newCmdAgent(name, command, askSub string) *cmdAgent {
	return &cmdAgent{name: name, command: command, askSub: askSub}
}

func (a *cmdAgent) Name() string { return a.name }

func (a *cmdAgent) Available() bool {
	_, err := exec.LookPath(a.command)
	return err == nil
}

func (a *cmdAgent) Ask(ctx context.Context, metaPrompt string) (string, error) {
	cmd := exec.CommandContext(ctx, a.command, a.askSub, metaPrompt)
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
	cmd := exec.CommandContext(ctx, a.command, finalPrompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
