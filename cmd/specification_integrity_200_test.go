package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSpecificationIntegrity200(t *testing.T) {
	saveGlobals(t)

	t.Run("canonical validator refuses copied hashes for every typed section", func(t *testing.T) {
		_, state := specificationIntegrity200ApprovedRepository(t)
		if state.Specification == nil {
			t.Fatal("approved fixture has no specification")
		}
		if err := validateCanonicalSpecificationState(*state.Specification); err != nil {
			t.Fatalf("valid approved specification rejected: %v", err)
		}

		cases := []struct {
			name   string
			field  string
			mutate func(*colony.SpecRevision)
		}{
			{name: "outcomes description", field: "outcomes", mutate: func(revision *colony.SpecRevision) { revision.Outcomes[0].Description += " forged" }},
			{name: "included behaviors description", field: "included_behaviors", mutate: func(revision *colony.SpecRevision) { revision.IncludedBehaviors[0].Description += " forged" }},
			{name: "exclusions description", field: "exclusions", mutate: func(revision *colony.SpecRevision) { revision.Exclusions[0].Description += " forged" }},
			{name: "binding decisions description", field: "binding_decisions", mutate: func(revision *colony.SpecRevision) { revision.BindingDecisions[0].Description += " forged" }},
			{name: "requirements description", field: "requirements", mutate: func(revision *colony.SpecRevision) { revision.Requirements[0].Description += " forged" }},
			{name: "acceptance checks description", field: "acceptance_checks", mutate: func(revision *colony.SpecRevision) { revision.AcceptanceChecks[0].Description += " forged" }},
			{name: "acceptance checks verification", field: "acceptance_checks", mutate: func(revision *colony.SpecRevision) { revision.AcceptanceChecks[0].Verification += " --forged" }},
			{name: "negative expectations description", field: "negative_expectations", mutate: func(revision *colony.SpecRevision) { revision.NegativeExpectations[0].Description += " forged" }},
			{name: "recovery expectations description", field: "recovery_expectations", mutate: func(revision *colony.SpecRevision) { revision.RecoveryExpectations[0].Description += " forged" }},
			{name: "affected public paths description", field: "affected_public_paths", mutate: func(revision *colony.SpecRevision) { revision.AffectedPublicPaths[0].Description += " forged" }},
			{name: "affected public paths path", field: "affected_public_paths", mutate: func(revision *colony.SpecRevision) { revision.AffectedPublicPaths[0].Path += "-forged" }},
			{name: "item evidence", field: "outcomes", mutate: func(revision *colony.SpecRevision) { revision.Outcomes[0].EvidenceIDs[0] += "-forged" }},
			{name: "item stable identity", field: "outcomes", mutate: func(revision *colony.SpecRevision) { revision.Outcomes[0].ID += "-forged" }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				tampered := specificationIntegrity200CloneSpecification(t, *state.Specification)
				tc.mutate(&tampered.Revisions[len(tampered.Revisions)-1])
				assertSpecificationIntegrity200Error(t, validateCanonicalSpecificationState(tampered), tc.field)
			})
		}
	})

	t.Run("canonical validator refuses revision scope delta timestamp and receipt forgery", func(t *testing.T) {
		_, state := specificationIntegrity200ApprovedRepository(t)
		approved := *state.Specification
		current := approved.Revisions[len(approved.Revisions)-1]

		cases := []struct {
			name   string
			field  string
			mutate func(*colony.Specification)
		}{
			{name: "specification lineage ID", field: "specification.id", mutate: func(specification *colony.Specification) {
				specification.ID += "-forged"
				for index := range specification.Revisions {
					specification.Revisions[index].SpecificationID = specification.ID
					if specification.Revisions[index].Approval != nil {
						specification.Revisions[index].Approval.SpecificationID = specification.ID
					}
				}
			}},
			{name: "scope", field: "content_hash", mutate: func(specification *colony.Specification) { specification.Revisions[0].Scope.SessionID += "-forged" }},
			{name: "created timestamp", field: "content_hash", mutate: func(specification *colony.Specification) {
				specification.Revisions[0].CreatedAt = specification.Revisions[0].CreatedAt.Add(time.Second)
			}},
			{name: "revision ID", field: "id", mutate: func(specification *colony.Specification) {
				revision := &specification.Revisions[0]
				revision.ID = "spec-revision-" + strings.Repeat("a", 12)
				specification.CurrentRevisionID = revision.ID
				revision.Approval.RevisionID = revision.ID
			}},
			{name: "revision content hash", field: "content_hash", mutate: func(specification *colony.Specification) {
				revision := &specification.Revisions[0]
				revision.ContentHash = strings.Repeat("b", 64)
				revision.ID = "spec-revision-" + revision.ContentHash[:12]
				specification.CurrentRevisionID = revision.ID
				revision.Approval.RevisionID = revision.ID
				revision.Approval.RevisionContentHash = revision.ContentHash
			}},
			{name: "approval receipt ID", field: "approval.id", mutate: func(specification *colony.Specification) { specification.Revisions[0].Approval.ID += "-forged" }},
			{name: "approval actor", field: "approval.id", mutate: func(specification *colony.Specification) { specification.Revisions[0].Approval.ApprovedBy += "-forged" }},
			{name: "approval timestamp", field: "approval.id", mutate: func(specification *colony.Specification) {
				specification.Revisions[0].Approval.ApprovedAt = specification.Revisions[0].Approval.ApprovedAt.Add(time.Second)
			}},
			{name: "approval token", field: "approval.approval_token_hash", mutate: func(specification *colony.Specification) {
				specification.Revisions[0].Approval.ApprovalTokenHash = strings.Repeat("c", 64)
			}},
			{name: "approval revision ID", field: "approval.revision_id", mutate: func(specification *colony.Specification) { specification.Revisions[0].Approval.RevisionID += "-forged" }},
			{name: "approval revision hash", field: "approval.revision_content_hash", mutate: func(specification *colony.Specification) {
				specification.Revisions[0].Approval.RevisionContentHash = strings.Repeat("d", 64)
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				tampered := specificationIntegrity200CloneSpecification(t, approved)
				tc.mutate(&tampered)
				assertSpecificationIntegrity200Error(t, validateCanonicalSpecificationState(tampered), tc.field)
			})
		}

		successor, _, _, err := buildSpecificationSuccessor(approved, state.Plan, specificationRevisionRequest{
			PredecessorRevisionID: current.ID, PredecessorContentHash: current.ContentHash,
			Scope: colony.SpecScope{
				Kind: colony.SpecScopeFeature, GoalID: current.Scope.GoalID, SessionID: current.Scope.SessionID,
				FeatureID: "integrity-successor", RequirementIDs: []string{current.Requirements[0].ID},
				AcceptanceCheckIDs: []string{current.AcceptanceChecks[0].ID},
			},
			Changes: []specificationRevisionChange{{
				Operation: specificationChangeModify, Section: specificationSectionRequirements, TargetID: current.Requirements[0].ID,
				Item: specificationItemInput{Description: "The canonical successor remains owner-approved separately.", EvidenceIDs: []string{"test:integrity-successor"}},
			}},
			CreatedAt: current.CreatedAt.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("build valid successor: %v", err)
		}
		if err := validateCanonicalSpecificationState(successor); err != nil {
			t.Fatalf("valid successor rejected: %v", err)
		}
		for _, tc := range []struct {
			name   string
			field  string
			mutate func(*colony.SpecRevision)
		}{
			{name: "predecessor", field: "predecessor_id", mutate: func(revision *colony.SpecRevision) {
				revision.PredecessorID = "spec-revision-forged"
				revision.Delta.PredecessorRevisionID = revision.PredecessorID
			}},
			{name: "classified delta", field: "delta", mutate: func(revision *colony.SpecRevision) { revision.Delta.Requirements.ModifiedIDs[0] += "-forged" }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				tampered := specificationIntegrity200CloneSpecification(t, successor)
				tc.mutate(&tampered.Revisions[len(tampered.Revisions)-1])
				assertSpecificationIntegrity200Error(t, validateCanonicalSpecificationState(tampered), tc.field)
			})
		}
	})

	t.Run("draft successor and approval retain separate owner authorities", func(t *testing.T) {
		root, state := specificationIntegrity200ApprovedRepository(t)
		beforePlan := state.Plan
		current, ok := currentSpecificationRevision(*state.Specification)
		if !ok {
			t.Fatal("approved current revision is missing")
		}
		revised, err := reviseSpecification(root, specificationRevisionRequest{
			PredecessorRevisionID: current.ID, PredecessorContentHash: current.ContentHash,
			Scope: colony.SpecScope{
				Kind: colony.SpecScopeFeature, GoalID: current.Scope.GoalID, SessionID: current.Scope.SessionID,
				FeatureID: "integrity-lifecycle", RequirementIDs: []string{current.Requirements[0].ID},
				AcceptanceCheckIDs: []string{current.AcceptanceChecks[0].ID},
			},
			Changes: []specificationRevisionChange{{
				Operation: specificationChangeModify, Section: specificationSectionRequirements, TargetID: current.Requirements[0].ID,
				Item: specificationItemInput{Description: "The revised owner contract is canonical and explicit.", EvidenceIDs: []string{"test:integrity-lifecycle"}},
			}},
			CreatedAt: current.CreatedAt.Add(2 * time.Hour),
		}, specificationMutationOptions{})
		if err != nil {
			t.Fatalf("create scoped successor: %v", err)
		}
		if revised.Revision.Status != colony.SpecStatusDraft || revised.Revision.Approval != nil {
			t.Fatalf("successor authority = %s/%#v, want draft without approval", revised.Revision.Status, revised.Revision.Approval)
		}
		approved, err := approveSpecification(root, specificationApprovalRequest{
			RevisionID: revised.Revision.ID, RevisionContentHash: revised.Revision.ContentHash,
			ApprovalToken: specificationApprovalToken(revised.Specification.ID, revised.Revision.ID, revised.Revision.ContentHash),
			ApprovedBy:    "owner:integrity-200", ApprovedAt: revised.Revision.CreatedAt.Add(time.Minute),
		}, specificationMutationOptions{})
		if err != nil {
			t.Fatalf("approve scoped successor: %v", err)
		}
		if err := validateCanonicalSpecificationState(approved.Specification); err != nil {
			t.Fatalf("approved successor rejected: %v", err)
		}
		after := mustReadSpecificationTestState(t, root)
		if !reflect.DeepEqual(beforePlan, after.Plan) {
			t.Fatal("specification approval also changed plan acceptance authority")
		}
		if len(approved.Specification.Revisions) != 2 || approved.Specification.Revisions[0].Status != colony.SpecStatusSuperseded || approved.Revision.Status != colony.SpecStatusApproved {
			t.Fatalf("revision lifecycle = %#v", approved.Specification.Revisions)
		}
	})

	t.Run("planning refuses body tamper even with a matching forged projection", func(t *testing.T) {
		root, state := specificationIntegrity200ApprovedRepository(t)
		original := state.Specification.Revisions[0].Requirements[0].Description
		forged := original + " forged"
		state.Specification.Revisions[0].Requirements[0].Description = forged
		specificationIntegrity200WriteState(t, root, state)
		projectionPath := filepath.Join(root, specificationProjectionRelativePath)
		projection, err := os.ReadFile(projectionPath)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(projection), original) {
			t.Fatalf("projection does not contain original requirement %q", original)
		}
		projection = []byte(strings.Replace(string(projection), original, forged, 1))
		if err := os.WriteFile(projectionPath, projection, 0o644); err != nil {
			t.Fatal(err)
		}
		before := planningRealRepo200Snapshot(t, root)
		if _, err := requireApprovedPlanningSpecification(root, state); err == nil || !strings.Contains(err.Error(), "requirements") {
			t.Fatalf("planning tamper error = %v", err)
		}
		assertSpecificationIntegrity200Snapshot(t, root, before)
	})

	t.Run("projection mismatch refuses planning without mutation", func(t *testing.T) {
		root, state := specificationIntegrity200ApprovedRepository(t)
		projectionPath := filepath.Join(root, specificationProjectionRelativePath)
		projection, err := os.ReadFile(projectionPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(projectionPath, append(projection, []byte("\nforged projection\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		before := planningRealRepo200Snapshot(t, root)
		if _, err := requireApprovedPlanningSpecification(root, state); err == nil || !strings.Contains(err.Error(), "projection") {
			t.Fatalf("projection tamper error = %v", err)
		}
		assertSpecificationIntegrity200Snapshot(t, root, before)
	})

	t.Run("candidate acceptance refuses tampered approval without mutation", func(t *testing.T) {
		root, _ := specificationIntegrity200ApprovedRepository(t)
		candidate := stagePlanningCandidate200(t, root)
		state := mustReadSpecificationTestState(t, root)
		state.Specification.Revisions[len(state.Specification.Revisions)-1].AcceptanceChecks[0].Verification += " forged"
		specificationIntegrity200WriteState(t, root, state)
		before := planningRealRepo200Snapshot(t, root)
		if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner:integrity-200"}); err == nil || !strings.Contains(err.Error(), "acceptance_checks") {
			t.Fatalf("candidate acceptance tamper error = %v", err)
		}
		assertSpecificationIntegrity200Snapshot(t, root, before)
	})

	t.Run("build and run refuse tampered approval without mutation", func(t *testing.T) {
		root, _ := specificationIntegrity200ApprovedRepository(t)
		acceptStagedPlanningCandidate200(t, root)
		state := mustReadSpecificationTestState(t, root)
		state.Specification.Revisions[len(state.Specification.Revisions)-1].NegativeExpectations[0].Description += " forged"
		specificationIntegrity200WriteState(t, root, state)
		before := planningRealRepo200Snapshot(t, root)

		_, buildDecision, buildErr := resolveCodexBuildPlanAuthority(root, state)
		if buildErr == nil || buildDecision.Eligible || !strings.Contains(buildDecision.Diagnostic, "negative_expectations") {
			t.Fatalf("build tamper decision = %+v err=%v", buildDecision, buildErr)
		}
		facts := lifecycleFactsFromStateSnapshot(state, false, time.Date(2026, time.September, 8, 18, 0, 0, 0, time.UTC))
		facts.Root = root
		run := buildAutopilotPreflight(facts)
		if run.Valid || run.PlanAuthority.Eligible || !strings.Contains(run.PlanAuthority.Diagnostic, "negative_expectations") {
			t.Fatalf("run tamper preflight = %+v", run)
		}
		assertSpecificationIntegrity200Snapshot(t, root, before)
	})

	t.Run("legacy specification-less plan remains explicitly compatible", func(t *testing.T) {
		taskID := "1.1"
		state := colony.ColonyState{CurrentPhase: 1, Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases:           []colony.Phase{{ID: 1, Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &taskID, Status: colony.TaskPending}}}},
		}}
		facts := lifecycleFactsFromStateSnapshot(state, false, time.Date(2026, time.September, 8, 18, 0, 0, 0, time.UTC))
		decision := validateAcceptedPlanAuthority(facts, planAuthorityVerifiedBindings{})
		if !decision.Eligible || decision.Classification != planAuthorityLegacyUnbound || state.Specification != nil {
			t.Fatalf("legacy compatibility decision = %+v", decision)
		}
	})
}

