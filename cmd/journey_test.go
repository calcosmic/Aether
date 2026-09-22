package cmd

// 207-01-PLAN.md Task 3: offline tests for the transcript parser, the
// failure classifier, the report writer and the gate verdict -- each
// proven by a fast test that needs no chat and no money. The transcript
// parser is proven against cmd/testdata/stop-hook/menu-command-transcript.jsonl,
// a genuinely captured Claude Code session (not a hand-typed fixture) --
// see deriveJourneyBashToolUseTranscript's own comment for why a fixture
// built in a shape the runtime cannot produce would be a false
// certificate.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const journeyRealTranscriptFixture = "menu-command-transcript.jsonl"

func journeyRealTranscriptPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join("testdata", "stop-hook", journeyRealTranscriptFixture)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("real transcript fixture missing at %s: %v", path, err)
	}
	return path
}

// TestJourneyTranscriptReadsRealCaptures proves journeyMenuCommandNames
// reads the shape Claude Code genuinely produces (verified against
// cmd/testdata/stop-hook/menu-command-transcript.jsonl, captured from a
// real /ant-status session -- see cmd/stop_hook_screen_test.go for this
// repo's existing precedent of reading the same fixture), not a shape
// someone typed by hand.
func TestJourneyTranscriptReadsRealCaptures(t *testing.T) {
	path := journeyRealTranscriptPath(t)

	names, err := journeyMenuCommandNames(path)
	if err != nil {
		t.Fatalf("journeyMenuCommandNames(%s): %v", path, err)
	}
	if !journeyContainsString(names, "/ant-status") {
		t.Fatalf("journeyMenuCommandNames(%s) = %v, want a list containing /ant-status", path, names)
	}
}

// deriveJourneyBashToolUseTranscript builds Bash-tool-use transcript
// fixtures by VARYING the real captured assistant tool_use line (index 2
// of cmd/testdata/stop-hook/menu-command-transcript.jsonl) rather than
// typing a plausible-looking JSON literal -- a fixture built in a shape
// the runtime cannot produce is a false certificate, not a weaker test
// (CLAUDE.md's Definition of Done). bashCopies controls how many times the
// real Bash tool_use line is repeated; nonBashCopies controls how many
// copies with the tool name swapped to something else are interleaved, so
// the "no Bash calls" fixture still carries a real, differently-shaped
// tool_use entry rather than an empty file.
func deriveJourneyBashToolUseTranscript(t *testing.T, bashCopies, nonBashCopies int) string {
	t.Helper()

	data, err := os.ReadFile(journeyRealTranscriptPath(t))
	if err != nil {
		t.Fatalf("read real transcript fixture: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected the real transcript to carry 4 lines, got %d", len(lines))
	}

	var assistantToolUseLine map[string]interface{}
	if err := json.Unmarshal([]byte(lines[2]), &assistantToolUseLine); err != nil {
		t.Fatalf("unmarshal real transcript line 2 (the assistant tool_use line): %v", err)
	}
	message, ok := assistantToolUseLine["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("real transcript line 2's message field is not an object: %#v", assistantToolUseLine["message"])
	}
	content, ok := message["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("real transcript line 2's message.content is not a non-empty array: %#v", message["content"])
	}
	block, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatalf("real transcript line 2's first content block is not an object: %#v", content[0])
	}
	if block["type"] != "tool_use" || block["name"] != "Bash" {
		t.Fatalf("real transcript line 2's first content block is not a Bash tool_use as expected: %#v", block)
	}

	var out []string
	for i := 0; i < bashCopies; i++ {
		out = append(out, lines[2])
	}
	for i := 0; i < nonBashCopies; i++ {
		variant := map[string]interface{}{}
		for k, v := range assistantToolUseLine {
			variant[k] = v
		}
		variantMessage := map[string]interface{}{}
		for k, v := range message {
			variantMessage[k] = v
		}
		variantBlock := map[string]interface{}{}
		for k, v := range block {
			variantBlock[k] = v
		}
		variantBlock["name"] = "Read" // a real, non-Bash tool -- proves the count is name-selective, not merely tool_use-selective
		variantMessage["content"] = []interface{}{variantBlock}
		variant["message"] = variantMessage
		encoded, err := json.Marshal(variant)
		if err != nil {
			t.Fatalf("marshal derived non-Bash transcript line: %v", err)
		}
		out = append(out, string(encoded))
	}

	path := filepath.Join(t.TempDir(), "derived-transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write derived transcript to %s: %v", path, err)
	}
	return path
}

