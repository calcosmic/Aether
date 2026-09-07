package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
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
	autopilotTriggerMissingAuthority                autopilotTriggerCode = "missing_authority"
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

// AutopilotPreflight is the immutable result of checking whether run may
// begin. It is deliberately built from LifecycleFacts: invalid entry can be
// rendered without invoking a compatibility loader that repairs state or
// writes welcome/session artifacts.
type AutopilotPreflight struct {
	Valid              bool                        `json:"valid"`
	Completed          bool                        `json:"completed,omitempty"`
	Paused             bool                        `json:"paused,omitempty"`
	PauseClass         string                      `json:"pause_class,omitempty"`
	Diagnostic         string                      `json:"diagnostic,omitempty"`
	Missing            string                      `json:"missing,omitempty"`
	Goal               string                      `json:"goal,omitempty"`
	FirstPhase         int                         `json:"first_phase,omitempty"`
	LastPhase          int                         `json:"last_phase,omitempty"`
	RemainingPhases    []int                       `json:"remaining_phases,omitempty"`
	ActivePheromones   []string                    `json:"active_pheromones"`
	ProjectionRevision string                      `json:"projection_revision"`
	CapturedAt         string                      `json:"captured_at"`
	Next               string                      `json:"next"`
	OutcomeKind        colony.OutcomeKind          `json:"outcome_kind"`
	StateEffect        colony.LifecycleStateEffect `json:"state_effect"`
	PlanAuthority      planAuthorityDecision       `json:"plan_authority"`
	Projection         LifecycleProjection         `json:"projection"`
}

// buildAutopilotPreflight verifies repository-backed authority read-only before
// delegating to the pure policy core. A confirmed, non-empty goal is the
// initialization proof available in the lifecycle snapshot; current plans
// additionally require exact accepted candidate artifacts.
func buildAutopilotPreflight(facts LifecycleFacts) AutopilotPreflight {
	// In-memory callers without a repository root are presentation helpers and
	// historical pure tests; the real run entry always supplies Root and takes
	// the artifact-backed path below. This avoids re-reading or inventing the
	// timeline merely to render an already-started run card.
	if strings.TrimSpace(facts.Root) == "" {
		return buildAutopilotPreflightCore(facts, nil)
	}
	if migration, err := migratePlanningState(facts.Root, facts.State.Value); err != nil {
		decision := refusePlanAuthority(planAuthorityDecision{}, planAuthorityRefusalLegacyInvalid, "aether plan", err.Error())
		return buildAutopilotPreflightCore(facts, &decision)
	} else if migration.Changed {
		facts.State.Value = migration.State
		facts.Planning.Value.AcceptancePolicy = migration.State.Plan.AcceptancePolicy
		facts.Planning.Value.LegacyUnbound = true
		facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingLegacyUnbound
	}
	bindings := loadPlanAuthorityVerifiedBindings(facts.Root, facts)
	return buildAutopilotPreflightWithAuthority(facts, bindings)
}

// buildAutopilotPreflightWithAuthority is the pure parity seam used by run
// after read-only artifact verification and by cross-surface policy tests.
func buildAutopilotPreflightWithAuthority(facts LifecycleFacts, bindings planAuthorityVerifiedBindings) AutopilotPreflight {
	decision := validateAcceptedPlanAuthority(facts, bindings)
	return buildAutopilotPreflightCore(facts, &decision)
}

