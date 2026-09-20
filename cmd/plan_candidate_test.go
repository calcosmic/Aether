package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestPlanCandidateAcceptActivatesInitialRevisionAtomically(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	beforeTimeline, err := verifiedPlanCandidateTimeline(root, candidate)
	if err != nil {
		t.Fatal(err)
	}
	acceptedAt := time.Date(2026, time.September, 7, 20, 0, 0, 0, time.UTC)
	result, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner",
		AcceptedAt: acceptedAt,
	})
	if err != nil {
		t.Fatalf("accept exact candidate: %v", err)
	}
	if result.Replayed {
		t.Fatal("first acceptance was reported as a replay")
	}
	if result.Receipt.CandidateID != candidate.ID || result.Receipt.CandidateContentHash != candidate.ContentHash ||
		result.Receipt.SpecificationRevisionID != candidate.SpecificationRevisionID || result.Receipt.SpecificationRevisionHash != candidate.SpecificationRevisionHash ||
		result.Receipt.BasePlanRevisionID != candidate.BasePlanRevisionID || result.Receipt.BasePlanRevisionHash != candidate.BasePlanRevisionHash ||
		result.Receipt.TimelineID != candidate.Timeline.ID || result.Receipt.TimelineDigest != candidate.Timeline.TimelineDigest ||
		result.Receipt.ProposalHash != candidate.ProposalHash || result.Receipt.AcceptedBy != "owner" || !result.Receipt.AcceptedAt.Equal(acceptedAt) {
		t.Fatalf("acceptance receipt does not bind the exact frontier: %+v", result.Receipt)
	}
	if result.Revision.ID != candidate.Proposal.ID || result.Revision.PlanHash != candidate.ProposalHash ||
		result.Receipt.ActivatedPlanRevisionID != result.Revision.ID || result.Receipt.ActivatedPlanRevisionHash != result.Revision.PlanHash {
		t.Fatalf("activated revision diverged from candidate proposal: result=%+v", result)
	}

	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePlanningState(state); err != nil {
		t.Fatalf("accepted state is invalid: %v", err)
	}
	if state.State != colony.StateREADY || state.CurrentPhase != 1 || state.Plan.AcceptancePolicy != colony.PlanAcceptanceExplicitOwner || state.Plan.ActiveRevisionID != result.Revision.ID || state.Plan.PendingCandidateID != "" || len(state.Plan.Revisions) != 1 || len(state.Plan.Candidates) != 1 {
		t.Fatalf("accepted plan lineage = %+v, want one explicit-owner revision and candidate", state.Plan)
	}
	if state.Plan.Candidates[0].Status != colony.PlanCandidateAccepted || !reflect.DeepEqual(state.Plan.Candidates[0].Acceptance, &result.Receipt) || !reflect.DeepEqual(state.Plan.Phases, result.Revision.Phases) {
		t.Fatalf("accepted state did not atomically activate candidate: %+v", state.Plan)
	}
	stage, err := loadPlanningStageState(root, candidate.Timeline.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if stage.Stage != planningStageAccepted || stage.AcceptanceReceiptID != result.Receipt.ID || stage.AcceptanceReceiptHash != result.Receipt.ContentHash {
		t.Fatalf("accepted stage = %+v, want receipt-bound accepted", stage)
	}
	receiptBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(planningRouteAcceptanceRepositoryPath(candidate.Timeline.RunID))))
	if err != nil {
		t.Fatal(err)
	}
	var persistedReceipt colony.PlanAcceptanceReceipt
	if err := json.Unmarshal(receiptBytes, &persistedReceipt); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(persistedReceipt, result.Receipt) {
		t.Fatalf("persisted acceptance receipt = %+v, want %+v", persistedReceipt, result.Receipt)
	}
	afterTimeline, err := verifiedPlanCandidateTimeline(root, state.Plan.Candidates[0])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeTimeline, afterTimeline) {
		t.Fatal("acceptance changed the complete verified planning timeline")
	}
}

