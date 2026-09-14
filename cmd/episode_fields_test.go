package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// 204-15 (SC3a/SC5c): proof that the nine previously writerless episode
// fields now have real production writers on the build lane and on BOTH
// check lanes, that an unreported figure stays absent rather than a
// fabricated zero, that an estimated figure is never presentable as a
// measured cost, and that the ordering/replay guarantees the ledger
// already made still hold with these fields populated.

// findEpisodeTerminalByKind returns the first episode_closed record in
// records whose EpisodeKind matches kind. Test fixtures in this file each
// run exactly one lane in an isolated store, so at most one such record
// ever exists.
func findEpisodeTerminalByKind(records []episodeLedgerRecord, kind string) (episodeLedgerRecord, bool) {
	for _, id := range episodeLedgerEpisodeIDs(records) {
		terminal, ok := episodeLedgerTerminalRecord(records, id)
		if ok && terminal.EpisodeKind == kind {
			return terminal, true
		}
	}
	return episodeLedgerRecord{}, false
}

// episodeFieldsUsageInvoker is a real, in-process codex.WorkerInvoker
// (satisfying the same three-method interface build_attempt_test.go's own
// fixture invokers do) that reports a genuine, provider-sourced usage
// figure -- unlike codex.FakeInvoker, which never sets Usage at all. Swapped
// in via the same newCodexWorkerInvoker package-level seam
// build_attempt_test.go already uses, so the build lane under test is the
// real runCodexBuildWithOptions, not a synthetic shortcut.
type episodeFieldsUsageInvoker struct{}

func (i *episodeFieldsUsageInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "episode-fields fixture worker completed",
		Handoff: codex.NormalizeWorkerHandoff(config.Root, codex.WorkerHandoff{
			VerificationStatus:     "not_run",
			NextWorkerInstructions: []string{"episode-fields fixture: no repo changes"},
		}),
		Usage: codex.WorkerUsage{
			InputTokens:  1000,
			OutputTokens: 500,
			TotalTokens:  1500,
			USDCost:      0.05,
			Source:       codex.UsageSourceProvider,
		},
	}, nil
}

func (i *episodeFieldsUsageInvoker) IsAvailable(context.Context) bool { return true }
func (i *episodeFieldsUsageInvoker) ValidateAgent(string) error       { return nil }

// TestBuildEpisodeRecordsItsOwnFacts drives the real direct build lane
// (runCodexBuildWithOptions via runCodexBuild) to completion and asserts its
// closed episode carries a non-empty hard-gate result map, a non-empty
// evidence identifier list, a real usage value, and a policy/runtime
// version -- the exact fields Task 1's <behavior> Test 1 names.
func TestBuildEpisodeRecordsItsOwnFacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Prove the build episode records its own facts"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Episode fields build", Description: "Prove build's own episode close carries real facts",
			Status:          colony.PhaseReady,
			Tasks:           []colony.Task{{ID: &taskID, Goal: "Wire the episode fields", Status: colony.TaskPending}},
			SuccessCriteria: []string{"Build closes its episode with real, non-fabricated facts"},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &episodeFieldsUsageInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("runCodexBuild: %v", err)
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	terminal, ok := findEpisodeTerminalByKind(records, events.EpisodeKindBuild)
	if !ok {
		t.Fatalf("expected a closed build episode in the ledger, got records: %+v", records)
	}
	if len(terminal.HardGateResults) == 0 {
		t.Fatalf("expected a non-empty HardGateResults map, got %#v", terminal.HardGateResults)
	}
	if len(terminal.EvidenceIDs) == 0 {
		t.Fatalf("expected a non-empty EvidenceIDs list, got %#v", terminal.EvidenceIDs)
	}
	if terminal.Usage == nil {
		t.Fatal("expected a non-nil Usage")
	}
	if terminal.RuntimeVersion == "" || terminal.PolicyVersion == "" {
		t.Fatalf("expected non-empty runtime/policy version, got record %+v", terminal)
	}
}

