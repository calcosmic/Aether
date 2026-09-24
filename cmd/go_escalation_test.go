package cmd

// Phase 209 plan 02 -- filling in the big route (D-01's other half) and
// D-03's self-escalation: when a small attempt proves bigger than it
// looked, the program moves it up to the planning route on its own
// authority, says so once in plain English, and never asks or blocks.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// snapshotAetherData hashes every file under root/.aether/data so a test can
// prove the big route wrote nothing there. A missing directory snapshots as
// an empty map, never an error -- the bare-folder case has no such
// directory at all.
func snapshotAetherData(t *testing.T, root string) map[string]string {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data")
	snapshot := map[string]string{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		snapshot[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	return snapshot
}

// TestGoBigRouteTakesTheJobToPlanning drives the real `aether go` command
// body (runGoJob) over two temporary repositories -- one with a recorded
// project that already has outstanding work, one bare -- with the worker
// invoker swapped for a capturing fake, and asserts the big route dispatches
// no helper, writes nothing under .aether/data, and reports a route and
// reason as the behaviour block describes.
func TestGoBigRouteTakesTheJobToPlanning(t *testing.T) {
	t.Run("recorded project", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)

		goal := "Ship the billing rewrite"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateEXECUTING,
			Plan: colony.Plan{Phases: []colony.Phase{
				{ID: 1, Name: "Foundations", Status: colony.PhaseInProgress},
			}},
		}
		writeGoRouteFixtureState(t, filepath.Join(root, ".aether", "data"), state)

		spy := &quickJobCaptureInvoker{}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

		before := snapshotAetherData(t, root)
		sentence := "invent a brand new capability nothing here has ever heard of"
		result, err := runGoJob(sentence, 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}
		after := snapshotAetherData(t, root)

		if len(spy.captured()) != 0 {
			t.Fatalf("the big route dispatched a helper: %d call(s)", len(spy.captured()))
		}
		if got := stringValue(result["route"]); got != string(jobSizeRouteBig) {
			t.Fatalf("route = %q, want %q", got, jobSizeRouteBig)
		}
		if strings.TrimSpace(stringValue(result["route_reason"])) == "" {
			t.Fatal("route_reason is empty")
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("the big route modified .aether/data:\nbefore=%v\nafter=%v", before, after)
		}

		rendered := renderGoVisual(result)
		if !isAetherBannerLine(strings.Split(rendered, "\n")[0]) {
			t.Fatalf("rendered screen's first line is not a banner line:\n%s", rendered)
		}
	})

	t.Run("bare folder", func(t *testing.T) {
		saveGlobals(t)
		// A genuinely no-project folder still opens a real store pointing
		// at an empty .aether/data (exactly what PersistentPreRunE does for
		// any non-read-only command in production) -- store == nil is a
		// different, unrelated case (storage genuinely unavailable) that
		// the shared next-action resolver reads as ambiguous recovery, not
		// as "no project has ever been started here".
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)

		spy := &quickJobCaptureInvoker{}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

		sentence := "invent a brand new capability nothing here has ever heard of"
		result, err := runGoJob(sentence, 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}

		if len(spy.captured()) != 0 {
			t.Fatalf("the big route dispatched a helper in a bare folder: %d call(s)", len(spy.captured()))
		}
		if got := stringValue(result["route"]); got != string(jobSizeRouteBig) {
			t.Fatalf("route = %q, want %q", got, jobSizeRouteBig)
		}
		reason := stringValue(result["route_reason"])
		if !strings.Contains(reason, sentence) {
			t.Errorf("expected the reason to name the job sentence as the goal being started, got %q", reason)
		}
		if got, ok := result["colony_active"].(bool); !ok || got {
			t.Errorf("colony_active = %v, want false for a bare folder", result["colony_active"])
		}

		answer := renderLifecycleClosing(result, "go")
		if !strings.Contains(strings.ToLower(answer), "init") {
			t.Errorf("expected the next-step line to recommend starting a project, got:\n%s", answer)
		}
	})
}

// writeGoRouteFixtureState (cmd/go_route_test.go) writes a colony state
// fixture; reused here rather than duplicated.

// quickJobFileWritingInvoker is a WorkerInvoker fake that genuinely writes
// extra files into the real working tree during Invoke, and reports them as
// FilesModified so runQuickJob's own real-changed-file union carries them
// even in a plain (non-git) temporary repository, where the snapshot's git
// fallback (quickWorkingTreeSnapshot) cannot run. The escalation reads
// runQuickJob's own "files" result key either way -- this fake exists to
// produce a genuine, disk-measured file count for it to read, not to
// exercise the git-snapshot path itself.
type quickJobFileWritingInvoker struct {
	extraFiles []string
	result     codex.WorkerResult
}

