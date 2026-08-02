package cmd

import (
	"bytes"
	"fmt"
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

	dispatches := plannedPhaseResearchDispatches(root, "balanced", "Build the exporter", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false, map[int]bool{1: true, 2: true})
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

// D-15: a fast run defaults every phase to skip (no approvals), but a phase
// the user explicitly flips on during a fast run still dispatches — speed is
// fast's default contract, not an unconditional block.
func TestFastPresetSkipsUnlessUserFlipsResearchOn(t *testing.T) {
	root := t.TempDir()
	seed := researchSeedWithDraft("Wire exporter")
	candidates := phaseResearchCandidates(colony.ColonyState{}, seed)

	if got := plannedPhaseResearchDispatches(root, "fast", "goal", candidates, codexSurveyContext{}, false, map[int]bool{}); len(got) != 0 {
		t.Fatalf("fast preset with no approvals dispatched research: %d dispatches", len(got))
	}
	if got := plannedPhaseResearchDispatches(root, "fast", "goal", candidates, codexSurveyContext{}, false, nil); len(got) != 0 {
		t.Fatalf("fast preset with nil approvals dispatched research: %d dispatches", len(got))
	}

	got := plannedPhaseResearchDispatches(root, "fast", "goal", candidates, codexSurveyContext{}, false, map[int]bool{1: true})
	if len(got) != 1 {
		t.Fatalf("fast preset with phase 1 flipped on dispatched %d, want 1 (D-15)", len(got))
	}
	if got[0].TaskID != "plan-research-phase-1" {
		t.Fatalf("fast preset flipped-on dispatch = %q, want plan-research-phase-1", got[0].TaskID)
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
	dispatches := plannedPhaseResearchDispatches(root, "balanced", "goal", phaseResearchCandidates(colony.ColonyState{}, seed), codexSurveyContext{}, false, map[int]bool{1: true, 2: true, 3: true})
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
	candidates := phaseResearchCandidates(colony.ColonyState{}, seed)
	approved := map[int]bool{1: true}

	// reresearch=true: stale findings on disk are not reused, phase 1 IS dispatched.
	dispatches := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, true, approved)
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
	if got := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, approved); len(got) != 0 {
		t.Fatalf("reresearch=false dispatches = %d, want 0 (single-run iterations still research once per phase)", len(got))
	}

	// planDepth "fast" with no approval still returns zero dispatches.
	if got := plannedPhaseResearchDispatches(root, "fast", "goal", candidates, codexSurveyContext{}, true, map[int]bool{}); len(got) != 0 {
		t.Fatalf("fast preset with reresearch=true and no approval dispatched %d, want 0 — speed is fast's default contract", len(got))
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

// seedPhaseResearchDecisions writes recs as fresh unresolved research
// decisions using the exact builder plan-research-approve reconstructs from.
func seedPhaseResearchDecisions(t *testing.T, recs []phaseResearchRecommendation) {
	t.Helper()
	file := PendingDecisionFile{Decisions: []PendingDecision{}}
	for _, rec := range recs {
		file.Decisions = append(file.Decisions, newPhaseResearchDecision(rec))
	}
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		t.Fatalf("seed pending decisions: %v", err)
	}
}

func runPlanResearchApprove(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs(append([]string{"plan-research-approve"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan-research-approve %v failed: %v", args, err)
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	return env["result"].(map[string]interface{})
}

func intsFromResult(t *testing.T, result map[string]interface{}, key string) []int {
	t.Helper()
	raw, ok := result[key].([]interface{})
	if !ok {
		t.Fatalf("result[%q] is not a list: %v (%T)", key, result[key], result[key])
	}
	ids := make([]int, 0, len(raw))
	for _, v := range raw {
		ids = append(ids, int(v.(float64)))
	}
	return ids
}

func defaultResearchSeed() []phaseResearchRecommendation {
	return []phaseResearchRecommendation{
		{PhaseID: 1, PhaseName: "New integration", Recommend: "research", Reason: "new external tech (api) is absent from the territory survey"},
		{PhaseID: 2, PhaseName: "Refactor internals", Recommend: "skip", Reason: "pure refactor -- the domain is already mapped by the territory survey"},
		{PhaseID: 3, PhaseName: "Clean up tests", Recommend: "skip", Reason: "no external technology signals found in the phase description -- the domain looks internal"},
	}
}

// TestPlanResearchApproveRecordsDecisions covers all six behaviours of
// `aether plan-research-approve` (RESEARCH-01, RESEARCH-02, RESEARCH-06).
func TestPlanResearchApproveRecordsDecisions(t *testing.T) {
	saveGlobals(t)

	t.Run("approve_all_accepts_queens_recommendation", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		seedPhaseResearchDecisions(t, defaultResearchSeed())

		result := runPlanResearchApprove(t, "--approve-all")
		approved := intsFromResult(t, result, "approved")
		skipped := intsFromResult(t, result, "skipped")
		overrides := intsFromResult(t, result, "overrides")
		if len(approved) != 1 || approved[0] != 1 {
			t.Fatalf("approved = %v, want [1]", approved)
		}
		if len(skipped) != 2 || skipped[0] != 2 || skipped[1] != 3 {
			t.Fatalf("skipped = %v, want [2 3]", skipped)
		}
		if len(overrides) != 0 {
			t.Fatalf("overrides = %v, want none (no flips)", overrides)
		}

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			t.Fatalf("load decisions: %v", err)
		}
		for _, d := range file.Decisions {
			if !d.Resolved {
				t.Fatalf("decision %s not resolved after --approve-all", d.ID)
			}
			if strings.Contains(d.Resolution, "user overrode") {
				t.Fatalf("decision %s wrongly marked as overridden: %s", d.ID, d.Resolution)
			}
		}
	})

	t.Run("flip_single_phase_overrides_direction", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		seedPhaseResearchDecisions(t, defaultResearchSeed())

		result := runPlanResearchApprove(t, "--flip", "3")
		overrides := intsFromResult(t, result, "overrides")
		if len(overrides) != 1 || overrides[0] != 3 {
			t.Fatalf("overrides = %v, want [3]", overrides)
		}

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			t.Fatalf("load decisions: %v", err)
		}
		found := false
		for _, d := range file.Decisions {
			if d.Phase == nil || *d.Phase != 3 {
				continue
			}
			found = true
			if !d.Resolved {
				t.Fatalf("phase 3 decision not resolved")
			}
			if !strings.Contains(d.Resolution, "user overrode") {
				t.Fatalf("phase 3 resolution = %q, want it to contain 'user overrode'", d.Resolution)
			}
		}
		if !found {
			t.Fatal("phase 3 decision not found in store")
		}
	})

	t.Run("flip_comma_list_flips_named_approves_rest", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		seed := append(defaultResearchSeed(), phaseResearchRecommendation{
			PhaseID: 4, PhaseName: "Adopt new SDK", Recommend: "research", Reason: "new external tech (sdk) is absent from the territory survey",
		})
		seedPhaseResearchDecisions(t, seed)

		result := runPlanResearchApprove(t, "--flip", "2,4")
		approved := intsFromResult(t, result, "approved")
		skipped := intsFromResult(t, result, "skipped")
		overrides := intsFromResult(t, result, "overrides")
		if len(approved) != 2 || approved[0] != 1 || approved[1] != 2 {
			t.Fatalf("approved = %v, want [1 2] (1 unflipped research, 2 flipped skip->research)", approved)
		}
		if len(skipped) != 2 || skipped[0] != 3 || skipped[1] != 4 {
			t.Fatalf("skipped = %v, want [3 4] (3 unflipped skip, 4 flipped research->skip)", skipped)
		}
		if len(overrides) != 2 || overrides[0] != 2 || overrides[1] != 4 {
			t.Fatalf("overrides = %v, want [2 4]", overrides)
		}
	})

	t.Run("auto_marks_every_resolution_auto_accepted", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		seedPhaseResearchDecisions(t, defaultResearchSeed())

		result := runPlanResearchApprove(t, "--auto")
		logLine, _ := result["log_line"].(string)
		if !strings.Contains(logLine, "auto-accepted:") {
			t.Fatalf("log_line = %q, want it to contain 'auto-accepted:'", logLine)
		}

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			t.Fatalf("load decisions: %v", err)
		}
		for _, d := range file.Decisions {
			if !strings.HasPrefix(d.Resolution, "auto-accepted (autopilot)") {
				t.Fatalf("decision %s resolution = %q, want prefix 'auto-accepted (autopilot)'", d.ID, d.Resolution)
			}
		}
	})

	t.Run("no_unresolved_decisions_returns_zero_not_crash", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := runPlanResearchApprove(t, "--approve-all")
		approved := intsFromResult(t, result, "approved")
		skipped := intsFromResult(t, result, "skipped")
		if len(approved) != 0 || len(skipped) != 0 {
			t.Fatalf("approved/skipped = %v/%v, want both empty", approved, skipped)
		}
	})

	t.Run("no_new_store_file_beyond_pending_decisions", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		seedPhaseResearchDecisions(t, defaultResearchSeed())

		before, err := os.ReadDir(store.BasePath())
		if err != nil {
			t.Fatalf("read store dir before: %v", err)
		}
		beforeNames := map[string]bool{}
		for _, entry := range before {
			beforeNames[entry.Name()] = true
		}

		runPlanResearchApprove(t, "--approve-all")

		after, err := os.ReadDir(store.BasePath())
		if err != nil {
			t.Fatalf("read store dir after: %v", err)
		}
		for _, entry := range after {
			if beforeNames[entry.Name()] {
				continue
			}
			if entry.Name() != pendingDecisionsFile {
				t.Fatalf("unexpected new file in store dir: %s", entry.Name())
			}
		}
	})
}

