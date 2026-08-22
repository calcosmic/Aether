package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// setupContinueCriterionEvidenceFinalizeFixture builds the shared fixture for
// 163.1-09 Task 3's end-to-end proof: a phase whose single task is completed,
// whose criterion binds a real file in the test workspace the task never
// modified, and whose last-build-claims.json claims only an unrelated file
// for that task. Modelled on TestReconcileTaskReadOnlyEvidenceSatisfiesCriterion
// (cmd/codex_continue_test.go:3998) and
// TestContinueFinalizeAcceptsEmptyDispatchesWhenSkipWatchersLight
// (cmd/codex_continue_test.go:409).
func setupContinueCriterionEvidenceFinalizeFixture(t *testing.T) (root string, taskID string, artifactPath string) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Continue criterion evidence finalize e2e"
	now := time.Now().UTC()
	taskID = "1.1"

	// Must share main.go's package (withTestWorkspace writes "package main")
	// so `go build ./...` succeeds and the criterion-specific blocking issue
	// can be isolated from an unrelated build failure.
	artifactPath = filepath.Join(root, "untouched_finalize_source.go")
	if err := os.WriteFile(artifactPath, []byte("package main\n\nvar untouchedFinalizeMarker = \"untouched\"\n"), 0644); err != nil {
		t.Fatalf("write untouched source: %v", err)
	}
	unrelatedPath := filepath.Join(root, "unrelated_finalize_test.go")
	if err := os.WriteFile(unrelatedPath, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write unrelated source: %v", err)
	}

	phase := colony.Phase{
		ID:     1,
		Name:   "Continue criterion evidence finalize",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{
				ID:              &taskID,
				Goal:            "Test-only task whose criterion binds an untouched source file",
				Status:          colony.TaskCompleted,
				SuccessCriteria: []string{"Untouched source still behaves"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{Criterion: "Untouched source still behaves", TaskID: taskID, Artifacts: []string{"untouched_finalize_source.go"}},
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

	seedContinueBuildPacket(t, dataDir, 1, "Continue criterion evidence finalize", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-901", Task: "Test-only task whose criterion binds an untouched source file", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-902", Task: "Independent verification before advancement", Status: "completed"},
	})

	// seedContinueBuildPacket doesn't set the bound-v1 criterion evidence
	// contract on the manifest; add it here.
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

	// The task's claims legitimately never mention the untouched file -- only
	// the unrelated file it actually touched, at both the flat and per-task
	// claim levels so general claim verification passes cleanly and only the
	// criterion-specific check fails.
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
		BuildPhase:    1,
		Timestamp:     now.Format(time.RFC3339),
		FilesModified: []string{"unrelated_finalize_test.go"},
		TaskClaims:    []codexBuildTaskClaim{{TaskID: taskID, FilesModified: []string{"unrelated_finalize_test.go"}}},
	}); err != nil {
		t.Fatalf("overwrite claims: %v", err)
	}

	return root, taskID, artifactPath
}

// TestContinuePlanOnlyCriterionEvidenceRequiresReadOnlyArtifact proves the
// plan-only half of 163.1-09 Task 3: a criterion bound to a source file the
// task never modified blocks `aether continue --plan-only --reconcile-task X`
// (criteria_passed:false, blocking issue naming the unclaimed artifact), and
// adding `--read-only-artifact X:<file>` flips criteria_passed to true and
// removes that specific blocking issue -- the escape hatch now works on the
// external-review path, not only on the direct `aether continue` path.
func TestContinuePlanOnlyCriterionEvidenceRequiresReadOnlyArtifact(t *testing.T) {
	root, taskID, _ := setupContinueCriterionEvidenceFinalizeFixture(t)

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
		ReconcileTaskIDs: []string{taskID},
		LightFlag:        true,
		SkipWatchers:     true,
	})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly (no read-only-artifact) returned error: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	if plan.Verification.CriteriaPassed {
		t.Fatalf("expected criteria_passed:false without --read-only-artifact, got %+v", plan.Verification)
	}
	if !anyContains(plan.Verification.BlockingIssues, "was not claimed by the current build") {
		t.Fatalf("expected blocking issue naming the unclaimed artifact, got %v", plan.Verification.BlockingIssues)
	}

	planResult2, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
		ReconcileTaskIDs:  []string{taskID},
		ReadOnlyArtifacts: []string{taskID + ":untouched_finalize_source.go"},
		LightFlag:         true,
		SkipWatchers:      true,
	})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly (with read-only-artifact) returned error: %v", err)
	}
	plan2, ok := planResult2["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult2["continue_manifest"])
	}
	if !plan2.Verification.CriteriaPassed {
		t.Fatalf("expected criteria_passed:true with --read-only-artifact, got %+v", plan2.Verification)
	}
	if anyContains(plan2.Verification.BlockingIssues, "was not claimed by the current build") {
		t.Fatalf("criterion-specific blocking issue survived --read-only-artifact: %v", plan2.Verification.BlockingIssues)
	}
}

