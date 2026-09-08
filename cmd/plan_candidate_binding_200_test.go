package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanCandidateSemanticIntegrity200TwoPassCandidateIsCanonical(t *testing.T) {
	root, candidate := classicPhase200TwoPassCandidate(t)
	if len(candidate.Timeline.CardIDs) != 2 || len(candidate.DimensionAssessments) != len(colony.PlanningDimensions()) {
		t.Fatalf("candidate does not expose the stopped two-pass review: timeline=%+v assessments=%d", candidate.Timeline, len(candidate.DimensionAssessments))
	}
	if candidate.StopDecision.Reason != colony.PlanningStopPassCap || candidate.Status != colony.PlanCandidatePendingReview || candidate.Acceptance != nil {
		t.Fatalf("candidate lifecycle = %s/%s acceptance=%+v, want pass_cap/pending_review/nil", candidate.StopDecision.Reason, candidate.Status, candidate.Acceptance)
	}
	if len(candidate.SemanticDelta.AuthorityImpacts) == 0 || len(candidate.ResidualGaps) != len(colony.PlanningDimensions()) || strings.TrimSpace(candidate.EvidenceThatWouldChange) == "" {
		t.Fatalf("candidate review payload is incomplete: %+v", candidate)
	}
	for i, assessment := range candidate.DimensionAssessments {
		if len(assessment.FreshEvidenceIDs) == 0 || strings.TrimSpace(assessment.RemainingGap.EvidenceThatWouldChange) == "" {
			t.Fatalf("assessment[%d] is not evidence-grounded: %+v", i, assessment)
		}
	}
	wantProposalHash, err := canonicalPlanCandidateProposalHash(candidate.Proposal)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ProposalHash != wantProposalHash || candidate.Proposal.PlanHash != wantProposalHash {
		t.Fatalf("proposal hashes = %s/%s, want %s", candidate.ProposalHash, candidate.Proposal.PlanHash, wantProposalHash)
	}
	if err := validateStandalonePlanRevision(candidate.Proposal); err != nil {
		t.Fatalf("valid stripped proposal rejected: %v", err)
	}
	if err := validatePlanningRecordHashes(candidate); err != nil {
		t.Fatalf("valid candidate identities rejected: %v", err)
	}
	review, err := reviewPlanCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(review.Candidate, candidate) || review.Acceptance.AcceptanceToken != planCandidateAcceptanceToken(candidate) {
		t.Fatalf("review changed the canonical candidate or token: review=%+v", review)
	}

	_, repeated := classicPhase200TwoPassCandidate(t)
	if candidate.ContentHash != repeated.ContentHash || candidate.ID != repeated.ID || planCandidateAcceptanceToken(candidate) != planCandidateAcceptanceToken(repeated) {
		t.Fatalf("repeated construction diverged: first=%s/%s second=%s/%s", candidate.ID, candidate.ContentHash, repeated.ID, repeated.ContentHash)
	}
}