func TestJourneyBashToolCallCountIsNameSelective(t *testing.T) {
	t.Run("no Bash tool_use block returns 0", func(t *testing.T) {
		path := deriveJourneyBashToolUseTranscript(t, 0, 1)
		count, err := journeyBashToolCallCount(path)
		if err != nil {
			t.Fatalf("journeyBashToolCallCount(%s): %v", path, err)
		}
		if count != 0 {
			t.Fatalf("journeyBashToolCallCount(%s) = %d, want 0 (the transcript carries a non-Bash tool_use block only)", path, count)
		}
	})

	t.Run("two Bash tool_use blocks returns 2", func(t *testing.T) {
		path := deriveJourneyBashToolUseTranscript(t, 2, 0)
		count, err := journeyBashToolCallCount(path)
		if err != nil {
			t.Fatalf("journeyBashToolCallCount(%s): %v", path, err)
		}
		if count != 2 {
			t.Fatalf("journeyBashToolCallCount(%s) = %d, want 2", path, count)
		}
	})
}

// TestJourneyFailureClassifierIsSmallAndExplicit proves classifyJourneyFailure
// recognizes exactly the small regex table (rate limit, overloaded, timed
// out, tolerating the hyphen and space spellings the Phase 206 proof
// script already tolerates) and nothing wider -- everything else,
// including the empty string, classifies as real.
func TestJourneyFailureClassifierIsSmallAndExplicit(t *testing.T) {
	transientCases := []string{
		"429 rate limit exceeded",
		"rate-limit exceeded",
		"ratelimit exceeded",
		"the service is overloaded right now",
		"OVERLOADED",
		"request timed out",
		"request timeout",
		"connection timed-out",
	}
	for _, text := range transientCases {
		if got := classifyJourneyFailure(text); got != journeyFailureTransient {
			t.Errorf("classifyJourneyFailure(%q) = %q, want %q", text, got, journeyFailureTransient)
		}
	}

	realCases := []string{
		"",
		"permission denied",
		"unknown flag: --frobnicate",
		"not authenticated",
		"exit status 1",
	}
	for _, text := range realCases {
		if got := classifyJourneyFailure(text); got != journeyFailureReal {
			t.Errorf("classifyJourneyFailure(%q) = %q, want %q", text, got, journeyFailureReal)
		}
	}
}

// TestJourneyReportRoundTrips proves writeJourneyReport then
// readJourneyReport returns the same report, with the schema version
// stamped even when the caller left it unset.
func TestJourneyReportRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "journey-report.json")
	report := journeyReport{
		Scope:         "whole-chain",
		Mode:          "live",
		StepsDeclared: 14,
		StepsExecuted: 14,
		Trials: []journeyTrial{
			{
				SessionID: "session-1",
				Caps: journeyCaps{
					MaxTurns:      6,
					MaxBudgetUSD:  1.00,
					BudgetCapped:  true,
					WallClockSecs: 300,
				},
				Steps: []journeyStepResult{
					{Name: "status", MenuCommand: "/ant-status", Status: "pass", BashToolCalls: 1},
				},
				Outcome: "pass",
			},
		},
		Verdict: "pass",
	}

	if err := writeJourneyReport(path, report); err != nil {
		t.Fatalf("writeJourneyReport(%s): %v", path, err)
	}

	readBack, err := readJourneyReport(path)
	if err != nil {
		t.Fatalf("readJourneyReport(%s): %v", path, err)
	}

	if readBack.SchemaVersion != journeyReportSchemaVersion {
		t.Fatalf("readJourneyReport(%s).SchemaVersion = %q, want %q", path, readBack.SchemaVersion, journeyReportSchemaVersion)
	}

	// Compare everything except SchemaVersion, which writeJourneyReport
	// stamps regardless of what the caller passed in.
	report.SchemaVersion = journeyReportSchemaVersion
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal expected report: %v", err)
	}
	readBackJSON, err := json.Marshal(readBack)
	if err != nil {
		t.Fatalf("marshal read-back report: %v", err)
	}
	if string(reportJSON) != string(readBackJSON) {
		t.Fatalf("round trip mismatch:\nwrote: %s\nread:  %s", reportJSON, readBackJSON)
	}
}

