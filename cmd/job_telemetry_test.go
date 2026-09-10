package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------
// Task 1: the eight-segment record with an honest unmeasured sentinel.
// ---------------------------------------------------------------------

func TestUnmeasuredSegmentIsNeverDerived(t *testing.T) {
	t.Run("an unmeasured segment reads as unmeasured, never as zero", func(t *testing.T) {
		capture := newJobTelemetryCapture()
		capture.measure(jobTelemetrySegmentContext, 2*time.Second, "brief assembly")
		capture.measure(jobTelemetrySegmentWork, 10*time.Second, "worker execution")
		capture.markUnmeasured(jobTelemetrySegmentQueue, "no queue boundary recorded in this fixture")

		record := newJobTelemetryRecord("attempt-x", "job-x", capture, time.Now())

		if record.Queue.Measured {
			t.Fatal("expected Queue to be recorded as unmeasured")
		}
		if record.Queue.Duration != 0 {
			t.Fatalf("expected Queue's untouched Duration field to be zero, got %v", record.Queue.Duration)
		}
		if got := record.Queue.RenderedDuration(); got != jobTelemetryUnmeasuredFigure {
			t.Fatalf("RenderedDuration() on an unmeasured segment = %q, want %q", got, jobTelemetryUnmeasuredFigure)
		}
		if record.Queue.Source == "" {
			t.Fatal("expected the unmeasured segment to still carry its stated reason")
		}

		// A segment the capture was never told about at all (Model,
		// ToolCall, Verification, Wait, Preflight here) must read
		// identically to one explicitly marked unmeasured -- an untouched
		// segment and a reasoned-unmeasured one both mean "no measurement",
		// and both must be indistinguishable to RenderedDuration.
		for _, name := range []string{
			jobTelemetrySegmentPreflight,
			jobTelemetrySegmentModel,
			jobTelemetrySegmentToolCall,
			jobTelemetrySegmentVerification,
			jobTelemetrySegmentWait,
		} {
			for _, named := range record.namedSegments() {
				if named.Name != name {
					continue
				}
				if named.Segment.Measured {
					t.Fatalf("segment %q was never captured but reads as measured", name)
				}
				if got := named.Segment.RenderedDuration(); got != jobTelemetryUnmeasuredFigure {
					t.Fatalf("segment %q RenderedDuration() = %q, want %q", name, got, jobTelemetryUnmeasuredFigure)
				}
			}
		}

		// The largest measured segment must be Work (10s), never Queue --
		// proving an unmeasured segment's zero-value Duration never enters
		// a comparison against real measurements.
		name, seg, ok := record.largestMeasuredSegment()
		if !ok || name != jobTelemetrySegmentWork || seg.Duration != 10*time.Second {
			t.Fatalf("largestMeasuredSegment() = (%q, %+v, %v), want (work, 10s, true)", name, seg, ok)
		}
	})

	t.Run("no function in this file computes a segment from the total or from other segments", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "job_telemetry.go", nil, 0)
		if err != nil {
			t.Fatalf("parse cmd/job_telemetry.go: %v", err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			be, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			if be.Op == token.SUB {
				t.Fatalf("found a subtraction at %s -- a timing segment must never be derived as total minus the other segments", fset.Position(be.Pos()))
			}
			return true
		})
	})
}

