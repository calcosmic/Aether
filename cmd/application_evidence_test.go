package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Shared fixture: a real phase, a real delivered instinct, and 0/1/2 real
// durable build attempts carrying real free-check reports and a real
// decision-kind knowledge delta -- exactly the shape recordPhaseApplicationCredit
// reads. Every fact flows through the real writers production uses
// (commitTestBuildStart/At, attachBuildFreeCheckReport,
// attachBuildKnowledgeDeltas, recordInstinctDeliveries via
// promoteRealInstinct) -- never a hand-typed recruitmentCreditRecord or
// buildAttemptRecord literal.
// ---------------------------------------------------------------------------

type applicationCreditFixtureOptions struct {
	InstinctContent   string
	HasEarlierAttempt bool
	EarlierFailed     []string
	LatestFailed      []string
	HasDecisionDelta  bool
}

func newApplicationCreditFixture(t *testing.T, opts applicationCreditFixtureOptions) (colony.Phase, colony.InstinctEntry) {
	t.Helper()

	taskOne := "1.1"
	seedPhase := colony.Phase{ID: 1, Name: "Application credit tracer"}
	seedPhase.Tasks = []colony.Task{{ID: &taskOne, Goal: "Exercise the credit tracer", Status: colony.TaskPending}}
	goal := "Exercise the phase application credit tracer"
	seedState := colony.ColonyState{Goal: &goal, State: colony.StateREADY, Plan: colony.Plan{
		AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
		EvidencePolicy:   colony.PlanEvidenceNotRequired,
		Phases:           []colony.Phase{seedPhase},
	}}

	var root string
	var latestAttemptPath string
	var phase colony.Phase

	if opts.HasEarlierAttempt {
		earlier := commitTestBuildStart(t, testBuildStartOptions{
			Variant: buildStartDirect, GeneratedAt: time.Now().UTC().Add(-time.Hour),
			SelectedTasks: []string{taskOne},
			Dispatches: []codexBuildDispatch{
				{Name: "Mason-1", Caste: "builder", TaskID: taskOne, CoveredTaskIDs: []string{taskOne}, Status: "completed"},
			},
			ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
			PrepareRoot: func(r string) {
				createTestColonyState(t, filepath.Join(r, ".aether", "data"), seedState)
			},
		})
		root = earlier.Root
		if err := attachBuildFreeCheckReport(earlier.AttemptPath, buildFreeCheckReport{
			RecordedAt: time.Now().UTC().Format(time.RFC3339), Phase: earlier.Phase.ID,
			ChecksRun: []string{"tests"}, Failed: opts.EarlierFailed, Passed: len(opts.EarlierFailed) == 0,
			Summary: "earlier attempt free checks",
		}); err != nil {
			t.Fatalf("attach earlier free-check report: %v", err)
		}

		latest := commitTestBuildStartAt(t, root, earlier.Phase.ID, time.Now().UTC(), testBuildStartOptions{
			Variant: buildStartDirect, GeneratedAt: time.Now().UTC(),
			SelectedTasks: []string{taskOne},
			Dispatches: []codexBuildDispatch{
				{Name: "Mason-2", Caste: "builder", TaskID: taskOne, CoveredTaskIDs: []string{taskOne}, Status: "completed"},
			},
			ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		})
		latestAttemptPath = latest.AttemptPath
		phase = latest.Phase
	} else {
		only := commitTestBuildStart(t, testBuildStartOptions{
			Variant: buildStartDirect, GeneratedAt: time.Now().UTC(),
			SelectedTasks: []string{taskOne},
			Dispatches: []codexBuildDispatch{
				{Name: "Mason-1", Caste: "builder", TaskID: taskOne, CoveredTaskIDs: []string{taskOne}, Status: "completed"},
			},
			ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
			PrepareRoot: func(r string) {
				createTestColonyState(t, filepath.Join(r, ".aether", "data"), seedState)
			},
		})
		root = only.Root
		latestAttemptPath = only.AttemptPath
		phase = only.Phase
	}
	_ = root

	if err := attachBuildFreeCheckReport(latestAttemptPath, buildFreeCheckReport{
		RecordedAt: time.Now().UTC().Format(time.RFC3339), Phase: phase.ID,
		ChecksRun: []string{"tests"}, Failed: opts.LatestFailed, Passed: len(opts.LatestFailed) == 0,
		Summary: "latest attempt free checks",
	}); err != nil {
		t.Fatalf("attach latest free-check report: %v", err)
	}

	if opts.HasDecisionDelta {
		if err := attachBuildKnowledgeDeltas(latestAttemptPath, []buildAttemptKnowledgeDelta{
			{Kind: buildKnowledgeDeltaKindDecision, Summary: "chose the real production writer over a second ledger"},
		}); err != nil {
			t.Fatalf("attach knowledge deltas: %v", err)
		}
	}

	instinct := promoteRealInstinct(t, store, opts.InstinctContent, "pattern")
	if recorded := recordInstinctDeliveries(phase.ID, "continue", instinct.Action); recorded != 1 {
		t.Fatalf("expected exactly one new instinct delivery, got %d", recorded)
	}

	return phase, instinct
}

