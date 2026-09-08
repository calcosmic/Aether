package cmd

import (
	"fmt"
	"reflect"
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

// derivedPlanCandidateAuthority is computed from repository facts rather than
// the candidate's persisted assertions. SemanticDelta is the complete
// base-to-proposal comparison; SpecificationImpact isolates successor-spec
// invalidation so a final iteration's affected/preserved partition can be
// checked without granting authority to candidate-authored scope markers.
type derivedPlanCandidateAuthority struct {
	SemanticDelta       colony.PlanningSemanticDelta
	Impact              planImpactClosure
	SpecificationImpact planImpactClosure
	SemanticIDs         []string
}

// derivePlanCandidateAuthority is the pure acceptance/build/run derivation.
// Callers must supply values loaded under one repository authority session;
// this function performs no I/O and never consults candidate.SemanticDelta or
// candidate proposal impact markers as proof.
func derivePlanCandidateAuthority(base colony.PlanRevision, specification colony.Specification, proposal colony.PlanRevision) (derivedPlanCandidateAuthority, error) {
	empty := derivedPlanCandidateAuthority{}
	if err := validateCanonicalSpecificationState(specification); err != nil {
		return empty, fmt.Errorf("derived specification: %w", err)
	}
	current, ok := currentSpecificationRevision(specification)
	if !ok || current.Status != colony.SpecStatusApproved || current.Approval == nil {
		return empty, fmt.Errorf("derived specification: current revision is not approved")
	}
	if proposal.SpecificationRevisionID != current.ID || proposal.SpecificationRevisionHash != current.ContentHash {
		return empty, fmt.Errorf("derived proposal: specification binding is not current")
	}
	if err := validateStandalonePlanRevision(proposal); err != nil {
		return empty, fmt.Errorf("derived proposal: %w", err)
	}

	beforeSpecification, err := planCandidateBaseSpecification(specification, base)
	if err != nil {
		return empty, err
	}
	beforeSource := planningSemanticSnapshotSource{Plan: colony.Plan{Phases: clonePhases(base.Phases)}, Specification: beforeSpecification}
	if strings.TrimSpace(base.ID) != "" && base.ID != "plan-unbound" {
		beforeCopy := base
		beforeSource.Revision = &beforeCopy
		beforeSource.CurrentSchema = strings.TrimSpace(base.SemanticID) != ""
	}
	before, err := buildPlanningSemanticSnapshot(beforeSource)
	if err != nil {
		return empty, fmt.Errorf("derive base plan semantics: %w", err)
	}
	afterSpecification := specification
	after, err := buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		CurrentSchema: true,
		Plan:          colony.Plan{AcceptancePolicy: colony.PlanAcceptanceExplicitOwner, Phases: clonePhases(proposal.Phases)},
		Revision:      &proposal,
		Specification: &afterSpecification,
	})
	if err != nil {
		return empty, fmt.Errorf("derive proposal semantics: %w", err)
	}
	delta, err := comparePlanningSemanticSnapshots(before, after)
	if err != nil {
		return empty, fmt.Errorf("derive proposal semantic delta: %w", err)
	}

	semanticIDs := make([]string, 0)
	changedSemanticIDs := make([]string, 0)
	for _, section := range derivedPlanSemanticSections(delta) {
		for _, change := range section.changes {
			// PlanRevision aggregates child proof links after Route validation;
			// those root-level convenience links were never iteration-delta
			// nodes and therefore do not participate in affected markers.
			if strings.HasPrefix(change.SemanticID, proposal.SemanticID+"::") {
				continue
			}
			semanticIDs = append(semanticIDs, change.SemanticID)
			if change.Kind != colony.PlanningSemanticChangePreserved {
				changedSemanticIDs = append(changedSemanticIDs, change.SemanticID)
			}
		}
	}
	semanticIDs = canonicalPlanImpactIDs(semanticIDs)

	specificationImpact, err := derivePlanCandidateSpecificationImpact(base, specification, current, proposal.Phases)
	if err != nil {
		return empty, err
	}
	seeds := canonicalPlanImpactIDs(append(changedSemanticIDs, specificationImpact.ChangedSpecItemIDs...))
	impact, err := computePlanImpactFromSeeds(current, proposal.Phases, seeds, planImpactSpecificationIDs(current))
	if err != nil {
		return empty, fmt.Errorf("derive proposal impact closure: %w", err)
	}
	return derivedPlanCandidateAuthority{
		SemanticDelta: delta, Impact: impact, SpecificationImpact: specificationImpact, SemanticIDs: semanticIDs,
	}, nil
}

