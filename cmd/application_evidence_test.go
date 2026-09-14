package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
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

// guidanceTransitionRuleFuncs names the two functions that implement the
// transition RULE itself -- the single writer (recordGuidanceApplicationState)
// and the refusal check it delegates to (guidanceApplicationTransitionRefusal).
// The AST scan below is scoped to exactly these two: comparisons against a
// named guidance state constant appear all over this file for entirely
// different, legitimate reasons (deciding which state corroboration reached,
// filtering which records a phase-close sweep should act on) -- what must
// never happen is the TRANSITION rule (what predecessor a state requires,
// what state it excludes) being reimplemented inline anywhere outside the
// one declared map.
var guidanceTransitionRuleFuncs = map[string]bool{
	"recordGuidanceApplicationState":       true,
	"guidanceApplicationTransitionRefusal": true,
}

// assertNoGuidanceTransitionComparisonOutsidePredecessorMap parses the real
// cmd/application_evidence.go and fails by name if either transition-rule
// function above (outside the guidanceApplicationPredecessors var
// declaration itself, whose map literal necessarily NAMES every state as a
// key and inside its Requires/Excludes values -- never as a comparison
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

func guidanceApplicationTransitionRefusal(state guidanceApplicationState) bool {
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

// guidanceStateComparisonViolations walks ONLY the function bodies named in
// guidanceTransitionRuleFuncs (the writer and its refusal check), never the
// whole file -- see that var's own doc comment for why other functions'
// comparisons against a named state constant are legitimate and excluded.
func guidanceStateComparisonViolations(fset *token.FileSet, file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !guidanceTransitionRuleFuncs[fn.Name.Name] || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
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

// ---------------------------------------------------------------------------
// Task 2 (204-06-PLAN.md): checking a worker's claim against what the
// runtime can see for itself. Every fixture below drives the real dispatch
// and handoff-persistence boundary (recordDispatchWorkerOutcome,
// cmd/memory_feed.go) -- never a direct write to the handoff store or a
// hand-typed guidanceApplicationRecord.
// ---------------------------------------------------------------------------

// setupGuidanceClaimFixture opens a fresh real store and promotes one real
// instinct through the real observation->promotion path, returning it.
func setupGuidanceClaimFixture(t *testing.T, action string) colony.InstinctEntry {
	t.Helper()
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	return promoteRealInstinct(t, s, action, "pattern")
}

// guidanceApplicationRecordedStates reads the real credit store and returns
// the set of states already recorded for (guidanceID, phaseID).
func guidanceApplicationRecordedStates(t *testing.T, guidanceID string, phaseID int) map[guidanceApplicationState]bool {
	t.Helper()
	var file recruitmentCreditFile
	if err := store.LoadJSON(recruitmentCreditPath, &file); err != nil {
		return map[guidanceApplicationState]bool{}
	}
	return guidanceReachedStates(file.GuidanceApplications, guidanceID, phaseID)
}

func TestRenderedAloneIsNotConsulted(t *testing.T) {
	inst := setupGuidanceClaimFixture(t, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early")
	const phaseID = 1

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "continue", Phase: phaseID, ContextCapsule: inst.Action}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "completed",
			Summary: "did unrelated work",
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "pass",
			},
		},
	}
	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	reached := guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if !reached[guidanceApplicationStateRendered] {
		t.Fatalf("expected rendered to be recorded, reached=%v", reached)
	}
	if reached[guidanceApplicationStateConsulted] {
		t.Fatalf("expected consulted NOT to be recorded when nothing claims to have used the guidance, reached=%v", reached)
	}
}

func TestCorroboratedClaimBecomesConsulted(t *testing.T) {
	inst := setupGuidanceClaimFixture(t, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early")
	const phaseID = 1

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "continue", Phase: phaseID, ContextCapsule: inst.Action}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "completed",
			Summary: "followed the guidance",
			Handoff: codex.WorkerHandoff{
				ChangedFiles:           []string{"cmd/example.go"},
				OpenDecisions:          []string{"decided to " + inst.Action},
				NextWorkerInstructions: []string{"consulted this guidance: " + inst.Action},
				VerificationStatus:     "pass",
			},
		},
	}
	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	reached := guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if !reached[guidanceApplicationStateConsulted] {
		t.Fatalf("expected consulted to be recorded, reached=%v", reached)
	}
	if !reached[guidanceApplicationStateActedOn] {
		t.Fatalf("expected acted_on to be recorded (corroborated by this worker's own changed files and decision), reached=%v", reached)
	}
}