func buildAutopilotPreflightCore(facts LifecycleFacts, authority *planAuthorityDecision) AutopilotPreflight {
	projection := projectLifecycle(facts, LifecycleViewFocused, detectPlatform())
	projection.Command = "run"
	preflight := AutopilotPreflight{
		ActivePheromones:   []string{},
		ProjectionRevision: projection.ProjectionRevision,
		CapturedAt:         facts.CapturedAt.UTC().Format(time.RFC3339Nano),
		OutcomeKind:        colony.OutcomeKindRefused,
		StateEffect:        colony.LifecycleStateEffectNone,
		Projection:         projection,
	}
	if authority != nil {
		preflight.PlanAuthority = *authority
	}

	goal := strings.TrimSpace(facts.Identity.Value.Goal)
	if goal == "" && facts.State.Value.Goal != nil {
		goal = strings.TrimSpace(*facts.State.Value.Goal)
	}
	stateUnavailableIsMissing := facts.State.Source.Provenance == LifecycleFactUnavailable &&
		strings.Contains(strings.ToLower(facts.State.Source.Diagnostic), "store is not initialized")
	if facts.State.Source.Provenance == LifecycleFactMalformed ||
		(facts.State.Source.Provenance == LifecycleFactUnavailable && !stateUnavailableIsMissing) {
		preflight.Paused = true
		preflight.PauseClass = "corrupt_state"
		preflight.Diagnostic = emptyFallback(strings.TrimSpace(facts.State.Source.Diagnostic), "authoritative colony state cannot be read safely")
		preflight.Next = "/ant-status"
		preflight.OutcomeKind = colony.OutcomeKindPaused
		return preflight
	}
	if facts.State.Source.Provenance != LifecycleFactConfirmed || goal == "" {
		preflight.Missing = "an initialized colony"
		preflight.Next = `/ant-init "goal"`
		return preflight
	}
	preflight.Goal = goal

	phases := facts.Progress.Value.Phases
	if len(phases) == 0 {
		phases = facts.State.Value.Plan.Phases
	}
	if len(phases) > 0 && !orderedAutopilotPhases(phases) {
		preflight.Paused = true
		preflight.PauseClass = "corrupt_state"
		preflight.Diagnostic = "accepted phases are not in a unique ascending order"
		preflight.Next = "/ant-status"
		preflight.OutcomeKind = colony.OutcomeKindPaused
		return preflight
	}
	if len(phases) == 0 {
		if authority != nil && authority.RefusalCode == planAuthorityRefusalCandidateNotAccepted {
			preflight.Missing = "accepted plan authority"
			preflight.Diagnostic = strings.TrimSpace(authority.Diagnostic)
			preflight.Next = authority.RecoveryCommand
			return preflight
		}
		preflight.Missing = "an accepted plan"
		preflight.Next = "/ant-plan"
		return preflight
	}
	if authority != nil && !authority.Eligible {
		preflight.Missing = "accepted plan authority"
		preflight.Diagnostic = strings.TrimSpace(authority.Diagnostic)
		preflight.Next = authority.RecoveryCommand
		return preflight
	}

	for _, phase := range phases {
		if phase.Status == colony.PhaseCompleted {
			continue
		}
		preflight.RemainingPhases = append(preflight.RemainingPhases, phase.ID)
	}
	if len(preflight.RemainingPhases) == 0 {
		preflight.Completed = true
		preflight.Next = "/ant-seal"
		preflight.OutcomeKind = colony.OutcomeKindCompleted
		return preflight
	}

	preflight.Valid = true
	preflight.FirstPhase = preflight.RemainingPhases[0]
	preflight.LastPhase = preflight.RemainingPhases[len(preflight.RemainingPhases)-1]
	preflight.Next = "/ant-run"
	preflight.OutcomeKind = colony.OutcomeKindInProgress
	for _, signal := range facts.Signals.Value {
		if !signal.Active {
			continue
		}
		typeName := strings.ToUpper(strings.TrimSpace(signal.Type))
		if typeName != "FOCUS" && typeName != "FEEDBACK" && typeName != "REDIRECT" {
			continue
		}
		content := strings.TrimSpace(extractContentText(signal.Content))
		if content == "" {
			content = "(no content recorded)"
		}
		preflight.ActivePheromones = append(preflight.ActivePheromones, typeName+": "+content)
	}
	return preflight
}

func orderedAutopilotPhases(phases []colony.Phase) bool {
	if len(phases) == 0 {
		return false
	}
	previous := 0
	for _, phase := range phases {
		if phase.ID <= 0 || phase.ID <= previous {
			return false
		}
		previous = phase.ID
	}
	return true
}

type autopilotAuthorityTarget string

const (
	autopilotAuthorityTasks            autopilotAuthorityTarget = "tasks"
	autopilotAuthorityDependencies     autopilotAuthorityTarget = "dependencies"
	autopilotAuthoritySequencing       autopilotAuthorityTarget = "sequencing"
	autopilotAuthorityImplementation   autopilotAuthorityTarget = "implementation_details"
	autopilotAuthorityGoal             autopilotAuthorityTarget = "goal"
	autopilotAuthorityPromisedBehavior autopilotAuthorityTarget = "promised_behavior"
	autopilotAuthorityScope            autopilotAuthorityTarget = "scope"
	autopilotAuthorityRisk             autopilotAuthorityTarget = "risk_authority"
	autopilotAuthorityAcceptance       autopilotAuthorityTarget = "acceptance_criteria"
)

type autopilotAuthorityProposal struct {
	Target autopilotAuthorityTarget `json:"target"`
	Before string                   `json:"before,omitempty"`
	After  string                   `json:"after,omitempty"`
	Reason string                   `json:"reason,omitempty"`
}

