package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// CasteRelevanceProfile defines when a caste is relevant to a phase.
type CasteRelevanceProfile struct {
	Caste      string
	Keywords   []string
	Conditions []string // e.g., "mode==discovery", "risk==high"
	BaseScore  int
	// ReasonTemplate is D-09's answer for the fallback (no-proposal) engine:
	// a plain-English sentence fragment with exactly one %q verb, filled with
	// the literal keyword that matched this phase. It replaces the old
	// "Score N >= threshold M for FLOW flow" string, which named an internal
	// number and the flow-type identifier -- neither actionable by a reader
	// who has never opened this repo. A caste with no entry here, and no
	// matched keyword to fill it, is not returned as a fallback candidate at
	// all (queenCandidateRationale): a pick the engine cannot word is not
	// sent.
	ReasonTemplate string
}

// CasteDispatch represents a caste the Queen has chosen to spawn.
type CasteDispatch struct {
	Caste     string
	Score     int
	Rationale string
	FlowType  string
}

// casteRelevanceRegistry holds profiles for dispatchable worker castes.
// Base scores are deliberately low (10-20) so castes only spawn when keywords match
// or a flow policy marks them as required.
var casteRelevanceRegistry = []CasteRelevanceProfile{
	{Caste: "builder", Keywords: []string{"implement", "build", "create", "add", "write", "fix", "code", "deploy"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, something to implement"},
	{Caste: "watcher", Keywords: []string{"verify", "test", "validate", "check", "review", "quality"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, something to verify"},
	{Caste: "scout", Keywords: []string{"research", "investigate", "survey", "analyze", "document", "readme", "spec", "explore"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, something to research"},
	{Caste: "route_setter", Keywords: []string{"plan", "route", "decompose", "structure", "organize"}, BaseScore: 15, ReasonTemplate: "the phase mentions %q, something to plan out"},
	{Caste: "architect", Keywords: []string{"design", "schema", "architecture", "interface", "boundary", "structure", "evaluate"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, a design decision to make before code is written"},
	{Caste: "oracle", Keywords: []string{"research", "spike", "investigate", "evaluate", "unknown", "deep dive", "survey"}, Conditions: []string{"mode==discovery"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, something worth deep research"},
	{Caste: "chaos", Keywords: []string{"resilience", "failure", "robustness", "crash", "error handling", "stress test"}, BaseScore: 15, ReasonTemplate: "the phase mentions %q, a resilience concern"},
	{Caste: "archaeologist", Keywords: []string{"legacy", "migration", "modernize", "rewrite", "history", "refactor old"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, history worth understanding before touching it"},
	{Caste: "gatekeeper", Keywords: []string{"auth", "crypto", "security", "token", "secrets", "permissions", "compliance", "audit"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, a security-relevant surface"},
	// The mode==production condition that used to sit here alongside
	// applySpecialRules' now-deleted 100-score branch is gone too (not just
	// the special rule): together the two implicit "production mode ⇒
	// auditor" floors survived even after only one was removed, since base
	// score (20) plus the condition boost (20) still cleared the spawn
	// threshold (30) with no keyword match at all. D-06 removes this
	// mechanism wherever it hides, not just the loudest copy of it.
	{Caste: "auditor", Keywords: []string{"compliance", "audit", "production", "release", "quality gate", "standards"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, a quality or compliance concern"},
	{Caste: "probe", Keywords: []string{"test coverage", "edge case", "validation", "verify", "missing tests", "coverage gap"}, BaseScore: 15, ReasonTemplate: "the phase mentions %q, a test-coverage concern"},
	{Caste: "measurer", Keywords: []string{"performance", "optimize", "latency", "scale", "benchmark", "memory", "cpu"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, a performance question"},
	{Caste: "ambassador", Keywords: []string{"api", "sdk", "oauth", "external service", "external integration", "integration", "webhook", "third-party", "stripe", "sendgrid", "twilio", "openai", "aws", "azure", "gcp"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, an external integration"},
	{Caste: "tracker", Keywords: []string{"bug", "fix", "regression", "investigate failure", "root cause", "issue"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, pointing at a bug to investigate"},
	{Caste: "weaver", Keywords: []string{"refactor", "cleanup", "modernize", "extract", "simplify", "restructure"}, BaseScore: 20, ReasonTemplate: "the phase mentions %q, a restructure without changing behaviour"},
	// "pattern", "standard" and "document" are everyday engineering words, and
	// at base 10 two incidental hits cleared the continue threshold — a phase
	// that merely mentioned documenting a standard summoned a Keeper.
	// "document" belongs to chronicler; keeper keys on preservation intent.
	// Chronicler's keyword is "document", not "documentation": keyword
	// matching anchors on the LEFT boundary only, so "documentation" never
	// matched the word "document" and the handoff the line above describes
	// was never actually made -- the word ended up owned by nobody.
	{Caste: "keeper", Keywords: []string{"knowledge", "convention", "preserve", "wisdom", "institutional"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, worth preserving as institutional knowledge"},
	{Caste: "chronicler", Keywords: []string{"document", "docs", "guide", "readme", "changelog", "manual"}, BaseScore: 15, ReasonTemplate: "the phase mentions %q, something to document"},
	{Caste: "includer", Keywords: []string{"accessibility", "a11y", "wcag", "screen reader", "aria", "inclusive"}, BaseScore: 15, ReasonTemplate: "the phase mentions %q, an accessibility concern"},
	{Caste: "surveyor-provisions", Keywords: []string{"dependency", "dependencies", "provisions", "external", "integration", "stack", "package"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a dependency worth mapping"},
	{Caste: "surveyor-nest", Keywords: []string{"architecture", "structure", "layout", "map", "chamber", "directory"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a structural question worth mapping"},
	{Caste: "surveyor-disciplines", Keywords: []string{"convention", "discipline", "testing", "pattern", "practice", "standard"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a convention worth documenting"},
	{Caste: "surveyor-pathogens", Keywords: []string{"pathogen", "debt", "fragile", "risk", "bug", "health", "failure"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a sign of technical debt"},
	{Caste: "medic", Keywords: []string{"health", "diagnose", "repair", "heal", "fix state"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a sign colony state needs repair"},
	{Caste: "fixer", Keywords: []string{"auto-fix", "repair", "patch", "remediate", "self-heal"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, something to autonomously repair"},
	{Caste: "porter", Keywords: []string{"deploy", "deliver", "ship", "publish", "release", "package"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a release or delivery step"},
	{Caste: "sage", Keywords: []string{"wisdom", "synthesize", "learn", "pattern", "retrospective"}, BaseScore: 10, ReasonTemplate: "the phase mentions %q, a lesson worth synthesising"},
}

// casteRelevanceScore returns a 0-100 score for how relevant a caste is to a phase.
// keywordGatedCastes only spawn on a real keyword hit, never on base score.
// Their value is entirely conditional on the phase touching an external
// surface; with no such surface there is nothing for them to review.
var keywordGatedCastes = map[string]bool{
	"ambassador": true,
	"gatekeeper": true,
}

// containsKeyword matches a keyword that begins at a word boundary. Plain
// substring matching dispatched an Ambassador to a local slugify phase because
// "api" sits inside "cAPItalize" — a false positive that spends a whole worker
// run and makes caste selection look arbitrary.
//
// The match is anchored at the start only, not both ends: "structure" must
// still match "structured" and "test" must match "tests", because those are
// the same concept inflected. Requiring a boundary at both ends silently
// dropped the Architect from design phases.
func containsKeyword(text, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], keyword)
		if idx < 0 {
			return false
		}
		start := offset + idx
		if start == 0 || !isWordByte(text[start-1]) {
			return true
		}
		offset = start + 1
	}
	return false
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func casteRelevanceScore(phase colony.Phase, caste string) int {
	profile := findProfile(caste)
	if profile == nil {
		return 0
	}

	score := profile.BaseScore
	text := collectPhaseText(phase)

	// Keyword matching
	keywordMatches := 0
	for _, kw := range profile.Keywords {
		if containsKeyword(text, kw) {
			keywordMatches++
		}
	}
	score += keywordMatches * 10

	// Castes whose work only exists when the phase actually touches their
	// domain must not ride in on base score alone. An ambassador dispatched to
	// a local string-utility phase can only report that it had no work to do,
	// which costs a worker run and reads to the user as the colony being
	// confused.
	if keywordMatches == 0 && keywordGatedCastes[caste] {
		return 0
	}

	// Condition matching
	for _, cond := range profile.Conditions {
		score += conditionScore(phase, cond)
	}

	// Special rules
	score = applySpecialRules(phase, caste, score)

	// Clamp to 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// queenOrchestrate decides which castes to dispatch for a given flow.
func queenOrchestrate(phase colony.Phase, flowType string, state colony.ColonyState) []CasteDispatch {
	return applyQueenSpawnBudget(queenFallbackTeam(phase, flowType, state), phase, flowType, state)
}

// queenSelectorIsGatedForFlow reports whether the no-proposal fallback team
// for flowType comes from the phase's own required-caste floor
// (queenFallbackTeam) rather than the keyword/relevance selector
// (queenCandidateDispatches). D-11 gates exactly two flows -- build and
// continue, the two flows autopilot and an unattended wrapper actually run
// with no proposal -- because a one-task bug fix was measured at eight
// workers there. Plan, colonize, swarm and seal are untouched by this
// ruling (194-CONTEXT.md D-12: "only build and continue change in this
// phase"), so they still select through the relevance engine. One function
// name is the gate everywhere the choice is made, rather than two copies of
// the same switch statement drifting apart.
func queenSelectorIsGatedForFlow(flowType string) bool {
	switch normalizeQueenFlowType(flowType) {
	case "build", "continue":
		return true
	default:
		return false
	}
}

// queenFallbackTeam is the no-proposal team for a phase: what dispatches when
// nobody -- neither the chat's --castes proposal nor a human on the check-in
// card -- said anything about who should go.
//
// Before this existed, that silence was answered by
// queenCandidateDispatches: every caste in the registry scored against the
// phase's wording, and anything clearing spawnThreshold rode along. That is
// the mechanism a 2026-08-22 measurement caught sending eight workers to a
// one-task bug fix (194-CONTEXT.md, the v1.27 milestone brief) -- keyword
// arithmetic standing in for a judgement nobody was asked to make.
//
// D-11 answers it instead: on build and continue, the fallback team is
// exactly what queenRequiredCastesForBudget says this phase cannot ship
// without -- the caste that writes the code, plus (on continue) any reviewer
// a named risk signal forces. D-12 adds one exception: a discovery-mode
// build suppresses the implementation caste today (isCasteSuppressed), so a
// discovery phase with nothing required at all would dispatch nobody; it
// gets one researcher instead, because research is the deliverable there.
//
// Every other flow (plan, colonize, swarm, seal) is untouched: this function
// delegates to queenCandidateDispatches for them exactly as queenOrchestrate
// used to call it directly, so their caste sets are byte-for-byte what they
// were before this phase (TestOtherFlowsKeepTheirRequiredSets).
func queenFallbackTeam(phase colony.Phase, flowType string, state colony.ColonyState) []CasteDispatch {
	flowType = normalizeQueenFlowType(flowType)
	if !queenSelectorIsGatedForFlow(flowType) {
		return queenCandidateDispatches(phase, flowType, state)
	}

	var forced []forcedReviewer
	if flowType == "continue" {
		forced = queenForcedReviewersForPhase(phase)
	}

	required := queenRequiredCastesForBudget(phase, flowType, state)
	dispatches := make([]CasteDispatch, 0, len(required)+1)
	for _, caste := range required {
		rationale := queenRuntimeReasonForCaste(caste, phase, forced)
		if rationale == "" {
			rationale = queenAlwaysRequiredReason(caste, flowType, phase)
		}
		dispatches = append(dispatches, CasteDispatch{
			Caste:     caste,
			Score:     100,
			Rationale: rationale,
			FlowType:  flowType,
		})
	}

	// D-12: discovery suppresses the implementation caste at build
	// (isCasteSuppressed), so a discovery phase's required list above is
	// empty. Findings are the deliverable there, so the fallback is one
	// researcher rather than nobody.
	if flowType == "build" && effectiveQueenPhaseMode(phase) == colony.PhaseModeDiscovery {
		dispatches = append(dispatches, CasteDispatch{
			Caste:     "scout",
			Score:     100,
			Rationale: "this is a discovery phase, so research is the deliverable",
			FlowType:  flowType,
		})
	}

	return dispatches
}

func queenCandidateDispatches(phase colony.Phase, flowType string, state colony.ColonyState) []CasteDispatch {
	var dispatches []CasteDispatch
	seen := make(map[string]struct{}, len(casteRelevanceRegistry))
	flowType = normalizeQueenFlowType(flowType)
	threshold := spawnThreshold(flowType, state)

	for _, profile := range casteRelevanceRegistry {
		if !casteAllowedForFlow(profile.Caste, flowType) {
			continue
		}
		if isCasteSuppressed(profile.Caste, flowType, phase, state) {
			continue
		}

		score := casteRelevanceScore(phase, profile.Caste)
		always := isAlwaysRequired(profile.Caste, flowType, phase, state)

		if always || score >= threshold {
			// D-09: no score arithmetic and no bare caste/flow-type name in a
			// reason slot, ever -- a candidate whose selection this phase
			// cannot word in plain English is not returned at all, rather than
			// sent with a placeholder.
			rationale := queenCandidateRationale(phase, profile, flowType, always)
			if rationale == "" {
				continue
			}
			if always {
				score = 100
			}
			if _, ok := seen[profile.Caste]; ok {
				continue
			}
			seen[profile.Caste] = struct{}{}
			dispatches = append(dispatches, CasteDispatch{
				Caste:     profile.Caste,
				Score:     score,
				Rationale: rationale,
				FlowType:  flowType,
			})
		}
	}

	return dispatches
}

// matchedKeywordFor returns the first of profile's keywords the phase's text
// contains, or "" if none matched -- the caste cleared the spawn threshold
// through base score plus a Conditions boost alone (e.g. Oracle on a
// discovery-mode phase with no keyword hit).
func matchedKeywordFor(phase colony.Phase, profile CasteRelevanceProfile) string {
	text := collectPhaseText(phase)
	for _, kw := range profile.Keywords {
		if containsKeyword(text, kw) {
			return kw
		}
	}
	return ""
}

// queenCandidateRationale is D-09's replacement for the deleted "Score N >=
// threshold M for FLOW flow" and "CASTE is always required for FLOW flow"
// strings: a sentence about what in THIS phase called for the caste. A caste
// this function cannot word for is not returned as a candidate at all
// (queenCandidateDispatches drops an empty result rather than sending a
// placeholder).
func queenCandidateRationale(phase colony.Phase, profile CasteRelevanceProfile, flowType string, always bool) string {
	if always {
		return queenAlwaysRequiredReason(profile.Caste, flowType, phase)
	}
	if keyword := matchedKeywordFor(phase, profile); keyword != "" && profile.ReasonTemplate != "" {
		return fmt.Sprintf(profile.ReasonTemplate, keyword)
	}
	// No literal keyword matched, so the caste cleared the threshold through
	// a phase-mode Conditions boost instead. Word the condition itself
	// rather than leaving a scored-but-unworded pick unexplained.
	return queenConditionRationale(phase, profile.Caste)
}

// queenAlwaysRequiredReason is D-09's plain-English answer for a caste the
// runtime requires regardless of relevance scoring (isAlwaysRequired) --
// never the deleted "X is always required for FLOW flow" phrasing, which
// named the internal flow-type identifier rather than a reason a reader could
// act on.
func queenAlwaysRequiredReason(caste, flowType string, phase colony.Phase) string {
	switch caste {
	case "builder":
		return "this phase has tasks that need code written"
	case "watcher":
		return "the work this phase produces needs an independent check before it advances"
	case "probe":
		return "this phase produces code worth testing, so a coverage gap is worth catching"
	case "gatekeeper":
		return "this phase's risk level calls for a security review before advancing"
	case "auditor":
		return "this phase's risk level calls for a quality review before advancing"
	case "route_setter":
		return "the goal still needs breaking down into ordered phases and tasks"
	case "scout":
		if effectiveQueenPhaseMode(phase) == colony.PhaseModeDiscovery {
			return "this is a discovery phase, so research is the deliverable"
		}
		return "the plan needs a look at the codebase before it can be broken down"
	case "surveyor-nest":
		return "the colony cannot start without a map of the codebase's structure"
	case "surveyor-provisions":
		return "the colony cannot start without knowing what this codebase depends on"
	case "surveyor-disciplines":
		return "the colony needs the codebase's conventions and testing patterns captured"
	case "surveyor-pathogens":
		return "the colony needs the codebase's technical debt and fragile areas mapped"
	case "tracker":
		return "a bug investigation always needs someone finding the root cause"
	}
	return ""
}

// queenConditionRationale words a caste whose score cleared the threshold
// through a phase-mode Conditions boost (conditionScore) rather than a
// literal keyword hit.
func queenConditionRationale(phase colony.Phase, caste string) string {
	switch caste {
	case "oracle":
		if phase.Mode == colony.PhaseModeDiscovery {
			return "this is a discovery phase and the approach is not yet known"
		}
	case "architect":
		if phaseRiskLevel(phase) == "high" {
			return "this is high-risk work, and a design boundary is worth setting before code is written"
		}
	}
	return ""
}

// findProfile looks up a caste's relevance profile.
func findProfile(caste string) *CasteRelevanceProfile {
	for i := range casteRelevanceRegistry {
		if casteRelevanceRegistry[i].Caste == caste {
			return &casteRelevanceRegistry[i]
		}
	}
	return nil
}

// conditionScore adds score for matched conditions.
func conditionScore(phase colony.Phase, condition string) int {
	switch condition {
	case "mode==discovery":
		if phase.Mode == colony.PhaseModeDiscovery {
			return 20
		}
	case "mode==production":
		if phase.Mode == colony.PhaseModeProduction {
			return 20
		}
	}
	return 0
}

// applySpecialRules handles hardcoded caste rules.
func applySpecialRules(phase colony.Phase, caste string, score int) int {
	text := collectPhaseText(phase)

	switch caste {
	case "builder":
		// Always include if any task has implementation keywords
		if hasImplementationTask(phase.Tasks) {
			return 100
		}
	case "architect":
		// Auto-include for high-risk phases (security work needs design boundaries)
		if phaseRiskLevel(phase) == "high" {
			return 100
		}
	case "watcher":
		// Flow policy decides whether watcher is mandatory.
		return score
	case "oracle":
		// Auto-include for discovery mode
		if phase.Mode == colony.PhaseModeDiscovery {
			return 100
		}
	case "chaos":
		// Exclude for discovery mode
		if phase.Mode == colony.PhaseModeDiscovery {
			return 0
		}
	// The gatekeeper-on-high-risk and auditor-on-production 100-score
	// branches that used to sit here are deleted (D-06): they were the
	// implicit "high risk ⇒ security review" and "production mode ⇒ quality
	// review" floors, restated as scores instead of as required castes --
	// the same rule Ruling D11 already removed from
	// queenBuildSafetyRequiredCastes, still alive here under a different
	// name. With the selector gated (plan 194-05's probe-refusal work), what
	// is left of these castes' scores now only decides REFUSAL (a proposed
	// caste scoring 0 is still refused for having nothing to do), never
	// selection -- neither caste is summoned by mode or risk alone anymore.
	case "scout":
		// Boost for research-heavy phases
		if strings.Contains(text, "research") || strings.Contains(text, "investigate") {
			score += 15
		}
	}

	return score
}

// spawnThreshold returns the minimum score to spawn for a flow type.
func spawnThreshold(flowType string, state colony.ColonyState) int {
	switch flowType {
	case "build", "continue":
		if flowType == "continue" && stateVerificationDepth(state) == colony.VerificationDepthHeavy {
			// Heavy is an explicit request for breadth, so the bar stays low.
			return 25
		}
		// A base-20 caste scores 30 on a single keyword hit, so a threshold of
		// 30 meant one incidental word bought a full agent run. The old value
		// carried the comment "Allow more castes through; Queen filters later"
		// — and no Queen filtered, because no Queen existed. The system
		// deliberately over-selected on the promise of a downstream filter that
		// was never written.
		//
		// Deliberately left at 30, and deliberately NOT raised. Raising it was
		// tried and measured, and the measurement killed the idea:
		//
		//   phase "Migrate user data to new schema"      -> architect, 1 hit  (correct)
		//   phase "latency and memory ... is unchanged"  -> measurer,  2 hits (waste)
		//
		// The false positive scores HIGHER than the true positive, because
		// counting keyword hits is anti-correlated with correctness once
		// negation is in play. At 40 the wasteful Measurer survived and the DB
		// migration lost its Architect — strictly worse on both counts. No
		// threshold separates those two phases, because the signal that
		// distinguishes them is meaning, not word frequency.
		//
		// So this number is not the lever. queenApplyJudgement is: a reader can
		// see that "unchanged" negates the words around it and that a migration
		// needs design review whether or not it repeats the word "schema".
		// Keep this as the floor that holds when no model intervenes, and put
		// the effort into the judgement layer instead of retuning this.
		//
		// As of plan 194-05 (D-11) this threshold no longer selects anything
		// on build or continue at all: queenOrchestrate calls
		// queenFallbackTeam for those two flows, which answers "nobody
		// proposed a team" with the phase's required-caste floor, not a
		// score comparison. This function and casteRelevanceScore survive
		// only as the REFUSAL check queenApplyJudgement still consults (a
		// proposed caste scoring 0 is refused) and as the candidate list
		// queenCandidateDispatches still builds for a reading Queen and for
		// every other flow (plan, colonize, swarm, seal). The number below
		// is unchanged, and this comment's history stays because the
		// reasoning about WHY the threshold cannot substitute for judgement
		// still applies to those other flows -- see the measured eight-
		// worker bug fix this ruling exists to stop
		// (.planning/decisions/2026-08-22-queen-decides-program-checks.md).
		return 30
	case "plan":
		return 40
	case "colonize", "swarm":
		return 35
	case "seal":
		return 50
	default:
		return 35
	}
}

// isAlwaysRequired checks if a caste must always spawn for a flow.
func isAlwaysRequired(caste, flowType string, phase colony.Phase, state colony.ColonyState) bool {
	switch flowType {
	case "build":
		return queenBuildSafetyRequiredCaste(caste, flowType, phase)
	case "continue":
		switch stateVerificationDepth(state) {
		case colony.VerificationDepthHeavy:
			// Heavy is the owner's one dial that still outranks the Queen's
			// judgement (D-13): the full review panel, gatekeeper + auditor
			// + probe. "Full panel" still means probe only where there is
			// something for it to cover -- probe used to stay unconditional
			// even on a documentation phase, billing a worker run to report
			// it found nothing. Watcher's unconditional membership here is
			// gone: it was the last place the 2026-08-22 ruling's removed
			// review floor survived under a different depth than light and
			// standard.
			return caste == "gatekeeper" || caste == "auditor" ||
				(caste == "probe" && queenPhaseProducesTestableCode(phase))
		default:
			// Light and standard require NOTHING unconditionally (D-13).
			// The review team at these depths is entirely the Queen's
			// judgement plus whatever a named risk signal forces
			// (queenForcedReviewersForPhase, folded into
			// queenRequiredCastesForBudget's own continue branch) -- neither
			// dial reaches this switch at all. Watcher and Probe's
			// unconditional membership here used to be the "verify once"
			// floor from Phase 193's own D-08; this phase (D-13) removes it
			// as a REQUIREMENT, matching the dispatch, which Phase 193
			// already stopped sending unconditionally.
			return false
		}
	case "plan":
		return caste == "scout" || caste == "route_setter"
	case "colonize":
		// Light keeps the two surveys a colony cannot start without —
		// structure (nest) and dependencies (provisions); conventions and
		// tech-debt surveys are the optional depth.
		if stateVerificationDepth(state) == colony.VerificationDepthLight {
			return caste == "surveyor-provisions" || caste == "surveyor-nest"
		}
		return caste == "surveyor-provisions" ||
			caste == "surveyor-nest" ||
			caste == "surveyor-disciplines" ||
			caste == "surveyor-pathogens"
	case "swarm":
		if caste == "gatekeeper" && phaseRiskLevel(phase) == "high" {
			return true
		}
		// Scout and Archaeologist used to be unconditionally required here, so
		// a one-line typo fix paid for a researcher and a git-history dig that
		// could only report finding nothing. They now ride on keyword
		// relevance like every other specialist; the investigate/fix/verify
		// trio stays mandatory because a swarm without them is not a swarm.
		return caste == "tracker" ||
			caste == "builder" ||
			caste == "watcher"
	case "seal":
		switch stateVerificationDepth(state) {
		case colony.VerificationDepthLight:
			return false
		case colony.VerificationDepthHeavy:
			return caste == "gatekeeper" || caste == "auditor" || caste == "probe"
		default:
			// Probe on a documentation-only final phase has no code to cover —
			// the same gate build and continue already apply. Heavy stays
			// unconditional above: heavy is an explicit request for breadth.
			return caste == "auditor" || (caste == "probe" && queenPhaseProducesTestableCode(phase))
		}
	}
	return false
}

func normalizeQueenFlowType(flowType string) string {
	flowType = strings.ToLower(strings.TrimSpace(flowType))
	if flowType == "" {
		return "build"
	}
	return flowType
}

func stateVerificationDepth(state colony.ColonyState) colony.VerificationDepth {
	return colony.NormalizeVerificationDepth(state.VerificationDepth)
}

func casteAllowedForFlow(caste, flowType string) bool {
	switch flowType {
	case "build":
		// route_setter plans phases; the build composer has no dispatch for
		// it, so selecting it only displaced a real specialist
		// (TestEveryBuildSelectableCasteCanDispatch).
		return caste != "route_setter" && !strings.HasPrefix(caste, "surveyor-")
	case "continue":
		return oneOf(caste, "watcher", "gatekeeper", "auditor", "probe", "measurer", "chaos", "includer", "keeper", "sage", "medic", "fixer")
	case "plan":
		return oneOf(caste, "scout", "route_setter", "architect", "oracle", "keeper", "chronicler", "includer", "gatekeeper")
	case "colonize":
		return strings.HasPrefix(caste, "surveyor-")
	case "swarm":
		return oneOf(caste, "tracker", "scout", "archaeologist", "builder", "watcher", "gatekeeper", "probe", "weaver", "medic", "fixer")
	case "seal":
		return oneOf(caste, "gatekeeper", "auditor", "probe", "porter", "chronicler", "keeper", "sage", "measurer", "includer")
	default:
		return !strings.HasPrefix(caste, "surveyor-")
	}
}

func isCasteSuppressed(caste, flowType string, phase colony.Phase, state colony.ColonyState) bool {
	if flowType == "build" && phase.Mode == colony.PhaseModeDiscovery {
		return oneOf(caste, "builder", "weaver", "fixer", "porter")
	}
	// Oracle in a build flow is gated by the phase's TYPED mode, never prose.
	// The word "research" in a production phase's description used to score
	// Oracle past the spawn threshold — the original prose-steers-dispatch bug.
	if flowType == "build" && caste == "oracle" && effectiveQueenPhaseMode(phase) != colony.PhaseModeDiscovery {
		return true
	}
	if flowType == "seal" && stateVerificationDepth(state) == colony.VerificationDepthLight {
		return oneOf(caste, "gatekeeper", "auditor", "probe")
	}
	if flowType == "colonize" && stateVerificationDepth(state) == colony.VerificationDepthLight {
		return oneOf(caste, "surveyor-disciplines", "surveyor-pathogens")
	}
	if flowType == "continue" || flowType == "seal" {
		return oneOf(caste, "builder", "weaver", "tracker", "archaeologist", "ambassador")
	}
	return false
}

func oneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

// hasImplementationTask checks if any task has implementation keywords.
func hasImplementationTask(tasks []colony.Task) bool {
	// Use word-boundary matching to avoid false positives like "research" matching "search"
	implKeywords := []string{"implement", "build", "create", "fix", "add", "write", "code", "deploy"}
	for _, task := range tasks {
		text := strings.ToLower(" " + task.Goal + " ")
		for _, kw := range implKeywords {
			if strings.Contains(text, " "+kw+" ") {
				return true
			}
		}
	}
	return false
}

// HasCaste checks if a caste is in the dispatch list.
func HasCaste(dispatches []CasteDispatch, caste string) bool {
	for _, d := range dispatches {
		if d.Caste == caste {
			return true
		}
	}
	return false
}

// FilterCastesByMinScore filters dispatches below a score threshold.
func FilterCastesByMinScore(dispatches []CasteDispatch, minScore int) []CasteDispatch {
	var filtered []CasteDispatch
	for _, d := range dispatches {
		if d.Score >= minScore {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
