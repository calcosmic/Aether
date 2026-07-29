package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// printBriefFixture points the process cwd and the store global at a fresh
// temp colony root, so skillWorkspaceRoot() (which printWorkerBriefs' CLI
// wiring depends on) resolves root correctly and every reader that keys off
// store.BasePath() or root sees the same colony. Mirrors the pattern
// TestBuildWritesDispatchArtifactsAndUpdatesState already established:
// setupBuildFlowTest-style store creation plus an explicit chdir.
func printBriefFixture(t *testing.T, state colony.ColonyState) (root, dataDir string) {
	t.Helper()
	root = t.TempDir()
	dataDir = filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	return root, dataDir
}

// basePrintBriefState is a minimal but realistic colony state: one phase,
// two tasks (so at least two dispatches, per the --worker scoping
// requirement), and an approved charter so the capsule/charter rows have
// real content to report on.
func basePrintBriefState() colony.ColonyState {
	goal := "print-brief inspector fixture"
	return colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Wire the exporter",
					Description:     "Connect the vault exporter to the dashboard command",
					Status:          colony.PhaseReady,
					SuccessCriteria: []string{"Dashboard renders exporter output"},
					Tasks: []colony.Task{
						{Goal: "Research the exporter contract", Status: colony.TaskPending, Hints: []string{"src/app.ts"}},
						{Goal: "Implement the exporter wiring", Status: colony.TaskPending, Hints: []string{"src/app.ts"}},
					},
				},
			},
		},
		Charter: &colony.Charter{
			Governance:  "Linting: golangci-lint",
			Constraints: "No new external dependencies",
		},
	}
}

func writeCodegraphFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	graph := codegraph.CodeGraph{
		Files: []codegraph.FileNode{
			{Path: "src/app.ts", Language: "typescript"},
			{Path: "src/util.ts", Language: "typescript"},
		},
		Edges: []codegraph.DepEdge{{Source: "src/app.ts", Target: "src/util.ts", Type: "import"}},
	}
	if err := graph.Save(filepath.Join(dir, "codebase-graph.json")); err != nil {
		t.Fatalf("save codegraph: %v", err)
	}
}

func writePhaseResearchFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir research dir: %v", err)
	}
	body := "## Key Patterns\n\nUse the widget factory pattern for the exporter wiring."
	if err := os.WriteFile(filepath.Join(dir, "phase-1-research.md"), []byte(body), 0644); err != nil {
		t.Fatalf("write research doc: %v", err)
	}
}

// gitInitPrintBriefRepo and gitCommitsPrintBriefRepo build a real git history
// so surveyStalenessNotice's commit-count path (which shells out to `git
// rev-list`) has something real to count, matching the pattern
// cmd/survey_staleness_test.go already uses for the same reason.
func gitInitPrintBriefRepo(t *testing.T, root string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@aether.invalid")
	run("config", "user.name", "Aether Test")
}

func gitCommitsPrintBriefRepo(t *testing.T, root string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		cmd := exec.Command("git", "commit", "--allow-empty", "-m", "commit "+strconv.Itoa(i))
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git commit failed: %v\n%s", err, out)
		}
	}
}

// runPrintBriefCmd executes `build 1 --print-brief` (plus any extra args)
// through the real CLI and returns captured stdout and stderr. Errors are
// reported through outputError to stderr as a JSON envelope, not through a
// Go error from Execute, so callers checking the "unknown worker" behavior
// must inspect stderr rather than the return value of Execute.
func runPrintBriefCmd(t *testing.T, extraArgs ...string) (stdoutText, stderrText string) {
	t.Helper()
	resetRootCmd(t)
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf

	args := append([]string{"build", "1", "--print-brief"}, extraArgs...)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build %v returned error: %v", args, err)
	}
	return outBuf.String(), errBuf.String()
}

