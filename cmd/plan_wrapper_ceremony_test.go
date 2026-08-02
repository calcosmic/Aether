package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "plan.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant-plan.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"),
	}

	required := []string{
		"## Decision Moment 1 — Depth Proposal",
		"result.depth_proposal_card",
		"The plan flow has exactly two decision moments",
		"Never ask the user to type a value.",
		"## Planning Manifest",
		"aether host plan --depth <choice> --planning-depth <choice2> --verification-depth <choice3> $ARGUMENTS",
		"The TS host is the sole entry point to the Go CLI for manifest generation.",
		"temporary manifest file outside `.aether/data/`",
		"result.plan_manifest",
		"result.planning_manifest",
		"## Decision Moment 2 — Research Batch",
		"result.research_proposal_card",
		"aether plan-research-approve --approve-all",
		"aether plan-research-approve --auto",
		"result.research_awaiting_approval",
		"result.research_warning",
		"## Clarification Gate",
		"unresolved_clarifications",
		"/ant-discuss",
		"implicit assumptions",
		"Do not set `run_in_background`",
		"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file",
		"## After Planning",
		"/ant-build 1",
		"Do NOT run direct `aether plan` from this wrapper for manifest generation; use `aether host plan`.",
		"Do NOT run `aether plan --synthetic` after real agent workers complete.",
		"Do NOT add a third decision moment",
	}

	inOrder := []string{
		"## Decision Moment 1 — Depth Proposal",
		"## Planning Manifest",
		"aether host plan",
		"## Decision Moment 2 — Research Batch",
		"## Clarification Gate",
		"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file",
		"## After Planning",
		"## Guardrails",
	}

	for _, wrapperPath := range wrapperPaths {
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}
		text := string(content)
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", wrapperPath, want)
			}
		}
		assertSubstringsInOrder(t, wrapperPath, text, inOrder)
		for _, forbidden := range []string{
			"Execute `AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS` directly.",
			"AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS",
			"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> $ARGUMENTS",
			"Do NOT run `aether plan` without `--plan-only` from this wrapper.",
			"Update watch files for tmux visibility",
			"Write COLONY_STATE.json",
			"## Depth Ceremony",
			"## Planning Depth",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old plan pass-through contract %q", wrapperPath, forbidden)
			}
		}
	}
}
