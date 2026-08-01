package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// parseReadOnlyArtifactSpec splits a --read-only-artifact value of the form
// "<task-id>:<path>" on its first ":" separator. Both halves must be
// non-empty after trimming, and the separator must be present.
func parseReadOnlyArtifactSpec(spec string) (taskID, path string, err error) {
	trimmed := strings.TrimSpace(spec)
	idx := strings.Index(trimmed, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid read-only artifact %q: expected the form <task-id>:<path>", spec)
	}
	taskID = strings.TrimSpace(trimmed[:idx])
	path = strings.TrimSpace(trimmed[idx+1:])
	if taskID == "" || path == "" {
		return "", "", fmt.Errorf("invalid read-only artifact %q: expected the form <task-id>:<path>", spec)
	}
	return taskID, path, nil
}

// validateReadOnlyArtifacts rejects a --read-only-artifact spec whose task ID
// is not present in the phase (same error shape as validateContinueReconcileTasks),
// or whose task ID was not also passed to --reconcile-task. Read-only evidence
// is an escape hatch for a task already being reconciled, not a standalone
// gate bypass (T-163.1-31).
func validateReadOnlyArtifacts(phase colony.Phase, reconcileTaskIDs, specs []string) error {
	if len(specs) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(phase.Tasks))
	for idx, task := range phase.Tasks {
		known[buildTaskID(task, idx)] = struct{}{}
	}
	reconciled := make(map[string]struct{}, len(reconcileTaskIDs))
	for _, taskID := range reconcileTaskIDs {
		reconciled[strings.TrimSpace(taskID)] = struct{}{}
	}

	unknown := make([]string, 0, len(specs))
	seenUnknown := map[string]struct{}{}
	notReconciled := make([]string, 0, len(specs))
	seenNotReconciled := map[string]struct{}{}

	for _, spec := range specs {
		taskID, _, err := parseReadOnlyArtifactSpec(spec)
		if err != nil {
			return err
		}
		if _, ok := known[taskID]; !ok {
			if _, dup := seenUnknown[taskID]; !dup {
				seenUnknown[taskID] = struct{}{}
				unknown = append(unknown, taskID)
			}
			continue
		}
		if _, ok := reconciled[taskID]; !ok {
			if _, dup := seenNotReconciled[taskID]; !dup {
				seenNotReconciled[taskID] = struct{}{}
				notReconciled = append(notReconciled, taskID)
			}
		}
	}

	if len(unknown) > 0 {
		return fmt.Errorf("unknown task id(s) for phase %d: %s", phase.ID, strings.Join(unknown, ", "))
	}
	if len(notReconciled) > 0 {
		return fmt.Errorf("--read-only-artifact task id(s) must also be passed to --reconcile-task: %s", strings.Join(notReconciled, ", "))
	}
	return nil
}

// recordReadOnlyArtifactEvidence records hash-verified evidence for each spec
// into claims.ArtifactEvidence, marked ReadOnly and scoped to the spec's task
// ID. It reuses normalizeCriterionArtifactPath (rejects "..", absolute, and
// .aether/data paths) and snapshotBuildArtifact (rejects symlinks) — the same
// integrity machinery claimed artifacts already go through — so no hashing or
// path validation is reimplemented here. It never calls
// attachBuildArtifactEvidence, which would wipe these entries by rebuilding
// ArtifactEvidence solely from the claimed file lists.
func recordReadOnlyArtifactEvidence(root string, claims *codexBuildClaims, specs []string) error {
	if claims == nil {
		return fmt.Errorf("no claims available to record read-only evidence into")
	}
	for _, spec := range specs {
		taskID, path, err := parseReadOnlyArtifactSpec(spec)
		if err != nil {
			return err
		}
		normalized, err := normalizeCriterionArtifactPath(path)
		if err != nil {
			return fmt.Errorf("invalid read-only artifact %q: %w", spec, err)
		}
		evidence, err := snapshotBuildArtifact(root, normalized)
		if err != nil {
			return fmt.Errorf("cannot record read-only evidence for %q: %w", spec, err)
		}
		evidence.ReadOnly = true
		evidence.ReadOnlyTaskID = taskID

		replaced := false
		for i := range claims.ArtifactEvidence {
			if claims.ArtifactEvidence[i].Path == evidence.Path {
				claims.ArtifactEvidence[i] = evidence
				replaced = true
				break
			}
		}
		if !replaced {
			claims.ArtifactEvidence = append(claims.ArtifactEvidence, evidence)
		}
	}
	return nil
}

// applyReadOnlyArtifactEvidence loads the current build claims named by
// manifest, records read-only evidence for specs via
// recordReadOnlyArtifactEvidence, and persists the result back to the same
// claims path loadCriterionEvidenceClaims reads from — so criterion
// evaluation later in the same continue invocation sees the recorded
// evidence.
func applyReadOnlyArtifactEvidence(root string, manifest codexContinueManifest, specs []string) error {
	if len(specs) == 0 {
		return nil
	}
	claims, err := loadCriterionEvidenceClaims(manifest)
	if err != nil {
		return fmt.Errorf("cannot record read-only evidence: %w", err)
	}
	if err := recordReadOnlyArtifactEvidence(root, &claims, specs); err != nil {
		return err
	}
	claimsRel := "last-build-claims.json"
	if manifest.Present && strings.TrimSpace(manifest.Data.ClaimsPath) != "" {
		claimsRel = strings.TrimPrefix(strings.TrimSpace(manifest.Data.ClaimsPath), ".aether/data/")
	}
	if store == nil {
		return fmt.Errorf("no state store initialized")
	}
	return store.SaveJSON(claimsRel, claims)
}
