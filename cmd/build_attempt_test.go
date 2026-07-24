package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildAttemptPersistsTransitionsAndTerminalEvidence(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	if err := os.WriteFile(filepath.Join(root, "app.txt"), []byte("attempt evidence\n"), 0644); err != nil {
		t.Fatalf("write attempt artifact: %v", err)
	}
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Journal proof"}
	state := colony.ColonyState{State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", TaskID: "1.1", Status: "spawned"}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, []string{"1.1"}, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", dispatches)
	if err != nil {
		t.Fatalf("begin build attempt: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "dispatch started", dispatches, nil, "real", nil); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	dispatches[0].Status = "completed"
	summary := &codex.ClaimsSummary{
		FilesCreated: []string{"app.txt"},
		TaskClaims:   []codex.TaskClaimsSummary{{TaskID: "1.1", FilesCreated: []string{"app.txt"}}},
	}
	claims, err := recordBuildAttemptTerminal(root, attemptRel, 1, startedAt, dispatches, summary, "real", nil)
	if err != nil {
		t.Fatalf("record terminal attempt: %v", err)
	}
	if len(claims.ArtifactEvidence) != 1 || claims.ArtifactEvidence[0].Path != "app.txt" || claims.ArtifactEvidence[0].SHA256 == "" {
		t.Fatalf("terminal claims lack artifact fingerprint: %+v", claims)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "built committed", dispatches, claims, "real", nil); err != nil {
		t.Fatalf("mark built: %v", err)
	}

	loadedRel, record, ok := loadLatestBuildAttempt(1)
	if !ok || loadedRel != attemptRel {
		t.Fatalf("latest attempt = %q %+v %v, want %q", loadedRel, record, ok, attemptRel)
	}
	if record.Status != buildAttemptBuilt || record.Recoverable || record.RecoveryCommand != "" || record.OriginalStateSHA == "" {
		t.Fatalf("attempt record = %+v", record)
	}
	if len(record.History) != 4 || record.History[0].Status != buildAttemptPrepared || record.History[1].Status != buildAttemptDispatching || record.History[2].Status != buildAttemptTerminal || record.History[3].Status != buildAttemptBuilt {
		t.Fatalf("attempt history = %+v", record.History)
	}
}

func TestForceRedispatchMarksActiveAttemptInterrupted(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	startedAt := time.Now().UTC().Add(-time.Hour)
	phase := colony.Phase{ID: 1, Name: "Interrupted journal"}
	state := colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 1, BuildStartedAt: &startedAt, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil)
	if err != nil {
		t.Fatalf("begin active attempt: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "workers active", nil, nil, "real", nil); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if err := interruptLatestBuildAttempt(1, "superseded by force redispatch"); err != nil {
		t.Fatalf("interrupt attempt: %v", err)
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.Status != buildAttemptInterrupted || record.CompletedAt == "" || record.Error == "" {
		t.Fatalf("interrupted attempt = %+v, present=%v", record, ok)
	}
}

func TestStatusSurfacesFailedAttemptAfterLifecycleRollback(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Rolled back phase", Status: colony.PhaseReady}
	state := colony.ColonyState{State: colony.StateREADY, CurrentPhase: 0, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil)
	if err != nil {
		t.Fatalf("begin failed attempt: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptFailed, "dispatch failed", nil, nil, "real", os.ErrDeadlineExceeded); err != nil {
		t.Fatalf("mark failed attempt: %v", err)
	}
	result := buildStatusResult(state, store)
	summary, ok := result["build_attempt"].(map[string]interface{})
	if !ok || summary["status"] != buildAttemptFailed || summary["recovery_command"] != "aether build 1 --force" {
		t.Fatalf("status build_attempt = %#v", result["build_attempt"])
	}
}

func TestBuildAttemptVisualShowsDurableRunAndWorkerState(t *testing.T) {
	attempt := buildAttemptRecord{
		ID:              "attempt-visual",
		Status:          buildAttemptDispatching,
		RunID:           "run-visual-proof",
		ManifestSHA256:  "1234567890abcdef",
		RecoveryCommand: "aether build 1 --force",
		Dispatches:      []codexBuildDispatch{{Name: "Builder-1"}, {Name: "Watcher-1"}},
		WorkerRuns: []buildAttemptWorkerRun{
			{WorkerName: "Builder-1", Status: buildWorkerCompleted},
			{WorkerName: "Watcher-1", Status: buildWorkerDispatching},
		},
	}

	output := renderBuildAttemptStatus(attempt)
	for _, expected := range []string{
		"run-visual-proof",
		"Manifest: 1234567890ab",
		"1 completed | 1 active",
		"aether build 1 --force",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("build attempt visual missing %q:\n%s", expected, output)
		}
	}
}

func TestResumeDashboardDoesNotRedispatchLiveBuildProcess(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	goal := "Keep live work isolated"
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Live phase", Status: colony.PhaseInProgress}
	state := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		Plan:           colony.Plan{Phases: []colony.Phase{phase}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save live state: %v", err)
	}
	if _, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil); err != nil {
		t.Fatalf("begin live attempt: %v", err)
	}
	result := buildResumeDashboardResult()
	recovery, ok := result["recovery"].(map[string]interface{})
	if !ok || recovery["next"] != "aether watch" {
		t.Fatalf("live build recovery = %#v, want aether watch", result["recovery"])
	}
}
