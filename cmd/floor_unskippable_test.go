package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/pflag"
)

// FLOOR-01 (193-03): no reviewer-skip path may leave the deterministic
// floor -- build, types, lint, tests, claimed-files-exist, criterion
// evidence -- unrun. This file walks every skip path that exists today
// (spec §7: "QUICK means zero reviewer agents, not zero deterministic
// validation") and guards the owner-facing flag text that used to claim
// otherwise.

// floorUnskippableFixture builds a temp repo with all four verification
// commands resolving to real, deterministic shell commands, a phase with one
// success criterion bound to the "tests" check (so criterion evidence is
// meaningfully exercised, not vacuously empty), and a manifest whose builder
// claims match a real file in the repo. testsCmd controls whether the tests
// command passes ("true") or fails ("false"), shared by
// TestDeterministicChecksCannotBeSkipped / TestExecutedCheckSetIsInvariantAcrossDepthAndProposal
// (testsCmd="true") and TestFailingCheckStillBlocksOnEverySkipPath
// (testsCmd="false"). Modeled on deterministicFloorFixtures in
// cmd/deterministic_floor_test.go.
func floorUnskippableFixture(t *testing.T, testsCmd string) (string, colony.Phase, codexContinueManifest) {
	t.Helper()
	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", fmt.Sprintf("- tests: %s", testsCmd))
	if err := os.WriteFile(filepath.Join(root, "changed.txt"), []byte("ok\n"), 0644); err != nil {
		t.Fatalf("write changed.txt: %v", err)
	}
	phase := colony.Phase{
		ID:              1,
		Name:            "Update the README",
		Description:     "Documentation-only change, no code",
		Mode:            colony.PhaseModeMaintenance,
		SuccessCriteria: []string{"all tests pass"},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: "all tests pass", Checks: []string{"tests"}},
		},
	}
	claims := codexBuildClaims{
		BuildPhase:    1,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		FilesModified: []string{"changed.txt"},
	}
	if err := s.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save claims: %v", err)
	}
	manifest := codexContinueManifest{
		Present: true,
		Data: codexBuildManifest{
			Phase:                   1,
			ClaimsPath:              ".aether/data/last-build-claims.json",
			DispatchMode:            "real",
			CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
			EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Update the README", Status: "completed"},
			},
		},
	}
	return root, phase, manifest
}

// floorSkipPathRow is one row of the skip-path table: a real, distinct
// condition that causes continue to send zero (or fewer) reviewer workers.
type floorSkipPathRow struct {
	name string
	// configure applies the row's real production condition (an invoker
	// override, a manifest shape, a state counter) and returns the
	// skipWatchers value both lane functions are called with.
	configure func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool)
	// verifyConditionIsReal exercises the actual production mechanism this
	// row claims to walk (never a stub of continueWatcherDecision's return
	// value), so a row can never pass on a condition that was silently
	// never engaged.
	verifyConditionIsReal func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest)
}

