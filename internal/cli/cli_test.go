package cli

import (
	"testing"

	"github.com/ravistakumar/prr/internal/config"
)

func TestResolveModeFlagsOverrideConfig(t *testing.T) {
	base := config.Defaults() // mode=confirm
	got := resolveMode(base, flagSet{auto: true})
	if got != config.ModeAuto {
		t.Fatalf("--auto should win, got %q", got)
	}
	got = resolveMode(base, flagSet{print: true})
	if got != config.ModePrint {
		t.Fatalf("--print should win, got %q", got)
	}
	got = resolveMode(base, flagSet{}) // no flag → keep config value
	if got != config.ModeConfirm {
		t.Fatalf("no flag should keep config mode, got %q", got)
	}
}
