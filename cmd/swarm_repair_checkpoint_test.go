package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

func mustWriteRepairFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// swarmRepairCheckpointLiveRecoveryStates replays the live.recovery.changed
// topic and returns, in emission order, the RecoveryState of every event
// belonging to episodeID -- the raw published stream, not the reduced
// snapshot (whose single RecoveryState field would overwrite "saved" with
// "restored" and hide that both were ever published).
func swarmRepairCheckpointLiveRecoveryStates(t *testing.T, episodeID string) []string {
	t.Helper()
	if store == nil {
		t.Fatal("swarmRepairCheckpointLiveRecoveryStates: store is nil")
	}
	bus := events.NewBus(store, events.DefaultConfig())
	raw, err := bus.Replay(context.Background(), events.LiveTopicRecoveryChanged, time.Time{}, 0)
	if err != nil {
		t.Fatalf("replay recovery events: %v", err)
	}
	var states []string
	for _, evt := range raw {
		var payload events.ColonyLivePayload
		if jsonErr := json.Unmarshal(evt.Payload, &payload); jsonErr != nil {
			continue
		}
		if payload.EpisodeID != episodeID {
			continue
		}
		states = append(states, payload.RecoveryState)
	}
	return states
}

// TestSwarmCheckpointIdentityIsStableAndDistinct proves
// swarmRepairCheckpointIdentity is stable for the same swarm identifier and
// target, invariant to whitespace/case normalization on the target text,
// and never collides across two genuinely different targets or two
// different swarm runs.
func TestSwarmCheckpointIdentityIsStableAndDistinct(t *testing.T) {
	base := swarmRepairCheckpointIdentity("swarm-1", "Auth panic when session is missing")
	again := swarmRepairCheckpointIdentity("swarm-1", "Auth panic when session is missing")
	if base != again {
		t.Fatalf("identity is not stable for equal inputs: %q != %q", base, again)
	}

	normalized := swarmRepairCheckpointIdentity("swarm-1", "  Auth Panic   WHEN Session Is Missing  ")
	if normalized != base {
		t.Fatalf("identity is not normalization-invariant across whitespace/case: %q != %q", normalized, base)
	}

	differentTarget := swarmRepairCheckpointIdentity("swarm-1", "Auth panic when session is missing entirely differently")
	if differentTarget == base {
		t.Fatalf("two genuinely different targets collided on identity %q", differentTarget)
	}

	differentSwarm := swarmRepairCheckpointIdentity("swarm-2", "Auth panic when session is missing")
	if differentSwarm == base {
		t.Fatalf("two different swarm runs collided on identity %q", differentSwarm)
	}
}

