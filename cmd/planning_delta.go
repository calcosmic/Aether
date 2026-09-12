package cmd

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	planningSemanticRequirement = "requirement"
	planningSemanticAcceptance  = "acceptance"
	planningSemanticNegative    = "negative"
	planningSemanticRecovery    = "recovery"
	planningSemanticPublicPath  = "public_path"
)

// planningSemanticSnapshotSource keeps executable plan input and authority
// state together while making their treatment explicit. Current-schema
// proposals require immutable semantic IDs; legacy plans may still be
// inspected without pretending that they possess current planning authority.
type planningSemanticSnapshotSource struct {
	CurrentSchema bool
	Plan          colony.Plan
	Revision      *colony.PlanRevision
	Specification *colony.Specification
}

// planProposalContract is the validation boundary for a newly generated plan.
// Files and user-facing applicability live here instead of on the legacy Plan
// model so loading an old active plan never fabricates current-schema facts.
type planProposalContract struct {
	Plan                  colony.Plan
	Revision              *colony.PlanRevision
	Specification         *colony.Specification
	TaskDeclarations      []planProposalTaskDeclaration
	UserFacingSemanticIDs []string
	Removals              []planProposalRemoval
}

type planProposalTaskDeclaration struct {
	TaskSemanticID string
	Files          []string
	NoFileReason   string
	UserFacing     bool
}

type planProposalRemoval struct {
	SemanticID     string
	Classification colony.PlanningSemanticChangeKind
	Rationale      string
}

type validatedPlanProposalTaskDeclaration struct {
	Files        []string
	NoFileReason string
	UserFacing   bool
}

type planProposalTaskLocation struct {
	PhaseIndex int
	TaskIndex  int
	Path       string
	SemanticID string
}

type planProposalDependencyEdge struct {
	Target string
	Path   string
}

type planningSemanticEntry struct {
	SemanticID  string `json:"semantic_id"`
	ContentHash string `json:"content_hash"`
}

type planningAuthorityState struct {
	Key                 string
	Kind                colony.PlanningAuthorityImpactKind
	SourceID            string
	ContentHash         string
	AffectedSemanticIDs []string
	Rationale           string
}

// planningSemanticSnapshot is a canonical, renderer-neutral view. ContentHash
// covers executable semantics only; Authority is intentionally excluded so an
// approval or acceptance transition cannot look like planning improvement.
type planningSemanticSnapshot struct {
	ContentHash          string
	Phases               []planningSemanticEntry
	Tasks                []planningSemanticEntry
	Dependencies         []planningSemanticEntry
	RequirementLinks     []planningSemanticEntry
	AcceptanceChecks     []planningSemanticEntry
	NegativeExpectations []planningSemanticEntry
	RecoveryExpectations []planningSemanticEntry
	PublicPaths          []planningSemanticEntry
	Authority            []planningAuthorityState
}

