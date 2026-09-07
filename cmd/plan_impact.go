package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planImpactClosure is the deterministic, flat authority boundary produced by
// a material specification successor. The category slices make preservation
// and candidate-coverage decisions without inferring node kinds from names.
type planImpactClosure struct {
	SpecificationRevisionID   string   `json:"specification_revision_id"`
	SpecificationRevisionHash string   `json:"specification_revision_hash"`
	ChangedSpecItemIDs        []string `json:"changed_spec_item_ids"`
	AffectedSemanticIDs       []string `json:"affected_semantic_ids"`
	PreservedSemanticIDs      []string `json:"preserved_semantic_ids"`
	AffectedPhaseIDs          []string `json:"affected_phase_ids"`
	AffectedTaskIDs           []string `json:"affected_task_ids"`
	AffectedProofIDs          []string `json:"affected_proof_ids"`
}

type planImpactTask struct {
	id        string
	phaseID   string
	aliases   []string
	dependsOn []string
	links     []string
}

type planImpactPhase struct {
	id    string
	links []string
	tasks []string
}

// computePlanImpactClosure expands one revision delta through requirement and
// proof links, task dependency edges, and containing phases. A task becoming
// affected also invalidates the checks that proved that task; those proof IDs
// can in turn affect another task sharing the same proof contract.
func computePlanImpactClosure(revision colony.SpecRevision, phases []colony.Phase) (planImpactClosure, error) {
	if err := revision.Delta.Validate(); err != nil {
		return planImpactClosure{}, fmt.Errorf("specification delta: %w", err)
	}
	return computePlanImpactFromSeeds(revision, phases, specificationDeltaChangedIDs(revision.Delta), specificationDeltaUnchangedIDs(revision.Delta))
}

