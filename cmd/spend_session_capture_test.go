package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// newSpendSessionTranscriptFixture points HOME at a fresh temp directory and
// creates a real <home>/.claude/projects/<encoded>/<sessionID>.jsonl file, so
// a "valid path" test case is genuinely valid because the file exists inside
// the boundary validateSpendTranscriptPath enforces -- not merely because
// validation was skipped.
func newSpendSessionTranscriptFixture(t *testing.T, sessionID string) (home, transcriptPath string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, ".claude", "projects", "-Users-test-repo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir claude projects fixture dir: %v", err)
	}
	transcriptPath = filepath.Join(projectDir, sessionID+".jsonl")
	if err := os.WriteFile(transcriptPath, []byte(`{"type":"summary"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript fixture: %v", err)
	}
	return home, transcriptPath
}

// TestSpendSessionCaptureRecordsTranscriptPath is the plan's first behavior:
// a PreToolUse payload with a non-empty transcript_path writes
// .aether/data/spend/session.json containing that path, the session id, the
// cwd, and an RFC3339 UTC capture time.
func TestSpendSessionCaptureRecordsTranscriptPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	_, transcriptPath := newSpendSessionTranscriptFixture(t, "sess-record")

	input := claudeHookInput{
		HookEventName:  "PreToolUse",
		ToolName:       "Agent",
		SessionID:      "sess-record",
		Cwd:            "/Users/callumcowie/repos/Aether",
		TranscriptPath: transcriptPath,
	}
	recordSpendSessionFromHook(input)

	var record spendSessionRecord
	if err := s.LoadJSON(spendSessionRel, &record); err != nil {
		t.Fatalf("load spend session record: %v", err)
	}
	if record.SchemaVersion != spendSessionSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", record.SchemaVersion, spendSessionSchemaVersion)
	}
	if record.Platform != "claude-code" {
		t.Fatalf("Platform = %q, want claude-code", record.Platform)
	}
	if record.SessionID != "sess-record" {
		t.Fatalf("SessionID = %q, want sess-record", record.SessionID)
	}
	if record.TranscriptPath != transcriptPath {
		t.Fatalf("TranscriptPath = %q, want %q", record.TranscriptPath, transcriptPath)
	}
	if record.Cwd != "/Users/callumcowie/repos/Aether" {
		t.Fatalf("Cwd = %q, want the payload cwd", record.Cwd)
	}
	parsed, err := time.Parse(time.RFC3339, record.CapturedAt)
	if err != nil {
		t.Fatalf("CapturedAt %q is not RFC3339: %v", record.CapturedAt, err)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("CapturedAt %q did not round-trip as UTC", record.CapturedAt)
	}
}

// TestSpendSessionCaptureIsStableAcrossRepeatedDispatches is the plan's
// second behavior: a second identical payload leaves the file
// byte-identical -- the capture rewrites only when the recorded transcript
// path or session id actually changes.
func TestSpendSessionCaptureIsStableAcrossRepeatedDispatches(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s

	_, transcriptPath := newSpendSessionTranscriptFixture(t, "sess-stable")
	input := claudeHookInput{
		HookEventName:  "PreToolUse",
		ToolName:       "Agent",
		SessionID:      "sess-stable",
		Cwd:            "/repo",
		TranscriptPath: transcriptPath,
	}
	recordSpendSessionFromHook(input)

	recordPath := filepath.Join(s.BasePath(), "spend", "session.json")
	if _, err := os.Stat(recordPath); err != nil {
		t.Fatalf("expected a spend session record after the first dispatch: %v", err)
	}
	before := hashFileForTest(t, recordPath)

	// A second, identical dispatch -- same transcript path, same session id.
	recordSpendSessionFromHook(input)
	after := hashFileForTest(t, recordPath)

	if before != after {
		t.Fatalf("spend session record changed on a repeated identical dispatch: before=%s after=%s", before, after)
	}
}

// assertSpendSessionCaptureRefused calls recordSpendSessionFromHook with the
// given transcript path and asserts no record file was written at all --
// the boundary check refused before any store write was attempted.
func assertSpendSessionCaptureRefused(t *testing.T, s *storage.Store, transcriptPath, sessionID string) {
	t.Helper()
	input := claudeHookInput{
		HookEventName:  "PreToolUse",
		ToolName:       "Agent",
		SessionID:      sessionID,
		Cwd:            "/repo",
		TranscriptPath: transcriptPath,
	}
	recordSpendSessionFromHook(input)

	recordPath := filepath.Join(s.BasePath(), "spend", "session.json")
	if _, err := os.Stat(recordPath); err == nil {
		t.Fatalf("expected no spend session record to be written for refused transcript path %q", transcriptPath)
	}
}

// TestSpendSessionCaptureRefusesPathOutsideClaudeProjects is the plan's
// fourth behavior: a payload whose transcript_path contains a null byte, is
// relative, or resolves outside $HOME/.claude/projects/ is refused and
// nothing is written. The sibling-directory-name subtest is T-174-01's
// direct proof that containment is decided by filepath.Rel, not a bare
// strings.HasPrefix that a crafted "projects-evil" sibling could defeat.
func TestSpendSessionCaptureRefusesPathOutsideClaudeProjects(t *testing.T) {
	t.Run("outside_home_tree", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, _ := newTestStore(t)
		store = s
		home := t.TempDir()
		t.Setenv("HOME", home)
		assertSpendSessionCaptureRefused(t, s, "/etc/passwd", "sess-outside-1")
	})

	t.Run("relative_path", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, _ := newTestStore(t)
		store = s
		home := t.TempDir()
		t.Setenv("HOME", home)
		assertSpendSessionCaptureRefused(t, s, "relative/session.jsonl", "sess-outside-2")
	})

	t.Run("null_byte", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, _ := newTestStore(t)
		store = s
		home := t.TempDir()
		t.Setenv("HOME", home)
		assertSpendSessionCaptureRefused(t, s, "/tmp/evil\x00.jsonl", "sess-outside-3")
	})

	t.Run("sibling_directory_name_collision_not_containment", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, _ := newTestStore(t)
		store = s
		home := t.TempDir()
		t.Setenv("HOME", home)

		siblingDir := filepath.Join(home, ".claude", "projects-evil")
		if err := os.MkdirAll(siblingDir, 0o755); err != nil {
			t.Fatalf("mkdir sibling dir: %v", err)
		}
		siblingPath := filepath.Join(siblingDir, "session.jsonl")
		if err := os.WriteFile(siblingPath, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write sibling fixture: %v", err)
		}
		assertSpendSessionCaptureRefused(t, s, siblingPath, "sess-outside-4")
	})
}

// TestSpendSessionCaptureFailureDoesNotAffectHookDecision is the plan's
// fifth behavior and T-174-09's direct proof: with the store deliberately
// made unwritable, the hook still returns its normal allow answer -- a
// capture failure is invisible to the dispatch decision. This drives the
// real `hook-pre-tool-use` CLI surface, not recordSpendSessionFromHook
// directly, because the claim under test is about the HOOK's answer.
func TestSpendSessionCaptureFailureDoesNotAffectHookDecision(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, _ := newTestStore(t)
	store = s

	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, ".claude", "projects", "-repo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir claude projects fixture dir: %v", err)
	}
	transcriptPath := filepath.Join(projectDir, "sess-unwritable.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write transcript fixture: %v", err)
	}

	// Make the store's data directory read-only so the "spend/" subdirectory
	// cannot be created and the capture write fails.
	dataDir := s.BasePath()
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatalf("chmod data dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dataDir, 0o755) })

	payload := `{"hook_event_name":"PreToolUse","tool_name":"Agent","session_id":"sess-unwritable","transcript_path":"` +
		transcriptPath + `","tool_input":{"description":"Dispatch a first-tier worker"}}`
	setHookStdin(t, payload)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("a spend-capture write failure changed the hook's decision, got %q", buf.String())
	}

	recordPath := filepath.Join(dataDir, "spend", "session.json")
	if _, err := os.Stat(recordPath); err == nil {
		t.Fatalf("expected the capture write to have failed against the read-only store, but a record was written")
	}
}
