package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const planRevisionSchemaVersion = 1

type codexPlanRevisionContext struct {
	BaseRevisionID    string                    `json:"base_revision_id,omitempty"`
	BasePlanStateHash string                    `json:"base_plan_state_hash"`
	ReasonType        colony.PlanRevisionReason `json:"reason_type"`
	Reason            string                    `json:"reason"`
	Evidence          []string                  `json:"evidence,omitempty"`
	EvidenceHash      string                    `json:"evidence_hash,omitempty"`
	CompletedPhases   []colony.Phase            `json:"completed_phases,omitempty"`
	SupersededPhases  []colony.Phase            `json:"superseded_phases,omitempty"`
}

func planStateHash(plan colony.Plan) (string, error) {
	return jsonSHA256(plan)
}

func activePlanRevisionID(plan colony.Plan) string {
	if id := strings.TrimSpace(plan.ActiveRevisionID); id != "" {
		return id
	}
	if len(plan.Phases) == 0 {
		return ""
	}
	hash, err := planDefinitionHash(plan.Phases)
	if err != nil || len(hash) < 12 {
		return "legacy-plan"
	}
	return "legacy-" + hash[:12]
}

func planDefinitionHash(phases []colony.Phase) (string, error) {
	definitions := clonePhases(phases)
	for i := range definitions {
		definitions[i].Status = ""
		definitions[i].WatcherFailureCount = 0
		for j := range definitions[i].Tasks {
			definitions[i].Tasks[j].Status = ""
		}
	}
	return jsonSHA256(definitions)
}

func normalizePlanRevisionReason(raw string) (colony.PlanRevisionReason, error) {
	reason := colony.PlanRevisionReason(strings.ToLower(strings.TrimSpace(raw)))
	if reason == "" {
		reason = colony.PlanRevisionManual
	}
	if !reason.Valid() || reason == colony.PlanRevisionInitial || reason == colony.PlanRevisionLegacyImport {
		return "", fmt.Errorf("revision type %q is invalid; use manual, user_feedback, research, verification_failure, or scope_change", raw)
	}
	return reason, nil
}

func completedPlanPrefix(phases []colony.Phase) (int, error) {
	prefix := 0
	seenIncomplete := false
	for _, phase := range phases {
		completed := phase.Status == colony.PhaseCompleted
		if completed && seenIncomplete {
			return 0, fmt.Errorf("cannot revise an inconsistent plan: completed phase %d appears after unfinished work", phase.ID)
		}
		if completed {
			prefix++
			continue
		}
		seenIncomplete = true
	}
	return prefix, nil
}

func buildPlanRevisionContext(root string, state colony.ColonyState, opts codexPlanOptions) (*codexPlanRevisionContext, error) {
	if !opts.Refresh {
		if strings.TrimSpace(opts.RevisionReason) != "" || strings.TrimSpace(opts.RevisionType) != "" || len(opts.RevisionEvidence) > 0 {
			return nil, fmt.Errorf("plan revision metadata requires --refresh")
		}
		return nil, nil
	}
	if len(state.Plan.Phases) == 0 {
		return nil, nil
	}

	prefix, err := completedPlanPrefix(state.Plan.Phases)
	if err != nil {
		return nil, err
	}
	if prefix == len(state.Plan.Phases) {
		return nil, fmt.Errorf("all plan phases are already completed; seal this colony or start a new goal instead of revising it")
	}
	if state.State == colony.StateEXECUTING {
		if _, attempt, ok := loadLatestBuildAttempt(state.CurrentPhase); ok && buildAttemptStatusActive(attempt.Status) {
			return nil, fmt.Errorf("cannot revise while build attempt %s is active for phase %d; finish or recover that attempt first", attempt.ID, state.CurrentPhase)
		}
	}

	reasonType, err := normalizePlanRevisionReason(opts.RevisionType)
	if err != nil {
		return nil, err
	}
	reason := strings.Join(strings.Fields(strings.TrimSpace(opts.RevisionReason)), " ")
	if prefix > 0 && reason == "" {
		return nil, fmt.Errorf("--revision-reason is required when revising a plan with completed phases")
	}
	if reason == "" {
		reason = "Plan refresh requested before any phase was completed"
	}
	evidence, err := validatePlanRevisionEvidence(root, opts.RevisionEvidence)
	if err != nil {
		return nil, err
	}
	if (reasonType == colony.PlanRevisionResearch || reasonType == colony.PlanRevisionVerificationFailure) && len(evidence) == 0 {
		return nil, fmt.Errorf("revision type %s requires at least one --revision-evidence path", reasonType)
	}
	evidenceHash, err := planRevisionInputEvidenceHash(root, evidence)
	if err != nil {
		return nil, err
	}
	baseHash, err := planStateHash(state.Plan)
	if err != nil {
		return nil, fmt.Errorf("hash current plan before revision: %w", err)
	}
	return &codexPlanRevisionContext{
		BaseRevisionID:    activePlanRevisionID(state.Plan),
		BasePlanStateHash: baseHash,
		ReasonType:        reasonType,
		Reason:            reason,
		Evidence:          evidence,
		EvidenceHash:      evidenceHash,
		CompletedPhases:   clonePhases(state.Plan.Phases[:prefix]),
		SupersededPhases:  clonePhases(state.Plan.Phases[prefix:]),
	}, nil
}