// researchProposalTestPhases returns two phases: one with an external-tech
// signal absent from any survey (recommends "research" via the no_survey
// signal) and one with no external-tech signal (recommends "skip").
func researchProposalTestPhases() []colony.Phase {
	return []colony.Phase{
		{ID: 1, Name: "Wire billing API", Description: "Integrate with the external billing API", Status: colony.PhaseReady},
		{ID: 2, Name: "Internal cleanup", Description: "Refactor internal helper functions", Status: colony.PhaseReady},
	}
}

func setupPhaseResearchManifestTest(t *testing.T, phases []colony.Phase) (dataDir, root string) {
	t.Helper()
	dataDir = setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Wire the exporter to the new billing API"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: phases},
	})
	return dataDir, root
}

func runPlanOnly(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs(append([]string{"plan", "--plan-only"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan --plan-only %v failed: %v", args, err)
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env)
	}
	return env["result"].(map[string]interface{})
}

// TestPlanManifestCarriesResearchProposal covers behaviours 1, 2 and 6 of
// Plan 05 Task 2: the plan-only result map carries a non-empty research
// proposal and rendered card whenever candidate phases exist, in both the
// existing-plan and fresh-plan branches, and a fast run still emits both.
func TestPlanManifestCarriesResearchProposal(t *testing.T) {
	t.Run("fresh_plan_branch_with_candidates", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "balanced")
		proposal, ok := result["research_proposal"].([]interface{})
		if !ok || len(proposal) != 2 {
			t.Fatalf("research_proposal = %v, want 2 entries", result["research_proposal"])
		}
		card, _ := result["research_proposal_card"].(string)
		if card == "" || !strings.Contains(card, "Research proposal") {
			t.Fatalf("research_proposal_card = %q, want non-empty proposal header", card)
		}
		manifest, ok := result["plan_manifest"].(map[string]interface{})
		if !ok {
			t.Fatalf("plan_manifest missing from result: %+v", result)
		}
		if _, ok := manifest["research_proposal"].([]interface{}); !ok {
			t.Fatalf("manifest missing research_proposal: %+v", manifest)
		}
		if manifestCard, _ := manifest["research_proposal_card"].(string); manifestCard == "" {
			t.Fatalf("manifest research_proposal_card is empty")
		}
	})

	t.Run("existing_plan_early_return_branch", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t)
		if existing, _ := result["existing_plan"].(bool); !existing {
			t.Fatalf("existing_plan = %v, want true (early-return branch)", result["existing_plan"])
		}
		proposal, ok := result["research_proposal"].([]interface{})
		if !ok || len(proposal) != 2 {
			t.Fatalf("research_proposal = %v, want 2 entries in early-return branch", result["research_proposal"])
		}
		card, _ := result["research_proposal_card"].(string)
		if card == "" {
			t.Fatalf("research_proposal_card empty in early-return branch")
		}
		if _, ok := result["research_awaiting_approval"]; !ok {
			t.Fatalf("research_awaiting_approval missing from early-return branch result")
		}
		if _, ok := result["research_warning"]; !ok {
			t.Fatalf("research_warning missing from early-return branch result")
		}
	})

	t.Run("fast_depth_still_emits_proposal", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "fast")
		proposal, ok := result["research_proposal"].([]interface{})
		if !ok || len(proposal) != 2 {
			t.Fatalf("fast depth research_proposal = %v, want 2 entries so the user can flip one on", result["research_proposal"])
		}
		for _, entry := range proposal {
			rec := entry.(map[string]interface{})
			if rec["recommend"] != "skip" {
				t.Fatalf("fast depth recommendation = %v, want skip (D-15)", rec["recommend"])
			}
		}
		card, _ := result["research_proposal_card"].(string)
		if !strings.Contains(card, "Flip:") {
			t.Fatalf("fast depth card missing flip instruction: %q", card)
		}
	})
}

