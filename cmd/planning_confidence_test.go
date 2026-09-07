package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningConfidenceScoreComputesLockedWeightedOverall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scores map[colony.PlanningDimension]int
		want   int
	}{
		{
			name: "all dimensions at their upper boundary",
			scores: map[colony.PlanningDimension]int{
				colony.PlanningDimensionKnowledge: 100, colony.PlanningDimensionRequirements: 100,
				colony.PlanningDimensionRisks: 100, colony.PlanningDimensionDependencies: 100,
				colony.PlanningDimensionEffort: 100,
			},
			want: 100,
		},
		{
			name: "locked weights",
			scores: map[colony.PlanningDimension]int{
				colony.PlanningDimensionKnowledge: 100, colony.PlanningDimensionRequirements: 80,
				colony.PlanningDimensionRisks: 60, colony.PlanningDimensionDependencies: 40,
				colony.PlanningDimensionEffort: 20,
			},
			want: 66,
		},
		{
			name: "fraction below half rounds down",
			scores: map[colony.PlanningDimension]int{
				colony.PlanningDimensionKnowledge: 1,
			},
			want: 0,
		},
		{
			name: "half rounds up",
			scores: map[colony.PlanningDimension]int{
				colony.PlanningDimensionKnowledge: 2,
			},
			want: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proposal := planningConfidenceProposalFixture(t, 0)
			for i := range proposal.Assessments {
				proposal.Assessments[i].After = test.scores[proposal.Assessments[i].Dimension]
			}

			result, err := evaluatePlanningConfidence(proposal)
			if err != nil {
				t.Fatalf("evaluatePlanningConfidence returned error: %v", err)
			}
			if result.Scores.Overall != test.want {
				t.Fatalf("overall = %d, want %d", result.Scores.Overall, test.want)
			}
			for dimension, want := range test.scores {
				if got := result.Scores.Value(dimension); got != want {
					t.Errorf("%s = %d, want %d", dimension, got, want)
				}
			}
		})
	}
}

