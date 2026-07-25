package cmd

import (
	"encoding/json"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Host Manifest Integration Tests
//
// These tests verify that JSON manifests produced by the TypeScript host
// can be correctly unmarshaled by the Go runtime structs. They serve as
// a schema-compatibility contract between the TS and Go layers.
// ---------------------------------------------------------------------------

func TestHostManifest_BuildManifest(t *testing.T) {
	fixture := `{
		"phase": 1,
		"phase_name": "Runtime Safety",
		"goal": "Implement build-reconcile and worktree safety",
		"root": "/tmp/test-repo",
		"colony_mode": "standard",
		"plan_only": false,
		"parallel_mode": "in-repo",
		"colony_depth": "full",
		"dispatch_mode": "dispatched",
		"host_platform": "claude",
		"generated_at": "2026-05-21T10:00:00Z",
		"state": "READY",
		"checkpoint": "build-start",
		"claims_path": ".aether/data/build-claims.json",
		"playbooks": [".aether/docs/command-playbooks/build-prep.md"],
		"worker_briefs": ["Build task brief"],
		"dispatches": [
			{
				"stage": "implement",
				"wave": 1,
				"caste": "builder",
				"name": "Builder-01",
				"task": "Implement feature X",
				"status": "pending",
				"task_id": "1.1",
				"skill_section": "TDD"
			},
			{
				"stage": "verify",
				"wave": 1,
				"caste": "watcher",
				"name": "Watcher-01",
				"task": "Verify tests pass",
				"status": "pending",
				"task_id": "1.2"
			}
		],
		"tasks": [
			{"id": "1.1", "goal": "Implement feature", "status": "pending"},
			{"id": "1.2", "goal": "Verify tests", "status": "pending"}
		],
		"success_criteria": ["Tests pass", "No regressions"],
		"review_depth": "standard",
		"queen_recommendation": {
			"review_depth": "standard",
			"reason": "Standard build with builder and watcher"
		},
		"queen_execution_policy": {
			"verification_depth": "Standard",
			"review_depth": "standard",
			"spawn_budget": {
				"max_workers": 10,
				"selected_workers": 2,
				"worker_count": 2,
				"counts": {"builder": 1, "watcher": 1}
			}
		}
	}`

	var manifest codexBuildManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("failed to unmarshal build manifest: %v", err)
	}

	if manifest.Phase != 1 {
		t.Errorf("phase = %d, want 1", manifest.Phase)
	}
	if manifest.PhaseName != "Runtime Safety" {
		t.Errorf("phase_name = %q, want 'Runtime Safety'", manifest.PhaseName)
	}
	if len(manifest.Dispatches) != 2 {
		t.Errorf("dispatches len = %d, want 2", len(manifest.Dispatches))
	}
	if manifest.Dispatches[0].Caste != "builder" {
		t.Errorf("dispatch[0].caste = %q, want 'builder'", manifest.Dispatches[0].Caste)
	}
	if manifest.Dispatches[0].SkillSection != "TDD" {
		t.Errorf("dispatch[0].skill_section = %q, want 'TDD'", manifest.Dispatches[0].SkillSection)
	}
	if manifest.QueenExecutionPolicy.SpawnBudget.MaxWorkers != 10 {
		t.Errorf("spawn_budget.max_workers = %d, want 10", manifest.QueenExecutionPolicy.SpawnBudget.MaxWorkers)
	}
	if manifest.QueenExecutionPolicy.SpawnBudget.Counts["builder"] != 1 {
		t.Errorf("spawn_budget.counts.builder = %d, want 1", manifest.QueenExecutionPolicy.SpawnBudget.Counts["builder"])
	}
}

func TestHostManifest_PlanManifest(t *testing.T) {
	fixture := `{
		"goal": "Plan the v1.23 milestone",
		"root": "/tmp/test-repo",
		"generated_at": "2026-05-21T10:00:00Z",
		"colony_mode": "standard",
		"refresh": false,
		"existing_plan": false,
		"depth": "full",
		"granularity": "medium",
		"granularity_min": 3,
		"granularity_max": 7,
		"planning_depth": "full",
		"verification_depth": "standard",
		"survey": {
			"SurveyDir": ".aether/data/survey",
			"Languages": ["go", "typescript"],
			"Frameworks": ["cobra", "node:test"],
			"Directories": ["cmd", "pkg", ".aether/ts-host"],
			"EntryPoints": ["cmd/aether"],
			"Dependencies": ["chalk", "boxen"],
			"TestFiles": ["*_test.go", "*.test.ts"],
			"Issues": [],
			"SecurityPatterns": [],
			"SourceAnchors": []
		},
		"dispatches": [
			{
				"caste": "scout",
				"name": "Scout-01",
				"task": "Research codebase structure",
				"outputs": [".aether/data/survey/BLUEPRINT.md"],
				"status": "pending"
			}
		],
		"dispatch_mode": "dispatched",
		"dispatch_contract": {
			"wave_count": 1,
			"worker_count": 1
		},
		"finalize_surface": "plan",
		"requires_finalizer": true
	}`

	var manifest codexPlanManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("failed to unmarshal plan manifest: %v", err)
	}

	if manifest.Goal != "Plan the v1.23 milestone" {
		t.Errorf("goal = %q, want 'Plan the v1.23 milestone'", manifest.Goal)
	}
	if manifest.Depth != "full" {
		t.Errorf("depth = %q, want 'full'", manifest.Depth)
	}
	if manifest.Survey.Languages[0] != "go" {
		t.Errorf("survey.languages[0] = %q, want 'go'", manifest.Survey.Languages[0])
	}
	if len(manifest.Dispatches) != 1 {
		t.Errorf("dispatches len = %d, want 1", len(manifest.Dispatches))
	}
	if manifest.Dispatches[0].Caste != "scout" {
		t.Errorf("dispatch[0].caste = %q, want 'scout'", manifest.Dispatches[0].Caste)
	}
	if !manifest.RequiresFinalizer {
		t.Error("requires_finalizer should be true")
	}
}

