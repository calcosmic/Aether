package cmd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// A dry run that writes is not a dry run. Both consolidation commands used to
// mutate on --dry-run: consolidation-phase-end routed straight into the
// mutating service (trust scores irreversibly decayed, instincts archived),
// and consolidation-seal ran the full pipeline unconditionally, promoting
// entries into QUEEN.md during a "preview". These tests hash every file the
// pipeline can touch, run the dry-run commands, and assert nothing changed.

func hashFileForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func seedConsolidationFixture(t *testing.T) (dataDir string) {
	t.Helper()
	s, _ := newTestStore(t)
	store = s
	// The store writes under <tmp>/.aether/data — watch THAT directory, not
	// the tmp root. The first version of these tests watched the tmp root,
	// where every path was permanently "absent", so the purity assertions
	// passed vacuously. TestConsolidationRealRunStillMutates exists precisely
	// to catch that class of self-deception.
	dataDir = s.BasePath()

	score := 0.9
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{
		{
			ID:         "inst_1_aaaaaa",
			Trigger:    "observed pattern in pkg/storage/store.go",
			Action:     "prefer atomic writes in pkg/storage/store.go",
			TrustScore: 0.9,
			TrustTier:  "trusted",
			Confidence: 0.85,
		},
		{
			ID:         "inst_2_bbbbbb",
			Trigger:    "stale pattern in cmd/old.go",
			Action:     "stale guidance for cmd/old.go",
			TrustScore: 0.05, // below the archive floor — bait for a mutating run
			TrustTier:  "untrusted",
			Confidence: 0.2,
		},
	}}); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{
		{
			ContentHash:      "sha256:obs1",
			Content:          "repeated failure signature in cmd/codex_build.go",
			WisdomType:       "pattern",
			SourceType:       "observation",
			ObservationCount: 3,
			TrustScore:       &score,
		},
	}}); err != nil {
		t.Fatalf("seed observations: %v", err)
	}
	return dataDir
}

func assertConsolidationDryRunIsPure(t *testing.T, command string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	dataDir := seedConsolidationFixture(t)

	// consolidationQueenPath() (Task 1 of this plan) moved the consolidation
	// promotion target OUT of dataDir (.aether/data) to the parent .aether
	// directory's QUEEN.md -- the file colony-prime actually reads. Without
	// watching it here, a dry run that wrote it would go undetected: the
	// same class of vacuous-pass bug this file's own comment already warns
	// about for the store-relative path.
	//
	// We do not widen the snapshot to a full recursive walk of the parent
	// .aether directory: storage.Store's FileLocker creates a sibling
	// .aether/locks/<name>.lock file the first time any path is locked, and
	// this file legitimately appears fresh on every run (dry or not) the
	// first time QUEEN.md is touched by the locker -- a recursive snapshot
	// would flag that as a false-positive mutation. Watching the exact
	// localQueenPath() file directly avoids that noise while still covering
	// the relocated target.
	watched := []string{
		filepath.Join(dataDir, "instincts.json"),
		filepath.Join(dataDir, "learning-observations.json"),
		filepath.Join(dataDir, "QUEEN.md"),
		filepath.Join(dataDir, "events.jsonl"),
		filepath.Join(filepath.Dir(dataDir), "QUEEN.md"),
	}
	before := make(map[string]string, len(watched))
	for _, p := range watched {
		before[p] = hashFileForTest(t, p)
	}

	rootCmd.SetArgs([]string{command, "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%s --dry-run failed: %v", command, err)
	}

	for _, p := range watched {
		if after := hashFileForTest(t, p); after != before[p] {
			t.Errorf("%s --dry-run mutated %s (before=%s after=%s)", command, filepath.Base(p), before[p][:12], after[:12])
		}
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("%s --dry-run not ok: %s", command, buf.String())
	}
}

func TestConsolidationPhaseEndDryRunDoesNotMutate(t *testing.T) {
	assertConsolidationDryRunIsPure(t, "consolidation-phase-end")
}

func TestConsolidationSealDryRunDoesNotMutate(t *testing.T) {
	assertConsolidationDryRunIsPure(t, "consolidation-seal")
}

// TestConsolidationDryRunDoesNotCreateInstinctsSection pins that
// ensureQueenInstinctsSection() (Task 1) is never reached from the dry-run
// branch of consolidation-phase-end. Starting from a local QUEEN.md in
// legacy format (no "## Instincts" header), a dry run must leave the
// header absent -- creating a section header is itself a mutation, and
// "Report without modifying" must mean it.
func TestConsolidationDryRunDoesNotCreateInstinctsSection(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	dataDir := seedConsolidationFixture(t)
	localQueen := filepath.Join(filepath.Dir(dataDir), "QUEEN.md")
	if err := writeLocalQueenText(legacyQueenDefaultContentFixture); err != nil {
		t.Fatalf("seed legacy local QUEEN.md: %v", err)
	}
	before, err := os.ReadFile(localQueen)
	if err != nil {
		t.Fatalf("read seeded local QUEEN.md: %v", err)
	}
	if strings.Contains(string(before), "## Instincts") {
		t.Fatalf("fixture already contains '## Instincts'; fixture is invalid for this test")
	}

	rootCmd.SetArgs([]string{"consolidation-phase-end", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("consolidation-phase-end --dry-run failed: %v", err)
	}

	after, err := os.ReadFile(localQueen)
	if err != nil {
		t.Fatalf("read local QUEEN.md after dry run: %v", err)
	}
	if strings.Contains(string(after), "## Instincts") {
		t.Fatalf("consolidation-phase-end --dry-run must not create the '## Instincts' section, got:\n%s", string(after))
	}
	if string(after) != string(before) {
		t.Fatalf("consolidation-phase-end --dry-run mutated local QUEEN.md.\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}
}

// TestConsolidationRealRunStillMutates proves two things at once: the real
// (non-dry) path still performs its writes — the fix did not neuter it — and
// the watched-files harness above genuinely detects mutation, so the dry-run
// purity tests are not vacuously green.
func TestConsolidationRealRunStillMutates(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	dataDir := seedConsolidationFixture(t)
	instincts := filepath.Join(dataDir, "instincts.json")
	before := hashFileForTest(t, instincts)

	rootCmd.SetArgs([]string{"consolidation-phase-end"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("consolidation-phase-end failed: %v", err)
	}

	if after := hashFileForTest(t, instincts); after == before {
		t.Fatal("real consolidation run left instincts.json untouched; either the pipeline was neutered or the harness cannot detect writes")
	}
}