func TestSegmentRequiresAnInstrumentationSource(t *testing.T) {
	if _, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, ""); err == nil {
		t.Fatal("expected a measured segment with no source to be refused")
	} else if !strings.Contains(err.Error(), jobTelemetrySegmentWork) {
		t.Fatalf("refusal error %q does not name the offending segment", err.Error())
	}
	if _, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, "   "); err == nil {
		t.Fatal("expected a measured segment with a whitespace-only source to be refused")
	}
	if seg, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, "worker execution"); err != nil {
		t.Fatalf("unexpected refusal of a valid measured segment: %v", err)
	} else if !seg.Measured || seg.Duration != 5*time.Second || seg.Source != "worker execution" {
		t.Fatalf("unexpected segment shape: %+v", seg)
	}

	if _, err := unmeasuredJobTelemetrySegment(jobTelemetrySegmentPreflight, ""); err == nil {
		t.Fatal("expected an unmeasured segment with no reason to be refused")
	} else if !strings.Contains(err.Error(), jobTelemetrySegmentPreflight) {
		t.Fatalf("refusal error %q does not name the offending segment", err.Error())
	}
	if seg, err := unmeasuredJobTelemetrySegment(jobTelemetrySegmentPreflight, "no first-response boundary is observable today"); err != nil {
		t.Fatalf("unexpected refusal of a valid unmeasured segment: %v", err)
	} else if seg.Measured || seg.Source == "" {
		t.Fatalf("unexpected segment shape: %+v", seg)
	}

	// jobTelemetryCapture.measure/markUnmeasured must swallow a refused
	// segment rather than ever panicking or corrupting the record --
	// instrumentation is never allowed to fail a build (this plan's own
	// must_have).
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, 5*time.Second, "")
	capture.markUnmeasured(jobTelemetrySegmentWait, "")
	record := newJobTelemetryRecord("attempt-refused", "job-refused", capture, time.Now())
	if record.Work.Measured {
		t.Fatal("a refused measured segment must not silently become measured")
	}
	if record.Wait.Measured {
		t.Fatal("a refused unmeasured-mark must not silently become measured")
	}
}

func TestAllUnmeasuredRecordWritesNoFile(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	capture := newJobTelemetryCapture()
	capture.markUnmeasured(jobTelemetrySegmentQueue, "no boundary recorded")
	capture.markUnmeasured(jobTelemetrySegmentContext, "no boundary recorded")
	record := newJobTelemetryRecord("attempt-all-unmeasured", "job-all-unmeasured", capture, time.Now())

	if record.hasAnyMeasuredSegment() {
		t.Fatal("test fixture is broken: expected an all-unmeasured record")
	}
	if err := writeJobTelemetryRecord(record); err != nil {
		t.Fatalf("writing an all-unmeasured record must not itself error, got: %v", err)
	}

	if _, ok := readJobTelemetryRecord("attempt-all-unmeasured"); ok {
		t.Fatal("an all-unmeasured record was written to disk, want no file at all")
	}
}

func TestTelemetryStorageFailureIsNonFatal(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, 3*time.Second, "worker execution")

	// A path-traversal attempt id is refused by attemptBoundArtifactPath's
	// own containment validation -- writeJobTelemetryRecord must surface
	// that as a returned error, never a panic, and never corrupt anything
	// on disk.
	record := newJobTelemetryRecord("../escape", "job-escape", capture, time.Now())
	if err := writeJobTelemetryRecord(record); err == nil {
		t.Fatal("expected a storage failure for a path-traversal attempt id")
	}

	// The caller's own result is unaffected: reading back the (never
	// written) record still reports nothing, and a second, legitimate
	// write for a different attempt still succeeds -- proving the failed
	// write left no dangling state behind that could affect anyone else.
	if _, ok := readJobTelemetryRecord("../escape"); ok {
		t.Fatal("a failed write must not be readable afterward")
	}
	goodRecord := newJobTelemetryRecord("attempt-after-failure", "job-after-failure", capture, time.Now())
	if err := writeJobTelemetryRecord(goodRecord); err != nil {
		t.Fatalf("a legitimate write after an induced storage failure must still succeed: %v", err)
	}
	if _, ok := readJobTelemetryRecord("attempt-after-failure"); !ok {
		t.Fatal("expected the legitimate write to be readable")
	}
}

