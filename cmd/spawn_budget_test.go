package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// SPAWN-03 red-proofs (D-02/D-03/D-04/D-09/D-11/D-19). Each test begins a
// spawn run first (via beginRuntimeSpawnRun, the same path every lifecycle
// command uses) so CurrentRun() resolves and the budget check has a run
// window to count against.

// readMiddenEntryCountForTest returns the number of entries currently in
// midden.json for the active store, treating a missing or unreadable file as
// zero entries (the natural state before any refusal has ever been logged).
func readMiddenEntryCountForTest(t *testing.T) int {
	t.Helper()
	var mf colony.MiddenFile
	if err := store.LoadJSON("midden.json", &mf); err != nil {
		return 0
	}
	return len(mf.Entries)
}

// readMiddenBytesOrNilForTest returns the raw bytes of midden.json, or nil
// if the file does not yet exist. Used for a byte-identical comparison
// rather than a count, so this test also catches an implementation that
// touches the file without changing its entry count (e.g. rewriting the
// same entries).
func readMiddenBytesOrNilForTest() []byte {
	data, err := store.ReadFile("midden.json")
	if err != nil {
		return nil
	}
	return data
}

// spawnTreeBudgetRedProofLiteralMax is 20, spelled out as a literal instead
// of referencing spawnTreeBudgetMax. Any test that drives its own spawn
// count off spawnTreeBudgetMax would still pass unchanged if that constant
// were widened during the D-22 red-proof exercise — the loop bound would
// widen right alongside the thing being proved, exactly the numeric
// coincidence plan 04's summary already hit once with the depth cap. Tests
// 1 and 5 below use this literal for their loop bounds so that widening
// spawnTreeBudgetMax genuinely inverts them.
const spawnTreeBudgetRedProofLiteralMax = 20

// TestSpawnTreeBudgetRefusesTheTwentyFirstHelper is D-02's direct proof: a
// run that never exceeds a single wave's own cap (8) is still refused once
// the whole run reaches 20 helpers, because the tree budget counts every
// helper spawned anywhere in the run, not one wave's dispatch list.
func TestSpawnTreeBudgetRefusesTheTwentyFirstHelper(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// 20 workers all parented to the coordinator sentinel, each recorded at
	// depth 1 — a legal shape under the depth cap (spawnMaxDelegationDepth
	// is 2), so any refusal at helper 21 can only be the budget, not depth.
	// The loop bound is the literal spawnTreeBudgetRedProofLiteralMax, not
	// spawnTreeBudgetMax itself — see that constant's comment.
	for i := 1; i <= spawnTreeBudgetRedProofLiteralMax; i++ {
		name := fmt.Sprintf("W%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	before, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt after the 20th spawn: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W21", "0"))
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the 21st spawn-log did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env := parseEnvelope(t, errBuf.String())
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "budget") {
		t.Fatalf("deny message does not name the budget: %s", errBuf.String())
	}

	after, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt after the refused 21st: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("spawn-tree.txt bytes changed after a refused spawn:\nbefore=%q\nafter=%q", before, after)
	}
}

