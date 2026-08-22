package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	criterionEvidencePolicyNotRequired   = string(colony.PlanEvidenceNotRequired)
	criterionEvidencePolicyLegacyUnbound = string(colony.PlanEvidenceLegacy)
	criterionEvidencePolicyBoundV1       = string(colony.PlanEvidenceBoundV1)
)

var supportedCriterionChecks = map[string]struct{}{
	"build":   {},
	"types":   {},
	"lint":    {},
	"tests":   {},
	"claims":  {},
	"watcher": {},
}

type codexBuildArtifactEvidence struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
	// ReadOnly and ReadOnlyTaskID are runtime-set only. They are recorded
	// exclusively by the --read-only-artifact reconcile path
	// (recordReadOnlyArtifactEvidence) and must never be accepted from a
	// worker's own completion packet: attachBuildArtifactEvidence replaces
	// claims.ArtifactEvidence wholesale from the claimed file lists, which
	// carry no ReadOnly concept, so any worker-submitted value here is
	// discarded before it can be evaluated (D-01, Pitfall 3, T-163.1-26).
	ReadOnly bool `json:"read_only,omitempty"`
	// ReadOnlyTaskID scopes the bypass to a single task's criteria, keeping
	// task claim sets disjoint (D-02): read-only evidence recorded for one
	// task can never satisfy another task's criterion, and an empty value
	// satisfies nothing.
	ReadOnlyTaskID string `json:"read_only_task_id,omitempty"`
}

type codexCriterionVerification struct {
	Criterion         string   `json:"criterion"`
	TaskID            string   `json:"task_id,omitempty"`
	Policy            string   `json:"policy"`
	Enforced          bool     `json:"enforced"`
	Passed            bool     `json:"passed"`
	RequiredArtifacts []string `json:"required_artifacts,omitempty"`
	RequiredChecks    []string `json:"required_checks,omitempty"`
	Evidence          []string `json:"evidence,omitempty"`
	BlockingIssues    []string `json:"blocking_issues,omitempty"`
	Summary           string   `json:"summary"`
	// State carries a criterion classification beyond plain pass/fail (D-05,
	// 193-CONTEXT.md). Empty for an ordinarily evaluated criterion.
	// criterionStateNeedsOwnerConfirmation (cmd/criterion_owner_confirmation.go)
	// marks a criterion no deterministic source could prove and no reviewer
	// was dispatched to judge -- Passed stays true (the phase still
	// advances) but this field is the separate signal the
	// owner_confirmation_pending gate and `aether seal` read.
	State string `json:"state,omitempty"`
}

type criterionEvidenceEvaluation struct {
	Policy         string
	Enforced       bool
	Passed         bool
	Deterministic  bool
	Criteria       []codexCriterionVerification
	BlockingIssues []string
}

func phaseCriterionEvidencePolicy(phase colony.Phase) string {
	criteriaCount := len(nonEmptyCriteria(phase.SuccessCriteria))
	requirementCount := len(phase.EvidenceRequirements)
	for _, task := range phase.Tasks {
		criteriaCount += len(nonEmptyCriteria(task.SuccessCriteria))
		requirementCount += len(task.EvidenceRequirements)
	}
	if criteriaCount == 0 {
		return criterionEvidencePolicyNotRequired
	}
	if requirementCount == 0 {
		return criterionEvidencePolicyLegacyUnbound
	}
	return criterionEvidencePolicyBoundV1
}

func inferredPlanEvidencePolicy(plan colony.Plan) colony.PlanEvidencePolicy {
	if plan.EvidencePolicy == colony.PlanEvidenceNotRequired || plan.EvidencePolicy == colony.PlanEvidenceLegacy || plan.EvidencePolicy == colony.PlanEvidenceBoundV1 {
		return plan.EvidencePolicy
	}
	if len(plan.Phases) == 0 {
		return colony.PlanEvidenceNotRequired
	}
	foundBound := false
	for _, phase := range plan.Phases {
		switch phaseCriterionEvidencePolicy(phase) {
		case criterionEvidencePolicyLegacyUnbound:
			return colony.PlanEvidenceLegacy
		case criterionEvidencePolicyBoundV1:
			foundBound = true
		}
	}
	if foundBound {
		return colony.PlanEvidenceBoundV1
	}
	return colony.PlanEvidenceNotRequired
}

func validateNewPlanEvidenceContract(phases []colony.Phase) error {
	for _, phase := range phases {
		criteriaCount := len(nonEmptyCriteria(phase.SuccessCriteria))
		for _, task := range phase.Tasks {
			criteriaCount += len(nonEmptyCriteria(task.SuccessCriteria))
		}
		if criteriaCount == 0 {
			return fmt.Errorf("phase %d (%s) has no success criteria; every new phase needs explicit acceptance criteria and evidence bindings", phase.ID, strings.TrimSpace(phase.Name))
		}
		if phaseCriterionEvidencePolicy(phase) != criterionEvidencePolicyBoundV1 {
			return fmt.Errorf("phase %d (%s) has unbound success criteria; every criterion in a newly accepted plan must declare evidence_requirements", phase.ID, strings.TrimSpace(phase.Name))
		}
		if err := validatePhaseCriterionEvidence(phase); err != nil {
			return err
		}
	}
	return nil
}

func bindSyntheticPlanEvidence(phases []colony.Phase) []colony.Phase {
	bound := append([]colony.Phase{}, phases...)
	for phaseIndex := range bound {
		phase := &bound[phaseIndex]
		phase.Tasks = append([]colony.Task{}, phase.Tasks...)
		phase.EvidenceRequirements = syntheticCriterionRequirements(phase.SuccessCriteria)
		for taskIndex := range phase.Tasks {
			phase.Tasks[taskIndex].EvidenceRequirements = syntheticCriterionRequirements(phase.Tasks[taskIndex].SuccessCriteria)
		}
	}
	return bound
}

