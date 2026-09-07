package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningStateCurrentRoundTripIsStable(t *testing.T) {
	t.Parallel()

	state, cards := validCurrentPlanningState(t)
	if err := validatePlanningState(state); err != nil {
		t.Fatalf("valid current planning state rejected: %v", err)
	}
	if err := validatePlanningTimelineBinding(state.Plan.Candidates[0].Timeline, cards); err != nil {
		t.Fatalf("valid current planning timeline rejected: %v", err)
	}

	normalized, err := normalizePlanningState(state)
	if err != nil {
		t.Fatalf("normalize current planning state: %v", err)
	}
	first, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal normalized state: %v", err)
	}

	var decoded colony.ColonyState
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("decode normalized state: %v", err)
	}
	secondState, err := normalizePlanningState(decoded)
	if err != nil {
		t.Fatalf("normalize round-tripped state: %v", err)
	}
	second, err := json.Marshal(secondState)
	if err != nil {
		t.Fatalf("marshal normalized round trip: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("normalization is not byte-stable:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestPlanningStateRejectsDistinctCorruption(t *testing.T) {
	t.Parallel()

	t.Run("dangling specification predecessor", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Specification.Revisions[0].PredecessorID = "missing-spec-revision"
		state.Specification.Revisions[0].Delta.PredecessorRevisionID = "missing-spec-revision"
		assertPlanningStateError(t, validatePlanningState(state), "predecessor")
	})

	t.Run("duplicate stable semantic ID", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Plan.Revisions[1].SemanticID = state.Plan.Phases[0].SemanticID
		assertPlanningStateError(t, validatePlanningState(state), "duplicate stable ID")
	})

	t.Run("candidate binding to absent revision", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Plan.Candidates[0].SpecificationRevisionID = "missing-spec-revision"
		assertPlanningStateError(t, validatePlanningState(state), "specification_revision_id")
	})

	t.Run("accepted candidate remains pending", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Plan.PendingCandidateID = state.Plan.Candidates[0].ID
		assertPlanningStateError(t, validatePlanningState(state), "pending_candidate_id")
	})

	t.Run("future specification major", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Specification.SchemaVersion = "specification/v2"
		assertPlanningStateError(t, validatePlanningState(state), "unsupported future")
	})

	t.Run("partial current acceptance policy", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Plan.Candidates = nil
		assertPlanningStateError(t, validatePlanningState(state), "explicit_owner")
	})
}

func TestPlanningStateTimelineRejectsDuplicateDimensionAndWrongDigest(t *testing.T) {
	t.Parallel()

	state, cards := validCurrentPlanningState(t)
	binding := state.Plan.Candidates[0].Timeline

	duplicate := clonePlanningCards(t, cards)
	duplicate[0].DimensionAssessments[1].Dimension = duplicate[0].DimensionAssessments[0].Dimension
	duplicate[0].DimensionAssessments[1].RemainingGap.Dimension = duplicate[0].DimensionAssessments[0].Dimension
	assertPlanningStateError(t, validatePlanningTimelineBinding(binding, duplicate), "duplicate")

	wrongDigest := binding
	wrongDigest.LastCardHash = planningStateTestDigest("wrong-last-card")
	assertPlanningStateError(t, validatePlanningTimelineBinding(wrongDigest, cards), "last_card_hash")

	wrongCardHash := clonePlanningCards(t, cards)
	wrongCardHash[0].ContentHash = planningStateTestDigest("tampered-card")
	assertPlanningStateError(t, validatePlanningTimelineBinding(binding, wrongCardHash), "content-addressed")
}

func TestPlanningStateTimelineRequiresStrictIterationOrder(t *testing.T) {
	t.Parallel()

	_, cards := validCurrentPlanningState(t)
	second := cards[0]
	second.Iteration = cards[0].Iteration
	second.ContentHash = planningStateTestDigest("second-card")
	second.ID = planningStateTestAddress("planning-iteration", second.ContentHash)
	cards = append(cards, second)

	binding := validPlanningTimelineBindingForTest(t, cards)
	assertPlanningStateError(t, validatePlanningTimelineBinding(binding, cards), "strictly increasing")
}

