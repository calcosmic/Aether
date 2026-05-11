package cmd

import (
	"fmt"
	"strings"
)

// workerIdentitySpec is the shared interface for worker result identity fields
// used across build, plan, and continue finalizers.
type workerIdentitySpec struct {
	Caste         string
	Stage         string
	TaskID        string
	Wave          int
	ExecutionWave int
}

// validateWorkerResultIdentity checks that a worker result's identity fields
// match the dispatch identity. Empty or zero-valued result fields are skipped
// (no conflict). Caste and stage comparisons are case-insensitive.
//
// The error format is: "external worker result {name} {field} = {got}, want {want}"
func validateWorkerResultIdentity(name string, dispatch, result workerIdentitySpec) error {
	if value := strings.TrimSpace(result.Caste); value != "" && !strings.EqualFold(value, dispatch.Caste) {
		return fmt.Errorf("external worker result %s caste = %q, want %q", name, value, dispatch.Caste)
	}
	if value := strings.TrimSpace(result.Stage); value != "" && !strings.EqualFold(value, dispatch.Stage) {
		return fmt.Errorf("external worker result %s stage = %q, want %q", name, value, dispatch.Stage)
	}
	if value := strings.TrimSpace(result.TaskID); value != "" && value != strings.TrimSpace(dispatch.TaskID) {
		return fmt.Errorf("external worker result %s task_id = %q, want %q", name, value, dispatch.TaskID)
	}
	if result.Wave > 0 && dispatch.Wave > 0 && result.Wave != dispatch.Wave {
		return fmt.Errorf("external worker result %s wave = %d, want %d", name, result.Wave, dispatch.Wave)
	}
	if result.ExecutionWave > 0 && dispatch.ExecutionWave > 0 && result.ExecutionWave != dispatch.ExecutionWave {
		return fmt.Errorf("external worker result %s execution_wave = %d, want %d", name, result.ExecutionWave, dispatch.ExecutionWave)
	}
	return nil
}