func buildPlanningSemanticSnapshot(source planningSemanticSnapshotSource) (planningSemanticSnapshot, error) {
	snapshot := emptyPlanningSemanticSnapshot()
	currentSchema := source.CurrentSchema || source.Plan.AcceptancePolicy == colony.PlanAcceptanceExplicitOwner

	phases := source.Plan.Phases
	planSemanticID := ""
	if source.Revision != nil {
		planSemanticID = canonicalPlanningText(source.Revision.SemanticID)
		if len(source.Revision.Phases) > 0 {
			phases = source.Revision.Phases
		}
	}
	if currentSchema && source.Revision == nil {
		return snapshot, fmt.Errorf("revision is required for a current-schema plan snapshot")
	}
	if currentSchema && planSemanticID == "" {
		return snapshot, fmt.Errorf("revision.semantic_id is required")
	}
	if planSemanticID == "" {
		planSemanticID = "legacy-plan"
	}

	seenSemanticIDs := make(map[string]string)
	if err := registerPlanningSemanticID(seenSemanticIDs, planSemanticID, "revision.semantic_id"); err != nil {
		return snapshot, err
	}

	type taskLocation struct {
		phaseIndex int
		taskIndex  int
		semanticID string
	}
	taskIDs := make(map[string]taskLocation)
	allSemanticIDs := []string{planSemanticID}

	for phaseIndex := range phases {
		phase := phases[phaseIndex]
		phasePath := fmt.Sprintf("phases[%d]", phaseIndex)
		phaseSemanticID := canonicalPlanningText(phase.SemanticID)
		if phaseSemanticID == "" {
			if currentSchema {
				return snapshot, fmt.Errorf("%s.semantic_id is required", phasePath)
			}
			phaseSemanticID = fmt.Sprintf("legacy-phase-%d", phase.ID)
		}
		if err := registerPlanningSemanticID(seenSemanticIDs, phaseSemanticID, phasePath+".semantic_id"); err != nil {
			return snapshot, err
		}
		allSemanticIDs = append(allSemanticIDs, phaseSemanticID)

		phaseEntry, err := newPlanningSemanticEntry(phaseSemanticID, struct {
			Name               string           `json:"name"`
			Description        string           `json:"description"`
			Mode               colony.PhaseMode `json:"mode"`
			ExpectFailingTests bool             `json:"expect_failing_tests"`
		}{
			Name:               canonicalPlanningText(phase.Name),
			Description:        canonicalPlanningText(phase.Description),
			Mode:               phase.Mode,
			ExpectFailingTests: phase.ExpectFailingTests,
		})
		if err != nil {
			return snapshot, fmt.Errorf("hash %s: %w", phasePath, err)
		}
		snapshot.Phases = append(snapshot.Phases, phaseEntry)

		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskPath := fmt.Sprintf("%s.tasks[%d]", phasePath, taskIndex)
			taskSemanticID := canonicalPlanningText(task.SemanticID)
			if taskSemanticID == "" {
				if currentSchema {
					return snapshot, fmt.Errorf("%s.semantic_id is required", taskPath)
				}
				taskSemanticID = legacyPlanningTaskSemanticID(phase.ID, taskIndex, task.ID)
			}
			if err := registerPlanningSemanticID(seenSemanticIDs, taskSemanticID, taskPath+".semantic_id"); err != nil {
				return snapshot, err
			}
			allSemanticIDs = append(allSemanticIDs, taskSemanticID)

			runtimeID := canonicalPlanningText(ptrStr(task.ID))
			if runtimeID != "" {
				if prior, exists := taskIDs[runtimeID]; exists {
					return snapshot, fmt.Errorf("%s.id %q duplicates phases[%d].tasks[%d].id", taskPath, runtimeID, prior.phaseIndex, prior.taskIndex)
				}
				taskIDs[runtimeID] = taskLocation{phaseIndex: phaseIndex, taskIndex: taskIndex, semanticID: taskSemanticID}
			}
			if prior, exists := taskIDs[taskSemanticID]; exists && prior.semanticID != taskSemanticID {
				return snapshot, fmt.Errorf("%s.semantic_id %q conflicts with a runtime task id", taskPath, taskSemanticID)
			}
			taskIDs[taskSemanticID] = taskLocation{phaseIndex: phaseIndex, taskIndex: taskIndex, semanticID: taskSemanticID}

			taskEntry, err := newPlanningSemanticEntry(taskSemanticID, struct {
				Goal        string   `json:"goal"`
				Constraints []string `json:"constraints"`
				Hints       []string `json:"hints"`
			}{
				Goal:        canonicalPlanningText(task.Goal),
				Constraints: canonicalPlanningSet(task.Constraints),
				Hints:       canonicalPlanningSet(task.Hints),
			})
			if err != nil {
				return snapshot, fmt.Errorf("hash %s: %w", taskPath, err)
			}
			snapshot.Tasks = append(snapshot.Tasks, taskEntry)
		}
	}

	specification := activePlanningSpecRevision(source.Specification)
	lookup, err := buildPlanningSpecLookup(specification)
	if err != nil {
		return snapshot, err
	}

	if source.Revision != nil {
		if err := appendPlanningNodeSemantics(&snapshot, "revision", planSemanticID, source.Revision.RequirementProofLinks, source.Revision.AcceptanceProofLinks, source.Revision.NegativeProofLinks, source.Revision.RecoveryProofLinks, source.Revision.PublicPathProofLinks, nil, nil, lookup); err != nil {
			return snapshot, err
		}
	}
	for phaseIndex := range phases {
		phase := phases[phaseIndex]
		phasePath := fmt.Sprintf("phases[%d]", phaseIndex)
		phaseSemanticID := canonicalPlanningText(phase.SemanticID)
		if phaseSemanticID == "" {
			phaseSemanticID = fmt.Sprintf("legacy-phase-%d", phase.ID)
		}
		if err := appendPlanningNodeSemantics(&snapshot, phasePath, phaseSemanticID, phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks, phase.RecoveryProofLinks, phase.PublicPathProofLinks, phase.SuccessCriteria, phase.EvidenceRequirements, lookup); err != nil {
			return snapshot, err
		}
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskPath := fmt.Sprintf("%s.tasks[%d]", phasePath, taskIndex)
			taskSemanticID := canonicalPlanningText(task.SemanticID)
			if taskSemanticID == "" {
				taskSemanticID = legacyPlanningTaskSemanticID(phase.ID, taskIndex, task.ID)
			}
			if err := appendPlanningNodeSemantics(&snapshot, taskPath, taskSemanticID, task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks, task.RecoveryProofLinks, task.PublicPathProofLinks, task.SuccessCriteria, task.EvidenceRequirements, lookup); err != nil {
				return snapshot, err
			}
			for dependencyIndex, dependency := range task.DependsOn {
				dependency = canonicalPlanningText(dependency)
				target, ok := taskIDs[dependency]
				if !ok {
					return snapshot, fmt.Errorf("%s.depends_on[%d] %q does not resolve to a task", taskPath, dependencyIndex, dependency)
				}
				edgeID := planningSemanticRelationID(taskSemanticID, "depends_on", target.semanticID)
				entry, err := newPlanningSemanticEntry(edgeID, struct {
					From string `json:"from"`
					To   string `json:"to"`
				}{From: taskSemanticID, To: target.semanticID})
				if err != nil {
					return snapshot, fmt.Errorf("hash %s.depends_on[%d]: %w", taskPath, dependencyIndex, err)
				}
				snapshot.Dependencies = append(snapshot.Dependencies, entry)
			}
		}
	}

	snapshot.normalize()
	snapshotHash, err := planningSemanticSnapshotHash(snapshot)
	if err != nil {
		return snapshot, err
	}
	snapshot.ContentHash = snapshotHash
	snapshot.Authority, err = buildPlanningAuthoritySnapshot(source, canonicalPlanningSet(allSemanticIDs))
	if err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func comparePlanningSemanticSnapshots(before, after planningSemanticSnapshot) (colony.PlanningSemanticDelta, error) {
	var err error
	delta := colony.PlanningSemanticDelta{SchemaVersion: colony.PlanningSchemaVersion}
	sections := []struct {
		name   colony.PlanningSemanticSection
		before []planningSemanticEntry
		after  []planningSemanticEntry
		set    func([]colony.PlanningSemanticChange)
	}{
		{name: colony.PlanningSemanticSectionPhases, before: before.Phases, after: after.Phases, set: func(value []colony.PlanningSemanticChange) { delta.Phases = value }},
		{name: colony.PlanningSemanticSectionTasks, before: before.Tasks, after: after.Tasks, set: func(value []colony.PlanningSemanticChange) { delta.Tasks = value }},
		{name: colony.PlanningSemanticSectionDependencies, before: before.Dependencies, after: after.Dependencies, set: func(value []colony.PlanningSemanticChange) { delta.Dependencies = value }},
		{name: colony.PlanningSemanticSectionRequirementLinks, before: before.RequirementLinks, after: after.RequirementLinks, set: func(value []colony.PlanningSemanticChange) { delta.RequirementLinks = value }},
		{name: colony.PlanningSemanticSectionAcceptanceChecks, before: before.AcceptanceChecks, after: after.AcceptanceChecks, set: func(value []colony.PlanningSemanticChange) { delta.AcceptanceChecks = value }},
		{name: colony.PlanningSemanticSectionNegativeExpectations, before: before.NegativeExpectations, after: after.NegativeExpectations, set: func(value []colony.PlanningSemanticChange) { delta.NegativeExpectations = value }},
		{name: colony.PlanningSemanticSectionRecoveryExpectations, before: before.RecoveryExpectations, after: after.RecoveryExpectations, set: func(value []colony.PlanningSemanticChange) { delta.RecoveryExpectations = value }},
		{name: colony.PlanningSemanticSectionPublicPaths, before: before.PublicPaths, after: after.PublicPaths, set: func(value []colony.PlanningSemanticChange) { delta.PublicPaths = value }},
	}
	for _, section := range sections {
		var changes []colony.PlanningSemanticChange
		changes, err = comparePlanningSemanticEntries(section.name, section.before, section.after)
		if err != nil {
			return colony.PlanningSemanticDelta{}, err
		}
		section.set(changes)
	}
	delta.AuthorityImpacts, err = comparePlanningAuthorityStates(before.Authority, after.Authority)
	if err != nil {
		return colony.PlanningSemanticDelta{}, err
	}

	if planningSemanticDeltaReadyForAddress(delta) {
		if err := colony.AddressPlanningSemanticDelta(&delta); err != nil {
			return colony.PlanningSemanticDelta{}, fmt.Errorf("address planning semantic delta: %w", err)
		}
	}
	return delta, nil
}

