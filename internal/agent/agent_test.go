package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// buildFakeAgent compiles a stub binary that prints a canned reply to stdout
// when invoked, simulating `claude -p` / `codex exec`.
func buildFakeAgent(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-agent build uses a unix-style path; skip on windows CI")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	code := `package main
import ("fmt";"os")
func main(){ _ = os.Args; fmt.Println("{\"confidence\":0.8,\"refined_prompt\":\"refined\",\"questions\":[]}") }
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "fake-agent")
	out, err := exec.Command("go", "build", "-o", bin, src).CombinedOutput()
	if err != nil {
		t.Fatalf("build fake agent: %v\n%s", err, out)
	}
	return bin
}

func TestCmdAgentAskReturnsStdout(t *testing.T) {
	bin := buildFakeAgent(t)
	a := newCmdAgent("fake", bin, "exec")
	if !a.Available() {
		t.Fatal("fake agent should be Available")
	}
	out, err := a.Ask(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if want := "refined"; !contains(out, want) {
		t.Fatalf("Ask output = %q, want it to contain %q", out, want)
	}
}

func TestAvailableFalseForMissingCommand(t *testing.T) {
	a := newCmdAgent("nope", "definitely-not-a-real-binary-xyz", "exec")
	if a.Available() {
		t.Fatal("Available should be false for a missing command")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
