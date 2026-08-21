package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompletionPacketSubmittedBytes proves, through the real CLI surface
// (`aether build-finalize`), that structural validation now inspects the
// bytes the wrapper actually submitted -- not a re-marshal of the decoded Go
// struct. Every defective fixture below starts from a conformant baseline
// decoded into a generic map[string]any, splices a defect directly into that
// map, and re-encodes it: literal JSON text carrying the exact defect. A
// struct-marshaled fixture cannot express a misspelled field name at all
// (there is no Go field called files_modifed to set) and would silently
// prove nothing -- the exact trap the pre-plan implementation fell into.
func TestCompletionPacketSubmittedBytes(t *testing.T) {
	t.Run("misspelled field is rejected with an additionalProperties violation naming it", func(t *testing.T) {
		root, completionPath := newSubmittedBytesFixture(t)
		raw := conformantBaselineRawForTest(t, root)
		firstDispatch := submittedBytesFirstDispatch(t, raw)
		filesModified, ok := firstDispatch["files_modified"]
		if !ok {
			t.Fatal("expected the conformant baseline's first dispatch to carry files_modified")
		}
		delete(firstDispatch, "files_modified")
		firstDispatch["files_modifed"] = filesModified
		writeSubmittedBytesCompletion(t, completionPath, raw)

		beforeState, beforeManifest := readSubmittedBytesGuardStateForTest(t, root)
		stdout, stderrText, err := runBuildFinalizeCLIForTest(t, completionPath)
		if err == nil {
			t.Fatalf("expected build-finalize to reject the misspelled packet, stdout=%s", stdout)
		}
		envelope := parseLifecycleEnvelope(t, stderrText)
		if ok, _ := envelope["ok"].(bool); ok {
			t.Fatalf("expected ok:false, got %+v", envelope)
		}
		details := detailsFromEnvelopeForTest(t, envelope)
		found := false
		for _, d := range details {
			message := fmt.Sprint(d["message"])
			if d["rule"] == "additionalProperties" && d["field"] == "/dispatches/0" && strings.Contains(message, "files_modifed") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected an additionalProperties violation at /dispatches/0 naming files_modifed, got %+v", details)
		}
		assertSubmittedBytesGuardStateUnchanged(t, root, beforeState, beforeManifest)
	})

	t.Run("a wrong-typed field and a second unrelated violation both surface in one rejection", func(t *testing.T) {
		root, completionPath := newSubmittedBytesFixture(t)
		raw := conformantBaselineRawForTest(t, root)
		firstDispatch := submittedBytesFirstDispatch(t, raw)
		// A type mismatch (schema violation) plus an escaping claim path
		// (a semantic violation from a completely different validator) on
		// the same worker -- two independent, unrelated problems.
		firstDispatch["status"] = float64(5)
		firstDispatch["files_modified"] = []any{"/etc/passwd"}
		writeSubmittedBytesCompletion(t, completionPath, raw)

		beforeState, beforeManifest := readSubmittedBytesGuardStateForTest(t, root)
		stdout, stderrText, err := runBuildFinalizeCLIForTest(t, completionPath)
		if err == nil {
			t.Fatalf("expected build-finalize to reject the multi-violation packet, stdout=%s", stdout)
		}
		envelope := parseLifecycleEnvelope(t, stderrText)
		details := detailsFromEnvelopeForTest(t, envelope)
		if len(details) < 2 {
			t.Fatalf("expected at least 2 violations in a single rejection, got %d: %+v", len(details), details)
		}
		foundTypeViolation := false
		for _, d := range details {
			if d["field"] == "/dispatches/0/status" {
				foundTypeViolation = true
			}
		}
		if !foundTypeViolation {
			t.Fatalf("expected a violation referencing /dispatches/0/status, got %+v", details)
		}
		assertSubmittedBytesGuardStateUnchanged(t, root, beforeState, beforeManifest)
	})

	// Mandatory positive control: without this, a build-finalize that
	// rejects everything (a broken structural gate that always fires)
	// would still make every other test in this file pass.
	t.Run("a conformant packet still finalizes successfully", func(t *testing.T) {
		root, completionPath := newSubmittedBytesFixture(t)
		raw := conformantBaselineRawForTest(t, root)
		writeSubmittedBytesCompletion(t, completionPath, raw)

		stdout, stderrText, err := runBuildFinalizeCLIForTest(t, completionPath)
		if err != nil {
			t.Fatalf("expected a conformant packet to finalize successfully, stderr=%s err=%v", stderrText, err)
		}
		envelope := parseLifecycleEnvelope(t, stdout)
		if ok, _ := envelope["ok"].(bool); !ok {
			t.Fatalf("expected ok:true, got %+v", envelope)
		}
	})
}

