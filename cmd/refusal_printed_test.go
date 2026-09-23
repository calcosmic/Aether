package cmd

// 208-07-PLAN.md (UED-10): proves journeyPrintedRefusals genuinely finds a
// refusal a real renderRefusal call printed, ignores ordinary prose that
// merely happens to carry a backticked command, and proves
// printedCommandToRuntimeCommand reverses the forward menu-command mapping
// without ever guessing.
//
// Fixtures are built by varying the real captured session
// cmd/testdata/stop-hook/menu-command-transcript.jsonl's own assistant
// tool_use / user tool_result pair -- only the tool_result's own "content"
// text changes, to a real renderRefusal(...) call's output -- never a
// hand-typed transcript shape the runtime could not produce
// (CLAUDE.md's Definition of Done).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cloneJSONObject deep-clones src through a marshal/unmarshal round trip so
// each derived transcript line can be mutated independently without
// aliasing the original decoded map.
func cloneJSONObject(t *testing.T, src map[string]interface{}) map[string]interface{} {
	t.Helper()
	encoded, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal for clone: %v", err)
	}
	var dst map[string]interface{}
	if err := json.Unmarshal(encoded, &dst); err != nil {
		t.Fatalf("unmarshal for clone: %v", err)
	}
	return dst
}

// journeyWriteRefusalTranscript derives a transcript fixture from the real
// captured session at journeyRealTranscriptPath: one assistant tool_use +
// user tool_result pair per entry in texts, each pair linked by its own
// fresh tool_use_id, with only the tool_result block's own "content" field
// replaced by the given text. Every other field -- role, type, envelope
// shape -- is the real captured session's own, unchanged.
func journeyWriteRefusalTranscript(t *testing.T, texts ...string) string {
	t.Helper()

	data, err := os.ReadFile(journeyRealTranscriptPath(t))
	if err != nil {
		t.Fatalf("read real transcript fixture: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected the real transcript to carry 4 lines, got %d", len(lines))
	}

	var assistantLine, resultLine map[string]interface{}
	if err := json.Unmarshal([]byte(lines[2]), &assistantLine); err != nil {
		t.Fatalf("unmarshal real transcript line 2 (assistant tool_use): %v", err)
	}
	if err := json.Unmarshal([]byte(lines[3]), &resultLine); err != nil {
		t.Fatalf("unmarshal real transcript line 3 (user tool_result): %v", err)
	}

	var out []string
	for i, text := range texts {
		toolUseID := fmt.Sprintf("toolu_refusal_fixture_%d", i)

		a := cloneJSONObject(t, assistantLine)
		aMessage, ok := a["message"].(map[string]interface{})
		if !ok {
			t.Fatalf("derived assistant line's message is not an object: %#v", a["message"])
		}
		aContent, ok := aMessage["content"].([]interface{})
		if !ok || len(aContent) == 0 {
			t.Fatalf("derived assistant line's message.content is not a non-empty array: %#v", aMessage["content"])
		}
		aBlock, ok := aContent[0].(map[string]interface{})
		if !ok {
			t.Fatalf("derived assistant line's first content block is not an object: %#v", aContent[0])
		}
		aBlock["id"] = toolUseID
		aEncoded, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("marshal derived assistant line: %v", err)
		}
		out = append(out, string(aEncoded))

		r := cloneJSONObject(t, resultLine)
		rMessage, ok := r["message"].(map[string]interface{})
		if !ok {
			t.Fatalf("derived tool_result line's message is not an object: %#v", r["message"])
		}
		rContent, ok := rMessage["content"].([]interface{})
		if !ok || len(rContent) == 0 {
			t.Fatalf("derived tool_result line's message.content is not a non-empty array: %#v", rMessage["content"])
		}
		rBlock, ok := rContent[0].(map[string]interface{})
		if !ok {
			t.Fatalf("derived tool_result line's first content block is not an object: %#v", rContent[0])
		}
		rBlock["tool_use_id"] = toolUseID
		rBlock["content"] = text
		rEncoded, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("marshal derived tool_result line: %v", err)
		}
		out = append(out, string(rEncoded))
	}

	path := filepath.Join(t.TempDir(), "derived-refusal-transcript.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write derived transcript to %s: %v", path, err)
	}
	return path
}

