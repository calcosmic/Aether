package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
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
// as a person would, and returns whatever it printed. Running it through
// rootCmd is what makes the command REGISTRATION part of the chain: cutting
// the AddCommand call makes this fail.
func runSpendDetailView(t *testing.T, phase int) string {
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
	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		t.Fatalf("aether spend printed nothing at all; the detail view is not reachable")
	}
	return raw
}

// spendDetailEnvelope decodes the machine surface of the detail view.
func spendDetailEnvelope(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var envelope struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
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

		detail := spendDetailEnvelope(t, runSpendDetailView(t, 1))
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
		// same workers and the same figures. This lane runs in the owner's
		// own reading mode, so the assertion is on the rendered view: the
		// exact recorded figure, unabbreviated, beside each worker's name.
		detail := stripANSI(runSpendDetailView(t, 1))
		for _, row := range rows {
			if !strings.Contains(detail, row.AgentName) {
				t.Errorf("worker %q is on disk but missing from the detail view:\n%s", row.AgentName, detail)
			}
		}
		if want := itoaForSpendTest(int(handSummedBilledTotal(rows))); !strings.Contains(detail, want) {
			t.Errorf("the detail view does not show the hand-summed disk total %s:\n%s", want, detail)
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

	t.Run("the direct check records what its own reviewers reported", func(t *testing.T) {
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

		screen.Reset()
		resetRootCmd(t)
		rootCmd.SetArgs([]string{"continue", "--heavy"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("aether continue returned error: %v", err)
		}

		// Whether the check advanced or blocked, it spent what it spent. The
		// build's rows must still be there, untouched, beside its own.
		checkRows := spendRowsOnDisk(t, dataDir, 1, spendWorkflowContinue)
		if len(checkRows) == 0 {
			t.Fatalf("`aether continue` ran its reviewers in-process and filed no token record at all")
		}
		measured := 0
		for _, row := range checkRows {
			if spendRowReportedUsage(row) {
				measured++
				if got := row.Usage.TotalTokens; got != e2eProviderBilledTotal {
					t.Errorf("checker %s's recorded total = %d, want the provider's own %d",
						row.AgentName, got, e2eProviderBilledTotal)
				}
			}
		}
		if measured == 0 {
			t.Errorf("every checker's tool reported a figure, but no row on disk records one: %+v", checkRows)
		}
		if len(spendRowsOnDisk(t, dataDir, 1, spendWorkflowBuild)) == 0 {
			t.Errorf("the check erased the build's rows")
		}

		out := stripANSI(screen.String())
		if got := countCostBlocks(out); got != 1 {
			t.Errorf("`aether continue` ended with %d cost block(s), want exactly 1:\n%s", got, out)
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
// was until plan 196-08: a designed input with no producer anywhere in the
// production tree, so the whole directly-spawned measurement path was dead.
//
// The FIRST version of this check enumerated nine symbol names by hand. It
// passed for the whole phase while spendPerWorkerAverageTokens,
// spendRollupByParent and spendParentRollup sat in cmd/spend_ledger.go with no
// production caller at all: the hand-written list omitted exactly the three
// orphans the guard existed to catch. A curated inventory is worse than no
// guard, because it reads as a proof.
//
// So nothing is enumerated here any more. The file set is derived from disk by
// name (spendPhaseProductionFileNames), every top-level symbol in those files
// is derived from the syntax tree, every struct field in them likewise, and
// "called" is computed as a reachability fixpoint over the whole production
// package rather than as a text search. Add a function, a type, a const or a
// field to any of this phase's files without a caller and this test names it
// the day it lands, whatever it is called. The planted-orphan subtest below
// proves that claim rather than asserting it.
func TestNothingThisPhaseAddedIsUncalled(t *testing.T) {
	sources := goSourceFilesForSpendReachability(t)
	report := analyzeSpendPhaseOrphans(t, sources)

	// Anti-vacuity: a guard that derives its own inventory must fail loudly if
	// the derivation finds nothing, otherwise deleting the subsystem would make
	// it pass. These floors are deliberately well below the real figures.
	if len(report.PhaseFiles) < 8 {
		t.Fatalf("derived only %d of this phase's production files (%v) — the derivation is broken or the subsystem was gutted",
			len(report.PhaseFiles), report.PhaseFiles)
	}
	if len(report.Symbols) < 60 {
		t.Fatalf("derived only %d top-level symbols from %v — the derivation is broken",
			len(report.Symbols), report.PhaseFiles)
	}
	if len(report.Fields) < 40 {
		t.Fatalf("derived only %d struct fields from %v — the derivation is broken",
			len(report.Fields), report.PhaseFiles)
	}

	// Anti-vacuity, second half: the derivation must actually be looking at the
	// spend subsystem. This is NOT the inventory — the inventory is derived
	// above — it is a floor that fails if the load-bearing wiring is renamed
	// away or deleted, which is what the hand-written list was genuinely good
	// for. Each entry says what stops working if the symbol vanishes.
	for _, required := range []struct {
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
		if _, declared := report.Symbols[required.symbol]; !declared {
			t.Errorf("%s is no longer declared in any of this phase's files — %s", required.symbol, required.why)
		}
	}

	for _, orphan := range report.UncalledSymbols {
		t.Errorf("%s is declared by this phase and called from no production file — "+
			"a symbol whose only caller is its own test is an orphan (CLAUDE.md names one as this repository's signature failure)",
			orphan)
	}
	for _, field := range report.UnusedFields {
		t.Errorf("%s is declared by this phase and no production file reads or writes it — "+
			"a field on a record production never fills is a schema lie (this is what spendRow.ParentName and spendRow.ToolCount were)",
			field)
	}
}

// TestTheAntiOrphanGuardCatchesAPlantedOrphan proves the guard above does what
// its name says instead of trusting it, in the style of the planted-violation
// subtest inside TestNoTokenCountIsDerivedFromLength.
//
// It takes the real production sources, adds ONE synthetic file that obeys this
// phase's own file-naming rule, and asserts the analysis names the uncalled
// function and the unread field in it — and does not name the field that IS
// read. Because the analysis is a pure function of a source map, the plant
// never touches the working tree.
func TestTheAntiOrphanGuardCatchesAPlantedOrphan(t *testing.T) {
	sources := goSourceFilesForSpendReachability(t)

	clean := analyzeSpendPhaseOrphans(t, sources)
	if len(clean.UncalledSymbols) != 0 || len(clean.UnusedFields) != 0 {
		t.Fatalf("the unplanted tree already reports orphans (%v / %v) — fix those before trusting this subtest",
			clean.UncalledSymbols, clean.UnusedFields)
	}

	planted := map[string]string{}
	for name, body := range sources {
		planted[name] = body
	}
	planted["spend_planted_orphan.go"] = `package cmd

type spendPlantedRecord struct {
	Read   string
	Unread string
}

func spendPlantedOrphanNobodyCalls(record spendPlantedRecord) string {
	return record.Read
}
`

	report := analyzeSpendPhaseOrphans(t, planted)
	if !spendReportNames(report.UncalledSymbols, "spendPlantedOrphanNobodyCalls") {
		t.Errorf("the guard did not name the planted uncalled function; it reported %v", report.UncalledSymbols)
	}
	if !spendReportNames(report.UnusedFields, "spendPlantedRecord.Unread") {
		t.Errorf("the guard did not name the planted unread field; it reported %v", report.UnusedFields)
	}
	if spendReportNames(report.UnusedFields, "spendPlantedRecord.Read") {
		t.Errorf("the guard named a field that IS read (%v) — it would fail on honest code", report.UnusedFields)
	}
}

// spendReportNames reports whether any entry in the report starts with name.
// Entries carry their file in parentheses, so an exact match would not do.
func spendReportNames(entries []string, name string) bool {
	for _, entry := range entries {
		if entry == name || strings.HasPrefix(entry, name+" ") {
			return true
		}
	}
	return false
}

// spendPhaseFileNamePrefixes and spendPhaseExtraFileNames define which
// production files belong to Phase 196, BY NAME, so the set is derived from
// disk rather than listed. A file added to this subsystem under either prefix
// is audited from the moment it exists.
var (
	spendPhaseFileNamePrefixes = []string{"spend_", "wrapper_usage_"}
	spendPhaseExtraFileNames   = []string{"caste_model_reason.go"}
)

// isSpendPhaseProductionFile reports whether a cmd/ production file name is one
// of this phase's own.
func isSpendPhaseProductionFile(name string) bool {
	for _, prefix := range spendPhaseFileNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, extra := range spendPhaseExtraFileNames {
		if name == extra {
			return true
		}
	}
	return false
}

// spendPhaseOrphanReport is what analyzeSpendPhaseOrphans derives.
//
// Symbols and Fields are the DERIVED inventory (symbol or Type.Field -> the
// file declaring it). UncalledSymbols and UnusedFields are the subset with no
// production user, each rendered as "name (file)".
type spendPhaseOrphanReport struct {
	PhaseFiles      []string
	Symbols         map[string]string
	Fields          map[string]string
	UncalledSymbols []string
	UnusedFields    []string
}

// analyzeSpendPhaseOrphans derives this phase's symbol inventory from the
// supplied production sources and works out which of it nothing uses.
//
// Reachability, not mention-counting. A phase symbol is reachable when some
// declaration OUTSIDE this phase's symbol set refers to it, or when a phase
// symbol that is itself already reachable refers to it — computed to a
// fixpoint. Mention-counting would call a symbol "used" merely because another
// orphan referred to it, which is how spendParentRollup would have escaped:
// its only reference was inside spendRollupByParent, an orphan itself.
//
// A field is used when a composite literal of its own struct type names it as a
// key anywhere in production, or when any of this phase's own files refers to
// it as a selector. The selector half is matched on field NAME within this
// phase's files only: a package-private struct declared here is constructed and
// read here, and scanning the whole package by bare name would let an unrelated
// type's identically-named field vouch for a dead one — which is exactly how
// spendRow.ToolCount would have passed (cmd/ceremony_emitter.go assigns a
// ToolCount of its own).
//
// That narrowing has TWO failure directions and only one of them is loud, so
// both are named here rather than just the flattering one:
//
//   - FALSE POSITIVE (loud, self-correcting): a field of one of these structs
//     read exclusively from a file outside this phase's set is reported as
//     dead. Whoever hits it sees the name and widens the set. No field is
//     currently in that position — of the fields derived today, none is
//     justified by an external selector alone.
//
//   - FALSE NEGATIVE (SILENT, and the direction that bit this phase): the
//     selector match is on the BARE field name across this phase's files, so a
//     dead field whose name collides with a live field on a DIFFERENT type in
//     those same files escapes unnoticed. Measured during Phase 196
//     verification: adding a never-written, never-read `Notes []string` to
//     spendRow passes clean, because spendWriteOutcome.Notes is selected in
//     spend_writer.go. This is the same species as the package-scope ToolCount
//     hole above, narrowed from the whole cmd package to nine files — smaller,
//     not closed. Closing it needs type-resolved selector matching (go/types),
//     which is why it was not done here.
//
// init functions are entry points the Go runtime calls, so they are roots
// rather than inventory. Blank identifiers are skipped.
func analyzeSpendPhaseOrphans(t *testing.T, sources map[string]string) spendPhaseOrphanReport {
	t.Helper()

	fset := token.NewFileSet()
	parsed := map[string]*ast.File{}
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		file, err := parser.ParseFile(fset, name, sources[name], 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		parsed[name] = file
	}

	report := spendPhaseOrphanReport{
		Symbols: map[string]string{},
		Fields:  map[string]string{},
	}
	for _, name := range names {
		if isSpendPhaseProductionFile(name) {
			report.PhaseFiles = append(report.PhaseFiles, name)
		}
	}

	// One entry per top-level declaration: what it declares, and every
	// identifier and selector name appearing inside it.
	type declUses struct {
		file     string
		declares []string
		uses     map[string]bool
	}
	var decls []declUses

	for _, name := range names {
		phaseFile := isSpendPhaseProductionFile(name)
		for _, decl := range parsed[name].Decls {
			declared := spendDeclaredNames(decl)
			uses := map[string]bool{}
			ast.Inspect(decl, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.Ident:
					uses[typed.Name] = true
				case *ast.SelectorExpr:
					uses[typed.Sel.Name] = true
				}
				return true
			})
			for _, own := range declared {
				delete(uses, own)
			}
			decls = append(decls, declUses{file: name, declares: declared, uses: uses})
			if !phaseFile {
				continue
			}
			for _, own := range declared {
				if own == "init" || own == "_" {
					continue
				}
				report.Symbols[own] = name
			}
			for typeName, fields := range spendDeclaredStructFields(decl) {
				for _, field := range fields {
					report.Fields[typeName+"."+field] = name
				}
			}
		}
	}

	reachable := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, decl := range decls {
			ownedByPhase, ownerReachable := false, false
			if isSpendPhaseProductionFile(decl.file) {
				for _, own := range decl.declares {
					if _, isPhaseSymbol := report.Symbols[own]; isPhaseSymbol {
						ownedByPhase = true
						if reachable[own] {
							ownerReachable = true
						}
					}
				}
			}
			if ownedByPhase && !ownerReachable {
				continue
			}
			for used := range decl.uses {
				if _, isPhaseSymbol := report.Symbols[used]; !isPhaseSymbol {
					continue
				}
				if !reachable[used] {
					reachable[used] = true
					changed = true
				}
			}
		}
	}

	literalKeys := map[string]bool{}
	selectorNames := map[string]bool{}
	for _, name := range names {
		phaseFile := isSpendPhaseProductionFile(name)
		ast.Inspect(parsed[name], func(node ast.Node) bool {
			if lit, ok := node.(*ast.CompositeLit); ok {
				spendCollectCompositeKeys(lit, "", literalKeys)
			}
			if phaseFile {
				if sel, ok := node.(*ast.SelectorExpr); ok {
					selectorNames[sel.Sel.Name] = true
				}
			}
			return true
		})
	}

	for symbol, file := range report.Symbols {
		if !reachable[symbol] {
			report.UncalledSymbols = append(report.UncalledSymbols, symbol+" ("+file+")")
		}
	}
	for field, file := range report.Fields {
		bare := field[strings.Index(field, ".")+1:]
		if literalKeys[field] || selectorNames[bare] {
			continue
		}
		report.UnusedFields = append(report.UnusedFields, field+" ("+file+")")
	}
	sort.Strings(report.UncalledSymbols)
	sort.Strings(report.UnusedFields)
	return report
}

// spendDeclaredNames returns every name a top-level declaration introduces:
// the function name (methods included, keyed by method name, which is how a
// call site refers to them), and every type, const and var name in a
// declaration group.
func spendDeclaredNames(decl ast.Decl) []string {
	var names []string
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		if typed.Name != nil {
			names = append(names, typed.Name.Name)
		}
	case *ast.GenDecl:
		for _, spec := range typed.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, spec.Name.Name)
			case *ast.ValueSpec:
				for _, ident := range spec.Names {
					names = append(names, ident.Name)
				}
			}
		}
	}
	return names
}

