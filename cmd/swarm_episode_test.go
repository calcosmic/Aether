package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// runSwarmDestroyFixture drives the real public Swarm path (`aether swarm
// <target>`) with the existing swarmTestInvoker fixture, returning the
// swarm ID the run produced.
func runSwarmDestroyFixture(t *testing.T, target string) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to test root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	goal := "Fix a bug"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Bug fix",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Fix the bug", Status: colony.TaskPending}},
			}},
		},
	})

	originalInvoker := newSwarmWorkerInvoker
	invoker := &swarmTestInvoker{}
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"swarm", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	swarmID, _ := result["swarm_id"].(string)
	if strings.TrimSpace(swarmID) == "" {
		t.Fatalf("expected swarm_id in result, got %v", result)
	}
	return swarmID
}

// swarmStrikeHistoryDigestFor is a test-only helper that hashes the JSON
// shape of the current strike history for a target -- the same derivation
// evaluateSwarmStrikeHistory (cmd/swarm_strikes.go) performs -- so a test
// can prove reading an episode never perturbs it.
func swarmStrikeHistoryDigestFor(t *testing.T, target string) string {
	t.Helper()
	history, err := evaluateSwarmStrikeHistory(store, target)
	if err != nil {
		t.Fatalf("evaluate swarm strike history: %v", err)
	}
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("marshal strike history: %v", err)
	}
	return string(data)
}

func TestSwarmRunProducesOneReplaySafeEpisode(t *testing.T) {
	target := "Episode fixture target one"
	swarmID := runSwarmDestroyFixture(t, target)

	swarmDir := filepath.Join(store.BasePath(), "swarms", swarmID)
	episodePath := filepath.Join(swarmDir, "episode.json")
	if _, err := os.Stat(episodePath); err != nil {
		t.Fatalf("expected exactly one episode file at %s: %v", episodePath, err)
	}

	episode, ok := loadSwarmEpisode(store, swarmID)
	if !ok {
		t.Fatalf("could not load episode for %s", swarmID)
	}
	if episode.SwarmID != swarmID {
		t.Fatalf("episode swarm id = %q, want %q", episode.SwarmID, swarmID)
	}
	if episode.Status != swarmEpisodeStatusCompleted {
		t.Fatalf("episode status = %q, want %q", episode.Status, swarmEpisodeStatusCompleted)
	}
	if len(episode.Lenses) != 4 {
		t.Fatalf("episode lenses = %d, want 4", len(episode.Lenses))
	}
	for _, lens := range episode.Lenses {
		if !lens.Reported {
			t.Fatalf("lens %s did not report: %+v", lens.ID, lens)
		}
	}
	if len(episode.Hypotheses) != 4 {
		t.Fatalf("episode hypotheses = %d, want 4", len(episode.Hypotheses))
	}
	if episode.Comparison.Selected == nil {
		t.Fatalf("expected a selected repair on the episode's comparison")
	}
	if !episode.Checkpoint.Saved {
		t.Fatalf("expected checkpoint.saved = true")
	}
	if episode.Checkpoint.Restored {
		t.Fatalf("expected checkpoint.restored = false on a held repair")
	}
	if episode.VerificationStatus != "completed" {
		t.Fatalf("verification status = %q, want completed", episode.VerificationStatus)
	}
	if episode.StrikeStanding.TargetFingerprint == "" {
		t.Fatalf("expected strike standing to carry a target fingerprint")
	}
	if strings.TrimSpace(episode.Cost.SpawnRunID) == "" {
		t.Fatalf("expected a non-empty spawn run id cost reference")
	}

	// The episode must carry no currency amount -- only a reference to the
	// ledger keys (the spawn run id). Round-trip through JSON and confirm no
	// numeric/monetary-shaped field exists on the Cost object.
	data, err := json.Marshal(episode.Cost)
	if err != nil {
		t.Fatalf("marshal cost reference: %v", err)
	}
	var costFields map[string]interface{}
	if err := json.Unmarshal(data, &costFields); err != nil {
		t.Fatalf("unmarshal cost reference: %v", err)
	}
	for key, value := range costFields {
		if _, isNumber := value.(float64); isNumber {
			t.Fatalf("cost reference field %q is numeric (%v) -- episode must carry no currency amount", key, value)
		}
	}
}

