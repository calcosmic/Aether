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
	// NextTransition is WORK-07's goal-level selection (201-11): the next
	// transition the controller would pick from these exact recorded facts,
	// without the owner naming a phase number. Computed once, additively, by
	// buildAutopilotPreflightCore -- it never changes any other field's
	// value or any existing return path's behavior.
	NextTransition autopilotGoalTransitionDecision `json:"next_transition,omitempty"`
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

// buildAutopilotPreflightCore builds the preflight and then attaches WORK-07's
// goal-level transition selection (201-11), computed once from the exact same
// facts and the preflight just built -- additive only, and it never changes
// which branch buildAutopilotPreflightCoreValue takes.
func buildAutopilotPreflightCore(facts LifecycleFacts, authority *planAuthorityDecision) AutopilotPreflight {
	preflight := buildAutopilotPreflightCoreValue(facts, authority)
	preflight.NextTransition = selectAutopilotGoalTransition(autopilotGoalLevelFactsFromLifecycle(facts, preflight))
	return preflight
}

func buildAutopilotPreflightCoreValue(facts LifecycleFacts, authority *planAuthorityDecision) AutopilotPreflight {
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
	// CheckpointID is D-09/SYN-201-10's idempotency key: the checkpoint
	// identity this receipt is bound to (cmd/work_repair.go's
	// runBoundedRepairRound), so a replayed call for the same identity
	// finds this receipt rather than starting a second round. Empty for
	// receipts predating that generalization (omitempty preserves their
	// existing serialization).
	CheckpointID string `json:"checkpoint_id,omitempty"`
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

// ---------------------------------------------------------------------------
// Goal-level transition selection (201-11, WORK-07).
//
// From one accepted goal the controller selects the next transition itself,
// without the owner naming a phase number. selectAutopilotGoalTransition is
// the pure decision core; autopilotGoalLevelFactsFromLifecycle is the one
// bridge from the read-only LifecycleFacts snapshot (plus the already
// computed preflight) into the selector's narrow input shape. Every input
// field is an exact recorded fact -- never a rounded score or a derived
// percentage (must_haves backstop truth).
// ---------------------------------------------------------------------------

// autopilotGoalTransition is a durable named transition. These values may be
// persisted and rendered, so changing one is a compatibility change --
// mirroring autopilotTriggerCode's own discipline.
type autopilotGoalTransition string

const (
	autopilotTransitionSurvey        autopilotGoalTransition = "survey"
	autopilotTransitionPlanning      autopilotGoalTransition = "planning"
	autopilotTransitionWork          autopilotGoalTransition = "work"
	autopilotTransitionVerification  autopilotGoalTransition = "verification"
	autopilotTransitionBoundedRepair autopilotGoalTransition = "bounded_repair"
	autopilotTransitionReplan        autopilotGoalTransition = "replan"
	autopilotTransitionReadyToSeal   autopilotGoalTransition = "ready_to_seal"
)

// autopilotGoalLevelFacts is the narrow, exact set of recorded facts the
// goal-level selector reads. RemainingPhaseIDs is the ordered, ascending set
// of phases not yet completed -- the selector never picks a phase index past
// its last element, because it only ever reads RemainingPhaseIDs[0].
type autopilotGoalLevelFacts struct {
	AcceptedGoal       string
	HasSurveyEvidence  bool
	HasAcceptedPlan    bool
	RemainingPhaseIDs  []int
	ReplanDue          bool
	WorkBuilt          bool
	VerificationFailed bool
}

// autopilotGoalTransitionDecision is the selector's typed result: the named
// transition, the exact recorded fact that drove it (never empty when a
// transition was selected), and -- when the transition is phase-specific --
// which phase. The zero value (empty Transition) means "no accepted goal is
// recorded," which callers must treat as a zero-write entry.
type autopilotGoalTransitionDecision struct {
	Transition autopilotGoalTransition `json:"transition,omitempty"`
	Reason     string                  `json:"reason,omitempty"`
	PhaseID    int                     `json:"phase_id,omitempty"`
}

// selectAutopilotGoalTransition is the pure goal-level transition selector
// (WORK-07): given one accepted goal and the immutable recorded lifecycle
// facts, it names the next transition without the caller supplying a phase
// number. It covers survey, planning, work, verification, bounded repair,
// replan, and ready-to-seal -- and, at the boundary where no phase remains,
// selects ready-to-seal, naming that nothing remains to run.
func selectAutopilotGoalTransition(facts autopilotGoalLevelFacts) autopilotGoalTransitionDecision {
	if strings.TrimSpace(facts.AcceptedGoal) == "" {
		return autopilotGoalTransitionDecision{Reason: "no accepted goal is recorded"}
	}
	if !facts.HasSurveyEvidence {
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionSurvey,
			Reason:     "no survey evidence is recorded for the accepted goal",
		}
	}
	if !facts.HasAcceptedPlan {
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionPlanning,
			Reason:     "survey evidence is recorded but no plan has been accepted",
		}
	}
	if len(facts.RemainingPhaseIDs) == 0 {
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionReadyToSeal,
			Reason:     "the accepted plan has no phase remaining -- nothing remaining to run",
		}
	}
	// facts.RemainingPhaseIDs[0] is the only phase this selector ever names --
	// it is, by construction, always a member of the caller's own recorded
	// remaining set, so a phase past the last remaining one is never selected.
	phase := facts.RemainingPhaseIDs[0]
	if facts.ReplanDue {
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionReplan,
			Reason:     "the configured replan cadence is due",
			PhaseID:    phase,
		}
	}
	switch {
	case facts.VerificationFailed:
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionBoundedRepair,
			Reason:     fmt.Sprintf("phase %d verification failed and bounded repair is available", phase),
			PhaseID:    phase,
		}
	case facts.WorkBuilt:
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionVerification,
			Reason:     fmt.Sprintf("phase %d has built, unverified work", phase),
			PhaseID:    phase,
		}
	default:
		return autopilotGoalTransitionDecision{
			Transition: autopilotTransitionWork,
			Reason:     fmt.Sprintf("phase %d has an accepted plan and unbuilt work", phase),
			PhaseID:    phase,
		}
	}
}