// validatePlanProposalContract rejects incomplete generated proposals before
// they can become iteration cards or candidates. It is deliberately pure: it
// canonicalizes into a returned snapshot but never repairs or mutates input.
func validatePlanProposalContract(proposal planProposalContract, predecessor *planningSemanticSnapshot) (planningSemanticSnapshot, error) {
	empty := emptyPlanningSemanticSnapshot()
	if proposal.Revision == nil {
		return empty, fmt.Errorf("revision is required")
	}
	if len(proposal.Revision.Phases) == 0 {
		return empty, fmt.Errorf("revision.phases is required")
	}

	lookup, err := proposalPlanningSpecLookup(proposal.Specification)
	if err != nil {
		return empty, err
	}

	userFacing, err := proposalUserFacingIDs(proposal.UserFacingSemanticIDs)
	if err != nil {
		return empty, err
	}
	declarations, _, err := proposalTaskDeclarations(proposal.TaskDeclarations)
	if err != nil {
		return empty, err
	}

	seenSemanticIDs := make(map[string]string)
	planSemanticID := canonicalPlanningText(proposal.Revision.SemanticID)
	if err := registerPlanningSemanticID(seenSemanticIDs, planSemanticID, "revision.semantic_id"); err != nil {
		return empty, err
	}

	phaseLocations := make(map[string]string)
	taskLocations := make(map[string]planProposalTaskLocation)
	taskReferences := make(map[string]planProposalTaskLocation)
	for phaseIndex := range proposal.Revision.Phases {
		phase := proposal.Revision.Phases[phaseIndex]
		phasePath := fmt.Sprintf("phases[%d]", phaseIndex)
		phaseSemanticID := canonicalPlanningText(phase.SemanticID)
		if err := registerPlanningSemanticID(seenSemanticIDs, phaseSemanticID, phasePath+".semantic_id"); err != nil {
			return empty, err
		}
		phaseLocations[phaseSemanticID] = phasePath
		if len(phase.Tasks) == 0 {
			return empty, fmt.Errorf("%s.tasks is required", phasePath)
		}
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskPath := fmt.Sprintf("%s.tasks[%d]", phasePath, taskIndex)
			taskSemanticID := canonicalPlanningText(task.SemanticID)
			if err := registerPlanningSemanticID(seenSemanticIDs, taskSemanticID, taskPath+".semantic_id"); err != nil {
				return empty, err
			}
			location := planProposalTaskLocation{
				PhaseIndex: phaseIndex,
				TaskIndex:  taskIndex,
				Path:       taskPath,
				SemanticID: taskSemanticID,
			}
			taskLocations[taskSemanticID] = location
			if prior, duplicate := taskReferences[taskSemanticID]; duplicate && prior.Path != taskPath {
				return empty, fmt.Errorf("%s.semantic_id %q conflicts with task reference at %s", taskPath, taskSemanticID, prior.Path)
			}
			taskReferences[taskSemanticID] = location
			if runtimeID := canonicalPlanningText(ptrStr(task.ID)); runtimeID != "" {
				if prior, duplicate := taskReferences[runtimeID]; duplicate && prior.Path != taskPath {
					return empty, fmt.Errorf("%s.id %q conflicts with task reference at %s", taskPath, runtimeID, prior.Path)
				}
				taskReferences[runtimeID] = location
			}
		}
	}

	knownSemanticIDs := make(map[string]struct{}, len(phaseLocations)+len(taskLocations))
	for semanticID := range phaseLocations {
		knownSemanticIDs[semanticID] = struct{}{}
	}
	for semanticID := range taskLocations {
		knownSemanticIDs[semanticID] = struct{}{}
	}
	for index, value := range proposal.UserFacingSemanticIDs {
		semanticID := canonicalPlanningText(value)
		if _, known := knownSemanticIDs[semanticID]; !known {
			return empty, fmt.Errorf("user_facing_semantic_ids[%d] %q does not resolve to a phase or task", index, semanticID)
		}
	}
	for index, declaration := range proposal.TaskDeclarations {
		semanticID := canonicalPlanningText(declaration.TaskSemanticID)
		if _, known := taskLocations[semanticID]; !known {
			return empty, fmt.Errorf("task_declarations[%d].task_semantic_id %q does not resolve to a task", index, semanticID)
		}
	}

	validatedDeclarations := make(map[string]validatedPlanProposalTaskDeclaration, len(taskLocations))
	dependencies := make(map[string][]planProposalDependencyEdge, len(taskLocations))
	for phaseIndex := range proposal.Revision.Phases {
		phase := proposal.Revision.Phases[phaseIndex]
		phasePath := fmt.Sprintf("phases[%d]", phaseIndex)
		phaseSemanticID := canonicalPlanningText(phase.SemanticID)
		if err := validateProposalNodeContract(
			phasePath,
			phase.RequirementProofLinks,
			phase.AcceptanceProofLinks,
			phase.NegativeProofLinks,
			phase.RecoveryProofLinks,
			phase.PublicPathProofLinks,
			phase.SuccessCriteria,
			phase.EvidenceRequirements,
			proposalSemanticIDIsUserFacing(userFacing, phaseSemanticID),
			lookup,
		); err != nil {
			return empty, err
		}

		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskPath := fmt.Sprintf("%s.tasks[%d]", phasePath, taskIndex)
			taskSemanticID := canonicalPlanningText(task.SemanticID)
			if canonicalPlanningText(task.Goal) == "" {
				return empty, fmt.Errorf("%s.goal is required", taskPath)
			}
			declaration, declared := declarations[taskSemanticID]
			if !declared {
				return empty, fmt.Errorf("%s.files requires exact repository-relative files or an explicit no_file_reason", taskPath)
			}
			validatedDeclaration, err := validateProposalTaskDeclaration(taskPath, declaration)
			if err != nil {
				return empty, err
			}
			validatedDeclarations[taskSemanticID] = validatedDeclaration
			userFacingTask := validatedDeclaration.UserFacing || proposalSemanticIDIsUserFacing(userFacing, taskSemanticID)
			if err := validateProposalNodeContract(
				taskPath,
				task.RequirementProofLinks,
				task.AcceptanceProofLinks,
				task.NegativeProofLinks,
				task.RecoveryProofLinks,
				task.PublicPathProofLinks,
				task.SuccessCriteria,
				task.EvidenceRequirements,
				userFacingTask,
				lookup,
			); err != nil {
				return empty, err
			}

			seenTargets := make(map[string]struct{}, len(task.DependsOn))
			for dependencyIndex, dependency := range task.DependsOn {
				dependencyPath := fmt.Sprintf("%s.depends_on[%d]", taskPath, dependencyIndex)
				dependency = canonicalPlanningText(dependency)
				if dependency == "" {
					return empty, fmt.Errorf("%s is required", dependencyPath)
				}
				target, resolved := taskReferences[dependency]
				if !resolved {
					return empty, fmt.Errorf("%s %q does not resolve to a task", dependencyPath, dependency)
				}
				if _, duplicate := seenTargets[target.SemanticID]; duplicate {
					return empty, fmt.Errorf("%s duplicates dependency on %q", dependencyPath, target.SemanticID)
				}
				seenTargets[target.SemanticID] = struct{}{}
				dependencies[taskSemanticID] = append(dependencies[taskSemanticID], planProposalDependencyEdge{
					Target: target.SemanticID,
					Path:   dependencyPath,
				})
			}
		}
	}
	if err := validatePlanProposalDependencyCycles(taskLocations, dependencies); err != nil {
		return empty, err
	}

	snapshot, err := buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		CurrentSchema: true,
		Plan:          proposal.Plan,
		Revision:      proposal.Revision,
		Specification: proposal.Specification,
	})
	if err != nil {
		return empty, err
	}
	if err := addPlanProposalDeclarationSemantics(&snapshot, validatedDeclarations, userFacing); err != nil {
		return empty, err
	}
	if err := validatePlanProposalRemovals(proposal.Removals, predecessor, snapshot); err != nil {
		return empty, err
	}
	return snapshot, nil
}

