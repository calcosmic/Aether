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
	candidatePlan            nextActionCandidateKey = "plan"
	candidatePlanRepair      nextActionCandidateKey = "plan_repair"
	candidateColonize        nextActionCandidateKey = "colonize"
	candidateBuildPhase      nextActionCandidateKey = "build_phase"
	candidateBuildForce      nextActionCandidateKey = "build_force"
	candidateContinue        nextActionCandidateKey = "continue"
	candidateSeal            nextActionCandidateKey = "seal"
	candidateEntomb          nextActionCandidateKey = "entomb"
	candidateResume          nextActionCandidateKey = "resume"
	candidateResumeColony    nextActionCandidateKey = "resume_colony"
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
		Key:      candidatePlan,
		Template: "aether plan",
		Why:      "Break the goal into numbered phases you can build one at a time.",
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
		Key:      candidateResumeColony,
		Template: "aether resume-colony",
		Why:      "Reload the fuller picture: the saved notes, the open questions and the task list.",
	},
	{
		// A read-only, non-mutating look at where things stand. Genuinely
		// distinct from candidateResume: "aether resume" and "aether
		// resume-colony" are two names for the exact same command (resume is
		// a declared Cobra alias of resume-colony), so offering both as if
		// they were different choices recommends one thing twice. This is the
		// quick view the pause card's "without the detail" alternative always
		// meant.
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

type nextActionChoice struct {
	key            nextActionCandidateKey
	args           []interface{}
	recommendation string
	// literal, when set, is a command that came from saved data rather than
	// from the candidate set (a recovery report names its own exact command).
	// It is gated exactly like any other candidate, and when it fails to
	// resolve the fallback below is used and the substitution is recorded.
	literal string
	// fallback is the key used when the primary choice does not resolve.
	fallback     nextActionCandidateKey
	fallbackArgs []interface{}
	alternatives []nextActionAlternativeChoice
}

type nextActionAlternativeChoice struct {
	key     nextActionCandidateKey
	args    []interface{}
	literal string
	why     string
}

// resolveNextAction is the one decision. Given the project's saved state it
// answers all eight of the owner's questions. It reads nothing.
func resolveNextAction(in nextActionInput) nextAction {
	state := normalizeLegacyColonyState(in.State)

	answer := nextAction{
		Standing:      standingFromInput(in, state),
		Changed:       changedFromInput(in, state),
		Open:          openItemsFromInput(in),
		Recovery:      recoveryFromInput(in, state),
		ContextHealth: contextHealthFromInput(in, state),
		// Straight passthrough -- see the doc comment on nextAction.Memory.
		// loadNextActionInput / loadNextActionInputForCommand leave in.Memory
		// zero, so every closing card but the greeting is unchanged.
		Memory: in.Memory,
	}

	choice := chooseNextAction(in, state)
	answer.Recommendation = choice.recommendation

	command, notes := gateChoice(choice)
	answer.Command = command
	answer.Notes = notes
	answer.Alternatives = gateAlternatives(choice.alternatives, command)

	return answer
}

// chooseNextAction is the ordered decision, merging the branches the five
// existing deciders each implement partially. Most specific first.
func chooseNextAction(in nextActionInput, state colony.ColonyState) nextActionChoice {
	if in.NoColony {
		return nextActionChoice{
			key: candidateInit,
			recommendation: "There is no Aether project set up in this folder yet. " +
				"Start one by saying, in a sentence, what you want built.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidateColonize},
				{key: candidateStatus},
			},
		}
	}

	// The caller's own knowledge of this exact run outranks everything the
	// saved state can say, because the saved state cannot know it: which half
	// of a part-built phase is still unwritten, or the exact command the last
	// check named. It is still gated like any other command below, so an
	// override naming something this version does not have falls back to the
	// ordinary answer rather than telling the owner to type a command that
	// does not exist.
	if in.Override != nil && strings.TrimSpace(in.Override.Command) != "" {
		recommendation := strings.TrimSpace(in.Override.Recommendation)
		if recommendation == "" {
			recommendation = "This run left something specific to do next, and this command does exactly that " +
				"rather than starting the general next step."
		}
		return nextActionChoice{
			literal:        strings.TrimSpace(in.Override.Command),
			fallback:       candidateContinue,
			recommendation: recommendation,
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateFlags},
			},
		}
	}

	// A sealed project -- one that has reached the final milestone, named
	// "Crowned Anthill" in this project's vocabulary -- has nothing left to
	// build. The only step left is filing it away.
	if colonyNeedsEntomb(state) {
		return nextActionChoice{
			key: candidateEntomb,
			recommendation: "This project is finished and signed off. The last step is to file it away in the " +
				"archive, which takes it off the active list without deleting anything.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateInit},
			},
		}
	}

	if state.Paused {
		return nextActionChoice{
			key: candidateResume,
			recommendation: "You paused this project. Picking it back up reloads everything that was " +
				"in progress and makes it runnable again.",
			// candidateResumeColony is NOT offered here: "aether resume" and
			// "aether resume-colony" are two names for the same command (a
			// declared Cobra alias), so pairing them recommends one thing
			// twice while calling it a different view. candidateResumeDashboard
			// is genuinely different -- a read-only look, not the same resume.
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateResumeDashboard},
			},
		}
	}

	if in.PlanBlocker != nil {
		description := compactActionText(in.PlanBlocker.Description, 120)
		if description == "" {
			description = "planning did not finish cleanly"
		}
		return nextActionChoice{
			key: candidateFlags,
			recommendation: "Planning stopped on a problem that needs you: " + description + ". " +
				"Read the open problem list before starting any build, because building on a broken " +
				"plan wastes the run.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidatePlanRepair},
				{key: candidateStatus},
			},
		}
	}

	if len(state.Plan.Phases) == 0 {
		return nextActionChoice{
			key: candidateDiscuss,
			recommendation: "The goal is saved but there is no plan yet. The next step is a short " +
				"conversation to pin down what you actually want, so the phases are built on your " +
				"answers rather than on a guess.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidatePlan},
				{key: candidateColonize},
				{key: candidateStatus},
			},
		}
	}

	// A phase that failed outranks everything below it: there is no point
	// starting new work on top of work that is known to be broken.
	for _, phase := range state.Plan.Phases {
		if phase.Status == "failed" {
			return nextActionChoice{
				key:  candidateBuildPhase,
				args: []interface{}{phase.ID},
				recommendation: fmt.Sprintf(
					"Phase %d (%s) failed. The next step is to retry it -- the helpers start from "+
						"what is already there rather than from scratch.", phase.ID, phase.Name),
				alternatives: []nextActionAlternativeChoice{
					{key: candidateStatus},
					{key: candidateFlags},
					{key: candidateHistory},
				},
			}
		}
	}

	// Every phase done, but the project has not been signed off yet. Note that
	// this deliberately says "sign it off", not "file it away": archiving a
	// project that was never signed off skips the step that records what was
	// learned. workflowSuggestionsForState used to advise archiving here; that
	// was the drift.
	if allPhasesCompleteInPlan(state) && !colonyNeedsEntomb(state) {
		return nextActionChoice{
			key: candidateSeal,
			recommendation: "Every planned phase is finished. Signing the project off marks it complete " +
				"and writes down what was learned so later projects reuse it. Nothing is deleted.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateHistory},
			},
		}
	}

	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.State == colony.StateEXECUTING && state.BuildStartedAt == nil && state.CurrentPhase > 0 {
			return nextActionChoice{
				key:  candidateBuildForce,
				args: []interface{}{state.CurrentPhase},
				recommendation: fmt.Sprintf(
					"Phase %d was interrupted before it did any work. Restarting it replaces the "+
						"abandoned attempt rather than running alongside it.", state.CurrentPhase),
				alternatives: []nextActionAlternativeChoice{
					{key: candidateStatus},
					{key: candidateResumeColony},
				},
			}
		}
		if in.BuildLooksAbandoned && state.CurrentPhase > 0 {
			return nextActionChoice{
				key:  candidateBuildForce,
				args: []interface{}{state.CurrentPhase},
				recommendation: fmt.Sprintf(
					"Phase %d has been sitting unfinished for a long time with nothing running behind "+
						"it. Restarting it replaces the stalled attempt.", state.CurrentPhase),
				alternatives: []nextActionAlternativeChoice{
					{key: candidateStatus},
					{key: candidateResumeColony},
				},
			}
		}
		if in.Recovery != nil && in.Recovery.HasTargetedRoute {
			alternatives := []nextActionAlternativeChoice{{key: candidateStatus}}
			if reconcile := strings.TrimSpace(in.Recovery.Recovery.ReconcileCommand); reconcile != "" && reconcile != in.Recovery.Next {
				alternatives = append(alternatives, nextActionAlternativeChoice{
					literal: reconcile,
					why:     "Use this instead if the code already landed and only the record of it is behind.",
				})
			}
			alternatives = append(alternatives, nextActionAlternativeChoice{key: candidateFlags})
			summary := compactActionText(in.Recovery.Summary, 160)
			if summary != "" {
				summary = " " + summary
			}
			return nextActionChoice{
				literal:      in.Recovery.Next,
				fallback:     candidateContinue,
				alternatives: alternatives,
				recommendation: fmt.Sprintf(
					"Phase %d is blocked.%s The saved report from the last check names the exact command "+
						"that clears it, so this is more targeted than a general re-check.",
					state.CurrentPhase, summary),
			}
		}
		return nextActionChoice{
			key: candidateContinue,
			recommendation: fmt.Sprintf(
				"Phase %d has produced work that has not been checked yet. The next step runs the "+
					"checks and, if they pass, moves on to the following phase.", state.CurrentPhase),
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateResumeColony},
			},
		}

	case colony.StateCOMPLETED:
		// Every one of the other four deciders except workflowSuggestionsForState
		// says "sign it off" here, and they are right: COMPLETED means the work
		// is done but the project has NOT been signed off -- signing off is what
		// sets the final milestone that the archive step looks for.
		return nextActionChoice{
			key: candidateSeal,
			recommendation: "All the work is done but the project has not been signed off yet. Signing " +
				"it off records what was learned and marks it complete.",
			alternatives: []nextActionAlternativeChoice{
				{key: candidateStatus},
				{key: candidateHistory},
			},
		}
	}

	// READY, IDLE with a plan, and anything else with phases: start the first
	// phase that is not finished.
	if phase := recoveryPhase(&state); phase != nil && phase.Status != colony.PhaseCompleted {
		return nextActionChoice{
			key:  candidateBuildPhase,
			args: []interface{}{phase.ID},
			recommendation: fmt.Sprintf(
				"Phase %d (%s) is ready to start. Helpers will write the code for it and report back "+
					"before anything is accepted.", phase.ID, phase.Name),
			alternatives: []nextActionAlternativeChoice{
				{key: candidateFocus},
				{key: candidateStatus},
				{key: candidatePheromones},
			},
		}
	}

	return nextActionChoice{
		key: candidateSeal,
		recommendation: "Every planned phase is finished. Signing the project off marks it complete and " +
			"records what was learned.",
		alternatives: []nextActionAlternativeChoice{
			{key: candidateStatus},
			{key: candidateHistory},
		},
	}
}

