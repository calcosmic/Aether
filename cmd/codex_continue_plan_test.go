package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestContinuePlanOnlyEnforcesCriterionEvidence proves 163.1-09 Task 1: the
// verification snapshot shared by runCodexContinuePlanOnly and
// runCodexContinueFinalize now evaluates criterion evidence, so a phase
// carrying the bound-v1 policy with an artifact absent from all claim lists
// blocks on the plan-only path -- not only on the direct `aether continue`
// path (cmd/codex_continue.go:1596).
func TestContinuePlanOnlyEnforcesCriterionEvidence(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Plan-only enforces criterion evidence"
	now := time.Now().UTC()
	taskID := "1.1"

	// Must share main.go's package (withTestWorkspace writes "package main")
	// so `go build ./...` succeeds and the criterion-specific blocking issue
	// can be isolated from an unrelated build failure -- the plan's behavior
	// requires ChecksPassed/Passed to go false from criteria alone "even when
	// every verification step passed".
	untouchedPath := filepath.Join(root, "untouched_plan_only.go")
	if err := os.WriteFile(untouchedPath, []byte("package main\n\nvar untouchedPlanOnlyMarker = \"untouched\"\n"), 0644); err != nil {
		t.Fatalf("write untouched source: %v", err)
	}
	// The claimed (but unrelated) file must actually exist, or claim
	// verification itself fails with "does not exist" -- a different
	// blocking issue than the one this test isolates.
	unrelatedPath := filepath.Join(root, "unrelated_plan_only_test.go")
	if err := os.WriteFile(unrelatedPath, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write unrelated source: %v", err)
	}

	phase := colony.Phase{
		ID:     1,
		Name:   "Plan-only criterion enforcement",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{
				ID:              &taskID,
				Goal:            "Test-only task whose criterion binds an untouched source file",
				Status:          colony.TaskCompleted,
				SuccessCriteria: []string{"Untouched source still behaves"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{Criterion: "Untouched source still behaves", TaskID: taskID, Artifacts: []string{"untouched_plan_only.go"}},
				},
			},
		},
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan:           colony.Plan{Phases: []colony.Phase{phase}},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Plan-only criterion enforcement", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-801", Task: "Test-only task whose criterion binds an untouched source file", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-802", Task: "Independent verification before advancement", Status: "completed"},
	})

	// seedContinueBuildPacket doesn't set the bound-v1 criterion evidence
	// contract on the manifest; add it here (mirrors
	// TestReconcileTaskReadOnlyEvidenceSatisfiesCriterion, cmd/codex_continue_test.go:3998).
	manifestRel := filepath.ToSlash(filepath.Join("build", "phase-1", "manifest.json"))
	var buildManifest codexBuildManifest
	if err := store.LoadJSON(manifestRel, &buildManifest); err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	buildManifest.CriterionEvidencePolicy = criterionEvidencePolicyBoundV1
	buildManifest.EvidenceRequirements = flattenPhaseCriterionEvidenceRequirements(phase)
	if err := store.SaveJSON(manifestRel, buildManifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	// The task's claims legitimately never mention the untouched file --
	// only the unrelated file it actually touched, at both the flat and
	// per-task claim levels so general claim verification passes cleanly
	// and only the criterion-specific check fails.
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
		FilesModified: []string{"unrelated_plan_only_test.go"},
		TaskClaims:    []codexBuildTaskClaim{{TaskID: taskID, FilesModified: []string{"unrelated_plan_only_test.go"}}},
	}); err != nil {
		t.Fatalf("overwrite claims: %v", err)
	}

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	verification := plan.Verification
	if !verification.CriteriaEnforced {
		t.Fatalf("expected CriteriaEnforced true on the plan-only verification snapshot, got %+v", verification)
	}
	if verification.CriteriaPassed {
		t.Fatalf("expected CriteriaPassed false, got %+v", verification)
	}
	if verification.Passed {
		t.Fatalf("expected Passed false when an unclaimed criterion artifact exists, got %+v", verification)
	}
	if len(verification.Criteria) == 0 {
		t.Fatalf("expected Criteria to be populated on the plan-only verification snapshot, got %+v", verification)
	}
	if verification.CriteriaPolicy != criterionEvidencePolicyBoundV1 {
		t.Fatalf("expected CriteriaPolicy %q, got %q", criterionEvidencePolicyBoundV1, verification.CriteriaPolicy)
	}
	if !anyContains(verification.BlockingIssues, "was not claimed by the current build") {
		t.Fatalf("expected a blocking issue naming the unclaimed artifact, got %v", verification.BlockingIssues)
	}
}

// TestContinuePlanOnlyWithoutCriterionRequirementsLeavesBlockersUnchanged
// proves the other half of Task 1's behavior: a phase carrying no evidence
// requirements produces CriteriaEnforced:false and an unaffected blocker
// list, so the new evaluation call is a no-op for phases that never opted
// into the bound-v1 policy.
func TestContinuePlanOnlyWithoutCriterionRequirementsLeavesBlockersUnchanged(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	root, _, _, _ := setupIntermediateContinueState(t, "No criterion requirements plan-only")

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	verification := plan.Verification
	if verification.CriteriaEnforced {
		t.Fatalf("expected CriteriaEnforced false for a phase without evidence requirements, got %+v", verification)
	}
	if len(verification.BlockingIssues) != 0 {
		t.Fatalf("expected no blocking issues for a phase without evidence requirements, got %v", verification.BlockingIssues)
	}
}