func computePlanImpactFromSeeds(revision colony.SpecRevision, phases []colony.Phase, changed, unchanged []string) (planImpactClosure, error) {
	changed = canonicalPlanImpactIDs(changed)
	unchanged = canonicalPlanImpactIDs(unchanged)
	affected := planImpactIDSet(changed)
	all := planImpactIDSet(append(append([]string(nil), changed...), unchanged...))
	proofs := make(map[string]struct{})
	phaseByID := make(map[string]planImpactPhase, len(phases))
	taskByID := make(map[string]planImpactTask)
	aliasToTask := make(map[string]string)

	for phaseIndex := range phases {
		phase := phases[phaseIndex]
		phaseID := planImpactPhaseID(phase)
		if _, duplicate := phaseByID[phaseID]; duplicate {
			return planImpactClosure{}, fmt.Errorf("plan impact contains duplicate phase semantic ID %q", phaseID)
		}
		phaseLinks := planImpactNodeLinks(phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks, phase.RecoveryProofLinks, phase.PublicPathProofLinks, phase.AffectedSemanticIDs)
		entry := planImpactPhase{id: phaseID, links: phaseLinks}
		all[phaseID] = struct{}{}
		planImpactAddIDs(all, phaseLinks)
		planImpactAddIDs(proofs, phaseLinks)
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskID := planImpactTaskID(task, phase.ID, taskIndex)
			if _, duplicate := taskByID[taskID]; duplicate {
				return planImpactClosure{}, fmt.Errorf("plan impact contains duplicate task semantic ID %q", taskID)
			}
			aliases := []string{taskID}
			if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
				aliases = append(aliases, strings.TrimSpace(*task.ID))
			}
			links := planImpactNodeLinks(task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks, task.RecoveryProofLinks, task.PublicPathProofLinks, task.AffectedSemanticIDs)
			taskByID[taskID] = planImpactTask{
				id: taskID, phaseID: phaseID, aliases: canonicalPlanImpactIDs(aliases),
				dependsOn: canonicalPlanImpactIDs(task.DependsOn), links: links,
			}
			for _, alias := range aliases {
				alias = strings.TrimSpace(alias)
				if prior, duplicate := aliasToTask[alias]; duplicate && prior != taskID {
					return planImpactClosure{}, fmt.Errorf("plan impact task alias %q is ambiguous", alias)
				}
				aliasToTask[alias] = taskID
			}
			entry.tasks = append(entry.tasks, taskID)
			all[taskID] = struct{}{}
			planImpactAddIDs(all, links)
			planImpactAddIDs(proofs, links)
		}
		entry.tasks = canonicalPlanImpactIDs(entry.tasks)
		phaseByID[phaseID] = entry
	}

	directPhases := make(map[string]struct{})
	phaseIDs := make([]string, 0, len(phaseByID))
	for phaseID := range phaseByID {
		phaseIDs = append(phaseIDs, phaseID)
	}
	sort.Strings(phaseIDs)
	phaseClosureChanged := true
	for phaseClosureChanged {
		phaseClosureChanged = false
		for _, phaseID := range phaseIDs {
			if _, already := directPhases[phaseID]; already || !planImpactIntersects(affected, phaseByID[phaseID].links) {
				continue
			}
			directPhases[phaseID] = struct{}{}
			affected[phaseID] = struct{}{}
			planImpactAddIDs(affected, phaseByID[phaseID].links)
			phaseClosureChanged = true
		}
	}
	affectedTasks := make(map[string]struct{})
	for taskID, task := range taskByID {
		_, phaseDirect := directPhases[task.phaseID]
		if phaseDirect || planImpactIntersects(affected, task.links) {
			affectedTasks[taskID] = struct{}{}
		}
	}

	changedClosure := true
	for changedClosure {
		changedClosure = false
		for taskID := range affectedTasks {
			before := len(affected)
			affected[taskID] = struct{}{}
			affected[taskByID[taskID].phaseID] = struct{}{}
			planImpactAddIDs(affected, taskByID[taskID].links)
			if len(affected) != before {
				changedClosure = true
			}
		}
		for taskID, task := range taskByID {
			if _, already := affectedTasks[taskID]; already {
				continue
			}
			if planImpactIntersects(affected, task.links) || planImpactDependencyAffected(task.dependsOn, affectedTasks, aliasToTask) {
				affectedTasks[taskID] = struct{}{}
				changedClosure = true
			}
		}
	}

	affectedPhases := make(map[string]struct{})
	for phaseID := range phaseByID {
		if _, ok := affected[phaseID]; ok {
			affectedPhases[phaseID] = struct{}{}
		}
	}
	affectedProofs := make(map[string]struct{})
	for proofID := range proofs {
		if _, ok := affected[proofID]; ok {
			affectedProofs[proofID] = struct{}{}
		}
	}
	preserved := make(map[string]struct{})
	for id := range all {
		if _, invalidated := affected[id]; !invalidated {
			preserved[id] = struct{}{}
		}
	}

	return planImpactClosure{
		SpecificationRevisionID: revision.ID, SpecificationRevisionHash: revision.ContentHash,
		ChangedSpecItemIDs:  canonicalPlanImpactIDs(changed),
		AffectedSemanticIDs: sortedPlanImpactSet(affected), PreservedSemanticIDs: sortedPlanImpactSet(preserved),
		AffectedPhaseIDs: sortedPlanImpactSet(affectedPhases), AffectedTaskIDs: sortedPlanImpactSet(affectedTasks),
		AffectedProofIDs: sortedPlanImpactSet(affectedProofs),
	}, nil
}