// TestPlanManifestWarnsWhenResearchBatchUnanswered covers behaviours 3, 4 and
// 5 of Plan 05 Task 2: an unanswered batch warns loudly without erroring, a
// resolved batch clears the warning, and re-running the manifest supersedes
// rather than accumulates unresolved decisions for the same phase.
func TestPlanManifestWarnsWhenResearchBatchUnanswered(t *testing.T) {
	t.Run("unanswered_batch_warns_never_errors", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		result := runPlanOnly(t, "--refresh", "--depth", "balanced")
		if awaiting, _ := result["research_awaiting_approval"].(bool); !awaiting {
			t.Fatalf("research_awaiting_approval = %v, want true before anyone answers", result["research_awaiting_approval"])
		}
		warning, _ := result["research_warning"].(string)
		if warning == "" || !strings.Contains(warning, "plan-research-approve") {
			t.Fatalf("research_warning = %q, want a non-empty warning naming plan-research-approve", warning)
		}
	})

	t.Run("resolved_batch_clears_warning", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		runPlanOnly(t, "--refresh", "--depth", "balanced")
		runPlanResearchApprove(t, "--approve-all")

		result := runPlanOnly(t, "--refresh", "--depth", "balanced")
		if awaiting, _ := result["research_awaiting_approval"].(bool); awaiting {
			t.Fatalf("research_awaiting_approval = %v, want false once decisions are resolved", result["research_awaiting_approval"])
		}
		if warning, _ := result["research_warning"].(string); warning != "" {
			t.Fatalf("research_warning = %q, want empty once decisions are resolved", warning)
		}
	})

	t.Run("rerun_supersedes_not_accumulates", func(t *testing.T) {
		saveGlobals(t)
		setupPhaseResearchManifestTest(t, researchProposalTestPhases())

		runPlanOnly(t, "--refresh", "--depth", "balanced")
		runPlanOnly(t, "--refresh", "--depth", "balanced")

		var file PendingDecisionFile
		if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
			t.Fatalf("load decisions: %v", err)
		}
		unresolvedByPhase := map[int]int{}
		for _, d := range file.Decisions {
			if d.Type != phaseResearchDecisionType || d.Resolved || d.Phase == nil {
				continue
			}
			unresolvedByPhase[*d.Phase]++
		}
		for phase, count := range unresolvedByPhase {
			if count != 1 {
				t.Fatalf("phase %d has %d unresolved research-decisions after two manifest runs, want 1 (no accumulation)", phase, count)
			}
		}
		if len(unresolvedByPhase) != 2 {
			t.Fatalf("expected unresolved decisions for 2 phases, got %v", unresolvedByPhase)
		}
	})
}

