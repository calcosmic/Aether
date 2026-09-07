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

func TestPlanningConfidenceStopTargetPrecedesCapAndMaterialOverride(t *testing.T) {
	t.Parallel()

	history := planningConfidenceStopHistoryFixture([]int{89, 90}, []string{"target-gap", "target-gap"})
	planningConfidenceSetMaterialGap(&history[len(history)-1])
	result, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
		Target: 90, PassCap: 2, History: history,
	})
	if err != nil {
		t.Fatalf("evaluatePlanningStopPolicy returned error: %v", err)
	}
	if result.Decision.Reason != colony.PlanningStopTargetMet || result.Trigger != colony.PlanningStopTargetMet {
		t.Fatalf("reason/trigger = %q/%q, want target_met", result.Decision.Reason, result.Trigger)
	}
	assertPlanningConfidenceDecisionExplained(t, result, history[len(history)-1])
}

func TestPlanningConfidenceStopPassCapUsesConfiguredPresetExactly(t *testing.T) {
	t.Parallel()

	for _, cap := range []int{4, 12} {
		cap := cap
		t.Run(string(rune('0'+cap/10))+string(rune('0'+cap%10))+" passes", func(t *testing.T) {
			t.Parallel()
			overalls := make([]int, cap)
			gaps := make([]string, cap)
			for i := range overalls {
				overalls[i] = 40 + i*2
				gaps[i] = "cap-gap-" + string(rune('a'+i))
			}
			history := planningConfidenceStopHistoryFixture(overalls, gaps)
			beforeCap, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
				Target: 99, PassCap: cap, History: history[:cap-1],
			})
			if err != nil {
				t.Fatalf("before cap: %v", err)
			}
			if beforeCap.Decision.Reason != colony.PlanningStopContinue {
				t.Fatalf("pass %d reason = %q, want continue", cap-1, beforeCap.Decision.Reason)
			}
			assertPlanningConfidenceDecisionExplained(t, beforeCap, history[cap-2])

			atCap, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
				Target: 99, PassCap: cap, History: history,
			})
			if err != nil {
				t.Fatalf("at cap: %v", err)
			}
			if atCap.Decision.Reason != colony.PlanningStopPassCap || atCap.Trigger != colony.PlanningStopPassCap {
				t.Fatalf("pass %d reason/trigger = %q/%q, want pass_cap", cap, atCap.Decision.Reason, atCap.Trigger)
			}
			assertPlanningConfidenceDecisionExplained(t, atCap, history[len(history)-1])
		})
	}
}

func TestPlanningConfidenceDiminishingRequiresTwoGroundedSubTwoMovements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		overalls []int
		edit     func([]planningConfidencePass)
		want     colony.PlanningStopReason
	}{
		{
			name:     "two one-point grounded movements",
			overalls: []int{70, 71, 72},
			want:     colony.PlanningStopDiminishingReturns,
		},
		{
			name:     "two-point movement is not below two",
			overalls: []int{70, 72, 73},
			want:     colony.PlanningStopContinue,
		},
		{
			name:     "only one completed movement",
			overalls: []int{70, 71},
			want:     colony.PlanningStopContinue,
		},
		{
			name:     "latest pass is not grounded",
			overalls: []int{70, 71, 72},
			edit: func(history []planningConfidencePass) {
				planningConfidenceSetPassGrounded(&history[2], false)
			},
			want: colony.PlanningStopContinue,
		},
		{
			name:     "material semantic change breaks convergence",
			overalls: []int{70, 71, 72},
			edit: func(history []planningConfidencePass) {
				planningConfidenceSetSemanticChange(&history[1])
			},
			want: colony.PlanningStopContinue,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gaps := make([]string, len(test.overalls))
			for i := range gaps {
				gaps[i] = "diminishing-gap-" + string(rune('a'+i))
			}
			history := planningConfidenceStopHistoryFixture(test.overalls, gaps)
			if test.edit != nil {
				test.edit(history)
			}
			result, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
				Target: 90, PassCap: 12, History: history,
			})
			if err != nil {
				t.Fatalf("evaluatePlanningStopPolicy returned error: %v", err)
			}
			if result.Decision.Reason != test.want {
				t.Fatalf("reason = %q, want %q; diagnostics=%+v", result.Decision.Reason, test.want, result.Diagnostics)
			}
			assertPlanningConfidenceDecisionExplained(t, result, history[len(history)-1])
		})
	}
}

