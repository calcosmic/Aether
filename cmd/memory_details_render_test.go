package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
)

// ---------------------------------------------------------------------------
// Shared fixture for the memory-details drill-down tests (198.1-06).
// ---------------------------------------------------------------------------

const (
	memDetailsQueenSentence    = "run go vet ./cmd/ before go test ./cmd/ or the failure output is unreadable"
	memDetailsPendingSentence  = "cmd/deterministic_floor.go is shared by both check lanes; changing it changes both"
	memDetailsDeferredSentence = "queen promotion review is paused until the metrics dashboard ships"
	memDetailsFailureSentence  = "go build ./cmd/aether failed: undefined identifier newTestStore"
)

// seedMemoryDetailsFixture writes a QUEEN.md, learning-observations.json,
// and midden.json carrying the four sentences above into the given store.
func seedMemoryDetailsFixture(t *testing.T) {
	t.Helper()

	queenPath := localQueenPath()
	if queenPath == "" {
		t.Fatal("localQueenPath() returned empty -- store not initialized")
	}
	if err := os.MkdirAll(filepath.Dir(queenPath), 0o755); err != nil {
		t.Fatalf("mkdir queen dir: %v", err)
	}
	queenContent := "# QUEEN.md — Colony Wisdom Hub\n\n" +
		"## Patterns\n" +
		"- " + memDetailsQueenSentence + "\n"
	if err := os.WriteFile(queenPath, []byte(queenContent), 0o644); err != nil {
		t.Fatalf("write QUEEN.md: %v", err)
	}

	trust := 0.30
	learnings := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash:      "hash_pending",
				Content:          memDetailsPendingSentence,
				WisdomType:       "redirect",
				ObservationCount: 2,
				TrustScore:       &trust,
				LastSeen:         "2026-08-20T10:00:00Z",
			},
			{
				ContentHash:      "hash_deferred",
				Content:          memDetailsDeferredSentence,
				WisdomType:       "pattern",
				ObservationCount: 1,
				SourceType:       "deferred",
				LastSeen:         "2026-08-21T10:00:00Z",
			},
		},
	}
	if err := store.SaveJSON("learning-observations.json", learnings); err != nil {
		t.Fatalf("save learning-observations.json: %v", err)
	}

	midden := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{
				ID:        "midden_1",
				Timestamp: "2026-08-22T09:00:00Z",
				Category:  "build_failure",
				Source:    "worker:builder",
				Message:   memDetailsFailureSentence,
			},
		},
	}
	if err := store.SaveJSON("midden.json", midden); err != nil {
		t.Fatalf("save midden.json: %v", err)
	}

	// Pre-seed the first-run marker so invoking a command through rootCmd
	// doesn't print the first-run welcome banner ahead of the command's own
	// output -- checkAndEmitFirstRun (cmd/ux_firstrun.go) fires whenever a
	// store has neither COLONY_STATE.json nor this marker.
	if err := os.WriteFile(filepath.Join(store.BasePath(), ".welcomed"), []byte(""), 0o600); err != nil {
		t.Fatalf("write .welcomed marker: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 1: buildMemoryDetails carries real content
// ---------------------------------------------------------------------------

func TestMemoryDetailsCarriesTheActualSentences(t *testing.T) {
	s, _ := newTestStore(t)
	store = s

	seedMemoryDetailsFixture(t)

	details := buildMemoryDetails(s)

	foundWisdom := false
	for _, cat := range details.Wisdom {
		for _, entry := range cat.Entries {
			if entry == memDetailsQueenSentence {
				foundWisdom = true
			}
		}
	}
	if !foundWisdom {
		t.Errorf("expected QUEEN.md wisdom sentence %q in details.Wisdom, got %+v", memDetailsQueenSentence, details.Wisdom)
	}

	var pendingEntry *pendingPromotion
	for i := range details.Pending {
		if details.Pending[i].Content == memDetailsPendingSentence {
			pendingEntry = &details.Pending[i]
		}
	}
	if pendingEntry == nil {
		t.Fatalf("expected pending sentence %q in details.Pending, got %+v", memDetailsPendingSentence, details.Pending)
	}
	if pendingEntry.ObservationCount != 2 {
		t.Errorf("expected pending ObservationCount == 2, got %d", pendingEntry.ObservationCount)
	}

	foundDeferred := false
	for _, d := range details.Deferred {
		if d.Content == memDetailsDeferredSentence {
			foundDeferred = true
		}
	}
	if !foundDeferred {
		t.Errorf("expected deferred sentence %q in details.Deferred, got %+v", memDetailsDeferredSentence, details.Deferred)
	}

	foundFailure := false
	for _, f := range details.Failures {
		if f.Message == memDetailsFailureSentence {
			foundFailure = true
		}
	}
	if !foundFailure {
		t.Errorf("expected failure sentence %q in details.Failures, got %+v", memDetailsFailureSentence, details.Failures)
	}
}

func TestQueenMarkdownUpdatedAtFallsBackToModTime(t *testing.T) {
	t.Run("metadata block present", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "QUEEN.md")
		content := "# QUEEN.md\n\n## Wisdom\n- something\n\n" +
			"<!-- METADATA\n" +
			`{  "version": "2.0.0",  "last_evolved": "2026-04-01T17:32:04Z"}` + "\n" +
			"-->\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		got := queenMarkdownUpdatedAt(path)
		if got != "2026-04-01T17:32:04Z" {
			t.Fatalf("expected metadata last_evolved value, got %q", got)
		}
	})

	t.Run("no metadata block falls back to mtime", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "QUEEN.md")
		if err := os.WriteFile(path, []byte("# QUEEN.md\n\n## Wisdom\n- something\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		got := queenMarkdownUpdatedAt(path)
		if got == "" {
			t.Fatal("expected a non-empty mtime fallback")
		}
		if _, err := time.Parse(time.RFC3339, got); err != nil {
			t.Fatalf("expected parseable RFC3339 timestamp, got %q: %v", got, err)
		}
	})

	t.Run("missing file returns empty string", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "does-not-exist.md")
		got := queenMarkdownUpdatedAt(path)
		if got != "" {
			t.Fatalf("expected empty string for missing file, got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2: renderMemoryDetailsVisual + the retired alias
// ---------------------------------------------------------------------------

func TestMemoryDetailsRenderShowsTheSentencesNotTheCounts(t *testing.T) {
	s, _ := newTestStore(t)
	store = s
	seedMemoryDetailsFixture(t)

	details := buildMemoryDetails(s)
	rendered := renderMemoryDetailsVisual(details)

	for _, sentence := range []string{
		memDetailsQueenSentence,
		memDetailsPendingSentence,
		memDetailsDeferredSentence,
		memDetailsFailureSentence,
	} {
		if !strings.Contains(rendered, sentence) {
			t.Errorf("expected rendered output to contain %q, got:\n%s", sentence, rendered)
		}
	}
	if !strings.Contains(rendered, "seen 2 times") {
		t.Errorf("expected rendered output to contain \"seen 2 times\", got:\n%s", rendered)
	}
}

func TestMemoryDetailsJSONFlagStillReturnsTheEnvelope(t *testing.T) {
	bindCommandTestRepository(t)
	seedMemoryDetailsFixture(t)

	t.Run("--json forces the machine envelope", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		rootCmd.SetArgs([]string{"memory-details", "--json"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("memory-details --json returned error: %v", err)
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
			t.Fatalf("expected valid JSON output, got error %v, output:\n%s", err, buf.String())
		}
		if envelope["ok"] != true {
			t.Fatalf("expected ok:true, got %v", envelope)
		}
	})

	t.Run("visual mode without --json does not parse as JSON", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf
		t.Setenv("AETHER_OUTPUT_MODE", "visual")

		rootCmd.SetArgs([]string{"memory-details"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("memory-details returned error: %v", err)
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &envelope); err == nil {
			t.Fatalf("expected non-JSON visual output, but it parsed as JSON:\n%s", buf.String())
		}
	})
}

func TestMemoryDetailsIsNoLongerAnAliasOfMemoryMetrics(t *testing.T) {
	for _, alias := range memoryMetricsCmd.Aliases {
		if alias == "memory-details" {
			t.Fatalf("memoryMetricsCmd.Aliases still contains \"memory-details\": %v", memoryMetricsCmd.Aliases)
		}
	}

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "memory-details" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a command named \"memory-details\" registered on the root command")
	}
}

func TestMemoryDetailsWritesNothing(t *testing.T) {
	binding := bindCommandTestRepository(t)
	seedMemoryDetailsFixture(t)

	before := snapshotDirFiles(t, binding.Root)

	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	rootCmd.SetArgs([]string{"memory-details"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("memory-details returned error: %v", err)
	}

	after := snapshotDirFiles(t, binding.Root)

	if len(before) != len(after) {
		t.Fatalf("file count changed: before %d, after %d", len(before), len(after))
	}
	for path, beforeBytes := range before {
		afterBytes, ok := after[path]
		if !ok {
			t.Fatalf("file %q disappeared after running memory-details", path)
		}
		if !bytes.Equal(beforeBytes, afterBytes) {
			t.Fatalf("file %q changed after running memory-details (read-only command must write nothing)", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Fatalf("file %q appeared after running memory-details (read-only command must write nothing)", path)
		}
	}
}

// snapshotDirFiles walks root and returns every regular file's content keyed
// by its path relative to root, for a byte-for-byte before/after comparison.
//
// Skips .aether/locks/: pkg/storage.FileLocker creates a persistent lock
// file (open with O_CREATE, never removed) the first time any path is
// locked -- including a plain read via LoadJSON. That is the storage
// layer's own concurrency-safety bookkeeping, not colony memory content;
// counting a first-touch lock file as a "write" would fail this test for
// any command that reads a JSON file it has never read before, which is
// not what T-198.1-14 (an inspection command must not mutate state) means.
func snapshotDirFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	snapshot := make(map[string][]byte)
	locksDir := filepath.Join(root, ".aether", "locks")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == locksDir {
				return filepath.SkipDir
			}
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
		snapshot[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return snapshot
}

// ---------------------------------------------------------------------------
// Task 3: status instinct list labelled by strength, ordered by it
// ---------------------------------------------------------------------------

func TestStatusListsTheStrongestInstinctFirst(t *testing.T) {
	s, _ := newTestStore(t)
	store = s

	now := time.Now().UTC()
	oldCreated := now.AddDate(0, 0, -40).Format(time.RFC3339)
	newCreated := now.Format(time.RFC3339)

	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_weak_new",
				Trigger:    "weak trigger",
				Action:     "weak action",
				Domain:     "testing",
				Confidence: 0.30,
				TrustScore: 0.25,
				TrustTier:  "emerging",
				Provenance: colony.InstinctProvenance{
					CreatedAt: newCreated,
				},
			},
			{
				ID:         "inst_strong_old",
				Trigger:    "strong trigger",
				Action:     "strong action",
				Domain:     "testing",
				Confidence: 0.85,
				TrustScore: 0.80,
				TrustTier:  "canonical",
				Provenance: colony.InstinctProvenance{
					CreatedAt: oldCreated,
				},
				ApplicationHistory: []interface{}{
					map[string]interface{}{"timestamp": now.AddDate(0, 0, -20).Format(time.RFC3339), "success": true},
					map[string]interface{}{"timestamp": now.AddDate(0, 0, -10).Format(time.RFC3339), "success": true},
					map[string]interface{}{"timestamp": now.AddDate(0, 0, -1).Format(time.RFC3339), "success": true},
				},
			},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("save instincts.json: %v", err)
	}

	// Derive the expected order the way the runtime derives it: score each
	// entry with memory.InstinctUsefulnessScore and sort descending. This is
	// what keeps the test from going stale if the weighting changes.
	scored := make([]colony.InstinctEntry, len(file.Instincts))
	copy(scored, file.Instincts)
	sort.Slice(scored, func(i, j int) bool {
		return memory.InstinctUsefulnessScore(scored[i], now) > memory.InstinctUsefulnessScore(scored[j], now)
	})

	got := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 instincts, got %d", len(got))
	}
	for i, entry := range scored {
		if got[i].ID != entry.ID {
			t.Fatalf("expected ranked order %v, got %v", []string{scored[0].ID, scored[1].ID}, []string{got[0].ID, got[1].ID})
		}
	}
	if got[0].ID != "inst_strong_old" {
		t.Fatalf("expected the older, stronger instinct first, got %+v", got)
	}
}

