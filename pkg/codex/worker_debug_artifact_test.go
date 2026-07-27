package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests cover LOUD-09 (decisions D-03/D-04/D-05): worker failure evidence
// must survive. The bar set in CONTEXT.md is that an operator could diagnose a
// failure from the terminal plus the debug file alone, without hunting session
// transcripts — so these assert on the CONTENT of the artifact (exit code,
// duration, provider run id, failure mode), not merely that a file appeared.

func readDebugArtifacts(t *testing.T, root string) []map[string]interface{} {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data", "worker-debug")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read worker-debug dir %s: %v", dir, err)
	}
	var out []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read artifact %s: %v", e.Name(), err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("artifact %s is not valid JSON: %v", e.Name(), err)
		}
		out = append(out, payload)
	}
	return out
}

// D-03: every failure mode leaves evidence, and that evidence carries the three
// facts CONTEXT.md named — exit code, duration, provider session id.
func TestWriteHostedWorkerOutputDebugRecordsEveryFailureMode(t *testing.T) {
	cases := []struct {
		failureMode string
		exitCode    int
		cause       error
	}{
		{"timeout", -1, errors.New("worker timed out after 600s")},
		{"non_zero_exit", 137, errors.New("worker exited 137")},
		{"provider_error_envelope", -1, errors.New("provider returned an error envelope")},
		{"parse_failure", -1, errors.New("could not parse worker output")},
	}

	for _, tc := range cases {
		t.Run(tc.failureMode, func(t *testing.T) {
			root := t.TempDir()
			config := WorkerConfig{
				WorkerName:    "Mason-67",
				Caste:         "builder",
				TaskID:        "160-05",
				AgentName:     "aether-builder",
				ProviderRunID: "run_abc123",
				Root:          root,
			}

			rel := writeHostedWorkerOutputDebug(
				root, "claude", config, []string{"--prompt", "secret prompt text"},
				"stdout text", "stderr text", tc.cause,
				hostedWorkerDebugDetails{Duration: 1500 * time.Millisecond, ExitCode: tc.exitCode, FailureMode: tc.failureMode},
			)
			if rel == "" {
				t.Fatalf("%s: no debug artifact path returned — the failure left no evidence", tc.failureMode)
			}
			if !strings.HasPrefix(rel, ".aether/data/worker-debug/") {
				t.Errorf("returned path %q should be repo-relative so the (debug: <path>) hint resolves from the user's cwd", rel)
			}

			artifacts := readDebugArtifacts(t, root)
			if len(artifacts) != 1 {
				t.Fatalf("%s: expected exactly 1 artifact, got %d", tc.failureMode, len(artifacts))
			}
			a := artifacts[0]

			if got := a["failure_mode"]; got != tc.failureMode {
				t.Errorf("failure_mode = %v, want %q", got, tc.failureMode)
			}
			if got, ok := a["exit_code"].(float64); !ok || int(got) != tc.exitCode {
				t.Errorf("exit_code = %v, want %d — an operator cannot distinguish a crash from a timeout without it", a["exit_code"], tc.exitCode)
			}
			if got, ok := a["duration_ms"].(float64); !ok || int(got) != 1500 {
				t.Errorf("duration_ms = %v, want 1500", a["duration_ms"])
			}
			if got := a["provider_run_id"]; got != "run_abc123" {
				t.Errorf("provider_run_id = %v, want run_abc123 — without it the provider-side session cannot be correlated", got)
			}
		})
	}
}

// D-03 (security): the new call sites must not bypass the redaction that the
// single chokepoint already performs. RESEARCH.md flags this as threat T-160-02.
func TestWriteHostedWorkerOutputDebugRedactsArgumentValues(t *testing.T) {
	root := t.TempDir()
	const secret = "sk-live-SHOULD-NOT-APPEAR-4f9a8b7c"

	rel := writeHostedWorkerOutputDebug(
		root, "claude", WorkerConfig{WorkerName: "w", Root: root},
		[]string{"--system-prompt", secret, "--prompt", secret},
		"", "", errors.New("boom"),
		hostedWorkerDebugDetails{Duration: time.Second, ExitCode: 1, FailureMode: "non_zero_exit"},
	)
	if rel == "" {
		t.Fatal("no artifact written")
	}

	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Errorf("debug artifact contains an unredacted argument value (%q) — a failing worker would leak credentials to disk", secret)
	}
}

