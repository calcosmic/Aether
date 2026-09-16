package cmd

// Lifecycle projection is the one pure semantic answer behind ordinary
// lifecycle screens. It does not read files, inspect the terminal, ask for the
// time, or mutate its input. View selection only chooses ordered sections;
// platform selection only changes display spelling at the final boundary.

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	LifecycleResultSchemaVersion = "lifecycle-result/v1"
	LifecycleProjectionRevision  = "lifecycle-projection/v1"
)

type LifecycleView string

const (
	LifecycleViewFull    LifecycleView = "full"
	LifecycleViewCompact LifecycleView = "compact"
	LifecycleViewFocused LifecycleView = "focused"
	LifecycleViewJSON    LifecycleView = "json"
	LifecycleViewVisual  LifecycleView = "visual"
)

type LifecycleProjectionSection struct {
	ID      string   `json:"id"`
	Domains []string `json:"domains"`
}

type LifecyclePhaseProjection struct {
	Current         *colony.Phase `json:"current,omitempty"`
	CurrentNumber   int           `json:"current_number"`
	TotalPhases     int           `json:"total_phases"`
	CompletedPhases int           `json:"completed_phases"`
	TotalTasks      int           `json:"total_tasks"`
	CompletedTasks  int           `json:"completed_tasks"`
}

type LifecycleLineage struct {
	Parent string `json:"parent"`
	Actor  string `json:"actor"`
	Caste  string `json:"caste"`
	Depth  int    `json:"depth"`
}

type LifecycleActionChoice struct {
	ID             string `json:"id"`
	RuntimeCommand string `json:"runtime_command"`
	DisplayCommand string `json:"display_command"`
	Reason         string `json:"reason,omitempty"`
	Rank           int    `json:"rank"`
	Recommended    bool   `json:"recommended"`
	Label          string `json:"label,omitempty"`
}

type LifecycleProjectedAction struct {
	ID             string                     `json:"id"`
	RuntimeCommand string                     `json:"runtime_command,omitempty"`
	DisplayCommand string                     `json:"display_command,omitempty"`
	Reason         string                     `json:"reason"`
	Evidence       []colony.LifecycleEvidence `json:"evidence,omitempty"`
	Choices        []LifecycleActionChoice    `json:"choices,omitempty"`
}

type LifecycleClosureProjection struct {
	Status       string `json:"status"`
	Forced       bool   `json:"forced"`
	OwnerReason  string `json:"owner_reason,omitempty"`
	Inspectable  bool   `json:"inspectable"`
	ArchiveReady bool   `json:"archive_ready"`
}

// LifecycleProjection deliberately mirrors the shared result contract from
// 199-UI-SPEC. Domain values retain their availability source while lifecycle
// claims use the typed evidence vocabulary introduced in Plan 03.
type LifecycleProjection struct {
	SchemaVersion      string             `json:"schema_version"`
	Command            string             `json:"command"`
	OutcomeKind        colony.OutcomeKind `json:"outcome_kind"`
	ProjectionRevision string             `json:"projection_revision"`
	View               LifecycleView      `json:"view"`
	Platform           string             `json:"platform"`

	Identity      LifecycleFact[LifecycleIdentityFacts]      `json:"identity"`
	Intent        LifecycleFact[LifecycleIntentFacts]        `json:"intent"`
	Specification LifecycleFact[LifecycleSpecificationFacts] `json:"specification"`
	Planning      LifecycleFact[LifecyclePlanningFacts]      `json:"planning"`
	Goal          LifecycleFact[string]                      `json:"goal"`
	Standing      LifecycleFact[string]                      `json:"standing"`
	Phase         LifecycleFact[LifecyclePhaseProjection]    `json:"phase"`
	Tasks         LifecycleFact[[]colony.Task]               `json:"tasks"`
	Actors        LifecycleFact[[]LifecycleActorFact]        `json:"actors"`
	Lineage       LifecycleFact[[]LifecycleLineage]          `json:"lineage"`
	Signals       LifecycleFact[[]colony.PheromoneSignal]    `json:"signals"`
	Research      LifecycleFact[LifecycleResearchFacts]      `json:"research"`
	Memory        LifecycleFact[LifecycleMemoryFacts]        `json:"memory"`
	Findings      LifecycleFact[[]colony.ReviewLedgerEntry]  `json:"findings"`

	Changes        []colony.LifecycleChange       `json:"changes,omitempty"`
	Evidence       []colony.LifecycleEvidence     `json:"evidence,omitempty"`
	Verification   []colony.LifecycleVerification `json:"verification,omitempty"`
	Warnings       []colony.LifecycleIssue        `json:"warnings,omitempty"`
	Debt           []colony.LifecycleIssue        `json:"debt,omitempty"`
	Blockers       []colony.LifecycleIssue        `json:"blockers,omitempty"`
	OwnerDecisions []colony.LifecycleDecision     `json:"owner_decisions,omitempty"`

	Elapsed      LifecycleFact[time.Duration]              `json:"elapsed"`
	ReportedCost LifecycleFact[LifecycleReportedCostFacts] `json:"reported_cost"`
	History      LifecycleFact[[]string]                   `json:"history"`

	NextAction   LifecycleProjectedAction              `json:"next_action"`
	Alternatives []LifecycleActionChoice               `json:"alternatives,omitempty"`
	StateEffect  colony.LifecycleStateEffect           `json:"state_effect"`
	Transaction  *colony.LifecycleTransactionReference `json:"transaction,omitempty"`
	Receipt      *colony.LifecycleReceipt              `json:"receipt,omitempty"`
	Recovery     *colony.LifecycleRecovery             `json:"recovery,omitempty"`
	Provenance   colony.RecoveryProvenance             `json:"provenance"`
	Closure      LifecycleClosureProjection            `json:"closure"`
	Sections     []LifecycleProjectionSection          `json:"sections"`
}

