package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
