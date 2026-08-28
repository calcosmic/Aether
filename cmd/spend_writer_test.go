package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// loadSpendLedgerFromDisk reads a phase's workflow ledger back off the disk,
// decoding the file's own bytes rather than asking the code under test what it
// thinks it wrote. A test that only inspects a writer's return value proves
// nothing about whether anything was filed.
func loadSpendLedgerFromDisk(t *testing.T, dataDir string, phase int, workflow string) spendLedger {
	t.Helper()
	path := filepath.Join(dataDir, "spend", "phase-"+itoaForSpendTest(phase)+"-"+workflow+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger %s: %v", path, err)
	}
	var ledger spendLedger
	if err := json.Unmarshal(raw, &ledger); err != nil {
		t.Fatalf("decode ledger %s: %v", path, err)
	}
	return ledger
}

func itoaForSpendTest(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func spendLedgerRowByName(t *testing.T, ledger spendLedger, name string) spendRow {
	t.Helper()
	for _, row := range ledger.Rows {
		if row.AgentName == name {
			return row
		}
	}
	t.Fatalf("worker %q has no row in the saved ledger; a worker that vanishes makes a run look cheaper than it was: %+v", name, ledger.Rows)
	return spendRow{}
}

// newSpendWriterFixture gives the writer a real store, a real OpenCode session
// store holding this repository's sessions, and the dispatches of a finished
// run. Mason-67 is measurable from the session store; Roam-90 is not, because
// the platform recorded nothing for it.
func newSpendWriterFixture(t *testing.T) (dataDir string, req spendWriteRequest) {
	t.Helper()
	s, tmpDir := newTestStore(t)
	store = s
	// newTestStore returns the repo-shaped temp root; the store itself lives
	// one level down, which is where the ledger file must appear.
	dataDir = filepath.Join(tmpDir, ".aether", "data")
	_, repoRoot := setupOpenCodeFixtureHome(t)

	return dataDir, spendWriteRequest{
		Phase:     7,
		PhaseName: "See what it cost",
		Workflow:  spendWorkflowBuild,
		RepoRoot:  repoRoot,
		Platform:  "opencode",
		StartedAt: openCodeFixtureWindowStart,
		EndedAt:   openCodeFixtureWindowEnd,
		// Name is the deterministic per-worker name the platform's own session
		// title carries ("🔨 Builder Mason-67: ..."); AgentName is the agent
		// definition it ran as. The ledger accounts a worker under the former.
		Dispatches: []codexBuildDispatch{
			{Caste: "builder", AgentName: "aether-builder", Name: "Mason-67", Task: "write the parser", Status: "completed"},
			{Caste: "scout", AgentName: "aether-scout", Name: "Roam-90", Task: "research the store", Status: "completed"},
		},
	}
}

func TestBuildFinalizeWritesOneRowPerWorker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, req := newSpendWriterFixture(t)

	if _, err := writeSpendRowsForRun(req); err != nil {
		t.Fatalf("writeSpendRowsForRun: %v", err)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 7, spendWorkflowBuild)
	if len(ledger.Rows) != 2 {
		t.Fatalf("got %d rows on disk, want one per worker (2): %+v", len(ledger.Rows), ledger.Rows)
	}
	if ledger.Phase != 7 || ledger.Workflow != spendWorkflowBuild {
		t.Errorf("ledger keyed as phase %d / %q, want 7 / %q", ledger.Phase, ledger.Workflow, spendWorkflowBuild)
	}
	if ledger.SchemaVersion != spendLedgerSchemaVersion {
		t.Errorf("schema version = %d, want %d", ledger.SchemaVersion, spendLedgerSchemaVersion)
	}

	mason := spendLedgerRowByName(t, ledger, "Mason-67")
	if mason.Caste != "builder" || mason.Status != "completed" {
		t.Errorf("Mason-67 row = %+v, want caste builder and status completed", mason)
	}
	// The hand-summed two-message total from cmd/testdata/spend/opencode/README.md.
	if got := mason.Usage.BilledTotalTokens(); got != openCodeFixtureSumTotal {
		t.Errorf("Mason-67 billed total on disk = %d, want %d", got, openCodeFixtureSumTotal)
	}

	roam := spendLedgerRowByName(t, ledger, "Roam-90")
	if roam.Status != "completed" {
		t.Errorf("Roam-90 row status = %q, want completed", roam.Status)
	}
}

