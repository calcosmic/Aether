package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Phase 196 plan 03 — decision D-04 as CORRECTED (2026-08-27).
//
// Claude Code writes one transcript line per content block of a reply — the
// thinking, the text, each tool call — and EVERY one of those lines repeats the
// same cumulative usage object for the whole reply. Add the lines up and you
// bill one reply two or three times.
//
// The first recorded reading of this defect was wrong about its mechanism: it
// said the repetition happened across the `user`, `queue-operation` and
// `attachment` line types and should be deduplicated by `tool_use_id`. Measured
// across the whole corpus on this machine, `queue-operation` and `attachment`
// carry usage ZERO times and `tool_use_id` is absent from usage-bearing lines —
// a parser built to that reading would have deduplicated nothing and reproduced
// the entire overcount, while its fixture-integrity test could never have been
// satisfied from a real capture. The earlier reading had matched a NESTED type
// field on a content block and mistaken it for the line type: the exact
// "structure, not substrings" mistake the decision exists to prevent.
//
// Re-measured here on 2026-08-28 across 1,726 transcripts (1.0 GB) under
// ~/.claude/projects, counting every line carrying a usage object by its
// TOP-LEVEL type:
//
//	assistant   142581   usage at .message.usage
//	user           252   usage at .toolUseResult.usage
//
// and nothing else, anywhere.

// claudeTranscriptFixturePath is a REAL captured transcript slice that keeps its
// duplicate usage blocks on purpose. See its README for provenance, the
// redaction rules, and both measured totals.
const claudeTranscriptFixturePath = "testdata/spend/claude/transcript-with-duplicates.jsonl"

// The literals below were measured on this capture with jq and awk and checked
// by hand addition. NOTHING here is produced by calling the reader, the billed-
// total helper, or any other code under test — that is the discipline this
// repository's 186x undercount was shipped for want of.
//
// Neither total is one of the two figures (8,930,280 / 4,237,379) recorded
// before D-04 was corrected. Those came from the mistaken reading above and are
// not carried forward.
const (
	// claudeFixtureNaiveOccurrenceSum adds up every usage-bearing occurrence,
	// which is what a reader that does not deduplicate produces.
	claudeFixtureNaiveOccurrenceSum int64 = 2764260

	// claudeFixtureDeduplicatedTotal counts each message.id once (last
	// occurrence wins) and counts the one identifier-less usage-bearing line
	// once on its own. This is the truth.
	claudeFixtureDeduplicatedTotal int64 = 1572834

	// Shape of the fixture, also measured by hand.
	claudeFixtureUsageBearingLines  = 18
	claudeFixtureDistinctMessageIDs = 9
	claudeFixtureIdentifierlessRows = 1
)

// readClaudeFixtureLines decodes the fixture with deliberately DIFFERENT
// machinery from the reader under test — untyped maps rather than the reader's
// structs — so this test cannot pass merely because the reader and the test
// agree with each other about how to read a transcript.
func readClaudeFixtureLines(t *testing.T, path string) []map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var lines []map[string]interface{}
	for _, text := range strings.Split(string(raw), "\n") {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		var line map[string]interface{}
		if err := json.Unmarshal([]byte(text), &line); err != nil {
			t.Fatalf("fixture line is not valid JSON: %v", err)
		}
		lines = append(lines, line)
	}
	return lines
}

// fixtureUsageOccurrence is one usage-bearing line as this test sees it.
type fixtureUsageOccurrence struct {
	LineType    string
	MessageID   string
	BilledTotal int64
}

