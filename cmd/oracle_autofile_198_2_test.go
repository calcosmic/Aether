package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198.2, Plan 02 (D-05, D-06, D-07, D-08): a finished or interrupted
// deep-research run reaches the next helper by itself -- filed, registered,
// and (when strong enough) promoted into a labelled habit -- with no
// hand-typed `aether oracle save`, `--research <path>`, or `aether oracle
// promote`.

// setupOracleAutofileTest wires a fresh store the same way
// promoteOracleFindingAsInstinct / promoteOracleFindingAsLearning and
// registerColonyResearchDoc read it at runtime: a real *storage.Store bound
// to <root>/.aether/data, with globals restored after the test, and a
// minimal COLONY_STATE.json already on disk (registerColonyResearchDoc
// requires one to exist -- an un-initialized colony is a distinct,
// non-fatal failure mode, not what these tests are about).
func setupOracleAutofileTest(t *testing.T) string {
	t.Helper()
	saveGlobalsCmd(t)
	s, root := newTestStore(t)
	store = s
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "3.0"}); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}
	return root
}

// --- Task 1: a finished run registers its own write-up on the colony ---
//
// 202-12 note: saveOracleResearchDocument now keys resaving a document on
// the run identifier derived from state.StartedAt (oracleLiveEpisodeID), so
// tests below that model two genuinely different runs on the same topic
// give each call its own StartedAt -- the same distinguishing signal two
// separate `aether oracle` invocations would carry for real.

func TestFinishedResearchRegistersItselfForTheNextHelper(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "Should the routing test use a cache?", Status: "answered", Confidence: 70,
				KeyFindings: []oracleFinding{
					{Text: "Distinctive registered finding sentence for the routing test."},
				},
			},
		},
	}
	state := oracleStateFile{
		Topic:            "colony routing test",
		CoreQuestion:     "Should the routing test use a cache?",
		Iteration:        5,
		TargetConfidence: 80,
	}

	result, err := finalizeOracleLoop(paths, state, plan, "go", []string{"go"}, nil, 5, "complete", "", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop: %v", err)
	}
	saved := stringValue(result["research_document"])
	if saved == "" {
		t.Fatal("a completed run did not report a saved research document")
	}

	docs := loadColonyResearchDocs(root)
	if len(docs) != 1 || docs[0] != saved {
		t.Fatalf("loadColonyResearchDocs = %v, want exactly [%s]", docs, saved)
	}

	section := resolveColonyResearchSection(root, docs)
	if !strings.Contains(section, "Distinctive registered finding sentence") {
		t.Fatalf("resolveColonyResearchSection did not carry a sentence from the write-up:\n%s", section)
	}
}

func TestResearchOnTheSameTopicKeepsBothWriteUps(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	state := oracleStateFile{Topic: "repeat topic", CoreQuestion: "Repeat topic question?", TargetConfidence: 80, StartedAt: "2024-01-01T00:00:00Z"}

	plan1 := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: "Q", Status: "answered", Confidence: 60, KeyFindings: []oracleFinding{{Text: "First run distinctive finding sentence."}}},
		},
	}
	result1, err := finalizeOracleLoop(paths, state, plan1, "go", nil, nil, 3, "complete", "", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop (first run): %v", err)
	}
	saved1 := stringValue(result1["research_document"])
	if saved1 == "" {
		t.Fatal("first run did not save a research document")
	}

	plan2 := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: "Q", Status: "answered", Confidence: 60, KeyFindings: []oracleFinding{{Text: "Second run distinctive finding sentence."}}},
		},
	}
	state.StartedAt = "2024-01-02T00:00:00Z" // a second, distinct run on the same topic
	result2, err := finalizeOracleLoop(paths, state, plan2, "go", nil, nil, 4, "complete", "", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop (second run): %v", err)
	}
	saved2 := stringValue(result2["research_document"])
	if saved2 == "" || saved2 == saved1 {
		t.Fatalf("second run on the same topic did not create a distinct document: %q vs %q", saved2, saved1)
	}

	if _, err := os.Stat(filepath.Join(root, saved1)); err != nil {
		t.Fatalf("first write-up was destroyed by the second run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, saved2)); err != nil {
		t.Fatalf("second write-up is missing: %v", err)
	}

	docs := loadColonyResearchDocs(root)
	if len(docs) != 2 {
		t.Fatalf("pointer list has %d entries, want 2 (both write-ups kept): %v", len(docs), docs)
	}
	if docs[0] != saved2 {
		t.Fatalf("newest write-up is not listed first: %v", docs)
	}
	if docs[1] != saved1 {
		t.Fatalf("first write-up is missing from the pointer list: %v", docs)
	}
}