type autopilotAuthorityDecision struct {
	Allowed     bool                 `json:"allowed"`
	Code        autopilotTriggerCode `json:"code,omitempty"`
	Disposition autopilotDisposition `json:"disposition,omitempty"`
	Next        string               `json:"next,omitempty"`
	Reason      string               `json:"reason"`
}

func evaluateAutopilotAuthorityProposal(proposal autopilotAuthorityProposal) autopilotAuthorityDecision {
	switch proposal.Target {
	case autopilotAuthorityTasks, autopilotAuthorityDependencies, autopilotAuthoritySequencing, autopilotAuthorityImplementation:
		return autopilotAuthorityDecision{Allowed: true, Reason: "within the displayed implementation authority"}
	default:
		decision := autopilotRunDecisionForCode(autopilotTriggerMissingAuthority, false, map[string]interface{}{
			"target": proposal.Target,
			"before": proposal.Before,
			"after":  proposal.After,
			"reason": proposal.Reason,
		})
		return autopilotAuthorityDecision{
			Allowed: false, Code: decision.Code, Disposition: decision.Disposition,
			Next: decision.Next, Reason: "missing authority",
		}
	}
}

type autopilotRepairStatus string

const (
	autopilotRepairPlanned autopilotRepairStatus = "planned"
	autopilotRepairPassed  autopilotRepairStatus = "passed"
	autopilotRepairFailed  autopilotRepairStatus = "failed"
)

type autopilotRepairFailure struct {
	Phase                  int      `json:"phase"`
	Attempt                string   `json:"attempt"`
	Check                  string   `json:"check"`
	Evidence               []string `json:"evidence,omitempty"`
	PlannedAction          string   `json:"planned_action"`
	Baseline               string   `json:"baseline"`
	PermittedScope         []string `json:"permitted_scope"`
	ScopeSafe              bool     `json:"scope_safe"`
	SafetySafe             bool     `json:"safety_safe"`
	AuthoritySafe          bool     `json:"authority_safe"`
	InvalidatingDependency bool     `json:"invalidating_dependency,omitempty"`
	AffectedPaths          []string `json:"affected_paths,omitempty"`
	IndependentPaths       []string `json:"independent_paths,omitempty"`
}

type autopilotRepairEvaluation struct {
	Eligible bool   `json:"eligible"`
	Pause    bool   `json:"pause"`
	Reason   string `json:"reason"`
}

type autopilotRepairVerification struct {
	Check    string   `json:"check"`
	Passed   bool     `json:"passed"`
	Evidence []string `json:"evidence,omitempty"`
}

type autopilotRepairReceipt struct {
	ID              string                      `json:"id"`
	Phase           int                         `json:"phase"`
	Attempt         string                      `json:"attempt"`
	Check           string                      `json:"check"`
	FailureEvidence []string                    `json:"failure_evidence,omitempty"`
	PermittedScope  []string                    `json:"permitted_scope"`
	PlannedAction   string                      `json:"planned_action"`
	Baseline        string                      `json:"baseline"`
	Status          autopilotRepairStatus       `json:"status"`
	PreparedAt      string                      `json:"prepared_at"`
	CompletedAt     string                      `json:"completed_at,omitempty"`
	BudgetBefore    int                         `json:"budget_before"`
	BudgetRemaining int                         `json:"budget_remaining"`
	Verification    autopilotRepairVerification `json:"verification"`
}

type autopilotRepairLedger struct {
	SchemaVersion  int                      `json:"schema_version"`
	InvocationID   string                   `json:"invocation_id"`
	InitialBudget  int                      `json:"initial_budget"`
	Remaining      int                      `json:"remaining_budget"`
	Receipts       []autopilotRepairReceipt `json:"receipts"`
	Debt           []colony.LifecycleIssue  `json:"debt"`
	Blockers       []colony.LifecycleIssue  `json:"blockers"`
	ContinuedPaths []string                 `json:"continued_paths"`
	SkippedPaths   []string                 `json:"skipped_paths"`
}

type autopilotRepairReport struct {
	Attempts        int                      `json:"attempts"`
	Receipts        []autopilotRepairReceipt `json:"receipts"`
	RemainingBudget int                      `json:"remaining_budget"`
	BudgetExhausted bool                     `json:"budget_exhausted"`
	Debt            []colony.LifecycleIssue  `json:"debt"`
	Blockers        []colony.LifecycleIssue  `json:"blockers"`
	ContinuedPaths  []string                 `json:"continued_paths"`
	SkippedPaths    []string                 `json:"skipped_paths"`
}

const autopilotRepairLedgerSchemaVersion = 1

