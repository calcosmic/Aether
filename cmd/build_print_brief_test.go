package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/codex"
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

// TestPrintBriefFullFlagAndCoverage pins Task 2's five behaviors: --full
// gates the raw prompt path, --worker scoping still works in both modes, and
// an unknown worker name still errors the same way it always has.
func TestPrintBriefFullFlagAndCoverage(t *testing.T) {
	t.Run("print-brief alone prints the checklist and not the raw brief", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}
		if !strings.Contains(out, "CONTEXT CHECKLIST") {
			t.Errorf("expected checklist heading, got:\n%s", out)
		}
		if strings.Contains(out, "COMPOSITION") {
			t.Errorf("--print-brief without --full should not print the composition table:\n%s", out)
		}
	})

	t.Run("print-brief --full prints the raw brief and the composition table", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t, "--full")
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}
		if !strings.Contains(out, "## Assignment") {
			t.Errorf("--full should print the raw brief with its Assignment heading:\n%s", out)
		}
		if !strings.Contains(out, "COMPOSITION") {
			t.Errorf("--full should print the composition table:\n%s", out)
		}
		if strings.Contains(out, "CONTEXT CHECKLIST") {
			t.Errorf("--full should not also print the checklist:\n%s", out)
		}
	})

	t.Run("print-brief --worker scopes to one worker in both checklist and full modes", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		fullOut, errOut := runPrintBriefCmd(t)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}
		names := workerNamesFromBanners(fullOut)
		if len(names) < 2 {
			t.Fatalf("fixture must produce at least two dispatches to prove scoping; got names: %v\noutput:\n%s", names, fullOut)
		}
		target := names[0]

		scopedChecklist, errOut := runPrintBriefCmd(t, "--worker", target)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}
		if got := workerNamesFromBanners(scopedChecklist); len(got) != 1 || got[0] != target {
			t.Errorf("checklist mode --worker %q scoped to %v, want exactly [%q]", target, got, target)
		}

		scopedFull, errOut := runPrintBriefCmd(t, "--full", "--worker", target)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}
		if got := workerNamesFromBanners(scopedFull); len(got) != 1 || got[0] != target {
			t.Errorf("--full mode --worker %q scoped to %v, want exactly [%q]", target, got, target)
		}
	})

	t.Run("full without print-brief is inert", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		printBriefFixture(t, basePrintBriefState())

		var outBuf bytes.Buffer
		stdout = &outBuf
		t.Setenv("AETHER_OUTPUT_MODE", "json")

		rootCmd.SetArgs([]string{"build", "1", "--full", "--synthetic"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build --full --synthetic returned error: %v", err)
		}

		var envelope map[string]interface{}
		if err := json.Unmarshal(bytes.TrimSpace(outBuf.Bytes()), &envelope); err != nil {
			t.Fatalf("build --full --synthetic did not produce normal build output: %v\n%s", err, outBuf.String())
		}
		if envelope["ok"] != true {
			t.Fatalf("build --full --synthetic should succeed like a normal build, got: %v", envelope)
		}
	})

	t.Run("unknown worker name still returns the existing error listing available names", func(t *testing.T) {
		saveGlobals(t)
		printBriefFixture(t, basePrintBriefState())

		out, errOut := runPrintBriefCmd(t, "--worker", "NoSuchWorker-999")
		if out != "" {
			t.Errorf("expected no checklist output on worker-not-found, got:\n%s", out)
		}
		if !strings.Contains(errOut, "no worker named") {
			t.Errorf("expected 'no worker named' error, got: %s", errOut)
		}
		if !strings.Contains(errOut, "NoSuchWorker-999") {
			t.Errorf("error should name the requested worker: %s", errOut)
		}
	})
}

