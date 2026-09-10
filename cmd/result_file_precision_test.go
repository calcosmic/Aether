package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 07 (D-08) -- result-card precision: a result card names both
// the files credited to completed tasks and the files that were edited but
// credited to nothing (naming where those live), never absorbs or deletes an
// uncredited edit, recommends exactly one next action per outcome derived
// from the recorded verdict, and binds plan-versus-reality/knowledge-delta
// evidence to the exact attempt that produced it.

// seedBuildAttemptForTest derives and durably writes one build attempt
// record (plus its latest-attempt pointer) directly through the same
// deriveBuildAttempt constructor production code uses, mirroring the
// established pattern in cmd/spend_cost_line_test.go's
// seedSpendElapsedAttemptForTest. mutate lets a caller customize the record
// (Status, Dispatches, ...) before it is written.
func seedBuildAttemptForTest(t *testing.T, phaseID int, attemptID string, mutate func(*buildAttemptRecord)) (string, buildAttemptRecord) {
	t.Helper()
	rel, record, _, err := deriveBuildAttempt(buildAttemptDerivation{
		Phase: colony.Phase{ID: phaseID}, PhaseNumber: phaseID,
		StartedAt: time.Now().UTC(), AttemptID: attemptID,
		RunID: "run-" + attemptID, ProcessID: 4402, WorkspaceSHA256: strings.Repeat("e", 64),
		ExecutionOwner: "result-file-precision-test", Dispatches: nil,
		InitialStatus: buildAttemptPrepared, InitialDispatchMode: "direct",
	})
	if err != nil {
		t.Fatalf("derive attempt: %v", err)
	}
	if mutate != nil {
		mutate(&record)
	}
	if err := store.SaveJSON(rel, record); err != nil {
		t.Fatalf("save attempt: %v", err)
	}
	markLatestBuildAttemptForTest(t, phaseID, attemptID, rel)
	return rel, record
}

