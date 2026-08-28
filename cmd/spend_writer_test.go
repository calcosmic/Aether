package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// --- Phase 196 plan 07, task 1: the continue lane files its own rows ---
//
// The build lane has filed rows since plan 196-05. Until this task the
// checking pass filed nothing at all, so a phase's recorded cost was only ever
// half of what it actually cost -- and the closeout line plan 196-07 renders
// would have understated every phase that was checked, which is every phase.
//
// The keying is the mistake the salvaged branch's own comments record making
// and correcting once already: one file per phase per workflow, each replaced
// only by its own workflow, so a check can never erase what the build spent.

// continueTerminalResultsForPlan answers every planned reviewer and watcher
// with a terminal result the finalizer accepts: a completed status and a
// non-empty handoff, which the continue brief promises is required.
func continueTerminalResultsForPlan(plan codexContinuePlanManifest) []codexContinueExternalDispatch {
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
			Summary: dispatch.Name + " checked the work and found nothing to flag",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"nothing outstanding"},
			},
		})
	}
	return results
}

// planAndFinalizeContinueForSpendTest drives the real plan-only + finalize pair
// the heavy-review wrapper drives, and returns the manifest it answered.
func planAndFinalizeContinueForSpendTest(t *testing.T, root string, opts codexContinueOptions) codexContinuePlanManifest {
	t.Helper()
	planResult, _, _, _, err := runCodexContinuePlanOnly(root, opts)
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	if _, _, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
		ContinueManifest: &plan,
		Dispatches:       continueTerminalResultsForPlan(plan),
	}, false, 0, false); err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	return plan
}

func TestContinueFinalizeWritesItsOwnRows(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	root, dataDir, _, _ := setupIntermediateContinueState(t, "The check files its own record")

	plan := planAndFinalizeContinueForSpendTest(t, root, codexContinueOptions{LightFlag: true})
	if len(plan.Dispatches) == 0 {
		t.Fatalf("the planned check ran no workers, so this test would prove nothing")
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 1, spendWorkflowContinue)
	if ledger.Workflow != spendWorkflowContinue {
		t.Errorf("ledger keyed as %q, want %q", ledger.Workflow, spendWorkflowContinue)
	}
	if len(ledger.Rows) != len(plan.Dispatches) {
		t.Fatalf("got %d rows on disk, want one per worker (%d): %+v", len(ledger.Rows), len(plan.Dispatches), ledger.Rows)
	}
	for _, dispatch := range plan.Dispatches {
		row := spendLedgerRowByName(t, ledger, dispatch.Name)
		if row.Status == "" {
			t.Errorf("worker %s's filed row states no outcome", dispatch.Name)
		}
		if row.Caste != dispatch.Caste {
			t.Errorf("worker %s filed under caste %q, want %q", dispatch.Name, row.Caste, dispatch.Caste)
		}
	}
}

func TestContinueDoesNotEraseBuildRowsEndToEnd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceBuildJSONOutput(t)

	root, dataDir, _, _ := setupIntermediateContinueState(t, "Two records, neither erasing the other")

	// The build lane's own writer, with the workflow word build-finalize uses.
	buildOutcome, err := writeSpendRowsForRun(spendWriteRequest{
		Phase:     1,
		PhaseName: "Two records, neither erasing the other",
		Workflow:  spendWorkflowBuild,
		RepoRoot:  root,
		Platform:  "claude",
		StartedAt: time.Now().UTC().Add(-time.Hour),
		EndedAt:   time.Now().UTC(),
		Dispatches: []codexBuildDispatch{
			{Caste: "builder", AgentName: "aether-builder", Name: "Forge-701", Task: "Complete intermediate work", Status: "completed"},
		},
	})
	if err != nil {
		t.Fatalf("write the build lane's rows: %v", err)
	}
	if buildOutcome.RowsWritten != 1 {
		t.Fatalf("build lane wrote %d rows, want 1", buildOutcome.RowsWritten)
	}

	plan := planAndFinalizeContinueForSpendTest(t, root, codexContinueOptions{LightFlag: true})

	buildLedger := loadSpendLedgerFromDisk(t, dataDir, 1, spendWorkflowBuild)
	continueLedger := loadSpendLedgerFromDisk(t, dataDir, 1, spendWorkflowContinue)

	if len(buildLedger.Rows) != 1 || buildLedger.Rows[0].AgentName != "Forge-701" {
		t.Fatalf("the check overwrote the build's rows; build ledger now holds %+v", buildLedger.Rows)
	}
	for _, row := range continueLedger.Rows {
		if row.AgentName == "Forge-701" {
			t.Errorf("the check's own file carries the build's worker %s", row.AgentName)
		}
	}
	continueNames := map[string]bool{}
	for _, row := range continueLedger.Rows {
		continueNames[row.AgentName] = true
	}
	for _, dispatch := range plan.Dispatches {
		if !continueNames[dispatch.Name] {
			t.Errorf("worker %s ran during the check but has no row in the check's own file", dispatch.Name)
		}
	}

	// And the phase total is both files added together, not whichever landed last.
	ledgers, ok := loadSpendLedgersForPhase(1)
	if !ok {
		t.Fatalf("no ledgers loaded for phase 1")
	}
	if got, want := len(spendRowsAcross(ledgers)), len(buildLedger.Rows)+len(continueLedger.Rows); got != want {
		t.Errorf("the phase spans %d rows, want %d (the build's plus the check's)", got, want)
	}
}