// gateChoice turns a choice into the exact command, resolved against the live
// command tree. A primary that does not resolve falls back to a command that is
// itself gated -- so the fallback cannot rot either -- and the substitution is
// recorded so the owner is never silently redirected.
func gateChoice(choice nextActionChoice) (string, []string) {
	var notes []string

	if literal := strings.TrimSpace(choice.literal); literal != "" {
		if command, ok := availableCommand(literal); ok {
			return command, nil
		}
		notes = append(notes, fmt.Sprintf(
			"The saved report suggested %q, which is not a command this version has. Falling back to the general next step.",
			literal))
		if command, why, ok := candidateCommand(choice.fallback); ok {
			if gated, ok := availableCommand(command); ok {
				_ = why
				return gated, notes
			}
		}
	} else if choice.key != "" {
		if command, _, ok := candidateCommand(choice.key, choice.args...); ok {
			if gated, ok := availableCommand(command); ok {
				return gated, notes
			}
			notes = append(notes, fmt.Sprintf(
				"%q is not a command this version has, so a general next step is offered instead.", command))
		}
		if choice.fallback != "" {
			if command, _, ok := candidateCommand(choice.fallback, choice.fallbackArgs...); ok {
				if gated, ok := availableCommand(command); ok {
					return gated, notes
				}
			}
		}
	}

	// Last resort: the dashboard, which changes nothing. Still gated.
	if command, _, ok := candidateCommand(candidateStatus); ok {
		if gated, ok := availableCommand(command); ok {
			return gated, notes
		}
	}
	return "", notes
}

