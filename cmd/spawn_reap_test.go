package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Plan 173-09 (SPAWN-08 red-proofs).
//
// RecordSpawn stamps time.Now().UTC() with no injectable clock (see
// pkg/agent/spawn_tree.go:RecordSpawn), so these tests cannot create a stale
// entry through the normal spawn-log path. Instead they write directly into
// the test store's spawn-tree.txt in the on-disk pipe format, with a
// timestamp several hours in the past. This is a test-fixture decision, not
// a production one -- no clock-injection seam was added to SpawnTree.

// writeRawSpawnLine appends one raw line to spawn-tree.txt for the active
// store, bypassing RecordSpawn/UpdateStatus entirely so the timestamp can be
// controlled directly.
func writeRawSpawnLine(t *testing.T, line string) {
	t.Helper()
	if err := store.UpdateFile("spawn-tree.txt", func(existing []byte) ([]byte, error) {
		if len(existing) > 0 && existing[len(existing)-1] != '\n' {
			existing = append(existing, '\n')
		}
		return append(existing, []byte(line+"\n")...), nil
	}); err != nil {
		t.Fatalf("write raw spawn line %q: %v", line, err)
	}
}

// writeStaleSpawnEntry writes a spawn-entry line (the 7-field
// timestamp|parent|caste|name|task|depth|status shape) with an explicit
// spawn timestamp, standing in for what RecordSpawn would have written at
// that moment in the past.
func writeStaleSpawnEntry(t *testing.T, spawnedAt time.Time, parent, caste, name, task string, depth int, status string) {
	t.Helper()
	line := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s", spawnedAt.UTC().Format(time.RFC3339), parent, caste, name, task, depth, status)
	writeRawSpawnLine(t, line)
}

// writeSpawnActivityRefresh writes a completion-shaped line (the 4-field
// timestamp|name|status|summary shape) that parseSpawnTreeBytes merges onto
// the matching spawn entry, refreshing its ActivityTimestamp and status --
// standing in for what an UpdateStatus call would have done at that moment
// in the past, without touching the entry's original spawn Timestamp.
func writeSpawnActivityRefresh(t *testing.T, activityAt time.Time, name, status string) {
	t.Helper()
	line := fmt.Sprintf("%s|%s|%s|%s", activityAt.UTC().Format(time.RFC3339), name, status, "")
	writeRawSpawnLine(t, line)
}

// findSpawnEntryForTest returns the entry named name from the current
// spawn-tree.txt, failing the test if it is not found.
func findSpawnEntryForTest(t *testing.T, st *agent.SpawnTree, name string) agent.SpawnEntry {
	t.Helper()
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	for _, e := range entries {
		if e.AgentName == name {
			return e
		}
	}
	t.Fatalf("entry %q not found in spawn tree", name)
	return agent.SpawnEntry{}
}

// TestSpawnReapReleasesBudgetForStaleEntryOnly is D-18 and SPAWN-08's core
// claim in one test: a run with one stale live entry and one fresh live
// entry has both counted toward the whole-run budget; after reaping, the
// stale entry is abandoned, the fresh one is untouched, and consumed budget
// went down by exactly one -- not merely that some status somewhere changed.
func TestSpawnReapReleasesBudgetForStaleEntryOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	runStart := now.Add(-5 * time.Hour)
	if _, err := beginRuntimeSpawnRun("test-run", runStart); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// Stale: spawned 3 hours ago, well past the 120-minute default threshold.
	writeStaleSpawnEntry(t, now.Add(-3*time.Hour), "Queen", "builder", "Ghost", "t", 1, "spawned")
	// Fresh: spawned 1 minute ago, comfortably inside the threshold.
	writeStaleSpawnEntry(t, now.Add(-1*time.Minute), "Queen", "builder", "Fresh", "t", 1, "spawned")

	before, err := spawnTreeBudgetState()
	if err != nil {
		t.Fatalf("budget state before reaping: %v", err)
	}
	if before.Consumed != 2 {
		t.Fatalf("Consumed before reaping = %d, want 2", before.Consumed)
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	reaped, err := spawnReapStaleEntries(st, now)
	if err != nil {
		t.Fatalf("spawnReapStaleEntries: %v", err)
	}
	if len(reaped) != 1 || reaped[0] != "Ghost" {
		t.Fatalf("reaped = %v, want exactly [\"Ghost\"]", reaped)
	}

	after, err := spawnTreeBudgetState()
	if err != nil {
		t.Fatalf("budget state after reaping: %v", err)
	}
	if after.Consumed != before.Consumed-1 {
		t.Fatalf("Consumed after reaping = %d, want %d (before - 1)", after.Consumed, before.Consumed-1)
	}

	ghost := findSpawnEntryForTest(t, st, "Ghost")
	if ghost.Status != agent.SpawnStatusAbandoned {
		t.Errorf("Ghost status = %q, want %q", ghost.Status, agent.SpawnStatusAbandoned)
	}
	fresh := findSpawnEntryForTest(t, st, "Fresh")
	if fresh.Status != "spawned" {
		t.Errorf("Fresh status = %q, want unchanged \"spawned\"", fresh.Status)
	}
}

