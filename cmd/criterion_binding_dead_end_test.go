package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// directoryBindingTestPhase builds a single-criterion, phase-level evidence
// binding to exactly one artifact path, for exercising the build-time
// directory refusal in isolation.
func directoryBindingTestPhase(artifactRel string) colony.Phase {
	criterion := fmt.Sprintf("No file under %s was modified", artifactRel)
	return colony.Phase{
		ID:              1,
		Name:            "Directory binding",
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{criterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: criterion, Artifacts: []string{artifactRel}},
		},
	}
}

// TestDirectoryArtifactIsRefusedByNameAtBuildTime is the field-report case
// (WINDOWS.md row 49): a phase binding an existing directory as a criterion
// artifact must be refused, by name, at the earliest point the phase is
// used -- never left to fail later as an artifact nobody could ever claim.
func TestDirectoryArtifactIsRefusedByNameAtBuildTime(t *testing.T) {
	t.Run("existing directory is refused", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
			t.Fatalf("create directory fixture: %v", err)
		}

		phase := directoryBindingTestPhase("server/app")
		err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
		if err == nil {
			t.Fatal("expected a directory binding to be refused, got nil")
		}
		var r refusal
		if !errors.As(err, &r) {
			t.Fatalf("expected a typed refusal, got: %v", err)
		}
		if r.ID != "criterion-artifact-is-a-directory" {
			t.Fatalf("unexpected refusal id: %q", r.ID)
		}
		if !strings.Contains(r.Why, "server/app") {
			t.Errorf("refusal does not name the offending path: %q", r.Why)
		}
		if !strings.Contains(r.Why, phase.SuccessCriteria[0]) {
			t.Errorf("refusal does not name the criterion text: %q", r.Why)
		}
	})

	t.Run("symlink to a directory is refused the same way", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		if err := os.MkdirAll(filepath.Join(root, "real-target"), 0755); err != nil {
			t.Fatalf("create symlink target: %v", err)
		}
		if err := os.Symlink(filepath.Join(root, "real-target"), filepath.Join(root, "shortcut")); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}

		phase := directoryBindingTestPhase("shortcut")
		err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
		if err == nil {
			t.Fatal("expected a symlink-to-a-directory binding to be refused, got nil")
		}
		var r refusal
		if !errors.As(err, &r) {
			t.Fatalf("expected a typed refusal, got: %v", err)
		}
		if !strings.Contains(r.Why, "shortcut") {
			t.Errorf("refusal does not name the offending shortcut path: %q", r.Why)
		}
	})

	t.Run("not-yet-existing path is never refused", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))

		phase := directoryBindingTestPhase("output/report-not-written-yet.md")
		if err := validatePhaseCriterionEvidenceAgainstDisk(root, phase); err != nil {
			t.Fatalf("a not-yet-existing artifact path must never be refused, got: %v", err)
		}
	})
}

// TestDirectoryArtifactRefusalNamesTheWayPast proves the refusal's next
// command is not just non-empty text, but a command genuinely registered on
// the live cobra tree -- derived by looking the command up (availableCommand,
// the same resolver the what-next card uses), never by comparing against a
// typed string that could silently rot.
func TestDirectoryArtifactRefusalNamesTheWayPast(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
		t.Fatalf("create directory fixture: %v", err)
	}

	phase := directoryBindingTestPhase("server/app")
	err := validatePhaseCriterionEvidenceAgainstDisk(root, phase)
	var r refusal
	if !errors.As(err, &r) {
		t.Fatalf("expected a typed refusal, got: %v", err)
	}
	if strings.TrimSpace(r.NextCommand) == "" {
		t.Fatal("expected a non-empty next command")
	}
	if r.ProtectsWork {
		t.Error("expected ProtectsWork=false: rewriting the binding loses no prior work")
	}
	if _, ok := availableCommand(r.NextCommand); !ok {
		t.Fatalf("next command %q does not resolve against the live cobra tree", r.NextCommand)
	}
}

