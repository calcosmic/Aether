package cmd

import (
	"encoding/json"
	"fmt"
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
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Shared fixtures (201-10)
// ---------------------------------------------------------------------------

// seedMinimalBuildAttempt persists just enough of a durable build attempt
// record (plus its latest-attempt pointer) for loadLatestBuildAttempt(phase)
// to resolve it -- the real mechanism recordDispatchWorkerOutcome now reads
// attempt/job identity from (cmd/memory_feed.go), without running the full
// commitBuildStart pipeline.
func seedMinimalBuildAttempt(t *testing.T, phase int, attemptID string, dispatches []codexBuildDispatch) {
	t.Helper()
	if store == nil {
		t.Fatal("seedMinimalBuildAttempt: store is nil")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	attemptRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".json"))
	record := buildAttemptRecord{
		SchemaVersion: buildAttemptSchemaVersion,
		ID:            attemptID,
		Phase:         phase,
		Status:        buildAttemptTerminal,
		StartedAt:     now,
		UpdatedAt:     now,
		Dispatches:    dispatches,
		History:       []buildAttemptTransition{},
	}
	if err := store.SaveJSON(attemptRel, record); err != nil {
		t.Fatalf("seed build attempt: %v", err)
	}
	pointer := latestBuildAttemptPointer{
		SchemaVersion: 1,
		AttemptID:     attemptID,
		Path:          attemptRel,
		UpdatedAt:     now,
	}
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phase), pointer); err != nil {
		t.Fatalf("seed latest attempt pointer: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 1: failure evidence and blocker truth bound to the exact attempt.
// ---------------------------------------------------------------------------

func TestFailureEvidenceCarriesTheAttemptIdentity(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 7
	const attemptID = "attempt-evidence-1"
	seedMinimalBuildAttempt(t, phase, attemptID, []codexBuildDispatch{
		{Name: "Mason-1", Caste: "builder", JobName: "job-alpha"},
	})

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: phase}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "failed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "failed",
			Blockers: []string{"go test ./cmd -run TestExpire failed: nil pointer dereference"},
		},
	}

	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	mf, err := loadMiddenFile(s)
	if err != nil {
		t.Fatalf("load midden: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	entry := mf.Entries[0]
	if got := middenEntryAttemptID(entry); got != attemptID {
		t.Fatalf("midden entry attempt ID = %q, want %q (tags: %v)", got, attemptID, entry.Tags)
	}
	if got := middenEntryJobName(entry); got != "job-alpha" {
		t.Fatalf("midden entry job name = %q, want %q (tags: %v)", got, "job-alpha", entry.Tags)
	}
	if !strings.HasPrefix(entry.Message, "go test ./cmd -run TestExpire failed") {
		t.Fatalf("midden entry message = %q, want it to start with the worker's own sentence", entry.Message)
	}
}

// TestEveryBuildLaneFeedsMemoryThroughOneBoundary (cmd/memory_feed_test.go)
// already proves both build lanes call recordDispatchWorkerOutcome and no
// other function persists a worker handoff -- this plan extends that one
// boundary rather than adding a second, so a single call through it (above)
// is sufficient evidence for "either lane": both lanes construct the exact
// same workerOutcomeFacts inside this one function.

func TestBlockerTruthIsOneStore(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 9
	const attemptID = "attempt-blocker-1"
	seedMinimalBuildAttempt(t, phase, attemptID, []codexBuildDispatch{
		{Name: "Hammer-3", Caste: "builder"},
	})

	dispatch := codex.WorkerDispatch{WorkerName: "Hammer-3", Caste: "builder", Workflow: "build", Phase: phase}
	result := codex.DispatchResult{
		WorkerName: "Hammer-3",
		Status:     "blocked",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Hammer-3", Caste: "builder", Status: "blocked",
			Blockers: []string{"cannot deploy: missing PROD_DB_URL secret"},
		},
	}

	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	// Surface 1: status/advancement read (readBlockerSnapshot,
	// cmd/blocker_snapshot.go).
	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		t.Fatalf("readBlockerSnapshot: %v", err)
	}
	if snapshot.Count != 1 || snapshot.EscalatedCount != 1 {
		t.Fatalf("blocker snapshot = %+v, want Count=1 EscalatedCount=1", snapshot)
	}

	// Surface 2: the advancement gate itself
	// (checkUnresolvedBlockerFlags, cmd/gate.go) -- the classic Iron Law.
	check := checkUnresolvedBlockerFlags()
	if check.Passed {
		t.Fatalf("checkUnresolvedBlockerFlags passed with an unresolved worker blocker outstanding: %+v", check)
	}

	// Surface 3: the closure read (LifecycleFacts.Blockers,
	// cmd/lifecycle_facts.go), which reads pending-decisions.json directly
	// via readLifecyclePendingDecisions -- the exact same file, proving all
	// three surfaces resolve from one store.
	_, flags, source := readLifecyclePendingDecisions(filepath.Join(s.BasePath(), "pending-decisions.json"))
	if source.Provenance != LifecycleFactConfirmed {
		t.Fatalf("readLifecyclePendingDecisions provenance = %v, want confirmed", source.Provenance)
	}
	found := false
	for _, flag := range flags.Decisions {
		if flag.AttemptID == attemptID && flag.Type == "blocker" {
			found = true
			if flag.Source != "escalation" {
				t.Errorf("blocker flag Source = %q, want %q", flag.Source, "escalation")
			}
		}
	}
	if !found {
		t.Fatalf("closure read (readLifecyclePendingDecisions) does not see the worker-reported blocker bound to attempt %q: %+v", attemptID, flags.Decisions)
	}
}

