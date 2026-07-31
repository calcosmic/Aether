package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func criterionEvidenceTestPhase() colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:              1,
		Name:            "Bound evidence",
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{"The guide exists"},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: "The guide exists", Artifacts: []string{"docs/guide.md"}},
		},
		Tasks: []colony.Task{
			{
				ID:              &taskID,
				Goal:            "Add focused coverage",
				Status:          colony.TaskCompleted,
				SuccessCriteria: []string{"Focused tests pass"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{Criterion: "Focused tests pass", Artifacts: []string{"feature_test.go"}, Checks: []string{"tests", "claims"}},
				},
			},
		},
	}
}

func setupCriterionEvidenceTest(t *testing.T, phase colony.Phase) (string, codexContinueManifest) {
	t.Helper()
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	for path, content := range map[string]string{
		"docs/guide.md":   "guide\n",
		"feature_test.go": "package fixture\n",
	} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("create artifact directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write artifact %s: %v", path, err)
		}
	}
	claims := codexBuildClaims{
		FilesCreated: []string{"docs/guide.md", "feature_test.go"},
		TaskClaims: []codexBuildTaskClaim{
			{TaskID: "1.1", FilesCreated: []string{"feature_test.go"}},
		},
		BuildPhase: 1,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	attachBuildArtifactEvidence(root, &claims)
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save claims: %v", err)
	}
	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:                   1,
			ClaimsPath:              ".aether/data/last-build-claims.json",
			CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
			EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
		},
	}
	return root, manifest
}

func passingCriterionSteps() []codexVerificationStep {
	return []codexVerificationStep{
		{Name: "tests", Command: "go test ./...", Passed: true, Summary: "tests passed"},
	}
}

func TestValidatePhaseCriterionEvidenceRequiresCompleteBindings(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	if err := validatePhaseCriterionEvidence(phase); err != nil {
		t.Fatalf("valid bindings rejected: %v", err)
	}
	phase.Tasks[0].EvidenceRequirements = nil
	if err := validatePhaseCriterionEvidence(phase); err == nil || !strings.Contains(err.Error(), "bind every criterion") {
		t.Fatalf("partial bindings error = %v, want complete-binding failure", err)
	}

	phase = criterionEvidenceTestPhase()
	phase.EvidenceRequirements[0].Artifacts = []string{".aether/data/verification.json"}
	if err := validatePhaseCriterionEvidence(phase); err == nil || !strings.Contains(err.Error(), "runtime state") {
		t.Fatalf("runtime-state artifact error = %v", err)
	}

	phase = colony.Phase{
		ID: 1,
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: "Orphaned criterion", Artifacts: []string{"output.txt"}},
		},
	}
	if err := validatePhaseCriterionEvidence(phase); err == nil || !strings.Contains(err.Error(), "has no success criteria") {
		t.Fatalf("orphaned requirement error = %v", err)
	}
}

func TestWorkerPlanCriterionEvidenceSurvivesCanonicalRoundTrip(t *testing.T) {
	artifact := codexWorkerPlanArtifact{
		Phases: []codexWorkerPlanPhase{
			{
				Name:            "Evidence phase",
				SuccessCriteria: []string{"Guide exists"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{Criterion: "Guide exists", Artifacts: []string{"./docs/guide.md"}, Checks: []string{"CLAIMS"}},
				},
				Tasks: []codexWorkerPlanTask{
					{
						Goal:            "Test the guide",
						SuccessCriteria: []string{"Focused tests pass"},
						EvidenceRequirements: []colony.CriterionEvidenceRequirement{
							{Criterion: "Focused tests pass", Artifacts: []string{"feature_test.go"}, Checks: []string{"Tests"}},
						},
					},
				},
			},
		},
	}
	phases := buildWorkerPlanPhases(artifact)
	if len(phases) != 1 || len(phases[0].Tasks) != 1 {
		t.Fatalf("canonical phases = %+v, want one phase and task", phases)
	}
	if err := validatePhaseCriterionEvidence(phases[0]); err != nil {
		t.Fatalf("canonical phase rejected evidence requirements: %v", err)
	}
	roundTrip := workerPlanArtifactFromPhases(codexPlanConfidence{}, nil, phases, codexPlanningLoop{})
	if len(roundTrip.Phases) != 1 || len(roundTrip.Phases[0].EvidenceRequirements) != 1 || len(roundTrip.Phases[0].Tasks) != 1 || len(roundTrip.Phases[0].Tasks[0].EvidenceRequirements) != 1 {
		t.Fatalf("round-trip artifact lost evidence requirements: %+v", roundTrip)
	}
	phaseRequirement := roundTrip.Phases[0].EvidenceRequirements[0]
	if strings.Join(phaseRequirement.Artifacts, ",") != "docs/guide.md" || strings.Join(phaseRequirement.Checks, ",") != "claims" {
		t.Fatalf("phase requirement = %+v, want normalized artifact and check", phaseRequirement)
	}
	taskRequirement := roundTrip.Phases[0].Tasks[0].EvidenceRequirements[0]
	if strings.Join(taskRequirement.Artifacts, ",") != "feature_test.go" || strings.Join(taskRequirement.Checks, ",") != "tests" {
		t.Fatalf("task requirement = %+v, want normalized artifact and check", taskRequirement)
	}
}

