package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const autopilotReportSchemaVersion = 1

// autopilotQueuedDecisionReport is the durable owner-work reference kept in a
// run report. The pending-decision file remains authoritative; this snapshot
// is what lets a later status render the run exactly as it ended.
type autopilotQueuedDecisionReport struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Phase    int    `json:"phase,omitempty"`
	Question string `json:"question,omitempty"`
}

// autopilotPhaseReport is the compact outcome for one phase touched by this
// invocation. Findings are deliberately the typed review records, never prose
// scraped back out of a rendered continue card.
type autopilotPhaseReport struct {
	Phase           int                             `json:"phase"`
	PhaseName       string                          `json:"phase_name,omitempty"`
	Outcome         string                          `json:"outcome"`
	HighFindings    []codexReviewFinding            `json:"high_findings,omitempty"`
	QueuedDecisions []autopilotQueuedDecisionReport `json:"queued_decisions,omitempty"`
}

// autopilotPhaseSpendReport is a frozen projection of the existing phase
// ledgers. A nil token pointer means no measured figure exists. Estimated rows
// are counted as unreported and their numbers never enter this durable report.
type autopilotPhaseSpendReport struct {
	Phase          int               `json:"phase"`
	PhaseName      string            `json:"phase_name,omitempty"`
	LedgerPresent  bool              `json:"ledger_present"`
	MeasuredTokens *int64            `json:"measured_tokens"`
	MeasuredRows   int               `json:"measured_rows"`
	UnreportedRows int               `json:"unreported_rows"`
	Workers        []spendWorkerJSON `json:"workers,omitempty"`
}

// autopilotSpendReport has no estimate or currency field by design. Numeric
// token totals exist only when at least one runtime-owned ledger row measured
// them; otherwise nil serializes as JSON null.
type autopilotSpendReport struct {
	MeasuredTokens *int64                      `json:"measured_tokens"`
	MeasuredRows   int                         `json:"measured_rows"`
	UnreportedRows int                         `json:"unreported_rows"`
	Phases         []autopilotPhaseSpendReport `json:"phases"`
}

// autopilotBlockerSnapshotReport preserves the difference between a verified
// empty blocker set and blocker truth that could not be read.
type autopilotBlockerSnapshotReport struct {
	Available bool             `json:"available"`
	Snapshot  *blockerSnapshot `json:"snapshot"`
	Error     string           `json:"error,omitempty"`
}

func availableAutopilotBlockerSnapshot(snapshot blockerSnapshot) autopilotBlockerSnapshotReport {
	return autopilotBlockerSnapshotReport{Available: true, Snapshot: &snapshot}
}

func captureAutopilotBlockerSnapshot(s *storage.Store) autopilotBlockerSnapshotReport {
	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		return autopilotBlockerSnapshotReport{
			Available: false,
			Snapshot:  nil,
			Error:     blockerSnapshotErrorDetail(s, err),
		}
	}
	return availableAutopilotBlockerSnapshot(snapshot)
}

// autopilotRecoveryReport is the bounded recovery handoff for a genuine stop.
// MedicAdvice is an owner command only: this integration never constructs or
// dispatches a worker.
type autopilotRecoveryReport struct {
	EntryID        string                `json:"entry_id"`
	Classification FailureClassification `json:"classification"`
	FailureType    FailureType           `json:"failure_type"`
	Rationale      string                `json:"rationale"`
	MedicAdvice    string                `json:"medic_advice,omitempty"`
	MedicReason    string                `json:"medic_reason,omitempty"`
	LogError       string                `json:"log_error,omitempty"`
}

// autopilotInvocationReport is the one versioned record written at every real
// run ending. Status reads this value; it never reconstructs elapsed time,
// spend, findings, decisions, blockers, or the next command from newer data.
type autopilotInvocationReport struct {
	SchemaVersion   int                             `json:"schema_version"`
	InvocationID    string                          `json:"invocation_id"`
	StartedAt       string                          `json:"started_at"`
	FinishedAt      string                          `json:"finished_at"`
	ElapsedSeconds  int64                           `json:"elapsed_seconds"`
	Outcome         string                          `json:"outcome"`
	StopReason      string                          `json:"stop_reason"`
	CurrentPhase    int                             `json:"current_phase"`
	PhasesCompleted int                             `json:"phases_completed"`
	QueuedDecisions []autopilotQueuedDecisionReport `json:"queued_decisions"`
	BlockersBefore  autopilotBlockerSnapshotReport  `json:"blockers_before"`
	BlockersAfter   autopilotBlockerSnapshotReport  `json:"blockers_after"`
	Spend           autopilotSpendReport            `json:"spend"`
	Recovery        *autopilotRecoveryReport        `json:"recovery,omitempty"`
	Next            string                          `json:"next"`
	Phases          []autopilotPhaseReport          `json:"phases"`
}

