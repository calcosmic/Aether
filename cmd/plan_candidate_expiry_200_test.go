package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanCandidateExpiry200StandingMatrix(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	authority := planCandidateExpiry200Authority(t, root, candidate)
	now := candidate.ExpiresAt.Add(-time.Nanosecond)

	if got := assessPlanCandidateStanding(candidate, authority, now); got.Standing != planCandidateStandingCurrent || !got.AcceptanceAvailable || got.WhyUnavailable != "" {
		t.Fatalf("current standing = %+v, want current and accept-eligible", got)
	}

	tests := []struct {
		name   string
		mutate func(*colony.PlanCandidate, *planCandidateCurrentAuthority, *time.Time)
		want   string
	}{
		{name: "specification revision", want: "specification_changed", mutate: func(_ *colony.PlanCandidate, current *planCandidateCurrentAuthority, _ *time.Time) {
			current.SpecificationRevisionID += "-successor"
		}},
		{name: "base revision", want: "base_plan_changed", mutate: func(_ *colony.PlanCandidate, current *planCandidateCurrentAuthority, _ *time.Time) {
			current.BasePlanRevisionHash = strings.Repeat("9", 64)
		}},
		{name: "proposal", want: "proposal_changed", mutate: func(value *colony.PlanCandidate, _ *planCandidateCurrentAuthority, _ *time.Time) {
			value.Proposal.Phases[0].Description += " divergent proposal"
		}},
		{name: "timeline", want: "timeline_changed", mutate: func(_ *colony.PlanCandidate, current *planCandidateCurrentAuthority, _ *time.Time) {
			current.Timeline.TimelineDigest = strings.Repeat("8", 64)
		}},
		{name: "candidate body", want: "candidate_body_changed", mutate: func(value *colony.PlanCandidate, _ *planCandidateCurrentAuthority, _ *time.Time) {
			value.EvidenceThatWouldChange += " copied-hash mutation"
		}},
		{name: "stage", want: "planning_stage_changed", mutate: func(_ *colony.PlanCandidate, current *planCandidateCurrentAuthority, _ *time.Time) {
			current.Stage.Stage = planningStageAccepted
		}},
		{name: "clock before creation", want: "clock_before_candidate_creation", mutate: func(value *colony.PlanCandidate, _ *planCandidateCurrentAuthority, currentTime *time.Time) {
			*currentTime = value.CreatedAt.Add(-time.Nanosecond)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changedCandidate := planCandidateSemanticIntegrity200Clone(t, candidate)
			changedAuthority := authority
			changedNow := now
			test.mutate(&changedCandidate, &changedAuthority, &changedNow)
			got := assessPlanCandidateStanding(changedCandidate, changedAuthority, changedNow)
			if got.Standing != planCandidateStandingStale || got.WhyUnavailable != test.want || got.AcceptanceAvailable || len(got.Evidence) == 0 {
				t.Fatalf("standing = %+v, want stale/%s with evidence and no acceptance", got, test.want)
			}
			if got.StateEffect != planCandidateStateEffectUnchanged || got.ActivePlanEffect != planCandidateActivePlanEffectUnchanged || got.RecoveryCommand != planCandidateRefreshCommand {
				t.Fatalf("stale recovery facts = %+v, want unchanged/unchanged/%q", got, planCandidateRefreshCommand)
			}
		})
	}

	for _, boundary := range []time.Time{candidate.ExpiresAt, candidate.ExpiresAt.Add(time.Nanosecond)} {
		got := assessPlanCandidateStanding(candidate, authority, boundary)
		if got.Standing != planCandidateStandingExpired || got.WhyUnavailable != "candidate_expired" || got.AcceptanceAvailable || got.RecoveryCommand != planCandidateRefreshCommand {
			t.Fatalf("expiry standing at %s = %+v", boundary, got)
		}
	}
}

