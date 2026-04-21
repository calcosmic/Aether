package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type freshVsStaleFixture struct {
	Goal                   string `json:"goal"`
	StaleHiveCount         int    `json:"stale_hive_count"`
	StaleHivePhrase        string `json:"stale_hive_phrase"`
	FreshDecisionCount     int    `json:"fresh_decision_count"`
	FreshDecisionPhrase    string `json:"fresh_decision_phrase"`
	FreshDecisionRationale string `json:"fresh_decision_rationale"`
	LearningCount          int    `json:"learning_count"`
	LearningPhrase         string `json:"learning_phrase"`
	CodexGoal              string `json:"codex_goal"`
	CodexDecisionPhrase    string `json:"codex_decision_phrase"`
	CodexDecisionPrefix    string `json:"codex_decision_prefix"`
	CodexLearningPhrase    string `json:"codex_learning_phrase"`
}

type protectedCapsuleFixture struct {
	Goal           string   `json:"goal"`
	RiskText       string   `json:"risk_text"`
	DecisionClaims []string `json:"decision_claims"`
}

func loadContextWeightingFixture(t *testing.T, name string, target interface{}) {
	t.Helper()
	path := filepath.Join("testdata", "context-weighting-fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
}