// mustFindApplicationCreditRecord derives the expected decision ID from the
// phase's own latest durable build attempt (the same derivation
// recordPhaseApplicationCredit itself uses) and looks up the credit record
// recorded for (instinctID, that decision).
func mustFindApplicationCreditRecord(t *testing.T, phaseID int, instinctID string) recruitmentCreditRecord {
	t.Helper()
	_, latest, ok := loadLatestBuildAttempt(phaseID)
	if !ok {
		t.Fatalf("expected a latest build attempt for phase %d", phaseID)
	}
	decisionID := phaseApplicationDecisionID(latest.ID, 0)
	record, found, err := recruitmentCreditForContribution(instinctID, decisionID)
	if err != nil {
		t.Fatalf("read credit record: %v", err)
	}
	if !found {
		t.Fatalf("expected a credit record for contribution %s / decision %s", instinctID, decisionID)
	}
	return record
}

// ---------------------------------------------------------------------------
// Task 1 (204-02-PLAN.md): the phase's tracer.
// ---------------------------------------------------------------------------

func TestPhaseApplicationCreditTracerEndToEnd(t *testing.T) {
	t.Run("helpful: a failing check turned green", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "run go build ./cmd/aether before go vet ./cmd/ to catch compile errors early",
			HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: nil,
			HasDecisionDelta: true,
		})
		summary := recordPhaseApplicationCredit(phase.ID)
		if !summary.Ran || summary.Recorded != 1 || summary.Helpful != 1 {
			t.Fatalf("summary = %+v, want Ran=true Recorded=1 Helpful=1", summary)
		}
		record := mustFindApplicationCreditRecord(t, phase.ID, instinct.ID)
		if record.Outcome != recruitmentCreditOutcomeHelpful {
			t.Fatalf("outcome = %q, want helpful", record.Outcome)
		}
		if record.ContributionKind != recruitmentContributionMemoryItem {
			t.Fatalf("contribution kind = %q, want %q", record.ContributionKind, recruitmentContributionMemoryItem)
		}
	})

	t.Run("neutral: the failing-check set is unchanged", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early",
			HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: []string{"tests"},
			HasDecisionDelta: true,
		})
		summary := recordPhaseApplicationCredit(phase.ID)
		if !summary.Ran || summary.Recorded != 1 || summary.Neutral != 1 {
			t.Fatalf("summary = %+v, want Ran=true Recorded=1 Neutral=1", summary)
		}
		record := mustFindApplicationCreditRecord(t, phase.ID, instinct.ID)
		if record.Outcome != recruitmentCreditOutcomeNeutral {
			t.Fatalf("outcome = %q, want neutral", record.Outcome)
		}
	})

	t.Run("harmful: a passing check regressed", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "check pkg/colony/context_ranking.go before assuming score-based trim order",
			HasEarlierAttempt: true, EarlierFailed: nil, LatestFailed: []string{"tests"},
			HasDecisionDelta: true,
		})
		summary := recordPhaseApplicationCredit(phase.ID)
		if !summary.Ran || summary.Recorded != 1 || summary.Harmful != 1 {
			t.Fatalf("summary = %+v, want Ran=true Recorded=1 Harmful=1", summary)
		}
		record := mustFindApplicationCreditRecord(t, phase.ID, instinct.ID)
		if record.Outcome != recruitmentCreditOutcomeHarmful {
			t.Fatalf("outcome = %q, want harmful", record.Outcome)
		}
	})

	t.Run("pending: no earlier attempt to compare against", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "run go test ./pkg/... before go build ./cmd/aether to catch package failures early",
			HasEarlierAttempt: false, LatestFailed: nil,
			HasDecisionDelta: true,
		})
		summary := recordPhaseApplicationCredit(phase.ID)
		if !summary.Ran || summary.Recorded != 1 || summary.Pending != 1 || summary.Helpful != 0 {
			t.Fatalf("summary = %+v, want Ran=true Recorded=1 Pending=1 Helpful=0", summary)
		}
		record := mustFindApplicationCreditRecord(t, phase.ID, instinct.ID)
		if record.Outcome != recruitmentCreditOutcomePending {
			t.Fatalf("outcome = %q, want pending, never helpful", record.Outcome)
		}
	})

	t.Run("no-record: no decision-kind delta on the latest attempt", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "check .planning/WINDOWS.md before assuming a known gap is unrecorded",
			HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: nil,
			HasDecisionDelta: false,
		})
		summary := recordPhaseApplicationCredit(phase.ID)
		if !summary.Ran || summary.Recorded != 0 || summary.Considered == 0 {
			t.Fatalf("summary = %+v, want Ran=true Recorded=0 Considered>0", summary)
		}
		all, err := recruitmentCreditAll()
		if err != nil {
			t.Fatalf("read credit records: %v", err)
		}
		for _, rec := range all {
			if rec.ContributionID == instinct.ID {
				t.Fatalf("expected no credit record for a contribution with no recorded decision, found %+v", rec)
			}
		}
	})
}