func newAutopilotRepairLedger(invocationID string, budget int) autopilotRepairLedger {
	if budget < 0 {
		budget = 0
	}
	return autopilotRepairLedger{
		SchemaVersion: autopilotRepairLedgerSchemaVersion, InvocationID: strings.TrimSpace(invocationID),
		InitialBudget: budget, Remaining: budget, Receipts: []autopilotRepairReceipt{},
		Debt: []colony.LifecycleIssue{}, Blockers: []colony.LifecycleIssue{},
		ContinuedPaths: []string{}, SkippedPaths: []string{},
	}
}

func classifyAutopilotRepairFailure(failure autopilotRepairFailure, remainingBudget int) autopilotRepairEvaluation {
	switch {
	case !failure.SafetySafe:
		return autopilotRepairEvaluation{Pause: true, Reason: "safety failure"}
	case !failure.AuthoritySafe || !failure.ScopeSafe:
		return autopilotRepairEvaluation{Pause: true, Reason: "missing authority"}
	case failure.InvalidatingDependency:
		return autopilotRepairEvaluation{Pause: true, Reason: "invalidating failed dependency"}
	case remainingBudget <= 0:
		return autopilotRepairEvaluation{Reason: "repair budget exhausted"}
	case failure.Phase <= 0 || strings.TrimSpace(failure.Check) == "" || strings.TrimSpace(failure.PlannedAction) == "" || strings.TrimSpace(failure.Baseline) == "" || len(failure.PermittedScope) == 0:
		return autopilotRepairEvaluation{Reason: "repair evidence is incomplete"}
	default:
		return autopilotRepairEvaluation{Eligible: true, Reason: "bounded repair is within scope"}
	}
}

func beginAutopilotRepair(ledger *autopilotRepairLedger, failure autopilotRepairFailure, now time.Time) (autopilotRepairReceipt, error) {
	if ledger == nil {
		return autopilotRepairReceipt{}, fmt.Errorf("repair ledger is nil")
	}
	evaluation := classifyAutopilotRepairFailure(failure, ledger.Remaining)
	if !evaluation.Eligible {
		return autopilotRepairReceipt{}, fmt.Errorf("repair is not eligible: %s", evaluation.Reason)
	}
	sequence := len(ledger.Receipts) + 1
	invocation := strings.TrimSpace(ledger.InvocationID)
	if invocation == "" {
		invocation = "unknown"
	}
	receipt := autopilotRepairReceipt{
		ID: fmt.Sprintf("repair-%s-%03d", invocation, sequence), Phase: failure.Phase,
		Attempt: strings.TrimSpace(failure.Attempt), Check: strings.TrimSpace(failure.Check),
		FailureEvidence: append([]string(nil), failure.Evidence...),
		PermittedScope:  append([]string(nil), failure.PermittedScope...),
		PlannedAction:   strings.TrimSpace(failure.PlannedAction), Baseline: strings.TrimSpace(failure.Baseline),
		Status: autopilotRepairPlanned, PreparedAt: now.UTC().Format(time.RFC3339Nano),
		BudgetBefore: ledger.Remaining, BudgetRemaining: ledger.Remaining,
		Verification: autopilotRepairVerification{Check: strings.TrimSpace(failure.Check)},
	}
	ledger.Receipts = append(ledger.Receipts, receipt)
	continued, skipped := autopilotDependencyRouting(failure)
	ledger.ContinuedPaths = mergeAutopilotPaths(ledger.ContinuedPaths, continued)
	ledger.SkippedPaths = mergeAutopilotPaths(ledger.SkippedPaths, skipped)
	return receipt, nil
}

func completeAutopilotRepair(ledger *autopilotRepairLedger, receiptID string, evidence []string, passed bool, now time.Time) error {
	if ledger == nil {
		return fmt.Errorf("repair ledger is nil")
	}
	for i := range ledger.Receipts {
		receipt := &ledger.Receipts[i]
		if receipt.ID != receiptID {
			continue
		}
		if receipt.Status != autopilotRepairPlanned {
			return fmt.Errorf("repair receipt %q is already complete", receiptID)
		}
		if ledger.Remaining <= 0 {
			return fmt.Errorf("repair budget exhausted before receipt %q completed", receiptID)
		}
		ledger.Remaining--
		receipt.BudgetRemaining = ledger.Remaining
		receipt.CompletedAt = now.UTC().Format(time.RFC3339Nano)
		receipt.Verification = autopilotRepairVerification{Check: receipt.Check, Passed: passed, Evidence: append([]string(nil), evidence...)}
		if passed {
			receipt.Status = autopilotRepairPassed
		} else {
			receipt.Status = autopilotRepairFailed
			if ledger.Remaining == 0 {
				recordAutopilotRepairDebt(ledger, autopilotRepairFailure{
					Phase: receipt.Phase, Attempt: receipt.Attempt, Check: receipt.Check,
					Evidence: receipt.Verification.Evidence, AffectedPaths: ledger.SkippedPaths,
				}, "repair verification failed and the repair budget is exhausted", false)
			}
		}
		return nil
	}
	return fmt.Errorf("repair receipt %q was not found", receiptID)
}

