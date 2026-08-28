package llm

import (
	"encoding/json"
	"reflect"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

// Phase 196 plan 02, D-05: one authoritative token count.
//
// This client's Usage carried only input and output. Anthropic reports input,
// cache_read and cache_creation as DISJOINT counts, so a value carrying two of
// the four is not "a slightly smaller number" — it is the exact shape of the
// 186x undercount already recorded in pkg/codex/usage.go's billedTotal comment,
// alive on a second accounting lane. Every expected number below is a literal
// taken from the provider's own documented disjoint-column example
// (50 input, 100,000 cache read, 2,000 cache creation, 500 output); none is
// produced by calling the code under test.
const (
	docExampleInputTokens         = 50
	docExampleCacheReadTokens     = 100000
	docExampleCacheCreationTokens = 2000
	docExampleOutputTokens        = 500
)

// TestUsageCarriesEveryBilledColumn covers the single-response path: what the
// SDK reported on a non-streamed message must reach the caller in full.
func TestUsageCarriesEveryBilledColumn(t *testing.T) {
	raw := `{
		"id": "msg_disjoint",
		"type": "message",
		"role": "assistant",
		"model": "claude-sonnet-4-20250514",
		"stop_reason": "end_turn",
		"content": [{"type": "text", "text": "hi"}],
		"usage": {
			"input_tokens": 50,
			"cache_read_input_tokens": 100000,
			"cache_creation_input_tokens": 2000,
			"output_tokens": 500
		}
	}`

	var msg anthropic.Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal provider message: %v", err)
	}

	got := convertSDKMessage(&msg)
	if got == nil {
		t.Fatal("convertSDKMessage returned nil for a real provider message")
	}

	if got.Usage.InputTokens != docExampleInputTokens {
		t.Errorf("InputTokens = %d, want %d", got.Usage.InputTokens, docExampleInputTokens)
	}
	if got.Usage.CacheReadInputTokens != docExampleCacheReadTokens {
		t.Errorf("CacheReadInputTokens = %d, want %d — a cache-blind usage value is the 186x-undercount shape",
			got.Usage.CacheReadInputTokens, docExampleCacheReadTokens)
	}
	if got.Usage.CacheCreationInputTokens != docExampleCacheCreationTokens {
		t.Errorf("CacheCreationInputTokens = %d, want %d", got.Usage.CacheCreationInputTokens, docExampleCacheCreationTokens)
	}
	if got.Usage.OutputTokens != docExampleOutputTokens {
		t.Errorf("OutputTokens = %d, want %d", got.Usage.OutputTokens, docExampleOutputTokens)
	}
}