// TestSwarmCheckpointRoundTripsDeclaredPaths proves the adapter's declared
// scope restores modified files, restores deleted files, and removes a file
// created after the save -- while never touching a file outside the
// declared scope -- and that an empty selected-repair scope falls back to
// the whole-root checkpoint saveRepairCheckpoint already implements.
func TestSwarmCheckpointRoundTripsDeclaredPaths(t *testing.T) {
	root := t.TempDir()
	mustWriteRepairFixtureFile(t, filepath.Join(root, "keep.txt"), "original\n")
	mustWriteRepairFixtureFile(t, filepath.Join(root, "to-delete.txt"), "will be deleted by the repair\n")
	mustWriteRepairFixtureFile(t, filepath.Join(root, "out-of-scope.txt"), "never declared as evidence\n")

	comparison := swarmComparison{
		Selected: &swarmRankedRepair{Repair: "patch the nil guard", Lenses: []string{swarmLensErrorPath}},
		Hypotheses: []swarmHypothesis{{
			Lens: swarmLensErrorPath,
			Evidence: []swarmHypothesisEvidence{
				{Location: "keep.txt"},
				{Location: "to-delete.txt"},
				// Declared as evidence even though it does not exist yet at
				// save time -- exactly the shape a repair that will create
				// a new file produces.
				{Location: "created-by-repair.txt"},
			},
		}},
	}

	checkpoint, err := saveSwarmRepairCheckpoint(root, "swarm-roundtrip", "round trip target", comparison)
	if err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	defer os.RemoveAll(checkpoint.BackupDir)
	if checkpoint.FullRoot {
		t.Fatalf("expected a scoped checkpoint, got a whole-root checkpoint: %+v", checkpoint)
	}
	for _, want := range []string{"keep.txt", "to-delete.txt", "created-by-repair.txt"} {
		found := false
		for _, got := range checkpoint.Paths {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("checkpoint paths %v missing %q", checkpoint.Paths, want)
		}
	}

	// Simulate the repair wave: mutate keep.txt, delete to-delete.txt,
	// create a brand-new file inside the declared scope, and touch a file
	// outside the declared scope.
	mustWriteRepairFixtureFile(t, filepath.Join(root, "keep.txt"), "mutated by the repair\n")
	if err := os.Remove(filepath.Join(root, "to-delete.txt")); err != nil {
		t.Fatalf("remove fixture file: %v", err)
	}
	mustWriteRepairFixtureFile(t, filepath.Join(root, "created-by-repair.txt"), "should not survive restore\n")
	mustWriteRepairFixtureFile(t, filepath.Join(root, "out-of-scope.txt"), "mutated but never declared\n")

	if err := restoreSwarmRepairCheckpoint(checkpoint); err != nil {
		t.Fatalf("restore checkpoint: %v", err)
	}

	if got, err := os.ReadFile(filepath.Join(root, "keep.txt")); err != nil || string(got) != "original\n" {
		t.Fatalf("keep.txt not restored: content=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "to-delete.txt")); err != nil || string(got) != "will be deleted by the repair\n" {
		t.Fatalf("to-delete.txt not restored: content=%q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, "created-by-repair.txt")); !os.IsNotExist(err) {
		t.Fatalf("created-by-repair.txt was not removed by restore: err=%v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "out-of-scope.txt")); err != nil || string(got) != "mutated but never declared\n" {
		t.Fatalf("out-of-scope file was unexpectedly touched by restore: content=%q err=%v", got, err)
	}

	// An empty scope (no selected repair) falls back to the whole-root
	// checkpoint the primitive already supports, unchanged.
	wholeRoot, err := saveSwarmRepairCheckpoint(root, "swarm-wholeroot", "no selected repair", swarmComparison{})
	if err != nil {
		t.Fatalf("save whole-root checkpoint: %v", err)
	}
	defer os.RemoveAll(wholeRoot.BackupDir)
	if !wholeRoot.FullRoot {
		t.Fatalf("expected whole-root fallback for an empty scope, got %+v", wholeRoot)
	}
}

// TestSwarmCheckpointAnnouncementsAreOrdered proves the save announcement
// (both the rendered text and the live event) is written before the repair
// dispatch marker, and the restore announcement only after it -- and that
// both the checkpoint-saved and checkpoint-restored live events reach the
// replayed stream for the run's episode.
func TestSwarmCheckpointAnnouncementsAreOrdered(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	saveGlobals(t)
	resetRootCmd(t)
	setupBuildFlowTest(t)

	var buf bytes.Buffer
	stdout = &buf

	swarmID := "swarm-announce-order"
	target := "order fixture target"

	announceSwarmCheckpointSaved(swarmID, target)
	writeVisualOutput(stdout, "dispatching the fix wave\n")
	announceSwarmCheckpointRestored(swarmID, target)

	visual := buf.String()
	savedAt := strings.Index(visual, "Saving your project's current state")
	dispatchAt := strings.Index(visual, "dispatching the fix wave")
	restoredAt := strings.Index(visual, "put back exactly to the state it was saved in")
	if savedAt < 0 || dispatchAt < 0 || restoredAt < 0 {
		t.Fatalf("expected all three markers in the rendered flow:\n%s", visual)
	}
	if !(savedAt < dispatchAt && dispatchAt < restoredAt) {
		t.Fatalf("announcements out of order: saved=%d dispatch=%d restored=%d\n%s", savedAt, dispatchAt, restoredAt, visual)
	}

	states := swarmRepairCheckpointLiveRecoveryStates(t, swarmID)
	want := []string{swarmRecoveryStateCheckpointSaved, swarmRecoveryStateCheckpointRestored}
	if !reflect.DeepEqual(states, want) {
		t.Fatalf("live recovery states = %v, want %v", states, want)
	}
}

// swarmRepairFixtureInvoker is Task 2's own stub worker invoker: unlike
// swarmTestInvoker (cmd/swarm_cmd_test.go), it genuinely mutates a real
// fixture file under root when it plays the builder, so the checkpoint
// round trip inside a real runSwarmDestroy pass has something concrete to
// protect and restore. Every investigation lens reports evidence naming
// the same fixture file; tracker and scout additionally share the exact
// same root-cause text so compareSwarmHypotheses' shared-cause detection
// selects a repair scoped to that file.
type swarmRepairFixtureInvoker struct {
	t             *testing.T
	root          string
	watcherStatus string
	configs       []codex.WorkerConfig
}

func (i *swarmRepairFixtureInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.configs = append(i.configs, cfg)

	response := swarmWorkerResponse{Role: cfg.Caste, Status: "completed", Summary: cfg.Caste + " ran the swarm pass."}
	switch cfg.Caste {
	case "tracker", "scout":
		response.RootCause = "target.go has a broken guard"
		response.Evidence = []string{"target.go"}
	case "archaeologist", "oracle":
		response.Summary = cfg.Caste + " found supporting context in target.go."
		response.Evidence = []string{"target.go"}
	case "builder":
		if err := os.WriteFile(filepath.Join(i.root, "target.go"), []byte("// repaired\npackage fixture\n"), 0o644); err != nil {
			i.t.Fatalf("builder failed to mutate fixture: %v", err)
		}
		response.ProposedFix = "patch the broken guard in target.go"
		response.FilesTouched = []string{"target.go"}
	case "watcher":
		status := i.watcherStatus
		if status == "" {
			status = "completed"
		}
		response.Status = status
		if status != "completed" {
			response.Summary = "verification still fails after the fix"
		}
	}

	result := codex.WorkerResult{
		WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID,
		Status: response.Status, Summary: response.Summary, Duration: time.Millisecond,
	}
	if strings.TrimSpace(cfg.ResponsePath) == "" {
		return codex.WorkerResult{}, context.Canceled
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ResponsePath), 0o755); err != nil {
		return codex.WorkerResult{}, err
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(cfg.ResponsePath, append(data, '\n'), 0o644); err != nil {
		return codex.WorkerResult{}, err
	}
	return result, nil
}

func (i *swarmRepairFixtureInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *swarmRepairFixtureInvoker) ValidateAgent(_ string) error       { return nil }

// setupSwarmRepairCheckpointFixture wires an isolated fixture root with one
// real source file the fix wave can mutate, a bound store, and a stub
// invoker whose watcher plays back watcherStatus -- driving the real
// runSwarmDestroy path end to end, per Task 2's plan instruction.
func setupSwarmRepairCheckpointFixture(t *testing.T, watcherStatus string) (root string, invoker *swarmRepairFixtureInvoker) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	mustWriteRepairFixtureFile(t, filepath.Join(root, "target.go"), "// original\npackage fixture\n")

	goal := "Destroy a stubborn checkpoint bug"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
	})

	invoker = &swarmRepairFixtureInvoker{t: t, root: root, watcherStatus: watcherStatus}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	return root, invoker
}