// TestJourneyGateRefusesAPartialRun proves journeyGateVerdict refuses a
// one-step report (naming the scope), refuses a report whose declared and
// executed step counts differ (naming both numbers), and accepts a genuine
// whole-chain live report whose counts match. 207-04-PLAN.md extended
// journeyGateVerdict with the trial-count and per-trial rules, so the
// declared/executed and "accepted" subtests below now exercise those rules
// against a full three-trial fixture (journeyPassingReportFixture) rather
// than a report with no trials at all, which the extended verdict now
// refuses for a different, earlier reason (too few trials) -- see
// TestJourneyGateVerdictRefusals for the seven rules individually.
func TestJourneyGateRefusesAPartialRun(t *testing.T) {
	t.Run("one-step scope is refused, naming the scope", func(t *testing.T) {
		r := journeyReport{Scope: "one-step", Mode: "live", StepsDeclared: 14, StepsExecuted: 1}
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected journeyGateVerdict to refuse a one-step report")
		}
		if !strings.Contains(err.Error(), "one-step") {
			t.Fatalf("journeyGateVerdict error does not name the scope: %v", err)
		}
	})

	t.Run("declared/executed mismatch is refused, naming both numbers", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.Trials[1].StepsExecuted = 9
		r.Trials[1].IncompleteAtStep = "build"
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected journeyGateVerdict to refuse a declared/executed mismatch")
		}
		if !strings.Contains(err.Error(), strconv.Itoa(r.Trials[1].StepsDeclared)) || !strings.Contains(err.Error(), "9") {
			t.Fatalf("journeyGateVerdict error does not name both step counts: %v", err)
		}
	})

	t.Run("a matching whole-chain live report is accepted", func(t *testing.T) {
		r := journeyPassingReportFixture()
		if err := journeyGateVerdict(r); err != nil {
			t.Fatalf("journeyGateVerdict refused a report that should pass: %v", err)
		}
	})
}

// journeyPassingReportFixture returns a genuinely passing whole-chain,
// live, three-trial report, shaped exactly the way the live harness
// actually populates a journeyReport: journeyRunOneTrial
// (cmd/journey_live_test.go) always sets every field this fixture sets on
// each trial, and journeyEvaluateExpectedRed (cmd/journey.go) always sets
// every field this fixture sets on each expected-red result. A fixture
// shaped any other way -- a field the real harness never populates, a
// value it could never produce -- would be a false certificate, not a
// weaker test (CLAUDE.md's Definition of Done: "derive fixture values the
// way the runtime derives them").
func journeyPassingReportFixture() journeyReport {
	stepNames := journeyStepNames()
	makeSteps := func() []journeyStepResult {
		out := make([]journeyStepResult, 0, len(stepNames))
		for _, n := range stepNames {
			cmd, _ := journeyStepMenuCommand(journeyStep(n))
			out = append(out, journeyStepResult{Name: n, MenuCommand: cmd, Status: "pass", BashToolCalls: 1})
		}
		return out
	}
	makeTrial := func(index int) journeyTrial {
		return journeyTrial{
			Index:         index,
			SessionID:     "session-" + strconv.Itoa(index),
			StartedAt:     "2026-09-22T00:00:00Z",
			EndedAt:       "2026-09-22T00:20:00Z",
			Caps:          journeyCaps{MaxTurns: 6, MaxBudgetUSD: 1.00, BudgetCapped: true, WallClockSecs: 300},
			Steps:         makeSteps(),
			StepsDeclared: len(stepNames),
			StepsExecuted: len(stepNames),
			Outcome:       string(journeyTrialPassed),
		}
	}
	return journeyReport{
		Scope:         "whole-chain",
		Mode:          "live",
		StepsDeclared: len(stepNames),
		StepsExecuted: len(stepNames),
		Trials:        []journeyTrial{makeTrial(0), makeTrial(1), makeTrial(2)},
		ExpectedRed: []journeyExpectedRedResult{
			{
				ID:       "status-card-advises-a-menu-command-that-does-not-exist",
				ClosedBy: "Phase 208 (UED-13)",
				Result:   "still-red",
				Detail:   "the status card advises `aether midden-review`, which has no menu wrapper",
			},
		},
		Verdict: "pending",
	}
}

