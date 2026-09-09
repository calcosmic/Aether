package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildStartTransaction200TargetMatrix(t *testing.T) {
	tests := []struct {
		variant buildStartVariant
		want    []string
	}{
		{
			variant: buildStartDirect,
			want: []string{
				"COLONY_STATE.json",
				"build/phase-1/manifest.json",
				"build/phase-1/worker-reports/stale.json",
				"build/phase-1/verification.json",
				"checkpoints/pre-build-phase-1.json",
				pendingDecisionsFile,
				phaseDispatchWindowFileName,
			},
		},
		{
			variant: buildStartPlanOnly,
			want: []string{
				"build/phase-1/manifest.json",
				phaseDispatchWindowFileName,
			},
		},
		{
			variant: buildStartQueenLed,
			want: []string{
				"build/phase-1/manifest.json",
				phaseDispatchWindowFileName,
			},
		},
		{
			variant: buildStartExternalUnbound,
			want: []string{
				"COLONY_STATE.json",
				"build/phase-1/attempts/ATTEMPT.completion.json",
				"checkpoints/pre-build-phase-1.json",
				"last-build-claims.json",
			},
		},
		{
			variant: buildStartAutomaticCheckFix,
			want: []string{
				pendingDecisionsFile,
				phaseDispatchWindowFileName,
			},
		},
		{
			variant: buildStartCoherentChildRetry,
			want:    nil,
		},
	}

	for _, test := range tests {
		t.Run(string(test.variant), func(t *testing.T) {
			root, request := buildStartTransaction200Fixture(t, test.variant)
			receipt, err := commitBuildStart(root, request, buildStartOptions{})
			if err != nil {
				t.Fatalf("commit %s build start: %v", test.variant, err)
			}

			attemptRel := buildStartTransaction200AttemptPath(request)
			receiptRel := buildStartTransaction200ReceiptPath(request)
			want := append([]string(nil), test.want...)
			want = append(want, attemptRel, receiptRel)
			if request.Effects.MakeLatest {
				want = append(want, latestBuildAttemptPointerPath(request.Phase))
			}
			for i := range want {
				want[i] = strings.ReplaceAll(want[i], "ATTEMPT", request.AttemptID)
			}
			sort.Strings(want)
			got := make([]string, 0, len(receipt.Targets)+1)
			for _, target := range receipt.Targets {
				got = append(got, target.Path)
			}
			got = append(got, receipt.Path)
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s target matrix =\n%v\nwant\n%v", test.variant, got, want)
			}
			if receipt.RequestSHA256 == "" || receipt.PlanAuthority != request.PlanAuthority || receipt.AttemptID != request.AttemptID {
				t.Fatalf("receipt does not bind request authority/attempt: %+v", receipt)
			}

			var attempt buildAttemptRecord
			if err := store.LoadJSON(attemptRel, &attempt); err != nil {
				t.Fatalf("load attempt: %v", err)
			}
			if attempt.ID != request.AttemptID || attempt.RunID != request.RunID || attempt.WorkspaceSHA256 != request.WorkspaceSHA256 ||
				attempt.ExecutionOwner != request.ExecutionOwner || !reflect.DeepEqual(attempt.SelectedTasks, request.SelectedTasks) {
				t.Fatalf("attempt lost canonical request identity: %+v", attempt)
			}
			switch test.variant {
			case buildStartAutomaticCheckFix:
				if attempt.CheckFix == nil || !reflect.DeepEqual(*attempt.CheckFix, *request.Effects.CheckFix) {
					t.Fatalf("check-fix provenance = %+v", attempt.CheckFix)
				}
			case buildStartCoherentChildRetry:
				if attempt.ParentAttemptID != request.Effects.ParentAttemptID || attempt.ParentJobName != request.Effects.ParentJobName {
					t.Fatalf("child provenance = %+v", attempt)
				}
				if _, err := os.Stat(filepath.Join(store.BasePath(), filepath.FromSlash(latestBuildAttemptPointerPath(request.Phase)))); !os.IsNotExist(err) {
					t.Fatalf("child retry moved latest pointer: %v", err)
				}
			}
		})
	}
}

