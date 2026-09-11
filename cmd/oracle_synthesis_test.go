package cmd

import (
	"strings"
	"testing"
)

// oracleSynthesisFixture builds a plan/state pair with one well-supported
// answered question, one question still below target, and one recorded open
// gap -- enough to exercise every section of renderOracleFinalSynthesis.
func oracleSynthesisFixture() (oracleStateFile, oraclePlanFile) {
	plan := oraclePlanFile{
		Sources: map[string]oracleSource{
			"S1": {URL: "https://example.com/docs", Title: "Official docs", Type: "official", AccessedAt: "2026-09-01"},
		},
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "Should the cache use SQLite or Postgres?", Status: "answered", Confidence: 92,
				KeyFindings: []oracleFinding{
					{Text: "SQLite handles the observed write volume.", SourceIDs: []string{"S1"}, Iteration: 2},
				},
				IterationsTouched: []int{1, 2},
			},
			{
				ID: "q2", Text: "What is the migration cost later?", Status: "open", Confidence: 30,
				KeyFindings:       []oracleFinding{},
				IterationsTouched: []int{3},
			},
		},
	}
	state := oracleStateFile{
		Topic:             "cache storage",
		CoreQuestion:      "Should the cache use SQLite or Postgres?",
		Iteration:         3,
		TargetConfidence:  90,
		OverallConfidence: 62,
		Status:            "active",
		Recommendation:    "Use SQLite for the cache.",
		OpenGaps:          []string{"need production write-volume numbers to confirm at scale"},
	}
	return state, plan
}

func TestSynthesisLeadsWithTheRecommendation(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	trimmed := strings.TrimSpace(doc)
	if !strings.HasPrefix(trimmed, "## Recommendation") {
		t.Fatalf("document does not open with the recommendation section:\n%s", truncateString(trimmed, 300))
	}
	idxRec := strings.Index(doc, "## Recommendation")
	idxConf := strings.Index(doc, "## Confidence")
	idxGaps := strings.Index(doc, "## What Is Still Unsettled")
	idxSources := strings.Index(doc, "## Sources")
	idxTrail := strings.Index(doc, "## Evidence Trail")
	if !(idxRec >= 0 && idxRec < idxConf && idxConf < idxGaps && idxGaps < idxSources && idxSources < idxTrail) {
		t.Fatalf("section order wrong: rec=%d conf=%d gaps=%d sources=%d trail=%d\n%s", idxRec, idxConf, idxGaps, idxSources, idxTrail, doc)
	}
	if !strings.Contains(doc, "Use SQLite for the cache.") {
		t.Errorf("recommendation section missing the run's own recommendation text:\n%s", doc)
	}
}

func TestSynthesisStatesConfidenceInOrdinaryWords(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	confSection := doc[strings.Index(doc, "## Confidence"):strings.Index(doc, "## What Is Still Unsettled")]
	if !strings.Contains(confSection, "62% / 90% target") {
		t.Fatalf("confidence section did not carry the recorded integer unchanged:\n%s", confSection)
	}
	if !strings.Contains(confSection, "moderate confidence") {
		t.Errorf("confidence section missing a plain-English sentence about what the figure means:\n%s", confSection)
	}
}

func TestSynthesisListsUnsettledQuestionsWithTheirEvidence(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	gapsSection := doc[strings.Index(doc, "## What Is Still Unsettled"):strings.Index(doc, "## Sources")]
	if !strings.Contains(gapsSection, "need production write-volume numbers to confirm at scale") {
		t.Fatalf("unsettled section dropped the recorded open gap:\n%s", gapsSection)
	}
	if !strings.Contains(gapsSection, "would be settled by") {
		t.Errorf("unsettled section did not name the evidence that would settle each question:\n%s", gapsSection)
	}
	if !strings.Contains(gapsSection, "What is the migration cost later?") {
		t.Errorf("unsettled section dropped the low-confidence question:\n%s", gapsSection)
	}
}

func TestSynthesisCarriesTheStandingLabelInTheFirstSection(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	state.Status = "stopped"
	state.StopReason = "manual_stop"
	state.Iteration = 3
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	recSection := doc[:strings.Index(doc, "## Confidence")]
	if !strings.Contains(recSection, "partial — stopped after 3 rounds") {
		t.Fatalf("early-stopped run's standing label is not inside the first section:\n%s", recSection)
	}
}

func TestSynthesisWithoutARecommendationSaysSo(t *testing.T) {
	state := oracleStateFile{
		Topic:             "unexplored topic",
		CoreQuestion:      "Is this worth building?",
		Iteration:         1,
		TargetConfidence:  90,
		OverallConfidence: 0,
		Status:            "active",
	}
	plan := oraclePlanFile{
		Sources:   map[string]oracleSource{},
		Questions: []oracleQuestion{{ID: "q1", Text: "Is this worth building?", Status: "open", Confidence: 0, KeyFindings: []oracleFinding{}}},
	}
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	trimmed := strings.TrimSpace(doc)
	if !strings.HasPrefix(trimmed, "## Recommendation") {
		t.Fatalf("document with no findings does not open with the recommendation section:\n%s", truncateString(trimmed, 300))
	}
	recSection := doc[:strings.Index(doc, "## Confidence")]
	if !strings.Contains(recSection, "No recommendation yet") {
		t.Fatalf("a run with no findings did not say plainly that it has no recommendation:\n%s", recSection)
	}
	if strings.Contains(recSection, "## Sources") {
		t.Errorf("a run with no recommendation began with a source list instead of saying so:\n%s", recSection)
	}
}