func TestPlanningStateLegacyAggregateRemainsAuthorityFree(t *testing.T) {
	t.Parallel()

	taskID := "4.1"
	legacy := colony.ColonyState{
		CurrentPhase: 4,
		Plan: colony.Plan{
			ActiveRevisionID: "legacy-plan",
			Phases: []colony.Phase{{
				ID:     4,
				Name:   "Legacy build",
				Status: colony.PhaseReady,
				Tasks: []colony.Task{{
					ID: taskIDPtr(taskID), Goal: "Keep the old plan buildable", Status: colony.TaskPending,
				}},
			}},
		},
	}

	normalized, err := normalizePlanningState(legacy)
	if err != nil {
		t.Fatalf("legacy planning state rejected: %v", err)
	}
	if normalized.Specification != nil {
		t.Fatal("legacy normalization fabricated a specification")
	}
	if normalized.Plan.PendingCandidateID != "" || len(normalized.Plan.Candidates) != 0 {
		t.Fatalf("legacy normalization fabricated a plan candidate: %#v", normalized.Plan)
	}
	if normalized.Plan.AcceptancePolicy != "" {
		t.Fatalf("validation normalization classified legacy authority early: %q", normalized.Plan.AcceptancePolicy)
	}
	if normalized.Plan.ActiveRevisionID != legacy.Plan.ActiveRevisionID || normalized.CurrentPhase != legacy.CurrentPhase || normalized.Plan.Phases[0].Tasks[0].Status != colony.TaskPending {
		t.Fatalf("legacy execution payload changed: %#v", normalized)
	}
}

