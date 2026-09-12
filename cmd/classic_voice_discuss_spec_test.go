package cmd

// Phase "Classic Visual Voice" plan 03 -- the discussion card and the
// specification card carry their own symbols.
//
// Neither card has a February ancestor (RESEARCH.md Pitfall 5): the
// discussion card's per-question content is materially richer than
// anything Classic produced, and the specification command did not exist
// before v1.28. What both lacked was entirely presentational -- this plan
// wires both through voiceLine/voiceGlyph (plan 01's funnel) without
// touching a single field's meaning or value.

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// classicVoiceDiscussQuestionsFixture is the unresolved-questions branch of
// renderDiscussVisual: two questions, one with a prior answer and one
// without, one marked a hard constraint, each carrying a full field set and
// at least two choices.
func classicVoiceDiscussQuestionsFixture() map[string]interface{} {
	return map[string]interface{}{
		"goal":           "Ship the billing rewrite",
		"question_count": 2,
		"created_count":  2,
		"existing_count": 0,
		"resolved_count": 1,
		"questions": []discussQuestion{
			{
				StableID: "D1",
				Question: "Should planning persist local file state or a remote store?",
				WhyNow:   "The next phase writes durable evidence that must survive a crash",
				Evidence: []colony.PlanningEvidenceRef{
					{ID: "EVIDENCE-01", Kind: colony.PlanningEvidenceResearch, Origin: "scout-pass-1"},
				},
				QueenRecommendation: "Keep state in the local durable store",
				Choices: []planningDecisionChoice{
					{ID: "CHOICE-01", Label: "Local durable store", Consequence: "No network dependency, but state is single-machine"},
					{ID: "CHOICE-02", Label: "Remote store", Consequence: "State survives machine loss, but adds a network dependency"},
				},
				AffectedSemanticIDs: []string{"REQ-01", "REC-01"},
				PriorAnswer:         "",
				Revalidation:        "Re-ask if the deployment target changes from single-machine to distributed",
				PlanningResumes:     "Scout pass 2",
				ExactAnswerSyntax:   "/ant-discuss --answer D1=local",
				HardConstraint:      true,
			},
			{
				StableID: "D2",
				Question: "Should the retry budget be fixed or configurable?",
				WhyNow:   "The recovery contract needs one settled number before it can be tested",
				Evidence: []colony.PlanningEvidenceRef{
					{ID: "EVIDENCE-02", Kind: colony.PlanningEvidenceDecision, Origin: "prior-decision-04"},
				},
				QueenRecommendation: "Keep the budget fixed at three attempts",
				Choices: []planningDecisionChoice{
					{ID: "CHOICE-03", Label: "Fixed at three attempts", Consequence: "Predictable, but not tunable per environment"},
					{ID: "CHOICE-04", Label: "Configurable", Consequence: "Tunable, but adds a new setting to document and test"},
				},
				AffectedSemanticIDs: []string{"REC-02"},
				PriorAnswer:         "Fixed at 3 attempts",
				Revalidation:        "Re-ask if a second environment needs a different budget",
				PlanningResumes:     "Route-Setter pass 1",
				ExactAnswerSyntax:   "/ant-discuss --answer D2=fixed",
				HardConstraint:      false,
			},
		},
	}
}

// classicVoiceDiscussResolvedFixture is the resolved-clarification branch:
// a locked clarification with a steering note (REDIRECT) emitted.
func classicVoiceDiscussResolvedFixture() map[string]interface{} {
	return map[string]interface{}{
		"resolved":         true,
		"id":               "D1",
		"answer":           "local",
		"redirect_emitted": true,
		"redirect_text":    "Never store planning evidence in a remote store without explicit owner approval",
	}
}