func TestUncorroboratedClaimIsRecordedAsUnverified(t *testing.T) {
	inst := setupGuidanceClaimFixture(t, "check pkg/colony/context_ranking.go before assuming score-based trim order")
	const phaseID = 1

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "continue", Phase: phaseID, ContextCapsule: inst.Action}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "completed",
			Summary: "claims to have used the guidance but leaves no corroborating evidence",
			Handoff: codex.WorkerHandoff{
				NextWorkerInstructions: []string{"consulted this guidance: " + inst.Action},
				VerificationStatus:     "pass",
			},
		},
	}
	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	reached := guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if reached[guidanceApplicationStateConsulted] {
		t.Fatalf("expected consulted NOT to be recorded for an uncorroborated claim, reached=%v", reached)
	}
	if reached[guidanceApplicationStateIgnored] {
		t.Fatalf("expected ignored NOT to be recorded directly by the worker-outcome fan-out, reached=%v", reached)
	}

	var file recruitmentCreditFile
	if err := store.LoadJSON(recruitmentCreditPath, &file); err != nil {
		t.Fatalf("read credit store: %v", err)
	}
	found := false
	for _, claim := range file.GuidanceClaims {
		if claim.GuidanceID == inst.ID && claim.Phase == phaseID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an unverified claim record for guidance %s on phase %d, got %+v", inst.ID, phaseID, file.GuidanceClaims)
	}
}

func TestRenderedAndUnusedBecomesIgnoredAtPhaseClose(t *testing.T) {
	inst := setupGuidanceClaimFixture(t, "run go vet ./cmd/ before go build ./cmd/aether to catch type errors early")
	const phaseID = 1

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "continue", Phase: phaseID, ContextCapsule: inst.Action}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "completed",
			Summary: "unrelated work",
			Handoff: codex.WorkerHandoff{VerificationStatus: "pass"},
		},
	}
	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	reached := guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if !reached[guidanceApplicationStateRendered] {
		t.Fatalf("expected rendered before phase close, reached=%v", reached)
	}
	if reached[guidanceApplicationStateIgnored] {
		t.Fatalf("expected ignored NOT to be recorded before phase close, reached=%v", reached)
	}

	recordPhaseApplicationCredit(phaseID)

	reached = guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if !reached[guidanceApplicationStateIgnored] {
		t.Fatalf("expected ignored to be recorded at phase close for rendered-and-unused guidance, reached=%v", reached)
	}
}

func TestContradictedIsRecordedFromARecordedDecision(t *testing.T) {
	inst := setupGuidanceClaimFixture(t, "always run go vet ./cmd/ before go test ./cmd/ here")
	const phaseID = 1

	dispatch := codex.WorkerDispatch{WorkerName: "Mason-1", Caste: "builder", Workflow: "continue", Phase: phaseID, ContextCapsule: inst.Action}
	result := codex.DispatchResult{
		WorkerName: "Mason-1",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Mason-1", Caste: "builder", Status: "completed",
			Summary: "chose to skip the guidance this time",
			Handoff: codex.WorkerHandoff{
				OpenDecisions:      []string{"went " + guidanceContradictionMarker + inst.Action},
				VerificationStatus: "pass",
			},
		},
	}
	if err := recordDispatchWorkerOutcome(dispatch, result); err != nil {
		t.Fatalf("recordDispatchWorkerOutcome: %v", err)
	}

	reached := guidanceApplicationRecordedStates(t, inst.ID, phaseID)
	if !reached[guidanceApplicationStateConsulted] {
		t.Fatalf("expected consulted to be recorded ahead of contradicted, reached=%v", reached)
	}
	if !reached[guidanceApplicationStateContradicted] {
		t.Fatalf("expected contradicted to be recorded from the worker's own recorded decision, reached=%v", reached)
	}
}

// forbiddenWorkerFreeTextFields are the WorkerHandoff fields carrying the
// worker's own prose. corroborateGuidanceClaim may touch
// facts.Handoff.ChangedFiles (a durable list of paths, not prose) and may
// call deriveBuildKnowledgeDeltas / loadLatestBuildAttempt (runtime-derived,
// sanitized data), but must never reference one of these fields directly.
var forbiddenWorkerFreeTextFields = map[string]bool{
	"NextWorkerInstructions": true,
	"DoNotRepeat":            true,
	"OpenDecisions":          true,
	"Assumptions":            true,
	"KnownFailures":          true,
	"Summary":                true,
}

// isFactsHandoffSelector reports whether expr is exactly the two-level
// selector `facts.Handoff` -- the base every forbidden field name is checked
// under. Scoping to this exact base (rather than flagging any `.Summary` /
// `.OpenDecisions` anywhere) is what lets a legitimately different struct's
// OWN field of the same name (e.g. buildAttemptKnowledgeDelta.Summary, a
// runtime-derived, sanitized value corroborateGuidanceClaim is allowed to
// read) pass cleanly.
func isFactsHandoffSelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Handoff" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "facts"
}

