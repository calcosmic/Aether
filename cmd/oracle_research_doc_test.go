package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func seedFinishedOracleRun(t *testing.T, root, topic, coreQuestion, synthesis string) (oraclePaths, oracleStateFile, oraclePlanFile) {
	t.Helper()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             topic,
		CoreQuestion:      coreQuestion,
		SuccessCriteria:   []string{"A recommendation with reasons", "Migration cost if we switch later"},
		Scope:             "both",
		Template:          "tech-eval",
		Phase:             "synthesize",
		Iteration:         18,
		MaxIterations:     30,
		OverallConfidence: 92,
		TargetConfidence:  90,
		Status:            "complete",
		StopReason:        "target_reached",
		StartedAt:         now,
		LastUpdated:       now,
	}
	plan := oraclePlanFile{
		Version: "1.1",
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: coreQuestion, Status: "answered", Confidence: 92, KeyFindings: []oracleFinding{{Text: "SQLite handles the write volume."}}, IterationsTouched: []int{1}},
			{ID: "q2", Text: "What is the migration cost?", Status: "open", Confidence: 40, KeyFindings: []oracleFinding{}, IterationsTouched: []int{2}},
		},
		CreatedAt:   now,
		LastUpdated: now,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if err := os.WriteFile(paths.SynthesisPath, []byte(synthesis), 0644); err != nil {
		t.Fatalf("write synthesis: %v", err)
	}
	return paths, state, plan
}

func TestOracleResearchDocumentRecordsWhatWasAsked(t *testing.T) {
	root := t.TempDir()
	paths, state, plan := seedFinishedOracleRun(t, root,
		"cache storage: sqlite vs postgres",
		"Should the local cache use SQLite or Postgres?",
		"# Tech Evaluation\n\nSQLite is sufficient for the observed write volume.\n")

	saved, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save research document: %v", err)
	}
	if !strings.HasPrefix(saved, filepath.Join(".aether", "research")) {
		t.Fatalf("saved to %q, want a path under .aether/research", saved)
	}
	if !strings.HasSuffix(saved, ".md") {
		t.Errorf("saved document %q is not markdown", saved)
	}

	data, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("read saved document: %v", err)
	}
	text := string(data)

	// Front matter is what makes a saved document self-describing months later.
	for _, want := range []string{
		`core_question: "Should the local cache use SQLite or Postgres?"`,
		"confidence: 92",
		"target_confidence: 90",
		"iterations: 18",
		"status: complete",
		"questions_answered: 1",
		"questions_total: 2",
		"- \"A recommendation with reasons\"",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("saved document missing front matter %q:\n%s", want, truncateString(text, 600))
		}
	}
	if !strings.Contains(text, "SQLite is sufficient for the observed write volume.") {
		t.Error("saved document dropped the research body")
	}

	entry := parseOracleResearchFrontMatter(text)
	if entry.Confidence != 92 || entry.Status != "complete" {
		t.Errorf("front matter did not round-trip: %+v", entry)
	}
}

// TestOracleResearchDocumentSurvivesNextOracleRun is the whole point of Stage 2.
// archiveOracleWorkspace sweeps the workspace before every new run, so research
// written only to synthesis.md was destroyed by the next question asked.
func TestOracleResearchDocumentSurvivesNextOracleRun(t *testing.T) {
	root := t.TempDir()
	pathsA, stateA, planA := seedFinishedOracleRun(t, root,
		"first topic", "Should the cache use SQLite or Postgres?",
		"# Run A\n\nThe first run's conclusions.\n")

	savedA, err := saveOracleResearchDocument(pathsA, stateA, planA, "")
	if err != nil {
		t.Fatalf("save run A: %v", err)
	}
	beforeA, err := os.ReadFile(filepath.Join(root, savedA))
	if err != nil {
		t.Fatalf("read run A document: %v", err)
	}

	// A second run archives the workspace and replaces synthesis.md.
	if err := archiveOracleWorkspace(pathsA); err != nil {
		t.Fatalf("archive workspace: %v", err)
	}
	seedFinishedOracleRun(t, root, "second topic", "Which queue should we use?", "# Run B\n\nA completely different question.\n")

	live, err := os.ReadFile(pathsA.SynthesisPath)
	if err != nil {
		t.Fatalf("read live synthesis: %v", err)
	}
	if !strings.Contains(string(live), "Run B") {
		t.Fatal("test setup wrong: the workspace should now hold run B")
	}

	afterA, err := os.ReadFile(filepath.Join(root, savedA))
	if err != nil {
		t.Fatalf("run A's saved research was destroyed by run B: %v", err)
	}
	if !bytes.Equal(beforeA, afterA) {
		t.Fatal("run A's saved research changed when run B started")
	}
}

func TestOracleSaveDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	seedFinishedOracleRun(t, root, "cache storage", "Should the cache use SQLite or Postgres?", "# Findings\n\nBody.\n")

	result, err := runOracleSave(root, "", true)
	if err != nil {
		t.Fatalf("save dry run: %v", err)
	}
	if destination, _ := result["destination"].(string); !strings.Contains(destination, ".aether/research") {
		t.Errorf("dry run did not name its destination: %v", result["destination"])
	}
	if _, err := os.Stat(oracleResearchDir(root)); !os.IsNotExist(err) {
		t.Fatal("oracle save --dry-run created the research directory")
	}
}