// fixtureUsageOccurrences walks the fixture the way D-04 as corrected says a
// transcript must be walked: the line type comes from the TOP-LEVEL `type`
// field, and each of the two usage-bearing types carries its usage object at
// its own documented position.
func fixtureUsageOccurrences(t *testing.T, lines []map[string]interface{}) []fixtureUsageOccurrence {
	t.Helper()
	var found []fixtureUsageOccurrence
	for _, line := range lines {
		lineType, _ := line["type"].(string)
		switch lineType {
		case "assistant":
			message, ok := line["message"].(map[string]interface{})
			if !ok {
				continue
			}
			usage, ok := message["usage"].(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := message["id"].(string)
			found = append(found, fixtureUsageOccurrence{
				LineType:    lineType,
				MessageID:   id,
				BilledTotal: fixtureBilledTotal(t, usage),
			})
		case "user":
			result, ok := line["toolUseResult"].(map[string]interface{})
			if !ok {
				continue
			}
			usage, ok := result["usage"].(map[string]interface{})
			if !ok {
				continue
			}
			found = append(found, fixtureUsageOccurrence{
				LineType:    lineType,
				MessageID:   "",
				BilledTotal: fixtureBilledTotal(t, usage),
			})
		}
	}
	return found
}

// fixtureBilledTotal adds the four disjoint columns a provider actually bills
// for. Written out here rather than borrowed from the production helper so this
// test is not asserting the code's arithmetic against itself.
func fixtureBilledTotal(t *testing.T, usage map[string]interface{}) int64 {
	t.Helper()
	column := func(key string) int64 {
		value, ok := usage[key].(float64)
		if !ok {
			return 0
		}
		return int64(value)
	}
	return column("input_tokens") +
		column("cache_creation_input_tokens") +
		column("cache_read_input_tokens") +
		column("output_tokens")
}

// TestClaudeTranscriptFixtureContainsRepeatedMessageIDs is the test that
// protects the other tests.
//
// The deleted branch worktree-agent-a59fd3ee68644ea21 shipped a hand-written
// fixture listing every usage block exactly once. Against that fixture no
// assertion could ever have detected a reader that counts the same reply three
// times — a fixture that cannot fail, which is precisely the shape of the 186x
// undercount this repository already shipped in this same subsystem, one phase
// earlier.
//
// If a future edit tidies this fixture so each message.id appears once, or
// drops one of the two usage-bearing line types, this test fails and says so.
func TestClaudeTranscriptFixtureContainsRepeatedMessageIDs(t *testing.T) {
	lines := readClaudeFixtureLines(t, claudeTranscriptFixturePath)
	occurrences := fixtureUsageOccurrences(t, lines)

	if len(occurrences) != claudeFixtureUsageBearingLines {
		t.Fatalf("fixture has %d usage-bearing line(s), want %d — the fixture has been changed; "+
			"re-measure it and update its README and every literal in this file",
			len(occurrences), claudeFixtureUsageBearingLines)
	}

	t.Run("at least one message id repeats among usage-bearing lines", func(t *testing.T) {
		perID := map[string]int{}
		for _, occurrence := range occurrences {
			if occurrence.MessageID == "" {
				continue
			}
			perID[occurrence.MessageID]++
		}
		if len(perID) != claudeFixtureDistinctMessageIDs {
			t.Fatalf("fixture carries %d distinct message id(s), want %d", len(perID), claudeFixtureDistinctMessageIDs)
		}
		var repeated []string
		for id, count := range perID {
			if count > 1 {
				repeated = append(repeated, id)
			}
		}
		if len(repeated) == 0 {
			t.Fatalf("NO message id occurs more than once among the fixture's %d usage-bearing line(s). "+
				"The fixture has been tidied into one block per message, and it can no longer fail a reader "+
				"that counts the same reply two or three times. Restore the repetition — it IS the fixture.",
				len(occurrences))
		}
		var singletons int
		for _, count := range perID {
			if count == 1 {
				singletons++
			}
		}
		if singletons == 0 {
			t.Errorf("every message id in the fixture repeats — a reader that deduplicated by simply keeping "+
				"the first line of each pair would still pass; the fixture needs at least one id that occurs "+
				"exactly once (%d distinct ids, all repeated)", len(perID))
		}
	})

	t.Run("both surviving usage-bearing line types are present", func(t *testing.T) {
		perType := map[string]int{}
		for _, occurrence := range occurrences {
			perType[occurrence.LineType]++
		}
		for _, want := range []string{"assistant", "user"} {
			if perType[want] == 0 {
				t.Errorf("no usage-bearing line of top-level type %q in the fixture — D-04 as corrected names "+
					"exactly two usage-bearing types and both must be represented (present: %v)", want, perType)
			}
		}
		var identifierless int
		for _, occurrence := range occurrences {
			if occurrence.MessageID == "" {
				identifierless++
			}
		}
		if identifierless != claudeFixtureIdentifierlessRows {
			t.Errorf("fixture carries %d usage-bearing line(s) with no message id, want %d — the subagent "+
				"completion row is the identifier-less case the reader must count once on its own",
				identifierless, claudeFixtureIdentifierlessRows)
		}
	})

	t.Run("the naive and deduplicated totals still measure what the README says", func(t *testing.T) {
		var naive int64
		lastPerID := map[string]int64{}
		var identifierlessSum int64
		for _, occurrence := range occurrences {
			naive += occurrence.BilledTotal
			if occurrence.MessageID == "" {
				identifierlessSum += occurrence.BilledTotal
				continue
			}
			lastPerID[occurrence.MessageID] = occurrence.BilledTotal
		}
		deduplicated := identifierlessSum
		for _, total := range lastPerID {
			deduplicated += total
		}

		if naive != claudeFixtureNaiveOccurrenceSum {
			t.Errorf("naive occurrence sum = %d, want %d", naive, claudeFixtureNaiveOccurrenceSum)
		}
		if deduplicated != claudeFixtureDeduplicatedTotal {
			t.Errorf("deduplicated total = %d, want %d", deduplicated, claudeFixtureDeduplicatedTotal)
		}
		if naive <= deduplicated {
			t.Fatalf("the fixture's naive sum (%d) is not strictly larger than its deduplicated total (%d) — "+
				"there is no duplication left to catch", naive, deduplicated)
		}
	})

	t.Run("non usage-bearing line types are present and carry no usage", func(t *testing.T) {
		var others int
		for _, line := range lines {
			lineType, _ := line["type"].(string)
			if lineType == "assistant" || lineType == "user" {
				continue
			}
			others++
			if _, ok := line["usage"]; ok {
				t.Errorf("line of type %q carries a top-level usage object — re-measure the corpus, "+
					"D-04's line-type table may be out of date", lineType)
			}
		}
		if others == 0 {
			t.Errorf("the fixture contains only usage-bearing line types; it can no longer show that " +
				"other line types are correctly skipped")
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2 — the reader itself.
//
// Every expected number below is a hand-derived literal. None is produced by
// calling parseClaudeTranscriptUsage, BilledTotalTokens, or any other code
// under test. This repository shipped a 186x undercount in this same subsystem
// because a unit test asserted the same wrong arithmetic as the code it was
// testing; that is the discipline these literals exist to keep.
// ---------------------------------------------------------------------------

// The fixture's two rows, column by column, measured with jq and checked by
// hand addition (17 + 13886 + 1321174 + 5955 = 1341032, and
// 2 + 330 + 230803 + 667 = 231802; 1341032 + 231802 = 1572834).
const (
	claudeFixtureSessionInput         int64 = 17
	claudeFixtureSessionCacheCreation int64 = 13886
	claudeFixtureSessionCacheRead     int64 = 1321174
	claudeFixtureSessionOutput        int64 = 5955
	claudeFixtureSessionTotal         int64 = 1341032

	claudeFixtureSubagentInput         int64 = 2
	claudeFixtureSubagentCacheCreation int64 = 330
	claudeFixtureSubagentCacheRead     int64 = 230803
	claudeFixtureSubagentOutput        int64 = 667
	claudeFixtureSubagentTotal         int64 = 231802

	claudeFixtureSubagentWorkerName = "gsd-executor"
	claudeFixtureSubagentAgentID    = "af5e8963db8c28bf7"

	// The first three fixture lines are three occurrences of ONE message id,
	// so a read bounded to them yields exactly this single row.
	claudeFixtureFirstThreeLinesTotal int64 = 143153
)

// writeClaudeTranscriptUnderTempHome puts a transcript inside a temporary
// $HOME/.claude/projects tree and points HOME at it, so the containment rule is
// genuinely exercised and the owner's real ~/.claude/projects directory is
// never read by any test in this file.
func writeClaudeTranscriptUnderTempHome(t *testing.T, content []byte) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".claude", "projects", "-fixture-project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

func claudeFixtureBytes(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(claudeTranscriptFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return raw
}

func claudeRowsByWorker(t *testing.T, rows []claudeTranscriptUsage) map[string]claudeTranscriptUsage {
	t.Helper()
	byWorker := make(map[string]claudeTranscriptUsage, len(rows))
	for _, row := range rows {
		if _, clash := byWorker[row.WorkerName]; clash {
			t.Fatalf("two rows share the worker name %q — a worker-keyed result cannot merge two workers", row.WorkerName)
		}
		byWorker[row.WorkerName] = row
	}
	return byWorker
}

// claudeAssistantLine builds one synthetic assistant transcript line. Used only
// for the adversarial cases (decoys, oversized lines, legacy shapes) where a
// real capture cannot supply the shape being guarded against.
func claudeAssistantLine(messageID string, input, cacheCreation, cacheRead, output int64, extraContent string) string {
	content := extraContent
	if content == "" {
		content = `{"type":"text","text":"[redacted]"}`
	}
	return `{"type":"assistant","uuid":"uuid-` + messageID + `","requestId":"req-` + messageID +
		`","message":{"id":"` + messageID + `","type":"message","role":"assistant","model":"claude-opus-5","content":[` +
		content + `],"usage":{"input_tokens":` + strconv.FormatInt(input, 10) +
		`,"cache_creation_input_tokens":` + strconv.FormatInt(cacheCreation, 10) +
		`,"cache_read_input_tokens":` + strconv.FormatInt(cacheRead, 10) +
		`,"output_tokens":` + strconv.FormatInt(output, 10) + `}}}`
}

func claudeTotalOf(rows []claudeTranscriptUsage) int64 {
	var total int64
	for _, row := range rows {
		total += row.Usage.BilledTotalTokens()
	}
	return total
}

// assertNoColumnEquals fails if any row carries the given value in any billed
// column. Used to prove a decoy's numbers never entered the accounting.
func assertNoColumnEquals(t *testing.T, rows []claudeTranscriptUsage, forbidden int64, why string) {
	t.Helper()
	for _, row := range rows {
		columns := map[string]int64{
			"InputTokens":         row.Usage.InputTokens,
			"CachedInputTokens":   row.Usage.CachedInputTokens,
			"CacheCreationTokens": row.Usage.CacheCreationTokens,
			"OutputTokens":        row.Usage.OutputTokens,
			"TotalTokens":         row.Usage.TotalTokens,
		}
		for name, value := range columns {
			if value == forbidden {
				t.Errorf("worker %q has %s = %d, which is the decoy value — %s", row.WorkerName, name, forbidden, why)
			}
		}
	}
}

// TestClaudeTranscriptDeduplicatesUsageByMessageID is the headline test: the
// same reply, republished across three transcript lines as its thinking, its
// text and its tool call are written out, must be billed once.
func TestClaudeTranscriptDeduplicatesUsageByMessageID(t *testing.T) {
	path := writeClaudeTranscriptUnderTempHome(t, claudeFixtureBytes(t))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}

	total := claudeTotalOf(rows)
	if total == claudeFixtureNaiveOccurrenceSum {
		t.Fatalf("the reader billed %d, which is the sum of EVERY usage-bearing occurrence in the fixture. "+
			"It is counting the same reply two and three times. The truth is %d.",
			total, claudeFixtureDeduplicatedTotal)
	}
	if total != claudeFixtureDeduplicatedTotal {
		t.Fatalf("total = %d, want %d (naive occurrence sum would be %d)",
			total, claudeFixtureDeduplicatedTotal, claudeFixtureNaiveOccurrenceSum)
	}

	// The fixture is only able to catch a naive reader while its naive sum is
	// strictly larger than its deduplicated total. Both are literals measured
	// off the capture, so this compares two hand-derived numbers, not the code
	// with itself.
	if claudeFixtureNaiveOccurrenceSum <= claudeFixtureDeduplicatedTotal {
		t.Fatalf("the fixture's naive sum (%d) is not strictly larger than its deduplicated total (%d) — "+
			"this test could not fail a reader that stopped deduplicating",
			claudeFixtureNaiveOccurrenceSum, claudeFixtureDeduplicatedTotal)
	}

	byWorker := claudeRowsByWorker(t, rows)
	session, ok := byWorker[claudeTranscriptMainSessionWorker]
	if !ok {
		t.Fatalf("no row for %q; got %v", claudeTranscriptMainSessionWorker, byWorker)
	}
	for _, check := range []struct {
		name string
		got  int64
		want int64
	}{
		{"InputTokens", session.Usage.InputTokens, claudeFixtureSessionInput},
		{"CacheCreationTokens", session.Usage.CacheCreationTokens, claudeFixtureSessionCacheCreation},
		{"CachedInputTokens", session.Usage.CachedInputTokens, claudeFixtureSessionCacheRead},
		{"OutputTokens", session.Usage.OutputTokens, claudeFixtureSessionOutput},
	} {
		if check.got != check.want {
			t.Errorf("session %s = %d, want %d", check.name, check.got, check.want)
		}
	}
	if got := session.Usage.BilledTotalTokens(); got != claudeFixtureSessionTotal {
		t.Errorf("session billed total = %d, want %d", got, claudeFixtureSessionTotal)
	}
}

// TestClaudeTranscriptReadsBothUsageBearingLineTypes covers the second, much
// rarer shape: a dispatched subagent's completion record, which lives on a
// top-level `user` line, carries its usage at a different position, and has no
// message id at all. It must be counted once on its own, never folded into the
// session's row and never dropped for lack of an identifier.
func TestClaudeTranscriptReadsBothUsageBearingLineTypes(t *testing.T) {
	path := writeClaudeTranscriptUnderTempHome(t, claudeFixtureBytes(t))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d worker row(s), want 2 (the session's own turns and one subagent): %+v", len(rows), rows)
	}

	byWorker := claudeRowsByWorker(t, rows)
	subagent, ok := byWorker[claudeFixtureSubagentWorkerName]
	if !ok {
		t.Fatalf("no row for the subagent %q — the identifier-less usage-bearing line was dropped or merged; got %v",
			claudeFixtureSubagentWorkerName, byWorker)
	}
	if subagent.AgentID != claudeFixtureSubagentAgentID {
		t.Errorf("subagent AgentID = %q, want %q", subagent.AgentID, claudeFixtureSubagentAgentID)
	}
	for _, check := range []struct {
		name string
		got  int64
		want int64
	}{
		{"InputTokens", subagent.Usage.InputTokens, claudeFixtureSubagentInput},
		{"CacheCreationTokens", subagent.Usage.CacheCreationTokens, claudeFixtureSubagentCacheCreation},
		{"CachedInputTokens", subagent.Usage.CachedInputTokens, claudeFixtureSubagentCacheRead},
		{"OutputTokens", subagent.Usage.OutputTokens, claudeFixtureSubagentOutput},
	} {
		if check.got != check.want {
			t.Errorf("subagent %s = %d, want %d", check.name, check.got, check.want)
		}
	}
	if got := subagent.Usage.BilledTotalTokens(); got != claudeFixtureSubagentTotal {
		t.Errorf("subagent billed total = %d, want %d", got, claudeFixtureSubagentTotal)
	}

	session, ok := byWorker[claudeTranscriptMainSessionWorker]
	if !ok {
		t.Fatalf("no row for %q", claudeTranscriptMainSessionWorker)
	}
	if got := session.Usage.BilledTotalTokens(); got != claudeFixtureSessionTotal {
		t.Errorf("session billed total = %d, want %d — the subagent's row may have been folded into it",
			got, claudeFixtureSessionTotal)
	}
	if session.AgentID != "" {
		t.Errorf("session row carries AgentID %q, want empty", session.AgentID)
	}
}

// TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType is the decoy that
// caught the superseded reading of this defect.
//
// The decoy line is USAGE-BEARING; its TOP-LEVEL type is OUTSIDE the accepted
// set while its NESTED type is INSIDE it. Built the other way round the
// assertion would be vacuous: a naive substring reader would skip the line for
// its own reasons and prove nothing.
func TestClaudeTranscriptReadsTopLevelLineTypeNotNestedType(t *testing.T) {
	const decoyValue int64 = 900000
	decoy := `{"type":"queue-operation","operation":"enqueue","uuid":"uuid-decoy",` +
		`"message":{"id":"msg_decoy","type":"assistant","role":"assistant",` +
		`"usage":{"input_tokens":900000,"cache_creation_input_tokens":900000,` +
		`"cache_read_input_tokens":900000,"output_tokens":900000}},` +
		`"content":[{"type":"user","text":"[redacted]"}]}`
	real := claudeAssistantLine("msg_real", 11, 22, 33, 44, "")

	path := writeClaudeTranscriptUnderTempHome(t, []byte(decoy+"\n"+real+"\n"))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	// 11 + 22 + 33 + 44 = 110, written out by hand.
	if got := claudeTotalOf(rows); got != 110 {
		t.Errorf("total = %d, want 110 — the queue-operation line's nested \"assistant\" type was treated "+
			"as the line type, which is exactly the mistake that produced the superseded reading of D-04", got)
	}
	assertNoColumnEquals(t, rows, decoyValue,
		"a line whose TOP-LEVEL type is queue-operation carries no billable usage, whatever its content blocks say")
}

// TestClaudeTranscriptIgnoresUsageInsideToolDescription proves the match is
// structural. A usage object quoted inside a tool call's own description is
// text, not accounting.
func TestClaudeTranscriptIgnoresUsageInsideToolDescription(t *testing.T) {
	const quotedValue int64 = 777777
	quoted := `{"type":"tool_use","id":"toolu_1","name":"Task","input":{"description":` +
		`"Report the spend. Example line: \"usage\": {\"input_tokens\": 777777, \"cache_creation_input_tokens\": 777777, ` +
		`\"cache_read_input_tokens\": 777777, \"output_tokens\": 777777}"}}`
	line := claudeAssistantLine("msg_real", 11, 22, 33, 44, quoted)

	path := writeClaudeTranscriptUnderTempHome(t, []byte(line+"\n"))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	if got := claudeTotalOf(rows); got != 110 {
		t.Errorf("total = %d, want 110 — a usage block quoted inside a tool description was counted", got)
	}
	assertNoColumnEquals(t, rows, quotedValue, "it was quoted inside a tool call's description, not reported as usage")
}

// TestClaudeTranscriptRefusesPathOutsideProjectsRoot exercises the same
// containment rule the session record already enforces, through a temporary
// home directory, so the real projects directory is never opened.
func TestClaudeTranscriptRefusesPathOutsideProjectsRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	elsewhere := filepath.Join(t.TempDir(), "elsewhere.jsonl")
	if err := os.WriteFile(elsewhere, claudeFixtureBytes(t), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"a path in a wholly unrelated directory", elsewhere},
		{"a crafted sibling of the projects root", filepath.Join(home, ".claude", "projects-evil", "session.jsonl")},
		{"a traversal out of the projects root", filepath.Join(home, ".claude", "projects", "..", "escaped.jsonl")},
		{"a relative path", "testdata/spend/claude/transcript-with-duplicates.jsonl"},
		{"an empty path", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := parseClaudeTranscriptUsage(tc.path)
			if err == nil {
				t.Fatalf("parseClaudeTranscriptUsage(%q) returned no error; a transcript outside "+
					"$HOME/.claude/projects must be refused", tc.path)
			}
			if rows != nil {
				t.Errorf("refused path still returned %d row(s)", len(rows))
			}
		})
	}
}

// TestClaudeTranscriptSkipsOversizedLineAndContinues — one pathological line
// must not cost the rest of the file.
func TestClaudeTranscriptSkipsOversizedLineAndContinues(t *testing.T) {
	const oversizedValue int64 = 900000
	first := claudeAssistantLine("msg_first", 11, 22, 33, 44, "")
	second := claudeAssistantLine("msg_second", 22, 44, 66, 88, "")
	huge := claudeAssistantLine("msg_huge", 900000, 900000, 900000, 900000,
		`{"type":"text","text":"`+strings.Repeat("x", claudeTranscriptMaxLineBytes+(1<<20))+`"}`)

	path := writeClaudeTranscriptUnderTempHome(t, []byte(first+"\n"+huge+"\n"+second+"\n"))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	// (11+22+33+44) + (22+44+66+88) = 110 + 220 = 330, written out by hand.
	if got := claudeTotalOf(rows); got != 330 {
		t.Errorf("total = %d, want 330 — the oversized line must be skipped and BOTH surrounding lines still read", got)
	}
	assertNoColumnEquals(t, rows, oversizedValue, "the line carrying it exceeded the per-line byte bound")
}

// TestClaudeTranscriptAbsentFileIsEmptyAndNoError — a session that never wrote
// a transcript is not an error condition.
func TestClaudeTranscriptAbsentFileIsEmptyAndNoError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".claude", "projects", "-fixture-project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	rows, err := parseClaudeTranscriptUsage(filepath.Join(dir, "never-written.jsonl"))
	if err != nil {
		t.Fatalf("absent transcript returned an error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("absent transcript returned %d row(s), want 0", len(rows))
	}
}

// TestClaudeTranscriptByteBoundStopsEarly — the bounded entry point stops at
// the bound instead of reading the whole file.
func TestClaudeTranscriptByteBoundStopsEarly(t *testing.T) {
	fixture := claudeFixtureBytes(t)
	path := writeClaudeTranscriptUnderTempHome(t, fixture)

	// The bound is an INPUT, computed from the fixture bytes by this test.
	// The expected total below is a literal.
	lines := strings.SplitAfter(string(fixture), "\n")
	if len(lines) < 3 {
		t.Fatalf("fixture has %d line(s); this test needs at least 3", len(lines))
	}
	var bound int64
	for _, line := range lines[:3] {
		bound += int64(len(line))
	}

	rows, err := parseClaudeTranscriptUsageWithBounds(path, bound)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsageWithBounds: %v", err)
	}
	if got := claudeTotalOf(rows); got != claudeFixtureFirstThreeLinesTotal {
		t.Errorf("bounded total = %d, want %d — the first three fixture lines are three occurrences of one "+
			"message id, so a bounded read yields exactly that one reply", got, claudeFixtureFirstThreeLinesTotal)
	}
	if len(rows) != 1 {
		t.Errorf("bounded read returned %d row(s), want 1", len(rows))
	}

	full, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	if claudeTotalOf(full) != claudeFixtureDeduplicatedTotal {
		t.Fatalf("unbounded total = %d, want %d", claudeTotalOf(full), claudeFixtureDeduplicatedTotal)
	}
	if claudeFixtureFirstThreeLinesTotal >= claudeFixtureDeduplicatedTotal {
		t.Fatalf("the bounded literal (%d) is not smaller than the whole-file literal (%d) — this test "+
			"could not detect a bound that was ignored", claudeFixtureFirstThreeLinesTotal, claudeFixtureDeduplicatedTotal)
	}
}

// TestClaudeTranscriptRowsAreNeverProviderGrade — a transcript is a real
// measurement the Go runtime read itself, but its accounting semantics are
// undocumented by the vendor, so it may never borrow the provider tag.
func TestClaudeTranscriptRowsAreNeverProviderGrade(t *testing.T) {
	path := writeClaudeTranscriptUnderTempHome(t, claudeFixtureBytes(t))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("parseClaudeTranscriptUsage: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("no rows returned; this test would pass vacuously")
	}
	for _, row := range rows {
		if row.Usage.Source != codex.UsageSourceSessionTranscript {
			t.Errorf("worker %q has source %q, want %q", row.WorkerName, row.Usage.Source, codex.UsageSourceSessionTranscript)
		}
		if row.Usage.Measured() {
			t.Errorf("worker %q reports provider-grade; only ParseUsage may ever set that", row.WorkerName)
		}
		if row.Usage.Estimated() {
			t.Errorf("worker %q reports an estimate; a transcript row is a real measurement, and the "+
				"estimate tag would file it under the ledger's guessed subtotal", row.WorkerName)
		}
		if row.Usage.Empty() {
			t.Errorf("worker %q returned an empty usage row", row.WorkerName)
		}
	}
}

// TestClaudeTranscriptLegacyColonFormIsTolerated — older transcripts carry a
// text usage block of the form "<usage>subagent_tokens: N</usage>" inside a
// tool result. It appears in older sessions and not once in a current one.
// Tolerated means: it does not break the read, and it is NOT counted — reading
// a number out of a text blob is the substring matching this whole plan exists
// to forbid.
func TestClaudeTranscriptLegacyColonFormIsTolerated(t *testing.T) {
	const legacyValue int64 = 110790
	legacy := `{"type":"user","uuid":"uuid-legacy","message":{"role":"user","content":[{"type":"tool_result",` +
		`"tool_use_id":"toolu_legacy","content":"<usage>subagent_tokens: 110790\ntool_uses: 19</usage>"}]},` +
		`"toolUseResult":"<usage>subagent_tokens: 110790\ntool_uses: 19</usage>"}`
	real := claudeAssistantLine("msg_real", 11, 22, 33, 44, "")

	path := writeClaudeTranscriptUnderTempHome(t, []byte(legacy+"\n"+real+"\n"))

	rows, err := parseClaudeTranscriptUsage(path)
	if err != nil {
		t.Fatalf("the legacy colon-form usage block must be tolerated, not fatal: %v", err)
	}
	if got := claudeTotalOf(rows); got != 110 {
		t.Errorf("total = %d, want 110 — the legacy text usage block was counted; it is text, not accounting", got)
	}
	assertNoColumnEquals(t, rows, legacyValue, "it was scraped out of a text blob rather than read from a usage object")
}