// TestOrdinaryCriterionIsUnaffected proves the ordinary, regular-file
// evidence path -- both the new build-time disk check and the existing
// evaluatePhaseCriterionEvidence run-time path -- is byte-for-byte unchanged
// by this phase's additions.
func TestOrdinaryCriterionIsUnaffected(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)

	if err := validatePhaseCriterionEvidenceAgainstDisk(root, phase); err != nil {
		t.Fatalf("ordinary regular-file bindings must validate against disk, got: %v", err)
	}

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if !evaluation.Enforced || !evaluation.Passed || !evaluation.Deterministic {
		t.Fatalf("evaluation = %+v, want enforced deterministic pass (unaffected by this phase)", evaluation)
	}
	if len(evaluation.Criteria) != 2 || !evaluation.Criteria[0].Passed || !evaluation.Criteria[1].Passed {
		t.Fatalf("criteria = %+v, want two passes", evaluation.Criteria)
	}
	for _, c := range evaluation.Criteria {
		if c.State == criterionStateNeedsOwnerConfirmation {
			t.Fatalf("ordinary criterion was wrongly routed to owner confirmation: %+v", c)
		}
	}
}

// unsatisfiableCriterionTestPhase binds two criteria: one to a real,
// existing directory (no build could ever satisfy it -- the case this phase
// closes), and one to a regular file the build genuinely never produced (a
// real, ongoing failure that must keep failing the ordinary way).
func unsatisfiableCriterionTestPhase() (colony.Phase, string, string) {
	directoryCriterion := "No file under server/app was modified"
	missingFileCriterion := "The migration script exists"
	phase := colony.Phase{
		ID:              9,
		Name:            "Owner confirmation routing",
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{directoryCriterion, missingFileCriterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: directoryCriterion, Artifacts: []string{"server/app"}},
			{Criterion: missingFileCriterion, Artifacts: []string{"reports/missing-migration.sql"}},
		},
	}
	return phase, directoryCriterion, missingFileCriterion
}

// TestUnsatisfiableCriterionRoutesToTheOwnerAnswer is the field-report case
// (WINDOWS.md row 52): an in-progress phase already stuck on a directory
// binding is routed to the existing owner-confirmation exit instead of
// blocking forever, while a criterion that simply failed (a file the build
// never produced) keeps failing the ordinary way.
func TestUnsatisfiableCriterionRoutesToTheOwnerAnswer(t *testing.T) {
	saveGlobals(t)
	phase, directoryCriterion, missingFileCriterion := unsatisfiableCriterionTestPhase()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	if err := os.MkdirAll(filepath.Join(root, "server", "app"), 0755); err != nil {
		t.Fatalf("create directory fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "server", "app", "main.go"), []byte("package app\n"), 0644); err != nil {
		t.Fatalf("write directory contents: %v", err)
	}
	// reports/missing-migration.sql is deliberately never created: the build
	// genuinely did not produce it.

	claims := codexBuildClaims{BuildPhase: phase.ID}
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save claims: %v", err)
	}
	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-9/manifest.json",
		Data: codexBuildManifest{
			Phase:                   phase.ID,
			ClaimsPath:              ".aether/data/last-build-claims.json",
			CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
			EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
		},
	}

	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if len(evaluation.Criteria) != 2 {
		t.Fatalf("expected two criteria, got %+v", evaluation.Criteria)
	}
	var directoryResult, missingResult codexCriterionVerification
	for _, c := range evaluation.Criteria {
		switch c.Criterion {
		case directoryCriterion:
			directoryResult = c
		case missingFileCriterion:
			missingResult = c
		default:
			t.Fatalf("unexpected criterion in evaluation: %q", c.Criterion)
		}
	}

	// The directory-bound criterion: routed to the owner, phase still
	// advances, and the screen says why.
	if directoryResult.State != criterionStateNeedsOwnerConfirmation {
		t.Fatalf("directory-bound criterion State = %q, want %q: %+v", directoryResult.State, criterionStateNeedsOwnerConfirmation, directoryResult)
	}
	if !directoryResult.Passed {
		t.Fatalf("directory-bound criterion must still Pass so the phase can advance: %+v", directoryResult)
	}
	if !strings.Contains(directoryResult.Summary, "server/app") {
		t.Errorf("summary does not name the folder: %q", directoryResult.Summary)
	}
	haystack := strings.Join(append(append([]string{}, directoryResult.Evidence...), directoryResult.BlockingIssues...), "\n")
	if strings.Contains(haystack, "was not claimed by the current build") {
		t.Errorf("directory-bound criterion still produced the misleading unclaimed-artifact message: %q", haystack)
	}
	if !strings.Contains(haystack, "server/app") {
		t.Errorf("directory-bound criterion evidence does not name the folder: %q", haystack)
	}

	outstanding := outstandingOwnerConfirmations(phase.ID, evaluation.Criteria)
	if len(outstanding) != 1 || outstanding[0].Criterion != directoryCriterion {
		t.Fatalf("outstandingOwnerConfirmations = %+v, want exactly the directory-bound criterion", outstanding)
	}

	// The negative case: a genuinely missing file keeps failing the
	// ordinary way, and is never routed to the owner.
	if missingResult.Passed {
		t.Fatalf("criterion for a genuinely missing file must still fail: %+v", missingResult)
	}
	if missingResult.State == criterionStateNeedsOwnerConfirmation {
		t.Fatalf("a genuinely missing file must never be routed to owner confirmation: %+v", missingResult)
	}
	if !strings.Contains(strings.Join(missingResult.BlockingIssues, "\n"), "reports/missing-migration.sql") {
		t.Errorf("missing-file criterion does not report the missing artifact: %+v", missingResult.BlockingIssues)
	}

	// Answering it through the real aether decision-answer path resolves it,
	// exactly like an ordinary owner-confirmation criterion.
	question := ownerConfirmationQuestionText(phase.ID, directoryResult.TaskID, directoryCriterion)
	if ownerConfirmationAnswered(phase.ID, directoryResult.TaskID, directoryCriterion) {
		t.Fatal("test premise broken: criterion already answered before recording anything")
	}
	if _, err := recordDecisionAnswer(question, "confirmed", phase.ID, "criterion-binding-dead-end-test"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}
	if !ownerConfirmationAnswered(phase.ID, directoryResult.TaskID, directoryCriterion) {
		t.Fatal("expected ownerConfirmationAnswered to resolve after recording the owner's answer")
	}
	if got := outstandingOwnerConfirmations(phase.ID, evaluation.Criteria); len(got) != 0 {
		t.Fatalf("expected no outstanding confirmations after the owner answered, got %+v", got)
	}
}