func TestOracleSaveRefusesWithoutResearch(t *testing.T) {
	root := t.TempDir()
	if _, err := runOracleSave(root, "", false); err == nil {
		t.Fatal("save succeeded with no research run")
	}

	// A run that exists but produced no write-up is equally unsavable.
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	if err := writeOracleStateFile(paths.StatePath, oracleStateFile{Version: "1.1", Topic: "x"}); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOraclePlanFile(paths.PlanPath, oraclePlanFile{Version: "1.1", Sources: map[string]oracleSource{}}); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if err := os.WriteFile(paths.SynthesisPath, []byte("   \n"), 0644); err != nil {
		t.Fatalf("write empty synthesis: %v", err)
	}
	if _, err := runOracleSave(root, "", false); err == nil {
		t.Fatal("save accepted an empty write-up")
	}
}

// TestOracleSaveKeepsBothRunsOnTheSameTopicAndDay proves two genuinely
// different runs on the same topic on the same day stay as two documents.
// 202-12 made resaving keyed on run identifier (state.StartedAt via
// oracleLiveEpisodeID) rather than on topic+day alone, so this fixture now
// gives the second save its own StartedAt -- the same distinguishing signal
// a second, later `aether oracle` invocation would carry for real. Resaving
// the *same* run (identical StartedAt) is covered separately by
// TestResaveOfOneRunDoesNotDuplicate, which proves the opposite case.
func TestOracleSaveKeepsBothRunsOnTheSameTopicAndDay(t *testing.T) {
	root := t.TempDir()
	paths, state, plan := seedFinishedOracleRun(t, root, "cache storage", "Should the cache use SQLite or Postgres?", "# First\n\nOne.\n")
	first, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := os.WriteFile(paths.SynthesisPath, []byte("# Second\n\nTwo.\n"), 0644); err != nil {
		t.Fatalf("rewrite synthesis: %v", err)
	}
	state.StartedAt = "2024-01-02T00:00:00Z" // a second, distinct run
	second, err := saveOracleResearchDocument(paths, state, plan, "")
	if err != nil {
		t.Fatalf("save second: %v", err)
	}
	if first == second {
		t.Fatalf("a second run on the same topic overwrote the first at %s", first)
	}
}

func TestResearchListReportsSavedDocuments(t *testing.T) {
	root := t.TempDir()

	pathsA, stateA, planA := seedFinishedOracleRun(t, root, "cache storage", "Should the cache use SQLite or Postgres?", "# A\n\nOne.\n")
	if _, err := saveOracleResearchDocument(pathsA, stateA, planA, "cache-storage"); err != nil {
		t.Fatalf("save A: %v", err)
	}

	stateB := stateA
	stateB.CoreQuestion = "Which queue should we use?"
	stateB.OverallConfidence = 41
	stateB.Status = "stopped"
	stateB.Iteration = 5
	// A genuinely different run (202-12 keys resaving on run identifier, not
	// topic+day) -- give it its own StartedAt so it does not replace A.
	stateB.StartedAt = "2024-01-02T00:00:00Z"
	if err := os.WriteFile(pathsA.SynthesisPath, []byte("# B\n\nTwo.\n"), 0644); err != nil {
		t.Fatalf("rewrite synthesis: %v", err)
	}
	if _, err := saveOracleResearchDocument(pathsA, stateB, planA, "queue-choice"); err != nil {
		t.Fatalf("save B: %v", err)
	}

	result, err := runResearchList(root)
	if err != nil {
		t.Fatalf("research list: %v", err)
	}
	if count, _ := result["count"].(int); count != 2 {
		t.Fatalf("research list found %v documents, want 2", result["count"])
	}

	entries, _ := result["documents"].([]oracleResearchEntry)
	byQuestion := map[string]oracleResearchEntry{}
	for _, entry := range entries {
		byQuestion[entry.CoreQuestion] = entry
	}
	// The listing must distinguish a finished document from a thin one, or
	// pointing a colony at the right file is guesswork.
	if got := byQuestion["Should the cache use SQLite or Postgres?"]; got.Confidence != 92 || got.Status != "complete" {
		t.Errorf("finished document listed as %+v", got)
	}
	if got := byQuestion["Which queue should we use?"]; got.Confidence != 41 || got.Status != "stopped" {
		t.Errorf("stopped document listed as %+v", got)
	}

	rendered := renderResearchList(result)
	for _, want := range []string{"92% confidence", "41% confidence", "aether init --research"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("listing output missing %q:\n%s", want, rendered)
		}
	}
}

// TestSessionClearNeverTargetsResearchDirectory: saved research is the
// operator's deliverable, not session scratch.
func TestSessionClearNeverTargetsResearchDirectory(t *testing.T) {
	for name, entry := range commandFileMap() {
		normalized := filepath.ToSlash(strings.TrimSpace(entry.dir))
		if normalized == ".aether/research" || strings.HasPrefix(normalized, ".aether/research/") {
			t.Errorf("session-clear entry %q targets %s; saved research must never be cleared", name, entry.dir)
		}
	}
}

// TestSessionClearRemovesOracleProgressLog: the round log is workspace scratch
// and must not survive a clear as a half-run the next `--follow` replays.
func TestSessionClearRemovesOracleProgressLog(t *testing.T) {
	entry, ok := commandFileMap()["oracle"]
	if !ok {
		t.Fatal("no oracle entry in the session-clear map")
	}
	found := false
	for _, file := range entry.files {
		if file == "progress.jsonl" {
			found = true
		}
	}
	if !found {
		t.Errorf("session-clear leaves progress.jsonl behind: %v", entry.files)
	}
}