// TestPhaseApplicationCreditIsReachedFromBothCheckLanes is an AST-based call-
// graph guard, in the style of cmd/worktree_destruction_reachability_test.go
// and cmd/verify_out_of_band_reachability_test.go: parse every non-test file
// in cmd/, and fail by name if recordPhaseApplicationCredit is not
// transitively reachable from EACH of the two functions both check lanes
// enter (runCodexContinue, cmd/codex_continue.go; runCodexContinueFinalize,
// cmd/codex_continue_finalize.go).
func TestPhaseApplicationCreditIsReachedFromBothCheckLanes(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}
	if g.funcsIndexed == 0 {
		t.Fatalf("found zero top-level functions while scanning %d files -- a guard that finds no functions to check would pass vacuously forever", g.filesScanned)
	}

	const target = "recordPhaseApplicationCredit"
	lanes := []string{"runCodexContinue", "runCodexContinueFinalize"}

	if _, ok := g.calls[target]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", target, g.funcsIndexed)
	}
	for _, lane := range lanes {
		if _, ok := g.calls[lane]; !ok {
			t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", lane, g.funcsIndexed)
		}
	}

	if unreached := applicationCreditUnreachedLanes(g, lanes, target); len(unreached) > 0 {
		t.Fatalf("%s is not transitively reachable from: %v -- both check lanes must reach the credit ledger writer", target, unreached)
	}

	t.Run("a synthetic unreachable fixture is caught by name", func(t *testing.T) {
		synthetic := &cmdFuncGraph{calls: map[string]map[string]bool{
			"runCodexContinue":             {"someOtherHelper": true},
			"runCodexContinueFinalize":     {"anotherHelper": true},
			"recordPhaseApplicationCredit": {},
		}}
		unreached := applicationCreditUnreachedLanes(synthetic, lanes, target)
		if len(unreached) != 2 {
			t.Fatalf("scanner failed to detect the synthetic unreachable fixture on both lanes, got unreached=%v", unreached)
		}
		for _, lane := range lanes {
			found := false
			for _, u := range unreached {
				if u == lane {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %q named in the unreached set %v", lane, unreached)
			}
		}
	})
}