func proposalPlanningSpecLookup(specification *colony.Specification) (planningSpecLookup, error) {
	if specification == nil {
		return nil, fmt.Errorf("specification is required")
	}
	currentRevisionID := canonicalPlanningText(specification.CurrentRevisionID)
	if currentRevisionID == "" {
		return nil, fmt.Errorf("specification.current_revision_id is required")
	}
	var current *colony.SpecRevision
	for index := range specification.Revisions {
		if canonicalPlanningText(specification.Revisions[index].ID) == currentRevisionID {
			current = &specification.Revisions[index]
			break
		}
	}
	if current == nil {
		return nil, fmt.Errorf("specification.current_revision_id %q does not resolve to a revision", currentRevisionID)
	}
	lookup, err := buildPlanningSpecLookup(current)
	if err != nil {
		return nil, err
	}
	return lookup, nil
}

func proposalUserFacingIDs(values []string) (map[string]int, error) {
	result := make(map[string]int, len(values))
	for index, value := range values {
		value = canonicalPlanningText(value)
		if value == "" {
			return nil, fmt.Errorf("user_facing_semantic_ids[%d] is required", index)
		}
		if prior, duplicate := result[value]; duplicate {
			return nil, fmt.Errorf("user_facing_semantic_ids[%d] %q duplicates user_facing_semantic_ids[%d]", index, value, prior)
		}
		result[value] = index
	}
	return result, nil
}

func proposalSemanticIDIsUserFacing(values map[string]int, semanticID string) bool {
	_, ok := values[canonicalPlanningText(semanticID)]
	return ok
}

func proposalTaskDeclarations(values []planProposalTaskDeclaration) (map[string]planProposalTaskDeclaration, map[string]int, error) {
	declarations := make(map[string]planProposalTaskDeclaration, len(values))
	indexes := make(map[string]int, len(values))
	for index, declaration := range values {
		semanticID := canonicalPlanningText(declaration.TaskSemanticID)
		if semanticID == "" {
			return nil, nil, fmt.Errorf("task_declarations[%d].task_semantic_id is required", index)
		}
		if prior, duplicate := indexes[semanticID]; duplicate {
			return nil, nil, fmt.Errorf("task_declarations[%d].task_semantic_id %q duplicates task_declarations[%d]", index, semanticID, prior)
		}
		declaration.TaskSemanticID = semanticID
		declarations[semanticID] = declaration
		indexes[semanticID] = index
	}
	return declarations, indexes, nil
}

func validateProposalTaskDeclaration(taskPath string, declaration planProposalTaskDeclaration) (validatedPlanProposalTaskDeclaration, error) {
	result := validatedPlanProposalTaskDeclaration{
		NoFileReason: canonicalPlanningText(declaration.NoFileReason),
		UserFacing:   declaration.UserFacing,
	}
	if len(declaration.Files) == 0 {
		if result.NoFileReason == "" {
			return result, fmt.Errorf("%s.files requires exact repository-relative files or an explicit no_file_reason", taskPath)
		}
		return result, nil
	}
	if result.NoFileReason != "" {
		return result, fmt.Errorf("%s.no_file_reason must be empty when files are declared", taskPath)
	}
	seen := make(map[string]int, len(declaration.Files))
	for index, file := range declaration.Files {
		filePath := fmt.Sprintf("%s.files[%d]", taskPath, index)
		normalized, err := normalizePlanProposalFile(file)
		if err != nil {
			return result, fmt.Errorf("%s: %w", filePath, err)
		}
		if prior, duplicate := seen[normalized]; duplicate {
			return result, fmt.Errorf("%s %q duplicates %s.files[%d]", filePath, normalized, taskPath, prior)
		}
		seen[normalized] = index
		result.Files = append(result.Files, normalized)
	}
	sort.Strings(result.Files)
	return result, nil
}

func normalizePlanProposalFile(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("file path is required")
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("file path %q must use repository-relative slash separators", value)
	}
	if path.IsAbs(value) || isWindowsAbsolutePlanProposalPath(value) {
		return "", fmt.Errorf("file path %q must be repository-relative", value)
	}
	if strings.ContainsAny(value, "*?[") {
		return "", fmt.Errorf("file path %q must name one exact file, not a pattern", value)
	}
	if strings.HasSuffix(value, "/") {
		return "", fmt.Errorf("file path %q must name a file, not a directory", value)
	}
	segments := strings.Split(value, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("file path %q is not canonical and repository-relative", value)
		}
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return "", fmt.Errorf("file path %q is not canonical and repository-relative", value)
	}
	return cleaned, nil
}

func isWindowsAbsolutePlanProposalPath(value string) bool {
	return len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z'))
}

func validateProposalNodeContract(path string, requirements, acceptance, negative, recovery, publicPaths, criteria []string, evidence []colony.CriterionEvidenceRequirement, userFacing bool, lookup planningSpecLookup) error {
	links := planningProofLinkSets(requirements, acceptance, negative, recovery, publicPaths)
	for _, kind := range planningProofLinkKinds() {
		if err := validateProposalProofLinks(path, kind, links[kind], planningProofLinkRequired(kind, userFacing), lookup); err != nil {
			return err
		}
	}
	return validateProposalAcceptanceContract(path, criteria, evidence)
}

// planningProofLinkKinds is every proof-link kind a plan node carries, in the
// order drafting and acceptance check them.
func planningProofLinkKinds() []string {
	return []string{planningSemanticRequirement, planningSemanticAcceptance, planningSemanticNegative, planningSemanticRecovery, planningSemanticPublicPath}
}

// planningProofLinkRequired is the one rule drafting and acceptance share for a
// single phase or task. Every node proves requirement, acceptance, negative and
// recovery links; public-path proof is owed only by work declared user-facing,
// because a housekeeping task touches no public path and forcing a link onto it
// would make the Route-Setter invent one. Acceptance cannot see the user-facing
// declaration (it is not persisted), so it asks with userFacing=false and relies
// on drafting having enforced the stricter case.
func planningProofLinkRequired(kind string, userFacing bool) bool {
	return kind != planningSemanticPublicPath || userFacing
}

// planningProofLinkSets keys one node's proof links by kind.
func planningProofLinkSets(requirements, acceptance, negative, recovery, publicPaths []string) map[string][]string {
	return map[string][]string{
		planningSemanticRequirement: requirements,
		planningSemanticAcceptance:  acceptance,
		planningSemanticNegative:    negative,
		planningSemanticRecovery:    recovery,
		planningSemanticPublicPath:  publicPaths,
	}
}

func validateProposalProofLinks(path, kind string, values []string, required bool, lookup planningSpecLookup) error {
	field := planningProofField(kind)
	if required && len(values) == 0 {
		return fmt.Errorf("%s.%s is required", path, field)
	}
	seen := make(map[string]int, len(values))
	for index, value := range values {
		valuePath := fmt.Sprintf("%s.%s[%d]", path, field, index)
		value = canonicalPlanningText(value)
		if value == "" {
			return fmt.Errorf("%s is required", valuePath)
		}
		if prior, duplicate := seen[value]; duplicate {
			return fmt.Errorf("%s %q duplicates %s.%s[%d]", valuePath, value, path, field, prior)
		}
		seen[value] = index
		if _, resolves := lookup[kind+"\x00"+value]; !resolves {
			return fmt.Errorf("%s %q does not resolve in the active specification revision", valuePath, value)
		}
	}
	return nil
}

