package cmd

import (
	"context"
	"encoding/json"
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

// TestCriterionReadOnlyEvidence proves the task-scoped read-only bypass (D-01,
// D-02): a criterion bound to an artifact the task legitimately did not
// modify can be satisfied by hash-verified evidence marked ReadOnly and
// scoped to that task's ID, integrity checks are unaffected, and the bypass
// never crosses task boundaries or applies with an empty task scope.
func TestCriterionReadOnlyEvidence(t *testing.T) {
	taskID := "1.1"
	phase := colony.Phase{
		ID:     1,
		Name:   "Read-only evidence",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{
				ID:              &taskID,
				Goal:            "Test-only task",
				Status:          colony.TaskCompleted,
				SuccessCriteria: []string{"Untouched source still behaves"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{Criterion: "Untouched source still behaves", TaskID: taskID, Artifacts: []string{"untouched.go"}},
				},
			},
		},
	}

	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	artifactPath := filepath.Join(root, "untouched.go")
	original := []byte("package untouched\n")
	if err := os.WriteFile(artifactPath, original, 0644); err != nil {
		t.Fatalf("write artifact: %v", err)
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

	saveClaimsWithReadOnly := func(t *testing.T, readOnlyTaskID string) {
		t.Helper()
		evidence, err := snapshotBuildArtifact(root, "untouched.go")
		if err != nil {
			t.Fatalf("snapshot artifact: %v", err)
		}
		evidence.ReadOnly = true
		evidence.ReadOnlyTaskID = readOnlyTaskID
		claims := codexBuildClaims{
			BuildPhase:       1,
			Timestamp:        time.Now().UTC().Format(time.RFC3339),
			ArtifactEvidence: []codexBuildArtifactEvidence{evidence},
		}
		if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
			t.Fatalf("save claims: %v", err)
		}
	}

	t.Run("matching task scoped read-only evidence satisfies the criterion", func(t *testing.T) {
		saveClaimsWithReadOnly(t, taskID)
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, nil, codexClaimVerification{}, codexWatcherVerification{})
		if !evaluation.Passed {
			t.Fatalf("evaluation = %+v, want pass via read-only evidence", evaluation)
		}
		if len(evaluation.Criteria) != 1 || len(evaluation.Criteria[0].BlockingIssues) != 0 {
			t.Fatalf("criteria = %+v, want zero blocking issues", evaluation.Criteria)
		}
		joined := strings.Join(evaluation.Criteria[0].Evidence, "\n")
		if !strings.Contains(joined, "(read-only)") {
			t.Fatalf("evidence = %q, want read-only marker", joined)
		}
	})

	t.Run("tampering after recording still blocks", func(t *testing.T) {
		saveClaimsWithReadOnly(t, taskID)
		if err := os.WriteFile(artifactPath, []byte("package untouched\n\n// changed\n"), 0644); err != nil {
			t.Fatalf("mutate artifact: %v", err)
		}
		defer func() {
			if err := os.WriteFile(artifactPath, original, 0644); err != nil {
				t.Fatalf("restore artifact: %v", err)
			}
		}()
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, nil, codexClaimVerification{}, codexWatcherVerification{})
		issues := strings.Join(evaluation.BlockingIssues, "\n")
		if evaluation.Passed || !strings.Contains(issues, "changed after build evidence was recorded") {
			t.Fatalf("evaluation = %+v, want tamper block", evaluation)
		}
	})

	t.Run("cross-task read-only evidence does not satisfy (D-02)", func(t *testing.T) {
		saveClaimsWithReadOnly(t, "2.1")
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, nil, codexClaimVerification{}, codexWatcherVerification{})
		issues := strings.Join(evaluation.BlockingIssues, "\n")
		if evaluation.Passed || !strings.Contains(issues, "was not claimed by the current build for task 1.1") {
			t.Fatalf("evaluation = %+v, want cross-task block", evaluation)
		}
	})

	t.Run("empty read-only task id satisfies nothing", func(t *testing.T) {
		saveClaimsWithReadOnly(t, "")
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, nil, codexClaimVerification{}, codexWatcherVerification{})
		issues := strings.Join(evaluation.BlockingIssues, "\n")
		if evaluation.Passed || !strings.Contains(issues, "was not claimed by the current build for task 1.1") {
			t.Fatalf("evaluation = %+v, want block for empty read-only task id", evaluation)
		}
	})

	t.Run("unclaimed non-read-only artifact still blocks unchanged", func(t *testing.T) {
		claims := codexBuildClaims{BuildPhase: 1, Timestamp: time.Now().UTC().Format(time.RFC3339)}
		if err := store.SaveJSON("last-build-claims.json", claims); err != nil {
			t.Fatalf("save claims: %v", err)
		}
		evaluation := evaluatePhaseCriterionEvidence(root, phase, manifest, nil, codexClaimVerification{}, codexWatcherVerification{})
		issues := strings.Join(evaluation.BlockingIssues, "\n")
		if evaluation.Passed || !strings.Contains(issues, "was not claimed by the current build for task 1.1") {
			t.Fatalf("evaluation = %+v, want unchanged blocking behavior", evaluation)
		}
	})
}

// TestArtifactEvidenceReadOnlyNotSettableByWorker locks the self-attestation
// guard (Pitfall 3, T-163.1-26): attachBuildArtifactEvidence replaces
// claims.ArtifactEvidence wholesale from the three claim lists, so a
// worker-submitted completion packet that sets read_only:true directly on an
// artifact_evidence entry must never survive into stored claims.
func TestArtifactEvidenceReadOnlyNotSettableByWorker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "handled.go"), []byte("package handled\n"), 0644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	packetJSON := `{
		"files_created": ["handled.go"],
		"artifact_evidence": [
			{"path":"x","sha256":"deadbeef","size":1,"read_only":true,"read_only_task_id":"1.1"}
		],
		"build_phase": 1
	}`
	var claims codexBuildClaims
	if err := json.Unmarshal([]byte(packetJSON), &claims); err != nil {
		t.Fatalf("unmarshal worker packet: %v", err)
	}
	if len(claims.ArtifactEvidence) != 1 || !claims.ArtifactEvidence[0].ReadOnly {
		t.Fatalf("test setup: expected worker-submitted read_only to decode true, got %+v", claims.ArtifactEvidence)
	}

	attachBuildArtifactEvidence(root, &claims)

	for _, entry := range claims.ArtifactEvidence {
		if entry.ReadOnly {
			t.Fatalf("worker-submitted read_only evidence survived attachBuildArtifactEvidence: %+v", claims.ArtifactEvidence)
		}
	}
}