// classicVoiceSpecFixture populates every section with at least one item, at
// least one item carrying a detail (AcceptanceChecks/AffectedPublicPaths
// naturally produce one), at least one empty section (Exclusions), a full
// classified delta, an approval and a transaction receipt.
func classicVoiceSpecFixture() specCommandResult {
	return specCommandResult{
		Command:          "spec",
		Operation:        specCommandOperationInspect,
		SpecificationID:  "SPEC-01",
		BeforeRevisionID: "SPEC-REV-00",
		AfterRevisionID:  "SPEC-REV-01",
		RevisionNumber:   1,
		Status:           colony.SpecStatusApproved,
		Scope:            colony.SpecScope{Kind: colony.SpecScopeFeature, FeatureID: "FEATURE-01"},
		Outcomes:         []colony.SpecOutcome{{ID: "OUT-01", Description: "A trusted plan"}},
		IncludedBehaviors: []colony.SpecIncludedBehavior{
			{ID: "BEH-01", Description: "Iterative research"},
		},
		Exclusions:       nil,
		BindingDecisions: []colony.SpecBindingDecision{{ID: "DEC-01", Description: "Owner accepts plans"}},
		Requirements:     []colony.SpecRequirement{{ID: "REQ-01", Description: "Keep stable identity"}},
		AcceptanceChecks: []colony.SpecAcceptanceCheck{
			{ID: "ACC-01", Description: "Shows evidence", Verification: "Inspect output"},
		},
		NegativeExpectations: []colony.SpecNegativeExpectation{{ID: "NEG-01", Description: "Never imply authority"}},
		RecoveryExpectations: []colony.SpecRecoveryExpectation{{ID: "REC-01", Description: "Show one recovery action"}},
		AffectedPublicPaths: []colony.SpecPublicPath{
			{ID: "PATH-01", Path: "aether plan", Description: "Planning entrypoint"},
		},
		ClassifiedDelta: colony.SpecRevisionDelta{
			Outcomes:             colony.SpecItemDelta{AddedIDs: []string{"OUT-01"}, UnchangedIDs: []string{}},
			IncludedBehaviors:    colony.SpecItemDelta{ModifiedIDs: []string{"BEH-01"}},
			Exclusions:           colony.SpecItemDelta{},
			BindingDecisions:     colony.SpecItemDelta{AddedIDs: []string{"DEC-01"}},
			Requirements:         colony.SpecItemDelta{AddedIDs: []string{"REQ-01"}},
			AcceptanceChecks:     colony.SpecItemDelta{AddedIDs: []string{"ACC-01"}},
			NegativeExpectations: colony.SpecItemDelta{AddedIDs: []string{"NEG-01"}},
			RecoveryExpectations: colony.SpecItemDelta{AddedIDs: []string{"REC-01"}},
			AffectedPublicPaths:  colony.SpecItemDelta{AddedIDs: []string{"PATH-01"}},
		},
		AffectedScope: specificationAffectedScope{
			SpecItemIDs:  []string{"OUT-01"},
			TaskIDs:      []string{"TASK-01"},
			ProofLinkIDs: []string{"PROOF-01"},
		},
		Approval: &colony.SpecApprovalReceipt{
			ID:              "APPROVAL-01",
			SpecificationID: "SPEC-01",
			RevisionID:      "SPEC-REV-01",
		},
		Receipt: &colony.LifecycleReceipt{
			ReceiptID: "RECEIPT-01",
			Command:   "spec",
		},
		Projection: specCommandProjection{
			Path:       "spec-projection.json",
			RevisionID: "SPEC-REV-01",
		},
		StateEffect: "none",
		NextAction:  "aether plan",
	}
}

// ---------------------------------------------------------------------------
// Corpus registration -- one init per screen, in this screen's own test file.
// ---------------------------------------------------------------------------

func init() {
	registerVoiceScreen("discuss-questions", func(t *testing.T) string {
		t.Helper()
		return renderDiscussVisual(classicVoiceDiscussQuestionsFixture())
	})
	registerVoiceScreen("discuss-resolved", func(t *testing.T) string {
		t.Helper()
		return renderDiscussVisual(classicVoiceDiscussResolvedFixture())
	})
	registerVoiceScreen("spec", func(t *testing.T) string {
		t.Helper()
		return renderSpecCommandVisual(classicVoiceSpecFixture())
	})
}

// ---------------------------------------------------------------------------
// Task 1 -- the discussion card.
// ---------------------------------------------------------------------------