// TestJobTelemetryJSONRoundTrip is the plan's declared behavior "A record
// round-trips through JSON preserving measured and unmeasured segments
// distinctly", verified directly rather than folded into another test.
func TestJobTelemetryJSONRoundTrip(t *testing.T) {
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentContext, 1500*time.Millisecond, "brief assembly")
	capture.measure(jobTelemetrySegmentVerification, 42*time.Second, "deterministic floor run")
	capture.markUnmeasured(jobTelemetrySegmentModel, "the platform reports total tokens per worker but no call duration")
	record := newJobTelemetryRecord("attempt-roundtrip", "job-roundtrip", capture, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded jobTelemetryRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(record, decoded) {
		t.Fatalf("round trip mismatch:\n  before: %+v\n  after:  %+v", record, decoded)
	}
	if decoded.Context.Measured != true || decoded.Context.Duration != 1500*time.Millisecond {
		t.Fatalf("measured segment lost precision across the round trip: %+v", decoded.Context)
	}
	if decoded.Model.Measured != false || decoded.Model.Source == "" {
		t.Fatalf("unmeasured segment lost its reason across the round trip: %+v", decoded.Model)
	}
}

// TestJobTelemetryKeyedByAttemptAndJob is the plan's declared behavior "The
// record is keyed by the attempt identifier and the job name."
func TestJobTelemetryKeyedByAttemptAndJob(t *testing.T) {
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, time.Second, "worker execution")
	record := newJobTelemetryRecord("  attempt-keyed  ", "  job-keyed  ", capture, time.Now())
	if record.AttemptID != "attempt-keyed" || record.JobName != "job-keyed" {
		t.Fatalf("expected trimmed attempt/job identity, got AttemptID=%q JobName=%q", record.AttemptID, record.JobName)
	}
}

// ---------------------------------------------------------------------
// Task 2: capturing the segments at real instrumentation points.
// ---------------------------------------------------------------------

// oneJobTelemetryFixturePhase builds a single-task, one-job colony state
// fixture ready for a real runCodexBuild call, writing it via
// createTestColonyState.
func oneJobTelemetryFixturePhase(t *testing.T, dataDir, taskID, hintFile string) {
	t.Helper()
	goal := "One job produces a real timing record"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "light", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "One job phase", Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskID, Goal: "Add the telemetry fixture helper", Hints: []string{hintFile}, Status: colony.TaskPending},
			},
		}}},
	})
}

// TestRealBuildProducesMeasuredSegments drives a real one-job build,
// followed by a real check, against an isolated fixture, and asserts the
// segments this plan's own architecture can genuinely observe are measured
// with a named source, and every segment it cannot observe is unmeasured
// with a stated reason.
func TestRealBuildProducesMeasuredSegments(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	taskID := "1.1"
	oneJobTelemetryFixturePhase(t, dataDir, taskID, "cmd/job_telemetry_fixture_a.go")

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return workIdentitySuccessInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}

	_, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatalf("no build attempt was recorded for phase 1")
	}
	if len(attempt.Dispatches) != 1 {
		t.Fatalf("fixture is broken: expected exactly one dispatch, got %d", len(attempt.Dispatches))
	}

	if _, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{}); err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}

	record, ok := readJobTelemetryRecord(attempt.ID)
	if !ok {
		t.Fatalf("expected a job telemetry record for attempt %s, found none", attempt.ID)
	}
	if record.AttemptID != attempt.ID {
		t.Fatalf("record attempt id = %q, want %q", record.AttemptID, attempt.ID)
	}

	required := map[string]bool{
		jobTelemetrySegmentQueue:        true,
		jobTelemetrySegmentContext:      true,
		jobTelemetrySegmentWork:         true,
		jobTelemetrySegmentVerification: true,
	}
	for _, named := range record.namedSegments() {
		if required[named.Name] {
			if !named.Segment.Measured {
				t.Fatalf("expected segment %q to be measured on a real one-job build+check, got %+v", named.Name, named.Segment)
			}
			if strings.TrimSpace(named.Segment.Source) == "" {
				t.Fatalf("segment %q is measured but names no instrumentation source", named.Name)
			}
			continue
		}
		if named.Segment.Measured {
			// A segment this test does not require may still legitimately
			// be measured; nothing more to assert about it.
			continue
		}
		if strings.TrimSpace(named.Segment.Source) == "" {
			t.Fatalf("segment %q is unmeasured but carries no stated reason", named.Name)
		}
	}
}

