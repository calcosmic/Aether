package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanningStageReceiptBindsManifestOutputAndResultingState(t *testing.T) {
	root := t.TempDir()
	running, manifest := planningStageReceiptTestScoutDispatch(t, root)
	output := []byte(`{"result_type":"planning-scout-result/v1","candidate":"snapshot-1"}`)
	reference, err := writePlanningStageOutput(root, manifest, output, planningStageWriteOptions{})
	if err != nil {
		t.Fatal(err)
	}

	candidateHash := planningStageTestHash("7")
	receipt, err := finalizePlanningStage(root, manifest, planningStageFinalizeRequest{
		To:                    planningStageRouteReady,
		CandidateSnapshotHash: candidateHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SchemaVersion != planningStageReceiptSchemaVersion || receipt.ID == "" || !planningSHA256Pattern.MatchString(receipt.ContentHash) {
		t.Fatalf("receipt is not content addressed: %+v", receipt)
	}
	if receipt.ManifestID != manifest.ID || receipt.ManifestHash != manifest.ContentHash || receipt.InputFrontierHash != manifest.InputFrontierHash {
		t.Fatalf("receipt lost manifest frontier: %+v", receipt)
	}
	if receipt.OutputPath != reference.Path || receipt.OutputHash != reference.ContentHash || receipt.Caste != planningStageCasteScout || receipt.Pass != manifest.Pass {
		t.Fatalf("receipt lost worker output identity: %+v", receipt)
	}
	if receipt.PriorReceiptID != "" || receipt.PriorReceiptHash != "" || receipt.ResultingState != planningStageRouteReady {
		t.Fatalf("first receipt has an invalid predecessor or state: %+v", receipt)
	}
	if err := validateStageReceipt(receipt); err != nil {
		t.Fatalf("stored receipt is invalid: %v", err)
	}

	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.Stage != planningStageRouteReady || state.CandidateSnapshotHash != candidateHash || state.ScoutReceipt == nil {
		t.Fatalf("receipt transaction did not persist the resulting state: %+v", state)
	}
	if state.ScoutReceipt.ID != receipt.ID || state.ScoutReceipt.ContentHash != receipt.ContentHash || state.ScoutReceipt.ManifestHash != manifest.ContentHash {
		t.Fatalf("resulting state does not bind the exact Scout receipt: %+v", state.ScoutReceipt)
	}
	if state.ActiveManifestID != "" || state.ActiveManifestHash != "" || running.ActiveManifestID != manifest.ID {
		t.Fatalf("active manifest lifecycle is inconsistent: running=%+v completed=%+v", running, state)
	}
}

func TestPlanningStageReceiptExactReplayIsByteStableAndDivergenceConflicts(t *testing.T) {
	root := t.TempDir()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	output := []byte(`{"result_type":"planning-scout-result/v1","candidate":"stable"}`)
	if _, err := writePlanningStageOutput(root, manifest, output, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	request := planningStageFinalizeRequest{To: planningStageRouteReady, CandidateSnapshotHash: planningStageTestHash("6")}
	first, err := finalizePlanningStage(root, manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, filepath.FromSlash(planningStageReceiptRepositoryPath(manifest.RunID, first.ID)))
	beforeReceipt := planningStageReceiptTestReadBytes(t, receiptPath)
	beforeState := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID))))

	replayed, err := finalizePlanningStage(root, manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != first.ID || replayed.ContentHash != first.ContentHash {
		t.Fatalf("exact replay returned a different receipt: first=%+v replay=%+v", first, replayed)
	}
	if after := planningStageReceiptTestReadBytes(t, receiptPath); !bytes.Equal(after, beforeReceipt) {
		t.Fatal("exact replay rewrote canonical receipt bytes")
	}
	if after := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))); !bytes.Equal(after, beforeState) {
		t.Fatal("exact replay rewrote the planning frontier")
	}

	if _, err := writePlanningStageOutput(root, manifest, []byte(`{"changed":true}`), planningStageWriteOptions{}); err == nil {
		t.Fatal("divergent output for the same manifest was accepted")
	} else {
		var conflict *planningStageReceiptConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("divergent output error = %T %v, want planningStageReceiptConflictError", err, err)
		}
	}
	if after := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))); !bytes.Equal(after, beforeState) {
		t.Fatal("divergent replay changed the planning frontier")
	}
}

