package cmd

// The lifecycle closeout is the small, shared result grammar shown after a
// lifecycle command. It consumes an already-decided LifecycleProjection: it
// neither reads project files nor chooses a next action. Commands may add
// command-local evidence and a plain-English summary, but identity, standing,
// open items, and Next Up continue to come from the one projection.

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	LifecycleCloseoutSchemaVersion = "lifecycle-closeout/v1"
	lifecycleCloseoutResultKey     = "lifecycle_closeout"
)

// LifecycleCloseoutSlot is a stable machine identifier. The human label is
// deliberately rendered in one place so JSON consumers do not have to parse
// terminal prose.
type LifecycleCloseoutSlot string

const (
	LifecycleCloseoutColony               LifecycleCloseoutSlot = "colony"
	LifecycleCloseoutParticipants         LifecycleCloseoutSlot = "participants"
	LifecycleCloseoutWhatHappened         LifecycleCloseoutSlot = "what_happened"
	LifecycleCloseoutEvidence             LifecycleCloseoutSlot = "evidence"
	LifecycleCloseoutStateChanges         LifecycleCloseoutSlot = "state_changes"
	LifecycleCloseoutStandingInstructions LifecycleCloseoutSlot = "standing_instructions"
	LifecycleCloseoutUnresolved           LifecycleCloseoutSlot = "unresolved"
	LifecycleCloseoutNextUp               LifecycleCloseoutSlot = "next_up"
)

var lifecycleCloseoutCanonicalSlots = [...]LifecycleCloseoutSlot{
	LifecycleCloseoutColony,
	LifecycleCloseoutParticipants,
	LifecycleCloseoutWhatHappened,
	LifecycleCloseoutEvidence,
	LifecycleCloseoutStateChanges,
	LifecycleCloseoutStandingInstructions,
	LifecycleCloseoutUnresolved,
	LifecycleCloseoutNextUp,
}

type LifecycleCloseoutColonyValue struct {
	Name     string `json:"name,omitempty"`
	Goal     string `json:"goal,omitempty"`
	Standing string `json:"standing,omitempty"`
}

type LifecycleCloseoutEvent struct {
	Summary string             `json:"summary"`
	Outcome colony.OutcomeKind `json:"outcome_kind"`
	// Verdict is the work verdict's owner-facing label, read from
	// colony.WorkOutcomeLabels() -- the single authority for that wording.
	// Empty when no work verdict was supplied for this closeout.
	Verdict string `json:"verdict,omitempty"`
}

