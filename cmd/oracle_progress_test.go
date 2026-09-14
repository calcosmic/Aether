package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// runOracleLoopForProgress drives a short real loop against the fake dispatcher
// and returns the progress lines it recorded.
func runOracleLoopForProgress(t *testing.T, iterations int) (oraclePaths, []oracleProgressEvent, int) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state := oracleStateFile{
		Version:          "1.1",
		Topic:            "progress logging",
		Scope:            "repo",
		Template:         "custom",
		Phase:            "survey",
		Iteration:        0,
		MaxIterations:    iterations,
		TargetConfidence: 60,
		StartedAt:        now,
		LastUpdated:      now,
		Status:           "active",
		ControllerPID:    os.Getpid(),
	}
	questions := make([]oracleQuestion, 0, iterations)
	for i := 1; i <= iterations; i++ {
		questions = append(questions, oracleQuestion{
			ID:                fmt.Sprintf("q%d", i),
			Text:              fmt.Sprintf("Progress question %d?", i),
			Status:            "open",
			KeyFindings:       []oracleFinding{},
			IterationsTouched: []int{},
		})
	}
	plan := oraclePlanFile{Version: "1.1", Sources: map[string]oracleSource{}, Questions: questions, CreatedAt: now, LastUpdated: now}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	result, err := runOracleCompatibility(root, []string{"run-loop"}, "", "")
	if err != nil {
		t.Fatalf("run-loop: %v", err)
	}
	ran, _ := result["iterations_run"].(int)

	return paths, readOracleProgressEvents(t, paths.ProgressPath), ran
}

func readOracleProgressEvents(t *testing.T, path string) []oracleProgressEvent {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read progress log %s: %v", path, err)
	}
	events := make([]oracleProgressEvent, 0, 8)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event oracleProgressEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("progress line is not valid JSON: %s", line)
		}
		events = append(events, event)
	}
	return events
}

func countOracleProgressEvents(events []oracleProgressEvent, kind string) int {
	n := 0
	for _, event := range events {
		if event.Event == kind {
			n++
		}
	}
	return n
}

// TestOracleLoopWritesProgressLinePerIteration asserts the relationship between
// rounds and recorded lines, not the presence of a field. If a future early
// return skips the terminal line, or an emission site is dropped, the counts
// stop matching whatever the fields are called by then.
func TestOracleLoopWritesProgressLinePerIteration(t *testing.T) {
	// The loop may stop early once it hits its confidence target, so the
	// invariant is "one recorded round per round actually run", not a fixed
	// count. That is the property that breaks if an emission site is dropped.
	_, events, iterations := runOracleLoopForProgress(t, 3)
	if iterations == 0 {
		t.Fatal("the loop reported running no iterations")
	}

	if got := countOracleProgressEvents(events, oracleProgressEventRunStart); got != 1 {
		t.Errorf("run_start recorded %d times, want exactly 1", got)
	}
	if got := countOracleProgressEvents(events, oracleProgressEventRunEnd); got != 1 {
		t.Errorf("run_end recorded %d times, want exactly 1 — every exit must record a terminal line", got)
	}

	starts := make([]oracleProgressEvent, 0, iterations)
	for _, event := range events {
		if event.Event == oracleProgressEventIterationStart {
			starts = append(starts, event)
		}
	}
	if len(starts) != iterations {
		t.Fatalf("the loop ran %d rounds but recorded %d — every round must leave a line for anyone following", iterations, len(starts))
	}
	for i, event := range starts {
		wantIteration := i + 1
		if event.Iteration != wantIteration {
			t.Errorf("round %d recorded iteration %d", i, event.Iteration)
		}
		if event.Remaining != event.MaxIterations-event.Iteration {
			t.Errorf("round %d: remaining=%d but max=%d iteration=%d", i, event.Remaining, event.MaxIterations, event.Iteration)
		}
		if strings.TrimSpace(event.Question) == "" {
			t.Errorf("round %d recorded no question — the operator cannot tell what it is investigating", i)
		}
		if event.TargetConfidence <= 0 {
			t.Errorf("round %d recorded no target confidence — the operator cannot tell how far there is to go", i)
		}
	}

	// The last round must end above where the first started, so the operator
	// sees confidence actually move.
	if events[len(events)-1].Confidence <= 0 {
		t.Errorf("run ended at %d%% confidence; the fake dispatcher answers every question, so this should have climbed", events[len(events)-1].Confidence)
	}
}

