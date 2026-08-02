package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestQueenResearchDecision(t *testing.T) {
	t.Run("external token absent from survey recommends research and names the token", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 1, Name: "Integrate billing provider", Description: "Wire up the new OAuth2 API for the third-party payment provider"},
		}
		survey := codexSurveyContext{
			Languages: []string{"go"},
		}

		proposal := computePhaseResearchProposal("balanced", false, survey, candidates, nil)

		if len(proposal.Phases) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(proposal.Phases))
		}
		rec := proposal.Phases[0]
		if rec.Recommend != "research" {
			t.Fatalf("expected recommend=research, got %q", rec.Recommend)
		}
		if len(rec.Hints.DomainGaps) == 0 {
			t.Fatalf("expected at least one domain gap")
		}
		if !strings.Contains(rec.Reason, rec.Hints.DomainGaps[0]) {
			t.Fatalf("expected reason %q to name the domain gap token %q", rec.Reason, rec.Hints.DomainGaps[0])
		}
	})

	t.Run("technology already in survey recommends skip and reason says domain is mapped", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 2, Name: "Refactor HTTP handlers", Description: "Clean up the existing http handler layer, no new endpoints"},
		}
		survey := codexSurveyContext{
			Languages:  []string{"go"},
			Frameworks: []string{"net/http"},
		}

		proposal := computePhaseResearchProposal("balanced", false, survey, candidates, nil)

		if len(proposal.Phases) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(proposal.Phases))
		}
		rec := proposal.Phases[0]
		if rec.Recommend != "skip" {
			t.Fatalf("expected recommend=skip, got %q", rec.Recommend)
		}
		if !strings.Contains(rec.Reason, "already mapped") {
			t.Fatalf("expected reason to say the domain is already mapped, got %q", rec.Reason)
		}
	})

	t.Run("fast preset recommends skip for every phase but lists every phase with a flip reason", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 1, Name: "Integrate billing provider", Description: "New external OAuth2 API"},
			{ID: 2, Name: "Refactor internals", Description: "Pure refactor"},
		}
		survey := codexSurveyContext{}

		proposal := computePhaseResearchProposal("fast", false, survey, candidates, nil)

		if len(proposal.Phases) != len(candidates) {
			t.Fatalf("expected %d recommendations, got %d", len(candidates), len(proposal.Phases))
		}
		for _, rec := range proposal.Phases {
			if rec.Recommend != "skip" {
				t.Fatalf("expected recommend=skip for phase %d on fast preset, got %q", rec.PhaseID, rec.Recommend)
			}
			if !strings.Contains(rec.Reason, "80%") || !strings.Contains(rec.Reason, "4") {
				t.Fatalf("expected fast-preset reason to mention flipping on at 80%%/4 iterations, got %q", rec.Reason)
			}
		}
	})

	t.Run("researchPhaseKeywords-named phase with no external tokens and full survey coverage is still skip", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 3, Name: "research architecture design planning discovery", Description: "Internal planning work only"},
		}
		survey := codexSurveyContext{
			Languages:  []string{"go"},
			Frameworks: []string{"cobra"},
		}

		proposal := computePhaseResearchProposal("balanced", false, survey, candidates, nil)

		if len(proposal.Phases) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(proposal.Phases))
		}
		rec := proposal.Phases[0]
		if rec.Recommend != "skip" {
			t.Fatalf("expected recommend=skip (keywords must not drive this), got %q", rec.Recommend)
		}
		if len(rec.Hints.ExternalTech) != 0 {
			t.Fatalf("expected no external tech signals, got %v", rec.Hints.ExternalTech)
		}
	})

	t.Run("every candidate gets a non-empty recommend and reason, count matches candidates", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 1, Name: "Alpha", Description: "External GraphQL API integration"},
			{ID: 2, Name: "Beta", Description: "Internal cleanup"},
			{ID: 3, Name: "Gamma", Description: "Migrate to a new vendor SDK"},
		}
		survey := codexSurveyContext{Languages: []string{"go"}}

		proposal := computePhaseResearchProposal("balanced", false, survey, candidates, nil)

		if len(proposal.Phases) != len(candidates) {
			t.Fatalf("expected %d recommendations, got %d", len(candidates), len(proposal.Phases))
		}
		for _, rec := range proposal.Phases {
			if rec.Recommend == "" {
				t.Fatalf("phase %d has empty Recommend", rec.PhaseID)
			}
			if rec.Reason == "" {
				t.Fatalf("phase %d has empty Reason", rec.PhaseID)
			}
		}
	})

	t.Run("empty phase mode produces empty PhaseMode hint without panicking", func(t *testing.T) {
		candidates := []phaseResearchCandidate{
			{ID: 5, Name: "Some phase", Description: "No external tech here"},
		}
		phases := []colony.Phase{
			{ID: 5, Name: "Some phase", Mode: ""},
		}
		survey := codexSurveyContext{Languages: []string{"go"}}

		var proposal phaseResearchProposal
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("computePhaseResearchProposal panicked: %v", r)
				}
			}()
			proposal = computePhaseResearchProposal("balanced", false, survey, candidates, phases)
		}()

		if len(proposal.Phases) != 1 {
			t.Fatalf("expected 1 recommendation, got %d", len(proposal.Phases))
		}
		if proposal.Phases[0].Hints.PhaseMode != "" {
			t.Fatalf("expected empty PhaseMode hint for invalid mode, got %q", proposal.Phases[0].Hints.PhaseMode)
		}
	})
}
