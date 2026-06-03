// Package signals does cheap, bounded filesystem peeks to characterize the
// working directory (language, framework, git status) without a full scan.
package signals

import (
	"os"
	"path/filepath"
	"strings"
)

// Signals describes the lightweight environment context for a prompt.
type Signals struct {
	Language  string // "go" | "node" | "rust" | "python" | "unknown"
	Framework string // best-effort, may be empty
	IsGitRepo bool
	GitBranch string
}

// Detect inspects dir for well-known marker files and git metadata.
func Detect(dir string) Signals {
	s := Signals{Language: "unknown"}
	switch {
	case exists(dir, "go.mod"):
		s.Language = "go"
	case exists(dir, "package.json"):
		s.Language = "node"
	case exists(dir, "Cargo.toml"):
		s.Language = "rust"
	case exists(dir, "pyproject.toml"), exists(dir, "requirements.txt"):
		s.Language = "python"
	}
	if exists(dir, ".git") {
		s.IsGitRepo = true
		s.GitBranch = gitBranch(filepath.Join(dir, ".git", "HEAD"))
	}
	return s
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// gitBranch reads a .git/HEAD file and returns the branch name, or "" if
// detached or unreadable.
func gitBranch(headPath string) string {
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if strings.HasPrefix(line, prefix) {
		return strings.TrimPrefix(line, prefix)
	}
	return ""
}