// applicationCreditUnreachedLanes returns every lane in lanes from which
// target is NOT transitively reachable in g, by name -- extracted from the
// assertion above so the synthetic-fixture subtest can inspect the exact
// failure content without actually failing the outer test.
func applicationCreditUnreachedLanes(g *cmdFuncGraph, lanes []string, target string) []string {
	var unreached []string
	for _, lane := range lanes {
		reachable := reachableFrom(g, []string{lane})
		if !reachable[target] {
			unreached = append(unreached, lane)
		}
	}
	return unreached
}

// TestPhaseApplicationCreditIsReplaySafe runs the phase-end pass twice on
// identical state and asserts the credit store's bytes are identical after
// the second run.
func TestPhaseApplicationCreditIsReplaySafe(t *testing.T) {
	saveGlobals(t)
	phase, _ := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
		InstinctContent:   "run go vet ./cmd/ before go build ./cmd/aether to catch type errors early",
		HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: nil,
		HasDecisionDelta: true,
	})

	first := recordPhaseApplicationCredit(phase.ID)
	if !first.Ran || first.Recorded == 0 {
		t.Fatalf("first pass = %+v, want a real recorded credit", first)
	}

	creditFile := filepath.Join(store.BasePath(), filepath.FromSlash(recruitmentCreditPath))
	before, err := os.ReadFile(creditFile)
	if err != nil {
		t.Fatalf("read credit store after first pass: %v", err)
	}

	second := recordPhaseApplicationCredit(phase.ID)
	if !second.Ran {
		t.Fatalf("second pass = %+v, want Ran=true", second)
	}

	after, err := os.ReadFile(creditFile)
	if err != nil {
		t.Fatalf("read credit store after second pass: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("credit store changed after a replay run:\nbefore=%s\nafter=%s", before, after)
	}
}

// TestTunerSeesGenuinelyProducedCredit proves the previously unreachable
// reader (pheromoneOutcomeReadCreditRecords, and the already-live
// tuneNoteStrengthFromOutcomes pass built on top of it, cmd/pheromone_outcome.go)
// is now fed by real production data rather than a test-only writer: after
// the tracer path writes a record, the reader genuinely sees a non-empty
// store, and the tuning pass runs cleanly (Ran=true) against it. The tuning
// pass's own RecordsConsidered counter is scoped to note-kind contributions
// only (cmd/pheromone_outcome.go) -- this tracer writes memory-item kind, so
// this test asserts on the shared reader both paths call, the honest proof
// available without altering pheromone_outcome.go (outside this plan's
// declared files_modified).
func TestTunerSeesGenuinelyProducedCredit(t *testing.T) {
	saveGlobals(t)
	phase, _ := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
		InstinctContent:   "run go build ./cmd/aether before go vet ./cmd/ to catch compile errors early",
		HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: nil,
		HasDecisionDelta: true,
	})

	summary := recordPhaseApplicationCredit(phase.ID)
	if !summary.Ran || summary.Recorded == 0 {
		t.Fatalf("tracer write = %+v, want a real recorded credit", summary)
	}

	records, err := pheromoneOutcomeReadCreditRecords()
	if err != nil {
		t.Fatalf("read credit records via the tuner's own reader: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("the tuner's own reader (pheromoneOutcomeReadCreditRecords) still sees an empty store after the tracer wrote a real record -- the reader is not fed by production data")
	}

	tuning := tuneNoteStrengthFromOutcomes()
	if !tuning.Ran {
		t.Fatalf("tuneNoteStrengthFromOutcomes = %+v, want Ran=true now that the credit store genuinely holds data", tuning)
	}
}

