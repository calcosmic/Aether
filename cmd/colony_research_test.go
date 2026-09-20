package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// writeColonyResearchDoc drops a research document with a sentinel phrase that
// can only reach a worker brief by being carried there.
func writeColonyResearchDoc(t *testing.T, root, name, sentinel string) string {
	t.Helper()
	dir := filepath.Join(root, ".aether", "research")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("create research dir: %v", err)
	}
	body := "---\ntitle: \"Saved research\"\nconfidence: 92\n---\n\n# Findings\n\n" + sentinel + "\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write research doc: %v", err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("relativize research doc: %v", err)
	}
	return filepath.ToSlash(rel)
}

func writeColonyStateWithResearch(t *testing.T, root string, docs []string) {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	goal := "Ship the ingest rewrite"
	state := colony.ColonyState{Goal: &goal, ResearchDocs: docs}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), data, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

// TestColonyResearchReachesRouteSetterOnFirstPlan is the anti-orphan test.
//
// This repo's recurring failure is capability that exists but is never called.
// A fresh colony with zero phases is precisely the case that matters: phase
// research cannot run before phases exist, so this injection point is the only
// way research the operator did up front reaches the worker who decides what
// the phases should be.
func TestColonyResearchReachesRouteSetterOnFirstPlan(t *testing.T) {
	root := t.TempDir()
	const sentinel = "Postgres was rejected because the write volume never exceeds 40 rows per second."
	doc := writeColonyResearchDoc(t, root, "2026-08-16-cache-storage.md", sentinel)
	writeColonyStateWithResearch(t, root, []string{doc})

	survey, _ := loadCodexSurveyContext(root)

	var routeSetter planningWorkerSpec
	for _, spec := range planningWorkerSpecs {
		if spec.Caste == "route_setter" {
			routeSetter = spec
		}
	}
	if routeSetter.Caste == "" {
		t.Fatal("no route_setter spec found")
	}

	brief := renderPlanningWorkerBrief(root, survey, routeSetter, nil)
	if !strings.Contains(brief, sentinel) {
		t.Fatalf("the Route-Setter's brief does not contain the research the operator pointed at.\nThe handoff is wired at the flag but not delivered.\nBrief:\n%s", truncateString(brief, 1200))
	}
	if !strings.Contains(brief, "## Colony Research") {
		t.Error("research reached the brief without its framing section")
	}
	// Framed as evidence, not orders: a worker must be free to contradict it.
	if !strings.Contains(brief, "not as instructions") {
		t.Error("research is not framed as evidence the worker may contradict")
	}
}

func TestColonyResearchReachesScoutAndPhaseResearchBriefs(t *testing.T) {
	root := t.TempDir()
	const sentinel = "The ingest path already retries on 429 in fetchBatch."
	doc := writeColonyResearchDoc(t, root, "2026-08-16-ingest.md", sentinel)
	writeColonyStateWithResearch(t, root, []string{doc})

	survey, _ := loadCodexSurveyContext(root)

	var scout planningWorkerSpec
	for _, spec := range planningWorkerSpecs {
		if spec.Caste == "scout" {
			scout = spec
		}
	}
	if brief := renderPlanningWorkerBrief(root, survey, scout, nil); !strings.Contains(brief, sentinel) {
		t.Error("the planning Scout does not receive the colony's research, so it may rediscover what is already known")
	}

	phaseBrief := renderPhaseResearchBrief(root, "Ship the ingest rewrite",
		phaseResearchCandidate{ID: 3, Name: "Retry handling"}, survey)
	if !strings.Contains(phaseBrief, sentinel) {
		t.Error("a phase-research Scout does not receive the colony's research")
	}
}

func TestColonyResearchSectionRespectsBudget(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("This paragraph documents a finding in detail.\n\n", 2000)
	doc := writeColonyResearchDoc(t, root, "2026-08-16-big.md", big)

	section := resolveColonyResearchSection(root, []string{doc})
	if len(section) == 0 {
		t.Fatal("a large research document produced no section at all")
	}
	if len(section) > colonyResearchTotalBudgetChars+1000 {
		t.Errorf("section is %d chars, over the %d budget", len(section), colonyResearchTotalBudgetChars)
	}
	if !strings.Contains(section, "truncated — full research:") || !strings.Contains(section, doc) {
		t.Error("a truncated document must point at the full file")
	}
}