// TestJourneyGateVerdictRefusals proves each of journeyGateVerdict's seven
// named refusal rules individually: a synthetic report built from
// journeyPassingReportFixture, mutated in exactly one way per subtest, is
// refused, and the refusal message names the offending value.
func TestJourneyGateVerdictRefusals(t *testing.T) {
	t.Run("scope must be whole-chain", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.Scope = "one-step"
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal for a non-whole-chain scope")
		}
		if !strings.Contains(err.Error(), "one-step") {
			t.Fatalf("refusal does not name the offending scope: %v", err)
		}
	})

	t.Run("mode must be live", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.Mode = "dry-run"
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal for a non-live mode")
		}
		if !strings.Contains(err.Error(), "dry-run") {
			t.Fatalf("refusal does not name the offending mode: %v", err)
		}
	})

	t.Run("at least three trials", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.Trials = r.Trials[:2]
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal for fewer than three trials")
		}
		if !strings.Contains(err.Error(), "2") {
			t.Fatalf("refusal does not name the actual trial count: %v", err)
		}
	})

	t.Run("every trial's declared step count equals its executed count", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.Trials[1].StepsExecuted = r.Trials[1].StepsDeclared - 3
		r.Trials[1].IncompleteAtStep = "build"
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal for a trial that stopped part way")
		}
		if !strings.Contains(err.Error(), strconv.Itoa(r.Trials[1].StepsDeclared)) || !strings.Contains(err.Error(), strconv.Itoa(r.Trials[1].StepsExecuted)) {
			t.Fatalf("refusal does not name both step counts: %v", err)
		}
	})

	t.Run("no trial may be real-failure, and not every trial may fail", func(t *testing.T) {
		realFailure := journeyPassingReportFixture()
		realFailure.Trials[1].Outcome = string(journeyTrialRealFailure)
		realFailure.Trials[1].Steps[3].Status = "fail"
		realFailure.Trials[1].Steps[3].FailureKind = string(journeyFailureReal)
		err := journeyGateVerdict(realFailure)
		if err == nil {
			t.Fatal("expected a refusal when any trial is a real failure")
		}
		if !strings.Contains(err.Error(), "1") {
			t.Fatalf("refusal does not name the offending trial index: %v", err)
		}

		allFlaky := journeyPassingReportFixture()
		for i := range allFlaky.Trials {
			allFlaky.Trials[i].Outcome = string(journeyTrialFlakyFailure)
			allFlaky.Trials[i].Steps[0].Status = "fail"
			allFlaky.Trials[i].Steps[0].FailureKind = string(journeyFailureTransient)
		}
		if err := journeyGateVerdict(allFlaky); err == nil {
			t.Fatal("expected a refusal when every trial failed, even if every failure classified transient -- three flaky failures in a row is itself a real failure")
		}
	})

	t.Run("every expected-red case must be still-red", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.ExpectedRed[0].Result = "now-green"
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal when a registered case reports now-green")
		}
		if !strings.Contains(err.Error(), r.ExpectedRed[0].ID) {
			t.Fatalf("refusal does not name the stale case id: %v", err)
		}
	})

	t.Run("any failing check with no register case fails outright", func(t *testing.T) {
		r := journeyPassingReportFixture()
		r.ExpectedRed = append(r.ExpectedRed, journeyExpectedRedResult{
			Result: "unregistered-gap",
			Detail: "the status card advises `aether some-new-thing`, which has no menu wrapper",
		})
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected a refusal for an unregistered gap")
		}
		if !strings.Contains(err.Error(), "some-new-thing") {
			t.Fatalf("refusal does not name the unregistered gap: %v", err)
		}
	})
}

