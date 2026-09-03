package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Streamed autopilot narration — the classic v5.4.0 AUTOPILOT experience
// restored on the Go loop. Every function here renders append-only scrollback
// per the one-terminal decision recorded in
// .planning/decisions/SEE-03-one-terminal-streaming.md: the run narrates
// itself in the terminal the user is already in, one block per event, no
// repainting. The loop emits these through emitVisualProgress, so JSON mode
// stays a clean machine surface.

func renderRunEngageLine(state colony.ColonyState, opts runCompatibilityOptions) string {
	_ = opts // The accepted range, not a presentation-only cap, defines this card.
	facts := lifecycleFactsFromStateSnapshot(state, false, autopilotNow())
	return renderAutopilotOperatingContract(buildAutopilotPreflight(facts))
}

func renderAutopilotPreflightRefusal(preflight AutopilotPreflight) string {
	var b strings.Builder
	if preflight.Paused {
		b.WriteString("⏸ Autopilot paused\n")
		fmt.Fprintf(&b, "Because: %s.\n", emptyFallback(strings.TrimSuffix(strings.TrimSpace(preflight.Diagnostic), "."), "authoritative colony state cannot be read safely"))
		b.WriteString("State: unchanged.\n")
		fmt.Fprintf(&b, "Next: %s", emptyFallback(strings.TrimSpace(preflight.Next), "/ant-status"))
		return b.String()
	}
	b.WriteString("⛔ Autopilot did not start\n")
	fmt.Fprintf(&b, "Missing: %s.\n", emptyFallback(strings.TrimSpace(preflight.Missing), "an accepted plan"))
	b.WriteString("State: unchanged.\n")
	fmt.Fprintf(&b, "Next: %s", emptyFallback(strings.TrimSpace(preflight.Next), "/ant-status"))
	return b.String()
}

func renderAutopilotOperatingContract(preflight AutopilotPreflight) string {
	pheromones := "none"
	if len(preflight.ActivePheromones) > 0 {
		pheromones = strings.Join(preflight.ActivePheromones, "; ")
	}
	var b strings.Builder
	b.WriteString("━━ ⚡ A U T O P I L O T ━━\n")
	fmt.Fprintf(&b, "Goal: %s\n", emptyFallback(strings.TrimSpace(preflight.Goal), "Not recorded"))
	fmt.Fprintf(&b, "Range: Phase %d through Phase %d\n", preflight.FirstPhase, preflight.LastPhase)
	fmt.Fprintf(&b, "Active pheromones: %s\n", pheromones)
	b.WriteString("May revise: tasks, dependencies, sequencing, and implementation details when evidence requires it.\n")
	b.WriteString("Will pause before changing: goal, promised behavior, scope, risk authority, or acceptance criteria.\n")
	b.WriteString("Also pauses for: safety failure, corrupt state, missing authority, a material owner decision, or an invalidating failed dependency.\n")
	b.WriteString("Starting now.")
	return b.String()
}

func renderRunPhaseHeader(phase *colony.Phase, totalPhases int) string {
	if phase == nil {
		return ""
	}
	return fmt.Sprintf("━━━ 🐜 Phase %d of %d: %s ━━━", phase.ID, totalPhases, phase.Name)
}

