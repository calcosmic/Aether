package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const overnightMeasuredUsageEvent = `{"type":"result","usage":{"input_tokens":1200,"output_tokens":300},"model":"overnight-fixture"}`

// overnightRunRecorder is shared by every invoker instance created during one
// run. The production factory creates a fresh invoker for build, verification,
// and review, so keeping observations here proves the phase-scoped preflight
// cache spans those independent consumers.
type overnightRunRecorder struct {
	mu             sync.Mutex
	preflightCalls map[int]int
	workerLog      []string
}

func newOvernightRunRecorder() *overnightRunRecorder {
	return &overnightRunRecorder{preflightCalls: map[int]int{}}
}

func (r *overnightRunRecorder) recordPreflight() {
	r.mu.Lock()
	defer r.mu.Unlock()
	var state colony.ColonyState
	if store != nil && store.LoadJSON("COLONY_STATE.json", &state) == nil {
		r.preflightCalls[state.CurrentPhase]++
	}
}

func (r *overnightRunRecorder) recordWorker(config codex.WorkerConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workerLog = append(r.workerLog, fmt.Sprintf("phase=%d caste=%s task=%s", phaseFromOvernightTaskID(config.TaskID), config.Caste, config.TaskID))
}

func (r *overnightRunRecorder) snapshot() (map[int]int, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make(map[int]int, len(r.preflightCalls))
	for phase, count := range r.preflightCalls {
		calls[phase] = count
	}
	return calls, append([]string{}, r.workerLog...)
}

// overnightRunInvoker extends completingRunInvoker: it keeps the genuine
// tiny workspace writes and verified claims, then adds only the evidence this
// joined-up overnight story needs (one UI file, measured usage, and a valid
// Auditor review artifact).
type overnightRunInvoker struct {
	completingRunInvoker
	recorder *overnightRunRecorder
}

func (i *overnightRunInvoker) Platform() codex.Platform { return codex.PlatformFake }

func (i *overnightRunInvoker) Preflight(ctx context.Context, root string) codex.AvailabilityStatus {
	i.recorder.recordPreflight()
	return i.completingRunInvoker.Preflight(ctx, root)
}

func (i *overnightRunInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result, err := i.completingRunInvoker.Invoke(ctx, config)
	return i.decorate(config, result, err)
}

func (i *overnightRunInvoker) InvokeWithProgress(ctx context.Context, config codex.WorkerConfig, observer codex.WorkerProgressObserver) (codex.WorkerResult, error) {
	result, err := i.completingRunInvoker.InvokeWithProgress(ctx, config, observer)
	return i.decorate(config, result, err)
}

func (i *overnightRunInvoker) decorate(config codex.WorkerConfig, result codex.WorkerResult, err error) (codex.WorkerResult, error) {
	i.recorder.recordWorker(config)
	if err != nil {
		return result, err
	}
	phase := phaseFromOvernightTaskID(config.TaskID)
	if config.Caste == "builder" && phase == 2 {
		rel := filepath.ToSlash(filepath.Join("web", "components", "OvernightStatus.tsx"))
		abs := filepath.Join(config.Root, filepath.FromSlash(rel))
		if writeErr := os.MkdirAll(filepath.Dir(abs), 0o755); writeErr != nil {
			return result, writeErr
		}
		if writeErr := os.WriteFile(abs, []byte("export const OvernightStatus = () => 'ready'\n"), 0o644); writeErr != nil {
			return result, writeErr
		}
		result.FilesCreated = append(result.FilesCreated, rel)
	}
	if config.Caste == "builder" && phase == 1 {
		result.RawOutput = strings.TrimRight(result.RawOutput, "\n") + "\n" + overnightMeasuredUsageEvent + "\n"
		result.Usage = codex.WorkerUsage{}
		result = codex.AttachWorkerUsage(result, config)
	}
	if config.Caste == "auditor" {
		result.Artifacts = map[string]json.RawMessage{
			"review": json.RawMessage(`{"overall_score":60,"findings":[{"domain":"quality","severity":"HIGH","title":"Morning follow-up","description":"Review the migration notes with the owner.","suggestion":"Keep this visible in the morning report."}]}`),
		}
	}
	return result, nil
}

func phaseFromOvernightTaskID(taskID string) int {
	prefix := strings.TrimSpace(taskID)
	if at := strings.IndexAny(prefix, ".-"); at >= 0 {
		prefix = prefix[:at]
	}
	phase, _ := strconv.Atoi(prefix)
	return phase
}

func seedOvernightRunFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir, root = seedRunFixture(t, 6)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load overnight fixture: %v", err)
	}
	boundary := time.Date(2026, time.August, 30, 8, 0, 0, 0, time.UTC)
	state.Plan.GeneratedAt = &boundary
	state.Plan.Phases[1].Name = "Visible overnight dashboard"
	state.Plan.Phases[2].Name = "Database migration review"
	state.Plan.Phases[2].Description = "Check a database migration before the morning handoff."
	criterion := "The owner confirms the overnight result in the morning"
	state.Plan.Phases[3].SuccessCriteria = []string{criterion}
	state.Plan.Phases[3].EvidenceRequirements = []colony.CriterionEvidenceRequirement{{
		Criterion: criterion,
		Checks:    []string{"watcher"},
	}}
	// This phase deliberately has no dispatched reviewer. Removing its
	// current-build claims immediately before the real continue call below
	// leaves no deterministic source that can pretend to judge the owner-only
	// criterion, which is exactly the runtime-verification queue condition.
	state.Plan.Phases[3].WatcherFailureCount = defaultWatcherFailureThreshold
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save overnight fixture: %v", err)
	}
	return dataDir, root
}

func installOvernightInvoker(t *testing.T, recorder *overnightRunRecorder) {
	t.Helper()
	original := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		return &overnightRunInvoker{recorder: recorder}
	}
	t.Cleanup(func() { newCodexWorkerInvoker = original })
}

func seedOrdinaryOvernightBlocker(t *testing.T) {
	t.Helper()
	stdout.(*bytes.Buffer).Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"flag-add", "blocker", "Known morning follow-up", "Ordinary blocker already known before the run", "overnight-fixture", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("write ordinary blocker through flag-add: %v", err)
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("flag-add exited %d: %s", code, stdout.(*bytes.Buffer).String())
	}
	if snapshot, err := readBlockerSnapshot(store); err != nil || snapshot.Count != 1 || snapshot.EscalatedCount != 0 {
		t.Fatalf("ordinary blocker fixture = %+v, want one non-escalated baseline", snapshot)
	}
	stdout.(*bytes.Buffer).Reset()
}

func installOvernightClock(t *testing.T) {
	t.Helper()
	original := autopilotNow
	base := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	var tick int64
	autopilotNow = func() time.Time {
		tick++
		return base.Add(time.Duration(tick) * time.Minute)
	}
	t.Cleanup(func() { autopilotNow = original })
}

func installOvernightOwnerCriterionGap(t *testing.T, phaseID int) {
	t.Helper()
	original := runAutopilotContinue
	runAutopilotContinue = func(root string, options codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, error) {
		state, err := loadCompatibilityColonyState()
		if err == nil && state.CurrentPhase == phaseID {
			// Reconciliation is the runtime's supported evidence path for work
			// completed without a readable builder-claims packet. It keeps task
			// evidence positive while correctly leaving the owner-only watcher
			// criterion without a machine proof.
			options.ReconcileTaskIDs = []string{fmt.Sprintf("%d.1", phaseID)}
			manifest := loadCodexContinueManifest(phaseID)
			claimsRel := strings.TrimPrefix(strings.TrimSpace(manifest.Data.ClaimsPath), ".aether/data/")
			if claimsRel == "" {
				claimsRel = "last-build-claims.json"
			}
			if removeErr := os.Remove(filepath.Join(store.BasePath(), filepath.FromSlash(claimsRel))); removeErr != nil && !os.IsNotExist(removeErr) {
				return nil, state, colony.Phase{}, nil, nil, false, removeErr
			}
		}
		return original(root, options)
	}
	t.Cleanup(func() { runAutopilotContinue = original })
}

