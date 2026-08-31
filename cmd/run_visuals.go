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
	goal := "(no goal recorded)"
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		goal = strings.TrimSpace(*state.Goal)
	}
	maxLabel := "all"
	if opts.MaxPhases > 0 {
		maxLabel = fmt.Sprintf("%d", opts.MaxPhases)
	}
	var b strings.Builder
	b.WriteString("━━━ 🤖 " + spacedTitle("Autopilot Engaged") + " ━━━\n")
	b.WriteString(fmt.Sprintf("Goal: %s | Phase %d of %d | Max: %s", goal, state.CurrentPhase, len(state.Plan.Phases), maxLabel))
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

func renderAutopilotComplete(phasesCompleted int) string {
	var b strings.Builder
	b.WriteString("━━━ ✅ " + spacedTitle("Autopilot Complete") + " ━━━\n")
	if phasesCompleted == 1 {
		b.WriteString("1 phase built, verified, and advanced.")
	} else {
		b.WriteString(fmt.Sprintf("%d phases built, verified, and advanced.", phasesCompleted))
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