func validCurrentPlanningState(t *testing.T) (colony.ColonyState, []colony.PlanningIterationCard) {
	t.Helper()

	now := time.Date(2026, time.September, 7, 10, 0, 0, 0, time.UTC)
	specHash := planningStateTestDigest("spec-revision-current")
	specRevisionID := planningStateTestAddress("spec-revision", specHash)
	specificationID := "spec-goal-200"
	itemHash := func(label string) string { return planningStateTestDigest("spec-item-" + label) }
	revision := colony.SpecRevision{
		SchemaVersion:   colony.SpecificationSchemaVersion,
		ID:              specRevisionID,
		SpecificationID: specificationID,
		CreatedAt:       now,
		ContentHash:     specHash,
		Scope: colony.SpecScope{
			Kind: colony.SpecScopeWholeGoal, GoalID: "goal-200", SessionID: "session-200",
		},
		Status: colony.SpecStatusApproved,
		Outcomes: []colony.SpecOutcome{{
			ID: "outcome-plan-understood", Description: "The owner understands the plan", ContentHash: itemHash("outcome"), EvidenceIDs: []string{"evidence-1"},
		}},
		IncludedBehaviors: []colony.SpecIncludedBehavior{{
			ID: "behavior-visible-loop", Description: "Show grounded iterations", ContentHash: itemHash("behavior"), EvidenceIDs: []string{"evidence-1"},
		}},
		Exclusions: []colony.SpecExclusion{{
			ID: "exclusion-worker-cycle", Description: "Do not change execution", ContentHash: itemHash("exclusion"), EvidenceIDs: []string{"evidence-1"},
		}},
		BindingDecisions: []colony.SpecBindingDecision{{
			ID: "decision-explicit-acceptance", Description: "Acceptance is explicit", ContentHash: itemHash("decision"), EvidenceIDs: []string{"evidence-1"},
		}},
		Requirements: []colony.SpecRequirement{{
			ID: "req-grounded-plan", Description: "The plan is grounded", ContentHash: itemHash("requirement"), EvidenceIDs: []string{"evidence-1"},
		}},
		AcceptanceChecks: []colony.SpecAcceptanceCheck{{
			ID: "check-grounded-plan", Description: "Grounding is proven", Verification: "run focused tests", ContentHash: itemHash("acceptance"), EvidenceIDs: []string{"evidence-1"},
		}},
		NegativeExpectations: []colony.SpecNegativeExpectation{{
			ID: "negative-no-fabrication", Description: "Do not invent approval", ContentHash: itemHash("negative"), EvidenceIDs: []string{"evidence-1"},
		}},
		RecoveryExpectations: []colony.SpecRecoveryExpectation{{
			ID: "recovery-replay", Description: "Replay is exact", ContentHash: itemHash("recovery"), EvidenceIDs: []string{"evidence-1"},
		}},
		AffectedPublicPaths: []colony.SpecPublicPath{{
			ID: "path-ant-plan", Path: "/ant-plan", Description: "Plan visibly", ContentHash: itemHash("path"), EvidenceIDs: []string{"evidence-1"},
		}},
	}
	revision.Delta = colony.SpecRevisionDelta{
		Outcomes:             colony.SpecItemDelta{AddedIDs: []string{revision.Outcomes[0].ID}},
		IncludedBehaviors:    colony.SpecItemDelta{AddedIDs: []string{revision.IncludedBehaviors[0].ID}},
		Exclusions:           colony.SpecItemDelta{AddedIDs: []string{revision.Exclusions[0].ID}},
		BindingDecisions:     colony.SpecItemDelta{AddedIDs: []string{revision.BindingDecisions[0].ID}},
		Requirements:         colony.SpecItemDelta{AddedIDs: []string{revision.Requirements[0].ID}},
		AcceptanceChecks:     colony.SpecItemDelta{AddedIDs: []string{revision.AcceptanceChecks[0].ID}},
		NegativeExpectations: colony.SpecItemDelta{AddedIDs: []string{revision.NegativeExpectations[0].ID}},
		RecoveryExpectations: colony.SpecItemDelta{AddedIDs: []string{revision.RecoveryExpectations[0].ID}},
		AffectedPublicPaths:  colony.SpecItemDelta{AddedIDs: []string{revision.AffectedPublicPaths[0].ID}},
	}
	revision.Approval = &colony.SpecApprovalReceipt{
		SchemaVersion: colony.SpecificationSchemaVersion, ID: "spec-approval-1", SpecificationID: specificationID,
		RevisionID: specRevisionID, RevisionContentHash: specHash, ApprovalTokenHash: planningStateTestDigest("spec-approval-token"),
		ApprovedBy: "owner", ApprovedAt: now.Add(time.Minute),
	}
	specification := &colony.Specification{
		SchemaVersion: colony.SpecificationSchemaVersion, ID: specificationID, GoalID: "goal-200",
		CurrentRevisionID: specRevisionID, Revisions: []colony.SpecRevision{revision},
	}

	cards := []colony.PlanningIterationCard{validPlanningIterationCardForTest(t, 1, now.Add(2*time.Minute))}
	timeline := validPlanningTimelineBindingForTest(t, cards)
	candidateHash := planningStateTestDigest("candidate-current")
	candidateID := planningStateTestAddress("plan-candidate", candidateHash)

	baseTaskID := "1.1"
	basePhases := []colony.Phase{{
		ID: 1, Name: "Base", Status: colony.PhaseReady,
		Tasks: []colony.Task{{ID: taskIDPtr(baseTaskID), Goal: "Base task", Status: colony.TaskPending}},
	}}
	baseHash, err := planDefinitionHash(basePhases)
	if err != nil {
		t.Fatalf("hash base plan: %v", err)
	}
	baseRevision := colony.PlanRevision{
		SchemaVersion: 1, Number: 1, ID: "plan-r1-" + baseHash[:12], CreatedAt: now.Format(time.RFC3339Nano),
		ReasonType: colony.PlanRevisionLegacyImport, Reason: "Imported base", PlanHash: baseHash, Phases: basePhases,
	}

	taskID := "1.1"
	activePhases := []colony.Phase{{
		ID: 1, Name: "Grounded", Status: colony.PhaseReady, SemanticID: "phase-grounded",
		RequirementProofLinks: []string{"req-grounded-plan"}, AcceptanceProofLinks: []string{"check-grounded-plan"},
		NegativeProofLinks: []string{"negative-no-fabrication"}, RecoveryProofLinks: []string{"recovery-replay"}, PublicPathProofLinks: []string{"path-ant-plan"},
		SpecificationRevisionID: specRevisionID, SpecificationRevisionHash: specHash,
		CandidateID: candidateID, CandidateContentHash: candidateHash, PlanningTimelineID: timeline.ID, PlanningTimelineDigest: timeline.TimelineDigest,
		Tasks: []colony.Task{{
			ID: taskIDPtr(taskID), Goal: "Execute the grounded plan", Status: colony.TaskPending, SemanticID: "task-grounded",
			RequirementProofLinks: []string{"req-grounded-plan"}, AcceptanceProofLinks: []string{"check-grounded-plan"},
			NegativeProofLinks: []string{"negative-no-fabrication"}, RecoveryProofLinks: []string{"recovery-replay"}, PublicPathProofLinks: []string{"path-ant-plan"},
			SpecificationRevisionID: specRevisionID, SpecificationRevisionHash: specHash,
			CandidateID: candidateID, CandidateContentHash: candidateHash, PlanningTimelineID: timeline.ID, PlanningTimelineDigest: timeline.TimelineDigest,
		}},
	}}
	activeHash, err := planDefinitionHash(activePhases)
	if err != nil {
		t.Fatalf("hash active plan: %v", err)
	}
	activeRevision := colony.PlanRevision{
		SchemaVersion: 1, Number: 2, ID: "plan-r2-" + activeHash[:12], ParentID: baseRevision.ID,
		CreatedAt: now.Add(3 * time.Minute).Format(time.RFC3339Nano), ReasonType: colony.PlanRevisionResearch,
		Reason: "Grounded iteration", PlanHash: activeHash, Phases: activePhases, SemanticID: "plan-grounded",
		RequirementProofLinks: []string{"req-grounded-plan"}, AcceptanceProofLinks: []string{"check-grounded-plan"},
		NegativeProofLinks: []string{"negative-no-fabrication"}, RecoveryProofLinks: []string{"recovery-replay"}, PublicPathProofLinks: []string{"path-ant-plan"},
		SpecificationRevisionID: specRevisionID, SpecificationRevisionHash: specHash,
		CandidateID: candidateID, CandidateContentHash: candidateHash, PlanningTimelineID: timeline.ID, PlanningTimelineDigest: timeline.TimelineDigest,
	}

	assessments := validPlanningAssessmentsForTest()
	delta := validPlanningSemanticDeltaForTest()
	stop := validPlanningStopForTest()
	residualGap := assessments[2].RemainingGap
	stop.SelectedGapID = residualGap.ID
	stop.ResidualGapIDs = []string{residualGap.ID}
	recommendationHash := planningStateTestDigest("queen-recommendation")
	recommendation := colony.QueenPlanRecommendation{
		SchemaVersion: colony.PlanningSchemaVersion, ID: planningStateTestAddress("queen-recommendation", recommendationHash), ContentHash: recommendationHash,
		CandidateID: candidateID, Disposition: colony.PlanRecommendationAccept, EvidenceIDs: []string{"evidence-1"},
		Rationale: "The grounded route is ready", Producer: colony.PlanRecommendationProducerQueen, ProducerID: "go-queen/v1", CreatedAt: now.Add(4 * time.Minute),
	}
	acceptanceHash := planningStateTestDigest("plan-acceptance")
	acceptance := &colony.PlanAcceptanceReceipt{
		SchemaVersion: colony.PlanAcceptanceSchemaVersion, ID: planningStateTestAddress("plan-acceptance", acceptanceHash), ContentHash: acceptanceHash,
		CandidateID: candidateID, CandidateContentHash: candidateHash,
		SpecificationRevisionID: specRevisionID, SpecificationRevisionHash: specHash,
		BasePlanRevisionID: baseRevision.ID, BasePlanRevisionHash: baseHash,
		TimelineID: timeline.ID, TimelineDigest: timeline.TimelineDigest, ProposalHash: activeHash,
		AcceptanceTokenHash: planningStateTestDigest("plan-acceptance-token"), AcceptedBy: "owner", AcceptedAt: now.Add(5 * time.Minute),
		ActivatedPlanRevisionID: activeRevision.ID, ActivatedPlanRevisionHash: activeHash,
	}
	candidate := colony.PlanCandidate{
		SchemaVersion: colony.PlanCandidateSchemaVersion, ID: candidateID, ContentHash: candidateHash, Status: colony.PlanCandidateAccepted,
		CreatedAt: now.Add(4 * time.Minute), ExpiresAt: now.Add(24 * time.Hour), Proposal: activeRevision, ProposalHash: activeHash,
		BasePlanRevisionID: baseRevision.ID, BasePlanRevisionHash: baseHash,
		SpecificationRevisionID: specRevisionID, SpecificationRevisionHash: specHash,
		Timeline: timeline, StopDecision: stop, DimensionAssessments: assessments, SemanticDelta: delta,
		ResidualGaps:            []colony.PlanningGap{residualGap},
		EvidenceThatWouldChange: "A new material risk would reopen planning", Recommendation: recommendation, Acceptance: acceptance,
	}

	goal := "Restore iterative planning"
	return colony.ColonyState{
		Goal: &goal, SessionID: planningStateStringPtr("session-200"), CurrentPhase: 1, State: colony.StateREADY,
		Specification: specification,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceExplicitOwner, ActiveRevisionID: activeRevision.ID,
			Candidates: []colony.PlanCandidate{candidate}, Revisions: []colony.PlanRevision{baseRevision, activeRevision}, Phases: activePhases,
		},
	}, cards
}