func TestBuildStartTransaction200FaultsRollbackAndNeverDispatch(t *testing.T) {
	root, request := buildStartTransaction200Fixture(t, buildStartDirect)
	var faultPoints []string
	dispatched := 0
	if _, err := commitBuildStart(root, request, buildStartOptions{
		Fault: func(point string) error {
			faultPoints = append(faultPoints, point)
			return nil
		},
		Dispatch: func(receipt buildStartReceipt) error {
			dispatched++
			if _, err := os.Stat(filepath.Join(store.BasePath(), filepath.FromSlash(receipt.Path))); err != nil {
				return fmt.Errorf("dispatch ran before durable receipt: %w", err)
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("discover transaction fault points: %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("successful start dispatch callbacks = %d, want 1", dispatched)
	}
	faultPoints = uniqueSortedStrings(faultPoints)
	if len(faultPoints) < 8 {
		t.Fatalf("fault surface too small: %v", faultPoints)
	}

	for _, point := range faultPoints {
		point := point
		t.Run(strings.ReplaceAll(point, ":", "_"), func(t *testing.T) {
			root, request := buildStartTransaction200Fixture(t, buildStartDirect)
			buildStartTransaction200PrimeSession(t, root)
			before := buildStartTransaction200Inventory(t, root)
			injected := errors.New("plan-33 injected transaction fault")
			dispatched := 0
			_, err := commitBuildStart(root, request, buildStartOptions{
				Fault: func(got string) error {
					if got == point {
						return injected
					}
					return nil
				},
				Dispatch: func(buildStartReceipt) error {
					dispatched++
					return nil
				},
			})
			if !errors.Is(err, injected) {
				t.Fatalf("fault %q error = %v, want injected", point, err)
			}
			if dispatched != 0 {
				t.Fatalf("fault %q authorized %d dispatches", point, dispatched)
			}
			buildStartTransaction200AssertInventory(t, root, before)
		})
	}

	t.Run("rename faults", func(t *testing.T) {
		root, request := buildStartTransaction200Fixture(t, buildStartDirect)
		renames := 0
		if _, err := commitBuildStart(root, request, buildStartOptions{Rename: func(oldPath, newPath string) error {
			renames++
			return os.Rename(oldPath, newPath)
		}}); err != nil {
			t.Fatalf("count target renames: %v", err)
		}
		if renames < 4 {
			t.Fatalf("target rename count = %d, want a multi-target transaction", renames)
		}
		for failAt := 1; failAt <= renames; failAt++ {
			failAt := failAt
			t.Run(fmt.Sprintf("rename_%02d", failAt), func(t *testing.T) {
				root, request := buildStartTransaction200Fixture(t, buildStartDirect)
				buildStartTransaction200PrimeSession(t, root)
				before := buildStartTransaction200Inventory(t, root)
				injected := errors.New("plan-33 injected rename fault")
				calls := 0
				dispatched := 0
				_, err := commitBuildStart(root, request, buildStartOptions{
					Rename: func(oldPath, newPath string) error {
						calls++
						if calls == failAt {
							return injected
						}
						return os.Rename(oldPath, newPath)
					},
					Dispatch: func(buildStartReceipt) error { dispatched++; return nil },
				})
				if !errors.Is(err, injected) {
					t.Fatalf("rename %d error = %v, want injected", failAt, err)
				}
				if dispatched != 0 {
					t.Fatalf("rename fault %d authorized dispatch", failAt)
				}
				buildStartTransaction200AssertInventory(t, root, before)
			})
		}
	})
}

func TestBuildStartTransaction200StaleAuthorityAndTasksAreZeroEffect(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*buildStartRequest)
	}{
		{name: "state digest", mutate: func(request *buildStartRequest) { request.StateSHA256 = strings.Repeat("f", 64) }},
		{name: "active revision", mutate: func(request *buildStartRequest) { request.PlanAuthority.ActiveRevision.Hash = strings.Repeat("e", 64) }},
		{name: "unknown selected task", mutate: func(request *buildStartRequest) { request.SelectedTasks = []string{"1.404"} }},
		{name: "manifest phase", mutate: func(request *buildStartRequest) { request.Effects.Manifest.Phase = 2 }},
		{name: "manifest authority", mutate: func(request *buildStartRequest) { request.Effects.Manifest.PlanAuthority.ActiveRevision.ID = "forged" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, request := buildStartTransaction200Fixture(t, buildStartDirect)
			test.mutate(&request)
			buildStartTransaction200PrimeSession(t, root)
			before := buildStartTransaction200Inventory(t, root)
			dispatched := 0
			if _, err := commitBuildStart(root, request, buildStartOptions{Dispatch: func(buildStartReceipt) error { dispatched++; return nil }}); err == nil {
				t.Fatalf("stale %s request was accepted", test.name)
			}
			if dispatched != 0 {
				t.Fatalf("stale %s request authorized dispatch", test.name)
			}
			buildStartTransaction200AssertInventory(t, root, before)
		})
	}

	t.Run("baseline changes after authority reload", func(t *testing.T) {
		root, request := buildStartTransaction200Fixture(t, buildStartDirect)
		buildStartTransaction200PrimeSession(t, root)
		mutated := false
		dispatched := 0
		_, err := commitBuildStart(root, request, buildStartOptions{
			Fault: func(point string) error {
				if point == buildStartBeforeCommitFaultPoint && !mutated {
					mutated = true
					content, readErr := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
					if readErr != nil {
						return readErr
					}
					content = bytes.Replace(content, []byte("Atomic build start"), []byte("Concurrent stale edit"), 1)
					return os.WriteFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"), content, 0o644)
				}
				return nil
			},
			Dispatch: func(buildStartReceipt) error { dispatched++; return nil },
		})
		if err == nil || !strings.Contains(err.Error(), "baseline") {
			t.Fatalf("stale held-session baseline error = %v", err)
		}
		if dispatched != 0 {
			t.Fatal("stale held-session baseline authorized dispatch")
		}
		if _, statErr := os.Stat(filepath.Join(store.BasePath(), filepath.FromSlash(buildStartTransaction200AttemptPath(request)))); !os.IsNotExist(statErr) {
			t.Fatalf("stale baseline created an attempt: %v", statErr)
		}
	})
}

func TestBuildStartTransaction200ExactReplayIsReadOnly(t *testing.T) {
	root, request := buildStartTransaction200Fixture(t, buildStartDirect)
	dispatched := 0
	first, err := commitBuildStart(root, request, buildStartOptions{Dispatch: func(buildStartReceipt) error { dispatched++; return nil }})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}
	before := buildStartTransaction200Inventory(t, root)
	second, err := commitBuildStart(root, request, buildStartOptions{Dispatch: func(buildStartReceipt) error { dispatched++; return nil }})
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("replayed receipt differs:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if dispatched != 1 {
		t.Fatalf("exact replay dispatched %d total times, want first commit only", dispatched)
	}
	buildStartTransaction200AssertInventory(t, root, before)

	conflict := request
	conflict.ExecutionOwner = "forged-owner"
	if _, err := commitBuildStart(root, conflict, buildStartOptions{}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflicting replay error = %v", err)
	}
	buildStartTransaction200AssertInventory(t, root, before)
}

func TestBuildStartTransaction200PureAttemptDerivation(t *testing.T) {
	_, request := buildStartTransaction200Fixture(t, buildStartCoherentChildRetry)
	state := buildStartTransaction200State(t)
	phase := state.Plan.Phases[0]
	input := buildAttemptDerivation{
		State: state, Phase: phase, PhaseNumber: 1, StartedAt: request.GeneratedAt,
		AttemptID: request.AttemptID, RunID: request.RunID, ProcessID: request.ProcessID,
		HostPlatform: request.HostPlatform, WorkspaceSHA256: request.WorkspaceSHA256,
		SelectedTaskIDs: request.SelectedTasks, ExecutionOwner: request.ExecutionOwner,
		Dispatches: request.Dispatches, MakeLatest: false,
		ParentAttemptID: request.Effects.ParentAttemptID, ParentJobName: request.Effects.ParentJobName,
	}
	firstPath, first, firstPointer, err := deriveBuildAttempt(input)
	if err != nil {
		t.Fatalf("first derivation: %v", err)
	}
	secondPath, second, secondPointer, err := deriveBuildAttempt(input)
	if err != nil {
		t.Fatalf("second derivation: %v", err)
	}
	if firstPath != secondPath || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstPointer, secondPointer) {
		t.Fatalf("attempt derivation is not deterministic:\n%q %+v %+v\n%q %+v %+v", firstPath, first, firstPointer, secondPath, second, secondPointer)
	}
	if firstPointer != nil {
		t.Fatalf("makeLatest=false derived pointer %+v", firstPointer)
	}
	if first.ParentAttemptID != input.ParentAttemptID || first.ParentJobName != input.ParentJobName {
		t.Fatalf("pure derivation lost retry provenance: %+v", first)
	}
}