// TestPrintBriefChecklist pins Task 1's five behaviors: the default
// `--print-brief` output is a checklist, not the raw prompt.
func TestPrintBriefChecklist(t *testing.T) {
	t.Run("default output lists every expected section with present/absent and a char count", func(t *testing.T) {
		saveGlobals(t)

		root, _ := printBriefFixture(t, basePrintBriefState())
		writeCodegraphFixture(t, root)
		writePhaseResearchFixture(t, root)

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		for _, label := range []string{
			"Assignment & Task Content",
			"Territory Survey",
			"Phase Research",
			"Codegraph Context",
			"Pheromone Signals",
			"Previous Worker Handoffs",
			"Expected Output",
			"Context Capsule",
			"Charter",
		} {
			if !strings.Contains(out, label) {
				t.Errorf("checklist missing expected row %q:\n%s", label, out)
			}
		}

		// Every row must carry a numeric char count, not just a label. A
		// present-and-populated row (Phase Research, real content on disk)
		// is the concrete proof: assert its line has a nonzero count next to
		// "present", not merely that the string "Phase Research" appears.
		researchLine := lineContaining(out, "Phase Research")
		if researchLine == "" {
			t.Fatalf("no Phase Research line found:\n%s", out)
		}
		if !strings.Contains(researchLine, "present") {
			t.Errorf("Phase Research line should read present with real content on disk: %q", researchLine)
		}
		if strings.Contains(researchLine, " present         0") || strings.Contains(researchLine, " present       0") {
			t.Errorf("Phase Research line reports zero chars despite real content: %q", researchLine)
		}
	})

	t.Run("no survey artifacts shows the survey row as ABSENT rather than omitted", func(t *testing.T) {
		saveGlobals(t)

		// Deliberately no survey dir and no TerritorySurveyed timestamp.
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		line := lineContaining(out, "Territory Survey")
		if line == "" {
			t.Fatalf("Territory Survey row missing entirely -- absence must be reported, not omitted:\n%s", out)
		}
		if !strings.Contains(line, "ABSENT") {
			t.Errorf("Territory Survey row should read ABSENT when no survey exists: %q", line)
		}
	})

	t.Run("context capsule appears as its own row sourced from the manifest-level value", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		capsuleLine := lineContaining(out, "Context Capsule")
		if capsuleLine == "" {
			t.Fatalf("Context Capsule row missing:\n%s", out)
		}
		if !strings.Contains(capsuleLine, "present") {
			t.Errorf("Context Capsule row should be present (charter + state always produce capsule content): %q", capsuleLine)
		}

		// The charter row is the direct proof this comes from the
		// manifest-level capsule, not any dispatch brief: renderCodexBuildWorkerBrief
		// (the per-dispatch renderer) never mentions Charter at all.
		charterLine := lineContaining(out, "Charter")
		if charterLine == "" {
			t.Fatalf("Charter row missing:\n%s", out)
		}
		if !strings.Contains(charterLine, "present") {
			t.Errorf("Charter row should read present -- state.Charter.Governance is populated in the fixture: %q", charterLine)
		}
	})

	t.Run("survey staleness notice appears as its own row or annotation when the map is stale", func(t *testing.T) {
		saveGlobals(t)

		state := basePrintBriefState()
		surveyedAt := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
		state.TerritorySurveyed = &surveyedAt

		root, dataDir := printBriefFixture(t, state)
		gitInitPrintBriefRepo(t, root)
		// A survey doc must exist for resolveSurveySection to emit the
		// Territory Survey heading in the first place; the staleness notice
		// is prepended inside that same section.
		surveyDir := filepath.Join(dataDir, "survey")
		if err := os.MkdirAll(surveyDir, 0755); err != nil {
			t.Fatalf("mkdir survey dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(surveyDir, "BLUEPRINT.md"), []byte("# Blueprint\n"), 0644); err != nil {
			t.Fatalf("write survey doc: %v", err)
		}
		gitCommitsPrintBriefRepo(t, root, surveyStaleCommitThreshold)

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		if !strings.Contains(out, "STALE MAP WARNING") {
			t.Errorf("checklist does not surface the stale-map warning as a row annotation:\n%s", out)
		}
	})

	t.Run("default output does not contain the raw brief body", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		if strings.Contains(out, "## Assignment") {
			t.Errorf("default checklist output should not print raw brief markdown headings:\n%s", out)
		}
		if strings.Contains(out, "Research the exporter contract") {
			t.Errorf("default checklist output should not print the raw task text:\n%s", out)
		}
	})
}

// lineContaining returns the first line of text containing substr, or "".
func lineContaining(text, substr string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}
