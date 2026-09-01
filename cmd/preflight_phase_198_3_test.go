package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func phaseScopedPreflightDispatch(root string, phase int, workflow string) []codex.WorkerDispatch {
	return []codex.WorkerDispatch{{
		ID:         "phase-preflight",
		WorkerName: "Builder-Phase",
		Caste:      "builder",
		TaskID:     "phase-preflight",
		TaskBrief:  "Prove readiness is shared only inside one phase",
		Root:       root,
		Wave:       1,
		Workflow:   workflow,
		Phase:      phase,
	}}
}

func TestPreflightSuccessIsScopedToWorkerDispatchPhase(t *testing.T) {
	withTestPreflightStore(t)
	withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	root := t.TempDir()
	for _, workflow := range []string{"build", "build", "continue"} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 4, workflow)); err != nil {
			t.Fatalf("phase 4 %s preflight = %v, want nil", workflow, err)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("phase 4 build/continue probes = %d, want 1", fake.calls)
	}

	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 5, "build")); err != nil {
		t.Fatalf("phase 5 build preflight = %v, want nil", err)
	}
	if fake.calls != 2 {
		t.Fatalf("probes after entering phase 5 = %d, want 2; changing phase must beat the old one-hour TTL", fake.calls)
	}

	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 5, "continue")); err != nil {
		t.Fatalf("phase 5 continue preflight = %v, want nil", err)
	}
	if fake.calls != 2 {
		t.Fatalf("phase 5 build/continue probes = %d, want 2 total", fake.calls)
	}
}

func TestPreflightPhaseZeroRetainsUnscopedTTLBehavior(t *testing.T) {
	withTestPreflightStore(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		status, _ := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)
		if !status.Available {
			t.Fatalf("unscoped preflight %d reported unavailable", i+1)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("phase 0 probes = %d, want 1 inside the TTL", fake.calls)
	}
}

func TestPreflightInvalidationClearsEveryPhaseScope(t *testing.T) {
	withTestPreflightStore(t)
	withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	root := t.TempDir()
	for _, phase := range []int{4, 5} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, phase, "build")); err != nil {
			t.Fatalf("warm phase %d preflight = %v, want nil", phase, err)
		}
	}
	if fake.calls != 2 {
		t.Fatalf("warm probes = %d, want 2 (one per phase)", fake.calls)
	}

	results := []codex.DispatchResult{{
		WorkerName: "Builder-Phase",
		Error:      errors.New("authentication failed"),
	}}
	if !invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results) {
		t.Fatal("provider auth failure did not trigger invalidation")
	}

	for _, phase := range []int{4, 5} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, phase, "continue")); err != nil {
			t.Fatalf("phase %d preflight after invalidation = %v, want nil", phase, err)
		}
	}
	if fake.calls != 4 {
		t.Fatalf("probes after platform invalidation = %d, want 4; every cached phase scope must be removed", fake.calls)
	}
}

func TestPreflightOldSchemaCannotWarmEveryPhase(t *testing.T) {
	s := withTestPreflightStore(t)
	withCapturedGateStderr(t)
	now := time.Now().UTC()

	legacy := preflightCacheFile{
		SchemaVersion: 1,
		Entries: map[string]preflightCacheEntry{
			string(codex.PlatformClaude): {
				Platform:  string(codex.PlatformClaude),
				Available: true,
				CheckedAt: now.Format(time.RFC3339),
			},
		},
	}
	if err := s.SaveJSON(preflightCachePathRel, &legacy); err != nil {
		t.Fatalf("write legacy cache: %v", err)
	}

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(t.TempDir(), 4, "build")); err != nil {
		t.Fatalf("phase 4 preflight over legacy cache = %v, want nil", err)
	}
	if fake.calls != 1 {
		t.Fatalf("phase 4 probes over legacy cache = %d, want 1; a platform-only success cannot authorize every phase", fake.calls)
	}

	var upgraded preflightCacheFile
	if err := s.LoadJSON(preflightCachePathRel, &upgraded); err != nil {
		t.Fatalf("load upgraded cache: %v", err)
	}
	if upgraded.SchemaVersion != 2 {
		t.Fatalf("cache schema_version = %d, want 2", upgraded.SchemaVersion)
	}
}