// LifecycleCloseoutRecommendedAction is the closeout's own next-action
// recommendation (D-07): exactly one command, a one-sentence reason for
// preferring it over the alternatives, and the ordered alternatives beneath
// it. Selected from the recorded colony.WorkOutcome and the build attempt
// record it was made against -- never from the rendered closeout text and
// never from a keyword scan of a summary string. See
// recommendedActionForWorkOutcome.
type LifecycleCloseoutRecommendedAction struct {
	Command      string   `json:"command"`
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type LifecycleCloseoutStateChangeSet struct {
	Effect  colony.LifecycleStateEffect `json:"effect"`
	Changes []colony.LifecycleChange    `json:"changes,omitempty"`
}

type LifecycleCloseoutOpenItems struct {
	Warnings  []colony.LifecycleIssue    `json:"warnings,omitempty"`
	Debt      []colony.LifecycleIssue    `json:"debt,omitempty"`
	Blockers  []colony.LifecycleIssue    `json:"blockers,omitempty"`
	Decisions []colony.LifecycleDecision `json:"decisions,omitempty"`
}

// LifecycleCloseout is the structured counterpart of the focused terminal
// block. Slots contains only applicable sections and is always in canonical
// order. The typed values remain available independently of the prose.
type LifecycleCloseout struct {
	SchemaVersion string             `json:"schema_version"`
	Command       string             `json:"command"`
	OutcomeKind   colony.OutcomeKind `json:"outcome_kind"`
	// WorkOutcome is the six-verdict work-cycle result (D-05), additive to
	// the existing OutcomeKind above. omitempty preserves byte-identical
	// serialization for a closeout built without a verdict.
	WorkOutcome          *colony.WorkOutcome             `json:"work_outcome,omitempty"`
	ProjectionRevision   string                          `json:"projection_revision"`
	Closure              LifecycleClosureProjection      `json:"closure"`
	Colony               LifecycleCloseoutColonyValue    `json:"colony"`
	Participants         []LifecycleActorFact            `json:"participants,omitempty"`
	WhatHappened         LifecycleCloseoutEvent          `json:"what_happened"`
	Evidence             []colony.LifecycleEvidence      `json:"evidence,omitempty"`
	Verification         []colony.LifecycleVerification  `json:"verification,omitempty"`
	StateChanges         LifecycleCloseoutStateChangeSet `json:"state_changes"`
	StandingInstructions []string                        `json:"standing_instructions,omitempty"`
	Unresolved           LifecycleCloseoutOpenItems      `json:"unresolved"`
	NextUp               LifecycleProjectedAction        `json:"next_up"`
	Alternatives         []LifecycleActionChoice         `json:"alternatives,omitempty"`
	// RecommendedAction is D-07's one derived next action for a non-success
	// (or success) work verdict, distinct from NextUp/Alternatives above
	// (the projection's own general lifecycle next-action policy). Nil when
	// no work verdict was supplied for this closeout.
	RecommendedAction *LifecycleCloseoutRecommendedAction `json:"recommended_action,omitempty"`
	Slots             []LifecycleCloseoutSlot             `json:"slots"`
}

// LifecycleCloseoutDetails contains facts known only by the command that just
// ran. Values are additive; they cannot replace projection-owned identity,
// state, closure, or Next Up policy.
type LifecycleCloseoutDetails struct {
	Summary string
	// WorkOutcome is the command's six-verdict work-cycle result (D-05).
	// The zero value means no verdict was supplied, and buildLifecycleCloseout
	// leaves every existing field exactly as it behaved before this field
	// existed.
	WorkOutcome          colony.WorkOutcome
	Participants         []LifecycleActorFact
	Evidence             []colony.LifecycleEvidence
	Verification         []colony.LifecycleVerification
	Changes              []colony.LifecycleChange
	StandingInstructions []string
	Warnings             []colony.LifecycleIssue
	Debt                 []colony.LifecycleIssue
	Blockers             []colony.LifecycleIssue
	Decisions            []colony.LifecycleDecision
}

// applyLifecycleCloseout folds a focused closeout into a result that already
// carries the authoritative lifecycle projection. Missing or disagreeing
// projection revisions fail closed instead of silently resolving another one.
func applyLifecycleCloseout(result map[string]interface{}, command string, details LifecycleCloseoutDetails) error {
	if result == nil {
		return fmt.Errorf("%s closeout result is nil", strings.TrimSpace(command))
	}
	projection, ok := lifecycleCloseoutProjectionFromValue(result[lifecycleProjectionKey])
	if !ok {
		return fmt.Errorf("%s closeout requires one lifecycle projection", strings.TrimSpace(command))
	}
	if strings.TrimSpace(projection.ProjectionRevision) == "" {
		return fmt.Errorf("%s closeout projection has no revision", strings.TrimSpace(command))
	}
	if revision := strings.TrimSpace(stringValue(result["projection_revision"])); revision != "" && revision != projection.ProjectionRevision {
		return fmt.Errorf("%s closeout projection revision %q disagrees with result revision %q", strings.TrimSpace(command), projection.ProjectionRevision, revision)
	}

	outcome, err := lifecycleCloseoutOutcome(result["outcome_kind"], projection.OutcomeKind)
	if err != nil {
		return fmt.Errorf("%s closeout: %w", strings.TrimSpace(command), err)
	}
	effect, err := lifecycleCloseoutStateEffect(result["state_effect"], projection.StateEffect)
	if err != nil {
		return fmt.Errorf("%s closeout: %w", strings.TrimSpace(command), err)
	}
	closeout, err := buildLifecycleCloseout(projection, command, outcome, effect, details)
	if err != nil {
		return err
	}

	result["projection_revision"] = projection.ProjectionRevision
	result["outcome_kind"] = outcome
	result["state_effect"] = effect
	result[nextActionCommandKey] = projection.NextAction.RuntimeCommand
	result[nextActionChoicesKey] = append([]LifecycleActionChoice(nil), projection.NextAction.Choices...)
	result[lifecycleCloseoutResultKey] = closeout
	return nil
}

// lifecycleCloseoutRefusalForState builds a zero-write refusal from a state
// the caller already inspected. It uses the existing next-action resolver and
// then immediately projects that same answer into the shared closeout.
func lifecycleCloseoutRefusalForState(state colony.ColonyState, command, summary, override, why string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"message":      strings.TrimSpace(summary),
		"outcome_kind": colony.OutcomeKindRefused,
		"state_effect": colony.LifecycleStateEffectNone,
	}
	in := nextActionInput{
		State:       normalizeLegacyColonyState(state),
		LastCommand: strings.TrimSpace(command),
		NoColony:    colonyStateIsUnstarted(state),
	}
	if override = strings.TrimSpace(override); override != "" {
		in.Override = &nextActionOverride{Command: override, Recommendation: strings.TrimSpace(why)}
	}
	applyNextActionToResult(result, resolveNextAction(in))
	if err := applyLifecycleCloseout(result, command, LifecycleCloseoutDetails{Summary: summary}); err != nil {
		return nil, err
	}
	return result, nil
}