// autopilotGoalLevelFactsFromLifecycle bridges the read-only LifecycleFacts
// snapshot and the already-computed preflight into the selector's narrow
// input shape -- the only place these two are translated into the exact
// facts selectAutopilotGoalTransition reads. HasAcceptedPlan and
// RemainingPhaseIDs are read from the preflight (which already validated
// plan authority and ordering), never re-derived here. Survey evidence is
// read from the recorded territory survey artifacts (facts.Research.Value.
// Territory, .aether/data/survey/) -- the one durable record colonize
// already writes.
func autopilotGoalLevelFactsFromLifecycle(facts LifecycleFacts, preflight AutopilotPreflight) autopilotGoalLevelFacts {
	goalLevel := autopilotGoalLevelFacts{
		AcceptedGoal:      preflight.Goal,
		HasSurveyEvidence: len(facts.Research.Value.Territory) > 0,
		HasAcceptedPlan:   preflight.Valid || preflight.Completed,
		RemainingPhaseIDs: append([]int(nil), preflight.RemainingPhases...),
	}
	switch facts.State.Value.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		goalLevel.WorkBuilt = true
	}
	for _, gate := range facts.Verification.Value.Gates {
		if !gate.Passed {
			goalLevel.VerificationFailed = true
			break
		}
	}
	return goalLevel
}

// ---------------------------------------------------------------------------
// Stop boundaries (201-11, WORK-07): the closed, four-member set naming
// which kind of person-required condition ended a run. Autopilot stops only
// at a declared owner, authority, physical, or unrecoverable boundary, and
// the stop names which of the four it was.
// ---------------------------------------------------------------------------

// autopilotStopBoundary is a durable named boundary. These values may be
// persisted and rendered on a stop card, so changing one is a compatibility
// change.
type autopilotStopBoundary string