func (q *quickJobFileWritingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (q *quickJobFileWritingInvoker) ValidateAgent(path string) error      { return nil }
func (q *quickJobFileWritingInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	for i, name := range q.extraFiles {
		content := []byte("// escalation fixture file " + strconv.Itoa(i) + "\n")
		if err := os.WriteFile(filepath.Join(config.Root, name), content, 0o644); err != nil {
			return codex.WorkerResult{}, err
		}
	}
	result := q.result
	result.WorkerName = config.WorkerName
	result.FilesModified = append(append([]string(nil), result.FilesModified...), q.extraFiles...)
	if result.Status == "" {
		result.Status = "completed"
	}
	return result, nil
}

// TestGoEscalatesWhenTheSmallAttemptProvesBigger covers D-03's escalation
// rule from three angles: a real disk-measured file count over budget, a
// failed project check, and the ordinary case that stays small.
func TestGoEscalatesWhenTheSmallAttemptProvesBigger(t *testing.T) {
	t.Run("file count over budget", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)
		if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}

		invoker := &quickJobFileWritingInvoker{extraFiles: []string{"a.go", "b.go", "c.go", "d.go", "e.go"}}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
		origChecks := runQuickDeterministicChecks
		runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
			return quickChecksPassed, []string{"go build ./...: passed"}, nil
		}
		t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

		result, err := runGoJob("update widget.go", 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}

		escalated, _ := result["escalated"].(bool)
		if !escalated {
			t.Fatalf("expected escalation, result=%+v", result)
		}
		if got := stringValue(result["route"]); got != string(jobSizeRouteBig) {
			t.Fatalf("route = %q, want %q", got, jobSizeRouteBig)
		}
		reason := stringValue(result["escalation_reason"])
		if !strings.Contains(reason, "5") {
			t.Errorf("escalation_reason does not name the measured count: %q", reason)
		}

		rendered := renderGoVisual(result)
		if got := strings.Count(rendered, "turned out bigger than it looked"); got != 1 {
			t.Fatalf("expected exactly one escalation line, found %d in:\n%s", got, rendered)
		}

		for _, name := range invoker.extraFiles {
			if _, statErr := os.Stat(filepath.Join(root, name)); statErr != nil {
				t.Errorf("expected %s to still exist after escalation: %v", name, statErr)
			}
		}
	})

	t.Run("checks failed", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)
		if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}

		invoker := &quickJobFileWritingInvoker{extraFiles: []string{"widget.go"}}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
		origChecks := runQuickDeterministicChecks
		runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
			return quickChecksFailed, []string{"go build ./...: FAILED"}, nil
		}
		t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

		result, err := runGoJob("update widget.go", 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}

		escalated, _ := result["escalated"].(bool)
		if !escalated {
			t.Fatalf("expected escalation on failed checks, result=%+v", result)
		}
		reason := stringValue(result["escalation_reason"])
		if !strings.Contains(reason, "checks") {
			t.Errorf("escalation_reason does not name the failed checks: %q", reason)
		}
	})

	t.Run("stays small", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)
		if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}

		invoker := &quickJobFileWritingInvoker{extraFiles: []string{"widget.go"}}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
		origChecks := runQuickDeterministicChecks
		runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
			return quickChecksPassed, []string{"go build ./...: passed"}, nil
		}
		t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

		result, err := runGoJob("update widget.go", 2*time.Second)
		if err != nil {
			t.Fatalf("runGoJob: %v", err)
		}

		if escalated, _ := result["escalated"].(bool); escalated {
			t.Fatalf("did not expect escalation, result=%+v", result)
		}
		if got := stringValue(result["route"]); got != string(jobSizeRouteSmall) {
			t.Fatalf("route = %q, want %q", got, jobSizeRouteSmall)
		}
	})
}

