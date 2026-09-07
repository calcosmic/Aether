package cmd

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestDiscussEvidenceResolvedEmitsNoGenericFallbackQuestions(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Keep the existing Go command surface and its current verification contract"
	sessionID := "session-discuss-evidence-resolved"
	initializedAt := time.Date(2026, time.September, 7, 10, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
		AcceptedCharter: &colony.AcceptedCharter{
			SchemaVersion: colony.AcceptedCharterSchemaVersion,
			EpisodeID:     "episode-discuss-evidence-resolved",
			Goal:          goal,
			Provenance:    "owner-approved test charter",
			AcceptedAt:    initializedAt,
			Charter: &colony.Charter{
				Intent:      goal,
				Vision:      "Preserve the current command behavior.",
				Governance:  "Existing Go validation remains authoritative.",
				Goals:       "No new surface, dependency, or acceptance policy.",
				TechStack:   "Go and Cobra",
				KeyRisks:    "Avoid contract drift.",
				Constraints: "Use existing contracts and focused regression tests.",
			},
		},
	})

	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("run evidence-resolved discuss: %v", err)
	}
	if got := intValue(result["question_count"]); got != 0 {
		t.Fatalf("question_count = %d, want 0; evidence-resolved intent must not trigger generic fallback questions: %#v", got, result["questions"])
	}
	if got := stringValue(result["discussion_status"]); got != "settled" {
		t.Fatalf("discussion_status = %q, want settled", got)
	}
	if got := intValue(result["evidence_count"]); got < 2 {
		t.Fatalf("evidence_count = %d, want charter plus current context", got)
	}

	file := loadPendingDecisionFile()
	if len(file.Decisions) != 0 {
		t.Fatalf("evidence-resolved discuss persisted fallback questions: %#v", file.Decisions)
	}
}

func TestDiscussMaterialBatchContainsAllFiveDomainsAndDecisionFields(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Choose the remaining owner contract boundaries"
	sessionID := "session-discuss-five-domains"
	initializedAt := time.Date(2026, time.September, 7, 11, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
	})

	domains := []string{"acceptance_meaning", "scope", "risk_tolerance", "authority", "behavior"}
	decisions := make([]PendingDecision, 0, len(domains))
	for index, domain := range domains {
		decisions = append(decisions, PendingDecision{
			ID:          "pd-material-" + domain,
			Type:        clarificationDecisionType,
			Description: formatClarificationDescription("Choose the "+domain+" contract", []string{"preserve current " + domain, "change " + domain}),
			Source:      "wrapper:" + domain + ":contract-" + domain,
			GoalHash:    pendingDecisionGoalHash(goal),
			SessionID:   sessionID,
			Grounding:   "Repository evidence leaves two materially different " + domain + " outcomes.",
			CreatedAt:   initializedAt.Add(time.Duration(index+1) * time.Minute).Format(time.RFC3339),
		})
	}
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: decisions}); err != nil {
		t.Fatalf("seed material decisions: %v", err)
	}

	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("run material discuss: %v", err)
	}
	if got := intValue(result["question_count"]); got != 5 {
		t.Fatalf("question_count = %d, want all 5 material domains despite legacy max=3", got)
	}
	batch := discussResultJSONMap(t, result["material_batch"])
	rawCards, ok := batch["cards"].([]interface{})
	if !ok || len(rawCards) != 5 {
		t.Fatalf("material batch cards = %#v, want 5", batch["cards"])
	}
	wantDomains := []string{"behavior", "authority", "risk_tolerance", "scope", "acceptance_meaning"}
	gotDomains := make([]string, 0, len(rawCards))
	for _, raw := range rawCards {
		card := raw.(map[string]interface{})
		gotDomains = append(gotDomains, stringValue(card["domain"]))
		for _, field := range []string{"decision", "why_now", "queen_recommendation", "planning_resumes", "exact_answer_syntax"} {
			if strings.TrimSpace(stringValue(card[field])) == "" {
				t.Errorf("%s card lacks %s: %#v", stringValue(card["domain"]), field, card)
			}
		}
		if evidence, ok := card["evidence"].([]interface{}); !ok || len(evidence) == 0 {
			t.Errorf("%s card lacks cited evidence: %#v", stringValue(card["domain"]), card)
		}
		if affected, ok := card["affected_semantic_ids"].([]interface{}); !ok || len(affected) == 0 {
			t.Errorf("%s card lacks affected scope: %#v", stringValue(card["domain"]), card)
		}
		choices, ok := card["choices"].([]interface{})
		if !ok || len(choices) != 2 {
			t.Errorf("%s choices = %#v, want two viable choices", stringValue(card["domain"]), card["choices"])
			continue
		}
		for _, rawChoice := range choices {
			choice := rawChoice.(map[string]interface{})
			if strings.TrimSpace(stringValue(choice["consequence"])) == "" {
				t.Errorf("%s choice lacks consequence: %#v", stringValue(card["domain"]), choice)
			}
		}
	}
	if !reflect.DeepEqual(gotDomains, wantDomains) {
		t.Fatalf("material domain order = %v, want %v", gotDomains, wantDomains)
	}
}

