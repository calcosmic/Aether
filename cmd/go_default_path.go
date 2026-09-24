package cmd

// Phase 209 plan 04, Task 1 -- planning reached through the single door
// (`/ant-go`) never stops for a missing specification (UED-17).
//
// requireApprovedPlanningSpecification (cmd/codex_plan.go) refuses planning
// outright when a project has no approved specification. That refusal is
// exactly right for `/ant-plan` typed directly -- the owner may still want
// to clarify intent or hand-approve a specification first -- but it is the
// wrong answer for the single door's big route, whose whole point is goal,
// one planning pass, build, with nothing typed in between. This file bridges
// that gap: ensureGoPlanningSpecification derives and approves a
// specification from the owner's own sentence when one is missing, so the
// planning pass that follows never refuses. An owner-approved specification
// is never touched -- the very first thing this function does is check
// whether one already exists, and if so it returns immediately having
// written nothing (behavior 4 in this plan's <behavior> block).
//
// Every identifier, hash, and approval token this file produces is derived
// the way the runtime already derives it -- pendingDecisionGoalHash for the
// goal identity, colony.CanonicalSpecificationApprovalToken for the approval
// token, and the specification machinery's own revision/content-hash
// addressing for everything else. Nothing here is a hand-typed literal.

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// goDerivedSpecificationLineage is the stable semantic lineage this file
// uses for the one outcome item it derives from the owner's own sentence.
// It is deliberately distinct from discuss.go's own "accepted-goal" lineage
// so a specification this file derived is never confused, in its own stable
// ID, with one the owner settled through /ant-discuss.
const goDerivedSpecificationLineage = "go-derived-outcome"

// goDefaultPlanningPreset (Task 2) names the shallowest planning preset the
// single door asks for by default -- "fast" (target confidence 80%, at most
// 4 passes; planningPresetPolicies, cmd/codex_plan.go). UED-17 asks for one
// planning pass with no depth question put to the owner: passing this named
// preset to `aether plan --preset` makes resolvePlanningPreset return
// PresetRequired=false immediately, so the planning pass the single door's
// big route hands off to never puts a depth choice to the owner. A deeper
// round stays available by asking for it explicitly (a different named
// preset, or --target/--max-iterations); this constant only changes what
// the single door itself asks for without being told otherwise.
const goDefaultPlanningPreset = "fast"

// The two values ensureGoPlanningSpecification's "specification_source"
// result field (cmd/go_cmd.go) can carry. "Owner-approved" also covers the
// no-op replay case: once a derived specification is approved, a second
// call with the same sentence finds it already approved and reports the
// same "nothing to do, it is already settled" answer an owner's own
// approval would -- the distinction that matters to this result is whether
// anything changed on this call, not who approved it originally. The
// permanent record (the approval receipt's ApprovedBy field) still says
// plainly, and permanently, that a derived approval came from the request
// rather than a considered sitting.
const (
	goSpecificationSourceOwnerApproved      = "owner-approved"
	goSpecificationSourceDerivedFromRequest = "derived-from-request"
)

// goAcceptedGoalEvidenceOrigin names the recorded state fact this file cites
// as evidence for the outcome item it derives -- the same "state:accepted-charter"
// origin primePlanningStartEvidence (cmd/codex_plan.go) already uses for the
// accepted charter/goal `aether init` records. The exact identity (the
// episode ID) is always read from state in goAcceptedGoalEvidenceID, never
// typed.
const goAcceptedGoalEvidenceOrigin = "state:accepted-charter"

// ensureGoPlanningSpecification is the single door's bridge over
// requireApprovedPlanningSpecification's refusal: a project with a recorded
// goal and no approved specification gets one, derived from the owner's own
// sentence, so the planning pass that follows can start immediately. It
// returns which source the specification came from -- the owner's own
// approval (untouched by this call), or derived from this request.
//
// aether init never creates a specification (only an accepted charter); a
// project reached purely through init-then-go therefore has no
// specification at all the first time this runs. This function derives a
// full draft in that case (createGoDerivedSpecificationDraft) rather than
// refusing -- the same "derive what is needed and carry on" standing rule
// this phase's CONTEXT.md states. When a specification already exists (for
// example a partial /ant-discuss draft that was never approved), it adds
// one outcome item to the existing lineage instead (addGoDerivedSpecificationOutcome).
// Either way, the result is approved through runSpecCommand exactly once.
func ensureGoPlanningSpecification(root, job string) (string, error) {
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return "", fmt.Errorf("load colony state for specification: %w", err)
	}

	if _, approvedErr := requireApprovedPlanningSpecification(root, state); approvedErr == nil {
		// Already approved and bound to this session -- owner-approved or a
		// prior derived approval, either way there is nothing to add and
		// nothing is touched.
		return goSpecificationSourceOwnerApproved, nil
	}

	evidenceID := goAcceptedGoalEvidenceID(state)

	var specificationID, revisionID, contentHash string
	if state.Specification == nil {
		specificationID, revisionID, contentHash, err = createGoDerivedSpecificationDraft(root, state, job, evidenceID)
	} else {
		specificationID, revisionID, contentHash, err = addGoDerivedSpecificationOutcome(root, job, evidenceID)
	}
	if err != nil {
		return "", fmt.Errorf("derive specification for %q: %w", job, err)
	}

	approvalToken := colony.CanonicalSpecificationApprovalToken(specificationID, revisionID, contentHash)
	if _, err := runSpecCommand(root, specCommandOperationApprove, specCommandOptions{
		Approve:       true,
		RevisionID:    revisionID,
		RevisionHash:  contentHash,
		ApprovalToken: approvalToken,
		ApprovedBy:    "aether go: " + job,
	}); err != nil {
		return "", fmt.Errorf("approve derived specification for %q: %w", job, err)
	}

	return goSpecificationSourceDerivedFromRequest, nil
}