func buildStartTransaction200Fixture(t *testing.T, variant buildStartVariant) (string, buildStartRequest) {
	t.Helper()
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	state := buildStartTransaction200State(t)
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save canonical state: %v", err)
	}

	generatedAt := time.Date(2026, time.September, 9, 10, 11, 12, 345000000, time.UTC)
	facts := lifecycleFactsFromStateSnapshot(state, false, generatedAt)
	facts.Root = root
	authority, err := preflightCodexBuildPlanAuthority(facts, planAuthorityVerifiedBindings{})
	if err != nil {
		t.Fatalf("derive legacy authority: %v", err)
	}
	stateSHA, err := jsonSHA256(state)
	if err != nil {
		t.Fatal(err)
	}
	workspaceSHA, err := codex.WorkspaceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	planSHA, err := planStateHash(state.Plan)
	if err != nil {
		t.Fatal(err)
	}
	dispatchMode := string(variant)
	executionOwner := "runtime-worker-dispatch"
	planOnly := false
	switch variant {
	case buildStartPlanOnly:
		dispatchMode, executionOwner, planOnly = "plan-only", "host-queen", true
	case buildStartQueenLed:
		dispatchMode, executionOwner, planOnly = "queen-led", "host-queen", true
	case buildStartExternalUnbound:
		dispatchMode, executionOwner = "external-task", "external-task"
	case buildStartAutomaticCheckFix:
		dispatchMode, executionOwner = "check-fix-attempt", "check-fix-attempt"
	case buildStartCoherentChildRetry:
		dispatchMode, executionOwner = "coherent-child-retry", "runtime-worker-dispatch"
	}
	dispatches := []codexBuildDispatch{{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-33", Task: "Implement atomic start", Status: "planned", TaskID: "1.1", TaskIndex: 1}}
	request := buildStartRequest{
		SchemaVersion: buildStartSchemaVersion,
		Variant:       variant, StateSHA256: stateSHA, PlanAuthority: authority,
		Phase: 1, SelectedTasks: []string{"1.1"}, ExecutionOwner: executionOwner,
		DispatchMode: dispatchMode, GeneratedAt: generatedAt,
		AttemptID: deriveBuildAttemptID(generatedAt, 3300), RunID: "run-33333333333333333333333333333333",
		ProcessID: 3300, HostPlatform: "test", WorkspaceSHA256: workspaceSHA,
		Dispatches: dispatches,
	}
	manifest := codexBuildManifest{
		Phase: 1, PhaseName: state.Plan.Phases[0].Name, Root: root, PlanOnly: planOnly,
		DispatchMode: dispatchMode, ExecutionOwner: executionOwner, GeneratedAt: generatedAt.Format(time.RFC3339),
		PlanAuthority: authority, PlanRevisionID: authority.ActiveRevision.ID, PlanStateHash: planSHA,
		State: string(state.State), Dispatches: append([]codexBuildDispatch(nil), dispatches...),
		SelectedTasks: append([]string(nil), request.SelectedTasks...),
	}
	request.Effects.MakeLatest = variant != buildStartCoherentChildRetry
	switch variant {
	case buildStartDirect:
		request.Effects.CheckpointPath = "checkpoints/pre-build-phase-1.json"
		request.Effects.ManifestPath, request.Effects.Manifest = "build/phase-1/manifest.json", &manifest
		request.Effects.PromoteState = true
		request.Effects.ReviewerWindow = buildStartReviewerClose
		request.Effects.StalePaths = []string{"build/phase-1/verification.json", "build/phase-1/worker-reports/stale.json"}
	case buildStartPlanOnly, buildStartQueenLed:
		request.Effects.ManifestPath, request.Effects.Manifest = "build/phase-1/manifest.json", &manifest
		request.Effects.ReviewerWindow = buildStartReviewerReopen
	case buildStartExternalUnbound:
		request.Effects.CheckpointPath = "checkpoints/pre-build-phase-1.json"
		legacyManifest := manifest
		legacyManifest.AttemptID, legacyManifest.AttemptPath, legacyManifest.ExecutionBinding = "", "", nil
		request.Effects.Completion = &codexExternalBuildCompletion{DispatchManifest: &legacyManifest}
		request.Effects.ClaimsPath = "last-build-claims.json"
		request.Effects.Claims = &codexBuildClaims{BuildPhase: 1, Timestamp: generatedAt.Format(time.RFC3339)}
		request.Effects.PromoteState = true
	case buildStartAutomaticCheckFix:
		request.Effects.ReviewerWindow = buildStartReviewerClose
		request.Effects.CheckFix = &checkFixAttemptRecord{
			RecordedAt: generatedAt.Format(time.RFC3339), Phase: 1, Check: "tests",
			Reason: "fixing the failed tests check", ParentAttemptID: "attempt-parent-check-fix", Outcome: "pending",
		}
	case buildStartCoherentChildRetry:
		request.Effects.ParentAttemptID = "attempt-parent-partial"
		request.Effects.ParentJobName = "atomic-start-job"
	}

	window := phaseDispatchWindowFile{Phases: map[string]string{"2": generatedAt.Add(-time.Hour).Format(time.RFC3339Nano)}}
	if variant == buildStartPlanOnly || variant == buildStartQueenLed {
		window.Phases["1"] = generatedAt.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	}
	if request.Effects.ReviewerWindow != buildStartReviewerNone {
		if err := store.SaveJSON(phaseDispatchWindowFileName, window); err != nil {
			t.Fatal(err)
		}
	}
	if request.Effects.ReviewerWindow == buildStartReviewerClose {
		phase := 1
		otherPhase := 2
		pending := PendingDecisionFile{Decisions: []PendingDecision{
			{ID: "remove", Description: "unresolved reviewer", Phase: &phase, Source: "forced-reviewer-waiver", AttemptID: request.AttemptID},
			{ID: "keep-resolved", Description: "resolved reviewer", Phase: &phase, Source: "forced-reviewer-waiver", Resolved: true},
			{ID: "keep-other", Description: "other phase", Phase: &otherPhase, Source: "forced-reviewer-waiver"},
		}}
		if err := store.SaveJSON(pendingDecisionsFile, pending); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range request.Effects.StalePaths {
		path := filepath.Join(dataDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("stale\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, request
}

func buildStartTransaction200State(t *testing.T) colony.ColonyState {
	t.Helper()
	taskID := "1.1"
	goal := "Atomic build start"
	return colony.ColonyState{
		Goal:  &goal,
		State: colony.StateREADY,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases: []colony.Phase{{
				ID: 1, Name: "Atomic build start", Status: colony.PhaseReady,
				Tasks: []colony.Task{{ID: &taskID, Goal: "Commit all effects", Status: colony.TaskPending}},
			}},
		},
	}
}

func buildStartTransaction200AttemptPath(request buildStartRequest) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", request.Phase), "attempts", request.AttemptID+".json"))
}

func buildStartTransaction200ReceiptPath(request buildStartRequest) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", request.Phase), "attempts", request.AttemptID+".start-receipt.json"))
}

func buildStartTransaction200PrimeSession(t *testing.T, root string) {
	t.Helper()
	if err := withPlanningMutationSession(root, "plan-33-test-prime", func(*planningMutationSession) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func buildStartTransaction200Inventory(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	dataRoot := filepath.Join(root, ".aether", "data")
	err := filepath.WalkDir(dataRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dataRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() && (rel == "transactions" || strings.Contains(rel, "/"+lifecycleTransactionDirectory) || rel == lifecycleTransactionDirectory || rel == ".locks") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[rel] = content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func buildStartTransaction200AssertInventory(t *testing.T, root string, want map[string][]byte) {
	t.Helper()
	got := buildStartTransaction200Inventory(t, root)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("artifact inventory changed:\ngot=%v\nwant=%v", buildStartTransaction200InventoryDigests(got), buildStartTransaction200InventoryDigests(want))
	}
}

func buildStartTransaction200InventoryDigests(inventory map[string][]byte) []string {
	result := make([]string, 0, len(inventory))
	for path, content := range inventory {
		result = append(result, fmt.Sprintf("%s=%x", path, content))
	}
	sort.Strings(result)
	return result
}