// autopilotInvocation is live, process-local bookkeeping. Its StartedAt is
// created after dry-run returns, so offline time and inspection commands can
// never inflate a real invocation's wall clock.
type autopilotInvocation struct {
	ID             string
	StartedAt      time.Time
	BlockersBefore autopilotBlockerSnapshotReport
	Phases         []autopilotPhaseReport
}

var (
	autopilotNow                = func() time.Time { return time.Now().UTC() }
	autopilotInvocationSequence atomic.Uint64
	autopilotNewInvocationID    = func(now time.Time) string {
		return fmt.Sprintf("run-%d-%d", now.UnixNano(), autopilotInvocationSequence.Add(1))
	}
)

func beginAutopilotInvocation(state colony.ColonyState) autopilotInvocation {
	now := autopilotNow().UTC()
	return autopilotInvocation{
		ID:             autopilotNewInvocationID(now),
		StartedAt:      now,
		BlockersBefore: captureAutopilotBlockerSnapshot(store),
		Phases:         []autopilotPhaseReport{},
	}
}

func (invocation *autopilotInvocation) phase(phase colony.Phase, outcome string) *autopilotPhaseReport {
	if invocation == nil || phase.ID <= 0 {
		return nil
	}
	for i := range invocation.Phases {
		if invocation.Phases[i].Phase == phase.ID {
			if strings.TrimSpace(phase.Name) != "" {
				invocation.Phases[i].PhaseName = phase.Name
			}
			if strings.TrimSpace(outcome) != "" {
				invocation.Phases[i].Outcome = outcome
			}
			return &invocation.Phases[i]
		}
	}
	invocation.Phases = append(invocation.Phases, autopilotPhaseReport{
		Phase:     phase.ID,
		PhaseName: phase.Name,
		Outcome:   outcome,
	})
	return &invocation.Phases[len(invocation.Phases)-1]
}

func phaseForAutopilotReport(state colony.ColonyState, phaseID int) colony.Phase {
	for _, phase := range state.Plan.Phases {
		if phase.ID == phaseID {
			return phase
		}
	}
	return colony.Phase{ID: phaseID}
}

func (invocation *autopilotInvocation) recordSignals(phase colony.Phase, signals codexContinueAutopilotSignals) {
	entry := invocation.phase(phase, "")
	if entry == nil {
		return
	}
	seenFinding := make(map[string]struct{}, len(entry.HighFindings))
	for _, finding := range entry.HighFindings {
		seenFinding[autopilotFindingKey(finding)] = struct{}{}
	}
	for _, finding := range signals.Findings {
		if !strings.EqualFold(strings.TrimSpace(finding.Severity), "HIGH") {
			continue
		}
		key := autopilotFindingKey(finding)
		if _, exists := seenFinding[key]; exists {
			continue
		}
		entry.HighFindings = append(entry.HighFindings, finding)
		seenFinding[key] = struct{}{}
	}
	for _, checkpoint := range signals.Checkpoints {
		invocation.recordQueuedDecision(autopilotQueuedDecisionReport{
			ID: checkpoint.ID, Type: checkpoint.Type, Phase: checkpoint.Phase, Question: checkpoint.Question,
		})
	}
}

func autopilotFindingKey(finding codexReviewFinding) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(finding.Domain)),
		strings.ToLower(strings.TrimSpace(finding.Severity)),
		strings.TrimSpace(finding.File),
		fmt.Sprintf("%d", finding.Line),
		strings.TrimSpace(finding.Title),
		strings.TrimSpace(finding.Description),
	}, "\x00")
}

