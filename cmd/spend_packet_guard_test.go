package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// D-06 (.planning/phases/174-spend-ledger/174-CONTEXT.md): a token figure
// relayed by the orchestrating LLM through the completion packet is an
// assertion, not a measurement. codexExternalBuildWorkerResult deliberately
// declares no "usage" field, so the packet has no place to put one -- the
// wrapper-path figure is instead read from the session transcript by plan
// 174-04 (cmd/spend_session_capture.go's loadSpendSessionRecord). If a
// future change adds a usage field to codexExternalBuildWorkerResult, THIS
// test is the thing that must be argued with first.
//
// Built as raw JSON bytes rather than a codexExternalBuildCompletion struct
// literal on purpose -- the point is submitting a key the Go struct does not
// declare, mirroring the misspelled-field-name proof in
// cmd/completion_packet_submitted_bytes_test.go.
func TestCompletionPacketRefusesAssertedWorkerUsage(t *testing.T) {
	manifest := minimalValidDispatchManifestForSpendGuardTest()
	packet := map[string]any{
		"dispatch_manifest": manifest,
		"dispatches": []any{
			map[string]any{
				"name":   "Builder-1",
				"status": "success",
				"usage": map[string]any{
					"total_tokens": 999999,
					"source":       "provider",
				},
			},
		},
	}
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("marshal completion packet fixture: %v", err)
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode completion packet fixture: %v", err)
	}

	violations := validateCompletionPacketStructure(raw)
	found := false
	for _, v := range violations {
		if v.Rule == "additionalProperties" && strings.Contains(v.Message, "usage") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an additionalProperties violation naming usage, got %+v", violations)
	}
}

// TestCompletionPacketSchemaHasNoUsageProperty is the sibling proof: the
// reflected schema itself has no property named "usage" anywhere in the
// document (worker result, manifest, or any nested $def), so a wrapper
// cannot even discover a place to put an asserted token figure. See D-06
// above.
func TestCompletionPacketSchemaHasNoUsageProperty(t *testing.T) {
	schemaBytes, err := generateCompletionPacketSchemaBytes()
	if err != nil {
		t.Fatalf("generate completion-packet schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode generated completion-packet schema: %v", err)
	}
	if _, ok := schema["properties"]; !ok && len(schema) == 0 {
		t.Fatalf("generated completion-packet schema decoded empty")
	}

	// A literal key search across the whole marshaled document is
	// sufficient and robust: every property name in an invopop/jsonschema
	// document appears as a JSON object key exactly as declared, so
	// "usage" cannot exist as a schema property without the literal text
	// `"usage"` appearing in the bytes.
	if bytes.Contains(schemaBytes, []byte(`"usage"`)) {
		t.Fatalf("generated completion-packet schema declares a usage property -- D-06 requires the packet have no place to put an asserted token figure")
	}
}

// minimalValidDispatchManifestForSpendGuardTest returns the smallest map
// that satisfies codexBuildManifest's required properties (schema section
// codexBuildManifest.required), so the fixture packet's structural
// validation failure is attributable ONLY to the spliced usage field, not to
// an incidentally-missing manifest field.
func minimalValidDispatchManifestForSpendGuardTest() map[string]any {
	return map[string]any{
		"phase":            1,
		"phase_name":       "spend-ledger",
		"root":             "/tmp/spend-guard-test",
		"colony_depth":     "standard",
		"generated_at":     "2026-08-14T00:00:00Z",
		"state":            "EXECUTING",
		"checkpoint":       "build",
		"claims_path":      "claims.json",
		"worker_briefs":    []string{},
		"dispatches":       []any{},
		"tasks":            []any{},
		"success_criteria": []string{},
	}
}
