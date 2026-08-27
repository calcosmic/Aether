package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestCoherentJobRetryContainsOnlyUnfinishedTasks is D-10's core fixture: a
// six-task job with four credited task IDs must yield exactly one retry job
// containing the remaining two, in their original dependency-safe order.
func TestCoherentJobRetryContainsOnlyUnfinishedTasks(t *testing.T) {
	tasks, ids := sixChainedTasks()
	phase := colony.Phase{ID: 1, Name: "Six-task retry fixture", Tasks: tasks}

	job, err := planCoherentJobRetry(phase, ids, ids[:4], "builder", "job-original")
	if err != nil {
		t.Fatalf("planCoherentJobRetry: %v", err)
	}
	if job == nil {
		t.Fatal("expected a retry job for a four-of-six credited fixture, got nil")
	}
	want := ids[4:]
	if !reflect.DeepEqual(job.TaskIDs, want) {
		t.Fatalf("retry job task IDs = %v, want exactly the unfinished %v in order", job.TaskIDs, want)
	}
	for _, id := range ids[:4] {
		for _, gotID := range job.TaskIDs {
			if gotID == id {
				t.Fatalf("retry job re-includes credited task %s, want it excluded entirely", id)
			}
		}
	}
	if job.OwnerCaste != "builder" {
		t.Fatalf("retry job owner caste = %q, want %q", job.OwnerCaste, "builder")
	}
}

// TestCoherentJobRetryTreatsCreditedDependenciesAsSatisfied proves the other
// half of D-10: an unfinished task that depends on a CREDITED task must not
// be refused for an "unmet dependency" -- the credited task's completion
// satisfies it. An unfinished-to-unfinished dependency, by contrast, must
// still be respected and ordered correctly.
func TestCoherentJobRetryTreatsCreditedDependenciesAsSatisfied(t *testing.T) {
	tasks, ids := sixChainedTasks() // 1.1 -> 1.2 -> 1.3 -> 1.4 -> 1.5 -> 1.6
	phase := colony.Phase{ID: 1, Name: "Chained retry fixture", Tasks: tasks}

	// Task 1.5 depends on credited 1.4; task 1.6 depends on unfinished 1.5.
	credited := ids[:4] // 1.1..1.4
	job, err := planCoherentJobRetry(phase, ids, credited, "builder", "job-original")
	if err != nil {
		t.Fatalf("planCoherentJobRetry with a credited dependency: %v", err)
	}
	if job == nil {
		t.Fatal("expected a retry job, got nil")
	}
	if !reflect.DeepEqual(job.TaskIDs, []string{"1.5", "1.6"}) {
		t.Fatalf("retry job task IDs = %v, want [1.5 1.6] preserving the unfinished-to-unfinished dependency order", job.TaskIDs)
	}
}

// TestCoherentJobRetryFullSuccessReturnsNoJob proves D-10's "no retry for
// full success" clause: every covered task credited means nothing to retry.
func TestCoherentJobRetryFullSuccessReturnsNoJob(t *testing.T) {
	tasks, ids := sixChainedTasks()
	phase := colony.Phase{ID: 1, Name: "Full success fixture", Tasks: tasks}

	job, err := planCoherentJobRetry(phase, ids, ids, "builder", "job-original")
	if err != nil {
		t.Fatalf("planCoherentJobRetry on full success: %v", err)
	}
	if job != nil {
		t.Fatalf("expected no retry job for a fully credited dispatch, got %+v", job)
	}
}

// TestCoherentJobRetryNamesRealCycle proves the named-hard-error path: a
// genuine cycle among the phase's own tasks must be refused by name, and
// must never produce a job.
func TestCoherentJobRetryNamesRealCycle(t *testing.T) {
	a, b := "a", "b"
	phase := colony.Phase{
		ID:   1,
		Name: "Cyclic retry fixture",
		Tasks: []colony.Task{
			{ID: &a, Goal: "Task A", Status: colony.TaskPending, DependsOn: []string{"b"}},
			{ID: &b, Goal: "Task B", Status: colony.TaskPending, DependsOn: []string{"a"}},
		},
	}

	job, err := planCoherentJobRetry(phase, []string{"a", "b"}, nil, "builder", "job-original")
	if err == nil {
		t.Fatalf("expected a named cycle error, got job=%+v", job)
	}
	if !strings.Contains(err.Error(), "cycle") && !strings.Contains(err.Error(), "dependency") {
		t.Fatalf("cycle error does not name the problem: %v", err)
	}
	if job != nil {
		t.Fatalf("a cyclic unfinished subgraph must never produce a job, got %+v", job)
	}
}

// TestCoherentJobRetryNamesMissingDependency proves the other named-hard-error
// path: an unfinished task depending on a task ID absent from the entire
// phase must be refused by name, not silently accepted.
func TestCoherentJobRetryNamesMissingDependency(t *testing.T) {
	only := "only-task"
	phase := colony.Phase{
		ID:   1,
		Name: "Missing dependency fixture",
		Tasks: []colony.Task{
			{ID: &only, Goal: "Depends on a task that does not exist", Status: colony.TaskPending, DependsOn: []string{"does-not-exist"}},
		},
	}

	job, err := planCoherentJobRetry(phase, []string{"only-task"}, nil, "builder", "job-original")
	if err == nil {
		t.Fatalf("expected a named missing-dependency error, got job=%+v", job)
	}
	if job != nil {
		t.Fatalf("a missing dependency must never produce a job, got %+v", job)
	}
}

