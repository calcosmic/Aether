package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