// ---------------------------------------------------------------------------
// Task 1 (204-06-PLAN.md): the nine-state guidance application vocabulary
// and its transition rule.
// ---------------------------------------------------------------------------

// mustRecordGuidanceState records state for (guidanceID, phaseID) via the
// real writer and fails the test if the write is refused.
func mustRecordGuidanceState(t *testing.T, guidanceID string, phaseID int, state guidanceApplicationState) {
	t.Helper()
	if _, _, err := recordGuidanceApplicationState(guidanceID, recruitmentContributionMemoryItem, phaseID, state, ""); err != nil {
		t.Fatalf("record guidance state %s: %v", state, err)
	}
}

func TestGuidanceStateVocabularyIsClosed(t *testing.T) {
	want := map[guidanceApplicationState]bool{
		guidanceApplicationStateAvailable:    true,
		guidanceApplicationStateRendered:     true,
		guidanceApplicationStateConsulted:    true,
		guidanceApplicationStateActedOn:      true,
		guidanceApplicationStateIgnored:      true,
		guidanceApplicationStateContradicted: true,
		guidanceApplicationStateHelpful:      true,
		guidanceApplicationStateNeutral:      true,
		guidanceApplicationStateHarmful:      true,
	}
	if len(guidanceApplicationStateVocabulary) != len(want) {
		t.Fatalf("guidanceApplicationStateVocabulary has %d members, want exactly %d: %v", len(guidanceApplicationStateVocabulary), len(want), guidanceApplicationStateVocabulary)
	}
	seen := map[guidanceApplicationState]bool{}
	for _, s := range guidanceApplicationStateVocabulary {
		if !want[s] {
			t.Fatalf("unexpected state %q in guidanceApplicationStateVocabulary", s)
		}
		if seen[s] {
			t.Fatalf("state %q appears more than once in guidanceApplicationStateVocabulary", s)
		}
		seen[s] = true
	}
	for s := range want {
		if !seen[s] {
			t.Fatalf("declared state %q is missing from guidanceApplicationStateVocabulary", s)
		}
	}
	if names := guidanceApplicationStateNames(); len(names) != len(want) {
		t.Fatalf("guidanceApplicationStateNames() returned %d names, want %d: %v", len(names), len(want), names)
	}

	t.Run("an undeclared state is refused and the refusal lists all nine", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")

		_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationState("bogus"), "")
		if err == nil {
			t.Fatal("expected an undeclared state to be refused")
		}
		if written {
			t.Fatal("expected written=false for a refused state")
		}
		for _, name := range guidanceApplicationStateNames() {
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("refusal message %q does not name declared state %q", err.Error(), name)
			}
		}
	})
}