// TestInstrumentationAddsNoDispatchOrPause proves the segment capture added
// in this plan never itself dispatches a worker or creates an owner-pause
// boundary: the same one-job fixture produces exactly one dispatch across
// both build and check, on the same attempt identifier throughout.
func TestInstrumentationAddsNoDispatchOrPause(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	taskID := "1.1"
	oneJobTelemetryFixturePhase(t, dataDir, taskID, "cmd/job_telemetry_fixture_b.go")

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return workIdentitySuccessInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	if _, err := runCodexBuild(root, 1, nil, false); err != nil {
		t.Fatalf("runCodexBuild returned error: %v", err)
	}
	_, attemptAfterBuild, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatalf("no build attempt was recorded for phase 1")
	}
	if len(attemptAfterBuild.Dispatches) != 1 {
		t.Fatalf("expected exactly one dispatch for this one-task fixture; instrumentation must never add a phantom dispatch, got %d: %#v", len(attemptAfterBuild.Dispatches), attemptAfterBuild.Dispatches)
	}

	if _, _, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{}); err != nil {
		t.Fatalf("runCodexContinue returned error: %v", err)
	}

	_, attemptAfterContinue, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatalf("no build attempt found after continue")
	}
	if attemptAfterContinue.ID != attemptAfterBuild.ID {
		t.Fatalf("continue created a new build attempt (%s -> %s); instrumentation must never trigger a fresh dispatch cycle", attemptAfterBuild.ID, attemptAfterContinue.ID)
	}
	if len(attemptAfterContinue.Dispatches) != 1 {
		t.Fatalf("expected exactly one dispatch after continue, got %d -- instrumentation must never add a dispatch", len(attemptAfterContinue.Dispatches))
	}

	record, ok := readJobTelemetryRecord(attemptAfterContinue.ID)
	if !ok {
		t.Fatal("expected a job telemetry record to exist after the run")
	}
	if !record.Work.Measured || !record.Verification.Measured {
		t.Fatalf("expected both work and verification segments measured across build+continue, got Work=%+v Verification=%+v", record.Work, record.Verification)
	}
}

// ---------------------------------------------------------------------
// Task 3: one timing line, a drill-down, and a report-only guard.
// ---------------------------------------------------------------------

// TestCloseoutRendersOneTimingLine proves the closeout's "Elapsed: " line
// (cmd/spend_cost_line.go) names the largest measured segment when a
// telemetry record exists, and plainly says the breakdown was not measured
// when it does not -- in both cases, exactly one block and one timing line.
func TestCloseoutRendersOneTimingLine(t *testing.T) {
	t.Run("names the largest measured segment", func(t *testing.T) {
		setupSpendTestStore(t)
		seedSpendLedgerForTest(t, 197, spendWorkflowBuild,
			measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		)
		seedSpendElapsedAttemptForTest(t, 197, "attempt-timing-line", "2026-01-01T00:00:00Z", "2026-01-01T00:10:00Z")

		capture := newJobTelemetryCapture()
		capture.measure(jobTelemetrySegmentWork, 7*time.Minute, "worker execution")
		capture.measure(jobTelemetrySegmentVerification, 2*time.Minute, "deterministic floor run")
		record := newJobTelemetryRecord("attempt-timing-line", "job-x", capture, time.Now())
		if err := writeJobTelemetryRecord(record); err != nil {
			t.Fatalf("seed telemetry record: %v", err)
		}

		block := renderSpendCostLine(197)
		cell := costLineElapsedCell(t, block)
		want := "10m0s (largest measured piece: worker execution, 7m0s)"
		if cell != want {
			t.Fatalf("elapsed cell = %q, want %q\nfull block:\n%s", cell, want, block)
		}
		if n := strings.Count(block, spendCostLineHeading); n != 1 {
			t.Fatalf("expected exactly one %q block, found %d:\n%s", spendCostLineHeading, n, block)
		}
		if n := strings.Count(block, "Elapsed:"); n != 1 {
			t.Fatalf("expected exactly one timing line, found %d:\n%s", n, block)
		}
	})

	t.Run("states the breakdown is unmeasured when no segment was measured", func(t *testing.T) {
		setupSpendTestStore(t)
		seedSpendLedgerForTest(t, 198, spendWorkflowBuild,
			measuredSpendRowForTest("Mason-67", "builder", 1_200_000),
		)
		seedSpendElapsedAttemptForTest(t, 198, "attempt-timing-unmeasured", "2026-01-01T00:00:00Z", "2026-01-01T00:05:00Z")
		// No telemetry record written for this attempt at all.

		block := renderSpendCostLine(198)
		cell := costLineElapsedCell(t, block)
		want := "5m0s (timing breakdown not measured)"
		if cell != want {
			t.Fatalf("elapsed cell = %q, want %q\nfull block:\n%s", cell, want, block)
		}
	})
}

