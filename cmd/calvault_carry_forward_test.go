package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCalVaultCarryForwardAcceptedClosureSupersedesBlockedHistory(t *testing.T) {
	root, _ := setupOutOfBandStuckFixture(t, boundPassingOutOfBandPhase())
	var state colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
		state.Plan.Phases = append(state.Plan.Phases, colony.Phase{ID: 2, Name: "Next phase", Status: colony.PhasePending})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	historical := map[string]string{
		"review.json":       `{"phase":1,"generated_at":"2026-01-01T00:00:00Z","blocking_issues":["old Auditor artifacts.review missing"]}`,
		"verification.json": `{"phase":1,"generated_at":"2026-01-01T00:00:00Z","steps":[{"name":"old check","passed":false}]}`,
		"outcome.md":        "Old blocked closeout: Auditor artifacts.review missing",
	}
	for name, content := range historical {
		if err := store.AtomicWrite(continuePlanArtifactsPath(1, name), []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if before := resolvePreviousPhaseCarryForward(2); !strings.Contains(before, "old Auditor") {
		t.Fatalf("missing blocked precondition: %s", before)
	}
	report, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err != nil || result["advanced"] != true {
		t.Fatalf("close: %v, %+v, %+v", err, report, result)
	}
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateREADY || state.CurrentPhase != 2 {
		t.Fatalf("not ready for next phase: %+v", state)
	}
	assertAccepted := func() {
		t.Helper()
		got := resolvePreviousPhaseCarryForward(2)
		if !strings.Contains(got, "accepted verify-out-of-band") || !strings.Contains(got, "superseded historical") || strings.Contains(got, "old Auditor") || strings.Contains(got, "Old blocked closeout") {
			t.Fatalf("stale carry-forward: %s", got)
		}
	}
	assertAccepted()
	// A closure accepted by older runtimes is recoverable from its existing
	// authoritative attempt provenance without rewriting any history.
	closurePath := continuePlanArtifactsPath(1, "out-of-band-closure.json")
	savedClosure, err := store.ReadFile(closurePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(store.BasePath(), closurePath)); err != nil {
		t.Fatal(err)
	}
	assertAccepted()
	_, acceptedAttempt, ok := loadLatestBuildAttempt(1)
	if !ok || acceptedAttempt.OutOfBandVerification == nil {
		t.Fatal("missing legacy provenance")
	}
	verifiedAt, err := time.Parse(time.RFC3339Nano, acceptedAttempt.OutOfBandVerification.VerifiedAt)
	if err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(store.BasePath(), continuePlanArtifactsPath(1, "review.json"))
	reviewInfo, err := os.Stat(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	roundedReport := fmt.Sprintf(`{"generated_at":%q,"blocking_issues":["newer same-second blocker"]}`, verifiedAt.Format(time.RFC3339))
	if err := os.WriteFile(reviewPath, []byte(roundedReport), 0600); err != nil {
		t.Fatal(err)
	}
	sameSecondLater := verifiedAt.Add(time.Nanosecond)
	if err := os.Chtimes(reviewPath, sameSecondLater, sameSecondLater); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("legacy closure suppressed same-second report: %s", got)
	}
	if err := os.WriteFile(reviewPath, []byte(historical["review.json"]), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(reviewPath, reviewInfo.ModTime(), reviewInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	assertAccepted()
	outcomePath := filepath.Join(store.BasePath(), continuePlanArtifactsPath(1, "outcome.md"))
	outcomeInfo, err := os.Stat(outcomePath)
	if err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(outcomePath, later, later); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("legacy closure suppressed later outcome: %s", got)
	}
	if err := os.Chtimes(outcomePath, outcomeInfo.ModTime(), outcomeInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	assertAccepted()
	if err := store.AtomicWrite(closurePath, savedClosure); err != nil {
		t.Fatal(err)
	}
	var malformed outOfBandCarryForward
	if err := store.LoadJSON(closurePath, &malformed); err != nil {
		t.Fatal(err)
	}
	malformed.Provenance.VerifiedAt = ""
	if err := store.SaveJSON(closurePath, malformed); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("malformed closure accepted: %s", got)
	}
	if err := store.AtomicWrite(closurePath, savedClosure); err != nil {
		t.Fatal(err)
	}
	for name, content := range historical {
		got, err := store.ReadFile(continuePlanArtifactsPath(1, name))
		if err != nil || string(got) != content {
			t.Fatalf("history changed: %s: %v", name, err)
		}
	}
	// A later report invalidates the closure's snapshot, even on the same attempt.
	if err := store.AtomicWrite(continuePlanArtifactsPath(1, "review.json"), []byte(`{"blocking_issues":["new review blocker"]}`)); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); !strings.Contains(got, "new review blocker") || strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("new report suppressed: %s", got)
	}
	if err := store.AtomicWrite(continuePlanArtifactsPath(1, "review.json"), []byte(historical["review.json"])); err != nil {
		t.Fatal(err)
	}
	assertAccepted()
	// The same historical files cannot make an old closure authorize a new attempt.
	_, priorAttempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("missing attempt")
	}
	rebuiltState := state
	rebuiltState.Plan.Phases = append([]colony.Phase(nil), state.Plan.Phases...)
	rebuiltState.Plan.Phases[0].Status = colony.PhaseReady
	rebuiltState.State = colony.StateREADY
	rebuiltState.CurrentPhase = 1
	rebuiltState.BuildStartedAt = nil
	if err := store.SaveJSON("COLONY_STATE.json", rebuiltState); err != nil {
		t.Fatal(err)
	}
	// The public plan-only path commits the replacement through the canonical
	// build-start transaction rather than manufacturing an attempt and pointer.
	if _, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	rel, replacement, ok := loadLatestBuildAttempt(1)
	if !ok || replacement.ID == priorAttempt.ID {
		t.Fatal("replacement attempt not authoritative")
	}
	if err := transitionBuildAttempt(rel, buildAttemptBuilt, "replacement finished", nil, nil, "", nil); err != nil {
		t.Fatal(err)
	}
	// Restore completed phase facts so identity, rather than an unfinished
	// phase status, is what prevents the old closure from being selected.
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("old closure reused for replacement attempt: %s", got)
	}
}

func TestCalVaultCarryForwardClosureWithoutAttempt(t *testing.T) {
	setupOutOfBandTest(t, boundPassingOutOfBandPhase())
	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatal(err)
	}
	state.Plan.Phases = append(state.Plan.Phases, colony.Phase{ID: 2, Status: colony.PhasePending})
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	report := outOfBandReport{Phase: 1, Passed: true, Policy: criterionEvidencePolicyBoundV1}
	if _, err := closeOutOfBandCeremony(1, state, report, "", buildAttemptRecord{}, false, false); err != nil {
		t.Fatal(err)
	}
	if got := resolvePreviousPhaseCarryForward(2); !strings.Contains(got, "accepted verify-out-of-band") {
		t.Fatalf("missing no-attempt closure: %s", got)
	}
}
