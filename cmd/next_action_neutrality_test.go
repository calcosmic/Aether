package cmd

// Phase 197 plan 01, constraint S-01: the resolver produces a PLATFORM-NEUTRAL
// command value and nothing else.
//
// `/ant-*` translation lives in the visual writer and applies only to verbs
// with a wrapper. JSON envelopes stay raw because wrappers and the TS host
// EXECUTE those strings -- handing `/ant-continue` to exec is a broken command,
// not a nicer one. So no platform's spelling may ever be baked into a value the
// resolver produces, and the resolver must ignore the platform entirely.

import (
	"os"
	"strings"
	"testing"
)

// recognisedPlatformValues are every value the runtime's own platform detector
// can settle on. The resolver must produce identical output for all of them.
var recognisedPlatformValues = []string{"claude", "opencode", "codex"}

func TestResolverCommandsArePlatformNeutral(t *testing.T) {
	original, had := os.LookupEnv("AETHER_PLATFORM")
	t.Cleanup(func() {
		if had {
			os.Setenv("AETHER_PLATFORM", original)
			return
		}
		os.Unsetenv("AETHER_PLATFORM")
	})

	assertNeutral := func(t *testing.T, platform, command string) {
		t.Helper()
		if command == "" {
			t.Fatalf("platform %q: the resolver produced no command", platform)
		}
		if !strings.HasPrefix(command, "aether ") {
			t.Errorf("platform %q: command %q is not in the runtime form a wrapper can execute", platform, command)
		}
		if strings.Contains(command, "/ant-") {
			t.Errorf("platform %q: command %q bakes in a platform's slash spelling; "+
				"handing that to exec is a broken command", platform, command)
		}
	}

	for _, tc := range nextActionLifecycleCases() {
		t.Run(tc.name, func(t *testing.T) {
			baseline := ""
			for _, platform := range recognisedPlatformValues {
				os.Setenv("AETHER_PLATFORM", platform)
				got := resolveNextAction(tc.input(t))

				assertNeutral(t, platform, got.Command)
				for _, alt := range got.Alternatives {
					assertNeutral(t, platform, alt.Command)
				}

				if baseline == "" {
					baseline = got.Command
					continue
				}
				if got.Command != baseline {
					t.Errorf("platform %q changed the recommended command to %q (was %q); "+
						"the resolver must ignore the platform entirely", platform, got.Command, baseline)
				}
			}
		})
	}
}

// TestCandidateSetIsPlatformNeutral checks the declaration itself, not just
// what happened to be emitted: an unreachable candidate carrying a slash form
// would be a defect waiting for the branch that reaches it.
func TestCandidateSetIsPlatformNeutral(t *testing.T) {
	for _, candidate := range nextActionCandidates {
		if !strings.HasPrefix(candidate.Template, "aether ") {
			t.Errorf("candidate %q spells %q, which is not the runtime form", candidate.Key, candidate.Template)
		}
		if strings.Contains(candidate.Template, "/ant-") {
			t.Errorf("candidate %q spells %q, baking in a platform's slash spelling", candidate.Key, candidate.Template)
		}
	}
}
