package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestResearchProposalBatchRendersTickToApprove(t *testing.T) {
	proposal := phaseResearchProposal{
		Depth:  "balanced",
		Replan: false,
		Phases: []phaseResearchRecommendation{
			{PhaseID: 1, PhaseName: "Alpha", Recommend: "research", Reason: "new external tech (api) is absent from the territory survey"},
			{PhaseID: 2, PhaseName: "Beta", Recommend: "skip", Reason: "pure refactor -- the domain is already mapped by the territory survey"},
			{PhaseID: 3, PhaseName: "Gamma", Recommend: "skip", Reason: "no external technology signals found in the phase description -- the domain looks internal"},
		},
	}

	block := renderPhaseResearchProposalBlock(proposal)

	approveAllCount := strings.Count(block, "--approve-all")
	if approveAllCount != 1 {
		t.Fatalf("expected exactly one --approve-all occurrence, got %d in:\n%s", approveAllCount, block)
	}
	flipCount := strings.Count(block, "--flip")
	if flipCount != 3 {
		t.Fatalf("expected exactly 3 --flip occurrences, got %d in:\n%s", flipCount, block)
	}
	for _, want := range []string{"Phase 1: Alpha", "Phase 2: Beta", "Phase 3: Gamma", "Reason:"} {
		if !strings.Contains(block, want) {
			t.Fatalf("expected block to contain %q, got:\n%s", want, block)
		}
	}

	t.Run("empty proposal renders empty string", func(t *testing.T) {
		empty := renderPhaseResearchProposalBlock(phaseResearchProposal{})
		if empty != "" {
			t.Fatalf("expected empty string for zero-phase proposal, got %q", empty)
		}
	})
}

func TestResearchDecisionRecordsUseExistingStore(t *testing.T) {
	rec := phaseResearchRecommendation{
		PhaseID:   4,
		PhaseName: "Delta",
		Recommend: "research",
		Reason:    "new external tech (api) is absent from the territory survey",
	}

	t.Run("newPhaseResearchDecision returns existing PendingDecision type", func(t *testing.T) {
		decision := newPhaseResearchDecision(rec)
		if decision.Type != phaseResearchDecisionType {
			t.Fatalf("expected Type=%q, got %q", phaseResearchDecisionType, decision.Type)
		}
		if decision.Phase == nil || *decision.Phase != rec.PhaseID {
			t.Fatalf("expected Phase to point at %d, got %v", rec.PhaseID, decision.Phase)
		}
		if decision.Resolved {
			t.Fatalf("expected Resolved=false for a freshly created decision")
		}
		if decision.ID == "" || decision.CreatedAt == "" {
			t.Fatalf("expected non-empty ID and CreatedAt, got ID=%q CreatedAt=%q", decision.ID, decision.CreatedAt)
		}
	})

	t.Run("resolvePhaseResearchDecisions returns research for a research resolution", func(t *testing.T) {
		phaseID := 4
		file := PendingDecisionFile{
			Decisions: []PendingDecision{
				{Type: phaseResearchDecisionType, Phase: &phaseID, Resolved: true, Resolution: "approved: research phase 4"},
			},
		}
		result := resolvePhaseResearchDecisions(file)
		if result[4] != "research" {
			t.Fatalf("expected phase 4 resolved to research, got %q", result[4])
		}
	})

	t.Run("resolvePhaseResearchDecisions returns skip for a skip resolution", func(t *testing.T) {
		phaseID := 5
		file := PendingDecisionFile{
			Decisions: []PendingDecision{
				{Type: phaseResearchDecisionType, Phase: &phaseID, Resolved: true, Resolution: "approved: skip phase 5"},
			},
		}
		result := resolvePhaseResearchDecisions(file)
		if result[5] != "skip" {
			t.Fatalf("expected phase 5 resolved to skip, got %q", result[5])
		}
	})

	t.Run("resolvePhaseResearchDecisions ignores unresolved decisions", func(t *testing.T) {
		phaseID := 6
		file := PendingDecisionFile{
			Decisions: []PendingDecision{
				{Type: phaseResearchDecisionType, Phase: &phaseID, Resolved: false, Resolution: ""},
			},
		}
		result := resolvePhaseResearchDecisions(file)
		if _, ok := result[6]; ok {
			t.Fatalf("expected unresolved decision to be ignored, got %q", result[6])
		}
	})

	t.Run("resolvePhaseResearchDecisions ignores decisions with a different Type", func(t *testing.T) {
		phaseID := 7
		file := PendingDecisionFile{
			Decisions: []PendingDecision{
				{Type: "some-other-decision", Phase: &phaseID, Resolved: true, Resolution: "approved: research phase 7"},
			},
		}
		result := resolvePhaseResearchDecisions(file)
		if _, ok := result[7]; ok {
			t.Fatalf("expected non-research decision type to be ignored, got %q", result[7])
		}
	})

	t.Run("flipped decision resolution names user override and direction", func(t *testing.T) {
		resolution := phaseResearchDecisionResolution(rec, true, false)
		if !strings.Contains(resolution, "user overrode") {
			t.Fatalf("expected resolution to contain 'user overrode', got %q", resolution)
		}
		if !strings.Contains(resolution, "skip") {
			t.Fatalf("expected flipped resolution (from research) to name skip, got %q", resolution)
		}
		if !strings.Contains(resolution, "4") {
			t.Fatalf("expected resolution to name phase 4, got %q", resolution)
		}
	})

	t.Run("auto-accepted resolution is prefixed", func(t *testing.T) {
		resolution := phaseResearchDecisionResolution(rec, false, true)
		if !strings.HasPrefix(resolution, "auto-accepted (autopilot)") {
			t.Fatalf("expected auto-accepted prefix, got %q", resolution)
		}
	})
}

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
