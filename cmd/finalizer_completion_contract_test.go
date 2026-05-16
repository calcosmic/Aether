package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