// TestOracleProgressLogWrittenInJSONOutputMode is the regression that matters.
// The loop already had a per-round renderer and nobody ever saw it: the wrapper
// always backgrounds Oracle, oracleBackgroundEnv forces AETHER_OUTPUT_MODE=json
// into the detached controller, and emitVisualProgress returns early in json
// mode. If anyone re-gates the progress writer behind visual output, this fails.
func TestOracleProgressLogWrittenInJSONOutputMode(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	_, events, iterations := runOracleLoopForProgress(t, 2)

	if len(events) == 0 {
		t.Fatal("no progress recorded in json output mode — this is exactly how the previous progress display became invisible")
	}
	if got := countOracleProgressEvents(events, oracleProgressEventIterationStart); got != iterations {
		t.Errorf("json mode recorded %d rounds for %d run", got, iterations)
	}
	if countOracleProgressEvents(events, oracleProgressEventRunEnd) != 1 {
		t.Error("json mode recorded no terminal line, so `--follow` would never know the run had ended")
	}
}

func TestOracleProgressLineNamesRoundConfidenceAndQuestion(t *testing.T) {
	line := renderOracleProgressLine(oracleProgressEvent{
		Event:            oracleProgressEventIterationStart,
		Iteration:        5,
		MaxIterations:    30,
		Remaining:        25,
		Phase:            "survey",
		QuestionID:       "q6",
		Question:         "Which files matter most to the ingest path?",
		Confidence:       35,
		TargetConfidence: 90,
		Reasoning:        "medium",
	})
	for _, want := range []string{"5/30", "survey", "q6", "35", "90", "Which files matter most"} {
		if !strings.Contains(line, want) {
			t.Errorf("progress line missing %q:\n%s", want, line)
		}
	}
	if strings.Contains(line, "\n") {
		t.Errorf("progress line must be a single line so rounds stack as scrollback:\n%q", line)
	}
}

func TestOracleStatusFollowStreamsExistingRoundsAndExitsOnRunEnd(t *testing.T) {
	saveGlobals(t)
	// followOracleProgress only ever runs inside `aether oracle ... --follow`
	// (cmd/compatibility_cmds.go:183), and root.go's PersistentPreRunE has by
	// then recorded that command's own name -- which is what lets
	// emitVisualLine stream at all. Derive the value from the real command
	// rather than inheriting whatever the previous test left behind: a follow
	// running while a quiet command is recorded is a state the runtime cannot
	// produce, and inheriting it silenced this test's entire output depending
	// on lane order.
	currentStreamingCommand = oracleCmd.Name()
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	seed := []oracleProgressEvent{
		{Event: oracleProgressEventRunStart, MaxIterations: 3, TargetConfidence: 90},
		{Event: oracleProgressEventIterationStart, Iteration: 1, MaxIterations: 3, Remaining: 2, Phase: "survey", QuestionID: "q1", Question: "First question?", Confidence: 10, TargetConfidence: 90},
		{Event: oracleProgressEventIterationStart, Iteration: 2, MaxIterations: 3, Remaining: 1, Phase: "investigate", QuestionID: "q2", Question: "Second question?", Confidence: 55, TargetConfidence: 90},
		{Event: oracleProgressEventRunEnd, Iteration: 3, MaxIterations: 3, Confidence: 92, TargetConfidence: 90, Status: "complete", StopReason: "target_reached"},
	}
	for _, event := range seed {
		appendOracleProgressEvent(paths.ProgressPath, event)
	}

	var out bytes.Buffer
	oldStdout := stdout
	stdout = &out
	t.Cleanup(func() { stdout = oldStdout })
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	done := make(chan error, 1)
	go func() { done <- followOracleProgress(root, 10*time.Millisecond) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("follow returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("follow did not exit after run_end; it must stop when the run finishes")
	}

	rendered := out.String()
	for _, want := range []string{"First question?", "Second question?", "finished after 3 rounds"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("follow output missing %q:\n%s", want, rendered)
		}
	}
}

