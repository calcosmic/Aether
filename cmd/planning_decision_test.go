package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningDecisionClassifyMaterialAndSuppressesNonOwnerChoices(t *testing.T) {
	t.Parallel()

	material := planningDecisionCandidate{
		StableID:            "decision-auth-mode",
		Domain:              planningDecisionDomainAuthority,
		Decision:            "May the generated plan change the authentication boundary?",
		WhyNow:              "The evidence supports two implementations with different authority.",
		Evidence:            []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-auth", false)},
		QueenRecommendation: "Keep the existing authentication boundary.",
		Impact: planningDecisionContractImpact{
			Authority: "changes who may approve authentication behavior",
		},
		AffectedSemanticIDs: []string{"requirement:auth-01"},
		ResumeInstruction:   "Answer once to resume planning from this batch.",
	}

	classification, err := classifyPlanningDecision(material)
	if err != nil {
		t.Fatalf("classify material decision: %v", err)
	}
	if !classification.RequiresOwner || classification.Reason != planningDecisionReasonMaterial {
		t.Fatalf("material classification = %+v, want owner material", classification)
	}

	evidenceAnswerable := material
	evidenceAnswerable.StableID = "decision-auth-evidence"
	evidenceAnswerable.Evidence = []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-auth-answer", true)}
	evidenceAnswerable.AnswerableEvidenceIDs = []string{"evidence-auth-answer"}
	classification, err = classifyPlanningDecision(evidenceAnswerable)
	if err != nil {
		t.Fatalf("classify evidence-answerable decision: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonEvidenceAnswerable {
		t.Fatalf("evidence-answerable classification = %+v, want autonomous suppression", classification)
	}

	preference := material
	preference.StableID = "decision-colour"
	preference.Decision = "Which colour should the planning heading use?"
	preference.Impact = planningDecisionContractImpact{}
	preference.AffectedSemanticIDs = nil
	classification, err = classifyPlanningDecision(preference)
	if err != nil {
		t.Fatalf("classify generic preference: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonGenericPreference {
		t.Fatalf("generic preference classification = %+v, want autonomous suppression", classification)
	}

	routine := material
	routine.StableID = "decision-research-depth"
	routine.Domain = planningDecisionDomainRoutineResearch
	routine.Decision = "How many Scout sources should be collected?"
	classification, err = classifyPlanningDecision(routine)
	if err != nil {
		t.Fatalf("classify routine research choice: %v", err)
	}
	if classification.RequiresOwner || classification.Reason != planningDecisionReasonRoutineAutonomous {
		t.Fatalf("routine classification = %+v, want autonomous handling", classification)
	}
}

func TestPlanningDecisionBatchStableFirstPass(t *testing.T) {
	t.Parallel()

	boundary := planningDecisionBoundary{
		RunID: "planning-run-1",
		Pass:  1,
		ScoutReceipt: planningDecisionStageReceipt{
			ID: "scout-receipt-1", ContentHash: strings.Repeat("a", 64), RunID: "planning-run-1", Pass: 1,
		},
		RecoveryCommand: "aether plan --resume planning-run-1",
	}
	candidates := []planningDecisionCandidate{
		planningDecisionMaterialFixture("scope-z", planningDecisionDomainScope, "task:scope-z"),
		planningDecisionMaterialFixture("behavior-a", planningDecisionDomainBehavior, "requirement:behavior-a"),
		planningDecisionMaterialFixture("risk-c", planningDecisionDomainRiskTolerance, "acceptance:risk-c"),
	}

	batch, err := buildPlanningDecisionBatch(boundary, candidates)
	if err != nil {
		t.Fatalf("build first-pass batch: %v", err)
	}
	if batch == nil || len(batch.Decisions) != 3 {
		t.Fatalf("batch = %+v, want one batch with all three decisions", batch)
	}
	wantOrder := []string{"behavior-a", "risk-c", "scope-z"}
	gotOrder := make([]string, 0, len(batch.Decisions))
	for _, decision := range batch.Decisions {
		gotOrder = append(gotOrder, decision.StableID)
		if len(decision.Evidence) == 0 || len(decision.AffectedSemanticIDs) == 0 {
			t.Fatalf("decision lost evidence or affected IDs: %+v", decision)
		}
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("decision order = %v, want %v", gotOrder, wantOrder)
	}

	reversed := []planningDecisionCandidate{candidates[2], candidates[1], candidates[0]}
	again, err := buildPlanningDecisionBatch(boundary, reversed)
	if err != nil {
		t.Fatalf("build reversed batch: %v", err)
	}
	if !reflect.DeepEqual(again, batch) {
		t.Fatalf("input order changed batch:\n got: %+v\nwant: %+v", again, batch)
	}
}

func TestPlanningDecisionBoundaryRequiresCompletedLaterPass(t *testing.T) {
	t.Parallel()

	candidate := planningDecisionMaterialFixture("acceptance-late", planningDecisionDomainAcceptanceMeaning, "acceptance:late")
	boundary := planningDecisionBoundary{
		RunID: "planning-run-2",
		Pass:  2,
		ScoutReceipt: planningDecisionStageReceipt{
			ID: "scout-receipt-2", ContentHash: strings.Repeat("b", 64), RunID: "planning-run-2", Pass: 2,
		},
		RecoveryCommand: "aether plan --resume planning-run-2",
	}

	if _, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate}); err == nil || !strings.Contains(err.Error(), "route-setter") {
		t.Fatalf("Scout-only later boundary error = %v, want route-setter requirement", err)
	}

	boundary.RouteSetterReceipt = &planningDecisionStageReceipt{
		ID: "route-receipt-2", ContentHash: strings.Repeat("c", 64), RunID: "planning-run-2", Pass: 2,
	}
	if _, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate}); err == nil || !strings.Contains(err.Error(), "iteration card") {
		t.Fatalf("receipt-only later boundary error = %v, want iteration card requirement", err)
	}

	boundary.IterationCard = &planningDecisionIterationCardReceipt{
		ID:                     "iteration-card-2",
		ContentHash:            strings.Repeat("d", 64),
		RunID:                  "planning-run-2",
		Pass:                   2,
		ScoutReceiptID:         boundary.ScoutReceipt.ID,
		ScoutReceiptHash:       boundary.ScoutReceipt.ContentHash,
		RouteSetterReceiptID:   boundary.RouteSetterReceipt.ID,
		RouteSetterReceiptHash: boundary.RouteSetterReceipt.ContentHash,
	}
	batch, err := buildPlanningDecisionBatch(boundary, []planningDecisionCandidate{candidate})
	if err != nil {
		t.Fatalf("build completed later-pass batch: %v", err)
	}
	if batch == nil || batch.BoundaryCardHash != boundary.IterationCard.ContentHash {
		t.Fatalf("later batch = %+v, want persisted card binding", batch)
	}
}