// TestDiscussScreenMeetsTheReferenceDensity asserts both the questions branch
// and the resolved branch measure at or above the figure derived from the
// reference commits.
func TestDiscussScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)

	cases := map[string]string{
		"questions branch": renderDiscussVisual(classicVoiceDiscussQuestionsFixture()),
		"resolved branch":  renderDiscussVisual(classicVoiceDiscussResolvedFixture()),
	}
	for name, rendered := range cases {
		name, rendered := name, rendered
		t.Run(name, func(t *testing.T) {
			led, total, ratio := voiceDensity(rendered)
			if total == 0 {
				t.Fatalf("the %s produced no content lines to measure:\n%s", name, rendered)
			}
			if ratio < reference {
				t.Errorf("the %s measures %v (led=%d total=%d), below the reference figure %v:\n%s",
					name, ratio, led, total, reference, rendered)
			}
		})
	}
}

// TestDiscussCardKeepsEveryQuestionField asserts every per-question field
// from the fixture still appears in the rendering, in text, unchanged, and
// that the exact-answer syntax the owner must type is present verbatim --
// as its own subtest, per the plan's acceptance criteria.
func TestDiscussCardKeepsEveryQuestionField(t *testing.T) {
	fixture := classicVoiceDiscussQuestionsFixture()
	rendered := renderDiscussVisual(fixture)
	questions := fixture["questions"].([]discussQuestion)

	for _, question := range questions {
		question := question
		t.Run(question.StableID+" fields", func(t *testing.T) {
			wants := []string{
				question.StableID,
				question.Question,
				question.WhyNow,
				question.QueenRecommendation,
				question.Revalidation,
				question.PlanningResumes,
				question.ExactAnswerSyntax,
			}
			for _, choice := range question.Choices {
				wants = append(wants, choice.Label, choice.Consequence)
			}
			for _, id := range question.AffectedSemanticIDs {
				wants = append(wants, id)
			}
			if strings.TrimSpace(question.PriorAnswer) == "" {
				wants = append(wants, "Prior answer: none")
			} else {
				wants = append(wants, question.PriorAnswer)
			}
			if question.HardConstraint {
				wants = append(wants, "This answer becomes a hard constraint.")
			}
			for _, want := range wants {
				if !strings.Contains(rendered, want) {
					t.Errorf("rendering lost field %q for question %s:\n%s", want, question.StableID, rendered)
				}
			}
		})
	}

	t.Run("exact answer syntax is verbatim", func(t *testing.T) {
		for _, question := range questions {
			if !strings.Contains(rendered, question.ExactAnswerSyntax) {
				t.Errorf("exact answer syntax %q for question %s does not appear verbatim in the rendering:\n%s",
					question.ExactAnswerSyntax, question.StableID, rendered)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2 -- the specification card.
// ---------------------------------------------------------------------------

// TestSpecScreenMeetsTheReferenceDensity asserts the specification card
// measures at or above the figure derived from the reference commits.
func TestSpecScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	rendered := renderSpecCommandVisual(classicVoiceSpecFixture())

	led, total, ratio := voiceDensity(rendered)
	if total == 0 {
		t.Fatalf("the spec card produced no content lines to measure:\n%s", rendered)
	}
	if ratio < reference {
		t.Errorf("the spec card measures %v (led=%d total=%d), below the reference figure %v:\n%s",
			ratio, led, total, reference, rendered)
	}
}

// TestSpecCardLabelsEveryIdentifier asserts every identifier field the
// fixture supplies -- the ones that let the owner tell one revision, one
// approval, or one receipt from another -- still appears on a line carrying
// a plain-English label beside it, rather than standing alone.
func TestSpecCardLabelsEveryIdentifier(t *testing.T) {
	result := classicVoiceSpecFixture()
	rendered := renderSpecCommandVisual(result)
	lines := strings.Split(rendered, "\n")

	checks := []struct {
		name  string
		label string
		id    string
	}{
		{"specification ID", "SPEC:", result.SpecificationID},
		{"before revision ID", "Before:", result.BeforeRevisionID},
		{"after revision ID", "After:", result.AfterRevisionID},
		{"feature ID", "Feature:", result.Scope.FeatureID},
		{"approval receipt ID", "Approval receipt:", result.Approval.ID},
		{"transaction receipt ID", "Transaction receipt:", result.Receipt.ReceiptID},
	}

	for _, check := range checks {
		check := check
		t.Run(check.name, func(t *testing.T) {
			for _, line := range lines {
				if strings.Contains(line, check.label) && strings.Contains(line, check.id) {
					return
				}
			}
			t.Errorf("%s (%q) does not appear on a line labelled %q:\n%s", check.name, check.id, check.label, rendered)
		})
	}
}