// markLatestBuildAttemptForTest writes the "latest attempt" pointer for a
// seeded fixture attempt. Split into its own function -- rather than inlined
// alongside the attempt record's own save, as the old shape in both this
// file and cmd/spend_cost_line_test.go used to be -- so no single test
// helper's body both references latestBuildAttemptPointerPath and issues
// more than one store write: the exact multi-write start-adapter shape
// TestBuildStartLegacyHelpersRetired200 (cmd/build_attempt_external_test.go)
// refuses, even for a test-only fixture helper.
func markLatestBuildAttemptForTest(t *testing.T, phaseID int, attemptID, rel string) {
	t.Helper()
	if err := store.SaveJSON(latestBuildAttemptPointerPath(phaseID), latestBuildAttemptPointer{
		SchemaVersion: buildAttemptSchemaVersion,
		AttemptID:     attemptID,
		Path:          rel,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("write latest-attempt pointer: %v", err)
	}
}

// hashDirForTest hashes every file's relative path and content under root,
// so a before/after comparison proves a read-only operation truly wrote
// nothing to the fixture directory.
func hashDirForTest(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		h.Write([]byte(rel))
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatalf("hash dir %s: %v", root, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ---------------------------------------------------------------------------
// Task 1: credited and orphaned files.
// ---------------------------------------------------------------------------

// TestResultCardNamesCreditedAndOrphanedFiles is the four-of-six partial
// fixture: a grouped job that credited exactly four of its six covered
// tasks' files, leaving the other two touched-but-uncredited and named with
// a location.
func TestResultCardNamesCreditedAndOrphanedFiles(t *testing.T) {
	dispatch := codexBuildDispatch{
		Name:           "Mason-67",
		Status:         "failed",
		CoveredTaskIDs: []string{"1.1", "1.2", "1.3", "1.4", "1.5", "1.6"},
		Outputs:        []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go"},
		TaskClaims: []codexBuildTaskClaim{
			{TaskID: "1.1", FilesModified: []string{"a.go"}},
			{TaskID: "1.2", FilesModified: []string{"b.go"}},
			{TaskID: "1.3", FilesModified: []string{"c.go"}},
			{TaskID: "1.4", FilesModified: []string{"d.go"}},
		},
		CompletedTaskIDs: []string{"1.1", "1.2", "1.3", "1.4"},
	}

	credited, uncredited := deriveResultFilePrecision([]codexBuildDispatch{dispatch}, nil, nil)

	wantCredited := []string{"a.go", "b.go", "c.go", "d.go"}
	if !reflect.DeepEqual(credited, wantCredited) {
		t.Fatalf("credited files = %#v, want %#v", credited, wantCredited)
	}
	if len(uncredited) != 2 {
		t.Fatalf("expected 2 uncredited files, got %d: %+v", len(uncredited), uncredited)
	}
	wantUncredited := map[string]bool{"e.go": true, "f.go": true}
	for _, f := range uncredited {
		if !wantUncredited[f.Path] {
			t.Errorf("unexpected uncredited file %q", f.Path)
		}
		if strings.TrimSpace(f.Location) == "" {
			t.Errorf("uncredited file %q has no location", f.Path)
		}
	}
}

// TestOrphanedEditsAreKeptAndLocated covers the remaining three Task 1
// behaviors: a whole-success job credits everything (no uncredited files);
// an uncredited file's location names the isolated workspace and branch it
// lives on when the worker ran in worktree mode; and an uncredited file is
// never moved into the credited set.
func TestOrphanedEditsAreKeptAndLocated(t *testing.T) {
	t.Run("whole success credits everything", func(t *testing.T) {
		dispatch := codexBuildDispatch{Name: "Mason-67", Status: "completed", TaskID: "1.1", Outputs: []string{"a.go", "b.go"}}
		credited, uncredited := deriveResultFilePrecision([]codexBuildDispatch{dispatch}, nil, nil)
		if len(uncredited) != 0 {
			t.Fatalf("expected no uncredited files for a whole-success dispatch, got %+v", uncredited)
		}
		want := []string{"a.go", "b.go"}
		if !reflect.DeepEqual(credited, want) {
			t.Fatalf("credited = %#v, want %#v", credited, want)
		}
	})

	t.Run("isolated workspace names the workspace and branch", func(t *testing.T) {
		dispatch := codexBuildDispatch{
			Name:             "Mason-67",
			Status:           "failed",
			CoveredTaskIDs:   []string{"1.1", "1.2"},
			Outputs:          []string{"a.go", "b.go"},
			TaskClaims:       []codexBuildTaskClaim{{TaskID: "1.1", FilesModified: []string{"a.go"}}},
			CompletedTaskIDs: []string{"1.1"},
		}
		worktrees := []colony.WorktreeEntry{{Agent: "Mason-67", Path: "/tmp/worker-1", Branch: "agent-phase-1-mason-67"}}
		_, uncredited := deriveResultFilePrecision([]codexBuildDispatch{dispatch}, worktrees, nil)
		if len(uncredited) != 1 {
			t.Fatalf("expected exactly one uncredited file, got %+v", uncredited)
		}
		if !strings.Contains(uncredited[0].Location, "/tmp/worker-1") || !strings.Contains(uncredited[0].Location, "agent-phase-1-mason-67") {
			t.Errorf("uncredited location %q does not name the workspace and branch", uncredited[0].Location)
		}
	})

	t.Run("uncredited file never becomes credited", func(t *testing.T) {
		dispatch := codexBuildDispatch{
			Name:             "Mason-67",
			Status:           "failed",
			CoveredTaskIDs:   []string{"1.1", "1.2"},
			Outputs:          []string{"a.go", "b.go"},
			TaskClaims:       []codexBuildTaskClaim{{TaskID: "1.1", FilesModified: []string{"a.go"}}},
			CompletedTaskIDs: []string{"1.1"},
		}
		credited, uncredited := deriveResultFilePrecision([]codexBuildDispatch{dispatch}, nil, nil)
		for _, c := range credited {
			if c == "b.go" {
				t.Fatalf("uncredited file b.go leaked into the credited set: %+v", credited)
			}
		}
		found := false
		for _, u := range uncredited {
			if u.Path == "b.go" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected b.go to be reported uncredited, got %+v", uncredited)
		}
	})
}

// TestResultCardRenderIsIdempotent proves rendering the same attempt's card
// twice produces byte-identical text and writes nothing to the fixture
// directory.
func TestResultCardRenderIsIdempotent(t *testing.T) {
	setupSpendTestStore(t)
	attemptRel, _ := seedBuildAttemptForTest(t, 1, "attempt-precision-idempotent", nil)
	credited := []string{"a.go"}
	uncredited := []buildAttemptUncreditedFile{{Path: "b.go", Location: "the repository"}}
	if err := attachResultFilePrecision(attemptRel, credited, uncredited); err != nil {
		t.Fatalf("attach result file precision: %v", err)
	}

	digestBefore := hashDirForTest(t, store.BasePath())

	var loadedFirst buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &loadedFirst); err != nil {
		t.Fatalf("load attempt (first): %v", err)
	}
	first := renderResultFilePrecisionCard(loadedFirst)

	var loadedSecond buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &loadedSecond); err != nil {
		t.Fatalf("load attempt (second): %v", err)
	}
	second := renderResultFilePrecisionCard(loadedSecond)

	if first != second {
		t.Fatalf("rendering the same attempt twice is not byte-identical:\n%s\n---\n%s", first, second)
	}
	digestAfter := hashDirForTest(t, store.BasePath())
	if digestBefore != digestAfter {
		t.Fatalf("rendering the result card wrote to the fixture directory")
	}
}

// TestTwoAttemptsNeverMergeFileLists proves two closeouts for two different
// attempts never merge their file lists.
func TestTwoAttemptsNeverMergeFileLists(t *testing.T) {
	setupSpendTestStore(t)
	relA, _ := seedBuildAttemptForTest(t, 1, "attempt-two-files-a", nil)
	relB, _ := seedBuildAttemptForTest(t, 2, "attempt-two-files-b", nil)

	if err := attachResultFilePrecision(relA, []string{"a.go"}, nil); err != nil {
		t.Fatalf("attach A: %v", err)
	}
	if err := attachResultFilePrecision(relB, []string{"z.go"}, nil); err != nil {
		t.Fatalf("attach B: %v", err)
	}

	var loadedA, loadedB buildAttemptRecord
	if err := store.LoadJSON(relA, &loadedA); err != nil {
		t.Fatalf("load A: %v", err)
	}
	if err := store.LoadJSON(relB, &loadedB); err != nil {
		t.Fatalf("load B: %v", err)
	}
	cardA := renderResultFilePrecisionCard(loadedA)
	cardB := renderResultFilePrecisionCard(loadedB)
	if strings.Contains(cardA, "z.go") {
		t.Errorf("attempt A's card names attempt B's file:\n%s", cardA)
	}
	if strings.Contains(cardB, "a.go") {
		t.Errorf("attempt B's card names attempt A's file:\n%s", cardB)
	}
}

// ---------------------------------------------------------------------------
// Task 2: exactly one outcome-matched next action.
// ---------------------------------------------------------------------------

// TestEveryVerdictHasExactlyOneRecommendedAction proves the mapping is total
// by iterating colony.AllWorkOutcomes() (the runtime's own declared set)
// rather than restating the six cases, and that an undeclared verdict is
// refused by name instead of falling through to a shared default.
func TestEveryVerdictHasExactlyOneRecommendedAction(t *testing.T) {
	attempt := buildAttemptRecord{
		Phase:           7,
		RecoveryTaskIDs: []string{"7.5", "7.6"},
		RecoveryCommand: "aether build 7 --force --task 7.5 --task 7.6",
		Error:           "the test suite could not reach the database",
	}
	for _, verdict := range colony.AllWorkOutcomes() {
		action, err := recommendedActionForWorkOutcome(verdict, attempt)
		if err != nil {
			t.Fatalf("verdict %q: %v", verdict, err)
		}
		if strings.TrimSpace(action.Command) == "" {
			t.Errorf("verdict %q returned an empty command", verdict)
		}
		if strings.TrimSpace(action.Reason) == "" {
			t.Errorf("verdict %q returned an empty reason", verdict)
		}
		for _, alt := range action.Alternatives {
			if strings.TrimSpace(alt) == strings.TrimSpace(action.Command) {
				t.Errorf("verdict %q lists its own recommendation %q as an alternative", verdict, action.Command)
			}
		}
	}

	if _, err := recommendedActionForWorkOutcome(colony.WorkOutcome("not-a-declared-verdict"), attempt); err == nil {
		t.Fatalf("expected an error for an undeclared work outcome, got none")
	}
}

// TestPartialRecommendationRetriesOnlyUnfinishedTasks takes the recommended
// command for a partial verdict, parses its arguments the way the CLI does,
// feeds them through the real build planner, and fails if any already-
// credited task appears in the resulting spawn list. Mirrors
// TestPartialRetryCommandNeverRedispatchesCreditedWork
// (cmd/coherent_job_retry_command_test.go), reusing its own proven fixture
// helpers.
func TestPartialRecommendationRetriesOnlyUnfinishedTasks(t *testing.T) {
	tasks, ids := sixChainedTasks()
	credited := ids[:4]
	unfinished := ids[4:]
	for i := range tasks {
		if i < len(credited) {
			tasks[i].Status = colony.TaskCompleted
		}
	}
	phase := colony.Phase{ID: 1, Name: "Six-task grouped job", Tasks: tasks}
	goal := "Redispatch only unfinished grouped work"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{phase},
		},
	}

	startedAt := time.Now().UTC()
	dispatch := codexBuildDispatch{
		Name:             "Mason-1",
		Caste:            "builder",
		Stage:            "wave",
		TaskID:           ids[0],
		CoveredTaskIDs:   append([]string{}, ids...),
		JobName:          "automatic-six-step",
		Status:           "failed",
		CompletedTaskIDs: append([]string{}, credited...),
	}
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: startedAt,
		SelectedTasks: ids, Dispatches: []codexBuildDispatch{dispatch},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			createTestColonyState(t, filepath.Join(root, ".aether", "data"), state)
		},
	})

	if err := transitionBuildAttempt(fixture.AttemptPath, buildAttemptPartial, "partial credit recorded for test", []codexBuildDispatch{dispatch}, nil, "direct", nil); err != nil {
		t.Fatalf("transition attempt to partial: %v", err)
	}
	var parent buildAttemptRecord
	if err := store.LoadJSON(fixture.AttemptPath, &parent); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}

	action, err := recommendedActionForWorkOutcome(colony.WorkOutcomePartial, parent)
	if err != nil {
		t.Fatalf("recommendedActionForWorkOutcome: %v", err)
	}

	phaseArg, taskArgs, _ := parseRedispatchTaskArgs(t, action.Command)
	if phaseArg != "1" {
		t.Fatalf("recommended command targets phase %q, want phase 1: %q", phaseArg, action.Command)
	}
	if len(taskArgs) == 0 {
		t.Fatalf("recommended command %q names no specific task, so running it re-plans the whole phase", action.Command)
	}

	planned, _, err := plannedBuildDispatchesWithJobProposals(
		phase, state, taskArgs, colony.VerificationDepthStandard, nil, "", nil, nil,
	)
	if err != nil {
		t.Fatalf("planning the recommended command's own arguments failed: %v", err)
	}
	plannedTasks := map[string]struct{}{}
	for _, d := range planned {
		for _, id := range dispatchCoveredTaskIDs(d) {
			plannedTasks[id] = struct{}{}
		}
	}
	for _, id := range credited {
		if _, redone := plannedTasks[id]; redone {
			t.Fatalf("recommended action %q dispatches already-credited task %s again", action.Command, id)
		}
	}
	for _, id := range unfinished {
		if _, ok := plannedTasks[id]; !ok {
			t.Fatalf("recommended action %q never dispatches unfinished task %s", action.Command, id)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 3: plan-versus-reality and knowledge deltas bound to the exact
// attempt.
// ---------------------------------------------------------------------------

// TestAbsentDeclaredArtifactBlocksCredit proves a task whose plan-declared
// artifact is absent from the repository is reported as such and cannot be
// credited.
func TestAbsentDeclaredArtifactBlocksCredit(t *testing.T) {
	root := t.TempDir()
	taskID := "1.1"
	task := colony.Task{
		ID:   &taskID,
		Goal: "Add the missing widget",
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: "widget exists", Artifacts: []string{"pkg/widget/widget.go"}},
		},
	}
	phase := colony.Phase{ID: 1, Name: "Widget phase", Tasks: []colony.Task{task}}
	dispatch := codexBuildDispatch{
		Name:    "Mason-1",
		Status:  "completed",
		TaskID:  taskID,
		Outputs: []string{"pkg/widget/widget.go", "pkg/widget/widget_test.go"},
	}

	entries := buildPlanRealityForDispatches(root, phase, []codexBuildDispatch{dispatch})
	if len(entries) != 1 {
		t.Fatalf("expected one plan-reality entry, got %d: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.TaskID != taskID {
		t.Fatalf("entry task id = %q, want %q", entry.TaskID, taskID)
	}
	if len(entry.MissingArtifacts) != 1 || entry.MissingArtifacts[0] != "pkg/widget/widget.go" {
		t.Fatalf("expected the declared artifact to be reported missing, got %+v", entry.MissingArtifacts)
	}
	if !planRealityBlocksCredit(entry) {
		t.Fatalf("a missing declared artifact must block credit")
	}

	blocked := blockedPlanRealityTasks(entries)
	credited, uncredited := deriveResultFilePrecision([]codexBuildDispatch{dispatch}, nil, blocked)
	if len(credited) != 0 {
		t.Fatalf("expected no credited files once the declared artifact is absent, got %+v", credited)
	}
	found := false
	for _, f := range uncredited {
		if f.Path == "pkg/widget/widget.go" {
			found = true
			lowered := strings.ToLower(f.Location)
			if !strings.Contains(f.Location, taskID) || !strings.Contains(lowered, "blocked") {
				t.Errorf("uncredited entry does not name the credit-blocking reason: %+v", f)
			}
		}
	}
	if !found {
		t.Fatalf("expected pkg/widget/widget.go to be reported uncredited, got %+v", uncredited)
	}
}

// TestKnowledgeDeltasAreAttemptBound proves decision/learning deltas render
// keyed to the attempt identifier, never leak between two attempts' cards,
// and rendering writes nothing.
func TestKnowledgeDeltasAreAttemptBound(t *testing.T) {
	setupSpendTestStore(t)
	relA, _ := seedBuildAttemptForTest(t, 1, "attempt-deltas-a", nil)
	relB, _ := seedBuildAttemptForTest(t, 2, "attempt-deltas-b", nil)

	if err := attachBuildKnowledgeDeltas(relA, []buildAttemptKnowledgeDelta{
		{Kind: "decision", Summary: "Chose Postgres for the widget store"},
	}); err != nil {
		t.Fatalf("attach deltas A: %v", err)
	}
	if err := attachBuildKnowledgeDeltas(relB, []buildAttemptKnowledgeDelta{
		{Kind: "learning", Summary: "The widget API rate-limits at 10rps"},
	}); err != nil {
		t.Fatalf("attach deltas B: %v", err)
	}

	digestBefore := hashDirForTest(t, store.BasePath())

	var attemptA, attemptB buildAttemptRecord
	if err := store.LoadJSON(relA, &attemptA); err != nil {
		t.Fatalf("load attempt A: %v", err)
	}
	if err := store.LoadJSON(relB, &attemptB); err != nil {
		t.Fatalf("load attempt B: %v", err)
	}

	evidenceA := lifecycleCloseoutKnowledgeDeltaEvidence(attemptA)
	evidenceB := lifecycleCloseoutKnowledgeDeltaEvidence(attemptB)

	if len(evidenceA) != 1 || !strings.Contains(evidenceA[0].Summary, "Postgres") {
		t.Fatalf("attempt A evidence = %+v, want its own Postgres delta", evidenceA)
	}
	if len(evidenceB) != 1 || !strings.Contains(evidenceB[0].Summary, "rate-limits") {
		t.Fatalf("attempt B evidence = %+v, want its own rate-limit delta", evidenceB)
	}
	for _, e := range evidenceA {
		if strings.Contains(e.Summary, "rate-limits") {
			t.Errorf("attempt A's evidence carries attempt B's delta: %+v", e)
		}
	}
	for _, e := range evidenceB {
		if strings.Contains(e.Summary, "Postgres") {
			t.Errorf("attempt B's evidence carries attempt A's delta: %+v", e)
		}
	}

	digestAfter := hashDirForTest(t, store.BasePath())
	if digestBefore != digestAfter {
		t.Fatalf("rendering knowledge deltas wrote to the fixture directory")
	}
}