func planningDecisionMaterialFixture(id string, domain planningDecisionDomain, semanticID string) planningDecisionCandidate {
	return planningDecisionCandidate{
		StableID:            id,
		Domain:              domain,
		Decision:            "Choose the material outcome for " + id,
		WhyNow:              "Current evidence leaves materially different outcomes.",
		Evidence:            []colony.PlanningEvidenceRef{planningDecisionEvidenceFixture("evidence-"+id, false)},
		QueenRecommendation: "Preserve the approved contract.",
		Impact:              planningDecisionContractImpact{Behavior: "changes " + id},
		AffectedSemanticIDs: []string{semanticID},
		ResumeInstruction:   "Answer this batch to resume planning.",
	}
}

func planningDecisionEvidenceFixture(id string, answerable bool) colony.PlanningEvidenceRef {
	return colony.PlanningEvidenceRef{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		ID:                      id,
		ContentHash:             strings.Repeat("e", 64),
		Kind:                    colony.PlanningEvidenceResearch,
		Origin:                  "fixture:" + id,
		GoalID:                  "goal-1",
		SessionID:               "session-1",
		SpecificationRevisionID: "spec-1",
		PlanRevisionID:          "plan-1",
		SourceRevision:          "source-1",
		ObservedAt:              time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ExcerptDigest:           strings.Repeat("f", 64),
		ApplicableDimensions:    []colony.PlanningDimension{colony.PlanningDimensionRequirements},
		Fresh:                   true,
		Admissible:              true,
		AdmissibilityReason:     map[bool]string{true: "directly answers the candidate", false: "supports why owner authority is required"}[answerable],
	}
}

