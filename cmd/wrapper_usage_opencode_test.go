package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// The fixture's fixed timestamps (see cmd/testdata/spend/opencode/README.md):
// ses_child_a and ses_child_b sit inside this window, ses_stale sits well
// before it.
var (
	openCodeFixtureWindowStart = time.UnixMilli(1786708800000)
	openCodeFixtureWindowEnd   = time.UnixMilli(1786712400000)
	openCodeFixtureStaleTime   = time.UnixMilli(1786352400000)
)

// copyOpenCodeFixture copies the committed OpenCode storage fixture into
// destRoot, rewriting the __REPO_ROOT__ placeholder to repoRoot so worktree
// discovery is exercised for real rather than stubbed. No test in this file
// reads the developer's real OpenCode store -- every test sets HOME to a
// t.TempDir() first (grep -c 'os.UserHomeDir()' on this file returns 0: the
// production code resolves HOME, this file never does).
func copyOpenCodeFixture(t *testing.T, destRoot, repoRoot string) {
	t.Helper()
	srcRoot := filepath.Join("testdata", "spend", "opencode")
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(srcRoot, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." || rel == "README.md" {
			return nil
		}
		destPath := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "__REPO_ROOT__") {
			data = []byte(strings.ReplaceAll(string(data), "__REPO_ROOT__", repoRoot))
		}
		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy opencode fixture: %v", err)
	}
}

func setupOpenCodeFixtureHome(t *testing.T) (storageRoot, repoRoot string) {
	t.Helper()
	home := t.TempDir()
	repoRoot = t.TempDir()
	t.Setenv("HOME", home)
	storageRoot = filepath.Join(home, ".local", "share", "opencode", "storage")
	copyOpenCodeFixture(t, storageRoot, repoRoot)
	return storageRoot, repoRoot
}

func TestOpenCodeSessionUsageReadsDisjointTokenColumns(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}

	var mason, vigil *openCodeSessionUsage
	for i := range entries {
		switch entries[i].WorkerName {
		case "Mason-67":
			mason = &entries[i]
		case "Vigil-12":
			vigil = &entries[i]
		}
	}
	if mason == nil {
		t.Fatalf("Mason-67 entry not found: %+v", entries)
	}
	// Literal figures from the committed fixture (cmd/testdata/spend/opencode/message/ses_child_a/msg_1.json),
	// never recomputed -- the four disjoint columns must survive intact.
	if mason.Usage.InputTokens != 543 {
		t.Errorf("mason InputTokens = %d, want 543", mason.Usage.InputTokens)
	}
	if mason.Usage.OutputTokens != 123 {
		t.Errorf("mason OutputTokens = %d, want 123", mason.Usage.OutputTokens)
	}
	if mason.Usage.CachedInputTokens != 19770 {
		t.Errorf("mason CachedInputTokens = %d, want 19770", mason.Usage.CachedInputTokens)
	}
	if mason.Usage.CacheCreationTokens != 0 {
		t.Errorf("mason CacheCreationTokens = %d, want 0", mason.Usage.CacheCreationTokens)
	}
	if mason.Usage.TotalTokens != 20436 {
		t.Errorf("mason TotalTokens = %d, want 20436", mason.Usage.TotalTokens)
	}
	if mason.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("mason Source = %q, want %q", mason.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	if vigil == nil {
		t.Fatalf("Vigil-12 entry not found: %+v", entries)
	}
	if vigil.Usage.TotalTokens != 8100 {
		t.Errorf("vigil TotalTokens = %d, want 8100", vigil.Usage.TotalTokens)
	}
	if vigil.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("vigil Source = %q, want %q", vigil.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	for _, e := range entries {
		if e.Usage.Source == codex.UsageSourceProvider {
			t.Errorf("entry %q tagged provider-grade; must never be", e.WorkerName)
		}
	}
}

func TestOpenCodeSessionUsageIgnoresSessionsOutsideTheRunWindow(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	// ses_stale carries the same worker name (Mason-67) but sits outside
	// the window -- without the time bound this would be ambiguous.
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want exactly 1 (stale decoy must be excluded by the window): %+v", len(entries), entries)
	}
	if entries[0].SessionID != "ses_child_a" {
		t.Fatalf("resolved session = %q, want ses_child_a", entries[0].SessionID)
	}
	if entries[0].Usage.TotalTokens == 999999 || entries[0].Usage.InputTokens == 999999 {
		t.Fatalf("stale session's impossible 999999 figure leaked through: %+v", entries[0].Usage)
	}
}

func TestOpenCodeSessionUsageRefusesAmbiguousWorkerMatch(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	// Widen the window to enclose both ses_child_a and ses_stale, so
	// Mason-67 now has two in-window candidates.
	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureStaleTime, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0 for an ambiguous match: %+v", len(entries), entries)
	}
	found := false
	for _, r := range reasons {
		if strings.Contains(r, "Mason-67") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a diagnostic naming Mason-67, got: %v", reasons)
	}
}

func TestOpenCodeSessionUsageIgnoresNonMatchingWorktree(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)
	otherRoot := t.TempDir()
	if otherRoot == repoRoot {
		t.Fatalf("test setup collision: otherRoot must differ from repoRoot")
	}

	entries, reasons := openCodeSessionUsageForRun(storageRoot, otherRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries for a non-matching worktree, want 0: %+v", len(entries), entries)
	}
	if len(reasons) == 0 {
		t.Fatalf("expected a diagnostic naming the non-matching repo root")
	}
}

func TestOpenCodeSessionUsageRefusesStorageRootOutsideHome(t *testing.T) {
	home := t.TempDir()
	repoRoot := t.TempDir()
	t.Setenv("HOME", home)

	outsideRoot := t.TempDir() // sibling temp dir, not under $HOME/.local/share/opencode/storage
	entries, reasons := openCodeSessionUsageForRun(outsideRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries for an out-of-root storage path, want 0", len(entries))
	}
	if len(reasons) == 0 {
		t.Fatalf("expected a diagnostic rejecting the out-of-root storage path")
	}

	// An absent storage root (no OpenCode installed at all) must fail
	// soft, never panic -- this is the normal case on a Claude-Code-only
	// machine.
	freshHome := t.TempDir()
	t.Setenv("HOME", freshHome)
	if _, ok := openCodeSessionUsageForRunOrNone(repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"}); ok {
		t.Fatalf("expected ok=false when the opencode storage root is entirely absent")
	}
}

func TestOpenCodeSessionUsageToleratesMalformedRecords(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	malformedPath := filepath.Join(storageRoot, "session", "prj_fixture", "ses_malformed.json")
	if err := os.WriteFile(malformedPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write malformed session file: %v", err)
	}

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 despite the malformed sibling session file: %+v", len(entries), entries)
	}
}