// TestSpawnTreeBudgetIsNotRestoredBetweenWaves is ROADMAP criterion 3's
// second half and D-04's proof: budget consumed in one wave is not restored
// once that wave completes. Three waves of 8, 8 and 4 (every one of them
// finished, none of them exceeding the wave cap of 8) still exhaust the
// whole-run budget of 20, and the 21st attempt across all three waves is
// refused.
func TestSpawnTreeBudgetIsNotRestoredBetweenWaves(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	spawnAndCompleteWave := func(prefix string, count int) {
		for i := 1; i <= count; i++ {
			name := fmt.Sprintf("%s%d", prefix, i)
			runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
		}
		for i := 1; i <= count; i++ {
			name := fmt.Sprintf("%s%d", prefix, i)
			buf.Reset()
			errBuf.Reset()
			renderedCommandExitCode.Store(0)
			rootCmd.SetArgs([]string{"spawn-complete", "--name", name})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("spawn-complete %s failed: %v\nstderr: %s", name, err, errBuf.String())
			}
			if code := int(renderedCommandExitCode.Load()); code != 0 {
				t.Fatalf("spawn-complete %s exited %d: %s", name, code, errBuf.String())
			}
		}
	}

	// Wave 1: 8 helpers, all completed. No single wave ever exceeds 8.
	spawnAndCompleteWave("A", 8)
	// Wave 2: 8 more helpers, all completed.
	spawnAndCompleteWave("B", 8)

	state, err := spawnTreeBudgetState()
	if err != nil {
		t.Fatalf("spawnTreeBudgetState after two completed waves: %v", err)
	}
	if state.Consumed != 16 {
		t.Fatalf("Consumed after two completed waves of 8 = %d, want 16 — the count must not reset between waves", state.Consumed)
	}

	// Wave 3: 4 more helpers, bringing the run total to 20 (the ceiling).
	for i := 1; i <= 4; i++ {
		name := fmt.Sprintf("C%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "D1", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the 21st spawn, spread across three waves none of which exceeded 8, did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
}

// queenSpawnBudgetIdentifiers lists the top-level identifiers declared in
// cmd/queen_spawn_budget.go as of this plan. TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities
// asserts cmd/spawn_budget.go references none of them — D-04 requires the
// wave cap and the tree budget to share no identifier, so a later merge of
// the two cannot happen silently through an accidental import of the other
// file's vocabulary.
var queenSpawnBudgetIdentifiers = []string{
	"queenSpawnBudgetDecision",
	"queenSpawnBudgetForPhase",
	"applyQueenSpawnBudget",
	"queenSpawnBudgetDecisions",
	"queenRequiredCastesForBudget",
	"queenBuildSafetyRequiredCaste",
	"queenBuildSafetyRequiredCastes",
	"queenPhaseHasSecuritySignal",
	"queenBuildSafetyReviewRequired",
	"queenBuildBaseWorkerBudget",
	"applyBuildDepthToBudget",
	"buildWorkerCapLight",
	"buildWorkerFloorHeavy",
	"queenMaxWorkersForBudget",
	"effectiveQueenPhaseMode",
	"queenPhaseProducesTestableCode",
	"containsDocumentationWord",
	"queenPhaseIsDocumentationOnly",
	"queenPhaseLooksDocumentationOrMaintenance",
	"appendQueenBudgetRationale",
	"appendQueenPrunedRationale",
	"MaxWorkers",
}

// TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities is D-04's invariant
// proof: the whole-run tree budget and the Queen's per-wave worker-selection
// cap must stay two visibly different quantities, not one number doing
// double duty. A test that only asserted spawnTreeBudgetMax == 20 would
// still pass if someone later set the wave cap to 20 and pointed one budget
// at the other — this asserts the RELATIONSHIP, and separately asserts by
// source inspection that the two files share no identifier.
func TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities(t *testing.T) {
	flowTypes := []string{"build", "continue", "plan", "colonize", "swarm", "seal", "some-other-flow"}
	modes := []colony.PhaseMode{
		colony.PhaseModeDiscovery,
		colony.PhaseModePrototype,
		colony.PhaseModeProduction,
		colony.PhaseModeMaintenance,
	}
	names := []string{
		"",
		"Security hardening",
		"Add a CSV export",
		"Final review before release",
		"Database migration cleanup",
	}
	depths := []colony.VerificationDepth{
		colony.VerificationDepthLight,
		colony.VerificationDepthStandard,
		colony.VerificationDepthHeavy,
	}

	for _, flow := range flowTypes {
		for _, mode := range modes {
			for _, name := range names {
				for _, depth := range depths {
					phase := colony.Phase{Name: name, Mode: mode}
					state := colony.ColonyState{VerificationDepth: string(depth)}
					budget := queenSpawnBudgetForPhase(phase, flow, state)
					if budget.MaxWorkers == spawnTreeBudgetMax {
						t.Fatalf(
							"queenSpawnBudgetForPhase(flow=%s, mode=%s, name=%q, depth=%s) returned MaxWorkers %d, colliding with spawnTreeBudgetMax (%d) — D-04 requires these to stay separate quantities",
							flow, mode, name, depth, budget.MaxWorkers, spawnTreeBudgetMax,
						)
					}
				}
			}
		}
	}

	budgetSrc, err := os.ReadFile("spawn_budget.go")
	if err != nil {
		t.Fatalf("read spawn_budget.go: %v", err)
	}
	for _, ident := range queenSpawnBudgetIdentifiers {
		if bytes.Contains(budgetSrc, []byte(ident)) {
			t.Fatalf("cmd/spawn_budget.go references %q, an identifier declared in cmd/queen_spawn_budget.go — D-04 requires the two budgets to share no identifier", ident)
		}
	}
}

// TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable is D-19's proof for
// the budget check specifically: when the run state the budget depends on
// cannot be read, the check must deny, never silently allow. The negative
// control (same call, readable state, consumption below the ceiling) is in
// the same function so a passing test cannot be satisfied by a check that
// always denies.
func TestSpawnTreeBudgetFailsClosedWhenTheTreeIsUnreadable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// Negative control: a readable run state, well below the ceiling, must
	// allow.
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log with a readable run state failed: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("spawn-log with a readable run state and consumption below the ceiling was refused: %s", errBuf.String())
	}

	// Corrupt the run-state file the budget check's CurrentRun/EntriesForRun
	// calls depend on. AtomicWrite validates JSON for .json paths (by
	// design, to prevent exactly this kind of corruption in real use), so
	// the corruption is written directly to the resolved path on disk —
	// standing in for the "file was corrupted by something else" case
	// AtomicWrite's own guard cannot prevent.
	runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
	if err := os.WriteFile(runStatePath, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("corrupt %s: %v", runStatePath, err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W2", "0"))
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("spawn-log with an unreadable run state did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env := parseEnvelope(t, errBuf.String())
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "budget") || !strings.Contains(errMsg, "unverifiable") {
		t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
	}
}

// TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot is D-11's split,
// asserted both ways: hitting the whole-run budget ceiling writes exactly
// one midden entry naming the delegation budget; a routine depth-cap
// refusal writes nothing. A test asserting only the first half would pass
// against an implementation that logs every refusal, which is the exact
// behaviour D-11 rejects (it would drown real faults and leak task text).
func TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// --- Half 1: a budget-ceiling refusal writes exactly one midden entry.
	s1, tmpDir1 := newTestStore(t)
	defer os.RemoveAll(tmpDir1)
	store = s1
	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	for i := 1; i <= spawnTreeBudgetRedProofLiteralMax; i++ {
		name := fmt.Sprintf("W%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	middenBefore := readMiddenEntryCountForTest(t)

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W21", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("budget-ceiling spawn attempt did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}

	middenAfterBudget := readMiddenEntryCountForTest(t)
	if middenAfterBudget-middenBefore != 1 {
		t.Fatalf("expected exactly one new midden entry after a budget-ceiling refusal, got before=%d after=%d", middenBefore, middenAfterBudget)
	}

	var mf colony.MiddenFile
	if err := store.LoadJSON("midden.json", &mf); err != nil {
		t.Fatalf("read midden.json after budget refusal: %v", err)
	}
	newest := mf.Entries[len(mf.Entries)-1]
	if !strings.Contains(strings.ToLower(newest.Category+" "+newest.Message), "budget") {
		t.Fatalf("new midden entry does not name the delegation budget: %+v", newest)
	}

	// --- Half 2: a routine depth-cap refusal writes nothing to the midden.
	s2, tmpDir2 := newTestStore(t)
	defer os.RemoveAll(tmpDir2)
	store = s2
	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// Distinct task text down the D1 -> D2 chain (both different from the D3
	// attempt's "t" below) avoids tripping plan 173-06's ancestor-cycle check.
	// This half's subject is the depth cap's midden silence, not ancestor-cycle
	// detection — a cycle refusal here would assert nothing about D-11's split.
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "D1", "builder", "coordinate the initial request")) // depth 1
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("D1", "D2", "builder", "carry out the coordinated request")) // depth 2

	middenBeforeDepth := readMiddenBytesOrNilForTest()

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("D2", "D3", "0")) // would be depth 3: depth-cap refusal
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("depth-cap spawn attempt did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}

	middenAfterDepth := readMiddenBytesOrNilForTest()
	if !bytes.Equal(middenBeforeDepth, middenAfterDepth) {
		t.Fatalf("midden.json changed after a routine depth-cap refusal — D-11 requires only budget-ceiling refusals to write to the midden:\nbefore=%q\nafter=%q", middenBeforeDepth, middenAfterDepth)
	}
}