func TestSwarmEpisodeRereadNeverChangesStrikeTruth(t *testing.T) {
	target := "Episode fixture target two"
	swarmID := runSwarmDestroyFixture(t, target)

	beforeDigest := swarmStrikeHistoryDigestFor(t, target)

	firstRead, ok := loadSwarmEpisode(store, swarmID)
	if !ok {
		t.Fatalf("could not load episode for %s (first read)", swarmID)
	}
	midDigest := swarmStrikeHistoryDigestFor(t, target)

	secondRead, ok := loadSwarmEpisode(store, swarmID)
	if !ok {
		t.Fatalf("could not load episode for %s (second read)", swarmID)
	}
	afterDigest := swarmStrikeHistoryDigestFor(t, target)

	if !reflect.DeepEqual(firstRead, secondRead) {
		t.Fatalf("reading the episode twice produced different values:\nfirst:  %+v\nsecond: %+v", firstRead, secondRead)
	}
	if beforeDigest != midDigest || midDigest != afterDigest {
		t.Fatalf("strike history changed across episode reads:\nbefore: %s\nmid:    %s\nafter:  %s", beforeDigest, midDigest, afterDigest)
	}
}

func TestInterruptedSwarmRunPersistsOneInterruptedEpisode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

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

	goal := "Fix a bug"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Bug fix",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Fix the bug", Status: colony.TaskPending}},
			}},
		},
	})

	originalInvoker := newSwarmWorkerInvoker
	invoker := &swarmTestInvoker{}
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	defer func() { newSwarmWorkerInvoker = originalInvoker }()

	// Cancel the run's own context right after the investigation wave
	// completes and before the fix wave dispatches -- a genuine ctx.Err()
	// surfaced by the real public Swarm path, deterministically, rather than
	// a timing-dependent sleep racing a timeout.
	originalInterrupt := swarmMidRunInterruptFunc
	swarmMidRunInterruptFunc = func(cancel func()) { cancel() }
	defer func() { swarmMidRunInterruptFunc = originalInterrupt }()

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	target := "Episode fixture interrupted target"
	rootCmd.SetArgs([]string{"swarm", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm cobra execution itself failed: %v", err)
	}
	combined := outBuf.String() + errBuf.String()
	if strings.TrimSpace(combined) == "" {
		t.Fatalf("expected some output for an interrupted swarm run, got nothing")
	}
	if !strings.Contains(combined, "\"ok\":false") {
		t.Fatalf("expected an error envelope for an interrupted swarm run, got: %s", combined)
	}

	swarmsDir := filepath.Join(dataDir, "swarms")
	entries, err := os.ReadDir(swarmsDir)
	if err != nil {
		t.Fatalf("read swarms dir: %v", err)
	}
	var swarmID string
	for _, e := range entries {
		if e.IsDir() {
			swarmID = e.Name()
		}
	}
	if swarmID == "" {
		t.Fatalf("expected exactly one swarm workspace, found none")
	}

	episode, ok := loadSwarmEpisode(store, swarmID)
	if !ok {
		t.Fatalf("expected an interrupted episode to have been persisted for %s", swarmID)
	}
	if episode.Status != swarmEpisodeStatusInterrupted {
		t.Fatalf("episode status = %q, want %q", episode.Status, swarmEpisodeStatusInterrupted)
	}
	if episode.InterruptedStage == "" {
		t.Fatalf("expected interrupted episode to name the stage it stopped at")
	}
	reported := 0
	for _, lens := range episode.Lenses {
		if lens.Reported {
			reported++
		}
	}
	if reported != 4 {
		t.Fatalf("expected all 4 investigation lenses to have reported before the interruption, got %d", reported)
	}
	if episode.VerificationStatus != "not_run" {
		t.Fatalf("verification status = %q, want not_run for an interrupted episode", episode.VerificationStatus)
	}

	// Only one episode file exists for this run.
	matches, err := filepath.Glob(filepath.Join(swarmsDir, "*", "episode.json"))
	if err != nil {
		t.Fatalf("glob episode files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one episode file, found %d: %v", len(matches), matches)
	}
}

