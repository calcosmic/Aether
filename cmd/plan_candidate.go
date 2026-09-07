package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

type planCandidateOperation string

const (
	planCandidateOperationNone   planCandidateOperation = ""
	planCandidateOperationReview planCandidateOperation = "candidate_review"
	planCandidateOperationDetail planCandidateOperation = "candidate_iteration_detail"
	planCandidateOperationAccept planCandidateOperation = "candidate_accept"
)

// planCandidateAcceptanceRequest names every owner-controlled input required
// to distinguish one candidate frontier from every stale or divergent one.
type planCandidateAcceptanceRequest struct {
	CandidateID               string `json:"candidate_id"`
	SpecificationRevisionID   string `json:"specification_revision_id"`
	SpecificationRevisionHash string `json:"specification_revision_hash"`
	BasePlanRevisionID        string `json:"base_plan_revision_id"`
	TimelineDigest            string `json:"timeline_digest"`
	ProposalHash              string `json:"proposal_hash"`
	AcceptanceToken           string `json:"acceptance_token"`
}

type planCandidateConfidenceScore struct {
	Dimension colony.PlanningDimension `json:"dimension"`
	Score     int                      `json:"score"`
}

type planCandidateReview struct {
	Operation               planCandidateOperation         `json:"operation"`
	Candidate               colony.PlanCandidate           `json:"candidate"`
	TargetConfidence        int                            `json:"target_confidence"`
	ActualConfidence        int                            `json:"actual_confidence"`
	Scores                  []planCandidateConfidenceScore `json:"scores"`
	StopDecision            colony.PlanningStopDecision    `json:"stop_decision"`
	ResidualGaps            []colony.PlanningGap           `json:"residual_gaps"`
	EvidenceThatWouldChange string                         `json:"evidence_that_would_change"`
	SemanticDelta           colony.PlanningSemanticDelta   `json:"semantic_delta"`
	Recommendation          colony.QueenPlanRecommendation `json:"recommendation"`
	Timeline                colony.PlanningTimelineBinding `json:"timeline"`
	Iterations              []colony.PlanningIterationCard `json:"iterations"`
	Acceptance              planCandidateAcceptanceRequest `json:"acceptance"`
	AcceptanceCommand       string                         `json:"acceptance_command"`
}

type planCandidateIterationDetail struct {
	Operation      planCandidateOperation       `json:"operation"`
	CandidateID    string                       `json:"candidate_id"`
	TimelineID     string                       `json:"timeline_id"`
	TimelineDigest string                       `json:"timeline_digest"`
	Card           colony.PlanningIterationCard `json:"card"`
}

type planCandidateArtifact struct {
	Candidate colony.PlanCandidate
	Stage     planningStageState
	Header    planningRunHeader
}

type planCandidateCommandInputs struct {
	DeprecatedAccept          bool
	Candidate                 bool
	ShowIteration             int
	ShowIterationSet          bool
	Details                   bool
	AcceptCandidate           string
	SpecificationRevisionID   string
	SpecificationRevisionHash string
	BasePlanRevisionID        string
	TimelineDigest            string
	ProposalHash              string
	AcceptanceToken           string
	ConflictingPlanFlags      []string
}

