package handoff

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmAccept(t *testing.T) {
	var out bytes.Buffer
	ok, err := Confirm(&out, strings.NewReader("y\n"), "refined prompt")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected accept on 'y'")
	}
	if !strings.Contains(out.String(), "refined prompt") {
		t.Error("confirm should display the refined prompt")
	}
}

func TestConfirmReject(t *testing.T) {
	var out bytes.Buffer
	ok, err := Confirm(&out, strings.NewReader("n\n"), "refined prompt")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected reject on 'n'")
	}
}

func TestConfirmEmptyDefaultsToAccept(t *testing.T) {
	var out bytes.Buffer
	ok, err := Confirm(&out, strings.NewReader("\n"), "refined prompt")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("empty input should default to accept")
	}
}
