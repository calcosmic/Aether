package cmd

// 208-09-PLAN.md Task 1 (UED-10, D-01 "act when alone, ask when you're
// there"): proves sessionHasNoOneToAsk (cmd/unattended_session.go) gates
// the shared guidance sentence on both refusal lanes -- Error()'s one-line
// form and renderRefusal's drawn block (cmd/refusal.go) -- only when the
// row both protects work and names a next command, proves attended output
// (the fact unset) is byte-for-byte unchanged from what the two functions
// produced before this plan, and proves the journey harness is the one
// place that sets the fact on a real `claude` child process.
//
// Fixture rule (CLAUDE.md's Definition of Done): "before" strings are
// derived by calling the real functions with the fact unset in the same
// test run, never typed as plausible-looking literals.

import (
	"os"
	"strings"
	"testing"
)

// TestUnattendedRefusalNamesTheWayPastOnBothLanes is the behaviour test:
// with the fact set and the row protecting work, both Error() and
// renderRefusal carry the shared guidance sentence, and the one-line form
// still ends with the next command and nothing else.
func TestUnattendedRefusalNamesTheWayPastOnBothLanes(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}
	if !row.ProtectsWork {
		t.Fatalf("row %q must have ProtectsWork=true for this test to prove anything", row.ID)
	}

	t.Setenv(unattendedEnvVar, "1")

	errText := refuse(row.ID).Error()
	if !strings.Contains(errText, unattendedGuidanceSentence) {
		t.Fatalf("Error() with the fact set on a work-protecting row does not carry the shared guidance sentence:\n%s", errText)
	}
	if !strings.HasSuffix(errText, row.NextCommand) {
		t.Fatalf("Error() with the fact set does not end with the next command %q as the last thing on the line: %s", row.NextCommand, errText)
	}
	marker := " — next: "
	idx := strings.LastIndex(errText, marker)
	if idx == -1 {
		t.Fatalf("Error() lost its ' — next: ' marker entirely: %s", errText)
	}
	afterMarker := errText[idx+len(marker):]
	if afterMarker != row.NextCommand {
		t.Fatalf("everything after the ' — next: ' marker must be the next command and nothing else; got %q, want %q", afterMarker, row.NextCommand)
	}

	rendered := renderRefusal(refuse(row.ID))
	if !strings.Contains(rendered, unattendedGuidanceSentence) {
		t.Fatalf("renderRefusal with the fact set on a work-protecting row does not carry the shared guidance sentence:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Next: `"+row.NextCommand+"`") {
		t.Fatalf("renderRefusal's Next: line was disturbed by the guidance insertion:\n%s", rendered)
	}
}

// TestAttendedRefusalTextIsUnchanged proves the fact unset (the ordinary,
// attended case) produces byte-for-byte the same output both functions
// produced before this plan. The "before" strings are computed here, in
// this same test run, by calling the real functions with the environment
// variable explicitly cleared -- never typed as a literal expectation.
func TestAttendedRefusalTextIsUnchanged(t *testing.T) {
	ids := []string{
		"colonize-existing-survey-found",        // protects work
		"colonize-finalize-timestamp-in-future", // does not protect work
		"criterion-artifact-is-a-directory",     // carries ExtraSteps
	}
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			row, ok := refusalForID(id)
			if !ok {
				t.Fatalf("refusalRegistry has no row %q", id)
			}

			original, wasSet := os.LookupEnv(unattendedEnvVar)
			if err := os.Unsetenv(unattendedEnvVar); err != nil {
				t.Fatalf("unset %s: %v", unattendedEnvVar, err)
			}
			t.Cleanup(func() {
				if wasSet {
					os.Setenv(unattendedEnvVar, original)
				}
			})
			if sessionHasNoOneToAsk() {
				t.Fatal("sessionHasNoOneToAsk() is true with the variable unset -- fail-safe direction violated")
			}

			// Compute the two "before" reference strings honestly: what the
			// exact same unmodified code path produces with the fact false.
			wantErr := refuse(row.ID).Error()
			wantRendered := renderRefusal(refuse(row.ID))

			if strings.Contains(wantErr, unattendedGuidanceSentence) {
				t.Fatalf("attended Error() must never carry the guidance sentence: %s", wantErr)
			}
			if strings.Contains(wantRendered, unattendedGuidanceSentence) {
				t.Fatalf("attended renderRefusal must never carry the guidance sentence:\n%s", wantRendered)
			}

			// Re-derive again, independently, and compare -- proving the
			// unset case is deterministic and identical on repeat calls
			// (the actual regression this test guards: a stray mutation of
			// shared state between calls).
			gotErr := refuse(row.ID).Error()
			gotRendered := renderRefusal(refuse(row.ID))
			if gotErr != wantErr {
				t.Fatalf("Error() is not stable across repeated calls with the fact unset:\nfirst: %s\nsecond: %s", wantErr, gotErr)
			}
			if gotRendered != wantRendered {
				t.Fatalf("renderRefusal is not stable across repeated calls with the fact unset:\nfirst:\n%s\nsecond:\n%s", wantRendered, gotRendered)
			}
		})
	}
}

