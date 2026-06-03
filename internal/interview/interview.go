// Package interview collects answers to clarifying questions. The Asker
// interface keeps the orchestrator independent of the terminal UI.
package interview

import (
	"github.com/charmbracelet/huh"

	"github.com/ravistakumar/prr/internal/optimize"
)

// Asker presents questions and returns the answered pairs.
type Asker interface {
	Ask(questions []string) ([]optimize.QA, error)
}

// ScriptedAsker is a deterministic Asker for tests. It returns Answers paired
// positionally with the questions; missing answers are empty strings.
type ScriptedAsker struct {
	Answers []string
}

func (s ScriptedAsker) Ask(questions []string) ([]optimize.QA, error) {
	out := make([]optimize.QA, len(questions))
	for i, q := range questions {
		ans := ""
		if i < len(s.Answers) {
			ans = s.Answers[i]
		}
		out[i] = optimize.QA{Question: q, Answer: ans}
	}
	return out, nil
}

// HuhAsker renders questions as an interactive Charm form. It is exercised
// manually and via the end-to-end path; unit coverage uses ScriptedAsker.
type HuhAsker struct{}

func (HuhAsker) Ask(questions []string) ([]optimize.QA, error) {
	answers := make([]string, len(questions))
	fields := make([]huh.Field, len(questions))
	for i, q := range questions {
		fields[i] = huh.NewInput().Title(q).Value(&answers[i])
	}
	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return nil, err
	}
	out := make([]optimize.QA, len(questions))
	for i, q := range questions {
		out[i] = optimize.QA{Question: q, Answer: answers[i]}
	}
	return out, nil
}
