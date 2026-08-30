package cmd

// Phase 198.2 plan 03 -- the session-start greeting now carries what the
// colony remembers about the owner.
//
// Task 1 settles the ranking rule the greeting and the status dashboard will
// share: CONTEXT.md described the dashboard as calling a most-recent loader
// named loadRecentRuntimeInstincts; the current source has no such function
// -- cmd/status.go already calls loadStrongestRuntimeInstincts
// (cmd/instinct_runtime.go), which ranks by usefulness score, not recency.
// This file's first test settles the discrepancy by proving the real
// behaviour, and locks the one-ranking-rule invariant so a second selector
// cannot be added silently.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// Task 1 -- one ranking rule for "strongest instincts", shared by the
// greeting card and the status dashboard.
// ---------------------------------------------------------------------------

// promoteRealInstinctVaried promotes an instinct through the real
// memory.PromoteService, with the source/evidence/observation-count
// combination the caller chooses -- never a hand-typed colony.InstinctEntry
// literal (D-04). content must satisfy memory.IsAdmissibleInstinctContent.
func promoteRealInstinctVaried(t *testing.T, s *storage.Store, content, sourceType, evidenceType string, observationCount int) colony.InstinctEntry {
	t.Helper()
	bus := events.NewBus(s, events.DefaultConfig())
	now := time.Now().UTC()
	hash := sha256.Sum256([]byte(content + ":" + sourceType + ":" + evidenceType))
	obs := colony.Observation{
		ContentHash:      "sha256:" + hex.EncodeToString(hash[:]),
		Content:          content,
		WisdomType:       "pattern",
		ObservationCount: observationCount,
		FirstSeen:        events.FormatTimestamp(now),
		LastSeen:         events.FormatTimestamp(now),
		Colonies:         []string{"test-colony"},
		SourceType:       sourceType,
		EvidenceType:     evidenceType,
	}
	svc := memory.NewPromoteService(s, bus)
	result, err := svc.Promote(context.Background(), obs, "test-colony")
	if err != nil {
		t.Fatalf("promote real instinct for %q: %v", content, err)
	}
	return result.Instinct
}

// instinctRankingSite is one place in a scanned directory that calls
// memory.InstinctUsefulnessScore -- the one instinct-ranking rule this test
// locks (198.2 plan 03).
type instinctRankingSite struct {
	File     string
	Function string
}

// scanInstinctRankingSource parses every non-test .go file directly under
// dir and finds every call to memory.InstinctUsefulnessScore, attributed to
// its enclosing top-level function. Mirrors scanNextActionHardcodeSource's
// shape (cmd/next_action_hardcode_ratchet_test.go): read the directory,
// parse each file with go/parser (a syntax parse -- it needs no resolved
// imports, so a synthetic fixture file need not actually import anything),
// and walk each top-level function body for the call this test cares about.
func scanInstinctRankingSource(dir string) ([]instinctRankingSite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var sites []instinctRankingSite
	fset := token.NewFileSet()
	filesScanned := 0

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", name, readErr)
		}
		file, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", name, parseErr)
		}
		filesScanned++

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if pkgIdent.Name == "memory" && sel.Sel.Name == "InstinctUsefulnessScore" {
					sites = append(sites, instinctRankingSite{File: name, Function: fn.Name.Name})
				}
				return true
			})
		}
	}

	if filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero non-test .go files in %s", dir)
	}
	return sites, nil
}