// unresolvedPlanImpact compares the active accepted plan's exact specification
// binding with the current lineage. Historical impact fields are not live
// blockers when they match the accepted candidate proposal that recorded them.
func unresolvedPlanImpact(state colony.ColonyState) (planImpactClosure, bool, error) {
	if state.Specification == nil {
		return planImpactClosure{}, false, nil
	}
	active, ok := activePlanRevision(state.Plan)
	if !ok || strings.TrimSpace(active.SpecificationRevisionID) == "" {
		return planImpactClosure{}, false, nil
	}
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		return planImpactClosure{}, false, fmt.Errorf("current specification revision is unavailable")
	}
	if active.SpecificationRevisionID == current.ID && active.SpecificationRevisionHash == current.ContentHash {
		if state.Plan.AcceptancePolicy != colony.PlanAcceptanceExplicitOwner {
			return planImpactClosure{}, false, nil
		}
		markers := canonicalPlanImpactIDs(active.AffectedSemanticIDs)
		if candidate, found := planImpactAcceptedCandidate(state.Plan, active); found {
			markers = planImpactDifference(markers, candidate.Proposal.AffectedSemanticIDs)
			if len(markers) != 0 && validatePlanCandidateImpactCoverage(candidate, planImpactClosure{AffectedSemanticIDs: markers}) == nil {
				markers = nil
			}
		}
		if len(markers) == 0 {
			return planImpactClosure{}, false, nil
		}
		return planImpactClosure{
			SpecificationRevisionID: current.ID, SpecificationRevisionHash: current.ContentHash,
			AffectedSemanticIDs: markers,
		}, true, nil
	}

	from := specificationRevisionIndex(*state.Specification, active.SpecificationRevisionID)
	to := specificationRevisionIndex(*state.Specification, current.ID)
	if from < 0 || to <= from {
		return planImpactClosure{}, false, fmt.Errorf("active plan specification %q is not an ancestor of current revision %q", active.SpecificationRevisionID, current.ID)
	}
	changed := make([]string, 0)
	for index := from + 1; index <= to; index++ {
		changed = append(changed, specificationDeltaChangedIDs(state.Specification.Revisions[index].Delta)...)
	}
	impact, err := computePlanImpactFromSeeds(current, active.Phases, changed, planImpactSpecificationIDs(current))
	if err != nil {
		return planImpactClosure{}, false, err
	}
	return impact, len(impact.AffectedSemanticIDs) > 0, nil
}

func planImpactAcceptedCandidate(plan colony.Plan, revision colony.PlanRevision) (colony.PlanCandidate, bool) {
	for _, candidate := range plan.Candidates {
		if candidate.ID == revision.CandidateID && candidate.Status == colony.PlanCandidateAccepted && candidate.Acceptance != nil &&
			candidate.SpecificationRevisionID == revision.SpecificationRevisionID && candidate.SpecificationRevisionHash == revision.SpecificationRevisionHash {
			return candidate, true
		}
	}
	return colony.PlanCandidate{}, false
}