type derivedPlanSemanticSection struct {
	name    colony.PlanningSemanticSection
	changes []colony.PlanningSemanticChange
}

func derivedPlanSemanticSections(delta colony.PlanningSemanticDelta) []derivedPlanSemanticSection {
	return []derivedPlanSemanticSection{
		{name: colony.PlanningSemanticSectionPhases, changes: delta.Phases},
		{name: colony.PlanningSemanticSectionTasks, changes: delta.Tasks},
		{name: colony.PlanningSemanticSectionDependencies, changes: delta.Dependencies},
		{name: colony.PlanningSemanticSectionRequirementLinks, changes: delta.RequirementLinks},
		{name: colony.PlanningSemanticSectionAcceptanceChecks, changes: delta.AcceptanceChecks},
		{name: colony.PlanningSemanticSectionNegativeExpectations, changes: delta.NegativeExpectations},
		{name: colony.PlanningSemanticSectionRecoveryExpectations, changes: delta.RecoveryExpectations},
		{name: colony.PlanningSemanticSectionPublicPaths, changes: delta.PublicPaths},
	}
}

func planCandidateBaseSpecification(specification colony.Specification, base colony.PlanRevision) (*colony.Specification, error) {
	if strings.TrimSpace(base.SpecificationRevisionID) == "" {
		copy := specification
		return &copy, nil
	}
	index := specificationRevisionIndex(specification, base.SpecificationRevisionID)
	if index < 0 {
		return nil, fmt.Errorf("derived base: specification revision %q is not retained", base.SpecificationRevisionID)
	}
	historical := specification
	historical.Revisions = append([]colony.SpecRevision(nil), specification.Revisions[:index+1]...)
	historical.CurrentRevisionID = base.SpecificationRevisionID
	last := &historical.Revisions[len(historical.Revisions)-1]
	if last.ContentHash != base.SpecificationRevisionHash || last.Approval == nil {
		return nil, fmt.Errorf("derived base: specification binding is not exact approved history")
	}
	last.Status = colony.SpecStatusApproved
	return &historical, nil
}

func derivePlanCandidateSpecificationImpact(base colony.PlanRevision, specification colony.Specification, current colony.SpecRevision, phases []colony.Phase) (planImpactClosure, error) {
	if strings.TrimSpace(base.SpecificationRevisionID) == "" || base.SpecificationRevisionID == current.ID {
		return planImpactClosure{SpecificationRevisionID: current.ID, SpecificationRevisionHash: current.ContentHash}, nil
	}
	from := specificationRevisionIndex(specification, base.SpecificationRevisionID)
	to := specificationRevisionIndex(specification, current.ID)
	if from < 0 || to <= from {
		return planImpactClosure{}, fmt.Errorf("derived specification impact: base revision %q is not an ancestor of %q", base.SpecificationRevisionID, current.ID)
	}
	changed := make([]string, 0)
	for index := from + 1; index <= to; index++ {
		changed = append(changed, specificationDeltaChangedIDs(specification.Revisions[index].Delta)...)
	}
	impact, err := computePlanImpactFromSeeds(current, phases, changed, planImpactSpecificationIDs(current))
	if err != nil {
		return planImpactClosure{}, fmt.Errorf("derived specification impact: %w", err)
	}
	return impact, nil
}

