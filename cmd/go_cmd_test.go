package cmd

// Phase 209 plan 01, Task 1 -- `/ant-go` end-to-end on the small route: one
// sentence naming a real file, one helper dispatched, the project's own
// checks run, and a rendered screen whose first content line names the
// route and why.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestGoSmallRouteRunsTheJobEndToEnd drives the real `aether go "fix the
// typo in README.md"` (via runGoJob, the command's own body) through a
// temp repository that genuinely contains README.md, with
// newQuickWorkerInvoker swapped for a quickJobCaptureInvoker.
func TestGoSmallRouteRunsTheJobEndToEnd(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Title\n\nSpeling mistake here.\n"), 0o644); err != nil {
		t.Fatalf("write fixture README: %v", err)
	}

	spy := &quickJobCaptureInvoker{result: codex.WorkerResult{Summary: "fixed the typo", FilesModified: []string{"README.md"}}}
	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
	origChecks := runQuickDeterministicChecks
	runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
		return quickChecksPassed, []string{"go build ./...: passed"}, nil
	}
	t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

	result, err := runGoJob("fix the typo in README.md", 2*time.Second)
	if err != nil {
		t.Fatalf("runGoJob: %v", err)
	}

	configs := spy.captured()
	if len(configs) != 1 {
		t.Fatalf("expected exactly one worker dispatch, got %d", len(configs))
	}
	if configs[0].Caste != "builder" {
		t.Fatalf("expected the builder to be dispatched, got Caste=%q", configs[0].Caste)
	}

	if got := stringValue(result["route"]); got != string(jobSizeRouteSmall) {
		t.Fatalf("result[\"route\"] = %q, want %q", got, jobSizeRouteSmall)
	}
	reason := strings.TrimSpace(stringValue(result["route_reason"]))
	if reason == "" {
		t.Fatal("result[\"route_reason\"] is empty")
	}
	if !strings.Contains(reason, "1") {
		t.Errorf("route_reason does not name the matched-file count: %q", reason)
	}

	// runQuickJob's own keys must reach the result rather than being
	// recomputed by runGoJob.
	for _, key := range []string{"work_outcome", "files", "checks_status"} {
		if _, ok := result[key]; !ok {
			t.Errorf("result is missing %q, which runQuickJob's own result map always sets", key)
		}
	}
	verdict, _ := result["work_outcome"].(colony.WorkOutcome)
	if verdict != colony.WorkOutcomeSuccess {
		t.Fatalf("work_outcome = %v, want %v", verdict, colony.WorkOutcomeSuccess)
	}

	rendered := renderGoVisual(result)
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("rendered screen has too few lines:\n%s", rendered)
	}
	if !isAetherBannerLine(lines[0]) {
		t.Fatalf("rendered screen's first line is not a banner line: %q", lines[0])
	}
	// lines[1] is the plain divider; the first CONTENT line is lines[2].
	firstContentLine := ""
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" || strings.Trim(line, "─━ ") == "" {
			continue
		}
		firstContentLine = line
		break
	}
	if !strings.Contains(firstContentLine, reason) {
		t.Fatalf("rendered screen's first content line does not carry the route reason:\nfirst content line: %q\nreason: %q\nfull screen:\n%s", firstContentLine, reason, rendered)
	}
}

// TestGoNeverRefusesTheOwner proves no new refusal was added: an empty
// repository (nothing matches, and no project is set up) still gets a
// route and a reason back, never an error.
func TestGoNeverRefusesTheOwner(t *testing.T) {
	saveGlobals(t)
	store = nil
	root := t.TempDir()
	chdirForTest190_05(t, root)

	result, err := runGoJob("build something entirely new that does not exist yet", 2*time.Second)
	if err != nil {
		t.Fatalf("runGoJob returned an error instead of naming a route: %v", err)
	}
	if got := stringValue(result["route"]); got != string(jobSizeRouteBig) {
		t.Fatalf("result[\"route\"] = %q, want %q for an empty repository", got, jobSizeRouteBig)
	}
	if strings.TrimSpace(stringValue(result["route_reason"])) == "" {
		t.Fatal("result[\"route_reason\"] is empty")
	}

	rendered := renderGoVisual(result)
	if strings.TrimSpace(rendered) == "" {
		t.Fatal("renderGoVisual produced an empty screen")
	}
}

// TestGoCommandIsRegistered is the minimal wiring check: `aether go` is a
// real, non-hidden cobra command with a --timeout flag matching quickCmd's.
func TestGoCommandIsRegistered(t *testing.T) {
	target, _, err := rootCmd.Find([]string{"go"})
	if err != nil || target == nil || target == rootCmd {
		t.Fatal("`aether go` is not a registered cobra command")
	}
	if target.Hidden {
		t.Fatal("`aether go` is registered but hidden")
	}
	if target.Flags().Lookup("timeout") == nil {
		t.Fatal("`aether go` has no --timeout flag")
	}
}

// TestGoCmdSourceCarriesNoSymbolLiteral asserts cmd/go_cmd.go contains no
// emoji or box-drawing character literal outside renderBanner's own call --
// every content line must be produced by voiceLine or renderBanner.
func TestGoCmdSourceCarriesNoSymbolLiteral(t *testing.T) {
	data, err := os.ReadFile("go_cmd.go")
	if err != nil {
		t.Fatalf("read go_cmd.go: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(trimmed, `renderBanner("🧭"`) {
			continue
		}
		for _, forbidden := range []string{"━", "✅", "❌", "⚠️", "🚩", "📁", "🐜"} {
			if strings.Contains(line, forbidden) {
				t.Errorf("go_cmd.go carries a symbol literal outside renderBanner: %q", line)
			}
		}
	}
}
