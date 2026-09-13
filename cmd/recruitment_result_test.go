package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

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