func (invocation *autopilotInvocation) recordRunDecision(state colony.ColonyState, decision autopilotRunDecision) {
	if invocation == nil {
		return
	}
	for _, checkpoint := range decision.Checkpoints {
		invocation.recordQueuedDecision(autopilotQueuedDecisionReport{
			ID: checkpoint.ID, Type: checkpoint.Type, Phase: checkpoint.Phase, Question: checkpoint.Question,
		})
	}
	phaseID := state.CurrentPhase
	hasEvidencePhase := false
	if raw, ok := decision.Evidence["phase"]; ok && intValue(raw) > 0 {
		phaseID = intValue(raw)
		hasEvidencePhase = true
	}
	if phaseID <= 0 {
		return
	}
	for i := range invocation.Phases {
		if invocation.Phases[i].Phase == phaseID {
			invocation.phase(phaseForAutopilotReport(state, phaseID), autopilotReportOutcome(decision))
			return
		}
	}
	if hasEvidencePhase || state.State == colony.StateBUILT || state.State == colony.StateEXECUTING {
		invocation.phase(phaseForAutopilotReport(state, phaseID), autopilotReportOutcome(decision))
	}
}

func (invocation *autopilotInvocation) recordPendingDecision(decision PendingDecision, fallbackPhase int) {
	phase := fallbackPhase
	if decision.Phase != nil {
		phase = *decision.Phase
	}
	invocation.recordQueuedDecision(autopilotQueuedDecisionReport{
		ID: decision.ID, Type: decision.Type, Phase: phase, Question: decision.Description,
	})
}

func (invocation *autopilotInvocation) recordQueuedDecision(queued autopilotQueuedDecisionReport) {
	if invocation == nil || strings.TrimSpace(queued.ID) == "" {
		return
	}
	phase := invocation.phase(colony.Phase{ID: queued.Phase}, "")
	if phase == nil {
		return
	}
	for _, existing := range phase.QueuedDecisions {
		if existing.ID == queued.ID {
			return
		}
	}
	phase.QueuedDecisions = append(phase.QueuedDecisions, queued)
}

func autopilotReportOutcome(decision autopilotRunDecision) string {
	if decision.Code == autopilotTriggerColonyComplete {
		return "completed"
	}
	switch decision.Disposition {
	case autopilotDispositionStop:
		return "genuine_stop"
	case autopilotDispositionPause:
		return "paused"
	case autopilotDispositionNormalStop:
		return "normal_stop"
	case autopilotDispositionQueueAndContinue:
		return "queued"
	default:
		return "stopped"
	}
}

func buildAutopilotInvocationReport(invocation autopilotInvocation, state colony.ColonyState, decision autopilotRunDecision, finished time.Time, blockersAfter autopilotBlockerSnapshotReport) autopilotInvocationReport {
	finished = finished.UTC()
	started := invocation.StartedAt.UTC()
	elapsed := finished.Sub(started)
	if started.IsZero() || elapsed < 0 {
		elapsed = 0
	}

	phases := make([]autopilotPhaseReport, len(invocation.Phases))
	copy(phases, invocation.Phases)
	queued := flattenAutopilotQueuedDecisions(phases)
	return autopilotInvocationReport{
		SchemaVersion:   autopilotReportSchemaVersion,
		InvocationID:    invocation.ID,
		StartedAt:       started.Format(time.RFC3339Nano),
		FinishedAt:      finished.Format(time.RFC3339Nano),
		ElapsedSeconds:  int64(elapsed / time.Second),
		Outcome:         autopilotReportOutcome(decision),
		StopReason:      string(decision.Code),
		CurrentPhase:    state.CurrentPhase,
		PhasesCompleted: autopilotReportCompletedPhases(phases, state, decision),
		QueuedDecisions: queued,
		BlockersBefore:  invocation.BlockersBefore,
		BlockersAfter:   blockersAfter,
		Spend:           snapshotAutopilotSpend(phases),
		Next:            resolveAutopilotReportNext(state, decision),
		Phases:          phases,
	}
}