// newSubmittedBytesFixture builds a fresh external-build-attempt test root
// (colony state ready, phase 1 awaiting a completion packet) and returns it
// alongside the path where this file's tests write their completion JSON.
func newSubmittedBytesFixture(t *testing.T) (string, string) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	return root, filepath.Join(root, "external-submitted-bytes-completion.json")
}

// conformantBaselineRawForTest builds a structurally valid completion packet
// via the same harness the rest of the external-build-attempt suite uses,
// marshals it, and decodes it back into a generic map[string]any -- the
// shape defects get spliced into before being re-encoded as the literal
// submitted bytes.
func conformantBaselineRawForTest(t *testing.T, root string) map[string]any {
	t.Helper()
	_, completion := prepareExternalBuildCompletion(t, root)
	data, err := json.Marshal(completion)
	if err != nil {
		t.Fatalf("marshal conformant baseline completion: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode conformant baseline completion: %v", err)
	}
	return raw
}

// submittedBytesFirstDispatch returns the first entry of the completion's
// top-level "dispatches" array (the wrapper's worker results) -- distinct
// from "dispatch_manifest"."dispatches" (the manifest's own dispatch plan),
// which sits one level deeper under a different key entirely.
func submittedBytesFirstDispatch(t *testing.T, raw map[string]any) map[string]any {
	t.Helper()
	dispatches, ok := raw["dispatches"].([]any)
	if !ok || len(dispatches) == 0 {
		t.Fatalf("expected a non-empty top-level dispatches array, got %+v", raw["dispatches"])
	}
	first, ok := dispatches[0].(map[string]any)
	if !ok {
		t.Fatalf("expected dispatches[0] to be an object, got %T", dispatches[0])
	}
	return first
}

func writeSubmittedBytesCompletion(t *testing.T, path string, raw map[string]any) {
	t.Helper()
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal spliced completion: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write completion file: %v", err)
	}
}

// runBuildFinalizeCLIForTest drives `aether build-finalize 1 --completion-file
// <path>` through rootCmd -- the real CLI surface, not the internal
// runCodexBuildFinalize function -- and returns the captured stdout/stderr
// text alongside rootCmd.Execute's error.
func runBuildFinalizeCLIForTest(t *testing.T, completionPath string) (string, string, error) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout, stderr = &outBuf, &errBuf
	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	err := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func detailsFromEnvelopeForTest(t *testing.T, envelope map[string]interface{}) []map[string]interface{} {
	t.Helper()
	raw, ok := envelope["details"].([]interface{})
	if !ok {
		t.Fatalf("expected envelope details to be an array, got %+v", envelope)
	}
	details := make([]map[string]interface{}, 0, len(raw))
	for _, entry := range raw {
		m, ok := entry.(map[string]interface{})
		if !ok {
			t.Fatalf("expected each detail entry to be an object, got %+v", entry)
		}
		details = append(details, m)
	}
	return details
}

func readColonyStateBytesForTest(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read colony state: %v", err)
	}
	return data
}

// readSubmittedBytesGuardStateForTest snapshots COLONY_STATE.json and the
// phase's build/phase-1/manifest.json (already written by the plan-only
// step prepareExternalBuildCompletion drove) before a build-finalize call,
// so a rejected completion's effect on both can be proven to be exactly
// zero -- not merely "the manifest still exists" but byte-identical.
func readSubmittedBytesGuardStateForTest(t *testing.T, root string) ([]byte, []byte) {
	t.Helper()
	state := readColonyStateBytesForTest(t, root)
	manifest, err := os.ReadFile(filepath.Join(root, ".aether", "data", "build", "phase-1", "manifest.json"))
	if err != nil {
		t.Fatalf("read build manifest before build-finalize: %v", err)
	}
	return state, manifest
}

// assertSubmittedBytesGuardStateUnchanged locks in D-06 atomicity through
// the real CLI: a rejected completion packet must leave both
// COLONY_STATE.json and the phase's build manifest byte-for-byte exactly as
// they were found -- no checkpoint save, no attempt mutation, nothing.
func assertSubmittedBytesGuardStateUnchanged(t *testing.T, root string, beforeState, beforeManifest []byte) {
	t.Helper()
	afterState := readColonyStateBytesForTest(t, root)
	if string(beforeState) != string(afterState) {
		t.Fatalf("colony state changed after a rejected build-finalize:\nbefore=%s\nafter=%s", beforeState, afterState)
	}
	afterManifest, err := os.ReadFile(filepath.Join(root, ".aether", "data", "build", "phase-1", "manifest.json"))
	if err != nil {
		t.Fatalf("read build manifest after build-finalize: %v", err)
	}
	if string(beforeManifest) != string(afterManifest) {
		t.Fatalf("build manifest changed after a rejected build-finalize:\nbefore=%s\nafter=%s", beforeManifest, afterManifest)
	}
}