// syntheticCriterionRequirements derives the default evidence checks for a
// criterion that has no explicit `evidence_requirements` binding. The
// synthetic default IS the deterministic floor (D-06, ruling D11 rule 2): it
// names "claims" plus whichever free check the criterion's own wording
// matches (build/types/lint/tests), and no reviewer caste. A reviewer worker
// is never a synthetic default requirement -- if one is dispatched anyway and
// fails, it still blocks (evaluateCriterionCheckDetail's "watcher" case), but
// nothing here asks for one to exist.
func syntheticCriterionRequirements(criteria []string) []colony.CriterionEvidenceRequirement {
	requirements := make([]colony.CriterionEvidenceRequirement, 0, len(criteria))
	for _, criterion := range nonEmptyCriteria(criteria) {
		lower := strings.ToLower(criterion)
		checks := []string{"claims"}
		switch {
		case strings.Contains(lower, "test") || strings.Contains(lower, "coverage"):
			checks = append(checks, "tests")
		case strings.Contains(lower, "type"):
			checks = append(checks, "types")
		case strings.Contains(lower, "lint") || strings.Contains(lower, "format"):
			checks = append(checks, "lint")
		case strings.Contains(lower, "build") || strings.Contains(lower, "compile") || strings.Contains(lower, "binary"):
			checks = append(checks, "build")
		}
		requirements = append(requirements, colony.CriterionEvidenceRequirement{
			Criterion: criterion,
			Checks:    uniqueSortedStrings(checks),
		})
	}
	return requirements
}

func validatePhaseCriterionEvidence(phase colony.Phase) error {
	criteriaCount := len(nonEmptyCriteria(phase.SuccessCriteria))
	requirementCount := len(phase.EvidenceRequirements)
	for _, task := range phase.Tasks {
		criteriaCount += len(nonEmptyCriteria(task.SuccessCriteria))
		requirementCount += len(task.EvidenceRequirements)
	}
	if requirementCount > 0 && criteriaCount == 0 {
		return fmt.Errorf("phase %d declares criterion evidence requirements but has no success criteria", phase.ID)
	}
	if phaseCriterionEvidencePolicy(phase) != criterionEvidencePolicyBoundV1 {
		return nil
	}

	required := map[string]string{}
	for _, criterion := range nonEmptyCriteria(phase.SuccessCriteria) {
		required[criterionEvidenceKey("", criterion)] = criterion
	}
	for idx, task := range phase.Tasks {
		taskID := buildTaskID(task, idx)
		for _, criterion := range nonEmptyCriteria(task.SuccessCriteria) {
			required[criterionEvidenceKey(taskID, criterion)] = criterion
		}
	}

	seen := map[string]struct{}{}
	for _, requirement := range flattenPhaseCriterionEvidenceRequirements(phase) {
		key := criterionEvidenceKey(requirement.TaskID, requirement.Criterion)
		criterion, ok := required[key]
		if !ok {
			return fmt.Errorf("criterion evidence requirement %q does not match a success criterion in phase %d", strings.TrimSpace(requirement.Criterion), phase.ID)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("criterion %q in phase %d has duplicate evidence requirements", criterion, phase.ID)
		}
		seen[key] = struct{}{}
		if len(requirement.Artifacts) == 0 && len(requirement.Checks) == 0 {
			return fmt.Errorf("criterion %q in phase %d has no artifact or verification check", criterion, phase.ID)
		}
		for _, artifact := range requirement.Artifacts {
			if _, err := normalizeCriterionArtifactPath(artifact); err != nil {
				return fmt.Errorf("criterion %q in phase %d: %w", criterion, phase.ID, err)
			}
		}
		for _, check := range requirement.Checks {
			check = strings.ToLower(strings.TrimSpace(check))
			if _, ok := supportedCriterionChecks[check]; !ok {
				return fmt.Errorf("criterion %q in phase %d uses unsupported verification check %q", criterion, phase.ID, check)
			}
		}
	}
	for key, criterion := range required {
		if _, ok := seen[key]; !ok {
			return fmt.Errorf("criterion %q in phase %d has no evidence requirement; bind every criterion or remove partial evidence bindings", criterion, phase.ID)
		}
	}
	return nil
}

func flattenPhaseCriterionEvidenceRequirements(phase colony.Phase) []colony.CriterionEvidenceRequirement {
	flattened := make([]colony.CriterionEvidenceRequirement, 0, len(phase.EvidenceRequirements))
	for _, requirement := range phase.EvidenceRequirements {
		flattened = append(flattened, normalizedCriterionRequirement(requirement, ""))
	}
	for idx, task := range phase.Tasks {
		taskID := buildTaskID(task, idx)
		for _, requirement := range task.EvidenceRequirements {
			flattened = append(flattened, normalizedCriterionRequirement(requirement, taskID))
		}
	}
	sort.SliceStable(flattened, func(i, j int) bool {
		left := criterionEvidenceKey(flattened[i].TaskID, flattened[i].Criterion)
		right := criterionEvidenceKey(flattened[j].TaskID, flattened[j].Criterion)
		return left < right
	})
	return flattened
}

