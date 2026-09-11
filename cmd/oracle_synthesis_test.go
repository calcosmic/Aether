package cmd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

func TestSynthesisSourcesCarryRoundAndKind(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	doc, err := renderOracleFinalSynthesis(state, plan, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	sourcesSection := doc[strings.Index(doc, "## Sources"):strings.Index(doc, "## Evidence Trail")]
	for _, want := range []string{"Official docs", "https://example.com/docs", "official", "round 2"} {
		if !strings.Contains(sourcesSection, want) {
			t.Errorf("sources section missing %q:\n%s", want, sourcesSection)
		}
	}
}

func TestUnsupportedConclusionIsRefusedByName(t *testing.T) {
	state, plan := oracleSynthesisFixture()
	// Corrupt one finding to cite a source ID the plan never recorded.
	q := plan.Questions[0]
	q.KeyFindings = append([]oracleFinding{{Text: "A dangling claim with no real source.", SourceIDs: []string{"S9"}}}, q.KeyFindings...)
	plan.Questions[0] = q

	_, err := renderOracleFinalSynthesis(state, plan, "")
	if err == nil {
		t.Fatal("render accepted a conclusion citing a source the plan never recorded")
	}
	if !strings.Contains(err.Error(), "A dangling claim with no real source.") {
		t.Errorf("refusal did not name the offending conclusion: %v", err)
	}
}

// directoryDigest hashes every file under root by relative path, size and
// content, so a test can prove a directory is byte-identical before and
// after an operation rather than merely "still exists."
func directoryDigest(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		paths = append(paths, rel)
		return nil
	})
	sort.Strings(paths)
	h := sha256.New()
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("digest read %s: %v", rel, err)
		}
		fmt.Fprintf(h, "%s:%d:", rel, len(data))
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestEmptyResearchRunWritesNoDocument(t *testing.T) {
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	// synthesis.md is deliberately never written -- this run gathered
	// nothing.
	state := oracleStateFile{Topic: "an unexplored idea", Status: "stopped", Iteration: 1}
	plan := oraclePlanFile{Sources: map[string]oracleSource{}}

	before := directoryDigest(t, root)
	saved := finalizeOracleResearchArtifacts(paths, state, plan)
	after := directoryDigest(t, root)

	if saved != "" {
		t.Fatalf("an empty run reported a saved document: %q", saved)
	}
	if before != after {
		t.Fatal("an empty run changed the workspace directory")
	}
	if _, err := os.Stat(oracleResearchDir(root)); !os.IsNotExist(err) {
		t.Fatal("an empty run created the research directory")
	}
}

func TestEmptyResearchRunIsReportedAsEmpty(t *testing.T) {
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	state := oracleStateFile{Topic: "cache eviction policy", Status: "stopped", Iteration: 2}
	plan := oraclePlanFile{Sources: map[string]oracleSource{}}

	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	oldStdout := stdout
	var buf bytes.Buffer
	stdout = &buf
	t.Cleanup(func() { stdout = oldStdout })

	saved := finalizeOracleResearchArtifacts(paths, state, plan)
	if saved != "" {
		t.Fatalf("an empty run reported a saved document: %q", saved)
	}
	out := buf.String()
	if !strings.Contains(out, "cache eviction policy") {
		t.Errorf("empty-run message did not name the topic:\n%s", out)
	}
	if !strings.Contains(out, "no evidence") {
		t.Errorf("empty-run message did not say why nothing was written:\n%s", out)
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

// --- Task 3: discoverable after the run ends ---

func TestSynthesisIsSavedOutOfTheWorkspace(t *testing.T) {
	root := t.TempDir()
	paths, state, plan := seedFinishedOracleRun(t, root,
		"cache storage: sqlite vs postgres",
		"Should the local cache use SQLite or Postgres?",
		"# Tech Evaluation\n\nSQLite is sufficient for the observed write volume.\n")

	saved, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("read saved document: %v", err)
	}

	// A workspace reset (what the next run does before starting) must not
	// touch the durable copy.
	if err := archiveOracleWorkspace(paths); err != nil {
		t.Fatalf("archive workspace: %v", err)
	}

	after, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("saved document did not survive the workspace reset: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("saved document changed when the workspace was reset")
	}
}

func TestSavedSynthesisFrontMatterCarriesStandingAndConfidence(t *testing.T) {
	root := t.TempDir()
	paths, state, plan := seedFinishedOracleRun(t, root,
		"cache storage", "Should the cache use SQLite or Postgres?",
		"# Findings\n\nBody.\n")
	state.Status = "stopped"
	state.Iteration = 6

	saved, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("read saved document: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"round_count: 6",
		`standing: "partial — stopped after 6 rounds"`,
		"confidence: 92",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("saved front matter missing %q:\n%s", want, truncateString(text, 800))
		}
	}
	entry := parseOracleResearchFrontMatter(text)
	if strings.TrimSpace(entry.RunIdentifier) == "" {
		t.Error("saved front matter did not carry a run identifier")
	}
	if entry.RoundCount != 6 {
		t.Errorf("round_count did not round-trip: got %d", entry.RoundCount)
	}
}