func validateProposalAcceptanceContract(path string, criteria []string, requirements []colony.CriterionEvidenceRequirement) error {
	if len(criteria) == 0 {
		return fmt.Errorf("%s.success_criteria is required", path)
	}
	criteriaByText := make(map[string]int, len(criteria))
	for index, criterion := range criteria {
		criterionPath := fmt.Sprintf("%s.success_criteria[%d]", path, index)
		criterion = canonicalPlanningText(criterion)
		if criterion == "" {
			return fmt.Errorf("%s is required", criterionPath)
		}
		if prior, duplicate := criteriaByText[criterion]; duplicate {
			return fmt.Errorf("%s %q duplicates %s.success_criteria[%d]", criterionPath, criterion, path, prior)
		}
		criteriaByText[criterion] = index
	}
	if len(requirements) == 0 {
		return fmt.Errorf("%s.evidence_requirements is required", path)
	}
	seenCriteria := make(map[string]int, len(requirements))
	for index, requirement := range requirements {
		requirementPath := fmt.Sprintf("%s.evidence_requirements[%d]", path, index)
		criterion := canonicalPlanningText(requirement.Criterion)
		if criterion == "" {
			return fmt.Errorf("%s.criterion is required", requirementPath)
		}
		if _, exists := criteriaByText[criterion]; !exists {
			return fmt.Errorf("%s.criterion %q does not match a success criterion", requirementPath, criterion)
		}
		if prior, duplicate := seenCriteria[criterion]; duplicate {
			return fmt.Errorf("%s.criterion %q duplicates %s.evidence_requirements[%d]", requirementPath, criterion, path, prior)
		}
		seenCriteria[criterion] = index
		if canonicalPlanningText(requirement.TaskID) != "" {
			return fmt.Errorf("%s.task_id must be empty at the proposal boundary", requirementPath)
		}
		for artifactIndex, artifact := range requirement.Artifacts {
			if _, err := normalizeCriterionArtifactPath(artifact); err != nil {
				return fmt.Errorf("%s.artifacts[%d]: %w", requirementPath, artifactIndex, err)
			}
		}
		if len(requirement.Checks) == 0 {
			return fmt.Errorf("%s.checks requires at least one automated check", requirementPath)
		}
		hasAutomatedCheck := false
		seenChecks := make(map[string]int, len(requirement.Checks))
		for checkIndex, check := range requirement.Checks {
			checkPath := fmt.Sprintf("%s.checks[%d]", requirementPath, checkIndex)
			check = strings.ToLower(canonicalPlanningText(check))
			if check == "" {
				return fmt.Errorf("%s is required", checkPath)
			}
			if _, supported := supportedCriterionChecks[check]; !supported {
				return fmt.Errorf("%s %q is not a supported verification check", checkPath, check)
			}
			if prior, duplicate := seenChecks[check]; duplicate {
				return fmt.Errorf("%s %q duplicates %s.checks[%d]", checkPath, check, requirementPath, prior)
			}
			seenChecks[check] = checkIndex
			switch check {
			case "build", "types", "lint", "tests":
				hasAutomatedCheck = true
			}
		}
		if !hasAutomatedCheck {
			return fmt.Errorf("%s.checks requires at least one automated check (build, types, lint, or tests)", requirementPath)
		}
	}
	for criterionIndex, rawCriterion := range criteria {
		criterion := canonicalPlanningText(rawCriterion)
		if _, bound := seenCriteria[criterion]; !bound {
			return fmt.Errorf("%s.success_criteria[%d] %q has no evidence requirement", path, criterionIndex, criterion)
		}
	}
	return nil
}

func validatePlanProposalDependencyCycles(locations map[string]planProposalTaskLocation, dependencies map[string][]planProposalDependencyEdge) error {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(locations))
	orderedIDs := make([]string, 0, len(locations))
	for semanticID := range locations {
		orderedIDs = append(orderedIDs, semanticID)
	}
	sort.Strings(orderedIDs)
	var visit func(string) error
	visit = func(semanticID string) error {
		state[semanticID] = visiting
		for _, edge := range dependencies[semanticID] {
			switch state[edge.Target] {
			case visiting:
				return fmt.Errorf("%s creates a dependency cycle involving %q", edge.Path, edge.Target)
			case unvisited:
				if err := visit(edge.Target); err != nil {
					return err
				}
			}
		}
		state[semanticID] = visited
		return nil
	}
	for _, semanticID := range orderedIDs {
		if state[semanticID] == unvisited {
			if err := visit(semanticID); err != nil {
				return err
			}
		}
	}
	return nil
}

func addPlanProposalDeclarationSemantics(snapshot *planningSemanticSnapshot, declarations map[string]validatedPlanProposalTaskDeclaration, userFacing map[string]int) error {
	for index, entry := range snapshot.Phases {
		replacement, err := newPlanningSemanticEntry(entry.SemanticID, struct {
			DefinitionHash string `json:"definition_hash"`
			UserFacing     bool   `json:"user_facing"`
		}{DefinitionHash: entry.ContentHash, UserFacing: proposalSemanticIDIsUserFacing(userFacing, entry.SemanticID)})
		if err != nil {
			return fmt.Errorf("hash phase declaration %q: %w", entry.SemanticID, err)
		}
		snapshot.Phases[index] = replacement
	}
	for index, entry := range snapshot.Tasks {
		declaration := declarations[entry.SemanticID]
		replacement, err := newPlanningSemanticEntry(entry.SemanticID, struct {
			DefinitionHash string   `json:"definition_hash"`
			Files          []string `json:"files"`
			NoFileReason   string   `json:"no_file_reason"`
			UserFacing     bool     `json:"user_facing"`
		}{
			DefinitionHash: entry.ContentHash,
			Files:          declaration.Files,
			NoFileReason:   declaration.NoFileReason,
			UserFacing:     declaration.UserFacing || proposalSemanticIDIsUserFacing(userFacing, entry.SemanticID),
		})
		if err != nil {
			return fmt.Errorf("hash task declaration %q: %w", entry.SemanticID, err)
		}
		snapshot.Tasks[index] = replacement
	}
	snapshot.normalize()
	hash, err := planningSemanticSnapshotHash(*snapshot)
	if err != nil {
		return fmt.Errorf("hash validated plan proposal: %w", err)
	}
	snapshot.ContentHash = hash
	return nil
}