func corroborateGuidanceClaimFreeTextViolations(fset *token.FileSet, file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "corroborateGuidanceClaim" || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if !forbiddenWorkerFreeTextFields[sel.Sel.Name] {
				return true
			}
			if !isFactsHandoffSelector(sel.X) {
				return true
			}
			violations = append(violations, fmt.Sprintf(
				"%s: corroborateGuidanceClaim references facts.Handoff.%s directly -- must go through a runtime-derived source instead",
				fset.Position(sel.Pos()).String(), sel.Sel.Name,
			))
			return true
		})
	}
	return violations
}

func TestCorroborationNeverReadsTheWorkersOwnText(t *testing.T) {
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

	violations := corroborateGuidanceClaimFreeTextViolations(fset, file)
	if len(violations) != 0 {
		t.Fatalf("corroborateGuidanceClaim touches the worker's own free-text field(s) directly:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic direct free-text read is caught", func(t *testing.T) {
		src := `package cmd

func corroborateGuidanceClaim(claim guidanceClaim, facts workerOutcomeFacts) (guidanceApplicationState, string, bool) {
	for _, s := range facts.Handoff.NextWorkerInstructions {
		_ = s
	}
	return "", "", false
}
`
		fset2 := token.NewFileSet()
		syntheticFile, parseErr := parser.ParseFile(fset2, "fixture_corroborate.go", src, 0)
		if parseErr != nil {
			t.Fatalf("parse fixture: %v", parseErr)
		}
		fixtureViolations := corroborateGuidanceClaimFreeTextViolations(fset2, syntheticFile)
		if len(fixtureViolations) == 0 {
			t.Fatal("scanner failed to detect a synthetic direct free-text read inside corroborateGuidanceClaim")
		}
	})
}

// ---------------------------------------------------------------------------
// Task 3 (204-06-PLAN.md): a lesson must have actually helped at least once
// before it reaches the shared instruction file.
// ---------------------------------------------------------------------------

// promoteRealInstinctForQueenEligibility mirrors promoteRealInstinct
// (cmd/instinct_application_test.go) but with ObservationCount: 2 --
// matching CLAUDE.md's own documented auto-promotion threshold ("Pattern
// observations need 2 captures to trigger; confidence starts at 0.75") --
// so the resulting instinct's starting confidence clears
// queenPromotionConfidenceFloor on its own, and this test's three rounds
// isolate exactly the ONE variable the plan's behaviour line names (whether
// one of those three applications was helpful), rather than being confused
// with an unrelated confidence deficit from promoteRealInstinct's own
// single-observation default.
func promoteRealInstinctForQueenEligibility(t *testing.T, s *storage.Store, content, wisdomType string) colony.InstinctEntry {
	t.Helper()
	bus := events.NewBus(s, events.DefaultConfig())
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte(content + ":" + wisdomType))
	obs := colony.Observation{
		ContentHash:      "sha256:" + hex.EncodeToString(hash[:]),
		Content:          content,
		WisdomType:       wisdomType,
		ObservationCount: 2,
		FirstSeen:        events.FormatTimestamp(now),
		LastSeen:         events.FormatTimestamp(now),
		Colonies:         []string{"test-colony"},
		SourceType:       "success_pattern",
		EvidenceType:     "single_phase",
	}
	svc := memory.NewPromoteService(s, bus)
	result, err := svc.Promote(context.Background(), obs, "test-colony")
	if err != nil {
		t.Fatalf("promote real instinct for %q: %v", content, err)
	}
	return result.Instinct
}

// runQueenEligibilityColony drives one real, isolated colony through three
// real phase-end rounds of delivery + application recording for one real
// promoted instinct, seeding a genuinely helpful application every round
// only when seedHelpful is true, then runs the real
// memory.ConsolidationService.Run (never a hand-typed ConsolidationResult)
// and returns its eligibility verdict for that one instinct.
func runQueenEligibilityColony(t *testing.T, seedHelpful bool) (eligible []string, declined []memory.QueenDeclinedReason, instinctID string) {
	t.Helper()
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	inst := promoteRealInstinctForQueenEligibility(t, s, "run go vet ./cmd/ before go test ./cmd/ to catch lint failures early", "pattern")

	for round := 1; round <= 3; round++ {
		if recordInstinctDeliveries(round, "continue", inst.Action) == 0 {
			t.Fatalf("round %d: expected a new delivery to be recorded", round)
		}
		if seedHelpful {
			seedOneHelpfulApplicationCreditBuildAttempt(t, round)
		}
		recordPhaseApplicationCredit(round)
		recordInstinctApplicationsForPhase(round)
	}

	bus := events.NewBus(s, events.DefaultConfig())
	svc := memory.NewConsolidationService(s, bus, "", "test-colony")
	result, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("consolidation Run: %v", err)
	}
	return result.QueenEligible, result.QueenDeclined, inst.ID
}