// TestStreamedUsageCarriesEveryBilledColumn covers the streamed path. A column
// populated on one path and silently zero on the other reproduces exactly the
// divergence D-05 exists to close, so the streamed result must match the
// single-response result column for column.
func TestStreamedUsageCarriesEveryBilledColumn(t *testing.T) {
	t.Run("message_start reports the input-side columns and message_delta the output", func(t *testing.T) {
		events := []sseEvent{
			{eventType: "message_start", data: `{"type":"message_start","message":{"id":"msg_stream_cache","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":50,"cache_read_input_tokens":100000,"cache_creation_input_tokens":2000,"output_tokens":0}}}`},
			{eventType: "content_block_delta", data: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`},
			{eventType: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":500}}`},
			{eventType: "message_stop", data: `{"type":"message_stop"}`},
		}

		server := newSSEServer(t, events)
		defer server.Close()

		result, err := AccumulateStream(newTestStream(t, server.URL), nil)
		if err != nil {
			t.Fatalf("AccumulateStream() error = %v", err)
		}

		if result.Usage.InputTokens != docExampleInputTokens {
			t.Errorf("InputTokens = %d, want %d", result.Usage.InputTokens, docExampleInputTokens)
		}
		if result.Usage.CacheReadInputTokens != docExampleCacheReadTokens {
			t.Errorf("CacheReadInputTokens = %d, want %d", result.Usage.CacheReadInputTokens, docExampleCacheReadTokens)
		}
		if result.Usage.CacheCreationInputTokens != docExampleCacheCreationTokens {
			t.Errorf("CacheCreationInputTokens = %d, want %d", result.Usage.CacheCreationInputTokens, docExampleCacheCreationTokens)
		}
		if result.Usage.OutputTokens != docExampleOutputTokens {
			t.Errorf("OutputTokens = %d, want %d", result.Usage.OutputTokens, docExampleOutputTokens)
		}
	})

	t.Run("a cumulative message_delta restating every column does not lose one", func(t *testing.T) {
		events := []sseEvent{
			{eventType: "message_start", data: `{"type":"message_start","message":{"id":"msg_stream_cum","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":50,"cache_read_input_tokens":100000,"cache_creation_input_tokens":2000,"output_tokens":0}}}`},
			{eventType: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":50,"cache_read_input_tokens":100000,"cache_creation_input_tokens":2000,"output_tokens":500}}`},
			{eventType: "message_stop", data: `{"type":"message_stop"}`},
		}

		server := newSSEServer(t, events)
		defer server.Close()

		result, err := AccumulateStream(newTestStream(t, server.URL), nil)
		if err != nil {
			t.Fatalf("AccumulateStream() error = %v", err)
		}

		want := Usage{
			InputTokens:              docExampleInputTokens,
			CacheReadInputTokens:     docExampleCacheReadTokens,
			CacheCreationInputTokens: docExampleCacheCreationTokens,
			OutputTokens:             docExampleOutputTokens,
		}
		if result.Usage != want {
			t.Errorf("Usage = %+v, want %+v", result.Usage, want)
		}
	})
}

// TestUsageWithoutCacheActivityIsUnchanged pins the no-regression half: a
// response with no cache activity still reports the two original columns and
// zero for the two new ones, on both paths.
func TestUsageWithoutCacheActivityIsUnchanged(t *testing.T) {
	t.Run("single response", func(t *testing.T) {
		raw := `{"id":"msg_nocache","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","stop_reason":"end_turn","content":[],"usage":{"input_tokens":20,"output_tokens":5}}`
		var msg anthropic.Message
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			t.Fatalf("unmarshal provider message: %v", err)
		}
		got := convertSDKMessage(&msg)
		want := Usage{InputTokens: 20, OutputTokens: 5}
		if got.Usage != want {
			t.Errorf("Usage = %+v, want %+v", got.Usage, want)
		}
	})

	t.Run("streamed response", func(t *testing.T) {
		events := []sseEvent{
			{eventType: "message_start", data: `{"type":"message_start","message":{"id":"msg_nocache_stream","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":20,"output_tokens":0}}}`},
			{eventType: "message_delta", data: `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`},
			{eventType: "message_stop", data: `{"type":"message_stop"}`},
		}
		server := newSSEServer(t, events)
		defer server.Close()

		result, err := AccumulateStream(newTestStream(t, server.URL), nil)
		if err != nil {
			t.Fatalf("AccumulateStream() error = %v", err)
		}
		want := Usage{InputTokens: 20, OutputTokens: 5}
		if result.Usage != want {
			t.Errorf("Usage = %+v, want %+v", result.Usage, want)
		}
	})

	// D-05's other half: the authoritative total lives on codex.WorkerUsage
	// and is reached through its single exported entry point. A second
	// summation in a second package is how the two lanes drifted apart in the
	// first place, so this type must expose no total-computing method of its
	// own.
	t.Run("the usage type exposes no total-computing method", func(t *testing.T) {
		typ := reflect.TypeOf(Usage{})
		for i := 0; i < typ.NumMethod(); i++ {
			t.Errorf("llm.Usage declares method %s — this type must expose no method at all, and above all no total: "+
				"the authoritative total is codex.WorkerUsage.BilledTotalTokens(), and a second implementation of that "+
				"arithmetic in a second package is the divergence D-05 closes", typ.Method(i).Name)
		}
		ptr := reflect.TypeOf(&Usage{})
		for i := 0; i < ptr.NumMethod(); i++ {
			t.Errorf("*llm.Usage declares method %s — see above", ptr.Method(i).Name)
		}
	})
}