// D-04: the cap is enforced on write, so the directory cannot grow unbounded
// even if nobody ever runs data-clean.
func TestWriteHostedWorkerOutputDebugEnforcesRetentionCapOnWrite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".aether", "data", "worker-debug")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Seed well past the cap with recent files (so age is not what prunes them).
	seeded := WorkerDebugRetentionMaxFiles + 25
	for i := 0; i < seeded; i++ {
		p := filepath.Join(dir, fmt.Sprintf("seed-%03d.json", i))
		if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		// Stagger mod times so oldest-first eviction is deterministic.
		mt := time.Now().Add(-time.Duration(seeded-i) * time.Minute)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	rel := writeHostedWorkerOutputDebug(
		root, "claude", WorkerConfig{WorkerName: "w", Root: root},
		nil, "", "", errors.New("boom"),
		hostedWorkerDebugDetails{Duration: time.Second, ExitCode: 1, FailureMode: "non_zero_exit"},
	)
	if rel == "" {
		t.Fatal("no artifact written")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > WorkerDebugRetentionMaxFiles {
		t.Errorf("worker-debug holds %d files, cap is %d — retention is not enforced on write", len(entries), WorkerDebugRetentionMaxFiles)
	}

	// The just-written artifact is the operator's evidence for the failure they
	// are about to see; pruning must never evict it.
	if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
		t.Errorf("the artifact just written was pruned away: %v", err)
	}
}

// D-04: age-based pruning removes stale evidence.
func TestPruneWorkerDebugArtifactsRemovesExpiredFiles(t *testing.T) {
	dir := t.TempDir()

	stale := filepath.Join(dir, "stale.json")
	fresh := filepath.Join(dir, "fresh.json")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-WorkerDebugRetentionMaxAge - time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	pruneWorkerDebugArtifacts(dir, "")

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("artifact older than %v should have been pruned", WorkerDebugRetentionMaxAge)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("recent artifact was pruned: %v", err)
	}
}

// D-05: in worktree mode the artifact must land on the tracking root, not
// inside the worktree, or `git worktree remove` destroys the evidence and the
// (debug: <path>) hint resolves to nothing from the user's cwd.
func TestWorkerDebugArtifactsResolveToTrackingRoot(t *testing.T) {
	trackingRoot := t.TempDir()
	worktreeRoot := t.TempDir()

	config := WorkerConfig{WorkerName: "w", Root: worktreeRoot, TrackingRoot: trackingRoot}
	resolved := workerTrackingRoot(config)

	if resolved != trackingRoot {
		t.Fatalf("workerTrackingRoot returned %q, want the tracking root %q — writing to the worktree root loses artifacts on `git worktree remove` (D-05)", resolved, trackingRoot)
	}

	rel := writeHostedWorkerOutputDebug(
		resolved, "claude", config, nil, "", "", errors.New("boom"),
		hostedWorkerDebugDetails{Duration: time.Second, ExitCode: 1, FailureMode: "non_zero_exit"},
	)
	if rel == "" {
		t.Fatal("no artifact written")
	}
	if _, err := os.Stat(filepath.Join(trackingRoot, rel)); err != nil {
		t.Errorf("artifact not found under the tracking root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktreeRoot, rel)); err == nil {
		t.Errorf("artifact was written inside the worktree at %s — it would not survive `git worktree remove`", filepath.Join(worktreeRoot, rel))
	}
}

// The tests above call writeHostedWorkerOutputDebug directly, so they would all
// still pass if the timeout and non-zero-exit CALL SITES were deleted — the very
// regression D-03 exists to prevent. (Plan 160-04 shipped exactly that blind
// spot in its first draft.) This asserts the wiring at the source level: every
// failure mode has a call site, and every call site resolves the tracking root.
func TestWorkerDebugCallSitesCoverEveryFailureModeAndUseTrackingRoot(t *testing.T) {
	raw, err := os.ReadFile("platform_dispatch.go")
	if err != nil {
		t.Fatalf("read platform_dispatch.go: %v", err)
	}
	src := string(raw)

	for _, mode := range []string{"timeout", "non_zero_exit", "provider_error_envelope", "parse_failure"} {
		needle := `FailureMode: "` + mode + `"`
		if !strings.Contains(src, needle) {
			t.Errorf("no writeHostedWorkerOutputDebug call site records FailureMode %q — that failure would leave no evidence on disk (D-03)", mode)
		}
	}

	callSites := 0
	for _, line := range strings.Split(src, "\n") {
		if !strings.Contains(line, "writeHostedWorkerOutputDebug(") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "func ") {
			continue
		}
		callSites++
		if !strings.Contains(line, "workerTrackingRoot(") {
			t.Errorf("writeHostedWorkerOutputDebug call site does not pass workerTrackingRoot(config); artifacts written to the worktree root are destroyed by `git worktree remove` (D-05). Line: %s", strings.TrimSpace(line))
		}
	}
	if callSites < 4 {
		t.Errorf("found %d writeHostedWorkerOutputDebug call sites, expected at least 4 (one per failure mode) — a failure path has lost its evidence write", callSites)
	}
}
