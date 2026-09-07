package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningMigrationClassifiesLegacyPlanAndIndexesArtifacts(t *testing.T) {
	root := t.TempDir()
	planningDir := filepath.Join(root, ".aether", "data", "planning")
	iterationsDir := filepath.Join(planningDir, "iterations")
	if err := os.MkdirAll(iterationsDir, 0o755); err != nil {
		t.Fatalf("create planning fixture directory: %v", err)
	}
	artifacts := map[string]string{
		".aether/data/planning/ROUTE-SETTER.md":                    "# Legacy route\n",
		".aether/data/planning/phase-plan.json":                    `{"phases":[{"name":"Legacy phase"}]}`,
		".aether/data/planning/iteration-state.json":               `{"planning_run_id":"legacy-run","last_iteration":1}`,
		".aether/data/planning/iterations/iteration-01-scout.json": `{"findings":["legacy evidence"]}`,
	}
	for rel, content := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	taskID := "1.1"
	state := colony.ColonyState{
		CurrentPhase: 1,
		Plan: colony.Plan{
			ActiveRevisionID: "plan-r1-legacy",
			Revisions: []colony.PlanRevision{{
				SchemaVersion: 1,
				Number:        1,
				ID:            "plan-r1-legacy",
				PlanHash:      strings.Repeat("a", 64),
			}},
			Phases: []colony.Phase{{
				ID:     1,
				Name:   "Legacy phase",
				Status: colony.PhaseReady,
				Tasks:  []colony.Task{{ID: &taskID, Goal: "Keep building", Status: colony.TaskPending}},
			}},
		},
	}
	originalPlan := state.Plan
	originalPhase := state.CurrentPhase

	first, err := migratePlanningState(root, state)
	if err != nil {
		t.Fatalf("migratePlanningState returned error: %v", err)
	}
	if !first.Changed {
		t.Fatal("expected the missing legacy classification to be migrated")
	}
	if first.State.Plan.AcceptancePolicy != colony.PlanAcceptanceLegacyUnbound {
		t.Fatalf("acceptance policy = %q, want %q", first.State.Plan.AcceptancePolicy, colony.PlanAcceptanceLegacyUnbound)
	}
	if first.State.CurrentPhase != originalPhase || !reflect.DeepEqual(first.State.Plan.Phases, originalPlan.Phases) {
		t.Fatalf("migration changed executable plan content: before=%+v after=%+v", originalPlan.Phases, first.State.Plan.Phases)
	}
	if first.State.Plan.ActiveRevisionID != originalPlan.ActiveRevisionID || !reflect.DeepEqual(first.State.Plan.Revisions, originalPlan.Revisions) {
		t.Fatalf("migration changed historical plan revisions: before=%+v after=%+v", originalPlan, first.State.Plan)
	}
	if first.State.Specification != nil || len(first.State.Plan.Candidates) != 0 || first.State.Plan.PendingCandidateID != "" {
		t.Fatalf("migration fabricated current authority: specification=%+v candidates=%+v pending=%q", first.State.Specification, first.State.Plan.Candidates, first.State.Plan.PendingCandidateID)
	}
	if len(first.LegacyArtifacts) != len(artifacts) {
		t.Fatalf("indexed %d legacy artifacts, want %d: %+v", len(first.LegacyArtifacts), len(artifacts), first.LegacyArtifacts)
	}
	for _, reference := range first.LegacyArtifacts {
		if reference.Classification != colony.PlanAcceptanceLegacyUnbound {
			t.Fatalf("artifact %s classification = %q", reference.RepositoryPath, reference.Classification)
		}
		wantHash, err := fileSHA256(filepath.Join(root, filepath.FromSlash(reference.RepositoryPath)))
		if err != nil {
			t.Fatalf("hash fixture %s: %v", reference.RepositoryPath, err)
		}
		if reference.ContentHash != wantHash {
			t.Fatalf("artifact %s hash = %s, want %s", reference.RepositoryPath, reference.ContentHash, wantHash)
		}
		wantMaterial := legacyPlanningEvidenceMaterial
		if reference.RepositoryPath == ".aether/data/planning/iteration-state.json" || strings.Contains(reference.RepositoryPath, "/iterations/") {
			wantMaterial = legacyPlanningTimelineMaterial
		}
		if reference.Material != wantMaterial {
			t.Fatalf("artifact %s material = %q, want %q", reference.RepositoryPath, reference.Material, wantMaterial)
		}
	}

	firstJSON, err := json.Marshal(first.State)
	if err != nil {
		t.Fatalf("marshal first migration: %v", err)
	}
	second, err := migratePlanningState(root, first.State)
	if err != nil {
		t.Fatalf("second migratePlanningState returned error: %v", err)
	}
	secondJSON, err := json.Marshal(second.State)
	if err != nil {
		t.Fatalf("marshal second migration: %v", err)
	}
	if second.Changed || string(firstJSON) != string(secondJSON) || !reflect.DeepEqual(first.LegacyArtifacts, second.LegacyArtifacts) {
		t.Fatalf("migration is not idempotent:\nfirst state:  %s\nsecond state: %s\nfirst refs:  %+v\nsecond refs: %+v", firstJSON, secondJSON, first.LegacyArtifacts, second.LegacyArtifacts)
	}
}

func TestPlanningMigrationRejectsEscapingArtifactBeforeReading(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	planningDir := filepath.Join(root, ".aether", "data", "planning")
	if err := os.MkdirAll(planningDir, 0o755); err != nil {
		t.Fatalf("create planning fixture directory: %v", err)
	}
	externalPath := filepath.Join(parent, "outside.json")
	if err := os.WriteFile(externalPath, []byte(`{"secret":"must not be read as planning evidence"}`), 0o644); err != nil {
		t.Fatalf("write external fixture: %v", err)
	}

	references, err := indexLegacyPlanningArtifacts(root, []string{
		".aether/data/planning/../../../../outside.json",
	})
	if err == nil || !strings.Contains(err.Error(), "escapes .aether/data/planning") {
		t.Fatalf("path escape error = %v, want .aether/data/planning containment refusal", err)
	}
	if len(references) != 0 {
		t.Fatalf("path escape returned references: %+v", references)
	}
}

func TestPlanningMigrationRejectsMalformedJSONArtifact(t *testing.T) {
	root := t.TempDir()
	rel := ".aether/data/planning/iterations/iteration-01-scout.json"
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create planning fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"findings":`), 0o644); err != nil {
		t.Fatalf("write malformed fixture: %v", err)
	}

	_, err := indexLegacyPlanningArtifacts(root, []string{rel})
	if err == nil || !strings.Contains(err.Error(), "malformed JSON") {
		t.Fatalf("malformed artifact error = %v, want malformed JSON refusal", err)
	}
}
