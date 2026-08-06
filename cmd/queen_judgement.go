package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The Queen's team choice was, until this file existed, entirely mechanical:
// casteRelevanceScore counted keyword hits against a fixed table, so "add a
// dark mode toggle" and "add a payment flow" were distinguished only by which
// literal words they happened to contain. A phase whose risk is obvious to any
// reader — "let users reset their password by email" — scored no differently
// from one that merely used the same vocabulary, and a phase that needed a
// specialist but avoided its keywords got nobody.
//
// A model reading the phase can see what the words only hint at. What it must
// not be able to do is talk the colony out of its safety floor: a proposal is
// judgement about which optional specialists help, never permission to drop
// the Watcher or to skip a security review on work that touches credentials.
//
// So the proposal is an input, not a decision. queenApplyJudgement takes it,
// unions in everything the phase requires regardless, caps the result by the
// operator's budget, and reports exactly what it overrode and why — so a Queen
// that proposes badly produces a visible correction rather than a silent one.

// queenCasteJudgement is the outcome of reconciling a proposed team with the
// floors and ceiling the runtime owns.
type queenCasteJudgement struct {
	// Proposed is the team the Queen asked for, normalized.
	Proposed []string
	// Final is the team that will actually spawn.
	Final []string
	// Added lists safety castes the proposal omitted and the runtime restored.
	Added []string
	// Dropped lists proposed castes removed to fit the worker budget.
	Dropped []string
	// Unknown lists proposed names that are not dispatchable castes.
	Unknown []string
	// Rationale is the Queen's stated reasoning, carried through for display.
	Rationale string
	// Source is "queen" when a proposal was supplied, "deterministic" when the
	// keyword engine chose.
	Source string
}

// Summary renders the judgement as a line a non-specialist can read.
func (j queenCasteJudgement) Summary() string {
	if len(j.Final) == 0 {
		return ""
	}
	var b strings.Builder
	if j.Source == "queen" {
		b.WriteString("Queen chose: ")
	} else {
		b.WriteString("Team: ")
	}
	b.WriteString(strings.Join(j.Final, ", "))
	b.WriteString(".")
	if len(j.Added) > 0 {
		b.WriteString(fmt.Sprintf(" Added %s — required for this phase regardless of the proposal.",
			strings.Join(j.Added, ", ")))
	}
	if len(j.Dropped) > 0 {
		b.WriteString(fmt.Sprintf(" Dropped %s — over the worker budget.",
			strings.Join(j.Dropped, ", ")))
	}
	if len(j.Unknown) > 0 {
		b.WriteString(fmt.Sprintf(" Ignored unknown caste(s): %s.",
			strings.Join(j.Unknown, ", ")))
	}
	return b.String()
}

// queenApplyJudgement reconciles a proposed team with what the phase requires
// and what the budget allows.
//
// An empty proposal is not an error and not an empty team: it means no
// judgement was offered, so the deterministic engine decides. That keeps every
// existing caller working unchanged and makes the model path additive.
func queenApplyJudgement(proposed []string, rationale string, phase colony.Phase, flowType string, state colony.ColonyState) queenCasteJudgement {
	normalized, unknown := normalizeProposedCastes(proposed)

	if len(normalized) == 0 {
		deterministic := casteNames(queenOrchestrate(phase, flowType, state))
		return queenCasteJudgement{
			Final:   deterministic,
			Unknown: unknown,
			Source:  "deterministic",
		}
	}

	budget := queenSpawnBudgetForPhase(phase, flowType, state)
	required := budget.RequiredCastes

	// Required castes are not negotiable. They are restored whether the Queen
	// left them out on purpose or overlooked them — the runtime cannot tell the
	// difference, and the failure mode of guessing wrong is a build that no one
	// checked.
	proposedSet := stringSet(normalized)
	added := []string{}
	for _, caste := range required {
		if !proposedSet[caste] {
			proposedSet[caste] = true
			added = append(added, caste)
		}
	}
	sort.Strings(added)

	// Order the final team: required first, then the Queen's picks in the order
	// it asked for them, so budget pressure trims its lowest priority rather
	// than an arbitrary one.
	requiredSet := stringSet(required)
	final := append([]string{}, required...)
	sort.Strings(final)
	for _, caste := range normalized {
		if !requiredSet[caste] {
			final = append(final, caste)
		}
	}

	dropped := []string{}
	if budget.MaxWorkers > 0 && len(final) > budget.MaxWorkers {
		limit := budget.MaxWorkers
		if len(required) > limit {
			// A phase whose own safety floor exceeds the budget keeps the floor.
			limit = len(required)
		}
		dropped = append(dropped, final[limit:]...)
		final = final[:limit]
	}

	return queenCasteJudgement{
		Proposed:  normalized,
		Final:     final,
		Added:     added,
		Dropped:   dropped,
		Unknown:   unknown,
		Rationale: strings.TrimSpace(rationale),
		Source:    "queen",
	}
}