// newCheckEpisodeFixture seeds a real, durable build attempt (MakeLatest
// true, so loadLatestBuildAttempt resolves it) via commitTestBuildStart --
// the SAME fixture builder application_evidence_test.go's own
// recordPhaseApplicationCredit tests use -- then documents fast,
// deterministic verification commands (mirroring continueLiveWiringFixture,
// cmd/live_lane_wiring_test.go) so the deterministic floor actually runs and
// writes gates.json/verification.json, giving both check lanes real facts
// to close their episode with.
func newCheckEpisodeFixture(t *testing.T, phaseName, goal string) string {
	t.Helper()
	taskID := "1.1"
	seedPhase := colony.Phase{
		ID:     1,
		Name:   phaseName,
		Status: colony.PhaseInProgress,
		Tasks:  []colony.Task{{ID: &taskID, Goal: "Verify using documented commands", Status: colony.TaskInProgress}},
	}
	seedState := colony.ColonyState{
		Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{seedPhase},
		},
	}

	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: time.Now().UTC(),
		SelectedTasks: []string{taskID},
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-1", Caste: "builder", TaskID: taskID, CoveredTaskIDs: []string{taskID}, Status: "completed"},
		},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(r string) {
			createTestColonyState(t, filepath.Join(r, ".aether", "data"), seedState)
		},
	})

	root := fixture.Root
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf live-build\n\n# Run Go tests\nprintf live-test\n```\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "codebase.md"), []byte("## Commands\n- Types: `printf live-types`\n- Lint: `printf live-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}
	seedContinueBuildPacket(t, filepath.Join(root, ".aether", "data"), 1, phaseName, goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-1", Task: "Verify using documented commands", Status: "completed", TaskID: taskID},
	})
	withWorkingDir(t, root)
	return root
}

// TestCheckEpisodeRecordsItsOwnFacts drives the real NATIVE check lane
// (runCodexContinue, reached via `aether continue`) to completion and
// asserts its closed episode carries gate results keyed by the gate names
// the check actually ran, evidence identifiers, and acceptance/evaluator
// digests -- Task 1's <behavior> Test 2.
func TestCheckEpisodeRecordsItsOwnFacts(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	newCheckEpisodeFixture(t, "Episode fields native check", "Prove the native check episode records its own facts")

	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	terminal, ok := findEpisodeTerminalByKind(records, events.EpisodeKindContinue)
	if !ok {
		t.Fatalf("expected a closed continue episode in the ledger, got records: %+v", records)
	}
	if len(terminal.HardGateResults) == 0 {
		t.Fatalf("expected a non-empty HardGateResults map keyed by the gates the check actually ran, got %#v", terminal.HardGateResults)
	}
	if len(terminal.EvidenceIDs) == 0 {
		t.Fatalf("expected a non-empty EvidenceIDs list, got %#v", terminal.EvidenceIDs)
	}
	if terminal.AcceptanceDigest == "" {
		t.Fatal("expected a non-empty AcceptanceDigest")
	}
	if terminal.EvaluatorDigest == "" {
		t.Fatal("expected a non-empty EvaluatorDigest")
	}
}