// TestPrintedRefusalExtractorFindsARealRefusal builds its fixture by
// calling renderRefusal on a row read from refusalRegistry (not a typed
// block) and asserts the extracted next command equals that row's own
// NextCommand.
func TestPrintedRefusalExtractorFindsARealRefusal(t *testing.T) {
	if len(refusalRegistry) < 2 {
		t.Fatal("refusalRegistry needs at least 2 rows for this test's fixtures")
	}

	t.Run("one refusal block", func(t *testing.T) {
		row := refusalRegistry[0]
		rendered := renderRefusal(refuse(row.ID))

		path := journeyWriteRefusalTranscript(t, rendered)
		refusals, err := journeyPrintedRefusals(path)
		if err != nil {
			t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
		}
		if len(refusals) != 1 {
			t.Fatalf("journeyPrintedRefusals(%s) = %d refusal(s), want 1: %+v", path, len(refusals), refusals)
		}
		if refusals[0].NextCommand != row.NextCommand {
			t.Fatalf("journeyPrintedRefusals(%s)[0].NextCommand = %q, want %q (refusalRegistry row %q)", path, refusals[0].NextCommand, row.NextCommand, row.ID)
		}
	})

	t.Run("two different refusal blocks yield two, in order", func(t *testing.T) {
		row1, row2 := refusalRegistry[0], refusalRegistry[1]
		rendered1 := renderRefusal(refuse(row1.ID))
		rendered2 := renderRefusal(refuse(row2.ID))

		path := journeyWriteRefusalTranscript(t, rendered1, rendered2)
		refusals, err := journeyPrintedRefusals(path)
		if err != nil {
			t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
		}
		if len(refusals) != 2 {
			t.Fatalf("journeyPrintedRefusals(%s) = %d refusal(s), want 2: %+v", path, len(refusals), refusals)
		}
		if refusals[0].NextCommand != row1.NextCommand || refusals[1].NextCommand != row2.NextCommand {
			t.Fatalf("journeyPrintedRefusals(%s) = %+v, want next commands %q then %q in order", path, refusals, row1.NextCommand, row2.NextCommand)
		}
	})

	t.Run("empty transcript yields no refusals and no error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.jsonl")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("write empty transcript: %v", err)
		}
		refusals, err := journeyPrintedRefusals(path)
		if err != nil {
			t.Fatalf("journeyPrintedRefusals(%s): %v, want no error on an empty transcript", path, err)
		}
		if len(refusals) != 0 {
			t.Fatalf("journeyPrintedRefusals(%s) = %+v, want zero refusals from an empty transcript", path, refusals)
		}
	})

	t.Run("a transcript line that does not decode is skipped, no error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "noise.jsonl")
		noise := "not json at all\n{\"also\": \"not a journey transcript entry, missing required shape\"\n"
		if err := os.WriteFile(path, []byte(noise), 0o644); err != nil {
			t.Fatalf("write noise transcript: %v", err)
		}
		refusals, err := journeyPrintedRefusals(path)
		if err != nil {
			t.Fatalf("journeyPrintedRefusals(%s): %v, want no error on undecodable lines", path, err)
		}
		if len(refusals) != 0 {
			t.Fatalf("journeyPrintedRefusals(%s) = %+v, want zero refusals from undecodable lines", path, refusals)
		}
	})
}

