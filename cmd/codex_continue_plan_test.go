package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
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

// TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection proves 189-02
// D-11: continue's external (wrapper-mediated) plan-only manifest carries a
// colony-wide context capsule and, when a signal is active, a pheromone
// section -- both resolved ONCE per manifest, mirroring how
// codexBuildManifest.ContextCapsule already works for build (cmd/codex_build.go).
// Before this plan's fix, codexContinuePlanManifest had neither field at all.
func TestContinuePlanOnlyManifestCarriesCapsuleAndPheromoneSection(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	root, dataDir, _, _ := setupIntermediateContinueState(t, "Capsule and pheromone on continue plan-only")

	distinctiveSignalText := "distinctive-continue-plan-only-pheromone-marker-3f9a"
	strength := 1.0
	writeTestPheromones(t, dataDir, colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_continue_plan_only_capsule_pheromone_test",
				Type:      "FOCUS",
				Priority:  "normal",
				Source:    "test",
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				Active:    true,
				Strength:  &strength,
				Content:   json.RawMessage(`{"text":"` + distinctiveSignalText + `"}`),
			},
		},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}

	if strings.TrimSpace(plan.ContextCapsule) == "" {
		t.Errorf("expected continue_manifest.context_capsule to be non-empty for a colony with an active goal/state, got %q", plan.ContextCapsule)
	}
	if !strings.Contains(plan.PheromoneSection, distinctiveSignalText) {
		t.Errorf("expected continue_manifest.pheromone_section to contain the seeded signal's text %q, got %q", distinctiveSignalText, plan.PheromoneSection)
	}

	// codexContinueExternalDispatch must NOT have grown a per-dispatch
	// capsule/pheromone field of its own -- both are manifest-level,
	// resolved once, not per-dispatch (D-11). This is a compile-time
	// invariant checked via reflection so a future per-dispatch addition
	// fails this test rather than silently duplicating the manifest-level
	// fields N times in the JSON payload.
	dispatchType := reflect.TypeOf(codexContinueExternalDispatch{})
	for i := 0; i < dispatchType.NumField(); i++ {
		name := dispatchType.Field(i).Name
		if name == "ContextCapsule" || name == "PheromoneSection" {
			t.Errorf("codexContinueExternalDispatch must not carry its own %s field -- capsule/pheromone are manifest-level (D-11), never per-dispatch", name)
		}
	}
}

// TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath
// proves 189-02 D-06: every wrapper-external continue dispatch's Brief (the
// watcher's and every reviewer's) states the exact handoff/return schema
// codex.ValidateWorkerHandoff enforces -- but renderCodexContinueWatcherBrief
// and renderCodexContinueReviewBrief's OWN raw output, the same functions
// continue's native-Codex dispatch path
// (plannedContinueReviewDispatches/plannedContinueWatcherDispatch) calls
// directly as TaskBrief, does NOT contain it. The native path already states
// the schema via a separate channel (AssembleHostedPrompt +
// renderResponseContract); adding it inside either render function would
// duplicate it there.
func TestContinueExternalDispatchBriefsStateHandoffSchemaOnceNotOnNativePath(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	root, _, _, taskID := setupIntermediateContinueState(t, "Handoff schema stated once on external continue dispatches")

	phase := colony.Phase{
		ID:     1,
		Name:   "Handoff schema stated once on external continue dispatches",
		Status: colony.PhaseInProgress,
		Tasks:  []colony.Task{{ID: &taskID, Goal: "Complete intermediate work", Status: colony.TaskInProgress}},
	}
	manifest := loadCodexContinueManifest(phase.ID)
	verification := codexContinueVerificationReport{Phase: phase.ID, ChecksPassed: true, Passed: true}
	assessment := codexContinueAssessment{Phase: phase.ID, Passed: true}

	// skipWatchers=false and no explicit Queen caste proposal (nil) drives
	// plannedExternalContinueDispatches through the same deterministic
	// queenOrchestrate path runCodexContinuePlanOnly itself uses -- a real,
	// not hand-picked, set of dispatches for a standard-depth continue run.
	dispatches := plannedExternalContinueDispatches(root, phase, manifest, verification, assessment, 0, colony.VerificationDepthStandard, false, nil, "")
	if len(dispatches) == 0 {
		t.Fatalf("expected at least one planned external continue dispatch (the watcher is always required)")
	}

	const schemaNeedle = "changed_files"
	if !strings.Contains(codex.HandoffFieldsSummary, schemaNeedle) {
		t.Fatalf("test setup error: %q must be a substring of codex.HandoffFieldsSummary -- fix the test, not the schema", schemaNeedle)
	}

	sawWatcher := false
	sawReviewer := false
	for _, dispatch := range dispatches {
		if !strings.Contains(dispatch.Brief, schemaNeedle) {
			t.Errorf("dispatch %s (caste %s)'s Brief is missing handoff-schema substring %q -- a wrapper-spawned continue worker was never told the finalizer's schema:\n%s", dispatch.Name, dispatch.Caste, schemaNeedle, dispatch.Brief)
		}
		if dispatch.Caste == "watcher" {
			sawWatcher = true
		} else {
			sawReviewer = true
		}
	}
	if !sawWatcher {
		t.Fatalf("expected the watcher dispatch to be present (skipWatchers=false); got castes %v", dispatchCastes(dispatches))
	}
	if !sawReviewer {
		t.Fatalf("expected at least one reviewer dispatch at standard depth on a phase with real source tasks; got castes %v", dispatchCastes(dispatches))
	}

	// Proof of no duplication: renderCodexContinueWatcherBrief's and
	// renderCodexContinueReviewBrief's own raw output -- what continue's
	// native-Codex dispatch path feeds as TaskBrief -- must NOT contain the
	// schema note plannedExternalContinueDispatches appends only for the
	// wrapper-external path.
	rawWatcherBrief := renderCodexContinueWatcherBrief(root, phase, manifest, verification.Steps, verification.Claims, verification.Watcher, 0)
	if strings.Contains(rawWatcherBrief, schemaNeedle) {
		t.Errorf("renderCodexContinueWatcherBrief's raw output already contains %q -- a native-Codex watcher would see the handoff schema twice:\n%s", schemaNeedle, rawWatcherBrief)
	}

	reviewSpec, ok := continueReviewSpecForCaste("probe")
	if !ok {
		t.Fatalf("test setup error: no review spec registered for caste %q", "probe")
	}
	rawReviewBrief := renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, reviewSpec)
	if strings.Contains(rawReviewBrief, schemaNeedle) {
		t.Errorf("renderCodexContinueReviewBrief's raw output already contains %q -- a native-Codex reviewer would see the handoff schema twice:\n%s", schemaNeedle, rawReviewBrief)
	}
}

func dispatchCastes(dispatches []codexContinueExternalDispatch) []string {
	castes := make([]string, 0, len(dispatches))
	for _, d := range dispatches {
		castes = append(castes, d.Caste)
	}
	return castes
}
