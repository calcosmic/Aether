package cmd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanOnlyManifestCarriesJournalBoundExecutionIdentity(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if manifest.ExecutionBinding == nil {
		t.Fatal("plan-only manifest has no execution binding")
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("durable build attempt is missing")
	}
	if manifest.ExecutionBinding.RunID != record.RunID || manifest.ExecutionBinding.AttemptID != record.ID || manifest.ExecutionBinding.ManifestSHA256 != record.ManifestSHA256 {
		t.Fatalf("manifest binding does not match journal: binding=%+v record=%+v", manifest.ExecutionBinding, record)
	}
}

func TestInternalBuildAdapterReturnsCachedTerminalResultWithoutRedispatch(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	t.Chdir(root)
	dispatch := manifest.Dispatches[0]
	requestPath := writeBoundBuildWorkerRequest(t, manifest, dispatch)

	first, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true)
	if err != nil {
		t.Fatalf("first adapter run: %v", err)
	}
	second, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true)
	if err != nil {
		t.Fatalf("cached adapter run: %v", err)
	}
	if first.Worker == nil || second.Worker == nil || first.ProviderRunID == "" || second.ProviderRunID != first.ProviderRunID {
		t.Fatalf("cached response identity mismatch: first=%+v second=%+v", first, second)
	}
	_, record, _ := loadLatestBuildAttempt(1)
	if len(record.WorkerRuns) != 1 || record.WorkerRuns[0].ResultSHA256 == "" {
		t.Fatalf("terminal result was redispatched or not journaled: %+v", record.WorkerRuns)
	}
}

func TestInternalBuildAdapterStagesCompletionWhenAllWorkersAreTerminal(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	t.Chdir(root)
	for _, dispatch := range manifest.Dispatches {
		requestPath := writeBoundBuildWorkerRequest(t, manifest, dispatch)
		if _, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true); err != nil {
			t.Fatalf("dispatch %s: %v", dispatch.Name, err)
		}
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.CompletionPath == "" || record.CompletionSHA256 == "" {
		t.Fatalf("all terminal workers did not stage completion: %+v", record)
	}
	completion, err := loadExternalBuildCompletion(filepath.Join(root, filepath.FromSlash(record.CompletionPath)))
	if err != nil {
		t.Fatal(err)
	}
	if len(completion.workerResults()) != len(manifest.Dispatches) {
		t.Fatalf("staged workers = %d, want %d", len(completion.workerResults()), len(manifest.Dispatches))
	}
	recovery := buildResumeDashboardResult()["recovery"].(map[string]interface{})
	if recovery["next"] != buildFinalizeRecoveryCommand(1, record.CompletionPath) {
		t.Fatalf("resume did not point at exact staged completion: %+v", recovery)
	}
}

func TestInternalBuildAdapterRejectsStaleRunIdentity(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	t.Chdir(root)
	stale := *manifest.ExecutionBinding
	stale.RunID = "run-stale"
	request := boundBuildWorkerRequest(manifest, manifest.Dispatches[0])
	request["execution_binding"] = stale
	requestPath := writeInternalWorkerRequestForTest(t, request)
	_, err := runInternalWorkerAdapter(context.Background(), requestPath, false, true)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("stale run identity should fail closed, got %v", err)
	}
}

func TestTerminalWorkerJournalRejectsChangedWorkspaceIdentity(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	dispatch := manifest.Dispatches[0]
	request := internalWorkerDispatchRequest{
		SchemaVersion:    1,
		Workflow:         "build",
		Phase:            manifest.Phase,
		Caste:            dispatch.Caste,
		WorkerName:       dispatch.Name,
		TaskID:           normalizedDispatchTaskID(dispatch),
		ExecutionBinding: manifest.ExecutionBinding,
	}
	if _, err := beginBuildAttemptWorkerRun(manifest.Phase, *manifest.ExecutionBinding, request, "provider-workspace-change", codex.PlatformFake); err != nil {
		t.Fatalf("begin worker run: %v", err)
	}
	changed := *manifest.ExecutionBinding
	changed.WorkspaceFingerprint = strings.Repeat("c", 64)
	err := recordBuildAttemptWorkerTerminal(manifest.Phase, changed, "provider-workspace-change", &internalWorkerResult{
		Name:   dispatch.Name,
		Caste:  dispatch.Caste,
		TaskID: normalizedDispatchTaskID(dispatch),
		Status: buildWorkerCompleted,
	})
	if err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("changed workspace identity should reject terminal evidence, got %v", err)
	}
}

func TestNativeBuildDispatchJournalsEveryTerminalWorker(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Plan.Phases) == 0 {
		t.Fatal("test colony has no phase")
	}
	phase := state.Plan.Phases[0]
	if err := recordCodexBuildDispatches(manifest.Dispatches); err != nil {
		t.Fatalf("record native build dispatches: %v", err)
	}
	results, _, _, err := executeCodexBuildDispatches(
		context.Background(),
		root,
		phase,
		append([]codexBuildDispatch(nil), manifest.Dispatches...),
		time.Now().UTC(),
		&codex.FakeInvoker{},
		colony.ModeInRepo,
		0,
		3,
		false,
		manifest.ExecutionBinding,
	)
	if err != nil {
		t.Fatalf("native build dispatch: %v", err)
	}
	if len(results) != len(manifest.Dispatches) {
		t.Fatalf("native dispatch results = %d, want %d", len(results), len(manifest.Dispatches))
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("durable attempt is missing")
	}
	if len(record.WorkerRuns) != len(manifest.Dispatches) {
		t.Fatalf("journaled worker runs = %d, want %d: %+v", len(record.WorkerRuns), len(manifest.Dispatches), record.WorkerRuns)
	}
	for _, workerRun := range record.WorkerRuns {
		if workerRun.Status != buildWorkerCompleted || workerRun.Result == nil || workerRun.ResultSHA256 == "" || workerRun.ProviderRunID == "" {
			t.Fatalf("worker terminal evidence is incomplete: %+v", workerRun)
		}
	}
}

func prepareBoundBuildManifestOnly(t *testing.T, root string) codexBuildManifest {
	t.Helper()
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result["dispatch_manifest"].(codexBuildManifest)
}

func writeBoundBuildWorkerRequest(t *testing.T, manifest codexBuildManifest, dispatch codexBuildDispatch) string {
	t.Helper()
	return writeInternalWorkerRequestForTest(t, boundBuildWorkerRequest(manifest, dispatch))
}

func boundBuildWorkerRequest(manifest codexBuildManifest, dispatch codexBuildDispatch) map[string]interface{} {
	return map[string]interface{}{
		"schema_version":     1,
		"workflow":           "build",
		"phase":              manifest.Phase,
		"caste":              dispatch.Caste,
		"worker_name":        dispatch.Name,
		"task_id":            normalizedDispatchTaskID(dispatch),
		"task":               dispatch.Task,
		"task_brief":         dispatch.Task,
		"permission_profile": dispatch.PermissionProfile,
		"execution_binding":  manifest.ExecutionBinding,
		"timeout_ms":         5000,
	}
}
