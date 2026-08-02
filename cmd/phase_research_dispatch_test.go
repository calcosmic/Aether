package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func researchSeedWithDraft(names ...string) codexPlanIterationState {
	phases := make([]codexWorkerPlanPhase, 0, len(names))
	for _, name := range names {
		phases = append(phases, codexWorkerPlanPhase{Name: name, Description: "Draft phase " + name})
	}
	return codexPlanIterationState{PreviousPlanDraft: &codexWorkerPlanArtifact{Phases: phases}}
}

// The v5 restoration contract: once the Route-Setter has drafted phases, the
// planner dispatches one phase_research Scout per phase. The old
// writePhaseResearchArtifacts stub dispatched nothing, ever.
func TestPlanEmitsPhaseResearchDispatchesFromDraft(t *testing.T) {
	root := t.TempDir()
	seed := researchSeedWithDraft("Wire exporter", "Ship dashboard")

	dispatches := plannedPhaseResearchDispatches(root, "balanced", "Build the exporter", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false)
	if len(dispatches) != 2 {
		t.Fatalf("dispatches = %d, want one research Scout per draft phase (2)", len(dispatches))
	}
	for i, d := range dispatches {
		if d.Stage != "phase_research" {
			t.Errorf("dispatch %d stage = %q, want phase_research", i, d.Stage)
		}
		if d.Caste != "scout" {
			t.Errorf("dispatch %d caste = %q, want scout", i, d.Caste)
		}
		if !strings.Contains(d.Brief, "Phase Domain Research") {
			t.Errorf("dispatch %d brief missing research mission", i)
		}
		if !strings.Contains(d.Brief, "## Recommended Approach") {
			t.Errorf("dispatch %d brief missing six-section output contract", i)
		}
		wantFile := "phase-" + string(rune('1'+i)) + "-research.md"
		if len(d.Outputs) != 1 || d.Outputs[0] != wantFile {
			t.Errorf("dispatch %d outputs = %v, want [%s]", i, d.Outputs, wantFile)
		}
	}
	if dispatches[0].Name == dispatches[1].Name {
		t.Errorf("research scouts share a name %q; merge-by-name in finalize would collide", dispatches[0].Name)
	}
}

// Fast preset skips research — speed is its contract.
func TestPlanFastPresetSkipsPhaseResearch(t *testing.T) {
	root := t.TempDir()
	seed := researchSeedWithDraft("Wire exporter")
	if got := plannedPhaseResearchDispatches(root, "fast", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false); len(got) != 0 {
		t.Fatalf("fast preset dispatched research: %d dispatches", len(got))
	}
}

// Research runs once per phase, not once per iteration: worker-authored
// research on disk suppresses a re-dispatch, while a finalize fallback
// template does not. This pins the reresearch=false path.
func TestPhaseResearchDispatchedOncePerPhase(t *testing.T) {
	root := t.TempDir()
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte("# Phase 1 Research\nworker findings\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(researchDir, "phase-2-research.md"), []byte("# Phase 2 Research\n**Research scope:** synthesized from territory survey and scout findings (no dedicated research worker ran for this phase)\n"), 0644); err != nil {
		t.Fatal(err)
	}

	seed := researchSeedWithDraft("Already researched", "Only templated", "Never researched")
	dispatches := plannedPhaseResearchDispatches(root, "balanced", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false)
	if len(dispatches) != 2 {
		t.Fatalf("dispatches = %d, want 2 (phase 1 has worker research; phases 2-3 need it)", len(dispatches))
	}
	gotTasks := []string{dispatches[0].TaskID, dispatches[1].TaskID}
	if gotTasks[0] != "plan-research-phase-2" || gotTasks[1] != "plan-research-phase-3" {
		t.Fatalf("dispatched %v, want research for phases 2 and 3 only", gotTasks)
	}
}

// RESEARCH-04/D-05: a replan's first iteration re-researches every candidate
// phase from scratch, ignoring stale worker-authored findings on disk. This
// is the inverse of TestPhaseResearchDispatchedOncePerPhase.
func TestReplanReResearchesPhases(t *testing.T) {
	root := t.TempDir()
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte("# Phase 1 Research\nstale worker findings\n"), 0644); err != nil {
		t.Fatal(err)
	}

	seed := researchSeedWithDraft("Already researched")

	// reresearch=true: stale findings on disk are not reused, phase 1 IS dispatched.
	dispatches := plannedPhaseResearchDispatches(root, "balanced", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, true)
	if len(dispatches) != 1 {
		t.Fatalf("reresearch=true dispatches = %d, want 1 (stale findings must not be reused)", len(dispatches))
	}
	if dispatches[0].TaskID != "plan-research-phase-1" {
		t.Fatalf("reresearch=true dispatched %q, want plan-research-phase-1", dispatches[0].TaskID)
	}
	if len(dispatches[0].Outputs) != 1 || dispatches[0].Outputs[0] != "phase-1-research.md" {
		t.Fatalf("reresearch=true outputs = %v, want [phase-1-research.md] (overwrite in place, no timestamped sibling)", dispatches[0].Outputs)
	}

	// reresearch=false with the same file on disk: phase 1 is NOT dispatched.
	if got := plannedPhaseResearchDispatches(root, "balanced", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false); len(got) != 0 {
		t.Fatalf("reresearch=false dispatches = %d, want 0 (single-run iterations still research once per phase)", len(got))
	}

	// planDepth "fast" still returns zero dispatches regardless of reresearch.
	if got := plannedPhaseResearchDispatches(root, "fast", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, true); len(got) != 0 {
		t.Fatalf("fast preset with reresearch=true dispatched %d, want 0 — speed is fast's contract", len(got))
	}
}

