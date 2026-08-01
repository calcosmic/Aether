package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLifecycleFinalizerLoadersRejectBadCompletionFiles(t *testing.T) {
	loaders := []struct {
		name         string
		manifestName string
		load         func(string) error
	}{
		{
			name:         "plan",
			manifestName: "plan_manifest",
			load: func(path string) error {
				_, err := loadExternalPlanCompletion(path)
				return err
			},
		},
		{
			name:         "build",
			manifestName: "dispatch_manifest",
			load: func(path string) error {
				_, err := loadExternalBuildCompletion(path)
				return err
			},
		},
		{
			name:         "continue",
			manifestName: "continue_manifest",
			load: func(path string) error {
				_, err := loadExternalContinueCompletion(path)
				return err
			},
		},
		{
			name:         "colonize",
			manifestName: "colonize_manifest",
			load: func(path string) error {
				_, err := loadExternalColonizeCompletion(path)
				return err
			},
		},
		{
			name:         "oracle",
			manifestName: "iteration_manifest",
			load: func(path string) error {
				_, err := loadExternalOracleIterationCompletion(path)
				return err
			},
		},
		{
			name:         "swarm",
			manifestName: "swarm_manifest",
			load: func(path string) error {
				_, err := loadExternalSwarmCompletion(path)
				return err
			},
		},
		{
			name:         "seal",
			manifestName: "seal_manifest",
			load: func(path string) error {
				_, err := loadExternalSealCompletion(path)
				return err
			},
		},
	}

	for _, tc := range loaders {
		t.Run(tc.name+"/missing_file", func(t *testing.T) {
			err := tc.load(filepath.Join(t.TempDir(), "missing-completion.json"))
			if err == nil {
				t.Fatal("expected missing completion file to be rejected")
			}
			if !strings.Contains(err.Error(), "read completion file") {
				t.Fatalf("expected read completion file error, got: %v", err)
			}
		})

		t.Run(tc.name+"/malformed_json", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "completion.json")
			if err := os.WriteFile(path, []byte("{not valid json}"), 0o644); err != nil {
				t.Fatalf("write malformed completion: %v", err)
			}
			err := tc.load(path)
			if err == nil {
				t.Fatal("expected malformed completion file to be rejected")
			}
			if !strings.Contains(err.Error(), "parse completion file") {
				t.Fatalf("expected parse completion file error, got: %v", err)
			}
		})

		t.Run(tc.name+"/missing_manifest", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "completion.json")
			if err := os.WriteFile(path, []byte(`{"dispatches":[{"name":"Worker-1","status":"completed"}]}`), 0o644); err != nil {
				t.Fatalf("write manifestless completion: %v", err)
			}
			err := tc.load(path)
			if err == nil {
				t.Fatal("expected manifestless completion file to be rejected")
			}
			if !strings.Contains(err.Error(), tc.manifestName) {
				t.Fatalf("expected %s error, got: %v", tc.manifestName, err)
			}
		})
	}
}

func TestLifecycleFinalizerCommandsReturnFailureForBadCompletionFiles(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{name: "plan", args: []string{"plan-finalize", "--completion-file"}},
		{name: "build", args: []string{"build-finalize", "1", "--completion-file"}},
		{name: "continue", args: []string{"continue-finalize", "--completion-file"}},
		{name: "colonize", args: []string{"colonize-finalize", "--completion-file"}},
		{name: "oracle", args: []string{"oracle-iterate-finalize", "--completion-file"}},
		{name: "swarm", args: []string{"swarm-finalize", "--completion-file"}},
		{name: "seal", args: []string{"seal-finalize", "--completion-file"}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			t.Setenv("AETHER_OUTPUT_MODE", "json")
			root := t.TempDir()
			t.Setenv("AETHER_ROOT", root)
			withWorkingDir(t, root)

			var outBuf, errBuf bytes.Buffer
			stdout, stderr = &outBuf, &errBuf
			missingPath := filepath.Join(root, "missing-completion.json")
			args := append([]string{}, tc.args...)
			args = append(args, missingPath)

			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("%s finalizer returned nil error for missing completion file", tc.name)
			}
			if !strings.Contains(errBuf.String(), "read completion file") {
				t.Fatalf("%s finalizer stderr missing read failure, got stdout=%q stderr=%q", tc.name, outBuf.String(), errBuf.String())
			}
		})
	}
}

func TestLifecycleFinalizerLoadersRejectAetherDataCompletionFiles(t *testing.T) {
	loaders := []struct {
		name string
		load func(string) error
	}{
		{name: "plan", load: func(path string) error { _, err := loadExternalPlanCompletion(path); return err }},
		{name: "build", load: func(path string) error { _, err := loadExternalBuildCompletion(path); return err }},
		{name: "continue", load: func(path string) error { _, err := loadExternalContinueCompletion(path); return err }},
		{name: "colonize", load: func(path string) error { _, err := loadExternalColonizeCompletion(path); return err }},
		{name: "oracle", load: func(path string) error { _, err := loadExternalOracleIterationCompletion(path); return err }},
		{name: "swarm", load: func(path string) error { _, err := loadExternalSwarmCompletion(path); return err }},
		{name: "seal", load: func(path string) error { _, err := loadExternalSealCompletion(path); return err }},
	}

	for _, tc := range loaders {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.load(filepath.Join(".aether", "data", tc.name+"-completion.json"))
			if err == nil {
				t.Fatal("expected .aether/data completion file to be rejected")
			}
			if !strings.Contains(err.Error(), "outside .aether/data") || !strings.Contains(err.Error(), "approved temp") {
				t.Fatalf("expected approved temp path guidance, got: %v", err)
			}
		})
	}
}

