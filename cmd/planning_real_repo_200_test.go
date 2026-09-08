package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningRealRepo200(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	git := exec.Command("git", "init", "-q", root)
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("initialize real git repository: %v\n%s", err, output)
	}
	if output, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output(); err != nil || !planningRealRepo200SamePath(t, strings.TrimSpace(string(output)), root) {
		t.Fatalf("temporary fixture is not a real repository: output=%q err=%v", output, err)
	}

	seedSettledDiscussState(t, dataDir, "real-repo-200")
	discussed, err := runDiscuss(root, 3, false)
	if err != nil {
		t.Fatalf("settle init/discuss intent: %v", err)
	}
	if stringValue(discussed["specification_status"]) != string(colony.SpecStatusDraft) || discussed["draft_spec"] == nil {
		t.Fatalf("settled discussion did not create a draft Specification: %#v", discussed)
	}
	draftState := mustReadSpecificationTestState(t, root)
	if draftState.Specification == nil {
		t.Fatal("discussion did not persist canonical Specification state")
	}
	draft, ok := currentSpecificationRevision(*draftState.Specification)
	if !ok {
		t.Fatal("draft Specification revision is unavailable")
	}
	assertSpecificationTestBodyComplete(t, draft)

	beforeDraftRefusal := planningRealRepo200Snapshot(t, root)
	if _, err := requireApprovedPlanningSpecification(root, draftState); err == nil || !strings.Contains(strings.ToLower(err.Error()), "approved") {
		t.Fatalf("planning from DRAFT error = %v", err)
	}
	if after := planningRealRepo200Snapshot(t, root); !reflect.DeepEqual(beforeDraftRefusal, after) {
		t.Fatal("draft planning refusal changed repository state or artifacts")
	}

	approved, _ := runSpecCommandSuccess(t, root,
		"--approve", "--revision-id", draft.ID, "--revision-hash", draft.ContentHash,
		"--approval-token", specificationApprovalToken(draft.SpecificationID, draft.ID, draft.ContentHash),
	)
	if approved.Status != colony.SpecStatusApproved || approved.Approval == nil || approved.Receipt == nil {
		t.Fatalf("exact specification approval did not persist authority: %+v", approved)
	}
	selection, err := resolvePlanningPreset(codexPlanOptions{Preset: "balanced", PresetSet: true})
	if err != nil || selection.PresetRequired || selection.Policy.ID != planningStagePresetBalanced {
		t.Fatalf("explicit preset selection = %+v err=%v", selection, err)
	}

	first, second, candidate := planningRealRepo200DriveTwoPasses(t, root)
	if first.Route.Card.Iteration != 1 || second.Route.Card.Iteration != 2 {
		t.Fatalf("cards are not ordered: first=%d second=%d", first.Route.Card.Iteration, second.Route.Card.Iteration)
	}
	if first.ScoutDispatch == nil || first.ScoutDispatch.Manifest.WeakestGap == nil || first.ScoutDispatch.Manifest.WeakestGap.ID != first.Route.Card.WeakestGap.ID {
		t.Fatalf("first weakest gap did not become next Scout focus: %+v", first.ScoutDispatch)
	}
	if second.Route.Card.ScoutReceiptID == first.Route.Card.ScoutReceiptID || second.Route.Card.RouteSetterReceiptID == first.Route.Card.RouteSetterReceiptID {
		t.Fatalf("two passes reused a receipt: first=%+v second=%+v", first.Route.Card, second.Route.Card)
	}
	if candidate.Status != colony.PlanCandidatePendingReview || candidate.Acceptance != nil {
		t.Fatalf("reasoned stop activated the candidate: %+v", candidate)
	}
	for _, card := range []colony.PlanningIterationCard{first.Route.Card, second.Route.Card} {
		if strings.TrimSpace(card.EvidenceThatWouldChange) == "" || strings.TrimSpace(card.WeakestGap.EvidenceThatWouldChange) == "" || strings.TrimSpace(card.Decision.EvidenceThatWouldChange) == "" {
			t.Fatalf("card %d lacks causal evidence-that-would-change: %+v", card.Iteration, card)
		}
	}
	if candidate.Recommendation.Producer != colony.PlanRecommendationProducerQueen || strings.TrimSpace(candidate.Recommendation.Rationale) == "" || len(candidate.Recommendation.EvidenceIDs) == 0 || strings.TrimSpace(candidate.EvidenceThatWouldChange) == "" {
		t.Fatalf("candidate lacks persisted Queen recommendation/evidence: %+v", candidate)
	}

	pendingState := mustReadSpecificationTestState(t, root)
	_, buildBefore, buildErr := resolveCodexBuildPlanAuthority(root, pendingState)
	if buildErr == nil || buildBefore.Eligible || buildBefore.RecoveryCommand != "aether plan --candidate" {
		t.Fatalf("pending candidate build gate = %+v err=%v", buildBefore, buildErr)
	}
	stale := planCandidateTestAcceptanceRequest(candidate)
	stale.ProposalHash = strings.Repeat("3", 64)
	beforeStale := planningRealRepo200Snapshot(t, root)
	if _, err := acceptPlanCandidate(root, stale, planCandidateAcceptanceOptions{AcceptedBy: "owner"}); err == nil || !strings.Contains(err.Error(), "proposal_hash") {
		t.Fatalf("stale acceptance error = %v", err)
	}
	if after := planningRealRepo200Snapshot(t, root); !reflect.DeepEqual(beforeStale, after) {
		t.Fatal("stale acceptance changed repository state or artifacts")
	}

	request := planCandidateTestAcceptanceRequest(candidate)
	acceptedResult, handled, err := runPlanCandidateCommand(root, planCandidateCommandInputs{
		AcceptCandidate: request.CandidateID, SpecificationRevisionID: request.SpecificationRevisionID,
		SpecificationRevisionHash: request.SpecificationRevisionHash, BasePlanRevisionID: request.BasePlanRevisionID,
		TimelineDigest: request.TimelineDigest, ProposalHash: request.ProposalHash, AcceptanceToken: request.AcceptanceToken,
	})
	if err != nil || !handled || acceptedResult["operation"] != planCandidateOperationAccept || acceptedResult["replayed"] != false {
		t.Fatalf("exact candidate acceptance = %#v handled=%t err=%v", acceptedResult, handled, err)
	}
	acceptedState := mustReadSpecificationTestState(t, root)
	if acceptedState.State != colony.StateREADY || acceptedState.Plan.PendingCandidateID != "" || acceptedState.Plan.ActiveRevisionID != candidate.Proposal.ID {
		t.Fatalf("accepted plan is not READY on the exact revision: %+v", acceptedState.Plan)
	}
	_, buildAfter, err := resolveCodexBuildPlanAuthority(root, acceptedState)
	if err != nil || !buildAfter.Eligible {
		t.Fatalf("accepted candidate did not grant build authority: %+v err=%v", buildAfter, err)
	}
	facts := lifecycleFactsFromStateSnapshot(acceptedState, false, time.Date(2026, time.September, 8, 2, 0, 0, 0, time.UTC))
	facts.Root = root
	runAfter := buildAutopilotPreflight(facts)
	if !runAfter.Valid || !runAfter.PlanAuthority.Eligible || runAfter.FirstPhase != 1 {
		t.Fatalf("build/run authority diverged: build=%+v run=%+v", buildAfter, runAfter)
	}

	t.Run("scoped successor preserves independent completed work and blocks execution", func(t *testing.T) {
		planningRealRepo200AssertScopedRevision(t)
	})
	t.Run("later material evidence reaches Route before the owner boundary", func(t *testing.T) {
		planningRealRepo200AssertLaterMaterialDecision(t, "keep-approved", false)
	})
	t.Run("later contract answer creates a successor approval boundary", func(t *testing.T) {
		planningRealRepo200AssertLaterMaterialDecision(t, "change-contract", true)
	})
}