func TestPlanningConfidenceScoreRejectsIncompleteOrForgedProposals(t *testing.T) {
	t.Parallel()

	overall := 73
	tests := []struct {
		name string
		edit func(*planningConfidenceProposal)
		want string
	}{
		{
			name: "missing dimension",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments = proposal.Assessments[:4]
			},
			want: "exactly five",
		},
		{
			name: "duplicate dimension",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[4].Dimension = proposal.Assessments[0].Dimension
				proposal.Assessments[4].RemainingGap.Dimension = proposal.Assessments[0].Dimension
			},
			want: "duplicate",
		},
		{
			name: "mismatched before frontier",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].Before++
			},
			want: "persisted prior",
		},
		{
			name: "before below lower boundary",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].Before = -1
			},
			want: "between 0 and 100",
		},
		{
			name: "after above upper boundary",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].After = 101
			},
			want: "between 0 and 100",
		},
		{
			name: "missing rationale",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].Rationale = "  "
			},
			want: "rationale",
		},
		{
			name: "route setter supplied overall",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.SuppliedOverall = &overall
			},
			want: "supplied overall",
		},
		{
			name: "remaining gap omits evidence that would change",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].RemainingGap.EvidenceThatWouldChange = ""
			},
			want: "evidence_that_would_change",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proposal := planningConfidenceProposalFixture(t, 50)
			test.edit(&proposal)
			_, err := evaluatePlanningConfidence(proposal)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestPlanningConfidenceScoreRequiresFreshApplicableEvidenceForEveryChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		edit func(*planningConfidenceProposal)
		want string
	}{
		{
			name: "missing evidence",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].FreshEvidenceIDs = nil
			},
			want: "fresh applicable evidence",
		},
		{
			name: "repeated prose under the same evidence identity",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.EvidenceFrontier.CitedEvidenceIDs = []string{proposal.Assessments[0].FreshEvidenceIDs[0]}
			},
			want: "already cited",
		},
		{
			name: "unrelated evidence",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].FreshEvidenceIDs = append([]string{}, proposal.Assessments[1].FreshEvidenceIDs...)
			},
			want: "does not explicitly claim",
		},
		{
			name: "unknown evidence",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].FreshEvidenceIDs = []string{"evidence-not-in-catalogue"}
			},
			want: "not present in the evidence catalogue",
		},
		{
			name: "score inflation without evidence",
			edit: func(proposal *planningConfidenceProposal) {
				proposal.Assessments[0].After = 100
				proposal.Assessments[0].FreshEvidenceIDs = nil
			},
			want: "fresh applicable evidence",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proposal := planningConfidenceProposalFixture(t, 50)
			proposal.Assessments[0].After = 60
			test.edit(&proposal)
			_, err := evaluatePlanningConfidence(proposal)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestPlanningConfidenceScoreAllowsEvidenceBackedDecrease(t *testing.T) {
	t.Parallel()

	proposal := planningConfidenceProposalFixture(t, 50)
	proposal.Assessments[0].After = 40
	result, err := evaluatePlanningConfidence(proposal)
	if err != nil {
		t.Fatalf("evaluatePlanningConfidence returned error: %v", err)
	}
	if result.Scores.Knowledge != 40 || result.Scores.Overall != 48 {
		t.Fatalf("scores = %+v, want knowledge=40 overall=48", result.Scores)
	}
}

func TestPlanningConfidenceGapRanksMaterialBeforeNumericDeficit(t *testing.T) {
	t.Parallel()

	proposal := planningConfidenceProposalFixture(t, 50)
	for i := range proposal.Assessments {
		assessment := &proposal.Assessments[i]
		assessment.Before = assessment.After
		proposal.PriorScores.Set(assessment.Dimension, assessment.After)
		assessment.RemainingGap.Materiality = colony.PlanningGapNonMaterial
	}
	proposal.Assessments[0].After = 95
	proposal.Assessments[0].Before = 95
	proposal.PriorScores.Set(colony.PlanningDimensionKnowledge, 95)
	proposal.Assessments[0].RemainingGap.Materiality = colony.PlanningGapMaterial
	proposal.Assessments[4].After = 5
	proposal.Assessments[4].Before = 5
	proposal.PriorScores.Set(colony.PlanningDimensionEffort, 5)

	result, err := evaluatePlanningConfidence(proposal)
	if err != nil {
		t.Fatalf("evaluatePlanningConfidence returned error: %v", err)
	}
	if result.WeakestGap.ID != proposal.Assessments[0].RemainingGap.ID {
		t.Fatalf("weakest gap = %s, want material gap %s", result.WeakestGap.ID, proposal.Assessments[0].RemainingGap.ID)
	}
}

func TestPlanningConfidenceGapRanksDeficitSeverityThenStableID(t *testing.T) {
	t.Parallel()

	proposal := planningConfidenceProposalFixture(t, 50)
	for i := range proposal.Assessments {
		assessment := &proposal.Assessments[i]
		assessment.FreshEvidenceIDs = nil
		assessment.RemainingGap.Materiality = colony.PlanningGapMaterial
		assessment.RemainingGap.Severity = 1
	}
	proposal.Assessments[0].After = 70
	proposal.Assessments[0].Before = 70
	proposal.PriorScores.Set(colony.PlanningDimensionKnowledge, 70)
	proposal.Assessments[1].After = 40
	proposal.Assessments[1].Before = 40
	proposal.PriorScores.Set(colony.PlanningDimensionRequirements, 40)
	proposal.Assessments[2].After = 40
	proposal.Assessments[2].Before = 40
	proposal.PriorScores.Set(colony.PlanningDimensionRisks, 40)
	proposal.Assessments[2].RemainingGap.Severity = 7
	proposal.Assessments[3].After = 40
	proposal.Assessments[3].Before = 40
	proposal.PriorScores.Set(colony.PlanningDimensionDependencies, 40)
	proposal.Assessments[4].After = 80
	proposal.Assessments[4].Before = 80
	proposal.PriorScores.Set(colony.PlanningDimensionEffort, 80)
	proposal.Assessments[3].RemainingGap.Severity = 7
	setPlanningConfidenceTestGapID(&proposal.Assessments[2].RemainingGap, "gap-z")
	setPlanningConfidenceTestGapID(&proposal.Assessments[3].RemainingGap, "gap-a")

	result, err := evaluatePlanningConfidence(proposal)
	if err != nil {
		t.Fatalf("evaluatePlanningConfidence returned error: %v", err)
	}
	want := []string{
		proposal.Assessments[3].RemainingGap.ID,
		proposal.Assessments[2].RemainingGap.ID,
		proposal.Assessments[1].RemainingGap.ID,
		proposal.Assessments[0].RemainingGap.ID,
	}
	got := make([]string, 0, len(result.RankedGaps))
	for _, gap := range result.RankedGaps[:4] {
		got = append(got, gap.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked gap IDs = %v, want %v", got, want)
	}

	reversed := proposal
	reversed.Assessments = append([]colony.PlanningDimensionAssessment{}, proposal.Assessments...)
	for left, right := 0, len(reversed.Assessments)-1; left < right; left, right = left+1, right-1 {
		reversed.Assessments[left], reversed.Assessments[right] = reversed.Assessments[right], reversed.Assessments[left]
	}
	replayed, err := evaluatePlanningConfidence(reversed)
	if err != nil {
		t.Fatalf("replayed evaluation returned error: %v", err)
	}
	if !reflect.DeepEqual(result.RankedGaps, replayed.RankedGaps) || !reflect.DeepEqual(result.Assessments, replayed.Assessments) {
		t.Fatal("assessment input order changed deterministic confidence output")
	}
}

func planningConfidenceProposalFixture(t *testing.T, prior int) planningConfidenceProposal {
	t.Helper()

	scope := planningEvidenceScope{
		GoalID: "goal-200", SessionID: "session-200",
		SpecificationRevisionID: "spec-revision-200", PlanRevisionID: "plan-revision-199",
	}
	proposal := planningConfidenceProposal{
		PriorScores: planningConfidenceScores{
			Knowledge: prior, Requirements: prior, Risks: prior, Dependencies: prior, Effort: prior,
		},
		EvidenceFrontier: planningEvidenceFrontier{
			Scope: scope, CurrentSourceHashes: map[string]string{}, SourceStates: map[string]planningEvidenceSourceState{},
		},
	}
	for _, dimension := range colony.PlanningDimensions() {
		record, err := normalizePlanningEvidence(planningEvidenceSource{
			Kind: colony.PlanningEvidenceResearch, Origin: "fixture:" + string(dimension),
			Content: []byte("Fresh grounded evidence for " + string(dimension)), Scope: scope,
			SourceRevision: "source-" + string(dimension), ObservedAt: time.Date(2026, time.September, 7, 13, 0, 0, 0, time.UTC),
			ApplicableDimensions: []colony.PlanningDimension{dimension}, State: planningEvidenceSourceCurrent,
		})
		if err != nil {
			t.Fatalf("normalize %s evidence: %v", dimension, err)
		}
		proposal.EvidenceCatalogue = append(proposal.EvidenceCatalogue, record.Reference)
		proposal.EvidenceFrontier.CurrentSourceHashes[planningEvidenceSourceKey(record.Reference)] = record.Reference.ContentHash
		proposal.EvidenceFrontier.SourceStates[planningEvidenceSourceKey(record.Reference)] = planningEvidenceSourceCurrent

		gapHash := planningConfidenceTestDigest("gap-" + string(dimension))
		assessmentHash := planningConfidenceTestDigest("assessment-" + string(dimension))
		proposal.Assessments = append(proposal.Assessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion,
			ID:            "assessment-" + assessmentHash[:12], ContentHash: assessmentHash,
			Dimension: dimension, Before: prior, After: prior,
			FreshEvidenceIDs: []string{record.Reference.ID},
			RemainingGap: colony.PlanningGap{
				SchemaVersion: colony.PlanningSchemaVersion,
				ID:            "planning-gap-" + gapHash[:12], ContentHash: gapHash,
				Dimension: dimension, Materiality: colony.PlanningGapNonMaterial, Severity: 1,
				Description: "Remaining " + string(dimension) + " uncertainty",
				EvidenceIDs: []string{record.Reference.ID}, EvidenceThatWouldChange: "Fresh proof resolving " + string(dimension),
			},
			Rationale:         "Fresh evidence supports the proposed " + string(dimension) + " readiness",
			ProducerReceiptID: "route-setter-receipt-200",
		})
	}
	return proposal
}

func setPlanningConfidenceTestGapID(gap *colony.PlanningGap, label string) {
	digest := planningConfidenceTestDigest(label)
	gap.ContentHash = digest
	gap.ID = "planning-gap-" + digest[:12]
}

func planningConfidenceTestDigest(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}