func validatePlanProposalRemovals(removals []planProposalRemoval, predecessor *planningSemanticSnapshot, current planningSemanticSnapshot) error {
	predecessorIDs := make(map[string]struct{})
	if predecessor != nil {
		for _, entry := range predecessor.Phases {
			predecessorIDs[entry.SemanticID] = struct{}{}
		}
		for _, entry := range predecessor.Tasks {
			predecessorIDs[entry.SemanticID] = struct{}{}
		}
	}
	currentIDs := make(map[string]struct{}, len(current.Phases)+len(current.Tasks))
	for _, entry := range current.Phases {
		currentIDs[entry.SemanticID] = struct{}{}
	}
	for _, entry := range current.Tasks {
		currentIDs[entry.SemanticID] = struct{}{}
	}

	declared := make(map[string]int, len(removals))
	for index, removal := range removals {
		removalPath := fmt.Sprintf("removals[%d]", index)
		semanticID := canonicalPlanningText(removal.SemanticID)
		if semanticID == "" {
			return fmt.Errorf("%s.semantic_id is required", removalPath)
		}
		if prior, duplicate := declared[semanticID]; duplicate {
			return fmt.Errorf("%s.semantic_id %q duplicates removals[%d]", removalPath, semanticID, prior)
		}
		declared[semanticID] = index
		if removal.Classification != colony.PlanningSemanticChangeRemoved {
			return fmt.Errorf("%s.classification must be %q", removalPath, colony.PlanningSemanticChangeRemoved)
		}
		if canonicalPlanningText(removal.Rationale) == "" {
			return fmt.Errorf("%s.rationale is required", removalPath)
		}
		if predecessor == nil {
			return fmt.Errorf("%s.semantic_id %q has no predecessor to remove", removalPath, semanticID)
		}
		if _, existed := predecessorIDs[semanticID]; !existed {
			return fmt.Errorf("%s.semantic_id %q does not exist in the predecessor", removalPath, semanticID)
		}
		if _, remains := currentIDs[semanticID]; remains {
			return fmt.Errorf("%s.semantic_id %q is still present in the proposal", removalPath, semanticID)
		}
	}

	missing := make([]string, 0)
	for semanticID := range predecessorIDs {
		if _, remains := currentIDs[semanticID]; remains {
			continue
		}
		if _, explained := declared[semanticID]; !explained {
			missing = append(missing, semanticID)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("removals is missing an explicit removed classification for predecessor semantic ID %q", missing[0])
	}
	return nil
}

func emptyPlanningSemanticSnapshot() planningSemanticSnapshot {
	return planningSemanticSnapshot{
		Phases:               []planningSemanticEntry{},
		Tasks:                []planningSemanticEntry{},
		Dependencies:         []planningSemanticEntry{},
		RequirementLinks:     []planningSemanticEntry{},
		AcceptanceChecks:     []planningSemanticEntry{},
		NegativeExpectations: []planningSemanticEntry{},
		RecoveryExpectations: []planningSemanticEntry{},
		PublicPaths:          []planningSemanticEntry{},
		Authority:            []planningAuthorityState{},
	}
}

func (s *planningSemanticSnapshot) normalize() {
	sections := []*[]planningSemanticEntry{
		&s.Phases,
		&s.Tasks,
		&s.Dependencies,
		&s.RequirementLinks,
		&s.AcceptanceChecks,
		&s.NegativeExpectations,
		&s.RecoveryExpectations,
		&s.PublicPaths,
	}
	for _, section := range sections {
		*section = uniqueSortedPlanningSemanticEntries(*section)
	}
}

func planningSemanticSnapshotHash(snapshot planningSemanticSnapshot) (string, error) {
	payload := struct {
		Phases               []planningSemanticEntry `json:"phases"`
		Tasks                []planningSemanticEntry `json:"tasks"`
		Dependencies         []planningSemanticEntry `json:"dependencies"`
		RequirementLinks     []planningSemanticEntry `json:"requirement_links"`
		AcceptanceChecks     []planningSemanticEntry `json:"acceptance_checks"`
		NegativeExpectations []planningSemanticEntry `json:"negative_expectations"`
		RecoveryExpectations []planningSemanticEntry `json:"recovery_expectations"`
		PublicPaths          []planningSemanticEntry `json:"public_paths"`
	}{
		Phases: snapshot.Phases, Tasks: snapshot.Tasks, Dependencies: snapshot.Dependencies,
		RequirementLinks: snapshot.RequirementLinks, AcceptanceChecks: snapshot.AcceptanceChecks,
		NegativeExpectations: snapshot.NegativeExpectations, RecoveryExpectations: snapshot.RecoveryExpectations,
		PublicPaths: snapshot.PublicPaths,
	}
	return jsonSHA256(payload)
}

func newPlanningSemanticEntry(semanticID string, payload interface{}) (planningSemanticEntry, error) {
	hash, err := jsonSHA256(payload)
	if err != nil {
		return planningSemanticEntry{}, err
	}
	return planningSemanticEntry{SemanticID: semanticID, ContentHash: hash}, nil
}

func registerPlanningSemanticID(seen map[string]string, semanticID, path string) error {
	semanticID = canonicalPlanningText(semanticID)
	if semanticID == "" {
		return fmt.Errorf("%s is required", path)
	}
	if prior, duplicate := seen[semanticID]; duplicate {
		return fmt.Errorf("%s %q duplicates stable ID already used by %s", path, semanticID, prior)
	}
	seen[semanticID] = path
	return nil
}

func legacyPlanningTaskSemanticID(phaseID, taskIndex int, taskID *string) string {
	if id := canonicalPlanningText(ptrStr(taskID)); id != "" {
		return "legacy-task-" + id
	}
	return fmt.Sprintf("legacy-task-%d-%d", phaseID, taskIndex+1)
}

type planningSpecLookup map[string]interface{}

func activePlanningSpecRevision(specification *colony.Specification) *colony.SpecRevision {
	if specification == nil {
		return nil
	}
	for index := range specification.Revisions {
		if specification.Revisions[index].ID == specification.CurrentRevisionID {
			return &specification.Revisions[index]
		}
	}
	if len(specification.Revisions) == 0 {
		return nil
	}
	return &specification.Revisions[len(specification.Revisions)-1]
}

func buildPlanningSpecLookup(revision *colony.SpecRevision) (planningSpecLookup, error) {
	lookup := make(planningSpecLookup)
	if revision == nil {
		return lookup, nil
	}
	add := func(kind, id string, payload interface{}) error {
		id = canonicalPlanningText(id)
		if id == "" {
			return nil
		}
		key := kind + "\x00" + id
		if _, duplicate := lookup[key]; duplicate {
			return fmt.Errorf("specification %s stable ID %q is duplicated", kind, id)
		}
		lookup[key] = payload
		return nil
	}
	for _, item := range revision.Requirements {
		if err := add(planningSemanticRequirement, item.ID, struct {
			Description string `json:"description"`
		}{canonicalPlanningText(item.Description)}); err != nil {
			return nil, err
		}
	}
	for _, item := range revision.AcceptanceChecks {
		if err := add(planningSemanticAcceptance, item.ID, struct {
			Description  string `json:"description"`
			Verification string `json:"verification"`
		}{canonicalPlanningText(item.Description), canonicalPlanningText(item.Verification)}); err != nil {
			return nil, err
		}
	}
	for _, item := range revision.NegativeExpectations {
		if err := add(planningSemanticNegative, item.ID, struct {
			Description string `json:"description"`
		}{canonicalPlanningText(item.Description)}); err != nil {
			return nil, err
		}
	}
	for _, item := range revision.RecoveryExpectations {
		if err := add(planningSemanticRecovery, item.ID, struct {
			Description string `json:"description"`
		}{canonicalPlanningText(item.Description)}); err != nil {
			return nil, err
		}
	}
	for _, item := range revision.AffectedPublicPaths {
		if err := add(planningSemanticPublicPath, item.ID, struct {
			Path        string `json:"path"`
			Description string `json:"description"`
		}{canonicalPlanningText(item.Path), canonicalPlanningText(item.Description)}); err != nil {
			return nil, err
		}
	}
	return lookup, nil
}

func appendPlanningNodeSemantics(snapshot *planningSemanticSnapshot, path, ownerID string, requirements, acceptance, negative, recovery, publicPaths, criteria []string, evidenceRequirements []colony.CriterionEvidenceRequirement, lookup planningSpecLookup) error {
	sections := []struct {
		name   string
		values []string
		target *[]planningSemanticEntry
	}{
		{name: planningSemanticRequirement, values: requirements, target: &snapshot.RequirementLinks},
		{name: planningSemanticAcceptance, values: acceptance, target: &snapshot.AcceptanceChecks},
		{name: planningSemanticNegative, values: negative, target: &snapshot.NegativeExpectations},
		{name: planningSemanticRecovery, values: recovery, target: &snapshot.RecoveryExpectations},
		{name: planningSemanticPublicPath, values: publicPaths, target: &snapshot.PublicPaths},
	}
	for _, section := range sections {
		values := canonicalPlanningSet(section.values)
		for index, value := range values {
			if value == "" {
				return fmt.Errorf("%s.%s[%d] is required", path, planningProofField(section.name), index)
			}
			payload := lookup[section.name+"\x00"+value]
			if payload == nil {
				payload = struct {
					StableID string `json:"stable_id"`
				}{StableID: value}
			}
			entry, err := newPlanningSemanticEntry(planningSemanticRelationID(ownerID, section.name, value), struct {
				Owner   string      `json:"owner"`
				Target  string      `json:"target"`
				Content interface{} `json:"content"`
			}{Owner: ownerID, Target: value, Content: payload})
			if err != nil {
				return fmt.Errorf("hash %s.%s[%d]: %w", path, planningProofField(section.name), index, err)
			}
			*section.target = append(*section.target, entry)
		}
	}
	if len(criteria) > 0 || len(evidenceRequirements) > 0 {
		canonicalRequirements := make([]struct {
			Criterion string   `json:"criterion"`
			Artifacts []string `json:"artifacts"`
			Checks    []string `json:"checks"`
		}, 0, len(evidenceRequirements))
		for _, requirement := range evidenceRequirements {
			canonicalRequirements = append(canonicalRequirements, struct {
				Criterion string   `json:"criterion"`
				Artifacts []string `json:"artifacts"`
				Checks    []string `json:"checks"`
			}{
				Criterion: canonicalPlanningText(requirement.Criterion),
				Artifacts: canonicalPlanningSet(requirement.Artifacts),
				Checks:    canonicalPlanningSet(requirement.Checks),
			})
		}
		sort.Slice(canonicalRequirements, func(i, j int) bool {
			left, _ := jsonSHA256(canonicalRequirements[i])
			right, _ := jsonSHA256(canonicalRequirements[j])
			return left < right
		})
		entry, err := newPlanningSemanticEntry(planningSemanticRelationID(ownerID, planningSemanticAcceptance, "local-contract"), struct {
			Criteria     []string    `json:"criteria"`
			Requirements interface{} `json:"evidence_requirements"`
		}{Criteria: canonicalPlanningSet(criteria), Requirements: canonicalRequirements})
		if err != nil {
			return fmt.Errorf("hash %s acceptance contract: %w", path, err)
		}
		snapshot.AcceptanceChecks = append(snapshot.AcceptanceChecks, entry)
	}
	return nil
}

func planningProofField(kind string) string {
	switch kind {
	case planningSemanticRequirement:
		return "requirement_proof_links"
	case planningSemanticAcceptance:
		return "acceptance_proof_links"
	case planningSemanticNegative:
		return "negative_proof_links"
	case planningSemanticRecovery:
		return "recovery_proof_links"
	case planningSemanticPublicPath:
		return "public_path_proof_links"
	default:
		return kind
	}
}

func planningSemanticRelationID(ownerID, kind, targetID string) string {
	return ownerID + "::" + kind + "::" + targetID
}

func uniqueSortedPlanningSemanticEntries(values []planningSemanticEntry) []planningSemanticEntry {
	byID := make(map[string]planningSemanticEntry, len(values))
	for _, value := range values {
		if existing, ok := byID[value.SemanticID]; ok {
			if existing.ContentHash == value.ContentHash {
				continue
			}
			// Conflicting duplicate entries are preserved under a deterministic
			// key so comparison cannot silently discard either meaning. Current
			// proposal validation rejects the source before persistence.
			value.SemanticID += "::conflict::" + value.ContentHash[:12]
		}
		byID[value.SemanticID] = value
	}
	result := make([]planningSemanticEntry, 0, len(byID))
	for _, value := range byID {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SemanticID < result[j].SemanticID })
	return result
}

