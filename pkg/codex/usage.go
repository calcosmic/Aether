package codex

import (
	"encoding/json"
	"strings"
)

// WorkerUsage is what a worker run actually cost.
//
// Nothing measured this before. Every budget in the codebase counts characters
// of assembled context, which is ~6k tokens against the ~117k a real worker
// spends — so the only thing being measured was the 5% the framework composes,
// and the 95% it does not was invisible. A framework that cannot see where its
// tokens go cannot be improved deliberately; every claim about efficiency was
// unfalsifiable.
//
// No tokenizer is involved. Both providers already report exact counts and the
// raw output is already captured verbatim; this only reads what was always
// there.
type WorkerUsage struct {
	InputTokens         int64   `json:"input_tokens,omitempty"`
	CachedInputTokens   int64   `json:"cached_input_tokens,omitempty"`
	CacheCreationTokens int64   `json:"cache_creation_tokens,omitempty"`
	OutputTokens        int64   `json:"output_tokens,omitempty"`
	TotalTokens         int64   `json:"total_tokens,omitempty"`
	USDCost             float64 `json:"usd_cost,omitempty"`
	Model               string  `json:"model,omitempty"`

	// Source distinguishes a provider-reported measurement from a local
	// estimate, or from a platform-harness measurement that is real but not
	// provider-grade. An estimate must never be presentable as a measurement:
	// the whole point of this ledger is that the numbers can be trusted, and a
	// silently-estimated row would make a regression look like an improvement.
	// Values: "provider", "session-transcript", "estimate".
	Source string `json:"source,omitempty"`
}

// Measured reports whether the usage is provider-grade: parsed directly from
// a raw provider API event by ParseUsage. Nothing outside ParseUsage may ever
// set UsageSourceProvider.
//
// A UsageSourceSessionTranscript row is a genuine measurement — the platform
// harness itself wrote it, not the model narrating its own behavior — and it
// still returns false here, because its accounting semantics are undocumented
// by the vendor (see 174-RESEARCH.md Assumptions A1/A2) and conflating it with
// a fully-verified provider event would erode the exact trust boundary this
// type exists to hold. The ledger's measured/estimated split therefore uses
// Estimated(), not Measured(), to decide what counts as a guess.
func (u WorkerUsage) Measured() bool { return u.Source == UsageSourceProvider }

// Estimated reports whether the usage is a local character-count guess rather
// than a real measurement of any kind (provider or session-transcript).
func (u WorkerUsage) Estimated() bool { return u.Source == UsageSourceEstimate }

// Empty reports whether nothing at all was recorded.
func (u WorkerUsage) Empty() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 && u.TotalTokens == 0 && u.Source == ""
}

const (
	UsageSourceProvider = "provider"

	// UsageSourceSessionTranscript tags a usage row read from a platform-
	// harness-written session artifact — a Claude Code `.jsonl` transcript or
	// an OpenCode session store — rather than parsed from a raw provider API
	// event. It is written by the Go runtime reading that artifact itself,
	// never by the orchestrating LLM relaying a number it read. It is not
	// "provider" because its accounting semantics (whether the harness sums
	// cache tokens the same way billedTotal does, whether the format is
	// stable across CLI versions) are undocumented by both vendors — see
	// 174-RESEARCH.md Assumptions A1/A2. Nothing outside ParseUsage may ever
	// set UsageSourceProvider; this tier exists precisely so a real
	// measurement never has to borrow that constant to be taken seriously.
	UsageSourceSessionTranscript = "session-transcript"

	UsageSourceEstimate = "estimate"
)

// estimateTokensPerChar is the fallback ratio when no provider usage is found.
// It is deliberately crude: an estimate exists so a dispatch never vanishes
// from the ledger entirely, not so it can stand in for a measurement. Rows
// carrying it are tagged UsageSourceEstimate and must be reported as such.
const estimateCharsPerToken = 4

// EstimateUsage produces a clearly-labelled fallback for a prompt whose worker
// reported nothing. A missing row would silently shrink the measured total and
// make a run look cheaper than it was.
func EstimateUsage(promptChars int) WorkerUsage {
	if promptChars <= 0 {
		return WorkerUsage{Source: UsageSourceEstimate}
	}
	tokens := int64(promptChars / estimateCharsPerToken)
	return WorkerUsage{
		InputTokens: tokens,
		TotalTokens: tokens,
		Source:      UsageSourceEstimate,
	}
}

// ParseUsage extracts provider-reported token usage from a worker's raw stdout.
//
// Codex (`codex exec --json`) emits NDJSON `token_count` events carrying a
// cumulative total; the last one wins. Claude (`--output-format stream-json`)
// emits a terminal `{"type":"result", "usage":{...}, "total_cost_usd":...}`.
// Both shapes are scanned regardless of the declared platform, because the
// dispatch layer can fall back between them and a usage row attributed to the
// wrong parser is worse than one parsed by shape.
func ParseUsage(rawOutput string) (WorkerUsage, bool) {
	if strings.TrimSpace(rawOutput) == "" {
		return WorkerUsage{}, false
	}

	var found bool
	var usage WorkerUsage

	for _, line := range strings.Split(rawOutput, "\n") {
		line = strings.TrimSpace(stripANSIEscapeCodes(line))
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if parsed, ok := usageFromEvent(event); ok {
			// Later events supersede earlier ones: both providers report
			// cumulative totals, so the last statement is the true one.
			usage = parsed
			found = true
		}
	}

	if !found {
		return WorkerUsage{}, false
	}
	usage.Source = UsageSourceProvider
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.billedTotal()
	}
	return usage, true
}

