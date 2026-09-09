package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// nativePartialInvoker stands in for a worker that genuinely finished part of
// its grouped job and then died: it writes the files for the first
// creditedCount tasks, receipts exactly those, and reports `failed`.
type nativePartialInvoker struct {
	root          string
	taskIDs       []string
	creditedCount int
	commitFailure func()
}

func (i *nativePartialInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	if config.Caste != "builder" {
		return codex.WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "completed",
			Summary:    "nothing to do",
			Duration:   time.Millisecond,
		}, nil
	}
	credited := i.taskIDs[:i.creditedCount]
	files := make([]string, 0, len(credited))
	receipts := make([]codex.TaskReceipt, 0, len(credited))
	for _, id := range credited {
		file := "task-" + id + ".go"
		if err := os.WriteFile(filepath.Join(i.root, file), []byte("package fixture\n"), 0o644); err != nil {
			return codex.WorkerResult{}, err
		}
		files = append(files, file)
		receipts = append(receipts, codex.TaskReceipt{
			TaskID:        id,
			Status:        codex.TaskReceiptStatusCompleted,
			Summary:       "finished " + id,
			FilesCreated:  []string{file},
			FilesModified: []string{},
			TestsWritten:  []string{},
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "pass",
				CommandsRun:        []string{"go build ./..."},
			},
		})
	}
	if i.commitFailure != nil {
		i.commitFailure()
	}
	return codex.WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing the first steps",
		FilesCreated:  files,
		FilesModified: []string{},
		TestsWritten:  []string{},
		TaskReceipts:  receipts,
		Duration:      time.Millisecond,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go build ./..."},
		},
	}, nil
}

func (i *nativePartialInvoker) IsAvailable(ctx context.Context) bool { return true }

func (i *nativePartialInvoker) ValidateAgent(path string) error { return nil }

// setUpNativePartialBuild stages a six-chained-task phase ready for a direct
// (non-wrapper) build whose single worker finishes four tasks and then fails.
func setUpNativePartialBuild(t *testing.T, goal string) (string, *nativePartialInvoker, []string) {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	tasks, ids := sixChainedTasks()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Six chained steps",
				Status: colony.PhaseReady,
				Tasks:  tasks,
			}},
		},
	})

	invoker := &nativePartialInvoker{root: root, taskIDs: ids, creditedCount: 4}
	original := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = original })
	return root, invoker, ids
}

// latestAttemptForPhase returns the phase's most recent attempt record.
func latestAttemptForPhase(t *testing.T, phaseNum int) buildAttemptRecord {
	t.Helper()
	_, record, ok := loadLatestBuildAttempt(phaseNum)
	if !ok {
		t.Fatalf("phase %d has no latest build attempt", phaseNum)
	}
	return record
}

// TestNativePartialCreditIsJournalledAsPartial is the permanent regression lock
// for the 195 review's fourth warning (WR-04).
//
// The direct build lane credited real task work and then left its own journal
// entry reading `failed` -- the status whose documented meaning is "nothing of
// this attempt was credited". Everything that reads the journal (status,
// recovery, audits) was therefore told the opposite of what the colony state
// recorded, and the two build lanes disagreed about the same outcome.
func TestNativePartialCreditIsJournalledAsPartial(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	root, _, ids := setUpNativePartialBuild(t, "Native partial is journalled honestly")

	result, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{})
	if err != nil {
		t.Fatalf("a build with genuine partial credit must not error: %v", err)
	}
	if recovery, _ := result["recovery_job"].(bool); !recovery {
		t.Fatalf("expected a partial-credit result, got %#v", result)
	}

	record := latestAttemptForPhase(t, 1)
	if record.Status != buildAttemptPartial {
		t.Fatalf("the build credited %d of %d tasks but its journal entry reads %q; %q means nothing of this attempt was credited", 4, len(ids), record.Status, record.Status)
	}
}