func lifecycleProjectionSections(view LifecycleView) []LifecycleProjectionSection {
	full := []LifecycleProjectionSection{
		{ID: "identity", Domains: []string{"identity", "intent", "goal", "standing"}},
		{ID: "progress", Domains: []string{"specification", "planning", "phase", "tasks"}},
		{ID: "actors", Domains: []string{"actors", "lineage"}},
		{ID: "signals", Domains: []string{"signals"}},
		{ID: "research", Domains: []string{"research", "dreams", "territory"}},
		{ID: "memory_evidence", Domains: []string{"memory", "findings", "verification", "evidence"}},
		{ID: "elapsed_cost", Domains: []string{"elapsed", "reported_cost"}},
		{ID: "history", Domains: []string{"history"}},
		{ID: "open_items", Domains: []string{"warnings", "debt", "blockers", "owner_decisions"}},
		{ID: "next_action", Domains: []string{"next_action", "alternatives"}},
	}
	switch view {
	case LifecycleViewCompact:
		return []LifecycleProjectionSection{full[0], full[1], full[2], full[4], full[6], full[8], full[9]}
	case LifecycleViewFocused:
		return []LifecycleProjectionSection{full[0], full[1], full[2], full[5], full[8], full[9]}
	default:
		return full
	}
}

func lifecycleCurrentPhase(progress LifecycleProgressFacts) (*colony.Phase, int) {
	for i := range progress.Phases {
		if progress.CurrentPhase > 0 && progress.Phases[i].ID == progress.CurrentPhase {
			phase := progress.Phases[i]
			return &phase, phase.ID
		}
	}
	for i := range progress.Phases {
		if progress.Phases[i].Status != colony.PhaseCompleted {
			phase := progress.Phases[i]
			return &phase, phase.ID
		}
	}
	if len(progress.Phases) > 0 {
		phase := progress.Phases[len(progress.Phases)-1]
		return &phase, phase.ID
	}
	return nil, progress.CurrentPhase
}

func lifecycleProjectedProgress(progress LifecycleProgressFacts) (LifecyclePhaseProjection, []colony.Task) {
	current, number := lifecycleCurrentPhase(progress)
	result := LifecyclePhaseProjection{Current: current, CurrentNumber: number, TotalPhases: len(progress.Phases)}
	var tasks []colony.Task
	for _, phase := range progress.Phases {
		if phase.Status == colony.PhaseCompleted {
			result.CompletedPhases++
		}
		result.TotalTasks += len(phase.Tasks)
		for _, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				result.CompletedTasks++
			}
		}
		if current != nil && phase.ID == current.ID {
			tasks = append([]colony.Task(nil), phase.Tasks...)
		}
	}
	return result, tasks
}