func TestResaveOfOneRunDoesNotDuplicate(t *testing.T) {
	root := t.TempDir()
	paths, state, plan := seedFinishedOracleRun(t, root,
		"cache storage", "Should the cache use SQLite or Postgres?",
		"# First save\n\nOne.\n")

	first, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := os.WriteFile(paths.SynthesisPath, []byte("# Second save\n\nTwo, more evidence gathered.\n"), 0644); err != nil {
		t.Fatalf("rewrite synthesis: %v", err)
	}
	// Same state (same StartedAt/run identifier) -- this is a resave of the
	// same run, not a new one.
	second, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save second: %v", err)
	}
	if first != second {
		t.Fatalf("resaving the same run created a second document: %q vs %q", first, second)
	}

	matches, err := filepath.Glob(filepath.Join(oracleResearchDir(root), "*.md"))
	if err != nil {
		t.Fatalf("glob research dir: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("research directory has %d documents for one run, want 1: %v", len(matches), matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read replaced document: %v", err)
	}
	if !strings.Contains(string(data), "Two, more evidence gathered.") {
		t.Error("the resaved document does not carry the newer content")
	}
}

func TestResearchListStillParsesOlderDocuments(t *testing.T) {
	root := t.TempDir()
	dir := oracleResearchDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir research dir: %v", err)
	}
	// The exact shape renderOracleResearchDocument wrote before 202-12 --
	// no standing, round_count, or run_identifier keys.
	oldDoc := `---
title: "Should we use Redis for sessions?"
core_question: "Should we use Redis for sessions?"
topic: "session storage"
generated: 2026-01-01T00:00:00Z
template: custom
scope: both
iterations: 12
confidence: 77
target_confidence: 90
status: stopped
questions_answered: 1
questions_total: 2
---

Old-shape body.
`
	if err := os.WriteFile(filepath.Join(dir, "2026-01-01-old-doc.md"), []byte(oldDoc), 0644); err != nil {
		t.Fatalf("write old-shape fixture: %v", err)
	}

	result, err := runResearchList(root)
	if err != nil {
		t.Fatalf("research list: %v", err)
	}
	entries, _ := result["documents"].([]oracleResearchEntry)
	if len(entries) != 1 {
		t.Fatalf("expected 1 document, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Confidence != 77 || entry.Status != "stopped" || entry.Iterations != 12 {
		t.Fatalf("old-shape document did not parse correctly: %+v", entry)
	}
	if entry.Standing != "" {
		t.Errorf("old-shape document unexpectedly carries a standing value: %q", entry.Standing)
	}

	rendered := renderResearchList(result)
	if !strings.Contains(rendered, "77% confidence after 12 rounds — stopped") {
		t.Errorf("listing output for an old-shape document changed shape:\n%s", rendered)
	}
}
