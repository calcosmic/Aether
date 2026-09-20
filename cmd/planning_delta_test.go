package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningDeltaCompareIgnoresFormattingAndOrdering(t *testing.T) {
	beforeSource := planningDeltaTestSource()
	afterSource := planningDeltaTestSource()

	core := afterSource.Revision.Phases[0]
	ui := afterSource.Revision.Phases[1]
	core.ID = 2
	core.Name = "  Core   engine  "
	core.Description = "Implement\n the   planning engine"
	core.Tasks[0].ID = planningDeltaTestStringPtr("2.1")
	core.Tasks[0].Goal = "Implement   the semantic\n comparator"
	ui.ID = 1
	ui.Tasks[0].ID = planningDeltaTestStringPtr("1.1")
	ui.Tasks[0].DependsOn = []string{"2.1"}
	afterSource.Revision.Phases = []colony.Phase{ui, core}
	afterSource.Plan.Phases = append([]colony.Phase(nil), afterSource.Revision.Phases...)

	specification := *afterSource.Specification
	specification.Revisions = append([]colony.SpecRevision(nil), specification.Revisions...)
	specRevision := specification.Revisions[0]
	specRevision.Requirements = append([]colony.SpecRequirement(nil), specRevision.Requirements...)
	specRevision.Requirements[0].Description = "  Compute   semantic\n changes "
	specification.Revisions[0] = specRevision
	afterSource.Specification = &specification

	before, err := buildPlanningSemanticSnapshot(beforeSource)
	if err != nil {
		t.Fatalf("build before snapshot: %v", err)
	}
	after, err := buildPlanningSemanticSnapshot(afterSource)
	if err != nil {
		t.Fatalf("build reordered snapshot: %v", err)
	}
	if before.ContentHash != after.ContentHash {
		t.Fatalf("formatting/order changed semantic snapshot hash: %s != %s", before.ContentHash, after.ContentHash)
	}

	delta, err := comparePlanningSemanticSnapshots(before, after)
	if err != nil {
		t.Fatalf("compare snapshots: %v", err)
	}
	assertPlanningDeltaNoExecutableChange(t, delta)
	if len(delta.AuthorityImpacts) != 0 {
		t.Fatalf("formatting/order produced authority impacts: %#v", delta.AuthorityImpacts)
	}
}

func TestPlanningDeltaCompareRecoveryInstructionIsOneModification(t *testing.T) {
	beforeSource := planningDeltaTestSource()
	afterSource := planningDeltaTestSource()
	specification := *afterSource.Specification
	specification.Revisions = append([]colony.SpecRevision(nil), specification.Revisions...)
	specRevision := specification.Revisions[0]
	specRevision.RecoveryExpectations = append([]colony.SpecRecoveryExpectation(nil), specRevision.RecoveryExpectations...)
	specRevision.RecoveryExpectations[0].Description = "Resume only after replaying the exact validated receipt"
	specification.Revisions[0] = specRevision
	afterSource.Specification = &specification

	before := mustPlanningDeltaTestSnapshot(t, beforeSource)
	after := mustPlanningDeltaTestSnapshot(t, afterSource)
	delta, err := comparePlanningSemanticSnapshots(before, after)
	if err != nil {
		t.Fatalf("compare snapshots: %v", err)
	}

	assertPlanningDeltaOnlyKind(t, delta.RecoveryExpectations, colony.PlanningSemanticChangeModified, 1)
	for _, section := range [][]colony.PlanningSemanticChange{
		delta.Phases,
		delta.Tasks,
		delta.Dependencies,
		delta.RequirementLinks,
		delta.AcceptanceChecks,
		delta.NegativeExpectations,
		delta.PublicPaths,
	} {
		assertPlanningDeltaOnlyKind(t, section, colony.PlanningSemanticChangePreserved, len(section))
	}
}

func TestPlanningDeltaCompareMakesDependencyAndNegativeRemovalVisible(t *testing.T) {
	beforeSource := planningDeltaTestSource()
	afterSource := planningDeltaTestSource()
	afterSource.Revision.Phases[1].Tasks[0].DependsOn = nil
	afterSource.Revision.Phases[1].Tasks[0].NegativeProofLinks = nil
	afterSource.Plan.Phases = append([]colony.Phase(nil), afterSource.Revision.Phases...)

	delta, err := comparePlanningSemanticSnapshots(
		mustPlanningDeltaTestSnapshot(t, beforeSource),
		mustPlanningDeltaTestSnapshot(t, afterSource),
	)
	if err != nil {
		t.Fatalf("compare snapshots: %v", err)
	}
	assertPlanningDeltaOnlyKind(t, delta.Dependencies, colony.PlanningSemanticChangeRemoved, 1)
	assertPlanningDeltaOnlyKind(t, delta.NegativeExpectations, colony.PlanningSemanticChangeRemoved, 1)
}