// Plan 173-12 (SPAWN-03, T-173-55/T-173-70/T-173-71). 173-VERIFICATION.md's
// Gap 1 reproduced the whole-run budget being reset to zero by overwriting
// spawn-tree.txt with one line of garbage. Planning found a second,
// tampering-free route to the identical outcome: deleting spawn-runs.json
// against a perfectly valid ledger. The three tests below prove both routes
// are closed, and that closing them does not lock ordinary use out.

// TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget reproduces
// 173-VERIFICATION.md's Gap 1 reproduction step for step, then asserts the
// opposite outcome: a spawn already refused for an exhausted whole-run
// budget is STILL refused after the ledger is overwritten with one line of
// garbage, whether or not spawn-runs.json is also removed alongside it.
func TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// Control: legitimately exhaust the budget first, exactly as
	// TestSpawnTreeBudgetRefusesTheTwentyFirstHelper does, so the tampering
	// below is proven against a genuinely full budget, not an empty one.
	for i := 1; i <= spawnTreeBudgetRedProofLiteralMax; i++ {
		name := fmt.Sprintf("W%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W21", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the 21st spawn-log did not exit non-zero before any tampering: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	controlEnv := parseEnvelope(t, errBuf.String())
	if msg, _ := controlEnv["error"].(string); !strings.Contains(msg, "budget") {
		t.Fatalf("legitimate ceiling refusal does not name the budget: %s", errBuf.String())
	}

	// Capture midden.json AFTER the legitimate ceiling refusal has already
	// written its own entry (D-11), so the comparison below isolates the
	// tampered-ledger refusal specifically.
	middenBeforeTamper := readMiddenBytesOrNilForTest()

	// Inject the exploit: 173-VERIFICATION.md's exact reproduction --
	// `echo "garbage" > spawn-tree.txt`, with a canary string the deny
	// message must never echo.
	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath, []byte("garbage not pipe format LEDGER-CONTENT-CANARY"), 0644); err != nil {
		t.Fatalf("inject corrupt spawn-tree.txt: %v", err)
	}
	corruptedBytes, err := os.ReadFile(treePath)
	if err != nil {
		t.Fatalf("read injected spawn-tree.txt: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W22-BYPASS", "0"))
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the identical spawn-log request succeeded against a corrupted ledger -- the exploit still works: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env := parseEnvelope(t, errBuf.String())
	msg, _ := env["error"].(string)
	for _, want := range []string{"budget", "unverifiable", "spawn-tree.txt"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("deny message %q does not contain %q", msg, want)
		}
	}
	if strings.Contains(msg, "LEDGER-CONTENT-CANARY") {
		t.Fatalf("deny message leaks the tampered ledger's own content: %s", msg)
	}

	afterTamperBytes, err := os.ReadFile(treePath)
	if err != nil {
		t.Fatalf("read spawn-tree.txt after the tampered-ledger refusal: %v", err)
	}
	if !bytes.Equal(corruptedBytes, afterTamperBytes) {
		t.Fatalf("spawn-tree.txt bytes changed after the tampered-ledger refusal:\nbefore=%q\nafter=%q", corruptedBytes, afterTamperBytes)
	}

	middenAfterTamper := readMiddenBytesOrNilForTest()
	if !bytes.Equal(middenBeforeTamper, middenAfterTamper) {
		t.Fatalf("midden.json changed after the tampered-ledger refusal -- D-11 reserves the midden for the budget-ceiling event, not a tampered ledger:\nbefore=%q\nafter=%q", middenBeforeTamper, middenAfterTamper)
	}

	// Harder variant: with the ledger STILL corrupt, also remove
	// spawn-runs.json. This is the assertion that fails if the integrity
	// check is ever moved below the no-run-yet early return -- without it,
	// this combination takes the no-run-yet exception and reports a fresh
	// budget.
	//
	// The assertion below checks the deny message names the BUDGET
	// specifically, not merely a non-zero exit code: RecordSpawn's own
	// write-time guard (plan 11) would ALSO refuse a write against this
	// corrupted content and produce a non-zero exit on its own, which would
	// make a bare exit-code check pass even if Gap A's integrity call were
	// removed from spawnTreeBudgetState (the decision would then reach
	// RecordSpawn instead of denying earlier at the budget check). Naming
	// "budget" and "unverifiable" pins the refusal to the budget check
	// specifically, which is the one this red-proof is about.
	runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
	if err := os.Remove(runStatePath); err != nil {
		t.Fatalf("remove spawn-runs.json: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W23-BYPASS", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the identical spawn-log request succeeded against a corrupted ledger AND a removed run-state file: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	harderEnv := parseEnvelope(t, errBuf.String())
	harderMsg, _ := harderEnv["error"].(string)
	if !strings.Contains(harderMsg, "budget") || !strings.Contains(harderMsg, "unverifiable") {
		t.Fatalf("deny message does not name the budget as unverifiable (the refusal must come from the budget check, not merely from RecordSpawn's separate write guard): %s", errBuf.String())
	}
}

// TestErasingTheRunRecordDoesNotResetTheWholeRunBudget is the new test for
// the second reset route, and it must NOT touch the ledger at all -- that is
// the whole point, because TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget
// always corrupts the ledger first and so never exercises this path alone.
func TestErasingTheRunRecordDoesNotResetTheWholeRunBudget(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	for i := 1; i <= spawnTreeBudgetRedProofLiteralMax; i++ {
		name := fmt.Sprintf("W%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W21", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("the 21st spawn-log did not exit non-zero against a valid full ledger: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}

	ledgerBefore, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt before erasing the run record: %v", err)
	}

	// Case 1: delete spawn-runs.json entirely, against a VALID, full ledger.
	// This does not touch spawn-tree.txt at all. Before this plan, this
	// request would succeed with budget_consumed:1 -- the no-run-yet
	// exception reporting a fresh budget against 20 live, un-abandoned
	// entries. After it, the whole ledger is counted and the ceiling still
	// holds.
	runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
	if err := os.Remove(runStatePath); err != nil {
		t.Fatalf("remove spawn-runs.json: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W22-BYPASS", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("erasing spawn-runs.json against a full valid ledger let the identical request succeed -- budget_consumed reset: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env := parseEnvelope(t, errBuf.String())
	if msg, _ := env["error"].(string); !strings.Contains(msg, "budget") {
		t.Fatalf("deny message does not name the budget: %s", errBuf.String())
	}

	ledgerAfter, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt after erasing the run record: %v", err)
	}
	if !bytes.Equal(ledgerBefore, ledgerAfter) {
		t.Fatalf("spawn-tree.txt bytes changed merely from erasing spawn-runs.json:\nbefore=%q\nafter=%q", ledgerBefore, ledgerAfter)
	}

	// Case 2: a fresh store, a full VALID ledger, and a DIRECTORY
	// obstructing spawn-runs.json's path -- a file that exists and cannot be
	// read is a fault, not an absent history, and must deny outright. This
	// case depends on plan 11's loadRunStateLocked change.
	s2, tmpDir2 := newTestStore(t)
	defer os.RemoveAll(tmpDir2)
	store = s2

	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run (second store): %v", err)
	}
	for i := 1; i <= spawnTreeBudgetRedProofLiteralMax; i++ {
		name := fmt.Sprintf("V%d", i)
		runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", name, "0"))
	}

	runStatePath2 := filepath.Join(store.BasePath(), "spawn-runs.json")
	if err := os.Remove(runStatePath2); err != nil {
		t.Fatalf("remove spawn-runs.json (second store): %v", err)
	}
	if err := os.MkdirAll(runStatePath2, 0755); err != nil {
		t.Fatalf("obstruct spawn-runs.json with a directory: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "V22-BYPASS", "0"))
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("an obstructed spawn-runs.json (directory at its path) did not deny: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env2 := parseEnvelope(t, errBuf.String())
	msg2, _ := env2["error"].(string)
	if !strings.Contains(msg2, "budget") || !strings.Contains(msg2, "unverifiable") {
		t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
	}
}

// TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn is the negative control
// that stops an always-deny implementation from passing the two tests above.
// Three states, none of which may be refused: an absent ledger, a zero-byte
// ledger, and a small ledger with no run ever recorded -- the exact shape
// cmd/spawn_enforce_test.go's and cmd/spawn_ancestor_test.go's own chains
// depend on, and the one .aether/workers.md:292 documents workers calling
// directly. A budget that refuses a colony which has genuinely never spawned
// anything -- or one that has simply never run a lifecycle command -- is not
// fail-closed, it is broken.
func TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	// Case 1: no spawn-tree.txt at all.
	s1, tmpDir1 := newTestStore(t)
	defer os.RemoveAll(tmpDir1)
	store = s1
	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run (case 1): %v", err)
	}
	if _, err := store.ReadFile("spawn-tree.txt"); err == nil {
		t.Fatalf("spawn-tree.txt unexpectedly exists before the first spawn in case 1")
	}
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log against an absent ledger failed: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("spawn-log against an absent ledger was refused: %s", errBuf.String())
	}
	if _, err := store.ReadFile("spawn-tree.txt"); err != nil {
		t.Fatalf("spawn-tree.txt was not created by the successful spawn: %v", err)
	}

	// Case 2: a zero-byte spawn-tree.txt.
	s2, tmpDir2 := newTestStore(t)
	defer os.RemoveAll(tmpDir2)
	store = s2
	if _, err := beginRuntimeSpawnRun("test-run", time.Now().UTC()); err != nil {
		t.Fatalf("begin run (case 2): %v", err)
	}
	treePath2 := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath2, []byte{}, 0644); err != nil {
		t.Fatalf("write zero-byte spawn-tree.txt: %v", err)
	}
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-log against a zero-byte ledger failed: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("spawn-log against a zero-byte ledger was refused: %s", errBuf.String())
	}

	// Case 3: a small ledger holding two recorded helpers, and no run ever
	// begun at all -- the exact shape cmd/spawn_enforce_test.go and
	// cmd/spawn_ancestor_test.go both depend on, and the one
	// .aether/workers.md:292 documents workers calling directly. This is the
	// case that fails if the no-run branch is ever "hardened" into a deny.
	// Distinct task text on the two calls avoids tripping the unrelated
	// ancestor-cycle check (same caste + same normalised task would deny on
	// its own terms, which would prove nothing about this budget branch).
	s3, tmpDir3 := newTestStore(t)
	defer os.RemoveAll(tmpDir3)
	store = s3
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "W1", "builder", "coordinate the initial request"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("W1", "H1", "builder", "carry out the coordinated request"))
}
