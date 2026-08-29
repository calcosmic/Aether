package cmd

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Task 1 (198.1-01): a failed build worker's own words reach the next build
// brief, on the wrapper-driven (delegate) build lane.
// ---------------------------------------------------------------------------

const testFailedWorkerSentence = "go test ./cmd/ -run TestExpireSignals failed: nil pointer dereference in expireSignalsByType"

// buildFailedDelegateCompletion constructs a real dispatch manifest (via
// runCodexBuildPlanOnlyWithOptions, D-04), then a completion packet where one
// dispatch succeeds with genuine file-modification evidence (so the
// phantom-build provenance guard is satisfied) and one dispatch fails,
// carrying sentence as its first blocker.
func buildFailedDelegateCompletion(t *testing.T, root string, sentence string) (codexBuildManifest, []codexExternalBuildWorkerResult) {
	t.Helper()
	manifest, _ := prepareExternalBuildCompletionWithProposal(t, root,
		[]string{"builder", "architect"}, []string{"architect=needs a design review before this lands"})
	if len(manifest.Dispatches) < 2 {
		t.Fatalf("fixture must produce at least 2 dispatches, got %d", len(manifest.Dispatches))
	}
	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}

	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		if dispatch.Caste == "builder" {
			results = append(results, codexExternalBuildWorkerResult{
				Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
				Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
				Status:        "completed",
				Summary:       dispatch.Name + " completed",
				FilesModified: []string{"external-evidence.txt"},
				Handoff: codex.WorkerHandoff{
					CommandsRun:            []string{"go test ./..."},
					VerificationStatus:     "pass",
					NextWorkerInstructions: []string{dispatch.Name + " work is complete"},
				},
			})
			continue
		}
		results = append(results, codexExternalBuildWorkerResult{
			Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
			Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
			Status:   "failed",
			Summary:  "worker hit a blocking failure",
			Blockers: []string{sentence},
		})
	}
	return manifest, results
}

// loadedCompletionFromManifest decodes a completion packet through the real
// runtime parser (loadExternalBuildCompletion), rather than constructing a
// codexExternalBuildCompletion struct literal (D-04, plan 198.1-01).
func loadedCompletionFromManifest(t *testing.T, manifest codexBuildManifest, results []codexExternalBuildWorkerResult) codexExternalBuildCompletion {
	t.Helper()
	raw, err := json.Marshal(map[string]interface{}{
		"dispatch_manifest": manifest,
		"dispatches":        results,
	})
	if err != nil {
		t.Fatalf("marshal completion packet: %v", err)
	}
	path := filepath.Join(t.TempDir(), "completion.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write completion packet: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("loadExternalBuildCompletion: %v", err)
	}
	return completion
}

func TestFailedBuildWorkerReachesTheNextBriefOnTheDelegateLane(t *testing.T) {
	root := setupExternalBuildAttemptTestWithVerifiableWork(t)
	manifest, results := buildFailedDelegateCompletion(t, root, testFailedWorkerSentence)
	completion := loadedCompletionFromManifest(t, manifest, results)

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("build-finalize: %v", err)
	}

	mf, err := loadMiddenFile(store)
	if err != nil {
		t.Fatalf("load midden after finalize: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	if !strings.HasPrefix(mf.Entries[0].Message, testFailedWorkerSentence) {
		t.Fatalf("midden entry message = %q, want it to start with %q", mf.Entries[0].Message, testFailedWorkerSentence)
	}

	// (b) Running the same completion input again must not grow the store.
	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("second build-finalize with the identical completion input: %v", err)
	}
	mf, err = loadMiddenFile(store)
	if err != nil {
		t.Fatalf("load midden after repeat finalize: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry after repeating the same completion input, got %d: %+v", len(mf.Entries), mf.Entries)
	}

	// (c) composeBuildManifestBrief for a dispatch on the same colony
	// carries the worker's own failure sentence verbatim.
	phase := colony.Phase{ID: 1, Name: "External attempt", Description: "Bind one wrapper dispatch to one lifecycle commit"}
	nextDispatch := codexBuildDispatch{Caste: "builder", Name: "Verify-1", Task: "Independent verification"}
	brief := composeBuildManifestBrief(root, phase, nextDispatch, time.Now(), true)
	if !strings.Contains(brief, testFailedWorkerSentence) {
		t.Fatalf("composeBuildManifestBrief did not carry the failed worker's own sentence verbatim:\n%s", brief)
	}
}