func TestPlanningDeltaCompareRejectsMissingOrDuplicateStableIDs(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*planningSemanticSnapshotSource)
		wantErr string
	}{
		{
			name: "missing phase",
			mutate: func(source *planningSemanticSnapshotSource) {
				source.Revision.Phases[0].SemanticID = ""
			},
			wantErr: "phases[0].semantic_id is required",
		},
		{
			name: "duplicate task",
			mutate: func(source *planningSemanticSnapshotSource) {
				source.Revision.Phases[1].Tasks[0].SemanticID = source.Revision.Phases[0].Tasks[0].SemanticID
			},
			wantErr: "phases[1].tasks[0].semantic_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := planningDeltaTestSource()
			tt.mutate(&source)
			_, err := buildPlanningSemanticSnapshot(source)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("build error = %v, want field path %q", err, tt.wantErr)
			}
		})
	}
}

func TestPlanningDeltaAuthorityApprovalDoesNotChangeSemantics(t *testing.T) {
	beforeSource := planningDeltaTestSource()
	afterSource := planningDeltaTestSource()
	specification := *afterSource.Specification
	specification.Revisions = append([]colony.SpecRevision(nil), specification.Revisions...)
	specRevision := specification.Revisions[0]
	specRevision.Status = colony.SpecStatusApproved
	specRevision.Approval = &colony.SpecApprovalReceipt{
		SchemaVersion:       colony.SpecificationSchemaVersion,
		ID:                  "spec-approval-1",
		SpecificationID:     specification.ID,
		RevisionID:          specRevision.ID,
		RevisionContentHash: specRevision.ContentHash,
		ApprovalTokenHash:   planningDeltaTestHash("approval-token"),
		ApprovedBy:          "owner",
		ApprovedAt:          time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC),
	}
	specification.Revisions[0] = specRevision
	afterSource.Specification = &specification

	before := mustPlanningDeltaTestSnapshot(t, beforeSource)
	after := mustPlanningDeltaTestSnapshot(t, afterSource)
	if before.ContentHash != after.ContentHash {
		t.Fatalf("approval changed semantic snapshot hash: %s != %s", before.ContentHash, after.ContentHash)
	}
	delta, err := comparePlanningSemanticSnapshots(before, after)
	if err != nil {
		t.Fatalf("compare snapshots: %v", err)
	}
	assertPlanningDeltaNoExecutableChange(t, delta)
	if len(delta.AuthorityImpacts) != 1 || delta.AuthorityImpacts[0].Kind != colony.PlanningAuthoritySpecApproval {
		t.Fatalf("authority impacts = %#v, want one specification approval", delta.AuthorityImpacts)
	}
}

func TestPlanningDeltaAuthorityCandidateAcceptanceStaysSeparate(t *testing.T) {
	beforeSource := planningDeltaTestSource()
	candidateHash := planningDeltaTestHash("candidate")
	beforeSource.Plan.Candidates = []colony.PlanCandidate{{
		ID: "candidate-1", ContentHash: candidateHash, Status: colony.PlanCandidatePendingReview,
	}}
	afterSource := beforeSource
	afterSource.Plan.Candidates = append([]colony.PlanCandidate(nil), beforeSource.Plan.Candidates...)
	acceptance := &colony.PlanAcceptanceReceipt{
		ID: "plan-acceptance-1", ContentHash: planningDeltaTestHash("acceptance"),
		CandidateID: "candidate-1", CandidateContentHash: candidateHash, AcceptedBy: "owner",
	}
	afterSource.Plan.Candidates[0].Status = colony.PlanCandidateAccepted
	afterSource.Plan.Candidates[0].Acceptance = acceptance

	before := mustPlanningDeltaTestSnapshot(t, beforeSource)
	after := mustPlanningDeltaTestSnapshot(t, afterSource)
	if before.ContentHash != after.ContentHash {
		t.Fatalf("candidate acceptance changed semantic snapshot hash: %s != %s", before.ContentHash, after.ContentHash)
	}
	delta, err := comparePlanningSemanticSnapshots(before, after)
	if err != nil {
		t.Fatalf("compare snapshots: %v", err)
	}
	assertPlanningDeltaNoExecutableChange(t, delta)
	wantKinds := map[colony.PlanningAuthorityImpactKind]bool{
		colony.PlanningAuthorityCandidateStatus: false,
		colony.PlanningAuthorityPlanAcceptance:  false,
	}
	for _, impact := range delta.AuthorityImpacts {
		if _, ok := wantKinds[impact.Kind]; ok {
			wantKinds[impact.Kind] = true
		}
	}
	for kind, found := range wantKinds {
		if !found {
			t.Fatalf("authority impacts %#v do not contain %s", delta.AuthorityImpacts, kind)
		}
	}
}