func TestInternalWorkerAdapterPreflightCLIUsesPhaseScope(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s := withTestPreflightStore(t)
	forceJSONOutputModeForTest(t)

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	rootCmd.SetArgs([]string{"internal-worker-adapter", "--preflight", "--simulate", "--phase", "7"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("internal-worker-adapter --preflight --phase 7 = %v; stderr=%s", err, errBuf.String())
	}

	var cache preflightCacheFile
	if err := s.LoadJSON(preflightCachePathRel, &cache); err != nil {
		t.Fatalf("load phase-scoped adapter cache: %v", err)
	}
	key := preflightCacheKey(codex.PlatformFake, 7)
	entry, ok := cache.Entries[key]
	if !ok {
		t.Fatalf("adapter cache keys = %v, want %q", cache.Entries, key)
	}
	if entry.Phase != 7 || !entry.Available {
		t.Fatalf("adapter phase entry = %+v, want successful phase 7 readiness", entry)
	}
	if _, ok := cache.Entries[preflightCacheKey(codex.PlatformFake, 0)]; ok {
		t.Fatal("--phase 7 warmed the unscoped phase-0 cache")
	}
}

func TestInternalWorkerAdapterPreflightCLIRejectsInvalidPhaseBeforeProbe(t *testing.T) {
	for _, phase := range []string{"-1", "not-a-number"} {
		t.Run(phase, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			s := withTestPreflightStore(t)
			forceJSONOutputModeForTest(t)
			if err := s.SaveJSON(preflightCachePathRel, &preflightCacheFile{
				SchemaVersion: preflightCacheSchemaVersion,
				Entries:       map[string]preflightCacheEntry{},
			}); err != nil {
				t.Fatalf("seed empty preflight cache: %v", err)
			}

			var outBuf, errBuf bytes.Buffer
			stdout = &outBuf
			stderr = &errBuf
			rootCmd.SetArgs([]string{"internal-worker-adapter", "--preflight", "--simulate", "--phase", phase})
			if err := rootCmd.Execute(); err == nil {
				t.Fatalf("--phase %q succeeded, want validation error", phase)
			}

			var cache preflightCacheFile
			if err := s.LoadJSON(preflightCachePathRel, &cache); err != nil {
				t.Fatalf("load cache after invalid phase: %v", err)
			}
			if len(cache.Entries) != 0 {
				t.Fatalf("--phase %q probed before validation; cache=%+v", phase, cache.Entries)
			}
		})
	}
}

type directBuildPreflightInvoker struct {
	available      bool
	preflightCalls int
	invokeCalls    int
}

func (i *directBuildPreflightInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	i.invokeCalls++
	if !i.available {
		return codex.WorkerResult{}, errors.New("Invoke must not run after failed readiness")
	}
	const changedPath = "direct-preflight-success.txt"
	if err := os.WriteFile(filepath.Join(config.Root, changedPath), []byte("provider-backed build change\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName:   config.WorkerName,
		Caste:        config.Caste,
		TaskID:       config.TaskID,
		Status:       "completed",
		Summary:      "provider readiness passed and the worker changed the project",
		FilesCreated: []string{changedPath},
	}, nil
}

func (i *directBuildPreflightInvoker) IsAvailable(context.Context) bool { return true }
func (i *directBuildPreflightInvoker) ValidateAgent(string) error       { return nil }
func (i *directBuildPreflightInvoker) Platform() codex.Platform         { return codex.PlatformCodex }

func (i *directBuildPreflightInvoker) Preflight(context.Context, string) codex.AvailabilityStatus {
	i.preflightCalls++
	if i.available {
		return codex.AvailabilityStatus{
			Platform:  codex.PlatformCodex,
			Available: true,
			Category:  codex.AvailabilityCategoryAvailable,
		}
	}
	return codex.AvailabilityStatus{
		Platform:  codex.PlatformCodex,
		Available: false,
		Category:  codex.AvailabilityCategoryProviderConfig,
		Reason:    "configured model is unavailable for this phase",
	}
}

type directBuildPreflightFixture struct {
	root       string
	phase      int
	options    codexBuildOptions
	attemptRel string
}

func readyDirectBuildPreflightState(goal *string, phaseID int, taskID *string) colony.ColonyState {
	return colony.ColonyState{
		Version:      "3.0",
		Goal:         goal,
		State:        colony.StateREADY,
		CurrentPhase: 0,
		ColonyDepth:  "light",
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     phaseID,
			Name:   fmt.Sprintf("Preflight phase %d", phaseID),
			Status: colony.PhaseReady,
			Tasks: []colony.Task{{
				ID:     taskID,
				Goal:   "Make one provider-backed change",
				Status: colony.TaskPending,
			}},
		}}},
		Events: []string{"2026-09-01T10:00:00Z|initialized|init|Preflight fixture ready"},
	}
}

