package cmd

import (
	"sort"
	"strings"
	"time"
)

// Select a single latest terminal attempt per task. Combining every historical
// status makes a repaired task remain both blocked and completed forever.
func latestPhaseTaskAttempts(phaseID int) map[string]buildAttemptRecord {
	records := listBuildAttemptsForPhase(phaseID)
	sort.SliceStable(records, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, records[i].StartedAt)
		b, _ := time.Parse(time.RFC3339Nano, records[j].StartedAt)
		if a.Equal(b) {
			return records[i].ID < records[j].ID
		}
		return a.Before(b)
	})
	latest := map[string]buildAttemptRecord{}
	for _, record := range records {
		mode := strings.ToLower(strings.TrimSpace(record.DispatchMode))
		if record.Phase != phaseID || mode == "" || mode == "simulated" || mode == "synthetic" || mode == "plan-only" {
			continue
		}
		switch record.Status {
		case buildAttemptBuilt, buildAttemptPartial, buildAttemptFailed, buildAttemptInterrupted:
		default:
			continue
		}
		for _, dispatch := range record.Dispatches {
			for _, id := range dispatchCoveredTaskIDs(dispatch) {
				latest[id] = record
			}
		}
	}
	return latest
}

// Reconstruct phase-wide evidence without rewriting the attempt journal or
// photographing files again. Current-task results always override old proof.
func phaseClaimsForContinue(manifest codexContinueManifest, current codexBuildClaims) codexBuildClaims {
	if !manifest.Present || manifestUsesSyntheticDispatch(manifest) || current.BuildPhase != manifest.Data.Phase {
		return current
	}
	claimed := map[string]bool{}
	for _, id := range manifest.Data.SelectedTasks {
		claimed[id] = true
	}
	for _, dispatch := range manifest.Data.Dispatches {
		for _, id := range dispatchCoveredTaskIDs(dispatch) {
			claimed[id] = true
		}
	}
	for _, task := range current.TaskClaims {
		claimed[task.TaskID] = true
	}
	evidence := map[string]bool{}
	for _, item := range current.ArtifactEvidence {
		evidence[item.Path] = true
	}
	latest := latestPhaseTaskAttempts(current.BuildPhase)
	ids := make([]string, 0, len(latest))
	for id := range latest {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		record := latest[id]
		if claimed[id] || (record.Status != buildAttemptBuilt && record.Status != buildAttemptPartial) || record.Claims == nil || record.Claims.BuildPhase != current.BuildPhase {
			continue
		}
		// Without an exact plan revision, historical evidence cannot safely be
		// attributed to this plan rather than an earlier goal with the same IDs.
		if record.PlanManifest == nil || manifest.Data.PlanRevisionID == "" || record.PlanManifest.PlanRevisionID != manifest.Data.PlanRevisionID || record.PlanManifest.Root != manifest.Data.Root {
			continue
		}
		completed := completedBuildTaskIDs(record.Dispatches)
		if _, ok := completed[id]; !ok {
			continue
		}
		for _, task := range record.Claims.TaskClaims {
			if task.TaskID != id {
				continue
			}
			current.TaskClaims = append(current.TaskClaims, task)
			current.FilesCreated = append(current.FilesCreated, task.FilesCreated...)
			current.FilesModified = append(current.FilesModified, task.FilesModified...)
			current.TestsWritten = append(current.TestsWritten, task.TestsWritten...)
			paths := append(append(append([]string{}, task.FilesCreated...), task.FilesModified...), task.TestsWritten...)
			for _, item := range record.Claims.ArtifactEvidence {
				if !evidence[item.Path] && containsString(paths, item.Path) {
					current.ArtifactEvidence = append(current.ArtifactEvidence, item)
					evidence[item.Path] = true
				}
			}
		}
	}
	current.FilesCreated = uniqueSortedStrings(current.FilesCreated)
	current.FilesModified = uniqueSortedStrings(current.FilesModified)
	current.TestsWritten = uniqueSortedStrings(current.TestsWritten)
	return current
}