func TestEscalatedCountHasOneCountingPath(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const phase = 11
	seedMinimalBuildAttempt(t, phase, "attempt-count-1", []codexBuildDispatch{
		{Name: "Worker-A", Caste: "builder"},
		{Name: "Worker-B", Caste: "builder"},
		{Name: "Worker-C", Caste: "builder"},
	})

	workers := []struct {
		name    string
		blocker string
	}{
		{"Worker-A", "blocked on missing API key A"},
		{"Worker-B", "blocked on missing API key B"},
		{"Worker-C", "blocked on missing API key C"},
	}
	for _, w := range workers {
		dispatch := codex.WorkerDispatch{WorkerName: w.name, Caste: "builder", Workflow: "build", Phase: phase}
		result := codex.DispatchResult{
			WorkerName: w.name, Status: "blocked",
			WorkerResult: &codex.WorkerResult{WorkerName: w.name, Caste: "builder", Status: "blocked", Blockers: []string{w.blocker}},
		}
		if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
			t.Fatalf("recordDispatchWorkerOutcome(%s): %v", w.name, err)
		}
	}

	// The fixture's own known count: 3 distinct worker blockers.
	const wantEscalated = 3

	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		t.Fatalf("readBlockerSnapshot: %v", err)
	}
	if snapshot.EscalatedCount != wantEscalated {
		t.Fatalf("readBlockerSnapshot.EscalatedCount = %d, want %d (fixture's own count)", snapshot.EscalatedCount, wantEscalated)
	}

	// addBlockerSnapshotFields (the status-surface renderer) must report the
	// exact same number -- computed from the same snapshot, never a second
	// count.
	result := map[string]interface{}{}
	addBlockerSnapshotFields(result, snapshot)
	if got := result["escalated_blockers"]; got != wantEscalated {
		t.Fatalf("addBlockerSnapshotFields[\"escalated_blockers\"] = %v, want %d", got, wantEscalated)
	}

	// The advancement gate's own detail names the same total blocker count.
	check := checkUnresolvedBlockerFlags()
	if check.Passed {
		t.Fatalf("checkUnresolvedBlockerFlags passed with %d unresolved blockers outstanding", wantEscalated)
	}
	if !strings.Contains(check.Detail, fmt.Sprintf("%d unresolved blocker", wantEscalated)) {
		t.Fatalf("checkUnresolvedBlockerFlags detail = %q, want it to name %d unresolved blockers", check.Detail, wantEscalated)
	}
}

