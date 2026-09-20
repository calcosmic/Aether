package cmd

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// --- Fixtures ---

// handbackStillFailingFixtureTestsCommand is a deterministic tests command
// that always fails with real, non-empty captured output -- unlike
// canonicalFloorFixFixture's plain "false" (which produces an empty
// step.Output, so its own diagnosis always falls back to the generic "the
// X check is still failing" text). This fixture exists so the diagnosis
// assertion below can genuinely check that real captured output reached
// the owner's screen, not just the fallback.
const handbackStillFailingFixtureTestsCommand = `sh -c 'echo "assertion failed: widget total mismatch"; exit 1'`

// handbackStillFailingFixture builds an isolated fixture root whose tests
// check always fails with the same real output, on every run -- the repair
// wave (a no-op FakeInvoker dispatch) never fixes it, so the check-fix
// attempt's outcome is deterministically "still_failing".
func handbackStillFailingFixture(t *testing.T) testBuildStartFixture {
	t.Helper()
	return commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt: time.Now().UTC(), ExecutionOwner: "test-owner",
		MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true",
				"- tests: "+handbackStillFailingFixtureTestsCommand)
		},
	})
}

// handbackFixSucceedsFixture builds an isolated fixture root whose tests
// check fails exactly once and passes on the second run, driven by a
// counter file the tests command itself increments -- the deterministic
// floor's own two real invocations (once before the check-fix attempt,
// once as part of it) are what flip the result, not a hand-typed outcome.
// This is a genuine "the fix worked" production path, not a fabricated one:
// nothing here manufactures a repairHandback or a checkFixAttemptRecord
// directly.
func handbackFixSucceedsFixture(t *testing.T) testBuildStartFixture {
	t.Helper()
	const testsCmd = `sh -c 'c=$(cat .repair-fix-counter 2>/dev/null || echo 0); c=$((c+1)); echo $c > .repair-fix-counter; test $c -ge 2'`
	return commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt: time.Now().UTC(), ExecutionOwner: "test-owner",
		MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: "+testsCmd)
		},
	})
}

// handbackNoEligibleFixFixture builds an isolated fixture root whose checks
// all pass from the very first run -- planCheckFixAttempt is never eligible
// (floor.ChecksPassed is already true), so no check-fix attempt runs at all
// and no handback can exist.
func handbackNoEligibleFixFixture(t *testing.T) testBuildStartFixture {
	t.Helper()
	return commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt: time.Now().UTC(), ExecutionOwner: "test-owner",
		MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: true")
		},
	})
}

// --- Task 3: TestFailedRepairHandsBackAllFourParts ---

