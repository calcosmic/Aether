package cmd

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestPlanAcceptanceGate200BuildAndAutopilotParity(t *testing.T) {
	type fixture struct {
		facts    LifecycleFacts
		bindings planAuthorityVerifiedBindings
	}
	cases := []struct {
		name     string
		fixture  func(*testing.T) fixture
		eligible bool
		code     planAuthorityRefusalCode
		recovery string
	}{
		{name: "current accepted", eligible: true, fixture: func(t *testing.T) fixture {
			facts, bindings := planAuthorityCurrentFixture(t)
			return fixture{facts: facts, bindings: bindings}
		}},
		{name: "candidate ready", code: planAuthorityRefusalCandidateNotAccepted, recovery: "aether plan --candidate", fixture: func(t *testing.T) fixture {
			facts, bindings := planAuthorityCurrentFixture(t)
			facts.State.Value.Plan.Candidates[0].Status = colony.PlanCandidatePendingReview
			facts.State.Value.Plan.Candidates[0].Acceptance = nil
			bindings.Candidate.Status = colony.PlanCandidatePendingReview
			bindings.Candidate.Acceptance = nil
			bindings.Acceptance = nil
			facts.Planning.Value.Stage = string(planningStageCandidateReady)
			return fixture{facts: facts, bindings: bindings}
		}},
		{name: "draft spec", code: planAuthorityRefusalSpecificationNotApproved, recovery: "aether spec", fixture: func(t *testing.T) fixture {
			facts, bindings := planAuthorityCurrentFixture(t)
			current := facts.State.Value.Specification.CurrentRevisionID
			for i := range facts.State.Value.Specification.Revisions {
				if facts.State.Value.Specification.Revisions[i].ID == current {
					facts.State.Value.Specification.Revisions[i].Status = colony.SpecStatusDraft
					facts.State.Value.Specification.Revisions[i].Approval = nil
				}
			}
			return fixture{facts: facts, bindings: bindings}
		}},
		{name: "stale revision", code: planAuthorityRefusalStaleSpecification, recovery: "aether plan", fixture: func(t *testing.T) fixture {
			facts, bindings := planAuthorityCurrentFixture(t)
			active, _ := activePlanRevision(facts.State.Value.Plan)
			for i := range facts.State.Value.Plan.Revisions {
				if facts.State.Value.Plan.Revisions[i].ID == active.ID {
					facts.State.Value.Plan.Revisions[i].SpecificationRevisionHash = planningStateTestDigest("gate-stale-spec")
				}
			}
			return fixture{facts: facts, bindings: bindings}
		}},
		{name: "affected scope", code: planAuthorityRefusalAffectedScope, recovery: "aether plan", fixture: func(t *testing.T) fixture {
			facts, bindings := planAuthorityCurrentFixture(t)
			facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
			facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"task:affected"}
			return fixture{facts: facts, bindings: bindings}
		}},
		{name: "legacy", eligible: true, fixture: func(t *testing.T) fixture {
			return fixture{facts: planAcceptanceGateLegacyFacts(), bindings: planAuthorityVerifiedBindings{}}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := tc.fixture(t)
			buildDecision, buildErr := preflightCodexBuildPlanAuthority(fx.facts, fx.bindings)
			autopilot := buildAutopilotPreflightWithAuthority(fx.facts, fx.bindings)
			if buildDecision != autopilot.PlanAuthority {
				t.Fatalf("build authority = %+v, autopilot authority = %+v", buildDecision, autopilot.PlanAuthority)
			}
			if buildDecision.Eligible != tc.eligible || buildDecision.RefusalCode != tc.code || buildDecision.RecoveryCommand != tc.recovery {
				t.Fatalf("decision = %+v, want eligible=%t code=%q recovery=%q", buildDecision, tc.eligible, tc.code, tc.recovery)
			}
			if tc.eligible && buildErr != nil {
				t.Fatalf("eligible build preflight returned error: %v", buildErr)
			}
			if !tc.eligible {
				var refusal *codexBuildPlanAuthorityError
				if !errors.As(buildErr, &refusal) || refusal.Decision != buildDecision {
					t.Fatalf("build error = %#v, want structured authority refusal %+v", buildErr, buildDecision)
				}
				if autopilot.Valid || autopilot.StateEffect != colony.LifecycleStateEffectNone {
					t.Fatalf("autopilot refusal = %+v, want invalid with zero state effect", autopilot)
				}
			}
		})
	}
}

func TestCodexBuildAuthorityRefusalPrecedesPreparationMutation(t *testing.T) {
	saveGlobals(t)
	facts, bindings := planAuthorityCurrentFixture(t)
	facts.State.Value.Plan.Candidates[0].Status = colony.PlanCandidatePendingReview
	facts.State.Value.Plan.Candidates[0].Acceptance = nil
	bindings.Candidate.Status = colony.PlanCandidatePendingReview
	bindings.Candidate.Acceptance = nil
	bindings.Acceptance = nil

	before, err := json.Marshal(facts.State.Value)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	store, err = storage.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	diskBefore := hashDirContents(t, root)
	decision, gateErr := preflightCodexBuildPlanAuthority(facts, bindings)
	if gateErr == nil || decision.RefusalCode != planAuthorityRefusalCandidateNotAccepted {
		t.Fatalf("decision = %+v, error = %v", decision, gateErr)
	}
	after, err := json.Marshal(facts.State.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("authority preflight mutated phase or task state")
	}
	if diskAfter := hashDirContents(t, root); diskAfter != diskBefore {
		t.Fatalf("authority refusal wrote build artifacts: before=%s after=%s", diskBefore, diskAfter)
	}
}

func planAcceptanceGateLegacyFacts() LifecycleFacts {
	taskID := "1.1"
	goal := "Keep legacy work buildable"
	state := colony.ColonyState{
		Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases: []colony.Phase{{
				ID: 1, Name: "Legacy", Status: colony.PhaseReady,
				Tasks: []colony.Task{{ID: &taskID, Goal: "Build it", Status: colony.TaskPending}},
			}},
		},
	}
	return lifecycleFactsFromStateSnapshot(state, false, time.Now().UTC())
}