// TestJourneyGateVerdictAcceptsAGenuineRun proves journeyGateVerdict accepts
// a genuinely passing whole-chain, live, three-trial report, and that a
// report with one flaky-failure trial (and two passing trials) is also
// accepted -- with the accepted result still recording the flaky trial
// rather than hiding it.
func TestJourneyGateVerdictAcceptsAGenuineRun(t *testing.T) {
	r := journeyPassingReportFixture()
	if err := journeyGateVerdict(r); err != nil {
		t.Fatalf("journeyGateVerdict refused a genuinely passing report: %v", err)
	}

	withOneFlaky := journeyPassingReportFixture()
	withOneFlaky.Trials[0].Outcome = string(journeyTrialFlakyFailure)
	withOneFlaky.Trials[0].Steps[0].Status = "fail"
	withOneFlaky.Trials[0].Steps[0].FailureKind = string(journeyFailureTransient)
	if err := journeyGateVerdict(withOneFlaky); err != nil {
		t.Fatalf("journeyGateVerdict refused a report with one flaky trial and two passing trials: %v", err)
	}
	if withOneFlaky.Trials[0].Outcome != string(journeyTrialFlakyFailure) {
		t.Fatal("the accepted result must still record the flaky trial, not hide it")
	}
}

// TestJourneyTrialOutcomeVocabularyIsClosed proves every outcome a trial
// can carry is in the declared vocabulary, and an undeclared value is
// refused by name.
func TestJourneyTrialOutcomeVocabularyIsClosed(t *testing.T) {
	names := journeyTrialOutcomeNames()
	if len(names) != len(journeyTrialOutcomeVocabulary) {
		t.Fatalf("journeyTrialOutcomeNames() returned %d names, want %d", len(names), len(journeyTrialOutcomeVocabulary))
	}
	for _, want := range []string{"passed", "flaky-failure", "real-failure", "incomplete"} {
		if !journeyContainsString(names, want) {
			t.Errorf("journeyTrialOutcomeNames() = %v, missing %q", names, want)
		}
	}

	if journeyTrialOutcomeDeclared(journeyTrialOutcome("bogus-outcome")) {
		t.Fatal("an undeclared trial outcome must not be reported as declared")
	}

	r := journeyPassingReportFixture()
	r.Trials[0].Outcome = "bogus-outcome"
	err := journeyGateVerdict(r)
	if err == nil {
		t.Fatal("expected journeyGateVerdict to refuse an undeclared trial outcome")
	}
	if !strings.Contains(err.Error(), "bogus-outcome") {
		t.Fatalf("refusal does not name the undeclared outcome: %v", err)
	}
}

