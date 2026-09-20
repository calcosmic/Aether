package colony

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPlanningClosedEnums(t *testing.T) {
	t.Parallel()

	evidenceKinds := []PlanningEvidenceKind{
		PlanningEvidenceSpecification,
		PlanningEvidenceSurvey,
		PlanningEvidenceCharter,
		PlanningEvidenceDecision,
		PlanningEvidenceContext,
		PlanningEvidenceResearch,
		PlanningEvidenceHive,
		PlanningEvidenceOutcome,
	}
	if len(evidenceKinds) != 8 {
		t.Fatalf("evidence kinds = %d, want 8", len(evidenceKinds))
	}
	assertPlanningEnumSet(t, "evidence kind", evidenceKinds, func(value PlanningEvidenceKind) bool { return value.Valid() })

	dimensions := PlanningDimensions()
	if len(dimensions) != 5 {
		t.Fatalf("planning dimensions = %d, want exactly 5", len(dimensions))
	}
	assertPlanningEnumSet(t, "dimension", dimensions, func(value PlanningDimension) bool { return value.Valid() })
	wantDimensions := []PlanningDimension{
		PlanningDimensionKnowledge,
		PlanningDimensionRequirements,
		PlanningDimensionRisks,
		PlanningDimensionDependencies,
		PlanningDimensionEffort,
	}
	if !reflect.DeepEqual(dimensions, wantDimensions) {
		t.Fatalf("planning dimensions = %#v, want %#v", dimensions, wantDimensions)
	}

	assertPlanningEnumSet(t, "gap materiality", []PlanningGapMateriality{
		PlanningGapNonMaterial,
		PlanningGapMaterial,
	}, func(value PlanningGapMateriality) bool { return value.Valid() })
	assertPlanningEnumSet(t, "semantic change", []PlanningSemanticChangeKind{
		PlanningSemanticChangeAdded,
		PlanningSemanticChangeModified,
		PlanningSemanticChangeRemoved,
		PlanningSemanticChangePreserved,
	}, func(value PlanningSemanticChangeKind) bool { return value.Valid() })
	assertPlanningEnumSet(t, "authority impact", []PlanningAuthorityImpactKind{
		PlanningAuthoritySpecApproval,
		PlanningAuthoritySpecSupersession,
		PlanningAuthorityOwnerDecision,
		PlanningAuthorityCandidateStatus,
		PlanningAuthorityPlanAcceptance,
	}, func(value PlanningAuthorityImpactKind) bool { return value.Valid() })
	assertPlanningEnumSet(t, "stop reason", []PlanningStopReason{
		PlanningStopContinue,
		PlanningStopTargetMet,
		PlanningStopDiminishingReturns,
		PlanningStopStalledGap,
		PlanningStopPassCap,
		PlanningStopOwnerDecision,
	}, func(value PlanningStopReason) bool { return value.Valid() })
	assertPlanningEnumSet(t, "candidate status", []PlanCandidateStatus{
		PlanCandidatePendingReview,
		PlanCandidateAccepted,
		PlanCandidateRejected,
		PlanCandidateSuperseded,
		PlanCandidateExpired,
		PlanCandidateInvalidated,
	}, func(value PlanCandidateStatus) bool { return value.Valid() })
	assertPlanningEnumSet(t, "recommendation disposition", []PlanRecommendationDisposition{
		PlanRecommendationAccept,
		PlanRecommendationRevise,
	}, func(value PlanRecommendationDisposition) bool { return value.Valid() })
	assertPlanningEnumSet(t, "recommendation producer", []PlanRecommendationProducer{
		PlanRecommendationProducerQueen,
	}, func(value PlanRecommendationProducer) bool { return value.Valid() })
	assertPlanningEnumSet(t, "acceptance policy", []PlanAcceptancePolicy{
		PlanAcceptanceLegacyUnbound,
		PlanAcceptanceExplicitOwner,
	}, func(value PlanAcceptancePolicy) bool { return value.Valid() })

	if PlanningDimension("overall").Valid() {
		t.Fatal("worker-supplied overall was accepted as a sixth planning dimension")
	}
	if _, err := json.Marshal(PlanningStopReason("future_stop")); err == nil {
		t.Fatal("unknown stop reason marshaled successfully")
	}
	var unknown PlanningEvidenceKind
	if err := json.Unmarshal([]byte(`"future_evidence"`), &unknown); err == nil {
		t.Fatal("unknown evidence kind unmarshaled successfully")
	}
}

