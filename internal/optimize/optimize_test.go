package optimize

import (
	"strings"
	"testing"

	"github.com/ravistakumar/prr/internal/signals"
)

func TestBuildMetaPromptIncludesRawAndSignals(t *testing.T) {
	req := Request{
		RawPrompt: "add dark mode",
		Signals:   signals.Signals{Language: "go", IsGitRepo: true, GitBranch: "main"},
	}
	meta := BuildMetaPrompt(req)
	for _, want := range []string{"add dark mode", "go", "main", "confidence", "refined_prompt", "questions"} {
		if !strings.Contains(meta, want) {
			t.Errorf("meta prompt missing %q", want)
		}
	}
}

func TestBuildMetaPromptIncludesPriorAnswers(t *testing.T) {
	req := Request{
		RawPrompt: "add dark mode",
		Answers:   []QA{{Question: "Follow OS theme?", Answer: "yes"}},
	}
	meta := BuildMetaPrompt(req)
	if !strings.Contains(meta, "Follow OS theme?") || !strings.Contains(meta, "yes") {
		t.Error("meta prompt should include prior Q&A")
	}
}

func TestParseResultPlainJSON(t *testing.T) {
	raw := `{"confidence":0.9,"refined_prompt":"Add a dark mode toggle","questions":[]}`
	r, err := ParseResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Confidence != 0.9 || r.RefinedPrompt != "Add a dark mode toggle" || len(r.Questions) != 0 {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestParseResultFencedJSONWithProse(t *testing.T) {
	raw := "Sure! Here is the result:\n```json\n{\"confidence\":0.4,\"refined_prompt\":\"x\",\"questions\":[\"Toggle or OS?\"]}\n```\nHope that helps."
	r, err := ParseResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Confidence != 0.4 || len(r.Questions) != 1 || r.Questions[0] != "Toggle or OS?" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestParseResultNoJSONErrors(t *testing.T) {
	if _, err := ParseResult("I could not produce JSON."); err == nil {
		t.Fatal("expected error when no JSON object is present")
	}
}