func lifecycleProjectedLineage(actors []LifecycleActorFact) []LifecycleLineage {
	lineage := make([]LifecycleLineage, 0, len(actors))
	for _, actor := range actors {
		lineage = append(lineage, LifecycleLineage{Parent: actor.Parent, Actor: actor.Name, Caste: actor.Caste, Depth: actor.Depth})
	}
	return lineage
}

func lifecycleProjectionWarnings(facts LifecycleFacts) []colony.LifecycleIssue {
	var warnings []colony.LifecycleIssue
	for _, source := range facts.Sources() {
		if source.Provenance == LifecycleFactConfirmed || source.Diagnostic == "" {
			continue
		}
		warnings = append(warnings, colony.LifecycleIssue{
			ID:      "fact-" + strings.ReplaceAll(source.Domain, " ", "-"),
			Summary: fmt.Sprintf("%s is %s: %s", source.Domain, source.Provenance, source.Diagnostic),
		})
	}
	return warnings
}

func lifecycleProjectionReceiptFields(facts LifecycleFacts) (
	[]colony.LifecycleChange,
	[]colony.LifecycleEvidence,
	[]colony.LifecycleVerification,
	[]colony.LifecycleIssue,
	[]colony.LifecycleIssue,
	[]colony.LifecycleDecision,
	*colony.LifecycleTransactionReference,
	*colony.LifecycleRecovery,
) {
	receipt := facts.Evidence.Value.Receipt
	if receipt == nil {
		return nil, nil, nil, nil, nil, nil, nil, nil
	}
	transaction := receipt.Transaction
	return append([]colony.LifecycleChange(nil), receipt.Changes...),
		append([]colony.LifecycleEvidence(nil), receipt.Evidence...),
		append([]colony.LifecycleVerification(nil), receipt.Verification...),
		append([]colony.LifecycleIssue(nil), receipt.Debt...),
		append([]colony.LifecycleIssue(nil), receipt.Blockers...),
		append([]colony.LifecycleDecision(nil), receipt.Decisions...),
		&transaction,
		receipt.Recovery
}

func lifecycleProjectionOpenItems(facts LifecycleFacts, receiptBlockers []colony.LifecycleIssue, receiptDecisions []colony.LifecycleDecision) ([]colony.LifecycleIssue, []colony.LifecycleDecision) {
	blockers := append([]colony.LifecycleIssue(nil), receiptBlockers...)
	decisions := append([]colony.LifecycleDecision(nil), receiptDecisions...)
	for _, flag := range facts.Blockers.Value {
		if flag.Resolved {
			continue
		}
		if strings.EqualFold(flag.Type, "blocker") {
			blockers = append(blockers, colony.LifecycleIssue{ID: flag.ID, Summary: flag.Description})
			continue
		}
		decisions = append(decisions, colony.LifecycleDecision{ID: flag.ID, Scope: flag.Type, Summary: flag.Description})
	}
	return blockers, decisions
}

func lifecycleProjectionCommand(runtimeCommand, platform string) string {
	if platform == "codex" {
		// Change only the command prefix, preserving every argument byte.
		if !strings.HasPrefix(runtimeCommand, "aether ") {
			return runtimeCommand
		}
		verb, _, _ := strings.Cut(strings.TrimPrefix(runtimeCommand, "aether "), " ")
		return platformCommandName(verb, platform) + strings.TrimPrefix(runtimeCommand, "aether "+verb)
	}
	runtimeCommand = strings.TrimSpace(runtimeCommand)
	if runtimeCommand == "" {
		return runtimeCommand
	}
	if !strings.HasPrefix(runtimeCommand, "aether ") {
		return runtimeCommand
	}
	rest := strings.TrimPrefix(runtimeCommand, "aether ")
	verb, tail, _ := strings.Cut(rest, " ")
	display := "/ant-" + verb
	if tail != "" {
		display += " " + tail
	}
	return display
}

func lifecycleChoice(id, runtime, reason string) LifecycleActionChoice {
	return LifecycleActionChoice{ID: id, RuntimeCommand: runtime, Reason: reason}
}

func lifecycleAction(id, runtime, reason string, evidence []colony.LifecycleEvidence) LifecycleProjectedAction {
	return LifecycleProjectedAction{ID: id, RuntimeCommand: runtime, Reason: reason, Evidence: append([]colony.LifecycleEvidence(nil), evidence...)}
}