// renderRunPhaseAdvancement is the classic between-phase ceremony: what just
// finished, what the colony learned, and the momentum ticker.
func renderRunPhaseAdvancement(phase colony.Phase, continueResult map[string]interface{}, phasesCompleted, totalPhases int) string {
	var b strings.Builder
	b.WriteString("━━━ ➡️ " + spacedTitle("Phase Advancement") + " ━━━\n")
	b.WriteString(fmt.Sprintf("✅ Phase %d: %s — COMPLETED\n", phase.ID, phase.Name))

	if consolidation, ok := continueResult["consolidation"].(map[string]interface{}); ok {
		if n := intValue(consolidation["learnings"]); n > 0 {
			b.WriteString(fmt.Sprintf("🧠 Learnings extracted: %d\n", n))
		}
		if n := intValue(consolidation["instincts"]); n > 0 {
			b.WriteString(fmt.Sprintf("🐜 Instincts updated: %d\n", n))
		}
	}

	nextName := strings.TrimSpace(stringValue(continueResult["next_phase_name"]))
	nextID := intValue(continueResult["next_phase"])
	if nextID > 0 {
		if nextName != "" {
			b.WriteString(fmt.Sprintf("--- Autopilot: Phase %d done -> Phase %d: %s (%d/%d) ---", phase.ID, nextID, nextName, phasesCompleted, totalPhases))
		} else {
			b.WriteString(fmt.Sprintf("--- Autopilot: Phase %d done -> Phase %d (%d/%d) ---", phase.ID, nextID, phasesCompleted, totalPhases))
		}
	} else {
		b.WriteString(fmt.Sprintf("--- Autopilot: Phase %d done (%d/%d) ---", phase.ID, phasesCompleted, totalPhases))
	}
	return b.String()
}

func renderRunReplanBanner(phasesCompleted, interval, lessonCount int) string {
	var b strings.Builder
	b.WriteString("━━━ 🔄 " + spacedTitle("Replan Suggested") + " ━━━\n")
	b.WriteString(fmt.Sprintf("%d phase(s) completed since the last checkpoint (interval: every %d), with %d unique evidence-confirmed lesson(s) since the active plan revision.\n", phasesCompleted, interval, lessonCount))
	b.WriteString("Review the plan with `aether plan`, or run `aether run --continue` to keep going.")
	return b.String()
}

func renderRunReplanQueued(decision PendingDecision) string {
	return fmt.Sprintf(
		"📝 Replan note %s queued for plan revision %s (%d confirmed lesson(s), checkpoints %d-%d). Autopilot continues.",
		decision.ID,
		decision.PlanRevisionID,
		decision.LessonCount,
		decision.FirstCheckpointPhase,
		decision.LatestCheckpointPhase,
	)
}

func renderRunCheckpointQueued(code autopilotTriggerCode, checkpoint autopilotCheckpointReference) string {
	return fmt.Sprintf(
		"📝 %s checkpoint %s queued for phase %d. Autopilot continues; owner recovery: `%s`",
		strings.ReplaceAll(string(code), "_", " "),
		checkpoint.ID,
		checkpoint.Phase,
		emptyFallback(checkpoint.RecoveryCommand, "aether decision-list"),
	)
}

func renderRunBlockerBaseline(snapshot blockerSnapshot) string {
	if snapshot.Count == 1 {
		return "ℹ️ 1 existing blocker recorded at this stage boundary; autopilot will continue unless the count grows or an escalation exists."
	}
	return fmt.Sprintf("ℹ️ %d existing blockers recorded at this stage boundary; autopilot will continue unless the count grows or an escalation exists.", snapshot.Count)
}

func renderRunTypedDecision(decision autopilotRunDecision) string {
	title := "Autopilot Stopped"
	icon := "⛔"
	guidance := fmt.Sprintf("Next step: `%s`", emptyFallback(decision.Next, "aether status"))
	switch decision.Disposition {
	case autopilotDispositionPause:
		title = "Autopilot Paused"
		icon = "⏸"
	case autopilotDispositionNormalStop:
		title = "Autopilot Finished This Run"
		icon = "⏹"
	}
	return renderDecisionBlock(icon, title, humanizeAutopilotPauseReason(string(decision.Code)), guidance)
}

func renderAutopilotComplete(phasesCompleted int) string {
	var b strings.Builder
	b.WriteString("━━━ ✅ " + spacedTitle("Autopilot Complete") + " ━━━\n")
	if phasesCompleted == 1 {
		b.WriteString("1 phase built and verified.\n")
	} else {
		b.WriteString(fmt.Sprintf("%d phases built and verified.\n", phasesCompleted))
	}
	b.WriteString("Sealing remains an explicit owner action. Next: `aether seal`.")
	return b.String()
}