// spendDeclaredStructFields returns the named fields of every struct type a
// declaration introduces, keyed by type name. Embedded fields carry no name of
// their own and are skipped.
func spendDeclaredStructFields(decl ast.Decl) map[string][]string {
	fields := map[string][]string{}
	gen, ok := decl.(*ast.GenDecl)
	if !ok {
		return fields
	}
	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok || structType.Fields == nil {
			continue
		}
		for _, field := range structType.Fields.List {
			for _, name := range field.Names {
				fields[typeSpec.Name.Name] = append(fields[typeSpec.Name.Name], name.Name)
			}
		}
	}
	return fields
}

// spendCollectCompositeKeys records every "Type.Field" a composite literal
// fills in. inherited carries the element type down into the untyped inner
// literals of []T{{...}} and map[K]T{k: {...}}, which is how the writer's rows
// are actually written.
func spendCollectCompositeKeys(lit *ast.CompositeLit, inherited string, into map[string]bool) {
	typeName := inherited
	elemName := ""
	switch litType := lit.Type.(type) {
	case *ast.Ident:
		typeName = litType.Name
	case *ast.ArrayType:
		typeName = ""
		elemName = spendTypeIdentName(litType.Elt)
	case *ast.MapType:
		typeName = ""
		elemName = spendTypeIdentName(litType.Value)
	case nil:
		// Untyped inner literal: keep the type inherited from the parent.
	default:
		typeName = ""
	}
	for _, element := range lit.Elts {
		if kv, ok := element.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok && typeName != "" {
				into[typeName+"."+ident.Name] = true
			}
			if inner, ok := kv.Value.(*ast.CompositeLit); ok {
				spendCollectCompositeKeys(inner, elemName, into)
			}
			continue
		}
		if inner, ok := element.(*ast.CompositeLit); ok {
			spendCollectCompositeKeys(inner, elemName, into)
		}
	}
}

// spendTypeIdentName returns the bare type name of an element type expression,
// looking through pointers, or "" when it is not a plain named type.
func spendTypeIdentName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return spendTypeIdentName(typed.X)
	}
	return ""
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
