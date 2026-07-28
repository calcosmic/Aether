package codex

import (
	"encoding/json"
	"strings"
	"testing"
)

// A worker that answers a list-typed field with a single string must not cost
// the colony the whole payload. This is the M4L plan failure of 27 July 2026:
// an 11 KB scout research result was discarded because
// handoff.next_worker_instructions arrived as a string instead of an array,
// and three /ant-plan runs failed in a row.
func TestParseWorkerOutputAcceptsScalarForListFields(t *testing.T) {
	payload := "```json\n" + `{
  "status": "completed",
  "summary": "Surveyed the survey artifacts and six targeted repo reads.",
  "files_created": [],
  "files_modified": [],
  "tests_written": [],
  "blockers": ["No pattern combines Instrument with Step Sequencer"],
  "handoff": {
    "verification_status": "not_applicable_research_only",
    "open_decisions": ["Instrument or MIDI effect for the first slice?"],
    "next_worker_instructions": "Phase 1 should confirm which device class hosts the engine.",
    "do_not_repeat": "Do not re-run the generic top-level survey.",
    "freshness": "as of 2026-07-27"
  }
}` + "\n```"

	claims, err := ParseWorkerOutput(payload)
	if err != nil {
		t.Fatalf("ParseWorkerOutput rejected a scalar list field: %v", err)
	}
	if claims.Status != "completed" {
		t.Fatalf("Status = %q, want %q", claims.Status, "completed")
	}
	if got := []string(claims.Handoff.NextWorkerInstructions); len(got) != 1 ||
		!strings.Contains(got[0], "which device class") {
		t.Fatalf("NextWorkerInstructions = %#v, want the single instruction promoted to a one-element list", got)
	}
	if got := []string(claims.Handoff.DoNotRepeat); len(got) != 1 {
		t.Fatalf("DoNotRepeat = %#v, want one element", got)
	}
	if got := []string(claims.Blockers); len(got) != 1 {
		t.Fatalf("Blockers = %#v, want the array form still to work", got)
	}
}

// Top-level list fields get the same tolerance, and null must stay empty
// rather than becoming a one-element list of "null".
func TestParseWorkerOutputScalarAndNullTopLevelLists(t *testing.T) {
	payload := `{
  "status": "completed",
  "summary": "one file touched",
  "files_modified": "cmd/main.go",
  "files_created": null,
  "blockers": []
}`

	claims, err := ParseWorkerOutput(payload)
	if err != nil {
		t.Fatalf("ParseWorkerOutput failed: %v", err)
	}
	if got := []string(claims.FilesModified); len(got) != 1 || got[0] != "cmd/main.go" {
		t.Fatalf("FilesModified = %#v, want [cmd/main.go]", got)
	}
	if got := []string(claims.FilesCreated); len(got) != 0 {
		t.Fatalf("FilesCreated = %#v, want empty for null", got)
	}
}

// An empty string must not become a list holding one empty entry — that would
// put content-free noise into the next worker's handoff context.
func TestStringListDropsEmptyScalar(t *testing.T) {
	var list stringList
	if err := json.Unmarshal([]byte(`"   "`), &list); err != nil {
		t.Fatalf("unmarshal blank scalar: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("blank scalar produced %#v, want empty list", list)
	}
}

// Tolerance must not become silence: a genuinely undecodable value still errors.
func TestStringListRejectsUndecodableValue(t *testing.T) {
	var list stringList
	if err := json.Unmarshal([]byte(`{"nested":"object"}`), &list); err == nil {
		t.Fatal("expected an error for an object in a string-list field")
	}
}