func TestPlanningCandidateRoundTripPreservesCausalContracts(t *testing.T) {
	t.Parallel()

	candidate := validPlanCandidate()
	if err := candidate.Validate(); err != nil {
		t.Fatalf("valid candidate rejected: %v", err)
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatalf("marshal candidate: %v", err)
	}
	var decoded PlanCandidate
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal candidate: %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped candidate rejected: %v", err)
	}
	if !reflect.DeepEqual(decoded, candidate) {
		t.Fatalf("candidate round trip changed data:\n got: %#v\nwant: %#v", decoded, candidate)
	}

	assessmentType := reflect.TypeOf(PlanningDimensionAssessment{})
	for _, fieldName := range []string{"Before", "After"} {
		field, ok := assessmentType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("PlanningDimensionAssessment missing %s", fieldName)
		}
		if field.Type.Kind() != reflect.Int {
			t.Fatalf("PlanningDimensionAssessment.%s uses %s, want whole-number int", fieldName, field.Type)
		}
	}

	if len(decoded.DimensionAssessments) != 5 {
		t.Fatalf("candidate dimensions = %d, want 5", len(decoded.DimensionAssessments))
	}
	if decoded.Timeline.TimelineDigest == "" || decoded.StopDecision.EvidenceThatWouldChange == "" || decoded.SemanticDelta.ContentHash == "" {
		t.Fatalf("candidate lost timeline, stop, or delta causality: %#v", decoded)
	}
}

func TestPlanningEvidenceThatWouldChangeIsRequiredEverywhere(t *testing.T) {
	t.Parallel()

	gap := validPlanningGap(PlanningDimensionRisks, "gap-risks")
	gap.EvidenceThatWouldChange = ""
	assertPlanningValidationError(t, gap.Validate(), "evidence_that_would_change")

	stop := validPlanningStopDecision()
	stop.EvidenceThatWouldChange = ""
	assertPlanningValidationError(t, stop.Validate(), "evidence_that_would_change")

	card := validPlanningIterationCard()
	card.EvidenceThatWouldChange = ""
	assertPlanningValidationError(t, card.Validate(), "evidence_that_would_change")

	candidate := validPlanCandidate()
	candidate.EvidenceThatWouldChange = ""
	assertPlanningValidationError(t, candidate.Validate(), "evidence_that_would_change")

	candidate = validPlanCandidate()
	candidate.ResidualGaps[0].EvidenceThatWouldChange = ""
	assertPlanningValidationError(t, candidate.Validate(), "residual_gaps[0].evidence_that_would_change")
}

func TestPlanningQueenRecommendationIsTypedAdviceOnly(t *testing.T) {
	t.Parallel()

	recommendation := validQueenPlanRecommendation()
	if err := recommendation.Validate(); err != nil {
		t.Fatalf("valid Queen recommendation rejected: %v", err)
	}
	if recommendation.Producer != PlanRecommendationProducerQueen || recommendation.ProducerID == "" {
		t.Fatalf("recommendation lacks authorized Queen producer identity: %#v", recommendation)
	}

	typeOf := reflect.TypeOf(QueenPlanRecommendation{})
	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		name := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
		if strings.Contains(name, "acceptance_receipt") || strings.Contains(name, "candidate_status") || field.Name == "Accepted" {
			t.Fatalf("QueenPlanRecommendation grants authority through field %s", field.Name)
		}
	}

	candidate := validPlanCandidate()
	beforeStatus := candidate.Status
	recommendation.Disposition = PlanRecommendationRevise
	if candidate.Status != beforeStatus || candidate.Acceptance != nil {
		t.Fatal("changing advisory recommendation mutated candidate or acceptance authority")
	}
}