func TestSuccessfulBuildWorkerWritesNoFailureRecord(t *testing.T) {
	for _, status := range []string{"completed", "completed_no_change", "in_progress"} {
		t.Run(status, func(t *testing.T) {
			s, tmpDir := newTestStore(t)
			defer os.RemoveAll(tmpDir)
			store = s

			dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: 1}
			var wr *codex.WorkerResult
			switch status {
			case "completed_no_change":
				wr = &codex.WorkerResult{WorkerName: "Mason-1", Caste: "builder", Status: status, Summary: "nothing to change"}
			case "in_progress":
				wr = nil
			default:
				wr = &codex.WorkerResult{WorkerName: "Mason-1", Caste: "builder", Status: status, Summary: "done"}
			}
			result := codex.DispatchResult{WorkerName: "Mason-1", Status: status, WorkerResult: wr}

			if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
				t.Fatalf("recordDispatchWorkerOutcome: %v", err)
			}

			mf, err := loadMiddenFile(s)
			if err == nil && len(mf.Entries) != 0 {
				t.Fatalf("status %q should write no midden entry, got %+v", status, mf.Entries)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 2 (198.1-01): a build worker's lessons become observations.
// ---------------------------------------------------------------------------

func TestBuildWorkerLessonsBecomeObservations(t *testing.T) {
	root := setupExternalBuildAttemptTestWithVerifiableWork(t)
	manifest, _ := prepareExternalBuildCompletionWithProposal(t, root,
		[]string{"builder", "architect"}, []string{"architect=needs a design review before this lands"})
	if len(manifest.Dispatches) < 1 {
		t.Fatal("fixture must produce at least 1 dispatch")
	}
	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}

	const doNotRepeat = "never edit cmd/codex_build.go dispatch loop without running go test ./cmd/ -run TestBuildWaveExecutionPlan"
	const nextWorker = "read cmd/deterministic_floor.go before touching any verification step; both continue lanes share it"

	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		wr := codexExternalBuildWorkerResult{
			Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch),
			Caste: dispatch.Caste, Name: dispatch.Name, TaskID: dispatch.TaskID,
			Status:  "completed",
			Summary: dispatch.Name + " completed",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{nextWorker},
				DoNotRepeat:            []string{doNotRepeat},
			},
		}
		if dispatch.Caste == "builder" {
			wr.FilesModified = []string{"external-evidence.txt"}
		}
		results = append(results, wr)
	}
	completion := loadedCompletionFromManifest(t, manifest, results)

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("build-finalize: %v", err)
	}

	var file colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &file); err != nil {
		t.Fatalf("load learning-observations.json: %v", err)
	}
	byContent := map[string]colony.Observation{}
	for _, obs := range file.Observations {
		byContent[obs.Content] = obs
	}
	redirect, ok := byContent[doNotRepeat]
	if !ok {
		t.Fatalf("expected an observation with content %q, got %+v", doNotRepeat, file.Observations)
	}
	if redirect.WisdomType != "redirect" {
		t.Errorf("do_not_repeat observation wisdom_type = %q, want %q", redirect.WisdomType, "redirect")
	}
	pattern, ok := byContent[nextWorker]
	if !ok {
		t.Fatalf("expected an observation with content %q, got %+v", nextWorker, file.Observations)
	}
	if pattern.WisdomType != "pattern" {
		t.Errorf("next_worker_instructions observation wisdom_type = %q, want %q", pattern.WisdomType, "pattern")
	}
}

func TestInadmissibleWorkerSentenceIsRejectedNotStored(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ok, reason := captureWorkerObservation("we did a good job today", "pattern", "success_pattern", "single_phase")
	if ok {
		t.Fatal("expected an inadmissible sentence to be rejected")
	}
	if strings.TrimSpace(reason) == "" {
		t.Fatal("expected a non-empty rejection reason")
	}

	var file colony.LearningFile
	loadErr := s.LoadJSON("learning-observations.json", &file)
	if loadErr == nil && len(file.Observations) != 0 {
		t.Fatalf("expected zero observations, got %d", len(file.Observations))
	}
}

