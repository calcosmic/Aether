package cmd

import (
	"encoding/json"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestInternalWorkerResultCarriesProviderUsage is the SPEND-01 regression
// test for the direct/Codex-CLI dispatch path: a genuine provider
// measurement already exists in memory on codex.WorkerResult.Usage by the
// time mapInternalWorkerResult runs, and before Phase 174 it was silently
// dropped rather than reaching the durable adapter result.
//
// Every expected value here is a literal, never a call back into
// BilledTotalTokens() or TotalInputTokens() -- this is the same discipline
// pkg/codex/usage_test.go uses to avoid restating the parser's own formula.
func TestInternalWorkerResultCarriesProviderUsage(t *testing.T) {
	usage := codex.WorkerUsage{
		InputTokens:         50,
		CachedInputTokens:   100000,
		CacheCreationTokens: 2000,
		OutputTokens:        500,
		TotalTokens:         102550,
		Source:              codex.UsageSourceProvider,
	}

	t.Run("mapping copies the value field for field", func(t *testing.T) {
		result := codex.WorkerResult{
			WorkerName: "Builder-Test",
			Caste:      "builder",
			Status:     "completed",
			Usage:      usage,
		}

		mapped := mapInternalWorkerResult(result, nil)
		if mapped == nil {
			t.Fatal("mapInternalWorkerResult returned nil")
		}
		if mapped.Usage != usage {
			t.Errorf("Usage = %+v, want %+v", mapped.Usage, usage)
		}
	})

	t.Run("JSON round-trip preserves every disjoint column, the total, and the source tag", func(t *testing.T) {
		result := codex.WorkerResult{
			WorkerName: "Builder-Test",
			Caste:      "builder",
			Status:     "completed",
			Usage:      usage,
		}
		mapped := mapInternalWorkerResult(result, nil)

		raw, err := json.Marshal(mapped)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var roundTripped internalWorkerResult
		if err := json.Unmarshal(raw, &roundTripped); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if roundTripped.Usage.InputTokens != 50 {
			t.Errorf("InputTokens = %d, want 50", roundTripped.Usage.InputTokens)
		}
		if roundTripped.Usage.CachedInputTokens != 100000 {
			t.Errorf("CachedInputTokens = %d, want 100000", roundTripped.Usage.CachedInputTokens)
		}
		if roundTripped.Usage.CacheCreationTokens != 2000 {
			t.Errorf("CacheCreationTokens = %d, want 2000", roundTripped.Usage.CacheCreationTokens)
		}
		if roundTripped.Usage.OutputTokens != 500 {
			t.Errorf("OutputTokens = %d, want 500", roundTripped.Usage.OutputTokens)
		}
		if roundTripped.Usage.TotalTokens != 102550 {
			t.Errorf("TotalTokens = %d, want 102550", roundTripped.Usage.TotalTokens)
		}
		if roundTripped.Usage.Source != codex.UsageSourceProvider {
			t.Errorf("Source = %q, want %q", roundTripped.Usage.Source, codex.UsageSourceProvider)
		}
	})

	t.Run("a zero-value Usage carries no usage figures through the round-trip", func(t *testing.T) {
		// NOTE: internalWorkerResult.Usage is codex.WorkerUsage BY VALUE (not
		// a pointer), matching the plan's required struct literal
		// `Usage: result.Usage,` and tag `json:"usage,omitempty"`. Go's
		// encoding/json "omitempty" never treats a struct-typed field as
		// empty -- that rule applies only to pointers, slices, maps, and
		// scalar zero values -- so a zero-value Usage still produces a
		// "usage" key, but every field *inside* it is individually
		// omitempty-tagged in pkg/codex/usage.go, so it marshals as an empty
		// object rather than a populated one. tool_count's own omitempty
		// works because ToolCount is a plain int, not a struct: the two
		// fields are not directly comparable despite sharing the tag name.
		result := codex.WorkerResult{
			WorkerName: "Builder-Test",
			Caste:      "builder",
			Status:     "completed",
		}
		mapped := mapInternalWorkerResult(result, nil)

		raw, err := json.Marshal(mapped)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("unmarshal into map: %v", err)
		}
		if usageRaw, present := fields["usage"]; present && string(usageRaw) != "{}" {
			t.Errorf("usage = %s, want an empty object for a zero-value Usage", usageRaw)
		}
		if _, present := fields["tool_count"]; present {
			t.Errorf("marshalled JSON contains a %q key for a zero-value ToolCount; want omitted", "tool_count")
		}

		var roundTripped internalWorkerResult
		if err := json.Unmarshal(raw, &roundTripped); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !roundTripped.Usage.Empty() {
			t.Errorf("Usage = %+v, want Empty() after round-tripping a zero-value Usage", roundTripped.Usage)
		}
	})
}
