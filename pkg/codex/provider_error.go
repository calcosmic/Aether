package codex

import (
	"encoding/json"
	"fmt"
	"strings"
)

// providerErrorEnvelope is a structured provider/API failure found in worker
// output. Providers report HTTP-level failures as well-formed JSON events on
// stdout with exit code 0 — e.g. opencode emits
// {"type":"error","error":{"name":"APIError","data":{"statusCode":404,...}}}.
// Before this detector existed, that envelope fell through to the claims
// parser and was reported as "parse worker output: no JSON found in output":
// both halves false, and the operator was sent to the wrong subsystem — four
// opencode workers failed exactly this way on 2026-05-05 and the M4L
// investigation of 2026-07-27 burned most of its time on the mislabel.
type providerErrorEnvelope struct {
	StatusCode int
	Name       string
	Message    string
	URL        string
}

func (e providerErrorEnvelope) classify() string {
	switch {
	case e.StatusCode == 401 || e.StatusCode == 403:
		return "provider auth failure"
	case e.StatusCode == 404:
		return "provider endpoint not found"
	case e.StatusCode == 429:
		return "provider rate limited"
	case e.StatusCode >= 500:
		return "provider unavailable"
	default:
		return "provider error"
	}
}

// Error renders the envelope as an operator-facing failure that names the
// provider layer, the status code, and the endpoint.
func (e providerErrorEnvelope) Error() string {
	var b strings.Builder
	b.WriteString(e.classify())
	b.WriteString(fmt.Sprintf(" (HTTP %d", e.StatusCode))
	if e.URL != "" {
		b.WriteString(" from " + e.URL)
	}
	b.WriteString(")")
	if e.Message != "" {
		b.WriteString(": " + firstLine(e.Message))
	}
	return b.String()
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		value = value[:idx]
	}
	const limit = 200
	if len(value) > limit {
		value = value[:limit] + "..."
	}
	return value
}

// detectProviderErrorEnvelope scans worker output (whole blob, then per line
// for JSONL streams) for a structured provider error event. Detection is
// structural — an object with "type":"error" or an "error" member carrying a
// numeric statusCode >= 400 — so a successful claims payload that merely
// mentions the word "error" cannot false-positive.
func detectProviderErrorEnvelope(output string) (providerErrorEnvelope, bool) {
	trimmed := strings.TrimSpace(stripANSIEscapeCodes(output))
	if trimmed == "" {
		return providerErrorEnvelope{}, false
	}
	if env, ok := providerErrorFromJSON(trimmed); ok {
		return env, true
	}
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		if env, ok := providerErrorFromJSON(line); ok {
			return env, true
		}
	}
	return providerErrorEnvelope{}, false
}

func providerErrorFromJSON(text string) (providerErrorEnvelope, bool) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(text), &event); err != nil {
		return providerErrorEnvelope{}, false
	}
	// A payload that parses as worker claims is never a provider error, even
	// if it happens to carry an "error" member.
	if isWorkerClaimsMap(event) {
		return providerErrorEnvelope{}, false
	}
	errObj, ok := event["error"].(map[string]interface{})
	if !ok {
		return providerErrorEnvelope{}, false
	}
	if typ, _ := event["type"].(string); typ != "error" && typ != "" {
		return providerErrorEnvelope{}, false
	}

	env := providerErrorEnvelope{}
	env.Name, _ = errObj["name"].(string)
	env.StatusCode = intField(errObj, "statusCode")
	env.Message, _ = errObj["message"].(string)
	if data, ok := errObj["data"].(map[string]interface{}); ok {
		if env.StatusCode == 0 {
			env.StatusCode = intField(data, "statusCode")
		}
		if env.Message == "" {
			env.Message, _ = data["message"].(string)
		}
		if meta, ok := data["metadata"].(map[string]interface{}); ok {
			env.URL, _ = meta["url"].(string)
		}
	}
	if env.StatusCode < 400 {
		return providerErrorEnvelope{}, false
	}
	return env, true
}

func intField(obj map[string]interface{}, key string) int {
	if value, ok := obj[key].(float64); ok {
		return int(value)
	}
	return 0
}