// TestOracleStatusFollowDoesNotMutateWorkspace enforces the Definition of Done
// corollary that an inspection command must not mutate state. It also guards a
// concrete hazard: follow polls every couple of seconds, and the status path it
// must not take repairs state whenever it sees a dead controller PID.
func TestOracleStatusFollowDoesNotMutateWorkspace(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	// A run that claims to be active behind a PID that is certainly gone.
	state := oracleStateFile{
		Version: "1.1", Topic: "stale", Status: "active", Phase: "survey",
		Iteration: 2, MaxIterations: 5, OverallConfidence: 30, TargetConfidence: 90,
		ControllerPID: 999999,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
		t.Fatalf("write loop marker: %v", err)
	}
	appendOracleProgressEvent(paths.ProgressPath, oracleProgressEvent{
		Event: oracleProgressEventIterationStart, Iteration: 2, MaxIterations: 5,
		Question: "Anything?", Confidence: 30, TargetConfidence: 90,
	})

	before := hashOracleWorkspace(t, root)

	var out bytes.Buffer
	oldStdout := stdout
	stdout = &out
	t.Cleanup(func() { stdout = oldStdout })

	done := make(chan error, 1)
	go func() { done <- followOracleProgress(root, 5*time.Millisecond) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("follow returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("follow hung on a run whose controller is gone")
	}

	if after := hashOracleWorkspace(t, root); after != before {
		t.Fatal("follow modified the oracle workspace; inspection must not mutate state")
	}
}

// TestOracleStatusIsReadOnlyWithDeadController pins the same rule on status
// itself, which used to repair state as a side effect of being asked.
func TestOracleStatusIsReadOnlyWithDeadController(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	state := oracleStateFile{
		Version: "1.1", Topic: "stale", Status: "active", Phase: "survey",
		Iteration: 2, MaxIterations: 5, ControllerPID: 999999,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
		t.Fatalf("write loop marker: %v", err)
	}
	stateBefore, _ := os.ReadFile(paths.StatePath)
	markerBefore, _ := os.ReadFile(paths.LoopPath)

	result, err := oracleStatusResult(root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if result["stop_reason"] != "stale_controller" {
		t.Errorf("status reported stop_reason=%v, want stale_controller — it must still detect the dead controller", result["stop_reason"])
	}
	if repair, _ := result["state_repair_available"].(bool); !repair {
		t.Error("status did not offer the repair it declined to perform")
	}
	if result["next"] != "aether oracle recover" {
		t.Errorf("status next=%v, want `aether oracle recover`", result["next"])
	}

	stateAfter, _ := os.ReadFile(paths.StatePath)
	if !bytes.Equal(stateBefore, stateAfter) {
		t.Error("oracle status rewrote state.json; inspection must not mutate")
	}
	markerAfter, err := os.ReadFile(paths.LoopPath)
	if err != nil || !bytes.Equal(markerBefore, markerAfter) {
		t.Error("oracle status deleted the loop marker; inspection must not mutate")
	}
}

// TestOracleRecoverRepairsStaleController is the paired positive: the read-only
// rule above must not be satisfiable by removing the capability entirely.
func TestOracleRecoverRepairsStaleController(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	state := oracleStateFile{
		Version: "1.1", Topic: "stale", Status: "active", Phase: "survey",
		Iteration: 2, MaxIterations: 5, ControllerPID: 999999,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
		t.Fatalf("write loop marker: %v", err)
	}

	result, err := oracleRecoverStaleRun(root)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if repaired, _ := result["repaired"].(bool); !repaired {
		t.Fatal("recover did not repair a stale controller")
	}

	repaired, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if repaired.Status != "blocked" || repaired.StopReason != "stale_controller" {
		t.Errorf("after recover: status=%q stop_reason=%q, want blocked/stale_controller", repaired.Status, repaired.StopReason)
	}
	if _, err := os.Stat(paths.LoopPath); !os.IsNotExist(err) {
		t.Error("recover left the loop marker in place, so a new run would still refuse to start")
	}

	// A healthy run is left alone.
	live := oracleStateFile{Version: "1.1", Topic: "live", Status: "active", ControllerPID: os.Getpid()}
	if err := writeOracleStateFile(paths.StatePath, live); err != nil {
		t.Fatalf("write live state: %v", err)
	}
	result, err = oracleRecoverStaleRun(root)
	if err != nil {
		t.Fatalf("recover on live run: %v", err)
	}
	if repaired, _ := result["repaired"].(bool); repaired {
		t.Error("recover repaired a run whose controller is alive")
	}
}

func TestOracleSelftestFailsWhenNoDispatcherAvailable(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return nil }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	result, err := runOracleSelftest(root, false)
	if err == nil {
		t.Fatal("selftest reported success with no worker dispatcher; it must fail exactly where a real run would")
	}
	if failed, _ := result["failed"].(int); failed == 0 {
		t.Error("selftest recorded no failed checks despite having no dispatcher")
	}
	if !strings.Contains(err.Error(), "dispatcher") {
		t.Errorf("selftest error does not name the dispatcher: %v", err)
	}
}

func TestOracleSelftestDryRunDoesNotWrite(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	before := hashOracleWorkspace(t, root)

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	result, err := runOracleSelftest(root, true)
	if err != nil {
		t.Fatalf("selftest dry run: %v", err)
	}
	if failed, _ := result["failed"].(int); failed != 0 {
		t.Errorf("selftest dry run reported %d failures against a working dispatcher", failed)
	}
	if after := hashOracleWorkspace(t, root); after != before {
		t.Fatal("selftest --dry-run wrote to the workspace; it must cost nothing")
	}
	if _, err := os.Stat(filepath.Join(paths.Dir, ".selftest")); !os.IsNotExist(err) {
		t.Error("selftest --dry-run created a probe workspace")
	}
}

func TestOracleSelftestPassesWithWorkingInvoker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	result, err := runOracleSelftest(root, false)
	if err != nil {
		t.Fatalf("selftest against a working dispatcher failed: %v\nchecks: %#v", err, result["checks"])
	}
	checks, _ := result["checks"].([]oracleSelftestCheck)
	if len(checks) < 5 {
		t.Fatalf("selftest ran only %d checks; it should cover dispatch, the agent, the round, and its artifacts", len(checks))
	}
	for _, check := range checks {
		if !check.Passed {
			t.Errorf("check %q failed: %s", check.Name, check.Detail)
		}
	}

	// The probe workspace must not survive, or it would be archived into the
	// operator's real research history.
	if _, err := os.Stat(filepath.Join(oracleWorkspacePaths(root).Dir, ".selftest")); !os.IsNotExist(err) {
		t.Error("selftest left its probe workspace behind")
	}

	// Nor may the probe round save a research document. Selftest runs a real
	// round, so without a guard its completion finalizes like any other and
	// files an answer to a question the operator never asked.
	saved, _ := filepath.Glob(filepath.Join(root, ".aether", "research", "*.md"))
	if len(saved) > 0 {
		t.Errorf("selftest saved %d research document(s) into the operator's saved research: %v", len(saved), saved)
	}
}

