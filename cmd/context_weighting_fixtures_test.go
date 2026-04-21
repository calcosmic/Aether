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

type competitiveProofFixtureSet struct {
	StackCases       []competitiveProofStackCase `json:"stack_cases"`
	SafeImport       competitiveProofImportCase  `json:"safe_import"`
	SuspiciousImport competitiveProofImportCase  `json:"suspicious_import"`
	WeightingFiles   []string                    `json:"weighting_files"`
}

type competitiveProofStackCase struct {
	Name             string   `json:"name"`
	Workspace        string   `json:"workspace"`
	Task             string   `json:"task"`
	ExpectedSkill    string   `json:"expected_skill"`
	ForbiddenSkill   string   `json:"forbidden_skill,omitempty"`
	ExpectedEvidence []string `json:"expected_evidence,omitempty"`
}

type competitiveProofImportCase struct {
	Name               string `json:"name"`
	XML                string `json:"xml"`
	ExpectedAction     string `json:"expected_action"`
	ExpectedTrustClass string `json:"expected_trust_class"`
	ExpectedSignalID   string `json:"expected_signal_id,omitempty"`
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

func competitiveProofFixturePath(parts ...string) string {
	return filepath.Join(append([]string{"testdata", "competitive-proof-fixtures"}, parts...)...)
}

func loadCompetitiveProofFixtures(t *testing.T) competitiveProofFixtureSet {
	t.Helper()
	var fixtures competitiveProofFixtureSet
	path := competitiveProofFixturePath("fixtures.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	return fixtures
}

func TestContextWeightingCompetitiveProofFixtureManifest(t *testing.T) {
	fixtures := loadCompetitiveProofFixtures(t)

	if len(fixtures.StackCases) < 2 {
		t.Fatalf("expected representative stack cases, got %+v", fixtures.StackCases)
	}
	if fixtures.SafeImport.XML == "" || fixtures.SuspiciousImport.XML == "" {
		t.Fatalf("expected import fixtures in competitive harness, got %+v", fixtures)
	}

	wantWeighting := map[string]bool{
		"fresh-vs-stale.json":    false,
		"protected-capsule.json": false,
	}
	for _, name := range fixtures.WeightingFiles {
		if _, ok := wantWeighting[name]; ok {
			wantWeighting[name] = true
		}
	}
	for name, present := range wantWeighting {
		if !present {
			t.Fatalf("competitive proof fixture manifest missing weighting file %q: %+v", name, fixtures.WeightingFiles)
		}
	}
}