func TestContinueWithNoWorkersWritesNoFile(t *testing.T) {
	t.Run("the writer files nothing when a run had no workers", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		dataDir, req := newSpendWriterFixture(t)
		req.Workflow = spendWorkflowContinue
		req.Dispatches = nil

		outcome, err := writeSpendRowsForRun(req)
		if err != nil {
			t.Fatalf("writeSpendRowsForRun: %v", err)
		}
		if outcome.RowsWritten != 0 {
			t.Errorf("wrote %d rows for a run with no workers", outcome.RowsWritten)
		}
		path := filepath.Join(dataDir, "spend", "phase-7-continue.json")
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("a run with no workers left a file at %s; an empty file is indistinguishable from a run that cost nothing", path)
		}
	})

	t.Run("a check that spawned nobody leaves no file", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		forceBuildJSONOutput(t)

		root, dataDir, _, _ := setupIntermediateContinueState(t, "A check with nobody to send")

		plan := planAndFinalizeContinueForSpendTest(t, root, codexContinueOptions{LightFlag: true, SkipWatchers: true})
		if len(plan.Dispatches) != 0 {
			t.Skipf("this depth still planned %d worker(s); the no-worker case cannot be reached here", len(plan.Dispatches))
		}
		path := filepath.Join(dataDir, "spend", "phase-1-continue.json")
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("a check that ran no workers left a file at %s", path)
		}
	})
}

// TestClaudeBuildAttributesEveryWorkersTokens is CR-01 at the level the owner
// actually sees: the filed ledger and the rendered cost block.
//
// Reproduced before the fix as exactly this, on the repository's own primary
// platform:
//
//	── What This Phase Has Cost ──
//	Cost: not known. ...
//	  🔨🐜 Builder Mason-67  —  not reported
//
// The transcript is written in the shape Claude Code really writes (see
// newResolverClaudeTranscript), so this test cannot pass by describing the
// platform incorrectly.
func TestClaudeBuildAttributesEveryWorkersTokens(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	store = s
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	newResolverClaudeTranscript(t)

	outcome, err := writeSpendRowsForRun(spendWriteRequest{
		Phase:     4,
		PhaseName: "See what it cost",
		Workflow:  spendWorkflowBuild,
		RepoRoot:  tmpDir,
		Platform:  "claude",
		StartedAt: time.Now().Add(-time.Hour),
		EndedAt:   time.Now(),
		Dispatches: []codexBuildDispatch{
			{Caste: "builder", AgentName: "aether-builder", Name: "Mason-67", Task: "implement the parser", Status: "completed"},
			{Caste: "watcher", AgentName: "aether-watcher", Name: "Vigil-12", Task: "verify the parser", Status: "completed"},
		},
	})
	if err != nil {
		t.Fatalf("writeSpendRowsForRun: %v", err)
	}
	if outcome.Reported != 2 {
		t.Errorf("%d of 2 workers were credited with a figure; on Claude Code the accounting key and the identity the transcript records are different fields, and a run that credits neither reports no cost at all. Notes: %v",
			outcome.Reported, outcome.Notes)
	}

	ledger := loadSpendLedgerFromDisk(t, dataDir, 4, spendWorkflowBuild)
	mason := spendLedgerRowByName(t, ledger, "Mason-67")
	if !spendRowReportedUsage(mason) {
		t.Fatalf("Mason-67's filed row carries no measurement: %+v", mason)
	}
	// 100 + 200 + 300 + 400, added by hand from the fixture lines.
	if got := mason.Usage.BilledTotalTokens(); got != resolverClaudeMasonTotal {
		t.Errorf("Mason-67 billed total on disk = %d, want %d", got, resolverClaudeMasonTotal)
	}

	block := stripANSI(renderSpendCostLineFromLedgers([]spendLedger{ledger}))
	if strings.Contains(block, "Cost: not known") {
		t.Errorf("the owner's cost block says the cost is not known for a build whose transcript recorded every worker's tokens:\n%s", block)
	}
	if strings.Contains(block, spendNotReportedFigure) {
		t.Errorf("a worker is shown with no figure although the transcript carries one for it:\n%s", block)
	}
}
