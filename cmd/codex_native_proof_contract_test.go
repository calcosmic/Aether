package cmd

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCodexNativeProofContractReplayIdentity(t *testing.T) {
	for _, mutation := range []string{"unknown-contract", "stale-amendment", "missing-contract", "missing-amendment"} {
		t.Run(mutation, func(t *testing.T) {
			r := codexNativeLiveReceipt{ProofContract: nativeCapabilityProofContract, ProofAmendmentSHA256: nativeCapabilityProofAmendmentSHA256, Outcome: "incomplete", Reason: "original capture"}
			switch mutation {
			case "unknown-contract":
				r.ProofContract = "unknown/v1"
			case "stale-amendment":
				r.ProofAmendmentSHA256 = "sha256:stale"
			case "missing-contract":
				r.ProofContract = ""
			case "missing-amendment":
				r.ProofAmendmentSHA256 = ""
			}
			before := r
			if err := nativeReplayQualificationReceipt(t, &r); err == nil || !strings.Contains(err.Error(), "prospective proof contract") {
				t.Fatalf("standalone replay accepted invalid contract before artifact access: %v", err)
			}
			if !reflect.DeepEqual(r, before) {
				t.Fatal("invalid contract changed replay state or derived outcome")
			}
		})
	}
}

func TestCodexNativeProofContractBinding(t *testing.T) {
	for _, boundary := range []string{"derived", "aggregate"} {
		for _, originalProspective := range []bool{false, true} {
			for _, mutation := range []string{"unchanged", "other-contract", "unknown-contract", "stale-amendment", "missing-contract", "missing-amendment"} {
				t.Run(boundary+"/"+map[bool]string{true: "prospective", false: "legacy"}[originalProspective]+"/"+mutation, func(t *testing.T) {
					original := codexNativeLiveReceipt{}
					if originalProspective {
						original.ProofContract, original.ProofAmendmentSHA256 = nativeCapabilityProofContract, nativeCapabilityProofAmendmentSHA256
					}
					candidate := original
					switch mutation {
					case "other-contract":
						candidate.ProofContract, candidate.ProofAmendmentSHA256 = nativeCapabilityProofContract, nativeCapabilityProofAmendmentSHA256
						if originalProspective {
							candidate.ProofContract, candidate.ProofAmendmentSHA256 = "", ""
						}
					case "unknown-contract":
						candidate.ProofContract = "unknown/v1"
					case "stale-amendment":
						candidate.ProofAmendmentSHA256 = "sha256:stale"
					case "missing-contract":
						candidate.ProofContract, candidate.ProofAmendmentSHA256 = "", nativeCapabilityProofAmendmentSHA256
					case "missing-amendment":
						candidate.ProofContract, candidate.ProofAmendmentSHA256 = nativeCapabilityProofContract, ""
					}
					contract, amendment := candidate.ProofContract, candidate.ProofAmendmentSHA256
					if boundary == "aggregate" {
						q := nativeEvidenceQualification{ProofContract: contract, ProofAmendmentSHA256: amendment}
						contract, amendment = q.ProofContract, q.ProofAmendmentSHA256
					}
					err := nativeGapProofBinding(contract, amendment, original.ProofContract, original.ProofAmendmentSHA256)
					if (err == nil) != (mutation == "unchanged") {
						t.Fatalf("%s binding mutation %s: %v", boundary, mutation, err)
					}
				})
			}
		}
	}
	if err := nativeGapProofBinding("unknown/v1", "sha256:unknown", "unknown/v1", "sha256:unknown"); err == nil {
		t.Fatal("identically invalid proof identities became an accepted contract")
	}
}

func TestCodexNativeProofContractDerivedReceiptIdentity(t *testing.T) {
	for _, originalProspective := range []bool{false, true} {
		t.Run(map[bool]string{true: "cannot-strip", false: "cannot-upgrade"}[originalProspective], func(t *testing.T) {
			original := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", Scenario: "ordinary"}
			derived := original
			if originalProspective {
				original.ProofContract, original.ProofAmendmentSHA256 = nativeCapabilityProofContract, nativeCapabilityProofAmendmentSHA256
			} else {
				derived.ProofContract, derived.ProofAmendmentSHA256 = nativeCapabilityProofContract, nativeCapabilityProofAmendmentSHA256
			}
			dir := t.TempDir()
			originalPath, derivedPath := filepath.Join(dir, "original.json"), filepath.Join(dir, "derived.json")
			liveSkillWriteJSON(t, originalPath, original)
			liveSkillWriteJSON(t, derivedPath, derived)
			c := nativeEvidenceCase{OriginalReceipt: originalPath, OriginalReceiptSHA256: liveSkillFileDigest(t, originalPath), DerivedReceipt: derivedPath, DerivedReceiptSHA256: liveSkillFileDigest(t, derivedPath)}
			err := nativeEvidenceCaseCheck(t, "ordinary", c, "", "", "", nil)
			if err == nil || !strings.Contains(err.Error(), "derived receipt changed the original proof contract") {
				t.Fatalf("case acceptance did not reject independent derived contract mutation: %v", err)
			}
		})
	}
}

