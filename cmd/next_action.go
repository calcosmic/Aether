package cmd

// Phase 197 plan 01 -- the single answer to "what next".
//
// Before this file, five separate functions each decided the next step from
// the same saved state and disagreed with one another:
// workflowSuggestionsForState, nextCommandFromState, nextCommandForHookState,
// nextUpSuggestionsForState and closeoutNextCommand. Around two hundred call
// sites then hand-typed advice around them. This is the one decision the rest
// of Phase 197 renders from.
//
// Two properties are load-bearing and are asserted by tests, not by this
// comment:
//
//  1. resolveNextAction is PURE. It reads no file, touches no package store,
//     reads no environment variable and never detects a platform. Everything
//     impure lives in loadNextActionInput (cmd/next_action_input.go), so every
//     branch below is testable without a filesystem.
//
//  2. Nothing this file can say names a command the program does not have.
//     Every command literal the resolver may emit is declared once, in
//     nextActionCandidates, and every emitted command -- including a command
//     read out of a recovery report on disk -- is resolved against the LIVE
//     cobra tree before it can leave. The candidate set records what the
//     resolver may SAY; rootCmd remains the sole authority on what EXISTS.
//
// S-01 binds every command value here to the platform-neutral runtime form
// (`aether continue`, `aether build 3`). A slash spelling is never produced:
// wrappers and the TS host EXECUTE these strings, so `/ant-continue` handed to
// exec is a broken command. Translation to a platform's spelling belongs to the
// visual writer and happens downstream.

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// The answer
// ---------------------------------------------------------------------------

// nextActionContextHealth is the machine-readable verdict on whether the owner
// can safely walk away from this chat. It is deliberately an enumeration plus a
// reason code rather than a sentence: the sentence is a rendering concern and
// belongs to the card, not to the decision.
type nextActionContextHealth string

const (
	// contextHealthKeep -- do not walk away yet; something would be lost.
	contextHealthKeep nextActionContextHealth = "KEEP"
	// contextHealthSafe -- everything is written down; leaving costs nothing.
	contextHealthSafe nextActionContextHealth = "SAFE"
	// contextHealthClearRecommended -- a natural break; a fresh chat is better.
	contextHealthClearRecommended nextActionContextHealth = "CLEAR_RECOMMENDED"
)

// Reason codes for context health. Machine-readable; the card turns them into
// a sentence.
const (
	contextReasonHandoffMissing  = "handoff_not_on_disk"
	contextReasonBuildInProgress = "build_in_progress"
	contextReasonNaturalBreak    = "at_a_natural_break"
	contextReasonHandoffSaved    = "handoff_saved"
)

type nextActionContextVerdict struct {
	Health nextActionContextHealth `json:"health"`
	Reason string                  `json:"reason"`
}

type nextActionAlternative struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
}

type nextActionStanding struct {
	State           string `json:"state"`
	Goal            string `json:"goal"`
	Milestone       string `json:"milestone"`
	CurrentPhase    int    `json:"current_phase"`
	PhaseName       string `json:"phase_name"`
	TotalPhases     int    `json:"total_phases"`
	CompletedPhases int    `json:"completed_phases"`
	Explanation     string `json:"explanation"`
}

type nextActionFlag struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type nextActionOpenItems struct {
	Flags   []nextActionFlag `json:"flags"`
	Signals []string         `json:"signals"`
}