func TestHostManifest_ColonizeManifest(t *testing.T) {
	fixture := `{
		"workflow": "colonize",
		"dispatch_mode": "dispatched",
		"requires_finalizer": true,
		"generated_at": "2026-05-21T10:00:00Z",
		"root": "/tmp/test-repo",
		"detected_type": "go-cli",
		"languages": ["go", "typescript"],
		"frameworks": ["cobra"],
		"domains": ["cli", "automation"],
		"entry_points": ["cmd/aether"],
		"key_dirs": ["cmd", "pkg", ".aether"],
		"existing_survey": false,
		"force_resurvey": false,
		"worker_timeout_seconds": 600,
		"dispatch_contract": {
			"wave_count": 2,
			"worker_count": 4
		},
		"dispatches": [
			{
				"caste": "surveyor-nest",
				"name": "Surveyor-Nest-01",
				"task": "Map directory structure",
				"status": "pending"
			},
			{
				"caste": "surveyor-disciplines",
				"name": "Surveyor-Disciplines-01",
				"task": "Document conventions",
				"status": "pending"
			}
		],
		"finalizer_command": "colonize-finalize",
		"stats": {
			"file_count": 150,
			"test_count": 512
		}
	}`

	var manifest codexColonizeManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("failed to unmarshal colonize manifest: %v", err)
	}

	if manifest.Workflow != "colonize" {
		t.Errorf("workflow = %q, want 'colonize'", manifest.Workflow)
	}
	if manifest.DetectedType != "go-cli" {
		t.Errorf("detected_type = %q, want 'go-cli'", manifest.DetectedType)
	}
	if len(manifest.Languages) != 2 {
		t.Errorf("languages len = %d, want 2", len(manifest.Languages))
	}
	if manifest.Dispatches[0].Caste != "surveyor-nest" {
		t.Errorf("dispatch[0].caste = %q, want 'surveyor-nest'", manifest.Dispatches[0].Caste)
	}
	if manifest.FinalizerCommand != "colonize-finalize" {
		t.Errorf("finalizer_command = %q, want 'colonize-finalize'", manifest.FinalizerCommand)
	}
	if manifest.Stats["file_count"] != float64(150) {
		t.Errorf("stats.file_count = %v, want 150", manifest.Stats["file_count"])
	}
}

func TestHostManifest_SealManifest(t *testing.T) {
	fixture := `{
		"workflow": "seal",
		"phase": 7,
		"phase_name": "End-to-End Proof",
		"root": "/tmp/test-repo",
		"generated_at": "2026-05-21T10:00:00Z",
		"colony_mode": "standard",
		"review_depth": "final-review",
		"dispatch_mode": "agent-delegate",
		"requires_finalizer": true,
		"finalize_surface": "seal",
		"finalizer_command": "seal-finalize",
		"worker_timeout_seconds": 600,
		"dispatches": [
			{
				"stage": "review",
				"wave": 1,
				"caste": "auditor",
				"name": "Auditor-01",
				"task": "Final quality review",
				"task_id": "7.1",
				"status": "pending"
			}
		],
		"dispatch_contract": {
			"wave_count": 1,
			"worker_count": 1
		},
		"post_seal_directives": ["Run /ant-entomb to archive"]
	}`

	var manifest sealPlanManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("failed to unmarshal seal manifest: %v", err)
	}

	if manifest.Workflow != "seal" {
		t.Errorf("workflow = %q, want 'seal'", manifest.Workflow)
	}
	if manifest.Phase != 7 {
		t.Errorf("phase = %d, want 7", manifest.Phase)
	}
	if manifest.ReviewDepth != "final-review" {
		t.Errorf("review_depth = %q, want 'final-review'", manifest.ReviewDepth)
	}
	if len(manifest.Dispatches) != 1 {
		t.Errorf("dispatches len = %d, want 1", len(manifest.Dispatches))
	}
	if manifest.Dispatches[0].Caste != "auditor" {
		t.Errorf("dispatch[0].caste = %q, want 'auditor'", manifest.Dispatches[0].Caste)
	}
	if len(manifest.PostSealDirectives) != 1 {
		t.Errorf("post_seal_directives len = %d, want 1", len(manifest.PostSealDirectives))
	}
}