func TestPlanningDecisionCardContainsEvidenceFirstFields(t *testing.T) {
	t.Parallel()

	candidate := planningDecisionMaterialFixture("behavior-card", planningDecisionDomainBehavior, "requirement:behavior-card")
	candidate.Choices = []planningDecisionChoice{
		{
			ID: "preserve", Label: "Preserve the approved behavior",
			Consequence: "The plan stays within the approved contract.",
			Impact:      candidate.Impact,
		},
		{
			ID: "expand", Label: "Expand the behavior",
			Consequence:         "A successor specification must approve the broader behavior.",
			Impact:              planningDecisionContractImpact{Behavior: "expanded behavior"},
			AffectedSemanticIDs: []string{"requirement:behavior-card"},
		},
	}
	scope := planningDecisionEquivalenceScopeFixture()
	priorKey, err := buildPlanningDecisionEquivalenceKey(scope, candidate)
	if err != nil {
		t.Fatalf("build prior equivalence key: %v", err)
	}
	prior := &planningDecisionAnswerRecord{
		DecisionID: candidate.StableID,
		ChoiceID:   "preserve",
		Answer:     "Preserve the approved behavior",
		Key:        priorKey,
	}
	driftedScope := scope
	driftedScope.BasePlanRevisionID = "plan-revision-2"

	card, err := projectPlanningDecisionCard(planningDecisionCardRequest{
		Candidate: candidate,
		Scope:     driftedScope,
		Prior:     prior,
	})
	if err != nil {
		t.Fatalf("project decision card: %v", err)
	}
	for name, value := range map[string]string{
		"decision":         card.Decision,
		"why now":          card.WhyNow,
		"recommendation":   card.QueenRecommendation,
		"prior answer":     card.PriorAnswer,
		"revalidation":     card.Revalidation,
		"planning resumes": card.PlanningResumes,
	} {
		if strings.TrimSpace(value) == "" {
			t.Errorf("card %s is empty: %+v", name, card)
		}
	}
	if len(card.Evidence) == 0 || strings.TrimSpace(card.Evidence[0].ID) == "" {
		t.Fatalf("card lacks evidence citation: %+v", card)
	}
	if len(card.Choices) != 2 {
		t.Fatalf("card choices = %d, want only two viable choices", len(card.Choices))
	}
	for _, choice := range card.Choices {
		if strings.TrimSpace(choice.Consequence) == "" {
			t.Fatalf("choice lacks consequence: %+v", choice)
		}
	}
	if !reflect.DeepEqual(card.AffectedSemanticIDs, []string{"requirement:behavior-card"}) {
		t.Fatalf("affected scope = %v", card.AffectedSemanticIDs)
	}
}