func resolvePlanCandidateOperation(inputs planCandidateCommandInputs) (planCandidateOperation, error) {
	if inputs.DeprecatedAccept {
		return planCandidateOperationNone, fmt.Errorf("--accept no longer accepts or activates a plan; inspect `aether plan --candidate`, then run the exact `aether plan --accept-candidate <candidate-id> ...` command shown there")
	}

	detailRequested := inputs.ShowIterationSet || inputs.ShowIteration != 0 || inputs.Details
	acceptanceFields := []struct {
		flag  string
		value string
	}{
		{flag: "--accept-candidate", value: inputs.AcceptCandidate},
		{flag: "--spec-revision", value: inputs.SpecificationRevisionID},
		{flag: "--spec-hash", value: inputs.SpecificationRevisionHash},
		{flag: "--base-plan-revision", value: inputs.BasePlanRevisionID},
		{flag: "--timeline-digest", value: inputs.TimelineDigest},
		{flag: "--proposal-hash", value: inputs.ProposalHash},
		{flag: "--acceptance-token", value: inputs.AcceptanceToken},
	}
	acceptanceRequested := false
	for _, field := range acceptanceFields {
		acceptanceRequested = acceptanceRequested || strings.TrimSpace(field.value) != ""
	}

	selected := 0
	if inputs.Candidate {
		selected++
	}
	if detailRequested {
		selected++
	}
	if acceptanceRequested {
		selected++
	}
	if selected > 1 {
		return planCandidateOperationNone, fmt.Errorf("candidate review, iteration detail, and acceptance cannot combine; run exactly one operation")
	}
	if selected > 0 && len(inputs.ConflictingPlanFlags) > 0 {
		return planCandidateOperationNone, fmt.Errorf("candidate operations cannot combine with planning flags %s", strings.Join(inputs.ConflictingPlanFlags, ", "))
	}

	if detailRequested {
		if !inputs.Details {
			return planCandidateOperationNone, fmt.Errorf("--show-iteration requires --details so the immutable card contract is explicit")
		}
		if !inputs.ShowIterationSet && inputs.ShowIteration == 0 {
			return planCandidateOperationNone, fmt.Errorf("--details requires --show-iteration N")
		}
		if inputs.ShowIteration <= 0 {
			return planCandidateOperationNone, fmt.Errorf("--show-iteration must be a positive pass ordinal")
		}
		return planCandidateOperationDetail, nil
	}
	if inputs.Candidate {
		return planCandidateOperationReview, nil
	}
	if acceptanceRequested {
		if strings.TrimSpace(inputs.AcceptCandidate) == "" {
			provided := "exact candidate bindings"
			for _, field := range acceptanceFields[1:] {
				if strings.TrimSpace(field.value) != "" {
					provided = field.flag
					break
				}
			}
			return planCandidateOperationNone, fmt.Errorf("%s requires --accept-candidate <candidate-id>; first run `aether plan --candidate`", provided)
		}
		missing := make([]string, 0)
		for _, field := range acceptanceFields[1:] {
			if strings.TrimSpace(field.value) == "" {
				missing = append(missing, field.flag)
			}
		}
		if len(missing) > 0 {
			return planCandidateOperationNone, fmt.Errorf("--accept-candidate requires %s from the exact command shown by `aether plan --candidate`", strings.Join(missing, ", "))
		}
		return planCandidateOperationAccept, nil
	}
	return planCandidateOperationNone, nil
}

func planCandidateRequestFromInputs(inputs planCandidateCommandInputs) planCandidateAcceptanceRequest {
	return planCandidateAcceptanceRequest{
		CandidateID: strings.TrimSpace(inputs.AcceptCandidate), SpecificationRevisionID: strings.TrimSpace(inputs.SpecificationRevisionID),
		SpecificationRevisionHash: strings.TrimSpace(inputs.SpecificationRevisionHash), BasePlanRevisionID: strings.TrimSpace(inputs.BasePlanRevisionID),
		TimelineDigest: strings.TrimSpace(inputs.TimelineDigest), ProposalHash: strings.TrimSpace(inputs.ProposalHash),
		AcceptanceToken: strings.TrimSpace(inputs.AcceptanceToken),
	}
}

func runPlanCandidateCommand(root string, inputs planCandidateCommandInputs) (map[string]interface{}, bool, error) {
	operation, err := resolvePlanCandidateOperation(inputs)
	if err != nil {
		return nil, true, err
	}
	switch operation {
	case planCandidateOperationReview:
		review, reviewErr := reviewPlanCandidate(root)
		if reviewErr != nil {
			return nil, true, reviewErr
		}
		return planCandidateReviewResult(review), true, nil
	case planCandidateOperationDetail:
		detail, detailErr := reviewPlanCandidateIteration(root, inputs.ShowIteration)
		if detailErr != nil {
			return nil, true, detailErr
		}
		return planCandidateDetailResult(detail), true, nil
	case planCandidateOperationAccept:
		accepted, acceptErr := acceptPlanCandidate(root, planCandidateRequestFromInputs(inputs), planCandidateAcceptanceOptions{AcceptedBy: "owner"})
		if acceptErr != nil {
			return nil, true, acceptErr
		}
		return map[string]interface{}{
			"operation": planCandidateOperationAccept, "candidate": accepted.Candidate,
			"revision": accepted.Revision, "acceptance_receipt": accepted.Receipt, "replayed": accepted.Replayed,
		}, true, nil
	default:
		return nil, false, nil
	}
}

