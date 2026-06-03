// Package config resolves prr configuration from defaults, a TOML file,
// environment variables, and (in the cli layer) flags — in that precedence.
package config

import (
	"errors"
	"io/fs"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Mode is the handoff behavior after a prompt is refined.
type Mode string

const (
	ModeConfirm Mode = "confirm"
	ModeAuto    Mode = "auto"
	ModePrint   Mode = "print"
)

// AgentConfig overrides how a named agent CLI is invoked.
type AgentConfig struct {
	Command string `toml:"command"`
}

// Config is the fully resolved prr configuration.
type Config struct {
	Agent     string                 `toml:"agent"`
	Mode      Mode                   `toml:"mode"`
	Threshold float64                `toml:"threshold"`
	MaxRounds int                    `toml:"max_rounds"`
	Agents    map[string]AgentConfig `toml:"agents"`
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	return Config{
		Agent:     "auto",
		Mode:      ModeConfirm,
		Threshold: 0.7,
		MaxRounds: 2,
		Agents: map[string]AgentConfig{
			"claude":   {Command: "claude"},
			"codex":    {Command: "codex"},
			"opencode": {Command: "opencode"},
			"aider":    {Command: "aider"},
		},
	}
}

// Load reads a TOML file over the defaults. A missing file is not an error:
// the defaults are returned unchanged.
// An [agents] table in the file is merged over the defaults, so omitting an
// agent there keeps its built-in default rather than removing it.
func Load(path string) (Config, error) {
	c := Defaults()
	_, err := toml.DecodeFile(path, &c)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return Config{}, err
	}
	if c.Agents == nil {
		c.Agents = Defaults().Agents
	}
	return c, nil
}

// ApplyEnv overlays PRR_* environment variables. getenv lets tests inject env.
func (c Config) ApplyEnv(getenv func(string) string) Config {
	if v := getenv("PRR_AGENT"); v != "" {
		c.Agent = v
	}
	if v := getenv("PRR_MODE"); v != "" {
		// PRR_MODE is cast as-is and not validated against the three known modes;
		// callers validate downstream.
		c.Mode = Mode(v)
	}
	if v := getenv("PRR_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Threshold = f
		}
	}
	if v := getenv("PRR_MAX_ROUNDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxRounds = n
		}
	}
	return c
}