func TestPlanCandidateSemanticIntegrity200CopiedHashesRejectEveryReviewMutation(t *testing.T) {
	root, candidate := classicPhase200TwoPassCandidate(t)
	mutations := []struct {
		name   string
		mutate func(*colony.PlanCandidate)
	}{
		{name: "proposal phase", mutate: func(value *colony.PlanCandidate) { value.Proposal.Phases[0].Description += " tampered" }},
		{name: "proposal hash", mutate: func(value *colony.PlanCandidate) { value.ProposalHash = strings.Repeat("1", 64) }},
		{name: "proposal plan hash", mutate: func(value *colony.PlanCandidate) { value.Proposal.PlanHash = strings.Repeat("2", 64) }},
		{name: "base revision id", mutate: func(value *colony.PlanCandidate) { value.BasePlanRevisionID += "-stale" }},
		{name: "base revision hash", mutate: func(value *colony.PlanCandidate) { value.BasePlanRevisionHash = strings.Repeat("3", 64) }},
		{name: "spec revision id", mutate: func(value *colony.PlanCandidate) { value.SpecificationRevisionID += "-stale" }},
		{name: "spec revision hash", mutate: func(value *colony.PlanCandidate) { value.SpecificationRevisionHash = strings.Repeat("4", 64) }},
		{name: "timeline id", mutate: func(value *colony.PlanCandidate) { value.Timeline.ID += "-stale" }},
		{name: "timeline content hash", mutate: func(value *colony.PlanCandidate) { value.Timeline.ContentHash = strings.Repeat("5", 64) }},
		{name: "timeline run", mutate: func(value *colony.PlanCandidate) { value.Timeline.RunID += "-stale" }},
		{name: "timeline card binding", mutate: func(value *colony.PlanCandidate) { value.Timeline.CardIDs[0] += "-stale" }},
		{name: "timeline first card", mutate: func(value *colony.PlanCandidate) { value.Timeline.FirstCardHash = strings.Repeat("6", 64) }},
		{name: "timeline last card", mutate: func(value *colony.PlanCandidate) { value.Timeline.LastCardHash = strings.Repeat("7", 64) }},
		{name: "timeline digest", mutate: func(value *colony.PlanCandidate) { value.Timeline.TimelineDigest = strings.Repeat("8", 64) }},
		{name: "timeline path", mutate: func(value *colony.PlanCandidate) { value.Timeline.Path += "-stale" }},
		{name: "stop reason", mutate: func(value *colony.PlanCandidate) { value.StopDecision.Reason = colony.PlanningStopTargetMet }},
		{name: "stop selected gap", mutate: func(value *colony.PlanCandidate) { value.StopDecision.SelectedGapID += "-stale" }},
		{name: "stop residual gaps", mutate: func(value *colony.PlanCandidate) { value.StopDecision.ResidualGapIDs[0] += "-stale" }},
		{name: "stop evidence", mutate: func(value *colony.PlanCandidate) { value.StopDecision.EvidenceIDs[0] += "-stale" }},
		{name: "stop rationale", mutate: func(value *colony.PlanCandidate) { value.StopDecision.Rationale += " tampered" }},
		{name: "stop causal evidence", mutate: func(value *colony.PlanCandidate) { value.StopDecision.EvidenceThatWouldChange += " tampered" }},
		{name: "assessment score", mutate: func(value *colony.PlanCandidate) { value.DimensionAssessments[0].After++ }},
		{name: "assessment evidence", mutate: func(value *colony.PlanCandidate) { value.DimensionAssessments[1].FreshEvidenceIDs[0] += "-stale" }},
		{name: "assessment resolved gap", mutate: func(value *colony.PlanCandidate) {
			value.DimensionAssessments[2].ResolvedGapIDs = []string{"resolved-tampered"}
		}},
		{name: "assessment rationale", mutate: func(value *colony.PlanCandidate) { value.DimensionAssessments[3].Rationale += " tampered" }},
		{name: "assessment producer", mutate: func(value *colony.PlanCandidate) { value.DimensionAssessments[4].ProducerReceiptID += "-stale" }},
		{name: "assessment remaining gap", mutate: func(value *colony.PlanCandidate) {
			value.DimensionAssessments[0].RemainingGap.Description += " tampered"
		}},
		{name: "delta id", mutate: func(value *colony.PlanCandidate) { value.SemanticDelta.ID += "-stale" }},
		{name: "delta hash", mutate: func(value *colony.PlanCandidate) { value.SemanticDelta.ContentHash = strings.Repeat("9", 64) }},
		{name: "residual gap dimension", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[0].Dimension = colony.PlanningDimensionEffort }},
		{name: "residual gap materiality", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[1].Materiality = colony.PlanningGapMaterial }},
		{name: "residual gap severity", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[2].Severity++ }},
		{name: "residual gap description", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[3].Description += " tampered" }},
		{name: "residual gap evidence", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[4].EvidenceIDs[0] += "-stale" }},
		{name: "residual gap causal evidence", mutate: func(value *colony.PlanCandidate) { value.ResidualGaps[0].EvidenceThatWouldChange += " tampered" }},
		{name: "candidate causal evidence", mutate: func(value *colony.PlanCandidate) { value.EvidenceThatWouldChange += " tampered" }},
		{name: "recommendation disposition", mutate: func(value *colony.PlanCandidate) { value.Recommendation.Disposition = colony.PlanRecommendationAccept }},
		{name: "recommendation evidence", mutate: func(value *colony.PlanCandidate) { value.Recommendation.EvidenceIDs[0] += "-stale" }},
		{name: "recommendation rationale", mutate: func(value *colony.PlanCandidate) { value.Recommendation.Rationale += " tampered" }},
		{name: "recommendation producer id", mutate: func(value *colony.PlanCandidate) { value.Recommendation.ProducerID += "-stale" }},
		{name: "recommendation created", mutate: func(value *colony.PlanCandidate) {
			value.Recommendation.CreatedAt = value.Recommendation.CreatedAt.Add(time.Second)
		}},
		{name: "created at", mutate: func(value *colony.PlanCandidate) { value.CreatedAt = value.CreatedAt.Add(time.Second) }},
		{name: "expires at", mutate: func(value *colony.PlanCandidate) { value.ExpiresAt = value.ExpiresAt.Add(time.Second) }},
		{name: "proposal candidate backref", mutate: func(value *colony.PlanCandidate) { value.Proposal.CandidateID += "-stale" }},
		{name: "phase candidate backref", mutate: func(value *colony.PlanCandidate) { value.Proposal.Phases[0].CandidateID += "-stale" }},
		{name: "task candidate hash backref", mutate: func(value *colony.PlanCandidate) {
			value.Proposal.Phases[0].Tasks[0].CandidateContentHash = strings.Repeat("a", 64)
		}},
		{name: "recommendation candidate backref", mutate: func(value *colony.PlanCandidate) { value.Recommendation.CandidateID += "-stale" }},
	}
	sections := []struct {
		name   string
		values func(*colony.PlanCandidate) []colony.PlanningSemanticChange
	}{
		{name: "phases", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange { return value.SemanticDelta.Phases }},
		{name: "tasks", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange { return value.SemanticDelta.Tasks }},
		{name: "dependencies", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.Dependencies
		}},
		{name: "requirement links", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.RequirementLinks
		}},
		{name: "acceptance checks", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.AcceptanceChecks
		}},
		{name: "negative expectations", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.NegativeExpectations
		}},
		{name: "recovery expectations", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.RecoveryExpectations
		}},
		{name: "public paths", values: func(value *colony.PlanCandidate) []colony.PlanningSemanticChange {
			return value.SemanticDelta.PublicPaths
		}},
	}
	for _, section := range sections {
		section := section
		if len(section.values(&candidate)) == 0 {
			continue
		}
		mutations = append(mutations, struct {
			name   string
			mutate func(*colony.PlanCandidate)
		}{name: "semantic change " + section.name, mutate: func(value *colony.PlanCandidate) {
			changes := section.values(value)
			changes[0].SemanticID += "-stale"
		}})
	}
	for index := range candidate.SemanticDelta.AuthorityImpacts {
		index := index
		mutations = append(mutations, struct {
			name   string
			mutate func(*colony.PlanCandidate)
		}{name: "authority impact " + candidate.SemanticDelta.AuthorityImpacts[index].ID, mutate: func(value *colony.PlanCandidate) {
			value.SemanticDelta.AuthorityImpacts[index].Rationale += " tampered"
		}})
	}

	path := filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)))
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			mutated := planCandidateSemanticIntegrity200Clone(t, candidate)
			test.mutate(&mutated)
			if err := validatePlanningRecordHashes(mutated); err == nil {
				t.Fatal("copied enclosing hashes accepted a one-field review mutation")
			}
			planCandidateSemanticIntegrity200Write(t, path, mutated)
			before := planCandidateTestSnapshot(t, root)
			if _, err := reviewPlanCandidate(root); err == nil {
				t.Fatal("review loaded a candidate with copied enclosing hashes")
			}
			planCandidateTestAssertSnapshot(t, root, before)
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPlanCandidateSemanticIntegrity200NestedAdjacencyEmptyAndOrdering(t *testing.T) {
	change := colony.PlanningSemanticChange{
		SemanticID: "task:semantic-change", Kind: colony.PlanningSemanticChangeModified,
		BeforeHash: strings.Repeat("b", 64), AfterHash: strings.Repeat("c", 64), EvidenceIDs: []string{"evidence:b", "evidence:a"},
	}
	if err := colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionPhases, &change); err != nil {
		t.Fatal(err)
	}
	delta := colony.PlanningSemanticDelta{SchemaVersion: colony.PlanningSchemaVersion, Phases: []colony.PlanningSemanticChange{change}}
	if err := colony.AddressPlanningSemanticDelta(&delta); err != nil {
		t.Fatal(err)
	}
	phaseChangeHash, phaseDeltaHash := change.ContentHash, delta.ContentHash
	adjacent := delta
	adjacent.Phases = nil
	adjacent.Tasks = []colony.PlanningSemanticChange{change}
	if err := colony.AddressPlanningSemanticDelta(&adjacent); err == nil {
		t.Fatal("moving a phase change into the adjacent task section retained its old section identity")
	}
	if err := colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionTasks, &adjacent.Tasks[0]); err != nil {
		t.Fatal(err)
	}
	if err := colony.AddressPlanningSemanticDelta(&adjacent); err != nil {
		t.Fatal(err)
	}
	if adjacent.Tasks[0].ContentHash == phaseChangeHash || adjacent.ContentHash == phaseDeltaHash {
		t.Fatal("adjacent plan sections collapsed to one semantic or delta identity")
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "empty material delta", run: func() error {
			value := colony.PlanningSemanticDelta{SchemaVersion: colony.PlanningSchemaVersion}
			return colony.AddressPlanningSemanticDelta(&value)
		}},
		{name: "empty changed evidence", run: func() error {
			value := change
			value.ContentHash = ""
			value.EvidenceIDs = nil
			return colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionPhases, &value)
		}},
		{name: "missing before", run: func() error {
			value := change
			value.ContentHash = ""
			value.BeforeHash = ""
			return colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionPhases, &value)
		}},
		{name: "missing after", run: func() error {
			value := change
			value.ContentHash = ""
			value.AfterHash = ""
			return colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionPhases, &value)
		}},
		{name: "duplicate authority impact", run: func() error {
			impact := colony.PlanningAuthorityImpact{Kind: colony.PlanningAuthoritySpecApproval, SourceID: "approval:1", AffectedSemanticIDs: []string{"task:1"}, Rationale: "Approval changes authority, not semantic readiness."}
			if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
				return err
			}
			value := colony.PlanningSemanticDelta{SchemaVersion: colony.PlanningSchemaVersion, AuthorityImpacts: []colony.PlanningAuthorityImpact{impact, impact}}
			return colony.AddressPlanningSemanticDelta(&value)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("degenerate semantic input was addressed")
			}
		})
	}

	recommendationA := colony.QueenPlanRecommendation{
		SchemaVersion: colony.PlanningSchemaVersion, Disposition: colony.PlanRecommendationRevise,
		EvidenceIDs: []string{"evidence:b", "evidence:a"}, Rationale: "More evidence is required.",
		Producer: colony.PlanRecommendationProducerQueen, ProducerID: "go-queen/v1", CreatedAt: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
	}
	recommendationB := recommendationA
	recommendationB.EvidenceIDs = []string{"evidence:a", "evidence:b"}
	if err := colony.AddressQueenPlanRecommendation(&recommendationA); err != nil {
		t.Fatal(err)
	}
	if err := colony.AddressQueenPlanRecommendation(&recommendationB); err != nil {
		t.Fatal(err)
	}
	if recommendationA.ContentHash != recommendationB.ContentHash || recommendationA.ID != recommendationB.ID {
		t.Fatal("set-like evidence ordering changed recommendation identity")
	}

	_, candidate := classicPhase200TwoPassCandidate(t)
	ordered := planCandidateSemanticIntegrity200Clone(t, candidate).Proposal
	ordered.Phases[0].Tasks[0].DependsOn = []string{"1.2", "1.3"}
	forward, err := canonicalPlanCandidateProposalHash(ordered)
	if err != nil {
		t.Fatal(err)
	}
	ordered.Phases[0].Tasks[0].DependsOn = []string{"1.3", "1.2"}
	reversed, err := canonicalPlanCandidateProposalHash(ordered)
	if err != nil {
		t.Fatal(err)
	}
	if forward == reversed {
		t.Fatal("sequence-bearing proposal dependency order was normalized away")
	}
}

