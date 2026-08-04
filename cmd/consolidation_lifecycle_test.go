package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// hasConsolidationPhaseEndEvent reports whether the real (non-dry-run)
// consolidation pipeline published its "consolidation.phase_end" event to
// the persisted event bus during this test -- the cleanest available signal
// that runPhaseEndConsolidation actually ran (RESEARCH.md's recommendation).
// Reuses readPersistedCeremonyEvents (cmd/ceremony_emitter_test.go).
func hasConsolidationPhaseEndEvent(t *testing.T) bool {
	t.Helper()
	for _, evt := range readPersistedCeremonyEvents(t) {
		if evt.Topic == "consolidation.phase_end" {
			return true
		}
	}
	return false
}

// seedConsolidationWiringFixture seeds a minimal but real, VALID consolidation
// fixture (an instinct plus an empty-but-parseable observations file) against
// the current store global, so a wiring test's "did consolidation actually
// run" assertion exercises the real mutating pipeline instead of a load
// failure or a no-op on missing files.
func seedConsolidationWiringFixture(t *testing.T) {
	t.Helper()
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{
		{ID: "inst_wiring_fixture", Trigger: "t", Action: "a", TrustScore: 0.05, TrustTier: "untrusted", Confidence: 0.2},
	}}); err != nil {
		t.Fatalf("seed instincts fixture: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed observations fixture: %v", err)
	}
}

// captureStderrForConsolidationTest redirects os.Stderr for the duration of
// fn and returns everything written to it. Modelled on the identical local
// helper in cmd/hive_policy_test.go (TestHiveRuntimePolicyUnrecognizedWarns).
func captureStderrForConsolidationTest(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy stderr: %v", err)
	}
	return buf.String()
}

// TestRunPhaseEndConsolidationMutatesOnRealPath proves runPhaseEndConsolidation
// takes the real (mutating) path, exactly as consolidationPhaseEndCmd's own
// non-dry-run branch does -- mirroring TestConsolidationRealRunStillMutates'
// shape (cmd/consolidation_dryrun_test.go).
func TestRunPhaseEndConsolidationMutatesOnRealPath(t *testing.T) {
	saveGlobals(t)

	dataDir := seedConsolidationFixture(t)
	instincts := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instincts)

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on the real path, got summary: %+v", summary)
	}
	if after := hashFileForTest(t, instincts); after == before {
		t.Fatal("runPhaseEndConsolidation left instincts.json untouched; real path did not mutate")
	}
}

// TestRunPhaseEndConsolidationIsNonBlockingOnFailure asserts D-05: a
// consolidation failure never panics or exits, and is reported through the
// summary rather than propagated as an error the caller could act on.
func TestRunPhaseEndConsolidationIsNonBlockingOnFailure(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	// instincts.json containing invalid JSON makes ConsolidationService.Run's
	// LoadJSON step fail; that failure lands in result.Errors rather than a
	// top-level error, which runPhaseEndConsolidation must still treat as a
	// non-blocking failure.
	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	if err := os.WriteFile(instinctsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid instincts.json: %v", err)
	}

	var summary phaseEndConsolidationSummary
	stderr := captureStderrForConsolidationTest(t, func() {
		summary = runPhaseEndConsolidation(1)
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on failure, got summary: %+v", summary)
	}
	if summary.Reason == "" {
		t.Fatal("expected a non-empty Reason on failure")
	}
	if !strings.Contains(stderr, "phase advanced WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderr)
	}
}

// TestRunPhaseEndConsolidationZeroState asserts D-06/D-07's zero-state
// contract: an empty but VALID store (files exist, parse cleanly, contain
// nothing) still runs cleanly and reports ZeroState() == true.
func TestRunPhaseEndConsolidationZeroState(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed empty instincts.json: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed empty learning-observations.json: %v", err)
	}

	summary := runPhaseEndConsolidation(1)

	if !summary.Ran {
		t.Fatalf("expected Ran == true on an empty-but-valid store, got summary: %+v", summary)
	}
	if summary.PromotionCandidates != 0 {
		t.Errorf("expected PromotionCandidates == 0, got %d", summary.PromotionCandidates)
	}
	if summary.QueenEligible != 0 {
		t.Errorf("expected QueenEligible == 0, got %d", summary.QueenEligible)
	}
	if !summary.ZeroState() {
		t.Fatal("expected ZeroState() == true for an empty-but-valid store")
	}
}

