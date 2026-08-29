package cmd

// Phase 197 plan 06, task 1 -- pausing, resuming and updating end with the
// card.
//
// These three are where the owner is stopping for the day or picking back up,
// which is exactly when wrong advice costs most. The pause card carried a real
// defect: its two alternatives were "aether resume-colony" and "aether
// resume", and "resume" is a declared Cobra alias of "resume-colony" -- the
// same command, offered twice, with one of them called "the quick version"
// when it is not a different view at all.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// sessionReadyState is a runnable one-phase project, written the way the
// runtime writes one.
func sessionReadyState(t *testing.T, dataDir string) {
	t.Helper()
	writeLifecycleState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         fixtureGoal("Ship the billing rewrite"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseReady),
			fixturePhase(2, "Billing engine", colony.PhasePending),
		}},
	})
}

func sessionLifecycleSurfaces() []lifecycleSurfaceCase {
	return []lifecycleSurfaceCase{
		{
			name:    "pausing",
			args:    []string{"pause"},
			command: "pause",
			prepare: sessionReadyState,
			keep:    []string{"Handoff"},
		},
		{
			name: "resuming, full form",
			args: []string{"resume-colony"},
			// "resume-colony" itself is not fed to the resolver as the
			// LastCommand: "colony" is a word this repo invented (S-05), and
			// the production code passes the plain phrase below instead so
			// the "what changed" sentence never needs it explained twice.
			command: "picking the project back up",
			prepare: sessionReadyState,
			keep:    []string{"Goal: "},
		},
		{
			name:    "resuming, compact form",
			args:    []string{"resume-dashboard"},
			command: "resume-dashboard",
			prepare: sessionReadyState,
			keep:    []string{"Goal: "},
		},
	}
}

// TestSessionLifecycleCardsComeFromTheResolver is the screen half for
// pausing and resuming in both its forms.
func TestSessionLifecycleCardsComeFromTheResolver(t *testing.T) {
	for _, surface := range sessionLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, false)
			if !strings.Contains(run.visual, run.card) {
				t.Errorf("%s does not end with the shared card.\n--- the card the resolver produced ---\n%s\n--- what was printed ---\n%s",
					surface.name, run.card, run.visual)
			}
			for _, keep := range surface.keep {
				if !strings.Contains(run.visual, keep) {
					t.Errorf("%s lost %q from above the closing block -- only the what-next block was supposed to change",
						surface.name, keep)
				}
			}
		})
	}
}

// TestSessionLifecycleEnvelopesMatchTheirCards is the machine-readable half.
func TestSessionLifecycleEnvelopesMatchTheirCards(t *testing.T) {
	for _, surface := range sessionLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, true)
			assertEnvelopeMatchesCard(t, surface.name, run)
		})
	}
}

// TestMigratedSessionClosingsSayItOnce -- the card carries the "is it safe to
// close this chat" verdict now, so no migrated screen may also carry the old
// sentence.
func TestMigratedSessionClosingsSayItOnce(t *testing.T) {
	for _, surface := range sessionLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, false)
			if strings.Contains(run.visual, "clear your context") {
				t.Errorf("%s still prints the old context sentence; the card carries that verdict now", surface.name)
			}
			if count := countCloseThisChatSentences(run.visual); count > 1 {
				t.Errorf("%s tells the owner whether to close the chat %d times", surface.name, count)
			}
		})
	}
}

// allBacktickCommandsRe pulls every runtime-form command quoted on a card,
// not just the first one -- the duplicate defect this test exists to catch is
// a SECOND mention of a command already recommended once.
var allBacktickCommandsRe = regexp.MustCompile("Run `(aether [^`]+)`")

func allCommandsInCard(card string) []string {
	var out []string
	for _, match := range allBacktickCommandsRe.FindAllStringSubmatch(card, -1) {
		out = append(out, match[1])
	}
	return out
}

// TestNoCardOffersTheSameCommandTwice lives in next_action_card_test.go
// (197-02's own file, strengthened here to resolve through the live Cobra
// command tree rather than compare raw strings -- "aether resume" and
// "aether resume-colony" are two spellings of the exact same *cobra.Command,
// which a string comparison cannot see).

// ---------------------------------------------------------------------------
// Updating
// ---------------------------------------------------------------------------

// runSessionUpdate drives the real `aether update --force` over a project
// built by the real install/setup path, in the requested output mode.
func runSessionUpdate(t *testing.T, homeDir, repoDir string, jsonMode bool) (visual string, envelope map[string]interface{}) {
	t.Helper()
	t.Setenv("AETHER_HUB_DIR", "")
	t.Setenv("HOME", homeDir)
	if jsonMode {
		t.Setenv("AETHER_OUTPUT_MODE", "json")
	} else {
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
	}
	t.Setenv("AETHER_PLATFORM", "codex")
	saveGlobals(t)
	resetRootCmd(t)

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"update", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update failed: %v\noutput:\n%s", err, buf.String())
	}

	if jsonMode {
		var parsed map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("update output is not valid JSON: %v\n%s", err, buf.String())
		}
		inner, _ := parsed["result"].(map[string]interface{})
		if inner == nil {
			t.Fatalf("update output has no result envelope: %s", buf.String())
		}
		return "", inner
	}
	return stripANSI(buf.String()), nil
}

// TestUpdateEndsWithTheCard covers both outcomes named in the plan: a run
// that repaired a missing command copy, and a run that found nothing to do.
func TestUpdateEndsWithTheCard(t *testing.T) {
	cases := []struct {
		name   string
		damage bool
	}{
		{"when it found nothing to do", false},
		{"when it repaired a missing command copy", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			homeDir, repoDir := setUpAliasReconcileProject(t)
			if tc.damage {
				missing := filepath.Join(homeDir, ".claude", "commands", "ant-pause-colony.md")
				if err := os.Remove(missing); err != nil {
					t.Fatalf("remove %s: %v", missing, err)
				}
			}

			visual, _ := runSessionUpdate(t, homeDir, repoDir, false)

			if marker := spacedTitle("What Next"); !strings.Contains(visual, marker) {
				t.Fatalf("update does not print the shared closing card at all:\n%s", visual)
			}
			commands := allCommandsInCard(visual)
			if len(commands) == 0 {
				t.Fatalf("update's closing card recommends no command:\n%s", visual)
			}
			if strings.Count(visual, "close this chat") > 1 {
				t.Errorf("update tells the owner whether to close the chat more than once:\n%s", visual)
			}

			if tc.damage {
				if !strings.Contains(visual, "Restored missing command") {
					t.Errorf("a run that repaired a missing command copy should say so where the owner reads what changed:\n%s", visual)
				}
			} else if strings.Contains(visual, "Restored missing command") {
				t.Errorf("a run that found nothing to do should not claim a repair:\n%s", visual)
			}
		})
	}
}

// TestUpdateEnvelopeCarriesTheCardsFields (the machine-readable half for
// update's no-op outcome: emits the same stable keys every migrated
// command's envelope carries) is now a full subset of
// TestEveryLifecycleCommandEndsWithNextAction's "updating" subtest
// (cmd/lifecycle_next_action_coverage_test.go), which drives the identical
// no-damage fixture and checks command presence, the "aether " prefix,
// resolution against the live command tree, and the recommendation field --
// deleted here rather than kept as a duplicate (Phase 197 plan 07).