func TestPlanCandidateSemanticIntegrity200IdentityTokenAndTransition(t *testing.T) {
	root, candidate := classicPhase200TwoPassCandidate(t)
	originalID, originalHash, originalToken := candidate.ID, candidate.ContentHash, planCandidateAcceptanceToken(candidate)
	changed := planCandidateSemanticIntegrity200Clone(t, candidate)
	changed.CreatedAt = changed.CreatedAt.Add(time.Nanosecond)
	if err := addressPlanCandidateReviewPayload(&changed); err != nil {
		t.Fatal(err)
	}
	if changed.ID == originalID || changed.ContentHash == originalHash || planCandidateAcceptanceToken(changed) == originalToken {
		t.Fatal("re-addressed review mutation did not change candidate ID, content hash, and exact acceptance token")
	}

	before := planCandidateTestSnapshot(t, root)
	request := planCandidateTestAcceptanceRequest(candidate)
	request.AcceptanceToken += "-stale"
	if _, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Now().UTC()}); err == nil {
		t.Fatal("invalid exact token was accepted")
	}
	planCandidateTestAssertSnapshot(t, root, before)

	accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner", AcceptedAt: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Candidate.ID != originalID || accepted.Candidate.ContentHash != originalHash || accepted.Candidate.Acceptance == nil {
		t.Fatalf("mutable acceptance transition rewrote immutable candidate identity: %+v", accepted.Candidate)
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	facts := lifecycleFactsFromStateSnapshot(state, false, time.Now().UTC())
	bindings := loadPlanAuthorityVerifiedBindings(root, facts)
	if decision := validateAcceptedPlanAuthority(facts, bindings); !decision.Eligible {
		t.Fatalf("valid accepted candidate was refused by build/run authority: %+v", decision)
	}

	mutatedFacts := facts
	mutatedFacts.State.Value = planCandidateSemanticIntegrity200CloneState(t, facts.State.Value)
	mutatedBindings := bindings
	mutatedCandidate := planCandidateSemanticIntegrity200Clone(t, *bindings.Candidate)
	mutatedCandidate.Proposal.Phases[0].Description += " copied-hash mutation"
	mutatedBindings.Candidate = &mutatedCandidate
	for i := range mutatedFacts.State.Value.Plan.Candidates {
		if mutatedFacts.State.Value.Plan.Candidates[i].ID == mutatedCandidate.ID {
			mutatedFacts.State.Value.Plan.Candidates[i] = mutatedCandidate
		}
	}
	if decision := validateAcceptedPlanAuthority(mutatedFacts, mutatedBindings); decision.Eligible || decision.RefusalCode != planAuthorityRefusalCandidateInvalid {
		t.Fatalf("accepted build/run authority trusted a copied proposal/candidate hash: %+v", decision)
	}
}