// TestSwarmRepairCheckpointsBeforeTheFixWave proves the checkpoint save (and
// its announcement) happens before the fix wave's own dispatch preview is
// rendered, and that a passing verification leaves the repair in place.
func TestSwarmRepairCheckpointsBeforeTheFixWave(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	root, _ := setupSwarmRepairCheckpointFixture(t, "completed")

	result, err := runSwarmDestroy(root, "Fix the broken guard in target.go")
	if err != nil {
		t.Fatalf("runSwarmDestroy: %v", err)
	}
	if got := result["status"]; got != "completed" {
		t.Fatalf("status = %v, want completed: %+v", got, result)
	}

	visual := stdout.(*bytes.Buffer).String()
	savedAt := strings.Index(visual, "Saving your project's current state")
	fixWaveAt := strings.Index(visual, spacedTitle("Fix Wave"))
	if savedAt < 0 || fixWaveAt < 0 {
		t.Fatalf("expected both the checkpoint save and the fix wave banner in output:\n%s", visual)
	}
	if savedAt > fixWaveAt {
		t.Fatalf("checkpoint save happened after the fix wave dispatch: saved=%d fixWave=%d\n%s", savedAt, fixWaveAt, visual)
	}

	got, err := os.ReadFile(filepath.Join(root, "target.go"))
	if err != nil || string(got) != "// repaired\npackage fixture\n" {
		t.Fatalf("expected the repair to stay in place, got %q err=%v", got, err)
	}
}