func TestGroupedWorkerRowCarriesItsJobName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, req := newSpendWriterFixture(t)

	// Phase 195 made one worker able to own a chain of tasks. The ledger must
	// still be able to say which job the tokens belong to.
	req.Dispatches[0].JobName = "parser-and-its-tests"
	req.Dispatches[0].CoveredTaskIDs = []string{"1.1", "1.2", "1.3"}

	if _, err := writeSpendRowsForRun(req); err != nil {
		t.Fatalf("writeSpendRowsForRun: %v", err)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 7, spendWorkflowBuild)
	mason := spendLedgerRowByName(t, ledger, "Mason-67")
	if mason.JobName != "parser-and-its-tests" {
		t.Errorf("Mason-67 job name on disk = %q, want %q", mason.JobName, "parser-and-its-tests")
	}

	roam := spendLedgerRowByName(t, ledger, "Roam-90")
	if roam.JobName != "" {
		t.Errorf("Roam-90 owned no grouped job but records job name %q", roam.JobName)
	}

	// An ungrouped worker serializes no job_name key at all rather than an
	// empty placeholder.
	path := filepath.Join(dataDir, "spend", "phase-7-build.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	var decoded struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode ledger rows: %v", err)
	}
	for _, row := range decoded.Rows {
		var name string
		if err := json.Unmarshal(row["name"], &name); err != nil {
			t.Fatalf("decode row name: %v", err)
		}
		_, present := row["job_name"]
		if name == "Roam-90" && present {
			t.Errorf("Roam-90's serialized row carries a job_name key; an ungrouped worker records none")
		}
		if name == "Mason-67" && !present {
			t.Errorf("Mason-67's serialized row carries no job_name key")
		}
	}
}

func TestUnreportedWorkerRowHoldsNoFigure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, req := newSpendWriterFixture(t)

	if _, err := writeSpendRowsForRun(req); err != nil {
		t.Fatalf("writeSpendRowsForRun: %v", err)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 7, spendWorkflowBuild)
	roam := spendLedgerRowByName(t, ledger, "Roam-90")

	if roam.Usage.Source != "" {
		t.Errorf("Roam-90's saved row carries source %q; the platform reported nothing for it", roam.Usage.Source)
	}
	if roam.Usage.BilledTotalTokens() != 0 || roam.Usage.InputTokens != 0 || roam.Usage.OutputTokens != 0 ||
		roam.Usage.CachedInputTokens != 0 || roam.Usage.CacheCreationTokens != 0 || roam.Usage.TotalTokens != 0 {
		t.Errorf("Roam-90's saved row carries a token figure %+v; D-01 as amended gives an unreported worker no number at all", roam.Usage)
	}

	// The row still exists, which is the other half of the rule.
	if len(ledger.Rows) != 2 {
		t.Errorf("got %d rows, want 2 -- an unreported worker keeps its row", len(ledger.Rows))
	}

	// And a reported worker is distinguishable from it WITHOUT reading a
	// number: the source tag is the signal.
	mason := spendLedgerRowByName(t, ledger, "Mason-67")
	if mason.Usage.Source == "" {
		t.Errorf("Mason-67's saved row carries no source tag, so nothing can tell its real measurement apart from an absent one")
	}
}