func TestPlanCandidateSemanticIntegrity200ProposalBackrefsAndVocabulary(t *testing.T) {
	_, candidate := classicPhase200TwoPassCandidate(t)
	mutations := []struct {
		name   string
		mutate func(*colony.PlanRevision)
	}{
		{name: "proposal body", mutate: func(value *colony.PlanRevision) { value.Phases[0].Description += " tampered" }},
		{name: "proposal backref", mutate: func(value *colony.PlanRevision) { value.CandidateID += "-stale" }},
		{name: "phase backref", mutate: func(value *colony.PlanRevision) { value.Phases[0].CandidateContentHash = strings.Repeat("d", 64) }},
		{name: "task backref", mutate: func(value *colony.PlanRevision) { value.Phases[0].Tasks[0].CandidateID += "-stale" }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			proposal := planCandidateSemanticIntegrity200Clone(t, candidate).Proposal
			test.mutate(&proposal)
			if err := validateStandalonePlanRevision(proposal); err == nil {
				t.Fatal("standalone proposal accepted a copied hash or inconsistent derived back-reference")
			}
		})
	}
	if string(colony.PlanningSemanticChangePreserved) != "preserved" || strings.Contains(string(colony.PlanningSemanticChangePreserved), "unchanged") {
		t.Fatalf("no-change vocabulary drifted: %q", colony.PlanningSemanticChangePreserved)
	}
	if candidate.Timeline.CardIDs[0] == candidate.Timeline.CardIDs[1] || candidate.StopDecision.Reason != colony.PlanningStopPassCap {
		t.Fatalf("primary iteration/pass vocabulary lost its exact two-pass meaning: %+v", candidate)
	}
}

func planCandidateSemanticIntegrity200Clone(t *testing.T, candidate colony.PlanCandidate) colony.PlanCandidate {
	t.Helper()
	content, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	var cloned colony.PlanCandidate
	if err := json.Unmarshal(content, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func planCandidateSemanticIntegrity200CloneState(t *testing.T, state colony.ColonyState) colony.ColonyState {
	t.Helper()
	content, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var cloned colony.ColonyState
	if err := json.Unmarshal(content, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func planCandidateSemanticIntegrity200Write(t *testing.T, path string, candidate colony.PlanCandidate) {
	t.Helper()
	content, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read, content) {
		t.Fatal("candidate mutation fixture was not written byte-exactly")
	}
}
