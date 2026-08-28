package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/llm"
)

// Phase 196 plan 02, D-05. The pool published a two-column usage payload and
// handed a two-number callback to whoever asked, both derived from a
// cache-blind llm.Usage. That is a second accounting path reporting a
// different answer from the provider parser for the same run — and a cost line
// rendered over two disagreeing sources is a confidently wrong number.
//
// Every expected value below is a literal from the provider's documented
// disjoint-column example (50 input, 100,000 cache read, 2,000 cache creation,
// 500 output); none is produced by calling the code under test.

// streamResultWithDocumentedColumns builds the one input every test in this
// file carries through the pool.
func streamResultWithDocumentedColumns() *llm.StreamResult {
	return &llm.StreamResult{
		Text:       "done",
		Role:       "assistant",
		Model:      "claude-sonnet-4-20250514",
		StopReason: "end_turn",
		Usage: llm.Usage{
			InputTokens:              50,
			CacheReadInputTokens:     100000,
			CacheCreationInputTokens: 2000,
			OutputTokens:             500,
		},
	}
}

// TestPoolCompletionEventCarriesEveryColumn asserts the published payload
// carries all four billed columns, not two.
func TestPoolCompletionEventCarriesEveryColumn(t *testing.T) {
	bus, _ := newTestBus(t)
	sub, err := bus.Subscribe("agent.*")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	h := &poolStreamHandler{agentName: "Mason-67", caste: CasteBuilder, bus: bus}
	h.OnComplete(streamResultWithDocumentedColumns())

	var payload map[string]interface{}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case evt := <-sub:
			if evt.Topic != "agent.Mason-67.complete" {
				continue
			}
			if err := json.Unmarshal(evt.Payload, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
		case <-deadline:
			t.Fatal("timed out waiting for the completion event")
		}
		if payload != nil {
			break
		}
	}

	usage, ok := payload["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("completion payload has no usage object: %+v", payload)
	}

	for _, want := range []struct {
		key   string
		value float64
	}{
		{"input_tokens", 50},
		{"cached_input_tokens", 100000},
		{"cache_creation_tokens", 2000},
		{"output_tokens", 500},
		{"total_tokens", 102550},
	} {
		got, present := usage[want.key]
		if !present {
			t.Errorf("completion payload usage has no %q column — a cache-blind payload is the 186x-undercount shape", want.key)
			continue
		}
		if got != want.value {
			t.Errorf("usage[%q] = %v, want %v", want.key, got, want.value)
		}
	}
}

// TestPoolUsageCallbackHandsOverAuthoritativeUsage asserts the callback hands
// its caller the one authoritative usage type rather than two loose numbers.
func TestPoolUsageCallbackHandsOverAuthoritativeUsage(t *testing.T) {
	var got codex.WorkerUsage
	var called bool

	h := &poolStreamHandler{
		agentName: "Mason-67",
		caste:     CasteBuilder,
		onTokenUsage: func(usage codex.WorkerUsage) {
			got = usage
			called = true
		},
	}
	h.OnComplete(streamResultWithDocumentedColumns())

	if !called {
		t.Fatal("usage callback was never invoked")
	}
	if got.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", got.InputTokens)
	}
	if got.CachedInputTokens != 100000 {
		t.Errorf("CachedInputTokens = %d, want 100000", got.CachedInputTokens)
	}
	if got.CacheCreationTokens != 2000 {
		t.Errorf("CacheCreationTokens = %d, want 2000", got.CacheCreationTokens)
	}
	if got.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", got.OutputTokens)
	}
	if got.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want the model the stream reported", got.Model)
	}
	if got.BilledTotalTokens() != 102550 {
		t.Errorf("BilledTotalTokens() = %d, want 102550", got.BilledTotalTokens())
	}
}

// TestPoolUsageIsNeverProviderGrade holds the source-tag boundary
// codex.WorkerUsage documents: nothing outside ParseUsage may set the
// provider tag. A pool-converted row is a genuine measurement, so it must not
// be tagged as a guess either — that would put a real figure in the ledger's
// estimated subtotal.
func TestPoolUsageIsNeverProviderGrade(t *testing.T) {
	var got codex.WorkerUsage
	h := &poolStreamHandler{
		agentName:    "Mason-67",
		caste:        CasteBuilder,
		onTokenUsage: func(usage codex.WorkerUsage) { got = usage },
	}
	h.OnComplete(streamResultWithDocumentedColumns())

	if got.Measured() {
		t.Error("a pool-converted row reported itself as provider-grade; only ParseUsage may set that tag")
	}
	if got.Source == codex.UsageSourceProvider {
		t.Errorf("Source = %q, want anything but the provider tag", got.Source)
	}
	if got.Estimated() {
		t.Error("a pool-converted row is a real measurement and must not be tagged as a guess — that would land it in the ledger's estimated subtotal")
	}

	t.Run("a stream that reported nothing carries no figure at all", func(t *testing.T) {
		var empty codex.WorkerUsage
		handler := &poolStreamHandler{
			agentName:    "Roam-90",
			caste:        CasteScout,
			onTokenUsage: func(usage codex.WorkerUsage) { empty = usage },
		}
		handler.OnComplete(&llm.StreamResult{Text: "done", Model: "claude-sonnet-4-20250514"})
		if !empty.Empty() {
			t.Errorf("usage = %+v, want an empty value — D-01 as amended: an unmeasured worker carries no number at all", empty)
		}
	})
}