type nextActionRecovery struct {
	Paused      bool   `json:"paused"`
	PausedAt    string `json:"paused_at,omitempty"`
	Blocked     bool   `json:"blocked"`
	ReportPath  string `json:"report_path,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Explanation string `json:"explanation"`
}

// nextAction is the eight-field answer NEXT-01 names. Every field carries a
// json tag because later plans marshal this whole struct into the
// machine-readable envelope -- an untagged field is a field the wrapper cannot
// see.
//
// Notes is diagnostic rather than one of the eight: it records any time the
// availability gate substituted a command, so the owner is never silently
// redirected to something other than what the decision chose.
//
// Memory is likewise not one of the eight (198.2 plan 03). It carries what
// the colony remembers about the owner -- preferences, the strongest learned
// habits, and the last helper's relay note -- for the session-start greeting
// only. It is a straight passthrough of nextActionInput.Memory: the resolver
// composes no memory content of its own, matching the "decides nothing" rule
// this file's header already binds every other field to.
type nextAction struct {
	Standing       nextActionStanding       `json:"standing"`
	Changed        []string                 `json:"changed"`
	Open           nextActionOpenItems      `json:"open"`
	Recommendation string                   `json:"recommendation"`
	Command        string                   `json:"command"`
	Alternatives   []nextActionAlternative  `json:"alternatives"`
	ContextHealth  nextActionContextVerdict `json:"context_health"`
	Recovery       nextActionRecovery       `json:"recovery"`
	Notes          []string                 `json:"notes,omitempty"`
	Memory         nextActionMemory         `json:"memory"`
	// Projection is the authoritative lifecycle answer. The legacy fields
	// above remain as compatibility adapters for callers migrated in later
	// plans; renderers consume this value directly.
	Projection *LifecycleProjection `json:"projection,omitempty"`
}

// nextActionMemory is what the colony remembers about the owner, shown only
// on the session-start greeting (D-12). Every part is optional and omitted
// on its own when empty (D-14) -- there is no "nothing learned yet" filler.
type nextActionMemory struct {
	// Preferences is one line naming how many preferences are saved plus one
	// example, e.g. "4 preferences set -- e.g. plain English replies". Empty
	// when the owner has set none.
	Preferences string
	// Habits is up to three sentences, the strongest learned habits in their
	// own words (loadStrongestRuntimeInstincts, the one ranking rule locked
	// by TestStrongestHabitsHaveOneRankingRule). Empty when the colony has
	// learned nothing yet.
	Habits []string
	// RelayNote is one sentence naming the last helper and what it left for
	// the next one. Empty when no handoff has been recorded.
	RelayNote string
}

// ---------------------------------------------------------------------------
// The input
// ---------------------------------------------------------------------------

// nextActionInput is everything the decision needs, already gathered. The
// resolver never reaches past this struct.
type nextActionInput struct {
	// Facts is the one read-only lifecycle snapshot. State remains below as a
	// compatibility field for callers that already hold an in-memory state;
	// resolveNextAction converts that state to facts exactly once when Facts is
	// absent, then delegates every lifecycle decision to projectLifecycle.
	Facts LifecycleFacts

	// NoColony distinguishes "there is no project here at all" from "a project
	// that has only just started". They lead to completely different advice and
	// conflating them is how a fresh checkout gets told to continue a build
	// that does not exist.
	NoColony bool

	State   colony.ColonyState
	Flags   []colony.FlagEntry
	Signals []string

	// PlanBlocker is the unresolved planning failure, when one is open.
	PlanBlocker *colony.FlagEntry

	// Recovery is the saved report from the last check of a blocked phase.
	Recovery *activeRecoveryGuidance

	// HandoffExists is whether the handoff document is verifiably on disk.
	// This preserves renderContextClearGuidanceForPlatform's rule exactly:
	// "safe to close the chat" is a CLAIM, and it may only be made when the
	// file that makes it true has been seen. Handoff-absent may never resolve
	// to safe.
	HandoffExists bool

	// BuildLooksAbandoned is set by the loader when a build has been sitting
	// past the abandoned-build threshold with no dispatch manifest behind it.
	BuildLooksAbandoned bool

	// LastCommand is the command that just ran, when the caller knows it.
	LastCommand string

	// Override is a command the CALLER knows is right for this exact run and
	// the resolver cannot work out from saved state alone -- the redispatch
	// that picks up only the unfinished half of a part-built phase, the exact
	// command a blocked check named, the clarification a question is waiting
	// on. It is an INPUT to the decision, never a way around it: it is
	// resolved against the live command tree like every other command, and
	// when it does not resolve the ordinary answer is used and the
	// substitution is recorded. Feeding it in here is what keeps the screen
	// and the machine-readable answer one answer rather than two.
	Override *nextActionOverride

	// ActiveTodos are the outstanding task goals for the current phase.
	ActiveTodos []string

	// Memory is what the colony remembers about the owner -- preferences,
	// learned habits, the last helper's relay note -- for the session-start
	// greeting only (198.2 plan 03). Populated ONLY by
	// loadNextActionInputForGreeting; loadNextActionInput and
	// loadNextActionInputForCommand leave it zero, which is what keeps every
	// other closing card byte-identical to before this field existed
	// (TestClosingCardsAreUnchangedByTheGreetingBlock).
	Memory nextActionMemory
}

// nextActionOverride is the caller's own knowledge of this exact run, offered
// to the decision rather than applied after it.
//
// Recommendation is the plain-English reason, written for someone who has never
// opened a file here. When it is empty a general one is used.
type nextActionOverride struct {
	Command        string
	Recommendation string
}

// ---------------------------------------------------------------------------
// The candidate command set -- the ONLY place a command is spelled
// ---------------------------------------------------------------------------

// nextActionCandidateKey identifies one candidate command.
type nextActionCandidateKey string

const (
	candidateInit            nextActionCandidateKey = "init"
	candidateDiscuss         nextActionCandidateKey = "discuss"
	candidateSpec            nextActionCandidateKey = "spec"
	candidateSpecApprove     nextActionCandidateKey = "spec_approve"
	candidateSpecRepair      nextActionCandidateKey = "spec_repair"
	candidatePlan            nextActionCandidateKey = "plan"
	candidatePlanPreset      nextActionCandidateKey = "plan_preset"
	candidatePlanRefresh     nextActionCandidateKey = "plan_preset_refresh"
	candidatePlanCandidate   nextActionCandidateKey = "plan_candidate"
	candidatePlanAcceptExact nextActionCandidateKey = "plan_accept_exact"
	candidatePlanRepair      nextActionCandidateKey = "plan_repair"
	candidateColonize        nextActionCandidateKey = "colonize"
	candidateBuildPhase      nextActionCandidateKey = "build_phase"
	candidateBuildForce      nextActionCandidateKey = "build_force"
	candidateRun             nextActionCandidateKey = "run"
	candidateContinue        nextActionCandidateKey = "continue"
	candidateSeal            nextActionCandidateKey = "seal"
	candidateEntomb          nextActionCandidateKey = "entomb"
	candidateResume          nextActionCandidateKey = "resume"
	candidateResumeDashboard nextActionCandidateKey = "resume_dashboard"
	candidateStatus          nextActionCandidateKey = "status"
	candidateFlags           nextActionCandidateKey = "flags"
	candidateHistory         nextActionCandidateKey = "history"
	candidatePheromones      nextActionCandidateKey = "pheromones"
	candidateFocus           nextActionCandidateKey = "focus"
)

// nextActionCandidate is one command the resolver is allowed to name.
//
// Template is in platform-neutral runtime form and may carry a fmt verb for an
// argument (a phase number). Why is the plain-English reason a non-technical
// reader needs, with every word this repo invented explained beside it.
type nextActionCandidate struct {
	Key      nextActionCandidateKey
	Template string
	Why      string
}

// nextActionCandidates is THE enumerable set of commands the resolver may say.
//
// This is not tidiness. The availability gate DROPS a candidate that does not
// resolve and falls back to one that does -- correct behaviour in a live
// session, and exactly why a sweep over emitted commands cannot catch a rename:
// the sweep would silently measure the fallback and stay green. Because the set
// is enumerable it can be walked and resolved BEFORE any gating, which is what
// turns a rename into a red build (TestEveryResolverCandidateResolves).
//
// No command name may be written inline at a branch. A branch picks a key; the
// set is the only place a command is spelled.
var nextActionCandidates = []nextActionCandidate{
	{
		Key:      candidateInit,
		Template: `aether init "goal"`,
		Why:      "Start a new project by naming, in one sentence, what you want built.",
	},
	{
		Key:      candidateDiscuss,
		Template: "aether discuss",
		Why:      "Talk the goal through first, so the plan is built on your answers instead of a guess.",
	},
	{
		Key:      candidateSpec,
		Template: "aether spec",
		Why:      "Review the readable specification and explicitly approve or revise that exact contract.",
	},
	{
		Key:      candidateSpecApprove,
		Template: "aether spec --approve --revision-id %s --revision-hash %s --approval-token '%s'",
		Why:      "Approve only the exact readable specification revision and content hash shown by inspection.",
	},
	{
		Key:      candidateSpecRepair,
		Template: "aether spec --repair-projection",
		Why:      "Restore the readable specification projection from canonical state without changing its authority.",
	},
	{
		Key:      candidatePlan,
		Template: "aether plan",
		Why:      "Break the goal into numbered phases you can build one at a time.",
	},
	{
		Key:      candidatePlanPreset,
		Template: "aether plan --preset %s",
		Why:      "Choose Fast, Balanced, Deep, or Exhaustive before any planning worker is authorized.",
	},
	{
		Key:      candidatePlanRefresh,
		Template: "aether plan --refresh --preset %s",
		Why:      "Refresh planning from current repository evidence using the owner's selected preset.",
	},
	{
		Key:      candidatePlanCandidate,
		Template: "aether plan --candidate",
		Why:      "Review the stopped plan candidate and its evidence; it is not active or buildable yet.",
	},
	{
		Key:      candidatePlanAcceptExact,
		Template: "aether plan --accept-candidate %s",
		Why:      "Accept only the named candidate through the exact binding checks shown by its review.",
	},
	{
		Key:      candidatePlanRepair,
		Template: "aether plan --repair-artifact",
		Why:      "Fix a plan that refers to a phase which is not in it, without redoing the planning.",
	},
	{
		Key:      candidateColonize,
		Template: "aether colonize",
		Why:      "Scan the existing code first, so planning starts from what is really here.",
	},
	{
		Key:      candidateBuildPhase,
		Template: "aether build %d",
		Why:      "Start the next phase: helpers write the code for it.",
	},
	{
		Key:      candidateBuildForce,
		Template: "aether build %d --force",
		Why:      "Restart a phase that stopped part-way, replacing the abandoned attempt.",
	},
	{
		Key:      candidateRun,
		Template: "aether run",
		Why:      "Run every remaining accepted phase in Autopilot.",
	},
	{
		Key:      candidateContinue,
		Template: "aether continue",
		Why:      "Check the work that was just built and move on to the next phase if it holds up.",
	},
	{
		Key:      candidateSeal,
		Template: "aether seal",
		Why:      "Mark the project finished and write down what was learned. Nothing is deleted.",
	},
	{
		Key:      candidateEntomb,
		Template: "aether entomb",
		Why:      "File the finished project away in the archive so it stops appearing as active work.",
	},
	{
		Key:      candidateResume,
		Template: "aether resume",
		Why:      "Reload where things stand, so you can carry on without repeating yourself.",
	},
	{
		// A read-only, non-mutating look at where things stand. It is distinct
		// from candidateResume, which is the sole lifecycle restoration route.
		Key:      candidateResumeDashboard,
		Template: "aether resume-dashboard",
		Why:      "Look at a quick view of where things stand, without restoring anything.",
	},
	{
		Key:      candidateStatus,
		Template: "aether status",
		Why:      "Look at the dashboard first, without changing anything.",
	},
	{
		Key:      candidateFlags,
		Template: "aether flags --status active",
		Why:      "Read the open problems waiting on a decision from you.",
	},
	{
		Key:      candidateHistory,
		Template: "aether history",
		Why:      "Read back what has happened on this project so far.",
	},
	{
		Key:      candidatePheromones,
		Template: "aether pheromones",
		Why:      "See the standing instructions you have given the helpers -- what to focus on and what to avoid.",
	},
	{
		Key:      candidateFocus,
		Template: `aether focus "area"`,
		Why:      "Point the helpers at one area before the next build starts.",
	},
}

func nextActionCandidateFor(key nextActionCandidateKey) (nextActionCandidate, bool) {
	for _, candidate := range nextActionCandidates {
		if candidate.Key == key {
			return candidate, true
		}
	}
	return nextActionCandidate{}, false
}

// candidateCommand formats a candidate from the set. It is the only way a
// command string is produced inside the resolver.
func candidateCommand(key nextActionCandidateKey, args ...interface{}) (string, string, bool) {
	candidate, ok := nextActionCandidateFor(key)
	if !ok {
		return "", "", false
	}
	command := candidate.Template
	if len(args) > 0 {
		command = fmt.Sprintf(candidate.Template, args...)
	}
	return command, candidate.Why, true
}

// availableCandidateCommand is the narrow adapter for command results that
// already know their exact authority boundary (for example an exact
// Specification approval receipt). Command spelling still comes from the one
// enumerable candidate set and still passes through the live Cobra gate.
func availableCandidateCommand(key nextActionCandidateKey, args ...interface{}) string {
	command, _, ok := candidateCommand(key, args...)
	if !ok {
		return ""
	}
	command, ok = availableCommand(command)
	if !ok {
		return ""
	}
	return command
}

// nextActionForCandidateOverride resolves a run-specific boundary through the
// same pure next-action decision used by lifecycle cards and JSON envelopes.
// The caller chooses only a candidate key and its plain-English reason; it
// cannot introduce another hand-typed command spelling.
func nextActionForCandidateOverride(state colony.ColonyState, lastCommand string, key nextActionCandidateKey, recommendation string, args ...interface{}) nextAction {
	state = normalizeLegacyColonyState(state)
	in := nextActionInput{
		State:       state,
		LastCommand: strings.TrimSpace(lastCommand),
		NoColony:    colonyStateIsUnstarted(state),
	}
	if command := availableCandidateCommand(key, args...); command != "" {
		in.Override = &nextActionOverride{Command: command, Recommendation: strings.TrimSpace(recommendation)}
	}
	return resolveNextAction(in)
}

func lifecycleActionFromCandidate(id string, key nextActionCandidateKey, reason string, evidence []colony.LifecycleEvidence, args ...interface{}) LifecycleProjectedAction {
	command, fallbackReason, _ := candidateCommand(key, args...)
	if strings.TrimSpace(reason) == "" {
		reason = fallbackReason
	}
	return lifecycleAction(id, command, reason, evidence)
}

func lifecycleChoiceFromCandidate(id string, key nextActionCandidateKey, reason string, args ...interface{}) LifecycleActionChoice {
	command, fallbackReason, _ := candidateCommand(key, args...)
	if strings.TrimSpace(reason) == "" {
		reason = fallbackReason
	}
	return lifecycleChoice(id, command, reason)
}

func lifecycleInspectionChoices() []LifecycleActionChoice {
	return []LifecycleActionChoice{
		lifecycleChoiceFromCandidate("status", candidateStatus, ""),
		lifecycleChoiceFromCandidate("history", candidateHistory, ""),
	}
}

func lifecycleSafeCommandToken(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && len(strings.Fields(value)) == 1
}

// lifecycleAuthorityNextAction is the Phase-200 authority state table. It is
// deliberately pure and lives beside the resolver's enumerable command set;
// lifecycleProjectionDecision calls it before ordinary completion/build
// routing so every surface receives the same answer.
func lifecycleAuthorityNextAction(facts LifecycleFacts, evidence []colony.LifecycleEvidence) (LifecycleProjectedAction, []LifecycleActionChoice, colony.OutcomeKind, bool) {
	intent := facts.Intent.Value
	specification := facts.Specification.Value
	planning := facts.Planning.Value

	if intent.CharterAccepted && (intent.UnresolvedDiscussionCount > 0 || !specification.Present) {
		return lifecycleActionFromCandidate(
			"discuss",
			candidateDiscuss,
			"The project goal is accepted, but its material intent must be settled before the specification or plan can advance.",
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	}

	if specification.Present && (!specification.Approved || specification.Status != colony.SpecStatusApproved) {
		return lifecycleActionFromCandidate(
				"specification",
				candidateSpec,
				"The current specification is not approved; review or revise that exact contract before planning or building.",
				evidence,
			), []LifecycleActionChoice{
				lifecycleChoiceFromCandidate("discuss", candidateDiscuss, "Reopen the owner intent behind this draft."),
				lifecycleChoiceFromCandidate("status", candidateStatus, ""),
				lifecycleChoiceFromCandidate("history", candidateHistory, ""),
			}, colony.OutcomeKindNoChange, true
	}

	if planning.Stage == string(planningStageSpecApprovalRequired) {
		return lifecycleActionFromCandidate(
			"specification",
			candidateSpec,
			"Planning found a contract change; approve or revise the exact successor specification before this run can resume.",
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	}

	if len(planning.WaitingCandidateIDs) > 1 {
		// A planning restart leaves the earlier plan waiting beside the new
		// one. The review names each and how to review it by name; no
		// acceptance is offered here, because choosing is the owner's.
		return lifecycleActionFromCandidate(
			"review_plan_candidate",
			candidatePlanCandidate,
			fmt.Sprintf("%d plans are waiting for review. The review names each one and how to review it by name; neither becomes active until you accept one.", len(planning.WaitingCandidateIDs)),
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	}

	switch planning.PendingCandidateStanding {
	case planCandidateStandingStale, planCandidateStandingExpired:
		recovery := strings.TrimSpace(planning.PendingCandidateRecoveryCommand)
		if recovery == "" {
			recovery = planCandidateRefreshCommand
		}
		return lifecycleAction(
			"refresh_plan_candidate",
			recovery,
			fmt.Sprintf("The latest plan candidate is %s (%s). Repository state is %s; refresh from current evidence before accepting or building.", planning.PendingCandidateStanding, planning.PendingCandidateWhyUnavailable, planning.PendingCandidateStateEffect),
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	case planCandidateStandingCurrent:
		if planning.PendingCandidateID != "" {
			alternatives := make([]LifecycleActionChoice, 0, 3)
			if planning.PendingCandidateAcceptanceAvailable && strings.TrimSpace(planning.PendingCandidateAcceptanceCommand) != "" {
				alternatives = append(alternatives, lifecycleChoice(
					"accept_plan_candidate",
					planning.PendingCandidateAcceptanceCommand,
					"Accept this candidate only through the exact specification, base-plan, timeline, proposal, and token bindings captured with its current standing.",
				))
			}
			alternatives = append(alternatives, lifecycleInspectionChoices()...)
			return lifecycleActionFromCandidate(
				"review_plan_candidate",
				candidatePlanCandidate,
				"Planning stopped with a current reviewable candidate. It remains inactive until you explicitly accept the exact captured bindings.",
				evidence,
			), alternatives, colony.OutcomeKindNoChange, true
		}
	}

	// Compatibility path for legacy snapshots that predate captured standing.
	if planning.PendingCandidateStanding == "" && planning.PendingCandidateID != "" && (planning.PendingCandidateStatus == colony.PlanCandidatePendingReview || planning.Stage == string(planningStageCandidateReady)) {
		alternatives := make([]LifecycleActionChoice, 0, 3)
		if lifecycleSafeCommandToken(planning.PendingCandidateID) {
			alternatives = append(alternatives, lifecycleChoiceFromCandidate(
				"accept_plan_candidate",
				candidatePlanAcceptExact,
				"Accept this candidate only after reviewing the exact specification, base-plan, timeline, and proposal bindings.",
				planning.PendingCandidateID,
			))
		}
		alternatives = append(alternatives, lifecycleInspectionChoices()...)
		return lifecycleActionFromCandidate(
			"review_plan_candidate",
			candidatePlanCandidate,
			"Planning stopped with a reviewable candidate. It remains inactive until you explicitly accept that exact candidate.",
			evidence,
		), alternatives, colony.OutcomeKindNoChange, true
	}

	switch planning.Stage {
	case string(planningStageOwnerDecision):
		return lifecycleActionFromCandidate(
			"planning_owner_decision",
			candidatePlan,
			"Planning is paused at a material owner decision; reopen the bound planning run to answer it and resume the exact frontier.",
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	case string(planningStageReconciliationRequired):
		return lifecycleActionFromCandidate(
				"reconcile_plan",
				candidatePlan,
				"The approved specification changed executable scope; resume planning to reconcile only the affected work.",
				evidence,
			), []LifecycleActionChoice{
				lifecycleChoiceFromCandidate("specification", candidateSpec, "Review the approved specification and its impact first."),
				lifecycleChoiceFromCandidate("status", candidateStatus, ""),
				lifecycleChoiceFromCandidate("history", candidateHistory, ""),
			}, colony.OutcomeKindNoChange, true
	case string(planningStageCandidateReady):
		return lifecycleActionFromCandidate(
			"review_plan_candidate",
			candidatePlanCandidate,
			"Planning stopped at a candidate boundary; review the candidate before any execution command is available.",
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindNoChange, true
	case "", string(planningStageAccepted):
		// No active stage remains. Durable acceptance or legacy classification
		// below decides whether execution is available.
	default:
		return lifecycleActionFromCandidate(
			"resume_planning",
			candidatePlan,
			fmt.Sprintf("Planning run %s is at %s; reopen it to resume that exact saved stage.", planning.RunID, planning.Stage),
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindInProgress, true
	}

	if planning.AcceptanceBindingStatus == LifecyclePlanBindingAffected || (planning.AcceptedPlan && len(planning.AffectedUnresolvedSemanticIDs) > 0) {
		return lifecycleActionFromCandidate(
				"reconcile_plan",
				candidatePlan,
				"The accepted plan has affected unfinished scope; reconcile it through a new candidate before building or sealing.",
				evidence,
			), []LifecycleActionChoice{
				lifecycleChoiceFromCandidate("specification", candidateSpec, "Review the specification revision that affected this plan."),
				lifecycleChoiceFromCandidate("status", candidateStatus, ""),
				lifecycleChoiceFromCandidate("history", candidateHistory, ""),
			}, colony.OutcomeKindNoChange, true
	}

	if planning.AcceptanceBindingStatus == LifecyclePlanBindingInvalid {
		return lifecycleActionFromCandidate(
			"resume",
			candidateResume,
			"The active plan does not carry a complete valid acceptance binding; resume must reconcile the retained evidence.",
			evidence,
		), lifecycleInspectionChoices(), colony.OutcomeKindRecoveryRequired, true
	}

	if specification.Approved && !planning.AcceptedPlan && !planning.LegacyUnbound {
		return lifecycleActionFromCandidate(
				"plan",
				candidatePlan,
				"The specification is approved, but no plan candidate has been accepted yet.",
				evidence,
			), []LifecycleActionChoice{
				lifecycleChoiceFromCandidate("specification", candidateSpec, "Review the approved contract before planning."),
				lifecycleChoiceFromCandidate("status", candidateStatus, ""),
				lifecycleChoiceFromCandidate("history", candidateHistory, ""),
			}, colony.OutcomeKindNoChange, true
	}

	return LifecycleProjectedAction{}, nil, "", false
}

// ---------------------------------------------------------------------------
// The availability gate -- criterion 6
// ---------------------------------------------------------------------------

// findChildByExactName returns the child command registered under exactly this
// name or one of its declared aliases. It deliberately does no prefix or fuzzy
// matching: a name that is merely a prefix of a real command is not a real
// command, and advising one would be the same defect as advising a deleted one.
func findChildByExactName(parent *cobra.Command, name string) *cobra.Command {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Commands() {
		if child.Name() == name {
			return child
		}
		for _, alias := range child.Aliases {
			if alias == name {
				return child
			}
		}
	}
	return nil
}

// availableCommand resolves a candidate command in runtime form against the
// LIVE cobra tree and returns it only if the verb it names is registered.
//
// S-02: the command tree is the authority, never a committed list. A list is
// the defect this phase exists to close.
//
// Arguments survive untouched: only the leading verb path is resolved, so
// `aether build 3` is checked on `build` and keeps its `3`.
func availableCommand(candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", false
	}
	fields := strings.Fields(candidate)
	if len(fields) < 2 || fields[0] != "aether" {
		return "", false
	}
	tokens := fields[1:]

	// Exact walk: consume leading verb tokens through children matched by name
	// or alias only. This is the anti-fuzzy guard -- it can never accept a
	// prefix or a suggestion.
	walked := rootCmd
	consumed := 0
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			break
		}
		child := findChildByExactName(walked, token)
		if child == nil {
			break
		}
		walked = child
		consumed++
	}
	if consumed == 0 || walked == rootCmd {
		return "", false
	}

	// Cross-check against cobra's own resolution, the same way the orphan
	// reachability ratchet does. If the two disagree, refuse: something in the
	// tree is not what it appears to be.
	target, _, err := rootCmd.Find(tokens)
	if err != nil || target == nil || target == rootCmd || target != walked {
		return "", false
	}
	return candidate, true
}

// ---------------------------------------------------------------------------
// The resolver
// ---------------------------------------------------------------------------

// nextActionFacts returns the authoritative aggregate supplied by the loader.
// Callers that already own a just-written in-memory state are adapted to the
// same shape with explicit unavailable provenance for domains they did not
// observe.
func nextActionFacts(in nextActionInput) LifecycleFacts {
	facts := in.Facts
	if facts.State.Source.Domain == "" {
		facts = lifecycleFactsFromStateSnapshot(in.State, in.NoColony, time.Time{})
	}

	flags := append([]colony.FlagEntry(nil), facts.Blockers.Value...)
	for _, flag := range in.Flags {
		flags = appendLifecycleFlagOnce(flags, flag)
	}
	if in.PlanBlocker != nil {
		flags = appendLifecycleFlagOnce(flags, *in.PlanBlocker)
	}
	if in.Recovery != nil {
		flags = appendLifecycleFlagOnce(flags, colony.FlagEntry{
			ID:          "active-recovery",
			Type:        "blocker",
			Description: strings.TrimSpace(in.Recovery.Summary),
			Source:      in.Recovery.ReportPath,
		})
	}
	if in.BuildLooksAbandoned {
		flags = appendLifecycleFlagOnce(flags, colony.FlagEntry{
			ID:          "abandoned-build-evidence",
			Type:        "blocker",
			Description: "the saved execution has no live dispatch evidence",
			Source:      "build manifest",
		})
	}
	facts.Blockers.Value = flags
	return facts
}

func appendLifecycleFlagOnce(flags []colony.FlagEntry, candidate colony.FlagEntry) []colony.FlagEntry {
	for _, existing := range flags {
		if candidate.ID != "" && existing.ID == candidate.ID {
			return flags
		}
	}
	return append(flags, candidate)
}

func applyNextActionDetailOverride(projection LifecycleProjection, override *nextActionOverride) LifecycleProjection {
	if override == nil {
		return projection
	}
	command := strings.TrimSpace(override.Command)
	if command == "" {
		return projection
	}
	if _, ok := availableCommand(command); !ok {
		return projection
	}
	reason := strings.TrimSpace(override.Recommendation)
	if reason == "" {
		reason = "This command exposes the additional detail produced by the run that just finished."
	}
	projection.NextAction = lifecycleAction("command_detail", command, reason, projection.Evidence)
	projection.Alternatives = nil
	projection.NextAction, projection.Alternatives = lifecycleApplyPlatform(
		projection.NextAction,
		projection.Alternatives,
		"codex",
	)
	return projection
}

func gateLifecycleProjection(projection LifecycleProjection) (LifecycleProjection, []string) {
	var notes []string
	if len(projection.NextAction.Choices) > 0 {
		choices := make([]LifecycleActionChoice, 0, len(projection.NextAction.Choices))
		for _, choice := range projection.NextAction.Choices {
			if command, ok := availableCommand(choice.RuntimeCommand); ok {
				choice.RuntimeCommand = command
				choice.DisplayCommand = command
				choices = append(choices, choice)
			}
		}
		if len(choices) != len(projection.NextAction.Choices) {
			notes = append(notes, "One of the projected execution modes is not available in this runtime; inspect status before continuing.")
			projection.NextAction = lifecycleAction(
				"inspect",
				"aether status",
				"The accepted execution modes are not both available in this runtime.",
				projection.Evidence,
			)
		} else {
			projection.NextAction.Choices = choices
		}
	}

	if command := strings.TrimSpace(projection.NextAction.RuntimeCommand); command != "" {
		if gated, ok := availableCommand(command); ok {
			projection.NextAction.RuntimeCommand = gated
			projection.NextAction.DisplayCommand = gated
		} else {
			notes = append(notes, fmt.Sprintf("%q is not available in this runtime; the read-only status view was substituted.", command))
			projection.NextAction = lifecycleAction(
				"inspect",
				"aether status",
				"The projected command is unavailable in this runtime, so inspect the retained state.",
				projection.Evidence,
			)
			projection.NextAction.DisplayCommand = projection.NextAction.RuntimeCommand
		}
	}

	alternatives := make([]LifecycleActionChoice, 0, len(projection.Alternatives))
	for _, alternative := range projection.Alternatives {
		if gated, ok := availableCommand(alternative.RuntimeCommand); ok {
			alternative.RuntimeCommand = gated
			alternative.DisplayCommand = gated
			alternatives = append(alternatives, alternative)
		}
	}
	projection.Alternatives = alternatives
	return projection, notes
}

func legacyAlternativesFromProjection(projection LifecycleProjection) []nextActionAlternative {
	result := make([]nextActionAlternative, 0, len(projection.Alternatives))
	for _, alternative := range projection.Alternatives {
		result = append(result, nextActionAlternative{
			Command:     alternative.RuntimeCommand,
			Explanation: alternative.Reason,
		})
	}
	return result
}

// resolveNextAction retains the established reporting envelope while delegating
// the lifecycle decision itself to projectLifecycle. It reads nothing.
func resolveNextAction(in nextActionInput) nextAction {
	facts := nextActionFacts(in)
	state := facts.State.Value
	reportInput := in
	reportInput.State = state
	reportInput.NoColony = facts.State.Source.Provenance == LifecycleFactMissing
	if len(reportInput.Flags) == 0 {
		reportInput.Flags = append([]colony.FlagEntry(nil), facts.Blockers.Value...)
	}

	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	projection = applyNextActionDetailOverride(projection, in.Override)
	projection, notes := gateLifecycleProjection(projection)

	answer := nextAction{
		Standing:       standingFromInput(reportInput, state),
		Changed:        changedFromInput(reportInput, state),
		Open:           openItemsFromInput(reportInput),
		Recovery:       recoveryFromInput(reportInput, state),
		ContextHealth:  contextHealthFromInput(reportInput, state),
		Memory:         in.Memory,
		Projection:     &projection,
		Recommendation: projection.NextAction.Reason,
		Command:        projection.NextAction.RuntimeCommand,
		Alternatives:   legacyAlternativesFromProjection(projection),
		Notes:          notes,
	}
	return answer
}

// ---------------------------------------------------------------------------
// The reporting fields
// ---------------------------------------------------------------------------

func standingFromInput(in nextActionInput, state colony.ColonyState) nextActionStanding {
	if in.NoColony {
		return nextActionStanding{
			State:       "none",
			Explanation: "No Aether project has been set up in this folder.",
		}
	}

	standing := nextActionStanding{
		State:           string(state.State),
		Milestone:       strings.TrimSpace(state.Milestone),
		CurrentPhase:    state.CurrentPhase,
		TotalPhases:     len(state.Plan.Phases),
		CompletedPhases: completedPhaseCount(state),
	}
	if state.Goal != nil {
		standing.Goal = strings.TrimSpace(*state.Goal)
	}
	if phase := recoveryPhase(&state); phase != nil {
		standing.PhaseName = strings.TrimSpace(phase.Name)
		if standing.CurrentPhase == 0 {
			standing.CurrentPhase = phase.ID
		}
	}

	switch {
	case standing.TotalPhases == 0:
		standing.Explanation = "The goal is saved. No phases have been drawn up yet."
	case standing.PhaseName != "":
		standing.Explanation = fmt.Sprintf("Phase %d of %d: %s. %d phase(s) finished so far.",
			standing.CurrentPhase, standing.TotalPhases, standing.PhaseName, standing.CompletedPhases)
	default:
		standing.Explanation = fmt.Sprintf("%d of %d phases finished.",
			standing.CompletedPhases, standing.TotalPhases)
	}
	return standing
}

func changedFromInput(in nextActionInput, state colony.ColonyState) []string {
	changed := []string{}
	if command := strings.TrimSpace(in.LastCommand); command != "" {
		changed = append(changed, "The last thing you ran was "+lastCommandPlainEnglish(command)+".")
	}
	for _, event := range lastEventTexts(state.Events, 3) {
		if sentence := nextActionEventSentence(event); sentence != "" {
			changed = append(changed, sentence)
		}
	}
	if len(changed) == 0 {
		changed = append(changed, "Nothing has changed since this project was last saved.")
	}
	return changed
}

// lastCommandPlainEnglish names the last-run command, adding a same-sentence
// explanation when the command's own name is also a word this repo invented
// (CLAUDE.md's vocabulary table) -- "seal" both names a command and requires
// one of its own explanatory cues nearby, which a bare command name cannot
// carry on its own (Phase "Classic Visual Voice" plan 04).
func lastCommandPlainEnglish(command string) string {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "seal":
		return "seal (marking the project finished)"
	default:
		return command
	}
}

// nextActionEventSentence turns one saved event into something a person can
// read. Events are stored as `timestamp|event_type|source|message`, which is the
// project's own record-keeping format: the timestamp and the internal code mean
// nothing to the owner, and printing them raw is the same defect as printing a
// word this repository invented without saying what it means. The message is
// kept; the bookkeeping is dropped.
//
// It also keeps this list stable: a timestamp in an owner-facing line is a value
// that differs on every run, which no recorded transcript can hold.
func nextActionEventSentence(event string) string {
	event = strings.TrimSpace(event)
	if event == "" {
		return ""
	}
	parts := strings.Split(event, "|")
	return strings.TrimSpace(parts[len(parts)-1])
}

func openItemsFromInput(in nextActionInput) nextActionOpenItems {
	items := nextActionOpenItems{
		Flags:   []nextActionFlag{},
		Signals: []string{},
	}
	for _, flag := range in.Flags {
		if flag.Resolved {
			continue
		}
		items.Flags = append(items.Flags, nextActionFlag{
			ID:          strings.TrimSpace(flag.ID),
			Type:        strings.TrimSpace(flag.Type),
			Description: compactActionText(flag.Description, 160),
		})
	}
	for _, signal := range in.Signals {
		if signal = strings.TrimSpace(signal); signal != "" {
			items.Signals = append(items.Signals, signal)
		}
	}
	return items
}

func recoveryFromInput(in nextActionInput, state colony.ColonyState) nextActionRecovery {
	recovery := nextActionRecovery{
		Paused:      state.Paused,
		Explanation: "Nothing is paused or blocked.",
	}
	if state.PausedAt != nil {
		recovery.PausedAt = strings.TrimSpace(*state.PausedAt)
	}
	if in.Recovery != nil {
		recovery.Blocked = true
		recovery.ReportPath = strings.TrimSpace(in.Recovery.ReportPath)
		recovery.Summary = compactActionText(in.Recovery.Summary, 240)
	}

	switch {
	case recovery.Paused && recovery.Blocked:
		recovery.Explanation = "This project is paused, and the phase it stopped on is blocked."
	case recovery.Paused:
		recovery.Explanation = "This project is paused. Nothing is running."
	case recovery.Blocked:
		recovery.Explanation = "The last check on this phase did not pass, and the report saying why is saved."
	}
	return recovery
}

// contextHealthFromInput preserves renderContextClearGuidanceForPlatform's rule
// exactly: "safe to close the chat" may only be claimed when the handoff file is
// verifiably on disk. A softer wording here would be a regression, not a style
// change -- the whole point is that the claim is only made when the file that
// makes it true has been seen.
func contextHealthFromInput(in nextActionInput, state colony.ColonyState) nextActionContextVerdict {
	if !in.HandoffExists {
		return nextActionContextVerdict{Health: contextHealthKeep, Reason: contextReasonHandoffMissing}
	}
	if state.State == colony.StateEXECUTING {
		return nextActionContextVerdict{Health: contextHealthKeep, Reason: contextReasonBuildInProgress}
	}
	if state.State == colony.StateBUILT || state.State == colony.StateCOMPLETED || colonyNeedsEntomb(state) {
		return nextActionContextVerdict{Health: contextHealthClearRecommended, Reason: contextReasonNaturalBreak}
	}
	return nextActionContextVerdict{Health: contextHealthSafe, Reason: contextReasonHandoffSaved}
}

func allPhasesCompleteInPlan(state colony.ColonyState) bool {
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