// TestPrintedRefusalExtractorIgnoresOrdinaryOutput asserts a transcript
// whose only backticked command sits in ordinary prose -- never preceded by
// the "Next:" label renderRefusal writes -- yields zero printed refusals.
//
// Mutation proof (recorded in 208-07-SUMMARY.md): with the "Next:" label
// requirement in journeyPrintedRefusalNextLineRe dropped (matching any
// backticked command anywhere), this test fails -- confirming the label,
// not merely the backtick pair, is what makes the marker stable.
func TestPrintedRefusalExtractorIgnoresOrdinaryOutput(t *testing.T) {
	ordinary := "Run `aether status` for details, then try again.\nEverything else here looks fine.\n"

	path := journeyWriteRefusalTranscript(t, ordinary)
	refusals, err := journeyPrintedRefusals(path)
	if err != nil {
		t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
	}
	if len(refusals) != 0 {
		t.Fatalf("journeyPrintedRefusals(%s) = %+v, want zero -- a backticked command in ordinary prose is not a refusal", path, refusals)
	}
}

// TestPrintedRefusalExtractorIgnoresAssistantOwnText is WR-05's own
// regression guard (208-REVIEW.md). Before this fix, journeyPrintedRefusals
// also scanned an assistant's own text block for the "Next: `...`" shape,
// on the theory that "the chat relaying what it saw" was an equally
// trustworthy sighting of a real renderRefusal block -- but the assistant's
// own text is free-form model output, not genuine tool output, and
// journeyRunPrintedNextCommands (cmd/journey_live_test.go) actually
// executes whatever NextCommand comes back. This derives a fixture from a
// real captured session (cmd/testdata/stop-hook/show-screen-transcript.jsonl,
// the same discipline journeyWriteRefusalTranscript uses) whose LAST
// assistant text block -- never a tool_result -- carries a real
// renderRefusal(...) block's "Next:" line, and asserts it is found nowhere:
// only genuine tool_result output is ever trusted.
func TestPrintedRefusalExtractorIgnoresAssistantOwnText(t *testing.T) {
	if len(refusalRegistry) == 0 {
		t.Fatal("refusalRegistry needs at least 1 row for this test's fixture")
	}
	row := refusalRegistry[0]
	rendered := renderRefusal(refuse(row.ID))

	data, err := os.ReadFile(filepath.Join("testdata", "stop-hook", "show-screen-transcript.jsonl"))
	if err != nil {
		t.Fatalf("read real transcript fixture: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected the real transcript to carry 5 lines, got %d", len(lines))
	}

	// Line 4 (0-indexed) is the transcript's own final assistant text
	// block -- verified above by decoding every line's shape.
	var lastLine map[string]interface{}
	if err := json.Unmarshal([]byte(lines[4]), &lastLine); err != nil {
		t.Fatalf("unmarshal real transcript line 4 (assistant text): %v", err)
	}
	message, ok := lastLine["message"].(map[string]interface{})
	if !ok || message["role"] != "assistant" {
		t.Fatalf("expected line 4's message.role to be assistant, got: %#v", lastLine["message"])
	}
	content, ok := message["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("expected line 4's message.content to be a non-empty array, got: %#v", message["content"])
	}
	block, ok := content[0].(map[string]interface{})
	if !ok || block["type"] != "text" {
		t.Fatalf("expected line 4's first content block to be type=text, got: %#v", content[0])
	}
	block["text"] = rendered
	mutated, err := json.Marshal(lastLine)
	if err != nil {
		t.Fatalf("marshal mutated transcript line: %v", err)
	}
	lines[4] = string(mutated)

	path := filepath.Join(t.TempDir(), "assistant-text-refusal.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write derived transcript: %v", err)
	}

	refusals, err := journeyPrintedRefusals(path)
	if err != nil {
		t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
	}
	if len(refusals) != 0 {
		t.Fatalf("journeyPrintedRefusals(%s) = %+v, want zero -- a refusal block sitting only in the assistant's own text must never be trusted", path, refusals)
	}
}

