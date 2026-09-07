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