// usageFromEvent recognises the two provider shapes in one place so a new
// transport only has to be added here.
func usageFromEvent(event map[string]interface{}) (WorkerUsage, bool) {
	// Claude: terminal result event.
	if eventType, _ := event["type"].(string); eventType == "result" {
		usage, ok := usageFromNestedMap(event, "usage")
		if !ok {
			return WorkerUsage{}, false
		}
		if cost, ok := numberValue(event["total_cost_usd"]); ok {
			usage.USDCost = cost
		}
		if model, _ := event["model"].(string); model != "" {
			usage.Model = model
		}
		return usage, true
	}

	// Codex: token_count events, either flat or nested under
	// total_token_usage / info.total_token_usage.
	if eventType, _ := event["type"].(string); eventType == "token_count" {
		for _, key := range []string{"total_token_usage", "usage", "info"} {
			if usage, ok := usageFromNestedMap(event, key); ok {
				return usage, true
			}
		}
		if usage, ok := usageFromFields(event); ok {
			return usage, true
		}
	}

	return WorkerUsage{}, false
}

func usageFromNestedMap(event map[string]interface{}, key string) (WorkerUsage, bool) {
	nested, ok := event[key].(map[string]interface{})
	if !ok {
		return WorkerUsage{}, false
	}
	if usage, ok := usageFromFields(nested); ok {
		return usage, true
	}
	// Codex nests total_token_usage one level deeper inside info.
	for _, inner := range []string{"total_token_usage", "usage"} {
		if deeper, ok := nested[inner].(map[string]interface{}); ok {
			if usage, ok := usageFromFields(deeper); ok {
				return usage, true
			}
		}
	}
	return WorkerUsage{}, false
}

// usageFromFields reads the token fields under either provider's naming.
func usageFromFields(fields map[string]interface{}) (WorkerUsage, bool) {
	var usage WorkerUsage
	var any bool

	for _, key := range []string{"input_tokens", "prompt_tokens"} {
		if v, ok := numberValue(fields[key]); ok {
			usage.InputTokens = int64(v)
			any = true
			break
		}
	}
	for _, key := range []string{"cached_input_tokens", "cache_read_input_tokens"} {
		if v, ok := numberValue(fields[key]); ok {
			usage.CachedInputTokens = int64(v)
			any = true
			break
		}
	}
	for _, key := range []string{"cache_creation_input_tokens", "cache_creation_tokens"} {
		if v, ok := numberValue(fields[key]); ok {
			usage.CacheCreationTokens = int64(v)
			any = true
			break
		}
	}
	for _, key := range []string{"output_tokens", "completion_tokens"} {
		if v, ok := numberValue(fields[key]); ok {
			usage.OutputTokens = int64(v)
			any = true
			break
		}
	}
	for _, key := range []string{"total_tokens", "total_token_count"} {
		if v, ok := numberValue(fields[key]); ok {
			usage.TotalTokens = int64(v)
			any = true
			break
		}
	}

	return usage, any
}

func numberValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}

// billedTotal sums the token counts a provider actually bills for.
//
// Anthropic reports input, cache_read and cache_creation as DISJOINT counts —
// total = input + cache_read + cache_creation. The first version of this file
// derived the total as input + output, ignoring both cache figures, and never
// parsed cache_creation at all. On a realistic run (50 input, 100,000 cache
// read, 500 output) that reported 550 against a true 102,550: a 186x
// undercount.
//
// The failure was not caught because the unit test asserted the same wrong
// arithmetic — total == input + output — so it passed while enshrining the bug.
// The cost figure was right throughout, which would have produced a dashboard
// showing a correct price beside a token count 186x too small, and the token
// count is the number anyone divides by.
func (u WorkerUsage) billedTotal() int64 {
	return u.InputTokens + u.CachedInputTokens + u.CacheCreationTokens + u.OutputTokens
}

// TotalInputTokens is the figure the spend report displays beside the four
// disjoint columns: the three input-side counts summed, excluding output.
// For Anthropic's documented example (50 input, 100,000 cache read, 2,000
// cache creation, 500 output) this returns 102,050 (ROADMAP Phase 174
// criterion 1).
func (u WorkerUsage) TotalInputTokens() int64 {
	return u.InputTokens + u.CachedInputTokens + u.CacheCreationTokens
}

// BilledTotalTokens is the single exported entry point every consumer in this
// phase must use for a row's token count. No caller anywhere may re-derive a
// total by adding fields itself — that is Pitfall 1 (the 186x undercount).
// The trace package's own per-call cost estimator once re-derived a total the
// same way; it was deleted as dead code in Phase 191 (its only caller was
// unreachable in production), not because the shape stopped being a pitfall —
// a future caller can still reintroduce it.
//
// It returns TotalTokens when the provider reported one, otherwise falls back
// to billedTotal(). The TotalTokens preference exists because Claude Code's
// transcript reports one aggregate subagent_tokens figure with no disjoint
// breakdown: a provider-reported aggregate with no disjoint breakdown is
// preferred over a zero sum.
func (u WorkerUsage) BilledTotalTokens() int64 {
	if u.TotalTokens > 0 {
		return u.TotalTokens
	}
	return u.billedTotal()
}
