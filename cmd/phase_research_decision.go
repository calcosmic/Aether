package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// phaseResearchExternalSignals identifies phrases suggesting a phase touches
// external technology or an integration surface. This list is intentionally
// separate from researchPhaseKeywords (cmd/plan_grounding.go), which serves
// the grounding-exemption gate and is a Phase 167 removal target (TYPED-05).
// Depending on that list here would mean un-wiring this work next phase.
var phaseResearchExternalSignals = []string{
	"api", "sdk", "webhook", "oauth", "oauth2", "protocol", "integration",
	"third-party", "external service", "http", "grpc", "websocket", "graphql",
	"provider", "vendor", "upstream", "migrate to", "upgrade to",
}

// phaseResearchReasons is a fixed lookup table of plain-English reason
// templates keyed by the signal that drove the recommendation. Mirrors the
// getSmartDefaultReason shape in cmd/review_depth.go. Only the matched signal
// token is interpolated -- never the raw phase description -- so a crafted
// phase description cannot inject instruction text into the reason line the
// Queen relays (T-164-04).
var phaseResearchReasons = map[string]string{
	"domain_gap":       "new external tech (%s) is absent from the territory survey",
	"no_survey":        "external tech (%s) mentioned, but the territory survey has no language/framework/dependency coverage yet",
	"discovery_mode":   "phase mode is discovery -- exploring unmapped territory before committing to an approach",
	"survey_covered":   "pure refactor -- the domain is already mapped by the territory survey",
	"no_external_tech": "no external technology signals found in the phase description -- the domain looks internal",
	"fast_preset":      "fast run -- speed is the default contract; flip this phase on to research it at 80% / 4 iterations",
}

// phaseResearchHint carries the cheap, deterministic signals computed for one
// candidate phase (D-02). PhaseMode is a best-effort signal: colony.Phase.Mode
// is omitempty and unpopulated for most phases until Phase 167 backfills it
// (164-RESEARCH.md Pitfall 3), so this field is frequently empty by design.
type phaseResearchHint struct {
	ExternalTech []string `json:"external_tech"`
	DomainGaps   []string `json:"domain_gaps"`
	PhaseMode    string   `json:"phase_mode"`
}

// phaseResearchRecommendation is the Queen's grounded research/skip call for
// one candidate phase, with a plain-English, non-empty reason.
type phaseResearchRecommendation struct {
	PhaseID   int               `json:"phase_id"`
	PhaseName string            `json:"phase_name"`
	Recommend string            `json:"recommend"` // "research" or "skip"
	Reason    string            `json:"reason"`
	Hints     phaseResearchHint `json:"hints"`
}

// phaseResearchProposal is the full batched, tick-to-approve proposal (D-01)
// for one plan run: one recommendation per candidate phase, always.
type phaseResearchProposal struct {
	Depth  string                        `json:"depth"`
	Replan bool                          `json:"replan"`
	Phases []phaseResearchRecommendation `json:"phases"`
}