func TestHostManifest_ContinueManifest(t *testing.T) {
	fixture := `{
		"phase": 1,
		"phase_name": "Runtime Safety",
		"root": "/tmp/test-repo",
		"generated_at": "2026-05-21T10:00:00Z",
		"colony_mode": "standard",
		"verification": {
			"phase": 1,
			"generated_at": "2026-05-21T10:05:00Z",
			"steps": [
				{
					"name": "go test",
					"command": "go test ./...",
					"passed": true,
					"summary": "All tests passed"
				}
			],
			"claims": {
				"present": true,
				"passed": true,
				"summary": "Claims verified",
				"checked": 2
			},
			"watcher": {
				"present": true,
				"passed": true,
				"status": "completed",
				"worker": "Watcher-01"
			},
			"checks_passed": true,
			"passed": true
		},
		"assessment": {
			"phase": 1,
			"generated_at": "2026-05-21T10:05:00Z",
			"tasks": [
				{
					"task_id": "1.1",
					"goal": "Implement feature",
					"outcome": "completed",
					"summary": "Feature implemented and tested",
					"verified": true,
					"reconciled": true
				}
			],
			"verification_passed": true,
			"positive_evidence": true,
			"passed": true,
			"summary": "Phase 1 complete"
		},
		"dispatches": [
			{
				"stage": "verify",
				"wave": 1,
				"caste": "watcher",
				"name": "Watcher-01",
				"task": "Verify phase completion",
				"task_id": "1.1",
				"status": "pending"
			}
		],
		"dispatch_mode": "dispatched",
		"finalize_surface": "continue",
		"requires_finalizer": true
	}`

	var manifest codexContinuePlanManifest
	if err := json.Unmarshal([]byte(fixture), &manifest); err != nil {
		t.Fatalf("failed to unmarshal continue manifest: %v", err)
	}

	if manifest.Phase != 1 {
		t.Errorf("phase = %d, want 1", manifest.Phase)
	}
	if manifest.PhaseName != "Runtime Safety" {
		t.Errorf("phase_name = %q, want 'Runtime Safety'", manifest.PhaseName)
	}
	if !manifest.Verification.Passed {
		t.Error("verification.passed should be true")
	}
	if manifest.Verification.Steps[0].Name != "go test" {
		t.Errorf("verification.steps[0].name = %q, want 'go test'", manifest.Verification.Steps[0].Name)
	}
	if !manifest.Assessment.Passed {
		t.Error("assessment.passed should be true")
	}
	if len(manifest.Dispatches) != 1 {
		t.Errorf("dispatches len = %d, want 1", len(manifest.Dispatches))
	}
	if manifest.Dispatches[0].Caste != "watcher" {
		t.Errorf("dispatch[0].caste = %q, want 'watcher'", manifest.Dispatches[0].Caste)
	}
}

func TestHostManifest_BuildManifestRoundtrip(t *testing.T) {
	original := codexBuildManifest{
		Phase:        2,
		PhaseName:    "Test Roundtrip",
		Root:         "/tmp/roundtrip",
		ColonyDepth:  "full",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		State:        "READY",
		Checkpoint:   "build-start",
		ClaimsPath:   ".aether/data/claims.json",
		Playbooks:    []string{"build-prep.md"},
		WorkerBriefs: []string{"brief"},
		Dispatches: []codexBuildDispatch{
			{
				Stage:  "implement",
				Wave:   1,
				Caste:  "builder",
				Name:   "Builder-01",
				Task:   "Build",
				Status: "pending",
			},
		},
		Tasks: []codexBuildTaskPlan{
			{Goal: "Build it", Status: "pending"},
		},
		SuccessCriteria: []string{"Done"},
	}

	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded codexBuildManifest
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Phase != original.Phase {
		t.Errorf("phase = %d, want %d", decoded.Phase, original.Phase)
	}
	if decoded.PhaseName != original.PhaseName {
		t.Errorf("phase_name = %q, want %q", decoded.PhaseName, original.PhaseName)
	}
	if len(decoded.Dispatches) != len(original.Dispatches) {
		t.Errorf("dispatches len = %d, want %d", len(decoded.Dispatches), len(original.Dispatches))
	}
	if decoded.Dispatches[0].Name != original.Dispatches[0].Name {
		t.Errorf("dispatch[0].name = %q, want %q", decoded.Dispatches[0].Name, original.Dispatches[0].Name)
	}
}
