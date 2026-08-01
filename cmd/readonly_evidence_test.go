package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestParseReadOnlyArtifactSpec(t *testing.T) {
	t.Run("valid spec splits on first colon", func(t *testing.T) {
		taskID, path, err := parseReadOnlyArtifactSpec("1.1:cmd/foo.go")
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if taskID != "1.1" || path != "cmd/foo.go" {
			t.Fatalf("taskID=%q path=%q, want 1.1 / cmd/foo.go", taskID, path)
		}
	})

	t.Run("path containing a colon keeps only the first split", func(t *testing.T) {
		taskID, path, err := parseReadOnlyArtifactSpec("1.1:C:\\weird\\path.go")
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if taskID != "1.1" || path != "C:\\weird\\path.go" {
			t.Fatalf("taskID=%q path=%q, want 1.1 / C:\\weird\\path.go", taskID, path)
		}
	})

	t.Run("missing separator is rejected with the expected form", func(t *testing.T) {
		_, _, err := parseReadOnlyArtifactSpec("no-separator-here")
		if err == nil || !strings.Contains(err.Error(), "<task-id>:<path>") {
			t.Fatalf("error = %v, want message containing <task-id>:<path>", err)
		}
	})

	t.Run("empty task id is rejected", func(t *testing.T) {
		_, _, err := parseReadOnlyArtifactSpec(":cmd/foo.go")
		if err == nil || !strings.Contains(err.Error(), "<task-id>:<path>") {
			t.Fatalf("error = %v, want message containing <task-id>:<path>", err)
		}
	})

	t.Run("empty path is rejected", func(t *testing.T) {
		_, _, err := parseReadOnlyArtifactSpec("1.1:")
		if err == nil || !strings.Contains(err.Error(), "<task-id>:<path>") {
			t.Fatalf("error = %v, want message containing <task-id>:<path>", err)
		}
	})
}

func readOnlyEvidenceTestPhase() colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:   1,
		Name: "Read-only evidence flag",
		Tasks: []colony.Task{
			{ID: &taskID, Goal: "Test-only task", Status: colony.TaskCompleted},
		},
	}
}

func TestValidateReadOnlyArtifacts(t *testing.T) {
	phase := readOnlyEvidenceTestPhase()

	t.Run("no specs is always valid", func(t *testing.T) {
		if err := validateReadOnlyArtifacts(phase, nil, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("task id not present in the phase is rejected", func(t *testing.T) {
		err := validateReadOnlyArtifacts(phase, []string{"9.9"}, []string{"9.9:x"})
		if err == nil || !strings.Contains(err.Error(), "unknown task id(s) for phase 1: 9.9") {
			t.Fatalf("error = %v, want unknown task id message", err)
		}
	})

	t.Run("task id not also passed to --reconcile-task is rejected", func(t *testing.T) {
		err := validateReadOnlyArtifacts(phase, nil, []string{"1.1:x"})
		if err == nil || !strings.Contains(err.Error(), "must also be passed to --reconcile-task") || !strings.Contains(err.Error(), "1.1") {
			t.Fatalf("error = %v, want reconcile-task requirement message", err)
		}
	})

	t.Run("task id present and reconciled is accepted", func(t *testing.T) {
		if err := validateReadOnlyArtifacts(phase, []string{"1.1"}, []string{"1.1:x"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("malformed spec surfaces the parse error", func(t *testing.T) {
		err := validateReadOnlyArtifacts(phase, []string{"1.1"}, []string{"no-colon-here"})
		if err == nil || !strings.Contains(err.Error(), "<task-id>:<path>") {
			t.Fatalf("error = %v, want malformed spec message", err)
		}
	})
}

func TestRecordReadOnlyArtifactEvidence(t *testing.T) {
	t.Run("records hash-verified evidence scoped to the task", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "foo.go"), []byte("package foo\n"), 0644); err != nil {
			t.Fatalf("write artifact: %v", err)
		}
		claims := &codexBuildClaims{}
		if err := recordReadOnlyArtifactEvidence(root, claims, []string{"1.1:foo.go"}); err != nil {
			t.Fatalf("record error: %v", err)
		}
		if len(claims.ArtifactEvidence) != 1 {
			t.Fatalf("evidence = %+v, want one entry", claims.ArtifactEvidence)
		}
		entry := claims.ArtifactEvidence[0]
		if entry.Path != "foo.go" || !entry.ReadOnly || entry.ReadOnlyTaskID != "1.1" || entry.SHA256 == "" {
			t.Fatalf("entry = %+v, want read-only evidence scoped to task 1.1 with a hash", entry)
		}
	})

	t.Run("replaces an existing entry for the same path instead of duplicating", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "foo.go"), []byte("package foo\n"), 0644); err != nil {
			t.Fatalf("write artifact: %v", err)
		}
		claims := &codexBuildClaims{ArtifactEvidence: []codexBuildArtifactEvidence{{Path: "foo.go", SHA256: "stale"}}}
		if err := recordReadOnlyArtifactEvidence(root, claims, []string{"1.1:foo.go"}); err != nil {
			t.Fatalf("record error: %v", err)
		}
		if len(claims.ArtifactEvidence) != 1 || claims.ArtifactEvidence[0].SHA256 == "stale" {
			t.Fatalf("evidence = %+v, want replaced single entry", claims.ArtifactEvidence)
		}
	})

	t.Run("rejects a symlinked artifact", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "evidence.txt"), []byte("outside\n"), 0644); err != nil {
			t.Fatalf("write outside evidence: %v", err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "external")); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}
		claims := &codexBuildClaims{}
		err := recordReadOnlyArtifactEvidence(root, claims, []string{"1.1:external/evidence.txt"})
		if err == nil || !strings.Contains(err.Error(), "outside the repository") {
			t.Fatalf("error = %v, want symlink escape rejection", err)
		}
	})

	t.Run("rejects escaping and runtime-state paths", func(t *testing.T) {
		root := t.TempDir()
		claims := &codexBuildClaims{}
		for _, spec := range []string{
			"1.1:../etc/passwd",
			"1.1:/etc/passwd",
			"1.1:.aether/data/COLONY_STATE.json",
		} {
			if err := recordReadOnlyArtifactEvidence(root, claims, []string{spec}); err == nil {
				t.Fatalf("spec %q: expected rejection, got nil error", spec)
			}
		}
	})

	t.Run("malformed spec surfaces the parse error", func(t *testing.T) {
		root := t.TempDir()
		claims := &codexBuildClaims{}
		err := recordReadOnlyArtifactEvidence(root, claims, []string{"no-colon-here"})
		if err == nil || !strings.Contains(err.Error(), "<task-id>:<path>") {
			t.Fatalf("error = %v, want malformed spec message", err)
		}
	})
}

