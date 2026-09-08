package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
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
		state.Plan.Revisions[len(state.Plan.Revisions)-1].SemanticID = state.Plan.Phases[0].SemanticID
		assertPlanningStateError(t, validatePlanningState(state), "duplicate stable ID")
	})

	t.Run("candidate binding to absent revision", func(t *testing.T) {
		state, _ := validCurrentPlanningState(t)
		state.Plan.Candidates[0].SpecificationRevisionID = "missing-spec-revision"
		assertPlanningStateError(t, validatePlanningState(state), "candidate content address")
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

var (
	currentPlanningStateFixtureOnce sync.Once
	currentPlanningStateFixtureJSON []byte
	currentPlanningCardsFixtureJSON []byte
)

func validCurrentPlanningState(t *testing.T) (colony.ColonyState, []colony.PlanningIterationCard) {
	t.Helper()
	currentPlanningStateFixtureOnce.Do(func() {
		root, candidate := planCandidateTestPending(t)
		accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
			AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 7, 20, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("accept production-addressed planning-state fixture: %v", err)
		}
		state, err := loadSpecificationColonyState(root)
		if err != nil {
			t.Fatalf("load production-addressed planning-state fixture: %v", err)
		}
		goal := "Restore iterative planning"
		state.Goal = &goal
		state.SessionID = planningStateStringPtr("session-200")
		timeline, err := loadPlanningTimeline(root, accepted.Candidate.Timeline.RunID)
		if err != nil {
			t.Fatalf("load production-addressed planning timeline: %v", err)
		}
		currentPlanningStateFixtureJSON, err = json.Marshal(state)
		if err != nil {
			t.Fatalf("marshal production-addressed planning-state fixture: %v", err)
		}
		currentPlanningCardsFixtureJSON, err = json.Marshal(timeline.Cards)
		if err != nil {
			t.Fatalf("marshal production-addressed planning cards: %v", err)
		}
	})
	var state colony.ColonyState
	if err := json.Unmarshal(currentPlanningStateFixtureJSON, &state); err != nil {
		t.Fatalf("clone production-addressed planning-state fixture: %v", err)
	}
	// Progress-focused callers historically model one runtime transition by
	// mutating Plan.Phases. Preserve that helper contract while keeping every
	// immutable identity itself sourced from the production constructors.
	for index := range state.Plan.Revisions {
		if state.Plan.Revisions[index].ID == state.Plan.ActiveRevisionID {
			state.Plan.Revisions[index].Phases = state.Plan.Phases
			break
		}
	}
	var cards []colony.PlanningIterationCard
	if err := json.Unmarshal(currentPlanningCardsFixtureJSON, &cards); err != nil {
		t.Fatalf("clone production-addressed planning cards: %v", err)
	}
	return state, cards
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
	cardIDs := make([]string, len(cards))
	for i := range cards {
		cardIDs[i] = cards[i].ID
	}
	binding := colony.PlanningTimelineBinding{
		SchemaVersion: colony.PlanningTimelineSchemaVersion,
		RunID:         cards[0].RunID, CardIDs: cardIDs, FirstCardHash: cards[0].ContentHash, LastCardHash: cards[len(cards)-1].ContentHash,
		TimelineDigest: digest, Path: ".aether/data/planning/planning-run-200/timeline.json",
	}
	bindingHash, err := planningTimelineBindingContentHash(binding)
	if err != nil {
		t.Fatalf("hash canonical planning timeline binding: %v", err)
	}
	binding.ContentHash = bindingHash
	binding.ID = "planning-timeline-" + bindingHash[:12]
	return binding
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

func TestCandidateAcceptanceBaseBindsFirstCurrentAndLegacyPlans(t *testing.T) {
	now := time.Date(2026, time.September, 7, 19, 0, 0, 0, time.UTC)

	fresh := colony.Plan{}
	freshHash, err := planStateHash(fresh)
	if err != nil {
		t.Fatal(err)
	}
	freshBase, freshBaseline, err := candidateAcceptanceBase(fresh, now)
	if err != nil {
		t.Fatal(err)
	}
	if freshBase.ID != "plan-unbound" || freshBase.Hash != freshHash || freshBase.Number != 0 || freshBaseline != nil {
		t.Fatalf("fresh candidate base = %+v baseline=%+v", freshBase, freshBaseline)
	}

	taskID := "1.1"
	phases := []colony.Phase{{ID: 1, Name: "Current", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Goal: "Current", Status: colony.TaskPending}}}}
	planHash, err := planDefinitionHash(phases)
	if err != nil {
		t.Fatal(err)
	}
	revision := colony.PlanRevision{
		SchemaVersion: planRevisionSchemaVersion, Number: 1, ID: "plan-r1-" + planHash[:12], CreatedAt: now.Format(time.RFC3339Nano),
		ReasonType: colony.PlanRevisionInitial, Reason: "Initial", PlanHash: planHash, Phases: clonePhases(phases),
	}
	current := colony.Plan{ActiveRevisionID: revision.ID, Revisions: []colony.PlanRevision{revision}, Phases: clonePhases(phases)}
	current.Phases[0].Status = colony.PhaseInProgress
	current.Phases[0].Tasks[0].Status = colony.TaskCompleted
	stateHash, err := planStateHash(current)
	if err != nil {
		t.Fatal(err)
	}
	if stateHash == planHash {
		t.Fatal("fixture did not distinguish mutable plan state from immutable revision hash")
	}
	currentBase, currentBaseline, err := candidateAcceptanceBase(current, now)
	if err != nil {
		t.Fatal(err)
	}
	if currentBase.ID != revision.ID || currentBase.Hash != planHash || currentBase.Number != 1 || currentBaseline != nil {
		t.Fatalf("current candidate base = %+v baseline=%+v, want active immutable revision", currentBase, currentBaseline)
	}

	legacy := colony.Plan{Phases: clonePhases(phases)}
	legacyBase, legacyBaseline, err := candidateAcceptanceBase(legacy, now)
	if err != nil {
		t.Fatal(err)
	}
	if legacyBaseline == nil || legacyBase.ID != legacyBaseline.ID || legacyBase.Hash != legacyBaseline.PlanHash || legacyBase.Number != 1 || legacyBaseline.ReasonType != colony.PlanRevisionLegacyImport {
		t.Fatalf("legacy candidate base = %+v baseline=%+v", legacyBase, legacyBaseline)
	}
}