func TestPlanningDeltaProposalCompleteProducesStableHash(t *testing.T) {
	proposal := planningDeltaCompleteProposal()
	snapshot, err := validatePlanProposalContract(proposal, nil)
	if err != nil {
		t.Fatalf("validate complete proposal: %v", err)
	}
	if len(snapshot.ContentHash) != 64 {
		t.Fatalf("snapshot hash = %q, want a SHA-256 digest", snapshot.ContentHash)
	}

	reordered := planningDeltaCompleteProposal()
	reordered.TaskDeclarations[0].Files = []string{" cmd/planning_delta_test.go ", "cmd/planning_delta.go"}
	reordered.TaskDeclarations[0], reordered.TaskDeclarations[1] = reordered.TaskDeclarations[1], reordered.TaskDeclarations[0]
	reordered.Revision.Phases[0].SuccessCriteria[0] = "  Semantic   changes are stable "
	reordered.Revision.Phases[0].EvidenceRequirements[0].Criterion = "Semantic changes are   stable"
	reordered.Plan.Phases = append([]colony.Phase(nil), reordered.Revision.Phases...)
	reorderedSnapshot, err := validatePlanProposalContract(reordered, nil)
	if err != nil {
		t.Fatalf("validate reordered proposal: %v", err)
	}
	if snapshot.ContentHash != reorderedSnapshot.ContentHash {
		t.Fatalf("normalized proposal hash changed: %s != %s", snapshot.ContentHash, reorderedSnapshot.ContentHash)
	}
}

func TestPlanningDeltaProposalRejectsIncompleteFieldsAtExactPaths(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*planProposalContract)
		wantErr string
	}{
		{
			name: "task objective",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[0].Tasks[0].Goal = "  "
			},
			wantErr: "phases[0].tasks[0].goal",
		},
		{
			name: "requirements",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[0].Tasks[0].RequirementProofLinks = nil
			},
			wantErr: "phases[0].tasks[0].requirement_proof_links",
		},
		{
			name: "automated acceptance",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[0].Tasks[0].EvidenceRequirements[0].Checks = nil
			},
			wantErr: "phases[0].tasks[0].evidence_requirements[0].checks",
		},
		{
			name: "negative check",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[0].Tasks[0].NegativeProofLinks = nil
			},
			wantErr: "phases[0].tasks[0].negative_proof_links",
		},
		{
			name: "recovery",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[0].Tasks[0].RecoveryProofLinks = nil
			},
			wantErr: "phases[0].tasks[0].recovery_proof_links",
		},
		{
			name: "dependency target",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[1].Tasks[0].DependsOn = []string{"9.9"}
			},
			wantErr: "phases[1].tasks[0].depends_on[0]",
		},
		{
			name: "files or reason",
			mutate: func(proposal *planProposalContract) {
				proposal.TaskDeclarations[0].Files = nil
				proposal.TaskDeclarations[0].NoFileReason = ""
			},
			wantErr: "phases[0].tasks[0].files",
		},
		{
			name: "public path",
			mutate: func(proposal *planProposalContract) {
				proposal.Revision.Phases[1].Tasks[0].PublicPathProofLinks = nil
			},
			wantErr: "phases[1].tasks[0].public_path_proof_links",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := planningDeltaCompleteProposal()
			tt.mutate(&proposal)
			_, err := validatePlanProposalContract(proposal, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validation error = %v, want field path %q", err, tt.wantErr)
			}
		})
	}
}

func TestPlanningDeltaProposalRejectsInexactFileDeclarations(t *testing.T) {
	for _, file := range []string{"/tmp/absolute.go", "cmd/*.go", "../outside.go", "cmd/"} {
		t.Run(file, func(t *testing.T) {
			proposal := planningDeltaCompleteProposal()
			proposal.TaskDeclarations[0].Files = []string{file}
			_, err := validatePlanProposalContract(proposal, nil)
			if err == nil || !strings.Contains(err.Error(), "phases[0].tasks[0].files[0]") {
				t.Fatalf("validation error = %v, want exact files[0] path", err)
			}
		})
	}
}