// lifecycleCloseoutRefusalFromDisk is the same adapter for a preflight that
// has not loaded state. closeLifecycleCommand is read-only and remains the
// sole place that resolves the action.
func lifecycleCloseoutRefusalFromDisk(command, summary, override, why string) (map[string]interface{}, error) {
	dataDir := storage.ResolveDataDir(context.Background())
	state, _, err := loadColonyStateWithCompatibilityRepairReadOnlyFromPath(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		state = colony.ColonyState{}
	}
	return lifecycleCloseoutRefusalForState(state, command, summary, override, why)
}

func buildLifecycleCloseout(projection LifecycleProjection, command string, outcome colony.OutcomeKind, effect colony.LifecycleStateEffect, details LifecycleCloseoutDetails) (LifecycleCloseout, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout command is empty")
	}

	// When a work verdict is supplied, it is the source of the lifecycle
	// outcome (never a second, independent inference) and of the verdict's
	// owner-facing wording (via colony.WorkOutcomeLabels(), the one
	// authority for that text). A closeout built without a verdict -- the
	// zero value, Valid() == false -- takes neither branch and behaves
	// exactly as it did before this field existed.
	var workOutcome *colony.WorkOutcome
	var verdictLabel string
	var recommendedAction *LifecycleCloseoutRecommendedAction
	var knowledgeDeltaEvidence []colony.LifecycleEvidence
	if details.WorkOutcome.Valid() {
		derived, err := details.WorkOutcome.LifecycleOutcome()
		if err != nil {
			return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout work outcome %q: %w", details.WorkOutcome, err)
		}
		outcome = derived
		verdict := details.WorkOutcome
		workOutcome = &verdict
		verdictLabel = colony.WorkOutcomeLabels()[details.WorkOutcome]

		// D-07/CAP-066: the attempt this verdict was recorded against, read
		// ONCE and reused for both the recommended next action and the
		// knowledge-delta evidence below -- never a second, independent
		// lookup, and never derived from the rendered text. A phase with no
		// recorded attempt (loadLatestBuildAttempt returns ok=false) yields
		// the zero-value record; recommendedActionForWorkOutcome still
		// returns a total, non-empty answer for every declared verdict.
		phaseNum := projection.Phase.Value.CurrentNumber
		_, attempt, _ := loadLatestBuildAttempt(phaseNum)
		action, actionErr := recommendedActionForWorkOutcome(details.WorkOutcome, attempt)
		if actionErr != nil {
			return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout recommended action: %w", actionErr)
		}
		recommendedAction = &action
		knowledgeDeltaEvidence = lifecycleCloseoutKnowledgeDeltaEvidence(attempt)
	}

	if !outcome.Valid() {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout outcome %q is invalid", outcome)
	}
	if !effect.Valid() {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout state effect %q is invalid", effect)
	}

	summary := strings.TrimSpace(details.Summary)
	if summary == "" {
		if workOutcome != nil && verdictLabel != "" {
			// The default summary for a verdict-carrying closeout is drawn
			// from WorkOutcomeLabels() too -- never the generic "finished
			// with outcome" phrasing, which would leak a success-shaped word
			// ("finished") into every non-success card regardless of verdict.
			summary = fmt.Sprintf("%s: %s.", command, verdictLabel)
		} else {
			summary = fmt.Sprintf("%s finished with outcome %s.", command, outcome)
		}
	}
	standing := lifecycleCloseoutSignalTexts(projection.Signals.Value)
	standing = appendUniqueLifecycleCloseoutStrings(standing, details.StandingInstructions...)
	participants := append([]LifecycleActorFact(nil), projection.Actors.Value...)
	participants = append(participants, details.Participants...)

	closeout := LifecycleCloseout{
		SchemaVersion:      LifecycleCloseoutSchemaVersion,
		Command:            command,
		OutcomeKind:        outcome,
		WorkOutcome:        workOutcome,
		ProjectionRevision: projection.ProjectionRevision,
		Closure:            projection.Closure,
		Colony: LifecycleCloseoutColonyValue{
			Name:     strings.TrimSpace(projection.Identity.Value.Name),
			Goal:     strings.TrimSpace(projection.Goal.Value),
			Standing: strings.TrimSpace(projection.Standing.Value),
		},
		Participants: participants,
		WhatHappened: LifecycleCloseoutEvent{Summary: summary, Outcome: outcome, Verdict: verdictLabel},
		Evidence:     append(append(append([]colony.LifecycleEvidence(nil), projection.Evidence...), details.Evidence...), knowledgeDeltaEvidence...),
		Verification: append(append([]colony.LifecycleVerification(nil), projection.Verification...), details.Verification...),
		StateChanges: LifecycleCloseoutStateChangeSet{
			Effect:  effect,
			Changes: append(append([]colony.LifecycleChange(nil), projection.Changes...), details.Changes...),
		},
		StandingInstructions: standing,
		Unresolved: LifecycleCloseoutOpenItems{
			Warnings:  append(append([]colony.LifecycleIssue(nil), projection.Warnings...), details.Warnings...),
			Debt:      append(append([]colony.LifecycleIssue(nil), projection.Debt...), details.Debt...),
			Blockers:  append(append([]colony.LifecycleIssue(nil), projection.Blockers...), details.Blockers...),
			Decisions: append(append([]colony.LifecycleDecision(nil), projection.OwnerDecisions...), details.Decisions...),
		},
		NextUp:            projection.NextAction,
		Alternatives:      append([]LifecycleActionChoice(nil), projection.Alternatives...),
		RecommendedAction: recommendedAction,
	}
	closeout.Slots = lifecycleCloseoutSlots(closeout)
	return closeout, nil
}

