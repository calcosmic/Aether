package cmd

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// TestRecruitmentTracerEndToEnd drives the REAL recruitCmd through rootCmd
// against a temporary store -- never by calling the internal functions
// directly -- proving 203-02-PLAN.md Task 1's single thin path end to end:
// intent, admission through the existing spawnCanSpawnDecision chokepoint, a
// leased-workspace child process, exactly-once result binding, and the one
// live-event boundary.
//
// The parent's recorded depth is derived by writing a real spawn entry
// through SpawnTree.RecordSpawn, never by typing a plausible ledger line, so
// this fixture is built the way the runtime itself builds one.
func TestRecruitmentTracerEndToEnd(t *testing.T) {
	if os.Getenv("AETHER_RECRUIT_CHILD") == "1" {
		// Recursively invoked AS the recruited child in the "admitted"
		// subtest below: dispatchRecruitment (cmd/recruitment_dispatch.go)
		// sets this env var on every real child it spawns, production or
		// test. This branch exists to BE a real, observable subprocess for
		// dispatchRecruitment to run and wait on -- it does no recruitment
		// work of its own and returns immediately.
		fmt.Println("recruited-child-ok")
		return
	}

	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "do a thing", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}

	t.Run("admitted recruitment starts a real child and binds one result", func(t *testing.T) {
		buf.Reset()
		errBuf.Reset()
		renderedCommandExitCode.Store(0)

		t.Setenv("AETHER_RECRUIT_BINARY", os.Args[0])
		t.Setenv("AETHER_RECRUIT_ARGS", "-test.run=^TestRecruitmentTracerEndToEnd$")
		t.Setenv("AETHER_RECRUIT_TIMEOUT", "30s")

		rootCmd.SetArgs([]string{
			"recruit",
			"--parent", "A1",
			"--caste", "builder",
			"--objective", "help with x",
			"--reason", "stuck on y",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("recruit command returned an error: %v", err)
		}
		if code := int(renderedCommandExitCode.Load()); code != 0 {
			t.Fatalf("admitted recruitment did not exit 0: code=%d stdout=%s stderr=%s", code, buf.String(), errBuf.String())
		}

		env := parseEnvelope(t, buf.String())
		result, _ := env["result"].(map[string]interface{})
		if result == nil {
			t.Fatalf("expected a result object in output: %s", buf.String())
		}
		if admitted, _ := result["admitted"].(bool); !admitted {
			t.Fatalf("expected admitted=true, got %v: %s", result["admitted"], buf.String())
		}
		childName, _ := result["child"].(string)
		if childName == "" {
			t.Fatalf("expected a non-empty child name: %s", buf.String())
		}

		entries, err := st.Parse()
		if err != nil {
			t.Fatalf("parse spawn tree: %v", err)
		}
		var childEntry *agent.SpawnEntry
		for i := range entries {
			if entries[i].AgentName == childName {
				childEntry = &entries[i]
			}
		}
		if childEntry == nil {
			t.Fatalf("no spawn-tree entry recorded for child %q", childName)
		}
		if childEntry.ParentName != "A1" {
			t.Fatalf("child entry parent = %q, want %q", childEntry.ParentName, "A1")
		}
		if childEntry.Depth != 2 {
			t.Fatalf("child entry depth = %d, want 2 (A1's own depth 1, plus 1)", childEntry.Depth)
		}

		var resultsFile recruitmentResultsFile
		if err := store.LoadJSON(recruitmentResultsPath, &resultsFile); err != nil {
			t.Fatalf("load recruitment results: %v", err)
		}
		if len(resultsFile.Entries) != 1 {
			t.Fatalf("expected exactly 1 bound recruitment result, got %d", len(resultsFile.Entries))
		}

		bus := events.NewBus(store, events.DefaultConfig())
		admittedEvents, err := bus.Replay(context.Background(), events.LiveTopicRecruitAdmitted, time.Time{}, 100)
		if err != nil {
			t.Fatalf("replay admitted events: %v", err)
		}
		if len(admittedEvents) != 1 {
			t.Fatalf("expected exactly 1 %s event, got %d", events.LiveTopicRecruitAdmitted, len(admittedEvents))
		}
	})

	t.Run("refused at the depth cap publishes one refusal, records nothing, exits 0", func(t *testing.T) {
		buf.Reset()
		errBuf.Reset()
		renderedCommandExitCode.Store(0)

		// A2 is recorded at depth 2 -- a helper spawned from it would be
		// depth 3, past spawnMaxDelegationDepth (2).
		if err := st.RecordSpawn("A1", "builder", "A2", "do a deeper thing", 2); err != nil {
			t.Fatalf("seed depth-cap parent: %v", err)
		}

		before, err := store.ReadFile("spawn-tree.txt")
		if err != nil {
			t.Fatalf("read spawn-tree.txt before refusal: %v", err)
		}

		// If dispatchRecruitment is ever reached for this attempt, invoking
		// this nonexistent binary makes that failure loud rather than
		// silently passing.
		t.Setenv("AETHER_RECRUIT_BINARY", "aether-recruit-must-not-be-invoked-"+t.Name())

		rootCmd.SetArgs([]string{
			"recruit",
			"--parent", "A2",
			"--caste", "builder",
			"--objective", "help deeper",
			"--reason", "stuck",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("recruit command returned an error: %v", err)
		}
		if code := int(renderedCommandExitCode.Load()); code != 0 {
			t.Fatalf("refused recruitment did not exit 0: code=%d stdout=%s stderr=%s", code, buf.String(), errBuf.String())
		}

		env := parseEnvelope(t, buf.String())
		result, _ := env["result"].(map[string]interface{})
		if result == nil {
			t.Fatalf("expected a result object in output: %s", buf.String())
		}
		if admitted, _ := result["admitted"].(bool); admitted {
			t.Fatalf("expected admitted=false: %s", buf.String())
		}
		if reason, _ := result["reason"].(string); reason != "depth" {
			t.Fatalf("expected reason class %q, got %q: %s", "depth", reason, buf.String())
		}

		after, err := store.ReadFile("spawn-tree.txt")
		if err != nil {
			t.Fatalf("read spawn-tree.txt after refusal: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("spawn-tree.txt changed after a refused recruitment:\nbefore=%q\nafter=%q", before, after)
		}

		bus := events.NewBus(store, events.DefaultConfig())
		refusedEvents, err := bus.Replay(context.Background(), events.LiveTopicRecruitRefused, time.Time{}, 100)
		if err != nil {
			t.Fatalf("replay refused events: %v", err)
		}
		if len(refusedEvents) != 1 {
			t.Fatalf("expected exactly 1 %s event, got %d", events.LiveTopicRecruitRefused, len(refusedEvents))
		}
	})

	t.Run("replaying the same terminal result twice is idempotent", func(t *testing.T) {
		result := recruitmentResult{
			SchemaVersion:  recruitmentSchemaVersion,
			RecruitmentID:  "replay-fixture-1",
			IntentID:       "replay-fixture-1",
			ChildName:      "Fixture-1",
			ParentName:     "A1",
			TerminalStatus: "completed",
			Transaction: colony.LifecycleTransactionReference{
				ID:    "replay-fixture-1",
				Stage: colony.TransactionStageCommitted,
			},
		}
		first, err := bindRecruitmentResult(result)
		if err != nil {
			t.Fatalf("first bind: %v", err)
		}
		before, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after first bind: %v", err)
		}
		second, err := bindRecruitmentResult(result)
		if err != nil {
			t.Fatalf("second bind: %v", err)
		}
		after, err := store.ReadFile(recruitmentResultsPath)
		if err != nil {
			t.Fatalf("read results file after second bind: %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("recruitment/results.json changed on replay:\nbefore=%q\nafter=%q", before, after)
		}
		if second.RecruitmentID != first.RecruitmentID || second.ChildName != first.ChildName || second.TerminalStatus != first.TerminalStatus {
			t.Fatalf("replay did not return the stored receipt: first=%+v second=%+v", first, second)
		}
	})
}

// TestRecruitmentDispatchTerminatesWholeProcessGroup proves
// dispatchRecruitment's process-group teardown (codex.ConfigureWorkerCommand)
// is load-bearing rather than decorative: the child (sh) exits almost
// instantly, backgrounding a grandchild that outlives the dispatch timeout
// by a wide margin and, absent process-group teardown, would keep the
// captured output pipe open until the grandchild itself exits -- exactly the
// bare-exec.CommandContext mistake runAvailabilityProbeOnce's own comment in
// pkg/codex/platform_dispatch.go documents. Deleting the
// codex.ConfigureWorkerCommand call from dispatchRecruitment makes this test
// fail: without it, this call blocks until the grandchild's own 30s sleep
// completes, well past the bound this test asserts.
func TestRecruitmentDispatchTerminatesWholeProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group signalling is unix-specific")
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available on PATH")
	}

	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Setenv("AETHER_RECRUIT_TIMEOUT", "300ms")
	t.Setenv("AETHER_RECRUIT_BINARY", shPath)
	// The direct child (sh) exits immediately; its own backgrounded
	// grandchild sleeps for 30s -- far past the 300ms dispatch timeout and
	// this test's own bound below.
	t.Setenv("AETHER_RECRUIT_ARGS", "-c\nsleep 30 & exit 0")

	intent := recruitmentIntent{
		SchemaVersion: recruitmentSchemaVersion,
		ParentName:    "A1",
		Caste:         "builder",
		Objective:     "prove process-group teardown",
		Reason:        "test",
		Workspace:     tmpDir,
	}

	done := make(chan *recruitmentDispatchResult, 1)
	go func() {
		result, dispatchErr := dispatchRecruitment(intent, "ProcessGroupFixture-1")
		if dispatchErr != nil {
			t.Errorf("dispatchRecruitment returned an unexpected error: %v", dispatchErr)
		}
		done <- result
	}()

	select {
	case result := <-done:
		if result == nil {
			t.Fatal("dispatchRecruitment returned a nil result")
		}
		if result.TerminalStatus != "timeout" && result.TerminalStatus != "failed" {
			t.Fatalf("expected a terminal (non-completed) status once the deadline elapsed, got %q", result.TerminalStatus)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("dispatchRecruitment did not return within 12s -- the whole process group was not torn down (the grandchild's 30s sleep is still holding the output pipe open)")
	}
}

// TestRecruitLiveEventsGoThroughTheOneBoundary is an AST-based scan, in the
// style of TestEveryLiveEventGoesThroughOneBoundary
// (cmd/live_projection_test.go), that fails by name if any file OTHER than
// cmd/live_events.go calls Publish or emitColonyLive carrying a
// live.recruit.* topic argument. It does not flag every
// events.ColonyLivePayload construction -- that composite literal is the
// shared wire shape every live-event kind uses (build waves, checks, Oracle
// rounds, Swarm lenses, ...), so a bare "constructed outside live_events.go"
// check would flag legitimate, unrelated live-event emission. What actually
// matters, and what this test asserts, is that a live.recruit.* TOPIC is
// never carried into a Publish/emitColonyLive call from anywhere but the one
// boundary file.
func TestRecruitLiveEventsGoThroughTheOneBoundary(t *testing.T) {
	const allowedFile = "live_events.go"

	fset := token.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}

	foundAllowedFile := false
	var violations []string
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == allowedFile {
			foundAllowedFile = true
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			calleeName := ""
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				calleeName = fn.Sel.Name
			case *ast.Ident:
				calleeName = fn.Name
			}
			if calleeName != "Publish" && calleeName != "emitColonyLive" {
				return true
			}
			for _, arg := range call.Args {
				if recruitTopicArg(arg) {
					violations = append(violations, fmt.Sprintf("%s: %s call carries a live.recruit.* topic", fset.Position(call.Pos()).String(), calleeName))
				}
			}
			return true
		})
	}

	if !foundAllowedFile {
		t.Fatal("fixture is broken: cmd/live_events.go was not part of the scanned file set")
	}
	if len(violations) != 0 {
		t.Fatalf("found live.recruit.* emission outside %s:\n%s", allowedFile, strings.Join(violations, "\n"))
	}

	t.Run("a fixture function publishing a recruit topic directly is caught", func(t *testing.T) {
		const fixtureSrc = `package cmd

func sneakyRecruitPublisher() {
	var bus interface {
		Publish(topic string, payload []byte) error
	}
	_ = bus.Publish("live.recruit.admitted", nil)
}
`
		fixtureFset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fixtureFset, "fixture_recruit_violation.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse synthetic fixture: %v", err)
		}
		found := false
		ast.Inspect(fixtureFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Publish" {
				return true
			}
			for _, arg := range call.Args {
				if recruitTopicArg(arg) {
					found = true
				}
			}
			return true
		})
		if !found {
			t.Fatal("scanner failed to detect a recruit-topic Publish call in the synthetic fixture")
		}
	})
}