func TestPlanningDeltaRemovalRequiresExplicitDeclaration(t *testing.T) {
	predecessorProposal := planningDeltaCompleteProposal()
	predecessor, err := validatePlanProposalContract(predecessorProposal, nil)
	if err != nil {
		t.Fatalf("validate predecessor: %v", err)
	}

	proposal := planningDeltaCompleteProposal()
	proposal.Revision.Phases = proposal.Revision.Phases[:1]
	proposal.Plan.Phases = append([]colony.Phase(nil), proposal.Revision.Phases...)
	proposal.TaskDeclarations = proposal.TaskDeclarations[:1]
	proposal.UserFacingSemanticIDs = nil
	_, err = validatePlanProposalContract(proposal, &predecessor)
	if err == nil || !strings.Contains(err.Error(), "removals") || !strings.Contains(err.Error(), "phase-ui") {
		t.Fatalf("silent removal error = %v, want removals field and missing stable ID", err)
	}

	proposal.Removals = []planProposalRemoval{
		{SemanticID: "phase-ui", Classification: colony.PlanningSemanticChangeRemoved, Rationale: "The presentation work moved to the existing route"},
		{SemanticID: "task-present", Classification: colony.PlanningSemanticChangeRemoved, Rationale: "The existing route now owns presentation"},
	}
	if _, err := validatePlanProposalContract(proposal, &predecessor); err != nil {
		t.Fatalf("validate explicit removal proposal: %v", err)
	}
}

func TestPlanningDeltaCycleRejectsProposalBeforePersistence(t *testing.T) {
	proposal := planningDeltaCompleteProposal()
	proposal.Revision.Phases[0].Tasks[0].DependsOn = []string{"2.1"}
	_, err := validatePlanProposalContract(proposal, nil)
	if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("cycle validation error = %v, want dependency cycle", err)
	}
}

func TestPlanningDeltaProposalLegacyLoadingRemainsOutsideBoundary(t *testing.T) {
	legacyTaskID := "1.1"
	legacy := planningSemanticSnapshotSource{
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Legacy phase", Tasks: []colony.Task{{ID: &legacyTaskID, Goal: "Legacy task"}},
		}}},
	}
	if _, err := buildPlanningSemanticSnapshot(legacy); err != nil {
		t.Fatalf("legacy snapshot should remain readable without current proposal validation: %v", err)
	}
}

func planningDeltaCompleteProposal() planProposalContract {
	source := planningDeltaTestSource()
	for phaseIndex := range source.Revision.Phases {
		phase := &source.Revision.Phases[phaseIndex]
		phase.RequirementProofLinks = append([]string(nil), phase.Tasks[0].RequirementProofLinks...)
		phase.AcceptanceProofLinks = append([]string(nil), phase.Tasks[0].AcceptanceProofLinks...)
		phase.NegativeProofLinks = append([]string(nil), phase.Tasks[0].NegativeProofLinks...)
		phase.RecoveryProofLinks = append([]string(nil), phase.Tasks[0].RecoveryProofLinks...)
		phase.PublicPathProofLinks = append([]string(nil), phase.Tasks[0].PublicPathProofLinks...)
		criterion := "Semantic changes are stable"
		if phaseIndex == 1 {
			criterion = "The delta card remains visible"
		}
		phase.SuccessCriteria = []string{criterion}
		phase.EvidenceRequirements = []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}}
		task := &phase.Tasks[0]
		task.SuccessCriteria = []string{criterion}
		task.EvidenceRequirements = []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}}
	}
	source.Plan.Phases = append([]colony.Phase(nil), source.Revision.Phases...)
	return planProposalContract{
		Plan:          source.Plan,
		Revision:      source.Revision,
		Specification: source.Specification,
		TaskDeclarations: []planProposalTaskDeclaration{
			{TaskSemanticID: "task-compare", Files: []string{"cmd/planning_delta.go", "cmd/planning_delta_test.go"}},
			{TaskSemanticID: "task-present", Files: []string{"cmd/codex_visuals.go"}, UserFacing: true},
		},
		UserFacingSemanticIDs: []string{"phase-ui"},
	}
}

