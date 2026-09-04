package cmd

// The lifecycle closeout is the small, shared result grammar shown after a
// lifecycle command. It consumes an already-decided LifecycleProjection: it
// neither reads project files nor chooses a next action. Commands may add
// command-local evidence and a plain-English summary, but identity, standing,
// open items, and Next Up continue to come from the one projection.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
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
	SchemaVersion        string                          `json:"schema_version"`
	Command              string                          `json:"command"`
	OutcomeKind          colony.OutcomeKind              `json:"outcome_kind"`
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
	Slots                []LifecycleCloseoutSlot         `json:"slots"`
}

// LifecycleCloseoutDetails contains facts known only by the command that just
// ran. Values are additive; they cannot replace projection-owned identity,
// state, closure, or Next Up policy.
type LifecycleCloseoutDetails struct {
	Summary              string
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

func buildLifecycleCloseout(projection LifecycleProjection, command string, outcome colony.OutcomeKind, effect colony.LifecycleStateEffect, details LifecycleCloseoutDetails) (LifecycleCloseout, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout command is empty")
	}
	if !outcome.Valid() {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout outcome %q is invalid", outcome)
	}
	if !effect.Valid() {
		return LifecycleCloseout{}, fmt.Errorf("lifecycle closeout state effect %q is invalid", effect)
	}

	summary := strings.TrimSpace(details.Summary)
	if summary == "" {
		summary = fmt.Sprintf("%s finished with outcome %s.", command, outcome)
	}
	standing := lifecycleCloseoutSignalTexts(projection.Signals.Value)
	standing = appendUniqueLifecycleCloseoutStrings(standing, details.StandingInstructions...)
	participants := append([]LifecycleActorFact(nil), projection.Actors.Value...)
	participants = append(participants, details.Participants...)

	closeout := LifecycleCloseout{
		SchemaVersion:      LifecycleCloseoutSchemaVersion,
		Command:            command,
		OutcomeKind:        outcome,
		ProjectionRevision: projection.ProjectionRevision,
		Closure:            projection.Closure,
		Colony: LifecycleCloseoutColonyValue{
			Name:     strings.TrimSpace(projection.Identity.Value.Name),
			Goal:     strings.TrimSpace(projection.Goal.Value),
			Standing: strings.TrimSpace(projection.Standing.Value),
		},
		Participants: participants,
		WhatHappened: LifecycleCloseoutEvent{Summary: summary, Outcome: outcome},
		Evidence:     append(append([]colony.LifecycleEvidence(nil), projection.Evidence...), details.Evidence...),
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
		NextUp:       projection.NextAction,
		Alternatives: append([]LifecycleActionChoice(nil), projection.Alternatives...),
	}
	closeout.Slots = lifecycleCloseoutSlots(closeout)
	return closeout, nil
}

func lifecycleCloseoutSlots(closeout LifecycleCloseout) []LifecycleCloseoutSlot {
	slots := make([]LifecycleCloseoutSlot, 0, len(lifecycleCloseoutCanonicalSlots))
	for _, slot := range lifecycleCloseoutCanonicalSlots {
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
		case LifecycleCloseoutEvidence:
			lifecycleCloseoutRenderEvidence(&b, closeout.Evidence, closeout.Verification)
		case LifecycleCloseoutStateChanges:
			fmt.Fprintf(&b, "State effect: %s\n", closeout.StateChanges.Effect)
			for _, change := range closeout.StateChanges.Changes {
				fmt.Fprintf(&b, "- %s: %s\n", emptyFallback(strings.TrimSpace(change.Target), "state"), emptyFallback(strings.TrimSpace(change.Action), "changed"))
			}
		case LifecycleCloseoutStandingInstructions:
			for _, instruction := range closeout.StandingInstructions {
				fmt.Fprintf(&b, "- %s\n", instruction)
			}
		case LifecycleCloseoutUnresolved:
			lifecycleCloseoutRenderOpenItems(&b, closeout.Unresolved)
		case LifecycleCloseoutNextUp:
			projection := LifecycleProjection{NextAction: closeout.NextUp, Alternatives: closeout.Alternatives}
			b.WriteString(lifecycleCloseoutNextUpBody(projection, platform))
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

// appendLifecycleCloseoutVisual replaces the old shared closing card when it
// is the terminal section, then appends the focused closeout. Result bodies
// retain their command-specific evidence; a second Next Up card does not.
func appendLifecycleCloseoutVisual(body string, result map[string]interface{}, platform string) string {
	closeout, ok := lifecycleCloseoutFromResult(result)
	if !ok {
		return body
	}
	legacy := renderBanner(commandEmoji("status"), "What Next")
	if index := strings.LastIndex(body, legacy); index >= 0 {
		body = strings.TrimRight(body[:index], "\n") + "\n"
	}
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + renderLifecycleCloseout(closeout, platform)
}