func TestStatusInstinctHeadingMatchesTheRanking(t *testing.T) {
	binding := bindCommandTestRepository(t)
	s := binding.Store

	file := colony.InstinctsFile{
		Version: "1.0",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_a",
				Trigger:    "trigger a",
				Action:     "action a",
				Domain:     "testing",
				Confidence: 0.7,
				TrustScore: 0.7,
				TrustTier:  "trusted",
				Provenance: colony.InstinctProvenance{CreatedAt: time.Now().UTC().Format(time.RFC3339)},
			},
		},
	}
	if err := s.SaveJSON("instincts.json", file); err != nil {
		t.Fatalf("save instincts.json: %v", err)
	}

	goal := "test colony goal"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Goal: &goal}); err != nil {
		t.Fatalf("save COLONY_STATE.json: %v", err)
	}

	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Memory") {
		t.Fatalf("expected lifecycle Memory heading, got:\n%s", output)
	}

	// Source scan: no non-comment line in cmd/status.go may still carry the
	// old, now-false, recency-implying heading.
	data, err := os.ReadFile("status.go")
	if err != nil {
		t.Fatalf("read status.go: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, "Recent Instincts") {
			t.Fatalf("cmd/status.go still labels the ranked instinct list by recency on a non-comment line: %q", line)
		}
	}
}