func TestPlanCandidateExpiry200RequiresStrictTemporalOrdering(t *testing.T) {
	_, original := planCandidateTestPending(t)
	for _, expiresAt := range []time.Time{original.CreatedAt.Add(-time.Nanosecond), original.CreatedAt} {
		candidate := planCandidateSemanticIntegrity200Clone(t, original)
		candidate.ExpiresAt = expiresAt
		if err := addressPlanCandidateReviewPayload(&candidate); err != nil {
			t.Fatal(err)
		}
		if err := candidate.Validate(); err == nil || !strings.Contains(err.Error(), "expires_at") {
			t.Fatalf("temporal validation = %v, want expires_at strictly after created_at", err)
		}
	}
}

func TestPlanCandidateExpiry200AcceptanceBoundaryIsAtomic(t *testing.T) {
	tests := []struct {
		name        string
		offset      time.Duration
		wantSuccess bool
	}{
		{name: "just before", offset: -time.Nanosecond, wantSuccess: true},
		{name: "exact boundary", offset: 0},
		{name: "just after", offset: time.Nanosecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, candidate := planCandidateTestPending(t)
			before := planCandidateTestSnapshot(t, root)
			beforeState, err := loadSpecificationColonyState(root)
			if err != nil {
				t.Fatal(err)
			}
			result, acceptErr := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
				AcceptedBy: "owner:expiry-200", AcceptedAt: candidate.ExpiresAt.Add(test.offset),
			})
			if test.wantSuccess {
				if acceptErr != nil || result.Candidate.Status != colony.PlanCandidateAccepted || !result.Receipt.AcceptedAt.Before(candidate.ExpiresAt) {
					t.Fatalf("just-before acceptance = %+v err=%v", result, acceptErr)
				}
				return
			}
			if acceptErr == nil || result.Refusal == nil {
				t.Fatalf("deadline acceptance = %+v err=%v, want typed refusal with error", result, acceptErr)
			}
			if result.Refusal.Standing != planCandidateStandingExpired || result.Refusal.StateEffect != planCandidateStateEffectMarkedExpired || result.Refusal.ActivePlanEffect != planCandidateActivePlanEffectUnchanged || result.Refusal.RecoveryCommand != planCandidateRefreshCommand {
				t.Fatalf("expiry refusal = %+v", result.Refusal)
			}

			persisted := planCandidateExpiry200ReadCandidate(t, root, candidate)
			if persisted.Status != colony.PlanCandidateExpired || persisted.Acceptance != nil {
				t.Fatalf("persisted candidate = %+v, want expired without acceptance", persisted)
			}
			stage, err := loadPlanningStageState(root, candidate.Timeline.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if stage.Stage != planningStageFailed || stage.FailureReason != planningStageFailureCandidateExpired {
				t.Fatalf("expired stage = %+v, want failed/%q", stage, planningStageFailureCandidateExpired)
			}
			afterState, err := loadSpecificationColonyState(root)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(beforeState, afterState) {
				t.Fatal("expiry transition changed the active plan or colony state")
			}
			planCandidateExpiry200AssertOnlyChanged(t, root, before,
				planningRouteCandidateRepositoryPath(candidate.Timeline.RunID),
				planningStageStateRepositoryPath(candidate.Timeline.RunID),
			)
		})
	}
}

func TestPlanCandidateExpiry200CommandUsesOneClockReadAndReturnsFacts(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	previousClock := planCandidateNow
	reads := 0
	planCandidateNow = func() time.Time {
		reads++
		if reads > 1 {
			return candidate.ExpiresAt.Add(-time.Hour)
		}
		return candidate.ExpiresAt
	}
	t.Cleanup(func() { planCandidateNow = previousClock })

	request := planCandidateTestAcceptanceRequest(candidate)
	result, handled, err := runPlanCandidateCommand(root, planCandidateCommandInputs{
		AcceptCandidate: request.CandidateID, SpecificationRevisionID: request.SpecificationRevisionID,
		SpecificationRevisionHash: request.SpecificationRevisionHash, BasePlanRevisionID: request.BasePlanRevisionID,
		TimelineDigest: request.TimelineDigest, ProposalHash: request.ProposalHash, AcceptanceToken: request.AcceptanceToken,
	})
	if !handled || err == nil || reads != 1 || result == nil {
		t.Fatalf("command handled=%t err=%v reads=%d result=%#v, want one-read typed refusal", handled, err, reads, result)
	}
	refusal, ok := result["refusal"].(planCandidateRefusalDetails)
	if !ok || refusal.Standing != planCandidateStandingExpired || refusal.StateEffect != planCandidateStateEffectMarkedExpired {
		t.Fatalf("command refusal = %#v", result["refusal"])
	}

	encoded, err := json.Marshal(refusal)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"candidate_id", "candidate_status", "standing", "expires_at", "why_unavailable", "evidence", "state_effect", "active_plan_effect", "recovery_command"} {
		if !bytes.Contains(encoded, []byte(`"`+key+`"`)) {
			t.Fatalf("typed refusal JSON %s omits %q", encoded, key)
		}
	}
}

