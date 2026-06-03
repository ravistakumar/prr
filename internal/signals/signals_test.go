package signals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		file string
		want string
	}{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"Cargo.toml", "rust"},
		{"pyproject.toml", "python"},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		s := Detect(dir)
		if s.Language != tc.want {
			t.Errorf("%s: language = %q, want %q", tc.file, s.Language, tc.want)
		}
	}
}

func TestDetectUnknownLanguage(t *testing.T) {
	s := Detect(t.TempDir())
	if s.Language != "unknown" {
		t.Errorf("language = %q, want unknown", s.Language)
	}
}

func TestDetectGitBranch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Detect(dir)
	if !s.IsGitRepo {
		t.Fatal("IsGitRepo = false, want true")
	}
	if s.GitBranch != "main" {
		t.Errorf("branch = %q, want main", s.GitBranch)
	}
}