// Silently dropping a document the operator pointed at is the failure mode
// this whole handoff exists to prevent, so an over-budget document is named.
func TestColonyResearchNamesDocumentsItCouldNotInclude(t *testing.T) {
	root := t.TempDir()
	filler := strings.Repeat("Finding paragraph with substance.\n\n", 1500)
	first := writeColonyResearchDoc(t, root, "2026-08-16-a.md", filler)
	second := writeColonyResearchDoc(t, root, "2026-08-16-b.md", filler)

	section := resolveColonyResearchSection(root, []string{first, second})
	if len(section) > colonyResearchTotalBudgetChars+1000 {
		t.Errorf("two documents blew the total budget: %d chars", len(section))
	}
	if !strings.Contains(section, second) {
		t.Errorf("the second document was dropped without being named:\n%s", truncateString(section, 800))
	}
}

func TestColonyResearchIsEmptyWithoutDocuments(t *testing.T) {
	root := t.TempDir()
	if section := resolveColonyResearchSection(root, nil); section != "" {
		t.Errorf("a colony with no research produced a section: %q", section)
	}
}

func TestColonyResearchStripsFrontMatter(t *testing.T) {
	root := t.TempDir()
	doc := writeColonyResearchDoc(t, root, "2026-08-16-fm.md", "The actual finding.")
	section := resolveColonyResearchSection(root, []string{doc})
	if strings.Contains(section, "confidence: 92") {
		t.Error("front matter leaked into the worker brief; workers should read the research, not its metadata")
	}
	if !strings.Contains(section, "The actual finding.") {
		t.Error("stripping front matter removed the body too")
	}
}

func TestResearchDocValidationFailsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aether", "research"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cases := map[string]string{
		"missing file":     ".aether/research/nope.md",
		"directory":        ".aether/research",
		"escapes the repo": "../outside.md",
		"absolute path":    "/etc/hosts",
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := validateColonyResearchDocs(root, []string{path}); err == nil {
				t.Fatalf("%s was accepted as a research document", name)
			}
		})
	}

	good := writeColonyResearchDoc(t, root, "2026-08-16-ok.md", "Fine.")
	if _, err := validateColonyResearchDocs(root, []string{good}); err != nil {
		t.Fatalf("a real research document was rejected: %v", err)
	}
}

// TestBuildWorkerBriefKeepsColonyResearchSmall: build briefs get a reduced
// share so the task stays the bulk of the brief.
func TestBuildWorkerBriefKeepsColonyResearchSmall(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("A finding paragraph about the ingest path.\n\n", 2000)
	doc := writeColonyResearchDoc(t, root, "2026-08-16-build.md", big)

	planning := resolveColonyResearchSection(root, []string{doc})
	building := resolveColonyResearchSection(root, []string{doc}, colonyResearchBuildBudgetChars)
	if len(building) >= len(planning) {
		t.Errorf("build brief research (%d chars) is not smaller than planning research (%d chars)", len(building), len(planning))
	}
	if len(building) > colonyResearchBuildBudgetChars+1000 {
		t.Errorf("build research section is %d chars, over the %d budget", len(building), colonyResearchBuildBudgetChars)
	}
}

func TestPlanRevisionEvidenceOnEmptyPlanIsRejectedNotDropped(t *testing.T) {
	root := t.TempDir()
	doc := writeColonyResearchDoc(t, root, "2026-08-16-evidence.md", "Some evidence.")
	state := colony.ColonyState{Plan: colony.Plan{}}

	_, err := buildPlanRevisionContext(root, state, codexPlanOptions{
		Refresh:          true,
		RevisionType:     "research",
		RevisionEvidence: []string{doc},
	})
	if err == nil {
		t.Fatal("revision evidence on a colony with no phases was silently dropped")
	}
	if !strings.Contains(err.Error(), "--research") {
		t.Errorf("the error should point at the flag that works on a first plan, got: %v", err)
	}

	// Without revision metadata there is nothing to warn about.
	if _, err := buildPlanRevisionContext(root, state, codexPlanOptions{Refresh: true}); err != nil {
		t.Errorf("a plain refresh on an empty plan should not error: %v", err)
	}
}

