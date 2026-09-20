package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func specPlanSealGate200Facts(t *testing.T) LifecycleFacts {
	t.Helper()

	state, _ := validCurrentPlanningState(t)
	state.Plan.Phases[0].Status = colony.PhaseCompleted
	state.Plan.Phases[0].Tasks[0].Status = colony.TaskCompleted

	facts := sealOutcome199Facts()
	facts.State.Value = state
	facts.Progress.Value = LifecycleProgressFacts{CurrentPhase: state.CurrentPhase, Phases: state.Plan.Phases}
	facts.Specification = lifecycleSpecificationFromSnapshot(state, facts.State.Source)
	facts.Planning = lifecyclePlanningFromSnapshot(state, facts.State.Source, nil, lifecyclePlanningRunID(state), LifecycleFactSource{})
	return facts
}

func TestSpecPlanSealGate200RefusesDraftSpecification(t *testing.T) {
	facts := specPlanSealGate200Facts(t)
	facts.Specification.Value.Status = colony.SpecStatusDraft
	facts.Specification.Value.Approved = false
	facts.Specification.Value.ApprovalReceiptID = ""

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err == nil {
		t.Fatalf("draft specification sealed as verified: %+v", preflight)
	}
	if preflight.Eligible || preflight.Disposition != colony.SealDispositionVerified {
		t.Fatalf("draft refusal changed the requested normal disposition: %+v", preflight)
	}
	if !strings.Contains(err.Error(), facts.Specification.Value.CurrentRevisionID) || !strings.Contains(err.Error(), "aether spec") {
		t.Fatalf("draft refusal omitted exact specification recovery: %v", err)
	}
	if !reflect.DeepEqual(preflight.RecoveryCommands, []string{"aether spec"}) {
		t.Fatalf("draft recovery commands = %#v", preflight.RecoveryCommands)
	}
}

func TestSpecPlanSealGate200RefusesApprovedUnreconciledScope(t *testing.T) {
	facts := specPlanSealGate200Facts(t)
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
	facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"req-grounded-plan", "task-grounded"}

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err == nil {
		t.Fatalf("approved but unreconciled scope sealed as verified: %+v", preflight)
	}
	if got, want := preflight.AffectedSemanticIDs, []string{"req-grounded-plan", "task-grounded"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("affected IDs = %#v, want %#v", got, want)
	}
	for _, required := range []string{"req-grounded-plan", "task-grounded", "aether plan"} {
		if !strings.Contains(err.Error(), required) {
			t.Fatalf("unreconciled refusal omitted %q: %v", required, err)
		}
	}
}

func TestSpecPlanSealGate200RefusesWrongSpecificationBinding(t *testing.T) {
	facts := specPlanSealGate200Facts(t)
	facts.Planning.Value.AcceptedPlan = false
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingInvalid

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err == nil {
		t.Fatalf("wrong-specification plan sealed as verified: %+v", preflight)
	}
	if !strings.Contains(err.Error(), facts.Specification.Value.CurrentRevisionID) || !strings.Contains(err.Error(), "aether plan") {
		t.Fatalf("wrong-specification refusal omitted binding and recovery detail: %v", err)
	}
}

func TestSpecPlanSealGate200RefusesMissingAffectedProof(t *testing.T) {
	facts := specPlanSealGate200Facts(t)
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
	facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"check-grounded-plan"}

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err == nil {
		t.Fatalf("missing affected proof sealed as verified: %+v", preflight)
	}
	if !reflect.DeepEqual(preflight.AffectedSemanticIDs, []string{"check-grounded-plan"}) || !strings.Contains(err.Error(), "check-grounded-plan") {
		t.Fatalf("missing proof was not named: preflight=%+v err=%v", preflight, err)
	}
}

func TestSpecPlanSealGate200AllowsExactReconciledPlan(t *testing.T) {
	preflight, err := BuildSealPreflight(specPlanSealGate200Facts(t), SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err != nil {
		t.Fatalf("exact reconciled plan could not reach verified seal checks: %v\n%+v", err, preflight)
	}
	if !preflight.Eligible || preflight.Disposition != colony.SealDispositionVerified || len(preflight.AffectedSemanticIDs) != 0 || len(preflight.RecoveryCommands) != 0 {
		t.Fatalf("reconciled preflight = %+v", preflight)
	}
}

func TestSealOutcome200LegacyUnboundRemainsCompatible(t *testing.T) {
	preflight, err := BuildSealPreflight(sealOutcome199Facts(), SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err != nil {
		t.Fatalf("historical legacy-unbound plan lost compatibility: %v", err)
	}
	if !preflight.Eligible || preflight.Disposition != colony.SealDispositionVerified || len(preflight.RecoveryCommands) != 0 {
		t.Fatalf("legacy preflight fabricated modern authority: %+v", preflight)
	}
}

func TestSealOutcome200ForcedIncompleteNeverClaimsVerification(t *testing.T) {
	facts := specPlanSealGate200Facts(t)
	facts.Specification.Value.Status = colony.SpecStatusDraft
	facts.Specification.Value.Approved = false
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
	facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"check-grounded-plan"}

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{
		Caller: SealCallerDirectOwner,
		Force:  true,
		Reason: "owner records structurally unreconciled closure",
	})
	if err != nil {
		t.Fatalf("explicit forced-incomplete closure failed: %v", err)
	}
	if !preflight.Eligible || preflight.Disposition != colony.SealDispositionForcedIncomplete || preflight.OutcomeKind != colony.OutcomeKindForcedIncompleteClosure {
		t.Fatalf("forced result claimed verified completion: %+v", preflight)
	}
	if len(preflight.UnresolvedItems) == 0 || !strings.Contains(SealConfirmationCopy(preflight), "does not verify completion") {
		t.Fatalf("forced result hid its non-verified status: %+v", preflight)
	}
}