func normalizedCriterionRequirement(requirement colony.CriterionEvidenceRequirement, taskID string) colony.CriterionEvidenceRequirement {
	artifacts := make([]string, 0, len(requirement.Artifacts))
	for _, artifact := range requirement.Artifacts {
		if normalized, err := normalizeCriterionArtifactPath(artifact); err == nil {
			artifacts = append(artifacts, normalized)
		} else {
			artifacts = append(artifacts, strings.TrimSpace(artifact))
		}
	}
	checks := make([]string, 0, len(requirement.Checks))
	for _, check := range requirement.Checks {
		checks = append(checks, strings.ToLower(strings.TrimSpace(check)))
	}
	return colony.CriterionEvidenceRequirement{
		Criterion: strings.TrimSpace(requirement.Criterion),
		TaskID:    strings.TrimSpace(taskID),
		Artifacts: uniqueSortedStrings(artifacts),
		Checks:    uniqueSortedStrings(checks),
	}
}

func normalizeCriterionArtifactPath(path string) (string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return "", fmt.Errorf("evidence artifact path is empty")
	}
	if filepath.IsAbs(filepath.FromSlash(path)) {
		return "", fmt.Errorf("evidence artifact %q must be repository-relative", path)
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("evidence artifact %q escapes the repository", path)
	}
	if cleaned == ".aether/data" || strings.HasPrefix(cleaned, ".aether/data/") {
		return "", fmt.Errorf("evidence artifact %q points at runtime state, not a project artifact", path)
	}
	return cleaned, nil
}

func attachBuildArtifactEvidence(root string, claims *codexBuildClaims) {
	if claims == nil {
		return
	}
	paths := append(append(append([]string{}, claims.FilesCreated...), claims.FilesModified...), claims.TestsWritten...)
	evidence := make([]codexBuildArtifactEvidence, 0, len(paths))
	for _, path := range uniqueSortedStrings(paths) {
		normalized, err := normalizeCriterionArtifactPath(path)
		if err != nil {
			continue
		}
		item, err := snapshotBuildArtifact(root, normalized)
		if err != nil {
			continue
		}
		evidence = append(evidence, item)
	}
	claims.ArtifactEvidence = evidence
}

func snapshotBuildArtifact(root, rel string) (codexBuildArtifactEvidence, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return codexBuildArtifactEvidence{}, fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return codexBuildArtifactEvidence{}, fmt.Errorf("resolve repository root: %w", err)
	}
	path := filepath.Join(rootPath, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return codexBuildArtifactEvidence{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return codexBuildArtifactEvidence{}, fmt.Errorf("artifact %s is a symbolic link", rel)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return codexBuildArtifactEvidence{}, err
	}
	relToRoot, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return codexBuildArtifactEvidence{}, fmt.Errorf("artifact %s resolves outside the repository", rel)
	}
	if !info.Mode().IsRegular() {
		return codexBuildArtifactEvidence{}, fmt.Errorf("artifact %s is not a regular file", rel)
	}
	file, err := os.Open(resolvedPath)
	if err != nil {
		return codexBuildArtifactEvidence{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return codexBuildArtifactEvidence{}, err
	}
	return codexBuildArtifactEvidence{
		Path:       rel,
		SHA256:     fmt.Sprintf("%x", hash.Sum(nil)),
		Size:       info.Size(),
		ModifiedAt: info.ModTime().UTC().Format(timeRFC3339Nano),
	}, nil
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func criterionEvidenceKey(taskID, criterion string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(criterion), " "))
	return strings.TrimSpace(taskID) + "\x00" + normalized
}

func nonEmptyCriteria(criteria []string) []string {
	result := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		if criterion = strings.TrimSpace(criterion); criterion != "" {
			result = append(result, criterion)
		}
	}
	return result
}

func phaseHasBoundArtifactRequirements(phase colony.Phase) bool {
	if phaseCriterionEvidencePolicy(phase) != criterionEvidencePolicyBoundV1 {
		return false
	}
	for _, requirement := range flattenPhaseCriterionEvidenceRequirements(phase) {
		if len(requirement.Artifacts) > 0 {
			return true
		}
	}
	return false
}