func seedDirectBuildPreflightCleanupSentinels(t *testing.T, phase int) {
	t.Helper()
	rels := []string{
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "verification.json"),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "gates.json"),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "continue.json"),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "review.json"),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "worker-reports", "sentinel.txt"),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "worker-briefs", "sentinel.md"),
		filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phase)),
		filepath.Join("build", fmt.Sprintf("phase-%d", phase), "manifest.json"),
		"heartbeat-preflight-sentinel.json",
	}
	for _, rel := range rels {
		path := filepath.Join(store.BasePath(), rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create sentinel parent for %s: %v", rel, err)
		}
		payload := []byte(fmt.Sprintf("{\"sentinel\":%q}\n", filepath.ToSlash(rel)))
		if err := os.WriteFile(path, payload, 0644); err != nil {
			t.Fatalf("write cleanup sentinel %s: %v", rel, err)
		}
	}
}

func directBuildPreflightProtectedSnapshot(t *testing.T) map[string]fileFingerprint {
	t.Helper()
	snapshot := snapshotProjectDataTree(t, store.BasePath())
	delete(snapshot, preflightCachePathRel)
	return snapshot
}

func readDirectBuildPreflightStateEvidence(t *testing.T) ([]byte, []string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read COLONY_STATE.json: %v", err)
	}
	var eventEnvelope struct {
		Events []string `json:"events"`
	}
	if err := json.Unmarshal(raw, &eventEnvelope); err != nil {
		t.Fatalf("decode persisted event slice: %v", err)
	}
	return raw, eventEnvelope.Events
}

func setupDirectBuildPreflightFixture(t *testing.T, name string) directBuildPreflightFixture {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Provider readiness must precede every build mutation"
	taskOneID := "1.1"
	fixture := directBuildPreflightFixture{root: root, phase: 1}

	switch name {
	case "ordinary", "force_active_attempt":
		state := readyDirectBuildPreflightState(&goal, 1, &taskOneID)
		if name == "force_active_attempt" {
			startedAt := time.Now().UTC().Add(-time.Hour)
			state.State = colony.StateEXECUTING
			state.CurrentPhase = 1
			state.BuildStartedAt = &startedAt
			state.Plan.Phases[0].Status = colony.PhaseInProgress
			state.Plan.Phases[0].Tasks[0].Status = colony.TaskInProgress
			createTestColonyState(t, dataDir, state)
			attemptRel, err := beginBuildAttempt(
				state, 1, state.Plan.Phases[0], startedAt, nil,
				"checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json",
				"go-runtime", []codexBuildDispatch{{Name: "Forge-active", Caste: "builder", TaskID: taskOneID}},
			)
			if err != nil {
				t.Fatalf("seed active build attempt: %v", err)
			}
			if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "fixture worker is active", nil, nil, "real", nil); err != nil {
				t.Fatalf("mark fixture attempt dispatching: %v", err)
			}
			fixture.options.Force = true
			fixture.attemptRel = attemptRel
		} else {
			createTestColonyState(t, dataDir, state)
		}

	case "legacy_numeric_current_phase":
		raw := []byte(`{
  "version": "3.0",
  "goal": "Provider readiness must precede legacy repair",
  "state": "READY",
  "current_phase": "1",
  "colony_depth": "light",
  "plan": {"phases": [{"id": 1, "name": "Legacy phase", "status": "ready", "tasks": [{"id": "1.1", "goal": "Make one provider-backed change", "status": "pending"}]}]},
  "events": ["2026-09-01T10:00:00Z|initialized|init|Legacy fixture ready"]
}`)
		if err := store.SaveRawJSON("COLONY_STATE.json", raw); err != nil {
			t.Fatalf("seed raw legacy colony state: %v", err)
		}

	case "missing_plan_recovery":
		fixture.phase = 3
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 2,
			ColonyDepth:  "light",
			Plan:         colony.Plan{Phases: []colony.Phase{}},
			Events: []string{
				"2026-04-21T07:50:00Z phase-1-complete: Audit complete.",
				"2026-04-21T08:20:00Z phase-2-complete: Design complete.",
			},
		})
		if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
			Confidence: codexPlanConfidence{Overall: 88},
			Phases: []codexWorkerPlanPhase{
				{Name: "Audit", Tasks: []codexWorkerPlanTask{{Goal: "Audit the current runtime"}}},
				{Name: "Design", Tasks: []codexWorkerPlanTask{{Goal: "Design the readiness boundary"}}},
				{Name: "Implement", Tasks: []codexWorkerPlanTask{{Goal: "Move readiness ahead of writes"}}},
			},
		}); err != nil {
			t.Fatalf("seed recoverable planning artifact: %v", err)
		}

	case "trusted_manifest_prior_task_reconciliation":
		fixture.phase = 2
		phaseTwoID := "2.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			CurrentPhase: 2,
			ColonyDepth:  "light",
			Plan: colony.Plan{Phases: []colony.Phase{
				{ID: 1, Name: "Completed prior phase", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskOneID, Goal: "Finish prior work", Status: colony.TaskPending}}},
				{ID: 2, Name: "Current phase", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &phaseTwoID, Goal: "Start after reconciliation", Status: colony.TaskPending}}},
			}},
			Events: []string{"2026-09-01T10:00:00Z|phase_completed|continue|Phase 1 completed"},
		})
		now := time.Now().UTC()
		if err := store.SaveJSON("build/phase-1/manifest.json", codexBuildManifest{
			Phase:        1,
			PhaseName:    "Completed prior phase",
			Goal:         goal,
			Root:         root,
			ColonyDepth:  "light",
			DispatchMode: "external-task",
			GeneratedAt:  now.Format(time.RFC3339),
			State:        string(colony.StateBUILT),
			ClaimsPath:   displayDataPath("last-build-claims.json"),
			Tasks:        []codexBuildTaskPlan{{ID: taskOneID, Goal: "Finish prior work", Status: colony.TaskCompleted}},
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-prior", Task: "Finish prior work", Status: "completed", TaskID: taskOneID, Outputs: []string{"main.go"}},
				{Stage: "verification", Caste: "watcher", Name: "Keen-prior", Task: "Verify prior phase", Status: "completed", Outputs: []string{"main_test.go"}},
			},
		}); err != nil {
			t.Fatalf("seed trusted prior manifest: %v", err)
		}
		if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
			FilesModified: []string{"main.go"},
			BuildPhase:    1,
			Timestamp:     now.Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("seed trusted prior claims: %v", err)
		}

	default:
		t.Fatalf("unknown direct-build preflight fixture %q", name)
	}

	seedDirectBuildPreflightCleanupSentinels(t, fixture.phase)
	return fixture
}