func flattenAutopilotQueuedDecisions(phases []autopilotPhaseReport) []autopilotQueuedDecisionReport {
	out := []autopilotQueuedDecisionReport{}
	seen := map[string]struct{}{}
	for _, phase := range phases {
		for _, queued := range phase.QueuedDecisions {
			key := queued.ID + "\x00" + queued.Type
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, queued)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Phase != out[j].Phase {
			return out[i].Phase < out[j].Phase
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func autopilotReportCompletedPhases(phases []autopilotPhaseReport, state colony.ColonyState, decision autopilotRunDecision) int {
	completed := 0
	for _, phase := range phases {
		if phase.Outcome == "completed" {
			completed++
		}
	}
	if decision.Code != autopilotTriggerColonyComplete || completed > 0 {
		return completed
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			completed++
		}
	}
	return completed
}

func resolveAutopilotReportNext(state colony.ColonyState, decision autopilotRunDecision) string {
	if autopilotReportStateIsSealReady(state) {
		return "aether seal"
	}
	override := strings.TrimSpace(decision.Next)
	// Checkpoint commands contain a raw, single-use authorization capability.
	// They belong in the immediate run/seal response only. A frozen report
	// keeps the queued IDs/questions and points the owner back through the
	// lifecycle so a later invocation issues fresh authorization.
	if len(decision.Checkpoints) > 0 || strings.Contains(strings.ToLower(override), "--checkpoint-capability") {
		override = ""
	}
	return lifecycleNextActionForState(state, "run", override, "The autopilot stop named this exact recovery step.").Command
}

func autopilotReportStateIsSealReady(state colony.ColonyState) bool {
	if len(state.Plan.Phases) == 0 {
		return false
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			return false
		}
	}
	return true
}

func recordAutopilotRecovery(report *autopilotInvocationReport, decision autopilotRunDecision, cause error) {
	if report == nil || decision.Disposition != autopilotDispositionStop {
		return
	}
	phase := report.CurrentPhase
	message := autopilotRecoveryMessage(decision, cause)
	status := autopilotRecoveryStatus(decision.Code)
	classification, failureType, rationale := classifyWorkerFailure(status, message)
	entryID := autopilotRecoveryEntryID(report.InvocationID, phase, decision.Code)
	entry := RecoveryLogEntry{
		ID: entryID,
		Failure: FailureRecord{
			WorkerName:     "Autopilot",
			Phase:          phase,
			Status:         status,
			Classification: classification,
			FailureType:    failureType,
			ErrorMessage:   message,
			Timestamp:      report.FinishedAt,
		},
		ActionTaken:   "recorded genuine autopilot stop",
		Outcome:       "awaiting bounded recovery or owner action",
		AttemptNumber: 0,
		Timestamp:     report.FinishedAt,
		Detail:        rationale,
	}
	recovery := &autopilotRecoveryReport{
		EntryID:        entryID,
		Classification: classification,
		FailureType:    failureType,
		Rationale:      rationale,
	}
	if phase <= 0 {
		recovery.LogError = "recovery log unavailable: current phase is not known"
		report.Recovery = recovery
		return
	}
	if store == nil {
		recovery.LogError = "recovery log unavailable: store is not initialized"
		report.Recovery = recovery
		return
	}
	if existing, _, err := upsertRecoveryLogEntryPhase(phase, entry); err != nil {
		recovery.LogError = err.Error()
	} else {
		recovery.EntryID = existing.ID
		recovery.Classification = existing.Failure.Classification
		recovery.FailureType = existing.Failure.FailureType
		recovery.Rationale = existing.Detail
	}

	eligibility := shouldAutoSpawnMedic(store.BasePath())
	budget := budgetFromRecoveryLog(phase, 1)
	if eligibility.ShouldSpawn && budget != nil && budget.remaining() > 0 {
		recovery.MedicAdvice = "aether medic --deep"
		recovery.MedicReason = eligibility.Reason
	}
	report.Recovery = recovery
}

func autopilotRecoveryStatus(code autopilotTriggerCode) string {
	switch code {
	case autopilotTriggerCriticalReviewFinding,
		autopilotTriggerBlockerCountIncreased,
		autopilotTriggerBlockerEscalated:
		return "blocked"
	default:
		return "failed"
	}
}

func autopilotRecoveryMessage(decision autopilotRunDecision, cause error) string {
	parts := []string{"autopilot trigger " + string(decision.Code)}
	if len(decision.Evidence) > 0 {
		if evidence, err := json.Marshal(decision.Evidence); err == nil {
			parts = append(parts, "evidence="+string(evidence))
		}
	}
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		parts = append(parts, "error="+cause.Error())
	}
	return strings.Join(parts, "; ")
}