// computePhaseResearchProposal returns exactly one grounded recommendation
// per candidate phase. On a fast run every phase is recommended "skip" (D-15)
// but every phase still appears so the user can flip one on. Otherwise the
// recommendation is grounded in runtime-computed hints, never in
// researchPhaseKeywords/isResearchPhase.
func computePhaseResearchProposal(planDepth string, replan bool, survey codexSurveyContext, candidates []phaseResearchCandidate, phases []colony.Phase) phaseResearchProposal {
	proposal := phaseResearchProposal{
		Depth:  planDepth,
		Replan: replan,
		Phases: make([]phaseResearchRecommendation, 0, len(candidates)),
	}

	fast := strings.ToLower(strings.TrimSpace(planDepth)) == "fast"
	surveyEmpty := len(survey.Languages) == 0 && len(survey.Frameworks) == 0 && len(survey.Dependencies) == 0

	phaseByID := make(map[int]colony.Phase, len(phases))
	for _, p := range phases {
		phaseByID[p.ID] = p
	}

	for _, candidate := range candidates {
		hint := computePhaseResearchHint(candidate, survey, phaseByID[candidate.ID])

		rec := phaseResearchRecommendation{
			PhaseID:   candidate.ID,
			PhaseName: candidate.Name,
			Hints:     hint,
		}

		switch {
		case fast:
			rec.Recommend = "skip"
			rec.Reason = phaseResearchReasons["fast_preset"]
		case len(hint.DomainGaps) > 0:
			rec.Recommend = "research"
			rec.Reason = fmt.Sprintf(phaseResearchReasons["domain_gap"], hint.DomainGaps[0])
		case len(hint.ExternalTech) > 0 && surveyEmpty:
			rec.Recommend = "research"
			rec.Reason = fmt.Sprintf(phaseResearchReasons["no_survey"], hint.ExternalTech[0])
		case hint.PhaseMode == string(colony.PhaseModeDiscovery):
			rec.Recommend = "research"
			rec.Reason = phaseResearchReasons["discovery_mode"]
		case len(hint.ExternalTech) == 0:
			rec.Recommend = "skip"
			rec.Reason = phaseResearchReasons["no_external_tech"]
		default:
			rec.Recommend = "skip"
			rec.Reason = phaseResearchReasons["survey_covered"]
		}

		proposal.Phases = append(proposal.Phases, rec)
	}

	return proposal
}

// computePhaseResearchHint computes the deterministic, lowercase-normalised
// signals for one candidate phase (D-02). phase is the zero value when no
// matching colony.Phase was found by ID -- Mode.Valid() correctly returns
// false in that case, matching the guard checkPlanGrounding already uses.
func computePhaseResearchHint(candidate phaseResearchCandidate, survey codexSurveyContext, phase colony.Phase) phaseResearchHint {
	text := strings.ToLower(candidate.Name + " " + candidate.Description)

	var externalTech []string
	for _, signal := range phaseResearchExternalSignals {
		if strings.Contains(text, signal) {
			externalTech = append(externalTech, signal)
		}
	}

	var domainGaps []string
	for _, tech := range externalTech {
		if surveyContainsSignal(survey.Languages, tech) ||
			surveyContainsSignal(survey.Frameworks, tech) ||
			surveyContainsSignal(survey.Dependencies, tech) {
			continue
		}
		domainGaps = append(domainGaps, tech)
	}

	var mode string
	if phase.Mode.Valid() {
		mode = string(phase.Mode)
	}

	return phaseResearchHint{
		ExternalTech: externalTech,
		DomainGaps:   domainGaps,
		PhaseMode:    mode,
	}
}

// surveyContainsSignal reports whether signal appears as a substring in any
// lowercased entry of entries.
func surveyContainsSignal(entries []string, signal string) bool {
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry), signal) {
			return true
		}
	}
	return false
}

// phaseResearchDecisionType is the PendingDecision.Type value used for
// research/skip decisions and their overrides. RESEARCH-06 forbids a new
// planning store, so these are recorded as PendingDecision entries, not in a
// new struct or a new JSON file.
const phaseResearchDecisionType = "research-decision"

// renderPhaseResearchProposalBlock renders the batched, tick-to-approve
// proposal (D-01): a header naming the depth and whether this is a replan,
// then per phase a [research]/[skip] tag line, an indented reason line, and
// an indented per-phase flip line. Exactly one approve-all line closes the
// whole batch -- one interaction for the batch, not one per phase. Mirrors
// renderPendingSuggestionsBlock's shape (cmd/ceremony_cmd.go).
func renderPhaseResearchProposalBlock(p phaseResearchProposal) string {
	if len(p.Phases) == 0 {
		return ""
	}

	var b strings.Builder
	header := fmt.Sprintf("Research proposal (%s depth", firstNonEmpty(strings.TrimSpace(p.Depth), "standard"))
	if p.Replan {
		header += ", replan"
	}
	header += "):"
	b.WriteString(header)
	b.WriteString("\n")

	for _, rec := range p.Phases {
		fmt.Fprintf(&b, "[%s] Phase %d: %s\n", rec.Recommend, rec.PhaseID, rec.PhaseName)
		fmt.Fprintf(&b, "  Reason: %s\n", rec.Reason)
		fmt.Fprintf(&b, "  Flip: aether plan-research-approve --flip %d\n", rec.PhaseID)
	}

	b.WriteString("Approve all: aether plan-research-approve --approve-all\n")
	return b.String()
}

