package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRetiringACandidateUnblocksPhaseInsertion reproduces the downstream dead
// end and proves the exit exists.
//
// Reported on v1.0.77: after a phase audit raised findings the owner wanted to
// fix, `aether insert-phase` refused with "cannot insert a phase while
// candidate <id> is already pending review". The candidate was stale against
// the accepted plan, `aether plan` offered only review and accept, accepting it
// would have replaced the good active plan with the stale one, and waiting does
// not help either -- loadPlanCandidateArtifactInSession treats an EXPIRED
// candidate as still reviewable, so the built-in expiry date never releases the
// guard. There was no third door.
//
// The test drives the REAL insert-phase entry point, not a simulation of its
// guard, because the guard refuses twice (on the state marker AND on any
// reviewable artifact on disk) and a fix that cleared only one would still
// leave the owner stuck.
func TestRetiringACandidateUnblocksPhaseInsertion(t *testing.T) {
	saveGlobals(t)
	// Establish an accepted plan first — insert-phase needs a base revision —
	// then create the phase-insert candidate that becomes the blocker. This is
	// the reported sequence: a colony with a good accepted plan, and a
	// candidate left waiting beside it.
	root, firstCandidate := planCandidateTestPending(t)
	if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(firstCandidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:base", AcceptedAt: time.Date(2026, time.September, 13, 11, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("accept the base plan: %v", err)
	}

	insertRequest := func(state colony.ColonyState, name, description string, at time.Time) phaseInsertCandidateRequest {
		baseRevision, ok := activePlanRevision(state.Plan)
		if !ok || state.Specification == nil {
			t.Fatal("fixture lacks an accepted plan or specification")
		}
		spec, ok := currentSpecificationRevision(*state.Specification)
		if !ok || len(spec.Requirements) == 0 {
			t.Fatal("fixture lacks a current requirement")
		}
		return phaseInsertCandidateRequest{
			After: len(state.Plan.Phases), Name: name,
			Description:                description,
			Constraints:                "Leave the accepted phases immutable.",
			SpecificationItemID:        spec.Requirements[0].ID,
			ExpectedBasePlanRevisionID: baseRevision.ID,
			CreatedAt:                  at,
		}
	}

	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	// The first insertion succeeds and itself leaves a candidate waiting.
	blocker, err := createPhaseInsertCandidate(root, insertRequest(state, "Stale draft phase", "A draft the owner will decide not to keep.", time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("create the candidate that will block: %v", err)
	}
	candidate := blocker.Candidate

	state, err = loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}

	// 1. The dead end, reproduced.
	_, err = createPhaseInsertCandidate(root, insertRequest(state, "Corrective phase", "Fix the findings the auditor raised before sealing.", time.Date(2026, time.September, 13, 12, 30, 0, 0, time.UTC)))
	if err == nil {
		t.Fatalf("premise broken: insert-phase succeeded while candidate %s was still waiting, so this test no longer reproduces the reported condition", candidate.ID)
	}
	if !strings.Contains(err.Error(), "already pending review") {
		t.Fatalf("expected the pending-review refusal, got: %v", err)
	}

	// 2. The refusal must NAME the way out. A refusal with no exit is the
	//    defect itself, not merely unhelpful -- that was the stated lesson of
	//    the earlier two-waiting-plans deadlock, and it was only half-learned.
	if !strings.Contains(err.Error(), "--retire-candidate") {
		t.Errorf("the refusal does not name the retire action, leaving the owner with review or accept and no third door: %v", err)
	}

	// 3. Retire it by exact ID.
	retired, err := retirePlanCandidate(root,
		planCandidateAcceptanceRequest{CandidateID: candidate.ID},
		planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 13, 12, 1, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("retire candidate %s: %v", candidate.ID, err)
	}
	if retired.Candidate.Status != colony.PlanCandidateRejected {
		t.Fatalf("expected the retired candidate to carry status %q, got %q", colony.PlanCandidateRejected, retired.Candidate.Status)
	}

	// 4. The insertion the owner wanted now goes through.
	state, err = loadSpecificationColonyState(root)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if pending := strings.TrimSpace(state.Plan.PendingCandidateID); pending != "" {
		t.Fatalf("expected the pending marker cleared after retirement, still %q", pending)
	}
	if _, err := createPhaseInsertCandidate(root, insertRequest(state, "Corrective phase", "Fix the findings the auditor raised before sealing.", time.Date(2026, time.September, 13, 12, 30, 0, 0, time.UTC))); err != nil {
		t.Fatalf("insert-phase is still blocked after retiring the candidate: %v", err)
	}
}

// TestRetiringACandidateRequiresItsExactID proves retirement can never be
// inferred. It discards the owner's work, so "whichever one is waiting" is not
// an acceptable target -- the same discipline --accept-candidate already holds.
func TestRetiringACandidateRequiresItsExactID(t *testing.T) {
	saveGlobals(t)
	root, _ := planCandidateSemanticIntegrity200Fixture(t)

	_, err := retirePlanCandidate(root,
		planCandidateAcceptanceRequest{CandidateID: "   "},
		planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("retiring without an exact candidate id succeeded; retirement must never be inferred")
	}
	if !strings.Contains(err.Error(), "exact id") {
		t.Errorf("expected the refusal to say an exact id is required, got: %v", err)
	}

	_, err = retirePlanCandidate(root,
		planCandidateAcceptanceRequest{CandidateID: "plan-candidate-does-not-exist"},
		planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("retiring an unknown candidate id succeeded; it must refuse by name")
	}
}