func TestPlanningDecisionReuseRequiresExactEquivalence(t *testing.T) {
	t.Parallel()

	candidate := planningDecisionMaterialFixture("scope-reuse", planningDecisionDomainScope, "requirement:scope-reuse")
	candidate.Impact = planningDecisionContractImpact{
		Behavior:   "preserve export semantics",
		Scope:      "include enterprise exports",
		Risk:       "owner accepts migration risk",
		Acceptance: "exports remain byte-for-byte compatible",
	}
	scope := planningDecisionEquivalenceScopeFixture()
	key, err := buildPlanningDecisionEquivalenceKey(scope, candidate)
	if err != nil {
		t.Fatalf("build equivalence key: %v", err)
	}
	prior := planningDecisionAnswerRecord{
		DecisionID: candidate.StableID,
		ChoiceID:   "include",
		Answer:     "Include enterprise exports",
		Key:        key,
	}

	reuse := assessPlanningDecisionAnswerReuse(key, &prior)
	if !reuse.Reused || reuse.RequiresRevalidation {
		t.Fatalf("exact semantic answer was not reused: %+v", reuse)
	}

	tests := []struct {
		name   string
		mutate func(*planningDecisionEquivalenceKey)
	}{
		{name: "goal", mutate: func(value *planningDecisionEquivalenceKey) { value.GoalID = "goal-2" }},
		{name: "session", mutate: func(value *planningDecisionEquivalenceKey) { value.SessionID = "session-2" }},
		{name: "spec revision", mutate: func(value *planningDecisionEquivalenceKey) { value.ApprovedSpecificationRevisionID = "spec-revision-2" }},
		{name: "plan revision", mutate: func(value *planningDecisionEquivalenceKey) { value.BasePlanRevisionID = "plan-revision-2" }},
		{name: "decision meaning", mutate: func(value *planningDecisionEquivalenceKey) {
			value.NormalizedDecisionText = "a different material decision"
		}},
		{name: "behavior", mutate: func(value *planningDecisionEquivalenceKey) { value.BehaviorImpact = "different behavior" }},
		{name: "scope", mutate: func(value *planningDecisionEquivalenceKey) { value.ScopeImpact = "different scope" }},
		{name: "risk", mutate: func(value *planningDecisionEquivalenceKey) { value.RiskImpact = "different risk" }},
		{name: "acceptance", mutate: func(value *planningDecisionEquivalenceKey) { value.AcceptanceImpact = "different acceptance" }},
		{name: "affected IDs", mutate: func(value *planningDecisionEquivalenceKey) { value.AffectedSemanticIDs = []string{"requirement:other"} }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			drifted := key
			drifted.AffectedSemanticIDs = append([]string(nil), key.AffectedSemanticIDs...)
			test.mutate(&drifted)
			drifted.ContentHash = ""
			contentHash, hashErr := planningDecisionEquivalenceHash(drifted)
			if hashErr != nil {
				t.Fatalf("rehash drifted key: %v", hashErr)
			}
			drifted.ContentHash = contentHash
			result := assessPlanningDecisionAnswerReuse(drifted, &prior)
			if result.Reused || !result.RequiresRevalidation || result.PriorAnswerEvidence != prior.Answer {
				t.Fatalf("%s drift reused prior answer: %+v", test.name, result)
			}
		})
	}
}

func TestPlanningDecisionReuseResolutionSeparatesContractChanges(t *testing.T) {
	t.Parallel()

	approved := planningDecisionContractImpact{
		Behavior:   "preserve export semantics",
		Scope:      "CLI exports only",
		Risk:       "no compatibility regression",
		Acceptance: "golden export remains identical",
	}
	direct, err := resolvePlanningDecisionAnswer(planningDecisionResolutionRequest{
		DecisionID:     "behavior-resolution",
		ApprovedImpact: approved,
		SelectedChoice: planningDecisionChoice{
			ID: "preserve", Label: "Preserve", Consequence: "No approved contract changes.", Impact: approved,
		},
	})
	if err != nil {
		t.Fatalf("resolve equivalent answer: %v", err)
	}
	if direct.Disposition != planningDecisionDispositionDirectResume || len(direct.AffectedSemanticIDs) != 0 {
		t.Fatalf("equivalent resolution = %+v, want direct resume", direct)
	}

	successor, err := resolvePlanningDecisionAnswer(planningDecisionResolutionRequest{
		DecisionID:     "behavior-resolution",
		ApprovedImpact: approved,
		SelectedChoice: planningDecisionChoice{
			ID: "expand", Label: "Expand", Consequence: "Broaden the promised behavior.",
			Impact: planningDecisionContractImpact{
				Behavior:   approved.Behavior,
				Scope:      "CLI and API exports",
				Risk:       approved.Risk,
				Acceptance: approved.Acceptance,
			},
			AffectedSemanticIDs: []string{"requirement:export-01"},
		},
	})
	if err != nil {
		t.Fatalf("resolve contract-changing answer: %v", err)
	}
	if successor.Disposition != planningDecisionDispositionSuccessorSpecRequired {
		t.Fatalf("contract-changing resolution = %+v, want successor specification", successor)
	}
	if !reflect.DeepEqual(successor.AffectedSemanticIDs, []string{"requirement:export-01"}) || len(successor.RevisionEvidence) == 0 {
		t.Fatalf("successor resolution lost affected IDs or revision evidence: %+v", successor)
	}
}