func TestBuildFinalizeRerunReplacesItsOwnRowsOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, req := newSpendWriterFixture(t)

	// A continue-keyed ledger for the same phase already exists. Writing the
	// build ledger must not read, rewrite or delete it.
	continueLedger := spendLedger{
		Phase:      7,
		Workflow:   spendWorkflowContinue,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Rows:       []spendRow{{AgentName: "Keen-11", Caste: "watcher", Status: "completed"}},
	}
	if err := saveSpendLedger(continueLedger); err != nil {
		t.Fatalf("seed continue ledger: %v", err)
	}
	continuePath := filepath.Join(dataDir, "spend", "phase-7-continue.json")
	before, err := os.ReadFile(continuePath)
	if err != nil {
		t.Fatalf("read seeded continue ledger: %v", err)
	}

	if _, err := writeSpendRowsForRun(req); err != nil {
		t.Fatalf("first writeSpendRowsForRun: %v", err)
	}
	if _, err := writeSpendRowsForRun(req); err != nil {
		t.Fatalf("second writeSpendRowsForRun: %v", err)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 7, spendWorkflowBuild)
	if len(ledger.Rows) != 2 {
		t.Fatalf("got %d rows after two finalizes, want 2 -- a rerun replaces this build's rows, it does not append them: %+v", len(ledger.Rows), ledger.Rows)
	}
	mason := spendLedgerRowByName(t, ledger, "Mason-67")
	if got := mason.Usage.BilledTotalTokens(); got != openCodeFixtureSumTotal {
		t.Errorf("Mason-67 billed total after two finalizes = %d, want %d -- a doubled figure means the rerun accumulated", got, openCodeFixtureSumTotal)
	}

	after, err := os.ReadFile(continuePath)
	if err != nil {
		t.Fatalf("read continue ledger after the build write: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("writing the build ledger changed the continue-keyed file:\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestLedgerWriteFailureDoesNotFailTheBuild(t *testing.T) {
	t.Run("the writer surfaces a save failure as an error", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		_, req := newSpendWriterFixture(t)
		req.Workflow = "audit" // not one of the two keyed workflows

		if _, err := writeSpendRowsForRun(req); err == nil {
			t.Fatalf("expected an error for an unknown workflow, got none")
		}
	})

	t.Run("a build whose accounting cannot be filed still finishes", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		forceBuildJSONOutput(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		defer func() { _ = os.Chdir(oldDir) }()

		goal := "Prove accounting never gates a build"
		taskID := "1.1"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ColonyDepth:  "standard",
			CurrentPhase: 0,
			Plan: colony.Plan{
				Phases: []colony.Phase{{
					ID:          1,
					Name:        "Accounting is a record, not a gate",
					Description: "A ledger write that fails must not lose the build",
					Status:      colony.PhaseReady,
					Tasks:       []colony.Task{{ID: &taskID, Goal: "Create evidence", Status: colony.TaskPending}},
				}},
			},
		})

		// Make the ledger's own directory impossible to create: a regular file
		// sits exactly where "spend/" must be, so every save under it fails.
		if err := os.WriteFile(filepath.Join(dataDir, "spend"), []byte("not a directory\n"), 0o644); err != nil {
			t.Fatalf("block the spend directory: %v", err)
		}

		completionPath := runBuildToCompletionPacketForSpendTest(t, root)
		rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build-finalize failed because its accounting could not be filed; a build must not be lost because of its bookkeeping: %v", err)
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
			t.Fatalf("parse finalize output: %v\n%s", err, stdout.(*bytes.Buffer).String())
		}
		if envelope["ok"] != true {
			t.Fatalf("expected ok:true despite the failed ledger write, got %v", envelope)
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("reload state: %v", err)
		}
		if state.State != colony.StateBUILT {
			t.Fatalf("state = %s, want BUILT -- the build itself must have completed", state.State)
		}

		result, ok := envelope["result"].(map[string]interface{})
		if !ok {
			t.Fatalf("finalize result missing: %v", envelope)
		}
		if _, reported := result["spend_ledger_note"]; !reported {
			t.Errorf("the failed ledger write was silent; it must be reported even though it is not fatal: %v", result)
		}
	})
}

// TestBuildFinalizeFilesTheRunsRows is the wiring proof: a real build-finalize
// run leaves per-worker rows on disk. Without the call in the finalize path
// this plan would have merged two readers nothing invokes, which is the
// condition the salvage assessment explicitly refuses.
func TestBuildFinalizeFilesTheRunsRows(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	goal := "Leave an honest record of what the build cost"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "Spend rows on disk",
				Description: "A finished build files one row per worker",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Create evidence", Status: colony.TaskPending}},
			}},
		},
	})

	completionPath := runBuildToCompletionPacketForSpendTest(t, root)
	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build-finalize returned error: %v", err)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 1, spendWorkflowBuild)
	if len(ledger.Rows) == 0 {
		t.Fatalf("the finished build filed no ledger rows at all; both platform readers would still be unreachable")
	}
	for _, row := range ledger.Rows {
		if row.AgentName == "" {
			t.Errorf("a filed row names no worker: %+v", row)
		}
		if row.Status == "" {
			t.Errorf("worker %s's filed row states no outcome", row.AgentName)
		}
	}

	// The continue-keyed file for the same phase must not exist: a build write
	// touches only the build key.
	if _, err := os.Stat(filepath.Join(dataDir, "spend", "phase-1-continue.json")); !os.IsNotExist(err) {
		t.Errorf("finalizing a build created or touched the continue-keyed ledger file")
	}
}

// runBuildToCompletionPacketForSpendTest plans a build, marks every dispatch
// completed with the evidence the finalizer requires, and writes the completion
// packet. It returns the packet's path.
func runBuildToCompletionPacketForSpendTest(t *testing.T, root string) string {
	t.Helper()

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if err := os.WriteFile(filepath.Join(root, "wrapper-evidence.txt"), []byte("external work\n"), 0o644); err != nil {
		t.Fatalf("write claimed file: %v", err)
	}
	writeClaimFileForTest(t, root, "cmd/main.go")

	dispatchResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed externally",
			Duration:      1.25,
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"work complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesCreated = []string{"wrapper-evidence.txt"}
			worker.FilesModified = []string{"cmd/main.go"}
			worker.TestsWritten = []string{"wrapper-evidence.txt"}
		}
		dispatchResults = append(dispatchResults, worker)
	}

	completion := codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       dispatchResults,
	}
	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "completion.json")
	if err := os.WriteFile(completionPath, completionData, 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	return completionPath
}