// holdStaleInstinctsLock simulates a crashed process that died holding the
// instincts.json lock: it acquires an exclusive flock through a SEPARATE
// FileLocker on the store's locks directory and deliberately never releases
// it. flock blocks a second acquirer even within one process when taken
// through a separate descriptor, so any store operation touching
// instincts.json parks indefinitely -- exactly the in-step block a context
// deadline alone cannot cancel (WR-02). The lock is intentionally leaked for
// the remaining life of the test binary: releasing it in cleanup would let
// the abandoned consolidation goroutine wake up and mutate a torn-down
// temp dir.
func holdStaleInstinctsLock(t *testing.T, dataDir string) {
	t.Helper()
	locker, err := storage.NewFileLocker(filepath.Join(filepath.Dir(dataDir), "locks"))
	if err != nil {
		t.Fatalf("create locker: %v", err)
	}
	if err := locker.Lock("instincts.json"); err != nil {
		t.Fatalf("acquire stale lock: %v", err)
	}
	// Pin the locker (and its open lock fd) until the test ends: os.File
	// carries a finalizer that closes the descriptor when the locker becomes
	// garbage, and closing the fd RELEASES the flock -- letting the "stale"
	// lock silently evaporate mid-test whenever GC runs.
	t.Cleanup(func() { _ = locker })
}

// TestRunPhaseEndConsolidationTimesOutOnStaleLock gives T-162-12's "can
// never hang a phase advance" claim teeth (WR-02): with a stale exclusive
// lock on instincts.json, runPhaseEndConsolidation must return within its
// timeout with Ran:false and the D-05 stderr warning -- not block forever
// inside storage.FileLocker's deadline-less flock.
func TestRunPhaseEndConsolidationTimesOutOnStaleLock(t *testing.T) {
	saveGlobals(t)
	dataDir := seedConsolidationFixture(t)

	origTimeout := consolidationLifecycleTimeout
	consolidationLifecycleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { consolidationLifecycleTimeout = origTimeout })

	holdStaleInstinctsLock(t, dataDir)

	var summary phaseEndConsolidationSummary
	stderrOut := captureStderrForConsolidationTest(t, func() {
		done := make(chan phaseEndConsolidationSummary, 1)
		go func() { done <- runPhaseEndConsolidation(1) }()
		select {
		case summary = <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("runPhaseEndConsolidation hung on a stale file lock; the consolidation timeout is decorative (WR-02)")
		}
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on timeout, got: %+v", summary)
	}
	if !strings.Contains(summary.Reason, "timed out") {
		t.Errorf("expected Reason to name the timeout, got: %q", summary.Reason)
	}
	if !strings.Contains(stderrOut, "phase advanced WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderrOut)
	}
}

// TestRunSealConsolidationTimesOutOnStaleLock is WR-02's seal-path twin: the
// curation orchestrator only polls ctx BETWEEN ant steps, so an in-step
// block (the sentinel's read of a lock-held instincts.json) must be bounded
// by the goroutine+select wrapper, never by cooperative cancellation.
func TestRunSealConsolidationTimesOutOnStaleLock(t *testing.T) {
	saveGlobals(t)
	dataDir := seedConsolidationFixture(t)

	origTimeout := consolidationLifecycleTimeout
	consolidationLifecycleTimeout = 200 * time.Millisecond
	t.Cleanup(func() { consolidationLifecycleTimeout = origTimeout })

	holdStaleInstinctsLock(t, dataDir)

	var summary sealConsolidationSummary
	stderrOut := captureStderrForConsolidationTest(t, func() {
		done := make(chan sealConsolidationSummary, 1)
		go func() { done <- runSealConsolidation() }()
		select {
		case summary = <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("runSealConsolidation hung on a stale file lock; the consolidation timeout is decorative (WR-02)")
		}
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on timeout, got: %+v", summary)
	}
	if !strings.Contains(summary.Reason, "timed out") {
		t.Errorf("expected Reason to name the timeout, got: %q", summary.Reason)
	}
	if !strings.Contains(stderrOut, "colony sealed WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderrOut)
	}
}

// TestContinueAdvanceInvokesPhaseEndConsolidation proves D-04's default-path
// wiring: a durable phase advance through the default continue path invokes
// runPhaseEndConsolidation, observable via the consolidation.phase_end event
// the real (non-dry-run) pipeline publishes. Deleting the call site in
// cmd/codex_continue.go must make this test fail.
func TestContinueAdvanceInvokesPhaseEndConsolidation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Wire phase-end consolidation on the default continue path"
	now := time.Now().UTC()
	taskOneID := "1.1"
	taskTwoID := "1.2"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Consolidation wiring (default path)",
					Description: "Prove runPhaseEndConsolidation fires beside captureContinueLearning",
					Status:      colony.PhaseInProgress,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "Implement the packet", Status: colony.TaskInProgress},
						{ID: &taskTwoID, Goal: "Verify the packet", Status: colony.TaskInProgress},
					},
				},
				{
					ID:     2,
					Name:   "Next slice",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Keep moving", Status: colony.TaskPending}},
				},
			},
		},
	})

	// A real fixture, not an empty store: proves consolidation actually ran
	// the mutating pipeline, not merely that a summary map was attached.
	seedConsolidationWiringFixture(t)

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-consolidation-1", Task: "Implement the packet", Status: "spawned", TaskID: taskOneID},
		{Stage: "wave", Wave: 1, Caste: "scout", Name: "Ranger-consolidation-1", Task: "Research the packet", Status: "spawned", TaskID: taskTwoID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-consolidation-1", Task: "Independent verification before advancement", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Consolidation wiring (default path)", goal, dispatches)

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true (precondition for D-04 wiring), got %v", result)
	}

	if !hasConsolidationPhaseEndEvent(t) {
		t.Fatal("expected a consolidation.phase_end event after a default continue advance; runPhaseEndConsolidation call site may be missing")
	}
}