// validateDerivedPlanCandidateAuthority checks the complete stopped iteration
// against independently rebuilt proposal semantics. The immutable final card
// supplies evidence attribution and iteration-relative classifications; it
// cannot introduce a semantic ID or after-hash absent from the pure derivation.
func validateDerivedPlanCandidateAuthority(candidate colony.PlanCandidate, finalCard colony.PlanningIterationCard, derived derivedPlanCandidateAuthority) error {
	if !reflect.DeepEqual(candidate.SemanticDelta.Phases, finalCard.SemanticDelta.Phases) ||
		!reflect.DeepEqual(candidate.SemanticDelta.Tasks, finalCard.SemanticDelta.Tasks) ||
		!reflect.DeepEqual(candidate.SemanticDelta.Dependencies, finalCard.SemanticDelta.Dependencies) ||
		!reflect.DeepEqual(candidate.SemanticDelta.RequirementLinks, finalCard.SemanticDelta.RequirementLinks) ||
		!reflect.DeepEqual(candidate.SemanticDelta.AcceptanceChecks, finalCard.SemanticDelta.AcceptanceChecks) ||
		!reflect.DeepEqual(candidate.SemanticDelta.NegativeExpectations, finalCard.SemanticDelta.NegativeExpectations) ||
		!reflect.DeepEqual(candidate.SemanticDelta.RecoveryExpectations, finalCard.SemanticDelta.RecoveryExpectations) ||
		!reflect.DeepEqual(candidate.SemanticDelta.PublicPaths, finalCard.SemanticDelta.PublicPaths) {
		return fmt.Errorf("derived semantic delta does not match the immutable final iteration")
	}

	allowedAuthorities := append([]colony.PlanningAuthorityImpact(nil), finalCard.SemanticDelta.AuthorityImpacts...)
	if len(candidate.SemanticDelta.AuthorityImpacts) == len(allowedAuthorities)+1 {
		expected, err := derivedPlanCandidateApprovalImpact(candidate)
		if err != nil {
			return err
		}
		allowedAuthorities = append(allowedAuthorities, expected)
	}
	if len(candidate.SemanticDelta.AuthorityImpacts) != len(allowedAuthorities) ||
		(len(allowedAuthorities) > 0 && !reflect.DeepEqual(candidate.SemanticDelta.AuthorityImpacts, allowedAuthorities)) {
		return fmt.Errorf("derived authority impacts do not match approved specification authority: candidate=%+v allowed=%+v", candidate.SemanticDelta.AuthorityImpacts, allowedAuthorities)
	}

	expected := make(map[string]colony.PlanningSemanticChange)
	for _, section := range derivedPlanSemanticSections(derived.SemanticDelta) {
		for _, change := range section.changes {
			expected[string(section.name)+"\x00"+change.SemanticID] = change
		}
	}
	for _, section := range derivedPlanSemanticSections(finalCard.SemanticDelta) {
		for _, change := range section.changes {
			truth, ok := expected[string(section.name)+"\x00"+change.SemanticID]
			if !ok {
				return fmt.Errorf("derived %s has no semantic node %q", section.name, change.SemanticID)
			}
			if change.Kind == colony.PlanningSemanticChangeRemoved {
				if truth.Kind != colony.PlanningSemanticChangeRemoved || truth.BeforeHash != change.BeforeHash {
					return fmt.Errorf("derived %s removal %q is false", section.name, change.SemanticID)
				}
				continue
			}
			// Phase/task Route snapshots additionally bind generation-only task
			// declarations (files and user-facing classification) that are not
			// duplicated into PlanRevision. Their stable IDs and complete final
			// card bytes remain verified here; proof/dependency hashes are fully
			// reproducible from the accepted tuple and must match exactly.
			reproducibleHash := section.name != colony.PlanningSemanticSectionPhases && section.name != colony.PlanningSemanticSectionTasks
			if truth.AfterHash == "" || (reproducibleHash && truth.AfterHash != change.AfterHash) {
				return fmt.Errorf("derived %s after-hash for %q does not match proposal", section.name, change.SemanticID)
			}
			if truth.Kind == colony.PlanningSemanticChangePreserved && change.Kind != colony.PlanningSemanticChangePreserved {
				return fmt.Errorf("derived %s semantic node %q is preserved, not changed", section.name, change.SemanticID)
			}
		}
	}

	affected, preserved := derivedPlanCandidateScope(finalCard.SemanticDelta, derived)
	if !reflect.DeepEqual(canonicalPlanImpactIDs(candidate.Proposal.AffectedSemanticIDs), affected) ||
		!reflect.DeepEqual(canonicalPlanImpactIDs(candidate.Proposal.PreservedSemanticIDs), preserved) {
		return fmt.Errorf("derived affected/preserved scope does not match proposal markers: candidate=%v/%v derived=%v/%v", candidate.Proposal.AffectedSemanticIDs, candidate.Proposal.PreservedSemanticIDs, affected, preserved)
	}
	return nil
}