func evaluatePhaseCriterionEvidence(root string, phase colony.Phase, manifest codexContinueManifest, steps []codexVerificationStep, claimsVerification codexClaimVerification, watcher codexWatcherVerification) criterionEvidenceEvaluation {
	policy := phaseCriterionEvidencePolicy(phase)
	evaluation := criterionEvidenceEvaluation{
		Policy:   policy,
		Enforced: policy == criterionEvidencePolicyBoundV1,
		Passed:   policy == criterionEvidencePolicyNotRequired,
	}
	if policy == criterionEvidencePolicyNotRequired {
		return evaluation
	}
	if policy == criterionEvidencePolicyLegacyUnbound {
		evaluation.Criteria = unboundCriterionVerifications(phase)
		return evaluation
	}
	if err := validatePhaseCriterionEvidence(phase); err != nil {
		evaluation.BlockingIssues = []string{err.Error()}
		return evaluation
	}

	requirements := flattenPhaseCriterionEvidenceRequirements(phase)
	if !manifest.Present || manifest.Data.CriterionEvidencePolicy != criterionEvidencePolicyBoundV1 {
		evaluation.BlockingIssues = []string{"build manifest does not contain the phase's bound criterion evidence contract; redispatch the phase before continuing"}
		return evaluation
	}
	if !criterionRequirementsEqual(requirements, manifest.Data.EvidenceRequirements) {
		evaluation.BlockingIssues = []string{"criterion evidence requirements changed after the build manifest was created; redispatch the phase before continuing"}
		return evaluation
	}

	claims, claimsErr := loadCriterionEvidenceClaims(manifest)
	claimSets := criterionClaimSets(claims)
	evidenceByPath := map[string]codexBuildArtifactEvidence{}
	for _, item := range claims.ArtifactEvidence {
		evidenceByPath[filepath.ToSlash(strings.TrimSpace(item.Path))] = item
	}

	// D-04: computed at most once per evaluation, and only if a "claims"
	// check actually needs the fallback (see the loop below) -- re-running a
	// builder's reported shell commands is real work and must not happen on
	// every phase whose claims already verify normally.
	var builderEvidenceCache *builderEvidenceResult
	builderEvidenceOnce := func() builderEvidenceResult {
		if builderEvidenceCache == nil {
			computed := reRunBuilderReportedEvidence(context.Background(), root, phase, effectiveContinueVerificationTimeout(0))
			builderEvidenceCache = &computed
		}
		return *builderEvidenceCache
	}

	evaluation.Passed = true
	for _, requirement := range requirements {
		result := codexCriterionVerification{
			Criterion:         requirement.Criterion,
			TaskID:            requirement.TaskID,
			Policy:            criterionEvidencePolicyBoundV1,
			Enforced:          true,
			Passed:            true,
			RequiredArtifacts: append([]string{}, requirement.Artifacts...),
			RequiredChecks:    append([]string{}, requirement.Checks...),
		}
		claimed := claimSets[requirement.TaskID]
		for _, artifact := range requirement.Artifacts {
			if claimsErr != nil {
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s has no readable current-build claims: %v", artifact, claimsErr))
				continue
			}
			// D-01 (amended): an artifact absent from the task's claim lists
			// can still pass the claimed gate if it has evidence explicitly
			// recorded as read-only. The recording is operator-authorized and
			// hash-verified, and it attests the path's state for this continue
			// run — so it satisfies EVERY requirement naming the path, task-
			// bound or phase-level. Scoping it to one task made a phase whose
			// two tasks both bind the same untouched file unsatisfiable (the
			// recording guard allows one task per path per run). (D-02: claim
			// sets stay disjoint; nothing here merges them.)
			recorded, ok := evidenceByPath[artifact]
			readOnlyMatch := ok && recorded.ReadOnly && strings.TrimSpace(recorded.ReadOnlyTaskID) != ""
			// A task-bound criterion may also verify against an artifact
			// claimed by a different task in the same build: the artifact is
			// hash-recorded at build time either way, so tamper detection is
			// identical. TDD plans routinely bind a later task's criterion to
			// the test file an earlier task wrote.
			buildClaimed := claimSets[""][artifact]
			if !claimed[artifact] && !readOnlyMatch && !buildClaimed {
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s was not claimed by the current build%s", artifact, criterionTaskSuffix(requirement.TaskID)))
				continue
			}
			if !ok || strings.TrimSpace(recorded.SHA256) == "" {
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s has no build-time content hash", artifact))
				continue
			}
			current, err := snapshotBuildArtifact(root, artifact)
			if err != nil {
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s cannot be verified: %v", artifact, err))
				continue
			}
			if current.SHA256 != recorded.SHA256 || current.Size != recorded.Size {
				// The build photographs every artifact it claims, and this
				// compares the photograph to what is on disk now. Its purpose is
				// to catch a change slipped in after the build signed off.
				//
				// But the colony's own reviewers run inside the build, and
				// fixing what they find necessarily lands after finalization.
				// With no allowance for that, Aether refused the work its own
				// Probe and Watcher had just asked for, and the only sanctioned
				// route was re-running most of the phase to re-take the
				// photograph — real cost, no new information.
				//
				// Continue re-runs the repository's real verification commands
				// live before advancing, so a changed artifact whose suite is
				// still green is an amendment, not tampering. A change that
				// breaks the suite still blocks, which is the case the hash
				// existed to catch. The evidence line names it as amended so the
				// substitution is visible rather than silent.
				// Only for artifacts the build actually wrote. A read-only
				// recording is the operator attesting "this file was NOT
				// modified"; if it changed, that attestation is false whatever
				// the test suite says, so it stays a hard block.
				if proof, ok := verificationReRunProvesArtifacts(steps); ok && !readOnlyMatch {
					result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s amended after build evidence; %s", artifact, proof))
					evaluation.Deterministic = true
					continue
				}
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s changed after build evidence was recorded and verification did not re-run green", artifact))
				continue
			}
			if readOnlyMatch {
				result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s sha256:%s (read-only)", artifact, recorded.SHA256))
			} else {
				result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s sha256:%s", artifact, recorded.SHA256))
			}
			evaluation.Deterministic = true
		}
		needsOwnerConfirmation := false
		var ownerConfirmationIssues []string
		for _, check := range requirement.Checks {
			outcome := evaluateCriterionCheckDetail(check, steps, claimsVerification, watcher)
			// D-04: the program's own re-run of what the builder's handoff
			// reported (commands_run re-executed, changed_files confirmed on
			// disk) is an ADDITIONAL deterministic evidence source for the
			// "claims" check -- never a substitute for a check that ran and
			// failed, and never satisfied by the handoff's mere presence
			// (reRunBuilderReportedEvidence only reports what it itself
			// observed). This never widens any other check.
			if !outcome.Passed && strings.EqualFold(strings.TrimSpace(check), "claims") {
				builderEvidence := builderEvidenceOnce()
				if ok, evidence := builderEvidence.satisfied(); ok {
					outcome = criterionCheckOutcome{Passed: true, Evidence: evidence, Deterministic: true}
				} else if detail := builderEvidence.blockingDetail(); detail != "" {
					outcome.Issue = outcome.Issue + "; " + detail
				}
			}
			if outcome.Passed {
				result.Evidence = append(result.Evidence, outcome.Evidence)
				if outcome.Deterministic {
					evaluation.Deterministic = true
				}
				continue
			}
			// D-05: an ABSENCE of any provable source (no reviewer dispatched
			// and no deterministic floor proof for a "watcher"-bound check) is
			// recorded for the owner instead of blocking -- a check that
			// genuinely ran and failed (outcome.AbsentProof stays false) still
			// blocks below, unchanged.
			if outcome.AbsentProof {
				needsOwnerConfirmation = true
				ownerConfirmationIssues = append(ownerConfirmationIssues, outcome.Issue)
				continue
			}
			result.BlockingIssues = append(result.BlockingIssues, outcome.Issue)
		}
		result.Passed = len(result.BlockingIssues) == 0
		switch {
		case !result.Passed:
			result.Summary = "criterion lacks required fresh evidence"
			evaluation.Passed = false
			for _, issue := range result.BlockingIssues {
				evaluation.BlockingIssues = append(evaluation.BlockingIssues, fmt.Sprintf("criterion %q%s: %s", requirement.Criterion, criterionTaskSuffix(requirement.TaskID), issue))
			}
		case needsOwnerConfirmation:
			// The phase still advances (result.Passed stays true): only the
			// State field marks this criterion as needing the owner's
			// confirmation. No worker is dispatched because of it.
			result.State = criterionStateNeedsOwnerConfirmation
			result.Summary = "no deterministic source or dispatched reviewer could prove this criterion; recorded for the owner to confirm"
			result.Evidence = append(result.Evidence, fmt.Sprintf("needs_owner_confirmation: %s", strings.Join(ownerConfirmationIssues, "; ")))
		default:
			result.Summary = "criterion satisfied by fresh bound evidence"
		}
		evaluation.Criteria = append(evaluation.Criteria, result)
	}
	return evaluation
}

