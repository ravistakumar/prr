// Package handoff renders the refined prompt and asks the user to confirm
// before the agent is launched.
package handoff

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Confirm prints the refined prompt and reads a y/N answer. Empty input
// defaults to accept (the common case after the user has read the prompt).
func Confirm(out io.Writer, in io.Reader, refined string) (bool, error) {
	fmt.Fprintln(out, "\nRefined prompt:")
	fmt.Fprintln(out, "  "+strings.ReplaceAll(refined, "\n", "\n  "))
	fmt.Fprint(out, "\nLaunch agent with this prompt? [Y/n] ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