func autopilotRecoveryEntryID(invocationID string, phase int, code autopilotTriggerCode) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s", invocationID, phase, code)))
	return fmt.Sprintf("rl-autopilot-%x", digest[:10])
}

func snapshotAutopilotSpend(phases []autopilotPhaseReport) autopilotSpendReport {
	report := autopilotSpendReport{Phases: make([]autopilotPhaseSpendReport, 0, len(phases))}
	var measuredTotal int64
	for _, phase := range phases {
		ledgers, ok := loadSpendLedgersForPhase(phase.Phase)
		phaseSpend := autopilotPhaseSpendReport{
			Phase:         phase.Phase,
			PhaseName:     phase.PhaseName,
			LedgerPresent: ok,
			Workers:       []spendWorkerJSON{},
		}
		if ok {
			rows := spendRowsAcross(ledgers)
			totals := computeSpendTotals(ledgers)
			phaseSpend.Workers = spendWorkersJSON(ledgers)
			phaseSpend.MeasuredRows = totals.MeasuredRows
			phaseSpend.UnreportedRows = len(rows) - totals.MeasuredRows
			if phaseSpend.MeasuredRows > 0 {
				phaseMeasured := totals.MeasuredTokens
				phaseSpend.MeasuredTokens = &phaseMeasured
				measuredTotal += totals.MeasuredTokens
			}
		}
		report.MeasuredRows += phaseSpend.MeasuredRows
		report.UnreportedRows += phaseSpend.UnreportedRows
		report.Phases = append(report.Phases, phaseSpend)
	}
	if report.MeasuredRows > 0 {
		report.MeasuredTokens = &measuredTotal
	}
	return report
}

func loadAutopilotLastReport(s *storage.Store) *autopilotInvocationReport {
	if s == nil {
		return nil
	}
	var state autopilotState
	if err := s.LoadJSON(autopilotStatePath, &state); err != nil || state.LastReport == nil {
		return nil
	}
	if state.LastReport.SchemaVersion != autopilotReportSchemaVersion {
		return nil
	}
	return state.LastReport
}

func autopilotReportFromValue(raw interface{}) (*autopilotInvocationReport, bool) {
	switch value := raw.(type) {
	case autopilotInvocationReport:
		if value.SchemaVersion == autopilotReportSchemaVersion {
			return &value, true
		}
	case *autopilotInvocationReport:
		if value != nil && value.SchemaVersion == autopilotReportSchemaVersion {
			return value, true
		}
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, false
		}
		var report autopilotInvocationReport
		if json.Unmarshal(data, &report) == nil && report.SchemaVersion == autopilotReportSchemaVersion {
			return &report, true
		}
	}
	return nil, false
}

func renderAutopilotReportFromResult(result map[string]interface{}) string {
	if result == nil {
		return ""
	}
	report, ok := autopilotReportFromValue(result["last_report"])
	if !ok {
		return ""
	}
	return renderAutopilotInvocationReport(*report)
}