// TestGoNeverMovesAJobBackDown drives resolveJobSizeRoute directly with a
// pre-attempt decision that is already big (an accepted plan with
// outstanding work) and attempt facts that would, on their own, indicate a
// tiny change -- and asserts the route stays big and Escalated stays false.
func TestGoNeverMovesAJobBackDown(t *testing.T) {
	tiny := smallAttemptFacts{FilesChanged: 1, ChecksStatus: quickChecksPassed, Verdict: colony.WorkOutcomeSuccess}
	facts := jobSizeFacts{
		Job:                 "fix the rounding bug in payment.go",
		MatchedPaths:        []string{"payment.go"},
		PlanAcceptedUnbuilt: true,
		Attempt:             &tiny,
	}
	decision := resolveJobSizeRoute(facts)
	if decision.Route != jobSizeRouteBig {
		t.Fatalf("route = %q, want %q -- a job already big before the attempt must never come back down", decision.Route, jobSizeRouteBig)
	}
	if decision.Escalated {
		t.Fatal("Escalated = true, want false -- this was already big before any attempt, so it is not a fresh escalation")
	}
}

// TestGoEscalationReadsOnlyMeasuredEvidence parses cmd/go_route.go with
// go/parser and fails if smallAttemptFactsFromQuickResult reads any result
// key other than "files", "checks_status" and "work_outcome" -- never
// "summary", "raw_output", or any other field carrying a helper's own words.
func TestGoEscalationReadsOnlyMeasuredEvidence(t *testing.T) {
	allowed := map[string]bool{"files": true, "checks_status": true, "work_outcome": true}

	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "go_route.go", nil, 0)
	if err != nil {
		t.Fatalf("parse go_route.go: %v", err)
	}

	found := false
	ast.Inspect(parsed, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "smallAttemptFactsFromQuickResult" {
			return true
		}
		found = true
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			idx, ok := n.(*ast.IndexExpr)
			if !ok {
				return true
			}
			lit, ok := idx.Index.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			key, unquoteErr := strconv.Unquote(lit.Value)
			if unquoteErr != nil {
				return true
			}
			if !allowed[key] {
				t.Errorf("smallAttemptFactsFromQuickResult reads result key %q at %v -- only files, checks_status "+
					"and work_outcome may ever be read here", key, fset.Position(lit.Pos()))
			}
			return true
		})
		return false
	})
	if !found {
		t.Fatal("smallAttemptFactsFromQuickResult was not found in go_route.go -- the AST walk is looking at the wrong function")
	}
}

