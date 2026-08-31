package cmd

import (
	"errors"
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

// TestAutopilotWholeBuildRetryIsNotWired supersedes 188-05's old outer-retry
// guard. Provider readiness owns its bounded timeout retry now; the run loop
// invokes the whole build exactly once and cannot write a misleading
// "two attempts exhausted" record for a retry it no longer performs.
func TestAutopilotWholeBuildRetryIsNotWired(t *testing.T) {
	src, err := os.ReadFile(autopilotRetryCompatibilitySourceFile)
	if err != nil {
		t.Fatalf("read %s: %v", autopilotRetryCompatibilitySourceFile, err)
	}
	body := string(src)
	start := strings.Index(body, "func runCompatibilityAutopilot(")
	end := strings.Index(body[start:], "\nfunc loadCompatibilityColonyState(")
	if start < 0 || end < 0 {
		t.Fatal("locate runCompatibilityAutopilot source body")
	}
	runBody := body[start : start+end]
	if got := strings.Count(runBody, "runAutopilotBuild("); got != 1 {
		t.Fatalf("whole-build invocation sites = %d, want exactly 1", got)
	}
	if strings.Contains(runBody, "recordAutopilotRetryExhaustion(") {
		t.Fatal("run loop still records exhaustion for the removed whole-build retry")
	}
}