// TestCoherentJobRetryRequiresCoveredTasks proves the retry planner refuses
// to operate on an empty covered-task list rather than silently no-op'ing.
func TestCoherentJobRetryRequiresCoveredTasks(t *testing.T) {
	phase := colony.Phase{ID: 1, Name: "Empty fixture"}
	job, err := planCoherentJobRetry(phase, nil, nil, "builder", "job-original")
	if err == nil {
		t.Fatalf("expected an error for zero covered tasks, got job=%+v", job)
	}
	if job != nil {
		t.Fatalf("expected no job for zero covered tasks, got %+v", job)
	}
}

// TestBuildAttemptChildLinksParentWithoutMutation proves the append-only half
// of D-10 at the attempt-journal layer: creating a child (retry) attempt
// leaves the parent's own attempt file byte-for-byte unchanged, while the
// child correctly records ParentAttemptID/ParentJobName and starts its own
// independent history.
func TestBuildAttemptChildLinksParentWithoutMutation(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	_ = root

	phase := colony.Phase{ID: 1, Name: "Parent-child journal proof"}
	state := colony.ColonyState{State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	parentStartedAt := time.Now().UTC()
	parentDispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", TaskID: "1.1", CoveredTaskIDs: []string{"1.1", "1.2"}, Status: "failed"}}
	parentRel, err := beginBuildAttempt(state, 1, phase, parentStartedAt, []string{"1.1", "1.2"}, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", parentDispatches)
	if err != nil {
		t.Fatalf("begin parent attempt: %v", err)
	}
	if err := transitionBuildAttempt(parentRel, buildAttemptFailed, "crashed after finishing 1.1", parentDispatches, nil, "real", fmt.Errorf("crashed after finishing 1.1")); err != nil {
		t.Fatalf("transition parent to failed: %v", err)
	}

	parentAbs := filepath.Join(dataDir, filepath.FromSlash(parentRel))
	beforeBytes, err := os.ReadFile(parentAbs)
	if err != nil {
		t.Fatalf("read parent attempt file before child creation: %v", err)
	}

	var parentBefore buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parentBefore); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}

	childStartedAt := parentStartedAt.Add(time.Second)
	retryDispatches := []codexBuildDispatch{{Name: "Mason-1-retry", Caste: "builder", TaskID: "1.2", CoveredTaskIDs: []string{"1.2"}, Status: "planned", JobName: "job-x-retry"}}
	childRel, err := beginChildBuildAttempt(state, 1, phase, childStartedAt, parentBefore.ID, "job-x", []string{"1.2"}, "", "", "", "runtime-worker-dispatch", retryDispatches)
	if err != nil {
		t.Fatalf("begin child attempt: %v", err)
	}
	if childRel == parentRel {
		t.Fatalf("child attempt reused the parent's own file %q", parentRel)
	}

	afterBytes, err := os.ReadFile(parentAbs)
	if err != nil {
		t.Fatalf("read parent attempt file after child creation: %v", err)
	}
	if !reflect.DeepEqual(beforeBytes, afterBytes) {
		t.Fatalf("parent attempt file changed after child creation:\nbefore: %s\nafter:  %s", beforeBytes, afterBytes)
	}

	var child buildAttemptRecord
	if err := store.LoadJSON(childRel, &child); err != nil {
		t.Fatalf("load child attempt: %v", err)
	}
	if child.ParentAttemptID != parentBefore.ID {
		t.Fatalf("child ParentAttemptID = %q, want %q", child.ParentAttemptID, parentBefore.ID)
	}
	if child.ParentJobName != "job-x" {
		t.Fatalf("child ParentJobName = %q, want %q", child.ParentJobName, "job-x")
	}
	if len(child.Dispatches) != 1 || child.Dispatches[0].TaskID != "1.2" {
		t.Fatalf("child dispatches = %+v, want exactly the retry dispatch for 1.2", child.Dispatches)
	}
	if len(child.History) != 1 || child.History[0].Status != buildAttemptPrepared {
		t.Fatalf("child history = %+v, want a single fresh prepared entry (append-only from its own start)", child.History)
	}

	// Parent's own dispatches/status must be untouched by the child's
	// creation, proven again through the typed record (not just raw bytes).
	var parentAfter buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parentAfter); err != nil {
		t.Fatalf("reload parent attempt: %v", err)
	}
	if !reflect.DeepEqual(parentBefore, parentAfter) {
		t.Fatalf("parent attempt record changed after child creation:\nbefore: %+v\nafter:  %+v", parentBefore, parentAfter)
	}
}

// TestLegacyBuildAttemptReadsWithoutParentFields proves backward
// compatibility: an attempt JSON written before ParentAttemptID/ParentJobName
// existed still decodes cleanly, with both fields simply empty.
func TestLegacyBuildAttemptReadsWithoutParentFields(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)

	legacyJSON := []byte(`{
		"schema_version": 1,
		"id": "attempt-legacy",
		"phase": 1,
		"status": "built",
		"dispatches": [],
		"history": [{"status": "built", "timestamp": "2026-01-01T00:00:00Z"}]
	}`)
	rel := "build/phase-1/attempts/attempt-legacy.json"
	if err := store.AtomicWrite(rel, legacyJSON); err != nil {
		t.Fatalf("seed legacy attempt file: %v", err)
	}

	var record buildAttemptRecord
	if err := store.LoadJSON(rel, &record); err != nil {
		t.Fatalf("load legacy attempt record: %v", err)
	}
	if record.ID != "attempt-legacy" || record.Status != buildAttemptBuilt {
		t.Fatalf("legacy attempt record decoded incorrectly: %+v", record)
	}
	if record.ParentAttemptID != "" || record.ParentJobName != "" {
		t.Fatalf("legacy attempt record without parent fields decoded non-empty parent info: %+v", record)
	}
}