func TestPlanningDecisionResumeExactRetryAndRejectsDrift(t *testing.T) {
	t.Parallel()

	candidate := planningDecisionMaterialFixture("resume-scope", planningDecisionDomainScope, "requirement:resume-scope")
	key, err := buildPlanningDecisionEquivalenceKey(planningDecisionEquivalenceScopeFixture(), candidate)
	if err != nil {
		t.Fatalf("build answer key: %v", err)
	}
	binding := planningDecisionResumeBinding{
		GoalID:              "goal-1",
		SessionID:           "session-1",
		BatchID:             "planning-decision-batch-1234",
		BatchHash:           strings.Repeat("1", 64),
		FrontierReceiptHash: strings.Repeat("2", 64),
		Answers: []planningDecisionBoundAnswer{
			{DecisionID: candidate.StableID, ChoiceID: "preserve", Answer: "Preserve", EquivalenceKey: key},
		},
		Disposition:     planningDecisionDispositionDirectResume,
		RecoveryCommand: "aether plan --resume planning-run-1",
	}
	token, err := issuePlanningDecisionResumeToken(binding)
	if err != nil {
		t.Fatalf("issue resume token: %v", err)
	}
	again, err := issuePlanningDecisionResumeToken(binding)
	if err != nil {
		t.Fatalf("issue identical resume token: %v", err)
	}
	if !reflect.DeepEqual(again, token) {
		t.Fatalf("token issuance is not content-addressed:\n got: %+v\nwant: %+v", again, token)
	}

	valid, err := validatePlanningDecisionResumeToken(token, binding)
	if err != nil {
		t.Fatalf("validate exact token: %v", err)
	}
	if !valid.Accepted || !valid.Idempotent || valid.Status != planningDecisionResumeValid {
		t.Fatalf("exact retry validation = %+v", valid)
	}

	staleBinding := binding
	staleBinding.SessionID = "session-2"
	stale, err := validatePlanningDecisionResumeToken(token, staleBinding)
	if err != nil {
		t.Fatalf("validate stale token: %v", err)
	}
	if stale.Accepted || stale.Status != planningDecisionResumeStale || stale.RecoveryCommand != staleBinding.RecoveryCommand {
		t.Fatalf("wrong-session validation = %+v, want stale with recovery", stale)
	}

	altered := token
	altered.Answers = append([]planningDecisionBoundAnswer(nil), token.Answers...)
	altered.Answers[0].ChoiceID = "expand"
	divergent, err := validatePlanningDecisionResumeToken(altered, binding)
	if err != nil {
		t.Fatalf("validate altered token: %v", err)
	}
	if divergent.Accepted || divergent.Status != planningDecisionResumeDivergent || divergent.RecoveryCommand != binding.RecoveryCommand {
		t.Fatalf("altered-token validation = %+v, want divergent with recovery", divergent)
	}

	cardBinding := binding
	cardBinding.FrontierReceiptHash = ""
	cardBinding.CompletedCardHash = strings.Repeat("3", 64)
	if _, err := issuePlanningDecisionResumeToken(cardBinding); err != nil {
		t.Fatalf("completed-card binding should be accepted: %v", err)
	}
	both := cardBinding
	both.FrontierReceiptHash = strings.Repeat("2", 64)
	if _, err := issuePlanningDecisionResumeToken(both); err == nil {
		t.Fatal("token bound to both frontier and card was accepted")
	}
}

func planningDecisionEquivalenceScopeFixture() planningDecisionEquivalenceScope {
	return planningDecisionEquivalenceScope{
		GoalID:                          "goal-1",
		SessionID:                       "session-1",
		ApprovedSpecificationRevisionID: "spec-revision-1",
		BasePlanRevisionID:              "plan-revision-1",
	}
}
