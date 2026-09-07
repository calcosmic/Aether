package cmd

import (
	"reflect"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanAuthorityCurrentAcceptedExactBindings(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)

	decision := validateAcceptedPlanAuthority(facts, bindings)
	if !decision.Eligible || decision.Classification != planAuthorityCurrentAccepted {
		t.Fatalf("decision = %+v, want eligible current authority", decision)
	}
	state := facts.State.Value
	active, ok := activePlanRevision(state.Plan)
	if !ok {
		t.Fatal("fixture has no active plan revision")
	}
	candidate := state.Plan.Candidates[0]
	want := planAuthorityDecision{
		Eligible:       true,
		Classification: planAuthorityCurrentAccepted,
		ActiveRevision: planAuthorityBinding{ID: active.ID, Hash: active.PlanHash},
		Specification:  planAuthorityBinding{ID: candidate.SpecificationRevisionID, Hash: candidate.SpecificationRevisionHash},
		Candidate:      planAuthorityBinding{ID: candidate.ID, Hash: candidate.ContentHash},
		Timeline:       planAuthorityBinding{ID: candidate.Timeline.ID, Hash: candidate.Timeline.TimelineDigest},
		Acceptance:     planAuthorityBinding{ID: candidate.Acceptance.ID, Hash: candidate.Acceptance.ContentHash},
	}
	if !reflect.DeepEqual(decision, want) {
		t.Fatalf("decision = %+v, want %+v", decision, want)
	}
}

func TestPlanAuthorityRefusesDraftSpecification(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)
	current := facts.State.Value.Specification.CurrentRevisionID
	for i := range facts.State.Value.Specification.Revisions {
		if facts.State.Value.Specification.Revisions[i].ID == current {
			facts.State.Value.Specification.Revisions[i].Status = colony.SpecStatusDraft
			facts.State.Value.Specification.Revisions[i].Approval = nil
		}
	}

	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(facts, bindings), planAuthorityRefusalSpecificationNotApproved, "aether spec")
}

func TestPlanAuthorityRefusesPendingAndRejectedCandidateDespiteBooleanShortcut(t *testing.T) {
	for _, status := range []colony.PlanCandidateStatus{colony.PlanCandidatePendingReview, colony.PlanCandidateRejected} {
		t.Run(string(status), func(t *testing.T) {
			facts, bindings := planAuthorityCurrentFixture(t)
			facts.Planning.Value.AcceptedPlan = true
			facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAccepted
			bindings.Candidate.Status = status
			bindings.Candidate.Acceptance = nil
			bindings.Acceptance = nil

			assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(facts, bindings), planAuthorityRefusalCandidateNotAccepted, "aether plan --candidate")
		})
	}
}

func TestPlanAuthorityRefusesStaleSpecification(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)
	active, _ := activePlanRevision(facts.State.Value.Plan)
	for i := range facts.State.Value.Plan.Revisions {
		if facts.State.Value.Plan.Revisions[i].ID == active.ID {
			facts.State.Value.Plan.Revisions[i].SpecificationRevisionHash = planningStateTestDigest("stale-spec")
		}
	}

	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(facts, bindings), planAuthorityRefusalStaleSpecification, "aether plan")
}

func TestPlanAuthorityRefusesStaleBaseRevision(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)
	staleHash := planningStateTestDigest("stale-base")
	bindings.Candidate.BasePlanRevisionHash = staleHash
	facts.State.Value.Plan.Candidates[0].BasePlanRevisionHash = staleHash

	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(facts, bindings), planAuthorityRefusalStaleBase, "aether plan")
}

func TestPlanAuthorityRefusesBrokenTimeline(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)
	bindings.Cards[0].ContentHash = planningStateTestDigest("broken-card")

	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(facts, bindings), planAuthorityRefusalBrokenTimeline, "aether plan")
}

func TestPlanAuthorityRefusesUnresolvedAffectedScope(t *testing.T) {
	facts, bindings := planAuthorityCurrentFixture(t)
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
	facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"task:z", "task:a", "task:z"}

	decision := validateAcceptedPlanAuthority(facts, bindings)
	assertPlanAuthorityRefusal(t, decision, planAuthorityRefusalAffectedScope, "aether plan")
	if want := []string{"task:a", "task:z"}; !reflect.DeepEqual(decision.AffectedSemanticIDs, want) {
		t.Fatalf("affected IDs = %v, want %v", decision.AffectedSemanticIDs, want)
	}
}

func TestPlanAuthorityLegacyRequiresExplicitMigrationMarkerAndActivePlan(t *testing.T) {
	taskID := "1.1"
	legacy := colony.ColonyState{
		Goal:         planningStateStringPtr("Keep the migrated plan buildable"),
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases: []colony.Phase{{
				ID: 1, Name: "Legacy", Status: colony.PhaseReady,
				Tasks: []colony.Task{{ID: &taskID, Goal: "Build it", Status: colony.TaskPending}},
			}},
		},
	}
	facts := lifecycleFactsFromStateSnapshot(legacy, false, time.Now().UTC())
	decision := validateAcceptedPlanAuthority(facts, planAuthorityVerifiedBindings{})
	if !decision.Eligible || decision.Classification != planAuthorityLegacyUnbound || decision.Acceptance.ID != "" {
		t.Fatalf("legacy decision = %+v, want eligible explicit legacy without synthesized receipt", decision)
	}

	missingMarker := facts
	missingMarker.State.Value.Plan.AcceptancePolicy = ""
	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(missingMarker, planAuthorityVerifiedBindings{}), planAuthorityRefusalMissingPolicy, "aether plan")

	noActivePlan := facts
	noActivePlan.State.Value.Plan.Phases = nil
	assertPlanAuthorityRefusal(t, validateAcceptedPlanAuthority(noActivePlan, planAuthorityVerifiedBindings{}), planAuthorityRefusalNoActivePlan, "aether plan")
}

func planAuthorityCurrentFixture(t *testing.T) (LifecycleFacts, planAuthorityVerifiedBindings) {
	t.Helper()
	state, cards := validCurrentPlanningState(t)
	active, ok := activePlanRevision(state.Plan)
	if !ok {
		t.Fatal("fixture has no active revision")
	}
	candidate := state.Plan.Candidates[0]
	receipt, err := newPlanCandidateAcceptanceReceipt(candidate, planCandidateAcceptanceToken(candidate), active, "owner", candidate.Acceptance.AcceptedAt)
	if err != nil {
		t.Fatalf("create exact acceptance receipt: %v", err)
	}
	candidate.Acceptance = &receipt
	state.Plan.Candidates[0] = candidate
	facts := lifecycleFactsFromStateSnapshot(state, false, time.Now().UTC())
	return facts, planAuthorityVerifiedBindings{
		Candidate:  &candidate,
		Acceptance: &receipt,
		Timeline:   &candidate.Timeline,
		Cards:      clonePlanningCards(t, cards),
	}
}

func assertPlanAuthorityRefusal(t *testing.T, decision planAuthorityDecision, code planAuthorityRefusalCode, recovery string) {
	t.Helper()
	if decision.Eligible || decision.RefusalCode != code || decision.RecoveryCommand != recovery {
		t.Fatalf("decision = %+v, want refusal code %q with recovery %q", decision, code, recovery)
	}
}