func TestEvaluatePhaseCriterionEvidencePassesFreshBoundEvidence(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)
	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	if !evaluation.Enforced || !evaluation.Passed || !evaluation.Deterministic {
		t.Fatalf("evaluation = %+v, want enforced deterministic pass", evaluation)
	}
	if len(evaluation.Criteria) != 2 || !evaluation.Criteria[0].Passed || !evaluation.Criteria[1].Passed {
		t.Fatalf("criteria = %+v, want two passes", evaluation.Criteria)
	}
}

func TestEvaluatePhaseCriterionEvidenceRejectsMissingAndStaleArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutate    func(t *testing.T, root string)
		wantIssue string
	}{
		{
			name: "missing",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				if err := os.Remove(filepath.Join(root, "docs", "guide.md")); err != nil {
					t.Fatalf("remove guide: %v", err)
				}
			},
			wantIssue: "cannot be verified",
		},
		{
			name: "changed after build",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("changed\n"), 0644); err != nil {
					t.Fatalf("change guide: %v", err)
				}
			},
			wantIssue: "changed after build evidence",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase := criterionEvidenceTestPhase()
			root, manifest := setupCriterionEvidenceTest(t, phase)
			tc.mutate(t, root)
			evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, passingCriterionSteps(), codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
			if evaluation.Passed || !strings.Contains(strings.Join(evaluation.BlockingIssues, "\n"), tc.wantIssue) {
				t.Fatalf("evaluation = %+v, want issue %q", evaluation, tc.wantIssue)
			}
		})
	}
}

func TestSnapshotBuildArtifactRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "evidence.txt"), []byte("outside\n"), 0644); err != nil {
		t.Fatalf("write outside evidence: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "external")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, err := snapshotBuildArtifact(root, "external/evidence.txt"); err == nil || !strings.Contains(err.Error(), "outside the repository") {
		t.Fatalf("symlink escape error = %v", err)
	}
}

func TestEvaluatePhaseCriterionEvidenceRequiresTaskOwnedClaimsAndExecutedChecks(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)
	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("load claims: %v", err)
	}
	claims.TaskClaims = nil
	if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
		t.Fatalf("save aggregate-only claims: %v", err)
	}
	evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, []codexVerificationStep{{Name: "tests", Skipped: true, Summary: "not resolved"}}, codexClaimVerification{Present: true, Passed: true}, codexWatcherVerification{})
	issues := strings.Join(evaluation.BlockingIssues, "\n")
	if evaluation.Passed || !strings.Contains(issues, "was not claimed by the current build for task 1.1") || !strings.Contains(issues, "required tests check was skipped") {
		t.Fatalf("evaluation issues = %q", issues)
	}
}

func TestRunCodexContinueVerificationAllowsBoundArtifactOnlyPhase(t *testing.T) {
	phase := colony.Phase{
		ID:              1,
		Name:            "Documentation",
		Mode:            colony.PhaseModeMaintenance,
		Status:          colony.PhaseInProgress,
		SuccessCriteria: []string{"Guide exists"},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: "Guide exists", Artifacts: []string{"docs/guide.md"}},
		},
	}
	root, manifest := setupCriterionEvidenceTest(t, phase)
	report, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, 0, time.Second, true)
	if !report.Passed || !report.CriteriaEnforced || !report.CriteriaPassed {
		t.Fatalf("report = %+v, want artifact-only verification pass", report)
	}
	if strings.Contains(strings.Join(report.BlockingIssues, "\n"), "no deterministic verification command") {
		t.Fatalf("artifact-only phase was incorrectly forced to run a shell command: %+v", report.BlockingIssues)
	}
}

// TestRunCodexContinueVerificationBlocksRequiredCheckWithNoResolvableCommand
// wires D-10 end to end through the real call site: a phase whose task
// criteria bind the "tests" check, run against a repo with no resolvable
// test command (setupCriterionEvidenceTest's temp root has no go.mod,
// package.json, or any other ecosystem marker). The "tests" step must report
// blocked, never passed, and the criterion gate must agree.
func TestRunCodexContinueVerificationBlocksRequiredCheckWithNoResolvableCommand(t *testing.T) {
	phase := criterionEvidenceTestPhase()
	root, manifest := setupCriterionEvidenceTest(t, phase)

	report, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, 0, time.Second, true)

	var testsStep *codexVerificationStep
	for i := range report.Steps {
		if report.Steps[i].Name == "tests" {
			testsStep = &report.Steps[i]
		}
	}
	if testsStep == nil {
		t.Fatalf("no tests step in report: %+v", report.Steps)
	}
	if testsStep.Passed {
		t.Fatalf("tests step reported passed with no resolvable command: %+v", testsStep)
	}
	if !testsStep.Blocked {
		t.Fatalf("tests step not marked blocked: %+v", testsStep)
	}
	if !strings.Contains(testsStep.Summary, "blocked") {
		t.Fatalf("tests step summary does not say blocked: %q", testsStep.Summary)
	}
	if report.CriteriaPassed {
		t.Fatalf("criteria passed despite blocked required tests check: %+v", report)
	}
}
