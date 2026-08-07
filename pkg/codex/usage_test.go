package codex

import "testing"

// TestParseUsageClaudeResultEvent pins the Claude stream-json shape: a terminal
// result event carrying usage and cost.
func TestParseUsageClaudeResultEvent(t *testing.T) {
	raw := `{"type":"system","subtype":"init","session_id":"abc"}
{"type":"assistant","message":{"content":[{"type":"text","text":"working"}]}}
{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.4213,"model":"claude-sonnet-5","usage":{"input_tokens":11542,"cache_read_input_tokens":98000,"output_tokens":3310}}`

	usage, ok := ParseUsage(raw)
	if !ok {
		t.Fatal("expected usage to be found in a Claude result event")
	}
	if !usage.Measured() {
		t.Errorf("source = %q, want provider", usage.Source)
	}
	if usage.InputTokens != 11542 {
		t.Errorf("input = %d, want 11542", usage.InputTokens)
	}
	if usage.CachedInputTokens != 98000 {
		t.Errorf("cached = %d, want 98000 — cache reads are where savings show", usage.CachedInputTokens)
	}
	if usage.OutputTokens != 3310 {
		t.Errorf("output = %d, want 3310", usage.OutputTokens)
	}
	if usage.USDCost != 0.4213 {
		t.Errorf("cost = %v, want 0.4213", usage.USDCost)
	}
	if usage.Model != "claude-sonnet-5" {
		t.Errorf("model = %q, want claude-sonnet-5", usage.Model)
	}
	if usage.TotalTokens != 11542+3310 {
		t.Errorf("total = %d, want input+output when not reported", usage.TotalTokens)
	}
}

// TestParseUsageCodexTokenCountEvent pins the Codex shape, including that a
// later cumulative event supersedes an earlier one.
func TestParseUsageCodexTokenCountEvent(t *testing.T) {
	raw := `{"type":"token_count","total_token_usage":{"input_tokens":100,"output_tokens":20,"total_tokens":120}}
{"type":"agent_message","message":"still going"}
{"type":"token_count","total_token_usage":{"input_tokens":8400,"cached_input_tokens":6000,"output_tokens":1200,"total_tokens":9600}}`

	usage, ok := ParseUsage(raw)
	if !ok {
		t.Fatal("expected usage from token_count events")
	}
	if usage.TotalTokens != 9600 {
		t.Errorf("total = %d, want 9600 — the last cumulative event must win", usage.TotalTokens)
	}
	if usage.CachedInputTokens != 6000 {
		t.Errorf("cached = %d, want 6000", usage.CachedInputTokens)
	}
	if !usage.Measured() {
		t.Errorf("source = %q, want provider", usage.Source)
	}
}

// TestParseUsageIgnoresNoiseAndPlainOutput keeps the parser from inventing
// numbers out of ordinary worker chatter.
func TestParseUsageIgnoresNoiseAndPlainOutput(t *testing.T) {
	for _, raw := range []string{
		"",
		"   \n\n",
		"Running tests...\nAll good.\n",
		`{"type":"assistant","message":{"content":[]}}`,
		"not json at all { input_tokens: 5000 }",
	} {
		if usage, ok := ParseUsage(raw); ok {
			t.Errorf("ParseUsage(%q) reported usage %+v; want none", raw, usage)
		}
	}
}

// TestEstimateUsageIsNeverMistakenForAMeasurement is the load-bearing property
// of the whole ledger. A run whose worker reported nothing must still appear —
// a missing row silently shrinks the total and makes a regression look like an
// improvement — but it must be impossible to read the estimate as measured.
func TestEstimateUsageIsNeverMistakenForAMeasurement(t *testing.T) {
	usage := EstimateUsage(24000)
	if usage.Measured() {
		t.Error("an estimate must not report itself as measured")
	}
	if usage.Source != UsageSourceEstimate {
		t.Errorf("source = %q, want %q", usage.Source, UsageSourceEstimate)
	}
	if usage.InputTokens == 0 {
		t.Error("an estimate should still carry a number so the row is not empty")
	}
	if usage.Empty() {
		t.Error("an estimate must not look like an absent record")
	}

	// A zero-length prompt still produces a labelled row, not a blank.
	zero := EstimateUsage(0)
	if zero.Source != UsageSourceEstimate {
		t.Errorf("zero-char estimate source = %q, want %q", zero.Source, UsageSourceEstimate)
	}
}

// TestParseUsageToleratesNestedCodexShape covers the info.total_token_usage
// nesting some codex versions emit, so a transport change does not silently
// drop every measurement back to estimates.
func TestParseUsageToleratesNestedCodexShape(t *testing.T) {
	raw := `{"type":"token_count","info":{"total_token_usage":{"input_tokens":500,"output_tokens":100,"total_tokens":600}}}`
	usage, ok := ParseUsage(raw)
	if !ok {
		t.Fatal("expected usage from nested codex shape")
	}
	if usage.TotalTokens != 600 {
		t.Errorf("total = %d, want 600", usage.TotalTokens)
	}
}

// TestEveryDispatchLeavesAUsageRow is the invariant that makes the ledger
// worth reading. Attaching usage at each dispatcher's own return would mean
// eight sites today and a ninth whenever a transport is added — and a dispatch
// path that quietly skips the ledger does not produce an obviously wrong
// number, it produces a total that is too low, which reads as an improvement.
//
// AttachWorkerUsage runs at the one boundary all dispatches converge on. This
// asserts the property directly: whatever a dispatcher returns, and whether or
// not the provider reported anything, the result carries a row.
func TestEveryDispatchLeavesAUsageRow(t *testing.T) {
	config := WorkerConfig{
		WorkerName:     "Hammer-1",
		Caste:          "builder",
		TaskBrief:      "do the thing",
		ContextCapsule: "colony context",
	}

	for _, tc := range []struct {
		name       string
		result     WorkerResult
		wantSource string
	}{
		{
			name:       "provider reported usage",
			result:     WorkerResult{RawOutput: `{"type":"result","usage":{"input_tokens":10,"output_tokens":2}}`},
			wantSource: UsageSourceProvider,
		},
		{
			name:       "worker produced no parseable usage",
			result:     WorkerResult{RawOutput: "plain prose, no events"},
			wantSource: UsageSourceEstimate,
		},
		{
			name:       "worker timed out with no output at all",
			result:     WorkerResult{Status: "timeout"},
			wantSource: UsageSourceEstimate,
		},
		{
			name:       "worker failed",
			result:     WorkerResult{Status: "failed", RawOutput: "boom"},
			wantSource: UsageSourceEstimate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := AttachWorkerUsage(tc.result, config)
			if got.Usage.Empty() {
				t.Fatal("dispatch left no usage row; the ledger total would silently under-report")
			}
			if got.Usage.Source != tc.wantSource {
				t.Errorf("source = %q, want %q", got.Usage.Source, tc.wantSource)
			}
		})
	}
}

// TestAttachWorkerUsageDoesNotOverwriteAProviderFigure keeps the boundary from
// clobbering a dispatcher that already reported its own usage.
func TestAttachWorkerUsageDoesNotOverwriteAProviderFigure(t *testing.T) {
	existing := WorkerResult{
		Usage:     WorkerUsage{InputTokens: 999, Source: UsageSourceProvider},
		RawOutput: `{"type":"result","usage":{"input_tokens":1,"output_tokens":1}}`,
	}
	got := AttachWorkerUsage(existing, WorkerConfig{})
	if got.Usage.InputTokens != 999 {
		t.Errorf("input = %d, want the pre-existing 999 preserved", got.Usage.InputTokens)
	}
}
