package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 196 plan 08 — the reachability proof.
//
// Every earlier plan in this phase passes its own tests. This file is the one
// that asks whether they are joined up, because that is the failure this
// repository actually keeps shipping: CLAUDE.md records four separate cases of
// work marked complete whose wiring was never added, and eighteen of
// twenty-five milestones have been framed around restoring something already
// declared done.
//
// So the assertions here deliberately never call the writer, the resolver or
// the renderer directly. They run a build and a check the way a person does —
// through the command tree — and then read the disk, the detail view and the
// ending screen. A test that calls writeSpendRowsForRun proves the writer
// works and proves nothing at all about whether a build calls it.
//
// The chain has four links, and cutting any one of them must fail this test:
//
//	the writer call        -> no rows on disk
//	the resolver call      -> rows carry no figure
//	the renderer call      -> no cost block on the ending screen
//	the command registration -> the detail view cannot be run
//
// Which failure each cut produces is recorded in 196-08-SUMMARY.md, verified
// by actually making the cuts rather than by reasoning about them.

// --- the raw provider event, and its total written out by hand ---
//
// The dispatcher below emits a provider result event on stdout exactly as a
// real one does, and the figure that reaches the ledger is produced by the
// production parser (codex.ParseUsage, via codex.AttachWorkerUsage) reading
// it. Nothing in this file computes a token count.
//
// The expected total is written as a literal, summed by hand from the four
// disjoint columns per Anthropic's own documented arithmetic. Calling
// BilledTotalTokens to produce the expectation would assert the code under
// test against itself, which is exactly how this subsystem shipped a 186x
// undercount once already.
const (
	e2eProviderInput         = 40_000
	e2eProviderCacheRead     = 800_000
	e2eProviderCacheCreation = 12_000
	e2eProviderOutput        = 5_000

	// 40,000 + 800,000 + 12,000 + 5,000
	e2eProviderBilledTotal int64 = 857_000

	// 857,000 truncated to its magnitude, the shape the ending screen uses.
	e2eProviderCompactFigure = "857K"
)

const e2eProviderResultEvent = `{"type":"result","usage":{"input_tokens":40000,` +
	`"cache_read_input_tokens":800000,"cache_creation_input_tokens":12000,` +
	`"output_tokens":5000},"total_cost_usd":12.34,"model":"test-model"}`

// spendReportingInvoker is a dispatcher whose workers finish normally and whose
// stdout carries a real provider usage event. It defers to the ordinary fake
// dispatcher for everything else, so task credit, claims and evidence behave
// exactly as they do on an unmeasured run and the only difference under test
// is that the provider reported a figure.
type spendReportingInvoker struct {
	fake codex.FakeInvoker
}

func (i *spendReportingInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result, err := i.fake.Invoke(ctx, config)
	if err != nil {
		return result, err
	}
	result.RawOutput = strings.TrimRight(result.RawOutput, "\n") + "\n" + e2eProviderResultEvent + "\n"
	result.Usage = codex.WorkerUsage{}
	return codex.AttachWorkerUsage(result, config), nil
}

func (i *spendReportingInvoker) IsAvailable(ctx context.Context) bool { return true }

func (i *spendReportingInvoker) ValidateAgent(path string) error { return nil }

func useSpendReportingInvoker(t *testing.T) {
	t.Helper()
	original := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &spendReportingInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = original })
}

// spendLedgerFilesForPhase lists the ledger files that exist on disk for a
// phase, read from the filesystem rather than from anything the runtime says
// it wrote.
func spendLedgerFilesForPhase(t *testing.T, dataDir string, phase int) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dataDir, "spend"))
	if err != nil {
		return nil
	}
	prefix := "phase-" + itoaForSpendTest(phase) + "-"
	var found []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			found = append(found, entry.Name())
		}
	}
	return found
}

