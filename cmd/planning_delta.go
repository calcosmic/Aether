package cmd

import (
	"fmt"
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
		before []planningSemanticEntry
		after  []planningSemanticEntry
		set    func([]colony.PlanningSemanticChange)
	}{
		{before: before.Phases, after: after.Phases, set: func(value []colony.PlanningSemanticChange) { delta.Phases = value }},
		{before: before.Tasks, after: after.Tasks, set: func(value []colony.PlanningSemanticChange) { delta.Tasks = value }},
		{before: before.Dependencies, after: after.Dependencies, set: func(value []colony.PlanningSemanticChange) { delta.Dependencies = value }},
		{before: before.RequirementLinks, after: after.RequirementLinks, set: func(value []colony.PlanningSemanticChange) { delta.RequirementLinks = value }},
		{before: before.AcceptanceChecks, after: after.AcceptanceChecks, set: func(value []colony.PlanningSemanticChange) { delta.AcceptanceChecks = value }},
		{before: before.NegativeExpectations, after: after.NegativeExpectations, set: func(value []colony.PlanningSemanticChange) { delta.NegativeExpectations = value }},
		{before: before.RecoveryExpectations, after: after.RecoveryExpectations, set: func(value []colony.PlanningSemanticChange) { delta.RecoveryExpectations = value }},
		{before: before.PublicPaths, after: after.PublicPaths, set: func(value []colony.PlanningSemanticChange) { delta.PublicPaths = value }},
	}
	for _, section := range sections {
		var changes []colony.PlanningSemanticChange
		changes, err = comparePlanningSemanticEntries(section.before, section.after)
		if err != nil {
			return colony.PlanningSemanticDelta{}, err
		}
		section.set(changes)
	}
	delta.AuthorityImpacts, err = comparePlanningAuthorityStates(before.Authority, after.Authority)
	if err != nil {
		return colony.PlanningSemanticDelta{}, err
	}

	payload := delta
	payload.ID = ""
	payload.ContentHash = ""
	delta.ContentHash, err = jsonSHA256(payload)
	if err != nil {
		return colony.PlanningSemanticDelta{}, fmt.Errorf("hash planning semantic delta: %w", err)
	}
	delta.ID = "planning-delta-" + delta.ContentHash[:12]
	if err := delta.Validate(); err != nil {
		return colony.PlanningSemanticDelta{}, fmt.Errorf("validate planning semantic delta: %w", err)
	}
	return delta, nil
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

func comparePlanningSemanticEntries(before, after []planningSemanticEntry) ([]colony.PlanningSemanticChange, error) {
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
		var err error
		change.ContentHash, err = jsonSHA256(struct {
			SemanticID string                            `json:"semantic_id"`
			Kind       colony.PlanningSemanticChangeKind `json:"kind"`
			BeforeHash string                            `json:"before_hash"`
			AfterHash  string                            `json:"after_hash"`
		}{change.SemanticID, change.Kind, change.BeforeHash, change.AfterHash})
		if err != nil {
			return nil, fmt.Errorf("hash semantic change %s: %w", id, err)
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
		transitionHash, err := jsonSHA256(struct {
			Key        string `json:"key"`
			BeforeHash string `json:"before_hash"`
			AfterHash  string `json:"after_hash"`
		}{key, beforeState.ContentHash, afterState.ContentHash})
		if err != nil {
			return nil, fmt.Errorf("hash authority impact %s: %w", key, err)
		}
		impacts = append(impacts, colony.PlanningAuthorityImpact{
			ID: "authority-impact-" + transitionHash[:12], ContentHash: transitionHash,
			Kind: state.Kind, SourceID: state.SourceID,
			AffectedSemanticIDs: canonicalPlanningSet(state.AffectedSemanticIDs),
			Rationale:           state.Rationale,
		})
	}
	return impacts, nil
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
