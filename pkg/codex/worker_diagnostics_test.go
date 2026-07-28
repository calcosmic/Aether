package codex

import (
	"encoding/json"
	"strings"
	"testing"
)

// A parse failure must name the field that broke, not the transport envelope
// wrapped around it. Without this the operator is told "no JSON found in
// output" for a payload that is entirely present and almost entirely valid.
func TestHostedParseErrorNamesTheOffendingField(t *testing.T) {
	inner := "```json\n" + `{
  "status": "completed",
  "summary": {"unexpected": "object where a string belongs"},
  "blockers": ["something"]
}` + "\n```"

	raw, err := json.Marshal(map[string]interface{}{
		"is_error": false,
		"subtype":  "success",
		"result":   inner,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	_, parseErr := parseHostedWorkerOutput("claude", combinedWorkerOutput(string(raw), ""))
	if parseErr == nil {
		t.Fatal("expected a parse error")
	}
	if !strings.Contains(parseErr.Error(), "summary") {
		t.Fatalf("error does not name the offending field: %v", parseErr)
	}
}

// The debug artifact must retain the end of an oversized output. The worker's
// answer is at the tail; the head is CLI usage metadata.
func TestWorkerOutputExcerptKeepsTail(t *testing.T) {
	head := strings.Repeat("H", 9000)
	tail := `"next_worker_instructions": "the field that broke"}`
	excerpt := workerOutputExcerpt(head + tail)

	if !strings.Contains(excerpt, "next_worker_instructions") {
		t.Fatal("excerpt dropped the tail, where malformed fields live")
	}
	if !strings.HasPrefix(excerpt, "HHH") {
		t.Fatal("excerpt dropped the head")
	}
	if !strings.Contains(excerpt, "omitted") {
		t.Fatal("excerpt does not mark the omission")
	}
}

// Short outputs must pass through untouched.
func TestWorkerOutputExcerptLeavesShortOutputIntact(t *testing.T) {
	value := `{"status":"completed","summary":"short"}`
	if got := workerOutputExcerpt(value); got != value {
		t.Fatalf("short output altered:\n got: %q\nwant: %q", got, value)
	}
}