func TestPlanningAcceptanceReceiptBindsDistinctExactAuthorities(t *testing.T) {
	t.Parallel()

	receipt := validPlanAcceptanceReceipt()
	if err := receipt.Validate(); err != nil {
		t.Fatalf("valid acceptance receipt rejected: %v", err)
	}
	typeOf := reflect.TypeOf(receipt)
	for _, fieldName := range []string{
		"CandidateID",
		"CandidateContentHash",
		"SpecificationRevisionID",
		"SpecificationRevisionHash",
		"BasePlanRevisionID",
		"BasePlanRevisionHash",
		"TimelineID",
		"TimelineDigest",
	} {
		field, ok := typeOf.FieldByName(fieldName)
		if !ok {
			t.Fatalf("PlanAcceptanceReceipt missing exact binding %s", fieldName)
		}
		copy := receipt
		reflect.ValueOf(&copy).Elem().FieldByIndex(field.Index).SetString("")
		assertPlanningValidationError(t, copy.Validate(), field.Tag.Get("json"))
	}
}

func TestPlanningPendingCandidateIsDistinctFromActiveRevision(t *testing.T) {
	t.Parallel()

	candidate := validPlanCandidate()
	plan := Plan{
		ActiveRevisionID:   "plan-rev-active",
		AcceptancePolicy:   PlanAcceptanceExplicitOwner,
		PendingCandidateID: candidate.ID,
		Candidates:         []PlanCandidate{candidate},
		Phases:             []Phase{},
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan with pending candidate: %v", err)
	}
	var decoded Plan
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal plan with pending candidate: %v", err)
	}
	if decoded.ActiveRevisionID != "plan-rev-active" {
		t.Fatalf("pending candidate replaced active revision: %q", decoded.ActiveRevisionID)
	}
	if decoded.PendingCandidateID != candidate.ID || len(decoded.Candidates) != 1 {
		t.Fatalf("custom Plan.UnmarshalJSON dropped candidate storage: %#v", decoded)
	}
	if decoded.Candidates[0].Status != PlanCandidatePendingReview || decoded.Candidates[0].Acceptance != nil {
		t.Fatalf("new candidate carries accepted authority: %#v", decoded.Candidates[0])
	}
	if !strings.Contains(string(encoded), `"active_revision_id":"plan-rev-active"`) || !strings.Contains(string(encoded), `"pending_candidate_id":"candidate-200-1"`) {
		t.Fatalf("active revision and pending candidate are not separately represented: %s", encoded)
	}
}