func containsInstinctID(ids []string, id string) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func declinedReasonFor(declined []memory.QueenDeclinedReason, id string) (string, bool) {
	for _, d := range declined {
		if d.InstinctID == id {
			return d.Reason, true
		}
	}
	return "", false
}

func TestPromotionRequiresAHelpfulApplication(t *testing.T) {
	noHelpfulEligible, noHelpfulDeclined, noHelpfulID := runQueenEligibilityColony(t, false)
	if containsInstinctID(noHelpfulEligible, noHelpfulID) {
		t.Fatalf("expected %s NOT to be queen-eligible with three applications and none helpful, eligible=%v", noHelpfulID, noHelpfulEligible)
	}
	if _, found := declinedReasonFor(noHelpfulDeclined, noHelpfulID); !found {
		t.Fatalf("expected %s to be named in QueenDeclined, got %+v", noHelpfulID, noHelpfulDeclined)
	}

	oneHelpfulEligible, _, oneHelpfulID := runQueenEligibilityColony(t, true)
	if !containsInstinctID(oneHelpfulEligible, oneHelpfulID) {
		t.Fatalf("expected %s to be queen-eligible with three applications, one of them helpful, eligible=%v", oneHelpfulID, oneHelpfulEligible)
	}
}

func TestDeclinedPromotionNamesItsReason(t *testing.T) {
	_, declined, instinctID := runQueenEligibilityColony(t, false)
	reason, found := declinedReasonFor(declined, instinctID)
	if !found {
		t.Fatalf("expected %s to be named in QueenDeclined, got %+v", instinctID, declined)
	}
	if !strings.Contains(reason, "helpful") {
		t.Fatalf("declined reason %q does not name the missing helpful-application condition", reason)
	}
}

// findQueenEligibilityCondition returns the *ast.IfStmt whose body appends
// to result.QueenEligible -- the exact eligibility condition
// TestPromotionThresholdsAreNamedConstants scans for numeric literals.
func findQueenEligibilityCondition(file *ast.File) *ast.IfStmt {
	var found *ast.IfStmt
	ast.Inspect(file, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		appendsQueenEligible := false
		ast.Inspect(ifStmt.Body, func(n2 ast.Node) bool {
			call, ok := n2.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "append" || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Args[0].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name == "QueenEligible" {
				appendsQueenEligible = true
			}
			return true
		})
		if appendsQueenEligible {
			found = ifStmt
		}
		return true
	})
	return found
}

// numericLiteralsIn returns every integer/float literal anywhere inside
// expr.
func numericLiteralsIn(expr ast.Expr) []*ast.BasicLit {
	var lits []*ast.BasicLit
	ast.Inspect(expr, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok {
			return true
		}
		if lit.Kind == token.INT || lit.Kind == token.FLOAT {
			lits = append(lits, lit)
		}
		return true
	})
	return lits
}

func TestPromotionThresholdsAreNamedConstants(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, "pkg", "memory", "consolidate.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse consolidate.go: %v", err)
	}
	cond := findQueenEligibilityCondition(file)
	if cond == nil {
		t.Fatal("fixture is broken: could not find the QueenEligible append's guarding if-condition in pkg/memory/consolidate.go")
	}
	if lits := numericLiteralsIn(cond.Cond); len(lits) != 0 {
		var msgs []string
		for _, lit := range lits {
			msgs = append(msgs, fmt.Sprintf("%s: literal %s", fset.Position(lit.Pos()).String(), lit.Value))
		}
		t.Fatalf("eligibility condition contains numeric literal(s), want named constants only:\n%s", strings.Join(msgs, "\n"))
	}

	t.Run("a synthetic literal is caught and reports its position", func(t *testing.T) {
		src := `package memory

func f() {
	if inst.Confidence >= 0.75 && summary.Applications >= 3 {
		result.QueenEligible = append(result.QueenEligible, inst.ID)
	}
}
`
		fset2 := token.NewFileSet()
		syntheticFile, parseErr := parser.ParseFile(fset2, "fixture_consolidate.go", src, 0)
		if parseErr != nil {
			t.Fatalf("parse fixture: %v", parseErr)
		}
		syntheticCond := findQueenEligibilityCondition(syntheticFile)
		if syntheticCond == nil {
			t.Fatal("fixture is broken: could not find the synthetic QueenEligible append's guarding if-condition")
		}
		lits := numericLiteralsIn(syntheticCond.Cond)
		if len(lits) != 2 {
			t.Fatalf("scanner found %d numeric literal(s) in the synthetic fixture, want 2 (0.75 and 3): %v", len(lits), lits)
		}
		for _, lit := range lits {
			if fset2.Position(lit.Pos()).Line == 0 {
				t.Fatalf("literal %s reports no position", lit.Value)
			}
		}
	})
}