// TestSkipPhaseForceNamesTheTruthfulAlternative proves WINDOWS.md row 52's
// second half: `aether skip-phase --force` is no longer the ONLY exit from a
// phase stuck on one criterion. Its screen now names the owner-answer route
// first, and still states plainly that skipping records the phase as
// abandoned -- while what the command actually records is unchanged.
func TestSkipPhaseForceNamesTheTruthfulAlternative(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)

	goal := "Ship the directory binding fix"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Only phase",
					Status: colony.PhaseReady,
					Tasks: []colony.Task{
						{Goal: "Do the work", Status: colony.TaskPending},
					},
				},
			},
		},
	})

	result, updated, skipped, nextPhase, err := runSkipPhase(1, true, "abandoning for test")
	if err != nil {
		t.Fatalf("runSkipPhase: %v", err)
	}

	// What the command records is unchanged from before this phase.
	if result["reason"] != "abandoning for test" {
		t.Errorf("result[reason] = %v, want the passed reason", result["reason"])
	}
	if result["skipped"] != true {
		t.Errorf("result[skipped] = %v, want true", result["skipped"])
	}
	if result["phase"] != 1 {
		t.Errorf("result[phase] = %v, want 1", result["phase"])
	}
	if updated.State != colony.StateCOMPLETED {
		t.Errorf("updated.State = %v, want COMPLETED (single-phase plan, last phase skipped)", updated.State)
	}

	visual := renderSkipPhaseVisual(updated, skipped, nextPhase, result)
	if !strings.Contains(visual, "aether decision-answer") {
		t.Errorf("skip-phase screen does not name the owner-answer route: %q", visual)
	}
	if !strings.Contains(visual, "abandoned") {
		t.Errorf("skip-phase screen does not plainly say the phase is recorded as abandoned: %q", visual)
	}

	routeIdx := strings.Index(visual, "aether decision-answer")
	confirmationIdx := strings.Index(visual, "Phase was force-skipped")
	if routeIdx < 0 || confirmationIdx < 0 || routeIdx > confirmationIdx {
		t.Errorf("owner-answer route must be named FIRST, before the force-skip confirmation: %q", visual)
	}
}