// TestDelegateCheckEpisodeRecordsItsOwnFacts drives the real DELEGATE check
// lane (runCodexContinueFinalize) the same way
// TestExternalContinueAdvanceInvokesPhaseEndConsolidation
// (cmd/consolidation_lifecycle_test.go) does, and asserts it now opens AND
// closes a durable episode carrying the same fact set the native lane
// records -- Task 1's <behavior> Test 2b. This lane recorded no episode at
// all before this plan, so this test fails before the wiring exists.
func TestDelegateCheckEpisodeRecordsItsOwnFacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Prove the delegate check episode records its own facts"
	now := time.Now().UTC()
	taskID := "1.1"
	nextTaskID := "2.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1, BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID: 1, Name: "Episode fields delegate check", Description: "Prove the delegate lane's own episode close",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Advance durably", Status: colony.TaskInProgress}},
				},
				{
					ID: 2, Name: "Next phase", Status: colony.PhasePending,
					Tasks: []colony.Task{{ID: &nextTaskID, Goal: "Continue forward", Status: colony.TaskPending}},
				},
			},
		},
	})

	seedContinueBuildPacket(t, dataDir, 1, "Episode fields delegate check", goal, []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-delegate-1", Task: "Advance durably", Status: "completed", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-delegate-1", Task: "Independent verification before advancement", Status: "completed"},
	})

	planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan := planResult["continue_manifest"].(codexContinuePlanManifest)
	results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
	for _, dispatch := range plan.Dispatches {
		results = append(results, codexContinueExternalDispatch{
			Stage: dispatch.Stage, Wave: dispatch.Wave, Caste: dispatch.Caste, Name: dispatch.Name,
			Task: dispatch.Task, TaskID: dispatch.TaskID, Status: "completed",
			Summary:   dispatch.Name + " cleared episode-fields review",
			Artifacts: validCompletedReviewerArtifacts(t, dispatch.Caste),
			Handoff:   codex.WorkerHandoff{VerificationStatus: "pass", NextWorkerInstructions: []string{dispatch.Name + " found no blocking issues"}},
		})
	}
	completion := codexExternalContinueCompletion{ContinueManifest: &plan, Dispatches: results}

	result, _, _, _, _, _, err := runCodexContinueFinalize(root, completion, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	if advanced, _ := result["advanced"].(bool); !advanced {
		t.Fatalf("expected advanced:true (precondition for the episode to close as a real success), got %v", result)
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	terminal, ok := findEpisodeTerminalByKind(records, events.EpisodeKindContinue)
	if !ok {
		t.Fatalf("expected the delegate check lane to have opened and closed a durable continue episode, got records: %+v", records)
	}
	if terminal.AcceptanceDigest == "" || terminal.EvaluatorDigest == "" {
		t.Fatalf("expected non-empty acceptance/evaluator digests on the delegate lane's own episode, got record %+v", terminal)
	}
}

// TestEpisodeOutcomeIsRecordedFromBothCheckLanes is an AST-based call-graph
// guard, in the exact style of
// TestPhaseApplicationCreditIsReachedFromBothCheckLanes
// (cmd/application_evidence_test.go): parse every non-test file in cmd/,
// and fail by name if emitColonyLiveOutcomeRecorded is not transitively
// reachable from EACH of the two functions both check lanes enter
// (runCodexContinue, cmd/codex_continue.go; runCodexContinueFinalize,
// cmd/codex_continue_finalize.go). Carries the same anti-vacuity guards:
// fail if zero functions were indexed, fail if the target or either lane is
// not an indexed function, and a synthetic graph in which neither lane
// reaches the target is reported by name on both lanes.
func TestEpisodeOutcomeIsRecordedFromBothCheckLanes(t *testing.T) {
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

	const target = "emitColonyLiveOutcomeRecorded"
	lanes := []string{"runCodexContinue", "runCodexContinueFinalize"}

	if _, ok := g.calls[target]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", target, g.funcsIndexed)
	}
	for _, lane := range lanes {
		if _, ok := g.calls[lane]; !ok {
			t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", lane, g.funcsIndexed)
		}
	}

	if unreached := episodeOutcomeUnreachedLanes(g, lanes, target); len(unreached) > 0 {
		t.Fatalf("%s is not transitively reachable from: %v -- both check lanes must reach the episode outcome writer", target, unreached)
	}

	t.Run("a synthetic unreachable fixture is caught by name", func(t *testing.T) {
		synthetic := &cmdFuncGraph{calls: map[string]map[string]bool{
			"runCodexContinue":              {"someOtherHelper": true},
			"runCodexContinueFinalize":      {"anotherHelper": true},
			"emitColonyLiveOutcomeRecorded": {},
		}}
		unreached := episodeOutcomeUnreachedLanes(synthetic, lanes, target)
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

// episodeOutcomeUnreachedLanes returns every lane in lanes from which target
// is NOT transitively reachable in g, by name.
func episodeOutcomeUnreachedLanes(g *cmdFuncGraph, lanes []string, target string) []string {
	var unreached []string
	for _, lane := range lanes {
		reachable := reachableFrom(g, []string{lane})
		if !reachable[target] {
			unreached = append(unreached, lane)
		}
	}
	return unreached
}

// TestUnreportedUsageStaysAbsentNotZero (Task 1 Test 3) proves a dispatch
// with no reported usage at all leaves both Usage and ReportedCostUSD nil
// on the assembled facts, never a fabricated zero value.
func TestUnreportedUsageStaysAbsentNotZero(t *testing.T) {
	facts := &episodeCloseFacts{}
	facts.addUsage(codex.WorkerUsage{})
	if facts.Usage != nil {
		t.Fatalf("expected Usage to stay nil for an unreported dispatch, got %+v", facts.Usage)
	}
	if facts.ReportedCostUSD != nil {
		t.Fatalf("expected ReportedCostUSD to stay nil for an unreported dispatch, got %v", *facts.ReportedCostUSD)
	}
}

// TestEstimatedUsageIsNeverWrittenAsReportedCost (Task 1 Test 4) proves a
// usage value whose own Source marks it an estimate still contributes its
// token counts (a genuine record of what was estimated) but never its cost
// -- an estimate must never be presentable as a measurement.
func TestEstimatedUsageIsNeverWrittenAsReportedCost(t *testing.T) {
	facts := &episodeCloseFacts{}
	facts.addUsage(codex.WorkerUsage{
		InputTokens: 100, OutputTokens: 50, TotalTokens: 150,
		USDCost: 9.99, Source: codex.UsageSourceEstimate,
	})
	if facts.Usage == nil {
		t.Fatal("expected Usage to be populated from an estimate's own token counts")
	}
	if facts.Usage.TotalTokens != 150 {
		t.Fatalf("expected TotalTokens=150 from the estimate, got %d", facts.Usage.TotalTokens)
	}
	if facts.ReportedCostUSD != nil {
		t.Fatalf("expected ReportedCostUSD to stay nil for an estimated-only usage, got %v -- an estimate must never be written as a measured cost", *facts.ReportedCostUSD)
	}

	// A later, genuinely measured usage on the SAME facts must still be
	// able to contribute its cost -- the exclusion is per-contribution, not
	// a one-shot poison on the whole accumulator.
	facts.addUsage(codex.WorkerUsage{
		InputTokens: 10, OutputTokens: 5, TotalTokens: 15,
		USDCost: 0.01, Source: codex.UsageSourceProvider,
	})
	if facts.ReportedCostUSD == nil || *facts.ReportedCostUSD != 0.01 {
		t.Fatalf("expected ReportedCostUSD=0.01 from the one measured contribution, got %v", facts.ReportedCostUSD)
	}
}

// TestEmptyEpisodeIsUnclassifiedNotSuccessful (Task 1 Test 5, LEARN-02
// empty edge) proves an episode closed with no evidence, no gate results
// and no usage is classified unclassified by buildImprovementReport, never
// a verified success.
func TestEmptyEpisodeIsUnclassifiedNotSuccessful(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "episode-fields-empty-edge"
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: episodeID, EpisodeKind: events.EpisodeKindBuild,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("record open: %v", err)
	}
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: episodeID, EpisodeKind: events.EpisodeKindBuild,
		EndedAt: time.Now().UTC().Format(time.RFC3339), TerminalResult: "completed",
	}); err != nil {
		t.Fatalf("record close: %v", err)
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	report := buildImprovementReport(records, nil, reportWindow{})
	for _, id := range report.UnclassifiedEpisodes {
		if id == episodeID {
			return
		}
	}
	t.Fatalf("expected episode %q (no evidence, no gates, no usage) to be unclassified, got report: %+v", episodeID, report)
}

// TestEpisodeRecordOrderingIsTotalAndStable (Task 1 Test 6, LEARN-02
// ordering edge) proves two ledger records whose sort timestamps compare
// equal still sort into a total, stable order, identical across two runs
// over the same data.
func TestEpisodeRecordOrderingIsTotalAndStable(t *testing.T) {
	sameTimestamp := "2026-01-01T00:00:00Z"
	base := []episodeLedgerRecord{
		{RecordID: "episode:zzz", EpisodeID: "b", RecordKind: episodeLedgerRecordKindOpened, StartedAt: sameTimestamp},
		{RecordID: "episode:aaa", EpisodeID: "a", RecordKind: episodeLedgerRecordKindOpened, StartedAt: sameTimestamp},
	}

	first := append([]episodeLedgerRecord{}, base...)
	sortEpisodeLedgerRecords(first)
	second := append([]episodeLedgerRecord{}, base...)
	sortEpisodeLedgerRecords(second)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected 2 records after sort, got first=%d second=%d", len(first), len(second))
	}
	if first[0].RecordID != second[0].RecordID || first[1].RecordID != second[1].RecordID {
		t.Fatalf("sort order differs between two runs over identical data: first=%+v second=%+v", first, second)
	}
	// The tie-break is RecordID ascending: "episode:aaa" < "episode:zzz".
	if first[0].RecordID != "episode:aaa" {
		t.Fatalf("expected the RecordID tie-break to order episode:aaa first when timestamps compare equal, got %+v", first)
	}
}