func TestEmptyResearchRegistersNothing(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	// synthesis.md is deliberately never written -- this is the shape of a
	// run that produced no write-up body at all.
	state := oracleStateFile{Topic: "nothing gathered topic", Status: "stopped", Iteration: 1}
	plan := oraclePlanFile{Sources: map[string]oracleSource{}}

	saved := finalizeOracleResearchArtifacts(paths, state, plan)
	if saved != "" {
		t.Fatalf("an empty write-up was reported as saved: %q", saved)
	}
	if docs := loadColonyResearchDocs(root); len(docs) != 0 {
		t.Fatalf("an empty write-up registered a pointer: %v", docs)
	}
	if _, err := os.Stat(oracleResearchDir(root)); !os.IsNotExist(err) {
		t.Fatal("an empty write-up created the research directory")
	}
}

// --- Task 2: an interrupted run still files what it gathered, marked partial ---

func TestStoppedResearchIsFiledAndLabelledPartial(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: "Partial question", Status: "partial", Confidence: 55, KeyFindings: []oracleFinding{
				{Text: "Partial run distinctive finding sentence for the label test."},
			}},
		},
	}
	state := oracleStateFile{Topic: "partial run topic", CoreQuestion: "Partial question", Iteration: 7, TargetConfidence: 90}

	result, err := finalizeOracleLoop(paths, state, plan, "go", nil, nil, 7, "stopped", "manual_stop", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop: %v", err)
	}
	saved := stringValue(result["research_document"])
	if saved == "" {
		t.Fatal("a stopped run with gathered content did not file a document")
	}

	data, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("read saved document: %v", err)
	}
	// This must be true of what a worker actually reads -- the body after
	// front matter is stripped away, not the raw file.
	stripped := strings.TrimSpace(stripResearchFrontMatter(string(data)))
	firstLine := strings.SplitN(stripped, "\n", 2)[0]
	if firstLine != "partial — stopped after 7 rounds" {
		t.Fatalf("first line of the document body = %q, want the partial label", firstLine)
	}

	docs := loadColonyResearchDocs(root)
	found := false
	for _, d := range docs {
		if d == saved {
			found = true
		}
	}
	if !found {
		t.Fatalf("the stopped run's document was not registered on the colony: %v", docs)
	}
}

func TestStoppedResearchWithNothingGatheredFilesNothing(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	// Only state exists -- no plan.json at all. This is the earliest an
	// owner stop can happen: before the run ever produced a plan, let alone
	// a synthesis.
	if err := writeOracleStateFile(paths.StatePath, oracleStateFile{Version: "1.1", Topic: "stopped before anything gathered"}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result, err := stopOracleCompatibility(root)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if doc, ok := result["research_document"]; ok {
		t.Fatalf("nothing was gathered but a research document was reported: %v", doc)
	}
	if docs := loadColonyResearchDocs(root); len(docs) != 0 {
		t.Fatalf("nothing was gathered but the colony was pointed at research: %v", docs)
	}
	if _, err := os.Stat(oracleResearchDir(root)); !os.IsNotExist(err) {
		t.Fatal("nothing was gathered but a research document was written to disk")
	}
}

func TestCompletedResearchCarriesNoPartialLabel(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{ID: "q1", Text: "Complete question", Status: "answered", Confidence: 90, KeyFindings: []oracleFinding{
				{Text: "Completed run distinctive finding sentence for the label test."},
			}},
		},
	}
	state := oracleStateFile{Topic: "complete run topic", CoreQuestion: "Complete question", Iteration: 9, TargetConfidence: 80}

	result, err := finalizeOracleLoop(paths, state, plan, "go", nil, nil, 9, "complete", "", "aether oracle status")
	if err != nil {
		t.Fatalf("finalizeOracleLoop: %v", err)
	}
	saved := stringValue(result["research_document"])
	if saved == "" {
		t.Fatal("a completed run did not file a document")
	}

	data, err := os.ReadFile(filepath.Join(root, saved))
	if err != nil {
		t.Fatalf("read saved document: %v", err)
	}
	stripped := strings.TrimSpace(stripResearchFrontMatter(string(data)))
	firstLine := strings.SplitN(stripped, "\n", 2)[0]
	if strings.HasPrefix(firstLine, "partial") {
		t.Fatalf("a completed run's document carries a partial label: %q", firstLine)
	}
}