func TestDiscussResumeExistingPendingDecisionUsesExactAnswerSyntax(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Resume the current scoped clarification"
	sessionID := "session-discuss-resume"
	initializedAt := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
	})
	decision := PendingDecision{
		ID:             "pd-existing-surface",
		Type:           clarificationDecisionType,
		Description:    formatClarificationDescription("Which existing surface owns the work?", []string{"current command", "new command"}),
		Source:         discussSource("surface", true),
		GoalHash:       pendingDecisionGoalHash(goal),
		SessionID:      sessionID,
		Grounding:      "The existing command and a new command would expose different public scope.",
		HardConstraint: true,
		CreatedAt:      initializedAt.Add(time.Minute).Format(time.RFC3339),
	}
	if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{decision}}); err != nil {
		t.Fatalf("seed pending decision: %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		result, err := runDiscuss(root, 1, false)
		if err != nil {
			t.Fatalf("run discuss attempt %d: %v", attempt, err)
		}
		batch := discussResultJSONMap(t, result["material_batch"])
		cards := batch["cards"].([]interface{})
		if len(cards) != 1 {
			t.Fatalf("attempt %d cards = %d, want 1", attempt, len(cards))
		}
		card := cards[0].(map[string]interface{})
		want := `aether discuss --resolve pd-existing-surface --answer "<answer>"`
		if got := stringValue(card["exact_answer_syntax"]); got != want {
			t.Fatalf("exact answer syntax = %q, want %q", got, want)
		}
	}
	if got := len(loadPendingDecisionFile().Decisions); got != 1 {
		t.Fatalf("exact replay duplicated pending answers: got %d decisions", got)
	}
}

func TestDiscussMaterialImpactDriftForcesRevalidationWithMatchingProse(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	goal := "Keep one answer bound to its exact contract impact"
	sessionID := "session-discuss-impact-drift"
	initializedAt := time.Date(2026, time.September, 7, 13, 0, 0, 0, time.UTC)
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
	})

	question := "Should the observable contract stay unchanged?"
	first, err := addComposedDiscussQuestion(question, "yes|no", "scope", "The current public surface defines the affected scope.", "contract-drift", false)
	if err != nil {
		t.Fatalf("add scoped question: %v", err)
	}
	if _, err := resolveDiscussQuestion(stringValue(first["id"]), "yes"); err != nil {
		t.Fatalf("resolve scoped question: %v", err)
	}

	drifted, err := addComposedDiscussQuestion(question, "yes|no", "acceptance_meaning", "The acceptance receipt changes what the same prose promises.", "contract-drift", false)
	if err != nil {
		t.Fatalf("add acceptance-drift question: %v", err)
	}
	if !boolValue(drifted["created"]) || !boolValue(drifted["revalidation_required"]) {
		t.Fatalf("impact drift reused prior authority instead of creating revalidation: %#v", drifted)
	}

	result, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("run drifted discuss: %v", err)
	}
	batch := discussResultJSONMap(t, result["material_batch"])
	cards := batch["cards"].([]interface{})
	if len(cards) != 1 {
		t.Fatalf("drifted batch cards = %d, want 1", len(cards))
	}
	card := cards[0].(map[string]interface{})
	if got := stringValue(card["domain"]); got != "acceptance_meaning" {
		t.Fatalf("drifted domain = %q, want acceptance_meaning", got)
	}
	if got := stringValue(card["prior_answer"]); got != "yes" {
		t.Fatalf("prior answer = %q, want evidence-only answer yes", got)
	}
	if got := stringValue(card["revalidation"]); !strings.Contains(got, "evidence only") {
		t.Fatalf("revalidation = %q, want changed-impact explanation", got)
	}
}

func discussResultJSONMap(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode discuss result: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("decode discuss result: %v", err)
	}
	return result
}