// TestSpawnReapNeverTouchesAnEntryInsideTheThreshold is D-16's boundary
// proof: an entry just inside the (default 120-minute) threshold is never
// reaped, and an entry just outside it is.
func TestSpawnReapNeverTouchesAnEntryInsideTheThreshold(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	runStart := now.Add(-6 * time.Hour)
	if _, err := beginRuntimeSpawnRun("test-run", runStart); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// Just inside the 120-minute default: 119 minutes elapsed.
	writeStaleSpawnEntry(t, now.Add(-119*time.Minute), "Queen", "builder", "Inside", "t", 1, "spawned")
	// Just outside: 121 minutes elapsed.
	writeStaleSpawnEntry(t, now.Add(-121*time.Minute), "Queen", "builder", "Outside", "t", 1, "spawned")

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	reaped, err := spawnReapStaleEntries(st, now)
	if err != nil {
		t.Fatalf("spawnReapStaleEntries: %v", err)
	}
	if len(reaped) != 1 || reaped[0] != "Outside" {
		t.Fatalf("reaped = %v, want exactly [\"Outside\"]", reaped)
	}

	inside := findSpawnEntryForTest(t, st, "Inside")
	if inside.Status != "spawned" {
		t.Errorf("Inside status = %q, want unchanged \"spawned\" -- the reaper touched an entry inside the threshold", inside.Status)
	}
	outside := findSpawnEntryForTest(t, st, "Outside")
	if outside.Status != agent.SpawnStatusAbandoned {
		t.Errorf("Outside status = %q, want %q", outside.Status, agent.SpawnStatusAbandoned)
	}
}

// TestSpawnReapRespectsRefreshedActivity is the test that stops the reaper
// killing live work (Pitfall 4): an entry whose original Timestamp is hours
// old but whose ActivityTimestamp was refreshed recently must NOT be reaped,
// because the later of the two timestamps is the only evidence of life this
// system has. An implementation reading only Timestamp would pass the two
// tests above and still kill this entry.
func TestSpawnReapRespectsRefreshedActivity(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	runStart := now.Add(-6 * time.Hour)
	if _, err := beginRuntimeSpawnRun("test-run", runStart); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// Original spawn timestamp is 5 hours old -- well past the threshold on
	// its own. Its activity was refreshed 10 minutes ago and it is still
	// reporting a live status ("active"), which is the only signal this
	// system has that it is not a ghost.
	writeStaleSpawnEntry(t, now.Add(-5*time.Hour), "Queen", "builder", "StillWorking", "t", 1, "spawned")
	writeSpawnActivityRefresh(t, now.Add(-10*time.Minute), "StillWorking", "active")

	st := agent.NewSpawnTree(store, "spawn-tree.txt")

	entry := findSpawnEntryForTest(t, st, "StillWorking")
	if !agent.IsLiveSpawnStatus(entry.Status) {
		t.Fatalf("fixture setup: StillWorking status %q is not live -- test would prove nothing", entry.Status)
	}

	reaped, err := spawnReapStaleEntries(st, now)
	if err != nil {
		t.Fatalf("spawnReapStaleEntries: %v", err)
	}
	for _, name := range reaped {
		if name == "StillWorking" {
			t.Fatalf("reaped = %v: StillWorking was reaped despite its recently-refreshed activity timestamp", reaped)
		}
	}

	after := findSpawnEntryForTest(t, st, "StillWorking")
	if after.Status != "active" {
		t.Errorf("StillWorking status = %q, want unchanged \"active\"", after.Status)
	}
}