// nextActionFillerKeys top up a short alternatives list. Every entry is gated
// like any other candidate.
var nextActionFillerKeys = []nextActionCandidateKey{
	candidateStatus,
	candidateResumeColony,
	candidateHistory,
	candidatePheromones,
}

func gateAlternatives(choices []nextActionAlternativeChoice, recommended string) []nextActionAlternative {
	result := make([]nextActionAlternative, 0, 4)
	seen := map[string]bool{recommended: true}

	add := func(command, why string) {
		if len(result) >= 4 {
			return
		}
		command = strings.TrimSpace(command)
		if command == "" || seen[command] {
			return
		}
		gated, ok := availableCommand(command)
		if !ok {
			return
		}
		seen[gated] = true
		result = append(result, nextActionAlternative{Command: gated, Explanation: why})
	}

	for _, choice := range choices {
		if literal := strings.TrimSpace(choice.literal); literal != "" {
			why := choice.why
			if strings.TrimSpace(why) == "" {
				why = "Another way forward from here."
			}
			add(literal, why)
			continue
		}
		command, defaultWhy, ok := candidateCommand(choice.key, choice.args...)
		if !ok {
			continue
		}
		why := choice.why
		if strings.TrimSpace(why) == "" {
			why = defaultWhy
		}
		add(command, why)
	}

	for _, key := range nextActionFillerKeys {
		if len(result) >= 2 {
			break
		}
		if command, why, ok := candidateCommand(key); ok {
			add(command, why)
		}
	}

	return result
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
		changed = append(changed, "The last thing you ran was "+command+".")
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