func TestPlanningSemanticProofBindingsRoundTripWithoutReplacingOrdinals(t *testing.T) {
	t.Parallel()

	bindings := validPlanningBindings()
	taskID := "task-ordinal-1"
	task := Task{
		ID:                        &taskID,
		Goal:                      "Keep build compatibility",
		Status:                    TaskPending,
		SemanticID:                "task-semantic-build-compatibility",
		RequirementProofLinks:     []string{"req-safe-revision"},
		AcceptanceProofLinks:      []string{"check-safe-revision"},
		NegativeProofLinks:        []string{"negative-no-auto-accept"},
		RecoveryProofLinks:        []string{"recovery-exact-replay"},
		PublicPathProofLinks:      []string{"path-ant-plan"},
		SpecificationRevisionID:   bindings.specID,
		SpecificationRevisionHash: bindings.specHash,
		CandidateID:               bindings.candidateID,
		CandidateContentHash:      bindings.candidateHash,
		PlanningTimelineID:        bindings.timelineID,
		PlanningTimelineDigest:    bindings.timelineDigest,
		AffectedSemanticIDs:       []string{"task-semantic-build-compatibility"},
		PreservedSemanticIDs:      []string{"task-semantic-existing"},
	}
	phase := Phase{
		ID:                        7,
		Name:                      "Compatibility",
		Status:                    PhasePending,
		Tasks:                     []Task{task},
		SemanticID:                "phase-semantic-compatibility",
		RequirementProofLinks:     []string{"req-safe-revision"},
		AcceptanceProofLinks:      []string{"check-safe-revision"},
		NegativeProofLinks:        []string{"negative-no-auto-accept"},
		RecoveryProofLinks:        []string{"recovery-exact-replay"},
		PublicPathProofLinks:      []string{"path-ant-plan"},
		SpecificationRevisionID:   bindings.specID,
		SpecificationRevisionHash: bindings.specHash,
		CandidateID:               bindings.candidateID,
		CandidateContentHash:      bindings.candidateHash,
		PlanningTimelineID:        bindings.timelineID,
		PlanningTimelineDigest:    bindings.timelineDigest,
		AffectedSemanticIDs:       []string{"phase-semantic-compatibility"},
		PreservedSemanticIDs:      []string{"phase-semantic-existing"},
	}
	revision := PlanRevision{
		SchemaVersion:             1,
		Number:                    2,
		ID:                        "plan-rev-2",
		CreatedAt:                 "2026-09-07T10:10:00Z",
		ReasonType:                PlanRevisionResearch,
		Reason:                    "Grounded evidence changed the route",
		PlanHash:                  "plan-rev-2-hash",
		Phases:                    []Phase{phase},
		SemanticID:                "plan-semantic-goal-200",
		RequirementProofLinks:     []string{"req-safe-revision"},
		AcceptanceProofLinks:      []string{"check-safe-revision"},
		NegativeProofLinks:        []string{"negative-no-auto-accept"},
		RecoveryProofLinks:        []string{"recovery-exact-replay"},
		PublicPathProofLinks:      []string{"path-ant-plan"},
		SpecificationRevisionID:   bindings.specID,
		SpecificationRevisionHash: bindings.specHash,
		CandidateID:               bindings.candidateID,
		CandidateContentHash:      bindings.candidateHash,
		PlanningTimelineID:        bindings.timelineID,
		PlanningTimelineDigest:    bindings.timelineDigest,
		AffectedSemanticIDs:       []string{"phase-semantic-compatibility"},
		PreservedSemanticIDs:      []string{"phase-semantic-existing"},
	}

	encoded, err := json.Marshal(revision)
	if err != nil {
		t.Fatalf("marshal bound plan revision: %v", err)
	}
	var decoded PlanRevision
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal bound plan revision: %v", err)
	}
	if !reflect.DeepEqual(decoded, revision) {
		t.Fatalf("semantic/proof bindings changed on round trip:\n got: %#v\nwant: %#v", decoded, revision)
	}
	if decoded.Phases[0].ID != 7 || decoded.Phases[0].Tasks[0].ID == nil || *decoded.Phases[0].Tasks[0].ID != taskID {
		t.Fatal("semantic identity replaced existing ordinal build/display identifiers")
	}
}

func TestPlanningLegacyPlanPhaseAndTaskStillDecode(t *testing.T) {
	t.Parallel()

	const legacy = `{
		"generated_at":null,
		"confidence":0.82,
		"active_revision_id":"legacy-active",
		"phases":[{
			"id":3,
			"name":"Legacy phase",
			"description":"Still buildable",
			"status":"ready",
			"tasks":[{"id":"3-1","goal":"Legacy task","status":"pending"}],
			"success_criteria":["Legacy behavior survives"]
		}]
	}`
	var plan Plan
	if err := json.Unmarshal([]byte(legacy), &plan); err != nil {
		t.Fatalf("unmarshal legacy plan: %v", err)
	}
	if plan.ActiveRevisionID != "legacy-active" || len(plan.Phases) != 1 || len(plan.Phases[0].Tasks) != 1 {
		t.Fatalf("legacy plan content was dropped: %#v", plan)
	}
	if plan.AcceptancePolicy != "" || plan.PendingCandidateID != "" || len(plan.Candidates) != 0 {
		t.Fatalf("legacy plan invented current planning authority: %#v", plan)
	}
	if plan.Phases[0].SemanticID != "" || plan.Phases[0].Tasks[0].SemanticID != "" {
		t.Fatal("legacy plan invented semantic identity")
	}
}