// TestSwarmRepairRollsBackOnFailedVerification proves a fix wave whose
// verification fails restores the checkpoint exactly (byte-identical to the
// pre-repair fixture, by digest) and reports the run as a failed repair
// rather than a success.
func TestSwarmRepairRollsBackOnFailedVerification(t *testing.T) {
	root, invoker := setupSwarmRepairCheckpointFixture(t, "failed")

	preRepairDigest, err := repairCheckpointDirectoryDigest(root, []string{"target.go"})
	if err != nil {
		t.Fatalf("pre-repair digest: %v", err)
	}

	result, err := runSwarmDestroy(root, "Fix the broken guard in target.go")
	if err != nil {
		t.Fatalf("runSwarmDestroy: %v", err)
	}
	if got := result["status"]; got == "completed" {
		t.Fatalf("status = %v, want a failed repair, not completed", got)
	}
	if got := result["status"]; got == swarmRepairNotRestoredStatus {
		t.Fatalf("status = %v, restore should have succeeded in this fixture", got)
	}

	postRestoreDigest, err := repairCheckpointDirectoryDigest(root, []string{"target.go"})
	if err != nil {
		t.Fatalf("post-restore digest: %v", err)
	}
	if postRestoreDigest != preRepairDigest {
		t.Fatalf("fixture tree not byte-identical after restore: pre=%s post=%s", preRepairDigest, postRestoreDigest)
	}
	got, err := os.ReadFile(filepath.Join(root, "target.go"))
	if err != nil || string(got) != "// original\npackage fixture\n" {
		t.Fatalf("target.go was not restored to its pre-repair content: %q err=%v", got, err)
	}

	for _, cfg := range invoker.configs {
		if strings.Contains(strings.ToLower(cfg.TaskBrief), "approve") {
			t.Fatalf("worker brief unexpectedly asked for approval: %s", cfg.TaskBrief)
		}
	}
}

// TestSwarmRestoreFailureIsReportedHonestly proves that when the checkpoint
// restore itself fails, the run reports the distinct not-restored state,
// names the saved copy's location and a runnable recovery command, and
// never renders the rollback-succeeded announcement text.
func TestSwarmRestoreFailureIsReportedHonestly(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	root, _ := setupSwarmRepairCheckpointFixture(t, "failed")

	original := swarmRestoreRepairCheckpointFunc
	swarmRestoreRepairCheckpointFunc = func(checkpoint repairCheckpoint) error {
		return fmt.Errorf("simulated restore failure")
	}
	t.Cleanup(func() { swarmRestoreRepairCheckpointFunc = original })

	result, err := runSwarmDestroy(root, "Fix the broken guard in target.go")
	if err != nil {
		t.Fatalf("runSwarmDestroy: %v", err)
	}
	if got := result["status"]; got != swarmRepairNotRestoredStatus {
		t.Fatalf("status = %v, want %v", got, swarmRepairNotRestoredStatus)
	}
	backupPath, _ := result["backup_path"].(string)
	if strings.TrimSpace(backupPath) == "" {
		t.Fatalf("expected a backup_path naming the saved copy: %+v", result)
	}
	recommendation, _ := result["recommendation"].(string)
	if !strings.Contains(recommendation, backupPath) {
		t.Fatalf("recommendation does not name the saved copy's directory: %q", recommendation)
	}
	if !strings.Contains(recommendation, "cp -r") {
		t.Fatalf("recommendation does not carry a runnable recovery command: %q", recommendation)
	}
	if strings.Contains(strings.ToLower(recommendation), "rolled back") || strings.Contains(strings.ToLower(recommendation), "rollback") {
		t.Fatalf("recommendation must never claim a rollback happened: %q", recommendation)
	}

	visual := stdout.(*bytes.Buffer).String()
	if strings.Contains(visual, "put back exactly to the state it was saved in") {
		t.Fatalf("rollback-succeeded announcement rendered despite a failed restore:\n%s", visual)
	}
}