// TestJourneyTrialsDoNotShareState proves two reports written concurrently
// to two different paths each contain their own trials and neither is
// truncated -- the multi-trial, concurrent-run analogue of the
// discovered==executed truncation discipline CLAUDE.md's Verification
// Commands section documents.
func TestJourneyTrialsDoNotShareState(t *testing.T) {
	pathA := filepath.Join(t.TempDir(), "run-a", "journey-report.json")
	pathB := filepath.Join(t.TempDir(), "run-b", "journey-report.json")

	reportA := journeyPassingReportFixture()
	reportA.Trials[0].SessionID = "session-a-0"
	reportB := journeyPassingReportFixture()
	reportB.Trials[0].SessionID = "session-b-0"

	var wg sync.WaitGroup
	var errA, errB error
	wg.Add(2)
	go func() { defer wg.Done(); errA = writeJourneyReport(pathA, reportA) }()
	go func() { defer wg.Done(); errB = writeJourneyReport(pathB, reportB) }()
	wg.Wait()
	if errA != nil {
		t.Fatalf("writeJourneyReport(%s): %v", pathA, errA)
	}
	if errB != nil {
		t.Fatalf("writeJourneyReport(%s): %v", pathB, errB)
	}

	readA, err := readJourneyReport(pathA)
	if err != nil {
		t.Fatalf("readJourneyReport(%s): %v", pathA, err)
	}
	readB, err := readJourneyReport(pathB)
	if err != nil {
		t.Fatalf("readJourneyReport(%s): %v", pathB, err)
	}

	if len(readA.Trials) != len(reportA.Trials) {
		t.Fatalf("report A carries %d trial(s) after the concurrent write, want %d -- truncated", len(readA.Trials), len(reportA.Trials))
	}
	if len(readB.Trials) != len(reportB.Trials) {
		t.Fatalf("report B carries %d trial(s) after the concurrent write, want %d -- truncated", len(readB.Trials), len(reportB.Trials))
	}
	if readA.Trials[0].SessionID != "session-a-0" {
		t.Fatalf("report A's first trial session id is %q, want %q -- it picked up report B's state", readA.Trials[0].SessionID, "session-a-0")
	}
	if readB.Trials[0].SessionID != "session-b-0" {
		t.Fatalf("report B's first trial session id is %q, want %q -- it picked up report A's state", readB.Trials[0].SessionID, "session-b-0")
	}
}

// TestJourneyIncompleteTrialIsNeverPassed proves a trial marked incomplete,
// with a step named as where it stopped, is never treated as passed --
// neither by journeyDeriveTrialOutcome (the harness's own classifier) nor
// by journeyGateVerdict (which refuses it via the declared/executed
// mismatch an incomplete trial always carries).
func TestJourneyIncompleteTrialIsNeverPassed(t *testing.T) {
	steps := []journeyStepResult{
		{Name: "start", Status: "pass"},
		{Name: "survey", Status: "fail", FailureKind: string(journeyFailureReal)},
		{Name: "discuss", Status: "not-reached"},
	}
	outcome, incompleteAt := journeyDeriveTrialOutcome(steps, 3, 2)
	if outcome == journeyTrialPassed {
		t.Fatal("an incomplete trial must never classify as passed")
	}
	if outcome != journeyTrialIncomplete {
		t.Fatalf("journeyDeriveTrialOutcome(...) outcome = %q, want %q", outcome, journeyTrialIncomplete)
	}
	if incompleteAt != "survey" {
		t.Fatalf("journeyDeriveTrialOutcome(...) incompleteAtStep = %q, want %q (the step it stopped at)", incompleteAt, "survey")
	}

	r := journeyPassingReportFixture()
	r.Trials[0].Outcome = string(journeyTrialIncomplete)
	r.Trials[0].StepsExecuted = r.Trials[0].StepsDeclared - 1
	r.Trials[0].IncompleteAtStep = "survey"
	if err := journeyGateVerdict(r); err == nil {
		t.Fatal("journeyGateVerdict must refuse a report carrying an incomplete trial")
	}
}

// TestJourneyStepVocabularyIsClosed proves journeyStepNames is a closed
// vocabulary: every declared step has a menu command, and every menu
// command maps back to a declared step.
func TestJourneyStepVocabularyIsClosed(t *testing.T) {
	names := journeyStepNames()
	if len(names) != len(journeyStepVocabulary) {
		t.Fatalf("journeyStepNames() returned %d names, want %d", len(names), len(journeyStepVocabulary))
	}

	for _, step := range journeyStepVocabulary {
		cmd, ok := journeyStepMenuCommand(step)
		if !ok {
			t.Errorf("declared step %q has no menu command", step)
		}
		if cmd == "" {
			t.Errorf("declared step %q maps to an empty menu command", step)
		}
	}

	for step := range journeyStepMenuCommandMap {
		if !journeyStepDeclared(step) {
			t.Errorf("journeyStepMenuCommandMap declares step %q, which is not in the declared vocabulary", step)
		}
	}
}
