package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/llm"
)

// Phase 196 plan 02, D-05: one authoritative token count.
//
// Two accounting paths carry a run's tokens. The provider parser
// (codex.ParseUsage) reads a raw provider event; the agent pool converts the
// chat-model client's streamed usage. Before this plan the second was
// cache-blind, so the same run produced two different totals — and a cost line
// rendered over two disagreeing sources is a confidently wrong number, which is
// worse than no number.
//
// This test does NOT check that the code agrees with itself. It carries ONE set
// of four raw column values through BOTH lanes and compares each independently
// produced total against a third value written by hand.

// wantBilledTotal is the hand-written literal both lanes must produce for the
// provider's documented disjoint-column example. It is written out here once,
// summed by hand, and never computed:
//
//	    50  input
//	100000  cache read
//	  2000  cache creation
//	   500  output
//	------
//	102550
const wantBilledTotal int64 = 102550

// rawColumns is the one set of column values both lanes carry.
const (
	rawInputTokens         int64 = 50
	rawCacheReadTokens     int64 = 100000
	rawCacheCreationTokens int64 = 2000
	rawOutputTokens        int64 = 500
)

// providerEventWithDocumentedColumns is a real provider terminal event
// carrying exactly the four raw values above.
const providerEventWithDocumentedColumns = `{"type":"result","usage":{"input_tokens":50,"cache_read_input_tokens":100000,"cache_creation_input_tokens":2000,"output_tokens":500}}`

func TestBothAccountingPathsAgreeOnTheTotal(t *testing.T) {
	// Lane A: the agent pool's conversion of the chat-model client's usage.
	poolLane := agent.WorkerUsageFromStreamUsage(llm.Usage{
		InputTokens:              rawInputTokens,
		CacheReadInputTokens:     rawCacheReadTokens,
		CacheCreationInputTokens: rawCacheCreationTokens,
		OutputTokens:             rawOutputTokens,
	}, "claude-sonnet-4-20250514")

	// Lane B: the provider parser.
	parserLane, ok := codex.ParseUsage(providerEventWithDocumentedColumns)
	if !ok {
		t.Fatal("the provider parser reported no usage for a real terminal event")
	}

	poolTotal := poolLane.BilledTotalTokens()
	parserTotal := parserLane.BilledTotalTokens()

	if poolTotal != wantBilledTotal {
		t.Errorf("the pool lane totals %d for the documented columns, want %d", poolTotal, wantBilledTotal)
	}
	if parserTotal != wantBilledTotal {
		t.Errorf("the provider-parser lane totals %d for the documented columns, want %d", parserTotal, wantBilledTotal)
	}
	if poolTotal != parserTotal {
		t.Errorf("the two accounting paths disagree on the same run: pool %d, provider parser %d — "+
			"a cost line rendered over two disagreeing sources is a confidently wrong number (D-05)", poolTotal, parserTotal)
	}

	t.Run("dropping a column on either lane breaks the agreement", func(t *testing.T) {
		// Proof this test is not vacuous: the equality above holds because
		// both lanes carry every column, not because both lanes are equally
		// blind. Drop the cache-read column from the pool lane and the two
		// must diverge.
		blinded := agent.WorkerUsageFromStreamUsage(llm.Usage{
			InputTokens:              rawInputTokens,
			CacheCreationInputTokens: rawCacheCreationTokens,
			OutputTokens:             rawOutputTokens,
		}, "claude-sonnet-4-20250514")
		if blinded.BilledTotalTokens() == parserTotal {
			t.Error("a cache-blind pool lane still agreed with the provider parser — this test would pass over the exact divergence it exists to catch")
		}

		// And the mirror: a provider event missing the same column.
		blindEvent := `{"type":"result","usage":{"input_tokens":50,"cache_creation_input_tokens":2000,"output_tokens":500}}`
		blindParsed, parsedOK := codex.ParseUsage(blindEvent)
		if !parsedOK {
			t.Fatal("the provider parser reported no usage for the column-dropped event")
		}
		if blindParsed.BilledTotalTokens() == poolTotal {
			t.Error("a cache-blind provider lane still agreed with the pool — the invariant is not load-bearing")
		}
	})

	t.Run("nothing outside the codex usage type adds token columns itself", func(t *testing.T) {
		offenders := tokenColumnSummationOffenders(t)
		if len(offenders) > 0 {
			sort.Strings(offenders)
			t.Errorf("%d production site(s) compute a token total by adding columns themselves — "+
				"the authoritative total is codex.WorkerUsage.BilledTotalTokens(), and a second summation in a "+
				"second place is Pitfall 1 (the 186x undercount):\n  %s",
				len(offenders), strings.Join(offenders, "\n  "))
		}
	})
}

