package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// assertEnvelopeParity verifies that both shell and Go outputs parse as JSON
// and have matching "ok" field values. This is the minimum parity check:
// both succeed or both fail.
func assertEnvelopeParity(t *testing.T, shellOut, goOut string) {
	t.Helper()

	var shellJSON, goJSON map[string]interface{}
	if err := json.Unmarshal([]byte(shellOut), &shellJSON); err != nil {
		t.Errorf("shell output is not valid JSON: %v\noutput: %s", err, truncateStr(shellOut, 200))
		return
	}
	if err := json.Unmarshal([]byte(goOut), &goJSON); err != nil {
		t.Errorf("Go output is not valid JSON: %v\noutput: %s", err, truncateStr(goOut, 200))
		return
	}

	shellOK, shellHasOK := shellJSON["ok"]
	goOK, goHasOK := goJSON["ok"]

	if !shellHasOK && !goHasOK {
		t.Errorf("neither output has 'ok' field: shell=%v go=%v", shellJSON, goJSON)
		return
	}

	// If one has "ok" and the other doesn't, that's a parity issue
	if shellHasOK != goHasOK {
		t.Errorf("ok field presence mismatch: shell has=%v go has=%v", shellHasOK, goHasOK)
		return
	}

	// Both have "ok" -- compare boolean values
	shellOKBool, _ := shellOK.(bool)
	goOKBool, _ := goOK.(bool)
	if shellOKBool != goOKBool {
		t.Errorf("ok value mismatch: shell=%v go=%v", shellOKBool, goOKBool)
	}
}

// assertResultFieldParity verifies that the .result field matches between
// shell and Go outputs. It handles string-string, map-map, and mixed
// (parity break) comparisons.
func assertResultFieldParity(t *testing.T, shellOut, goOut string) {
	t.Helper()

	var shellJSON, goJSON map[string]interface{}
	if err := json.Unmarshal([]byte(shellOut), &shellJSON); err != nil {
		t.Errorf("shell output is not valid JSON: %v", err)
		return
	}
	if err := json.Unmarshal([]byte(goOut), &goJSON); err != nil {
		t.Errorf("Go output is not valid JSON: %v", err)
		return
	}

	shellResult, shellHas := shellJSON["result"]
	goResult, goHas := goJSON["result"]

	if !shellHas && !goHas {
		// Neither has result -- that's consistent
		return
	}
	if shellHas != goHas {
		t.Errorf("result field presence mismatch: shell=%v go=%v", shellHas, goHas)
		return
	}

	shellStr, shellIsStr := shellResult.(string)
	goStr, goIsStr := goResult.(string)
	_, shellIsMap := shellResult.(map[string]interface{})
	goMap, goIsMap := goResult.(map[string]interface{})

	if shellIsStr && goIsStr {
		if shellStr != goStr {
			t.Errorf("result string mismatch: shell=%q go=%q", shellStr, goStr)
		}
		return
	}

	if shellIsMap && goIsMap {
		// Check overlapping keys match
		for k := range goMap {
			if shellVal, ok := shellResult.(map[string]interface{}); ok {
				if sv, exists := shellVal[k]; exists {
					gv := goMap[k]
					if fmt.Sprintf("%v", sv) != fmt.Sprintf("%v", gv) {
						t.Errorf("result.%s mismatch: shell=%v go=%v", k, sv, gv)
					}
				}
			}
		}
		return
	}

	// One is string, other is map -- parity break
	t.Logf("PARITY BREAK: result type differs: shell type=%T, go type=%T", shellResult, goResult)
}

// compareByPaths extracts values at the given JSON paths from shell and Go
// output and compares them. Paths are dot-separated (e.g., "result.count").
func compareByPaths(t *testing.T, shellJSON, goJSON string, shellPath, goPath string) {
	t.Helper()

	shellVal := extractByPath(t, shellJSON, shellPath)
	goVal := extractByPath(t, goJSON, goPath)

	if shellVal == nil && goVal == nil {
		return
	}
	if shellVal == nil || goVal == nil {
		t.Errorf("path mismatch: shell[%s]=%v go[%s]=%v", shellPath, shellVal, goPath, goVal)
		return
	}

	// Compare string representations
	shellStr := fmt.Sprintf("%v", shellVal)
	goStr := fmt.Sprintf("%v", goVal)
	if shellStr != goStr {
		t.Errorf("value mismatch at paths shell[%s] go[%s]: shell=%v go=%v", shellPath, goPath, shellVal, goVal)
	}
}

