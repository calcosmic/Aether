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
	// Refused lists proposed castes the phase gives no work to. Distinct from
	// Dropped: a dropped caste was affordable-but-last, a refused one had
	// nothing to do at any budget.
	Refused []string
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
	if len(j.Refused) > 0 {
		b.WriteString(fmt.Sprintf(" Refused %s — nothing in this phase for it to do.",
			strings.Join(j.Refused, ", ")))
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

	// A proposal is judgement about which optional specialists help. It is not
	// permission to summon one the phase gives no work to.
	//
	// casteRelevanceScore already returns 0 for a keyword-gated caste with no
	// match, for exactly this reason -- but only the deterministic engine
	// consulted it, so the Queen's proposal walked straight past the check. On a
	// phase copying markdown files the runtime scored the security specialist at
	// 0, did not require it, and dispatched it anyway for 107,155 tokens because
	// the Queen asked.
	//
	// Required castes are exempt. Cost control and the thing that checks the
	// work are different decisions, and this floor must never touch the second.
	requiredForExemption := stringSet(required)
	refused := []string{}
	kept := normalized[:0]
	for _, caste := range normalized {
		if !requiredForExemption[caste] && casteRelevanceScore(phase, caste) == 0 {
			refused = append(refused, caste)
			continue
		}
		kept = append(kept, caste)
	}
	normalized = kept
	sort.Strings(refused)

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
	// it asked for them.
	//
	// The trim below takes from the tail, which makes proposal order a priority
	// ranking. That is only fair if the Queen knows it is one — a model listing
	// alphabetically would otherwise lose its most important pick with no
	// signal that ordering mattered. The wrapper states the contract ("list in
	// priority order, most important first; the tail is dropped if the phase is
	// over budget") and TestBudgetTrimsTheQueensOptionalPicksNotItsRequiredOnes
	// asserts required castes survive it.
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
		Refused:   refused,
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
			caste := resolveCasteName(strings.TrimSpace(part), valid)
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

// casteNameAliases maps names a model plausibly writes onto the registry name.
// The Queen is a model naming castes from a roster it read moments ago; a
// request that misses by a synonym should land, not vanish into
// unknown_ignored where the phase silently runs without the specialist the
// Queen believed it had asked for.
var casteNameAliases = map[string]string{
	"security":     "gatekeeper",
	"performance":  "measurer",
	"perf":         "measurer",
	"docs":         "chronicler",
	"doc":          "chronicler",
	"test":         "probe",
	"tests":        "probe",
	"coverage":     "probe",
	"quality":      "auditor",
	"refactor":     "weaver",
	"researcher":   "scout",
	"research":     "scout",
	"debug":        "tracker",
	"debugger":     "tracker",
	"accessiblity": "includer",
	"a11y":         "includer",
	"verifier":     "watcher",
	"reviewer":     "watcher",
}

// resolveCasteName normalises a proposed name to a registry caste.
//
// The registry mixes separators — route_setter with an underscore,
// surveyor-nest with a hyphen — so a model writing "route-setter" was reported
// as unknown and dropped. Separator style is not a meaningful distinction
// between a request that landed and one that did not.
func resolveCasteName(raw string, valid map[string]bool) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return ""
	}
	if valid[name] {
		return name
	}
	if alias, ok := casteNameAliases[name]; ok {
		return alias
	}
	// Try the other separator style before giving up.
	for _, candidate := range []string{
		strings.ReplaceAll(name, "-", "_"),
		strings.ReplaceAll(name, "_", "-"),
		strings.ReplaceAll(name, " ", "_"),
		strings.ReplaceAll(name, " ", "-"),
	} {
		if valid[candidate] {
			return candidate
		}
	}
	return name
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

// casteCapability describes a caste in the terms a reader needs to decide
// whether this phase needs it: what it produces, and when it is a waste.
//
// The anti-goal is the load-bearing half. The first version of this roster
// built `good_at` from strings.Join(profile.Keywords, ", ") — so the judgement
// layer built to out-reason keyword scoring was handed the keyword table as its
// only description of each caste. A model told Measurer means "performance,
// optimize, latency, scale, benchmark, memory, cpu", then shown a phase saying
// "latency and memory behaviour is unchanged", will agree with the keyword
// engine because it was given the keyword engine's worldview. Naming the
// anti-goal is what makes disagreeing with the words possible.
type casteCapability struct {
	Produces string
	AvoidFor string
}