func unboundCriterionVerifications(phase colony.Phase) []codexCriterionVerification {
	results := make([]codexCriterionVerification, 0)
	for _, criterion := range nonEmptyCriteria(phase.SuccessCriteria) {
		results = append(results, codexCriterionVerification{
			Criterion: criterion,
			Policy:    criterionEvidencePolicyLegacyUnbound,
			Summary:   "legacy criterion has no explicit artifact or check binding",
		})
	}
	for idx, task := range phase.Tasks {
		for _, criterion := range nonEmptyCriteria(task.SuccessCriteria) {
			results = append(results, codexCriterionVerification{
				Criterion: criterion,
				TaskID:    buildTaskID(task, idx),
				Policy:    criterionEvidencePolicyLegacyUnbound,
				Summary:   "legacy criterion has no explicit artifact or check binding",
			})
		}
	}
	return results
}

func criterionRequirementsEqual(left, right []colony.CriterionEvidenceRequirement) bool {
	normalize := func(requirements []colony.CriterionEvidenceRequirement) []colony.CriterionEvidenceRequirement {
		result := make([]colony.CriterionEvidenceRequirement, 0, len(requirements))
		for _, requirement := range requirements {
			result = append(result, normalizedCriterionRequirement(requirement, requirement.TaskID))
		}
		sort.SliceStable(result, func(i, j int) bool {
			return criterionEvidenceKey(result[i].TaskID, result[i].Criterion) < criterionEvidenceKey(result[j].TaskID, result[j].Criterion)
		})
		return result
	}
	leftJSON, _ := json.Marshal(normalize(left))
	rightJSON, _ := json.Marshal(normalize(right))
	return string(leftJSON) == string(rightJSON)
}

func loadCriterionEvidenceClaims(manifest codexContinueManifest) (codexBuildClaims, error) {
	claimsRel := "last-build-claims.json"
	if manifest.Present && strings.TrimSpace(manifest.Data.ClaimsPath) != "" {
		claimsRel = strings.TrimPrefix(strings.TrimSpace(manifest.Data.ClaimsPath), ".aether/data/")
	}
	var claims codexBuildClaims
	if store == nil {
		return claims, fmt.Errorf("no state store initialized")
	}
	if err := store.LoadJSON(claimsRel, &claims); err != nil {
		return claims, err
	}
	if manifest.Present && manifest.Data.Phase > 0 && claims.BuildPhase != manifest.Data.Phase {
		return claims, fmt.Errorf("claims phase %d does not match manifest phase %d", claims.BuildPhase, manifest.Data.Phase)
	}
	return claims, nil
}

func criterionClaimSets(claims codexBuildClaims) map[string]map[string]bool {
	sets := map[string]map[string]bool{"": {}}
	add := func(target map[string]bool, paths ...[]string) {
		for _, values := range paths {
			for _, path := range values {
				if normalized, err := normalizeCriterionArtifactPath(path); err == nil {
					target[normalized] = true
				}
			}
		}
	}
	add(sets[""], claims.FilesCreated, claims.FilesModified, claims.TestsWritten)
	for _, taskClaim := range claims.TaskClaims {
		taskID := strings.TrimSpace(taskClaim.TaskID)
		if sets[taskID] == nil {
			sets[taskID] = map[string]bool{}
		}
		add(sets[taskID], taskClaim.FilesCreated, taskClaim.FilesModified, taskClaim.TestsWritten)
	}
	return sets
}

