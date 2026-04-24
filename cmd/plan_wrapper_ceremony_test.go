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
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"),
	}

	required := []string{
		"## Depth Ceremony",
		"Fast — sprint granularity, 1-3 phases",
		"Balanced — milestone granularity, 4-7 phases. Recommended default",
		"Deep — quarter granularity, 8-12 phases",
		"Exhaustive — major granularity, 13-20 phases",
		"AETHER_OUTPUT_MODE=visual aether status",
		"## Planning Manifest",
		"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
		"result.plan_manifest",
		"result.planning_manifest",
		"## Wave Execution",
		"Wave 1 Scout must complete before wave 2 Route-Setter starts.",
		`subagent_type="{agent_name}"`,
		"AETHER_OUTPUT_MODE=json aether spawn-log",
		"AETHER_OUTPUT_MODE=json aether spawn-complete",
		"phase_plan",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
		"## After Planning",
		"/ant-build 1",
		"Do NOT run `aether plan` without `--plan-only` from this wrapper.",
		"Do NOT run `aether plan --synthetic` after real agent workers complete.",
	}

	inOrder := []string{
		"## Depth Ceremony",
		"## Colony Context",
		"## Planning Manifest",
		"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
		"## Wave Execution",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
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
			"Update watch files for tmux visibility",
			"Write COLONY_STATE.json",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old plan pass-through contract %q", wrapperPath, forbidden)
			}
		}
	}
}
