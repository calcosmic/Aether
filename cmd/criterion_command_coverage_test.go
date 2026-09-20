package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// commandCoverageTestPhase builds a minimal bound-evidence phase around a
// single phase-level criterion, check-bound only (no artifacts) -- these
// fixtures only exercise setupCriterionEvidenceTest's manifest/claims/store
// wiring, never its file-writing, since no criterion here binds an
// artifact.
func commandCoverageTestPhase(criterion string, checks []string) colony.Phase {
	return colony.Phase{
		ID:              1,
		Name:            "Command coverage",
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{criterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: criterion, Checks: checks},
		},
	}
}

// fourWayPassingSteps mirrors the field report's own scenario (finding 7):
// build, types, lint and the general test command all resolved and passed,
// none of them the specific named suite the criterion's own wording called
// out.
func fourWayPassingSteps() []codexVerificationStep {
	return []codexVerificationStep{
		{Name: "build", Command: "go build ./cmd/aether", Passed: true, Summary: "build passed"},
		{Name: "types", Command: "go vet ./cmd/...", Passed: true, Summary: "types passed"},
		{Name: "lint", Command: "golangci-lint run", Passed: true, Summary: "lint passed"},
		{Name: "tests", Command: "go test ./...", Passed: true, Summary: "tests passed"},
	}
}

// TestCriterionNamingAnUnrunCommandIsRefused reproduces, as a permanent
// regression fixture, the real failure recorded in
// .planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md
// (finding 7): "Phase 3 task 3.10's criterion includes '...and the operator
// broker suite all pass'. The runtime marked it passed from build, types,
// lint and Jest alone ... npm run test:operator is not among the repo's
// verification commands".
//
// The field report only ever quotes the criterion's tail -- the ellipsis is
// the report's own truncation, not part of the original sentence, and the
// full original wording (including wherever it actually named the command)
// was never printed. This fixture reconstructs a complete sentence that
// carries the report's exact quoted tail phrase ("the operator broker
// suite ... all pass") plus the backtick-quoted command RESEARCH.md
// identifies as the gap ("a named `test:operator` command was never
// configured"), so the historical failure mode is exercised precisely: all
// four generic checks (build/types/lint/tests) resolve and pass, but the
// specific suite the criterion's own wording names never ran.
func TestCriterionNamingAnUnrunCommandIsRefused(t *testing.T) {
	criterion := "Build succeeds, types check, lint passes, and the operator broker suite (`npm run test:operator`) all pass."
	phase := commandCoverageTestPhase(criterion, []string{"tests"})
	root, manifest := setupCriterionEvidenceTest(t, phase)

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, fourWayPassingSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})

	if evaluation.Passed {
		t.Fatalf("evaluation wrongly credited a criterion naming a never-run command (the exact field-report failure): %+v", evaluation)
	}
	issues := strings.Join(evaluation.BlockingIssues, "\n")
	if !strings.Contains(issues, "npm run test:operator") {
		t.Fatalf("blocking issues do not name the missing command: %q", issues)
	}
	if !strings.Contains(issues, criterion) {
		t.Fatalf("blocking issues do not name the criterion itself: %q", issues)
	}
}

// TestCriterionNamingARunCommandIsCredited proves the new refusal never
// over-fires: a criterion naming a command that genuinely is among the
// commands this phase ran is credited exactly as it always has been.
func TestCriterionNamingARunCommandIsCredited(t *testing.T) {
	criterion := "The full suite (`go test ./...`) passes."
	phase := commandCoverageTestPhase(criterion, []string{"tests"})
	root, manifest := setupCriterionEvidenceTest(t, phase)

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, fourWayPassingSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})

	if !evaluation.Passed {
		t.Fatalf("evaluation refused a criterion whose named command genuinely ran: %+v", evaluation)
	}
	if len(evaluation.Criteria) != 1 || len(evaluation.Criteria[0].BlockingIssues) != 0 {
		t.Fatalf("criteria = %+v, want zero blocking issues", evaluation.Criteria)
	}
}

