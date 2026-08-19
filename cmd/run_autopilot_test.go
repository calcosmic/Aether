package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// hashDirContents returns a stable digest of every file under dir — path,
// size, and content — so a test can assert an inspection command wrote
// nothing.
func hashDirContents(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		fmt.Fprintf(h, "%s:%d:", rel, info.Size())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatalf("hash dir %s: %v", dir, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// completingRunInvoker embeds FakeInvoker's deterministic completed results
// but is a distinct type, so the build records dispatch mode "real" — and its
// builders do real (tiny) work: each writes a file into the workspace and
// claims it, so continue's claim verification cross-checks genuine paths and
// the autopilot loop advances the way a real provider run does.
type completingRunInvoker struct{ codex.FakeInvoker }

func (c *completingRunInvoker) completeWithEvidence(result codex.WorkerResult, config codex.WorkerConfig) codex.WorkerResult {
	if config.Caste != "builder" {
		return result
	}
	rel := filepath.Join("fixture-output", config.WorkerName+".txt")
	abs := filepath.Join(config.Root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return result
	}
	if err := os.WriteFile(abs, []byte("fixture work\n"), 0644); err != nil {
		return result
	}
	result.FilesCreated = []string{rel}
	return result
}

func (c *completingRunInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result, err := c.FakeInvoker.Invoke(ctx, config)
	if err != nil {
		return result, err
	}
	return c.completeWithEvidence(result, config), nil
}

func (c *completingRunInvoker) InvokeWithProgress(ctx context.Context, config codex.WorkerConfig, observer codex.WorkerProgressObserver) (codex.WorkerResult, error) {
	result, err := c.FakeInvoker.InvokeWithProgress(ctx, config, observer)
	if err != nil {
		return result, err
	}
	return c.completeWithEvidence(result, config), nil
}

// withCompletingInvoker swaps the build invoker for the duration of a test.
func withCompletingInvoker(t *testing.T) {
	t.Helper()
	original := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &completingRunInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = original })
}

// seedRunFixture creates a colony with N single-task phases in the build-flow
// test harness, where dispatch runs on the fake invoker (simulated mode).
func seedRunFixture(t *testing.T, phaseCount int) (dataDir, root string) {
	t.Helper()
	dataDir = setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Autopilot restoration fixture"
	now := time.Now().UTC()
	phases := make([]colony.Phase, 0, phaseCount)
	for i := 1; i <= phaseCount; i++ {
		taskID := fmt.Sprintf("%d.1", i)
		status := colony.PhasePending
		if i == 1 {
			status = colony.PhaseReady
		}
		phases = append(phases, colony.Phase{
			ID:     i,
			Name:   fmt.Sprintf("Fixture phase %d", i),
			Status: status,
			Tasks:  []colony.Task{{ID: &taskID, Goal: fmt.Sprintf("Task for phase %d", i), Status: colony.TaskPending}},
		})
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		CurrentPhase:  1,
		InitializedAt: &now,
		Plan:          colony.Plan{Phases: phases},
	})
	return dataDir, root
}

// TestRunAutopilotCompletesMultiPhase proves the loop actually loops: a
// two-phase colony runs build -> continue -> advance twice and finishes with
// stopped_reason "completed". This is the executable evidence WORKFLOW-06
// claims — the previous evidence test asserted phases_completed == 0.
func TestRunAutopilotCompletesMultiPhase(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	seedRunFixture(t, 2)
	withCompletingInvoker(t)

	rootCmd.SetArgs([]string{"run", "--continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	t.Logf("run result: stopped_reason=%v phases_completed=%v next=%v steps=%v", result["stopped_reason"], result["phases_completed"], result["next"], result["steps"])
	if result["stopped_reason"] != "completed" {
		t.Fatalf("stopped_reason = %v, want completed", result["stopped_reason"])
	}
	if intValue(result["phases_completed"]) < 1 {
		t.Fatalf("phases_completed = %v, want >= 1", result["phases_completed"])
	}
	if result["completed"] != true {
		t.Fatalf("completed = %v, want true", result["completed"])
	}
}

// TestRunAutopilotStreamsAdvancement asserts the classic narration is emitted
// while the run executes: engage banner, phase advancement, momentum ticker,
// and the completion celebrations. Fails if any emit call is removed from the
// loop.
func TestRunAutopilotStreamsAdvancement(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")
	saveGlobals(t)
	resetRootCmd(t)
	seedRunFixture(t, 2)
	withCompletingInvoker(t)

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"run", "--continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"A U T O P I L O T   E N G A G E D",
		"P H A S E   A D V A N C E M E N T",
		"✅ Phase 1: Fixture phase 1 — COMPLETED",
		"--- Autopilot: Phase 1 done",
		"A U T O P I L O T   C O M P L E T E",
		"P R O J E C T   C O M P L E T E",
		"The colony rests. Well done!",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("streamed run output missing %q", want)
		}
	}
}

