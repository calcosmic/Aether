package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func phaseResearchAutomaticTestGap(t *testing.T, materiality colony.PlanningGapMateriality, severity int) colony.PlanningGap {
	t.Helper()
	gap := colony.PlanningGap{
		SchemaVersion:           colony.PlanningSchemaVersion,
		Dimension:               colony.PlanningDimensionKnowledge,
		Materiality:             materiality,
		Severity:                severity,
		Description:             "The billing provider contract is not yet grounded.",
		EvidenceThatWouldChange: "Current authoritative billing API documentation.",
	}
	payload := gap
	hash, err := jsonSHA256(payload)
	if err != nil {
		t.Fatalf("hash gap: %v", err)
	}
	gap.ContentHash = hash
	gap.ID = "planning-gap-" + hash[:16]
	if err := gap.Validate(); err != nil {
		t.Fatalf("validate gap: %v", err)
	}
	return gap
}

func phaseResearchAutomaticTestEvidence(t *testing.T, fresh bool) planningEvidenceRecord {
	t.Helper()
	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:           colony.PlanningEvidenceResearch,
		Origin:         "https://example.test/billing-api/v2",
		Content:        []byte("Billing API v2 uses signed webhooks and idempotency keys."),
		SourceRevision: "billing-api-v2",
		Scope: planningEvidenceScope{
			GoalID:                  "goal-1",
			SessionID:               "session-1",
			SpecificationRevisionID: "spec-1",
			PlanRevisionID:          "plan-1",
		},
		ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
		State:                planningEvidenceSourceCurrent,
	})
	if err != nil {
		t.Fatalf("normalize research evidence: %v", err)
	}
	record.Reference.Fresh = fresh
	return record
}

func TestPhaseResearchAutomaticPolicyUsesPresetGapAndFreshness(t *testing.T) {
	candidates := []phaseResearchCandidate{{
		ID:          1,
		Name:        "Wire billing provider",
		Description: "Integrate the external billing API and webhook protocol",
	}}
	survey := codexSurveyContext{Languages: []string{"go"}}

	t.Run("balanced researches an uncovered gap without asking the owner", func(t *testing.T) {
		gap := phaseResearchAutomaticTestGap(t, colony.PlanningGapNonMaterial, 75)
		policy := computeAutomaticPhaseResearchPolicy(planningStagePresetBalanced, &gap, survey, candidates, nil, nil, false)

		if !policy.ResearchRequired {
			t.Fatalf("ResearchRequired = false, want true: %+v", policy)
		}
		if policy.RequiresOwnerPrompt {
			t.Fatal("routine research requested an owner prompt")
		}
		if len(policy.Phases) != 1 || !policy.Phases[0].ResearchNeeded {
			t.Fatalf("phase policy = %+v, want phase 1 automatically selected", policy.Phases)
		}
	})

	t.Run("fast skips a nonmaterial gap but researches a material one", func(t *testing.T) {
		nonMaterial := phaseResearchAutomaticTestGap(t, colony.PlanningGapNonMaterial, 75)
		policy := computeAutomaticPhaseResearchPolicy(planningStagePresetFast, &nonMaterial, survey, candidates, nil, nil, false)
		if policy.ResearchRequired {
			t.Fatalf("fast nonmaterial policy unexpectedly requires research: %+v", policy)
		}

		material := phaseResearchAutomaticTestGap(t, colony.PlanningGapMaterial, 75)
		policy = computeAutomaticPhaseResearchPolicy(planningStagePresetFast, &material, survey, candidates, nil, nil, false)
		if !policy.ResearchRequired {
			t.Fatalf("fast material policy skipped evidence needed for a material gap: %+v", policy)
		}
		if policy.RequiresOwnerPrompt {
			t.Fatal("material research need itself must not prompt; Scout reports product decisions after the pass")
		}
	})

	t.Run("fresh applicable evidence suppresses routine research", func(t *testing.T) {
		gap := phaseResearchAutomaticTestGap(t, colony.PlanningGapNonMaterial, 75)
		fresh := phaseResearchAutomaticTestEvidence(t, true)
		policy := computeAutomaticPhaseResearchPolicy(planningStagePresetDeep, &gap, survey, candidates, nil, []planningEvidenceRecord{fresh}, false)
		if policy.ResearchRequired {
			t.Fatalf("fresh evidence should suppress duplicate research: %+v", policy)
		}
		if len(policy.FreshEvidenceIDs) != 1 || policy.FreshEvidenceIDs[0] != fresh.Reference.ID {
			t.Fatalf("FreshEvidenceIDs = %v, want [%s]", policy.FreshEvidenceIDs, fresh.Reference.ID)
		}
	})

	t.Run("stale evidence or explicit refresh runs research again", func(t *testing.T) {
		gap := phaseResearchAutomaticTestGap(t, colony.PlanningGapNonMaterial, 75)
		stale := phaseResearchAutomaticTestEvidence(t, false)
		policy := computeAutomaticPhaseResearchPolicy(planningStagePresetDeep, &gap, survey, candidates, nil, []planningEvidenceRecord{stale}, false)
		if !policy.ResearchRequired {
			t.Fatalf("stale evidence incorrectly suppressed research: %+v", policy)
		}

		fresh := phaseResearchAutomaticTestEvidence(t, true)
		policy = computeAutomaticPhaseResearchPolicy(planningStagePresetDeep, &gap, survey, candidates, nil, []planningEvidenceRecord{fresh}, true)
		if !policy.ResearchRequired || !policy.Refresh {
			t.Fatalf("explicit refresh did not force attributable research: %+v", policy)
		}
	})
}

