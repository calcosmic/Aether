package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/events"
)

// swarmRepairCheckpoint.go is Swarm's own thin adapter over the ONE
// checkpoint/restore mechanism Phase 201 built and proved
// (saveRepairCheckpoint/restoreRepairCheckpoint, cmd/work_repair.go). This
// is not a second checkpoint mechanism -- SYN-202-07 (202-CLASSIC-SYNTHESIS.md)
// is explicit that Swarm's repair checkpoint must be the same concept the
// bounded repair round (runBoundedRepairRound) and the direct check-fix
// lane (applyBoundedCheckFixRepair) already use. Every function here either
// calls one of those two primitives directly or renders the plain-English
// announcement/live event around that call -- nothing here re-derives or
// re-implements a snapshot, a restore, or a digest.

// swarmRepairCheckpointIdentity is Swarm's own stable checkpoint identity:
// keyed by the runtime-issued swarm identifier plus a normalized fingerprint
// of the target text, mirroring the exact normalization
// repairCheckpointIdentity (cmd/work_repair.go) already performs on a check
// name (lowercase, trimmed, internal whitespace collapsed to single
// hyphens). The same swarm identifier and target always produce the same
// identity; two genuinely different targets, or two different swarm runs,
// never collide.
func swarmRepairCheckpointIdentity(swarmID, target string) string {
	normalized := strings.ToLower(strings.TrimSpace(target))
	normalized = strings.Join(strings.Fields(normalized), "-")
	return fmt.Sprintf("swarm-repair-checkpoint-%s-%s", strings.TrimSpace(swarmID), normalized)
}

// swarmRepairCheckpointPaths resolves the checkpoint scope for the
// comparison's selected repair: the evidence locations cited by the lenses
// that support that repair (comparison.Selected.Lenses). An empty result
// (no selected repair, or no lens evidence carried a plausible file path)
// falls back to saveRepairCheckpoint's own whole-root behaviour, unchanged.
func swarmRepairCheckpointPaths(comparison swarmComparison) []string {
	if comparison.Selected == nil {
		return nil
	}
	supporting := make(map[string]bool, len(comparison.Selected.Lenses))
	for _, lens := range comparison.Selected.Lenses {
		supporting[lens] = true
	}
	var paths []string
	for _, h := range comparison.Hypotheses {
		if !supporting[h.Lens] {
			continue
		}
		for _, ev := range h.Evidence {
			loc := strings.TrimSpace(ev.Location)
			if loc == "" || strings.ContainsAny(loc, " \t\n") {
				// Not a plausible bare file path (a sentence, a URL with a
				// query, ...) -- skip rather than checkpoint something that
				// was never a path in the first place.
				continue
			}
			paths = append(paths, loc)
		}
	}
	return uniqueSortedStrings(paths)
}

// saveSwarmRepairCheckpoint saves a checkpoint scoped to the selected
// repair's supporting evidence by calling saveRepairCheckpoint directly --
// never a second implementation of the snapshot itself.
func saveSwarmRepairCheckpoint(root, swarmID, target string, comparison swarmComparison) (repairCheckpoint, error) {
	checkpointID := swarmRepairCheckpointIdentity(swarmID, target)
	paths := swarmRepairCheckpointPaths(comparison)
	return saveRepairCheckpoint(root, checkpointID, paths)
}

// restoreSwarmRepairCheckpoint restores exactly the checkpoint
// saveSwarmRepairCheckpoint took, by calling restoreRepairCheckpoint
// directly -- never a second implementation of the restore itself.
func restoreSwarmRepairCheckpoint(checkpoint repairCheckpoint) error {
	return restoreRepairCheckpoint(checkpoint)
}

// announceSwarmCheckpointSaved renders D-10-style plain English naming what
// is being saved and why, then publishes the matching typed live event
// through the existing per-lane helper (emitColonyLiveRecoveryChanged) so
// the live cockpit sees the save at the exact moment it happens -- before
// the fix wave is ever dispatched.
func announceSwarmCheckpointSaved(swarmID, target string) {
	if shouldRenderVisualOutput(stdout) {
		writeVisualOutput(stdout, fmt.Sprintf(
			"Saving your project's current state before trying the top-ranked automatic fix for %q, so it can be put back exactly if the fix does not work.\n",
			strings.TrimSpace(target),
		))
	}
	emitColonyLiveRecoveryChanged(swarmID, events.EpisodeKindSwarm, swarmRecoveryStateCheckpointSaved)
}

// announceSwarmCheckpointRestored renders D-10-style plain English naming
// that the fix did not work and the project has been put back, then
// publishes the matching typed live event -- called only after a failed
// re-verification, and only once the restore itself has genuinely
// succeeded (never on a restore that failed).
func announceSwarmCheckpointRestored(swarmID, target string) {
	if shouldRenderVisualOutput(stdout) {
		writeVisualOutput(stdout, fmt.Sprintf(
			"The automatic fix for %q did not work, so your project has been put back exactly to the state it was saved in.\n",
			strings.TrimSpace(target),
		))
	}
	emitColonyLiveRecoveryChanged(swarmID, events.EpisodeKindSwarm, swarmRecoveryStateCheckpointRestored)
}

// swarmRecoveryStateCheckpointSaved and swarmRecoveryStateCheckpointRestored
// are the RecoveryState values this adapter publishes on the existing
// live.recovery.changed topic (events.LiveTopicRecoveryChanged) -- read
// back by replaying that topic for the run's episode ID.
const (
	swarmRecoveryStateCheckpointSaved    = "checkpoint_saved"
	swarmRecoveryStateCheckpointRestored = "checkpoint_restored"
)

// renderSwarmCheckpointRestoreFailureMessage renders the plain-English
// message for the one case D-05/LIVE-04 requires be told honestly: the
// automatic fix failed verification AND the checkpoint restore itself could
// not put the project back. States plainly that the project was not put
// back, names exactly where the saved copy is, and gives the exact command
// to recover by hand -- and never uses rollback/restore-succeeded language.
func renderSwarmCheckpointRestoreFailureMessage(target string, checkpoint repairCheckpoint, restoreErr error) string {
	return fmt.Sprintf(
		"The automatic fix for %q did not work, and your project could NOT be put back automatically (%v). "+
			"Your project's state from just before the fix was saved at %s -- nothing there has been touched. "+
			"To restore it by hand, copy everything from that folder back over your project: cp -r %s/* %s/",
		strings.TrimSpace(target), restoreErr, checkpoint.BackupDir, checkpoint.BackupDir, checkpoint.Root,
	)
}