func planCandidateAcceptanceToken(candidate colony.PlanCandidate) string {
	material := strings.Join([]string{
		"plan-candidate-acceptance/v1", candidate.ID, candidate.ContentHash,
		candidate.SpecificationRevisionID, candidate.SpecificationRevisionHash,
		candidate.BasePlanRevisionID, candidate.BasePlanRevisionHash,
		candidate.Timeline.ID, candidate.Timeline.TimelineDigest, candidate.ProposalHash,
	}, "\n")
	digest := strings.TrimPrefix(lifecycleDigest([]byte(material)), "sha256:")
	return "accept-plan-" + digest[:24]
}

func reviewPlanCandidate(root string) (planCandidateReview, error) {
	artifact, err := loadPlanCandidateArtifact(root, "")
	if err != nil {
		return planCandidateReview{}, err
	}
	timeline, err := verifiedPlanCandidateTimeline(root, artifact.Candidate)
	if err != nil {
		return planCandidateReview{}, err
	}

	scores := make([]planCandidateConfidenceScore, 0, len(colony.PlanningDimensions()))
	weighted := planningConfidenceScores{}
	byDimension := make(map[colony.PlanningDimension]int, len(artifact.Candidate.DimensionAssessments))
	for _, assessment := range artifact.Candidate.DimensionAssessments {
		byDimension[assessment.Dimension] = assessment.After
		weighted.Set(assessment.Dimension, assessment.After)
	}
	for _, dimension := range colony.PlanningDimensions() {
		scores = append(scores, planCandidateConfidenceScore{Dimension: dimension, Score: byDimension[dimension]})
	}
	request := planCandidateAcceptanceRequest{
		CandidateID: artifact.Candidate.ID, SpecificationRevisionID: artifact.Candidate.SpecificationRevisionID,
		SpecificationRevisionHash: artifact.Candidate.SpecificationRevisionHash, BasePlanRevisionID: artifact.Candidate.BasePlanRevisionID,
		TimelineDigest: artifact.Candidate.Timeline.TimelineDigest, ProposalHash: artifact.Candidate.ProposalHash,
		AcceptanceToken: planCandidateAcceptanceToken(artifact.Candidate),
	}
	return planCandidateReview{
		Operation: planCandidateOperationReview, Candidate: artifact.Candidate,
		TargetConfidence: artifact.Header.TargetConfidence, ActualConfidence: weighted.Overall, Scores: scores,
		StopDecision: artifact.Candidate.StopDecision, ResidualGaps: append([]colony.PlanningGap(nil), artifact.Candidate.ResidualGaps...),
		EvidenceThatWouldChange: artifact.Candidate.EvidenceThatWouldChange, SemanticDelta: artifact.Candidate.SemanticDelta,
		Recommendation: artifact.Candidate.Recommendation, Timeline: artifact.Candidate.Timeline,
		Iterations: append([]colony.PlanningIterationCard(nil), timeline.Cards...), Acceptance: request,
		AcceptanceCommand: planCandidateAcceptanceCommand(request),
	}, nil
}

func reviewPlanCandidateIteration(root string, iteration int) (planCandidateIterationDetail, error) {
	if iteration <= 0 {
		return planCandidateIterationDetail{}, fmt.Errorf("planning iteration must be positive")
	}
	artifact, err := loadPlanCandidateArtifact(root, "")
	if err != nil {
		return planCandidateIterationDetail{}, err
	}
	timeline, err := verifiedPlanCandidateTimeline(root, artifact.Candidate)
	if err != nil {
		return planCandidateIterationDetail{}, err
	}
	if iteration > len(timeline.Cards) || timeline.Cards[iteration-1].Iteration != iteration {
		return planCandidateIterationDetail{}, fmt.Errorf("candidate %s has no immutable iteration %d (timeline contains passes 1-%d)", artifact.Candidate.ID, iteration, len(timeline.Cards))
	}
	card := timeline.Cards[iteration-1]
	return planCandidateIterationDetail{
		Operation: planCandidateOperationDetail, CandidateID: artifact.Candidate.ID,
		TimelineID: artifact.Candidate.Timeline.ID, TimelineDigest: artifact.Candidate.Timeline.TimelineDigest, Card: card,
	}, nil
}