// TestMenuFormNextCommandMapsBackToTheRuntimeCommand table-drives the three
// behaviour cases and asserts arguments survive unchanged. It derives the
// menu form by calling platformCommandName rather than typing "/ant-…" by
// hand, so the fixture can never silently drift from what the forward
// mapping actually produces.
func TestMenuFormNextCommandMapsBackToTheRuntimeCommand(t *testing.T) {
	if !wrapperCommandNames["colonize"] {
		t.Fatal("this test needs \"colonize\" to be a real wrapper command")
	}

	cases := []struct {
		name    string
		printed string
		want    string
		wantOK  bool
	}{
		{
			name:    "menu form with arguments preserved",
			printed: platformCommandName("colonize", "claude") + " --plan-only --force-resurvey",
			want:    "aether colonize --plan-only --force-resurvey",
			wantOK:  true,
		},
		{
			name:    "already the runtime form, no wrapper needed",
			printed: "aether flag-resolve --id x",
			want:    "aether flag-resolve --id x",
			wantOK:  true,
		},
		{
			name:    "menu form naming a verb with no wrapper -- neither form",
			printed: "/ant-not-a-real-verb --whatever",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "plain text matching neither the menu nor the runtime form",
			printed: "make install",
			want:    "",
			wantOK:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := printedCommandToRuntimeCommand(tc.printed)
			if ok != tc.wantOK {
				t.Fatalf("printedCommandToRuntimeCommand(%q) ok = %v, want %v (got %q)", tc.printed, ok, tc.wantOK, got)
			}
			if got != tc.want {
				t.Fatalf("printedCommandToRuntimeCommand(%q) = %q, want %q", tc.printed, got, tc.want)
			}
		})
	}
}

// TestPrintedRefusalExtractorFindsAHostRelayedRefusal proves
// journeyPrintedRefusals also finds refusal.Error()'s one-line host-relayed
// form -- the shape the TypeScript host actually relays to a chat
// (sanitizeBridgeMessage, .aether/ts-host/src/go-bridge.ts) -- not just the
// drawn block form. The fixture is real captured text
// (cmd/testdata/refusals/host-relay-refusal.txt), copied verbatim from the
// one real journey walk's own on-disk session transcript
// (208-JOURNEY-RUN.md), never typed by hand.
func TestPrintedRefusalExtractorFindsAHostRelayedRefusal(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "refusals", "host-relay-refusal.txt"))
	if err != nil {
		t.Fatalf("read host-relay-refusal.txt fixture: %v", err)
	}
	fixture := string(data)
	if !strings.Contains(fixture, "— next: ") {
		t.Fatalf("fixture does not carry the relayed marker at all -- fixture may have been mis-captured: %q", fixture)
	}

	path := journeyWriteRefusalTranscript(t, fixture)
	refusals, err := journeyPrintedRefusals(path)
	if err != nil {
		t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
	}
	if len(refusals) != 1 {
		t.Fatalf("journeyPrintedRefusals(%s) = %d refusal(s), want 1: %+v", path, len(refusals), refusals)
	}
	want := "aether colonize --force-resurvey"
	if refusals[0].NextCommand != want {
		t.Fatalf("journeyPrintedRefusals(%s)[0].NextCommand = %q, want %q", path, refusals[0].NextCommand, want)
	}
	if strings.HasSuffix(refusals[0].NextCommand, ".") || strings.HasSuffix(refusals[0].NextCommand, `"`) {
		t.Fatalf("journeyPrintedRefusals(%s)[0].NextCommand carries trailing punctuation: %q", path, refusals[0].NextCommand)
	}
}

// TestPrintedRefusalExtractorIgnoresRelayedProse proves an ordinary
// sentence -- either one that never carries the " — next: " marker at all,
// or one that carries the marker but whose tail is not itself a runnable
// command -- yields zero refusals, never a refusal fed to the subprocess
// runner that would then fail the step for no real reason.
func TestPrintedRefusalExtractorIgnoresRelayedProse(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{
			name: "no marker at all, just the word next",
			text: "The next step is to review the logs before continuing.\n",
		},
		{
			name: "marker present but the tail is not a runnable command",
			text: "Checking configuration — next: check with your admin before proceeding.\n",
		},
		{
			// The whole reason the relayed pattern carries Aether's own
			// host-relay prefix: whatever this extractor returns is later
			// EXECUTED as a subprocess by journeyRunPrintedNextCommands, so
			// a coincidental marker in genuine unrelated tool output (a
			// build log, an echoed commit message) must never be treated as
			// a refusal merely because a runnable-looking command follows
			// it. Without the anchor this line matches and gets run.
			name: "runnable tail but no host-relay prefix -- unrelated captured output",
			text: "commit a1b2c3d: document recovery — next: aether colonize --force-resurvey\n",
		},
		{
			name: "host-relay prefix on a different line from the marker",
			text: "Fatal: Go command failed: colonize --plan-only\nSomething else — next: aether colonize --force-resurvey\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := journeyWriteRefusalTranscript(t, tc.text)
			refusals, err := journeyPrintedRefusals(path)
			if err != nil {
				t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
			}
			if len(refusals) != 0 {
				t.Fatalf("journeyPrintedRefusals(%s) = %+v, want zero -- relayed prose with no runnable command tail is not a refusal", path, refusals)
			}
		})
	}
}