func TestSwarmEpisodeIdentityComesFromIssuance(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	bogusID := "../not-a-real-swarm-id"
	err := persistSwarmEpisode(store, swarmEpisodeRecord{
		SwarmID: bogusID,
		Target:  "some target",
		Status:  swarmEpisodeStatusCompleted,
	})
	if err == nil {
		t.Fatalf("expected persistSwarmEpisode to refuse a caller-supplied identifier")
	}
	if !strings.Contains(err.Error(), bogusID) && !strings.Contains(err.Error(), "not-a-real-swarm-id") {
		t.Fatalf("expected refusal to name the identifier %q, got: %v", bogusID, err)
	}

	// Nothing should have been written anywhere under the swarms directory.
	swarmsDir := filepath.Join(store.BasePath(), "swarms")
	matches, _ := filepath.Glob(filepath.Join(swarmsDir, "*", "episode.json"))
	if len(matches) != 0 {
		t.Fatalf("expected no episode file to be written on a refused identity, found: %v", matches)
	}

	// A caller-supplied identifier that merely doesn't follow the
	// runtime-issued convention (no "swarm-" prefix) is refused too.
	badFormat := "not-swarm-shaped"
	err = persistSwarmEpisode(store, swarmEpisodeRecord{
		SwarmID: badFormat,
		Target:  "some target",
		Status:  swarmEpisodeStatusCompleted,
	})
	if err == nil {
		t.Fatalf("expected persistSwarmEpisode to refuse a non-conventional identifier")
	}
	if !strings.Contains(err.Error(), badFormat) {
		t.Fatalf("expected refusal to name the identifier %q, got: %v", badFormat, err)
	}

	// A runtime-issued-shaped identifier is accepted and round-trips.
	goodID := newSwarmRunID(time.Now().UTC())
	if err := persistSwarmEpisode(store, swarmEpisodeRecord{
		SwarmID: goodID,
		Target:  "some target",
		Status:  swarmEpisodeStatusCompleted,
	}); err != nil {
		t.Fatalf("persistSwarmEpisode with a valid runtime-issued id: %v", err)
	}
	if _, ok := loadSwarmEpisode(store, goodID); !ok {
		t.Fatalf("expected episode %s to have been written", goodID)
	}
}

// --- Task 2: retention and cleanup that act on a named episode ---

func mustSaveSwarmResultFixture(t *testing.T, swarmID, target, status string, completedAt time.Time) {
	t.Helper()
	if err := saveSwarmResultRecord(store, swarmResultRecord{
		SwarmID:     swarmID,
		Target:      target,
		Status:      status,
		CompletedAt: completedAt.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("save swarm result fixture: %v", err)
	}
}

func mustPersistSwarmEpisodeFixture(t *testing.T, swarmID, target string, endedAt time.Time) swarmEpisodeRecord {
	t.Helper()
	record := buildSwarmEpisodeRecord(swarmEpisodeBuildParams{
		SwarmID:            swarmID,
		Target:             target,
		Status:             swarmEpisodeStatusCompleted,
		StartedAt:          endedAt.Add(-time.Minute),
		EndedAt:            endedAt,
		Comparison:         swarmComparison{},
		VerificationStatus: "completed",
	})
	if err := persistSwarmEpisode(store, record); err != nil {
		t.Fatalf("persist swarm episode fixture: %v", err)
	}
	loaded, ok := loadSwarmEpisode(store, swarmID)
	if !ok {
		t.Fatalf("could not reload persisted episode fixture %s", swarmID)
	}
	return loaded
}

// storeDirDigest hashes the sorted (relative path, content) pairs of every
// regular file under dir -- a whole-tree content digest used to prove a
// read-only operation changed nothing on disk.
func storeDirDigest(t *testing.T, dir string) string {
	t.Helper()
	var paths []string
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, rel)
		return nil
	}); err != nil {
		t.Fatalf("walk store dir: %v", err)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestSwarmRetentionNeverRemovesTheLatestOrAStrikeEpisode(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)

	// Target A: two ordinary episodes, no live strike sequence. The older is
	// eligible; the newer (most recent) is not.
	targetA := "Retention target ordinary"
	swarmA1 := newSwarmRunID(base)
	swarmA2 := newSwarmRunID(base.Add(time.Minute))
	mustSaveSwarmResultFixture(t, swarmA1, targetA, "completed", base)
	mustSaveSwarmResultFixture(t, swarmA2, targetA, "completed", base.Add(time.Minute))
	mustPersistSwarmEpisodeFixture(t, swarmA1, targetA, base)
	mustPersistSwarmEpisodeFixture(t, swarmA2, targetA, base.Add(time.Minute))

	// Target B: two failed episodes, no completed reset in between -- both
	// are part of B's currently unresolved (2-strike) sequence.
	targetB := "Retention target unresolved strikes"
	swarmB1 := newSwarmRunID(base.Add(2 * time.Minute))
	swarmB2 := newSwarmRunID(base.Add(3 * time.Minute))
	mustSaveSwarmResultFixture(t, swarmB1, targetB, "failed", base.Add(2*time.Minute))
	mustSaveSwarmResultFixture(t, swarmB2, targetB, "failed", base.Add(3*time.Minute))
	mustPersistSwarmEpisodeFixture(t, swarmB1, targetB, base.Add(2*time.Minute))
	mustPersistSwarmEpisodeFixture(t, swarmB2, targetB, base.Add(3*time.Minute))

	plan, err := planSwarmEpisodeRetention(s, base.Add(time.Hour))
	if err != nil {
		t.Fatalf("plan swarm episode retention: %v", err)
	}
	byID := map[string]swarmEpisodeRetentionEntry{}
	for _, entry := range plan {
		byID[entry.SwarmID] = entry
	}

	if entry, ok := byID[swarmA1]; !ok || !entry.Eligible {
		t.Fatalf("expected the older ordinary episode to be eligible: %+v", entry)
	}
	if entry, ok := byID[swarmA2]; !ok || entry.Eligible {
		t.Fatalf("expected the most recent episode for its target to be ineligible: %+v", entry)
	} else if !strings.Contains(entry.Reason, "recent") {
		t.Fatalf("expected a stated 'most recent' reason, got %q", entry.Reason)
	}
	if entry, ok := byID[swarmB1]; !ok || entry.Eligible {
		t.Fatalf("expected an episode in an unresolved strike sequence to be ineligible: %+v", entry)
	} else if !strings.Contains(entry.Reason, "strike") {
		t.Fatalf("expected a stated strike-sequence reason, got %q", entry.Reason)
	}
	if entry, ok := byID[swarmB2]; !ok || entry.Eligible {
		t.Fatalf("expected the other episode in the unresolved strike sequence to be ineligible: %+v", entry)
	}
}

