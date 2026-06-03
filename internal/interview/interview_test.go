package interview

import "testing"

func TestScriptedAskerPairsQuestionsWithAnswers(t *testing.T) {
	a := ScriptedAsker{Answers: []string{"yes", "persist it"}}
	qs := []string{"Follow OS theme?", "Persist preference?"}
	got, err := a.Ask(qs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d QA, want 2", len(got))
	}
	if got[0].Question != "Follow OS theme?" || got[0].Answer != "yes" {
		t.Errorf("QA[0] = %+v", got[0])
	}
	if got[1].Answer != "persist it" {
		t.Errorf("QA[1].Answer = %q, want 'persist it'", got[1].Answer)
	}
}

func TestScriptedAskerMissingAnswerIsEmpty(t *testing.T) {
	a := ScriptedAsker{Answers: []string{"only one"}}
	got, err := a.Ask([]string{"Q1", "Q2"})
	if err != nil {
		t.Fatal(err)
	}
	if got[1].Answer != "" {
		t.Errorf("QA[1].Answer = %q, want empty", got[1].Answer)
	}
}