func derivedPlanCandidateScope(finalDelta colony.PlanningSemanticDelta, derived derivedPlanCandidateAuthority) ([]string, []string) {
	affected, finalPreserved := planningRouteDeltaSemanticIDs(finalDelta)
	// Root-level aggregate proof nodes are generation conveniences rather than
	// universal proposal nodes. Include them only when the immutable final card
	// actually classified them (phase insertion does; normal Route output does
	// not), so neither candidate form gains or loses synthetic scope.
	universe := planImpactIDSet(append(append([]string(nil), derived.SemanticIDs...), append(affected, finalPreserved...)...))
	for _, id := range derived.SpecificationImpact.AffectedSemanticIDs {
		if _, belongs := universe[id]; belongs {
			affected = append(affected, id)
		}
	}
	affected = canonicalPlanImpactIDs(affected)
	return affected, planImpactDifference(sortedPlanImpactSet(universe), affected)
}

func derivedPlanCandidateApprovalImpact(candidate colony.PlanCandidate) (colony.PlanningAuthorityImpact, error) {
	impact := colony.PlanningAuthorityImpact{
		Kind: colony.PlanningAuthoritySpecApproval, SourceID: candidate.SpecificationRevisionID,
		AffectedSemanticIDs: canonicalPlanImpactIDs(candidate.Proposal.AffectedSemanticIDs),
		Rationale:           "The owner-approved specification authorizes this exact review payload.",
	}
	if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
		return colony.PlanningAuthorityImpact{}, fmt.Errorf("derive specification approval impact: %w", err)
	}
	return impact, nil
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
		if len(markers) == 0 {
			return planImpactClosure{}, false, nil
		}
		candidate, found := planImpactAcceptedCandidate(state.Plan, active)
		if !found {
			return planImpactClosure{SpecificationRevisionID: current.ID, SpecificationRevisionHash: current.ContentHash, AffectedSemanticIDs: markers}, true, nil
		}
		if err := validatePlanningRecordHashes(candidate); err != nil {
			return planImpactClosure{}, false, fmt.Errorf("accepted candidate canonical identity: %w", err)
		}
		base := colony.PlanRevision{ID: candidate.BasePlanRevisionID, PlanHash: candidate.BasePlanRevisionHash}
		if candidate.BasePlanRevisionID != "plan-unbound" {
			var ok bool
			base, ok = planImpactRevisionByID(state.Plan.Revisions, candidate.BasePlanRevisionID)
			if !ok || base.PlanHash != candidate.BasePlanRevisionHash {
				return planImpactClosure{}, false, fmt.Errorf("accepted candidate base revision is unavailable")
			}
		}
		derived, err := derivePlanCandidateAuthority(base, *state.Specification, candidate.Proposal)
		if err != nil {
			return planImpactClosure{}, false, fmt.Errorf("derive accepted plan impact: %w", err)
		}
		independentlyAffected := planImpactIDSet(derived.Impact.AffectedSemanticIDs)
		unexplained := make([]string, 0)
		for _, marker := range markers {
			if _, ok := independentlyAffected[marker]; !ok {
				unexplained = append(unexplained, marker)
			}
		}
		if len(unexplained) == 0 {
			return planImpactClosure{}, false, nil
		}
		return planImpactClosure{
			SpecificationRevisionID: current.ID, SpecificationRevisionHash: current.ContentHash,
			AffectedSemanticIDs: unexplained,
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

func planImpactRevisionByID(revisions []colony.PlanRevision, id string) (colony.PlanRevision, bool) {
	for index := range revisions {
		if revisions[index].ID == id {
			return revisions[index], true
		}
	}
	return colony.PlanRevision{}, false
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
