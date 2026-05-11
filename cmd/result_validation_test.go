package cmd

import (
	"strings"
	"testing"
)

// --- Shared validateWorkerResultIdentity tests (Task 4.1) ---
// These tests target the shared validateWorkerResultIdentity() helper
// implemented in cmd/result_validation.go.
//
// The helper consolidates three structurally identical identity validation
// functions at build:474, plan:330, continue:734 into a single function.
//
// Expected error format: "external worker result {name} {field} = {got}, want {want}"

func TestValidateWorkerResultIdentity_AcceptsMatchingIdentity(t *testing.T) {
	dispatch := workerIdentitySpec{
		Caste:         "builder",
		Stage:         "wave",
		TaskID:        "1.1",
		Wave:          1,
		ExecutionWave: 1,
	}
	result := workerIdentitySpec{
		Caste:         "builder",
		Stage:         "wave",
		TaskID:        "1.1",
		Wave:          1,
		ExecutionWave: 1,
	}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected matching identity to pass, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_AcceptsEmptyResultFields(t *testing.T) {
	dispatch := workerIdentitySpec{
		Caste:  "builder",
		Stage:  "wave",
		TaskID: "1.1",
		Wave:   1,
	}
	// Result has all identity fields empty — should be accepted (no conflict)
	result := workerIdentitySpec{}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected empty result fields to pass, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_RejectsWrongCaste(t *testing.T) {
	dispatch := workerIdentitySpec{Caste: "builder"}
	result := workerIdentitySpec{Caste: "watcher"}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err == nil {
		t.Fatal("expected wrong caste error")
	}
	// Verify consistent error format
	if !strings.Contains(err.Error(), "external worker result Forge-67") {
		t.Fatalf("error missing worker name prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "caste") {
		t.Fatalf("error missing field name: %v", err)
	}
	if !strings.Contains(err.Error(), "watcher") {
		t.Fatalf("error missing got value: %v", err)
	}
	if !strings.Contains(err.Error(), "builder") {
		t.Fatalf("error missing want value: %v", err)
	}
}

func TestValidateWorkerResultIdentity_RejectsWrongStage(t *testing.T) {
	dispatch := workerIdentitySpec{Stage: "wave"}
	result := workerIdentitySpec{Stage: "verification"}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err == nil {
		t.Fatal("expected wrong stage error")
	}
	if !strings.Contains(err.Error(), "stage") {
		t.Fatalf("error missing field name: %v", err)
	}
}

func TestValidateWorkerResultIdentity_RejectsWrongTaskID(t *testing.T) {
	dispatch := workerIdentitySpec{TaskID: "1.1"}
	result := workerIdentitySpec{TaskID: "2.2"}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err == nil {
		t.Fatal("expected wrong task_id error")
	}
	if !strings.Contains(err.Error(), "task_id") {
		t.Fatalf("error missing field name: %v", err)
	}
}

func TestValidateWorkerResultIdentity_RejectsWrongWave(t *testing.T) {
	dispatch := workerIdentitySpec{Wave: 1}
	result := workerIdentitySpec{Wave: 2}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err == nil {
		t.Fatal("expected wrong wave error")
	}
	if !strings.Contains(err.Error(), "wave") {
		t.Fatalf("error missing field name: %v", err)
	}
}

func TestValidateWorkerResultIdentity_RejectsWrongExecutionWave(t *testing.T) {
	dispatch := workerIdentitySpec{ExecutionWave: 1}
	result := workerIdentitySpec{ExecutionWave: 3}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err == nil {
		t.Fatal("expected wrong execution_wave error")
	}
	if !strings.Contains(err.Error(), "execution_wave") {
		t.Fatalf("error missing field name: %v", err)
	}
}

func TestValidateWorkerResultIdentity_CaseInsensitiveCaste(t *testing.T) {
	dispatch := workerIdentitySpec{Caste: "Builder"}
	result := workerIdentitySpec{Caste: "builder"}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected case-insensitive caste match, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_CaseInsensitiveStage(t *testing.T) {
	dispatch := workerIdentitySpec{Stage: "Wave"}
	result := workerIdentitySpec{Stage: "wave"}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected case-insensitive stage match, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_ZeroWaveSkipped(t *testing.T) {
	// When result wave is 0 (unset), it should not be compared
	dispatch := workerIdentitySpec{Wave: 1}
	result := workerIdentitySpec{Wave: 0}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected zero wave to be skipped, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_ZeroExecutionWaveSkipped(t *testing.T) {
	// When result execution_wave is 0 (unset), it should not be compared
	dispatch := workerIdentitySpec{ExecutionWave: 1}
	result := workerIdentitySpec{ExecutionWave: 0}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected zero execution_wave to be skipped, got: %v", err)
	}
}

func TestValidateWorkerResultIdentity_EmptyTaskIDResultSkipped(t *testing.T) {
	// When result task_id is empty, it should not be compared
	dispatch := workerIdentitySpec{TaskID: "1.1"}
	result := workerIdentitySpec{TaskID: ""}
	err := validateWorkerResultIdentity("Forge-67", dispatch, result)
	if err != nil {
		t.Fatalf("expected empty task_id to be skipped, got: %v", err)
	}
}