func TestPlanCandidateAcceptRejectsDivergentBindingsWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*planCandidateAcceptanceRequest)
		want   string
	}{
		{name: "candidate", mutate: func(request *planCandidateAcceptanceRequest) { request.CandidateID += "-stale" }, want: "candidate_id"},
		{name: "specification revision", mutate: func(request *planCandidateAcceptanceRequest) { request.SpecificationRevisionID += "-stale" }, want: "specification_revision_id"},
		{name: "specification hash", mutate: func(request *planCandidateAcceptanceRequest) {
			request.SpecificationRevisionHash = strings.Repeat("1", 64)
		}, want: "specification_revision_hash"},
		{name: "base revision", mutate: func(request *planCandidateAcceptanceRequest) { request.BasePlanRevisionID += "-stale" }, want: "base_plan_revision_id"},
		{name: "timeline", mutate: func(request *planCandidateAcceptanceRequest) { request.TimelineDigest = strings.Repeat("2", 64) }, want: "timeline_digest"},
		{name: "proposal", mutate: func(request *planCandidateAcceptanceRequest) { request.ProposalHash = strings.Repeat("3", 64) }, want: "proposal_hash"},
		{name: "token", mutate: func(request *planCandidateAcceptanceRequest) { request.AcceptanceToken += "-stale" }, want: "acceptance_token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, candidate := planCandidateTestPending(t)
			request := planCandidateTestAcceptanceRequest(candidate)
			test.mutate(&request)
			before := planCandidateTestSnapshot(t, root)
			if _, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Now().UTC()}); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("acceptance error = %v, want field-specific %q refusal", err, test.want)
			}
			planCandidateTestAssertSnapshot(t, root, before)
		})
	}
}

func TestPlanCandidateAcceptRejectsRejectedCandidateWithoutMutation(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	candidate.Status = colony.PlanCandidateRejected
	content, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID))), content, 0600); err != nil {
		t.Fatal(err)
	}
	before := planCandidateTestSnapshot(t, root)
	if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Now().UTC()}); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("rejected candidate acceptance error = %v, want status refusal", err)
	}
	planCandidateTestAssertSnapshot(t, root, before)
}

func TestPlanCandidateAcceptTransactionFaultLeavesFrontierByteIdentical(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	before := planCandidateTestSnapshot(t, root)
	injected := errors.New("injected candidate acceptance fault")
	_, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner", AcceptedAt: time.Now().UTC(),
		Fault: func(point string) error {
			if point == "after_validation" {
				return injected
			}
			return nil
		},
	})
	if !errors.Is(err, injected) {
		t.Fatalf("acceptance fault = %v, want injected transaction fault", err)
	}
	planCandidateTestAssertSnapshot(t, root, before)
}

func TestPlanCandidateAcceptCommandActivatesExactFrontier(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	request := planCandidateTestAcceptanceRequest(candidate)
	result, handled, err := runPlanCandidateCommand(root, planCandidateCommandInputs{
		AcceptCandidate: request.CandidateID, SpecificationRevisionID: request.SpecificationRevisionID,
		SpecificationRevisionHash: request.SpecificationRevisionHash, BasePlanRevisionID: request.BasePlanRevisionID,
		TimelineDigest: request.TimelineDigest, ProposalHash: request.ProposalHash, AcceptanceToken: request.AcceptanceToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !handled || result["operation"] != planCandidateOperationAccept || result["replayed"] != false {
		t.Fatalf("candidate acceptance command result = %#v handled=%t", result, handled)
	}
	revision, ok := result["revision"].(colony.PlanRevision)
	if !ok || revision.ID != candidate.Proposal.ID {
		t.Fatalf("candidate acceptance command revision = %#v", result["revision"])
	}
	receipt, ok := result["acceptance_receipt"].(colony.PlanAcceptanceReceipt)
	if !ok || receipt.CandidateID != candidate.ID || receipt.ActivatedPlanRevisionID != revision.ID {
		t.Fatalf("candidate acceptance command receipt = %#v", result["acceptance_receipt"])
	}
}

func planCandidateTestAcceptanceRequest(candidate colony.PlanCandidate) planCandidateAcceptanceRequest {
	return planCandidateAcceptanceRequest{
		CandidateID: candidate.ID, SpecificationRevisionID: candidate.SpecificationRevisionID,
		SpecificationRevisionHash: candidate.SpecificationRevisionHash, BasePlanRevisionID: candidate.BasePlanRevisionID,
		TimelineDigest: candidate.Timeline.TimelineDigest, ProposalHash: candidate.ProposalHash,
		AcceptanceToken: planCandidateAcceptanceToken(candidate),
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
