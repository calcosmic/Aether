package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningScoutStageFinalizeCommitsReceiptWithoutCard(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)

	completed, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Receipt.Caste != planningStageCasteScout || completed.Receipt.ResultingState != planningStageRouteReady {
		t.Fatalf("Scout receipt = %+v, want Scout -> route_ready", completed.Receipt)
	}
	if completed.Artifact.Path != completed.Receipt.OutputPath || completed.Artifact.ContentHash != completed.Receipt.OutputHash {
		t.Fatalf("Scout artifact is not bound by receipt: artifact=%+v receipt=%+v", completed.Artifact, completed.Receipt)
	}
	if len(completed.Result.Findings) != 1 || len(completed.Result.NewEvidence) != 1 || len(completed.Result.UnresolvedGaps) != 1 {
		t.Fatalf("normalized Scout result lost content: %+v", completed.Result)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(completed.Artifact.Path))); err != nil {
		t.Fatalf("Scout artifact was not persisted: %v", err)
	}
	chain, err := readPlanningStageReceiptChain(root, manifest.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Receipts) != 1 || chain.Receipts[0].ID != completed.Receipt.ID || len(chain.Cards) != 0 {
		t.Fatalf("Scout boundary = %+v, want one receipt and no iteration card", chain)
	}
	state := planningStageReceiptTestReadState(t, root, manifest.RunID)
	if state.ScoutReceipt == nil || state.ScoutReceipt.ID != completed.Receipt.ID || state.ActiveManifestID != "" {
		t.Fatalf("Scout completion did not durably advance the stage: %+v", state)
	}
}

func TestPlanningScoutStageRejectsMismatchedAuthorityWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		change func(*planningScoutStageResult)
		want   string
	}{
		{name: "manifest hash", change: func(result *planningScoutStageResult) { result.ManifestHash = planningStageTestHash("0") }, want: "manifest"},
		{name: "run", change: func(result *planningScoutStageResult) { result.RunID = "another-run" }, want: "run"},
		{name: "pass", change: func(result *planningScoutStageResult) { result.Pass++ }, want: "pass"},
		{name: "caste", change: func(result *planningScoutStageResult) { result.Caste = planningStageCasteRouteSetter }, want: "caste"},
		{name: "specification", change: func(result *planningScoutStageResult) { result.Specification.ContentHash = planningStageTestHash("0") }, want: "specification"},
		{name: "base revision", change: func(result *planningScoutStageResult) { result.BasePlanRevisionHash = planningStageTestHash("0") }, want: "base plan"},
		{name: "frontier", change: func(result *planningScoutStageResult) { result.InputFrontierHash = planningStageTestHash("0") }, want: "frontier"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, manifest, result := planningScoutStageTestFixture(t)
			test.change(&result)
			statePath := filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))
			before := planningStageReceiptTestReadBytes(t, statePath)

			_, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
				t.Fatalf("error = %v, want authority mismatch containing %q", err, test.want)
			}
			if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, before) {
				t.Fatal("rejected Scout result changed stage state")
			}
			planningScoutStageAssertNoResultWrites(t, root, manifest)
		})
	}
}

func TestPlanningScoutStageRejectsForbiddenAndUnknownFields(t *testing.T) {
	for _, field := range []string{
		"confidence", "dimension_scores", "semantic_delta", "stop_reason", "candidate",
		"accepted", "activation", "state_patch", "unexpected_worker_claim",
	} {
		t.Run(field, func(t *testing.T) {
			root, manifest, result := planningScoutStageTestFixture(t)
			var payload map[string]any
			if err := json.Unmarshal(planningScoutStageTestBytes(t, result), &payload); err != nil {
				t.Fatal(err)
			}
			payload[field] = true
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := finalizePlanningScoutStage(root, manifest, raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Scout field %q was not rejected strictly: %v", field, err)
			}
			planningScoutStageAssertNoResultWrites(t, root, manifest)
		})
	}
}

func TestPlanningScoutStageRejectsUncitedFindingAndAcceptsExplicitUnknown(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	result.Findings[0].EvidenceIDs = nil
	if _, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result)); err == nil || !strings.Contains(err.Error(), "evidence or an explicit unknown") {
		t.Fatalf("uncited finding error = %v", err)
	}
	planningScoutStageAssertNoResultWrites(t, root, manifest)

	result.Findings[0].Unknown = true
	result.Findings[0].UnknownReason = "The repository contains no authoritative retention period."
	completed, err := finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	if err != nil {
		t.Fatalf("explicit unknown was rejected: %v", err)
	}
	if !completed.Result.Findings[0].Unknown || completed.Result.Findings[0].UnknownReason == "" {
		t.Fatalf("explicit unknown was not preserved: %+v", completed.Result.Findings[0])
	}
}