// TestFailedRepairHandsBackAllFourParts drives a real, genuinely failing
// check through the production check-repair path with no reviewer
// dispatched, proves the resulting handback's four parts are each derived
// the way the runtime derives them (never a typed literal), and proves the
// rendered blocked screen carries all four. It then proves the two negative
// cases -- a fix attempt that succeeds, and a check with no eligible fix
// attempt -- render no handback section at all, and leave the rest of the
// screen unchanged.
func TestFailedRepairHandsBackAllFourParts(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("a still-failing repair hands back all four parts", func(t *testing.T) {
		saveGlobals(t)
		newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
		fixture := handbackStillFailingFixture(t)
		root, phase, state := fixture.Root, fixture.Phase, fixture.State
		manifest := codexContinueManifest{}

		verification, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, true)

		if verification.CheckFixAttempt == nil || verification.CheckFixAttempt.Outcome != "still_failing" {
			t.Fatalf("expected a still-failing check fix attempt, got %+v", verification.CheckFixAttempt)
		}
		if verification.RepairHandback == nil {
			t.Fatalf("expected a repair handback on a still-failing check fix, got nil")
		}
		handback := *verification.RepairHandback

		// Derive every expected value the way the runtime derives it --
		// never a typed literal (CLAUDE.md's fixture-value rule).
		wantCheckpointID := repairCheckpointIdentity(phase.ID, verification.CheckFixAttempt.Check)
		failingStep := firstFailingVerificationStep(verification.Steps)
		if failingStep == nil {
			t.Fatalf("expected a failing verification step on the re-run floor, got steps=%+v", verification.Steps)
		}
		wantExcerpts, _ := compactFailureExcerpts(failingStep.Output, failingStep.Summary)
		if len(wantExcerpts) == 0 {
			t.Fatalf("expected at least one excerpt from the failing step's own output %q", failingStep.Output)
		}
		_, attempt, ok := loadLatestBuildAttempt(phase.ID)
		if !ok {
			t.Fatalf("expected a build attempt to be loadable for phase %d", phase.ID)
		}
		wantAction, actionErr := recommendedActionForWorkOutcome(colony.WorkOutcomeBlocker, attempt)
		if actionErr != nil {
			t.Fatalf("recommendedActionForWorkOutcome: %v", actionErr)
		}

		t.Run("diagnosis quotes text that genuinely appeared in the failing step's own output", func(t *testing.T) {
			if strings.TrimSpace(handback.Diagnosis) == "" {
				t.Fatalf("Diagnosis is empty")
			}
			matched := false
			for _, excerpt := range wantExcerpts {
				if strings.Contains(handback.Diagnosis, excerpt) {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("Diagnosis %q does not quote any excerpt %v genuinely captured from the failing step's own output", handback.Diagnosis, wantExcerpts)
			}
			if !strings.Contains(handback.Diagnosis, "widget total mismatch") {
				t.Fatalf("Diagnosis %q does not carry the real failure text the fixture's own command produced", handback.Diagnosis)
			}
		})

		t.Run("attempted part names the check", func(t *testing.T) {
			if strings.TrimSpace(handback.AttemptedAndWhy) == "" {
				t.Fatalf("AttemptedAndWhy is empty")
			}
			if !strings.Contains(handback.AttemptedAndWhy, verification.CheckFixAttempt.Check) {
				t.Fatalf("AttemptedAndWhy %q does not name the check %q", handback.AttemptedAndWhy, verification.CheckFixAttempt.Check)
			}
		})

		t.Run("restored part names the real checkpoint identity", func(t *testing.T) {
			if strings.TrimSpace(handback.RestoredPosition) == "" {
				t.Fatalf("RestoredPosition is empty")
			}
			if !strings.Contains(handback.RestoredPosition, wantCheckpointID) {
				t.Fatalf("RestoredPosition %q does not name the real checkpoint identity %q", handback.RestoredPosition, wantCheckpointID)
			}
		})

		t.Run("action part carries a runnable command and its reason", func(t *testing.T) {
			if strings.TrimSpace(handback.OwnerAction.Command) == "" {
				t.Fatalf("OwnerAction.Command is empty")
			}
			if handback.OwnerAction.Command != wantAction.Command {
				t.Fatalf("OwnerAction.Command = %q, want %q (recommendedActionForWorkOutcome for the same blocker verdict)", handback.OwnerAction.Command, wantAction.Command)
			}
			if strings.TrimSpace(handback.OwnerAction.Reason) == "" {
				t.Fatalf("OwnerAction.Reason is empty")
			}
		})

		t.Run("the rendered blocked screen carries all four parts", func(t *testing.T) {
			result := map[string]interface{}{"verification": verification}
			rendered := renderContinueBlockedVisual(state, phase, result, colony.VerificationDepthStandard)
			for _, want := range []string{
				handback.Diagnosis,
				handback.AttemptedAndWhy,
				wantCheckpointID,
				wantAction.Command,
			} {
				if !strings.Contains(rendered, want) {
					t.Errorf("rendered blocked screen missing %q\n--- rendered ---\n%s", want, rendered)
				}
			}
		})
	})

	t.Run("a fix attempt that succeeds renders no handback section", func(t *testing.T) {
		saveGlobals(t)
		newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
		fixture := handbackFixSucceedsFixture(t)
		root, phase, state := fixture.Root, fixture.Phase, fixture.State
		// The fixture's own phase carries a bound criterion evidence
		// requirement (from buildStartTransaction200's canonical fixture),
		// which independently blocks ChecksPassed unless the manifest
		// carries the matching bound policy and requirements -- derived
		// here the same way runCodexContinuePlanOnly derives it
		// (cmd/codex_build.go:2900-2901), never hand-typed, so the tests
		// step passing on the second run is what actually flips
		// ChecksPassed true, not a manufactured manifest shape.
		manifest := codexContinueManifest{
			Present: true,
			Data: codexBuildManifest{
				CriterionEvidencePolicy: phaseCriterionEvidencePolicy(phase),
				EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
			},
		}

		verification, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, true)

		if verification.CheckFixAttempt == nil || verification.CheckFixAttempt.Outcome != "fixed" {
			t.Fatalf("expected the check-fix attempt to have fixed the check, got %+v", verification.CheckFixAttempt)
		}
		if verification.RepairHandback != nil {
			t.Fatalf("expected no handback for a fix attempt that succeeded, got %+v", verification.RepairHandback)
		}
		assertNoHandbackSectionAndScreenUnchanged(t, state, phase, verification)
	})

	t.Run("no eligible fix attempt renders no handback section", func(t *testing.T) {
		saveGlobals(t)
		newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
		fixture := handbackNoEligibleFixFixture(t)
		root, phase, state := fixture.Root, fixture.Phase, fixture.State
		manifest := codexContinueManifest{}

		verification, _ := runCodexContinueVerification(context.Background(), root, state, phase, manifest, time.Second, 5*time.Second, true)

		if verification.CheckFixAttempt != nil {
			t.Fatalf("expected no check-fix attempt to have run, got %+v", verification.CheckFixAttempt)
		}
		if verification.RepairHandback != nil {
			t.Fatalf("expected no handback when no fix attempt was eligible, got %+v", verification.RepairHandback)
		}
		assertNoHandbackSectionAndScreenUnchanged(t, state, phase, verification)
	})
}