// TestExternalContinueAdvanceInvokesPhaseEndConsolidation proves D-04's
// external/finalize-path wiring: runPhaseEndConsolidation fires after
// advanceExternalContinue returns with err == nil (the stricter-correct
// placement per RESEARCH.md assumption A1, since PhaseCompleted is written
// INSIDE that call). Deleting the call site in
// cmd/codex_continue_finalize.go must make this test fail.
func TestExternalContinueAdvanceInvokesPhaseEndConsolidation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Wire phase-end consolidation on the external finalize path"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Consolidation wiring (external path)",
					Description: "Prove runPhaseEndConsolidation fires after advanceExternalContinue",
					Status:      colony.PhaseInProgress,
					Tasks:       []colony.Task{{ID: &taskID, Goal: "Advance durably", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedConsolidationWiringFixture(t)

	seedContinueBuildPacket(t, dataDir, 1, "Consolidation wiring (external path)", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-consolidation-2", Task: "Advance durably", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-consolidation-2", Task: "Independent verification before advancement", Status: "completed"},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan := planResult["continue_manifest"].(codexContinuePlanManifest)
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		results = append(results, codexContinueExternalDispatch{
			Stage:   dispatch.Stage,
			Wave:    dispatch.Wave,
			Caste:   dispatch.Caste,
			Name:    dispatch.Name,
			Task:    dispatch.Task,
			TaskID:  dispatch.TaskID,
			Status:  "completed",
			Summary: dispatch.Name + " cleared consolidation wiring review",
		})
	}
	completion := codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       results,
	}

	result, _, _, _, _, _, err := runCodexContinueFinalize(root, completion, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true (precondition for D-04 wiring), got %v", result)
	}

	if !hasConsolidationPhaseEndEvent(t) {
		t.Fatal("expected a consolidation.phase_end event after an external continue-finalize advance; runPhaseEndConsolidation call site may be missing")
	}
}

// TestContinueWithoutAdvanceDoesNotConsolidate gives D-04's "on real advance
// only" half teeth: a continue run that fails gates (a blocked continue
// watcher) must produce no consolidation side effect at all. Moving the
// default-path call above the atomic COLONY_STATE.json write must make this
// test fail.
func TestContinueWithoutAdvanceDoesNotConsolidate(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	now := time.Now().UTC()
	goal := "Continue that does not advance must not consolidate"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Consolidation wiring (blocked path)",
			Status: colony.PhaseInProgress,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Should not advance", Status: colony.TaskPending}},
		}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	seedConsolidationWiringFixture(t)
	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	before := hashFileForTest(t, instinctsPath)

	seedContinueBuildPacket(t, s.BasePath(), 1, "Consolidation wiring (blocked path)", goal, []codexBuildDispatch{{
		Stage:  "implementation",
		Caste:  "builder",
		Name:   "Mason-consolidation-3",
		TaskID: taskID,
		Task:   "Should not advance",
		Status: "completed",
	}})
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		return &continueWatcherTestInvoker{watcherStatus: "blocked", watcherSummary: "Continue watcher rejected the phase"}
	}

	result, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{})
	if err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Fatalf("expected advanced:false for a blocked continue, got %v", result)
	}

	if hasConsolidationPhaseEndEvent(t) {
		t.Fatal("a continue that did not advance must not consolidate (D-04); the call site may have moved above the gate")
	}
	if after := hashFileForTest(t, instinctsPath); after != before {
		t.Fatal("a continue that did not advance mutated instincts.json; consolidation must not run without a durable advance")
	}
}