func validPlanningEvidence() PlanningEvidenceRef {
	return PlanningEvidenceRef{
		SchemaVersion:           PlanningEvidenceSchemaVersion,
		ID:                      "evidence-spec-1",
		ContentHash:             "evidence-content-hash",
		Kind:                    PlanningEvidenceSpecification,
		Origin:                  ".aether/data/COLONY_STATE.json",
		RepositoryPath:          ".aether/data/COLONY_STATE.json",
		GoalID:                  "goal-200",
		SessionID:               "session-200",
		SpecificationRevisionID: "spec-rev-1",
		PlanRevisionID:          "plan-rev-base",
		SourceRevision:          "source-rev-1",
		ObservedAt:              time.Date(2026, time.September, 7, 10, 1, 0, 0, time.UTC),
		ExcerptDigest:           "excerpt-digest",
		ApplicableDimensions:    PlanningDimensions(),
		Fresh:                   true,
		Admissible:              true,
		AdmissibilityReason:     "current exact specification revision",
	}
}

func validPlanningGap(dimension PlanningDimension, id string) PlanningGap {
	gap := PlanningGap{
		SchemaVersion:           PlanningSchemaVersion,
		Dimension:               dimension,
		Materiality:             PlanningGapNonMaterial,
		Severity:                2,
		Description:             "Clarify the remaining " + string(dimension) + " uncertainty",
		EvidenceIDs:             []string{"evidence-spec-1"},
		EvidenceThatWouldChange: "A repository receipt resolving " + string(dimension),
	}
	if err := AddressPlanningGap(&gap); err != nil {
		panic(err)
	}
	return gap
}

func validPlanningAssessments() []PlanningDimensionAssessment {
	dimensions := PlanningDimensions()
	assessments := make([]PlanningDimensionAssessment, 0, len(dimensions))
	for i, dimension := range dimensions {
		gap := validPlanningGap(dimension, "gap-"+string(dimension))
		assessment := PlanningDimensionAssessment{
			SchemaVersion:     PlanningSchemaVersion,
			Dimension:         dimension,
			Before:            60 + i,
			After:             62 + i,
			FreshEvidenceIDs:  []string{"evidence-spec-1"},
			ResolvedGapIDs:    []string{"resolved-" + string(dimension)},
			RemainingGap:      gap,
			Rationale:         "Fresh evidence improved " + string(dimension),
			ProducerReceiptID: "route-receipt-1",
		}
		if err := AddressPlanningDimensionAssessment(&assessment); err != nil {
			panic(err)
		}
		assessments = append(assessments, assessment)
	}
	return assessments
}