func TestPlanningConfidenceDiminishingDoesNotConfuseAuthorityImpactWithSemanticChange(t *testing.T) {
	t.Parallel()

	history := planningConfidenceStopHistoryFixture([]int{70, 71, 72}, []string{"gap-a", "gap-b", "gap-c"})
	impactHash := planningConfidenceTestDigest("authority-impact-only")
	history[1].SemanticDelta.AuthorityImpacts = []colony.PlanningAuthorityImpact{{
		ID: "authority-impact-" + impactHash[:12], ContentHash: impactHash,
		Kind: colony.PlanningAuthorityOwnerDecision, SourceID: "decision-200",
		AffectedSemanticIDs: []string{"task-200"}, Rationale: "Authority remains separately visible",
	}}
	result, err := evaluatePlanningStopPolicy(planningStopPolicyInput{Target: 90, PassCap: 12, History: history})
	if err != nil {
		t.Fatalf("evaluatePlanningStopPolicy returned error: %v", err)
	}
	if result.Decision.Reason != colony.PlanningStopDiminishingReturns {
		t.Fatalf("reason = %q, want diminishing_returns", result.Decision.Reason)
	}
	if result.Diagnostics[1].MaterialSemanticChange {
		t.Fatal("authority impact was incorrectly folded into semantic plan movement")
	}
}

func TestPlanningConfidenceStallRequiresTwoUnimprovedRepeats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		overalls []int
		gaps     []string
		want     colony.PlanningStopReason
	}{
		{
			name: "one repeat is not a stall", overalls: []int{60, 60},
			gaps: []string{"same-gap", "same-gap"}, want: colony.PlanningStopContinue,
		},
		{
			name: "two repeats with no improvement stall", overalls: []int{60, 60, 60},
			gaps: []string{"same-gap", "same-gap", "same-gap"}, want: colony.PlanningStopStalledGap,
		},
		{
			name: "evidence-backed improvement breaks stall", overalls: []int{60, 60, 61},
			gaps: []string{"same-gap", "same-gap", "same-gap"}, want: colony.PlanningStopDiminishingReturns,
		},
		{
			name: "nonconsecutive repeat does not stall", overalls: []int{60, 60, 60},
			gaps: []string{"same-gap", "other-gap", "same-gap"}, want: colony.PlanningStopDiminishingReturns,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			history := planningConfidenceStopHistoryFixture(test.overalls, test.gaps)
			result, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
				Target: 90, PassCap: 12, History: history,
			})
			if err != nil {
				t.Fatalf("evaluatePlanningStopPolicy returned error: %v", err)
			}
			if result.Decision.Reason != test.want {
				t.Fatalf("reason = %q, want %q; diagnostics=%+v", result.Decision.Reason, test.want, result.Diagnostics)
			}
			assertPlanningConfidenceDecisionExplained(t, result, history[len(history)-1])
		})
	}
}

func TestPlanningConfidenceStopMaterialGapConvertsBelowTargetStopsToOwnerDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		history []planningConfidencePass
		cap     int
		trigger colony.PlanningStopReason
	}{
		{
			name:    "diminishing returns",
			history: planningConfidenceStopHistoryFixture([]int{70, 71, 72}, []string{"gap-a", "gap-b", "gap-c"}),
			cap:     12, trigger: colony.PlanningStopDiminishingReturns,
		},
		{
			name:    "stalled gap",
			history: planningConfidenceStopHistoryFixture([]int{60, 60, 60}, []string{"same-gap", "same-gap", "same-gap"}),
			cap:     12, trigger: colony.PlanningStopStalledGap,
		},
		{
			name:    "pass cap",
			history: planningConfidenceStopHistoryFixture([]int{60, 62, 64, 66}, []string{"gap-a", "gap-b", "gap-c", "gap-d"}),
			cap:     4, trigger: colony.PlanningStopPassCap,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			planningConfidenceSetMaterialGap(&test.history[len(test.history)-1])
			result, err := evaluatePlanningStopPolicy(planningStopPolicyInput{
				Target: 90, PassCap: test.cap, History: test.history,
			})
			if err != nil {
				t.Fatalf("evaluatePlanningStopPolicy returned error: %v", err)
			}
			if result.Decision.Reason != colony.PlanningStopOwnerDecision || result.Trigger != test.trigger {
				t.Fatalf("reason/trigger = %q/%q, want owner_decision/%q", result.Decision.Reason, result.Trigger, test.trigger)
			}
			assertPlanningConfidenceDecisionExplained(t, result, test.history[len(test.history)-1])
		})
	}
}

