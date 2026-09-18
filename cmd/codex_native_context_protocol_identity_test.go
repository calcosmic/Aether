package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the real aggregate admission boundary with independently retained
// original and derived files. Valid protocol tuples deliberately stop at the
// later missing-artifact gate; these small fixtures never qualify live work.
func TestCodexNativeContextProtocolIdentity(t *testing.T) {
	known := codexNativeContextProtocolChildFetch
	cases := []struct {
		name, aggregate, original, derived string
		valid                              bool
	}{
		{"legacy", "", "", "", true},
		{"child-fetch", known, known, known, true},
		{"missing-aggregate", "", known, known, false},
		{"missing-original", known, "", known, false},
		{"missing-derived", known, known, "", false},
		{"unknown-aggregate", "child-fetch/v99", known, known, false},
		{"unknown-original", known, "child-fetch/v99", known, false},
		{"unknown-derived", known, known, "child-fetch/v99", false},
		{"all-unknown", "child-fetch/v99", "child-fetch/v99", "child-fetch/v99", false},
		{"legacy-aggregate-upgrade", known, "", "", false},
		{"legacy-derived-upgrade", "", "", known, false},
		{"noncanonical-protocol", known + " ", known, known, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			originalPath, derivedPath := filepath.Join(dir, "original.json"), filepath.Join(dir, "derived.json")
			original := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", Scenario: "ordinary", ContextProtocol: test.original}
			liveSkillWriteJSON(t, originalPath, original)
			derived := original
			derived.ContextProtocol = test.derived
			derived.ValidationOriginalReceipt = originalPath
			derived.ValidationOriginalSHA256 = liveSkillFileDigest(t, originalPath)
			liveSkillWriteJSON(t, derivedPath, derived)
			candidate := nativeEvidenceCase{ContextProtocol: test.aggregate, OriginalReceipt: originalPath, OriginalReceiptSHA256: liveSkillFileDigest(t, originalPath), DerivedReceipt: derivedPath, DerivedReceiptSHA256: liveSkillFileDigest(t, derivedPath)}
			// The aggregate field must survive its actual JSON boundary.
			raw, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			var decoded nativeEvidenceCase
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			err = nativeEvidenceCaseCheck(t, "ordinary", decoded, "", "", "", nil)
			if test.valid {
				if err == nil || !strings.Contains(err.Error(), "aggregate artifact accounting differs") {
					t.Fatalf("valid protocol failed before the independent artifact gate: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "context protocol") {
				t.Fatalf("protocol mutation reached later evidence admission: %v", err)
			}
		})
	}
}

func TestCodexNativeContextProtocolCaseWireCompatibility(t *testing.T) {
	legacy, err := json.Marshal(nativeEvidenceCase{})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy, []byte(`"context_protocol"`)) {
		t.Fatal("legacy aggregate acquired an implicit transport upgrade")
	}
	var qualification nativeEvidenceQualification
	if err := json.Unmarshal([]byte(`{"cases":{"claude":{},"ordinary":{"context_protocol":"child-fetch/v1"}}}`), &qualification); err != nil {
		t.Fatal(err)
	}
	if qualification.Cases["claude"].ContextProtocol != "" || qualification.Cases["ordinary"].ContextProtocol != codexNativeContextProtocolChildFetch {
		t.Fatal("aggregate collapsed independent per-case transport identities")
	}
	if err := nativeEvidenceContextProtocolBinding(qualification.Cases["claude"].ContextProtocol, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := nativeEvidenceContextProtocolBinding(qualification.Cases["ordinary"].ContextProtocol, codexNativeContextProtocolChildFetch, codexNativeContextProtocolChildFetch); err != nil {
		t.Fatal(err)
	}
}
