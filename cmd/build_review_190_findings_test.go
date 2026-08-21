package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// seedHandoffColonyForBriefTests builds a populated colony with one stored
// worker handoff and returns the store root plus the phase.
//
// The colony carries enough memory that resolveCodexWorkerContext() clears its
// 128-character minimum threshold — below that it returns an empty capsule and
// every assertion about capsule content would pass vacuously.
func seedHandoffColonyForBriefTests(t *testing.T, sentinel string) (string, colony.Phase, colony.ColonyState) {
	t.Helper()

	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	now := time.Now().Format(time.RFC3339)
	goal := "Handoff duplication regression colony"

	taskID1 := "1.1"
	taskID2 := "1.2"
	phase := colony.Phase{
		ID:     1,
		Name:   "Handoff Duplication Phase",
		Status: colony.PhaseReady,
		Tasks: []colony.Task{
			{ID: &taskID1, Goal: "First task", Status: colony.TaskPending},
			{ID: &taskID2, Goal: "Second task", Status: colony.TaskPending},
		},
	}

	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
		Memory: colony.Memory{
			Decisions: []colony.Decision{
				{ID: "d1", Phase: 1, Claim: "One home for each context section", Rationale: "Duplicated steering wastes the worker's budget", Timestamp: now},
			},
			PhaseLearnings: []colony.PhaseLearning{
				{
					ID: "pl1", Phase: 1, PhaseName: "Handoff Duplication Phase", Timestamp: now,
					Learnings: []colony.Learning{
						{Claim: "Prior handoffs belong to the capsule alone", Status: "validated", Tested: true},
					},
				},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// One stored handoff, carrying a sentinel string that appears nowhere
	// else, so counting its occurrences counts channels of delivery.
	dispatch := codex.WorkerDispatch{
		WorkerName: "Builder-Seed",
		Caste:      "builder",
		TaskID:     "1.1",
		Workflow:   "build",
		Phase:      1,
		Wave:       1,
		Root:       tmpDir,
	}
	result := codex.DispatchResult{
		WorkerName: "Builder-Seed",
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: "Builder-Seed",
			Caste:      "builder",
			TaskID:     "1.1",
			Status:     "completed",
			// The sentinel goes in exactly ONE rendered field. Seeding it in
			// two makes a single correct delivery look like a duplicate.
			Summary: "seeded prior work",
			Handoff: codex.WorkerHandoff{
				ChangedFiles:           []string{filepath.Join(tmpDir, "cmd", "seeded.go")},
				CommandsRun:            []string{"go test ./cmd -run TestSeeded"},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{sentinel},
				Freshness:              now,
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("persist handoff: %v", err)
	}
	if section := renderWorkerHandoffSection("build", phase.ID, ""); !strings.Contains(section, sentinel) {
		t.Fatalf("fixture is not exercising anything: the seeded handoff does not render, so nothing below can detect a duplicate.\nrendered: %q", section)
	}

	return tmpDir, phase, state
}

func planOnlyDispatchesForBriefTests(t *testing.T, root string, phase colony.Phase, state colony.ColonyState) []codexBuildDispatch {
	t.Helper()
	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, nil, colony.VerificationDepthStandard)
	if len(dispatches) < 2 {
		t.Fatalf("expected 2+ planned dispatches from a 2-task phase, got %d", len(dispatches))
	}
	for i := range dispatches {
		dispatches[i].Status = "planned"
	}
	dispatches, err := ensureUniqueBuildDispatchNames(dispatches, phase.ID)
	if err != nil {
		t.Fatalf("ensureUniqueBuildDispatchNames: %v", err)
	}
	attachBuildDispatchContext(root, phase, dispatches, time.Now())
	return dispatches
}

// TestPlanOnlyDispatchesCarryNoHandoffSection is the regression lock for
// 190-190/CR-01.
//
// Phase 190-03 claimed prior-worker handoffs had "one home". They did not:
// gating the BRIEF's copy left attachBuildDispatchContext still populating the
// adjacent HandoffSection struct field, which ships on the wire in
// result.dispatches[] AND — via its json:"handoff_section,omitempty" tag on the
// typed manifest — in result.dispatch_manifest.dispatches[]. Every plan-only
// dispatch therefore carried a second copy of the exact "## Previous Worker
// Handoffs" content the manifest-level capsule already delivered.
//
// It shipped unnoticed because no test anywhere inspected handoff_section's
// CONTENT. This one does, by sentinel count rather than by heading name, so a
// rename cannot make it pass while the duplication returns.
//
// .claude/commands/ant/build.md tells the wrapper the capsule "is the SOLE
// source of pheromone signals and prior worker handoffs" — this test is what
// makes that sentence true rather than aspirational.
func TestPlanOnlyDispatchesCarryNoHandoffSection(t *testing.T) {
	saveGlobalsCmd(t)

	const sentinel = "SENTINEL-HANDOFF-ONE-HOME-190190"
	root, phase, state := seedHandoffColonyForBriefTests(t, sentinel)
	dispatches := planOnlyDispatchesForBriefTests(t, root, phase, state)

	for i, dispatch := range dispatches {
		if strings.TrimSpace(dispatch.HandoffSection) != "" {
			t.Fatalf("CR-01 regressed: dispatch[%d] (%s) carries a populated handoff_section (%d bytes). The manifest-level context capsule is the sole channel for prior worker handoffs; this field ships the same content a second time in result.dispatches[] and result.dispatch_manifest.dispatches[].\ngot: %q",
				i, dispatch.Name, len(dispatch.HandoffSection), dispatch.HandoffSection)
		}
	}

	// The full wire payload, exactly as a wrapper receives it: the sentinel
	// must appear once (in the capsule) and not once per dispatch.
	generatedAt := time.Now()
	manifest := buildCodexBuildManifest(root, state, phase, "", "", dispatches, generatedAt, "plan-only", nil, nil, true, colony.VerificationDepthStandard)
	if !strings.Contains(manifest.ContextCapsule, sentinel) {
		t.Fatalf("fixture broken: the capsule does not carry the seeded handoff, so a missing handoff_section would not prove one-home — it would prove no-home")
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if n := strings.Count(string(manifestJSON), sentinel); n != 1 {
		t.Fatalf("CR-01 regressed: prior-handoff sentinel appears %d time(s) in the plan-only manifest JSON, want exactly 1 (the capsule). More than one means a dispatch-level field is shipping the same handoff content again.", n)
	}

	// codexBuildDispatchMaps is the other serialization path (result.dispatches[]).
	entries := codexBuildDispatchMaps(dispatches)
	for i, entry := range entries {
		if _, ok := entry["handoff_section"]; ok {
			t.Fatalf("CR-01 regressed: result.dispatches[%d] still emits a handoff_section key", i)
		}
	}
}

// TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs is the regression lock
// for 190-190/WR-01.
//
// The direct build path cleared build/phase-N/worker-briefs/ before writing;
// the plan-only path — the one phase 190-01 actually added brief-file writing
// to — never did. Brief filenames are "{dispatch.Name}.md" and dispatch names
// hash the task's goal text, so editing a task between two ordinary --plan-only
// runs orphaned the first run's files permanently. Nothing else ever pruned
// that directory: clearActiveColonyRuntimeFiles (entomb/abandon) and `aether
// init`'s sweep both leave .aether/data/build/ alone.
//
// This drives runCodexBuildPlanOnly END TO END rather than calling the
// cleanup helper directly. A test that called the helper itself passed even
// with the call site deleted — it would have proved the helper works while
// the path that needs it never ran it, which is the exact failure mode this
// repo's definition of done exists to prevent.
//
// It asserts the invariant (the directory holds this manifest's briefs and
// nothing else) rather than a file count, so it still fails if some future
// change alters how many briefs a phase produces.
func TestPlanOnlyRerunDoesNotAccumulateStaleWorkerBriefs(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	briefDir := filepath.Join(store.BasePath(), "build", "phase-1", "worker-briefs")

	planOnce := func() map[string]bool {
		t.Helper()
		result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
		if err != nil {
			t.Fatalf("runCodexBuildPlanOnly: %v", err)
		}
		manifest := result["dispatch_manifest"].(codexBuildManifest)
		names := map[string]bool{}
		for _, dispatch := range manifest.Dispatches {
			names[dispatch.Name+".md"] = true
		}
		if len(names) == 0 {
			t.Fatal("plan-only run produced no dispatches; nothing to write or go stale")
		}
		return names
	}

	first := planOnce()

	// An ordinary edit to a task's wording between two --plan-only runs. This
	// is what changes the deterministic dispatch-name hashes, and re-running
	// plan-only without --force is explicitly supported (the idle-plan-only
	// auto-supersede path exists so it does not need one).
	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.Plan.Phases[0].Tasks[0].Goal = "Write durable evidence, reworded so the dispatch name hash changes"
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	second := planOnce()

	if !anyDisjointBriefName(first, second) {
		t.Skip("rewording did not change any dispatch name on this build; the accumulation this test guards cannot be exercised")
	}

	onDisk, err := os.ReadDir(briefDir)
	if err != nil {
		t.Fatalf("read worker-briefs dir: %v", err)
	}
	for _, entry := range onDisk {
		if !second[entry.Name()] {
			t.Fatalf("WR-01 regressed: %q is left over from a previous --plan-only run — it belongs to no dispatch in the current manifest, and nothing else ever prunes build/phase-N/worker-briefs/", entry.Name())
		}
	}
	if len(onDisk) != len(second) {
		t.Fatalf("WR-01 regressed: worker-briefs holds %d file(s) for a manifest of %d dispatch(es)", len(onDisk), len(second))
	}
}

func anyDisjointBriefName(first, second map[string]bool) bool {
	for name := range first {
		if !second[name] {
			return true
		}
	}
	return false
}

// TestPrintBriefGateCatchesAbsentSectionNotJustDuplicated is the regression
// lock for 190-190/WR-02.
//
// duplicatedBriefSections only ever recorded a heading seen MORE than once, so
// a change that stopped delivering an active pheromone signal or a stored
// handoff to either channel left --print-brief exiting 0 — identical to a
// healthy build — while its own error text promised "must appear exactly
// once". A CI step gating on exit code could catch a duplicate and never a
// silent drop.
//
// The eviction case is separated deliberately: a section the token budget
// dropped is explicable and warned about, not failed on, or --print-brief
// would cry wolf on any context-heavy colony.
func TestPrintBriefGateCatchesAbsentSectionNotJustDuplicated(t *testing.T) {
	saveGlobalsCmd(t)

	expected := []briefExpectedSection{
		{Heading: "Pheromone Signals", SectionName: "pheromones", Expected: func() bool { return true }},
		{Heading: "Previous Worker Handoffs", SectionName: "worker_handoffs", Expected: func() bool { return true }},
	}

	// 1. Both delivered — healthy.
	healthy := "## Pheromone Signals\n\n- [FOCUS] here\n\n## Previous Worker Handoffs\n\n### Builder-1\n"
	if missing, evicted := absentBriefSections(healthy, expected, nil); len(missing) != 0 || len(evicted) != 0 {
		t.Fatalf("healthy context reported missing=%v evicted=%v, want neither", missing, evicted)
	}

	// 2. Silently dropped, no eviction on record — this is the case that used
	//    to exit 0.
	dropped := "## Pheromone Signals\n\n- [FOCUS] here\n"
	missing, evicted := absentBriefSections(dropped, expected, nil)
	if len(missing) != 1 || missing[0] != "Previous Worker Handoffs" {
		t.Fatalf("WR-02 regressed: a section with live source data and no heading in the assembled context reported missing=%v, want exactly [Previous Worker Handoffs] — this is the silent drop the gate could not see", missing)
	}
	if len(evicted) != 0 {
		t.Fatalf("a drop with no trim ledger entry must not be excused as a budget eviction; got evicted=%v", evicted)
	}

	// 3. Same absence, but the capsule's trim ledger accounts for it: warn,
	//    do not fail.
	missing, evicted = absentBriefSections(dropped, expected, []string{"worker_handoffs"})
	if len(missing) != 0 {
		t.Fatalf("a budget-evicted section must not be reported as a delivery defect; got missing=%v", missing)
	}
	if len(evicted) != 1 || evicted[0] != "Previous Worker Handoffs" {
		t.Fatalf("evicted=%v, want exactly [Previous Worker Handoffs]", evicted)
	}

	// 4. No source data — an empty set is not a missing section.
	noData := []briefExpectedSection{
		{Heading: "Pheromone Signals", SectionName: "pheromones", Expected: func() bool { return false }},
	}
	if missing, evicted := absentBriefSections("", noData, nil); len(missing) != 0 || len(evicted) != 0 {
		t.Fatalf("a section with no underlying data must not be reported at all; got missing=%v evicted=%v", missing, evicted)
	}
}

// TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker proves the absence
// gate is WIRED, not merely present.
//
// The unit test above exercises absentBriefSections; this one drives
// printWorkerBriefs on a real colony with a stored handoff and injects a
// capsule that has lost the section. Deleting the gate from printWorkerBriefs
// makes this test pass a build it should have failed, so it fails instead.
//
// It also asserts the eviction case does NOT fail the command: a section the
// token budget dropped is warned about, because failing on it would make
// --print-brief unusable on any context-heavy colony.
func TestPrintBriefFailsWhenAnExpectedSectionReachesNoWorker(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)

	// Seed a handoff so "## Previous Worker Handoffs" is genuinely expected.
	if err := persistDispatchWorkerHandoff(
		codex.WorkerDispatch{WorkerName: "Builder-Seed", Caste: "builder", TaskID: "1.1", Workflow: "build", Phase: 1, Wave: 1, Root: root},
		codex.DispatchResult{WorkerName: "Builder-Seed", Status: "completed", WorkerResult: &codex.WorkerResult{
			WorkerName: "Builder-Seed", Caste: "builder", TaskID: "1.1", Status: "completed", Summary: "seeded prior work",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"pick up where this left off"},
				Freshness:              time.Now().Format(time.RFC3339),
			},
		}},
	); err != nil {
		t.Fatalf("persist handoff: %v", err)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	state.CurrentPhase = 1
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if strings.TrimSpace(renderWorkerHandoffSection("build", 1, "")) == "" {
		t.Fatal("fixture is not exercising anything: no handoff renders, so no section is expected and the gate has nothing to catch")
	}

	restore := resolveBriefContextCapsule
	t.Cleanup(func() { resolveBriefContextCapsule = restore })

	// Long enough to clear the capsule's own minimum-length guard, and
	// deliberately carrying no "## Previous Worker Handoffs" heading.
	starved := "## Colony Context\n\n" + strings.Repeat("grounding prose that is not a handoff. ", 12)

	// 1. Absent with nothing on the trim ledger — a silent drop. Must fail.
	resolveBriefContextCapsule = func() (string, []string) { return starved, nil }
	err = printWorkerBriefs(root, 1, nil, "", codexBuildOptions{})
	if err == nil {
		t.Fatal("WR-02 regressed: --print-brief exited 0 while a stored handoff reached no worker through any channel. The gate that catches this is either missing from printWorkerBriefs or not reached.")
	}
	if !strings.Contains(err.Error(), "Previous Worker Handoffs") {
		t.Fatalf("gate fired but did not name the undelivered section, so a reader cannot act on it: %v", err)
	}

	// 2. Same absence, accounted for by the capsule's trim ledger. Must warn,
	//    not fail — otherwise every context-heavy colony fails this command.
	resolveBriefContextCapsule = func() (string, []string) { return starved, []string{"worker_handoffs"} }
	if err := printWorkerBriefs(root, 1, nil, "", codexBuildOptions{}); err != nil {
		t.Fatalf("a budget-evicted section must be warned about, not failed on: %v", err)
	}
}

// Ported from the parallel 190-04 fix branch (rescue-190-04): the native/direct
// dispatch path must deliver stored handoff content exactly once. The wrapper
// plan-only path is covered by TestPlanOnlyDispatchesCarryNoHandoffSection above;
// this is its sibling for the other delivery route (the zero-vs-once trap).
func TestNativeDispatchHandoffStaysExactlyOnceViaCapsule(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "Prove the native dispatch path keeps handoff delivery to exactly once"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Native dispatch handoff",
					Description:     "Native/direct dispatch must keep handoff delivery to exactly once",
					Status:          colony.PhaseReady,
					SuccessCriteria: []string{"Handoff content reaches the worker exactly once"},
				},
			},
		},
	})

	recent := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	handoffs := workerHandoffFile{
		Entries: []workerHandoffRecord{
			{
				ID:                 "native-path-fixture-1",
				Workflow:           "build",
				Phase:              1,
				WorkerName:         "PriorFixtureWorker-1",
				Status:             "completed",
				VerificationStatus: "pass",
				Summary:            "sentinel-native-handoff-probe",
				Freshness:          recent,
			},
		},
	}
	if err := store.SaveJSON(workerHandoffsPath, handoffs); err != nil {
		t.Fatalf("failed to save worker handoffs: %v", err)
	}

	phase := colony.Phase{ID: 1, Name: "Native dispatch handoff"}
	dispatches := []codexBuildDispatch{{Name: "NativeWorker-1", Caste: "builder", Task: "do the thing"}}
	// attachBuildDispatchContext (the ONLY setter of HandoffSection) is
	// deliberately never called here -- this mirrors writeCodexBuildArtifacts,
	// which never calls it on the direct/native path either.

	// executeCodexBuildDispatches marks the spawn tree entry "starting" as
	// its first move; the real runCodexBuildWithOptions flow pre-seeds this
	// entry via spawnTree.RecordSpawn before dispatch (cmd/codex_build.go:3053),
	// so a direct call here must seed it the same way.
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "NativeWorker-1", "do the thing", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}

	invoker := &codex.FakeInvoker{}
	results, _, _, err := executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), invoker, colony.ModeInRepo, 0, 3, false, nil)
	if err != nil {
		t.Fatalf("executeCodexBuildDispatches failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one dispatch result")
	}

	capsule := resolveCodexWorkerContext()
	if !strings.Contains(capsule, "sentinel-native-handoff-probe") {
		t.Fatalf("expected the stored handoff to reach the native path's own capsule:\n%s", capsule)
	}
	if strings.TrimSpace(results[0].HandoffSection) != "" {
		t.Fatalf("native/direct dispatch path must never populate HandoffSection (attachBuildDispatchContext is not on this path) -- got %q", results[0].HandoffSection)
	}
	assembled := codex.AssembleHostedPrompt(capsule, results[0].HandoffSection, "", "", "task brief")
	if n := strings.Count(assembled, "## Previous Worker Handoffs"); n != 1 {
		t.Fatalf("native dispatch's assembled prompt has %d \"## Previous Worker Handoffs\" headings, want exactly 1 (via the capsule alone):\n%s", n, assembled)
	}
	if n := strings.Count(assembled, "sentinel-native-handoff-probe"); n != 1 {
		t.Fatalf("native dispatch's assembled prompt carries the stored handoff's own text %d times, want exactly 1:\n%s", n, assembled)
	}
}