// TestStatusDrillDownRendersAllEightSegments proves renderJobTelemetryDrillDown
// (cmd/status.go) renders all eight named segments for a selected attempt,
// naming unmeasured ones as such, and that calling it never writes to the
// data directory -- status.go is a reader only.
func TestStatusDrillDownRendersAllEightSegments(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentContext, 900*time.Millisecond, "brief assembly")
	capture.measure(jobTelemetrySegmentVerification, 12*time.Second, "deterministic floor run")
	capture.markUnmeasured(jobTelemetrySegmentPreflight, "no first-response boundary is observable today")
	record := newJobTelemetryRecord("attempt-drilldown", "job-drilldown", capture, time.Now())
	if err := writeJobTelemetryRecord(record); err != nil {
		t.Fatalf("seed telemetry record: %v", err)
	}

	before := hashDirForTest(t, store.BasePath())
	rendered := renderJobTelemetryDrillDown("attempt-drilldown")
	after := hashDirForTest(t, store.BasePath())
	if before != after {
		t.Fatalf("renderJobTelemetryDrillDown wrote to the data directory: before=%s after=%s", before, after)
	}

	for _, name := range jobTelemetrySegmentOrder {
		label := jobTelemetrySegmentLabel(name)
		if !strings.Contains(rendered, label) {
			t.Fatalf("drill-down is missing segment %q (label %q):\n%s", name, label, rendered)
		}
	}
	if !strings.Contains(rendered, "900ms") {
		t.Fatalf("drill-down does not render the measured context duration:\n%s", rendered)
	}
	if !strings.Contains(rendered, jobTelemetryUnmeasuredFigure) {
		t.Fatalf("drill-down does not name any unmeasured segment as such:\n%s", rendered)
	}

	// A selected attempt with no telemetry record at all renders nothing --
	// distinct from a record existing with every segment unmeasured (which
	// still renders the eight lines above), matching
	// renderJobTelemetryClosingLine's identical choice in
	// cmd/spend_cost_line.go.
	if got := renderJobTelemetryDrillDown("attempt-never-recorded"); got != "" {
		t.Fatalf("expected no drill-down output for an attempt with no telemetry record, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------
// The report-only guard (D-16): no selection function ever reads telemetry.
// ---------------------------------------------------------------------

// jobTelemetrySelectionEntryPoints names the four kinds of decision D-16
// (WORK-08's own must_have) forbids from ever reading a timing record:
// model selection, team selection, test-scope selection, and brief
// assembly.
var jobTelemetrySelectionEntryPoints = []string{
	"resolveCasteModel",
	"queenApplyJudgement",
	"deriveVerificationScope",
	"composeBuildManifestBrief",
}

// jobTelemetryReadFunctionNames are the functions this guard treats as "a
// telemetry read" -- currently just readJobTelemetryRecord, the one
// function that loads a written jobTelemetryRecord back off disk.
var jobTelemetryReadFunctionNames = map[string]bool{
	"readJobTelemetryRecord": true,
}

// jobTelemetryReportOnlyViolation is what jobTelemetryEntryPointReachesTelemetryRead
// returns when an entry point's call graph reaches a telemetry read: the
// exact function that made the offending call, and where.
type jobTelemetryReportOnlyViolation struct {
	Function string
	Position token.Position
}

// jobTelemetryCalleeCallPositions returns, for one function body, every
// plain top-level identifier call it makes, mapped to the position of its
// first occurrence -- both a name set (for recursion) and a position (for
// reporting exactly where an offending call sits).
func jobTelemetryCalleeCallPositions(fn *ast.FuncDecl) map[string]token.Pos {
	calls := map[string]token.Pos{}
	if fn == nil || fn.Body == nil {
		return calls
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok {
			if _, seen := calls[ident.Name]; !seen {
				calls[ident.Name] = call.Pos()
			}
		}
		return true
	})
	return calls
}

// jobTelemetryEntryPointReachesTelemetryRead walks the call graph from
// entry -- resolving package-level function-value aliases the same way
// autopilotCallGraphReaches (cmd/autopilot_goal_level_test.go) does -- and
// returns the first function whose body directly calls a name in
// jobTelemetryReadFunctionNames, or nil when entry's call graph never
// reaches one.
func jobTelemetryEntryPointReachesTelemetryRead(fset *token.FileSet, funcs map[string]*ast.FuncDecl, aliases map[string]string, entry string) *jobTelemetryReportOnlyViolation {
	visited := map[string]bool{}
	var walk func(name string) *jobTelemetryReportOnlyViolation
	walk = func(name string) *jobTelemetryReportOnlyViolation {
		if visited[name] {
			return nil
		}
		visited[name] = true
		resolvedName := name
		if resolved, ok := aliases[name]; ok {
			resolvedName = resolved
		}
		fn, ok := funcs[resolvedName]
		if !ok {
			return nil
		}
		calls := jobTelemetryCalleeCallPositions(fn)
		for callee, pos := range calls {
			if jobTelemetryReadFunctionNames[callee] {
				return &jobTelemetryReportOnlyViolation{Function: resolvedName, Position: fset.Position(pos)}
			}
		}
		for callee := range calls {
			if v := walk(callee); v != nil {
				return v
			}
		}
		return nil
	}
	return walk(entry)
}

// TestTelemetryIsReportOnly proves D-16: none of the four selection entry
// points reaches a telemetry read in the real package, and the guard
// itself is proven to bite against a temporary fixture call path that
// does.
func TestTelemetryIsReportOnly(t *testing.T) {
	fset := token.NewFileSet()
	funcs := continueDecisionPackageFuncs(t, fset)
	aliases := autopilotDispatchAliasMap(t, fset)

	t.Run("the report-only guard passes against the real package", func(t *testing.T) {
		for _, entry := range jobTelemetrySelectionEntryPoints {
			if _, ok := funcs[entry]; !ok {
				t.Fatalf("expected selection entry point %q to be declared in the cmd package", entry)
			}
			if v := jobTelemetryEntryPointReachesTelemetryRead(fset, funcs, aliases, entry); v != nil {
				t.Fatalf("%s reaches a telemetry read via %s at %s -- a selection function must never read timing data (D-16)", entry, v.Function, v.Position)
			}
		}
	})

	t.Run("the guard fails, naming the function and position, against a temporary fixture call path", func(t *testing.T) {
		src := `package cmd

func jobTelemetryGuardFixtureSelection() {
	jobTelemetryGuardFixtureIntermediate()
}

func jobTelemetryGuardFixtureIntermediate() {
	readJobTelemetryRecord("fixture-attempt")
}
`
		fixtureFset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fixtureFset, "job_telemetry_guard_fixture.go", src, 0)
		if err != nil {
			t.Fatalf("parse fixture source: %v", err)
		}
		fixtureFuncs := continueDecisionPackageFuncs(t, fixtureFset, fixtureFile)
		v := jobTelemetryEntryPointReachesTelemetryRead(fixtureFset, fixtureFuncs, aliases, "jobTelemetryGuardFixtureSelection")
		if v == nil {
			t.Fatal("expected the guard to catch the fixture's call path to a telemetry read, but it did not fire")
		}
		if v.Function != "jobTelemetryGuardFixtureIntermediate" {
			t.Fatalf("expected the guard to name jobTelemetryGuardFixtureIntermediate as the offending function, got %q", v.Function)
		}
		if v.Position.Line == 0 {
			t.Fatal("expected the guard to report a real file position for the offending call")
		}
	})
}