func validPlanningSemanticDelta() PlanningSemanticDelta {
	change := PlanningSemanticChange{
		SemanticID: "phase-semantic-1", Kind: PlanningSemanticChangeModified,
		BeforeHash: strings.Repeat("a", 64), AfterHash: strings.Repeat("b", 64), EvidenceIDs: []string{"evidence-spec-1"},
	}
	if err := AddressPlanningSemanticChange(PlanningSemanticSectionPhases, &change); err != nil {
		panic(err)
	}
	impact := PlanningAuthorityImpact{
		Kind: PlanningAuthoritySpecApproval, SourceID: "spec-approval-1", AffectedSemanticIDs: []string{"phase-semantic-1"}, Rationale: "The exact specification is approved",
	}
	if err := AddressPlanningAuthorityImpact(&impact); err != nil {
		panic(err)
	}
	delta := PlanningSemanticDelta{
		SchemaVersion:        PlanningSchemaVersion,
		Phases:               []PlanningSemanticChange{change},
		Tasks:                []PlanningSemanticChange{},
		Dependencies:         []PlanningSemanticChange{},
		RequirementLinks:     []PlanningSemanticChange{},
		AcceptanceChecks:     []PlanningSemanticChange{},
		NegativeExpectations: []PlanningSemanticChange{},
		RecoveryExpectations: []PlanningSemanticChange{},
		PublicPaths:          []PlanningSemanticChange{},
		AuthorityImpacts:     []PlanningAuthorityImpact{impact},
	}
	if err := AddressPlanningSemanticDelta(&delta); err != nil {
		panic(err)
	}
	return delta
}

func validPlanningStopDecision() PlanningStopDecision {
	return PlanningStopDecision{
		SchemaVersion:           PlanningSchemaVersion,
		ID:                      "stop-decision-1",
		ContentHash:             "stop-decision-1-hash",
		Reason:                  PlanningStopDiminishingReturns,
		SelectedGapID:           "gap-risks",
		ResidualGapIDs:          []string{"gap-risks"},
		EvidenceIDs:             []string{"evidence-spec-1"},
		Rationale:               "Two grounded passes produced less than two points of movement",
		EvidenceThatWouldChange: "A new risk receipt changes the remaining gap",
	}
}

func validPlanningIterationCard() PlanningIterationCard {
	return PlanningIterationCard{
		SchemaVersion:           PlanningIterationSchemaVersion,
		ID:                      "iteration-card-1",
		ContentHash:             "iteration-card-1-hash",
		RunID:                   "planning-run-1",
		Iteration:               1,
		ScoutReceiptID:          "scout-receipt-1",
		ScoutReceiptHash:        "scout-receipt-1-hash",
		RouteSetterReceiptID:    "route-receipt-1",
		RouteSetterReceiptHash:  "route-receipt-1-hash",
		EvidenceIDs:             []string{"evidence-spec-1"},
		DimensionAssessments:    validPlanningAssessments(),
		WeakestGap:              validPlanningGap(PlanningDimensionRisks, "gap-risks"),
		SemanticDelta:           validPlanningSemanticDelta(),
		Decision:                validPlanningStopDecision(),
		EvidenceThatWouldChange: "A fresh receipt resolves the weakest risk gap",
		CreatedAt:               time.Date(2026, time.September, 7, 10, 2, 0, 0, time.UTC),
	}
}

func validPlanningTimeline() PlanningTimelineBinding {
	return PlanningTimelineBinding{
		SchemaVersion:  PlanningTimelineSchemaVersion,
		ID:             "timeline-1",
		ContentHash:    "timeline-binding-hash",
		RunID:          "planning-run-1",
		CardIDs:        []string{"iteration-card-1"},
		FirstCardHash:  "iteration-card-1-hash",
		LastCardHash:   "iteration-card-1-hash",
		TimelineDigest: "timeline-digest-1",
		Path:           ".aether/data/planning/planning-run-1/timeline.json",
	}
}

func validQueenPlanRecommendation() QueenPlanRecommendation {
	recommendation := QueenPlanRecommendation{
		SchemaVersion: PlanningSchemaVersion,
		CandidateID:   "candidate-200-1",
		Disposition:   PlanRecommendationRevise,
		EvidenceIDs:   []string{"evidence-spec-1"},
		Rationale:     "Residual uncertainty is non-material but should remain visible",
		Producer:      PlanRecommendationProducerQueen,
		ProducerID:    "go-queen-policy/v1",
		CreatedAt:     time.Date(2026, time.September, 7, 10, 3, 0, 0, time.UTC),
	}
	if err := AddressQueenPlanRecommendation(&recommendation); err != nil {
		panic(err)
	}
	return recommendation
}