func TestPlanningConfidenceStopReplayIsDeterministic(t *testing.T) {
	t.Parallel()

	history := planningConfidenceStopHistoryFixture([]int{70, 71, 72}, []string{"gap-a", "gap-b", "gap-c"})
	input := planningStopPolicyInput{Target: 90, PassCap: 12, History: history}
	first, err := evaluatePlanningStopPolicy(input)
	if err != nil {
		t.Fatalf("first evaluation: %v", err)
	}
	second, err := evaluatePlanningStopPolicy(input)
	if err != nil {
		t.Fatalf("second evaluation: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical history changed stop result:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.InputDigest == "" || first.InputDigest != second.InputDigest {
		t.Fatalf("input digest = %q/%q, want stable nonempty digest", first.InputDigest, second.InputDigest)
	}
}

func TestPlanningConfidenceStopRejectsInvalidDiagnosticInput(t *testing.T) {
	t.Parallel()

	base := planningConfidenceStopHistoryFixture([]int{70, 72}, []string{"gap-a", "gap-b"})
	tests := []struct {
		name  string
		input planningStopPolicyInput
		want  string
	}{
		{name: "invalid target", input: planningStopPolicyInput{Target: 101, PassCap: 4, History: base}, want: "target"},
		{name: "invalid cap", input: planningStopPolicyInput{Target: 90, PassCap: 0, History: base}, want: "pass cap"},
		{name: "missing history", input: planningStopPolicyInput{Target: 90, PassCap: 4}, want: "history"},
	}
	wrongOrdinal := planningConfidenceStopHistoryFixture([]int{70, 72}, []string{"gap-a", "gap-b"})
	wrongOrdinal[1].Iteration = 3
	tests = append(tests, struct {
		name  string
		input planningStopPolicyInput
		want  string
	}{name: "skipped pass ordinal", input: planningStopPolicyInput{Target: 90, PassCap: 4, History: wrongOrdinal}, want: "iteration"})
	forgedOverall := planningConfidenceStopHistoryFixture([]int{70, 72}, []string{"gap-a", "gap-b"})
	forgedOverall[1].Evaluation.Scores.Overall++
	tests = append(tests, struct {
		name  string
		input planningStopPolicyInput
		want  string
	}{name: "forged overall", input: planningStopPolicyInput{Target: 90, PassCap: 4, History: forgedOverall}, want: "Go-derived overall"})

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := evaluatePlanningStopPolicy(test.input)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func assertPlanningConfidenceDecisionExplained(t *testing.T, result planningStopPolicyEvaluation, current planningConfidencePass) {
	t.Helper()
	if err := result.Decision.Validate(); err != nil {
		t.Fatalf("decision is invalid: %v", err)
	}
	if err := validatePlanningStopShape("decision", result.Decision); err != nil {
		t.Fatalf("decision shape is invalid: %v", err)
	}
	if result.Decision.SelectedGapID != current.Evaluation.WeakestGap.ID {
		t.Fatalf("selected gap = %q, want %q", result.Decision.SelectedGapID, current.Evaluation.WeakestGap.ID)
	}
	if result.Decision.EvidenceThatWouldChange != current.Evaluation.WeakestGap.EvidenceThatWouldChange {
		t.Fatalf("evidence_that_would_change = %q, want selected gap evidence %q", result.Decision.EvidenceThatWouldChange, current.Evaluation.WeakestGap.EvidenceThatWouldChange)
	}
	if result.Decision.Rationale == "" || len(result.Decision.ResidualGapIDs) != len(current.Evaluation.RankedGaps) {
		t.Fatalf("decision omitted rationale or residual gaps: %+v", result.Decision)
	}
}

func planningConfidenceStopHistoryFixture(overalls []int, selectedGapLabels []string) []planningConfidencePass {
	if len(overalls) != len(selectedGapLabels) {
		panic("planning confidence stop fixture requires one gap label per pass")
	}
	history := make([]planningConfidencePass, 0, len(overalls))
	previous := 0
	for index, overall := range overalls {
		if index == 0 {
			previous = overall
		}
		pass := planningConfidencePass{
			Iteration: index + 1,
			Evaluation: planningConfidenceEvaluation{Scores: planningConfidenceScores{
				Knowledge: overall, Requirements: overall, Risks: overall, Dependencies: overall, Effort: overall, Overall: overall,
			}},
			SemanticDelta: planningConfidenceStopDeltaFixture(index+1, false),
		}
		for dimensionIndex, dimension := range colony.PlanningDimensions() {
			gapLabel := selectedGapLabels[index] + "-" + string(dimension)
			severity := 1
			if dimension == colony.PlanningDimensionKnowledge {
				gapLabel = selectedGapLabels[index]
				severity = 10
			}
			gapHash := planningConfidenceTestDigest("stop-gap-" + gapLabel)
			assessmentHash := planningConfidenceTestDigest("stop-assessment-" + string(rune('0'+index)) + "-" + string(dimension))
			evidenceID := "evidence-pass-" + string(rune('1'+index)) + "-" + string(dimension)
			assessment := colony.PlanningDimensionAssessment{
				SchemaVersion: colony.PlanningSchemaVersion,
				ID:            "assessment-" + assessmentHash[:12], ContentHash: assessmentHash,
				Dimension: dimension, Before: previous, After: overall,
				FreshEvidenceIDs: []string{evidenceID},
				RemainingGap: colony.PlanningGap{
					SchemaVersion: colony.PlanningSchemaVersion,
					ID:            "planning-gap-" + gapHash[:12], ContentHash: gapHash,
					Dimension: dimension, Materiality: colony.PlanningGapNonMaterial, Severity: severity,
					Description: "Unresolved " + gapLabel, EvidenceIDs: []string{evidenceID},
					EvidenceThatWouldChange: "Obtain exact evidence for " + gapLabel,
				},
				Rationale:         "Route-Setter grounded pass " + string(rune('1'+index)),
				ProducerReceiptID: "route-receipt-" + string(rune('1'+index)),
			}
			if dimensionIndex == 0 {
				assessment.RemainingGap.Description = "Weakest gap " + selectedGapLabels[index]
			}
			pass.Evaluation.Assessments = append(pass.Evaluation.Assessments, assessment)
		}
		pass.Evaluation.RankedGaps = rankPlanningConfidenceGaps(pass.Evaluation.Assessments, pass.Evaluation.Scores)
		pass.Evaluation.WeakestGap = clonePlanningConfidenceGap(pass.Evaluation.RankedGaps[0])
		history = append(history, pass)
		previous = overall
	}
	return history
}

func planningConfidenceStopDeltaFixture(iteration int, material bool) colony.PlanningSemanticDelta {
	deltaHash := planningConfidenceTestDigest("stop-delta-" + string(rune('0'+iteration)))
	delta := colony.PlanningSemanticDelta{
		SchemaVersion: colony.PlanningSchemaVersion,
		ID:            "planning-delta-" + deltaHash[:12], ContentHash: deltaHash,
	}
	if material {
		changeHash := planningConfidenceTestDigest("stop-change-" + string(rune('0'+iteration)))
		delta.Tasks = []colony.PlanningSemanticChange{{
			SemanticID: "task-material-change", ContentHash: changeHash,
			Kind:        colony.PlanningSemanticChangeModified,
			BeforeHash:  planningConfidenceTestDigest("before-material-change"),
			AfterHash:   planningConfidenceTestDigest("after-material-change"),
			EvidenceIDs: []string{"evidence-material-change"},
		}}
	}
	return delta
}

func planningConfidenceSetMaterialGap(pass *planningConfidencePass) {
	for i := range pass.Evaluation.Assessments {
		if pass.Evaluation.Assessments[i].Dimension == colony.PlanningDimensionKnowledge {
			pass.Evaluation.Assessments[i].RemainingGap.Materiality = colony.PlanningGapMaterial
		}
	}
	pass.Evaluation.RankedGaps = rankPlanningConfidenceGaps(pass.Evaluation.Assessments, pass.Evaluation.Scores)
	pass.Evaluation.WeakestGap = clonePlanningConfidenceGap(pass.Evaluation.RankedGaps[0])
}

func planningConfidenceSetPassGrounded(pass *planningConfidencePass, grounded bool) {
	if grounded {
		return
	}
	for i := range pass.Evaluation.Assessments {
		pass.Evaluation.Assessments[i].FreshEvidenceIDs = nil
	}
}

func planningConfidenceSetSemanticChange(pass *planningConfidencePass) {
	pass.SemanticDelta = planningConfidenceStopDeltaFixture(pass.Iteration, true)
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
