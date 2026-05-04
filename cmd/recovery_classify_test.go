package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- Classification registry tests ---

// TestFailureClassifications_CoversWorkerStatuses verifies that every terminal
// worker status from isTerminalExternalBuildStatus has a deterministic failure
// classification. Per RECV-01, no status should return an empty classification.
func TestFailureClassifications_CoversWorkerStatuses(t *testing.T) {
	// Terminal worker statuses from cmd/codex_build_finalize.go isTerminalExternalBuildStatus
	terminalStatuses := []string{"failed", "blocked", "timeout", "manually-reconciled"}
	for _, status := range terminalStatuses {
		classification, failureType, rationale := classifyWorkerFailure(status, "")
		if classification == "" {
			t.Errorf("status %q has no classification", status)
		}
		if failureType == "" {
			t.Errorf("status %q has no failure type", status)
		}
		if rationale == "" {
			t.Errorf("status %q has no rationale", status)
		}
	}
}

// TestFailureClassifications_HasAllRegistryEntries verifies the classification
// registry has exactly 13 entries, each with non-empty fields.
func TestFailureClassifications_HasAllRegistryEntries(t *testing.T) {
	if len(failureClassifications) != 13 {
		t.Errorf("expected 13 failure classifications, got %d", len(failureClassifications))
	}
	for pattern, entry := range failureClassifications {
		if entry.Classification == "" {
			t.Errorf("pattern %q has empty Classification", pattern)
		}
		if entry.FailureType == "" {
			t.Errorf("pattern %q has empty FailureType", pattern)
		}
		if entry.Rationale == "" {
			t.Errorf("pattern %q has empty Rationale", pattern)
		}
	}
}

// --- classifyWorkerFailure tests ---