func comparePlanningSemanticEntries(section colony.PlanningSemanticSection, before, after []planningSemanticEntry) ([]colony.PlanningSemanticChange, error) {
	beforeByID := make(map[string]planningSemanticEntry, len(before))
	afterByID := make(map[string]planningSemanticEntry, len(after))
	ids := make(map[string]struct{}, len(before)+len(after))
	for _, entry := range before {
		beforeByID[entry.SemanticID] = entry
		ids[entry.SemanticID] = struct{}{}
	}
	for _, entry := range after {
		afterByID[entry.SemanticID] = entry
		ids[entry.SemanticID] = struct{}{}
	}
	orderedIDs := make([]string, 0, len(ids))
	for id := range ids {
		orderedIDs = append(orderedIDs, id)
	}
	sort.Strings(orderedIDs)

	changes := make([]colony.PlanningSemanticChange, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		beforeEntry, hadBefore := beforeByID[id]
		afterEntry, hasAfter := afterByID[id]
		change := colony.PlanningSemanticChange{SemanticID: id}
		switch {
		case !hadBefore && hasAfter:
			change.Kind = colony.PlanningSemanticChangeAdded
			change.AfterHash = afterEntry.ContentHash
		case hadBefore && !hasAfter:
			change.Kind = colony.PlanningSemanticChangeRemoved
			change.BeforeHash = beforeEntry.ContentHash
		case beforeEntry.ContentHash != afterEntry.ContentHash:
			change.Kind = colony.PlanningSemanticChangeModified
			change.BeforeHash = beforeEntry.ContentHash
			change.AfterHash = afterEntry.ContentHash
		default:
			change.Kind = colony.PlanningSemanticChangePreserved
			change.BeforeHash = beforeEntry.ContentHash
			change.AfterHash = afterEntry.ContentHash
		}
		if change.Kind == colony.PlanningSemanticChangePreserved {
			if err := colony.AddressPlanningSemanticChange(section, &change); err != nil {
				return nil, fmt.Errorf("address semantic change %s: %w", id, err)
			}
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func buildPlanningAuthoritySnapshot(source planningSemanticSnapshotSource, allSemanticIDs []string) ([]planningAuthorityState, error) {
	states := make([]planningAuthorityState, 0)
	if source.Specification != nil {
		for _, revision := range source.Specification.Revisions {
			if revision.Approval != nil || revision.Status == colony.SpecStatusApproved {
				sourceID := revision.ID
				approvalID := ""
				approvalTokenHash := ""
				approvedBy := ""
				if revision.Approval != nil {
					sourceID = revision.Approval.ID
					approvalID = revision.Approval.ID
					approvalTokenHash = revision.Approval.ApprovalTokenHash
					approvedBy = revision.Approval.ApprovedBy
				}
				hash, err := jsonSHA256(struct {
					RevisionID        string                    `json:"revision_id"`
					RevisionHash      string                    `json:"revision_hash"`
					Status            colony.SpecRevisionStatus `json:"status"`
					ApprovalID        string                    `json:"approval_id"`
					ApprovalTokenHash string                    `json:"approval_token_hash"`
					ApprovedBy        string                    `json:"approved_by"`
				}{revision.ID, revision.ContentHash, revision.Status, approvalID, approvalTokenHash, canonicalPlanningText(approvedBy)})
				if err != nil {
					return nil, err
				}
				states = append(states, planningAuthorityState{
					Key: "specification_approval::" + revision.ID, Kind: colony.PlanningAuthoritySpecApproval,
					SourceID: sourceID, ContentHash: hash, AffectedSemanticIDs: allSemanticIDs,
					Rationale: "The exact specification revision is approved by the owner",
				})
			}
			if revision.Status == colony.SpecStatusSuperseded {
				hash, err := jsonSHA256(struct {
					RevisionID string `json:"revision_id"`
					Successor  string `json:"successor_id"`
				}{revision.ID, source.Specification.CurrentRevisionID})
				if err != nil {
					return nil, err
				}
				states = append(states, planningAuthorityState{
					Key: "specification_supersession::" + revision.ID, Kind: colony.PlanningAuthoritySpecSupersession,
					SourceID: revision.ID, ContentHash: hash, AffectedSemanticIDs: allSemanticIDs,
					Rationale: "The specification revision is superseded by its immutable successor",
				})
			}
		}
	}
	for _, candidate := range source.Plan.Candidates {
		statusHash, err := jsonSHA256(struct {
			CandidateID   string                     `json:"candidate_id"`
			CandidateHash string                     `json:"candidate_hash"`
			Status        colony.PlanCandidateStatus `json:"status"`
		}{candidate.ID, candidate.ContentHash, candidate.Status})
		if err != nil {
			return nil, err
		}
		affected := canonicalPlanningSet(candidate.Proposal.AffectedSemanticIDs)
		if len(affected) == 0 {
			affected = allSemanticIDs
		}
		states = append(states, planningAuthorityState{
			Key: "candidate_status::" + candidate.ID, Kind: colony.PlanningAuthorityCandidateStatus,
			SourceID: candidate.ID, ContentHash: statusHash, AffectedSemanticIDs: affected,
			Rationale: "The exact plan candidate status changed independently of plan semantics",
		})
		if candidate.Acceptance != nil {
			receipt := candidate.Acceptance
			acceptanceHash, err := jsonSHA256(struct {
				ID                    string `json:"id"`
				CandidateID           string `json:"candidate_id"`
				CandidateContentHash  string `json:"candidate_content_hash"`
				SpecificationID       string `json:"specification_revision_id"`
				SpecificationHash     string `json:"specification_revision_hash"`
				BaseRevisionID        string `json:"base_plan_revision_id"`
				BaseRevisionHash      string `json:"base_plan_revision_hash"`
				TimelineID            string `json:"timeline_id"`
				TimelineDigest        string `json:"timeline_digest"`
				ProposalHash          string `json:"proposal_hash"`
				AcceptanceTokenHash   string `json:"acceptance_token_hash"`
				AcceptedBy            string `json:"accepted_by"`
				ActivatedRevisionID   string `json:"activated_plan_revision_id"`
				ActivatedRevisionHash string `json:"activated_plan_revision_hash"`
			}{
				receipt.ID, receipt.CandidateID, receipt.CandidateContentHash,
				receipt.SpecificationRevisionID, receipt.SpecificationRevisionHash,
				receipt.BasePlanRevisionID, receipt.BasePlanRevisionHash,
				receipt.TimelineID, receipt.TimelineDigest, receipt.ProposalHash,
				receipt.AcceptanceTokenHash, canonicalPlanningText(receipt.AcceptedBy),
				receipt.ActivatedPlanRevisionID, receipt.ActivatedPlanRevisionHash,
			})
			if err != nil {
				return nil, err
			}
			states = append(states, planningAuthorityState{
				Key: "plan_acceptance::" + candidate.ID, Kind: colony.PlanningAuthorityPlanAcceptance,
				SourceID: receipt.ID, ContentHash: acceptanceHash, AffectedSemanticIDs: affected,
				Rationale: "The owner accepted the exact candidate without changing executable semantics",
			})
		}
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Key < states[j].Key })
	return states, nil
}

func comparePlanningAuthorityStates(before, after []planningAuthorityState) ([]colony.PlanningAuthorityImpact, error) {
	beforeByKey := make(map[string]planningAuthorityState, len(before))
	afterByKey := make(map[string]planningAuthorityState, len(after))
	keys := make(map[string]struct{}, len(before)+len(after))
	for _, state := range before {
		beforeByKey[state.Key] = state
		keys[state.Key] = struct{}{}
	}
	for _, state := range after {
		afterByKey[state.Key] = state
		keys[state.Key] = struct{}{}
	}
	orderedKeys := make([]string, 0, len(keys))
	for key := range keys {
		orderedKeys = append(orderedKeys, key)
	}
	sort.Strings(orderedKeys)

	impacts := make([]colony.PlanningAuthorityImpact, 0)
	for _, key := range orderedKeys {
		beforeState, hadBefore := beforeByKey[key]
		afterState, hasAfter := afterByKey[key]
		if hadBefore && hasAfter && beforeState.ContentHash == afterState.ContentHash {
			continue
		}
		state := afterState
		if !hasAfter {
			state = beforeState
			state.Rationale = "The prior authority record is no longer present"
		}
		impact := colony.PlanningAuthorityImpact{
			Kind: state.Kind, SourceID: state.SourceID,
			AffectedSemanticIDs: canonicalPlanningSet(state.AffectedSemanticIDs),
			Rationale:           state.Rationale,
		}
		if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
			return nil, fmt.Errorf("address authority impact %s: %w", key, err)
		}
		impacts = append(impacts, impact)
	}
	return impacts, nil
}

func planningSemanticDeltaReadyForAddress(delta colony.PlanningSemanticDelta) bool {
	count := len(delta.AuthorityImpacts)
	for _, section := range [][]colony.PlanningSemanticChange{
		delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks,
		delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths,
	} {
		count += len(section)
		for _, change := range section {
			if strings.TrimSpace(change.ContentHash) == "" {
				return false
			}
		}
	}
	return count > 0
}

func canonicalPlanningText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func canonicalPlanningSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = canonicalPlanningText(value)
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