// criterionCheckOutcome is the result of evaluating one named check
// (claims/watcher/build/types/lint/tests) against the current verification
// run. Deterministic distinguishes evidence the program produced itself
// (a shell check, a claims re-verify, a re-hashed artifact) from evidence
// that came from a dispatched reviewer's verdict -- the distinction the
// per-criterion loop uses to set criterionEvidenceEvaluation.Deterministic.
type criterionCheckOutcome struct {
	Passed        bool
	Evidence      string
	Issue         string
	Deterministic bool
	// AbsentProof is true only for the "watcher" check's no-dispatch-and-no-
	// deterministic-proof outcome (D-05): the criterion asked for a
	// reviewer's judgment, none was dispatched, and no free check can
	// substitute -- an ABSENCE of any provable source, not a check that ran
	// and failed. evaluatePhaseCriterionEvidence uses this to distinguish
	// "mark needs_owner_confirmation" from "block" -- a dispatched watcher
	// that failed, or any other check that genuinely ran and failed, leaves
	// this false and still blocks.
	AbsentProof bool
}

// evaluateCriterionCheckDetail evaluates a single named check. The "watcher"
// case has three outcomes (D-06, FLOOR-03): a dispatched reviewer that
// passed satisfies it (not deterministic -- a worker's word); a dispatched
// reviewer that did not pass still blocks; and no reviewer dispatched at all
// (or a "skipped" status, which is the same thing in effect) is satisfied
// only when deterministicFloorSatisfies finds genuine proof -- so a criterion
// asking for "watcher" review can still fail with nothing behind it, which is
// the "an always-pass check is worse than no check" precedent this file
// already follows for the removed operational_evidence gate.
func evaluateCriterionCheckDetail(check string, steps []codexVerificationStep, claims codexClaimVerification, watcher codexWatcherVerification) criterionCheckOutcome {
	check = strings.ToLower(strings.TrimSpace(check))
	switch check {
	case "claims":
		if claims.Present && claims.Passed && !claims.Skipped {
			return criterionCheckOutcome{Passed: true, Evidence: "current-build claims verified", Deterministic: true}
		}
		return criterionCheckOutcome{Issue: "current-build claims were missing, skipped, or failed verification"}
	case "watcher":
		dispatchedStatusSkipped := watcher.Present && strings.EqualFold(strings.TrimSpace(watcher.Status), "skipped")
		if watcher.Present && watcher.Passed && !dispatchedStatusSkipped {
			return criterionCheckOutcome{Passed: true, Evidence: fmt.Sprintf("watcher %s passed", firstNonEmpty(watcher.Worker, "verification")), Deterministic: false}
		}
		if watcher.Present && !watcher.Passed && !dispatchedStatusSkipped {
			return criterionCheckOutcome{Issue: "an executed Watcher review did not pass"}
		}
		// No reviewer was dispatched at all, or its status is "skipped" --
		// the check is satisfied only if the deterministic floor genuinely
		// supplies proof.
		if evidence, ok := deterministicFloorSatisfies(steps, claims); ok {
			return criterionCheckOutcome{Passed: true, Evidence: evidence, Deterministic: true}
		}
		return criterionCheckOutcome{
			Issue:       "no reviewer was dispatched and the deterministic floor (verification checks and current-build claims) did not supply proof",
			AbsentProof: true,
		}
	default:
		for _, step := range steps {
			if strings.EqualFold(strings.TrimSpace(step.Name), check) {
				if step.Passed && !step.Skipped {
					// FIELD-03 (191.1-CONTEXT.md D-05): report the verified
					// outcome (step.Summary, e.g. "tests passed"), never the
					// raw configured shell command (step.Command) -- a
					// downstream colony's embedded checker was reporting the
					// check's own definition as if it were the finding.
					return criterionCheckOutcome{Passed: true, Evidence: fmt.Sprintf("%s check passed: %s", check, strings.TrimSpace(step.Summary)), Deterministic: true}
				}
				if step.Skipped {
					return criterionCheckOutcome{Issue: fmt.Sprintf("required %s check was skipped", check)}
				}
				return criterionCheckOutcome{Issue: fmt.Sprintf("required %s check failed: %s", check, step.Summary)}
			}
		}
		return criterionCheckOutcome{Issue: fmt.Sprintf("required %s check was not present", check)}
	}
}

// evaluateCriterionCheck is a thin three-value wrapper around
// evaluateCriterionCheckDetail so cmd/verify_out_of_band.go and every other
// existing caller compiles unchanged.
func evaluateCriterionCheck(check string, steps []codexVerificationStep, claims codexClaimVerification, watcher codexWatcherVerification) (bool, string, string) {
	outcome := evaluateCriterionCheckDetail(check, steps, claims, watcher)
	return outcome.Passed, outcome.Evidence, outcome.Issue
}

// deterministicFloorSatisfies reports whether the program's own checks --
// current-build claims plus verification that actually executed and passed,
// or no verification command resolving for this repository at all (D-01) --
// provide genuine proof for a criterion that would otherwise need a
// dispatched reviewer's verdict. It returns false whenever any step that ran
// did not pass, when any step is Blocked, or when claims failed, so this
// path can still fail -- an always-pass check is worse than no check.
func deterministicFloorSatisfies(steps []codexVerificationStep, claims codexClaimVerification) (string, bool) {
	if !claims.Passed {
		return "", false
	}
	executedAny := false
	for _, step := range steps {
		if step.Blocked {
			return "", false
		}
		if !step.Skipped && !step.Passed {
			return "", false
		}
		if !step.Skipped {
			executedAny = true
		}
	}
	if proof, ok := verificationReRunProvesArtifacts(steps); ok {
		return fmt.Sprintf("no reviewer was dispatched; %s and current-build claims verified", proof), true
	}
	if !executedAny {
		return "no reviewer was dispatched; no verification command resolved in this repository and current-build claims verified", true
	}
	return "", false
}