// TestClassifyWorkerFailure_Timeout verifies timeout is Recoverable+Transient per RECV-05.
func TestClassifyWorkerFailure_Timeout(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("timeout", "worker timed out after 300s")
	if classification != Recoverable {
		t.Errorf("timeout should be Recoverable, got %q", classification)
	}
	if failureType != Transient {
		t.Errorf("timeout should be Transient, got %q", failureType)
	}
	if rationale == "" {
		t.Error("timeout should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_BadTaskSpec verifies bad_task_spec is Blocking+Systemic per RECV-01.
func TestClassifyWorkerFailure_BadTaskSpec(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("bad_task_spec", "")
	if classification != Blocking {
		t.Errorf("bad_task_spec should be Blocking, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("bad_task_spec should be Systemic, got %q", failureType)
	}
	if rationale == "" {
		t.Error("bad_task_spec should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_MissingDependency verifies missing_dependency is Blocking+Systemic.
func TestClassifyWorkerFailure_MissingDependency(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("missing_dependency", "required package not found")
	if classification != Blocking {
		t.Errorf("missing_dependency should be Blocking, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("missing_dependency should be Systemic, got %q", failureType)
	}
	if rationale == "" {
		t.Error("missing_dependency should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_ContextOverflowViaErrorMessage verifies error message
// pattern matching: "context window" in error detects context overflow per RECV-05.
// Uses a status not in the registry so error message pattern matching is exercised.
func TestClassifyWorkerFailure_ContextOverflowViaErrorMessage(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("error", "exceeded context window limit")
	if classification != Recoverable {
		t.Errorf("context overflow should be Recoverable, got %q", classification)
	}
	if failureType != Transient {
		t.Errorf("context overflow should be Transient, got %q", failureType)
	}
	if rationale == "" {
		t.Error("context overflow should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_FileNotFoundViaErrorMessage verifies error message
// pattern matching: "no such file" in error maps to Blocking+Systemic.
// Uses a status not in the registry so error message pattern matching is exercised.
func TestClassifyWorkerFailure_FileNotFoundViaErrorMessage(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("error", "no such file or directory: config.json")
	if classification != Blocking {
		t.Errorf("file not found should be Blocking, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("file not found should be Systemic, got %q", failureType)
	}
	if rationale == "" {
		t.Error("file not found should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_PermissionDenied verifies error message pattern
// matching: "permission denied" maps to Blocking+Systemic.
// Uses a status not in the registry so error message pattern matching is exercised.
func TestClassifyWorkerFailure_PermissionDenied(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("error", "permission denied: /root/secret")
	if classification != Blocking {
		t.Errorf("permission denied should be Blocking, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("permission denied should be Systemic, got %q", failureType)
	}
	if rationale == "" {
		t.Error("permission denied should have non-empty rationale")
	}
}

// TestClassifyWorkerFailure_UnknownDefaultsToRequiresAttempt verifies D-11:
// unknown errors default to RequiresAttempt+Systemic.
func TestClassifyWorkerFailure_UnknownDefaultsToRequiresAttempt(t *testing.T) {
	classification, failureType, rationale := classifyWorkerFailure("bizarre_unknown_status", "something weird happened")
	if classification != RequiresAttempt {
		t.Errorf("unknown failure should default to RequiresAttempt, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("unknown failure should default to Systemic, got %q", failureType)
	}
	if !strings.Contains(rationale, "defaulting to requires-attempt") {
		t.Errorf("rationale should mention defaulting, got %q", rationale)
	}
}

// TestClassifyWorkerFailure_CaseInsensitive verifies status normalization.
func TestClassifyWorkerFailure_CaseInsensitive(t *testing.T) {
	classification, _, _ := classifyWorkerFailure("TIMEOUT", "msg")
	if classification != Recoverable {
		t.Errorf("TIMEOUT should be Recoverable after normalization, got %q", classification)
	}

	classification2, _, _ := classifyWorkerFailure("Failed", "msg")
	if classification2 != RequiresAttempt {
		t.Errorf("Failed should be RequiresAttempt after normalization, got %q", classification2)
	}
}

// TestClassifyWorkerFailure_EmptyInput verifies empty status+error defaults safely.
func TestClassifyWorkerFailure_EmptyInput(t *testing.T) {
	classification, failureType, _ := classifyWorkerFailure("", "")
	if classification != RequiresAttempt {
		t.Errorf("empty input should default to RequiresAttempt, got %q", classification)
	}
	if failureType != Systemic {
		t.Errorf("empty input should default to Systemic, got %q", failureType)
	}
}

// TestFailureClassifications_NoCrossDomainImports is a build-time guarantee.
// cmd/recovery_classify.go must never import GateClassificationTier.
// The grep gate in acceptance criteria covers this at CI time.
func TestFailureClassifications_NoCrossDomainImports(t *testing.T) {
	// This test exists as documentation. The actual enforcement is via CI grep.
	// If recovery_classify.go imports gate classification types, that violates
	// the domain boundary between failure classification and gate classification.
}

// --- JSON roundtrip tests ---

// TestFailureRecord_JSONRoundtrip verifies FailureRecord marshal/unmarshal
// preserves all fields.
func TestFailureRecord_JSONRoundtrip(t *testing.T) {
	original := FailureRecord{
		WorkerName:     "Builder-67",
		TaskID:         "2.1",
		Caste:          "builder",
		Phase:          2,
		Status:         "timeout",
		Classification: Recoverable,
		FailureType:    Transient,
		ErrorMessage:   "worker timed out after 300s",
		Timestamp:      "2026-05-03T12:00:00Z",
		RetryCount:     1,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded FailureRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.WorkerName != "Builder-67" {
		t.Errorf("worker_name mismatch: got %q", decoded.WorkerName)
	}
	if decoded.TaskID != "2.1" {
		t.Errorf("task_id mismatch: got %q", decoded.TaskID)
	}
	if decoded.Caste != "builder" {
		t.Errorf("caste mismatch: got %q", decoded.Caste)
	}
	if decoded.Phase != 2 {
		t.Errorf("phase mismatch: got %d", decoded.Phase)
	}
	if decoded.Status != "timeout" {
		t.Errorf("status mismatch: got %q", decoded.Status)
	}
	if decoded.Classification != Recoverable {
		t.Errorf("classification mismatch: got %q", decoded.Classification)
	}
	if decoded.FailureType != Transient {
		t.Errorf("failure_type mismatch: got %q", decoded.FailureType)
	}
	if decoded.ErrorMessage != "worker timed out after 300s" {
		t.Errorf("error_message mismatch: got %q", decoded.ErrorMessage)
	}
	if decoded.Timestamp != "2026-05-03T12:00:00Z" {
		t.Errorf("timestamp mismatch: got %q", decoded.Timestamp)
	}
	if decoded.RetryCount != 1 {
		t.Errorf("retry_count mismatch: got %d", decoded.RetryCount)
	}
}

// TestRecoveryLogEntry_BackwardCompat_MinimalJSON verifies that minimal JSON
// without optional fields deserializes cleanly (backward compatibility).
func TestRecoveryLogEntry_BackwardCompat_MinimalJSON(t *testing.T) {
	minimal := `{"id":"rl_1","failure":{"worker_name":"W-1","phase":1,"status":"failed","classification":"requires-attempt","failure_type":"systemic","error_message":"oops","timestamp":"2026-05-01T00:00:00Z"},"action_taken":"retry","outcome":"recovered","attempt_number":1,"timestamp":"2026-05-01T00:00:01Z"}`

	var entry RecoveryLogEntry
	if err := json.Unmarshal([]byte(minimal), &entry); err != nil {
		t.Fatalf("minimal JSON should deserialize: %v", err)
	}

	if entry.Failure.WorkerName != "W-1" {
		t.Errorf("worker_name mismatch: got %q", entry.Failure.WorkerName)
	}
	if entry.Detail != "" {
		t.Errorf("detail should be empty for minimal JSON, got %q", entry.Detail)
	}
	if entry.Failure.TaskID != "" {
		t.Errorf("task_id should be empty for minimal JSON, got %q", entry.Failure.TaskID)
	}
	if entry.Failure.Caste != "" {
		t.Errorf("caste should be empty for minimal JSON, got %q", entry.Failure.Caste)
	}
	if entry.Failure.RetryCount != 0 {
		t.Errorf("retry_count should be 0 for minimal JSON, got %d", entry.Failure.RetryCount)
	}
}

// TestRecoveryLogEntry_FullJSON_Roundtrip verifies full RecoveryLogEntry
// marshal/unmarshal preserves all fields including optional ones.
func TestRecoveryLogEntry_FullJSON_Roundtrip(t *testing.T) {
	original := RecoveryLogEntry{
		ID: "rl_42",
		Failure: FailureRecord{
			WorkerName:     "Watcher-23",
			TaskID:         "3.2",
			Caste:          "watcher",
			Phase:          3,
			Status:         "timeout",
			Classification: Recoverable,
			FailureType:    Transient,
			ErrorMessage:   "watcher timed out",
			Timestamp:      "2026-05-03T14:00:00Z",
			RetryCount:     2,
		},
		ActionTaken:   "retry",
		Outcome:       "recovered",
		AttemptNumber: 2,
		Timestamp:     "2026-05-03T14:00:01Z",
		Detail:        "Retried after 30s pause",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded RecoveryLogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != "rl_42" {
		t.Errorf("id mismatch: got %q", decoded.ID)
	}
	if decoded.Failure.WorkerName != "Watcher-23" {
		t.Errorf("failure.worker_name mismatch: got %q", decoded.Failure.WorkerName)
	}
	if decoded.Failure.TaskID != "3.2" {
		t.Errorf("failure.task_id mismatch: got %q", decoded.Failure.TaskID)
	}
	if decoded.Failure.Caste != "watcher" {
		t.Errorf("failure.caste mismatch: got %q", decoded.Failure.Caste)
	}
	if decoded.Failure.RetryCount != 2 {
		t.Errorf("failure.retry_count mismatch: got %d", decoded.Failure.RetryCount)
	}
	if decoded.ActionTaken != "retry" {
		t.Errorf("action_taken mismatch: got %q", decoded.ActionTaken)
	}
	if decoded.Outcome != "recovered" {
		t.Errorf("outcome mismatch: got %q", decoded.Outcome)
	}
	if decoded.AttemptNumber != 2 {
		t.Errorf("attempt_number mismatch: got %d", decoded.AttemptNumber)
	}
	if decoded.Detail != "Retried after 30s pause" {
		t.Errorf("detail mismatch: got %q", decoded.Detail)
	}
}

// --- Recovery log persistence tests ---

// TestRecoveryLog_WriteRead verifies recovery log entries persist to disk and
// read back with all fields preserved.
func TestRecoveryLog_WriteRead(t *testing.T) {
	s, _ := newTestStore(t)
	saveStore := store
	store = s
	t.Cleanup(func() { store = saveStore })

	entries := []RecoveryLogEntry{
		{
			ID: "rl_1001",
			Failure: FailureRecord{
				WorkerName: "Builder-67", Phase: 1, Status: "timeout",
				Classification: Recoverable, FailureType: Transient,
				ErrorMessage: "timed out", Timestamp: "2026-05-03T12:00:00Z",
			},
			ActionTaken: "retry", Outcome: "recovered",
			AttemptNumber: 1, Timestamp: "2026-05-03T12:00:01Z",
		},
		{
			ID: "rl_1002",
			Failure: FailureRecord{
				WorkerName: "Watcher-23", Phase: 1, Status: "bad_task_spec",
				Classification: Blocking, FailureType: Systemic,
				ErrorMessage: "invalid spec", Timestamp: "2026-05-03T12:01:00Z",
			},
			ActionTaken: "escalate", Outcome: "escalated",
			AttemptNumber: 1, Timestamp: "2026-05-03T12:01:01Z",
		},
	}

	if err := recoveryLogWritePhase(1, entries); err != nil {
		t.Fatalf("recoveryLogWritePhase failed: %v", err)
	}

	readBack, err := recoveryLogReadPhase(1)
	if err != nil {
		t.Fatalf("recoveryLogReadPhase failed: %v", err)
	}

	if readBack.Phase != 1 {
		t.Errorf("phase mismatch: got %d", readBack.Phase)
	}
	if len(readBack.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readBack.Entries))
	}
	if readBack.Entries[0].Failure.WorkerName != "Builder-67" {
		t.Errorf("entry 0 worker_name mismatch: got %q", readBack.Entries[0].Failure.WorkerName)
	}
	if readBack.Entries[0].ActionTaken != "retry" {
		t.Errorf("entry 0 action_taken mismatch: got %q", readBack.Entries[0].ActionTaken)
	}
	if readBack.Entries[0].Outcome != "recovered" {
		t.Errorf("entry 0 outcome mismatch: got %q", readBack.Entries[0].Outcome)
	}
	if readBack.Entries[1].Failure.WorkerName != "Watcher-23" {
		t.Errorf("entry 1 worker_name mismatch: got %q", readBack.Entries[1].Failure.WorkerName)
	}
	if readBack.Entries[1].ActionTaken != "escalate" {
		t.Errorf("entry 1 action_taken mismatch: got %q", readBack.Entries[1].ActionTaken)
	}
}

// TestRecoveryLog_ReadNonexistent verifies reading a nonexistent phase returns error.
func TestRecoveryLog_ReadNonexistent(t *testing.T) {
	s, _ := newTestStore(t)
	saveStore := store
	store = s
	t.Cleanup(func() { store = saveStore })

	_, err := recoveryLogReadPhase(999)
	if err == nil {
		t.Error("expected error for nonexistent phase, got nil")
	}
}

// --- CLI command tests ---

// failureClassifyJSONEntry is a local type for deserializing failure-classify --json
// output (the failureClassificationEntry is unexported but accessible in same package).
type failureClassifyJSONEntry struct {
	Classification string `json:"Classification"`
	FailureType    string `json:"FailureType"`
	Rationale      string `json:"Rationale"`
}

// TestFailureClassifyCmd_JSONOutput verifies failure-classify --json outputs valid JSON
// with classification data.
func TestFailureClassifyCmd_JSONOutput(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"failure-classify", "--json"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	// outputOK wraps in {"ok":true,"result":...}
	var wrapper struct {
		OK     bool                                `json:"ok"`
		Result map[string]failureClassifyJSONEntry `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &wrapper); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if !wrapper.OK {
		t.Error("expected ok:true in output")
	}
	if _, ok := wrapper.Result["timeout"]; !ok {
		t.Error("expected JSON to contain 'timeout' key")
	}
	if wrapper.Result["timeout"].Classification != string(Recoverable) {
		t.Errorf("timeout Classification mismatch: got %q", wrapper.Result["timeout"].Classification)
	}
}

// TestFailureClassifyCmd_TableOutput verifies failure-classify (default) outputs a
// table with expected headers and content.
func TestFailureClassifyCmd_TableOutput(t *testing.T) {
	gateCmdTestSetup(t)

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"failure-classify"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	for _, header := range []string{"PATTERN", "CLASSIFICATION", "FAILURE TYPE", "RATIONALE"} {
		if !strings.Contains(output, header) {
			t.Errorf("expected table to contain header %q", header)
		}
	}
	if !strings.Contains(output, "timeout") {
		t.Error("expected table to contain 'timeout' pattern")
	}
	if !strings.Contains(output, "recoverable") {
		t.Error("expected table to contain 'recoverable' classification")
	}
}

// TestRecoveryLogReadCmd verifies recovery-log-read CLI command works for a phase
// without an existing log file (returns empty entries).
func TestRecoveryLogReadCmd(t *testing.T) {
	gateCmdTestSetup(t)
	s, _ := newTestStore(t)
	saveStore := store
	store = s
	t.Cleanup(func() { store = saveStore })

	var buf bytes.Buffer
	rootCmd.SetArgs([]string{"recovery-log-read", "--phase", "999"})
	stdout = &buf
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"entries"`) {
		t.Errorf("expected output to contain 'entries', got: %s", output)
	}
	if !strings.Contains(output, `"total":0`) {
		t.Errorf("expected output to contain 'total:0', got: %s", output)
	}
}
