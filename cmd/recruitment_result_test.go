package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

func newRecruitmentResultFixture(recruitmentID string) recruitmentResult {
	return recruitmentResult{
		SchemaVersion:  RecruitmentResultSchemaVersion,
		RecruitmentID:  recruitmentID,
		IntentID:       recruitmentID,
		ChildName:      "Fixture-" + recruitmentID,
		ParentName:     "A1",
		TerminalStatus: RecruitmentTerminalStatusCompleted,
		Summary:        "did the thing",
		Transaction: colony.LifecycleTransactionReference{
			ID:    recruitmentID,
			Stage: colony.TransactionStageCommitted,
		},
	}
}

// TestRecruitmentResultBinding covers 203-07-PLAN.md Task 1's full field set
// and its replay/conflict/vocabulary/generation guarantees.
func TestRecruitmentResultBinding(t *testing.T) {
	t.Run("declares the schema version and terminal status accessor", func(t *testing.T) {
		if RecruitmentResultSchemaVersion == "" {
			t.Fatal("RecruitmentResultSchemaVersion must be non-empty")
		}
		statuses := recruitmentTerminalStatuses()
		if len(statuses) == 0 {
			t.Fatal("recruitmentTerminalStatuses() must return at least one status")
		}
		for _, want := range []string{RecruitmentTerminalStatusCompleted, RecruitmentTerminalStatusFailed, RecruitmentTerminalStatusTimeout} {
			found := false
			for _, s := range statuses {
				if s == want {
					found = true
				}
			}
			if !found {
				t.Fatalf("recruitmentTerminalStatuses() missing %q", want)
			}
		}
	})

	t.Run("a replay with identical content leaves the store byte-identical and returns the first receipt", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("identical-replay-1")
		first, err := bindRecruitmentResult(result)
		if err != nil {
			t.Fatalf("first bind: %v", err)
		}
		if first.Receipt == nil || first.Receipt.ID == "" {
			t.Fatalf("expected a populated receipt on first bind, got %+v", first)
		}
		before, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after first bind: %v", err)
		}

		second, err := bindRecruitmentResult(result)
		if err != nil {
			t.Fatalf("second bind (replay) returned an unexpected error: %v", err)
		}
		after, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after replay: %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("recruitment/results.json changed on replay:\nbefore=%s\nafter=%s", before, after)
		}
		if second.Receipt == nil || second.Receipt.ID != first.Receipt.ID {
			t.Fatalf("replay did not return the first bind's receipt: first=%+v second=%+v", first.Receipt, second.Receipt)
		}
	})

	t.Run("a replay with one changed field is refused as a conflict naming that field", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("conflicting-replay-1")
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		before, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after first bind: %v", err)
		}

		changed := result
		changed.Summary = "a completely different summary"
		_, err = bindRecruitmentResult(changed)
		if err == nil {
			t.Fatal("expected an error binding a conflicting replay, got nil")
		}
		if !strings.Contains(err.Error(), "summary") {
			t.Fatalf("expected the conflict error to name the differing field %q, got: %v", "summary", err)
		}
		after, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after conflicting replay: %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("recruitment/results.json changed on a refused conflicting replay:\nbefore=%s\nafter=%s", before, after)
		}
	})

	t.Run("an out-of-vocabulary terminal status is refused and named", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("bad-status-1")
		result.TerminalStatus = "definitely-not-a-real-status"
		_, err := bindRecruitmentResult(result)
		if err == nil {
			t.Fatal("expected an error binding an out-of-vocabulary terminal status, got nil")
		}
		if !strings.Contains(err.Error(), "definitely-not-a-real-status") {
			t.Fatalf("expected the error to name the offending status, got: %v", err)
		}
	})

	t.Run("a stale execution generation is refused and names both generations", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("stale-generation-1")
		result.ExecutionGeneration = "5"
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("first bind: %v", err)
		}

		stale := result
		stale.ExecutionGeneration = "3"
		_, err := bindRecruitmentResult(stale)
		if err == nil {
			t.Fatal("expected an error binding a stale execution generation, got nil")
		}
		if !strings.Contains(err.Error(), `"3"`) || !strings.Contains(err.Error(), `"5"`) {
			t.Fatalf("expected the error to name both generations (3 and 5), got: %v", err)
		}
	})

	t.Run("a source scan finds no second deduplication structure in cmd/recruitment_result.go", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "recruitment_result.go", nil, 0)
		if err != nil {
			t.Fatalf("parse cmd/recruitment_result.go: %v", err)
		}
		mapDecls := 0
		ast.Inspect(file, func(n ast.Node) bool {
			if _, ok := n.(*ast.MapType); ok {
				mapDecls++
			}
			return true
		})
		if mapDecls != 0 {
			t.Fatalf("cmd/recruitment_result.go declares %d map type(s) -- the only permitted dedupe structure is the store's own recruitment/results.json entries list, matched by RecruitmentID equality", mapDecls)
		}
	})
}

