package run

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ravistakumar/prr/internal/config"
	"github.com/ravistakumar/prr/internal/interview"
)

var errFake = errors.New("fake agent failure")

// fakeAgent implements agent.Agent in-memory. replies are returned from Ask in
// order; launched records the prompt passed to Launch.
type fakeAgent struct {
	replies  []string
	asks     []string
	launched string
}

func (f *fakeAgent) Name() string    { return "fake" }
func (f *fakeAgent) Available() bool { return true }
func (f *fakeAgent) Ask(_ context.Context, meta string) (string, error) {
	f.asks = append(f.asks, meta)
	r := f.replies[0]
	if len(f.replies) > 1 {
		f.replies = f.replies[1:]
	}
	return r, nil
}
func (f *fakeAgent) Launch(_ context.Context, final string) error {
	f.launched = final
	return nil
}

func baseCfg(mode config.Mode) config.Config {
	c := config.Defaults()
	c.Mode = mode
	return c
}

func TestRunConfidentFastPathPrints(t *testing.T) {
	fa := &fakeAgent{replies: []string{`{"confidence":0.95,"refined_prompt":"REFINED","questions":[]}`}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModePrint), Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "add dark mode"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "REFINED") {
		t.Fatalf("print mode should output refined prompt, got %q", out.String())
	}
	if len(fa.asks) != 1 {
		t.Fatalf("confident prompt should ask once, asked %d", len(fa.asks))
	}
	if fa.launched != "" {
		t.Fatal("print mode must not launch the agent")
	}
}

func TestRunLowConfidenceRunsInterviewThenAuto(t *testing.T) {
	fa := &fakeAgent{replies: []string{
		`{"confidence":0.3,"refined_prompt":"V1","questions":["Follow OS theme?"]}`,
		`{"confidence":0.9,"refined_prompt":"V2-FINAL","questions":[]}`,
	}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{Answers: []string{"yes"}}, Cfg: baseCfg(config.ModeAuto), Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "add dark mode"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "V2-FINAL" {
		t.Fatalf("auto mode should launch final prompt, launched %q", fa.launched)
	}
	if len(fa.asks) != 2 {
		t.Fatalf("low confidence should trigger a second round, asked %d", len(fa.asks))
	}
	if !strings.Contains(fa.asks[1], "yes") {
		t.Error("second round meta-prompt should include the interview answer")
	}
}

func TestRunBoundedByMaxRounds(t *testing.T) {
	// Always low confidence: must stop after MaxRounds asks (no infinite loop).
	low := `{"confidence":0.1,"refined_prompt":"STILL","questions":["?"]}`
	fa := &fakeAgent{replies: []string{low, low, low, low}}
	cfg := baseCfg(config.ModeAuto)
	cfg.MaxRounds = 2
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{Answers: []string{"x"}}, Cfg: cfg, Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "vague"); err != nil {
		t.Fatal(err)
	}
	if len(fa.asks) != 2 {
		t.Fatalf("should ask exactly MaxRounds=2 times, asked %d", len(fa.asks))
	}
	if fa.launched != "STILL" {
		t.Fatalf("should proceed with best refinement, launched %q", fa.launched)
	}
}

func TestRunAgentFailureFallsBackToOriginal(t *testing.T) {
	fa := &failingAgent{}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeAuto), Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "original prompt"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "original prompt" {
		t.Fatalf("on Ask failure, should launch original prompt, launched %q", fa.launched)
	}
	if !strings.Contains(out.String(), "warning") {
		t.Error("pass-through should warn the user on stderr")
	}
}

type failingAgent struct{ launched string }

func (f *failingAgent) Name() string    { return "failing" }
func (f *failingAgent) Available() bool { return true }
func (f *failingAgent) Ask(context.Context, string) (string, error) {
	return "", errFake
}
func (f *failingAgent) Launch(_ context.Context, final string) error {
	f.launched = final
	return nil
}

func TestRunRetriesOnUnparseableJSON(t *testing.T) {
	// First reply is not JSON; prr must retry once, then succeed.
	fa := &fakeAgent{replies: []string{
		"sorry, I cannot answer in JSON",
		`{"confidence":0.95,"refined_prompt":"RECOVERED","questions":[]}`,
	}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeAuto), Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "RECOVERED" {
		t.Fatalf("retry should recover and launch, launched %q", fa.launched)
	}
	if len(fa.asks) != 2 {
		t.Fatalf("expected 1 ask + 1 retry = 2, got %d", len(fa.asks))
	}
}

func TestRunEmptyRefinementFallsBackToOriginal(t *testing.T) {
	fa := &fakeAgent{replies: []string{`{"confidence":0.95,"refined_prompt":"","questions":[]}`}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeAuto), Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "ORIGINAL"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "ORIGINAL" {
		t.Fatalf("empty refinement should fall back to original, launched %q", fa.launched)
	}
	if !strings.Contains(out.String(), "warning") {
		t.Error("empty refinement should warn on stderr")
	}
}

func TestRunConfirmModeLaunchesOnYes(t *testing.T) {
	fa := &fakeAgent{replies: []string{`{"confidence":0.95,"refined_prompt":"REFINED","questions":[]}`}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeConfirm), Interactive: true, Out: &out, Err: &out, In: strings.NewReader("y\n")}
	if err := r.Run(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "REFINED" {
		t.Fatalf("confirm+yes should launch refined, launched %q", fa.launched)
	}
}

func TestRunConfirmModeRejectsOnNo(t *testing.T) {
	fa := &fakeAgent{replies: []string{`{"confidence":0.95,"refined_prompt":"REFINED","questions":[]}`}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeConfirm), Interactive: true, Out: &out, Err: &out, In: strings.NewReader("n\n")}
	if err := r.Run(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "" {
		t.Fatalf("confirm+no should NOT launch, launched %q", fa.launched)
	}
	if !strings.Contains(out.String(), "REFINED") {
		t.Error("confirm+no should still print the refined prompt so nothing is lost")
	}
}

func TestRunConfirmNonInteractiveDowngradesToPrint(t *testing.T) {
	fa := &fakeAgent{replies: []string{`{"confidence":0.95,"refined_prompt":"REFINED","questions":[]}`}}
	var out bytes.Buffer
	r := Runner{Agent: fa, Asker: interview.ScriptedAsker{}, Cfg: baseCfg(config.ModeConfirm), Interactive: false, Out: &out, Err: &out, In: strings.NewReader("")}
	if err := r.Run(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if fa.launched != "" {
		t.Fatalf("non-interactive confirm should downgrade to print (no launch), launched %q", fa.launched)
	}
	if !strings.Contains(out.String(), "REFINED") {
		t.Error("non-interactive confirm should print the refined prompt")
	}
}