// TestVerifyPlanReadOnlyArtifactEvidence proves 163.1-09 Task 2: continue-finalize
// (via runCodexContinueFinalize) must see hash-verified, task-scoped read-only
// evidence already recorded in the claims file by an operator-invoked
// plan-only run before it is honored -- a manifest-only claim (T-163.1-45)
// must fail loudly rather than being silently trusted.
func TestVerifyPlanReadOnlyArtifactEvidence(t *testing.T) {
	phase := readOnlyEvidenceTestPhase()

	t.Run("empty read_only_artifacts returns nil without touching claims", func(t *testing.T) {
		setupBuildFlowTest(t)
		plan := &codexContinuePlanManifest{}
		manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ClaimsPath: "last-build-claims.json"}}
		if err := verifyPlanReadOnlyArtifactEvidence("", phase, plan, manifest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("evidence recorded for the matching task satisfies the spec", func(t *testing.T) {
		setupBuildFlowTest(t)
		if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
			ArtifactEvidence: []codexBuildArtifactEvidence{
				{Path: "foo.go", SHA256: "abc", ReadOnly: true, ReadOnlyTaskID: "1.1"},
			},
		}); err != nil {
			t.Fatalf("save claims: %v", err)
		}
		plan := &codexContinuePlanManifest{ReadOnlyArtifacts: []string{"1.1:foo.go"}}
		manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ClaimsPath: "last-build-claims.json"}}
		if err := verifyPlanReadOnlyArtifactEvidence("", phase, plan, manifest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unrecorded spec fails loudly naming the spec and the recovery command", func(t *testing.T) {
		setupBuildFlowTest(t)
		if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{}); err != nil {
			t.Fatalf("save claims: %v", err)
		}
		plan := &codexContinuePlanManifest{ReadOnlyArtifacts: []string{"1.1:foo.go"}}
		manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ClaimsPath: "last-build-claims.json"}}
		err := verifyPlanReadOnlyArtifactEvidence("", phase, plan, manifest)
		if err == nil || !strings.Contains(err.Error(), "1.1:foo.go") || !strings.Contains(err.Error(), "--read-only-artifact") {
			t.Fatalf("error = %v, want message naming spec 1.1:foo.go and --read-only-artifact", err)
		}
	})

	t.Run("evidence recorded for a different task does not satisfy the spec", func(t *testing.T) {
		setupBuildFlowTest(t)
		if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
			ArtifactEvidence: []codexBuildArtifactEvidence{
				{Path: "foo.go", SHA256: "abc", ReadOnly: true, ReadOnlyTaskID: "1.1"},
			},
		}); err != nil {
			t.Fatalf("save claims: %v", err)
		}
		plan := &codexContinuePlanManifest{ReadOnlyArtifacts: []string{"2.1:foo.go"}}
		manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ClaimsPath: "last-build-claims.json"}}
		err := verifyPlanReadOnlyArtifactEvidence("", phase, plan, manifest)
		if err == nil || !strings.Contains(err.Error(), "2.1:foo.go") {
			t.Fatalf("error = %v, want message naming cross-task spec 2.1:foo.go", err)
		}
	})

	t.Run("evidence present but not marked read-only is rejected", func(t *testing.T) {
		setupBuildFlowTest(t)
		if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{
			ArtifactEvidence: []codexBuildArtifactEvidence{
				{Path: "foo.go", SHA256: "abc", ReadOnly: false, ReadOnlyTaskID: "1.1"},
			},
		}); err != nil {
			t.Fatalf("save claims: %v", err)
		}
		plan := &codexContinuePlanManifest{ReadOnlyArtifacts: []string{"1.1:foo.go"}}
		manifest := codexContinueManifest{Present: true, Data: codexBuildManifest{ClaimsPath: "last-build-claims.json"}}
		err := verifyPlanReadOnlyArtifactEvidence("", phase, plan, manifest)
		if err == nil || !strings.Contains(err.Error(), "1.1:foo.go") {
			t.Fatalf("error = %v, want message naming spec 1.1:foo.go", err)
		}
	})
}