func validPlanningIterationCardForTest(t *testing.T, iteration int, createdAt time.Time) colony.PlanningIterationCard {
	t.Helper()
	cardHash := planningStateTestDigest("iteration-card-" + string(rune('0'+iteration)))
	assessments := validPlanningAssessmentsForTest()
	weakestGap := assessments[2].RemainingGap
	decision := validPlanningStopForTest()
	decision.SelectedGapID = weakestGap.ID
	decision.ResidualGapIDs = []string{weakestGap.ID}
	return colony.PlanningIterationCard{
		SchemaVersion: colony.PlanningIterationSchemaVersion,
		ID:            planningStateTestAddress("planning-iteration", cardHash), ContentHash: cardHash,
		RunID: "planning-run-200", Iteration: iteration,
		ScoutReceiptID: "scout-receipt-1", ScoutReceiptHash: planningStateTestDigest("scout-receipt-1"),
		RouteSetterReceiptID: "route-receipt-1", RouteSetterReceiptHash: planningStateTestDigest("route-receipt-1"),
		EvidenceIDs: []string{"evidence-1"}, DimensionAssessments: assessments,
		WeakestGap:    weakestGap,
		SemanticDelta: validPlanningSemanticDeltaForTest(), Decision: decision,
		EvidenceThatWouldChange: "A new material risk would reopen planning", CreatedAt: createdAt,
	}
}