func TestDirectBuildPreflightFailureMutatesNothing(t *testing.T) {
	for _, name := range []string{
		"ordinary",
		"force_active_attempt",
		"legacy_numeric_current_phase",
		"missing_plan_recovery",
		"trusted_manifest_prior_task_reconciliation",
	} {
		t.Run(name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			fixture := setupDirectBuildPreflightFixture(t, name)

			invoker := &directBuildPreflightInvoker{available: false}
			factoryCalls := 0
			originalInvoker := newCodexWorkerInvoker
			newCodexWorkerInvoker = func() codex.WorkerInvoker {
				factoryCalls++
				return invoker
			}
			t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

			beforeTree := directBuildPreflightProtectedSnapshot(t)
			beforeState, beforeEvents := readDirectBuildPreflightStateEvidence(t)
			var beforeAttempt []byte
			if fixture.attemptRel != "" {
				var err error
				beforeAttempt, err = os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(fixture.attemptRel)))
				if err != nil {
					t.Fatalf("read active attempt before build: %v", err)
				}
			}

			_, err := runCodexBuildWithOptions(fixture.root, fixture.phase, nil, false, fixture.options)
			if !isWorkerProviderPreflightError(err) {
				t.Fatalf("runCodexBuildWithOptions error = %v (%T), want workerProviderPreflightError", err, err)
			}
			if factoryCalls != 1 {
				t.Fatalf("newCodexWorkerInvoker called %d times, want exactly 1", factoryCalls)
			}
			if invoker.preflightCalls != 1 {
				t.Fatalf("Preflight called %d times, want exactly 1 failed readiness probe", invoker.preflightCalls)
			}
			if invoker.invokeCalls != 0 {
				t.Fatalf("Invoke called %d times after failed readiness, want 0", invoker.invokeCalls)
			}

			afterTree := directBuildPreflightProtectedSnapshot(t)
			afterState, afterEvents := readDirectBuildPreflightStateEvidence(t)
			if !reflect.DeepEqual(beforeTree, afterTree) {
				t.Fatalf("failed readiness changed the protected data tree\nbefore: %#v\nafter:  %#v", beforeTree, afterTree)
			}
			if !bytes.Equal(beforeState, afterState) {
				t.Fatalf("failed readiness changed COLONY_STATE.json bytes\nbefore:\n%s\nafter:\n%s", beforeState, afterState)
			}
			if !reflect.DeepEqual(beforeEvents, afterEvents) {
				t.Fatalf("failed readiness changed persisted events\nbefore: %v\nafter:  %v", beforeEvents, afterEvents)
			}
			for _, repairEvent := range []string{"state_repaired|load", "plan_recovered|state", "phase_tasks_repaired|build"} {
				if strings.Contains(strings.Join(afterEvents, "\n"), repairEvent) {
					t.Fatalf("failed readiness persisted forbidden repair event %q: %v", repairEvent, afterEvents)
				}
			}
			if fixture.attemptRel != "" {
				afterAttempt, readErr := os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(fixture.attemptRel)))
				if readErr != nil {
					t.Fatalf("read active attempt after build: %v", readErr)
				}
				if !bytes.Equal(beforeAttempt, afterAttempt) {
					t.Fatalf("Force:true interrupted or rewrote the active attempt before readiness\nbefore:\n%s\nafter:\n%s", beforeAttempt, afterAttempt)
				}
			}
		})
	}
}