func TestGuidanceStateTransitionsRequireTheirPredecessors(t *testing.T) {
	setup := func(t *testing.T) colony.InstinctEntry {
		t.Helper()
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		t.Cleanup(func() { os.RemoveAll(tmpDir) })
		store = s
		return promoteRealInstinct(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")
	}

	t.Run("rendered without available is refused, naming available", func(t *testing.T) {
		inst := setup(t)
		_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateRendered, "")
		if err == nil || written {
			t.Fatalf("expected rendered without available to be refused, got written=%v err=%v", written, err)
		}
		if !strings.Contains(err.Error(), "available") {
			t.Fatalf("refusal %q does not name the missing predecessor %q", err.Error(), "available")
		}
	})

	t.Run("consulted without rendered is refused, naming rendered", func(t *testing.T) {
		inst := setup(t)
		mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
		_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateConsulted, "")
		if err == nil || written {
			t.Fatalf("expected consulted without rendered to be refused, got written=%v err=%v", written, err)
		}
		if !strings.Contains(err.Error(), "rendered") {
			t.Fatalf("refusal %q does not name the missing predecessor %q", err.Error(), "rendered")
		}
	})

	t.Run("acted_on without consulted is refused, naming consulted", func(t *testing.T) {
		inst := setup(t)
		mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
		mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateRendered)
		_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateActedOn, "")
		if err == nil || written {
			t.Fatalf("expected acted_on without consulted to be refused, got written=%v err=%v", written, err)
		}
		if !strings.Contains(err.Error(), "consulted") {
			t.Fatalf("refusal %q does not name the missing predecessor %q", err.Error(), "consulted")
		}
	})

	t.Run("contradicted without consulted is refused, naming consulted", func(t *testing.T) {
		inst := setup(t)
		mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
		mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateRendered)
		_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateContradicted, "")
		if err == nil || written {
			t.Fatalf("expected contradicted without consulted to be refused, got written=%v err=%v", written, err)
		}
		if !strings.Contains(err.Error(), "consulted") {
			t.Fatalf("refusal %q does not name the missing predecessor %q", err.Error(), "consulted")
		}
	})

	for _, terminal := range []guidanceApplicationState{guidanceApplicationStateHelpful, guidanceApplicationStateNeutral, guidanceApplicationStateHarmful} {
		terminal := terminal
		t.Run(fmt.Sprintf("%s without acted_on is refused, naming acted_on", terminal), func(t *testing.T) {
			inst := setup(t)
			mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
			mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateRendered)
			mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateConsulted)
			_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, terminal, "")
			if err == nil || written {
				t.Fatalf("expected %s without acted_on to be refused, got written=%v err=%v", terminal, written, err)
			}
			if !strings.Contains(err.Error(), "acted_on") {
				t.Fatalf("refusal %q does not name the missing predecessor %q", err.Error(), "acted_on")
			}
		})
	}

	t.Run("no transition comparison outside the predecessor map", func(t *testing.T) {
		assertNoGuidanceTransitionComparisonOutsidePredecessorMap(t)
	})
}

// guidanceApplicationStateConstNames is the exact set of declared guidance
// application state constant identifiers, used by the AST scan below to spot
// an inline comparison against one of them.
var guidanceApplicationStateConstNames = map[string]bool{
	"guidanceApplicationStateAvailable":    true,
	"guidanceApplicationStateRendered":     true,
	"guidanceApplicationStateConsulted":    true,
	"guidanceApplicationStateActedOn":      true,
	"guidanceApplicationStateIgnored":      true,
	"guidanceApplicationStateContradicted": true,
	"guidanceApplicationStateHelpful":      true,
	"guidanceApplicationStateNeutral":      true,
	"guidanceApplicationStateHarmful":      true,
}

// assertNoGuidanceTransitionComparisonOutsidePredecessorMap parses the real
// cmd/application_evidence.go and fails by name if any equality/inequality
// comparison anywhere in the file (outside the guidanceApplicationPredecessors
// var declaration itself, whose map literal necessarily NAMES every state as
// a key and inside its Requires/Excludes values -- never as a comparison
// operand) tests a named guidance state constant against anything. Every
// transition rule this system enforces must live in that one map.
func assertNoGuidanceTransitionComparisonOutsidePredecessorMap(t *testing.T) {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, "cmd", "application_evidence.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse application_evidence.go: %v", err)
	}
	violations := guidanceStateComparisonViolations(fset, file)
	if len(violations) != 0 {
		t.Fatalf("found a transition comparison against a named guidance state constant outside guidanceApplicationPredecessors:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic inline comparison is caught", func(t *testing.T) {
		src := `package cmd

func f(state guidanceApplicationState) bool {
	if state == guidanceApplicationStateConsulted {
		return true
	}
	return false
}
`
		fset2 := token.NewFileSet()
		syntheticFile, parseErr := parser.ParseFile(fset2, "fixture_guidance_state.go", src, 0)
		if parseErr != nil {
			t.Fatalf("parse fixture: %v", parseErr)
		}
		fixtureViolations := guidanceStateComparisonViolations(fset2, syntheticFile)
		if len(fixtureViolations) == 0 {
			t.Fatal("scanner failed to detect a synthetic inline guidance state comparison")
		}
	})
}

