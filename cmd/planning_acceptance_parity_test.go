package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// A plan is checked twice: when the Route-Setter hands it in (drafting), and
// again when the owner accepts it -- the same check that runs on every state
// load and before every build. These tests hold the two to one rule: anything
// drafting admits, acceptance accepts. They exist because the two disagreed
// about public-path proof, and Go handed an owner a reviewed plan it could
// never accept (CosmicDashboard, plan-candidate-dbf1e0e37a9e, 2026-09-12).

// TestBehindTheScenesStepPlanIsAcceptable is the downstream failure in its
// real shape: a user-facing phase whose plan also carries a housekeeping task
// that touches no public path. Drafting admits it; acceptance and build must
// too.
func TestBehindTheScenesStepPlanIsAcceptable(t *testing.T) {
	saveGlobals(t)
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	planningParityAddHousekeepingTask(&result)

	candidate, err := planningParityDraft(t, root, manifest, result)
	if err != nil {
		t.Fatalf("drafting refused a plan with a behind-the-scenes step: %v", err)
	}
	if err := planningParityAccept(root, candidate); err != nil {
		t.Fatalf("drafting admitted this plan, so acceptance must accept it: %v", err)
	}
	planningParityAssertBuildable(t, root)
}

// TestPlanProvingNoPublicPathIsRefusedAtDrafting covers the plan-wide half of
// the rule. Every approved specification names at least one affected public
// path, so a plan proving none of them can never be accepted; drafting must
// say so to the Route-Setter instead of handing the owner a dead candidate.
func TestPlanProvingNoPublicPathIsRefusedAtDrafting(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	result.Proposal.UserFacingSemanticIDs = nil
	for index := range result.Proposal.TaskDeclarations {
		result.Proposal.TaskDeclarations[index].UserFacing = false
	}
	for phaseIndex := range result.Proposal.Phases {
		phase := &result.Proposal.Phases[phaseIndex]
		phase.PublicPathProofLinks = nil
		for taskIndex := range phase.Tasks {
			phase.Tasks[taskIndex].PublicPathProofLinks = nil
		}
	}

	before := planCandidateTestSnapshot(t, root)
	_, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err == nil || !strings.Contains(err.Error(), "public_path_proof_links") {
		t.Fatalf("drafting error = %v, want a refusal naming public_path_proof_links", err)
	}
	planCandidateTestAssertSnapshot(t, root, before)
	planningParityAssertStillWaiting(t, root, manifest)
}

// TestEveryDraftedPlanIsAcceptable is the invariant. It drops each proof kind
// from a phase or a task, with that node declared user-facing or not, and
// requires that every variant drafting admits is one acceptance accepts. It
// fails whenever the two checks drift apart again, whatever the rule is called.
func TestEveryDraftedPlanIsAcceptable(t *testing.T) {
	// Asking whether drafting admits a variant writes nothing, so one fixture
	// answers it for every variant; only admitted variants need a run of their
	// own, through the real submission and the owner's exact acceptance.
	sharedRoot, sharedManifest, sharedResult := planningRouteStageTestFixture(t)
	admitted, refused := 0, 0
	for _, kind := range planningProofLinkKinds() {
		for _, node := range []string{"phase", "task"} {
			for _, userFacing := range []bool{true, false} {
				variant := planningRouteStageCloneResult(t, sharedResult)
				planningParityDropLinks(t, &variant, kind, node, userFacing)
				if _, err := validatePlanningRouteStageResult(sharedRoot, sharedManifest, planningRouteStageTestBytes(t, variant)); err != nil {
					refused++
					continue
				}
				admitted++
				t.Run(fmt.Sprintf("%s/%s/user_facing=%t", kind, node, userFacing), func(t *testing.T) {
					root, manifest, result := planningRouteStageTestFixture(t)
					planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
					planningParityDropLinks(t, &result, kind, node, userFacing)
					candidate, err := planningParityDraft(t, root, manifest, result)
					if err != nil {
						t.Fatalf("drafting admitted this variant when asked, then refused it on submission: %v", err)
					}
					if err := planningParityAccept(root, candidate); err != nil {
						t.Fatalf("drafting admitted a plan whose %s has no %s (user-facing=%t), but acceptance refused it: %v", node, planningProofField(kind), userFacing, err)
					}
				})
			}
		}
	}
	if admitted == 0 || refused == 0 {
		t.Fatalf("variants admitted=%d refused=%d; the invariant proves nothing unless drafting both admits and refuses some", admitted, refused)
	}
}

// planningParityDropLinks removes one proof kind from the fixture's phase or
// task, first withdrawing that node's user-facing declaration when asked.
func planningParityDropLinks(t *testing.T, result *planningRouteStageResult, kind, node string, userFacing bool) {
	t.Helper()
	phase := &result.Proposal.Phases[0]
	task := &phase.Tasks[0]
	semanticID := phase.SemanticID
	links := planningParityLinks(t, phase, nil, kind)
	if node == "task" {
		semanticID = task.SemanticID
		links = planningParityLinks(t, nil, task, kind)
	}
	if !userFacing {
		planningParityUnmarkUserFacing(&result.Proposal, semanticID)
	}
	*links = nil
}