func TestInitRecordsResearchDocumentPointer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	doc := writeColonyResearchDoc(t, root, "2026-08-16-init.md", "A conclusion worth planning from.")

	rootCmd.SetArgs([]string{"init", "--research", doc, "Ship the ingest rewrite"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.ResearchDocs) != 1 || state.ResearchDocs[0] != doc {
		t.Fatalf("colony state recorded research_docs=%v, want [%s]", state.ResearchDocs, doc)
	}

	// Research is evidence, not governance: it must not be folded into the
	// charter, whose fields are capped and rendered to workers as hard rules.
	if state.Charter != nil {
		combined := state.Charter.Intent + state.Charter.Vision + state.Charter.Governance +
			state.Charter.Goals + state.Charter.TechStack + state.Charter.KeyRisks + state.Charter.Constraints
		if strings.Contains(combined, "A conclusion worth planning from") {
			t.Error("research content was written into the charter; it must stay a pointer")
		}
	}
}

func TestInitRejectsMissingResearchDocument(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	rootCmd.SetArgs([]string{"init", "--research", ".aether/research/does-not-exist.md", "Ship it"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned an unexpected error type: %v", err)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err == nil {
		t.Fatal("init created a colony despite being pointed at a research document that does not exist")
	}
}

func TestLoadColonyResearchDocsReadsState(t *testing.T) {
	root := t.TempDir()
	doc := writeColonyResearchDoc(t, root, "2026-08-16-state.md", "Body.")
	writeColonyStateWithResearch(t, root, []string{doc})

	got := loadColonyResearchDocs(root)
	if len(got) != 1 || got[0] != doc {
		t.Fatalf("loadColonyResearchDocs = %v, want [%s]", got, doc)
	}

	// A colony with no state must not error or invent documents.
	if got := loadColonyResearchDocs(t.TempDir()); len(got) != 0 {
		t.Errorf("a repo with no colony state returned research docs: %v", got)
	}
}

// TestPlanResearchAccumulatesAndPersists proves "point once": a document named
// on one run is still there on the next, and a second document adds to it
// rather than replacing it.
func TestPlanResearchAccumulatesAndPersists(t *testing.T) {
	root := t.TempDir()
	first := writeColonyResearchDoc(t, root, "2026-08-16-first.md", "First conclusion.")
	second := writeColonyResearchDoc(t, root, "2026-08-17-second.md", "Second conclusion.")

	merged, changed, err := mergeColonyResearchDocs(root, nil, []string{first})
	if err != nil || !changed || len(merged) != 1 {
		t.Fatalf("first --research: merged=%v changed=%v err=%v", merged, changed, err)
	}

	merged, changed, err = mergeColonyResearchDocs(root, merged, []string{second})
	if err != nil || !changed || len(merged) != 2 {
		t.Fatalf("second --research did not accumulate: merged=%v changed=%v err=%v", merged, changed, err)
	}

	// A later flagless run keeps what the colony already carries.
	kept, changed, err := mergeColonyResearchDocs(root, merged, nil)
	if err != nil || changed || len(kept) != 2 {
		t.Fatalf("a run without --research dropped or rewrote the colony's research: kept=%v changed=%v err=%v", kept, changed, err)
	}

	// Repeating the same path is not a change worth rewriting state for.
	if _, changed, err := mergeColonyResearchDocs(root, merged, []string{first}); err != nil || changed {
		t.Errorf("re-passing an existing document reported a change: changed=%v err=%v", changed, err)
	}

	// A bad path fails the run rather than being quietly ignored.
	if _, _, err := mergeColonyResearchDocs(root, merged, []string{".aether/research/ghost.md"}); err == nil {
		t.Error("plan accepted a research path that does not exist")
	}
}

func TestColonyResearchSurvivesStateRoundTrip(t *testing.T) {
	// omitempty must keep older state files loading unchanged.
	var state colony.ColonyState
	if err := json.Unmarshal([]byte(`{"goal":"legacy colony"}`), &state); err != nil {
		t.Fatalf("legacy state failed to load: %v", err)
	}
	if len(state.ResearchDocs) != 0 {
		t.Errorf("legacy state gained research docs: %v", state.ResearchDocs)
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), "research_docs") {
		t.Errorf("a colony with no research emits an empty research_docs key: %s", encoded)
	}

	state.ResearchDocs = []string{".aether/research/x.md"}
	encoded, err = json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal with docs: %v", err)
	}
	var reloaded colony.ColonyState
	if err := json.Unmarshal(encoded, &reloaded); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if fmt.Sprint(reloaded.ResearchDocs) != fmt.Sprint(state.ResearchDocs) {
		t.Errorf("research docs did not round-trip: %v vs %v", reloaded.ResearchDocs, state.ResearchDocs)
	}
}