// recommendedActionForWorkOutcome derives the ONE recommended next action
// for a work verdict (D-07), total across colony.AllWorkOutcomes(): every
// declared verdict returns a non-empty command and a non-empty one-sentence
// reason, and an undeclared verdict is refused by name rather than falling
// through to a shared default. It is a pure function of the verdict and the
// attempt evidence already recorded on the attempt -- never the rendered
// closeout text, and never a keyword scan of a summary string. Command
// vocabulary is the owner-facing spelling this repository already locked
// elsewhere (`aether resume` -- Phase 199's sole recovery door;
// `aether unblock --dispatch` -- the Fixer's existing intake; the
// `aether build <phase> --force --task <id>...` recovery family
// buildUnfinishedRetryRedispatchCommand already produces) -- no new
// spelling is invented here.
func recommendedActionForWorkOutcome(verdict colony.WorkOutcome, attempt buildAttemptRecord) (LifecycleCloseoutRecommendedAction, error) {
	const statusAlternative = "aether status"
	switch verdict {
	case colony.WorkOutcomeSuccess:
		return LifecycleCloseoutRecommendedAction{
			Command:      "aether continue",
			Reason:       "the work finished cleanly, so the next step is the program's own check and advance.",
			Alternatives: []string{statusAlternative},
		}, nil
	case colony.WorkOutcomeNoChange:
		return LifecycleCloseoutRecommendedAction{
			Command:      "aether continue",
			Reason:       "nothing needed changing, so the next step is the same check and advance a clean success would take.",
			Alternatives: []string{statusAlternative},
		}, nil
	case colony.WorkOutcomePartial:
		unfinished := uniqueSortedStrings(attempt.RecoveryTaskIDs)
		command := buildUnfinishedRetryRedispatchCommand(attempt.Phase, unfinished)
		reason := "only the unfinished tasks need to run again -- everything else already has credited evidence."
		if len(unfinished) > 0 {
			reason = fmt.Sprintf("only the unfinished tasks (%s) need to run again -- everything else already has credited evidence.", strings.Join(unfinished, ", "))
		}
		return LifecycleCloseoutRecommendedAction{
			Command:      command,
			Reason:       reason,
			Alternatives: []string{statusAlternative},
		}, nil
	case colony.WorkOutcomeBlocker:
		reason := "a blocker is stopping the work and needs an owner answer before anything else can run."
		if blocker := strings.TrimSpace(attempt.Error); blocker != "" {
			reason = fmt.Sprintf("the blocker (%s) needs an owner answer before anything else can run.", blocker)
		}
		return LifecycleCloseoutRecommendedAction{
			Command:      "aether unblock --dispatch",
			Reason:       reason,
			Alternatives: []string{statusAlternative},
		}, nil
	case colony.WorkOutcomeTimeout:
		command := strings.TrimSpace(attempt.RecoveryCommand)
		reason := "no completion evidence was recorded before the timeout, so the phase needs a fresh dispatch."
		if strings.TrimSpace(attempt.CompletionPath) != "" && strings.TrimSpace(attempt.CompletionSHA256) != "" {
			reason = "a completion packet was already staged before the timeout, so finalizing that staged evidence is the recorded evidence's own recommended path."
		}
		if command == "" {
			command = buildForceRedispatchCommand(attempt.Phase)
		}
		return LifecycleCloseoutRecommendedAction{
			Command:      command,
			Reason:       reason,
			Alternatives: []string{statusAlternative},
		}, nil
	case colony.WorkOutcomeInterrupted:
		return LifecycleCloseoutRecommendedAction{
			Command:      "aether resume",
			Reason:       "the attempt was stopped before it finished, and aether resume is the one recovery door back into it.",
			Alternatives: []string{statusAlternative},
		}, nil
	default:
		return LifecycleCloseoutRecommendedAction{}, fmt.Errorf("work outcome %q has no recommended next action", verdict)
	}
}

