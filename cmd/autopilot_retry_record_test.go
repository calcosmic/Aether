package cmd

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Task 1 (TDD): recordAutopilotRetryExhaustion's content, proven directly.
// ---------------------------------------------------------------------------

// TestRecordAutopilotRetryExhaustion is 188-05's Tier 1 test (D-14): it
// proves recordAutopilotRetryExhaustion's own record content is correct,
// independent of ever forcing a real build to fail. Every subtest reads the
// written record back via loadMiddenFile (cmd/midden_shared.go, 188-01's
// output) or, in the last subtest, via loadMemoryHealthSummary -- both are
// the exact functions criterion 1's unified consumers (autopilot's own
// pause-check, /ant-resume, colony-prime's context capsule, memory-health,
// immune-auto-scar) call themselves, so this is already the "normal
// consumer path," not a raw-file read invented just for this test.
func TestRecordAutopilotRetryExhaustion(t *testing.T) {
	t.Run("distinct errors: message names the phase and both attempts' text", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)

		firstErr := errors.New("worker Mason-3 timed out after 300s")
		secondErr := errors.New("verification gate failed: 2 tests red")

		if err := recordAutopilotRetryExhaustion(s, 7, firstErr, secondErr); err != nil {
			t.Fatalf("recordAutopilotRetryExhaustion: %v", err)
		}

		mf, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile after recordAutopilotRetryExhaustion: %v", err)
		}
		if len(mf.Entries) != 1 {
			t.Fatalf("Entries length = %d, want exactly 1", len(mf.Entries))
		}
		e := mf.Entries[0]
		if e.Category != "autopilot_retry_exhausted" {
			t.Errorf("Category = %q, want %q", e.Category, "autopilot_retry_exhausted")
		}
		if e.Source != "aether run" {
			t.Errorf("Source = %q, want %q", e.Source, "aether run")
		}
		if !strings.Contains(e.Message, strconv.Itoa(7)) {
			t.Errorf("Message %q does not name phase 7", e.Message)
		}
		if !strings.Contains(e.Message, firstErr.Error()) {
			t.Errorf("Message %q does not contain the first attempt's error text %q", e.Message, firstErr.Error())
		}
		if !strings.Contains(e.Message, secondErr.Error()) {
			t.Errorf("Message %q does not contain the second attempt's error text %q", e.Message, secondErr.Error())
		}
	})

	t.Run("identical errors on both attempts: still one entry, still says two attempts happened", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)

		sameText := "dependency install failed: network unreachable"

		if err := recordAutopilotRetryExhaustion(s, 3, errors.New(sameText), errors.New(sameText)); err != nil {
			t.Fatalf("recordAutopilotRetryExhaustion: %v", err)
		}

		mf, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile: %v", err)
		}
		if len(mf.Entries) != 1 {
			t.Fatalf("Entries length = %d, want exactly 1 (both attempts collapse into one record, not two)", len(mf.Entries))
		}
		e := mf.Entries[0]
		if !strings.Contains(e.Message, sameText) {
			t.Errorf("Message %q does not name the shared failure text %q", e.Message, sameText)
		}
		lower := strings.ToLower(e.Message)
		if !strings.Contains(lower, "both") && !strings.Contains(lower, "twice") && !strings.Contains(lower, "two attempts") {
			t.Errorf("Message %q does not say two attempts happened -- a reader could mistake this for a single try, not an exhausted retry", e.Message)
		}
	})

	t.Run("a write failure is returned, not swallowed", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)

		// Put a directory at the canonical midden path so appendMiddenEntry's
		// own read step fails -- the same directory-at-a-file-path fault
		// injection cmd/spawn_failclosed_test.go already uses for the same
		// purpose.
		middenPath := filepath.Join(s.BasePath(), middenCanonicalPath)
		if err := os.MkdirAll(middenPath, 0755); err != nil {
			t.Fatalf("create directory at midden path: %v", err)
		}

		err := recordAutopilotRetryExhaustion(s, 9, errors.New("first"), errors.New("second"))
		if err == nil {
			t.Fatal("expected a non-nil error when appendMiddenEntry cannot write, got nil -- the write's own error must never be swallowed inside this function")
		}
	})

	t.Run("the record is readable through memory-health's real consumer path, closing 188-01's loop end-to-end", func(t *testing.T) {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)

		before := loadMemoryHealthSummary(s)

		if err := recordAutopilotRetryExhaustion(s, 12, errors.New("build failed: exit 1"), errors.New("build failed: exit 1 (again)")); err != nil {
			t.Fatalf("recordAutopilotRetryExhaustion: %v", err)
		}

		after := loadMemoryHealthSummary(s)
		if after.RecentFailures != before.RecentFailures+1 {
			t.Fatalf("loadMemoryHealthSummary's RecentFailures = %d after the write, want %d (before=%d) -- memory-health (one of criterion 1's four unified consumers) did not see the record through its own real read path", after.RecentFailures, before.RecentFailures+1, before.RecentFailures)
		}
		if after.LastFailure == "" {
			t.Fatal("loadMemoryHealthSummary's LastFailure is still empty after a write -- the timestamp did not surface through the real consumer either")
		}
	})
}