func planningRealRepo200AssertLaterMaterialDecision(t *testing.T, choiceID string, wantSuccessor bool) {
	t.Helper()
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("initialize later-material repository: %v\n%s", err, output)
	}
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil {
		t.Fatal("first Route pass did not authorize the later Scout")
	}

	scoutManifest := first.ScoutDispatch.Manifest
	fresh := planningRouteStageEvidence(t, scoutManifest.Specification, scoutManifest.BasePlanRevisionID, "real-repo-late-material", "A later Scout found a material owner-visible behavior choice.", time.Date(2026, time.September, 8, 2, 30, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("real-repo-late-scout-gap", colony.PlanningDimensionRisks, fresh.Reference.ID, colony.PlanningGapNonMaterial, 20)
	material := planningScoutStageMaterialCandidate(fresh.Reference, "real-repo-late-material-decision")
	targetID := firstResult.Proposal.Phases[0].RequirementProofLinks[0]
	material.Choices = append(material.Choices, planningDecisionChoice{
		ID:                  "change-contract",
		Label:               "Change the approved requirement",
		Consequence:         "A successor Specification must be approved and reconciled before planning resumes.",
		Impact:              planningDecisionContractImpact{Behavior: "Require the newly discovered owner-visible behavior."},
		AffectedSemanticIDs: []string{targetID},
	})
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID: "real-repo-late-material-finding", Summary: "The complete Route pass must persist before owner review.", EvidenceIDs: []string{fresh.Reference.ID},
		}},
		NewEvidence:        []planningEvidenceRecord{fresh},
		UnresolvedGaps:     []colony.PlanningGap{scoutGap},
		DecisionCandidates: []planningDecisionCandidate{material},
	}
	scoutCompleted, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if scoutCompleted.DecisionCheckpoint != nil || scoutCompleted.RouteDispatch == nil {
		t.Fatalf("later material Scout boundary = %+v, want Route-Setter before any owner pause", scoutCompleted)
	}
	if carried := scoutCompleted.RouteDispatch.MaterialDecisionCandidates; len(carried) != 1 || carried[0].StableID != material.StableID {
		t.Fatalf("later Scout decision was not carried through Route authorization: %+v", carried)
	}

	routeManifest := scoutCompleted.RouteDispatch.Manifest
	secondResult := planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: routeManifest.Specification, BasePlanRevisionID: routeManifest.BasePlanRevisionID, BasePlanRevisionHash: routeManifest.BasePlanRevisionHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: firstResult.Proposal, ProposalEvidenceIDs: []string{fresh.Reference.ID}, MaterialDecisionCandidates: []planningDecisionCandidate{material},
	}
	for index, dimension := range colony.PlanningDimensions() {
		before := first.Route.Card.DimensionAssessments[index].After
		secondResult.DimensionAssessments = append(secondResult.DimensionAssessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "real-repo-late-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('a' + index))),
			Dimension: dimension, Before: before, After: before, FreshEvidenceIDs: []string{fresh.Reference.ID},
			RemainingGap: planningRouteStageGap("real-repo-late-gap-"+string(dimension), dimension, fresh.Reference.ID, colony.PlanningGapNonMaterial, 15+index),
			Rationale:    "The later Scout evidence preserves this score pending the owner decision.", ProducerReceiptID: routeManifest.ID,
		})
	}
	second, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, secondResult))
	if err != nil {
		t.Fatal(err)
	}
	if second.Route.Card.Iteration != 2 || second.DecisionCheckpoint == nil || second.ScoutDispatch != nil || second.Candidate != nil {
		t.Fatalf("later material Route completion = %+v, want a complete second card followed by one owner boundary", second)
	}
	checkpoint := second.DecisionCheckpoint
	if checkpoint.CompletedCardHash != second.Route.Card.ContentHash || checkpoint.Batch.BoundaryCardHash != second.Route.Card.ContentHash {
		t.Fatalf("owner boundary is not bound to the complete second card: checkpoint=%+v card=%+v", checkpoint, second.Route.Card)
	}
	timeline, err := loadPlanningTimeline(root, routeManifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Cards) != 2 || timeline.Cards[1].ContentHash != second.Route.Card.ContentHash {
		t.Fatalf("owner pause preceded complete card persistence: %+v", timeline.Cards)
	}

	decisionCard := checkpoint.Cards[0]
	var choice planningDecisionChoice
	for _, candidate := range material.Choices {
		if candidate.ID == choiceID {
			choice = candidate
			break
		}
	}
	if choice.ID == "" {
		t.Fatalf("choice %q is absent from material decision", choiceID)
	}
	resume, err := buildPlanningScoutDecisionResumeToken(*checkpoint, []planningScoutDecisionAnswer{{
		DecisionID: decisionCard.DecisionID, ChoiceID: choice.ID, Answer: choice.Label,
	}})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := resumePlanningRouteDecision(root, routeManifest.RunID, resume, time.Date(2026, time.September, 8, 2, 35, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadPlanningStageState(root, routeManifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if wantSuccessor {
		if resumed.SuccessorSpecification == nil || resumed.SuccessorSpecification.Revision.Status != colony.SpecStatusDraft || resumed.ScoutDispatch != nil || resumed.Candidate != nil {
			t.Fatalf("contract-affecting answer = %+v, want successor DRAFT without progression", resumed)
		}
		if state.Stage != planningStageSpecApprovalRequired || state.PendingSpecification == nil || state.PendingSpecification.RevisionID != resumed.SuccessorSpecification.Revision.ID || len(state.PendingAffectedSemanticIDs) == 0 {
			t.Fatalf("contract-affecting answer state = %+v, want exact approval and reconciliation boundary", state)
		}
		return
	}
	if resumed.ScoutDispatch == nil || resumed.SuccessorSpecification != nil || resumed.ResumeToken == nil || state.Stage != planningStageScoutRunning {
		t.Fatalf("contract-equivalent answer = %+v state=%+v, want direct next-Scout resume", resumed, state)
	}
}

func planningRealRepo200SamePath(t *testing.T, left, right string) bool {
	t.Helper()
	canonicalLeft, err := filepath.EvalSymlinks(left)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRight, err := filepath.EvalSymlinks(right)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(canonicalLeft) == filepath.Clean(canonicalRight)
}

type planningRealRepo200File struct {
	Path string
	Hash string
}

func planningRealRepo200Snapshot(t *testing.T, root string) []planningRealRepo200File {
	t.Helper()
	var result []planningRealRepo200File
	for _, start := range []string{filepath.Join(root, ".aether", "data"), filepath.Join(root, specificationProjectionRelativePath)} {
		info, err := os.Lstat(start)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			data, err := os.ReadFile(start)
			if err != nil {
				t.Fatal(err)
			}
			rel, _ := filepath.Rel(root, start)
			result = append(result, planningRealRepo200File{Path: filepath.ToSlash(rel), Hash: lifecycleDigest(data)})
			continue
		}
		if err := filepath.WalkDir(start, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			result = append(result, planningRealRepo200File{Path: filepath.ToSlash(rel), Hash: lifecycleDigest(data)})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func planningRealRepo200DriveTwoPasses(t *testing.T, root string) (planningRouteStageCoordination, planningRouteStageCoordination, colony.PlanCandidate) {
	t.Helper()
	firstManifest, firstResult := planningRealRepo200FirstRouteFixture(t, root)
	planningRouteStageSetPolicy(t, root, firstManifest.RunID, 99, 2)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil || first.Candidate != nil {
		t.Fatalf("first Route pass did not continue: %+v", first)
	}

	scoutManifest := first.ScoutDispatch.Manifest
	fresh := planningRealRepo200Evidence(t, root, scoutManifest.Specification, scoutManifest.BasePlanRevisionID, "second-pass", "A new repository inspection resolves the first weakest gap and changes the semantic plan.", time.Date(2026, time.September, 8, 2, 10, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("real-repo-second-scout-gap", colony.PlanningDimensionRisks, fresh.Reference.ID, colony.PlanningGapNonMaterial, 9)
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings:          []planningScoutStageFinding{{StableID: "real-repo-second-finding", Summary: "Fresh evidence changes the executable route.", EvidenceIDs: []string{fresh.Reference.ID}}},
		NewEvidence:       []planningEvidenceRecord{fresh}, UnresolvedGaps: []colony.PlanningGap{scoutGap},
	}
	scoutCompleted, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if scoutCompleted.RouteDispatch == nil {
		t.Fatal("second Scout did not authorize Route-Setter")
	}
	routeManifest := scoutCompleted.RouteDispatch.Manifest
	proposal := planningRouteStageCloneResult(t, firstResult).Proposal
	proposal.Phases[0].Description = "Finalize the evidence-improved route in the real repository"
	proposal.Phases[0].Tasks[0].Goal = "Verify the evidence-improved route"
	secondResult := planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: routeManifest.Specification, BasePlanRevisionID: routeManifest.BasePlanRevisionID, BasePlanRevisionHash: routeManifest.BasePlanRevisionHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: proposal, ProposalEvidenceIDs: []string{fresh.Reference.ID},
	}
	for index, dimension := range colony.PlanningDimensions() {
		before := first.Route.Card.DimensionAssessments[index].After
		secondResult.DimensionAssessments = append(secondResult.DimensionAssessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "real-repo-second-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('p' + index))),
			Dimension: dimension, Before: before, After: min(before+12, 98), FreshEvidenceIDs: []string{fresh.Reference.ID},
			RemainingGap: planningRouteStageGap("real-repo-second-gap-"+string(dimension), dimension, fresh.Reference.ID, colony.PlanningGapNonMaterial, 3+index),
			Rationale:    "The fresh repository evidence authorizes this proposed movement.", ProducerReceiptID: routeManifest.ID,
		})
	}
	second, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, secondResult))
	if err != nil {
		t.Fatal(err)
	}
	if second.Candidate == nil || second.Route.Card.Decision.Reason != colony.PlanningStopPassCap {
		t.Fatalf("second pass did not stop reasonedly at the selected cap: %+v", second)
	}
	return first, second, *second.Candidate
}

func planningRealRepo200FirstRouteFixture(t *testing.T, root string) (planningStageManifest, planningRouteStageResult) {
	t.Helper()
	state := mustReadSpecificationTestState(t, root)
	if state.Specification == nil {
		t.Fatal("approved Specification is missing")
	}
	specification, ok := currentSpecificationRevision(*state.Specification)
	if !ok || specification.Status != colony.SpecStatusApproved || specification.Approval == nil {
		t.Fatalf("planning fixture requires an approved Specification: %+v", specification)
	}
	approvalHash, err := jsonSHA256(*specification.Approval)
	if err != nil {
		t.Fatal(err)
	}
	baseHash, err := planStateHash(state.Plan)
	if err != nil {
		t.Fatal(err)
	}
	baseID, baseHash := planningBaseRevisionIdentity(state.Plan, baseHash)
	binding := planningStageSpecificationBinding{
		RevisionID: specification.ID, ContentHash: specification.ContentHash, Status: colony.SpecStatusApproved,
		ApprovalReceiptID: specification.Approval.ID, ApprovalReceiptHash: approvalHash,
	}
	seed := planningRealRepo200Evidence(t, root, binding, baseID, "seed", "The approved Specification and repository state seed this run.", time.Date(2026, time.September, 8, 2, 1, 0, 0, time.UTC))
	initial := planningStageState{
		Stage: planningStageScoutReady, RunID: "planning-real-repo-200", Pass: 1, Preset: planningStagePresetBalanced,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		PriorCardHash: planningStageTestHash("d"), InputFrontierHash: planningStageTestHash("e"),
	}
	authorization := planningStageAuthorization{
		ID: "authorization-real-repo-scout", ExpectedCaste: planningStageCasteScout, InputFrontierHash: initial.InputFrontierHash,
		EvidenceFrontier: []planningStageEvidenceBinding{{ID: seed.Reference.ID, ContentHash: seed.Reference.ContentHash}},
		WeakestGap:       planningStageTestGap("real-repo-initial-gap"),
	}
	running, scoutManifest, err := reducePlanningStage(initial, planningStageTransition{To: planningStageScoutRunning, Authorization: &authorization})
	if err != nil {
		t.Fatal(err)
	}
	if err := recordPlanningStageDispatch(root, running, *scoutManifest, planningStageWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	header := planningRunHeader{
		SchemaVersion: planningRunHeaderSchemaVersion, RunID: scoutManifest.RunID,
		Goal: strings.TrimSpace(derefGoal(state.Goal)), GoalID: specification.Scope.GoalID, SessionID: specification.Scope.SessionID,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		Preset: planningStagePresetBalanced, TargetConfidence: 90, PassCap: 6,
		EvidenceCatalogue: []planningEvidenceRecord{seed}, EvidenceFrontier: append([]planningStageEvidenceBinding(nil), scoutManifest.EvidenceFrontier...),
		InputFrontierHash: scoutManifest.InputFrontierHash,
		ResearchPolicy:    phaseResearchAutomaticPolicy{SchemaVersion: phaseResearchAutomaticPolicySchemaVersion, Preset: planningStagePresetBalanced, OwnerDecisionBoundary: "after_scout", EvidenceContract: automaticPhaseResearchEvidenceContract()},
		WeakestGap:        *scoutManifest.WeakestGap, StageManifestID: scoutManifest.ID, StageManifestHash: scoutManifest.ContentHash,
		CreatedAt: time.Date(2026, time.September, 8, 2, 0, 0, 0, time.UTC),
	}
	payload := header
	payload.ID, payload.ContentHash = "", ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		t.Fatal(err)
	}
	header.ContentHash, header.ID = hash, "planning-run-header-"+hash[:16]
	planningStageReceiptTestWriteJSON(t, filepath.Join(root, ".aether", "data", "planning", scoutManifest.RunID, "run-header.json"), header)

	fresh := planningRealRepo200Evidence(t, root, binding, baseID, "first-pass", "The first Scout maps the initial executable route.", time.Date(2026, time.September, 8, 2, 5, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("real-repo-first-scout-gap", colony.PlanningDimensionKnowledge, fresh.Reference.ID, colony.PlanningGapNonMaterial, 25)
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings:          []planningScoutStageFinding{{StableID: "real-repo-first-finding", Summary: "The first route can now be proposed.", EvidenceIDs: []string{fresh.Reference.ID}}},
		NewEvidence:       []planningEvidenceRecord{fresh}, UnresolvedGaps: []colony.PlanningGap{scoutGap},
	}
	scoutCompleted, err := coordinatePlanningScoutStage(root, *scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if scoutCompleted.RouteDispatch == nil {
		t.Fatal("first Scout did not authorize Route-Setter")
	}
	routeManifest := scoutCompleted.RouteDispatch.Manifest
	proposal := planningRouteStageProposal(specification)
	assessments := make([]colony.PlanningDimensionAssessment, 0, len(colony.PlanningDimensions()))
	for index, dimension := range colony.PlanningDimensions() {
		assessments = append(assessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "real-repo-first-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('u' + index))),
			Dimension: dimension, Before: 0, After: 70 + index, FreshEvidenceIDs: []string{fresh.Reference.ID},
			RemainingGap: planningRouteStageGap("real-repo-first-gap-"+string(dimension), dimension, fresh.Reference.ID, colony.PlanningGapNonMaterial, 10+index),
			Rationale:    "The first fresh Scout evidence supports the proposed score.", ProducerReceiptID: routeManifest.ID,
		})
	}
	return routeManifest, planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: binding, BasePlanRevisionID: baseID, BasePlanRevisionHash: baseHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: proposal, ProposalEvidenceIDs: []string{fresh.Reference.ID}, DimensionAssessments: assessments,
	}
}

func planningRealRepo200Evidence(t *testing.T, root string, binding planningStageSpecificationBinding, baseID, origin, content string, observedAt time.Time) planningEvidenceRecord {
	t.Helper()
	state := mustReadSpecificationTestState(t, root)
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("current Specification is unavailable for evidence scope")
	}
	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind: colony.PlanningEvidenceResearch, Origin: "real-repo-200:" + origin, Content: []byte(content),
		Scope:          planningEvidenceScope{GoalID: current.Scope.GoalID, SessionID: current.Scope.SessionID, SpecificationRevisionID: binding.RevisionID, PlanRevisionID: baseID},
		SourceRevision: origin + "-revision", ObservedAt: observedAt, ApplicableDimensions: colony.PlanningDimensions(), State: planningEvidenceSourceCurrent,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func planningRealRepo200AssertScopedRevision(t *testing.T) {
	t.Helper()
	state := specificationTestPlanState()
	state.Plan.Phases[0].Tasks[1].Status = colony.TaskCompleted
	root := newSpecificationTestRepository(t, state)
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("initialize scoped revision repository: %v\n%s", err, output)
	}
	draftRequest := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draft, err := createSpecificationDraft(root, draftRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	requirementID := draft.Revision.Requirements[0].ID
	decision := specificationTestSuccessorDecision(requirementID)
	request := specificationRevisionRequest{
		PredecessorRevisionID: draft.Revision.ID, PredecessorContentHash: draft.Revision.ContentHash,
		Scope: colony.SpecScope{
			Kind: colony.SpecScopeFeature, GoalID: draft.Revision.Scope.GoalID, SessionID: "session-real-repo-scoped",
			FeatureID: "feature-visible-planning", RequirementIDs: []string{requirementID}, AcceptanceCheckIDs: []string{draft.Revision.AcceptanceChecks[0].ID},
		},
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify, Section: specificationSectionRequirements, TargetID: requirementID,
			Item: specificationItemInput{Description: "The owner sees the revised causal planning loop.", EvidenceIDs: []string{"owner:scoped-revision"}},
		}},
		DecisionResolution: &decision, CreatedAt: time.Date(2026, time.September, 8, 2, 20, 0, 0, time.UTC),
	}
	planBefore := mustReadSpecificationTestState(t, root).Plan
	stateBefore := mustReadSpecificationTestStateBytes(t, root)
	revised, err := reviseSpecification(root, request, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	stateAfter := mustReadSpecificationTestState(t, root)
	if bytes.Equal(stateBefore, mustReadSpecificationTestStateBytes(t, root)) {
		t.Fatal("scoped revision did not create a successor lineage")
	}
	if !reflect.DeepEqual(planBefore, stateAfter.Plan) || stateAfter.Plan.Phases[0].Tasks[1].Status != colony.TaskCompleted {
		t.Fatalf("scoped revision invalidated independent completed work: before=%+v after=%+v", planBefore, stateAfter.Plan)
	}
	if !reflect.DeepEqual(revised.AffectedScope.TaskIDs, []string{"1.1"}) || !reflect.DeepEqual(revised.AffectedScope.ProofLinkIDs, []string{requirementID}) {
		t.Fatalf("affected closure = %+v, want only linked task/proof", revised.AffectedScope)
	}
	if revised.Revision.Status != colony.SpecStatusDraft {
		t.Fatalf("successor status = %s, want DRAFT", revised.Revision.Status)
	}
	_, authority, buildErr := resolveCodexBuildPlanAuthority(root, stateAfter)
	if buildErr == nil || authority.Eligible {
		t.Fatalf("unapproved successor did not block execution: %+v err=%v", authority, buildErr)
	}
	facts := specPlanSealGate200Facts(t)
	facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
	facts.Planning.Value.AffectedUnresolvedSemanticIDs = append([]string(nil), revised.AffectedScope.TaskIDs...)
	if preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner}); err == nil || preflight.Eligible {
		t.Fatalf("affected scope did not block verified seal: %+v err=%v", preflight, err)
	}
}