func specificationIntegrity200ApprovedRepository(t *testing.T) (string, colony.ColonyState) {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("initialize integrity repository: %v\n%s", err, output)
	}
	seedSettledDiscussState(t, dataDir, "specification-integrity-200")
	if _, err := runDiscuss(root, 3, false); err != nil {
		t.Fatalf("create settled draft: %v", err)
	}
	state := mustReadSpecificationTestState(t, root)
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("settled draft has no current revision")
	}
	if _, err := approveSpecification(root, specificationApprovalRequest{
		RevisionID: current.ID, RevisionContentHash: current.ContentHash,
		ApprovalToken: specificationApprovalToken(state.Specification.ID, current.ID, current.ContentHash),
		ApprovedBy:    "owner:integrity-200", ApprovedAt: current.CreatedAt.Add(time.Minute),
	}, specificationMutationOptions{}); err != nil {
		t.Fatalf("approve integrity specification: %v", err)
	}
	return root, mustReadSpecificationTestState(t, root)
}

func specificationIntegrity200CloneSpecification(t *testing.T, specification colony.Specification) colony.Specification {
	t.Helper()
	cloned, err := cloneSpecification(specification)
	if err != nil {
		t.Fatal(err)
	}
	return cloned
}

func specificationIntegrity200WriteState(t *testing.T, root string, state colony.ColonyState) {
	t.Helper()
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertSpecificationIntegrity200Error(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected canonical specification error containing %q", field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Fatalf("canonical specification error %q does not identify %q", err, field)
	}
}

func assertSpecificationIntegrity200Snapshot(t *testing.T, root string, before []planningRealRepo200File) {
	t.Helper()
	after := planningRealRepo200Snapshot(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("refusal mutated repository artifacts:\nbefore: %#v\nafter:  %#v", before, after)
	}
}
