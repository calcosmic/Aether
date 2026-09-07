package cmd

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanCandidateReviewExposesExactEvidenceAndBindingsWithoutWrites(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	before := planCandidateTestSnapshot(t, root)

	review, err := reviewPlanCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(review.Candidate.Proposal, candidate.Proposal) {
		t.Fatalf("review proposal diverged from persisted candidate:\nreview=%+v\ncandidate=%+v", review.Candidate.Proposal, candidate.Proposal)
	}
	if len(review.Scores) != len(colony.PlanningDimensions()) || review.ActualConfidence == 0 || review.TargetConfidence != 70 {
		t.Fatalf("review confidence = target:%d actual:%d scores:%+v, want target 70 and all five scores", review.TargetConfidence, review.ActualConfidence, review.Scores)
	}
	if !reflect.DeepEqual(review.StopDecision, candidate.StopDecision) || review.EvidenceThatWouldChange != candidate.EvidenceThatWouldChange || review.EvidenceThatWouldChange == "" {
		t.Fatalf("review stop diagnostics diverged: %+v", review)
	}
	if len(review.ResidualGaps) != len(colony.PlanningDimensions()) {
		t.Fatalf("review residual gaps = %d, want %d", len(review.ResidualGaps), len(colony.PlanningDimensions()))
	}
	for _, gap := range review.ResidualGaps {
		if gap.Materiality != colony.PlanningGapNonMaterial || strings.TrimSpace(gap.EvidenceThatWouldChange) == "" {
			t.Fatalf("review residual gap is not non-material and causal: %+v", gap)
		}
	}
	if !reflect.DeepEqual(review.SemanticDelta, candidate.SemanticDelta) {
		t.Fatalf("review semantic/authority delta diverged: %+v", review.SemanticDelta)
	}
	if !reflect.DeepEqual(review.Recommendation, candidate.Recommendation) {
		t.Fatalf("review invented or changed Queen recommendation:\nreview=%+v\ncandidate=%+v", review.Recommendation, candidate.Recommendation)
	}
	if review.Recommendation.Rationale == "" || len(review.Recommendation.EvidenceIDs) == 0 || review.Recommendation.Producer != colony.PlanRecommendationProducerQueen || review.Recommendation.ProducerID == "" {
		t.Fatalf("review recommendation is incomplete: %+v", review.Recommendation)
	}
	if !reflect.DeepEqual(review.Timeline, candidate.Timeline) || len(review.Iterations) != len(candidate.Timeline.CardIDs) {
		t.Fatalf("review timeline = %+v cards=%d, want candidate binding %+v", review.Timeline, len(review.Iterations), candidate.Timeline)
	}
	wantInputs := planCandidateAcceptanceRequest{
		CandidateID: candidate.ID, SpecificationRevisionID: candidate.SpecificationRevisionID,
		SpecificationRevisionHash: candidate.SpecificationRevisionHash, BasePlanRevisionID: candidate.BasePlanRevisionID,
		TimelineDigest: candidate.Timeline.TimelineDigest, ProposalHash: candidate.ProposalHash,
		AcceptanceToken: planCandidateAcceptanceToken(candidate),
	}
	if !reflect.DeepEqual(review.Acceptance, wantInputs) || !strings.Contains(review.AcceptanceCommand, candidate.ID) || !strings.Contains(review.AcceptanceCommand, wantInputs.AcceptanceToken) {
		t.Fatalf("review acceptance frontier = %+v command=%q, want %+v", review.Acceptance, review.AcceptanceCommand, wantInputs)
	}

	planCandidateTestAssertSnapshot(t, root, before)
}

