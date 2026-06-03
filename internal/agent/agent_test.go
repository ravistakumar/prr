package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/ravistakumar/prr/internal/config"
)

// buildFakeAgent compiles a stub binary that simulates a coding-agent CLI: it
// prints a canned JSON reply to stdout, and (when FAKE_ARGV_FILE is set in the
// environment) records the argv it was invoked with so tests can assert how
// prr built the command line.
func buildFakeAgent(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-agent build uses a unix-style path; skip on windows CI")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	code := `package main
import ("fmt";"os";"strings")
func main(){
	if p := os.Getenv("FAKE_ARGV_FILE"); p != "" {
		_ = os.WriteFile(p, []byte(strings.Join(os.Args[1:], "\n")), 0o644)
	}
	fmt.Println("{\"confidence\":0.8,\"refined_prompt\":\"refined\",\"questions\":[]}")
}
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

func recordedArgv(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read recorded argv: %v", err)
	}
	return strings.Split(string(data), "\n")
}

func TestCmdAgentAskReturnsStdout(t *testing.T) {
	bin := buildFakeAgent(t)
	a := newCmdAgent("fake", bin, []string{"exec"}, nil)
	if !a.Available() {
		t.Fatal("fake agent should be Available")
	}
	out, err := a.Ask(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if want := "refined"; !strings.Contains(out, want) {
		t.Fatalf("Ask output = %q, want it to contain %q", out, want)
	}
}

func TestAvailableFalseForMissingCommand(t *testing.T) {
	a := newCmdAgent("nope", "definitely-not-a-real-binary-xyz", []string{"exec"}, nil)
	if a.Available() {
		t.Fatal("Available should be false for a missing command")
	}
}

func TestAskBuildsArgvWithPromptLast(t *testing.T) {
	bin := buildFakeAgent(t)
	argvFile := filepath.Join(t.TempDir(), "argv")
	t.Setenv("FAKE_ARGV_FILE", argvFile)
	a := newCmdAgent("fake", bin, []string{"run"}, nil)
	if _, err := a.Ask(context.Background(), "META PROMPT"); err != nil {
		t.Fatal(err)
	}
	if got, want := recordedArgv(t, argvFile), []string{"run", "META PROMPT"}; !slices.Equal(got, want) {
		t.Fatalf("Ask argv = %v, want %v", got, want)
	}
}

func TestLaunchBuildsArgvWithPromptLast(t *testing.T) {
	bin := buildFakeAgent(t)
	argvFile := filepath.Join(t.TempDir(), "argv")
	t.Setenv("FAKE_ARGV_FILE", argvFile)
	a := newCmdAgent("fake", bin, nil, []string{"--prompt"})
	if err := a.Launch(context.Background(), "FINAL"); err != nil {
		t.Fatal(err)
	}
	if got, want := recordedArgv(t, argvFile), []string{"--prompt", "FINAL"}; !slices.Equal(got, want) {
		t.Fatalf("Launch argv = %v, want %v", got, want)
	}
}

// TestNewBuildsCorrectArgs is the key correctness check for every supported
// agent: it pins the exact non-interactive (Ask) and handoff (Launch) argument
// lists each CLI is driven with, derived from each tool's documented interface.
func TestNewBuildsCorrectArgs(t *testing.T) {
	cases := []struct {
		name   string
		ask    []string
		launch []string
	}{
		{"claude", []string{"-p"}, nil},
		{"codex", []string{"exec"}, nil},
		{"opencode", []string{"run"}, []string{"--prompt"}},
		{"aider", []string{"--yes", "--no-auto-commits", "--message"}, []string{"--yes", "--message"}},
	}
	for _, c := range cases {
		a, err := New(c.name, config.Defaults())
		if err != nil {
			t.Fatalf("New(%q): %v", c.name, err)
		}
		ca, ok := a.(*cmdAgent)
		if !ok {
			t.Fatalf("%s: New returned %T, want *cmdAgent", c.name, a)
		}
		if !slices.Equal(ca.askArgs, c.ask) {
			t.Errorf("%s askArgs = %v, want %v", c.name, ca.askArgs, c.ask)
		}
		if !slices.Equal(ca.launchArgs, c.launch) {
			t.Errorf("%s launchArgs = %v, want %v", c.name, ca.launchArgs, c.launch)
		}
	}
}

func TestNewUnknownAgentErrors(t *testing.T) {
	if _, err := New("does-not-exist", config.Defaults()); err == nil {
		t.Fatal("New should error on an unknown agent")
	}
}

func TestNewAppliesCommandOverride(t *testing.T) {
	cfg := config.Defaults()
	cfg.Agents["opencode"] = config.AgentConfig{Command: "opencode-nightly"}
	a, err := New("opencode", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := a.(*cmdAgent).command; got != "opencode-nightly" {
		t.Fatalf("command = %q, want opencode-nightly", got)
	}
}