// RESEARCH-05: the research Scout's brief must name the territory survey
// docs it should read before scanning the repo, or explicitly say none exist.
func TestRenderPhaseResearchBriefIncludesSurvey(t *testing.T) {
	candidate := phaseResearchCandidate{ID: 1, Name: "Wire exporter", Description: "Ship the exporter"}

	populated := codexSurveyContext{
		SurveyDocs:   []string{"nest.md", "provisions.md"},
		Languages:    []string{"Go", "TypeScript"},
		Frameworks:   []string{"cobra"},
		Dependencies: []string{"cobra", "testify"},
	}
	brief := renderPhaseResearchBrief("Build the exporter", candidate, populated)
	if !strings.Contains(brief, filepath.ToSlash(filepath.Join(".aether", "data", "survey", "nest.md"))) {
		t.Errorf("brief missing survey doc nest.md:\n%s", brief)
	}
	if !strings.Contains(brief, filepath.ToSlash(filepath.Join(".aether", "data", "survey", "provisions.md"))) {
		t.Errorf("brief missing survey doc provisions.md:\n%s", brief)
	}
	if !strings.Contains(brief, "Go") || !strings.Contains(brief, "TypeScript") || !strings.Contains(brief, "cobra") {
		t.Errorf("brief missing already-mapped territory (Go, TypeScript, cobra):\n%s", brief)
	}

	empty := renderPhaseResearchBrief("Build the exporter", candidate, codexSurveyContext{})
	if !strings.Contains(empty, "No territory survey available — scan the repository directly.") {
		t.Errorf("brief with zero-value survey missing explicit fallback sentence:\n%s", empty)
	}

	for _, brief := range []string{brief, empty} {
		if !strings.Contains(brief, "## Recommended Approach") {
			t.Errorf("six-section output contract regressed — missing ## Recommended Approach:\n%s", brief)
		}
		if !strings.Contains(brief, "## Files to Study") {
			t.Errorf("six-section output contract regressed — missing ## Files to Study:\n%s", brief)
		}
	}
}

// The finalize chain validator accepts phase_research Scouts running before
// the Route-Setter (research informs the route) and still rejects everything
// else (covered in the dynamic worker-set test).
func TestPlanningWorkerChainAcceptsResearchScouts(t *testing.T) {
	chain := []codexPlanningDispatch{
		{Stage: "scouting", Wave: 1, Caste: "scout", Name: "Scout-1"},
		{Stage: "routing", Wave: 2, Caste: "route_setter", Name: "Route-1"},
		{Stage: "phase_research", Wave: 1, Caste: "scout", Name: "Research-1"},
		{Stage: "phase_research", Wave: 1, Caste: "scout", Name: "Research-2"},
	}
	if err := validatePlanningWorkerChain("test", chain); err != nil {
		t.Fatalf("research chain rejected: %v", err)
	}
	misordered := append([]codexPlanningDispatch{}, chain...)
	misordered[2].Wave = 3
	if err := validatePlanningWorkerChain("test", misordered); err == nil {
		t.Fatal("research after route-setter should be rejected — research informs the route")
	}
}

// Build briefs carry the phase's research — the point of researching at all.
// Evidence contract: `aether build N --print-brief` surfaces the Scout's
// Recommended Approach to every worker.
func TestBuildBriefCarriesPhaseResearch(t *testing.T) {
	root := t.TempDir()
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatal(err)
	}
	research := "# Phase 3 Research: Wire exporter\n\n## Recommended Approach\nUse the existing retry helper in cmd/retry.go rather than a new loop.\n"
	if err := os.WriteFile(filepath.Join(researchDir, "phase-3-research.md"), []byte(research), 0644); err != nil {
		t.Fatal(err)
	}

	section := resolvePhaseResearchSection(root, 3)
	if !strings.Contains(section, "## Phase Research") {
		t.Fatalf("section missing heading:\n%s", section)
	}
	if !strings.Contains(section, "Recommended Approach") || !strings.Contains(section, "cmd/retry.go") {
		t.Fatalf("section missing research content:\n%s", section)
	}
	if resolvePhaseResearchSection(root, 4) != "" {
		t.Error("phase without research should produce no section")
	}
}

// A runaway research file must not drown the worker's assignment.
func TestPhaseResearchSectionIsBounded(t *testing.T) {
	root := t.TempDir()
	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatal(err)
	}
	huge := "# Phase 1 Research: big\n\n" + strings.Repeat("Paragraph of findings that goes on and on.\n\n", 400)
	if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte(huge), 0644); err != nil {
		t.Fatal(err)
	}
	section := resolvePhaseResearchSection(root, 1)
	if len(section) > phaseResearchBriefBudgetChars+300 {
		t.Fatalf("section = %d chars, exceeds budget %d", len(section), phaseResearchBriefBudgetChars)
	}
	if !strings.Contains(section, "truncated — full research") {
		t.Error("truncated section should point at the full file")
	}
}