func loadPlanCandidateArtifact(root, requestedID string) (planCandidateArtifact, error) {
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return planCandidateArtifact{}, err
	}
	planningRoot := filepath.Join(repositoryRoot, ".aether", "data", "planning")
	entries, err := os.ReadDir(planningRoot)
	if err != nil {
		return planCandidateArtifact{}, fmt.Errorf("read planning candidates: %w", err)
	}
	wanted := strings.TrimSpace(requestedID)
	matches := make([]planCandidateArtifact, 0, 1)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return planCandidateArtifact{}, fmt.Errorf("planning run %q must not be a symlink", entry.Name())
		}
		if !entry.IsDir() {
			continue
		}
		runID := entry.Name()
		if err := validatePlanningTimelineSegment("run_id", runID); err != nil {
			return planCandidateArtifact{}, err
		}
		content, exists, readErr := readOptionalPlanningStageFile(repositoryRoot, planningRouteCandidateRepositoryPath(runID))
		if readErr != nil {
			return planCandidateArtifact{}, readErr
		}
		if !exists {
			continue
		}
		var candidate colony.PlanCandidate
		if err := decodePlanningStageJSON(content, &candidate); err != nil {
			return planCandidateArtifact{}, fmt.Errorf("decode candidate for run %q: %w", runID, err)
		}
		if wanted != "" && candidate.ID != wanted {
			continue
		}
		if wanted == "" && candidate.Status != colony.PlanCandidatePendingReview {
			continue
		}
		if candidate.Timeline.RunID != runID {
			return planCandidateArtifact{}, fmt.Errorf("candidate %s path does not match timeline run", candidate.ID)
		}
		if err := validatePlanningRecordHashes(candidate); err != nil {
			return planCandidateArtifact{}, fmt.Errorf("candidate %s: %w", candidate.ID, err)
		}
		if err := candidate.Validate(); err != nil {
			return planCandidateArtifact{}, fmt.Errorf("candidate %s: %w", candidate.ID, err)
		}
		stage, err := loadPlanningStageState(repositoryRoot, runID)
		if err != nil {
			return planCandidateArtifact{}, err
		}
		if stage.Stage != planningStageCandidateReady && stage.Stage != planningStageAccepted {
			return planCandidateArtifact{}, fmt.Errorf("candidate %s is not at a reviewable planning boundary (stage %s)", candidate.ID, stage.Stage)
		}
		header, err := loadPlanCandidateRunHeader(repositoryRoot, candidate)
		if err != nil {
			return planCandidateArtifact{}, err
		}
		matches = append(matches, planCandidateArtifact{Candidate: candidate, Stage: stage, Header: header})
	}
	if len(matches) == 0 {
		if wanted == "" {
			return planCandidateArtifact{}, fmt.Errorf("no reviewable plan candidate found; finish iterative planning first")
		}
		return planCandidateArtifact{}, fmt.Errorf("plan candidate %q was not found at a reviewable boundary", wanted)
	}
	if len(matches) > 1 {
		sort.Slice(matches, func(i, j int) bool { return matches[i].Candidate.CreatedAt.Before(matches[j].Candidate.CreatedAt) })
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].Candidate.ID
		}
		return planCandidateArtifact{}, fmt.Errorf("multiple reviewable plan candidates are present (%s); refuse ambiguous review until obsolete runs are resolved", strings.Join(ids, ", "))
	}
	return matches[0], nil
}

func loadPlanCandidateRunHeader(root string, candidate colony.PlanCandidate) (planningRunHeader, error) {
	headerPath := filepath.ToSlash(filepath.Join(".aether", "data", "planning", candidate.Timeline.RunID, "run-header.json"))
	content, exists, err := readOptionalPlanningStageFile(root, headerPath)
	if err != nil {
		return planningRunHeader{}, err
	}
	if !exists {
		return planningRunHeader{}, fmt.Errorf("candidate %s planning run header is missing", candidate.ID)
	}
	var header planningRunHeader
	if err := decodePlanningStageJSON(content, &header); err != nil {
		return planningRunHeader{}, fmt.Errorf("decode candidate planning run header: %w", err)
	}
	manifest, err := loadPlanningStageManifest(root, header.RunID, header.StageManifestID)
	if err != nil {
		return planningRunHeader{}, err
	}
	header, err = loadPlanningScoutRunHeader(root, manifest)
	if err != nil {
		return planningRunHeader{}, err
	}
	if header.Specification.RevisionID != candidate.SpecificationRevisionID || header.Specification.ContentHash != candidate.SpecificationRevisionHash {
		return planningRunHeader{}, fmt.Errorf("candidate %s does not match its immutable planning run header", candidate.ID)
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return planningRunHeader{}, err
	}
	if candidate.Status == colony.PlanCandidateAccepted {
		if err := validatePlanningState(state); err != nil {
			return planningRunHeader{}, fmt.Errorf("candidate %s accepted state: %w", candidate.ID, err)
		}
		return header, nil
	}
	if err := validateCandidateRunBase(state.Plan, header); err != nil {
		return planningRunHeader{}, fmt.Errorf("candidate %s: %w", candidate.ID, err)
	}
	base, _, err := candidateAcceptanceBase(state.Plan, candidate.CreatedAt)
	if err != nil {
		return planningRunHeader{}, err
	}
	if candidate.BasePlanRevisionID != base.ID || candidate.BasePlanRevisionHash != base.Hash {
		return planningRunHeader{}, fmt.Errorf("candidate %s does not bind the immutable revision represented by its planning run base", candidate.ID)
	}
	return header, nil
}