func lifecycleAllPhasesComplete(progress LifecycleProgressFacts) bool {
	if len(progress.Phases) == 0 {
		return false
	}
	for _, phase := range progress.Phases {
		if phase.Status != colony.PhaseCompleted {
			return false
		}
	}
	return true
}

func lifecycleProjectionDecision(facts LifecycleFacts, blockers []colony.LifecycleIssue, evidence []colony.LifecycleEvidence) (LifecycleProjectedAction, []LifecycleActionChoice, colony.OutcomeKind, LifecycleClosureProjection, colony.RecoveryProvenance) {
	state := facts.State.Value
	progress := facts.Progress.Value
	closure := LifecycleClosureProjection{Status: "open"}
	provenance := colony.RecoveryProvenanceUnknown

	// An absent state is an honest empty repository. Any present-but-unusable
	// state or lifecycle-evidence source is ambiguous recovery, not emptiness.
	if facts.State.Source.Provenance == LifecycleFactMissing ||
		(strings.TrimSpace(facts.Identity.Value.Goal) == "" && len(progress.Phases) == 0 && progress.CurrentPhase < 1 && facts.State.Source.Provenance == LifecycleFactConfirmed) {
		return lifecycleActionFromCandidate("initialize", candidateInit, "No colony goal is active in this repository.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect this repository without changing it."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review any retained activity before starting."),
		}, colony.OutcomeKindNoChange, closure, colony.RecoveryProvenanceUnknown
	}
	if facts.State.Source.Provenance == LifecycleFactMalformed || facts.State.Source.Provenance == LifecycleFactUnavailable ||
		facts.Evidence.Source.Provenance == LifecycleFactMalformed || facts.Evidence.Source.Provenance == LifecycleFactUnavailable ||
		facts.Specification.Source.Provenance == LifecycleFactMalformed || facts.Specification.Source.Provenance == LifecycleFactUnavailable ||
		facts.Planning.Source.Provenance == LifecycleFactMalformed || facts.Planning.Source.Provenance == LifecycleFactUnavailable ||
		facts.Intent.Source.Provenance == LifecycleFactMalformed || (facts.Root != "" && facts.Intent.Source.Provenance == LifecycleFactUnavailable) {
		closure.Status = "unknown"
		return lifecycleActionFromCandidate("resume", candidateResume, "Saved lifecycle evidence is incomplete or conflicting; resume is the single recovery door.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect the retained evidence without changing it."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the recorded activity before recovery."),
		}, colony.OutcomeKindRecoveryRequired, closure, colony.RecoveryProvenanceUnknown
	}

	if recorded := facts.Evidence.Value.Recovery; recorded != nil && recorded.Valid() {
		provenance = *recorded
	} else if receipt := facts.Evidence.Value.Receipt; receipt != nil && receipt.Provenance.Valid() {
		provenance = receipt.Provenance
	} else if seal := facts.Evidence.Value.Seal; seal != nil && seal.Provenance.Valid() {
		provenance = seal.Provenance
	}
	if seal := facts.Evidence.Value.Seal; seal != nil {
		closure.Inspectable = true
		closure.ArchiveReady = true
		closure.OwnerReason = seal.OwnerReason
		outcome := colony.OutcomeKindVerifiedCompletion
		if seal.Disposition == colony.SealDispositionForcedIncomplete {
			closure.Status = "forced_incomplete"
			closure.Forced = true
			outcome = colony.OutcomeKindForcedIncompleteClosure
		} else {
			closure.Status = "verified"
		}
		return lifecycleActionFromCandidate("inspect_sealed", candidateStatus, "The finished project's record remains active and available to inspect.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("entomb", candidateEntomb, "Optionally archive and clear the finished project's retained record."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the retained lifecycle history."),
		}, outcome, closure, provenance
	}
	// The pre-lifecycle/v1 sealed marker remains inspectable, but it is not
	// promoted to verified closure without a typed seal outcome.
	if state.State == colony.StateCOMPLETED && strings.EqualFold(strings.TrimSpace(state.Milestone), "Crowned Anthill") {
		closure.Status = "sealed_legacy"
		closure.Inspectable = true
		closure.ArchiveReady = true
		return lifecycleActionFromCandidate("inspect_sealed", candidateStatus, "This older finished record remains available to inspect. It does not contain enough evidence to call the closure verified.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("entomb", candidateEntomb, "Optionally archive and clear the finished project's retained record."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the retained lifecycle history."),
		}, colony.OutcomeKindNoChange, closure, colony.RecoveryProvenanceReconstructed
	}

	if state.Paused {
		return lifecycleActionFromCandidate("resume", candidateResume, "The colony is paused; resume validates the handoff before restoring work.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect the paused state without restoring it."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review what happened before the pause."),
		}, colony.OutcomeKindPaused, closure, provenance
	}
	for _, phase := range progress.Phases {
		if phase.Status == "failed" {
			blockers = append(blockers, colony.LifecycleIssue{ID: fmt.Sprintf("phase-%d-failed", phase.ID), Summary: fmt.Sprintf("Phase %d failed", phase.ID)})
		}
	}
	if len(blockers) > 0 {
		return lifecycleActionFromCandidate("resume", candidateResume, "Lifecycle work is blocked; resume reconciles the durable evidence before work continues.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect the blockers without changing state."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the evidence leading to the block."),
		}, colony.OutcomeKindRecoveryRequired, closure, provenance
	}
	if action, alternatives, outcome, handled := lifecycleAuthorityNextAction(facts, evidence); handled {
		return action, alternatives, outcome, closure, provenance
	}
	if state.State == colony.StateCOMPLETED || lifecycleAllPhasesComplete(progress) {
		closure.Status = "ready_to_seal"
		return lifecycleActionFromCandidate("seal", candidateSeal, "All accepted work is complete; sealing records the verified retained closure.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Review the completed work before sealing."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the recorded activity first."),
		}, colony.OutcomeKindCompleted, closure, provenance
	}
	if len(progress.Phases) == 0 {
		return lifecycleActionFromCandidate("plan", candidatePlan, "The goal is accepted but no plan has been accepted yet.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Review the accepted goal before planning."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the recorded setup activity."),
		}, colony.OutcomeKindNoChange, closure, provenance
	}
	if state.State == colony.StateEXECUTING && state.CurrentPhase > 0 && state.BuildStartedAt == nil {
		return lifecycleActionFromCandidate("resume", candidateResume, "The saved state says work was executing but carries no start evidence; resume must reconcile it before another run.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect the uncertain execution state."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the last recorded activity."),
		}, colony.OutcomeKindRecoveryRequired, closure, colony.RecoveryProvenanceUnknown
	}
	if state.State == colony.StateEXECUTING || state.State == colony.StateBUILT {
		return lifecycleActionFromCandidate("continue", candidateContinue, "The current phase has work that must be checked and advanced.", evidence), []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Inspect the current phase before checking it."),
			lifecycleChoiceFromCandidate("history", candidateHistory, "Review the activity behind the current phase."),
		}, colony.OutcomeKindInProgress, closure, provenance
	}

	_, phase := lifecycleCurrentPhase(progress)
	if phase < 1 {
		phase = 1
	}
	build := lifecycleChoiceFromCandidate("build", candidateBuildPhase, "Run the current phase with guided checkpoints.", phase)
	run := lifecycleChoiceFromCandidate("run", candidateRun, "Run every remaining accepted phase in Autopilot.")
	// Deliberately equal: zero rank, no recommendation marker, stable plan order.
	return LifecycleProjectedAction{
			ID:       "choose_execution_mode",
			Reason:   "Choose guided operation or Autopilot; both execute the same accepted plan.",
			Evidence: append([]colony.LifecycleEvidence(nil), evidence...),
			Choices:  []LifecycleActionChoice{build, run},
		}, []LifecycleActionChoice{
			lifecycleChoiceFromCandidate("status", candidateStatus, "Review the accepted plan before choosing a mode."),
			lifecycleChoiceFromCandidate("pheromones", candidatePheromones, "Review the standing instructions before starting."),
		}, colony.OutcomeKindNoChange, closure, provenance
}