// TestContinuePlanOnlyReadOnlyArtifactRefusesClaimedArtifact proves the
// CR-163.1-01 guard end-to-end on the plan-only path: naming an artifact the
// build actually claimed in --read-only-artifact fails loudly instead of
// silently re-hashing its current (possibly tampered) content over the
// build-time evidence in last-build-claims.json. Without this refusal, a
// claimed-then-tampered artifact's tamper block could be laundered away by a
// flag whose whole premise (D-01) is artifacts the build did NOT claim.
func TestContinuePlanOnlyReadOnlyArtifactRefusesClaimedArtifact(t *testing.T) {
	root, taskID, _ := setupContinueCriterionEvidenceFinalizeFixture(t)

	// unrelated_finalize_test.go IS claimed by task 1.1 in the fixture's
	// claims (both flat and per-task). Tamper with it after the "build",
	// then try to launder the tamper through the escape hatch.
	if err := os.WriteFile(filepath.Join(root, "unrelated_finalize_test.go"), []byte("package main\n\n// tampered after build\n"), 0644); err != nil {
		t.Fatalf("tamper claimed artifact: %v", err)
	}

	_, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
		ReconcileTaskIDs:  []string{taskID},
		ReadOnlyArtifacts: []string{taskID + ":unrelated_finalize_test.go"},
		LightFlag:         true,
		SkipWatchers:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "claimed") {
		t.Fatalf("expected plan-only to refuse read-only evidence for a claimed artifact, got err=%v", err)
	}
}