// TestNativePartialCreditCommitsBeforeCreatingTheRecoveryRecord is the
// permanent regression lock for the 195 review's tenth warning (WR-10).
//
// The recovery record was created BEFORE the credit it exists to describe was
// committed. When the commit then failed -- a concurrent pause, a store write
// error -- the credit was rolled back and an orphan recovery record pointing at
// unfinished tasks was left behind, describing work no longer recorded as
// partially done.
func TestNativePartialCreditCommitsBeforeCreatingTheRecoveryRecord(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	root, invoker, _ := setUpNativePartialBuild(t, "Recovery record follows the credit")

	// Pause the colony while the worker runs. commitPartialBuildCredit's own
	// concurrency guard (validateRuntimeStateStillCurrent) then refuses the
	// commit, which is the failure mode WR-10 describes.
	paused := false
	invoker.commitFailure = func() {
		if paused {
			return
		}
		paused = true
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("load state during invoke: %v", err)
		}
		at := time.Now().UTC().Format(time.RFC3339)
		state.Paused = true
		state.PausedAt = &at
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("pause state during invoke: %v", err)
		}
	}

	_, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{})
	if err == nil {
		t.Fatal("expected the partial-credit commit to be refused while the colony is paused")
	}
	if !errors.Is(err, errRuntimeStateSuperseded) {
		t.Fatalf("expected a superseded-state refusal, got %v", err)
	}

	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ParentAttemptID != "" {
			t.Fatalf("recovery record %s was created for parent %s even though the credit it describes was never committed", record.ID, record.ParentAttemptID)
		}
	}
}

// TestPartialBuildDoesNotShowTheOrdinaryBuildDoneScreen is the permanent
// regression lock for the 195 review's fifth warning (WR-05).
//
// After a partially credited build the CLI fell back to the ordinary build
// screen, which unconditionally says verification happens next, names the
// following phase, and tells the owner to run the continue command. The
// recovery command was never shown at all. A half-built phase read as a
// finished one on the surface the owner actually looks at.
func TestPartialBuildDoesNotShowTheOrdinaryBuildDoneScreen(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	root, _, ids := setUpNativePartialBuild(t, "Partial builds do not look finished")
	pending := ids[4:]
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	result, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{})
	if err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	partial, ok := result["partial_recovery"].(partialBuildRetryOutcome)
	if !ok {
		t.Fatalf("native partial result has no typed partial_recovery projection: %#v", result["partial_recovery"])
	}
	wantCommand := buildUnfinishedRetryRedispatchCommand(1, pending)
	if !reflect.DeepEqual(partial.UnfinishedTaskIDs, pending) || partial.RedispatchCommand != wantCommand {
		t.Fatalf("typed native partial projection = %+v, want unfinished=%v command=%q", partial, pending, wantCommand)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load partial colony state: %v", err)
	}
	stdout = &bytes.Buffer{}
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	outputWorkflow(result, renderBuildPartialCreditResultVisual(state, state.Plan.Phases[0], result))

	out := stdout.(*bytes.Buffer).String()
	for _, claim := range []string{
		"Verification happens during",
		"follows after continue",
		"after the work is implemented",
	} {
		if strings.Contains(out, claim) {
			t.Errorf("a partially built phase still shows the ordinary finished-build line %q:\n%s", claim, out)
		}
	}
	for _, want := range append([]string{wantCommand}, pending...) {
		if !strings.Contains(out, want) {
			t.Errorf("the partial-build screen never mentions %q, so the owner is not told what is left or how to finish it:\n%s", want, out)
		}
	}

	stdout = &bytes.Buffer{}
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	outputWorkflow(result, "visual output must not leak into JSON")
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			RecoveryJob       bool                     `json:"recovery_job"`
			UnfinishedTaskIDs []string                 `json:"unfinished_task_ids"`
			RecoveryCommand   string                   `json:"recovery_command"`
			PartialRecovery   partialBuildRetryOutcome `json:"partial_recovery"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("native partial JSON is invalid: %v\n%s", err, stdout.(*bytes.Buffer).String())
	}
	if !envelope.OK || !envelope.Result.RecoveryJob {
		t.Fatalf("native partial JSON lost its machine outcome: %+v", envelope)
	}
	if !reflect.DeepEqual(envelope.Result.UnfinishedTaskIDs, pending) || envelope.Result.RecoveryCommand != wantCommand {
		t.Fatalf("native partial JSON top-level facts = %+v, want unfinished=%v command=%q", envelope.Result, pending, wantCommand)
	}
	if !reflect.DeepEqual(envelope.Result.PartialRecovery.UnfinishedTaskIDs, pending) || envelope.Result.PartialRecovery.RedispatchCommand != wantCommand {
		t.Fatalf("native partial JSON typed facts = %+v, want unfinished=%v command=%q", envelope.Result.PartialRecovery, pending, wantCommand)
	}
}