// TestSwarmRepairAsksNothingMidFlight proves the automatic repair path never
// blocks on input and never renders an approval prompt, on both the
// passing and the failing verification branch.
func TestSwarmRepairAsksNothingMidFlight(t *testing.T) {
	for _, watcherStatus := range []string{"completed", "failed"} {
		t.Run(watcherStatus, func(t *testing.T) {
			root, _ := setupSwarmRepairCheckpointFixture(t, watcherStatus)

			done := make(chan struct{})
			go func() {
				defer close(done)
				if _, err := runSwarmDestroy(root, "Fix the broken guard in target.go"); err != nil {
					t.Errorf("runSwarmDestroy: %v", err)
				}
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("runSwarmDestroy did not return -- it is likely blocked waiting on a prompt")
			}

			visual := stdout.(*bytes.Buffer).String()
			for _, jargon := range []string{"y/n", "approve?", "proceed?", "confirm the repair"} {
				if strings.Contains(strings.ToLower(visual), jargon) {
					t.Fatalf("output contains an approval prompt %q:\n%s", jargon, visual)
				}
			}
		})
	}
}

// TestSwarmRepairAsNoCheckpointForNoRepair proves a run that selected no
// repair (no lens produced usable evidence) saves no checkpoint at all.
func TestSwarmRepairAsNoCheckpointForNoRepair(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Destroy a bug nobody can reproduce"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
	})

	invoker := &swarmNoEvidenceInvoker{}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	result, err := runSwarmDestroy(root, "A bug with no reproducible evidence")
	if err != nil {
		t.Fatalf("runSwarmDestroy: %v", err)
	}
	if noEvidence, _ := result["no_evidence"].(bool); !noEvidence {
		t.Fatalf("expected a no-evidence outcome, got %+v", result)
	}
	if invoker.builderDispatched {
		t.Fatalf("no repair was selected, but a fix-wave worker was still dispatched")
	}
}

// swarmNoEvidenceInvoker reports no usable claim from every investigation
// lens, so compareSwarmHypotheses selects no repair and the fix wave (and
// therefore the checkpoint) is never reached.
type swarmNoEvidenceInvoker struct {
	builderDispatched bool
}

func (i *swarmNoEvidenceInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if cfg.Caste == "builder" {
		i.builderDispatched = true
	}
	response := swarmWorkerResponse{Role: cfg.Caste, Status: "blocked", Summary: ""}
	result := codex.WorkerResult{
		WorkerName: cfg.WorkerName, Caste: cfg.Caste, TaskID: cfg.TaskID,
		Status: "blocked", Summary: "", Duration: time.Millisecond,
		Blockers: []string{"no evidence available"},
	}
	if strings.TrimSpace(cfg.ResponsePath) == "" {
		return codex.WorkerResult{}, context.Canceled
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ResponsePath), 0o755); err != nil {
		return codex.WorkerResult{}, err
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(cfg.ResponsePath, append(data, '\n'), 0o644); err != nil {
		return codex.WorkerResult{}, err
	}
	return result, nil
}

func (i *swarmNoEvidenceInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *swarmNoEvidenceInvoker) ValidateAgent(_ string) error       { return nil }