func planningDeltaTestSource() planningSemanticSnapshotSource {
	coreTaskID := "1.1"
	uiTaskID := "2.1"
	phases := []colony.Phase{
		{
			ID: 1, SemanticID: "phase-core", Name: "Core engine", Description: "Implement the planning engine", Mode: colony.PhaseModeProduction,
			Tasks: []colony.Task{{
				ID: &coreTaskID, SemanticID: "task-compare", Goal: "Implement the semantic comparator",
				RequirementProofLinks: []string{"requirement-semantic-delta"}, AcceptanceProofLinks: []string{"acceptance-semantic-delta"},
				NegativeProofLinks: []string{"negative-no-text-diff"}, RecoveryProofLinks: []string{"recovery-exact-replay"}, PublicPathProofLinks: []string{"path-ant-plan"},
			}},
		},
		{
			ID: 2, SemanticID: "phase-ui", Name: "Planning cards", Description: "Present each planning pass", Mode: colony.PhaseModePrototype,
			Tasks: []colony.Task{{
				ID: &uiTaskID, SemanticID: "task-present", Goal: "Present the compact delta card", DependsOn: []string{"1.1"},
				RequirementProofLinks: []string{"requirement-visible-card"}, AcceptanceProofLinks: []string{"acceptance-visible-card"},
				NegativeProofLinks: []string{"negative-no-authority-blur"}, RecoveryProofLinks: []string{"recovery-card-replay"}, PublicPathProofLinks: []string{"path-ant-plan"},
			}},
		},
	}
	revision := &colony.PlanRevision{
		SchemaVersion: 1,
		ID:            "plan-r1-test",
		SemanticID:    "plan-iterative-planning",
		Phases:        phases,
	}
	specRevision := colony.SpecRevision{
		ID: "spec-r1", ContentHash: planningDeltaTestHash("spec-r1"), Status: colony.SpecStatusDraft,
		Requirements: []colony.SpecRequirement{
			{ID: "requirement-semantic-delta", Description: "Compute semantic changes"},
			{ID: "requirement-visible-card", Description: "Show each pass"},
		},
		AcceptanceChecks: []colony.SpecAcceptanceCheck{
			{ID: "acceptance-semantic-delta", Description: "Formatting is ignored", Verification: "go test ./cmd"},
			{ID: "acceptance-visible-card", Description: "Changes stay visible", Verification: "go test ./cmd"},
		},
		NegativeExpectations: []colony.SpecNegativeExpectation{
			{ID: "negative-no-text-diff", Description: "Raw text is not semantic progress"},
			{ID: "negative-no-authority-blur", Description: "Authority cannot raise confidence"},
		},
		RecoveryExpectations: []colony.SpecRecoveryExpectation{
			{ID: "recovery-exact-replay", Description: "Resume from the exact validated receipt"},
			{ID: "recovery-card-replay", Description: "Render the existing card after replay"},
		},
		AffectedPublicPaths: []colony.SpecPublicPath{
			{ID: "path-ant-plan", Path: "/ant-plan", Description: "Plan through the Queen"},
		},
	}
	specification := &colony.Specification{
		ID: "specification-1", CurrentRevisionID: specRevision.ID, Revisions: []colony.SpecRevision{specRevision},
	}
	return planningSemanticSnapshotSource{
		CurrentSchema: true,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceExplicitOwner,
			Phases:           append([]colony.Phase(nil), phases...),
		},
		Revision:      revision,
		Specification: specification,
	}
}

func mustPlanningDeltaTestSnapshot(t *testing.T, source planningSemanticSnapshotSource) planningSemanticSnapshot {
	t.Helper()
	snapshot, err := buildPlanningSemanticSnapshot(source)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	return snapshot
}

func assertPlanningDeltaNoExecutableChange(t *testing.T, delta colony.PlanningSemanticDelta) {
	t.Helper()
	for _, section := range [][]colony.PlanningSemanticChange{
		delta.Phases,
		delta.Tasks,
		delta.Dependencies,
		delta.RequirementLinks,
		delta.AcceptanceChecks,
		delta.NegativeExpectations,
		delta.RecoveryExpectations,
		delta.PublicPaths,
	} {
		for _, change := range section {
			if change.Kind != colony.PlanningSemanticChangePreserved {
				t.Fatalf("unexpected executable change: %#v", change)
			}
		}
	}
}

func assertPlanningDeltaOnlyKind(t *testing.T, changes []colony.PlanningSemanticChange, kind colony.PlanningSemanticChangeKind, want int) {
	t.Helper()
	found := 0
	for _, change := range changes {
		if change.Kind == kind {
			found++
			continue
		}
		if change.Kind != colony.PlanningSemanticChangePreserved {
			t.Fatalf("unexpected change kind %s in %#v", change.Kind, changes)
		}
	}
	if found != want {
		t.Fatalf("%s changes = %d, want %d in %#v", kind, found, want, changes)
	}
}

func planningDeltaTestStringPtr(value string) *string {
	return &value
}

func planningDeltaTestHash(value string) string {
	hash, err := jsonSHA256(value)
	if err != nil {
		panic(err)
	}
	return hash
}
