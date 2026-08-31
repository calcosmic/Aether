package cmd

import (
	"fmt"
	"strings"
)

// autopilotDisposition is the mode-specific action attached to a trigger.
// Stable typed values, rather than rendered prose, are the only inputs future
// run-loop enforcement may use.
type autopilotDisposition string

const (
	autopilotDispositionStop             autopilotDisposition = "stop"
	autopilotDispositionQueueAndContinue autopilotDisposition = "queue_and_continue"
	autopilotDispositionPause            autopilotDisposition = "pause"
	autopilotDispositionNormalStop       autopilotDisposition = "normal_stop"
)

// autopilotTriggerCode is a durable policy identifier. These values may be
// persisted and rendered, so changing one is a compatibility change.
type autopilotTriggerCode string

const (
	autopilotTriggerDeterministicVerificationFailed autopilotTriggerCode = "deterministic_verification_failed"
	autopilotTriggerAuditorScoreBelowFloor          autopilotTriggerCode = "auditor_score_below_floor"
	autopilotTriggerCriticalReviewFinding           autopilotTriggerCode = "critical_review_finding"
	autopilotTriggerBlockerCountIncreased           autopilotTriggerCode = "blocker_count_increased"
	autopilotTriggerBlockerEscalated                autopilotTriggerCode = "blocker_escalated"
	autopilotTriggerColonyNotRunnable               autopilotTriggerCode = "colony_not_runnable"
	autopilotTriggerProviderUnavailable             autopilotTriggerCode = "provider_unavailable"
	autopilotTriggerRuntimeVerificationNeeded       autopilotTriggerCode = "runtime_verification_needed"
	autopilotTriggerVisualCheckpointNeeded          autopilotTriggerCode = "visual_checkpoint_needed"
	autopilotTriggerReplanDue                       autopilotTriggerCode = "replan_due"
	autopilotTriggerCancelled                       autopilotTriggerCode = "cancelled"
	autopilotTriggerWorkerTimeout                   autopilotTriggerCode = "worker_timeout"
	autopilotTriggerMaxPhasesReached                autopilotTriggerCode = "max_phases_reached"
	autopilotTriggerColonyComplete                  autopilotTriggerCode = "colony_complete"
)

// autopilotTriggerSpec describes one overnight stop, queued checkpoint, or
// ordinary ending. Detection is explanatory only; evidence evaluation is
// added by the stage-specific integration plans.
type autopilotTriggerSpec struct {
	Code                   autopilotTriggerCode `json:"code"`
	Label                  string               `json:"label"`
	Detection              string               `json:"detection"`
	NextActionTemplate     string               `json:"next_action_template"`
	HeadlessDisposition    autopilotDisposition `json:"headless_disposition"`
	InteractiveDisposition autopilotDisposition `json:"interactive_disposition"`
}