// assertNoHandbackSectionAndScreenUnchanged proves the blocked screen
// carries no handback section's own label, and is byte-identical to the
// same screen built from a copy of the report with RepairHandback forced
// nil -- an identity that only holds when the field really was nil to begin
// with, so this also catches a regression that starts leaking a handback
// only in the map (JSON-round-tripped) shape but not the typed shape, or
// vice versa.
func assertNoHandbackSectionAndScreenUnchanged(t *testing.T, state colony.ColonyState, phase colony.Phase, verification codexContinueVerificationReport) {
	t.Helper()

	asIs := map[string]interface{}{"verification": verification}
	renderedAsIs := renderContinueBlockedVisual(state, phase, asIs, colony.VerificationDepthStandard)

	cleared := verification
	cleared.RepairHandback = nil
	clearedResult := map[string]interface{}{"verification": cleared}
	renderedCleared := renderContinueBlockedVisual(state, phase, clearedResult, colony.VerificationDepthStandard)

	if renderedAsIs != renderedCleared {
		t.Fatalf("rendering diverged from the no-handback baseline even though RepairHandback is nil:\n%s", firstDiffLine(renderedAsIs, renderedCleared))
	}
	if strings.Contains(renderedAsIs, "What happened with the automatic fix") {
		t.Fatalf("rendered blocked screen unexpectedly contains a repair handback section:\n%s", renderedAsIs)
	}

	// Dual-type parity: the JSON-round-tripped completion-file shape must
	// agree with the in-process typed shape.
	asMap := roundTripToMap(t, asIs)
	renderedFromMap := renderContinueBlockedVisual(state, phase, asMap, colony.VerificationDepthStandard)
	if renderedFromMap != renderedAsIs {
		t.Fatalf("JSON-round-tripped render diverged from the typed render:\n%s", firstDiffLine(renderedFromMap, renderedAsIs))
	}
}

// --- Task 3: TestFailedRepairHandbackHasProductionCallers ---

// TestFailedRepairHandbackHasProductionCallers is the structural guard:
// buildFailedRepairHandback and renderFailedRepairHandback each have at
// least one direct caller among the package's non-test functions, derived
// from the real call graph (continueDecisionPackageFuncs /
// continueDecisionDirectCallers, cmd/codex_verify_advance_test.go) rather
// than a hand-maintained list -- and a synthetic function with no caller
// demonstrates, by name and position, that the guard is actually capable of
// failing.
func TestFailedRepairHandbackHasProductionCallers(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("both functions have at least one production caller", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		for _, target := range []string{"buildFailedRepairHandback", "renderFailedRepairHandback"} {
			if _, ok := funcs[target]; !ok {
				t.Fatalf("%s is not declared in the cmd package", target)
			}
			callers := continueDecisionDirectCallers(funcs, target)
			if len(callers) == 0 {
				t.Errorf("%s has no direct caller among the package's non-test functions -- the handback would be assembled or rendered by nothing", target)
			}
		}
	})

	t.Run("guard detects a synthetic function with no caller, by name and position", func(t *testing.T) {
		fset := token.NewFileSet()
		src := `package cmd

func fakeOrphanedRepairHandbackAssembler() {
	_ = 1
}
`
		extra, err := parser.ParseFile(fset, "fixture_orphaned_handback_assembler.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic orphaned-function fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		const offender = "fakeOrphanedRepairHandbackAssembler"
		callers := continueDecisionDirectCallers(funcs, offender)
		if len(callers) != 0 {
			t.Fatalf("expected the synthetic orphaned function to have zero callers, got %v -- the guard cannot demonstrate failure if it finds a caller here", callers)
		}
		fn, ok := funcs[offender]
		if !ok {
			t.Fatalf("synthetic fixture function %s was not indexed by continueDecisionPackageFuncs", offender)
		}
		pos := fset.Position(fn.Pos())
		if pos.Filename == "" || pos.Line == 0 {
			t.Fatalf("synthetic offender %s carries no file/line position: %+v -- a real guard failure must be able to name where the missing wiring is", offender, pos)
		}
	})
}