// newRecruitmentEvidenceFixture writes a real evidence file to disk and
// returns a colony.LifecycleEvidence entry whose Digest matches its current
// content, mirroring how a real dispatch would compute one.
func newRecruitmentEvidenceFixture(t *testing.T, dir, name, content string) colony.LifecycleEvidence {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write evidence fixture %s: %v", path, err)
	}
	sum := sha256.Sum256([]byte(content))
	return colony.LifecycleEvidence{
		ID:     name,
		Kind:   "file",
		Source: path,
		Digest: hex.EncodeToString(sum[:]),
	}
}

// TestRecruitmentRecovery covers 203-07-PLAN.md Task 2's five named recovery
// classes, each derived from real durable state rather than a typed
// fixture.
func TestRecruitmentRecovery(t *testing.T) {
	t.Run("recruitmentRecoveryClasses returns exactly five members, each with a next action", func(t *testing.T) {
		classes := recruitmentRecoveryClasses()
		if len(classes) != 5 {
			t.Fatalf("expected exactly 5 recovery classes, got %d: %v", len(classes), classes)
		}
		q := recruitmentRecoveryQuery{RecruitmentID: "next-action-fixture-1", ChildName: "Fixture-Child-1"}
		for _, class := range classes {
			action := recruitmentRecoveryNextAction(class, q)
			if strings.TrimSpace(action) == "" {
				t.Fatalf("recovery class %q has no next action -- every declared class must have exactly one", class)
			}
		}
	})

	t.Run("an amendment recorded via the real admission path with no result and no live process classifies as missing", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "MissingFixture-1", "help with x", 1); err != nil {
			t.Fatalf("seed real admission (RecordSpawn): %v", err)
		}
		// Simulate the dispatch process being killed before it could ever
		// call UpdateStatus or bindRecruitmentResult -- the entry stays at
		// its initial "spawned" status forever.

		class, next, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{
			RecruitmentID: "missing-recruitment-1",
			ChildName:     "MissingFixture-1",
		})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class != recruitmentRecoveryMissing {
			t.Fatalf("expected class %q, got %q", recruitmentRecoveryMissing, class)
		}
		if !strings.Contains(next, "missing-recruitment-1") {
			t.Fatalf("expected the next action to name the recruitment id, got: %s", next)
		}
	})

	t.Run("a child recorded as timed out with no bound result classifies as timed-out", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "TimedOutFixture-1", "help with x", 1); err != nil {
			t.Fatalf("seed real admission (RecordSpawn): %v", err)
		}
		if err := st.UpdateStatus("TimedOutFixture-1", "timeout", "deadline exceeded"); err != nil {
			t.Fatalf("record real timeout (UpdateStatus): %v", err)
		}

		class, _, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{
			RecruitmentID: "timed-out-recruitment-1",
			ChildName:     "TimedOutFixture-1",
		})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class != recruitmentRecoveryTimedOut {
			t.Fatalf("expected class %q, got %q", recruitmentRecoveryTimedOut, class)
		}
	})

	t.Run("a bound result with no incoming candidate classifies as replayed", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("replayed-recruitment-1")
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("bind: %v", err)
		}

		class, _, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{RecruitmentID: "replayed-recruitment-1"})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class != recruitmentRecoveryReplayed {
			t.Fatalf("expected class %q, got %q", recruitmentRecoveryReplayed, class)
		}
	})

	t.Run("a bound result with a conflicting incoming candidate classifies as duplicated", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("duplicated-recruitment-1")
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("bind: %v", err)
		}

		conflicting := result
		conflicting.Summary = "a different summary entirely"
		class, _, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{
			RecruitmentID: "duplicated-recruitment-1",
			Candidate:     &conflicting,
		})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class != recruitmentRecoveryDuplicated {
			t.Fatalf("expected class %q, got %q", recruitmentRecoveryDuplicated, class)
		}
	})

	t.Run("an altered evidence file produces altered and the store is byte-identical afterwards", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		evidenceDir := t.TempDir()
		evidence := newRecruitmentEvidenceFixture(t, evidenceDir, "output.txt", "original content")

		result := newRecruitmentResultFixture("altered-recruitment-1")
		result.Evidence = []colony.LifecycleEvidence{evidence}
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("bind: %v", err)
		}
		before, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after bind: %v", err)
		}

		// Tamper with the evidence file after it was bound.
		if err := os.WriteFile(evidence.Source, []byte("tampered content"), 0o644); err != nil {
			t.Fatalf("tamper with evidence file: %v", err)
		}

		class, next, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{RecruitmentID: "altered-recruitment-1"})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class != recruitmentRecoveryAltered {
			t.Fatalf("expected class %q, got %q", recruitmentRecoveryAltered, class)
		}
		if !strings.Contains(next, "altered-recruitment-1") {
			t.Fatalf("expected the next action to name the recruitment id, got: %s", next)
		}

		after, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after classification: %v", err)
		}
		if string(before) != string(after) {
			t.Fatalf("recruitment/results.json changed as a side effect of classification:\nbefore=%s\nafter=%s", before, after)
		}
	})

	t.Run("missing evidence on disk is unknown, not altered", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		result := newRecruitmentResultFixture("missing-evidence-recruitment-1")
		result.Evidence = []colony.LifecycleEvidence{{
			ID:     "gone.txt",
			Kind:   "file",
			Source: filepath.Join(t.TempDir(), "never-written.txt"),
			Digest: "0000000000000000000000000000000000000000000000000000000000000",
		}}
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("bind: %v", err)
		}

		class, _, err := classifyRecruitmentRecovery(recruitmentRecoveryQuery{RecruitmentID: "missing-evidence-recruitment-1"})
		if err != nil {
			t.Fatalf("classifyRecruitmentRecovery: %v", err)
		}
		if class == recruitmentRecoveryAltered {
			t.Fatalf("missing evidence must classify as unknown (falls through to replayed here), never altered -- Phase 199's rule")
		}
	})

	t.Run("aether recruit --status prints the class and one exact command", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf strings.Builder
		stdout = &buf

		result := newRecruitmentResultFixture("cli-status-recruitment-1")
		if _, err := bindRecruitmentResult(result); err != nil {
			t.Fatalf("bind: %v", err)
		}

		rootCmd.SetArgs([]string{"recruit", "--status", "cli-status-recruitment-1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("recruit --status returned an error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "replayed") {
			t.Fatalf("expected output to name the replayed class, got: %s", out)
		}
		if !strings.Contains(out, "aether recruit --status cli-status-recruitment-1") {
			t.Fatalf("expected output to carry one exact command, got: %s", out)
		}
	})
}