func TestPhaseResearchAutomaticEvidenceContractIsScoutAttributed(t *testing.T) {
	policy := computeAutomaticPhaseResearchPolicy(planningStagePresetExhaustive, nil, codexSurveyContext{}, nil, nil, nil, false)
	contract := policy.EvidenceContract

	if contract.ProducerCaste != planningStageCasteScout {
		t.Fatalf("ProducerCaste = %q, want %q", contract.ProducerCaste, planningStageCasteScout)
	}
	if contract.SourceKind != colony.PlanningEvidenceResearch {
		t.Fatalf("SourceKind = %q, want %q", contract.SourceKind, colony.PlanningEvidenceResearch)
	}
	for _, field := range []string{"origin", "source_revision", "observed_at", "applicable_dimensions", "content_hash"} {
		if !phaseResearchStringSliceContains(contract.RequiredFields, field) {
			t.Errorf("RequiredFields = %v, want %q", contract.RequiredFields, field)
		}
	}
	if contract.MayApproveSpecification || contract.MayAcceptCandidate || contract.MayActivatePlan {
		t.Fatalf("research contract grants forbidden authority: %+v", contract)
	}
	if policy.RequiresOwnerPrompt {
		t.Fatal("automatic evidence contract requires an owner prompt")
	}
}

func TestPhaseResearchAutomaticSourcesFailClosed(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	scope := planningEvidenceScope{
		GoalID: "goal-1", SessionID: "session-1", SpecificationRevisionID: "spec-1", PlanRevisionID: "plan-1",
	}
	observedAt := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)

	t.Run("out of scope", func(t *testing.T) {
		outside := filepath.Join(filepath.Dir(root), "outside-research.md")
		if err := os.WriteFile(outside, []byte("public finding"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(outside) })
		_, err := collectScoutPhaseResearchEvidence(root, []string{"../outside-research.md"}, scope, observedAt)
		if err == nil || !strings.Contains(err.Error(), "outside") {
			t.Fatalf("out-of-scope source error = %v, want safe rejection", err)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		_, err := collectScoutPhaseResearchEvidence(root, []string{".aether/data/phase-research/missing.md"}, scope, observedAt)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "read") {
			t.Fatalf("unavailable source error = %v, want safe read failure", err)
		}
	})

	t.Run("secret-bearing", func(t *testing.T) {
		path := filepath.Join(inside, "phase-1-research.md")
		secret := "sk-abcdefghijklmnopqrstuvwxyz123456"
		if err := os.WriteFile(path, []byte("credential "+secret), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := collectScoutPhaseResearchEvidence(root, []string{".aether/data/phase-research/phase-1-research.md"}, scope, observedAt)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "secret") {
			t.Fatalf("secret-bearing source error = %v, want safe rejection", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatal("secret-bearing rejection echoed credential material")
		}
	})
}

func phaseResearchStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
