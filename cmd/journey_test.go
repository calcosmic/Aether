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
	"strings"
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
// executed step counts differ (naming both numbers), and accepts a
// whole-chain live report whose counts match.
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
		r := journeyReport{Scope: "whole-chain", Mode: "live", StepsDeclared: 14, StepsExecuted: 9}
		err := journeyGateVerdict(r)
		if err == nil {
			t.Fatal("expected journeyGateVerdict to refuse a declared/executed mismatch")
		}
		if !strings.Contains(err.Error(), "14") || !strings.Contains(err.Error(), "9") {
			t.Fatalf("journeyGateVerdict error does not name both step counts: %v", err)
		}
	})

	t.Run("a matching whole-chain live report is accepted", func(t *testing.T) {
		r := journeyReport{Scope: "whole-chain", Mode: "live", StepsDeclared: 14, StepsExecuted: 14}
		if err := journeyGateVerdict(r); err != nil {
			t.Fatalf("journeyGateVerdict refused a report that should pass: %v", err)
		}
	})
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