// lifecycleCloseoutKnowledgeDeltaEvidence converts an attempt's own CAP-066
// decision/learning deltas into LifecycleEvidence entries, read straight off
// the attempt record already loaded for this closeout's own phase -- never
// a second lookup, and never a write. Two different attempts' deltas can
// never appear on the same card because each closeout only ever reads the
// one attempt bound to its own phase (loadLatestBuildAttempt).
func lifecycleCloseoutKnowledgeDeltaEvidence(attempt buildAttemptRecord) []colony.LifecycleEvidence {
	if len(attempt.KnowledgeDeltas) == 0 {
		return nil
	}
	evidence := make([]colony.LifecycleEvidence, 0, len(attempt.KnowledgeDeltas))
	for i, delta := range attempt.KnowledgeDeltas {
		evidence = append(evidence, colony.LifecycleEvidence{
			ID:      fmt.Sprintf("%s-knowledge-delta-%d", attempt.ID, i),
			Kind:    strings.TrimSpace(delta.Kind),
			Source:  attempt.ID,
			Summary: strings.TrimSpace(delta.Summary),
		})
	}
	return evidence
}

// lifecycleCloseoutSlots decides which canonical slots render. A closeout
// carrying a work verdict (D-05) renders EVERY canonical slot regardless of
// content -- a non-success verdict must look as considered as a success, so
// an empty slot is never silently dropped, it says plainly there is nothing
// there (see the render helpers below). A closeout with no verdict keeps the
// exact pre-existing behavior: an empty optional slot is omitted.
func lifecycleCloseoutSlots(closeout LifecycleCloseout) []LifecycleCloseoutSlot {
	fullCeremony := closeout.WorkOutcome != nil
	slots := make([]LifecycleCloseoutSlot, 0, len(lifecycleCloseoutCanonicalSlots))
	for _, slot := range lifecycleCloseoutCanonicalSlots {
		if !fullCeremony {
			switch slot {
			case LifecycleCloseoutColony:
				if closeout.Colony.Name == "" && closeout.Colony.Goal == "" && closeout.Colony.Standing == "" {
					continue
				}
			case LifecycleCloseoutParticipants:
				if len(closeout.Participants) == 0 {
					continue
				}
			case LifecycleCloseoutEvidence:
				if len(closeout.Evidence) == 0 && len(closeout.Verification) == 0 {
					continue
				}
			case LifecycleCloseoutStandingInstructions:
				if len(closeout.StandingInstructions) == 0 {
					continue
				}
			case LifecycleCloseoutUnresolved:
				if !lifecycleCloseoutHasOpenItems(closeout.Unresolved) {
					continue
				}
			}
		}
		slots = append(slots, slot)
	}
	return slots
}

func lifecycleCloseoutHasOpenItems(items LifecycleCloseoutOpenItems) bool {
	return len(items.Warnings) > 0 || len(items.Debt) > 0 || len(items.Blockers) > 0 || len(items.Decisions) > 0
}