func criterionTaskSuffix(taskID string) string {
	if strings.TrimSpace(taskID) == "" {
		return ""
	}
	return " for task " + strings.TrimSpace(taskID)
}

// verificationReRunProvesArtifacts reports whether this continue run actually
// executed the repository's verification and found it green, and returns a
// human-readable proof naming the command that ran.
//
// "Actually executed" is the load-bearing word. runVerificationStep marks an
// optional step Passed:true when no command could be resolved for it — a
// convenience so a missing linter cannot block a phase. Trusting Passed alone
// would therefore accept an amendment on the strength of three checks that
// never ran, which is precisely the substitution this function exists to
// prevent. A step only counts if it carried a command, was not skipped, and
// exited clean; and any failing or blocked step disqualifies the whole run.
func verificationReRunProvesArtifacts(steps []codexVerificationStep) (string, bool) {
	executed := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.Blocked || (!step.Skipped && !step.Passed) {
			return "", false
		}
		if step.Skipped || strings.TrimSpace(step.Command) == "" || !step.Passed {
			continue
		}
		executed = append(executed, step.Name)
	}
	if len(executed) == 0 {
		return "", false
	}
	return fmt.Sprintf("verification re-ran green (%s)", strings.Join(executed, ", ")), true
}

// builderCommandRerunResult is the program's own outcome from re-executing
// one command a builder's persisted worker handoff (pkg/codex.WorkerHandoff,
// via workerHandoffRecord) reported having run (commands_run). The worker's
// own claim that it ran the command and it passed is never trusted directly
// -- only this struct's Passed/Unresolvable fields, set from the program's
// own re-execution, count as evidence (D-04).
type builderCommandRerunResult struct {
	TaskID       string
	Command      string
	Passed       bool
	Unresolvable bool
	Summary      string
}

// builderFileCheckResult is the program's own existence check, right now,
// for one file a builder's handoff reported having changed (changed_files).
type builderFileCheckResult struct {
	TaskID string
	Path   string
	Exists bool
}

// builderEvidenceResult is reRunBuilderReportedEvidence's outcome: exactly
// what the program itself re-ran and re-checked from this phase's already-
// persisted worker handoffs. It creates no build dispatch, no claims record,
// and no reviewer verdict -- see TestEvidenceReRunNeverFabricatesAWorkerReceipt.
type builderEvidenceResult struct {
	Commands []builderCommandRerunResult
	Files    []builderFileCheckResult
}

// satisfied reports whether this phase's builder-reported evidence, re-run
// and re-checked by the program itself, proves the "claims" check: at least
// one reported command was actually re-executed (not merely unresolvable in
// this repository) and passed, no re-executed command failed, and every
// reported changed file exists on disk right now. An empty handoff (nothing
// recorded) proves nothing either way.
func (r builderEvidenceResult) satisfied() (bool, string) {
	ranAndPassed := make([]string, 0, len(r.Commands))
	for _, c := range r.Commands {
		if c.Unresolvable {
			continue
		}
		if !c.Passed {
			return false, ""
		}
		ranAndPassed = append(ranAndPassed, c.Command)
	}
	if len(ranAndPassed) == 0 {
		return false, ""
	}
	for _, f := range r.Files {
		if !f.Exists {
			return false, ""
		}
	}
	return true, fmt.Sprintf("program re-ran the builder-reported command(s) itself (%s) and confirmed the builder-reported changed file(s) exist", strings.Join(uniqueSortedStrings(ranAndPassed), ", "))
}

// blockingDetail names the first concrete problem the re-run found -- a
// command that genuinely ran and failed, or a reported changed file that
// does not exist -- so the criterion's blocking issue names something real
// rather than just "claims were missing". Returns "" when there is nothing
// substantive to add (no handoffs, or nothing failed).
func (r builderEvidenceResult) blockingDetail() string {
	for _, c := range r.Commands {
		if !c.Unresolvable && !c.Passed {
			return fmt.Sprintf("builder-reported command re-run failed: %s (%s)", c.Command, c.Summary)
		}
	}
	for _, f := range r.Files {
		if !f.Exists {
			return fmt.Sprintf("builder-reported changed file does not exist on disk: %s", f.Path)
		}
	}
	return ""
}