func validPlanningAssessmentsForTest() []colony.PlanningDimensionAssessment {
	dimensions := colony.PlanningDimensions()
	result := make([]colony.PlanningDimensionAssessment, 0, len(dimensions))
	for i, dimension := range dimensions {
		hash := planningStateTestDigest("assessment-" + string(dimension))
		result = append(result, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: planningStateTestAddress("assessment", hash), ContentHash: hash,
			Dimension: dimension, Before: 50 + i, After: 55 + i, FreshEvidenceIDs: []string{"evidence-1"},
			RemainingGap: validPlanningGapForTest(dimension, "remaining-"+string(dimension)),
			Rationale:    "Fresh evidence changed readiness", ProducerReceiptID: "route-receipt-1",
		})
	}
	return result
}

func validPlanningGapForTest(dimension colony.PlanningDimension, label string) colony.PlanningGap {
	hash := planningStateTestDigest("gap-" + label)
	return colony.PlanningGap{
		SchemaVersion: colony.PlanningSchemaVersion, ID: planningStateTestAddress("planning-gap", hash), ContentHash: hash,
		Dimension: dimension, Materiality: colony.PlanningGapNonMaterial, Severity: 1,
		Description: "Remaining " + string(dimension) + " gap", EvidenceIDs: []string{"evidence-1"},
		EvidenceThatWouldChange: "Fresh evidence resolves " + string(dimension),
	}
}