// --- Task 3: strong findings become labelled learned habits ---

func TestStrongResearchFindingBecomesALabelledHabit(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "How should the exporter retry?", Status: "answered", Confidence: 90,
				KeyFindings: []oracleFinding{
					{Text: "The retry helper in cmd/retry_198.go already handles exponential backoff; reuse it instead of writing a new loop."},
				},
			},
		},
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("seed plan.json: %v", err)
	}
	state := oracleStateFile{
		Topic:            "exporter retry strategy",
		CoreQuestion:     "How should the exporter retry?",
		Iteration:        6,
		TargetConfidence: 80,
	}

	if _, err := finalizeOracleLoop(paths, state, plan, "go", nil, nil, 6, "complete", "", "aether oracle status"); err != nil {
		t.Fatalf("finalizeOracleLoop: %v", err)
	}

	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	if len(instFile.Instincts) != 1 {
		t.Fatalf("instincts on disk = %d, want 1: %+v", len(instFile.Instincts), instFile.Instincts)
	}
	inst := instFile.Instincts[0]

	label := inst.Provenance.OriginLabel
	wantDate := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(label, "How should the exporter retry?") {
		t.Errorf("origin label %q is missing the run's topic", label)
	}
	if !strings.Contains(label, wantDate) {
		t.Errorf("origin label %q is missing today's date (%s)", label, wantDate)
	}

	wantFinding := "The retry helper in cmd/retry_198.go already handles exponential backoff; reuse it instead of writing a new loop."
	if inst.Trigger != wantFinding {
		t.Errorf("the finding text was changed by adding the label: got %q, want %q", inst.Trigger, wantFinding)
	}
	if strings.Contains(inst.Trigger, "from research:") {
		t.Errorf("the origin label leaked into the finding text: %q", inst.Trigger)
	}
}

func TestWeakResearchFindingBecomesNothing(t *testing.T) {
	root := setupOracleAutofileTest(t)
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "Low-confidence question", Status: "partial", Confidence: 40,
				KeyFindings: []oracleFinding{
					{Text: "cmd/never_promoted_198.go should never be promoted from a 40 percent question."},
				},
			},
		},
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("seed plan.json: %v", err)
	}
	state := oracleStateFile{Topic: "low confidence topic", CoreQuestion: "Low-confidence question", Iteration: 2}

	if _, err := finalizeOracleLoop(paths, state, plan, "go", nil, nil, 2, "complete", "", "aether oracle status"); err != nil {
		t.Fatalf("finalizeOracleLoop: %v", err)
	}

	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err == nil && len(instFile.Instincts) != 0 {
		t.Fatalf("a below-bar finding was promoted into an instinct: %+v", instFile.Instincts)
	}
}

func TestResearchPromotionDryRunDoesNotMutate(t *testing.T) {
	root := setupOracleAutofileTest(t)
	dataDir := filepath.Join(root, ".aether", "data")
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	plan := oraclePlanFile{
		Sources: map[string]oracleSource{},
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "Dry run question", Status: "answered", Confidence: 95,
				KeyFindings: []oracleFinding{
					{Text: "cmd/dryrun_198.go proves the dry-run path writes nothing at all."},
				},
			},
		},
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("seed plan.json: %v", err)
	}

	watchFiles := []string{"instincts.json", "entries.json", "instinct-graph.json", "COLONY_STATE.json"}
	type snapshot struct {
		data   []byte
		exists bool
	}
	before := map[string]snapshot{}
	readSnapshot := func(name string) snapshot {
		data, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			return snapshot{}
		}
		return snapshot{data: data, exists: true}
	}
	for _, f := range watchFiles {
		before[f] = readSnapshot(f)
	}

	provenance := oracleResearchProvenanceLabel(oracleStateFile{Topic: "dry run topic"})
	if _, err := runOraclePromote(root, 80, true, provenance); err != nil {
		t.Fatalf("dry-run promote: %v", err)
	}

	for _, f := range watchFiles {
		after := readSnapshot(f)
		if before[f].exists != after.exists {
			t.Fatalf("%s existence changed by the dry run (was %v, now %v)", f, before[f].exists, after.exists)
		}
		if after.exists && !bytes.Equal(before[f].data, after.data) {
			t.Fatalf("%s was modified by the dry run", f)
		}
	}
}