// floorSkipPathRows returns the table FLOOR-01's tests walk: the
// skip-watchers flag, light depth, heavy depth, an explicit verification
// depth of light/standard/heavy, a caste proposal naming no reviewer, an
// unavailable worker provider, the host-boundary skip, and the
// consecutive-failure auto-skip -- ten named rows, satisfying "at least
// seven distinct subtests."
//
// skipWatchers is true for every row whose mechanism lies outside
// continueWatcherDecision (light/heavy/explicit-depth/caste-proposal):
// those parameters are never even passed to runCodexContinueVerification or
// runCodexContinueVerificationSnapshot, so there is nothing to force through
// those functions -- the structural absence of the parameter IS the proof.
// Each such row's verifyConditionIsReal instead drives the real,
// independent production function (resolveVerificationDepth,
// queenContinueReviewSpecsWithJudgement) that the row's condition actually
// governs, so the row is not vacuous.
//
// For the provider-unavailable, host-boundary, and consecutive-failure
// rows, skipWatchers is false and the real condition (the worker invoker's
// availability, the manifest's dispatch mode, the phase's watcher failure
// counter) is set up directly, so continueWatcherDecision is walked for
// real rather than short-circuited by skipWatchers itself.
func floorSkipPathRows() []floorSkipPathRow {
	return []floorSkipPathRow{
		{
			name: "the skip-watchers flag",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				buildWatcher := evaluateContinueWatcherVerification(manifest)
				dispatch, decision := continueWatcherDecision(state, phase, manifest, buildWatcher, true)
				if dispatch {
					t.Fatalf("--skip-watchers did not skip the reviewer dispatch decision")
				}
				if decision.Status != "skipped" {
					t.Fatalf("--skip-watchers decision status = %q, want skipped", decision.Status)
				}
			},
		},
		{
			name: "light depth (--light flag)",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if depth := resolveVerificationDepth(phase, 1, true, false, ""); depth != colony.VerificationDepthLight {
					t.Fatalf("--light flag resolved to %q, want light", depth)
				}
			},
		},
		{
			name: "heavy depth (--heavy flag)",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if depth := resolveVerificationDepth(phase, 1, false, true, ""); depth != colony.VerificationDepthHeavy {
					t.Fatalf("--heavy flag resolved to %q, want heavy", depth)
				}
			},
		},
		{
			name: "explicit verification depth: light",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if depth := resolveVerificationDepth(phase, 1, false, false, "light"); depth != colony.VerificationDepthLight {
					t.Fatalf("--verification-depth light resolved to %q, want light", depth)
				}
			},
		},
		{
			name: "explicit verification depth: standard",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if depth := resolveVerificationDepth(phase, 1, false, false, "standard"); depth != colony.VerificationDepthStandard {
					t.Fatalf("--verification-depth standard resolved to %q, want standard", depth)
				}
			},
		},
		{
			name: "explicit verification depth: heavy",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if depth := resolveVerificationDepth(phase, 1, false, false, "heavy"); depth != colony.VerificationDepthHeavy {
					t.Fatalf("--verification-depth heavy resolved to %q, want heavy", depth)
				}
			},
		},
		{
			name: "a caste proposal naming no reviewer",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, true
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				// Light depth is used here (rather than standard) so this
				// row's premise -- "a proposal naming no reviewer produces no
				// review specs" -- holds independent of whether this test
				// binary's own workspace reads as containing testable code:
				// at standard depth isAlwaysRequired (cmd/caste_relevance.go)
				// forces "probe" back in whenever queenPhaseProducesTestableCode
				// is true, which is Phase 194's territory (193-02-SUMMARY.md),
				// not this row's. At light depth the only caste isAlwaysRequired
				// ever restores is "watcher", which queenContinueReviewSpecsWithJudgement
				// always excludes from review specs regardless.
				specs := queenContinueReviewSpecsWithJudgement(phase, colony.VerificationDepthLight, []string{"builder"}, "team asked for a builder only", nil)
				if len(specs) != 0 {
					t.Fatalf("a caste proposal naming no reviewer still produced review specs: %+v", specs)
				}
			},
		},
		{
			name: "an unavailable worker provider",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				newCodexWorkerInvoker = func() codex.WorkerInvoker { return &continueUnavailableInvoker{} }
				return state, manifest, false
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				buildWatcher := evaluateContinueWatcherVerification(manifest)
				dispatch, decision := continueWatcherDecision(state, phase, manifest, buildWatcher, false)
				if dispatch {
					t.Fatalf("an unavailable worker provider did not auto-skip the reviewer dispatch decision")
				}
				if decision.Worker != "auto-skip" {
					t.Fatalf("unavailable-provider decision worker = %q, want auto-skip", decision.Worker)
				}
			},
		},
		{
			name: "the host-boundary skip",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				manifest.Data.DispatchMode = "external-task"
				return state, manifest, false
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if _, ok := continueWatcherHostBoundarySkipSummary(manifest); !ok {
					t.Fatalf("host-boundary manifest shape did not trip continueWatcherHostBoundarySkipSummary")
				}
				buildWatcher := evaluateContinueWatcherVerification(manifest)
				dispatch, _ := continueWatcherDecision(state, phase, manifest, buildWatcher, false)
				if dispatch {
					t.Fatalf("the host-boundary manifest shape did not skip the reviewer dispatch decision")
				}
			},
		},
		{
			name: "the consecutive-failure auto-skip",
			configure: func(t *testing.T, state colony.ColonyState, manifest codexContinueManifest) (colony.ColonyState, codexContinueManifest, bool) {
				return state, manifest, false
			},
			verifyConditionIsReal: func(t *testing.T, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest) {
				if count := getWatcherFailureCount(state, phase.ID); count < defaultWatcherFailureThreshold {
					t.Fatalf("consecutive-failure fixture only recorded %d failures, want >= %d", count, defaultWatcherFailureThreshold)
				}
				buildWatcher := evaluateContinueWatcherVerification(manifest)
				dispatch, _ := continueWatcherDecision(state, phase, manifest, buildWatcher, false)
				if dispatch {
					t.Fatalf("the consecutive-failure threshold did not auto-skip the reviewer dispatch decision")
				}
			},
		},
	}
}

