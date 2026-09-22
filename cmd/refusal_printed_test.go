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
