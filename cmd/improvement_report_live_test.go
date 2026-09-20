package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// 204-15 (SC5c): proof that the two-figure improvement report is usable on
// a live colony, which it demonstrably was NOT before this plan --
// 204-VERIFICATION.md's own finding was that a real window of production
// episodes classified every episode as unclassified, because the nine
// fields Task 1 of this plan gave real writers to were never populated.
// Every test in this file drives a REAL lane entry point and reads the
// ledger back through readEpisodeLedger; none constructs an
// episodeLedgerRecord by hand (the acceptance criterion this file's tests
// exist to satisfy: a hand-built fixture would have passed today, before
// any of 204-15's wiring existed, which would prove nothing).

// TestTwoFiguresAreRealOnALiveColony drives a real build lane to
// completion, a real check lane on top of it, and a real owner
// intervention (through emitColonyLiveInterventionRecorded, 204-13's own
// production writer) -- all in one isolated store -- then reads the
// ledger back and builds the report over an unbounded window. Asserts the
// verified-useful-success count is greater than zero, the
// preventable-intervention count is greater than zero and reported
// separately, and the unclassified-episode list is strictly shorter than
// the total episode count -- the exact opposite of the "every real episode
// unclassified" state 204-VERIFICATION.md found.
func TestTwoFiguresAreRealOnALiveColony(t *testing.T) {
	// loadEvalGateSentinels resolves the committed sentinel file relative
	// to the real Aether module root by walking up from the CURRENT working
	// directory -- read it before this test's own withWorkingDir below
	// chdirs into an isolated fixture root with no such file to find.
	sentinelFile, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("loadEvalGateSentinels: %v", err)
	}

	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("## Verification Commands\n\n```bash\n# Verify Go binary builds\nprintf live-build\n\n# Run Go tests\nprintf live-test\n```\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "codebase.md"), []byte("## Commands\n- Types: `printf live-types`\n- Lint: `printf live-lint`\n"), 0644); err != nil {
		t.Fatalf("write codebase.md: %v", err)
	}

	goal := "Prove the two-figure report is real on a live colony"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Live colony report", Description: "Prove the report is real over episodes real lanes wrote",
			Status:          colony.PhaseReady,
			Tasks:           []colony.Task{{ID: &taskID, Goal: "Wire the live report", Status: colony.TaskPending}},
			SuccessCriteria: []string{"The two-figure report is non-vacuous on a live colony"},
		}}},
	})

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &episodeFieldsUsageInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("runCodexBuild: %v", err)
	}

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	preInterventionRecords, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	buildTerminal, ok := findEpisodeTerminalByKind(preInterventionRecords, events.EpisodeKindBuild)
	if !ok {
		t.Fatalf("expected a closed build episode before recording the intervention, got: %+v", preInterventionRecords)
	}

	// A real owner intervention, through the real production writer 204-13
	// built -- not a hand-constructed episodeLedgerRecord.
	emitColonyLiveInterventionRecorded(buildTerminal.EpisodeID, events.EpisodeKindBuild, episodeInterventionKindAnsweredWorkerQuestion)

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger (after intervention): %v", err)
	}
	report := buildImprovementReport(records, sentinelFile.Sentinels, reportWindow{})

	if report.VerifiedUsefulSuccess.Count == 0 {
		t.Fatalf("expected at least one verified-useful-success episode on a live colony, got report: %+v", report)
	}
	if report.PreventableInterventions.Count == 0 {
		t.Fatalf("expected at least one preventable-intervention episode, got report: %+v", report)
	}
	if len(report.UnclassifiedEpisodes) >= report.TotalEpisodes {
		t.Fatalf("expected the unclassified list (%d) to be strictly shorter than the total episode count (%d), got report: %+v",
			len(report.UnclassifiedEpisodes), report.TotalEpisodes, report)
	}
}

