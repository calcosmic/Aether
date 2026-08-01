package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
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
//
// It also rejects two specs naming the same artifact path for different task
// IDs (WR-163.1-03): codexBuildArtifactEvidence carries a single
// ReadOnlyTaskID per path and recordReadOnlyArtifactEvidence is
// last-spec-wins on path replacement, so such a pair is silently
// unsatisfiable -- whichever spec records last erases the other task's
// scope, that task's criterion blocks, and finalize's rerun hint sends the
// operator in a circle flipping the recorded task ID back and forth. Failing
// up front with both specs named is the only honest outcome.
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
	type readOnlyPathScope struct {
		taskID string
		spec   string
	}
	pathScopes := make(map[string]readOnlyPathScope, len(specs))

	for _, spec := range specs {
		taskID, path, err := parseReadOnlyArtifactSpec(spec)
		if err != nil {
			return err
		}
		pathKey := filepath.ToSlash(strings.TrimSpace(path))
		if normalized, normErr := normalizeCriterionArtifactPath(path); normErr == nil {
			pathKey = normalized
		}
		if prior, exists := pathScopes[pathKey]; exists && prior.taskID != taskID {
			return fmt.Errorf("read-only artifact specs %q and %q name the same path %q for different tasks; a path can carry read-only evidence for only one task per run (recording is last-spec-wins, so the other task's criterion would silently block) -- pass the path for a single task only", prior.spec, spec, pathKey)
		}
		pathScopes[pathKey] = readOnlyPathScope{taskID: taskID, spec: spec}
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
//
// CR-163.1-01 guard: the escape hatch exists only for artifacts the build did
// NOT claim (the D-01 premise: "an artifact absent from the task's claim
// lists"). A path that appears in any task's claim lists, or that already
// carries non-read-only (build-time) evidence, is refused outright: re-hashing
// its CURRENT content here would overwrite the build-time hash in
// last-build-claims.json and silently launder any post-build tampering past
// evaluatePhaseCriterionEvidence's current-vs-recorded hash check — the exact
// property T-163.1-46 declares must never be destroyed. Replacing an existing
// ReadOnly entry (a re-run of plan-only with the same spec) remains allowed:
// that is the operator deliberately re-invoking the escape hatch, not evidence
// laundering.
func recordReadOnlyArtifactEvidence(root string, claims *codexBuildClaims, specs []string) error {
	if claims == nil {
		return fmt.Errorf("no claims available to record read-only evidence into")
	}
	claimSets := criterionClaimSets(*claims)
	for _, spec := range specs {
		taskID, path, err := parseReadOnlyArtifactSpec(spec)
		if err != nil {
			return err
		}
		normalized, err := normalizeCriterionArtifactPath(path)
		if err != nil {
			return fmt.Errorf("invalid read-only artifact %q: %w", spec, err)
		}
		if owner, claimed := readOnlyArtifactClaimOwner(claimSets, normalized); claimed {
			return fmt.Errorf("cannot record read-only evidence for %q: artifact %s was claimed by the current build (%s); --read-only-artifact is only for artifacts the build did not claim -- re-recording a claimed artifact would overwrite its build-time hash and destroy post-build tamper detection", spec, normalized, owner)
		}
		for i := range claims.ArtifactEvidence {
			existingPath := filepath.ToSlash(strings.TrimSpace(claims.ArtifactEvidence[i].Path))
			if existingPath == normalized && !claims.ArtifactEvidence[i].ReadOnly {
				return fmt.Errorf("cannot record read-only evidence for %q: artifact %s already has build-time claimed evidence; --read-only-artifact is only for artifacts the build did not claim -- re-recording would overwrite the build-time hash and destroy post-build tamper detection", spec, normalized)
			}
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

// readOnlyArtifactClaimOwner reports whether normalized appears in any of the
// build's claim sets (as built by criterionClaimSets: the flat claim lists
// under "" plus one set per task), and names every owning scope for the
// refusal message so the operator can see whose tamper evidence the spec
// would have overwritten.
func readOnlyArtifactClaimOwner(claimSets map[string]map[string]bool, normalized string) (string, bool) {
	owners := make([]string, 0, 2)
	for taskID, set := range claimSets {
		if !set[normalized] {
			continue
		}
		if taskID == "" {
			owners = append(owners, "the build's claim lists")
		} else {
			owners = append(owners, "task "+taskID)
		}
	}
	if len(owners) == 0 {
		return "", false
	}
	sort.Strings(owners)
	return strings.Join(owners, ", "), true
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

// verifyPlanReadOnlyArtifactEvidence is continue-finalize's loud check that
// --read-only-artifact actually took effect on the external-review path
// (T-163.1-45): it never records evidence itself (that already happened, or
// should have, at `aether continue --plan-only` time via
// applyReadOnlyArtifactEvidence) -- it only confirms every spec on the plan
// manifest already has hash-verified, task-scoped evidence sitting in the
// claims file. Re-recording here would re-hash the artifact's CURRENT
// content and silently overwrite the hash captured at plan-only time,
// destroying tamper detection across the external review window (T-163.1-46)
// -- the one property that makes this escape hatch safe. A manifest whose
// read_only_artifacts were hand-edited in or never actually recorded by an
// operator-invoked plan-only run fails loudly here instead of being trusted.
func verifyPlanReadOnlyArtifactEvidence(root string, phase colony.Phase, plan *codexContinuePlanManifest, manifest codexContinueManifest) error {
	if plan == nil || len(plan.ReadOnlyArtifacts) == 0 {
		return nil
	}
	claims, err := loadCriterionEvidenceClaims(manifest)
	if err != nil {
		return fmt.Errorf("cannot verify read-only artifact evidence: %w", err)
	}
	evidenceByPath := map[string]codexBuildArtifactEvidence{}
	for _, item := range claims.ArtifactEvidence {
		evidenceByPath[filepath.ToSlash(strings.TrimSpace(item.Path))] = item
	}

	missing := make([]string, 0, len(plan.ReadOnlyArtifacts))
	for _, spec := range plan.ReadOnlyArtifacts {
		taskID, path, parseErr := parseReadOnlyArtifactSpec(spec)
		if parseErr != nil {
			return parseErr
		}
		normalized, normalizeErr := normalizeCriterionArtifactPath(path)
		if normalizeErr != nil {
			return fmt.Errorf("invalid read-only artifact %q: %w", spec, normalizeErr)
		}
		recorded, ok := evidenceByPath[normalized]
		if !ok || !recorded.ReadOnly || strings.TrimSpace(recorded.ReadOnlyTaskID) == "" || recorded.ReadOnlyTaskID != taskID {
			missing = append(missing, spec)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	details := make([]string, 0, len(missing))
	for _, spec := range missing {
		taskID, _, _ := parseReadOnlyArtifactSpec(spec)
		details = append(details, fmt.Sprintf(
			"%s (rerun `aether continue --plan-only --reconcile-task %s --read-only-artifact %s`)",
			spec, taskID, spec))
	}
	return fmt.Errorf("read-only artifact evidence missing or not recorded for: %s", strings.Join(details, "; "))
}