// TestUnacceptablePlanNeverReachesTheOwner proves the rehearsal is wired where
// it matters: when acceptance would refuse the plan, drafting refuses the
// Route-Setter's submission before anything commits, so the owner is never
// shown a plan they cannot accept and the run keeps waiting for a corrected
// submission.
func TestUnacceptablePlanNeverReachesTheOwner(t *testing.T) {
	root, manifest, result := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	refusal := errors.New("rehearsed acceptance refused the plan")
	original := planningRouteAcceptanceRehearsal
	planningRouteAcceptanceRehearsal = func(colony.Plan, *colony.Specification, planningRoutePlanProposal) error { return refusal }
	t.Cleanup(func() { planningRouteAcceptanceRehearsal = original })

	before := planCandidateTestSnapshot(t, root)
	_, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err == nil || !strings.Contains(err.Error(), refusal.Error()) {
		t.Fatalf("drafting error = %v, want the rehearsal's refusal", err)
	}
	planCandidateTestAssertSnapshot(t, root, before)
	planningParityAssertStillWaiting(t, root, manifest)
}

// planningParityAddHousekeepingTask appends a behind-the-scenes task in the
// shape a real Route-Setter sends: the four proof kinds every step carries,
// copied from the fixture's own specification-backed links, no public-path
// link, and no user-facing declaration.
func planningParityAddHousekeepingTask(result *planningRouteStageResult) {
	phase := &result.Proposal.Phases[0]
	source := phase.Tasks[0]
	taskID := "1.2"
	criterion := "Staging folders are excluded from version control"
	phase.Tasks = append(phase.Tasks, colony.Task{
		ID: &taskID, SemanticID: "task-route-housekeeping", Goal: "Exclude staging folders from version control",
		RequirementProofLinks: append([]string(nil), source.RequirementProofLinks...),
		AcceptanceProofLinks:  append([]string(nil), source.AcceptanceProofLinks...),
		NegativeProofLinks:    append([]string(nil), source.NegativeProofLinks...),
		RecoveryProofLinks:    append([]string(nil), source.RecoveryProofLinks...),
		SuccessCriteria:       []string{criterion},
		EvidenceRequirements:  []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}},
	})
	result.Proposal.TaskDeclarations = append(result.Proposal.TaskDeclarations, planningRouteTaskDeclaration{
		TaskSemanticID: "task-route-housekeeping", Files: []string{".gitignore"},
	})
}

func planningParityUnmarkUserFacing(proposal *planningRoutePlanProposal, semanticID string) {
	kept := proposal.UserFacingSemanticIDs[:0]
	for _, id := range proposal.UserFacingSemanticIDs {
		if id != semanticID {
			kept = append(kept, id)
		}
	}
	proposal.UserFacingSemanticIDs = kept
	for index := range proposal.TaskDeclarations {
		if proposal.TaskDeclarations[index].TaskSemanticID == semanticID {
			proposal.TaskDeclarations[index].UserFacing = false
		}
	}
}

func planningParityLinks(t *testing.T, phase *colony.Phase, task *colony.Task, kind string) *[]string {
	t.Helper()
	if task != nil {
		switch kind {
		case planningSemanticRequirement:
			return &task.RequirementProofLinks
		case planningSemanticAcceptance:
			return &task.AcceptanceProofLinks
		case planningSemanticNegative:
			return &task.NegativeProofLinks
		case planningSemanticRecovery:
			return &task.RecoveryProofLinks
		case planningSemanticPublicPath:
			return &task.PublicPathProofLinks
		}
	} else {
		switch kind {
		case planningSemanticRequirement:
			return &phase.RequirementProofLinks
		case planningSemanticAcceptance:
			return &phase.AcceptanceProofLinks
		case planningSemanticNegative:
			return &phase.NegativeProofLinks
		case planningSemanticRecovery:
			return &phase.RecoveryProofLinks
		case planningSemanticPublicPath:
			return &phase.PublicPathProofLinks
		}
	}
	t.Fatalf("no proof-link field for kind %q; add it here so the invariant covers it", kind)
	return nil
}

// planningParityDraft runs a Route-Setter result through the real drafting
// path. It returns drafting's refusal, or the candidate exactly as persisted.
func planningParityDraft(t *testing.T, root string, manifest planningStageManifest, result planningRouteStageResult) (colony.PlanCandidate, error) {
	t.Helper()
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		return colony.PlanCandidate{}, err
	}
	if coordinated.Candidate == nil {
		t.Fatalf("drafting admitted the proposal but produced no candidate: %+v", coordinated)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(manifest.RunID))))
	if err != nil {
		t.Fatal(err)
	}
	var persisted colony.PlanCandidate
	if err := json.Unmarshal(content, &persisted); err != nil {
		t.Fatal(err)
	}
	return persisted, nil
}

// planningParityAccept runs the owner's exact acceptance.
func planningParityAccept(root string, candidate colony.PlanCandidate) error {
	_, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner", AcceptedAt: candidate.CreatedAt.Add(time.Minute),
	})
	return err
}

// planningParityAssertStillWaiting proves a refused submission left the run
// waiting for the same Route-Setter, rather than advancing it.
func planningParityAssertStillWaiting(t *testing.T, root string, manifest planningStageManifest) {
	t.Helper()
	state, err := loadPlanningStageState(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Stage != planningStageRouteRunning || state.ActiveManifestID != manifest.ID {
		t.Fatalf("refused Route-Setter result moved the run to %+v, want it still waiting for the Route-Setter", state)
	}
}

// planningParityAssertBuildable asks the build gate itself, which re-runs the
// accepted-plan check before any work starts. Callers must saveGlobals first.
func planningParityAssertBuildable(t *testing.T, root string) {
	t.Helper()
	var err error
	store, err = storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatal(err)
	}
	state := mustReadSpecificationTestState(t, root)
	goal := "Execute the exactly accepted plan"
	state.Goal = &goal
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	if _, decision, err := resolveCodexBuildPlanAuthority(root, state); err != nil || !decision.Eligible {
		t.Fatalf("build authority = %+v error=%v, want the accepted plan buildable", decision, err)
	}
}