func TestSwarmRemovalRequiresIdentifierAndDigest(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	target := "Removal target requires identifier"
	oldID := newSwarmRunID(base)
	newID := newSwarmRunID(base.Add(time.Minute))
	mustSaveSwarmResultFixture(t, oldID, target, "completed", base)
	mustSaveSwarmResultFixture(t, newID, target, "completed", base.Add(time.Minute))
	mustPersistSwarmEpisodeFixture(t, oldID, target, base)
	mustPersistSwarmEpisodeFixture(t, newID, target, base.Add(time.Minute))

	plan, err := planSwarmEpisodeRetention(s, base.Add(time.Hour))
	if err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	var digest string
	for _, entry := range plan {
		if entry.SwarmID == oldID {
			digest = entry.Digest
		}
	}
	if digest == "" {
		t.Fatalf("expected a digest for the eligible older episode")
	}

	// A prefix instead of the exact identifier is refused, naming what was
	// supplied.
	prefix := oldID[:len(oldID)-2]
	if _, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: prefix, Digest: digest}}); err == nil {
		t.Fatalf("expected removal by prefix to be refused")
	} else if !strings.Contains(err.Error(), prefix) {
		t.Fatalf("expected refusal to name the supplied identifier %q, got: %v", prefix, err)
	}

	// A glob is refused.
	if _, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: "swarm-*", Digest: digest}}); err == nil {
		t.Fatalf("expected removal by glob to be refused")
	}

	// A missing digest is refused.
	if _, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: oldID, Digest: ""}}); err == nil {
		t.Fatalf("expected removal without a digest to be refused")
	}

	// Exact identifier with the correct digest succeeds.
	result, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: oldID, Digest: digest}})
	if err != nil {
		t.Fatalf("expected exact-identifier removal to succeed: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != oldID {
		t.Fatalf("expected removed = [%s], got %v", oldID, result.Removed)
	}
	if _, ok := loadSwarmEpisode(s, oldID); ok {
		t.Fatalf("expected episode %s to be gone after removal", oldID)
	}
}

