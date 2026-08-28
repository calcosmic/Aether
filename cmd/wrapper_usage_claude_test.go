package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
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