// tokenColumnNames are the field names that hold a raw billed token column on
// either usage type. Adding two or more of them together is re-deriving the
// authoritative total by hand.
var tokenColumnNames = map[string]bool{
	"InputTokens":              true,
	"CachedInputTokens":        true,
	"CacheCreationTokens":      true,
	"CacheReadInputTokens":     true,
	"CacheCreationInputTokens": true,
	"OutputTokens":             true,
}

// summationExemptFile is the one place the arithmetic legitimately lives:
// codex.WorkerUsage's own billedTotal and TotalInputTokens, reached by every
// consumer through BilledTotalTokens().
const summationExemptFile = "usage.go"

// tokenColumnSummationOffenders parses every production file in the packages
// this phase touches and names any `+` expression whose operands include two
// or more raw token columns.
func tokenColumnSummationOffenders(t *testing.T) []string {
	t.Helper()

	var offenders []string
	for _, dir := range scannedTokenAccountingPackages(t) {
		for _, path := range productionGoFilesIn(t, dir) {
			if filepath.Base(path) == summationExemptFile && strings.HasSuffix(dir, filepath.Join("pkg", "codex")) {
				continue
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				bin, isBinary := n.(*ast.BinaryExpr)
				if !isBinary || bin.Op != token.ADD {
					return true
				}
				columns := tokenColumnsInAdditionChain(bin)
				if len(columns) >= 2 {
					sort.Strings(columns)
					offenders = append(offenders, fmt.Sprintf("%s:%d adds %s",
						relativeToRepoRoot(t, path), fset.Position(bin.Pos()).Line, strings.Join(columns, " + ")))
				}
				return true
			})
		}
	}
	return offenders
}

// tokenColumnsInAdditionChain collects the raw token column names appearing as
// operands anywhere in a chain of `+` expressions.
func tokenColumnsInAdditionChain(expr ast.Expr) []string {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return nil
		}
		return append(tokenColumnsInAdditionChain(e.X), tokenColumnsInAdditionChain(e.Y)...)
	case *ast.ParenExpr:
		return tokenColumnsInAdditionChain(e.X)
	case *ast.SelectorExpr:
		if tokenColumnNames[e.Sel.Name] {
			return []string{e.Sel.Name}
		}
	case *ast.Ident:
		if tokenColumnNames[e.Name] {
			return []string{e.Name}
		}
	}
	return nil
}

// scannedTokenAccountingPackages returns the absolute directories of the
// packages this phase touches.
func scannedTokenAccountingPackages(t *testing.T) []string {
	t.Helper()
	root := repoRootForTokenAccountingTest(t)
	return []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "pkg", "codex"),
		filepath.Join(root, "pkg", "llm"),
		filepath.Join(root, "pkg", "agent"),
	}
}

// productionGoFilesIn lists the Go source files in a package directory,
// exempting test files by the single stated rule that a file whose name ends
// in _test.go is a test.
func productionGoFilesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir %s: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files
}

// repoRootForTokenAccountingTest walks up from the test's working directory
// until it finds go.mod.
func repoRootForTokenAccountingTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find the repository root (no go.mod found walking up)")
		}
		dir = parent
	}
}

// relativeToRepoRoot renders a path the way a reader of the failure would
// type it.
func relativeToRepoRoot(t *testing.T, path string) string {
	t.Helper()
	rel, err := filepath.Rel(repoRootForTokenAccountingTest(t), path)
	if err != nil {
		return path
	}
	return rel
}
