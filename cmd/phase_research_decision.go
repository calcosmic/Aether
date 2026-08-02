package cmd

import (
	"fmt"
	"strings"

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