func TestPlanCandidateExpiry200StaleAndExpiredReviewStayReadOnly(t *testing.T) {
	t.Run("stale body", func(t *testing.T) {
		root, candidate := planCandidateTestPending(t)
		candidate.EvidenceThatWouldChange += " stale copied-hash body"
		planCandidateSemanticIntegrity200Write(t, planCandidateExpiry200CandidatePath(root, candidate), candidate)
		before := planCandidateTestSnapshot(t, root)
		review, err := reviewPlanCandidateAt(root, candidate.CreatedAt.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		planCandidateExpiry200AssertSafeReview(t, review, planCandidateStandingStale, "candidate_body_changed")
		planCandidateTestAssertSnapshot(t, root, before)
	})

	t.Run("persisted expired", func(t *testing.T) {
		root, candidate := planCandidateTestPending(t)
		if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
			AcceptedBy: "owner:expiry-review-200", AcceptedAt: candidate.ExpiresAt,
		}); err == nil {
			t.Fatal("deadline acceptance unexpectedly succeeded")
		}
		before := planCandidateTestSnapshot(t, root)
		review, err := reviewPlanCandidateAt(root, candidate.ExpiresAt.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		planCandidateExpiry200AssertSafeReview(t, review, planCandidateStandingExpired, "candidate_expired")
		planCandidateTestAssertSnapshot(t, root, before)
	})
}

