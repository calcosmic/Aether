package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuilderAgentsIncludeReadCacheDiscipline(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".claude/agents/ant/aether-builder.md",
		".opencode/agents/aether-builder.md",
		".codex/agents/aether-builder.toml",
		".aether/codex/aether-builder.toml",
	} {
		assertReadCacheDiscipline(t, filepath.Join(repoRoot, filepath.FromSlash(rel)), rel)
	}
}

func TestLifecycleWorkerBriefsIncludeReadCacheDiscipline(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".aether/docs/command-playbooks/build-full.md",
		".aether/docs/command-playbooks/build-wave.md",
		".aether/docs/command-playbooks/build-verify.md",
		".aether/docs/command-playbooks/continue-verify.md",
		".aether/docs/command-playbooks/continue-gates.md",
		".aether/docs/command-playbooks/continue-full.md",
	} {
		assertReadCacheDiscipline(t, filepath.Join(repoRoot, filepath.FromSlash(rel)), rel)
	}
}

func TestLifecycleOrchestratorsMentionReadCacheLoopHandling(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".aether/skills/colony/aether-colony-build-cycle/SKILL.md",
		".aether/commands/continue.yaml",
		".claude/commands/ant/continue.md",
		".opencode/commands/ant/continue.md",
	} {
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(content)
		for _, want := range []string{"read cache discipline", "re-reading the same unchanged", "blocked"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing read-cache loop handling marker %q", rel, want)
			}
		}
	}
}

func TestRuntimeBuildAndContinueBriefsIncludeReadCacheDiscipline(t *testing.T) {
	phase := colony.Phase{
		ID:          2,
		Name:        "Card redesign",
		Description: "Make card layouts denser.",
		Tasks: []colony.Task{{
			Goal: "Update CardNode spacing",
		}},
	}
	dispatch := codexBuildDispatch{
		Name:  "Brick-79",
		Caste: "builder",
		Task:  "Redesign CardNode wrapper",
	}

	buildBrief := renderCodexBuildWorkerBrief("/tmp/repo", phase, dispatch, nil, time.Now())
	assertReadCacheText(t, buildBrief, "renderCodexBuildWorkerBrief")

	reviewBrief := renderCodexContinueReviewBrief(
		"/tmp/repo",
		phase,
		codexContinueManifest{},
		codexContinueVerificationReport{},
		codexContinueAssessment{},
		codexContinueReviewSpec{Caste: "probe", Task: "Probe the implementation evidence."},
	)
	assertReadCacheText(t, reviewBrief, "renderCodexContinueReviewBrief")

	watcherBrief := renderCodexContinueWatcherBrief(
		"/tmp/repo",
		phase,
		codexContinueManifest{},
		nil,
		codexClaimVerification{},
		codexWatcherVerification{},
		time.Minute,
	)
	assertReadCacheText(t, watcherBrief, "renderCodexContinueWatcherBrief")
}

func assertReadCacheDiscipline(t *testing.T, path string, label string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	assertReadCacheText(t, string(content), label)
}

func assertReadCacheText(t *testing.T, text string, label string) {
	t.Helper()
	for _, want := range []string{
		"Read Cache Discipline",
		"File unchanged since last read",
		"Do not re-read",
		"Do not loop full-file reads",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s missing read cache discipline marker %q", label, want)
		}
	}
}
