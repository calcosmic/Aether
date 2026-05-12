package cmd

import "fmt"

// validateBuildProvenance checks that at least one worker completed successfully
// and reported file changes (created, modified, or tests written). It rejects
// phantom builds where no worker actually produced output changes.
//
// SAFE-01: Rejects builds where all workers are in a non-success state.
// SAFE-02: Rejects builds where all completed workers have zero file changes.
func validateBuildProvenance(results []codexExternalBuildWorkerResult) error {
	if len(results) == 0 {
		return fmt.Errorf("build provenance: no worker results provided")
	}

	completedCount := 0
	for _, r := range results {
		if r.Status == "completed" {
			completedCount++
			if len(r.FilesModified) > 0 || len(r.FilesCreated) > 0 || len(r.TestsWritten) > 0 {
				return nil // At least one valid provenance entry found
			}
		}
	}

	if completedCount == 0 {
		return fmt.Errorf("build provenance: no workers completed successfully -- all %d worker(s) are in a non-success state", len(results))
	}
	return fmt.Errorf("build provenance: %d worker(s) completed but none reported file changes (created, modified, or tests) -- the build produced no changes", completedCount)
}

// traceContinueProvenance verifies that every completed worker dispatch has
// non-empty Outputs, ensuring claims trace back to valid worker results.
//
// SAFE-03: Rejects claims where completed dispatches have missing provenance.
// SAFE-04: Rejects claims when no completed dispatches exist (stale/missing).
//
// Per D-03: rejection causes halt. There is no warn-and-allow path.
func traceContinueProvenance(dispatches []codexBuildDispatch) error {
	if len(dispatches) == 0 {
		return fmt.Errorf("continue provenance: no worker dispatches found -- build did not produce verifiable results")
	}

	completedCount := 0
	for _, d := range dispatches {
		if d.Status == "completed" {
			completedCount++
			if len(d.Outputs) == 0 {
				return fmt.Errorf("continue provenance: worker %q claims completion but has no file outputs -- possible phantom build", d.Name)
			}
		}
	}

	if completedCount == 0 {
		return fmt.Errorf("continue provenance: no completed worker dispatches found -- build did not produce verifiable results")
	}

	return nil
}