func TestEvidenceStorageFailureNeverFailsTheRun(t *testing.T) {
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
	// including the worker-handoffs file, midden.json, and
	// pending-decisions.json -- fails.
	if err := os.Chmod(dataDir, 0o500); err != nil {
		t.Fatalf("chmod data dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dataDir, 0o755) })

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "build", Phase: 1}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "failed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "failed",
			Blockers: []string{"go build ./cmd/aether failed: unwritable store"},
		},
	}

	wantErr := persistDispatchWorkerHandoff(dispatch, result)
	if wantErr == nil {
		t.Skip("environment did not make the store directory unwritable (e.g. running as root); nothing to assert")
	}
	gotErr := recordDispatchWorkerOutcome(dispatch, result)
	if gotErr == nil || gotErr.Error() != wantErr.Error() {
		t.Fatalf("recordDispatchWorkerOutcome error = %v, want %v (persistDispatchWorkerHandoff's own error, unchanged -- the blocker/midden writes must never add their own failure)", gotErr, wantErr)
	}
}

// ---------------------------------------------------------------------------
// Task 2: bounded recovery consumes flags and recurring failure classes.
// ---------------------------------------------------------------------------

func validAutopilotRepairFailureFixture(phase int, check string) autopilotRepairFailure {
	return autopilotRepairFailure{
		Phase: phase, Attempt: "attempt-repair-eval", Check: check,
		Evidence: []string{"go test ./cmd failed"}, PlannedAction: "fix the failing test",
		Baseline: "abc123", PermittedScope: []string{"cmd/example.go"},
		ScopeSafe: true, SafetySafe: true, AuthoritySafe: true,
	}
}

func seedUnresolvedBlockerFlag(t *testing.T, phase int, id string) {
	t.Helper()
	if store == nil {
		t.Fatal("seedUnresolvedBlockerFlag: store is nil")
	}
	phasePtr := phase
	ff := colony.FlagsFile{Decisions: []colony.FlagEntry{{
		ID: id, Type: "blocker", Description: "a manually-flagged blocker", Source: "manual",
		Phase: &phasePtr, CreatedAt: time.Now().UTC().Format(time.RFC3339), Resolved: false,
	}}}
	if err := store.SaveJSON("pending-decisions.json", ff); err != nil {
		t.Fatalf("seed unresolved blocker flag: %v", err)
	}
}

func seedRecurringFailureRedirectSignal(t *testing.T) {
	t.Helper()
	if store == nil {
		t.Fatal("seedRecurringFailureRedirectSignal: store is nil")
	}
	content, _ := json.Marshal(map[string]string{"text": "this failure has recurred 3 or more times unacknowledged"})
	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{{
		ID: "sig_recurring_1", Type: "REDIRECT", Priority: "high", Source: "aether continue",
		CreatedAt: time.Now().UTC().Format(time.RFC3339), Active: true, Content: content,
	}}}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed recurring failure signal: %v", err)
	}
}