// TestOracleSelftestLeavesRealResearchAlone: running a check must never cost
// the operator research they have not saved yet.
func TestOracleSelftestLeavesRealResearchAlone(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	precious := []byte("# Real research the operator has not saved yet\n")
	if err := os.WriteFile(paths.SynthesisPath, precious, 0644); err != nil {
		t.Fatalf("seed synthesis: %v", err)
	}
	wantHash := sha256.Sum256(precious)

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	t.Cleanup(func() { newOracleWorkerInvoker = originalInvoker })

	if _, err := runOracleSelftest(root, false); err != nil {
		t.Fatalf("selftest: %v", err)
	}

	after, err := os.ReadFile(paths.SynthesisPath)
	if err != nil {
		t.Fatalf("the operator's research was deleted by a selftest: %v", err)
	}
	if sha256.Sum256(after) != wantHash {
		t.Fatal("selftest overwrote the operator's existing research")
	}
}

// TestOracleFromBriefBackgroundFollowStartsTheRun pins the wrapper's flagship
// invocation. `oracle --from-brief --background --follow` has zero positional
// args, and the bare `status --follow` interception used to swallow it: the
// research never started, follow replayed the PREVIOUS run's log, and the
// command exited as if work had happened — failing soft, which is this repo's
// documented disease. The test runs the real command line end to end and
// fails unless a NEW run with the brief's topic actually started.
func TestOracleFromBriefBackgroundFollowStartsTheRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	// A previous, finished run in the workspace — the decoy the broken
	// dispatch used to follow instead of starting anything.
	paths := oracleWorkspacePaths(root)
	if err := ensureOracleWorkspace(paths); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	stale := oracleStateFile{Version: "1.1", Topic: "an old finished question", Status: "complete", Iteration: 5, MaxIterations: 5}
	if err := writeOracleStateFile(paths.StatePath, stale); err != nil {
		t.Fatalf("seed stale state: %v", err)
	}
	appendOracleProgressEvent(paths.ProgressPath, oracleProgressEvent{Event: oracleProgressEventRunEnd, Iteration: 5, MaxIterations: 5, Status: "complete"})

	if _, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "cache storage",
		CoreQuestion: "Should the local cache use SQLite or Postgres?",
		Depth:        "quick",
	}, false); err != nil {
		t.Fatalf("approve brief: %v", err)
	}

	// Stub the detached controller: record that it was asked to start, and
	// write a terminal progress line so --follow exits instead of polling.
	started := false
	originalStart := startOracleBackgroundController
	startOracleBackgroundController = func(paths oraclePaths) (int, string, error) {
		started = true
		appendOracleProgressEvent(paths.ProgressPath, newOracleProgressEvent(oracleProgressEventRunStart, oracleStateFile{MaxIterations: 5, TargetConfidence: 60}))
		appendOracleProgressEvent(paths.ProgressPath, oracleProgressEvent{Event: oracleProgressEventRunEnd, Iteration: 1, MaxIterations: 5, Status: "complete"})
		return os.Getpid(), filepath.Join(paths.Dir, "oracle.log"), nil
	}
	t.Cleanup(func() { startOracleBackgroundController = originalStart })

	rootCmd.SetArgs([]string{"oracle", "--from-brief", "--background", "--follow", "--follow-interval", "10ms"})
	done := make(chan error, 1)
	go func() { done <- rootCmd.Execute() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("command returned error: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("oracle --from-brief --background --follow hung")
	}

	if !started {
		t.Fatal("the run never started: --follow swallowed --from-brief and followed the previous run instead")
	}
	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.Topic != "cache storage" {
		t.Fatalf("workspace still holds the old run (topic %q); the brief's run was never created", state.Topic)
	}
	if state.CoreQuestion != "Should the local cache use SQLite or Postgres?" {
		t.Errorf("the approved core question did not reach the run: %q", state.CoreQuestion)
	}
}