// TestPrintBriefFailsOnDuplicatedSection is a direct unit test against
// duplicatedBriefSections itself (D-05): a real repeated owned heading is
// named, a real repeated handoff-schema sentence is named via its own label
// (the sentence carries no heading of its own, per 189's D-04 design), and a
// healthy construction with each present exactly once finds nothing.
func TestPrintBriefFailsOnDuplicatedSection(t *testing.T) {
	handoffSentence := "Your final result's handoff object must include " + codex.HandoffFieldsSummary + ". An empty handoff is rejected.\n"

	t.Run("a repeated owned heading is reported by name", func(t *testing.T) {
		assembled := "## Pheromone Signals\n\nFOCUS: watch the exporter\n\n## Pheromone Signals\n\nFOCUS: watch the exporter again\n"
		got := duplicatedBriefSections(assembled)
		if len(got) != 1 || got[0] != "Pheromone Signals" {
			t.Fatalf("duplicatedBriefSections = %v, want [\"Pheromone Signals\"]", got)
		}
	})

	t.Run("a repeated handoff-schema sentence is reported", func(t *testing.T) {
		assembled := "## Assignment\n\nDo the thing.\n\n" + handoffSentence + "\n" + handoffSentence
		got := duplicatedBriefSections(assembled)
		found := false
		for _, name := range got {
			if name == duplicatedBriefSectionHandoffLabel {
				found = true
			}
		}
		if !found {
			t.Fatalf("duplicatedBriefSections = %v, want it to contain %q", got, duplicatedBriefSectionHandoffLabel)
		}
	})

	t.Run("each owned heading once and the handoff sentence once finds nothing", func(t *testing.T) {
		assembled := "## Assignment\n\nDo the thing.\n\n## Pheromone Signals\n\nFOCUS: watch the exporter\n\n" + handoffSentence
		got := duplicatedBriefSections(assembled)
		if len(got) != 0 {
			t.Fatalf("duplicatedBriefSections = %v, want empty slice for healthy input, got %v", got, got)
		}
	})
}

// TestPrintBriefCommandStaysCleanOnHealthyFixtureWithoutDuplication is the
// positive-case proof this check does not misfire on ordinary,
// already-shipping output: basePrintBriefState()'s fixture seeds no
// pheromone signals, so neither the manifest-level capsule nor the composed
// brief carries a "## Pheromone Signals" section at all -- --print-brief must
// exit cleanly in both checklist and --full mode.
func TestPrintBriefCommandStaysCleanOnHealthyFixtureWithoutDuplication(t *testing.T) {
	saveGlobals(t)
	printBriefFixture(t, basePrintBriefState())

	out, errOut := runPrintBriefCmd(t)
	if errOut != "" {
		t.Fatalf("unexpected stderr on healthy fixture (checklist mode): %s", errOut)
	}
	if !strings.Contains(out, "CONTEXT CHECKLIST") {
		t.Errorf("expected checklist output on healthy fixture, got:\n%s", out)
	}

	fullOut, fullErrOut := runPrintBriefCmd(t, "--full")
	if fullErrOut != "" {
		t.Fatalf("unexpected stderr on healthy fixture (--full mode): %s", fullErrOut)
	}
	if !strings.Contains(fullOut, "COMPOSITION") {
		t.Errorf("expected composition table on healthy fixture, got:\n%s", fullOut)
	}
}

// TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication proves the
// duplication check catches a REAL duplicate through the full --print-brief
// command path, not just a hand-built string: colony-prime's own
// manifest-level capsule (cmd/colony_prime_context.go:571) and
// composeBuildManifestBrief's per-dispatch resolvePheromoneSection call
// (cmd/codex_build.go) both independently render a "## Pheromone Signals"
// section from the same pheromones.json whenever a signal is active, so
// seeding one active signal reproduces a genuine, already-shipping
// duplication rather than a synthetic one.
func TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication(t *testing.T) {
	saveGlobals(t)
	printBriefFixture(t, basePrintBriefState())

	recent := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{Type: "FOCUS", Content: json.RawMessage(`{"text":"sentinel-print-brief-duplication-probe"}`), Active: true, Strength: floatPtr(0.8), CreatedAt: recent},
		},
	}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("failed to save pheromones: %v", err)
	}

	out, errOut := runPrintBriefCmd(t)
	if out != "" {
		t.Errorf("expected no checklist output once duplication is found, got:\n%s", out)
	}
	if !strings.Contains(errOut, "duplicated context") {
		t.Fatalf("expected a duplicated-context error on stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Pheromone Signals") {
		t.Errorf("error should name the duplicated section, got: %s", errOut)
	}
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

// workerNamesFromBanners parses the "emoji  Name  (caste)" banner lines
// printWorkerBriefs writes above each worker's section, in both checklist
// and --full modes, and returns the ordered list of worker names found.
func workerNamesFromBanners(text string) []string {
	var names []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasSuffix(line, ")") {
			continue
		}
		open := strings.LastIndex(line, "(")
		if open < 0 {
			continue
		}
		nameBody := strings.TrimSpace(line[:open])
		fields := strings.Fields(nameBody)
		if len(fields) < 2 {
			continue
		}
		// First field is the caste emoji, the rest is the deterministic name.
		names = append(names, strings.Join(fields[1:], " "))
	}
	return names
}