// floorSkipPathState seeds the colony state each row starts from -- the
// consecutive-failure row is the only one that needs a phase record with a
// non-zero WatcherFailureCount, since getWatcherFailureCount reads it from
// state.Plan.Phases, not from the phase value passed alongside it.
func floorSkipPathState(rowName string, phase colony.Phase) colony.ColonyState {
	if rowName != "the consecutive-failure auto-skip" {
		return colony.ColonyState{}
	}
	failing := phase
	failing.WatcherFailureCount = defaultWatcherFailureThreshold
	return colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{failing}}}
}

// assertFloorFullyExecuted is the core FLOOR-01 assertion, identical for
// every row and both lanes: build, types, lint and tests all carried a
// command, were not skipped, and produced an outcome; the claimed-files-exist
// check ran; and criterion evidence was evaluated. A row that leaves any of
// those unrun fails by name.
func assertFloorFullyExecuted(t *testing.T, rowName, laneName string, report codexContinueVerificationReport) {
	t.Helper()
	wantSteps := []string{"build", "types", "lint", "tests"}
	got := map[string]codexVerificationStep{}
	for _, step := range report.Steps {
		got[step.Name] = step
	}
	for _, name := range wantSteps {
		step, ok := got[name]
		if !ok {
			t.Fatalf("%s / %s lane: %s check is missing from the deterministic floor entirely", rowName, laneName, name)
		}
		if step.Skipped {
			t.Fatalf("%s / %s lane: %s check was skipped (%s) instead of running", rowName, laneName, name, step.Summary)
		}
		if strings.TrimSpace(step.Command) == "" {
			t.Fatalf("%s / %s lane: %s check carried no command", rowName, laneName, name)
		}
		if strings.TrimSpace(step.Summary) == "" {
			t.Fatalf("%s / %s lane: %s check produced no outcome", rowName, laneName, name)
		}
	}
	if !report.Claims.Present {
		t.Fatalf("%s / %s lane: the claimed-files-exist check did not run (claims.Present=false)", rowName, laneName)
	}
	if len(report.Criteria) == 0 {
		t.Fatalf("%s / %s lane: criterion evidence was never evaluated", rowName, laneName)
	}
}

// executedCheckNames returns the sorted names of every step that carried a
// command and was not skipped -- the "executed check set" the invariant test
// compares across rows.
func executedCheckNames(report codexContinueVerificationReport) []string {
	names := []string{}
	for _, step := range report.Steps {
		if !step.Skipped {
			names = append(names, step.Name)
		}
	}
	return sortedCopy(names)
}

// TestDeterministicChecksCannotBeSkipped walks every reviewer-skip path that
// exists today -- the skip-watchers flag, light and heavy depth, an explicit
// verification depth, a caste proposal naming no reviewer, an unavailable
// worker provider, the host-boundary skip, and the consecutive-failure
// auto-skip -- and fails by name if any one of them leaves a deterministic
// check unrun. This is FLOOR-01 as a runnable command, not a claim.
func TestDeterministicChecksCannotBeSkipped(t *testing.T) {
	for _, row := range floorSkipPathRows() {
		t.Run(row.name, func(t *testing.T) {
			saveGlobals(t)
			newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
			root, phase, manifest := floorUnskippableFixture(t, "true")
			state := floorSkipPathState(row.name, phase)
			state, manifest, skipWatchers := row.configure(t, state, manifest)
			row.verifyConditionIsReal(t, state, phase, manifest)

			inProcess, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, skipWatchers)
			snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, skipWatchers)

			assertFloorFullyExecuted(t, row.name, "in-process", inProcess)
			assertFloorFullyExecuted(t, row.name, "snapshot", snapshot)
		})
	}
}

