package cmd

import "testing"

// --- Provenance validation tests (Phase 88, Plan 01, Task 1) ---

func TestValidateBuildProvenance_AllFailed(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "failed", Task: "task-1"},
		{Name: "builder-2", Status: "failed", Task: "task-2"},
	}
	err := validateBuildProvenance(results)
	if err == nil {
		t.Fatal("expected error when all workers failed, got nil")
	}
	if !contains(err.Error(), "no workers completed successfully") {
		t.Errorf("error should mention no workers completed, got: %s", err.Error())
	}
}

func TestValidateBuildProvenance_AllBlocked(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "blocked", Task: "task-1"},
		{Name: "builder-2", Status: "blocked", Task: "task-2"},
	}
	err := validateBuildProvenance(results)
	if err == nil {
		t.Fatal("expected error when all workers blocked, got nil")
	}
	if !contains(err.Error(), "no workers completed successfully") {
		t.Errorf("error should mention no workers completed, got: %s", err.Error())
	}
}

func TestValidateBuildProvenance_ZeroFilesModified(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "completed", Task: "task-1", FilesModified: []string{}},
		{Name: "builder-2", Status: "completed", Task: "task-2", FilesModified: nil},
	}
	err := validateBuildProvenance(results)
	if err == nil {
		t.Fatal("expected error when all completed workers have zero FilesModified, got nil")
	}
	if !contains(err.Error(), "none reported file changes") {
		t.Errorf("error should mention no file changes, got: %s", err.Error())
	}
}

func TestValidateBuildProvenance_OneSuccessful(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "failed", Task: "task-1"},
		{Name: "builder-2", Status: "completed", Task: "task-2", FilesModified: []string{"cmd/main.go"}},
		{Name: "builder-3", Status: "blocked", Task: "task-3"},
	}
	err := validateBuildProvenance(results)
	if err != nil {
		t.Fatalf("expected nil when at least one worker completed with FilesModified, got: %s", err)
	}
}

func TestValidateBuildProvenance_NilSlice(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "completed", Task: "task-1", FilesModified: nil},
	}
	err := validateBuildProvenance(results)
	if err == nil {
		t.Fatal("expected error when completed worker has nil FilesModified, got nil")
	}
	if !contains(err.Error(), "none reported file changes") {
		t.Errorf("error should mention no file changes, got: %s", err.Error())
	}
}

func TestValidateBuildProvenance_FilesCreatedOnlyAccepted(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "completed", Task: "task-1", FilesCreated: []string{"cmd/new.go"}, FilesModified: nil},
	}
	err := validateBuildProvenance(results)
	if err != nil {
		t.Fatalf("expected nil when completed worker has FilesCreated, got: %s", err)
	}
}

func TestValidateBuildProvenance_TestsWrittenOnlyAccepted(t *testing.T) {
	results := []codexExternalBuildWorkerResult{
		{Name: "builder-1", Status: "completed", Task: "task-1", TestsWritten: []string{"cmd/main_test.go"}, FilesModified: nil},
	}
	err := validateBuildProvenance(results)
	if err != nil {
		t.Fatalf("expected nil when completed worker has TestsWritten, got: %s", err)
	}
}

func TestValidateBuildProvenance_EmptyResults(t *testing.T) {
	err := validateBuildProvenance([]codexExternalBuildWorkerResult{})
	if err == nil {
		t.Fatal("expected error for empty results, got nil")
	}
}

// --- Continue provenance tracing tests ---

func TestTraceContinueProvenance_ValidClaims(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "builder-1", Status: "completed", Task: "task-1", Outputs: []string{"cmd/main.go"}},
		{Name: "builder-2", Status: "completed", Task: "task-2", Outputs: []string{"pkg/util.go", "pkg/util_test.go"}},
	}
	err := traceContinueProvenance(dispatches)
	if err != nil {
		t.Fatalf("expected nil for valid claims, got: %s", err)
	}
}

func TestTraceContinueProvenance_MissingProvenance(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Name: "builder-1", Status: "completed", Task: "task-1", Outputs: []string{}},
		{Name: "builder-2", Status: "completed", Task: "task-2", Outputs: nil},
	}
	err := traceContinueProvenance(dispatches)
	if err == nil {
		t.Fatal("expected error when completed dispatches have empty Outputs, got nil")
	}
	if !contains(err.Error(), "provenance") {
		t.Errorf("error should mention provenance, got: %s", err.Error())
	}
}

func TestTraceContinueProvenance_StaleProvenance(t *testing.T) {
	// No completed dispatches at all -- build produced no verifiable results
	dispatches := []codexBuildDispatch{
		{Name: "builder-1", Status: "failed", Task: "task-1"},
		{Name: "builder-2", Status: "blocked", Task: "task-2"},
	}
	err := traceContinueProvenance(dispatches)
	if err == nil {
		t.Fatal("expected error when no completed dispatches exist, got nil")
	}
	if !contains(err.Error(), "no completed worker dispatches") {
		t.Errorf("error should mention no completed dispatches, got: %s", err.Error())
	}
}

func TestTraceContinueProvenance_RejectsWithHalt(t *testing.T) {
	// Per D-03: rejection causes halt (returns error). No warn-and-allow.
	dispatches := []codexBuildDispatch{
		{Name: "builder-1", Status: "completed", Task: "task-1", Outputs: nil},
	}
	err := traceContinueProvenance(dispatches)
	if err == nil {
		t.Fatal("expected error (halt) when provenance is missing, got nil")
	}
	// The function must return an error, not log a warning and return nil.
}

func TestTraceContinueProvenance_EmptyDispatches(t *testing.T) {
	err := traceContinueProvenance([]codexBuildDispatch{})
	if err == nil {
		t.Fatal("expected error for empty dispatches, got nil")
	}
}

func TestTraceContinueProvenance_MixedValidAndInvalid(t *testing.T) {
	// One completed with outputs, one completed without -- should fail
	dispatches := []codexBuildDispatch{
		{Name: "builder-1", Status: "completed", Task: "task-1", Outputs: []string{"cmd/main.go"}},
		{Name: "builder-2", Status: "completed", Task: "task-2", Outputs: nil},
	}
	err := traceContinueProvenance(dispatches)
	if err == nil {
		t.Fatal("expected error when one completed dispatch has no outputs, got nil")
	}
}

// contains helper is provided by cmd/medic_scanner_test.go in the same package.