func validPlanningSemanticDeltaForTest() colony.PlanningSemanticDelta {
	hash := planningStateTestDigest("semantic-delta")
	impactHash := planningStateTestDigest("authority-impact")
	return colony.PlanningSemanticDelta{
		SchemaVersion: colony.PlanningSchemaVersion, ID: planningStateTestAddress("planning-delta", hash), ContentHash: hash,
		Phases: []colony.PlanningSemanticChange{{
			SemanticID: "phase-grounded", ContentHash: planningStateTestDigest("phase-change"), Kind: colony.PlanningSemanticChangeModified,
			BeforeHash: planningStateTestDigest("phase-before"), AfterHash: planningStateTestDigest("phase-after"), EvidenceIDs: []string{"evidence-1"},
		}},
		AuthorityImpacts: []colony.PlanningAuthorityImpact{{
			ID: planningStateTestAddress("authority-impact", impactHash), ContentHash: impactHash,
			Kind: colony.PlanningAuthoritySpecApproval, SourceID: "spec-approval-1", AffectedSemanticIDs: []string{"phase-grounded"},
			Rationale: "The exact specification is approved",
		}},
	}
}

func validPlanningStopForTest() colony.PlanningStopDecision {
	hash := planningStateTestDigest("stop-decision")
	return colony.PlanningStopDecision{
		SchemaVersion: colony.PlanningSchemaVersion, ID: planningStateTestAddress("planning-stop", hash), ContentHash: hash,
		Reason: colony.PlanningStopTargetMet, ResidualGapIDs: []string{"residual-risk"}, EvidenceIDs: []string{"evidence-1"},
		Rationale: "The target is sufficient", EvidenceThatWouldChange: "A new material risk would reopen planning",
	}
}

func validPlanningTimelineBindingForTest(t *testing.T, cards []colony.PlanningIterationCard) colony.PlanningTimelineBinding {
	t.Helper()
	digest, err := planningTimelineDigest(cards)
	if err != nil {
		t.Fatalf("hash planning timeline: %v", err)
	}
	bindingHash := planningStateTestDigest("timeline-binding-" + digest)
	cardIDs := make([]string, len(cards))
	for i := range cards {
		cardIDs[i] = cards[i].ID
	}
	return colony.PlanningTimelineBinding{
		SchemaVersion: colony.PlanningTimelineSchemaVersion, ID: planningStateTestAddress("planning-timeline", bindingHash), ContentHash: bindingHash,
		RunID: cards[0].RunID, CardIDs: cardIDs, FirstCardHash: cards[0].ContentHash, LastCardHash: cards[len(cards)-1].ContentHash,
		TimelineDigest: digest, Path: ".aether/data/planning/planning-run-200/timeline.json",
	}
}

func clonePlanningCards(t *testing.T, cards []colony.PlanningIterationCard) []colony.PlanningIterationCard {
	t.Helper()
	raw, err := json.Marshal(cards)
	if err != nil {
		t.Fatalf("marshal planning cards: %v", err)
	}
	var cloned []colony.PlanningIterationCard
	if err := json.Unmarshal(raw, &cloned); err != nil {
		t.Fatalf("clone planning cards: %v", err)
	}
	return cloned
}

func planningStateTestDigest(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func planningStateTestAddress(prefix, digest string) string {
	return prefix + "-" + digest[:12]
}

func taskIDPtr(value string) *string { return &value }

func planningStateStringPtr(value string) *string { return &value }

func assertPlanningStateError(t *testing.T, err error, fragment string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected planning-state error containing %q", fragment)
	}
	if !strings.Contains(err.Error(), fragment) {
		t.Fatalf("planning-state error %q does not contain %q", err, fragment)
	}
}