func TestLifecycleFinalizerLoadersRejectStdinCompletionFiles(t *testing.T) {
	loaders := []struct {
		name string
		load func(string) error
	}{
		{name: "plan", load: func(path string) error { _, err := loadExternalPlanCompletion(path); return err }},
		{name: "build", load: func(path string) error { _, err := loadExternalBuildCompletion(path); return err }},
		{name: "continue", load: func(path string) error { _, err := loadExternalContinueCompletion(path); return err }},
		{name: "colonize", load: func(path string) error { _, err := loadExternalColonizeCompletion(path); return err }},
		{name: "oracle", load: func(path string) error { _, err := loadExternalOracleIterationCompletion(path); return err }},
		{name: "swarm", load: func(path string) error { _, err := loadExternalSwarmCompletion(path); return err }},
		{name: "seal", load: func(path string) error { _, err := loadExternalSealCompletion(path); return err }},
	}

	for _, tc := range loaders {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.load("-")
			if err == nil {
				t.Fatal("expected stdin completion file to be rejected")
			}
			if !strings.Contains(err.Error(), "cannot be read from stdin") || !strings.Contains(err.Error(), "approved temp") {
				t.Fatalf("expected approved temp path guidance, got: %v", err)
			}
		})
	}
}

func TestFinalizerManifestFreshnessRejectsStaleAndFutureManifests(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		generatedAt string
		want        string
	}{
		{name: "missing", generatedAt: "", want: "generated_at is required"},
		{name: "stale", generatedAt: now.Add(-25 * time.Hour).Format(time.RFC3339), want: "stale dispatch_manifest"},
		{name: "future", generatedAt: now.Add(10 * time.Minute).Format(time.RFC3339), want: "too far in the future"},
		{name: "invalid", generatedAt: "not-a-time", want: "not RFC3339"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFinalizerManifestFreshness("dispatch_manifest", tc.generatedAt, now)
			if err == nil {
				t.Fatal("expected freshness error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
	if err := validateFinalizerManifestFreshness("dispatch_manifest", now.Add(-time.Hour).Format(time.RFC3339), now); err != nil {
		t.Fatalf("fresh manifest should pass: %v", err)
	}
}

func TestLifecycleResultMergersPreferCompletedResultOverTimeoutPlaceholder(t *testing.T) {
	t.Run("build", func(t *testing.T) {
		manifest := codexBuildManifest{PlanOnly: true, Dispatches: []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1"}}}
		results := []codexExternalBuildWorkerResult{
			{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "timeout", Summary: "timeout placeholder"},
			{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "completed", Summary: "valid result", FilesModified: []string{"cmd/codex_build_finalize.go"}},
		}
		dispatches, violations, err := mergeExternalBuildResults(manifest, results)
		if err != nil {
			t.Fatalf("mergeExternalBuildResults: %v", err)
		}
		if len(violations) != 0 {
			t.Fatalf("expected no violations, got %+v", violations)
		}
		if dispatches[0].Status != "completed" || dispatches[0].Summary != "valid result" {
			t.Fatalf("expected completed result to win over timeout, got %+v", dispatches[0])
		}
	})

	t.Run("plan", func(t *testing.T) {
		manifest := codexPlanManifest{Dispatches: []codexPlanningDispatch{{Name: "Route-1", Caste: "route_setter", Stage: "routing", TaskID: "plan-route"}}}
		results := []codexPlanningDispatch{
			{Name: "Route-1", Caste: "route_setter", Stage: "routing", TaskID: "plan-route", Status: "timeout", Summary: "timeout placeholder"},
			{Name: "Route-1", Caste: "route_setter", Stage: "routing", TaskID: "plan-route", Status: "completed", Summary: "valid route", PhasePlan: testWorkerPlanArtifact()},
		}
		dispatches, err := mergeExternalPlanResults(manifest, results)
		if err != nil {
			t.Fatalf("mergeExternalPlanResults: %v", err)
		}
		if dispatches[0].Status != "completed" || dispatches[0].Summary != "valid route" {
			t.Fatalf("expected completed planning result to win over timeout, got %+v", dispatches[0])
		}
	})

	t.Run("continue", func(t *testing.T) {
		plan := codexContinuePlanManifest{Dispatches: []codexContinueExternalDispatch{{Name: "Hawk-1", Caste: "watcher", Stage: "review", Task: "verify", TaskID: "review"}}}
		results := []codexContinueExternalDispatch{
			{Name: "Hawk-1", Caste: "watcher", Stage: "review", Task: "verify", TaskID: "review", Status: "timeout", Summary: "timeout placeholder"},
			{Name: "Hawk-1", Caste: "watcher", Stage: "review", Task: "verify", TaskID: "review", Status: "completed", Summary: "valid review"},
		}
		flow, err := mergeExternalContinueResults(plan, results)
		if err != nil {
			t.Fatalf("mergeExternalContinueResults: %v", err)
		}
		if flow[0].Status != "completed" || flow[0].Summary != "valid review" {
			t.Fatalf("expected completed continue result to win over timeout, got %+v", flow[0])
		}
	})
}

func TestLifecycleResultMergersRejectDuplicateTerminalResults(t *testing.T) {
	manifest := codexBuildManifest{PlanOnly: true, Dispatches: []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1"}}}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "completed", FilesModified: []string{"cmd/codex_build_finalize.go"}},
		{Name: "Mason-1", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "completed", FilesModified: []string{"cmd/codex_build_finalize_test.go"}},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for duplicate completed results, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected duplicate completed results to be rejected as exactly 1 violation, got %d: %+v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "duplicate external worker result") {
		t.Fatalf("expected duplicate result violation message, got: %v", violations[0].Message)
	}
}