func guidanceStateComparisonViolations(fset *token.FileSet, file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			isPredecessorsDecl := false
			for _, spec := range genDecl.Specs {
				vspec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vspec.Names {
					if name.Name == "guidanceApplicationPredecessors" {
						isPredecessorsDecl = true
					}
				}
			}
			if isPredecessorsDecl {
				continue
			}
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			bin, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			if bin.Op != token.EQL && bin.Op != token.NEQ {
				return true
			}
			if guidanceStateIdent(bin.X) || guidanceStateIdent(bin.Y) {
				violations = append(violations, fmt.Sprintf(
					"%s: comparison against a named guidance state constant outside guidanceApplicationPredecessors",
					fset.Position(bin.Pos()).String(),
				))
			}
			return true
		})
	}
	return violations
}

func guidanceStateIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return guidanceApplicationStateConstNames[ident.Name]
}

func TestIgnoredAndConsultedAreMutuallyExclusive(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	inst := promoteRealInstinct(t, s, "check pkg/colony/context_ranking.go before assuming score-based trim order", "pattern")

	mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
	mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateRendered)
	mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateConsulted)

	_, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateIgnored, "")
	if err == nil || written {
		t.Fatalf("expected ignored to be refused once consulted is recorded, got written=%v err=%v", written, err)
	}
	if !strings.Contains(err.Error(), "consulted") {
		t.Fatalf("refusal %q does not name the conflicting state %q", err.Error(), "consulted")
	}

	t.Run("ignored is reachable when only rendered (never consulted) is recorded", func(t *testing.T) {
		inst2 := promoteRealInstinct(t, s, "run go test ./pkg/... before go build ./cmd/aether to catch package failures early", "pattern")
		mustRecordGuidanceState(t, inst2.ID, 1, guidanceApplicationStateAvailable)
		mustRecordGuidanceState(t, inst2.ID, 1, guidanceApplicationStateRendered)
		record, written, err := recordGuidanceApplicationState(inst2.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateIgnored, "")
		if err != nil || !written {
			t.Fatalf("expected ignored to be recordable for rendered-only guidance, got written=%v err=%v", written, err)
		}
		if record.State != guidanceApplicationStateIgnored {
			t.Fatalf("record.State = %q, want %q", record.State, guidanceApplicationStateIgnored)
		}
	})
}

func TestGuidanceStateRepeatWritesNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	inst := promoteRealInstinct(t, s, "run go vet ./cmd/ before go build ./cmd/aether to catch type errors early", "pattern")

	mustRecordGuidanceState(t, inst.ID, 1, guidanceApplicationStateAvailable)
	first, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateRendered, "")
	if err != nil || !written {
		t.Fatalf("expected the first rendered write to succeed, got written=%v err=%v", written, err)
	}

	creditFile := filepath.Join(s.BasePath(), filepath.FromSlash(recruitmentCreditPath))
	before, err := os.ReadFile(creditFile)
	if err != nil {
		t.Fatalf("read credit store after first write: %v", err)
	}

	second, written, err := recordGuidanceApplicationState(inst.ID, recruitmentContributionMemoryItem, 1, guidanceApplicationStateRendered, "")
	if err != nil {
		t.Fatalf("repeat write returned an error: %v", err)
	}
	if written {
		t.Fatal("expected the repeat write to report written=false")
	}
	if second.RecordID != first.RecordID {
		t.Fatalf("repeat write returned a different record: first=%+v second=%+v", first, second)
	}

	after, err := os.ReadFile(creditFile)
	if err != nil {
		t.Fatalf("read credit store after repeat write: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("credit store changed after a repeat write:\nbefore=%s\nafter=%s", before, after)
	}
}