func TestOvernightRunCompletesSixPhases(t *testing.T) {
	saveGlobalsCmd(t)
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	_, _ = seedOvernightRunFixture(t)
	recorder := newOvernightRunRecorder()
	installOvernightInvoker(t, recorder)
	installOvernightClock(t)
	installOvernightOwnerCriterionGap(t, 4)
	seedOrdinaryOvernightBlocker(t)

	rootCmd.SetArgs([]string{"run", "--headless", "--replan-interval", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("overnight run returned error: %v", err)
	}
	envelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := envelope["result"].(map[string]interface{})
	if result["stopped_reason"] != "completed" || intValue(result["phases_completed"]) != 6 {
		t.Fatalf("overnight run did not finish six phases: stopped_reason=%v phases_completed=%v result=%+v", result["stopped_reason"], result["phases_completed"], result)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load completed colony: %v", err)
	}
	if state.State != colony.StateCOMPLETED || len(state.Plan.Phases) != 6 {
		t.Fatalf("durable colony state = %s with %d phases, want COMPLETED with 6", state.State, len(state.Plan.Phases))
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			t.Errorf("phase %d status = %q, want completed", phase.ID, phase.Status)
		}
		for _, task := range phase.Tasks {
			if task.Status != colony.TaskCompleted {
				t.Errorf("phase %d task %v status = %q, want completed", phase.ID, task.ID, task.Status)
			}
		}
	}

	var decisions PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &decisions); err != nil {
		t.Fatalf("load morning decisions: %v", err)
	}
	types := map[string]int{}
	for _, decision := range decisions.Decisions {
		if !decision.Resolved {
			types[decision.Type]++
		}
	}
	for _, want := range []string{autopilotCheckpointTypeVisual, autopilotCheckpointTypeRuntimeVerification, autopilotReplanDecisionType} {
		if types[want] == 0 {
			t.Errorf("morning queue has no unresolved %q item: %+v", want, decisions.Decisions)
		}
	}

	var persisted autopilotState
	if err := store.LoadJSON(autopilotStatePath, &persisted); err != nil || persisted.LastReport == nil {
		t.Fatalf("load final autopilot report: report=%+v err=%v", persisted.LastReport, err)
	}
	report := persisted.LastReport
	if report.Outcome != "completed" || report.StopReason != string(autopilotTriggerColonyComplete) || report.Next != "aether seal" {
		t.Errorf("final report outcome/stop/next = %q/%q/%q", report.Outcome, report.StopReason, report.Next)
	}
	if report.ElapsedSeconds <= 0 || len(report.Phases) != 6 || report.PhasesCompleted != 6 {
		t.Errorf("final report elapsed/phases = %d/%d/%d", report.ElapsedSeconds, len(report.Phases), report.PhasesCompleted)
	}
	if report.BlockersBefore.Snapshot == nil || report.BlockersBefore.Snapshot.Count != 1 || !reflect.DeepEqual(report.BlockersBefore, report.BlockersAfter) {
		t.Errorf("blocker movement = before %+v after %+v, want one unchanged ordinary blocker", report.BlockersBefore, report.BlockersAfter)
	}
	if report.Spend.MeasuredTokens == nil || report.Spend.MeasuredRows == 0 || report.Spend.UnreportedRows == 0 {
		t.Errorf("spend report is not measured/unreported honest: %+v", report.Spend)
	}
	high := 0
	for _, phase := range report.Phases {
		high += len(phase.HighFindings)
	}
	if high == 0 {
		t.Error("final report omitted the passing Auditor's High finding")
	}

	preflights, workerLog := recorder.snapshot()
	if len(preflights) != 6 {
		t.Errorf("preflight phases = %+v, want one entry per six phases", preflights)
	}
	for phase := 1; phase <= 6; phase++ {
		if preflights[phase] != 1 {
			t.Errorf("phase %d preflight count = %d, want 1", phase, preflights[phase])
		}
	}
	gateResults, err := gateResultsReadPhase(6)
	if err != nil {
		t.Fatalf("load final phase gates: %v", err)
	}
	baselineGateSeen := false
	for _, gate := range gateResults {
		if gate.Name != "no_unresolved_blockers" {
			continue
		}
		baselineGateSeen = true
		if gate.Status != "passed" || !strings.Contains(gate.Detail, "remain unresolved") {
			t.Errorf("autopilot blocker-baseline gate = %+v, want truthful pass with unresolved follow-up", gate)
		}
	}
	if !baselineGateSeen {
		t.Error("final phase omitted the no_unresolved_blockers gate")
	}

	// Omitting the internal run-only baseline is the direct-continue path.
	// It must keep the classic strict Iron Law against the same live flag.
	directGates := runCodexContinueGates(
		state.Plan.Phases[5],
		codexContinueManifest{Present: true},
		codexContinueVerificationReport{ChecksPassed: true},
		codexContinueAssessment{PositiveEvidence: true},
		time.Now().UTC(),
		nil,
	)
	directBlockerSeen := false
	for _, gate := range directGates.Checks {
		if gate.Name == "no_unresolved_blockers" {
			directBlockerSeen = true
			if gate.Passed {
				t.Errorf("direct continue blocker gate unexpectedly passed: %+v", gate)
			}
		}
	}
	if !directBlockerSeen {
		t.Error("direct continue gate set omitted no_unresolved_blockers")
	}
	joinedLog := strings.ToLower(strings.Join(workerLog, "\n"))
	for _, forbidden := range []string{"seal", "provider switch", "skip-phase", "phase=7"} {
		if strings.Contains(joinedLog, forbidden) {
			t.Errorf("worker log contains forbidden automatic action %q:\n%s", forbidden, joinedLog)
		}
	}

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status after overnight run: %v", err)
	}
	statusResult := parseEnvelope(t, stdout.(*bytes.Buffer).String())["result"].(map[string]interface{})
	statusJSON, _ := json.Marshal(statusResult["last_report"])
	var statusReport autopilotInvocationReport
	if err := json.Unmarshal(statusJSON, &statusReport); err != nil {
		t.Fatalf("decode status report: %v", err)
	}
	if !reflect.DeepEqual(statusReport, *report) {
		reportJSON, _ := json.Marshal(report)
		t.Fatalf("status changed the stored overnight report:\nstatus=%s\nstored=%s", statusJSON, reportJSON)
	}
}