// validatePlanCandidateImpactCoverage ensures a current-spec candidate either
// contains each affected node/proof or explicitly removes it in its semantic
// delta. Merely binding a newer specification is not reconciliation.
func validatePlanCandidateImpactCoverage(candidate colony.PlanCandidate, impact planImpactClosure) error {
	coverage := make(map[string]struct{})
	addRevision := func(revision colony.PlanRevision) {
		planImpactAddIDs(coverage, append([]string{revision.SemanticID}, revision.RequirementProofLinks...))
		planImpactAddIDs(coverage, revision.AcceptanceProofLinks)
		planImpactAddIDs(coverage, revision.NegativeProofLinks)
		planImpactAddIDs(coverage, revision.RecoveryProofLinks)
		planImpactAddIDs(coverage, revision.PublicPathProofLinks)
		for _, phase := range revision.Phases {
			planImpactAddIDs(coverage, append([]string{planImpactPhaseID(phase)}, phase.RequirementProofLinks...))
			planImpactAddIDs(coverage, phase.AcceptanceProofLinks)
			planImpactAddIDs(coverage, phase.NegativeProofLinks)
			planImpactAddIDs(coverage, phase.RecoveryProofLinks)
			planImpactAddIDs(coverage, phase.PublicPathProofLinks)
			for index, task := range phase.Tasks {
				planImpactAddIDs(coverage, append([]string{planImpactTaskID(task, phase.ID, index)}, task.RequirementProofLinks...))
				planImpactAddIDs(coverage, task.AcceptanceProofLinks)
				planImpactAddIDs(coverage, task.NegativeProofLinks)
				planImpactAddIDs(coverage, task.RecoveryProofLinks)
				planImpactAddIDs(coverage, task.PublicPathProofLinks)
			}
		}
	}
	addRevision(candidate.Proposal)
	for _, section := range planImpactDeltaSections(candidate.SemanticDelta) {
		for _, change := range section {
			if change.Kind != colony.PlanningSemanticChangePreserved {
				planImpactAddIDs(coverage, []string{change.SemanticID})
			}
		}
	}
	for _, authority := range candidate.SemanticDelta.AuthorityImpacts {
		planImpactAddIDs(coverage, authority.AffectedSemanticIDs)
	}
	missing := make([]string, 0)
	for _, id := range impact.AffectedSemanticIDs {
		if _, ok := coverage[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("candidate omits affected semantic IDs: %s", strings.Join(missing, ", "))
	}
	return nil
}

// preserveCompletedCandidateWorkForImpact restores lifecycle completion only
// for semantically compatible nodes outside the affected closure. An affected
// completed phase may be reopened while an independent completed task inside
// it remains credited.
func preserveCompletedCandidateWorkForImpact(previous, proposal []colony.Phase, impact planImpactClosure) ([]colony.Phase, []int, error) {
	result := clonePhases(proposal)
	prefix, err := completedPlanPrefix(previous)
	if err != nil {
		return nil, nil, err
	}
	affected := planImpactIDSet(impact.AffectedSemanticIDs)
	phaseIndexes := make(map[string]int, len(result))
	for index := range result {
		id := planImpactPhaseID(result[index])
		if _, duplicate := phaseIndexes[id]; duplicate {
			return nil, nil, fmt.Errorf("proposal_hash: candidate duplicates phase semantic ID %q", id)
		}
		phaseIndexes[id] = index
	}
	var preservedPhases []int
	for index := 0; index < prefix; index++ {
		prior := previous[index]
		phaseID := planImpactPhaseID(prior)
		candidateIndex, found := phaseIndexes[phaseID]
		_, phaseAffected := affected[phaseID]
		if !found {
			if phaseAffected {
				continue
			}
			return nil, nil, fmt.Errorf("proposal_hash: candidate omits completed compatible phase %d", prior.ID)
		}
		if !phaseAffected {
			priorHash, hashErr := completedPhaseCompatibilityHash(prior)
			if hashErr != nil {
				return nil, nil, hashErr
			}
			candidateHash, hashErr := completedPhaseCompatibilityHash(result[candidateIndex])
			if hashErr != nil {
				return nil, nil, hashErr
			}
			if priorHash != candidateHash {
				return nil, nil, fmt.Errorf("proposal_hash: candidate changes completed phase %d outside affected scope", prior.ID)
			}
			result[candidateIndex].Status = prior.Status
			result[candidateIndex].WatcherFailureCount = prior.WatcherFailureCount
			for taskIndex := range result[candidateIndex].Tasks {
				result[candidateIndex].Tasks[taskIndex].Status = prior.Tasks[taskIndex].Status
			}
			preservedPhases = append(preservedPhases, prior.ID)
			continue
		}

		candidateTasks := make(map[string]int, len(result[candidateIndex].Tasks))
		for taskIndex := range result[candidateIndex].Tasks {
			taskID := planImpactTaskID(result[candidateIndex].Tasks[taskIndex], result[candidateIndex].ID, taskIndex)
			if _, duplicate := candidateTasks[taskID]; duplicate {
				return nil, nil, fmt.Errorf("proposal_hash: candidate duplicates task semantic ID %q", taskID)
			}
			candidateTasks[taskID] = taskIndex
		}
		for taskIndex, priorTask := range prior.Tasks {
			if priorTask.Status != colony.TaskCompleted {
				continue
			}
			taskID := planImpactTaskID(priorTask, prior.ID, taskIndex)
			if _, taskAffected := affected[taskID]; taskAffected {
				continue
			}
			candidateTaskIndex, found := candidateTasks[taskID]
			if !found {
				return nil, nil, fmt.Errorf("proposal_hash: candidate omits completed compatible task %s", taskID)
			}
			priorHash, hashErr := completedTaskCompatibilityHash(priorTask)
			if hashErr != nil {
				return nil, nil, hashErr
			}
			candidateHash, hashErr := completedTaskCompatibilityHash(result[candidateIndex].Tasks[candidateTaskIndex])
			if hashErr != nil {
				return nil, nil, hashErr
			}
			if priorHash != candidateHash {
				return nil, nil, fmt.Errorf("proposal_hash: candidate changes completed task %s outside affected scope", taskID)
			}
			result[candidateIndex].Tasks[candidateTaskIndex].Status = priorTask.Status
		}
	}
	return result, preservedPhases, nil
}

func completedTaskCompatibilityHash(task colony.Task) (string, error) {
	copyTask := cloneTasks([]colony.Task{task})[0]
	copyTask.ID = nil
	copyTask.Status = ""
	copyTask.SpecificationRevisionID = ""
	copyTask.SpecificationRevisionHash = ""
	copyTask.CandidateID = ""
	copyTask.CandidateContentHash = ""
	copyTask.PlanningTimelineID = ""
	copyTask.PlanningTimelineDigest = ""
	copyTask.AffectedSemanticIDs = nil
	copyTask.PreservedSemanticIDs = nil
	return jsonSHA256(copyTask)
}

func planImpactDeltaSections(delta colony.PlanningSemanticDelta) [][]colony.PlanningSemanticChange {
	return [][]colony.PlanningSemanticChange{
		delta.Phases, delta.Tasks, delta.Dependencies, delta.RequirementLinks,
		delta.AcceptanceChecks, delta.NegativeExpectations, delta.RecoveryExpectations, delta.PublicPaths,
	}
}

func specificationDeltaUnchangedIDs(delta colony.SpecRevisionDelta) []string {
	var result []string
	for _, section := range []colony.SpecItemDelta{
		delta.Outcomes, delta.IncludedBehaviors, delta.Exclusions, delta.BindingDecisions,
		delta.Requirements, delta.AcceptanceChecks, delta.NegativeExpectations,
		delta.RecoveryExpectations, delta.AffectedPublicPaths,
	} {
		result = append(result, section.UnchangedIDs...)
	}
	return canonicalPlanImpactIDs(result)
}

func planImpactSpecificationIDs(revision colony.SpecRevision) []string {
	result := make([]string, 0)
	for _, entry := range revision.Outcomes {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.IncludedBehaviors {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.Exclusions {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.BindingDecisions {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.Requirements {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.AcceptanceChecks {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.NegativeExpectations {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.RecoveryExpectations {
		result = append(result, entry.ID)
	}
	for _, entry := range revision.AffectedPublicPaths {
		result = append(result, entry.ID)
	}
	return canonicalPlanImpactIDs(result)
}

func planImpactNodeLinks(groups ...[]string) []string {
	var result []string
	for _, group := range groups {
		result = append(result, group...)
	}
	return canonicalPlanImpactIDs(result)
}

func planImpactPhaseID(phase colony.Phase) string {
	if id := strings.TrimSpace(phase.SemanticID); id != "" {
		return id
	}
	return fmt.Sprintf("phase:%d", phase.ID)
}

func planImpactTaskID(task colony.Task, phaseID, taskIndex int) string {
	if id := strings.TrimSpace(task.SemanticID); id != "" {
		return id
	}
	if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
		return strings.TrimSpace(*task.ID)
	}
	return fmt.Sprintf("%d.%d", phaseID, taskIndex+1)
}

func planImpactDependencyAffected(dependencies []string, affected map[string]struct{}, aliases map[string]string) bool {
	for _, dependency := range dependencies {
		if semanticID, ok := aliases[strings.TrimSpace(dependency)]; ok {
			if _, invalidated := affected[semanticID]; invalidated {
				return true
			}
		}
	}
	return false
}

func planImpactIntersects(set map[string]struct{}, values []string) bool {
	for _, value := range values {
		if _, ok := set[strings.TrimSpace(value)]; ok {
			return true
		}
	}
	return false
}

func planImpactAddIDs(destination map[string]struct{}, values []string) {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			destination[value] = struct{}{}
		}
	}
}

func planImpactIDSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	planImpactAddIDs(result, values)
	return result
}

func canonicalPlanImpactIDs(values []string) []string {
	return sortedPlanImpactSet(planImpactIDSet(values))
}

func sortedPlanImpactSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	if result == nil {
		return []string{}
	}
	return result
}

func planImpactDifference(left, right []string) []string {
	rightSet := planImpactIDSet(right)
	result := make([]string, 0, len(left))
	for _, value := range canonicalPlanImpactIDs(left) {
		if _, found := rightSet[value]; !found {
			result = append(result, value)
		}
	}
	return result
}