func TestPlanningScoutStageReplayReturnsOriginalReceiptAndRejectsDivergence(t *testing.T) {
	root, manifest, result := planningScoutStageTestFixture(t)
	raw := planningScoutStageTestBytes(t, result)
	first, err := finalizePlanningScoutStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(manifest.RunID)))
	beforeState := planningStageReceiptTestReadBytes(t, statePath)
	beforeArtifact := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(first.Artifact.Path)))

	replayed, err := finalizePlanningScoutStage(root, manifest, raw)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Receipt.ID != first.Receipt.ID || replayed.Receipt.ContentHash != first.Receipt.ContentHash {
		t.Fatalf("exact replay changed receipt: first=%+v replay=%+v", first.Receipt, replayed.Receipt)
	}
	if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, beforeState) {
		t.Fatal("exact replay changed planning frontier")
	}

	result.Findings[0].Summary = "Divergent finding for the same manifest"
	_, err = finalizePlanningScoutStage(root, manifest, planningScoutStageTestBytes(t, result))
	var conflict *planningStageReceiptConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("divergent replay error = %T %v, want planningStageReceiptConflictError", err, err)
	}
	if after := planningStageReceiptTestReadBytes(t, statePath); !bytes.Equal(after, beforeState) {
		t.Fatal("divergent replay changed planning frontier")
	}
	if after := planningStageReceiptTestReadBytes(t, filepath.Join(root, filepath.FromSlash(first.Artifact.Path))); !bytes.Equal(after, beforeArtifact) {
		t.Fatal("divergent replay changed original Scout artifact")
	}
}

func planningScoutStageTestFixture(t *testing.T) (string, planningStageManifest, planningScoutStageResult) {
	t.Helper()
	root := t.TempDir()
	_, manifest := planningStageReceiptTestScoutDispatch(t, root)
	header := planningScoutStageTestHeader(t, manifest)
	planningStageReceiptTestWriteJSON(t, filepath.Join(root, ".aether", "data", "planning", manifest.RunID, "run-header.json"), header)

	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:    colony.PlanningEvidenceResearch,
		Origin:  "scout:pass-1:repository-observation",
		Content: []byte("The lifecycle transaction already provides a durable Scout receipt boundary."),
		Scope: planningEvidenceScope{
			GoalID:                  header.GoalID,
			SessionID:               header.SessionID,
			SpecificationRevisionID: manifest.Specification.RevisionID,
			PlanRevisionID:          manifest.BasePlanRevisionID,
		},
		SourceRevision:       "scout-result-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 18, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	})
	if err != nil {
		t.Fatal(err)
	}
	gap := *planningStageTestGap("scout-unresolved-gap")
	gap.EvidenceIDs = []string{record.Reference.ID}
	return root, manifest, planningScoutStageResult{
		ResultType:           planningStageResultScout,
		ManifestID:           manifest.ID,
		ManifestHash:         manifest.ContentHash,
		RunID:                manifest.RunID,
		Pass:                 manifest.Pass,
		Caste:                planningStageCasteScout,
		Specification:        manifest.Specification,
		BasePlanRevisionID:   manifest.BasePlanRevisionID,
		BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		InputFrontierHash:    manifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID:    "scout-finding-stage-boundary",
			Summary:     "The existing transaction can commit a Scout receipt before Route-Setter starts.",
			EvidenceIDs: []string{record.Reference.ID},
		}},
		NewEvidence:    []planningEvidenceRecord{record},
		UnresolvedGaps: []colony.PlanningGap{gap},
	}
}

func planningScoutStageTestHeader(t *testing.T, manifest planningStageManifest) planningRunHeader {
	t.Helper()
	header := planningRunHeader{
		SchemaVersion:        planningRunHeaderSchemaVersion,
		RunID:                manifest.RunID,
		Goal:                 "Restore visible staged planning",
		GoalID:               "goal-200",
		SessionID:            "session-200",
		Specification:        manifest.Specification,
		BasePlanRevisionID:   manifest.BasePlanRevisionID,
		BasePlanRevisionHash: manifest.BasePlanRevisionHash,
		Preset:               manifest.Preset,
		TargetConfidence:     90,
		PassCap:              6,
		EvidenceFrontier:     append([]planningStageEvidenceBinding(nil), manifest.EvidenceFrontier...),
		InputFrontierHash:    manifest.InputFrontierHash,
		ResearchPolicy: phaseResearchAutomaticPolicy{
			SchemaVersion:         phaseResearchAutomaticPolicySchemaVersion,
			Preset:                manifest.Preset,
			OwnerDecisionBoundary: "after_scout",
			EvidenceContract:      automaticPhaseResearchEvidenceContract(),
		},
		WeakestGap:        *manifest.WeakestGap,
		StageManifestID:   manifest.ID,
		StageManifestHash: manifest.ContentHash,
		CreatedAt:         time.Date(2026, time.September, 7, 17, 0, 0, 0, time.UTC),
	}
	payload := header
	payload.ID = ""
	payload.ContentHash = ""
	hash, err := jsonSHA256(payload)
	if err != nil {
		t.Fatal(err)
	}
	header.ContentHash = hash
	header.ID = "planning-run-header-" + hash[:16]
	return header
}

func planningScoutStageTestBytes(t *testing.T, result planningScoutStageResult) []byte {
	t.Helper()
	content, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func planningScoutStageAssertNoResultWrites(t *testing.T, root string, manifest planningStageManifest) {
	t.Helper()
	for _, path := range []string{
		planningStageOutputRepositoryPath(manifest),
		planningStageReceiptIndexRepositoryPath(manifest.RunID),
	} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Fatalf("rejected Scout result wrote %s: %v", path, err)
		}
	}
}