// newPhaseResearchDecision builds a PendingDecision for a phase research
// recommendation, using the same ID/CreatedAt stamping shape
// pendingDecisionAddCmd uses (cmd/pending_decision.go). No new struct is
// declared for decision storage -- the existing PendingDecision type carries
// this record end to end.
func newPhaseResearchDecision(rec phaseResearchRecommendation) PendingDecision {
	phaseID := rec.PhaseID
	return PendingDecision{
		ID:          fmt.Sprintf("prd_%d_%d", rec.PhaseID, time.Now().UnixNano()),
		Type:        phaseResearchDecisionType,
		Description: fmt.Sprintf("%s phase %d (%s): %s", rec.Recommend, rec.PhaseID, rec.PhaseName, rec.Reason),
		Source:      "queen-research-proposal",
		Phase:       &phaseID,
		Resolved:    false,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

// phaseResearchDecisionResolution builds the resolution string written when a
// research decision is resolved -- the durable record of the flip (T-164-06),
// so it carries the resulting direction, not just the fact of a flip.
func phaseResearchDecisionResolution(rec phaseResearchRecommendation, flipped bool, auto bool) string {
	var resolution string
	if flipped {
		resolution = fmt.Sprintf("user overrode: %s research on phase %d", oppositeRecommend(rec.Recommend), rec.PhaseID)
	} else {
		resolution = fmt.Sprintf("approved: %s phase %d", rec.Recommend, rec.PhaseID)
	}
	if auto {
		resolution = "auto-accepted (autopilot) -- " + resolution
	}
	return resolution
}

// oppositeRecommend returns the opposite research direction of recommend.
func oppositeRecommend(recommend string) string {
	if recommend == "skip" {
		return "research"
	}
	return "skip"
}

// resolvePhaseResearchDecisions walks a PendingDecisionFile and returns a map
// from phase ID to the resolved direction ("research" or "skip"). Decisions
// whose Type is not phaseResearchDecisionType, whose Phase pointer is nil, or
// whose Resolved is false are skipped. When two resolved decisions exist for
// the same phase, the later one in the slice wins (map assignment order),
// matching D-06: a replan re-proposes and the newest record is the live one.
func resolvePhaseResearchDecisions(file PendingDecisionFile) map[int]string {
	result := make(map[int]string)
	for _, d := range file.Decisions {
		if d.Type != phaseResearchDecisionType {
			continue
		}
		if d.Phase == nil {
			continue
		}
		if !d.Resolved {
			continue
		}
		direction := extractResearchDirection(d.Resolution)
		if direction == "" {
			continue
		}
		result[*d.Phase] = direction
	}
	return result
}

// extractResearchDirection derives "research" or "skip" from a resolution
// string by checking the text after the first colon. "skip" is checked
// before "research" so a flip-to-skip resolution (which also names
// "research" as the noun being skipped, e.g. "skip research on phase N")
// resolves to "skip", the direction that actually took effect.
func extractResearchDirection(resolution string) string {
	tail := resolution
	if idx := strings.Index(resolution, ":"); idx >= 0 {
		tail = resolution[idx+1:]
	}
	tail = strings.ToLower(tail)
	switch {
	case strings.Contains(tail, "skip"):
		return "skip"
	case strings.Contains(tail, "research"):
		return "research"
	default:
		return ""
	}
}