// ---------------------------------------------------------------------------
// Task 2: structural proof that the call site is actually wired.
// ---------------------------------------------------------------------------

// autopilotRetryCompatibilitySourceFile is the file the second-failure
// branch this guard checks against lives in.
const autopilotRetryCompatibilitySourceFile = "compatibility_cmds.go"

// TestAutopilotRetryExhaustionCallSiteIsWired is 188-05's Tier 2 test
// (D-14): a structural regression guard, not a full forced-double-build-
// failure harness -- no reliable way to force runCodexBuildWithOptions to
// fail deterministically twice exists in this codebase today (confirmed by
// grep across cmd/*_test.go; see 188-CONTEXT.md D-14). It parses
// compatibility_cmds.go with go/parser, locates runCompatibilityAutopilot's
// own function body by byte range (not a whole-file grep, which would also
// match this function's own definition were it colocated in this file), and
// confirms recordAutopilotRetryExhaustion( is called strictly between the
// retry's own runCodexBuildWithOptions( call and the "paused" sync call and
// "return nil, err" that follow it -- proving the record write actually
// sits inside the second-failure branch, not merely somewhere earlier in
// the function that would also satisfy a looser "appears before the
// return" check.
func TestAutopilotRetryExhaustionCallSiteIsWired(t *testing.T) {
	src, err := os.ReadFile(autopilotRetryCompatibilitySourceFile)
	if err != nil {
		t.Fatalf("read %s: %v", autopilotRetryCompatibilitySourceFile, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, autopilotRetryCompatibilitySourceFile, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", autopilotRetryCompatibilitySourceFile, err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Name.Name == "runCompatibilityAutopilot" {
			fn = fd
			break
		}
	}
	if fn == nil {
		t.Fatalf("runCompatibilityAutopilot not found in %s -- a rename would silently blind this guard", autopilotRetryCompatibilitySourceFile)
	}
	if fn.Body == nil {
		t.Fatal("runCompatibilityAutopilot has no body")
	}

	startOffset := fset.Position(fn.Body.Pos()).Offset
	endOffset := fset.Position(fn.Body.End()).Offset
	if startOffset < 0 || endOffset > len(src) || startOffset >= endOffset {
		t.Fatalf("invalid function body byte range [%d, %d) for a %d-byte file", startOffset, endOffset, len(src))
	}
	body := string(src[startOffset:endOffset])

	const buildCall = "runCodexBuildWithOptions("
	firstBuildIdx := strings.Index(body, buildCall)
	if firstBuildIdx == -1 {
		t.Fatalf("runCompatibilityAutopilot does not call %s at all -- this guard's anchor point is gone; update it to match the new retry structure", buildCall)
	}
	afterFirst := body[firstBuildIdx+len(buildCall):]
	secondBuildRel := strings.Index(afterFirst, buildCall)
	if secondBuildRel == -1 {
		t.Fatalf("runCompatibilityAutopilot calls %s only once -- expected a second call (the single retry); this guard's anchor point assumes exactly one retry and needs updating if that changed", buildCall)
	}
	secondBuildIdx := firstBuildIdx + len(buildCall) + secondBuildRel
	if strings.Contains(body[secondBuildIdx+len(buildCall):], buildCall) {
		t.Fatalf("runCompatibilityAutopilot calls %s a third time -- this guard assumes exactly one retry (two total calls) and needs updating if the retry count changed", buildCall)
	}

	tail := body[secondBuildIdx:]

	const recordCall = "recordAutopilotRetryExhaustion("
	recordRel := strings.Index(tail, recordCall)
	if recordRel == -1 {
		t.Fatal("recordAutopilotRetryExhaustion( does not appear after the retry's runCodexBuildWithOptions( call inside runCompatibilityAutopilot -- the retry-exhaustion record is not wired into the second-failure branch")
	}

	const pauseSyncCall = `syncRunAutopilotState(state, opts, "paused", "")`
	pauseRel := strings.Index(tail, pauseSyncCall)
	if pauseRel == -1 {
		t.Fatalf("%s not found after the retry's build call inside runCompatibilityAutopilot -- the second-failure pause call this guard anchors on may have changed; update this test to match", pauseSyncCall)
	}

	const returnStmt = "return nil, err"
	returnRel := strings.Index(tail, returnStmt)
	if returnRel == -1 {
		t.Fatal("\"return nil, err\" not found after the retry's build call inside runCompatibilityAutopilot -- the second-failure return this guard checks against may have changed; update this test to match")
	}

	if recordRel > pauseRel {
		t.Fatalf("recordAutopilotRetryExhaustion( appears AFTER the \"paused\" sync call (byte offset %d vs %d, relative to the retry's build call) -- it must run before the pause so the record is written on a best-effort basis, not after autopilot has already started pausing", recordRel, pauseRel)
	}
	if recordRel > returnRel {
		t.Fatalf("recordAutopilotRetryExhaustion( appears AFTER \"return nil, err\" (byte offset %d vs %d, relative to the retry's build call) -- dead code that never runs before the function returns", recordRel, returnRel)
	}
}