func TestPlanCandidateDetailsReturnsOneExactCardAndRejectsOutOfRange(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	before := planCandidateTestSnapshot(t, root)

	detail, err := reviewPlanCandidateIteration(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if detail.CandidateID != candidate.ID || detail.Card.Iteration != 1 || detail.Card.ID != candidate.Timeline.CardIDs[0] || detail.Card.EvidenceThatWouldChange == "" {
		t.Fatalf("iteration detail = %+v, want exact first candidate card", detail)
	}
	if detail.TimelineDigest != candidate.Timeline.TimelineDigest || detail.Card.ContentHash != candidate.Timeline.LastCardHash {
		t.Fatalf("iteration detail is not timeline-bound: %+v", detail)
	}
	if _, err := reviewPlanCandidateIteration(root, 2); err == nil || !strings.Contains(err.Error(), "iteration 2") {
		t.Fatalf("out-of-range detail error = %v, want exact ordinal refusal", err)
	}
	if _, err := reviewPlanCandidateIteration(root, 0); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("zero detail error = %v, want positive ordinal refusal", err)
	}

	planCandidateTestAssertSnapshot(t, root, before)
}

func TestPlanCandidateInputsRejectPartialAmbiguousAndDeprecatedBeforeStateAccess(t *testing.T) {
	complete := planCandidateCommandInputs{
		AcceptCandidate: "plan-candidate-exact", SpecificationRevisionID: "spec-revision-exact",
		SpecificationRevisionHash: strings.Repeat("a", 64), BasePlanRevisionID: "plan-r1-exact",
		TimelineDigest: strings.Repeat("b", 64), ProposalHash: strings.Repeat("c", 64),
		AcceptanceToken: "accept-plan-candidate-exact",
	}
	tests := []struct {
		name   string
		inputs planCandidateCommandInputs
		want   string
	}{
		{name: "deprecated bare accept", inputs: planCandidateCommandInputs{DeprecatedAccept: true}, want: "--accept no longer"},
		{name: "partial exact acceptance", inputs: planCandidateCommandInputs{AcceptCandidate: complete.AcceptCandidate}, want: "requires --spec-revision"},
		{name: "review mixed with acceptance", inputs: func() planCandidateCommandInputs { value := complete; value.Candidate = true; return value }(), want: "cannot combine"},
		{name: "details without ordinal", inputs: planCandidateCommandInputs{Details: true}, want: "--details requires --show-iteration"},
		{name: "ordinal without details", inputs: planCandidateCommandInputs{ShowIteration: 1, ShowIterationSet: true}, want: "requires --details"},
		{name: "acceptance field without operation", inputs: planCandidateCommandInputs{ProposalHash: complete.ProposalHash}, want: "requires --accept-candidate"},
	}

	previousStore := store
	store = nil
	t.Cleanup(func() { store = previousStore })
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := runPlanCandidateCommand("", test.inputs); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("input error = %v, want %q before state access", err, test.want)
			}
		})
	}
	if _, err := resolvePlanCandidateOperation(complete); err != nil {
		t.Fatalf("complete exact acceptance inputs rejected: %v", err)
	}
}

func planCandidateTestPending(t *testing.T) (string, colony.PlanCandidate) {
	t.Helper()
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.Candidate == nil {
		t.Fatal("fixture did not produce a pending candidate")
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(manifest.RunID))))
	if err != nil {
		t.Fatal(err)
	}
	var persisted colony.PlanCandidate
	if err := json.Unmarshal(content, &persisted); err != nil {
		t.Fatal(err)
	}
	return root, persisted
}

func planCandidateTestSnapshot(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	for _, relRoot := range []string{filepath.Join(".aether", "data"), filepath.Join(".aether", "SPEC.md")} {
		start := filepath.Join(root, relRoot)
		info, err := os.Lstat(start)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			content, readErr := os.ReadFile(start)
			if readErr != nil {
				t.Fatal(readErr)
			}
			result[relRoot] = content
			continue
		}
		err = filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			result[rel] = content
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func planCandidateTestAssertSnapshot(t *testing.T, root string, before map[string][]byte) {
	t.Helper()
	after := planCandidateTestSnapshot(t, root)
	if !reflect.DeepEqual(before, after) {
		for path, old := range before {
			if current, ok := after[path]; !ok || !bytes.Equal(old, current) {
				t.Fatalf("read-only candidate operation changed %s", path)
			}
		}
		t.Fatalf("read-only candidate operation changed artifact set")
	}
}