func lifecycleCloseoutSignalTexts(signals []colony.PheromoneSignal) []string {
	values := make([]string, 0, len(signals))
	for _, signal := range signals {
		if !signal.Active {
			continue
		}
		text := strings.TrimSpace(extractContentText(signal.Content))
		if text == "" {
			continue
		}
		values = appendUniqueLifecycleCloseoutStrings(values, text)
	}
	return values
}

func appendUniqueLifecycleCloseoutStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	result := make([]string, 0, len(values)+len(additions))
	for _, value := range append(append([]string(nil), values...), additions...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func lifecycleCloseoutProjectionFromValue(value interface{}) (LifecycleProjection, bool) {
	switch projection := value.(type) {
	case LifecycleProjection:
		return projection, strings.TrimSpace(projection.ProjectionRevision) != ""
	case *LifecycleProjection:
		if projection != nil {
			return *projection, strings.TrimSpace(projection.ProjectionRevision) != ""
		}
		return LifecycleProjection{}, false
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return LifecycleProjection{}, false
		}
		var decoded LifecycleProjection
		if json.Unmarshal(data, &decoded) != nil || strings.TrimSpace(decoded.ProjectionRevision) == "" {
			return LifecycleProjection{}, false
		}
		return decoded, true
	}
}

func lifecycleCloseoutOutcome(value interface{}, fallback colony.OutcomeKind) (colony.OutcomeKind, error) {
	if value == nil || strings.TrimSpace(stringValue(value)) == "" {
		if fallback.Valid() {
			return fallback, nil
		}
		return colony.OutcomeKindNoChange, nil
	}
	var outcome colony.OutcomeKind
	switch typed := value.(type) {
	case colony.OutcomeKind:
		outcome = typed
	case string:
		outcome = colony.OutcomeKind(strings.TrimSpace(typed))
	default:
		outcome = colony.OutcomeKind(strings.TrimSpace(stringValue(value)))
	}
	if !outcome.Valid() {
		return "", fmt.Errorf("outcome kind %q is invalid", outcome)
	}
	return outcome, nil
}

func lifecycleCloseoutStateEffect(value interface{}, fallback colony.LifecycleStateEffect) (colony.LifecycleStateEffect, error) {
	if value == nil || strings.TrimSpace(stringValue(value)) == "" {
		if fallback.Valid() {
			return fallback, nil
		}
		return colony.LifecycleStateEffectNone, nil
	}
	var effect colony.LifecycleStateEffect
	switch typed := value.(type) {
	case colony.LifecycleStateEffect:
		effect = typed
	case string:
		effect = colony.LifecycleStateEffect(strings.TrimSpace(typed))
	default:
		effect = colony.LifecycleStateEffect(strings.TrimSpace(stringValue(value)))
	}
	if !effect.Valid() {
		return "", fmt.Errorf("state effect %q is invalid", effect)
	}
	return effect, nil
}

func lifecycleCloseoutFromResult(result map[string]interface{}) (LifecycleCloseout, bool) {
	if result == nil || result[lifecycleCloseoutResultKey] == nil {
		return LifecycleCloseout{}, false
	}
	switch closeout := result[lifecycleCloseoutResultKey].(type) {
	case LifecycleCloseout:
		return closeout, closeout.SchemaVersion == LifecycleCloseoutSchemaVersion
	case *LifecycleCloseout:
		if closeout != nil {
			return *closeout, closeout.SchemaVersion == LifecycleCloseoutSchemaVersion
		}
		return LifecycleCloseout{}, false
	default:
		data, err := json.Marshal(closeout)
		if err != nil {
			return LifecycleCloseout{}, false
		}
		var decoded LifecycleCloseout
		if json.Unmarshal(data, &decoded) != nil || decoded.SchemaVersion != LifecycleCloseoutSchemaVersion {
			return LifecycleCloseout{}, false
		}
		return decoded, true
	}
}