func planRevisionInputEvidenceHash(root string, evidence []string) (string, error) {
	if len(evidence) == 0 {
		return "", nil
	}
	type evidenceFingerprint struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	}
	fingerprints := make([]evidenceFingerprint, 0, len(evidence))
	for _, rel := range evidence {
		path := filepath.Join(root, filepath.FromSlash(rel))
		hash, err := fileSHA256(path)
		if err != nil {
			return "", fmt.Errorf("hash revision evidence %s: %w", rel, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat revision evidence %s: %w", rel, err)
		}
		fingerprints = append(fingerprints, evidenceFingerprint{Path: rel, SHA256: hash, Size: info.Size()})
	}
	hash, err := jsonSHA256(fingerprints)
	if err != nil {
		return "", fmt.Errorf("hash revision evidence manifest: %w", err)
	}
	return hash, nil
}

func validatePlanRevisionEvidence(root string, values []string) ([]string, error) {
	result := uniqueSortedStrings(values)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root for revision evidence: %w", err)
	}
	for _, rel := range result {
		if filepath.IsAbs(rel) {
			return nil, fmt.Errorf("revision evidence must be repository-relative: %s", rel)
		}
		clean := filepath.Clean(filepath.FromSlash(rel))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("revision evidence escapes the repository: %s", rel)
		}
		path := filepath.Join(root, clean)
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("revision evidence %s is unavailable: %w", rel, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("revision evidence must name a file: %s", rel)
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("resolve revision evidence %s: %w", rel, err)
		}
		relToRoot, err := filepath.Rel(realRoot, realPath)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("revision evidence resolves outside the repository: %s", rel)
		}
	}
	return result, nil
}