// sealCurationAntOrder is the fixed sequential order pkg/agent/curation's
// Orchestrator runs its eight ants in (pkg/agent/curation/orchestrator.go).
var sealCurationAntOrder = []string{"sentinel", "nurse", "critic", "herald", "janitor", "archivist", "librarian", "scribe"}

// TestRunSealConsolidationRunsAllEightAnts proves runSealConsolidation calls
// the curation orchestrator directly and preserves all eight individual
// StepResults, in orchestrator order, rather than collapsing them into one
// aggregate string the way consolidationSealCmd's own stepInfo does.
func TestRunSealConsolidationRunsAllEightAnts(t *testing.T) {
	saveGlobals(t)
	seedConsolidationFixture(t)

	summary := runSealConsolidation()

	if !summary.Ran {
		t.Fatalf("expected Ran == true against a seeded, valid store, got: %+v", summary)
	}
	if len(summary.Ants) != 8 {
		t.Fatalf("expected exactly 8 ant beats, got %d: %+v", len(summary.Ants), summary.Ants)
	}
	for i, name := range sealCurationAntOrder {
		if summary.Ants[i].Name != name {
			t.Errorf("ant beat %d: got name %q, want %q (beats collapsed or reordered)", i, summary.Ants[i].Name, name)
		}
	}
}