// TestExecutedCheckSetIsInvariantAcrossDepthAndProposal is the proportion
// assertion: the multiset of executed check names is byte-identical across
// every depth flag, review policy, and caste proposal combination in the
// table, on both lanes independently. It fails if a future flag reduces what
// is checked, whatever that flag is called.
func TestExecutedCheckSetIsInvariantAcrossDepthAndProposal(t *testing.T) {
	var referenceInProcess, referenceSnapshot []string
	for _, row := range floorSkipPathRows() {
		t.Run(row.name, func(t *testing.T) {
			saveGlobals(t)
			newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
			root, phase, manifest := floorUnskippableFixture(t, "true")
			state := floorSkipPathState(row.name, phase)
			state, manifest, skipWatchers := row.configure(t, state, manifest)

			inProcess, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, skipWatchers)
			snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, skipWatchers)

			gotInProcess := executedCheckNames(inProcess)
			gotSnapshot := executedCheckNames(snapshot)
			if referenceInProcess == nil {
				referenceInProcess = gotInProcess
				referenceSnapshot = gotSnapshot
				return
			}
			if strings.Join(gotInProcess, ",") != strings.Join(referenceInProcess, ",") {
				t.Fatalf("%s: in-process executed check set = %v, want %v (byte-identical to the first row)", row.name, gotInProcess, referenceInProcess)
			}
			if strings.Join(gotSnapshot, ",") != strings.Join(referenceSnapshot, ",") {
				t.Fatalf("%s: snapshot executed check set = %v, want %v (byte-identical to the first row)", row.name, gotSnapshot, referenceSnapshot)
			}
		})
	}
}

// TestFailingCheckStillBlocksOnEverySkipPath re-runs the same table with the
// tests command exiting non-zero: every row must report the failure and must
// not pass verification. A floor that runs but is then ignored is the same
// defect as one that does not run.
func TestFailingCheckStillBlocksOnEverySkipPath(t *testing.T) {
	for _, row := range floorSkipPathRows() {
		t.Run(row.name, func(t *testing.T) {
			saveGlobals(t)
			newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
			root, phase, manifest := floorUnskippableFixture(t, "false")
			state := floorSkipPathState(row.name, phase)
			state, manifest, skipWatchers := row.configure(t, state, manifest)

			inProcess, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, skipWatchers)
			snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, skipWatchers)

			for _, lane := range []struct {
				name   string
				report codexContinueVerificationReport
			}{{"in-process", inProcess}, {"snapshot", snapshot}} {
				if lane.report.ChecksPassed {
					t.Fatalf("%s / %s lane: a failing tests check still reported ChecksPassed=true", row.name, lane.name)
				}
				if len(lane.report.BlockingIssues) == 0 {
					t.Fatalf("%s / %s lane: a failing tests check produced no blocking issues", row.name, lane.name)
				}
			}
		})
	}
}

// continueFlagCheckSkipPairingAllowance records flags whose help text
// legitimately pairs a skip/disable word with a check word, with the reason
// it is not a false claim. An exception here is a recorded decision, not a
// silent pass -- keep this list short (Task 2, 193-03).
var continueFlagCheckSkipPairingAllowance = map[string]string{
	"skip-watchers": "the corrected text pairs \"skip\" with the check words on purpose, to say the checks are NOT skipped -- the exact clarification D-01 requires",
}

// TestNoContinueFlagClaimsToSkipAChecked walks continueCmd's registered
// flags and fails if any flag's usage string pairs a skip/disable word with
// a check word (build, type, lint, test, verification, check), unless the
// flag is named in continueFlagCheckSkipPairingAllowance with a reason. This
// asserts on the pairing, not on any single flag name, so a future flag
// inherits the guard automatically.
func TestNoContinueFlagClaimsToSkipAChecked(t *testing.T) {
	skipWords := []string{"skip", "disable"}
	checkWords := []string{"build", "type", "lint", "test", "verification", "check"}

	continueCmd.Flags().VisitAll(func(f *pflag.Flag) {
		usage := strings.ToLower(f.Usage)
		hasSkip := false
		for _, w := range skipWords {
			if strings.Contains(usage, w) {
				hasSkip = true
				break
			}
		}
		if !hasSkip {
			return
		}
		var checkWord string
		for _, w := range checkWords {
			if strings.Contains(usage, w) {
				checkWord = w
				break
			}
		}
		if checkWord == "" {
			return
		}
		if reason, ok := continueFlagCheckSkipPairingAllowance[f.Name]; ok {
			if strings.TrimSpace(reason) == "" {
				t.Errorf("--%s is allow-listed with an empty reason; every allowance needs a one-line reason", f.Name)
			}
			return
		}
		t.Errorf("--%s help text pairs a skip/disable word with the check word %q, implying a check can be turned off: %q", f.Name, checkWord, f.Usage)
	})
}