func validatePlanManifestBase(root string, manifest codexPlanManifest, state colony.ColonyState) error {
	if strings.TrimSpace(manifest.BasePlanStateHash) == "" {
		return fmt.Errorf("plan_manifest base_plan_state_hash is required")
	}
	currentHash, err := planStateHash(state.Plan)
	if err != nil {
		return fmt.Errorf("hash current plan: %w", err)
	}
	if currentHash != strings.TrimSpace(manifest.BasePlanStateHash) {
		return fmt.Errorf("plan state changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
	}
	if strings.TrimSpace(manifest.BaseRevisionID) != activePlanRevisionID(state.Plan) {
		return fmt.Errorf("plan revision changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
	}
	if manifest.Refresh && len(state.Plan.Phases) > 0 {
		if manifest.Revision == nil {
			return fmt.Errorf("plan_manifest revision context is required when refreshing an existing plan")
		}
		if manifest.Revision.BasePlanStateHash != currentHash || strings.TrimSpace(manifest.Revision.BaseRevisionID) != activePlanRevisionID(state.Plan) {
			return fmt.Errorf("plan_manifest revision context no longer matches the active plan")
		}
		currentEvidenceHash, err := planRevisionInputEvidenceHash(root, manifest.Revision.Evidence)
		if err != nil {
			return err
		}
		if currentEvidenceHash != strings.TrimSpace(manifest.Revision.EvidenceHash) {
			return fmt.Errorf("plan revision evidence changed after this planning packet was created; discard the stale packet and rerun `aether plan --plan-only`")
		}
	}
	return nil
}

func activateGeneratedPlan(previous colony.Plan, candidate []colony.Phase, generatedAt time.Time, confidence *float64, evidencePolicy colony.PlanEvidencePolicy, manifest codexPlanManifest, evidenceHash string) (colony.Plan, colony.PlanRevision, error) {
	if len(candidate) == 0 {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("generated plan contains no phases")
	}
	if !manifest.Refresh || len(previous.Phases) == 0 {
		phases := renumberRevisionPhases(candidate, 0)
		if err := validateUniquePlanTaskIDs(phases); err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		if err := colony.DetectCycles(phases); err != nil {
			return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("accepted plan dependency validation failed: %w", err)
		}
		plan := colony.Plan{
			GeneratedAt:    &generatedAt,
			Confidence:     confidence,
			EvidencePolicy: evidencePolicy,
			Phases:         phases,
		}
		revision, err := newPlanRevision(plan, 1, "", colony.PlanRevisionInitial, "Initial plan generated", []string{filepath.ToSlash(filepath.Join(".aether", "data", "planning", "phase-plan.json"))}, "", evidenceHash, manifest.PlanningRunID, nil, nil, phaseIDs(phases), generatedAt)
		if err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		plan.ActiveRevisionID = revision.ID
		plan.Revisions = []colony.PlanRevision{revision}
		return plan, revision, nil
	}

	prefix, err := completedPlanPrefix(previous.Phases)
	if err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	if prefix == len(previous.Phases) {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("cannot replace a fully completed plan")
	}
	preserved := clonePhases(previous.Phases[:prefix])
	replacements := renumberRevisionPhases(candidate, prefix)
	combined := append(preserved, replacements...)
	if err := validateUniquePlanTaskIDs(combined); err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	if err := colony.DetectCycles(combined); err != nil {
		return colony.Plan{}, colony.PlanRevision{}, fmt.Errorf("revised plan dependency validation failed: %w", err)
	}

	revisions := append([]colony.PlanRevision{}, previous.Revisions...)
	parentID := activePlanRevisionID(previous)
	if len(revisions) == 0 {
		baseline, err := newPlanRevision(previous, 1, "", colony.PlanRevisionLegacyImport, "Existing plan imported when revision tracking was enabled", nil, "", "", "", nil, nil, phaseIDs(previous.Phases), revisionCreatedAt(previous, generatedAt))
		if err != nil {
			return colony.Plan{}, colony.PlanRevision{}, err
		}
		revisions = append(revisions, baseline)
		parentID = baseline.ID
	}
	number := nextPlanRevisionNumber(revisions)
	plan := colony.Plan{
		GeneratedAt:      &generatedAt,
		Confidence:       confidence,
		EvidencePolicy:   evidencePolicy,
		ActiveRevisionID: parentID,
		Revisions:        revisions,
		Phases:           combined,
	}
	reasonType := colony.PlanRevisionManual
	reason := "Plan refreshed"
	var evidence []string
	if manifest.Revision != nil {
		reasonType = manifest.Revision.ReasonType
		reason = manifest.Revision.Reason
		evidence = append([]string{}, manifest.Revision.Evidence...)
	}
	inputEvidenceHash := ""
	if manifest.Revision != nil {
		inputEvidenceHash = manifest.Revision.EvidenceHash
	}
	revision, err := newPlanRevision(plan, number, parentID, reasonType, reason, evidence, inputEvidenceHash, evidenceHash, manifest.PlanningRunID, phaseIDs(preserved), phaseIDs(previous.Phases[prefix:]), phaseIDs(replacements), generatedAt)
	if err != nil {
		return colony.Plan{}, colony.PlanRevision{}, err
	}
	plan.ActiveRevisionID = revision.ID
	plan.Revisions = append(plan.Revisions, revision)
	return plan, revision, nil
}

func newPlanRevision(plan colony.Plan, number int, parentID string, reasonType colony.PlanRevisionReason, reason string, evidence []string, inputEvidenceHash, evidenceHash, planningRunID string, preserved, superseded, replacements []int, createdAt time.Time) (colony.PlanRevision, error) {
	hash, err := planDefinitionHash(plan.Phases)
	if err != nil {
		return colony.PlanRevision{}, fmt.Errorf("hash accepted plan revision: %w", err)
	}
	id := fmt.Sprintf("plan-r%d-%s", number, hash[:12])
	return colony.PlanRevision{
		SchemaVersion:       planRevisionSchemaVersion,
		Number:              number,
		ID:                  id,
		ParentID:            strings.TrimSpace(parentID),
		CreatedAt:           createdAt.UTC().Format(time.RFC3339Nano),
		ReasonType:          reasonType,
		Reason:              strings.TrimSpace(reason),
		Evidence:            append([]string{}, evidence...),
		InputEvidenceHash:   strings.TrimSpace(inputEvidenceHash),
		EvidenceHash:        strings.TrimSpace(evidenceHash),
		PlanningRunID:       strings.TrimSpace(planningRunID),
		PlanHash:            hash,
		PreservedPhaseIDs:   append([]int{}, preserved...),
		SupersededPhaseIDs:  append([]int{}, superseded...),
		ReplacementPhaseIDs: append([]int{}, replacements...),
		Phases:              clonePhases(plan.Phases),
	}, nil
}

func renumberRevisionPhases(phases []colony.Phase, offset int) []colony.Phase {
	result := clonePhases(phases)
	for i := range result {
		result[i].ID = offset + i + 1
		result[i].Status = colony.PhasePending
		result[i].WatcherFailureCount = 0
		if i == 0 {
			result[i].Status = colony.PhaseReady
		}
		for j := range result[i].Tasks {
			id := fmt.Sprintf("%d.%d", result[i].ID, j+1)
			result[i].Tasks[j].ID = &id
			result[i].Tasks[j].Status = colony.TaskPending
			for depIndex, dependency := range result[i].Tasks[j].DependsOn {
				parts := strings.Split(strings.TrimSpace(dependency), ".")
				if len(parts) != 2 {
					continue
				}
				phaseID, phaseErr := strconv.Atoi(parts[0])
				taskID, taskErr := strconv.Atoi(parts[1])
				if phaseErr == nil && taskErr == nil {
					result[i].Tasks[j].DependsOn[depIndex] = fmt.Sprintf("%d.%d", phaseID+offset, taskID)
				}
			}
		}
	}
	return result
}

func validateUniquePlanTaskIDs(phases []colony.Phase) error {
	seen := map[string]struct{}{}
	for _, phase := range phases {
		for _, task := range phase.Tasks {
			id := strings.TrimSpace(ptrStr(task.ID))
			if id == "" {
				return fmt.Errorf("phase %d contains a task without an id", phase.ID)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("revised plan duplicates task id %s", id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func clonePhases(phases []colony.Phase) []colony.Phase {
	if phases == nil {
		return nil
	}
	copyPhases := make([]colony.Phase, len(phases))
	for i, phase := range phases {
		copyPhases[i] = phase
		copyPhases[i].Tasks = cloneTasks(phase.Tasks)
		copyPhases[i].SuccessCriteria = cloneStrings(phase.SuccessCriteria)
		copyPhases[i].EvidenceRequirements = cloneEvidenceRequirements(phase.EvidenceRequirements)
		for j := range copyPhases[i].Tasks {
			task := &copyPhases[i].Tasks[j]
			if task.ID != nil {
				id := *task.ID
				task.ID = &id
			}
			task.Constraints = cloneStrings(task.Constraints)
			task.Hints = cloneStrings(task.Hints)
			task.SuccessCriteria = cloneStrings(task.SuccessCriteria)
			task.EvidenceRequirements = cloneEvidenceRequirements(task.EvidenceRequirements)
			task.DependsOn = cloneStrings(task.DependsOn)
		}
	}
	return copyPhases
}

func cloneTasks(tasks []colony.Task) []colony.Task {
	if tasks == nil {
		return nil
	}
	return append([]colony.Task(nil), tasks...)
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneEvidenceRequirements(requirements []colony.CriterionEvidenceRequirement) []colony.CriterionEvidenceRequirement {
	if requirements == nil {
		return nil
	}
	cloned := append([]colony.CriterionEvidenceRequirement(nil), requirements...)
	for i := range cloned {
		cloned[i].Artifacts = cloneStrings(requirements[i].Artifacts)
		cloned[i].Checks = cloneStrings(requirements[i].Checks)
	}
	return cloned
}

func phaseIDs(phases []colony.Phase) []int {
	ids := make([]int, 0, len(phases))
	for _, phase := range phases {
		ids = append(ids, phase.ID)
	}
	return ids
}

func nextPlanRevisionNumber(revisions []colony.PlanRevision) int {
	max := 0
	for _, revision := range revisions {
		if revision.Number > max {
			max = revision.Number
		}
	}
	return max + 1
}

func revisionCreatedAt(plan colony.Plan, fallback time.Time) time.Time {
	if plan.GeneratedAt != nil {
		return plan.GeneratedAt.UTC()
	}
	return fallback.UTC()
}

func activePlanRevision(plan colony.Plan) (colony.PlanRevision, bool) {
	id := strings.TrimSpace(plan.ActiveRevisionID)
	for _, revision := range plan.Revisions {
		if revision.ID == id {
			return revision, true
		}
	}
	return colony.PlanRevision{}, false
}

func planRevisionSummary(plan colony.Plan) map[string]interface{} {
	if revision, ok := activePlanRevision(plan); ok {
		return map[string]interface{}{
			"id":                    revision.ID,
			"number":                revision.Number,
			"reason_type":           revision.ReasonType,
			"reason":                revision.Reason,
			"evidence":              append([]string{}, revision.Evidence...),
			"created_at":            revision.CreatedAt,
			"parent_id":             revision.ParentID,
			"preserved_phase_ids":   append([]int{}, revision.PreservedPhaseIDs...),
			"superseded_phase_ids":  append([]int{}, revision.SupersededPhaseIDs...),
			"replacement_phase_ids": append([]int{}, revision.ReplacementPhaseIDs...),
		}
	}
	if len(plan.Phases) > 0 {
		return map[string]interface{}{
			"id":          activePlanRevisionID(plan),
			"number":      0,
			"reason_type": colony.PlanRevisionLegacyImport,
			"reason":      "Legacy plan has not yet been revised",
		}
	}
	return map[string]interface{}{}
}

func planRevisionCapsuleLine(plan colony.Plan) string {
	if revision, ok := activePlanRevision(plan); ok {
		return fmt.Sprintf("Plan revision: %s (r%d, %s) - %s\n", revision.ID, revision.Number, revision.ReasonType, revision.Reason)
	}
	if len(plan.Phases) > 0 {
		return fmt.Sprintf("Plan revision: %s (legacy plan)\n", activePlanRevisionID(plan))
	}
	return ""
}

func renderPlanRevisionWorkerAppendix(revision *codexPlanRevisionContext) string {
	if revision == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Accepted Plan Revision Boundary\n")
	b.WriteString(fmt.Sprintf("- Revision reason: %s: %s\n", revision.ReasonType, revision.Reason))
	if len(revision.Evidence) > 0 {
		b.WriteString("- Read these revision evidence files before proposing replacement work: " + strings.Join(revision.Evidence, ", ") + "\n")
	}
	if len(revision.CompletedPhases) > 0 {
		b.WriteString("- Completed phases are immutable and must not appear in phase-plan.json:\n")
		for _, phase := range revision.CompletedPhases {
			b.WriteString(fmt.Sprintf("  - Phase %d: %s\n", phase.ID, phase.Name))
		}
	}
	if len(revision.SupersededPhases) > 0 {
		b.WriteString("- Replace only this unfinished suffix:\n")
		for _, phase := range revision.SupersededPhases {
			b.WriteString(fmt.Sprintf("  - Phase %d: %s (%s)\n", phase.ID, phase.Name, phase.Status))
		}
	}
	b.WriteString("- phase-plan.json must contain only replacement unfinished phases. Aether will preserve completed phases and assign final phase/task IDs atomically.\n")
	b.WriteString("- Do not repeat completed task goals. Dependencies inside the replacement suffix use local artifact IDs starting at 1.1; Aether offsets them after the immutable prefix.\n")
	return b.String()
}

func planRevisionCLIArgs(revision *codexPlanRevisionContext) string {
	if revision == nil {
		return ""
	}
	parts := []string{
		"--refresh",
		"--revision-type", shellQuotePlanArg(string(revision.ReasonType)),
		"--revision-reason", shellQuotePlanArg(revision.Reason),
	}
	for _, evidence := range revision.Evidence {
		parts = append(parts, "--revision-evidence", shellQuotePlanArg(evidence))
	}
	return strings.Join(parts, " ")
}

func planRevisionRecommendation(reasonType colony.PlanRevisionReason, reason string, evidence ...string) map[string]interface{} {
	context := &codexPlanRevisionContext{
		ReasonType: reasonType,
		Reason:     strings.Join(strings.Fields(strings.TrimSpace(reason)), " "),
		Evidence:   uniqueSortedStrings(evidence),
	}
	return map[string]interface{}{
		"automatic":   false,
		"reason_type": context.ReasonType,
		"reason":      context.Reason,
		"evidence":    append([]string{}, context.Evidence...),
		"command":     "aether plan " + planRevisionCLIArgs(context),
	}
}

func planRevisionContextsEqual(left, right *codexPlanRevisionContext) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if left.BaseRevisionID != right.BaseRevisionID ||
		left.BasePlanStateHash != right.BasePlanStateHash ||
		left.ReasonType != right.ReasonType ||
		left.Reason != right.Reason ||
		left.EvidenceHash != right.EvidenceHash {
		return false
	}
	leftEvidence := sortedPlanRevisionEvidence(left.Evidence)
	rightEvidence := sortedPlanRevisionEvidence(right.Evidence)
	if len(leftEvidence) != len(rightEvidence) {
		return false
	}
	for i := range leftEvidence {
		if leftEvidence[i] != rightEvidence[i] {
			return false
		}
	}
	return true
}

func shellQuotePlanArg(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func sortedPlanRevisionEvidence(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}
