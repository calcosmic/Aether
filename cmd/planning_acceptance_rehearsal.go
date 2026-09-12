package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningRouteAcceptanceRehearsal is the check drafting runs, before anything
// commits, to prove the owner will be able to accept the plan they are about
// to be shown. It is a variable only so a test can prove that a failing
// rehearsal blocks the commit.
var planningRouteAcceptanceRehearsal = rehearsePlanningRouteAcceptance

// rehearsePlanningRouteAcceptance runs acceptance's own worker-content checks
// on the exact phases the candidate will carry: after Go restores the completed
// prefix and renumbers the tasks, both of which happen after the proposal
// contract was checked. Bindings Go stamps itself -- candidate, timeline and
// revision chain -- are not rehearsed: they cannot be wrong by construction,
// and they do not exist yet.
//
// Without this, a plan that passed drafting but failed acceptance reached the
// owner as a finished candidate with no revise path (CosmicDashboard,
// plan-candidate-dbf1e0e37a9e, 2026-09-12).
func rehearsePlanningRouteAcceptance(plan colony.Plan, specification *colony.Specification, proposal planningRoutePlanProposal) error {
	if specification == nil {
		return fmt.Errorf("specification is required")
	}
	current, ok := currentSpecificationRevision(*specification)
	if !ok {
		return fmt.Errorf("specification has no current revision")
	}
	input, _, err := planningRouteCandidateProposalInput(plan, proposal.Phases)
	if err != nil {
		return err
	}
	phases := planningRouteRenumberPhases(input)
	requirements, acceptance, negative, recovery, publicPaths := planningRouteProposalProofLinks(phases)
	if err := validatePlanWideProofLinks("active revision", requirements, acceptance, negative, recovery, publicPaths, current); err != nil {
		return err
	}
	for phaseIndex := range phases {
		phase := phases[phaseIndex]
		label := fmt.Sprintf("phases[%d]", phaseIndex)
		if err := validateProofLinks(label, phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks, phase.RecoveryProofLinks, phase.PublicPathProofLinks, current); err != nil {
			return err
		}
		for taskIndex := range phase.Tasks {
			task := phase.Tasks[taskIndex]
			taskLabel := fmt.Sprintf("%s.tasks[%d]", label, taskIndex)
			if err := validateProofLinks(taskLabel, task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks, task.RecoveryProofLinks, task.PublicPathProofLinks, current); err != nil {
				return err
			}
		}
	}

	// Acceptance re-derives the plan's semantics from these phases, which is
	// where a dependency stranded by renumbering surfaces.
	revision := colony.PlanRevision{
		SemanticID:            proposal.SemanticID,
		RequirementProofLinks: requirements, AcceptanceProofLinks: acceptance,
		NegativeProofLinks: negative, RecoveryProofLinks: recovery, PublicPathProofLinks: publicPaths,
		Phases: phases,
	}
	spec := *specification
	if _, err := buildPlanningSemanticSnapshot(planningSemanticSnapshotSource{
		CurrentSchema: true,
		Plan:          colony.Plan{AcceptancePolicy: colony.PlanAcceptanceExplicitOwner, Phases: clonePhases(phases)},
		Revision:      &revision,
		Specification: &spec,
	}); err != nil {
		return fmt.Errorf("derive proposal semantics: %w", err)
	}
	return nil
}

// validateProposalLinkSpelling refuses a proof-link ID sent with stray
// whitespace. Drafting trims IDs before resolving them, but the candidate
// stores them as sent and acceptance compares them exactly, so a padded ID
// would reach the owner as a plan acceptance refuses. Like the rehearsal it
// judges fresh submissions only; a prior pass cannot be repaired.
func validateProposalLinkSpelling(phases []colony.Phase) error {
	check := func(path string, links map[string][]string) error {
		for _, kind := range planningProofLinkKinds() {
			for index, raw := range links[kind] {
				if canonicalPlanningText(raw) != raw {
					return fmt.Errorf("%s.%s[%d] %q must be copied exactly as the approved specification spells it, without extra spaces", path, planningProofField(kind), index, raw)
				}
			}
		}
		return nil
	}
	for phaseIndex, phase := range phases {
		path := fmt.Sprintf("phases[%d]", phaseIndex)
		if err := check(path, planningProofLinkSets(phase.RequirementProofLinks, phase.AcceptanceProofLinks, phase.NegativeProofLinks, phase.RecoveryProofLinks, phase.PublicPathProofLinks)); err != nil {
			return err
		}
		for taskIndex, task := range phase.Tasks {
			taskPath := fmt.Sprintf("%s.tasks[%d]", path, taskIndex)
			if err := check(taskPath, planningProofLinkSets(task.RequirementProofLinks, task.AcceptanceProofLinks, task.NegativeProofLinks, task.RecoveryProofLinks, task.PublicPathProofLinks)); err != nil {
				return err
			}
		}
	}
	return nil
}