func verifiedPlanCandidateTimeline(root string, candidate colony.PlanCandidate) (planningTimeline, error) {
	timeline, err := loadPlanningTimeline(root, candidate.Timeline.RunID)
	if err != nil {
		return planningTimeline{}, err
	}
	if timeline.Binding == nil || timeline.Index == nil {
		return planningTimeline{}, fmt.Errorf("candidate %s requires a complete indexed timeline", candidate.ID)
	}
	wantHash, err := jsonSHA256(candidate.Timeline)
	if err != nil {
		return planningTimeline{}, err
	}
	gotHash, err := jsonSHA256(*timeline.Binding)
	if err != nil {
		return planningTimeline{}, err
	}
	if wantHash != gotHash {
		return planningTimeline{}, fmt.Errorf("candidate %s timeline binding is stale or divergent", candidate.ID)
	}
	if err := validatePlanningTimelineBindingContent(candidate.Timeline, timeline.Cards); err != nil {
		return planningTimeline{}, fmt.Errorf("candidate %s timeline: %w", candidate.ID, err)
	}
	return timeline, nil
}

func planCandidateAcceptanceCommand(request planCandidateAcceptanceRequest) string {
	return strings.Join([]string{
		"aether plan", "--accept-candidate", shellQuotePlanArg(request.CandidateID),
		"--spec-revision", shellQuotePlanArg(request.SpecificationRevisionID),
		"--spec-hash", shellQuotePlanArg(request.SpecificationRevisionHash),
		"--base-plan-revision", shellQuotePlanArg(request.BasePlanRevisionID),
		"--timeline-digest", shellQuotePlanArg(request.TimelineDigest),
		"--proposal-hash", shellQuotePlanArg(request.ProposalHash),
		"--acceptance-token", shellQuotePlanArg(request.AcceptanceToken),
	}, " ")
}

func planCandidateReviewResult(review planCandidateReview) map[string]interface{} {
	confidence := codexPlanConfidence{}
	for _, score := range review.Scores {
		switch score.Dimension {
		case colony.PlanningDimensionKnowledge:
			confidence.Knowledge = planScore(score.Score)
		case colony.PlanningDimensionRequirements:
			confidence.Requirements = planScore(score.Score)
		case colony.PlanningDimensionRisks:
			confidence.Risks = planScore(score.Score)
		case colony.PlanningDimensionDependencies:
			confidence.Dependencies = planScore(score.Score)
		case colony.PlanningDimensionEffort:
			confidence.Effort = planScore(score.Score)
		}
	}
	confidence.Overall = planScore(review.ActualConfidence)
	return map[string]interface{}{
		"operation": review.Operation, "candidate": review.Candidate, "phases": review.Candidate.Proposal.Phases,
		"target_confidence": review.TargetConfidence, "actual_confidence": review.ActualConfidence, "scores": review.Scores,
		"confidence": confidence, "stop_decision": review.StopDecision, "residual_gaps": review.ResidualGaps,
		"evidence_that_would_change": review.EvidenceThatWouldChange, "semantic_delta": review.SemanticDelta,
		"recommendation": review.Recommendation, "timeline": review.Timeline, "iterations": review.Iterations,
		"acceptance": review.Acceptance, "acceptance_command": review.AcceptanceCommand,
	}
}

func planCandidateDetailResult(detail planCandidateIterationDetail) map[string]interface{} {
	return map[string]interface{}{
		"operation": detail.Operation, "candidate_id": detail.CandidateID,
		"timeline_id": detail.TimelineID, "timeline_digest": detail.TimelineDigest, "iteration": detail.Card,
	}
}