func TestPlanningStageResumeCrashBeforeReceiptFinalizesSameManifest(t *testing.T) {
	root := t.TempDir()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	if _, err := writePlanningStageOutput(root, manifest, []byte(`{"result_type":"planning-scout-result/v1"}`), planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	crash := errors.New("simulated crash before receipt write")
	request := planningStageFinalizeRequest{
		To:                    planningStageRouteReady,
		CandidateSnapshotHash: planningStageTestHash("7"),
		Fault: func(point string) error {
			if point == "after_intent" {
				return crash
			}
			return nil
		},
	}
	if _, err := finalizePlanningStage(root, manifest, request); !errors.Is(err, crash) {
		t.Fatalf("interrupted finalizer error = %v, want simulated crash", err)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(planningStageReceiptIndexRepositoryPath(manifest.RunID)))); !os.IsNotExist(err) {
		t.Fatalf("receipt index exists before receipt commit: %v", err)
	}

	resume, err := resumePlanningStage(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resume.Action != planningStageResumeFinalize || resume.Manifest == nil || resume.Manifest.ID != manifest.ID {
		t.Fatalf("crash-before-receipt resume = %+v, want finalize same manifest", resume)
	}

	request.Fault = nil
	receipt, err := finalizePlanningStage(root, manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	resume, err = resumePlanningStage(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resume.Action != planningStageResumeNextStage || resume.LastReceipt == nil || resume.LastReceipt.ID != receipt.ID {
		t.Fatalf("completed retry resume = %+v, want one next-stage action", resume)
	}
}

func TestPlanningStageResumeCrashAfterReceiptAdvancesOnce(t *testing.T) {
	root := t.TempDir()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	if _, err := writePlanningStageOutput(root, manifest, []byte(`{"result_type":"planning-scout-result/v1"}`), planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	crash := errors.New("simulated crash after stage targets")
	request := planningStageFinalizeRequest{
		To:                    planningStageRouteReady,
		CandidateSnapshotHash: planningStageTestHash("8"),
		Fault: func(point string) error {
			if point == "after_global_verification" {
				return crash
			}
			return nil
		},
	}
	if _, err := finalizePlanningStage(root, manifest, request); !errors.Is(err, crash) {
		t.Fatalf("interrupted finalizer error = %v, want simulated crash", err)
	}

	resume, err := resumePlanningStage(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resume.Action != planningStageResumeNextStage || resume.State.Stage != planningStageRouteReady || resume.LastReceipt == nil {
		t.Fatalf("crash-after-receipt resume = %+v, want exactly one advance", resume)
	}
	request.Fault = nil
	replayed, err := finalizePlanningStage(root, manifest, request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != resume.LastReceipt.ID {
		t.Fatalf("post-crash replay duplicated receipt: resume=%q replay=%q", resume.LastReceipt.ID, replayed.ID)
	}
	index := planningStageReceiptTestReadIndex(t, root, manifest.RunID)
	if len(index.Entries) != 1 {
		t.Fatalf("crash replay appended %d receipts, want 1", len(index.Entries))
	}
}

func TestPlanningStageReceiptRouteStageAppendsAndVerifiesIterationCard(t *testing.T) {
	root := t.TempDir()
	scoutReceipt, routeState := planningStageReceiptTestCompleteScout(t, root)
	routeRunning, routeManifest := planningStageReceiptTestRouteDispatch(t, root, routeState)
	if routeRunning.ActiveManifestID != routeManifest.ID {
		t.Fatalf("Route dispatch did not persist its active manifest: %+v", routeRunning)
	}
	if _, err := writePlanningStageOutput(root, routeManifest, []byte(`{"result_type":"planning-route-setter-result/v1"}`), planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 16, 30, 0, 0, time.UTC))
	card.RunID = routeManifest.RunID
	receipt, err := finalizePlanningStage(root, routeManifest, planningStageFinalizeRequest{
		To:        planningStageCandidateReady,
		RouteCard: &card,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.PriorReceiptID != scoutReceipt.ID || receipt.PriorReceiptHash != scoutReceipt.ContentHash || receipt.Caste != planningStageCasteRouteSetter {
		t.Fatalf("Route receipt does not chain to Scout: %+v", receipt)
	}
	timeline, err := loadPlanningTimeline(root, routeManifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Cards) != 1 || timeline.Cards[0].RouteSetterReceiptID != receipt.ID || timeline.Cards[0].RouteSetterReceiptHash != receipt.ContentHash {
		t.Fatalf("iteration card does not bind exact Route receipt: %+v", timeline.Cards)
	}
	if timeline.Cards[0].ScoutReceiptID != scoutReceipt.ID || timeline.Cards[0].ScoutReceiptHash != scoutReceipt.ContentHash {
		t.Fatalf("iteration card does not bind exact Scout receipt: %+v", timeline.Cards[0])
	}
	resume, err := resumePlanningStage(root, routeManifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resume.Action != planningStageResumeCandidateReview || resume.State.Stage != planningStageCandidateReady {
		t.Fatalf("candidate resume = %+v, want candidate review", resume)
	}
}

func TestPlanningStageResumeRejectsTamperWithoutRewritingState(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(t *testing.T, root string, receipt StageReceipt)
	}{
		{
			name: "output",
			tamper: func(t *testing.T, root string, receipt StageReceipt) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(receipt.OutputPath)), []byte("tampered\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "prior hash",
			tamper: func(t *testing.T, root string, receipt StageReceipt) {
				t.Helper()
				originalID := receipt.ID
				receipt.PriorReceiptHash = planningStageTestHash("0")
				if err := addressStageReceipt(&receipt); err != nil {
					t.Fatal(err)
				}
				planningStageReceiptTestWriteJSON(t, filepath.Join(root, filepath.FromSlash(planningStageReceiptRepositoryPath(receipt.RunID, originalID))), receipt)
			},
		},
		{
			name: "caste",
			tamper: func(t *testing.T, root string, receipt StageReceipt) {
				t.Helper()
				originalID := receipt.ID
				receipt.Caste = planningStageCasteScout
				if err := addressStageReceipt(&receipt); err != nil {
					t.Fatal(err)
				}
				planningStageReceiptTestWriteJSON(t, filepath.Join(root, filepath.FromSlash(planningStageReceiptRepositoryPath(receipt.RunID, originalID))), receipt)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			_, routeState := planningStageReceiptTestCompleteScout(t, root)
			_, routeManifest := planningStageReceiptTestRouteDispatch(t, root, routeState)
			if _, err := writePlanningStageOutput(root, routeManifest, []byte(`{"result_type":"planning-route-setter-result/v1"}`), planningStageWriteOptions{}); err != nil {
				t.Fatal(err)
			}
			card := validPlanningIterationCardForTest(t, 1, time.Date(2026, time.September, 7, 16, 45, 0, 0, time.UTC))
			card.RunID = routeManifest.RunID
			receipt, err := finalizePlanningStage(root, routeManifest, planningStageFinalizeRequest{To: planningStageCandidateReady, RouteCard: &card})
			if err != nil {
				t.Fatal(err)
			}
			statePath := filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(routeManifest.RunID)))
			before := planningStageReceiptTestReadBytes(t, statePath)
			test.tamper(t, root, receipt)
			if _, err := resumePlanningStage(root, routeManifest.RunID); err == nil {
				t.Fatal("tampered receipt chain was accepted")
			}
			if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, before) {
				t.Fatal("failed resume rewrote planning state")
			}
		})
	}
}

func TestPlanningStageReceiptHasNoGeneralEventBusDependency(t *testing.T) {
	content := planningStageReceiptTestReadBytes(t, "planning_stage_receipt.go")
	for _, forbidden := range []string{"pkg/events", "event-bus", "event_bus"} {
		if strings.Contains(string(content), forbidden) {
			t.Fatalf("planning stage receipt introduced forbidden general event transport %q", forbidden)
		}
	}
}

func planningStageReceiptTestScoutDispatch(t *testing.T, root string) (planningStageState, planningStageManifest) {
	t.Helper()
	state := planningStageTestState(planningStageScoutReady)
	authorization := planningStageAuthorization{
		ID:                "authorization-scout-receipt-test",
		ExpectedCaste:     planningStageCasteScout,
		InputFrontierHash: state.InputFrontierHash,
		EvidenceFrontier:  []planningStageEvidenceBinding{{ID: "evidence", ContentHash: planningStageTestHash("1")}},
		WeakestGap:        planningStageTestGap("receipt-gap"),
	}
	running, manifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil {
		t.Fatal("Scout dispatch did not emit a manifest")
	}
	if err := recordPlanningStageDispatch(root, running, *manifest, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	return running, *manifest
}

func planningStageReceiptTestCompleteScout(t *testing.T, root string) (StageReceipt, planningStageState) {
	t.Helper()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	if _, err := writePlanningStageOutput(root, manifest, []byte(`{"result_type":"planning-scout-result/v1"}`), planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	receipt, err := finalizePlanningStage(root, manifest, planningStageFinalizeRequest{
		To:                    planningStageRouteReady,
		CandidateSnapshotHash: planningStageTestHash("7"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt, planningStageReceiptTestReadState(t, root, manifest.RunID)
}

func planningStageReceiptTestRouteDispatch(t *testing.T, root string, state planningStageState) (planningStageState, planningStageManifest) {
	t.Helper()
	authorization := planningStageAuthorization{
		ID:                    "authorization-route-receipt-test",
		ExpectedCaste:         planningStageCasteRouteSetter,
		InputFrontierHash:     state.InputFrontierHash,
		ScoutReceipt:          state.ScoutReceipt,
		CandidateSnapshotHash: state.CandidateSnapshotHash,
	}
	running, manifest, err := reducePlanningStage(state, planningStageTransition{To: planningStageRouteRunning, Authorization: &authorization})
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil {
		t.Fatal("Route dispatch did not emit a manifest")
	}
	if err := recordPlanningStageDispatch(root, running, *manifest, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	return running, *manifest
}

func planningStageReceiptTestReadState(t *testing.T, root, runID string) planningStageState {
	t.Helper()
	content := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(runID))))
	var state planningStageState
	if err := decodePlanningStageJSON(content, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func planningStageReceiptTestReadIndex(t *testing.T, root, runID string) planningStageReceiptIndex {
	t.Helper()
	content := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(planningStageReceiptIndexRepositoryPath(runID))))
	var index planningStageReceiptIndex
	if err := decodePlanningStageJSON(content, &index); err != nil {
		t.Fatal(err)
	}
	return index
}

func planningStageReceiptTestReadBytes(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func planningStageReceiptTestWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
