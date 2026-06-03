package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	c := Defaults()
	if c.Mode != ModeConfirm {
		t.Fatalf("default mode = %q, want confirm", c.Mode)
	}
	if c.Agent != "auto" {
		t.Fatalf("default agent = %q, want auto", c.Agent)
	}
	if c.Threshold != 0.7 {
		t.Fatalf("default threshold = %v, want 0.7", c.Threshold)
	}
	if c.MaxRounds != 2 {
		t.Fatalf("default max rounds = %d, want 2", c.MaxRounds)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	data := "mode = \"auto\"\nthreshold = 0.5\n\n[agents.claude]\ncommand = \"claude-beta\"\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode != ModeAuto {
		t.Fatalf("mode = %q, want auto", c.Mode)
	}
	if c.Threshold != 0.5 {
		t.Fatalf("threshold = %v, want 0.5", c.Threshold)
	}
	if c.MaxRounds != 2 {
		t.Fatalf("max rounds = %d, want default 2", c.MaxRounds)
	}
	if got := c.Agents["claude"].Command; got != "claude-beta" {
		t.Fatalf("claude command = %q, want claude-beta", got)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if c.Mode != ModeConfirm {
		t.Fatalf("mode = %q, want default confirm", c.Mode)
	}
}

func TestApplyEnv(t *testing.T) {
	c := Defaults()
	env := map[string]string{"PRR_MODE": "print", "PRR_THRESHOLD": "0.9"}
	c = c.ApplyEnv(func(k string) string { return env[k] })
	if c.Mode != ModePrint {
		t.Fatalf("mode = %q, want print", c.Mode)
	}
	if c.Threshold != 0.9 {
		t.Fatalf("threshold = %v, want 0.9", c.Threshold)
	}
}