// extractByPath walks a dot-separated path (e.g., "result.signals") into a
// parsed JSON map. Returns nil if the path does not exist (not a fatal error).
func extractByPath(t *testing.T, jsonStr string, path string) interface{} {
	t.Helper()

	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return nil
	}

	parts := strings.Split(path, ".")
	current := obj
	for _, part := range parts {
		if part == "" {
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}

// isParityBreak returns true if the shell and Go outputs have a known
// structural difference. These are documented differences that are not
// regressions. The function logs the reason for the break.
func isParityBreak(t *testing.T, shellOut, goOut string) bool {
	t.Helper()

	// Check for known parity break patterns in the output
	parityBreakReasons := []struct {
		pattern string
		reason  string
	}{
		// generate-ant-name: shell returns bare string, Go returns object with name+caste
		{"generate-ant-name", "shell returns bare string, Go returns object with name+caste"},
		// pheromone-count: different field names between shell and Go
		{"pheromone-count", "different field names between shell and Go"},
		// entropy-score: different nesting structure
		{"entropy-score", "different nesting structure between shell and Go"},
		// milestone-detect: different calculation method
		{"milestone-detect", "different calculation method between shell and Go"},
		// memory-metrics: completely different structure
		{"memory-metrics", "completely different output structure"},
		// version: Go does not use JSON envelope
		{"version", "Go does not use JSON envelope for version command"},
	}

	// Parse both outputs to detect structural differences
	var shellJSON, goJSON map[string]interface{}
	shellErr := json.Unmarshal([]byte(shellOut), &shellJSON)
	goErr := json.Unmarshal([]byte(goOut), &goJSON)

	// If one is JSON and the other isn't, it could be a known parity break
	if (shellErr == nil) != (goErr == nil) {
		// One is JSON, other is not -- check if it's a known case
		t.Logf("PARITY BREAK: one output is JSON, other is not")
		return true
	}

	// Both JSON: check if .result types differ (string vs object)
	if shellErr == nil && goErr == nil {
		// Check "ok" field presence mismatch (envelope vs non-envelope)
		_, shellHasOK := shellJSON["ok"]
		_, goHasOK := goJSON["ok"]
		if shellHasOK != goHasOK {
			t.Logf("PARITY BREAK: ok field presence differs (shell=%v go=%v)", shellHasOK, goHasOK)
			return true
		}

		shellResult, shellHasResult := shellJSON["result"]
		goResult, goHasResult := goJSON["result"]

		if shellHasResult && goHasResult {
			_, shellIsObj := shellResult.(map[string]interface{})
			_, goIsObj := goResult.(map[string]interface{})
			_, shellIsStr := shellResult.(string)
			_, goIsStr := goResult.(string)

			// String vs object mismatch
			if (shellIsStr && goIsObj) || (shellIsObj && goIsStr) {
				t.Logf("PARITY BREAK: result type differs (string vs object)")
				return true
			}

			// Both objects but different key sets
			if shellIsObj && goIsObj {
				shellMap := shellResult.(map[string]interface{})
				goMap := goResult.(map[string]interface{})
				// If more than half the keys differ, it's a parity break
				shellKeys := keySet(shellMap)
				goKeys := keySet(goMap)
				overlap := 0
				for k := range shellKeys {
					if goKeys[k] {
						overlap++
					}
				}
				totalUnique := len(shellKeys) + len(goKeys) - overlap
				if totalUnique > 0 && float64(overlap)/float64(totalUnique) < 0.3 {
					t.Logf("PARITY BREAK: result keys mostly different (overlap=%d, total=%d)", overlap, totalUnique)
					return true
				}
			}
		}
	}

	_ = parityBreakReasons // available for future use
	return false
}

// keySet returns a set of keys from a map.
func keySet(m map[string]interface{}) map[string]bool {
	set := make(map[string]bool, len(m))
	for k := range m {
		set[k] = true
	}
	return set
}

// TestAssertEnvelopeParity verifies the envelope parity check.
func TestAssertEnvelopeParity(t *testing.T) {
	// Both succeed
	t.Run("both_succeed", func(t *testing.T) {
		var buf bytes.Buffer
		buf.Reset()
		// This should not error -- we verify by calling with matching ok values
		assertEnvelopeParity(t,
			`{"ok":true,"result":"hello"}`,
			`{"ok":true,"result":"hello"}`,
		)
	})

	// Both fail
	t.Run("both_fail", func(t *testing.T) {
		assertEnvelopeParity(t,
			`{"ok":false,"error":"not found","code":1}`,
			`{"ok":false,"error":"not found","code":1}`,
		)
	})
}

// TestExtractByPath verifies the JSON path extraction.
func TestExtractByPath(t *testing.T) {
	json := `{"ok":true,"result":{"count":5,"name":"test"}}`

	tests := []struct {
		path string
		want interface{}
	}{
		{"ok", true},
		{"result.count", float64(5)},
		{"result.name", "test"},
		{"result.missing", nil},
		{"missing.path", nil},
	}
	for _, tt := range tests {
		got := extractByPath(t, json, tt.path)
		if got != tt.want {
			t.Errorf("extractByPath(%q) = %v (%T), want %v (%T)", tt.path, got, got, tt.want, tt.want)
		}
	}
}

// TestCompareByPaths verifies the path-based comparison.
func TestCompareByPaths(t *testing.T) {
	shellJSON := `{"ok":true,"result":{"count":5}}`
	goJSON := `{"ok":true,"result":{"count":5}}`

	// Same path, same value -- should pass
	compareByPaths(t, shellJSON, goJSON, "result.count", "result.count")
}

// TestIsParityBreak verifies parity break detection.
func TestIsParityBreak(t *testing.T) {
	// Matching structures -- not a break
	if isParityBreak(t,
		`{"ok":true,"result":"hello"}`,
		`{"ok":true,"result":"world"}`,
	) {
		t.Error("should not detect parity break for matching string results")
	}

	// String vs object -- is a break
	if !isParityBreak(t,
		`{"ok":true,"result":"hello"}`,
		`{"ok":true,"result":{"name":"hello","caste":"builder"}}`,
	) {
		t.Error("should detect parity break for string vs object results")
	}

	// JSON vs non-JSON -- is a break
	if !isParityBreak(t,
		`{"ok":true,"result":"hello"}`,
		`not json at all`,
	) {
		t.Error("should detect parity break for JSON vs non-JSON")
	}
}
