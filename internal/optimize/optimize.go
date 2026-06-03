// Package optimize builds the meta-prompt sent to the user's agent and parses
// the structured response. It performs no IO and is fully unit-testable.
package optimize

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ravistakumar/prr/internal/signals"
)

// QA is one answered clarifying question from a prior interview round.
type QA struct {
	Question string
	Answer   string
}

// Request is the input to a single optimization round.
type Request struct {
	RawPrompt string
	Signals   signals.Signals
	Answers   []QA
}

// Result is the structured response we expect back from the agent.
type Result struct {
	Confidence    float64  `json:"confidence"`
	RefinedPrompt string   `json:"refined_prompt"`
	Questions     []string `json:"questions"`
}

// BuildMetaPrompt renders the instruction we send to the agent in headless mode.
func BuildMetaPrompt(req Request) string {
	var b strings.Builder
	b.WriteString("You are a prompt-refinement assistant for an AI coding agent.\n")
	b.WriteString("Given a developer's raw prompt, return ONLY a JSON object with keys:\n")
	b.WriteString(`  "confidence" (0.0-1.0: how buildable/unambiguous the prompt is),` + "\n")
	b.WriteString(`  "refined_prompt" (an improved, specific version of the prompt),` + "\n")
	b.WriteString(`  "questions" (an array of at most 3 targeted clarifying questions; empty if confident).` + "\n\n")
	b.WriteString("Environment signals:\n")
	b.WriteString(fmt.Sprintf("  language: %s\n", req.Signals.Language))
	if req.Signals.Framework != "" {
		b.WriteString(fmt.Sprintf("  framework: %s\n", req.Signals.Framework))
	}
	if req.Signals.IsGitRepo {
		b.WriteString(fmt.Sprintf("  git branch: %s\n", req.Signals.GitBranch))
	}
	b.WriteString("\nRaw prompt:\n")
	b.WriteString(req.RawPrompt + "\n")
	if len(req.Answers) > 0 {
		b.WriteString("\nThe developer already answered these clarifying questions:\n")
		for _, qa := range req.Answers {
			b.WriteString(fmt.Sprintf("  Q: %s\n  A: %s\n", qa.Question, qa.Answer))
		}
		b.WriteString("Incorporate these answers and raise confidence accordingly.\n")
	}
	b.WriteString("\nRespond with the JSON object only.\n")
	return b.String()
}

// ParseResult tolerantly extracts the JSON object from the agent's reply,
// which may be wrapped in markdown fences or surrounded by prose.
func ParseResult(raw string) (Result, error) {
	obj := extractJSONObject(raw)
	if obj == "" {
		return Result{}, errors.New("no JSON object found in agent response")
	}
	var r Result
	if err := json.Unmarshal([]byte(obj), &r); err != nil {
		return Result{}, fmt.Errorf("parse agent JSON: %w", err)
	}
	return r, nil
}

// extractJSONObject returns the substring from the first '{' to its matching
// closing '}', accounting for nesting. Returns "" if none is found.
// NOTE: brace characters inside JSON string values are not skipped, so a lone
// '}' inside a string literal can end the match early. This is rare for
// natural-language prompts, and the run layer retries on a parse failure.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