// recruitTopicArg reports whether arg is a literal or selector expression
// naming a live.recruit.* topic.
func recruitTopicArg(arg ast.Expr) bool {
	switch a := arg.(type) {
	case *ast.BasicLit:
		if a.Kind == token.STRING {
			if unquoted, err := strconv.Unquote(a.Value); err == nil && strings.HasPrefix(unquoted, "live.recruit.") {
				return true
			}
		}
	case *ast.SelectorExpr:
		if a.Sel.Name == "LiveTopicRecruitAdmitted" || a.Sel.Name == "LiveTopicRecruitRefused" {
			return true
		}
	}
	return false
}

// recruitmentSymbols is the closed set of identifiers introduced by this
// plan's recruitment subsystem. TestPlainBuildPathDoesNotTouchRecruitment
// fails by name if any of them appears as a real identifier reference in the
// ordinary build-dispatch source files -- the folded-todo constraint ("a
// build that never recruits pays nothing for this feature") expressed as a
// failing check rather than a comment.
var recruitmentSymbols = map[string]bool{
	"dispatchRecruitment":              true,
	"bindRecruitmentResult":            true,
	"recruitCmd":                       true,
	"recruitmentIntent":                true,
	"recruitmentResult":                true,
	"recruitmentDispatchResult":        true,
	"recruitmentResultsFile":           true,
	"recruitmentDecisionResult":        true,
	"emitColonyLiveRecruitAdmitted":    true,
	"emitColonyLiveRecruitRefused":     true,
	"resolvedRecruitmentTimeout":       true,
	"resolvedRecruitmentBinary":        true,
	"recruitmentDispatchArgv":          true,
	"recruitmentResultsPath":           true,
	"recruitmentSchemaVersion":         true,
	"currentLiveRecruitmentEpisode":    true,
	"errRecruitmentResultAlreadyBound": true,
}