func validPlanCandidate() PlanCandidate {
	return PlanCandidate{
		SchemaVersion:             PlanCandidateSchemaVersion,
		ID:                        "candidate-200-1",
		ContentHash:               "candidate-200-1-hash",
		Status:                    PlanCandidatePendingReview,
		CreatedAt:                 time.Date(2026, time.September, 7, 10, 4, 0, 0, time.UTC),
		ExpiresAt:                 time.Date(2026, time.September, 14, 10, 4, 0, 0, time.UTC),
		Proposal:                  PlanRevision{SchemaVersion: 1, Number: 2, ID: "plan-rev-proposed", CreatedAt: "2026-09-07T10:04:00Z", ReasonType: PlanRevisionResearch, Reason: "Grounded planning iteration", PlanHash: "proposal-plan-hash", Phases: []Phase{}},
		ProposalHash:              "proposal-plan-hash",
		BasePlanRevisionID:        "plan-rev-base",
		BasePlanRevisionHash:      "plan-rev-base-hash",
		SpecificationRevisionID:   "spec-rev-1",
		SpecificationRevisionHash: "spec-rev-1-content-hash",
		Timeline:                  validPlanningTimeline(),
		StopDecision:              validPlanningStopDecision(),
		DimensionAssessments:      validPlanningAssessments(),
		SemanticDelta:             validPlanningSemanticDelta(),
		ResidualGaps:              []PlanningGap{validPlanningGap(PlanningDimensionRisks, "gap-risks")},
		EvidenceThatWouldChange:   "Fresh risk evidence would reopen the recommendation",
		Recommendation:            validQueenPlanRecommendation(),
		Acceptance:                nil,
	}
}

func validPlanAcceptanceReceipt() PlanAcceptanceReceipt {
	return PlanAcceptanceReceipt{
		SchemaVersion:             PlanAcceptanceSchemaVersion,
		ID:                        "plan-acceptance-1",
		ContentHash:               "plan-acceptance-1-hash",
		CandidateID:               "candidate-200-1",
		CandidateContentHash:      "candidate-200-1-hash",
		SpecificationRevisionID:   "spec-rev-1",
		SpecificationRevisionHash: "spec-rev-1-content-hash",
		BasePlanRevisionID:        "plan-rev-base",
		BasePlanRevisionHash:      "plan-rev-base-hash",
		TimelineID:                "timeline-1",
		TimelineDigest:            "timeline-digest-1",
		ProposalHash:              "proposal-plan-hash",
		AcceptanceTokenHash:       "acceptance-token-hash",
		AcceptedBy:                "owner-callum",
		AcceptedAt:                time.Date(2026, time.September, 7, 10, 5, 0, 0, time.UTC),
		ActivatedPlanRevisionID:   "plan-rev-2",
		ActivatedPlanRevisionHash: "plan-rev-2-hash",
	}
}

type planningBindingFixture struct {
	specID, specHash           string
	candidateID, candidateHash string
	timelineID, timelineDigest string
}

func validPlanningBindings() planningBindingFixture {
	return planningBindingFixture{
		specID:         "spec-rev-1",
		specHash:       "spec-rev-1-content-hash",
		candidateID:    "candidate-200-1",
		candidateHash:  "candidate-200-1-hash",
		timelineID:     "timeline-1",
		timelineDigest: "timeline-digest-1",
	}
}

func assertPlanningEnumSet[T ~string](t *testing.T, name string, values []T, valid func(T) bool) {
	t.Helper()
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			t.Fatalf("%s %q is not valid", name, value)
		}
		if _, duplicate := seen[value]; duplicate {
			t.Fatalf("%s %q is duplicated", name, value)
		}
		seen[value] = struct{}{}
	}
}

func assertPlanningValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q", field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Fatalf("validation error %q does not contain %q", err, field)
	}
}