// renderLifecycleCloseout is pure presentation. The Next Up body is delegated
// to the existing projection renderer after removing only its own banner; the
// closeout does not reinterpret or resolve the action.
func renderLifecycleCloseout(closeout LifecycleCloseout, platform string) string {
	var b strings.Builder
	for _, slot := range closeout.Slots {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderStageMarker(lifecycleCloseoutSlotLabel(slot)))
		switch slot {
		case LifecycleCloseoutColony:
			lifecycleCloseoutRenderColony(&b, closeout.Colony)
		case LifecycleCloseoutParticipants:
			lifecycleCloseoutRenderParticipants(&b, closeout.Participants)
		case LifecycleCloseoutWhatHappened:
			fmt.Fprintf(&b, "%s\nOutcome: %s\n", closeout.WhatHappened.Summary, closeout.WhatHappened.Outcome)
			if closeout.WhatHappened.Verdict != "" {
				fmt.Fprintf(&b, "Verdict: %s\n", closeout.WhatHappened.Verdict)
			}
			if closeout.RecommendedAction != nil {
				fmt.Fprintf(&b, "Recommended: %s — %s\n", closeout.RecommendedAction.Command, closeout.RecommendedAction.Reason)
				for _, alt := range closeout.RecommendedAction.Alternatives {
					fmt.Fprintf(&b, "Alternative: %s\n", alt)
				}
			}
		case LifecycleCloseoutEvidence:
			lifecycleCloseoutRenderEvidence(&b, closeout.Evidence, closeout.Verification)
		case LifecycleCloseoutStateChanges:
			fmt.Fprintf(&b, "State effect: %s\n", closeout.StateChanges.Effect)
			for _, change := range closeout.StateChanges.Changes {
				fmt.Fprintf(&b, "- %s: %s\n", emptyFallback(strings.TrimSpace(change.Target), "state"), emptyFallback(strings.TrimSpace(change.Action), "changed"))
			}
		case LifecycleCloseoutStandingInstructions:
			if len(closeout.StandingInstructions) == 0 {
				b.WriteString("No standing instructions recorded.\n")
			}
			for _, instruction := range closeout.StandingInstructions {
				fmt.Fprintf(&b, "- %s\n", instruction)
			}
		case LifecycleCloseoutUnresolved:
			lifecycleCloseoutRenderOpenItems(&b, closeout.Unresolved)
		case LifecycleCloseoutNextUp:
			projection := LifecycleProjection{NextAction: closeout.NextUp, Alternatives: closeout.Alternatives}
			b.WriteString(lifecycleCloseoutNextUpBody(projection, platform))
			if command := lifecycleProjectionCommand(closeout.NextUp.RuntimeCommand, platform); command != "" {
				fmt.Fprintf(&b, "Next Up: %s\n", command)
			}
		}
	}
	return b.String()
}

func renderLifecycleCloseoutFromResult(result map[string]interface{}, platform string) string {
	closeout, ok := lifecycleCloseoutFromResult(result)
	if !ok {
		return ""
	}
	return renderLifecycleCloseout(closeout, platform)
}

func lifecycleCloseoutSlotLabel(slot LifecycleCloseoutSlot) string {
	switch slot {
	case LifecycleCloseoutColony:
		return "Colony"
	case LifecycleCloseoutParticipants:
		return "Participants"
	case LifecycleCloseoutWhatHappened:
		return "What happened"
	case LifecycleCloseoutEvidence:
		return "Evidence"
	case LifecycleCloseoutStateChanges:
		return "State changes"
	case LifecycleCloseoutStandingInstructions:
		return "Standing instructions"
	case LifecycleCloseoutUnresolved:
		return "Unresolved"
	case LifecycleCloseoutNextUp:
		return "Next Up"
	default:
		return string(slot)
	}
}

func lifecycleCloseoutRenderColony(b *strings.Builder, value LifecycleCloseoutColonyValue) {
	if value.Name == "" && value.Goal == "" && value.Standing == "" {
		b.WriteString("Nothing recorded.\n")
		return
	}
	if value.Name != "" {
		fmt.Fprintf(b, "Name: %s\n", value.Name)
	}
	if value.Goal != "" {
		fmt.Fprintf(b, "Goal: %s\n", value.Goal)
	}
	if value.Standing != "" {
		fmt.Fprintf(b, "Standing: %s\n", value.Standing)
	}
}

func lifecycleCloseoutRenderParticipants(b *strings.Builder, participants []LifecycleActorFact) {
	if len(participants) == 0 {
		b.WriteString("No participants recorded.\n")
		return
	}
	for _, participant := range participants {
		identity := strings.TrimSpace(strings.Join([]string{participant.Caste, participant.Name}, " "))
		if identity == "" {
			identity = "recorded participant"
		}
		detail := strings.TrimSpace(participant.Status)
		if detail == "" {
			detail = strings.TrimSpace(participant.Task)
		}
		if detail == "" {
			fmt.Fprintf(b, "- %s\n", identity)
		} else {
			fmt.Fprintf(b, "- %s — %s\n", identity, detail)
		}
	}
}