// TestStrongestHabitsHaveOneRankingRule is 198.2 plan 03 task 1's proof.
//
// CONTEXT.md described the status dashboard as calling a most-recent loader
// named loadRecentRuntimeInstincts; the current source has no such function
// -- cmd/status.go already calls loadStrongestRuntimeInstincts
// (cmd/instinct_runtime.go), which ranks by usefulness score, not recency.
// This test settles the discrepancy by proving the real behaviour rather
// than trusting either document: the one loader ranks strongest-first, its
// ties are broken deterministically, and no second selector can be added
// without this test catching it by name.
func TestStrongestHabitsHaveOneRankingRule(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// (a) Highest confidence/trust first.
	strong := promoteRealInstinctVaried(t, s,
		"Run `go test ./cmd/...` before claiming a fix works.",
		"user_feedback", "test_verified", 5)
	weak := promoteRealInstinctVaried(t, s,
		"Check the error log at cmd/server.go for the panic before restarting.",
		"heuristic", "anecdotal", 1)

	ranked := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 2)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked instincts, got %d: %+v", len(ranked), ranked)
	}
	if ranked[0].ID != strong.ID || ranked[1].ID != weak.ID {
		t.Fatalf("expected the higher-confidence/trust instinct first; got order %s, %s want %s, %s",
			ranked[0].ID, ranked[1].ID, strong.ID, weak.ID)
	}

	// (b) An exact tie is broken deterministically, and stays broken the
	// same way across repeated calls.
	tieA := promoteRealInstinctVaried(t, s,
		"Read pkg/memory/promote.go before changing the trust score formula.",
		"success_pattern", "single_phase", 1)
	tieB := promoteRealInstinctVaried(t, s,
		"Verify `git status` is clean before starting a new task.",
		"success_pattern", "single_phase", 1)

	// tieA and tieB share source, evidence and observation count, so their
	// trust score and confidence are byte-identical. The one thing real
	// Promote() calls a few milliseconds apart cannot guarantee is an
	// identical CreatedAt, and a tiny freshness gap would (correctly) break
	// the tie on its own -- proving nothing about the ID tie-break this
	// sub-test exists to check. Aligning CreatedAt here forces the genuine
	// tie a fast enough machine would already have produced.
	if tieA.Provenance.CreatedAt != tieB.Provenance.CreatedAt {
		var file colony.InstinctsFile
		if err := s.LoadJSON("instincts.json", &file); err != nil {
			t.Fatalf("reload instincts.json: %v", err)
		}
		for i := range file.Instincts {
			if file.Instincts[i].ID == tieB.ID {
				file.Instincts[i].Provenance.CreatedAt = tieA.Provenance.CreatedAt
			}
		}
		if err := s.SaveJSON("instincts.json", file); err != nil {
			t.Fatalf("align tie timestamps: %v", err)
		}
	}

	wantFirst, wantSecond := tieA.ID, tieB.ID
	if tieB.ID < tieA.ID {
		wantFirst, wantSecond = tieB.ID, tieA.ID
	}

	for call := 0; call < 2; call++ {
		got := loadStrongestRuntimeInstincts(s, &colony.ColonyState{}, 4)
		idxA, idxB := -1, -1
		for i, inst := range got {
			switch inst.ID {
			case tieA.ID:
				idxA = i
			case tieB.ID:
				idxB = i
			}
		}
		if idxA == -1 || idxB == -1 {
			t.Fatalf("call %d: the tied instincts did not both come back: %+v", call, got)
		}
		gotFirst, gotSecond := tieA.ID, tieB.ID
		if idxB < idxA {
			gotFirst, gotSecond = tieB.ID, tieA.ID
		}
		if gotFirst != wantFirst || gotSecond != wantSecond {
			t.Fatalf("call %d: the tie was not broken by instinct ID ascending; got order %s, %s want %s, %s",
				call, gotFirst, gotSecond, wantFirst, wantSecond)
		}
	}

	// The CONTEXT.md-named function does not exist -- the discrepancy this
	// task's action resolved, checked directly rather than trusted.
	t.Run("loadRecentRuntimeInstincts does not exist", func(t *testing.T) {
		entries, err := os.ReadDir(".")
		if err != nil {
			t.Fatalf("read cmd/: %v", err)
		}
		needle := "func " + "loadRecentRuntimeInstincts"
		for _, entry := range entries {
			name := entry.Name()
			// Test files are excluded, the same way scanInstinctRankingSource
			// excludes them above -- otherwise this loop would find the
			// needle inside this very file's own source, which quotes it to
			// build the string being searched for.
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, readErr := os.ReadFile(name)
			if readErr != nil {
				continue
			}
			if strings.Contains(string(data), needle) {
				t.Fatalf("cmd/%s defines loadRecentRuntimeInstincts -- CONTEXT.md's description was not actually superseded", name)
			}
		}
	})

	// (c) No second function may rank instincts for display.
	t.Run("no second ranking function exists in cmd/", func(t *testing.T) {
		sites, err := scanInstinctRankingSource(".")
		if err != nil {
			t.Fatalf("scan cmd/ for instinct-ranking sites: %v", err)
		}
		for _, site := range sites {
			if site.Function != "rankedInstinctEntries" {
				t.Fatalf("a second instinct-ranking function was found: %s (%s) -- only rankedInstinctEntries "+
					"may call memory.InstinctUsefulnessScore to rank instincts for display", site.Function, site.File)
			}
		}
	})

	// The guard on the guard: a planted second ranking function, in a clean
	// synthetic file the real cmd/ tree never sees, must be caught by name.
	t.Run("the scan can fail", func(t *testing.T) {
		tmp := t.TempDir()
		planted := "package cmd\n\n" +
			"func secondSelector(file colony.InstinctsFile, now time.Time) []colony.InstinctEntry {\n" +
			"\t_ = memory.InstinctUsefulnessScore(file.Instincts[0], now)\n" +
			"\treturn nil\n" +
			"}\n"
		if err := os.WriteFile(filepath.Join(tmp, "synthetic.go"), []byte(planted), 0644); err != nil {
			t.Fatalf("write synthetic fixture: %v", err)
		}
		plantedSites, err := scanInstinctRankingSource(tmp)
		if err != nil {
			t.Fatalf("scan synthetic fixture: %v", err)
		}
		found := false
		for _, site := range plantedSites {
			if site.Function == "secondSelector" {
				found = true
			}
		}
		if !found {
			t.Fatalf("a planted second ranking function in a clean synthetic file was not detected; found: %+v", plantedSites)
		}
	})
}