// TestGoEscalationNeverAsksAndNeverBlocks is the standing guard for the
// milestone's own rule: the size router and the escalation derive, say what
// they decided, and continue -- they never refuse, ask, or block. It drives
// the real command across the small/big/escalated matrix and then parses
// cmd/go_cmd.go and cmd/go_route.go to fail if either ever grows a new
// refusal of its own.
func TestGoEscalationNeverAsksAndNeverBlocks(t *testing.T) {
	flagsFilePath := func(root string) string {
		return filepath.Join(root, ".aether", "data", "pending-decisions.json")
	}
	fileHash := func(t *testing.T, path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ""
			}
			t.Fatalf("read %s: %v", path, err)
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:])
	}
	assertResultNeverParksADecision := func(t *testing.T, result map[string]interface{}) {
		t.Helper()
		for _, key := range []string{"decision_required", "pending_decision", "awaiting_decision", "question", "checkpoint"} {
			if _, present := result[key]; present {
				t.Errorf("result carries %q, which would park the job on an owner decision", key)
			}
		}
	}

	cases := []struct {
		name  string
		setup func(t *testing.T) (job string, root string, s *storage.Store)
	}{
		{
			name: "small completed",
			setup: func(t *testing.T) (string, string, *storage.Store) {
				s, root := newTestStore(t)
				if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
					t.Fatalf("write fixture file: %v", err)
				}
				invoker := &quickJobFileWritingInvoker{extraFiles: []string{"widget.go"}}
				origInvoker := newQuickWorkerInvoker
				newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
				t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
				origChecks := runQuickDeterministicChecks
				runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
					return quickChecksPassed, nil, nil
				}
				t.Cleanup(func() { runQuickDeterministicChecks = origChecks })
				return "update the widget", root, s
			},
		},
		{
			name: "small escalated by file count",
			setup: func(t *testing.T) (string, string, *storage.Store) {
				s, root := newTestStore(t)
				if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
					t.Fatalf("write fixture file: %v", err)
				}
				invoker := &quickJobFileWritingInvoker{extraFiles: []string{"a.go", "b.go", "c.go", "d.go", "e.go"}}
				origInvoker := newQuickWorkerInvoker
				newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
				t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
				origChecks := runQuickDeterministicChecks
				runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
					return quickChecksPassed, nil, nil
				}
				t.Cleanup(func() { runQuickDeterministicChecks = origChecks })
				return "update the widget", root, s
			},
		},
		{
			// Deliberately "not resolved" rather than a genuine "failed"
			// (quickChecksFailed) run: a genuine failure independently
			// raises a tracked issue flag through the project's own,
			// pre-existing runQuickJob behaviour (raiseQuickIssueFlag) --
			// real, correct, and out of this plan's scope -- which would
			// make the shared-decision-store assertion below fail for a
			// reason that has nothing to do with this plan's own
			// escalation logic. "Not resolved" still escalates (a partial
			// verdict is one of D-03's two escalation signals) without
			// tripping that unrelated write.
			name: "small escalated by failed checks",
			setup: func(t *testing.T) (string, string, *storage.Store) {
				s, root := newTestStore(t)
				if err := os.WriteFile(filepath.Join(root, "widget.go"), []byte("package main\n"), 0o644); err != nil {
					t.Fatalf("write fixture file: %v", err)
				}
				invoker := &quickJobFileWritingInvoker{extraFiles: []string{"widget.go"}}
				origInvoker := newQuickWorkerInvoker
				newQuickWorkerInvoker = func() codex.WorkerInvoker { return invoker }
				t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
				origChecks := runQuickDeterministicChecks
				runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
					return quickChecksNotResolved, nil, nil
				}
				t.Cleanup(func() { runQuickDeterministicChecks = origChecks })
				return "update the widget", root, s
			},
		},
		{
			name: "big with a recorded project",
			setup: func(t *testing.T) (string, string, *storage.Store) {
				s, root := newTestStore(t)
				goal := "Ship the billing rewrite"
				state := colony.ColonyState{
					Version: "3.0",
					Goal:    &goal,
					State:   colony.StateEXECUTING,
					Plan: colony.Plan{Phases: []colony.Phase{
						{ID: 1, Name: "Foundations", Status: colony.PhaseInProgress},
					}},
				}
				writeGoRouteFixtureState(t, filepath.Join(root, ".aether", "data"), state)
				spy := &quickJobCaptureInvoker{}
				origInvoker := newQuickWorkerInvoker
				newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
				t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
				return "invent a brand new capability that nothing here names", root, s
			},
		},
		{
			name: "big in a bare folder",
			setup: func(t *testing.T) (string, string, *storage.Store) {
				root := t.TempDir()
				return "invent a brand new capability that nothing here names", root, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			store = nil
			job, root, s := tc.setup(t)
			store = s
			chdirForTest190_05(t, root)

			beforeHash := fileHash(t, flagsFilePath(root))

			result, err := runGoJob(job, 2*time.Second)
			if err != nil {
				t.Fatalf("runGoJob returned an error instead of deciding and continuing: %v", err)
			}
			assertResultNeverParksADecision(t, result)

			rendered := renderGoVisual(result)
			for _, line := range strings.Split(rendered, "\n") {
				if strings.Contains(line, "?") {
					t.Errorf("the program's own screen carries a question mark: %q", line)
				}
			}

			afterHash := fileHash(t, flagsFilePath(root))
			if beforeHash != afterHash {
				t.Errorf("the shared decision store changed during a %q run: before=%s after=%s", tc.name, beforeHash, afterHash)
			}
		})
	}

	// The AST half: neither file may call outputError for any reason other
	// than the empty-input usage message or relaying an error an existing
	// path already returned.
	for _, filename := range []string{"go_cmd.go", "go_route.go"} {
		t.Run("no new refusal in "+filename, func(t *testing.T) {
			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, filename, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", filename, err)
			}
			ast.Inspect(parsed, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if !ok || ident.Name != "outputError" {
					return true
				}
				if len(call.Args) < 2 {
					return true
				}
				switch arg := call.Args[1].(type) {
				case *ast.BasicLit:
					if arg.Kind == token.STRING {
						if unquoted, uErr := strconv.Unquote(arg.Value); uErr == nil && strings.Contains(unquoted, "usage:") {
							return true // the empty-input usage message
						}
					}
				case *ast.CallExpr:
					if sel, selOk := arg.Fun.(*ast.SelectorExpr); selOk && sel.Sel.Name == "Error" {
						return true // relays an error an existing path already returned
					}
				}
				t.Errorf("%s calls outputError with a new refusal at %v -- the size router and the escalation "+
					"must never refuse the owner's job", filename, fset.Position(call.Pos()))
				return true
			})
		})
	}
}