// executeAutopilotRepair is the policy's ordering primitive. The planned
// receipt is persisted before action, the named check is rerun exactly once,
// and the completed receipt is persisted after the one budget decrement.
func executeAutopilotRepair(
	ledger *autopilotRepairLedger,
	failure autopilotRepairFailure,
	now func() time.Time,
	persist func(autopilotRepairLedger) error,
	action func() error,
	verify func(string) (bool, []string, error),
) (autopilotRepairReceipt, error) {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	receipt, err := beginAutopilotRepair(ledger, failure, now())
	if err != nil {
		return autopilotRepairReceipt{}, err
	}
	if persist == nil {
		return autopilotRepairReceipt{}, fmt.Errorf("repair receipt persistence is unavailable")
	}
	if err := persist(*ledger); err != nil {
		return autopilotRepairReceipt{}, fmt.Errorf("persist planned repair receipt: %w", err)
	}
	if action == nil || verify == nil {
		return autopilotRepairReceipt{}, fmt.Errorf("repair action and verification are required")
	}
	if err := action(); err != nil {
		_ = completeAutopilotRepair(ledger, receipt.ID, []string{err.Error()}, false, now())
		_ = persist(*ledger)
		return ledger.Receipts[len(ledger.Receipts)-1], err
	}
	passed, evidence, verifyErr := verify(receipt.Check)
	if verifyErr != nil {
		evidence = append(evidence, verifyErr.Error())
		passed = false
	}
	if err := completeAutopilotRepair(ledger, receipt.ID, evidence, passed, now()); err != nil {
		return autopilotRepairReceipt{}, err
	}
	if err := persist(*ledger); err != nil {
		return autopilotRepairReceipt{}, fmt.Errorf("persist completed repair receipt: %w", err)
	}
	return ledger.Receipts[len(ledger.Receipts)-1], verifyErr
}

func recordAutopilotRepairDebt(ledger *autopilotRepairLedger, failure autopilotRepairFailure, reason string, blocking bool) {
	if ledger == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "repair could not be completed"
	}
	id := fmt.Sprintf("autopilot-debt-phase-%d-%s", failure.Phase, strings.ReplaceAll(strings.ToLower(strings.TrimSpace(failure.Check)), " ", "-"))
	issue := colony.LifecycleIssue{ID: id, Summary: fmt.Sprintf("Phase %d %s: %s", failure.Phase, emptyFallback(failure.Check, "verification"), reason), EvidenceIDs: append([]string(nil), failure.Evidence...)}
	target := &ledger.Debt
	if blocking {
		target = &ledger.Blockers
	}
	for _, existing := range *target {
		if existing.ID == issue.ID {
			return
		}
	}
	*target = append(*target, issue)
	continued, skipped := autopilotDependencyRouting(failure)
	ledger.ContinuedPaths = mergeAutopilotPaths(ledger.ContinuedPaths, continued)
	ledger.SkippedPaths = mergeAutopilotPaths(ledger.SkippedPaths, skipped)
}

func autopilotDependencyRouting(failure autopilotRepairFailure) (continued, skipped []string) {
	continued = mergeAutopilotPaths(nil, failure.IndependentPaths)
	skipped = mergeAutopilotPaths(nil, failure.AffectedPaths)
	return continued, skipped
}

func mergeAutopilotPaths(existing, added []string) []string {
	seen := map[string]struct{}{}
	for _, path := range append(append([]string(nil), existing...), added...) {
		path = strings.TrimSpace(path)
		if path != "" {
			seen[path] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func autopilotRepairReportFields(ledger autopilotRepairLedger) autopilotRepairReport {
	return autopilotRepairReport{
		Attempts: len(ledger.Receipts), Receipts: append([]autopilotRepairReceipt(nil), ledger.Receipts...),
		RemainingBudget: ledger.Remaining, BudgetExhausted: ledger.Remaining == 0,
		Debt: append([]colony.LifecycleIssue(nil), ledger.Debt...), Blockers: append([]colony.LifecycleIssue(nil), ledger.Blockers...),
		ContinuedPaths: append([]string(nil), ledger.ContinuedPaths...), SkippedPaths: append([]string(nil), ledger.SkippedPaths...),
	}
}