// spendRowsOnDisk reads a phase's rows for one workflow straight off the disk.
func spendRowsOnDisk(t *testing.T, dataDir string, phase int, workflow string) []spendRow {
	t.Helper()
	return loadSpendLedgerFromDisk(t, dataDir, phase, workflow).Rows
}

// handSummedBilledTotal adds the four disjoint provider columns itself rather
// than calling BilledTotalTokens, so the expected total cannot inherit an
// arithmetic bug from the function that produced the printed one. It prefers a
// provider-stated aggregate for the same reason BilledTotalTokens does — a
// transcript that reports one total and no breakdown is still a measurement —
// but it never asks the code under test what that means.
func handSummedBilledTotal(rows []spendRow) int64 {
	var total int64
	for _, row := range rows {
		if !spendRowReportedUsage(row) {
			continue
		}
		if row.Usage.TotalTokens > 0 {
			total += row.Usage.TotalTokens
			continue
		}
		total += row.Usage.InputTokens + row.Usage.CachedInputTokens +
			row.Usage.CacheCreationTokens + row.Usage.OutputTokens
	}
	return total
}

// runSpendDetailView runs `aether spend --phase N` through the command tree,
// as a person would, and returns its machine envelope. Running it through
// rootCmd is what makes the command REGISTRATION part of the chain: cutting
// the AddCommand call makes this fail.
func runSpendDetailView(t *testing.T, phase int) map[string]interface{} {
	t.Helper()
	buf := &bytes.Buffer{}
	previous := stdout
	stdout = buf
	defer func() { stdout = previous }()

	resetRootCmd(t)
	rootCmd.SetArgs([]string{"spend", "--phase", itoaForSpendTest(phase)})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("aether spend returned an error: %v", err)
	}

	var envelope struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		t.Fatalf("aether spend printed nothing at all; the detail view is not reachable")
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("aether spend output was not the machine envelope (%v): %s", err, raw)
	}
	if !envelope.OK {
		t.Fatalf("aether spend reported failure: %s", raw)
	}
	return envelope.Result
}

func spendDetailWorkerNames(t *testing.T, result map[string]interface{}) map[string]bool {
	t.Helper()
	workers, ok := result["workers"].([]interface{})
	if !ok {
		t.Fatalf("the detail view carried no workers list: %#v", result)
	}
	names := map[string]bool{}
	for _, entry := range workers {
		worker, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := worker["name"].(string)
		if name != "" {
			names[name] = true
		}
	}
	return names
}

// countCostBlocks counts the cost blocks on an ending screen. Counting, rather
// than checking presence, is the whole point: a test that asserts the block
// appears cannot catch a second one appearing.
func countCostBlocks(screen string) int {
	return strings.Count(stripANSI(screen), spendCostLineHeading)
}