// TestRunSealConsolidationWritesReportArtifact proves the scribe's report
// string (previously built into Summary["report"] and discarded, since
// Summary["path"] was hardcoded to "") is now persisted to a real file whose
// path the summary returns.
func TestRunSealConsolidationWritesReportArtifact(t *testing.T) {
	saveGlobals(t)
	seedConsolidationFixture(t)

	summary := runSealConsolidation()

	if !summary.Ran {
		t.Fatalf("expected Ran == true, got: %+v", summary)
	}
	if summary.ReportPath == "" {
		t.Fatal("expected a non-empty ReportPath")
	}
	expectedPath := filepath.Join(filepath.Dir(store.BasePath()), "CURATION-REPORT.md")
	if summary.ReportPath != expectedPath {
		t.Errorf("ReportPath = %q, want %q", summary.ReportPath, expectedPath)
	}
	data, err := os.ReadFile(summary.ReportPath)
	if err != nil {
		t.Fatalf("CURATION-REPORT.md was not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("CURATION-REPORT.md is empty")
	}
	if !strings.Contains(string(data), "Curation Report") {
		t.Errorf("expected report to contain the scribe's 'Curation Report' heading, got: %q", string(data))
	}
}

// TestRunSealConsolidationNonBlockingOnFailure asserts D-05 on the seal
// path: a corrupt instincts.json triggers a sentinel abort inside the
// curation orchestrator, and runSealConsolidation must report Ran: false
// with the sentinel abort named in Reason, warn loudly to stderr, and never
// panic or exit.
func TestRunSealConsolidationNonBlockingOnFailure(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	instinctsPath := filepath.Join(s.BasePath(), "instincts.json")
	if err := os.WriteFile(instinctsPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid instincts.json: %v", err)
	}

	var summary sealConsolidationSummary
	stderrOut := captureStderrForConsolidationTest(t, func() {
		summary = runSealConsolidation()
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on a sentinel abort, got summary: %+v", summary)
	}
	if !strings.Contains(summary.Reason, "sentinel abort") {
		t.Errorf("expected Reason to name the sentinel abort, got: %q", summary.Reason)
	}
	if !strings.Contains(stderrOut, "colony sealed WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderrOut)
	}
}

// TestRunSealConsolidationSentinelAbortShortCircuitsPipeline pins CR-01: when
// the curation sentinel aborts (a corrupt store OTHER than instincts.json, so
// the consolidation pipeline itself would run cleanly), runSealConsolidation
// must NOT run the mutating pipeline at all. Before the short-circuit, this
// asymmetric path decayed/archived instincts and promoted into QUEEN.md, then
// returned Ran:false with an empty QueenPromotedIDs -- a false "WITHOUT
// consolidation" report plus an empty D-09 skip-set.
func TestRunSealConsolidationSentinelAbortShortCircuitsPipeline(t *testing.T) {
	saveGlobals(t)

	dataDir := seedConsolidationFixture(t)
	// Corrupt a sentinel-checked store that the pipeline does NOT read, so
	// curation fails while consolidation would succeed.
	if err := os.WriteFile(filepath.Join(dataDir, "pheromones.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid pheromones.json: %v", err)
	}

	instinctsPath := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instinctsPath)

	var summary sealConsolidationSummary
	stderrOut := captureStderrForConsolidationTest(t, func() {
		summary = runSealConsolidation()
	})

	if summary.Ran {
		t.Fatalf("expected Ran == false on a sentinel abort, got summary: %+v", summary)
	}
	if !strings.Contains(summary.Reason, "sentinel abort") {
		t.Errorf("expected Reason to name the sentinel abort, got: %q", summary.Reason)
	}
	if !strings.Contains(stderrOut, "colony sealed WITHOUT consolidation —") {
		t.Fatalf("expected unmissable D-05 stderr warning, got: %q", stderrOut)
	}
	if len(summary.QueenPromotedIDs) != 0 {
		t.Errorf("expected empty QueenPromotedIDs when the pipeline was short-circuited, got: %v", summary.QueenPromotedIDs)
	}
	if after := hashFileForTest(t, instinctsPath); after != before {
		t.Fatal("sentinel abort must short-circuit the mutating pipeline (CR-01); instincts.json was mutated after corruption was detected")
	}
}

// TestRunSealConsolidationQueenPromotedIDsExcludesFailedWrites pins WR-01 at
// the seal summary boundary: an instinct that is QueenEligible but whose
// QUEEN.md write failed must NOT appear in QueenPromotedIDs. If it did, the
// D-09 skip-set would suppress completeSealRuntime's promoteInstinctLocal
// fallback AND CROWNED-ANTHILL.md would report a promotion that never
// reached QUEEN.md.
func TestRunSealConsolidationQueenPromotedIDsExcludesFailedWrites(t *testing.T) {
	saveGlobals(t)

	s, root := newTestStore(t)
	store = s

	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Version: "1", Instincts: []colony.InstinctEntry{{
		ID:         "inst_queen_write_fails",
		Trigger:    "eligible pattern behind a failing QUEEN.md write",
		Action:     "keep QueenPromotedIDs honest about failed writes",
		Domain:     "testing",
		TrustScore: 0.9,
		TrustTier:  "trusted",
		Confidence: 0.9,
		Provenance: colony.InstinctProvenance{ApplicationCount: 3},
	}}}); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed observations: %v", err)
	}

	// Force every QUEEN.md write to fail: put a DIRECTORY where the local
	// QUEEN.md file lives, so AtomicWrite's rename onto it must error while
	// the rest of the pipeline runs cleanly.
	queenPath := filepath.Join(root, ".aether", "QUEEN.md")
	if err := os.MkdirAll(queenPath, 0o755); err != nil {
		t.Fatalf("mkdir queen dir: %v", err)
	}

	var summary sealConsolidationSummary
	_ = captureStderrForConsolidationTest(t, func() {
		summary = runSealConsolidation()
	})

	if !summary.Ran {
		t.Fatalf("precondition: expected Ran == true (a failed queen write is log-and-continue, not a pipeline error), got: %+v", summary)
	}
	if summary.QueenEligible != 1 {
		t.Fatalf("precondition: expected QueenEligible == 1, got: %+v", summary)
	}
	if len(summary.QueenPromotedIDs) != 0 {
		t.Fatalf("expected QueenPromotedIDs to exclude the failed write (WR-01), got: %v", summary.QueenPromotedIDs)
	}
}

// TestRunSealConsolidationIsNeverDryRun proves runSealConsolidation always
// takes the real mutating path: it never references
// learn.NewDryRunConsolidationService, and a real run against a seeded store
// actually mutates instincts.json.
func TestRunSealConsolidationIsNeverDryRun(t *testing.T) {
	saveGlobals(t)

	dataDir := seedConsolidationFixture(t)
	instinctsPath := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instinctsPath)

	summary := runSealConsolidation()

	if !summary.Ran {
		t.Fatalf("expected Ran == true on the real path, got summary: %+v", summary)
	}
	if after := hashFileForTest(t, instinctsPath); after == before {
		t.Fatal("runSealConsolidation left instincts.json untouched; real path did not mutate")
	}
}