func TestSwarmRemovalRefusesChangedDigest(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	target := "Removal target changed digest"
	oldID := newSwarmRunID(base)
	newID := newSwarmRunID(base.Add(time.Minute))
	mustSaveSwarmResultFixture(t, oldID, target, "completed", base)
	mustSaveSwarmResultFixture(t, newID, target, "completed", base.Add(time.Minute))
	mustPersistSwarmEpisodeFixture(t, oldID, target, base)
	mustPersistSwarmEpisodeFixture(t, newID, target, base.Add(time.Minute))

	plan, err := planSwarmEpisodeRetention(s, base.Add(time.Hour))
	if err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	var staleDigest string
	for _, entry := range plan {
		if entry.SwarmID == oldID {
			staleDigest = entry.Digest
		}
	}
	if staleDigest == "" {
		t.Fatalf("expected a digest for the eligible episode")
	}

	// The episode changes after the preview (re-persisted with a different
	// verification status), so its digest no longer matches the preview.
	changed := mustPersistSwarmEpisodeFixture(t, oldID, target, base)
	changed.VerificationStatus = "failed"
	if err := persistSwarmEpisode(s, changed); err != nil {
		t.Fatalf("re-persist changed episode: %v", err)
	}

	before, err := os.ReadFile(filepath.Join(s.BasePath(), "swarms", oldID, "episode.json"))
	if err != nil {
		t.Fatalf("read episode before refused removal: %v", err)
	}

	if _, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: oldID, Digest: staleDigest}}); err == nil {
		t.Fatalf("expected removal with a stale digest to be refused")
	} else if !strings.Contains(err.Error(), oldID) {
		t.Fatalf("expected refusal to name the mismatched episode %q, got: %v", oldID, err)
	}

	after, err := os.ReadFile(filepath.Join(s.BasePath(), "swarms", oldID, "episode.json"))
	if err != nil {
		t.Fatalf("read episode after refused removal: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("a refused removal must delete nothing -- episode content changed")
	}
}

func TestSwarmRetentionPreviewIsReadOnly(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	target := "Retention preview read only target"
	swarmID := newSwarmRunID(base)
	mustSaveSwarmResultFixture(t, swarmID, target, "completed", base)
	mustPersistSwarmEpisodeFixture(t, swarmID, target, base)

	before := storeDirDigest(t, s.BasePath())
	if _, err := planSwarmEpisodeRetention(s, base.Add(time.Hour)); err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	after := storeDirDigest(t, s.BasePath())
	if before != after {
		t.Fatalf("planSwarmEpisodeRetention must not write anything: digest changed from %s to %s", before, after)
	}
}

func TestSwarmRemovalLeavesStrikeHistoryUnchanged(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	target := "Removal leaves strike history unchanged"
	oldID := newSwarmRunID(base)
	newID := newSwarmRunID(base.Add(time.Minute))
	mustSaveSwarmResultFixture(t, oldID, target, "completed", base)
	mustSaveSwarmResultFixture(t, newID, target, "completed", base.Add(time.Minute))
	mustPersistSwarmEpisodeFixture(t, oldID, target, base)
	mustPersistSwarmEpisodeFixture(t, newID, target, base.Add(time.Minute))

	beforeHistory := swarmStrikeHistoryDigestFor(t, target)

	plan, err := planSwarmEpisodeRetention(s, base.Add(time.Hour))
	if err != nil {
		t.Fatalf("plan retention: %v", err)
	}
	var digest string
	for _, entry := range plan {
		if entry.SwarmID == oldID {
			digest = entry.Digest
		}
	}
	if digest == "" {
		t.Fatalf("expected a digest for the eligible episode")
	}
	if _, err := removeSwarmEpisodes(s, []swarmEpisodeRemovalRequest{{SwarmID: oldID, Digest: digest}}); err != nil {
		t.Fatalf("remove swarm episode: %v", err)
	}

	afterHistory := swarmStrikeHistoryDigestFor(t, target)
	if beforeHistory != afterHistory {
		t.Fatalf("removing an episode changed strike history:\nbefore: %s\nafter:  %s", beforeHistory, afterHistory)
	}

	resultPath := filepath.Join(s.BasePath(), "swarms", oldID, "result.json")
	if _, err := os.Stat(resultPath); err != nil {
		t.Fatalf("expected result.json to survive episode removal: %v", err)
	}
}