// TestSpendPipelineIsReachableEndToEnd is the phase's joined-up proof.
func TestSpendPipelineIsReachableEndToEnd(t *testing.T) {
	t.Run("a finished build files rows and the detail view shows them", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		forceBuildJSONOutput(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withWorkingDir(t, root)

		seedSpendPipelinePhase(t, dataDir)

		completionPath := runBuildToCompletionPacketForSpendTest(t, root)
		rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build-finalize returned error: %v", err)
		}

		rows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowBuild)
		if len(rows) == 0 {
			t.Fatalf("a finished build left no rows on disk; the writer is not called from the build path")
		}

		detail := runSpendDetailView(t, 1)
		shown := spendDetailWorkerNames(t, detail)
		for _, row := range rows {
			if !shown[row.AgentName] {
				t.Errorf("worker %q has a row on disk but the detail view does not show it: %#v", row.AgentName, detail)
			}
		}

		screen := appendSpendCostLine("ending screen\n", 1)
		if got := countCostBlocks(screen); got != 1 {
			t.Errorf("the ending screen carries %d cost block(s), want exactly 1:\n%s", got, screen)
		}
	})

	t.Run("the direct lane records what its own workers reported", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withTestWorkspace(t, root)
		withWorkingDir(t, root)
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		t.Setenv("AETHER_PLATFORM", "claude")
		useSpendReportingInvoker(t)

		seedSpendPipelinePhase(t, dataDir)

		screen := &bytes.Buffer{}
		stdout = screen
		rootCmd.SetArgs([]string{"build", "1"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("aether build 1 returned error: %v", err)
		}

		// LINK 1 — the writer call. A build that ran real workers must leave
		// rows, on this lane exactly as on the other.
		files := spendLedgerFilesForPhase(t, dataDir, 1)
		if len(files) == 0 {
			t.Fatalf("`aether build 1` ran its workers in-process and filed no token record at all; " +
				"the cost line on this lane can only ever say nothing was recorded")
		}
		rows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowBuild)
		if len(rows) == 0 {
			t.Fatalf("the direct build filed a ledger with no rows in it")
		}

		// LINK 2 — the resolver call. The provider reported a figure for
		// every worker, so every row must carry it.
		for _, row := range rows {
			if !spendRowReportedUsage(row) {
				t.Errorf("worker %s's tool reported a figure, but its row records none: %+v", row.AgentName, row)
				continue
			}
			if got := row.Usage.TotalTokens; got != e2eProviderBilledTotal {
				t.Errorf("worker %s's recorded total = %d, want the provider's own %d",
					row.AgentName, got, e2eProviderBilledTotal)
			}
		}

		// LINK 4 — the command registration. The detail view must show the
		// same workers and the same figures.
		detail := runSpendDetailView(t, 1)
		shown := spendDetailWorkerNames(t, detail)
		for _, row := range rows {
			if !shown[row.AgentName] {
				t.Errorf("worker %q is on disk but missing from the detail view", row.AgentName)
			}
		}
		if got, ok := detail["measured_tokens"].(float64); !ok || int64(got) != handSummedBilledTotal(rows) {
			t.Errorf("the detail view's measured total = %v, want the hand-summed disk total %d",
				detail["measured_tokens"], handSummedBilledTotal(rows))
		}

		// LINK 3 — the renderer call. Exactly one cost block ended the
		// screen, and its total agrees with the rows read back off disk.
		out := screen.String()
		if got := countCostBlocks(out); got != 1 {
			t.Fatalf("`aether build 1` ended with %d cost block(s), want exactly 1:\n%s", got, out)
		}
		want := handSummedBilledTotal(rows)
		if want != e2eProviderBilledTotal*int64(len(rows)) {
			t.Fatalf("the rows on disk sum to %d, want %d — the recorded figures are not the provider's",
				want, e2eProviderBilledTotal*int64(len(rows)))
		}
		if len(rows) == 1 && !strings.Contains(stripANSI(out), e2eProviderCompactFigure) {
			t.Errorf("the cost line does not print the recorded total %s:\n%s", e2eProviderCompactFigure, out)
		}
	})

	t.Run("the check files its own rows beside the build's", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		forceBuildJSONOutput(t)

		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		withTestWorkspace(t, root)
		withWorkingDir(t, root)

		seedSpendPipelinePhase(t, dataDir)

		completionPath := runBuildToCompletionPacketForSpendTest(t, root)
		rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("build-finalize returned error: %v", err)
		}
		buildRows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowBuild)

		planAndFinalizeContinueForSpendTest(t, root, codexContinueOptions{HeavyFlag: true})

		checkRows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowContinue)
		if len(checkRows) == 0 {
			t.Fatalf("the finished check filed no rows of its own")
		}
		afterBuildRows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowBuild)
		if len(afterBuildRows) != len(buildRows) {
			t.Errorf("the check changed the build's rows: %d before, %d after",
				len(buildRows), len(afterBuildRows))
		}

		screen := appendSpendCostLine("ending screen\n", 1)
		if got := countCostBlocks(screen); got != 1 {
			t.Errorf("the check's ending screen carries %d cost block(s), want exactly 1:\n%s", got, screen)
		}
		for _, row := range append(append([]spendRow{}, afterBuildRows...), checkRows...) {
			if !strings.Contains(stripANSI(screen), row.AgentName) {
				t.Errorf("worker %q has a row on disk but is missing from the one cost block:\n%s",
					row.AgentName, screen)
			}
		}
	})
}

