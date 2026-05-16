package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type goreleaserConfig struct {
	Before struct {
		Hooks []string `yaml:"hooks"`
	} `yaml:"before"`
}

type githubWorkflowConfig struct {
	Permissions map[string]string            `yaml:"permissions"`
	Jobs        map[string]githubWorkflowJob `yaml:"jobs"`
}

type githubWorkflowJob struct {
	If          string               `yaml:"if"`
	Outputs     map[string]string    `yaml:"outputs"`
	Permissions map[string]string    `yaml:"permissions"`
	Steps       []githubWorkflowStep `yaml:"steps"`
}

type githubWorkflowStep struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
}

func TestGoReleaserBeforeHooksGuardGoModDiff(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yml"))
	if err != nil {
		t.Fatalf("failed to read .goreleaser.yml: %v", err)
	}

	var cfg goreleaserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse .goreleaser.yml: %v", err)
	}

	tidyIndex := -1
	guardIndex := -1
	for index, hook := range cfg.Before.Hooks {
		normalized := strings.Join(strings.Fields(hook), " ")
		if normalized == "go mod tidy" {
			tidyIndex = index
			continue
		}
		if strings.Contains(normalized, "git diff") &&
			strings.Contains(normalized, "--exit-code") &&
			strings.Contains(normalized, "go.mod") &&
			strings.Contains(normalized, "go.sum") {
			guardIndex = index
		}
	}

	if tidyIndex == -1 {
		t.Fatal("before.hooks missing go mod tidy hook")
	}
	if guardIndex == -1 {
		t.Fatal("before.hooks missing go.mod/go.sum git diff guard")
	}
	if guardIndex <= tidyIndex {
		t.Fatalf("go.mod/go.sum git diff guard must run after go mod tidy: tidy index %d, guard index %d", tidyIndex, guardIndex)
	}
}

func TestReleaseWorkflowsVerifyTSHostProviderAuth(t *testing.T) {
	tests := []struct {
		rel            string
		job            string
		mustPrecedeAny []string
	}{
		{
			rel: filepath.Join(".github", "workflows", "ci.yml"),
			job: "go",
		},
		{
			rel: filepath.Join(".github", "workflows", "release.yml"),
			job: "goreleaser",
			mustPrecedeAny: []string{
				"Build GoReleaser snapshot",
				"Run GoReleaser",
			},
		},
	}
	for _, tt := range tests {
		rel := tt.rel
		t.Run(rel, func(t *testing.T) {
			workflow := readGitHubWorkflow(t, rel)
			job, ok := workflow.Jobs[tt.job]
			if !ok {
				t.Fatalf("%s missing job %q", rel, tt.job)
			}
			stepIndex := workflowStepIndex(job, "Verify TypeScript host package")
			if stepIndex == -1 {
				t.Fatalf("%s missing Verify TypeScript host package step", rel)
			}
			runText := job.Steps[stepIndex].Run
			for _, want := range []string{
				"npm --prefix .aether/ts-host ci",
				"npm --prefix .aether/ts-host run typecheck",
				"npm --prefix .aether/ts-host test",
				"npm --prefix .aether/ts-host run build",
			} {
				if !strings.Contains(runText, want) {
					t.Fatalf("%s missing TS-host release verification command %q", rel, want)
				}
			}
			for _, laterStep := range tt.mustPrecedeAny {
				laterIndex := workflowStepIndex(job, laterStep)
				if laterIndex == -1 {
					t.Fatalf("%s missing %q step", rel, laterStep)
				}
				if stepIndex >= laterIndex {
					t.Fatalf("%s TS-host verification must run before %q: got %d >= %d", rel, laterStep, stepIndex, laterIndex)
				}
			}
		})
	}
}

func TestReleaseWorkflowKeepsSecretChecksOutOfJobConditionals(t *testing.T) {
	workflow := readGitHubWorkflow(t, filepath.Join(".github", "workflows", "release.yml"))
	for name, job := range workflow.Jobs {
		if strings.Contains(job.If, "secrets.") {
			t.Fatalf("release job %s references secrets directly in job if: %s", name, job.If)
		}
	}

	goreleaser, ok := workflow.Jobs["goreleaser"]
	if !ok {
		t.Fatal("release workflow missing goreleaser job")
	}
	if goreleaser.Outputs["npm_token_available"] == "" {
		t.Fatal("goreleaser job must expose npm_token_available output")
	}
	if workflow.Permissions["contents"] != "read" {
		t.Fatalf("release workflow default contents permission = %q, want read", workflow.Permissions["contents"])
	}
	if goreleaser.Permissions["contents"] != "write" {
		t.Fatalf("goreleaser job contents permission = %q, want write", goreleaser.Permissions["contents"])
	}

	npm, ok := workflow.Jobs["npm-bootstrap"]
	if !ok {
		t.Fatal("release workflow missing npm-bootstrap job")
	}
	if !strings.Contains(npm.If, "needs.goreleaser.outputs.npm_token_available == 'true'") {
		t.Fatalf("npm-bootstrap job if does not use npm_token_available output: %s", npm.If)
	}
	if !strings.Contains(npm.If, "needs.goreleaser.outputs.publish_release == 'true'") {
		t.Fatalf("npm-bootstrap job if does not preserve dry-run publish_release guard: %s", npm.If)
	}
}

func readGitHubWorkflow(t *testing.T, rel string) githubWorkflowConfig {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("failed to read %s: %v", rel, err)
	}
	var workflow githubWorkflowConfig
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("failed to parse %s: %v", rel, err)
	}
	return workflow
}

func workflowStepIndex(job githubWorkflowJob, name string) int {
	for index, step := range job.Steps {
		if step.Name == name {
			return index
		}
	}
	return -1
}