func TestRepairEvaluationNamesItsDrivingInput(t *testing.T) {
	cases := []struct {
		name          string
		withFlags     bool
		withRecurring bool
		wantContains  []string
		wantAbsent    []string
	}{
		{name: "flags only", withFlags: true, wantContains: []string{"unresolved flag"}, wantAbsent: []string{"recurring"}},
		{name: "recurring only", withRecurring: true, wantContains: []string{"recurring failure class"}, wantAbsent: []string{"unresolved flag"}},
		{name: "both", withFlags: true, withRecurring: true, wantContains: []string{"unresolved flag", "recurring failure class"}},
		{name: "neither", wantAbsent: []string{"unresolved flag", "recurring failure class"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, tmpDir := newTestStore(t)
			defer os.RemoveAll(tmpDir)
			store = s

			if tc.withFlags {
				seedUnresolvedBlockerFlag(t, 13, "flag-driving-1")
			}
			if tc.withRecurring {
				seedRecurringFailureRedirectSignal(t)
			}

			failure := validAutopilotRepairFailureFixture(13, "go test ./cmd")
			evaluation := repairEligibilityEvaluation(failure, 3)
			if !evaluation.Eligible {
				t.Fatalf("evaluation = %+v, want Eligible=true for a valid, safe fixture", evaluation)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(evaluation.Reason, want) {
					t.Errorf("evaluation.Reason = %q, want it to contain %q", evaluation.Reason, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(evaluation.Reason, absent) {
					t.Errorf("evaluation.Reason = %q, want it to NOT contain %q", evaluation.Reason, absent)
				}
			}
		})
	}
}

func TestUnresolvedFlagsNeverEvaluateAsNothingToRepair(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	seedUnresolvedBlockerFlag(t, 17, "flag-driving-2")

	failure := validAutopilotRepairFailureFixture(17, "go vet ./cmd")
	evaluation := repairEligibilityEvaluation(failure, 3)
	if !strings.Contains(evaluation.Reason, "unresolved flag") {
		t.Fatalf("a phase with unresolved flags must never be evaluated without naming them: reason = %q", evaluation.Reason)
	}
}

// TestOneRecoveryModelConsumesFlagsAndFailures asserts, from the parsed
// syntax tree of cmd/work_repair.go, that exactly one function in the file
// calls the repair-ledger-mutating entry point (executeAutopilotRepair /
// beginAutopilotRepair) -- runBoundedRepairRound. A second call site would
// be exactly the second recovery path CAP-024 prohibits.
func TestOneRecoveryModelConsumesFlagsAndFailures(t *testing.T) {
	fset := token.NewFileSet()
	path := filepath.Join(".", "work_repair.go")
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	entryPoints := map[string]bool{"executeAutopilotRepair": true, "beginAutopilotRepair": true}
	var callers []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if entryPoints[ident.Name] {
				callers = append(callers, fn.Name.Name)
			}
			return true
		})
	}
	if len(callers) != 1 || callers[0] != "runBoundedRepairRound" {
		t.Fatalf("expected exactly one call site (runBoundedRepairRound) into the repair-ledger entry point in cmd/work_repair.go, got %v", callers)
	}
}

// ---------------------------------------------------------------------------
// Task 3: this phase's own failure-born signal reaches the repair wave's
// brief, in the same run.
// ---------------------------------------------------------------------------

func seedFailureBornMiddenEntry(t *testing.T, phase int, attemptID, message string) {
	t.Helper()
	if store == nil {
		t.Fatal("seedFailureBornMiddenEntry: store is nil")
	}
	if err := appendMiddenEntry(store, middenCategoryWorkerFailed, "aether build", message, []string{"build", "builder", middenAttemptTagPrefix + attemptID}); err != nil {
		t.Fatalf("seed failure-born midden entry: %v", err)
	}
}

func TestFailureBornSignalReachesTheRepairBrief(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const attemptID = "attempt-repair-brief-1"
	const failureText = "go test ./cmd -run TestWidget failed: nil pointer dereference in widget.go"
	seedFailureBornMiddenEntry(t, 5, attemptID, failureText+" — build phase 5, worker Mason-1 (builder), status failed")

	delivered := deliverFailureBornRepairSignal(5, "go test ./cmd", attemptID)
	if delivered == "" {
		t.Fatal("deliverFailureBornRepairSignal returned empty text for a known failure record")
	}
	if !strings.Contains(delivered, failureText) {
		t.Fatalf("delivered signal = %q, want it to name the specific failure %q", delivered, failureText)
	}

	// The repair worker's brief: plannedCheckFixBuilderDispatch
	// (cmd/check_fix_attempt.go) already sets ContextCapsule to
	// resolveCodexWorkerContext() -- the same function every other build
	// worker's steering content flows through. Read it back here to prove
	// the signal actually reaches that existing channel.
	brief := resolveCodexWorkerContext()
	if !strings.Contains(brief, failureText) {
		t.Fatalf("resolveCodexWorkerContext() does not carry the same-run failure signal:\n%s", brief)
	}

	// composeBuildManifestBrief's own "## Pheromone Signals" steering
	// section reads the identical active-signal store -- no new section was
	// added for this.
	pheromoneSection := resolvePheromoneSection()
	if !strings.Contains(pheromoneSection, failureText) {
		t.Fatalf("resolvePheromoneSection() does not carry the same-run failure signal:\n%s", pheromoneSection)
	}
	if strings.Contains(brief, "## Same-Run Repair Signal") || strings.Contains(pheromoneSection, "## Same-Run Repair Signal") {
		t.Fatal("a new brief section was introduced; the signal must ride the existing pheromone-signal channel only")
	}
}

