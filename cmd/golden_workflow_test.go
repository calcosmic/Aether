package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// goldenTestdataDir returns the absolute path to cmd/testdata/ in the source tree.
// This is needed because some tests change the working directory.
func goldenTestdataDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata"
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// stripANSI removes ANSI escape sequences from a string.
// Reimplements pkg/codex/platform_dispatch.go:stripANSIEscapeCodes (unexported).
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// workerNameRe matches deterministic worker name patterns like "Hammer-22", "Forge-41".
// The prefix varies based on temp directory paths used as hash seeds, so we replace
// the entire match with a caste-agnostic placeholder for golden file stability.
var workerNameRe = regexp.MustCompile(`\b[A-Z][a-z]+-\d{1,3}\b`)

// normalizeWorkerNames replaces all worker name patterns (CapitalWord-Number)
// with a fixed placeholder so golden files are stable across test runs.
// Worker names are hash-based on temp directory paths, making them non-deterministic.
func normalizeWorkerNames(s string) string {
	return workerNameRe.ReplaceAllString(s, "Worker-XX")
}

// normalizeForGolden prepares output for golden comparison by:
// 1. Stripping ANSI escape codes
// 2. Normalizing non-deterministic worker names
// 3. Removing ceremony activity lines (non-deterministic concurrent output)
func normalizeForGolden(s string) string {
	clean := stripANSI(s)
	clean = normalizeWorkerNames(clean)

	var filtered strings.Builder
	for _, line := range strings.Split(clean, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip ceremony log lines -- these are concurrent and non-deterministic
		if strings.HasPrefix(trimmed, "[CEREMONY]") {
			continue
		}
		// Skip COLONY ACTIVITY blocks and their indented content
		if strings.HasPrefix(trimmed, "COLONY ACTIVITY") {
			continue
		}
		// Skip watch-status wave progress lines (e.g., "Wave 11: 0/1 starting")
		if matched, _ := regexp.MatchString(`^Wave \d+: \d+/\d+ `, trimmed); matched {
			continue
		}
		// Skip activity block section headers
		if trimmed == "Context:" || trimmed == "Completed:" || trimmed == "Active:" {
			continue
		}
		// Skip indented ceremony context lines (activity block content)
		if strings.HasPrefix(trimmed, "ceremony.") {
			continue
		}
		// Skip indented worker status within activity blocks
		// (e.g., "  🔨 Builder:Worker-XX completed task=g-task-1...")
		if strings.HasPrefix(line, "  ") && (strings.Contains(line, "completed task=") ||
			strings.Contains(line, "starting task=") ||
			strings.Contains(line, "running task=") ||
			strings.Contains(line, "simulated worker heartbeat")) {
			continue
		}
		// Skip indented worker name references in activity blocks
		// (e.g., "  🔨 Builder:Worker-XX starting task=g-task-1 Worker: Worker-XX")
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "Worker: Worker-XX") {
			continue
		}
		// Skip wave progress table lines
		if strings.Contains(trimmed, "WAVE") && strings.Contains(trimmed, "DISPATCHED") {
			continue
		}
		if strings.Contains(trimmed, "Total") && strings.Contains(trimmed, "dispatches") && strings.Contains(trimmed, "succeeded") {
			continue
		}
		// Skip table separator lines (pure +---+---+)
		if matched, _ := regexp.MatchString(`^\+[-+]+\+$`, trimmed); matched {
			continue
		}
		if matched, _ := regexp.MatchString(`^\|.*\|.*\|.*\|`, trimmed); matched {
			continue
		}

		filtered.WriteString(line)
		filtered.WriteString("\n")
	}
	return strings.TrimSpace(filtered.String()) + "\n"
}

// compareGolden strips ANSI from got, normalizes non-deterministic content, and
// (if -update-golden) or reads and compares against the existing golden file.
// Uses the shared updateGolden flag from audit_catalog_test.go.
func compareGolden(t *testing.T, goldenPath, got string) {
	t.Helper()
	clean := normalizeForGolden(got)

	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(clean), 0644); err != nil {
			t.Fatalf("write golden file %s: %v", goldenPath, err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %s: %v (run with -update-golden to create)", goldenPath, err)
	}

	if clean != strings.TrimRight(string(data), "\n\t ")+"\n" {
		t.Errorf("golden mismatch for %s; run with -update-golden to refresh", goldenPath)
		// Show first difference for debugging
		gotLines := strings.Split(clean, "\n")
		wantLines := strings.Split(string(data), "\n")
		maxLen := len(gotLines)
		if len(wantLines) > maxLen {
			maxLen = len(wantLines)
		}
		for i := 0; i < maxLen; i++ {
			var g, w string
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Logf("  first diff at line %d:\n    got:  %q\n    want: %q", i+1, g, w)
				return
			}
		}
	}
}

func TestGoldenPlanVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goldenPath := filepath.Join(goldenTestdataDir(), "golden_plan.txt")

	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Golden workflow test colony"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"plan"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	compareGolden(t, goldenPath, output)

	// Verify golden content expectations (only when not updating)
	if !*updateGolden {
		clean := normalizeForGolden(output)
		for _, want := range []string{"P L A N", "P L A N   D I S P A T C H", "Planning Wave", "aether build 1"} {
			if !strings.Contains(clean, want) {
				t.Errorf("plan golden output missing %q", want)
			}
		}
	}
}

func TestGoldenBuildVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goldenPath := filepath.Join(goldenTestdataDir(), "golden_build.txt")

	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Golden workflow test colony"
	taskOneID := "g-task-1"
	taskTwoID := "g-task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Golden phase",
					Status: colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskOneID, Goal: "First golden task", Status: colony.TaskPending},
						{ID: &taskTwoID, Goal: "Second golden task", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
					},
				},
			},
		},
	})

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	compareGolden(t, goldenPath, output)

	// Verify golden content expectations (only when not updating)
	if !*updateGolden {
		clean := normalizeForGolden(output)
		for _, want := range []string{
			"B U I L D   D I S P A T C H   1", "S P A W N   P L A N",
			"Builder", "Watcher",
			"── Context ──", "── Tasks ──", "── Dispatch ──",
			"── Verification", "── Housekeeping ──",
			"── Colony Complete ──",
			"It's safe to clear your context now.",
		} {
			if !strings.Contains(clean, want) {
				t.Errorf("build golden output missing %q", want)
			}
		}
	}
}

func TestGoldenContinueVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goldenPath := filepath.Join(goldenTestdataDir(), "golden_continue.txt")

	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Golden workflow test colony"
	now := mustParseRFC3339(t, "2026-04-20T11:00:00Z")
	taskID := "g-task-1"
	nextTaskID := "g-task-2"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Golden phase",
					Status: colony.PhaseInProgress,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Golden builder task", Status: colony.TaskInProgress}},
				},
				{
					ID:     2,
					Name:   "Next golden phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Next golden task", Status: colony.TaskPending}},
				},
			},
		},
	})

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Golden builder task", Status: "spawned", TaskID: taskID},
		{Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification", Status: "spawned"},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Golden phase", goal, dispatches)

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	compareGolden(t, goldenPath, output)

	// Verify golden content expectations (only when not updating)
	if !*updateGolden {
		clean := normalizeForGolden(output)
		for _, want := range []string{"Verification"} {
			if !strings.Contains(clean, want) {
				t.Errorf("continue golden output missing %q", want)
			}
		}
	}
}