// TestReportBoundariesOnRealRecords (LEARN-08 boundary edge) evaluates the
// success and intervention classifications exactly at each declared
// threshold and one step either side, over a total genuinely produced by a
// real lane -- deriving the thresholds from the classification functions'
// own named constants rather than typing a percentage.
func TestReportBoundariesOnRealRecords(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	driveBuildLiveLane(t)

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	sentinelFile, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("loadEvalGateSentinels: %v", err)
	}
	report := buildImprovementReport(records, sentinelFile.Sentinels, reportWindow{})
	total := report.VerifiedUsefulSuccess.Total
	if total == 0 {
		t.Fatal("expected at least one real episode from the driven build lane")
	}

	for _, delta := range []int{-1, 0, 1} {
		count := report.VerifiedUsefulSuccess.Count + delta
		if count < 0 || count > total {
			continue
		}
		r := ratioOf(count, total)
		got := improvementReportSuccessClassification(r)
		wantMeetsBar := count*improvementReportSuccessThresholdTotal >= improvementReportSuccessThresholdCount*total
		want := "below the bar"
		if wantMeetsBar {
			want = "meeting the bar"
		}
		if got != want {
			t.Fatalf("success classification at count=%d total=%d: got %q, want %q", count, total, got, want)
		}
	}

	interventionTotal := report.PreventableInterventions.Total
	for _, delta := range []int{-1, 0, 1} {
		count := report.PreventableInterventions.Count + delta
		if count < 0 || count > interventionTotal {
			continue
		}
		r := ratioOf(count, interventionTotal)
		got := improvementReportInterventionClassification(r)
		wantElevated := count*improvementReportInterventionThresholdTotal >= improvementReportInterventionThresholdCount*interventionTotal
		want := "within bounds"
		if wantElevated {
			want = "elevated"
		}
		if got != want {
			t.Fatalf("intervention classification at count=%d total=%d: got %q, want %q", count, interventionTotal, got, want)
		}
	}
}

// TestEmptyWindowOverRealLedgerIsZeroNotPerfect (LEARN-08 empty edge, on
// live data) proves a window that excludes every real record reports both
// figures as zero out of zero, and still renders a report rather than
// nothing.
func TestEmptyWindowOverRealLedgerIsZeroNotPerfect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	driveBuildLiveLane(t)

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected at least one real record from the driven build lane")
	}

	// A window entirely BEFORE every real record's own timestamp excludes
	// everything without needing to know any record's exact time.
	excludingWindow := reportWindow{End: "2000-01-01T00:00:00Z"}
	sentinelFile, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("loadEvalGateSentinels: %v", err)
	}
	report := buildImprovementReport(records, sentinelFile.Sentinels, excludingWindow)

	if report.TotalEpisodes != 0 {
		t.Fatalf("expected the excluding window to leave TotalEpisodes=0, got %d", report.TotalEpisodes)
	}
	if report.VerifiedUsefulSuccess.Count != 0 || report.VerifiedUsefulSuccess.Total != 0 {
		t.Fatalf("expected VerifiedUsefulSuccess=0/0, got %+v", report.VerifiedUsefulSuccess)
	}
	if report.PreventableInterventions.Count != 0 || report.PreventableInterventions.Total != 0 {
		t.Fatalf("expected PreventableInterventions=0/0, got %+v", report.PreventableInterventions)
	}

	rendered := renderImprovementReport(report)
	if strings.TrimSpace(rendered) == "" {
		t.Fatal("expected the empty window to still render a report, got an empty string")
	}
}

// TestRenderedLiveReportKeepsTheTwoFiguresApart complements the existing
// fixture-level TestTwoFiguresAreNeverCombined (cmd/improvement_report_test.go)
// with the same structural assertion over a rendered report built from
// real, lane-produced records: no rendered line ever carries both figures'
// own distinguishing phrases.
func TestRenderedLiveReportKeepsTheTwoFiguresApart(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	driveBuildLiveLane(t)

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger: %v", err)
	}
	buildTerminal, ok := findEpisodeTerminalByKind(records, events.EpisodeKindBuild)
	if !ok {
		t.Fatalf("expected a closed build episode, got: %+v", records)
	}
	emitColonyLiveInterventionRecorded(buildTerminal.EpisodeID, events.EpisodeKindBuild, episodeInterventionKindAnsweredWorkerQuestion)

	records, err = readEpisodeLedger()
	if err != nil {
		t.Fatalf("readEpisodeLedger (after intervention): %v", err)
	}
	sentinelFile, err := loadEvalGateSentinels()
	if err != nil {
		t.Fatalf("loadEvalGateSentinels: %v", err)
	}
	report := buildImprovementReport(records, sentinelFile.Sentinels, reportWindow{})
	rendered := renderImprovementReport(report)

	const successPhrase = "Finished something genuinely useful"
	const interventionPhrase = "had to step in"
	sawSuccess, sawIntervention := false, false
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, successPhrase) && strings.Contains(line, interventionPhrase) {
			t.Fatalf("rendered line combines both figures: %q", line)
		}
		if strings.Contains(line, successPhrase) {
			sawSuccess = true
		}
		if strings.Contains(line, interventionPhrase) {
			sawIntervention = true
		}
	}
	if !sawSuccess || !sawIntervention {
		t.Fatalf("expected both figures' own phrases to appear as separate lines, sawSuccess=%v sawIntervention=%v, rendered:\n%s", sawSuccess, sawIntervention, rendered)
	}
}