// goAcceptedGoalEvidenceID names the recorded accepted goal as this file's
// evidence for the item it derives, following the exact origin convention
// primePlanningStartEvidence already uses for the accepted charter. The
// episode ID that makes the reference concrete is read from state, never
// typed as a literal.
func goAcceptedGoalEvidenceID(state colony.ColonyState) string {
	if state.AcceptedCharter != nil {
		if episodeID := strings.TrimSpace(state.AcceptedCharter.EpisodeID); episodeID != "" {
			return goAcceptedGoalEvidenceOrigin + ":" + episodeID
		}
	}
	return goAcceptedGoalEvidenceOrigin
}

// createGoDerivedSpecificationDraft derives a full initial specification
// draft from the owner's own sentence when a project has none at all --
// aether init records an accepted charter and a goal, but never a
// specification, so this is the ordinary first-run shape for a project
// reached purely through init-then-go. Every one of the nine owner-readable
// body sections must carry at least one item for the draft to validate
// (buildSpecificationDraft, cmd/specification.go); the outcome item alone
// carries the owner's exact sentence, and every other section names, in
// plain language, that its content was derived from that same sentence
// rather than settled by hand -- never a silent placeholder. The write goes
// through createSpecificationDraft, the same specification-writing engine
// runSpecCommand's own operations use (specificationStateProjectionTargets,
// commitSpecificationTargets): this is the specification machinery's other
// entry point, for the one case (nothing exists yet) runSpecCommand's own
// CLI operations do not cover, never a second, competing implementation.
func createGoDerivedSpecificationDraft(root string, state colony.ColonyState, job, evidenceID string) (string, string, string, error) {
	goalID := pendingDecisionGoalHash(derefGoal(state.Goal))
	if goalID == "" {
		return "", "", "", fmt.Errorf("cannot derive a specification without a recorded goal")
	}
	sessionID := "session-" + goalID[:16]
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
		sessionID = strings.TrimSpace(*state.SessionID)
	}

	filler := func(lineage, description string) specificationItemInput {
		return specificationItemInput{Lineage: lineage, Description: description, EvidenceIDs: []string{evidenceID}}
	}

	request := specificationDraftRequest{
		Scope: colony.SpecScope{
			Kind:      colony.SpecScopeWholeGoal,
			GoalID:    goalID,
			SessionID: sessionID,
		},
		Outcomes: []specificationItemInput{
			{Lineage: goDerivedSpecificationLineage, Description: job, EvidenceIDs: []string{evidenceID}},
		},
		IncludedBehaviors: []specificationItemInput{
			filler("go-derived-behavior", "Implement only behavior needed to satisfy the requested job: "+job),
		},
		Exclusions: []specificationItemInput{
			filler("go-derived-exclusion", "Behavior outside the requested job is excluded unless the owner revises this specification."),
		},
		BindingDecisions: []specificationItemInput{
			filler("go-derived-authority", "This specification was derived from the owner's own request rather than a considered approval; only the owner may approve or revise it further."),
		},
		Requirements: []specificationItemInput{
			filler("go-derived-requirement", "Required result: "+job),
		},
		AcceptanceChecks: []specificationItemInput{
			{
				Lineage:      "go-derived-acceptance",
				Description:  "The owner can verify that the delivered behavior satisfies the requested job: " + job,
				Verification: "Review the delivered behavior and its verification evidence against the requested job.",
				EvidenceIDs:  []string{evidenceID},
			},
		},
		NegativeExpectations: []specificationItemInput{
			filler("go-derived-negative", "The result must not add behavior outside this derived specification."),
		},
		RecoveryExpectations: []specificationItemInput{
			filler("go-derived-recovery", "If the requested job cannot be delivered safely, preserve the last valid state and report the blocker before continuing."),
		},
		AffectedPublicPaths: []specificationItemInput{
			{
				Lineage:     "go-derived-projection",
				Path:        specificationProjectionRelativePath,
				Description: "The owner-readable specification projection reflects this derived draft.",
				EvidenceIDs: []string{evidenceID},
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	mutation, err := createSpecificationDraft(root, request, specificationMutationOptions{})
	if err != nil {
		return "", "", "", err
	}
	return mutation.Specification.ID, mutation.Revision.ID, mutation.Revision.ContentHash, nil
}

// addGoDerivedSpecificationOutcome adds one outcome item carrying the
// owner's own sentence to an existing specification lineage that has not
// yet been approved (for example a partial /ant-discuss draft) -- inspect,
// then add, exactly through runSpecCommand, with the predecessor revision
// id and content hash taken from the inspect result rather than typed.
func addGoDerivedSpecificationOutcome(root, job, evidenceID string) (string, string, string, error) {
	inspect, err := runSpecCommand(root, specCommandOperationInspect, specCommandOptions{})
	if err != nil {
		return "", "", "", err
	}
	add, err := runSpecCommand(root, specCommandOperationAdd, specCommandOptions{
		Add:                 true,
		Section:              string(specificationSectionOutcomes),
		Lineage:              goDerivedSpecificationLineage,
		Text:                 job,
		PredecessorRevision:  inspect.AfterRevisionID,
		PredecessorHash:      inspect.ContentHash,
		EvidenceIDs:          []string{evidenceID},
	})
	if err != nil {
		return "", "", "", err
	}
	return add.SpecificationID, add.AfterRevisionID, add.ContentHash, nil
}