// TestRunAutopilotReplanDue proves the replan checkpoint pauses the run, and
// that the classic default interval of 2 is back on the flag.
func TestRunAutopilotReplanDue(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)

	if flag := runCompatibilityCmd.Flags().Lookup("replan-interval"); flag == nil || flag.DefValue != "2" {
		t.Fatalf("replan-interval default = %v, want 2 (classic cadence)", flag)
	}

	seedRunFixture(t, 3)
	withCompletingInvoker(t)

	rootCmd.SetArgs([]string{"run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["stopped_reason"] != "replan_due" {
		t.Fatalf("stopped_reason = %v, want replan_due", result["stopped_reason"])
	}
	if intValue(result["phases_completed"]) != 2 {
		t.Fatalf("phases_completed = %v, want 2 (paused at the default interval)", result["phases_completed"])
	}
	if result["next"] != "aether plan" {
		t.Fatalf("next = %v, want aether plan", result["next"])
	}
}

// TestGoldenAutopilotPauseConditions is the parity anchor named by
// .aether/docs/PARITY_CLASSIC_VS_GO.md — table-driven proof that each seeded
// pause condition stops the REAL run loop with a paused:<condition> reason.
func TestGoldenAutopilotPauseConditions(t *testing.T) {
	cases := []struct {
		name       string
		seed       func(t *testing.T, dataDir string)
		wantReason string
	}{
		{
			name: "active_blockers",
			seed: func(t *testing.T, dataDir string) {
				if err := store.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
					ID: "pd_test", Type: "blocker", Description: "unresolved blocker", Resolved: false, CreatedAt: "2026-08-16T00:00:00Z",
				}}}); err != nil {
					t.Fatalf("seed pending decisions: %v", err)
				}
			},
			wantReason: "paused:active_blockers:1",
		},
		{
			name: "critical_chaos_findings",
			seed: func(t *testing.T, dataDir string) {
				middenJSON := `{"entries":[{"id":"m1","category":"chaos","message":"boom","tags":["critical"]}]}`
				if err := os.WriteFile(filepath.Join(dataDir, "midden.json"), []byte(middenJSON), 0644); err != nil {
					t.Fatalf("seed midden: %v", err)
				}
			},
			wantReason: "paused:critical_chaos_findings",
		},
		{
			name: "uncommitted_changes",
			seed: func(t *testing.T, dataDir string) {
				if err := os.WriteFile(filepath.Join(dataDir, "uncommitted-changes.marker"), []byte("1"), 0644); err != nil {
					t.Fatalf("seed marker: %v", err)
				}
			},
			wantReason: "paused:uncommitted_changes",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AETHER_OUTPUT_MODE", "json")
			saveGlobals(t)
			resetRootCmd(t)
			dataDir, _ := seedRunFixture(t, 2)
			tc.seed(t, dataDir)

			rootCmd.SetArgs([]string{"run", "--continue"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("run returned error: %v", err)
			}

			env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
			result := env["result"].(map[string]interface{})
			reason := stringValue(result["stopped_reason"])
			if reason != tc.wantReason {
				t.Fatalf("stopped_reason = %q, want %q", reason, tc.wantReason)
			}
			if stringValue(result["pause_reason"]) == "" {
				t.Fatalf("pause_reason missing from result: %v", result)
			}

			var ap autopilotState
			if err := store.LoadJSON(autopilotStatePath, &ap); err != nil {
				t.Fatalf("load autopilot state: %v", err)
			}
			if ap.Status != "paused" {
				t.Fatalf("autopilot status = %q, want paused", ap.Status)
			}
			if ap.Reason == "" {
				t.Fatalf("autopilot state reason empty, want the pause condition recorded")
			}
		})
	}
}

// TestRunDryRunDoesNotMutateState locks the inspection law: a --dry-run
// hashes identically before and after across the entire data directory.
func TestRunDryRunDoesNotMutateState(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, _ := seedRunFixture(t, 2)

	before := hashDirContents(t, dataDir)

	rootCmd.SetArgs([]string{"run", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run --dry-run returned error: %v", err)
	}

	after := hashDirContents(t, dataDir)
	if before != after {
		t.Fatalf("dry-run mutated .aether/data (hash %s -> %s)", before, after)
	}
}

// TestRunDryRunListsPhasesAndPauseTriggers asserts the rich preview: every
// planned phase is named, and the pause-trigger catalog is on screen.
func TestRunDryRunListsPhasesAndPauseTriggers(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")
	saveGlobals(t)
	resetRootCmd(t)
	seedRunFixture(t, 2)

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"run", "--dry-run", "--continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run --dry-run returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"Fixture phase 1",
		"Fixture phase 2",
		"Pause Triggers",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("dry-run output missing %q", want)
		}
	}
	for _, trigger := range autopilotPauseTriggerCatalog() {
		if !strings.Contains(output, trigger.Condition) {
			t.Errorf("dry-run output missing pause trigger %q", trigger.Condition)
		}
	}
}

// TestRunHeadlessQueuesPendingDecisionOnPause: under --headless, a pause is
// recorded on the pending-decision queue for later review — the classic
// headless contract.
func TestRunHeadlessQueuesPendingDecisionOnPause(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	dataDir, _ := seedRunFixture(t, 2)

	if err := os.WriteFile(filepath.Join(dataDir, "uncommitted-changes.marker"), []byte("1"), 0644); err != nil {
		t.Fatalf("seed marker: %v", err)
	}

	rootCmd.SetArgs([]string{"run", "--continue", "--headless"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	file := loadPendingDecisionFile()
	found := false
	for _, decision := range file.Decisions {
		if decision.Type == "autopilot_pause" && !decision.Resolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an unresolved autopilot_pause pending decision, got %+v", file.Decisions)
	}
}