func TestObservationSourceTypeIsAKnownTrustWeight(t *testing.T) {
	for _, succeeded := range []bool{true, false} {
		sourceType := observationSourceTypeForOutcome(succeeded)
		result := memory.Calculate(memory.TrustInput{SourceType: sourceType, Evidence: "single_phase", DaysSince: 0})
		if result.Score <= 0.50 {
			t.Errorf("observationSourceTypeForOutcome(%v) = %q, trust score %v via memory.Calculate is not above 0.50", succeeded, sourceType, result.Score)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 3 (198.1-01): one boundary, both lanes, proved structurally.
// ---------------------------------------------------------------------------

// TestEveryBuildLaneFeedsMemoryThroughOneBoundary is scoped to the build
// lanes this plan wires: cmd/memory_feed.go (the boundary itself), the
// delegate lane's cmd/codex_build_finalize.go, and the native lane's
// cmd/codex_build.go. It deliberately does NOT scan the whole cmd package --
// cmd/codex_continue_finalize.go carries a pre-existing call to
// persistDispatchWorkerHandoff on the continue lane, which is out of scope
// for this plan (198.1-CONTEXT.md scope_fences: this plan is the build half
// of FEED-01/FEED-02; continue's own memory feed is a later plan in this
// phase). Scanning the whole package would fail this guard against work this
// plan never touched.
func TestEveryBuildLaneFeedsMemoryThroughOneBoundary(t *testing.T) {
	const allowedCaller = "recordDispatchWorkerOutcome"
	files := []string{"memory_feed.go", "codex_build.go", "codex_build_finalize.go"}

	fset := token.NewFileSet()
	for _, name := range files {
		path := filepath.Join(".", name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcName := fn.Name.Name
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if !ok || ident.Name != "persistDispatchWorkerHandoff" {
					return true
				}
				if funcName != allowedCaller {
					t.Errorf("%s:%d: %s calls persistDispatchWorkerHandoff directly; only %s may persist a handoff without also feeding memory (cmd/memory_feed.go is the one boundary both build lanes must go through)",
						name, fset.Position(call.Pos()).Line, funcName, allowedCaller)
				}
				return true
			})
		}
	}
}

// singleFailingInvoker is a minimal codex.WorkerInvoker that always reports a
// failed status carrying a fixed blocker sentence, for the native (in-process)
// build lane test below.
type singleFailingInvoker struct{ sentence string }

func (i *singleFailingInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "failed",
		Summary:    "worker hit a blocking failure",
		Blockers:   []string{i.sentence},
		Duration:   time.Millisecond,
	}, nil
}

func (i *singleFailingInvoker) IsAvailable(ctx context.Context) bool { return true }

func (i *singleFailingInvoker) ValidateAgent(path string) error { return nil }

const testNativeFailedWorkerSentence = "go build ./cmd/aether failed: undefined identifier in cmd/memory_feed.go"

func TestNativeBuildLaneFeedsTheSameMemory(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	if len(state.Plan.Phases) == 0 {
		t.Fatal("test colony has no phase")
	}
	phase := state.Plan.Phases[0]
	dispatches := []codexBuildDispatch{{
		Caste:  "builder",
		Name:   "Mason-1",
		Task:   "Implement the boundary",
		Stage:  "build",
		Wave:   1,
		TaskID: "1.1",
	}}

	if err := recordCodexBuildDispatches(dispatches); err != nil {
		t.Fatalf("record native build dispatches: %v", err)
	}

	// A wave containing a failed worker legitimately makes
	// executeCodexBuildDispatches itself return a non-nil "did not complete
	// cleanly" error (queenWaveLifecycle) -- that is expected and orthogonal
	// to this test. The memory-feed loop (recordDispatchWorkerOutcome) runs
	// BEFORE that error is returned, so the midden write below is what this
	// test actually verifies.
	invoker := &singleFailingInvoker{sentence: testNativeFailedWorkerSentence}
	_, _, _, _ = executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), invoker, colony.ModeInRepo, 0, 3, false, nil)

	mf, err := loadMiddenFile(store)
	if err != nil {
		t.Fatalf("load midden: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	if !strings.HasPrefix(mf.Entries[0].Message, testNativeFailedWorkerSentence) {
		t.Fatalf("midden entry message = %q, want prefix %q", mf.Entries[0].Message, testNativeFailedWorkerSentence)
	}
}

func TestFeedingMemoryNeverFailsABuild(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Make the store's directory unwritable so any write inside it --
	// including both the worker-handoffs file and midden.json -- fails.
	if err := os.Chmod(dataDir, 0o500); err != nil {
		t.Fatalf("chmod data dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dataDir, 0o755) })

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: 1}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "failed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1",
			Caste:      "builder",
			Status:     "failed",
			Blockers:   []string{"go build ./cmd/aether failed: unwritable store"},
		},
	}

	wantErr := persistDispatchWorkerHandoff(dispatch, result)
	if wantErr == nil {
		t.Skip("environment did not make the store directory unwritable (e.g. running as root); nothing to assert")
	}
	gotErr := recordDispatchWorkerOutcome(dispatch, result)
	if gotErr == nil || gotErr.Error() != wantErr.Error() {
		t.Fatalf("recordDispatchWorkerOutcome error = %v, want %v (persistDispatchWorkerHandoff's own error, unchanged)", gotErr, wantErr)
	}
}