// seedSpendPipelinePhase gives the fixture a single buildable phase.
func seedSpendPipelinePhase(t *testing.T, dataDir string) {
	t.Helper()
	goal := "Leave an honest record of what the work cost"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{{
				ID:          1,
				Name:        "See what it cost",
				Description: "A finished run files one row per worker",
				Status:      colony.PhaseReady,
				Tasks:       []colony.Task{{ID: &taskID, Goal: "Create evidence", Status: colony.TaskPending}},
			}},
		},
	})
}

// TestNothingThisPhaseAddedIsUncalled is the anti-orphan check for the symbols
// this phase introduced.
//
// The orphan ratchet in cmd/subcommand_reachability_ratchet_test.go covers
// registered COMMANDS. It cannot see an internal entry point that exists, has
// tests, and is invoked by nothing — which is precisely the shape the salvage
// assessment refused to merge, and precisely what wrapperUsageRequest.Attached
// was until this plan: a designed input with no producer anywhere in the
// production tree, so the whole directly-spawned measurement path was dead.
func TestNothingThisPhaseAddedIsUncalled(t *testing.T) {
	sources := goSourceFilesForSpendReachability(t)

	// Each entry names a symbol this phase added and the production files that
	// must call it. Test files never count as callers: a symbol whose only
	// caller is its own test is the definition of an orphan here.
	for _, check := range []struct {
		symbol string
		why    string
	}{
		{"writeSpendRowsForRun", "nothing would ever file a token row"},
		{"resolveWrapperWorkerUsage", "no platform's reported figures would ever be read"},
		{"renderSpendCostLine", "no ending screen would carry the cost block"},
		{"appendSpendCostLine", "the cost block would be built and never shown"},
		{"parseClaudeTranscriptUsage", "Claude Code sessions would report nothing"},
		{"openCodeSessionUsageForRunOrNone", "OpenCode sessions would report nothing"},
		{"spendDispatchesFromContinueFlow", "the checking pass would file nothing"},
		{"loadSpendLedgersForPhase", "nothing would read the rows back"},
		{"computeSpendTotals", "no total would ever be computed from the rows"},
	} {
		if callers := productionCallersOf(sources, check.symbol); len(callers) == 0 {
			t.Errorf("%s is called from no production file — %s", check.symbol, check.why)
		}
	}

	// Attached is a struct FIELD, not a function, so a caller search on a name
	// would not find it. It is checked by name because it is the one input on
	// the resolver that had no producer at all.
	if callers := productionCallersOf(sources, "Attached:"); len(callers) == 0 {
		t.Errorf("wrapperUsageRequest.Attached is populated by no production file, " +
			"so every provider measurement taken at the dispatch boundary is discarded before it reaches the ledger")
	}
}

// goSourceFilesForSpendReachability returns every non-test Go file under cmd/.
func goSourceFilesForSpendReachability(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Dir(goldenTestdataDir())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	sources := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		sources[name] = string(raw)
	}
	if len(sources) == 0 {
		t.Fatalf("no production Go files found under %s", dir)
	}
	return sources
}

// productionCallersOf names the production files that mention symbol outside
// of the line that declares it and outside comment lines.
func productionCallersOf(sources map[string]string, symbol string) []string {
	var callers []string
	for name, body := range sources {
		for _, line := range strings.Split(body, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if strings.HasPrefix(trimmed, "func "+symbol) || strings.HasPrefix(trimmed, "func (") &&
				strings.Contains(trimmed, ") "+symbol+"(") {
				continue
			}
			if !strings.Contains(trimmed, symbol) {
				continue
			}
			callers = append(callers, name)
			break
		}
	}
	return callers
}