var casteCapabilities = map[string]casteCapability{
	"builder":       {"Writes the implementation for the phase's tasks.", "Nothing to implement — the phase only inspects, researches, or documents."},
	"watcher":       {"Independently verifies the work actually does what was claimed.", "Never skip. A build nobody checked reports success by assertion."},
	"probe":         {"Finds coverage gaps and untested edge cases in new code.", "No new code was produced — documentation, research, or config-only phases."},
	"auditor":       {"Reviews code quality and standards compliance before advancing.", "Throwaway prototypes and discovery spikes where standards are not the question."},
	"gatekeeper":    {"Reviews a security surface: credentials, auth, tokens, permissions, supply chain.", "The phase touches no security surface. Merely naming a security word is not a surface."},
	"measurer":      {"Establishes whether a real performance, latency, memory, or cost regression exists, with numbers.", "The phase states performance is unchanged or out of scope. A phase mentioning speed is not the same as a phase changing it."},
	"architect":     {"Designs boundaries and interfaces before code is written.", "The design is already settled or the change is local and obvious."},
	"chaos":         {"Probes failure modes, bad input, and resilience under stress.", "The phase has no error path worth attacking, or nothing is deployed."},
	"tracker":       {"Finds the root cause of a defect whose cause is not yet known.", "The cause is already understood and the phase is just applying the fix."},
	"weaver":        {"Restructures existing code without changing behaviour.", "The phase adds new behaviour rather than reshaping existing code."},
	"chronicler":    {"Writes user-facing documentation, guides, and changelogs.", "No documentation surface changes in this phase."},
	"keeper":        {"Captures a reusable pattern or decision worth carrying to later work.", "Routine work that teaches nothing a future phase would want."},
	"ambassador":    {"Integrates a third-party API, SDK, or external service.", "Everything in the phase is local. An internal function named 'api' is not an integration."},
	"includer":      {"Reviews accessibility of a user interface against WCAG criteria.", "No user interface is involved."},
	"oracle":        {"Deep research into an unknown, producing a written recommendation.", "The approach is already known — this is expensive and slow."},
	"scout":         {"Quick targeted research to answer a specific question.", "Nothing is unknown."},
	"archaeologist": {"Excavates git history to explain why code is the way it is, preventing regressions.", "The area has no meaningful history, or is brand new."},
	"medic":         {"Diagnoses and repairs corrupt or stale colony state.", "Colony state is healthy — this reviews the framework, not your project."},
	"fixer":         {"Repairs a failing gate autonomously.", "No gate has failed."},
	"porter":        {"Publishes, packages, or deploys finished work.", "Nothing is being released in this phase."},
	"sage":          {"Synthesises lessons across the project's history.", "Mid-flight implementation work with nothing to retrospect on yet."},
	"route_setter":  {"Breaks a goal into ordered phases and tasks.", "The plan already exists."},

	"surveyor-nest":        {"Maps directory structure and architecture of an unfamiliar codebase.", "The codebase is already surveyed or well understood."},
	"surveyor-disciplines": {"Documents the conventions and testing patterns a codebase follows.", "Conventions are already captured."},
	"surveyor-pathogens":   {"Identifies technical debt and fragile areas.", "Not an assessment phase."},
	"surveyor-provisions":  {"Inventories dependencies and external integrations.", "Dependencies are unchanged and already known."},
}

// queenCasteRoster describes the castes a Queen may choose from. It exists so a
// decision is made against the live registry rather than the model's memory of
// caste names, and so each choice is made against what the caste produces
// rather than which words trigger it.
func queenCasteRoster() []map[string]string {
	roster := make([]map[string]string, 0, len(casteRelevanceRegistry))
	for _, profile := range casteRelevanceRegistry {
		capability, ok := casteCapabilities[profile.Caste]
		if !ok {
			// A caste with no written capability falls back to its keywords so
			// it is still choosable, but TestEveryCasteHasACapability fails so
			// the gap gets closed rather than shipped.
			capability = casteCapability{Produces: strings.Join(profile.Keywords, ", ")}
		}
		entry := map[string]string{
			"caste":    profile.Caste,
			"produces": capability.Produces,
		}
		if capability.AvoidFor != "" {
			entry["avoid_for"] = capability.AvoidFor
		}
		roster = append(roster, entry)
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