func lifecycleApplyPlatform(action LifecycleProjectedAction, alternatives []LifecycleActionChoice, platform string) (LifecycleProjectedAction, []LifecycleActionChoice) {
	action.DisplayCommand = lifecycleProjectionCommand(action.RuntimeCommand, platform)
	for i := range action.Choices {
		action.Choices[i].DisplayCommand = lifecycleProjectionCommand(action.Choices[i].RuntimeCommand, platform)
	}
	for i := range alternatives {
		alternatives[i].DisplayCommand = lifecycleProjectionCommand(alternatives[i].RuntimeCommand, platform)
	}
	return action, alternatives
}

// projectLifecycle is pure: facts, view, and platform fully determine the
// result. The function intentionally receives no root, store, clock, terminal,
// or environment handle.
func projectLifecycle(facts LifecycleFacts, view LifecycleView, platform string) LifecycleProjection {
	phase, tasks := lifecycleProjectedProgress(facts.Progress.Value)
	changes, evidence, verification, debt, receiptBlockers, receiptDecisions, transaction, recovery := lifecycleProjectionReceiptFields(facts)
	blockers, ownerDecisions := lifecycleProjectionOpenItems(facts, receiptBlockers, receiptDecisions)
	action, alternatives, outcome, closure, provenance := lifecycleProjectionDecision(facts, blockers, evidence)
	action, alternatives = lifecycleApplyPlatform(action, alternatives, platform)

	for _, gate := range facts.Verification.Value.Gates {
		verification = append(verification, colony.LifecycleVerification{Name: gate.Name, Passed: gate.Passed, Detail: gate.Detail})
	}
	standingSource := lifecycleDerivedSource("standing", facts.State.Source)
	goalSource := lifecycleDerivedSource("goal", facts.Identity.Source)
	phaseSource := lifecycleDerivedSource("phase", facts.Progress.Source)
	taskSource := lifecycleDerivedSource("tasks", facts.Progress.Source)
	lineageSource := lifecycleDerivedSource("lineage", facts.Actors.Source)
	findingSource := lifecycleDerivedSource("findings", facts.Memory.Source)

	standing := facts.Identity.Value.Standing
	if facts.State.Source.Provenance != LifecycleFactConfirmed {
		standing = "UNKNOWN"
	}
	projection := LifecycleProjection{
		SchemaVersion:      LifecycleResultSchemaVersion,
		Command:            "lifecycle_projection",
		OutcomeKind:        outcome,
		ProjectionRevision: LifecycleProjectionRevision,
		View:               view,
		Platform:           platform,
		Identity:           facts.Identity,
		Intent:             facts.Intent,
		Specification:      facts.Specification,
		Planning:           facts.Planning,
		Goal:               LifecycleFact[string]{Value: facts.Identity.Value.Goal, Source: goalSource},
		Standing:           LifecycleFact[string]{Value: standing, Source: standingSource},
		Phase:              LifecycleFact[LifecyclePhaseProjection]{Value: phase, Source: phaseSource},
		Tasks:              LifecycleFact[[]colony.Task]{Value: tasks, Source: taskSource},
		Actors:             facts.Actors,
		Lineage:            LifecycleFact[[]LifecycleLineage]{Value: lifecycleProjectedLineage(facts.Actors.Value), Source: lineageSource},
		Signals:            facts.Signals,
		Research:           facts.Research,
		Memory:             facts.Memory,
		Findings:           LifecycleFact[[]colony.ReviewLedgerEntry]{Value: append([]colony.ReviewLedgerEntry(nil), facts.Memory.Value.Findings...), Source: findingSource},
		Changes:            changes,
		Evidence:           evidence,
		Verification:       verification,
		Warnings:           lifecycleProjectionWarnings(facts),
		Debt:               debt,
		Blockers:           blockers,
		OwnerDecisions:     ownerDecisions,
		Elapsed:            LifecycleFact[time.Duration]{Value: facts.Timing.Value.Elapsed, Source: lifecycleDerivedSource("elapsed", facts.Timing.Source)},
		ReportedCost:       facts.ReportedCost,
		History:            facts.History,
		NextAction:         action,
		Alternatives:       alternatives,
		StateEffect:        colony.LifecycleStateEffectNone,
		Transaction:        transaction,
		Receipt:            facts.Evidence.Value.Receipt,
		Recovery:           recovery,
		Provenance:         provenance,
		Closure:            closure,
		Sections:           lifecycleProjectionSections(view),
	}
	return projection
}