// TestSignalDeliveredBeforeRepairDispatch asserts, from the parsed syntax
// tree of applyBoundedCheckFixRepair, that the call to
// deliverFailureBornRepairSignal appears (as a statement) before the call
// to applyAutomaticCheckFixAttempt that actually dispatches the repair
// wave -- proving delivery happens inside this run, before the dispatch,
// not merely "eventually".
func TestSignalDeliveredBeforeRepairDispatch(t *testing.T) {
	fset := token.NewFileSet()
	path := filepath.Join(".", "work_repair.go")
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var deliverPos, dispatchPos token.Pos
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name.Name != "applyBoundedCheckFixRepair" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			switch ident.Name {
			case "deliverFailureBornRepairSignal":
				if deliverPos == token.NoPos {
					deliverPos = call.Pos()
				}
			case "applyAutomaticCheckFixAttempt":
				// There are two call sites: an early-return fallback taken
				// ONLY when saving the checkpoint itself fails (before any
				// signal could be delivered), and the real dispatch taken
				// after the checkpoint succeeds. The LAST occurrence in
				// source order is always the real dispatch -- the fallback
				// is textually first because it sits in the checkpoint's
				// own error-handling block, earlier in the function body.
				dispatchPos = call.Pos()
			}
			return true
		})
	}
	if deliverPos == token.NoPos {
		t.Fatal("applyBoundedCheckFixRepair does not call deliverFailureBornRepairSignal")
	}
	if dispatchPos == token.NoPos {
		t.Fatal("applyBoundedCheckFixRepair does not call applyAutomaticCheckFixAttempt")
	}
	if deliverPos >= dispatchPos {
		t.Fatalf("deliverFailureBornRepairSignal (pos %d) must be called before the real repair dispatch (pos %d)", deliverPos, dispatchPos)
	}
}

func TestRepairBriefRespectsItsContentBudget(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const attemptID = "attempt-repair-brief-2"
	// Deliberately far longer than signalContentSafeLimit so truncation is
	// forced.
	longFailure := strings.Repeat("this failure text is exactly the same repeated phrase, over and over again, ", 10)
	seedFailureBornMiddenEntry(t, 9, attemptID, longFailure+" — build phase 9, worker Hammer-1 (builder), status failed")

	delivered := deliverFailureBornRepairSignal(9, "go vet ./cmd", attemptID)
	if delivered == "" {
		t.Fatal("deliverFailureBornRepairSignal returned empty text for a known (over-long) failure record")
	}
	if len(delivered) > signalContentSafeLimit {
		t.Fatalf("delivered signal is %d chars, want it capped at the declared budget (%d chars)", len(delivered), signalContentSafeLimit)
	}
	if !strings.HasSuffix(delivered, "...") {
		t.Fatalf("delivered signal = %q, want a forced trim to be reported (trailing ellipsis) rather than silently dropped", delivered)
	}

	// The trimmed signal must still have been genuinely stored -- never
	// silently discarded because it was too long.
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones.json: %v", err)
	}
	found := false
	for _, sig := range pf.Signals {
		var content struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(sig.Content, &content); err == nil && content.Text == delivered {
			found = true
		}
	}
	if !found {
		t.Fatalf("the trimmed signal was not durably stored in pheromones.json: %+v", pf.Signals)
	}
}