func TestDirectBuildPreflightSuccessUsesOneProbeAcrossBothDefenses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	fixture := setupDirectBuildPreflightFixture(t, "ordinary")

	invoker := &directBuildPreflightInvoker{available: true}
	factoryCalls := 0
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		factoryCalls++
		return invoker
	}
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuildWithOptions(fixture.root, fixture.phase, nil, false, fixture.options); err != nil {
		t.Fatalf("runCodexBuildWithOptions returned error after successful readiness: %v", err)
	}
	if factoryCalls != 1 {
		t.Fatalf("newCodexWorkerInvoker called %d times, want exactly 1 shared invoker", factoryCalls)
	}
	if invoker.preflightCalls != 1 {
		t.Fatalf("provider Preflight called %d times, want one real probe shared by early and late defenses", invoker.preflightCalls)
	}
	if invoker.invokeCalls < 1 {
		t.Fatalf("Invoke called %d times, want at least one worker invocation", invoker.invokeCalls)
	}
}

func TestDirectBuildPreflightSourceOrderRetainsLateDispatchDefense(t *testing.T) {
	source, err := os.ReadFile("codex_build.go")
	if err != nil {
		t.Fatalf("read codex_build.go: %v", err)
	}
	text := string(source)
	runStart := strings.Index(text, "func runCodexBuildWithOptions(")
	runEnd := strings.Index(text, "func validateCodexBuildState(")
	if runStart < 0 || runEnd <= runStart {
		t.Fatal("could not isolate runCodexBuildWithOptions source")
	}
	runBody := text[runStart:runEnd]

	ordered := []string{
		"loadActiveColonyStateReadOnly()",
		"applyPriorCompletedPhaseTaskRepairs(",
		"preflightWorkerProvider(",
		"loadActiveColonyState()",
		"reconcilePriorCompletedPhaseTasksFromTrustedManifests(",
		"interruptLatestBuildAttempt(",
		"beginRuntimeSpawnRun(",
		"cleanupStaleBuildAttemptArtifacts(",
	}
	last := -1
	for _, call := range ordered {
		idx := strings.Index(runBody, call)
		if idx < 0 {
			t.Fatalf("runCodexBuildWithOptions is missing ordered call %q", call)
		}
		if idx <= last {
			t.Fatalf("runCodexBuildWithOptions call %q is out of order; required sequence is %v", call, ordered)
		}
		last = idx
	}

	executeStart := strings.Index(text, "func executeCodexBuildDispatches(")
	executeEnd := strings.Index(text, "func validateRuntimeNoChangeEvidence(")
	if executeStart < 0 || executeEnd <= executeStart {
		t.Fatal("could not isolate executeCodexBuildDispatches source")
	}
	executeBody := text[executeStart:executeEnd]
	if !strings.Contains(executeBody, "preflightWorkerProvider(ctx, invoker, workerDispatches)") {
		t.Fatal("executeCodexBuildDispatches lost the late full-dispatch preflight defense")
	}
}
