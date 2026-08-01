package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

func syntheticCriterionRequirements(criteria []string) []colony.CriterionEvidenceRequirement {
	requirements := make([]colony.CriterionEvidenceRequirement, 0, len(criteria))
	for _, criterion := range nonEmptyCriteria(criteria) {
		lower := strings.ToLower(criterion)
		checks := []string{"claims", "watcher"}
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
			// D-01: an artifact absent from the task's claim lists can still
			// pass the claimed gate if it has evidence explicitly recorded
			// as read-only and scoped to this exact requirement's task. An
			// empty ReadOnlyTaskID never matches, and a match for a
			// different task never matches (D-02: claim sets stay disjoint).
			recorded, ok := evidenceByPath[artifact]
			readOnlyMatch := ok && recorded.ReadOnly && strings.TrimSpace(recorded.ReadOnlyTaskID) != "" && recorded.ReadOnlyTaskID == requirement.TaskID
			if !claimed[artifact] && !readOnlyMatch {
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
				result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("artifact %s changed after build evidence was recorded", artifact))
				continue
			}
			if readOnlyMatch {
				result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s sha256:%s (read-only)", artifact, recorded.SHA256))
			} else {
				result.Evidence = append(result.Evidence, fmt.Sprintf("artifact %s sha256:%s", artifact, recorded.SHA256))
			}
			evaluation.Deterministic = true
		}
		for _, check := range requirement.Checks {
			passed, evidence, issue := evaluateCriterionCheck(check, steps, claimsVerification, watcher)
			if passed {
				result.Evidence = append(result.Evidence, evidence)
				if check != "watcher" {
					evaluation.Deterministic = true
				}
				continue
			}
			result.BlockingIssues = append(result.BlockingIssues, issue)
		}
		result.Passed = len(result.BlockingIssues) == 0
		if result.Passed {
			result.Summary = "criterion satisfied by fresh bound evidence"
		} else {
			result.Summary = "criterion lacks required fresh evidence"
			evaluation.Passed = false
			for _, issue := range result.BlockingIssues {
				evaluation.BlockingIssues = append(evaluation.BlockingIssues, fmt.Sprintf("criterion %q%s: %s", requirement.Criterion, criterionTaskSuffix(requirement.TaskID), issue))
			}
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

func evaluateCriterionCheck(check string, steps []codexVerificationStep, claims codexClaimVerification, watcher codexWatcherVerification) (bool, string, string) {
	check = strings.ToLower(strings.TrimSpace(check))
	switch check {
	case "claims":
		if claims.Present && claims.Passed && !claims.Skipped {
			return true, "current-build claims verified", ""
		}
		return false, "", "current-build claims were missing, skipped, or failed verification"
	case "watcher":
		if watcher.Present && watcher.Passed && !strings.EqualFold(strings.TrimSpace(watcher.Status), "skipped") {
			return true, fmt.Sprintf("watcher %s passed", firstNonEmpty(watcher.Worker, "verification")), ""
		}
		return false, "", "an executed Watcher review did not pass"
	default:
		for _, step := range steps {
			if strings.EqualFold(strings.TrimSpace(step.Name), check) {
				if step.Passed && !step.Skipped {
					return true, fmt.Sprintf("%s check passed: %s", check, strings.TrimSpace(step.Command)), ""
				}
				if step.Skipped {
					return false, "", fmt.Sprintf("required %s check was skipped", check)
				}
				return false, "", fmt.Sprintf("required %s check failed: %s", check, step.Summary)
			}
		}
		return false, "", fmt.Sprintf("required %s check was not present", check)
	}
}

func criterionTaskSuffix(taskID string) string {
	if strings.TrimSpace(taskID) == "" {
		return ""
	}
	return " for task " + strings.TrimSpace(taskID)
}