// TestSpawnReapThresholdIsConfigurable is D-17's proof: the threshold is
// read from colony state, not merely declared. The same 119-minutes-elapsed
// entry that test 2 spared under the default 120-minute threshold is reaped
// once colony state configures a smaller threshold, and a second entry with
// no configured threshold at all falls back to the generous default.
func TestSpawnReapThresholdIsConfigurable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	runStart := now.Add(-6 * time.Hour)
	if _, err := beginRuntimeSpawnRun("test-run", runStart); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	// A small configured threshold makes a 119-minute-old entry reapable,
	// where the 120-minute default (test 2) would have spared it.
	small := 30
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version:                   "3.0",
		State:                     "READY",
		SpawnReapThresholdMinutes: &small,
	}); err != nil {
		t.Fatalf("save colony state with configured threshold: %v", err)
	}
	writeStaleSpawnEntry(t, now.Add(-119*time.Minute), "Queen", "builder", "Configured", "t", 1, "spawned")

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	reaped, err := spawnReapStaleEntries(st, now)
	if err != nil {
		t.Fatalf("spawnReapStaleEntries (configured threshold): %v", err)
	}
	if len(reaped) != 1 || reaped[0] != "Configured" {
		t.Fatalf("reaped = %v, want exactly [\"Configured\"] once the threshold is configured to 30 minutes", reaped)
	}

	// Now unset the configured threshold entirely and prove the default
	// (120 minutes) is what applies -- an identically-aged entry is spared.
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		State:   "READY",
	}); err != nil {
		t.Fatalf("save colony state with threshold unset: %v", err)
	}
	writeStaleSpawnEntry(t, now.Add(-119*time.Minute), "Queen", "builder", "Default", "t", 1, "spawned")

	reaped, err = spawnReapStaleEntries(st, now)
	if err != nil {
		t.Fatalf("spawnReapStaleEntries (default threshold): %v", err)
	}
	for _, name := range reaped {
		if name == "Default" {
			t.Fatalf("reaped = %v: Default was reaped even with the threshold unset -- the generous 120-minute default did not apply", reaped)
		}
	}
	defaultEntry := findSpawnEntryForTest(t, st, "Default")
	if defaultEntry.Status != "spawned" {
		t.Errorf("Default status = %q, want unchanged \"spawned\"", defaultEntry.Status)
	}
}

// snapshotStoreFileHashesForTest returns a name->sha256 map of every file
// under basePath, used to prove a command touched nothing (or, in the
// --clear case, touched exactly what was expected).
func snapshotStoreFileHashesForTest(t *testing.T, basePath string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sum := sha256.Sum256(data)
		rel, rerr := filepath.Rel(basePath, path)
		if rerr != nil {
			return rerr
		}
		hashes[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot store file hashes: %v", err)
	}
	return hashes
}

func assertHashSnapshotsEqualForTest(t *testing.T, label string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("%s: file count changed: before=%d after=%d", label, len(before), len(after))
	}
	for name, beforeHash := range before {
		afterHash, ok := after[name]
		if !ok {
			t.Fatalf("%s: file %q present before, missing after", label, name)
		}
		if beforeHash != afterHash {
			t.Fatalf("%s: file %q hash changed: before=%s after=%s", label, name, beforeHash, afterHash)
		}
	}
}

// TestSpawnOrphansListingMutatesNothing is CLAUDE.md's --dry-run corollary,
// applied to spawn-orphans: the listing path (no --clear) must not change a
// single byte anywhere in the store, proven by hashing every file across two
// consecutive runs and asserting byte-identical stdout both times. The
// --clear path is then proven to be a real mutation, not a no-op, by
// asserting the tree file DID change afterward.
func TestSpawnOrphansListingMutatesNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	now := time.Now().UTC()
	runStart := now.Add(-6 * time.Hour)
	if _, err := beginRuntimeSpawnRun("test-run", runStart); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	// One real orphan, so the clear path below has something to mutate.
	writeStaleSpawnEntry(t, now.Add(-3*time.Hour), "Queen", "builder", "Ghost", "t", 1, "spawned")

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runListing := func() string {
		buf.Reset()
		errBuf.Reset()
		renderedCommandExitCode.Store(0)
		rootCmd.SetArgs([]string{"spawn-orphans"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("spawn-orphans failed: %v\nstderr: %s", err, errBuf.String())
		}
		if code := int(renderedCommandExitCode.Load()); code != 0 {
			t.Fatalf("spawn-orphans exited %d: %s", code, errBuf.String())
		}
		return buf.String()
	}

	before := snapshotStoreFileHashesForTest(t, store.BasePath())
	firstOutput := runListing()
	afterFirst := snapshotStoreFileHashesForTest(t, store.BasePath())
	assertHashSnapshotsEqualForTest(t, "after first listing", before, afterFirst)

	secondOutput := runListing()
	afterSecond := snapshotStoreFileHashesForTest(t, store.BasePath())
	assertHashSnapshotsEqualForTest(t, "after second listing", before, afterSecond)

	if firstOutput != secondOutput {
		t.Fatalf("spawn-orphans listing output is not byte-identical across two runs:\nfirst:  %q\nsecond: %q", firstOutput, secondOutput)
	}

	treeBefore, err := os.ReadFile(filepath.Join(store.BasePath(), "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("read spawn-tree.txt before --clear: %v", err)
	}

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-orphans", "--clear"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-orphans --clear failed: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("spawn-orphans --clear exited %d: %s", code, errBuf.String())
	}

	treeAfter, err := os.ReadFile(filepath.Join(store.BasePath(), "spawn-tree.txt"))
	if err != nil {
		t.Fatalf("read spawn-tree.txt after --clear: %v", err)
	}
	if bytes.Equal(treeBefore, treeAfter) {
		t.Fatalf("spawn-tree.txt is byte-identical before and after --clear -- the clear path is a no-op")
	}
}