// buildDispatchFilesForRecruitmentScan names the core build-dispatch source
// files a plain, non-recruiting build actually executes through. This list
// is intentionally the dispatch surface, not the whole package -- scanning
// the package that DEFINES the recruitment symbols would trivially "find"
// every one of them inside their own declarations.
var buildDispatchFilesForRecruitmentScan = []string{
	"codex_build.go",
	"codex_build_worktree.go",
	"codex_dispatch_contract.go",
	"coherent_jobs.go",
}

// TestPlainBuildPathDoesNotTouchRecruitment proves, from the parsed syntax
// tree of the ordinary build-dispatch files rather than from review, that an
// unmodified build path references none of this plan's recruitment symbols.
// If a later change wires dispatchRecruitment (or any sibling symbol) into
// the plain build path, this test fails and names the file.
func TestPlainBuildPathDoesNotTouchRecruitment(t *testing.T) {
	fset := token.NewFileSet()
	scanned := 0
	var violations []string
	for _, name := range buildDispatchFilesForRecruitmentScan {
		if _, statErr := os.Stat(name); statErr != nil {
			continue
		}
		scanned++
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if recruitmentSymbols[ident.Name] {
				violations = append(violations, fmt.Sprintf("%s: references recruitment symbol %q", fset.Position(ident.Pos()).String(), ident.Name))
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("fixture is broken: none of the expected build-dispatch files were found in the cmd package directory")
	}
	if len(violations) != 0 {
		t.Fatalf("plain build-dispatch files reference recruitment symbols (a build that never recruits must pay nothing for this feature):\n%s", strings.Join(violations, "\n"))
	}

	// A build that never recruits also reads no recruitment file. store is
	// nil here (no test store was ever created in this test), so the
	// recruitment store path genuinely cannot have been written by
	// anything this test exercised.
	if _, err := os.Stat(filepath.Join(t.TempDir(), recruitmentResultsPath)); err == nil {
		t.Fatal("fixture is broken: an unrelated recruitment results file exists")
	}
}