func nativeClaudeStartupFixture(t *testing.T, events ...map[string]any) []byte {
	t.Helper()
	var raw []byte
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, append(line, '\n')...)
	}
	return raw
}

func TestCodexNativeClaudeStartupIdentity(t *testing.T) {
	for _, mutation := range []string{"none", "child-init", "other-model-fields", "missing-init", "child-init-only", "missing-model", "missing-session", "wrong-parent-session", "duplicate-init", "different-init", "wrong-event-type", "malformed", "untyped"} {
		t.Run(mutation, func(t *testing.T) {
			startup := map[string]any{"type": "system", "subtype": "init", "session_id": "parent-session", "model": "reported-parent-model", "parent_tool_use_id": nil}
			parent := map[string]any{"type": "assistant", "session_id": "parent-session", "parent_tool_use_id": nil}
			child := map[string]any{"type": "system", "subtype": "init", "session_id": "child-session", "model": "reported-child-model", "parent_tool_use_id": "actual-helper"}
			events := []map[string]any{startup, parent}
			switch mutation {
			case "child-init":
				events = append(events, child)
			case "other-model-fields":
				parent["model"] = "arbitrary-parent-field"
				events = append(events, map[string]any{"type": "result", "session_id": "parent-session", "model": "arbitrary-result-field"})
			case "missing-init":
				parent["model"] = "arbitrary-parent-field"
				events = []map[string]any{parent}
			case "child-init-only":
				events = []map[string]any{child, parent}
			case "missing-model":
				delete(startup, "model")
				events = append(events, child)
			case "missing-session":
				delete(startup, "session_id")
			case "wrong-parent-session":
				parent["session_id"] = "different-parent"
			case "duplicate-init":
				events = append(events, startup)
			case "different-init":
				events = append(events, map[string]any{"type": "system", "subtype": "init", "session_id": "other-session", "model": "other-model"})
			case "wrong-event-type":
				startup["type"] = "assistant"
			}
			raw := nativeClaudeStartupFixture(t, events...)
			if mutation == "malformed" {
				raw = append(raw, []byte("broken\n")...)
			}
			if mutation == "untyped" {
				raw = append(raw, []byte("null\n")...)
			}
			r := codexNativeLiveReceipt{SessionID: "cached-session", Model: "cached-model"}
			nativeCollectClaudeEvidence(&r, raw, []byte("{}"))
			err := nativeValidateClaudeInvocationIdentity(r, raw)
			valid := mutation == "none" || mutation == "child-init" || mutation == "other-model-fields"
			if valid {
				if err != nil || r.SessionID != "parent-session" || r.Model != "reported-parent-model" {
					t.Fatalf("parent startup identity missing or substituted: session=%q model=%q err=%v", r.SessionID, r.Model, err)
				}
				for _, field := range []string{"session", "model"} {
					altered := r
					if field == "session" {
						altered.SessionID = "substituted-session"
					} else {
						altered.Model = "substituted-model"
					}
					if err := nativeValidateClaudeInvocationIdentity(altered, raw); err == nil {
						t.Fatalf("substituted receipt %s accepted", field)
					}
				}
			} else if err == nil || r.SessionID != "" || r.Model != "" {
				t.Fatalf("unproved startup retained cached or child metadata: session=%q model=%q err=%v", r.SessionID, r.Model, err)
			}
		})
	}
}

func TestCodexNativeClaudeStartupReplayIdentity(t *testing.T) {
	for _, mutation := range []string{"missing-session", "wrong-session", "missing-model", "child-model"} {
		t.Run(mutation, func(t *testing.T) {
			raw := nativeClaudeStartupFixture(t,
				map[string]any{"type": "system", "subtype": "init", "session_id": "parent-session", "model": "reported-parent-model"},
				map[string]any{"type": "system", "subtype": "init", "session_id": "child-session", "model": "reported-child-model", "parent_tool_use_id": "helper"},
			)
			dir := t.TempDir()
			path := filepath.Join(dir, "claude-events.jsonl")
			liveSkillWrite(t, path, raw)
			r := codexNativeLiveReceipt{ProofContract: nativeCapabilityProofContract, ProofAmendmentSHA256: nativeCapabilityProofAmendmentSHA256, Scenario: "claude", SessionID: "parent-session", Model: "reported-parent-model", RawEvents: path, FixtureRoot: filepath.Join(dir, "repository"), Artifacts: map[string]string{path: lifecycleDigest(raw)}}
			switch mutation {
			case "missing-session":
				r.SessionID = ""
			case "wrong-session":
				r.SessionID = "child-session"
			case "missing-model":
				r.Model = ""
			case "child-model":
				r.Model = "reported-child-model"
			}
			if err := nativeReplayQualificationReceipt(t, &r); err == nil || !strings.Contains(err.Error(), "receipt session/model differs from actual parent startup") {
				t.Fatalf("standalone replay accepted substituted original startup identity: %v", err)
			}
			if r.Outcome != "incomplete" {
				t.Fatal("substituted startup identity qualified")
			}
		})
	}
}