// TestContinueFinalizeReadOnlyArtifactEvidenceCriteriaPassAndDetectsTamper is
// the full plan-only -> continue-finalize proof for 163.1-09 Task 3. It
// exercises three mandatory paths: the happy path (recorded evidence flips
// the bound criteria to passing without manufacturing a false files_modified
// claim -- the phase does NOT advance on this route, and the expected blocked
// state is asserted explicitly below), the tamper path (editing the artifact
// between plan-only and finalize re-blocks), and the unrecorded-spec path (a
// plan manifest naming a read-only artifact that was never actually recorded
// fails loudly instead of silently proceeding -- T-163.1-45). A suite
// containing only the happy path would pass even if --read-only-artifact
// disabled criterion checking on this route entirely.
//
// Scope note (193-04, FLOOR-03): this function's "allowed blocking classes"
// were pinned before commit 9208ff48 (H-04) made a reconciled+verified task
// advance in the general case; the happy-path subtest below already reflects
// that fix and is unaffected by 193-04's further change to
// continueTasksSupportAdvancement (dropping the claimsSatisfied requirement
// for a reconciled task -- this fixture's general claim check already passes
// via files_modified, so that further change is a no-op here). The scenario
// 193-04 actually changes -- a reconciled task with NO general claims at all,
// on both continue lanes -- is covered by TestFinalizeCountsReconcileTaskAsEvidence
// and TestReconcileIsNotABypass (cmd/floor_reviewer_free_gate_test.go), not
// by this file, which stays scoped to the --read-only-artifact criterion-
// evidence path specifically.
func TestContinueFinalizeReadOnlyArtifactEvidenceCriteriaPassAndDetectsTamper(t *testing.T) {
	t.Run("criteria pass and claims are preserved when evidence is recorded", func(t *testing.T) {
		root, taskID, _ := setupContinueCriterionEvidenceFinalizeFixture(t)

		planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
			ReconcileTaskIDs:  []string{taskID},
			ReadOnlyArtifacts: []string{taskID + ":untouched_finalize_source.go"},
			LightFlag:         true,
			SkipWatchers:      true,
		})
		if err != nil {
			t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
		}
		plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
		if !ok {
			t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
		}
		plan.ColonyMode = ""

		completion := codexExternalContinueCompletion{ContinueManifest: &plan, Dispatches: []codexContinueExternalDispatch{}}
		completionData, err := json.MarshalIndent(completion, "", "  ")
		if err != nil {
			t.Fatalf("marshal completion: %v", err)
		}
		completionPath := filepath.Join(root, "continue-completion-recorded.json")
		if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
			t.Fatalf("write completion: %v", err)
		}

		var outBuf bytes.Buffer
		stdout = &outBuf
		t.Cleanup(func() { stdout = os.Stdout })
		rootCmd.SetArgs([]string{"continue-finalize", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("continue-finalize returned error: %v", err)
		}
		env := parseLifecycleEnvelope(t, outBuf.String())
		result := env["result"].(map[string]interface{})
		// Upgraded per this test's original instruction: the H-04 fix made
		// reconciled tasks with passing verification advance, so this route
		// now proves advancement. The reconcile remains visible as a
		// non-blocking operational note, and the reconcile-gating blockers
		// ("manually reconciled" as a blocking issue, "no implementation
		// evidence or reconciliation was recorded") must be gone.
		if blocked, _ := result["blocked"].(bool); blocked {
			t.Fatalf("expected the reconciled+verified finalize path to advance (H-04), got blocked: %v", result)
		}
		gateBlockers := stringSliceValue(result["blocking_issues"])
		if len(gateBlockers) != 0 {
			t.Fatalf("expected no blocking issues on the reconciled+verified finalize path, got: %v", gateBlockers)
		}
		operational := stringSliceValue(result["operational_issues"])
		if !anyContains(operational, "manually reconciled") {
			t.Fatalf("expected the reconcile to stay visible as an operational note, got: %v", operational)
		}
		verification, ok := result["verification"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected verification field in result, got %#v", result)
		}
		if enforced, _ := verification["criteria_enforced"].(bool); !enforced {
			t.Fatalf("expected criteria_enforced:true, got %v", verification)
		}
		if passed, _ := verification["criteria_passed"].(bool); !passed {
			t.Fatalf("expected criteria_passed:true once read-only evidence is recorded, got %v", verification)
		}
		blockingIssues := stringSliceValue(result["blocking_issues"])
		if anyContains(blockingIssues, "was not claimed by the current build") {
			t.Fatalf("criterion-specific blocking issue survived --read-only-artifact at finalize: %v", blockingIssues)
		}

		// D-01 proof: the claims file carries a read_only entry scoped to
		// the task, and the task's files_modified claim list is unchanged --
		// no false modification claim was manufactured.
		var claims codexBuildClaims
		if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
			t.Fatalf("load claims: %v", err)
		}
		var recordedEvidence *codexBuildArtifactEvidence
		for i := range claims.ArtifactEvidence {
			if claims.ArtifactEvidence[i].Path == "untouched_finalize_source.go" {
				recordedEvidence = &claims.ArtifactEvidence[i]
			}
		}
		if recordedEvidence == nil || !recordedEvidence.ReadOnly || recordedEvidence.ReadOnlyTaskID != taskID {
			t.Fatalf("expected read-only evidence for untouched_finalize_source.go scoped to task %s, got %+v", taskID, claims.ArtifactEvidence)
		}
		foundTaskClaim := false
		for _, tc := range claims.TaskClaims {
			if tc.TaskID != taskID {
				continue
			}
			foundTaskClaim = true
			if len(tc.FilesModified) != 1 || tc.FilesModified[0] != "unrelated_finalize_test.go" {
				t.Fatalf("task %s files_modified changed unexpectedly: %v", taskID, tc.FilesModified)
			}
			for _, f := range tc.FilesModified {
				if f == "untouched_finalize_source.go" {
					t.Fatalf("a false files_modified claim was created for untouched_finalize_source.go")
				}
			}
		}
		if !foundTaskClaim {
			t.Fatalf("expected a task claim entry for %s, got %+v", taskID, claims.TaskClaims)
		}
	})

	t.Run("blocks at finalize when the artifact changed after evidence was recorded", func(t *testing.T) {
		root, taskID, artifactPath := setupContinueCriterionEvidenceFinalizeFixture(t)

		planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
			ReconcileTaskIDs:  []string{taskID},
			ReadOnlyArtifacts: []string{taskID + ":untouched_finalize_source.go"},
			LightFlag:         true,
			SkipWatchers:      true,
		})
		if err != nil {
			t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
		}
		plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
		if !ok {
			t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
		}
		plan.ColonyMode = ""

		// Tamper: edit the artifact after evidence was recorded at plan-only
		// time, but before continue-finalize runs.
		if err := os.WriteFile(artifactPath, []byte("package main\n\nvar untouchedFinalizeMarker = \"tampered\"\n"), 0644); err != nil {
			t.Fatalf("mutate artifact: %v", err)
		}

		completion := codexExternalContinueCompletion{ContinueManifest: &plan, Dispatches: []codexContinueExternalDispatch{}}
		completionData, err := json.MarshalIndent(completion, "", "  ")
		if err != nil {
			t.Fatalf("marshal completion: %v", err)
		}
		completionPath := filepath.Join(root, "continue-completion-tamper.json")
		if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
			t.Fatalf("write completion: %v", err)
		}

		// A verification-level block (checksum mismatch) is reported inside
		// the normal envelope's blocking_issues, not as a Go error from
		// rootCmd.Execute() -- verifyPlanReadOnlyArtifactEvidence only
		// confirms evidence WAS recorded; it is evaluatePhaseCriterionEvidence
		// (inside the shared verification snapshot) that re-checks the
		// CURRENT hash and reports the tamper as a blocking issue.
		var outBuf bytes.Buffer
		stdout = &outBuf
		t.Cleanup(func() { stdout = os.Stdout })
		rootCmd.SetArgs([]string{"continue-finalize", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("continue-finalize returned error: %v", err)
		}
		env := parseLifecycleEnvelope(t, outBuf.String())
		result := env["result"].(map[string]interface{})
		blockingIssues := stringSliceValue(result["blocking_issues"])
		if !anyContains(blockingIssues, "changed after build evidence was recorded") {
			t.Fatalf("expected tamper blocking issue after editing the artifact, got %v", blockingIssues)
		}
	})

	t.Run("fails loudly when read_only_artifacts were never recorded", func(t *testing.T) {
		root, taskID, _ := setupContinueCriterionEvidenceFinalizeFixture(t)

		// Plan-only WITHOUT --read-only-artifact: nothing gets recorded into
		// the claims file.
		planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{
			ReconcileTaskIDs: []string{taskID},
			LightFlag:        true,
			SkipWatchers:     true,
		})
		if err != nil {
			t.Fatalf("runCodexContinuePlanOnly returned error: %v", err)
		}
		plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
		if !ok {
			t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
		}
		plan.ColonyMode = ""
		// Simulate a wrapper hand-editing read_only_artifacts into the plan
		// manifest without an operator ever running
		// `--read-only-artifact` (T-163.1-45): nothing was recorded for this
		// spec, so continue-finalize must refuse to trust it.
		plan.ReadOnlyArtifacts = []string{taskID + ":untouched_finalize_source.go"}

		completion := codexExternalContinueCompletion{ContinueManifest: &plan, Dispatches: []codexContinueExternalDispatch{}}
		completionData, err := json.MarshalIndent(completion, "", "  ")
		if err != nil {
			t.Fatalf("marshal completion: %v", err)
		}
		completionPath := filepath.Join(root, "continue-completion-unrecorded.json")
		if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
			t.Fatalf("write completion: %v", err)
		}

		// verifyPlanReadOnlyArtifactEvidence's error is a hard failure:
		// runCodexContinueFinalize returns it directly, and continueFinalizeCmd
		// renders it via outputError (which writes the JSON error envelope to
		// stderr) before returning a generic renderedCommandError sentinel to
		// Cobra -- so the actual message is asserted from the rendered
		// envelope, not err.Error().
		var errBuf bytes.Buffer
		stderr = &errBuf
		t.Cleanup(func() { stderr = os.Stderr })
		rootCmd.SetArgs([]string{"continue-finalize", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err == nil {
			t.Fatalf("expected continue-finalize to error on an unrecorded read-only spec")
		}
		env := parseEnvelope(t, errBuf.String())
		if env["ok"] != false {
			t.Fatalf("expected error envelope, got %#v", env)
		}
		errText := stringValue(env["error"])
		if !strings.Contains(errText, "--read-only-artifact") {
			t.Fatalf("error = %q, want missing-evidence message naming --read-only-artifact", errText)
		}
	})
}