func renderAutopilotInvocationReport(report autopilotInvocationReport) string {
	var b strings.Builder
	b.WriteString(renderStageMarker("Last Autopilot Run"))
	fmt.Fprintf(&b, "Outcome: %s\n", renderAutopilotOutcome(report))
	fmt.Fprintf(&b, "Stop reason: %s\n", emptyFallback(report.StopReason, "none"))

	b.WriteString("Queued decisions\n")
	if len(report.QueuedDecisions) == 0 {
		b.WriteString("  none\n")
	} else {
		for _, queued := range report.QueuedDecisions {
			fmt.Fprintf(&b, "  - %s [%s]", emptyFallback(queued.ID, "unknown"), emptyFallback(queued.Type, "unknown"))
			if queued.Phase > 0 {
				fmt.Fprintf(&b, " phase %d", queued.Phase)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString(renderAutopilotBlockerMovement(report.BlockersBefore, report.BlockersAfter))
	if report.Recovery != nil {
		fmt.Fprintf(&b, "Recovery: %s (%s)\n", report.Recovery.Classification, report.Recovery.FailureType)
		if strings.TrimSpace(report.Recovery.LogError) != "" {
			fmt.Fprintf(&b, "Recovery log error: %s\n", report.Recovery.LogError)
		}
		if strings.TrimSpace(report.Recovery.MedicAdvice) != "" {
			fmt.Fprintf(&b, "Medic advice: %s\n", report.Recovery.MedicAdvice)
		}
	}
	fmt.Fprintf(&b, "Elapsed: %s\n", renderAutopilotElapsed(report.ElapsedSeconds))

	b.WriteString(renderAutopilotSpendReport(report.Spend))
	fmt.Fprintf(&b, "Next: %s\n", emptyFallback(report.Next, "aether status"))

	b.WriteString("Phase outcomes\n")
	if len(report.Phases) == 0 {
		b.WriteString("  none in this invocation\n")
	} else {
		for _, phase := range report.Phases {
			fmt.Fprintf(&b, "  - Phase %d", phase.Phase)
			if strings.TrimSpace(phase.PhaseName) != "" {
				fmt.Fprintf(&b, ": %s", phase.PhaseName)
			}
			fmt.Fprintf(&b, " — %s", emptyFallback(phase.Outcome, "observed"))
			if len(phase.HighFindings) > 0 {
				fmt.Fprintf(&b, "; %d HIGH finding(s)", len(phase.HighFindings))
			}
			if len(phase.QueuedDecisions) > 0 {
				fmt.Fprintf(&b, "; %d decision(s) queued", len(phase.QueuedDecisions))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderAutopilotBlockerMovement(before, after autopilotBlockerSnapshotReport) string {
	if !before.Available || before.Snapshot == nil || !after.Available || after.Snapshot == nil {
		return "Blocker movement: unavailable\n"
	}
	return fmt.Sprintf("Blocker movement: %d -> %d active; %d -> %d escalated\n",
		before.Snapshot.Count, after.Snapshot.Count,
		before.Snapshot.EscalatedCount, after.Snapshot.EscalatedCount)
}

func renderAutopilotOutcome(report autopilotInvocationReport) string {
	switch report.Outcome {
	case "completed":
		return fmt.Sprintf("completed %d phase(s)", report.PhasesCompleted)
	case "genuine_stop", "normal_stop", "paused":
		if report.CurrentPhase > 0 {
			return fmt.Sprintf("stopped at Phase %d (%s)", report.CurrentPhase, report.Outcome)
		}
	}
	return emptyFallback(report.Outcome, "unknown")
}

func renderAutopilotElapsed(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	duration := time.Duration(seconds) * time.Second
	if duration < time.Minute {
		return duration.String()
	}
	return duration.Round(time.Second).String()
}

func renderAutopilotSpendReport(report autopilotSpendReport) string {
	var b strings.Builder
	b.WriteString(renderStageMarker(spendCostLineHeading))
	if report.MeasuredTokens == nil {
		fmt.Fprintf(&b, "Cost: not known %s no measured token figure was reported.\n", spendNotReportedFigure)
	} else {
		fmt.Fprintf(&b, "Cost: %s measured tokens across %d reported worker run(s).\n",
			spendCompactTokenFigure(*report.MeasuredTokens), report.MeasuredRows)
	}
	for _, phase := range report.Phases {
		fmt.Fprintf(&b, "  Phase %d", phase.Phase)
		if strings.TrimSpace(phase.PhaseName) != "" {
			fmt.Fprintf(&b, " (%s)", phase.PhaseName)
		}
		if phase.MeasuredTokens == nil {
			fmt.Fprintf(&b, "  %s  not reported\n", spendNotReportedFigure)
		} else {
			fmt.Fprintf(&b, "  %s  measured\n", spendCompactTokenFigure(*phase.MeasuredTokens))
		}
		for _, worker := range phase.Workers {
			name := strings.TrimSpace(strings.Join([]string{worker.Role, worker.Name}, " "))
			if worker.Reported && worker.Tokens != nil {
				fmt.Fprintf(&b, "    %s  %s  measured\n", emptyFallback(name, "worker"), spendCompactTokenFigure(*worker.Tokens))
			} else {
				fmt.Fprintf(&b, "    %s  %s  not reported\n", emptyFallback(name, "worker"), spendNotReportedFigure)
			}
		}
	}
	if report.UnreportedRows > 0 {
		fmt.Fprintf(&b, "(%d worker run(s) did not report usage; no estimate was substituted.)\n", report.UnreportedRows)
	}
	return b.String()
}