func TestOvernightRunBlockerBaselineExceptionIsNarrow(t *testing.T) {
	saveGlobalsCmd(t)
	saveGlobals(t)

	tests := []struct {
		name           string
		before         []colony.FlagEntry
		after          []colony.FlagEntry
		supplyBaseline bool
		wantPassed     bool
	}{
		{
			name:           "unchanged autopilot baseline",
			before:         []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			after:          []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			supplyBaseline: true,
			wantPassed:     true,
		},
		{
			name:           "new blocker count",
			before:         []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			after:          []colony.FlagEntry{{ID: "known", Type: "blocker"}, {ID: "new", Type: "blocker"}},
			supplyBaseline: true,
			wantPassed:     false,
		},
		{
			name:           "new escalation evidence",
			before:         []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			after:          []colony.FlagEntry{{ID: "known", Type: "blocker", Source: "escalation"}},
			supplyBaseline: true,
			wantPassed:     false,
		},
		{
			name:       "direct continue remains strict",
			before:     []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			after:      []colony.FlagEntry{{ID: "known", Type: "blocker"}},
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestStore(t)
			store = s
			writeTestFlags(t, tt.before...)
			baseline, err := readBlockerSnapshot(store)
			if err != nil {
				t.Fatalf("read blocker baseline: %v", err)
			}
			writeTestFlags(t, tt.after...)
			flagsBeforeGate := append([]colony.FlagEntry{}, readTestFlags(t)...)

			var report codexContinueGateReport
			if tt.supplyBaseline {
				report = runCodexContinueGatesWithAutopilotBaseline(
					colony.Phase{ID: 1, Name: "overnight blocker contract"},
					codexContinueManifest{Present: true},
					codexContinueVerificationReport{ChecksPassed: true},
					codexContinueAssessment{PositiveEvidence: true},
					time.Now().UTC(),
					nil,
					&baseline,
				)
			} else {
				report = runCodexContinueGates(
					colony.Phase{ID: 1, Name: "direct continue blocker contract"},
					codexContinueManifest{Present: true},
					codexContinueVerificationReport{ChecksPassed: true},
					codexContinueAssessment{PositiveEvidence: true},
					time.Now().UTC(),
					nil,
				)
			}

			seen := false
			for _, gate := range report.Checks {
				if gate.Name != "no_unresolved_blockers" {
					continue
				}
				seen = true
				if gate.Passed != tt.wantPassed {
					t.Errorf("blocker gate passed = %v, want %v: %+v", gate.Passed, tt.wantPassed, gate)
				}
				if tt.wantPassed && !strings.Contains(gate.Detail, "remain unresolved") {
					t.Errorf("stable baseline detail hides unresolved work: %q", gate.Detail)
				}
			}
			if !seen {
				t.Fatal("continue gates omitted no_unresolved_blockers")
			}
			if got := readTestFlags(t); !reflect.DeepEqual(got, flagsBeforeGate) {
				t.Errorf("blocker gate rewrote live flags:\nbefore=%+v\nafter=%+v", flagsBeforeGate, got)
			}
		})
	}
}

// The end-to-end test above is the stamina proof. This small companion locks
// the catalogue side of its mutation sensitivity: changing any of the three
// owner-only conditions away from queue-and-continue makes the live six-phase
// test stop before completion (the execution summary records the three manual
// mutation runs required because production intentionally exposes no policy
// override seam).
func TestOvernightRunQueueableMutationStopsEarly(t *testing.T) {
	saveGlobalsCmd(t)
	for _, code := range []autopilotTriggerCode{
		autopilotTriggerVisualCheckpointNeeded,
		autopilotTriggerRuntimeVerificationNeeded,
		autopilotTriggerReplanDue,
	} {
		spec, ok := autopilotTriggerSpecByCode(code)
		if !ok {
			t.Errorf("queueable trigger %q disappeared from the catalogue", code)
			continue
		}
		if spec.HeadlessDisposition != autopilotDispositionQueueAndContinue {
			t.Errorf("queueable trigger %q headless disposition = %q; the live overnight test must fail before phase six", code, spec.HeadlessDisposition)
		}
	}
}