const (
	// autopilotStopBoundaryOwner: an unanswered owner decision, or a
	// reviewer forced by one of the five named risk signals that the owner
	// has not waived.
	autopilotStopBoundaryOwner autopilotStopBoundary = "owner"
	// autopilotStopBoundaryAuthority: a typed owner-authority proposal is
	// pending -- a change outside the displayed implementation authority.
	autopilotStopBoundaryAuthority autopilotStopBoundary = "authority"
	// autopilotStopBoundaryPhysical: a missing external prerequisite (a
	// required worker provider is unavailable).
	autopilotStopBoundaryPhysical autopilotStopBoundary = "physical"
	// autopilotStopBoundaryUnrecoverable: a bounded repair failed and its
	// checkpoint was restored -- D-09's one bounded, checkpointed repair
	// path found nothing further it could safely try.
	autopilotStopBoundaryUnrecoverable autopilotStopBoundary = "unrecoverable"
)

// autopilotStopBoundaries is the closed, total set this run loop may name on
// a stop card. Tests iterate this slice rather than re-typing the four
// values, so a fifth boundary added later is caught by name.
func autopilotStopBoundaries() []autopilotStopBoundary {
	return []autopilotStopBoundary{
		autopilotStopBoundaryOwner,
		autopilotStopBoundaryAuthority,
		autopilotStopBoundaryPhysical,
		autopilotStopBoundaryUnrecoverable,
	}
}

func validAutopilotStopBoundary(boundary autopilotStopBoundary) bool {
	for _, candidate := range autopilotStopBoundaries() {
		if candidate == boundary {
			return true
		}
	}
	return false
}

// autopilotStopBoundaryForTriggerCode names which of the four declared
// boundaries a trigger code represents, if any. This is the ONLY place that
// maps the existing, already-recorded trigger-code catalogue
// (autopilotTriggerSpecs) onto the closed boundary set --
// finishAutopilotInvocation calls it to name the boundary on every stop
// card, additively, without altering any existing field or disposition.
//
// A NormalStop code (cancelled, worker_timeout, max_phases_reached,
// colony_complete) is an ordinary ending, never a person-required boundary,
// and replan_due is a queued checkpoint, not a stop -- neither appears here,
// so autopilotStopBoundaryForTriggerCode reports ok=false for them: "any
// other condition does not stop the run" (as a named boundary).
func autopilotStopBoundaryForTriggerCode(code autopilotTriggerCode) (autopilotStopBoundary, bool) {
	switch code {
	case autopilotTriggerRuntimeVerificationNeeded, autopilotTriggerVisualCheckpointNeeded,
		autopilotTriggerBlockerCountIncreased, autopilotTriggerBlockerEscalated:
		return autopilotStopBoundaryOwner, true
	case autopilotTriggerMissingAuthority:
		return autopilotStopBoundaryAuthority, true
	case autopilotTriggerProviderUnavailable:
		return autopilotStopBoundaryPhysical, true
	case autopilotTriggerDeterministicVerificationFailed, autopilotTriggerAuditorScoreBelowFloor, autopilotTriggerCriticalReviewFinding:
		return autopilotStopBoundaryUnrecoverable, true
	default:
		return "", false
	}
}

// autopilotStopBoundaryWorkOutcome maps every declared boundary onto the
// six-verdict work-outcome vocabulary (colony.WorkOutcome, D-05) so a stop
// card renders through the exact same full-ceremony closeout
// (buildLifecycleCloseout / lifecycleCloseoutSlots) a success card renders
// through: lifecycleCloseoutSlots's only ceremony gate is "was a work
// verdict supplied," never which one, so every declared boundary -- carrying
// a non-nil verdict -- gets the identical canonical slot set and the same
// cost-and-time block a success card gets. All four boundaries map to
// colony.WorkOutcomeBlocker: each is "something the run could not get past"
// without an owner, matching that verdict's own recommended action (aether
// unblock --dispatch).
func autopilotStopBoundaryWorkOutcome(autopilotStopBoundary) colony.WorkOutcome {
	return colony.WorkOutcomeBlocker
}