func TestPlanCandidateExpiry200AcceptedReplaySurvivesWallClockButLateForgeryDoesNot(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	request := planCandidateTestAcceptanceRequest(candidate)
	acceptedAt := candidate.ExpiresAt.Add(-time.Nanosecond)
	accepted, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner:expiry-200", AcceptedAt: acceptedAt})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "ignored", AcceptedAt: candidate.ExpiresAt.Add(24 * time.Hour)})
	if err != nil || !replayed.Replayed || !reflect.DeepEqual(accepted.Receipt, replayed.Receipt) {
		t.Fatalf("accepted replay after wall-clock expiry = %+v err=%v", replayed, err)
	}

	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	facts := lifecycleFactsFromStateSnapshot(state, false, candidate.ExpiresAt.Add(30*24*time.Hour))
	bindings := loadPlanAuthorityVerifiedBindings(root, facts)
	if decision := validateAcceptedPlanAuthority(facts, bindings); !decision.Eligible {
		t.Fatalf("timely accepted plan expired later: %+v", decision)
	}

	lateFacts, lateBindings := planAuthorityCurrentFixture(t)
	active, ok := activePlanRevision(lateFacts.State.Value.Plan)
	if !ok || lateBindings.Candidate == nil {
		t.Fatal("late-forgery fixture has no accepted authority")
	}
	lateCandidate := *lateBindings.Candidate
	lateReceipt, err := newPlanCandidateAcceptanceReceipt(lateCandidate, planCandidateAcceptanceToken(lateCandidate), active, "forger", lateCandidate.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	lateCandidate.Acceptance = &lateReceipt
	lateFacts.State.Value.Plan.Candidates[0] = lateCandidate
	lateBindings.Candidate = &lateCandidate
	lateBindings.Acceptance = &lateReceipt
	decision := validateAcceptedPlanAuthority(lateFacts, lateBindings)
	if decision.Eligible || decision.RefusalCode != planAuthorityRefusalAcceptanceInvalid {
		t.Fatalf("forged at-deadline receipt authority = %+v", decision)
	}
}

func planCandidateExpiry200Authority(t *testing.T, root string, candidate colony.PlanCandidate) planCandidateCurrentAuthority {
	t.Helper()
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Specification == nil {
		t.Fatal("candidate fixture has no specification")
	}
	specification, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("candidate fixture has no current specification revision")
	}
	base, _, err := candidateAcceptanceBase(state.Plan, candidate.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	timeline, err := verifiedPlanCandidateTimeline(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := loadPlanningStageState(root, candidate.Timeline.RunID)
	if err != nil {
		t.Fatal(err)
	}
	return planCandidateCurrentAuthority{
		SpecificationRevisionID: specification.ID, SpecificationRevisionHash: specification.ContentHash,
		BasePlanRevisionID: base.ID, BasePlanRevisionHash: base.Hash,
		ProposalHash: candidate.ProposalHash, Timeline: *timeline.Binding, Stage: stage,
	}
}

func planCandidateExpiry200AssertSafeReview(t *testing.T, review planCandidateReview, standing planCandidateStanding, why string) {
	t.Helper()
	if review.Standing != standing || review.Refusal == nil || review.Refusal.WhyUnavailable != why {
		t.Fatalf("review standing = %+v refusal=%+v, want %s/%s", review.Standing, review.Refusal, standing, why)
	}
	if review.Acceptance != (planCandidateAcceptanceRequest{}) || review.AcceptanceCommand != "" || review.Refusal.RecoveryCommand != planCandidateRefreshCommand {
		t.Fatalf("unsafe review acceptance/recovery = %+v command=%q refusal=%+v", review.Acceptance, review.AcceptanceCommand, review.Refusal)
	}
	if len(review.Iterations) == 0 || review.Timeline.TimelineDigest == "" {
		t.Fatalf("stale/expired review lost immutable timeline: %+v", review)
	}
	result := planCandidateReviewResult(review)
	if _, exposed := result["acceptance"]; exposed {
		t.Fatalf("safe review exposed acceptance request: %#v", result["acceptance"])
	}
	if _, exposed := result["acceptance_command"]; exposed {
		t.Fatalf("safe review exposed acceptance command: %#v", result["acceptance_command"])
	}
}

func planCandidateExpiry200ReadCandidate(t *testing.T, root string, original colony.PlanCandidate) colony.PlanCandidate {
	t.Helper()
	content, err := readOptionalPlanningStageFile(root, planningRouteCandidateRepositoryPath(original.Timeline.RunID))
	if err != nil || !contentExists(content) {
		t.Fatalf("read persisted candidate: bytes=%d err=%v", len(content), err)
	}
	var candidate colony.PlanCandidate
	if err := json.Unmarshal(content, &candidate); err != nil {
		t.Fatal(err)
	}
	return candidate
}

func contentExists(content []byte) bool { return len(content) > 0 }

func planCandidateExpiry200CandidatePath(root string, candidate colony.PlanCandidate) string {
	return root + "/" + planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)
}

func planCandidateExpiry200AssertOnlyChanged(t *testing.T, root string, before map[string][]byte, expected ...string) {
	t.Helper()
	after := planCandidateTestSnapshot(t, root)
	changed := make([]string, 0)
	for path, old := range before {
		if current, ok := after[path]; !ok || !bytes.Equal(old, current) {
			changed = append(changed, path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			changed = append(changed, path)
		}
	}
	for index := range expected {
		expected[index] = strings.TrimPrefix(expected[index], ".aether/data/")
		expected[index] = ".aether/data/" + expected[index]
	}
	sort.Strings(changed)
	sort.Strings(expected)
	if !reflect.DeepEqual(changed, expected) {
		t.Fatalf("expiry changed %v, want only %v", changed, expected)
	}
}