// reRunBuilderReportedEvidence re-runs, itself, what a builder's persisted
// handoff for this phase reported having run (commands_run), and confirms on
// disk right now that every file it reported having changed (changed_files)
// actually exists. It never trusts the handoff's own word -- only what the
// program itself observes counts (D-04, ruling D11: "the worker's word alone
// never satisfies a criterion"). It reads the phase's already-persisted
// worker handoffs (loadWorkerHandoffRecords) and re-executes each reported
// command through the same primitive runVerificationStep uses elsewhere; it
// creates no build dispatch, no claims record, and no reviewer verdict.
// TestEvidenceReRunNeverFabricatesAWorkerReceipt asserts this directly.
func reRunBuilderReportedEvidence(ctx context.Context, root string, phase colony.Phase, timeout time.Duration) builderEvidenceResult {
	if ctx == nil {
		ctx = context.Background()
	}
	result := builderEvidenceResult{}
	records, err := loadWorkerHandoffRecords()
	if err != nil || len(records) == 0 {
		return result
	}
	timeout = effectiveContinueVerificationTimeout(timeout)
	seenCommands := map[string]struct{}{}
	seenFiles := map[string]struct{}{}
	for _, record := range records {
		if record.Phase != phase.ID {
			continue
		}
		for _, command := range record.CommandsRun {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}
			key := record.TaskID + "\x00" + command
			if _, dup := seenCommands[key]; dup {
				continue
			}
			seenCommands[key] = struct{}{}
			// CR-01 (193-REVIEW.md): command is a builder's own self-reported
			// text -- an LLM agent's output that may itself have been
			// influenced by content it read during the phase. It is never
			// passed to a shell (runVerificationStep/runShellCommandContext,
			// which use sh -c, are for THIS PROJECT'S OWN resolved
			// build/test/lint commands, not attacker- or LLM-influenced
			// strings). reRunOneBuilderCommand refuses and records as
			// Unresolvable anything that is not a plain, argv-shaped
			// build/test runner invocation.
			passed, unresolvable, summary := reRunOneBuilderCommand(ctx, root, command, timeout)
			result.Commands = append(result.Commands, builderCommandRerunResult{
				TaskID:       record.TaskID,
				Command:      command,
				Passed:       passed,
				Unresolvable: unresolvable,
				Summary:      summary,
			})
		}
		for _, path := range record.ChangedFiles {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			key := record.TaskID + "\x00" + path
			if _, dup := seenFiles[key]; dup {
				continue
			}
			seenFiles[key] = struct{}{}
			exists := false
			if normalized, normErr := normalizeCriterionArtifactPath(path); normErr == nil {
				if _, statErr := snapshotBuildArtifact(root, normalized); statErr == nil {
					exists = true
				}
			}
			result.Files = append(result.Files, builderFileCheckResult{TaskID: record.TaskID, Path: path, Exists: exists})
		}
	}
	return result
}

// builderReportedCommandRunners lists the first-token build/test runners
// this program trusts enough to actually execute when a builder
// self-reports having run them (CR-01, 193-REVIEW.md). This is
// deliberately narrower than looksLikeVerificationCommand
// (cmd/codex_continue.go), which also treats echo/printf/true/false/sh/
// bash as "verification-shaped" for display/classification purposes only
// -- none of those belong in a list that leads to real execution of
// builder-authored text, because every one of them is a way to run
// something else (echo/printf can carry payloads a caller pipes
// elsewhere, true/false are no-ops that prove nothing, sh/bash are a
// shell in disguise).
var builderReportedCommandRunners = map[string]bool{
	"go": true, "npm": true, "npx": true, "pnpm": true, "yarn": true, "bun": true,
	"cargo": true, "pytest": true, "python": true, "python3": true, "uv": true,
	"make": true, "mvn": true, "gradle": true, "dotnet": true,
	"golangci-lint": true, "ruff": true, "pyright": true, "mypy": true,
}

// builderReportedCommandMetacharacters are shell metacharacters that must
// never appear in a builder-reported command this program is about to
// execute itself. Their presence means the string is not a plain,
// argv-shaped runner invocation -- it is an attempt to chain commands,
// substitute output, redirect, or otherwise reach a shell (CR-01,
// 193-REVIEW.md). reRunOneBuilderCommand never runs a command through
// sh -c, but this check is defense in depth: it also blocks a string like
// "go test ./..." from resolving to a runner and then silently carrying a
// metacharacter-laden argument no test runner would ever need.
const builderReportedCommandMetacharacters = ";&|$`><(){}\n\r"

// commandSafeToReRun reports whether a builder-reported command is safe
// for this program to execute itself: its first token names a recognised
// build/test/lint runner, and the string contains none of the shell
// metacharacters that would let it do anything beyond invoking that
// runner with plain arguments.
func commandSafeToReRun(command string) bool {
	if strings.ContainsAny(command, builderReportedCommandMetacharacters) {
		return false
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	return builderReportedCommandRunners[fields[0]]
}

// reRunOneBuilderCommand re-executes ONE builder-reported command itself,
// via argv (exec.CommandContext with the command's own fields, never a
// shell), and only when commandSafeToReRun allows it. CR-01
// (193-REVIEW.md): a builder's self-reported commands_run text is
// untrusted input -- an LLM agent's own output, possibly influenced by
// content it read during the phase -- and must never reach sh -c the way
// this project's own resolved verification commands do
// (runVerificationStep/runShellCommandContext). A refused command is
// always reported unresolvable, never silently dropped and never counted
// as a pass.
func reRunOneBuilderCommand(ctx context.Context, root, command string, timeout time.Duration) (passed, unresolvable bool, summary string) {
	if !commandSafeToReRun(command) {
		return false, true, fmt.Sprintf("%q is not a recognised build/test runner command (or contains shell metacharacters); the program refused to execute it", command)
	}
	fields := strings.Fields(command)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, fields[0], fields[1:]...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "AETHER_OUTPUT_MODE=")
	output, err := cmd.CombinedOutput()
	trimmed := trimCommandOutput(string(output))
	if err != nil {
		if cmd.ProcessState == nil {
			// The runner binary itself could not be started (not on PATH
			// in this environment) -- that is this program's environment
			// lacking the tool, not the builder-reported command genuinely
			// failing.
			return false, true, fmt.Sprintf("%s: command unavailable in this repository; skipped", command)
		}
		exitCode := cmd.ProcessState.ExitCode()
		if isCommandUnresolvable(trimmed, exitCode) {
			return false, true, fmt.Sprintf("%s: command unavailable in this repository (exit %d); skipped", command, exitCode)
		}
		return false, false, fmt.Sprintf("builder-reported command re-run failed: %s (exit %d)", command, exitCode)
	}
	return true, false, fmt.Sprintf("program re-ran the builder-reported command itself (%s) and it passed", command)
}