// TestEpisodeCloseWithFactsIsStillReplaySafe (Task 1 Test 7) proves closing
// an episode twice with identical facts through emitColonyLiveOutcomeRecorded
// writes one record, not two -- the existing replay-identity rule still
// holds with the fields this plan added populated.
func TestEpisodeCloseWithFactsIsStillReplaySafe(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	episodeID := "episode-fields-replay-safe"
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: episodeID, EpisodeKind: events.EpisodeKindBuild,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("record open: %v", err)
	}

	record := episodeLedgerRecord{
		TerminalResult:  "completed",
		HardGateResults: map[string]bool{"forced-reviewer": true},
		EvidenceIDs:     []string{"effect:episode-fields-replay-safe"},
	}
	record.RuntimeVersion, record.PolicyVersion, record.EndedAt, record.ElapsedSeconds = episodeCloseBasics(episodeID, "completed")

	emitColonyLiveOutcomeRecorded(episodeID, events.EpisodeKindBuild, record)
	firstRecords, err := episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("episodeLedgerForEpisode: %v", err)
	}
	firstClosedCount := 0
	for _, r := range firstRecords {
		if r.RecordKind == episodeLedgerRecordKindClosed {
			firstClosedCount++
		}
	}
	if firstClosedCount != 1 {
		t.Fatalf("expected exactly 1 closed record after the first close, got %d", firstClosedCount)
	}

	// An IDENTICAL replay (same episode, same fields, same EndedAt/Elapsed
	// this time frozen to the first close's own values) must collapse
	// silently rather than append a second record.
	replay := record
	emitColonyLiveOutcomeRecorded(episodeID, events.EpisodeKindBuild, replay)

	secondRecords, err := episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("episodeLedgerForEpisode (after replay): %v", err)
	}
	secondClosedCount := 0
	for _, r := range secondRecords {
		if r.RecordKind == episodeLedgerRecordKindClosed {
			secondClosedCount++
		}
	}
	if secondClosedCount != 1 {
		t.Fatalf("expected an identical replay to still leave exactly 1 closed record, got %d", secondClosedCount)
	}
}