// TestCriterionNamingNoCommandKeepsExistingBehaviour covers both halves of
// the plan's own behaviour spec for a criterion that names no command at
// all: the pre-existing "no artifact and no check" refusal is untouched,
// and an ordinary criterion bound to a check still passes.
func TestCriterionNamingNoCommandKeepsExistingBehaviour(t *testing.T) {
	t.Run("no artifact and no check still refused exactly as before", func(t *testing.T) {
		phase := commandCoverageTestPhase("It works", nil)
		evaluation := evaluatePhaseCriterionEvidence("", phase, codexContinueManifest{}, nil, codexClaimVerification{}, codexWatcherVerification{})
		if evaluation.Passed {
			t.Fatalf("evaluation passed for a criterion with neither artifact nor check: %+v", evaluation)
		}
		issues := strings.Join(evaluation.BlockingIssues, "\n")
		if !strings.Contains(issues, "has no artifact or verification check") {
			t.Fatalf("blocking issues = %q, want the pre-existing no-evidence refusal text unchanged", issues)
		}
	})

	t.Run("criterion with a check and no named command still passes", func(t *testing.T) {
		phase := commandCoverageTestPhase("Tests pass.", []string{"tests"})
		root, manifest := setupCriterionEvidenceTest(t, phase)
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, fourWayPassingSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
		if !evaluation.Passed {
			t.Fatalf("evaluation refused a criterion that names no command at all: %+v", evaluation)
		}
	})
}

// TestCriterionCommandExtractionIgnoresProse proves a criterion whose prose
// merely contains words that look like a command -- "build", "test",
// "suite" -- without being written as one (no backticks, no bare runner
// invocation) is never treated as naming a command, so it is evaluated
// exactly as before: the ordinary bound check decides it, not the new
// command-coverage step.
func TestCriterionCommandExtractionIgnoresProse(t *testing.T) {
	for _, criterion := range []string{
		"The build succeeds and the tests all pass.",
		"We manually test the login flow end to end.",
		"The operator broker suite and the general test plan both pass.",
		"The file `README.md` documents test coverage.",
		"Run `echo done` to confirm completion.",
	} {
		t.Run(criterion, func(t *testing.T) {
			phase := commandCoverageTestPhase(criterion, []string{"tests"})
			root, manifest := setupCriterionEvidenceTest(t, phase)
			evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, fourWayPassingSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
			if !evaluation.Passed {
				t.Fatalf("prose criterion %q was wrongly treated as naming a command: %+v", criterion, evaluation)
			}
		})
	}
}

// TestCriterionCommandCoverageIsReachedFromBothCheckLanes asserts the same
// refusal fires on both lanes that credit a criterion: the in-process lane
// (runCodexContinueVerification, cmd/codex_continue.go) and the
// wrapper/external lane (runCodexContinueVerificationSnapshot,
// cmd/codex_continue_plan.go). Both call the shared runDeterministicFloor
// body (cmd/deterministic_floor.go), so this fails if a future change ever
// re-introduces a second, un-shared crediting path -- a guarantee that
// holds only on the path nobody uses is worth nothing (CLAUDE.md, "How
// much proof a change needs").
func TestCriterionCommandCoverageIsReachedFromBothCheckLanes(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: true")

	criterion := "Build succeeds, types check, lint passes, and the operator broker suite (`npm run test:operator`) all pass."
	phase := colony.Phase{
		ID:              1,
		Name:            "Command coverage parity",
		SuccessCriteria: []string{criterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: criterion, Checks: []string{"tests"}},
		},
	}
	manifest := codexContinueManifest{
		Present: true,
		Data: codexBuildManifest{
			Phase:                   1,
			CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
			EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
		},
	}

	inProcess, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, true)

	if inProcess.CriteriaPassed {
		t.Fatalf("in-process lane credited a criterion naming a never-run command: %+v", inProcess)
	}
	if snapshot.CriteriaPassed {
		t.Fatalf("wrapper/external lane credited a criterion naming a never-run command: %+v", snapshot)
	}
	if inProcess.CriteriaPassed != snapshot.CriteriaPassed {
		t.Fatalf("CriteriaPassed disagree: in-process=%v snapshot=%v", inProcess.CriteriaPassed, snapshot.CriteriaPassed)
	}

	inProcessIssues := strings.Join(inProcess.BlockingIssues, "\n")
	snapshotIssues := strings.Join(snapshot.BlockingIssues, "\n")
	if !strings.Contains(inProcessIssues, "npm run test:operator") {
		t.Fatalf("in-process blocking issues do not name the missing command: %q", inProcessIssues)
	}
	if !strings.Contains(snapshotIssues, "npm run test:operator") {
		t.Fatalf("snapshot blocking issues do not name the missing command: %q", snapshotIssues)
	}
}