// TestResearchDispatchGatedOnApproval covers Plan 05 Task 3's six behaviours:
// dispatch is gated on approval first, and approval never overrides the
// re-research or fast-depth rules that sit behind it.
func TestResearchDispatchGatedOnApproval(t *testing.T) {
	seed := researchSeedWithDraft("Phase one", "Phase two", "Phase three")
	candidates := phaseResearchCandidates(colony.ColonyState{}, seed)

	t.Run("only_approved_phase_dispatches", func(t *testing.T) {
		root := t.TempDir()
		dispatches := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, map[int]bool{2: true})
		if len(dispatches) != 1 {
			t.Fatalf("dispatches = %d, want 1 (only phase 2 approved)", len(dispatches))
		}
		if dispatches[0].TaskID != "plan-research-phase-2" {
			t.Fatalf("dispatch TaskID = %q, want plan-research-phase-2", dispatches[0].TaskID)
		}
	})

	t.Run("empty_approved_map_dispatches_nothing", func(t *testing.T) {
		root := t.TempDir()
		if got := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, map[int]bool{}); len(got) != 0 {
			t.Fatalf("dispatches = %d, want 0 with an empty approved map", len(got))
		}
	})

	t.Run("nil_approved_map_dispatches_nothing", func(t *testing.T) {
		root := t.TempDir()
		if got := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, nil); len(got) != 0 {
			t.Fatalf("dispatches = %d, want 0 with a nil approved map (nobody has answered yet)", len(got))
		}
	})

	t.Run("approval_does_not_override_reresearch_rule", func(t *testing.T) {
		root := t.TempDir()
		researchDir := filepath.Join(root, ".aether", "data", "phase-research")
		if err := os.MkdirAll(researchDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(researchDir, "phase-2-research.md"), []byte("# Phase 2 Research\nworker findings\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, map[int]bool{2: true}); len(got) != 0 {
			t.Fatalf("dispatches = %d, want 0 -- approved phase 2 already has worker-authored research and reresearch=false", len(got))
		}
	})

	t.Run("approval_does_not_override_fast_depth_rule", func(t *testing.T) {
		root := t.TempDir()
		// A fast run with an approved phase DOES dispatch it (D-15) -- approval
		// flips the default on, it does not get vetoed by depth.
		got := plannedPhaseResearchDispatches(root, "fast", "goal", candidates, codexSurveyContext{}, false, map[int]bool{2: true})
		if len(got) != 1 || got[0].TaskID != "plan-research-phase-2" {
			t.Fatalf("fast depth with phase 2 approved dispatched %+v, want exactly plan-research-phase-2 (D-15)", got)
		}
	})

	t.Run("dispatch_shape_unchanged_for_approved_phases", func(t *testing.T) {
		root := t.TempDir()
		got := plannedPhaseResearchDispatches(root, "balanced", "goal", candidates, codexSurveyContext{}, false, map[int]bool{2: true})
		if len(got) != 1 {
			t.Fatalf("dispatches = %d, want 1", len(got))
		}
		d := got[0]
		if d.Stage != phaseResearchStage {
			t.Errorf("stage = %q, want %q", d.Stage, phaseResearchStage)
		}
		if d.Caste != "scout" {
			t.Errorf("caste = %q, want scout", d.Caste)
		}
		if d.Wave != 1 {
			t.Errorf("wave = %d, want 1", d.Wave)
		}
		if len(d.Outputs) != 1 || d.Outputs[0] != "phase-2-research.md" {
			t.Errorf("outputs = %v, want [phase-2-research.md]", d.Outputs)
		}
		wantName := deterministicAntName("scout", fmt.Sprintf("%s|plan-research|phase-%d", root, 2))
		if d.Name != wantName {
			t.Errorf("name = %q, want deterministic name %q", d.Name, wantName)
		}
	})
}