// autopilotTriggerEvaluation is the typed bridge between stage evidence and
// policy enforcement. Renderers may explain it, but must never parse their
// own prose to recreate it.
type autopilotTriggerEvaluation struct {
	Spec     autopilotTriggerSpec   `json:"spec"`
	Active   bool                   `json:"active"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

// autopilotTriggerSpecs is the one ordered catalogue used by dry-run today
// and by live enforcement/reporting in later plans.
func autopilotTriggerSpecs() []autopilotTriggerSpec {
	specs := []autopilotTriggerSpec{
		{
			Code:                   autopilotTriggerDeterministicVerificationFailed,
			Label:                  "Deterministic verification failed",
			Detection:              "The deterministic verification floor still fails after its single bounded repair attempt.",
			NextActionTemplate:     "aether continue",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerAuditorScoreBelowFloor,
			Label:                  "Auditor score below 60",
			Detection:              "An Auditor that actually ran completed with an overall score below 60.",
			NextActionTemplate:     "aether continue",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerCriticalReviewFinding,
			Label:                  "Critical review finding",
			Detection:              "Current reviewer, audit, or security evidence contains a Critical finding.",
			NextActionTemplate:     "aether flags",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerBlockerCountIncreased,
			Label:                  "Blocker count increased",
			Detection:              "The unresolved blocker count after a step is greater than its live baseline.",
			NextActionTemplate:     "aether unblock",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerBlockerEscalated,
			Label:                  "Blocker escalation added",
			Detection:              "An unresolved blocker is newly marked with the escalation source after the baseline snapshot.",
			NextActionTemplate:     "aether unblock",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerColonyNotRunnable,
			Label:                  "Colony state is not runnable",
			Detection:              "Colony state is missing, corrupt, or cannot validly enter build or continue.",
			NextActionTemplate:     "aether status",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerProviderUnavailable,
			Label:                  "Required worker provider unavailable",
			Detection:              "The required provider is still unavailable after its bounded readiness policy, or immediately reports an auth or binary failure.",
			NextActionTemplate:     "aether run",
			HeadlessDisposition:    autopilotDispositionStop,
			InteractiveDisposition: autopilotDispositionStop,
		},
		{
			Code:                   autopilotTriggerRuntimeVerificationNeeded,
			Label:                  "Runtime verification needed",
			Detection:              "A current criterion needs hands-on owner verification that the program cannot perform.",
			NextActionTemplate:     "aether decision-answer {decision_id} {answer}",
			HeadlessDisposition:    autopilotDispositionQueueAndContinue,
			InteractiveDisposition: autopilotDispositionPause,
		},
		{
			Code:                   autopilotTriggerVisualCheckpointNeeded,
			Label:                  "Visual checkpoint needed",
			Detection:              "Current build claims include user-interface work that needs an owner's visual judgement.",
			NextActionTemplate:     "aether decision-answer {decision_id} {answer}",
			HeadlessDisposition:    autopilotDispositionQueueAndContinue,
			InteractiveDisposition: autopilotDispositionPause,
		},
		{
			Code:                   autopilotTriggerReplanDue,
			Label:                  "Replan due",
			Detection:              "The configured phase interval is reached with at least one unique confirmed lesson since the last successful plan.",
			NextActionTemplate:     "aether plan",
			HeadlessDisposition:    autopilotDispositionQueueAndContinue,
			InteractiveDisposition: autopilotDispositionPause,
		},
		{
			Code:                   autopilotTriggerCancelled,
			Label:                  "Run cancelled",
			Detection:              "The operator or parent process cancelled this run after durable progress was preserved.",
			NextActionTemplate:     "aether run",
			HeadlessDisposition:    autopilotDispositionNormalStop,
			InteractiveDisposition: autopilotDispositionNormalStop,
		},
		{
			Code:                   autopilotTriggerWorkerTimeout,
			Label:                  "Worker timeout",
			Detection:              "A bounded worker timeout ended the current unfinished work after durable progress was preserved.",
			NextActionTemplate:     "aether run",
			HeadlessDisposition:    autopilotDispositionNormalStop,
			InteractiveDisposition: autopilotDispositionNormalStop,
		},
		{
			Code:                   autopilotTriggerMaxPhasesReached,
			Label:                  "Maximum phases reached",
			Detection:              "The explicit --max-phases limit for this invocation has been reached.",
			NextActionTemplate:     "aether run",
			HeadlessDisposition:    autopilotDispositionNormalStop,
			InteractiveDisposition: autopilotDispositionNormalStop,
		},
		{
			Code:                   autopilotTriggerColonyComplete,
			Label:                  "Colony complete",
			Detection:              "Every planned phase has completed; autopilot hands the colony back to the owner before seal.",
			NextActionTemplate:     "aether seal",
			HeadlessDisposition:    autopilotDispositionNormalStop,
			InteractiveDisposition: autopilotDispositionNormalStop,
		},
	}

	if err := validateAutopilotTriggerSpecs(specs); err != nil {
		panic(fmt.Sprintf("invalid canonical autopilot trigger catalogue: %v", err))
	}
	return specs
}

func validAutopilotDisposition(disposition autopilotDisposition) bool {
	switch disposition {
	case autopilotDispositionStop,
		autopilotDispositionQueueAndContinue,
		autopilotDispositionPause,
		autopilotDispositionNormalStop:
		return true
	default:
		return false
	}
}

func validateAutopilotTriggerSpecs(specs []autopilotTriggerSpec) error {
	if len(specs) == 0 {
		return fmt.Errorf("trigger catalogue is empty")
	}

	seen := make(map[autopilotTriggerCode]struct{}, len(specs))
	for i, spec := range specs {
		code := autopilotTriggerCode(strings.TrimSpace(string(spec.Code)))
		if code == "" {
			return fmt.Errorf("trigger row %d has a blank code", i)
		}
		if code != spec.Code {
			return fmt.Errorf("trigger code %q has surrounding whitespace", spec.Code)
		}
		if _, exists := seen[code]; exists {
			return fmt.Errorf("duplicate trigger code %q", code)
		}
		seen[code] = struct{}{}

		if strings.TrimSpace(spec.Label) == "" {
			return fmt.Errorf("trigger %q has a blank label", code)
		}
		if strings.TrimSpace(spec.Detection) == "" {
			return fmt.Errorf("trigger %q has blank detection text", code)
		}
		if strings.TrimSpace(spec.NextActionTemplate) == "" {
			return fmt.Errorf("trigger %q has a blank next-action template", code)
		}
		if !validAutopilotDisposition(spec.HeadlessDisposition) {
			return fmt.Errorf("trigger %q has unknown headless disposition %q", code, spec.HeadlessDisposition)
		}
		if !validAutopilotDisposition(spec.InteractiveDisposition) {
			return fmt.Errorf("trigger %q has unknown interactive disposition %q", code, spec.InteractiveDisposition)
		}
	}
	return nil
}

func autopilotTriggerSpecByCode(code autopilotTriggerCode) (autopilotTriggerSpec, bool) {
	for _, spec := range autopilotTriggerSpecs() {
		if spec.Code == code {
			return spec, true
		}
	}
	return autopilotTriggerSpec{}, false
}