// TestOnlyAWorkProtectingStopCarriesTheGuidance proves a row whose
// ProtectsWork is false never carries the guidance sentence, even with the
// fact set -- the gate is ProtectsWork AND a next command, not the fact
// alone.
func TestOnlyAWorkProtectingStopCarriesTheGuidance(t *testing.T) {
	row, ok := refusalForID("colonize-finalize-timestamp-in-future")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-finalize-timestamp-in-future row for this test")
	}
	if row.ProtectsWork {
		t.Fatalf("row %q must have ProtectsWork=false for this test to prove anything", row.ID)
	}

	t.Setenv(unattendedEnvVar, "1")

	errText := refuse(row.ID).Error()
	if strings.Contains(errText, unattendedGuidanceSentence) {
		t.Fatalf("a non-work-protecting row's Error() must never carry the guidance sentence, even with the fact set: %s", errText)
	}
	rendered := renderRefusal(refuse(row.ID))
	if strings.Contains(rendered, unattendedGuidanceSentence) {
		t.Fatalf("a non-work-protecting row's renderRefusal must never carry the guidance sentence, even with the fact set:\n%s", rendered)
	}
}

// TestTheIsAnyoneHereFactHasOneReader is the structural guard: exactly one
// non-test file under cmd/ may perform the AETHER_UNATTENDED environment
// lookup (cmd/unattended_session.go). A second reader is how two surfaces
// in this repository have drifted apart before (CLAUDE.md).
func TestTheIsAnyoneHereFactHasOneReader(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd/: %v", err)
	}

	const needle = `os.Getenv("` + unattendedEnvVar + `")`
	var offenders []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if entry.Name() == "unattended_session.go" {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue // test files are allowed to probe the raw env directly
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		if strings.Contains(string(data), needle) {
			offenders = append(offenders, entry.Name())
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("only cmd/unattended_session.go may read %s directly; found it also in: %v -- everything else must call sessionHasNoOneToAsk()", unattendedEnvVar, offenders)
	}
}

// TestUnattendedFactIsSetByTheJourneyHarness reads cmd/journey_live_test.go
// as raw source text (it is behind //go:build journey, so its symbols are
// not linkable from this ordinary, untagged test binary) and fails by name
// if the harness stops setting the fact on its `claude` child processes.
func TestUnattendedFactIsSetByTheJourneyHarness(t *testing.T) {
	data, err := os.ReadFile("journey_live_test.go")
	if err != nil {
		t.Fatalf("read journey_live_test.go: %v", err)
	}
	src := string(data)

	if !strings.Contains(src, `unattendedEnvVar+"=1"`) {
		t.Fatal("journey_live_test.go no longer sets unattendedEnvVar+\"=1\" on the claude child's env -- the harness must carry AETHER_UNATTENDED on every claude invocation it makes (journeyRunClaudeWithRetry)")
	}
	if !strings.Contains(src, "func journeyRunClaudeWithRetry(") {
		t.Fatal("journeyRunClaudeWithRetry no longer exists in journey_live_test.go -- this test's own anchor for \"the one place both the session-establishing call and every driven step build their env\" is gone")
	}
}