func renderAutopilotRepairReceipt(receipt autopilotRepairReceipt) string {
	var b strings.Builder
	b.WriteString("━━━ 🔧 " + spacedTitle("Bounded Repair") + " ━━━\n")
	fmt.Fprintf(&b, "Receipt: %s | Phase %d | Attempt: %s\n", receipt.ID, receipt.Phase, emptyFallback(receipt.Attempt, "not recorded"))
	fmt.Fprintf(&b, "Failing check: %s\n", emptyFallback(receipt.Check, "not recorded"))
	fmt.Fprintf(&b, "Permitted scope: %s\n", emptyFallback(strings.Join(receipt.PermittedScope, ", "), "none"))
	fmt.Fprintf(&b, "Action: %s\n", emptyFallback(receipt.PlannedAction, "not recorded"))
	fmt.Fprintf(&b, "Verification: %s — %s\n", emptyFallback(receipt.Verification.Check, receipt.Check), repairVerificationLabel(receipt.Verification.Passed))
	fmt.Fprintf(&b, "Remaining repair budget: %d", receipt.BudgetRemaining)
	return b.String()
}

func repairVerificationLabel(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}

func renderAutopilotRepairReport(value interface{}) string {
	var report autopilotRepairReport
	switch typed := value.(type) {
	case autopilotRepairReport:
		report = typed
	case *autopilotRepairReport:
		if typed == nil {
			return ""
		}
		report = *typed
	default:
		return ""
	}
	if report.Attempts == 0 && len(report.Debt) == 0 && len(report.Blockers) == 0 && len(report.ContinuedPaths) == 0 && len(report.SkippedPaths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nRepair and debt\n")
	fmt.Fprintf(&b, "Attempts: %d | Remaining budget: %d | Exhausted: %t\n", report.Attempts, report.RemainingBudget, report.BudgetExhausted)
	for _, receipt := range report.Receipts {
		fmt.Fprintf(&b, "  - %s: phase %d %s — %s\n", receipt.ID, receipt.Phase, receipt.Check, receipt.Status)
	}
	for _, debt := range report.Debt {
		fmt.Fprintf(&b, "  - Debt: %s\n", debt.Summary)
	}
	for _, blocker := range report.Blockers {
		fmt.Fprintf(&b, "  - Blocker: %s\n", blocker.Summary)
	}
	if len(report.ContinuedPaths) > 0 {
		fmt.Fprintf(&b, "Continued independent paths: %s\n", strings.Join(report.ContinuedPaths, ", "))
	}
	if len(report.SkippedPaths) > 0 {
		fmt.Fprintf(&b, "Stopped affected paths: %s\n", strings.Join(report.SkippedPaths, ", "))
	}
	return b.String()
}

// renderRunPauseBlock is the visually distinct decision block for a pause: the
// run stopped on purpose, here is why in plain English, here is what to do.
func renderRunPauseBlock(reason, next string) string {
	guidance := "Fix the issue, then run `aether run` to resume."
	if strings.TrimSpace(next) != "" {
		guidance = fmt.Sprintf("Fix the issue, then run `aether run` to resume. Suggested first step: `%s`", next)
	}
	return renderDecisionBlock("⏸", "Autopilot Paused", humanizeAutopilotPauseReason(reason), guidance)
}

// humanizeAutopilotPauseReason translates a checkAutopilotPauseConditions
// reason code into a sentence a person can act on without reading source.
func humanizeAutopilotPauseReason(reason string) string {
	switch {
	case reason == string(autopilotTriggerDeterministicVerificationFailed):
		return "The current phase did not clear deterministic verification, so continuing would be unsafe."
	case reason == string(autopilotTriggerAuditorScoreBelowFloor):
		return "The Auditor scored the current result below the overnight safety floor of 60."
	case reason == string(autopilotTriggerCriticalReviewFinding):
		return "The current review produced a Critical finding."
	case reason == string(autopilotTriggerBlockerCountIncreased):
		return "The live blocker count increased during this stage."
	case reason == string(autopilotTriggerBlockerEscalated):
		return "A blocker escalation is unresolved at this stage boundary."
	case reason == string(autopilotTriggerColonyNotRunnable):
		return "The colony cannot safely enter its next build or verification step."
	case reason == string(autopilotTriggerMissingAuthority):
		return "The proposed change needs owner authority before it can be applied."
	case reason == string(autopilotTriggerProviderUnavailable):
		return "The required worker provider is unavailable; the current phase remains ready to resume."
	case reason == string(autopilotTriggerRuntimeVerificationNeeded):
		return "The current phase needs hands-on owner verification."
	case reason == string(autopilotTriggerVisualCheckpointNeeded):
		return "The current phase needs an owner to inspect its user-interface changes."
	case reason == string(autopilotTriggerReplanDue):
		return "The interval was reached with confirmed lessons that may change the remaining plan."
	case reason == string(autopilotTriggerCancelled):
		return "The run was cancelled after preserving its current phase state."
	case reason == string(autopilotTriggerWorkerTimeout):
		return "A bounded worker timeout ended this run; the unfinished phase is ready to resume."
	case reason == string(autopilotTriggerMaxPhasesReached):
		return "This invocation reached its requested phase limit."
	case reason == string(autopilotTriggerColonyComplete):
		return "Every planned phase is complete; sealing remains an explicit owner action."
	case reason == "blocked":
		return "Verification could not confirm this phase's work."
	case strings.HasPrefix(reason, "active_blockers:"):
		count := strings.TrimPrefix(reason, "active_blockers:")
		return fmt.Sprintf("%s unresolved blocker(s) need a decision before the colony keeps building.", count)
	case reason == "test_failures":
		return "Test failures were signalled during verification."
	case strings.HasPrefix(reason, "gate_failure:"):
		return fmt.Sprintf("The %q quality gate failed.", strings.TrimPrefix(reason, "gate_failure:"))
	case reason == "critical_chaos_findings":
		return "Resilience testing logged a critical finding."
	case reason == "uncommitted_changes":
		return "Uncommitted changes were flagged mid-run."
	default:
		return fmt.Sprintf("Pause condition triggered: %s.", reason)
	}
}

// autopilotPauseNextCommand suggests the most direct next step per condition.
func autopilotPauseNextCommand(reason string) string {
	switch {
	case strings.HasPrefix(reason, "active_blockers:"):
		return "aether unblock"
	case reason == "test_failures", strings.HasPrefix(reason, "gate_failure:"):
		return "aether continue"
	case reason == "critical_chaos_findings":
		return "aether flags"
	case reason == "uncommitted_changes":
		return "aether status"
	default:
		return "aether status"
	}
}

// pauseAutopilotRun is the single exit for a mid-run pause: it records the
// paused status with its reason, queues a pending decision under --headless,
// streams the pause block, and shapes the result envelope.
func pauseAutopilotRun(state colony.ColonyState, opts runCompatibilityOptions, steps []map[string]interface{}, phasesCompleted int, reason string) map[string]interface{} {
	_ = syncRunAutopilotState(state, opts, "paused", reason)
	next := autopilotPauseNextCommand(reason)
	if opts.Headless {
		queueAutopilotPauseDecision(reason, state.CurrentPhase)
	}
	emitVisualProgress(renderRunPauseBlock(reason, next))
	result := buildRunExecutionResult(state, opts, steps, phasesCompleted, "paused:"+reason, next)
	result["pause_reason"] = reason
	return result
}

// queueAutopilotPauseDecision records a headless pause on the pending-decision
// queue — the classic headless behaviour: the run still stops, but the reason
// is waiting for review instead of lost in scrollback nobody watched.
func queueAutopilotPauseDecision(reason string, phase int) {
	if store == nil {
		return
	}
	description := fmt.Sprintf("Autopilot paused: %s", humanizeAutopilotPauseReason(reason))
	file := loadPendingDecisionFile()
	for _, existing := range file.Decisions {
		if !existing.Resolved && existing.Type == "autopilot_pause" && existing.Description == description {
			return
		}
	}
	decision := PendingDecision{
		ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
		Type:        "autopilot_pause",
		Description: description,
		Source:      "autopilot",
		Resolved:    false,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if phase > 0 {
		decision.Phase = &phase
	}
	file.Decisions = append(file.Decisions, decision)
	_ = store.SaveJSON(pendingDecisionsFile, file)
}

// renderRunDryRunTriggerCatalogue renders the structured policy rows carried
// by the dry-run result. It explains typed control data; no caller parses this
// prose to decide what the run does.
func renderRunDryRunTriggerCatalogue(value interface{}) string {
	specs := autopilotTriggerSpecsFromDryRunValue(value)
	var b strings.Builder
	b.WriteString("\nPause Triggers and Normal Stops (canonical overnight contract)\n")
	for _, spec := range specs {
		b.WriteString(fmt.Sprintf("  %s — %s\n", spec.Code, spec.Label))
		b.WriteString(fmt.Sprintf("    Detects: %s\n", spec.Detection))
		b.WriteString(fmt.Sprintf("    Headless: %s | Interactive: %s\n", spec.HeadlessDisposition, spec.InteractiveDisposition))
		b.WriteString(fmt.Sprintf("    Next: `%s`\n", spec.NextActionTemplate))
	}
	return b.String()
}

func autopilotTriggerSpecsFromDryRunValue(value interface{}) []autopilotTriggerSpec {
	switch rows := value.(type) {
	case []autopilotTriggerSpec:
		return rows
	case []map[string]interface{}:
		return autopilotTriggerSpecsFromMaps(rows)
	case []interface{}:
		maps := make([]map[string]interface{}, 0, len(rows))
		for _, raw := range rows {
			row, ok := raw.(map[string]interface{})
			if !ok {
				return nil
			}
			maps = append(maps, row)
		}
		return autopilotTriggerSpecsFromMaps(maps)
	default:
		return nil
	}
}

func autopilotTriggerSpecsFromMaps(rows []map[string]interface{}) []autopilotTriggerSpec {
	specs := make([]autopilotTriggerSpec, 0, len(rows))
	for _, row := range rows {
		specs = append(specs, autopilotTriggerSpec{
			Code:                   autopilotTriggerCode(stringValue(row["code"])),
			Label:                  stringValue(row["label"]),
			Detection:              stringValue(row["detection"]),
			NextActionTemplate:     stringValue(row["next_action_template"]),
			HeadlessDisposition:    autopilotDisposition(stringValue(row["headless_disposition"])),
			InteractiveDisposition: autopilotDisposition(stringValue(row["interactive_disposition"])),
		})
	}
	if err := validateAutopilotTriggerSpecs(specs); err != nil {
		return nil
	}
	return specs
}

// autopilotPauseTrigger is a compatibility projection for older focused tests.
// It contains no independent policy list: every row is derived from the typed
// canonical catalogue above.
type autopilotPauseTrigger struct {
	Condition string
	Meaning   string
}

func autopilotPauseTriggerCatalog() []autopilotPauseTrigger {
	specs := autopilotTriggerSpecs()
	triggers := make([]autopilotPauseTrigger, 0, len(specs))
	for _, spec := range specs {
		triggers = append(triggers, autopilotPauseTrigger{
			Condition: string(spec.Code),
			Meaning:   spec.Detection,
		})
	}
	return triggers
}