// normalizeProposedCastes trims, lowercases, de-duplicates and validates the
// proposal, separating names that are not dispatchable castes. An unknown name
// is reported rather than silently ignored: a Queen asking for "security" when
// the caste is "gatekeeper" should see that its request did not land.
func normalizeProposedCastes(proposed []string) (known []string, unknown []string) {
	valid := make(map[string]bool, len(casteRelevanceRegistry))
	for _, profile := range casteRelevanceRegistry {
		valid[profile.Caste] = true
	}

	seen := map[string]bool{}
	for _, raw := range proposed {
		// Accept comma-separated values in a single argument so the flag is
		// forgiving about --castes "a,b" versus --castes a --castes b.
		for _, part := range strings.Split(raw, ",") {
			caste := strings.ToLower(strings.TrimSpace(part))
			if caste == "" || seen[caste] {
				continue
			}
			seen[caste] = true
			if valid[caste] {
				known = append(known, caste)
			} else {
				unknown = append(unknown, caste)
			}
		}
	}
	return known, unknown
}

func casteNames(dispatches []CasteDispatch) []string {
	names := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste != "" {
			names = append(names, caste)
		}
	}
	return names
}

// queenCasteRoster describes the castes a Queen may choose from, for inclusion
// in the decision request handed to the wrapper. Without it the model is
// guessing at both the names and what each one is for.
func queenCasteRoster() []map[string]string {
	roster := make([]map[string]string, 0, len(casteRelevanceRegistry))
	for _, profile := range casteRelevanceRegistry {
		roster = append(roster, map[string]string{
			"caste":   profile.Caste,
			"good_at": strings.Join(profile.Keywords, ", "),
		})
	}
	sort.Slice(roster, func(i, j int) bool { return roster[i]["caste"] < roster[j]["caste"] })
	return roster
}

// queenCasteDecisionSummary renders the judgement as manifest data: what was
// proposed, what will spawn, and every override the runtime applied.
//
// It is emitted whether or not a proposal was supplied. On the no-proposal
// path it reports source "deterministic", which is what tells a reading Queen
// that the team it is looking at came from keyword scoring and is therefore
// the thing it may want to correct.
func queenCasteDecisionSummary(phase colony.Phase, state colony.ColonyState, reviewDepth colony.VerificationDepth, proposed []string, reason string) map[string]interface{} {
	judgementState := state
	judgementState.VerificationDepth = string(reviewDepth)
	judgement := queenApplyJudgement(proposed, reason, phase, "build", judgementState)

	summary := map[string]interface{}{
		"source":  judgement.Source,
		"final":   judgement.Final,
		"summary": judgement.Summary(),
	}
	if len(judgement.Proposed) > 0 {
		summary["proposed"] = judgement.Proposed
	}
	if len(judgement.Added) > 0 {
		summary["added_by_runtime"] = judgement.Added
	}
	if len(judgement.Dropped) > 0 {
		summary["dropped_over_budget"] = judgement.Dropped
	}
	if len(judgement.Unknown) > 0 {
		summary["unknown_ignored"] = judgement.Unknown
	}
	if judgement.Rationale != "" {
		summary["rationale"] = judgement.Rationale
	}
	return summary
}