func lifecycleCloseoutRenderEvidence(b *strings.Builder, evidence []colony.LifecycleEvidence, verification []colony.LifecycleVerification) {
	if len(evidence) == 0 && len(verification) == 0 {
		b.WriteString("No evidence recorded.\n")
		return
	}
	for _, item := range evidence {
		label := emptyFallback(strings.TrimSpace(item.Summary), emptyFallback(strings.TrimSpace(item.Source), item.ID))
		fmt.Fprintf(b, "- %s\n", label)
	}
	for _, check := range verification {
		verdict := "failed"
		if check.Passed {
			verdict = "passed"
		}
		fmt.Fprintf(b, "- %s: %s", emptyFallback(strings.TrimSpace(check.Name), "verification"), verdict)
		if detail := strings.TrimSpace(check.Detail); detail != "" {
			fmt.Fprintf(b, " — %s", detail)
		}
		b.WriteString("\n")
	}
}

func lifecycleCloseoutRenderOpenItems(b *strings.Builder, items LifecycleCloseoutOpenItems) {
	if !lifecycleCloseoutHasOpenItems(items) {
		b.WriteString("Nothing unresolved.\n")
		return
	}
	for _, group := range [][]colony.LifecycleIssue{items.Warnings, items.Debt, items.Blockers} {
		for _, item := range group {
			fmt.Fprintf(b, "- %s\n", emptyFallback(strings.TrimSpace(item.Summary), item.ID))
		}
	}
	for _, decision := range items.Decisions {
		fmt.Fprintf(b, "- %s\n", emptyFallback(strings.TrimSpace(decision.Summary), decision.ID))
	}
}

func lifecycleCloseoutNextUpBody(projection LifecycleProjection, platform string) string {
	rendered := strings.TrimPrefix(renderLifecycleProjectionNextUp(projection, platform), "\n")
	return strings.TrimPrefix(rendered, renderBanner(commandEmoji("next-up"), "Next Up"))
}

// appendLifecycleCloseoutVisual inserts the focused closeout before the shared
// resolver card when that card is the terminal section. Result bodies retain
// their command-specific evidence, while the one resolver-owned card remains
// the final answer on screen and in the machine-readable envelope.
func appendLifecycleCloseoutVisual(body string, result map[string]interface{}, platform string) string {
	closeout, ok := lifecycleCloseoutFromResult(result)
	if !ok {
		return body
	}
	legacy := renderBanner(commandEmoji("status"), "What Next")
	var out string
	if index := strings.LastIndex(body, legacy); index >= 0 {
		prefix := strings.TrimRight(body[:index], "\n")
		card := strings.TrimLeft(body[index:], "\n")
		if prefix != "" {
			prefix += "\n"
		}
		out = prefix + renderLifecycleCloseout(closeout, platform) + card
	} else {
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		out = body + renderLifecycleCloseout(closeout, platform)
		if answer, ok := nextActionFromResult(result); ok {
			out = out + "\n" + renderNextActionCardForPlatform(answer, platform)
		}
	}
	return appendLifecycleCloseoutSpendLine(out, closeout, result)
}

// appendLifecycleCloseoutSpendLine puts the one cost-and-time block
// (cmd/spend_cost_line.go) as the FINAL element of a rendered closeout, for
// every one of the six work verdicts (D-05) -- not only the success verdict.
// It reuses appendSpendCostLine, the one existing append helper, rather than
// inventing a second placement rule: the placement being one rule instead of
// two is what makes exactly-one-per-lane structural (Phase 201, D-06).
//
// Gated on closeout.WorkOutcome != nil: a closeout with no work verdict --
// colonize, plan, init, entomb, pause, resume, seal -- is not a work-cycle
// result in the sense this block reports, so appending an empty "What The
// Helpers Cost" heading over those screens would be a heading over an answer
// nobody asked for. Only a closeout that carries a verdict (build, check, or
// autopilot) gets this block.
func appendLifecycleCloseoutSpendLine(body string, closeout LifecycleCloseout, result map[string]interface{}) string {
	if closeout.WorkOutcome == nil {
		return body
	}
	phaseID := intValue(result["completion_phase"])
	if phaseID == 0 {
		phaseID = intValue(result["current_phase"])
	}
	return appendSpendCostLine(body, phaseID)
}
