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
						{Goal: "Implement the exporter wiring", Status: colony.TaskPending, Hints: []string{"src/util.ts"}},
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

// TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs is the
// fail-then-pass proof for Phase 190 Plan 03 (D-190-01-A, "one home each").
//
// Its predecessor, TestPrintBriefCommandFailsWhenPrintWorkerBriefsFindsDuplication,
// proved the opposite fact: before this fix, colony-prime's own manifest-level
// capsule (cmd/colony_prime_context.go:571,695) and composeBuildManifestBrief's
// per-dispatch resolvePheromoneSection/HandoffSection embedding
// (cmd/codex_build.go) both independently rendered "## Pheromone Signals" and
// "## Previous Worker Handoffs" from the same underlying data, so seeding one
// active signal plus one stored handoff reproduced a genuine,
// already-shipping duplication:
//
//	{"ok":false,"error":"dispatch Dash-21 delivers duplicated context: Pheromone
//	Signals, Previous Worker Handoffs (each owned section and the handoff
//	schema must appear exactly once in the assembled worker context)","code":1}
//
// composeBuildManifestBrief now omits both sections when a capsule will also
// be prepended (attachBuildDispatchContext's includeSteeringSections=false),
// so the identical fixture must now pass cleanly, with the capsule as the
// sole channel. Once-ness is asserted positively (count == 1 in the --full
// assembled output), not just by the absence of an error -- an error-free
// exit would also happen if both sections had silently dropped to zero,
// which is not what "one home each" means.
func TestPrintBriefStaysCleanWithActiveSignalAndStoredHandoffs(t *testing.T) {
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

	// Stored handoff from a worker that will not itself be dispatched in this
	// fixture, so renderWorkerHandoffSection's own-worker exclusion never
	// filters it out for any dispatch under test.
	handoffs := workerHandoffFile{
		Entries: []workerHandoffRecord{
			{
				ID:                 "print-brief-fixture-1",
				Workflow:           "build",
				Phase:              1,
				WorkerName:         "PriorFixtureWorker-1",
				Status:             "completed",
				VerificationStatus: "pass",
				Summary:            "sentinel-print-brief-handoff-probe",
				Freshness:          recent,
			},
		},
	}
	if err := store.SaveJSON(workerHandoffsPath, handoffs); err != nil {
		t.Fatalf("failed to save worker handoffs: %v", err)
	}

	// Checklist mode (the default): must exit clean, and both sections must
	// report present -- proving delivery moved to the capsule, not to
	// nowhere. checklistRowForEither is what makes this row honest once the
	// content lives in the capsule instead of the brief.
	checklistOut, checklistErr := runPrintBriefCmd(t)
	if checklistErr != "" {
		t.Fatalf("expected --print-brief to pass cleanly with an active signal and a stored handoff, got stderr: %s", checklistErr)
	}
	pheromoneLine := lineContaining(checklistOut, "Pheromone Signals")
	if pheromoneLine == "" || !strings.Contains(pheromoneLine, "present") {
		t.Errorf("Pheromone Signals checklist row should read present (delivered via the capsule): %q\n%s", pheromoneLine, checklistOut)
	}
	handoffLine := lineContaining(checklistOut, "Previous Worker Handoffs")
	if handoffLine == "" || !strings.Contains(handoffLine, "present") {
		t.Errorf("Previous Worker Handoffs checklist row should read present (delivered via the capsule): %q\n%s", handoffLine, checklistOut)
	}

	// --full mode, scoped to a single worker with --worker: must also exit
	// clean, and that ONE worker's assembled output (capsule + brief +
	// skills, verbatim) must contain each owned heading EXACTLY once -- the
	// positive once-ness check the checklist's present/absent marker alone
	// cannot make (constraint 2's trap: absence-only checks would also pass
	// on zero). --full's own output covers every dispatch in one printout
	// (one composed section per worker, by design), so counting across the
	// unscoped output would find one occurrence per dispatch and not test
	// "once per worker" at all -- --worker scopes to exactly one dispatch's
	// own assembled context, matching what "each path receives each section
	// exactly once" actually means.
	names := workerNamesFromBanners(checklistOut)
	if len(names) == 0 {
		t.Fatalf("could not find any dispatch names in checklist output:\n%s", checklistOut)
	}
	target := names[0]
	fullOut, fullErr := runPrintBriefCmd(t, "--full", "--worker", target)
	if fullErr != "" {
		t.Fatalf("expected --print-brief --full --worker %s to pass cleanly with an active signal and a stored handoff, got stderr: %s", target, fullErr)
	}
	if n := strings.Count(fullOut, "## Pheromone Signals"); n != 1 {
		t.Errorf("expected exactly one \"## Pheromone Signals\" heading in %s's assembled context, found %d:\n%s", target, n, fullOut)
	}
	if n := strings.Count(fullOut, "## Previous Worker Handoffs"); n != 1 {
		t.Errorf("expected exactly one \"## Previous Worker Handoffs\" heading in %s's assembled context, found %d:\n%s", target, n, fullOut)
	}
	if !strings.Contains(fullOut, "sentinel-print-brief-duplication-probe") {
		t.Errorf("the active FOCUS signal's own text did not reach %s's assembled context:\n%s", target, fullOut)
	}
	if !strings.Contains(fullOut, "sentinel-print-brief-handoff-probe") {
		t.Errorf("the stored handoff's own summary text did not reach %s's assembled context:\n%s", target, fullOut)
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
