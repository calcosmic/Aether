package colony

import "testing"

func TestContextRankingProtectedCandidatesPreserved(t *testing.T) {
	result := RankContextCandidates([]ContextCandidate{
		{
			Name:              "blockers",
			Title:             "Blockers",
			Content:           "critical blocker",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    1,
			ConfirmationScore: 1,
			RelevanceScore:    1,
			Protected:         true,
			PreserveReason:    "active blocker",
		},
		{
			Name:              "narrative",
			Title:             "Narrative",
			Content:           "background",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    0.4,
			ConfirmationScore: 0.2,
			RelevanceScore:    0.1,
		},
	}, 10)

	if len(result.Preserved) != 1 {
		t.Fatalf("preserved = %d, want 1", len(result.Preserved))
	}
	if result.Preserved[0].Name != "blockers" {
		t.Fatalf("preserved[0].name = %q, want blockers", result.Preserved[0].Name)
	}
	if len(result.Included) == 0 || result.Included[0].Name != "blockers" {
		t.Fatalf("included should start with protected blocker, got %+v", result.Included)
	}
}

func TestContextRankingFreshCandidateBeatsStaleCandidate(t *testing.T) {
	result := RankContextCandidates([]ContextCandidate{
		{
			Name:              "stale_hive",
			Title:             "Hive",
			Content:           "stale",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    0.05,
			ConfirmationScore: 0.55,
			RelevanceScore:    0.4,
		},
		{
			Name:              "fresh_decision",
			Title:             "Decision",
			Content:           "fresh",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    1.0,
			ConfirmationScore: 0.7,
			RelevanceScore:    0.6,
		},
	}, len("fresh"))

	if len(result.Included) != 1 {
		t.Fatalf("included = %d, want 1", len(result.Included))
	}
	if result.Included[0].Name != "fresh_decision" {
		t.Fatalf("included[0].name = %q, want fresh_decision", result.Included[0].Name)
	}
	if len(result.Trimmed) != 1 || result.Trimmed[0].Name != "stale_hive" {
		t.Fatalf("trimmed = %+v, want stale_hive trimmed", result.Trimmed)
	}
}

func TestContextScoreBreakdownUsesAllInputs(t *testing.T) {
	result := RankContextCandidates([]ContextCandidate{
		{
			Name:              "candidate",
			Title:             "Candidate",
			Content:           "content",
			TrustClass:        PromptTrustTrusted,
			FreshnessScore:    0.75,
			ConfirmationScore: 0.5,
			RelevanceScore:    0.8,
		},
	}, 100)

	if len(result.Included) != 1 {
		t.Fatalf("included = %d, want 1", len(result.Included))
	}
	breakdown := result.Included[0].Score
	if breakdown.Trust != 0.9 {
		t.Fatalf("trust = %f, want 0.9", breakdown.Trust)
	}
	if breakdown.Freshness != 0.75 {
		t.Fatalf("freshness = %f, want 0.75", breakdown.Freshness)
	}
	if breakdown.Confirmation != 0.5 {
		t.Fatalf("confirmation = %f, want 0.5", breakdown.Confirmation)
	}
	if breakdown.Relevance != 0.8 {
		t.Fatalf("relevance = %f, want 0.8", breakdown.Relevance)
	}
	if breakdown.Total <= 0 {
		t.Fatalf("total = %f, want > 0", breakdown.Total)
	}
}

func TestContextRankingTrimsHighValueCandidateToFitBudget(t *testing.T) {
	result := RankContextCandidates([]ContextCandidate{
		{
			Name:              "state",
			Title:             "State",
			Content:           "state",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    1,
			ConfirmationScore: 1,
			RelevanceScore:    1,
			Protected:         true,
			PreserveReason:    "authoritative runtime state",
		},
		{
			Name:              "decisions",
			Title:             "Decisions",
			Content:           "## Key Decisions\n\n- Fresh decision one\n- Fresh decision two\n- Fresh decision three\n",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    1,
			ConfirmationScore: 0.8,
			RelevanceScore:    0.7,
			BudgetMetric:      "chars",
		},
		{
			Name:              "hive_wisdom",
			Title:             "Hive Wisdom",
			Content:           "## HIVE WISDOM\n\n- stale\n",
			TrustClass:        PromptTrustAuthorized,
			FreshnessScore:    0.05,
			ConfirmationScore: 0.5,
			RelevanceScore:    0.25,
			BudgetMetric:      "chars",
		},
	}, len("state")+len("## Key Decisions\n\n- Fresh decision one\n"))

	if len(result.Included) != 2 {
		t.Fatalf("included = %d, want 2", len(result.Included))
	}
	if result.Included[1].Name != "decisions" {
		t.Fatalf("included[1].name = %q, want decisions", result.Included[1].Name)
	}
	if result.Included[1].Cost >= len("## Key Decisions\n\n- Fresh decision one\n- Fresh decision two\n- Fresh decision three\n") {
		t.Fatalf("expected decisions candidate to be trimmed to fit budget, got cost %d", result.Included[1].Cost)
	}
	if len(result.Trimmed) != 1 || result.Trimmed[0].Name != "hive_wisdom" {
		t.Fatalf("trimmed = %+v, want hive_wisdom trimmed", result.Trimmed)
	}
}