// TestRelayedCommandLosesTrailingPunctuation proves the trailing-punctuation
// trim is a claim a command can fail on, rather than an assertion that only
// passes because the one captured fixture happens not to exercise it. The
// relayed form has no closing delimiter, so a relayed message quoted or
// bracketed inside surrounding prose carries that stray tail character into
// the command; without the trim the command maps to nothing and the refusal
// is silently lost. The command itself is derived from a real registry row,
// never typed.
func TestRelayedCommandLosesTrailingPunctuation(t *testing.T) {
	var row refusalRow
	for _, candidate := range refusalRegistry {
		if strings.TrimSpace(candidate.NextCommand) != "" {
			row = candidate
			break
		}
	}
	if strings.TrimSpace(row.NextCommand) == "" {
		t.Fatal("refusalRegistry needs at least one row naming a next command for this fixture")
	}

	for _, trailing := range []string{".", "\"", "'", ")", ",", ";", "\"."} {
		t.Run("trailing "+trailing, func(t *testing.T) {
			text := "Fatal: Go command failed: " + row.ID + ": " + strings.TrimSpace(row.What) +
				" — next: " + row.NextCommand + trailing + "\n"
			path := journeyWriteRefusalTranscript(t, text)
			refusals, err := journeyPrintedRefusals(path)
			if err != nil {
				t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
			}
			if len(refusals) != 1 {
				t.Fatalf("journeyPrintedRefusals(%s) = %+v, want exactly 1 -- a relayed command followed by %q must still be found", path, refusals, trailing)
			}
			if got := refusals[0].NextCommand; got != row.NextCommand {
				t.Fatalf("NextCommand = %q, want %q -- the trailing %q must not ride along into the command", got, row.NextCommand, trailing)
			}
		})
	}
}

// TestPrintedRefusalsAreDeduplicatedWithinOneTranscript proves the same
// next command appearing twice in one transcript -- once relayed on the
// error stream, once on the output stream, or repeated across steps --
// yields exactly one entry, first-seen order preserved, not two.
func TestPrintedRefusalsAreDeduplicatedWithinOneTranscript(t *testing.T) {
	if len(refusalRegistry) == 0 {
		t.Fatal("refusalRegistry needs at least 1 row for this test's fixture")
	}
	row := refusalRegistry[0]

	blockForm := renderRefusal(refuse(row.ID))
	relayedForm := "Fatal: Go command failed: " + row.ID + ": " + strings.TrimSpace(row.What) + " — next: " + row.NextCommand + "\n"

	path := journeyWriteRefusalTranscript(t, blockForm, relayedForm)
	refusals, err := journeyPrintedRefusals(path)
	if err != nil {
		t.Fatalf("journeyPrintedRefusals(%s): %v", path, err)
	}
	if len(refusals) != 1 {
		t.Fatalf("journeyPrintedRefusals(%s) = %d refusal(s), want 1 (the same next command relayed twice must dedupe): %+v", path, len(refusals), refusals)
	}
	if refusals[0].NextCommand != row.NextCommand {
		t.Fatalf("journeyPrintedRefusals(%s)[0].NextCommand = %q, want %q", path, refusals[0].NextCommand, row.NextCommand)
	}
}
