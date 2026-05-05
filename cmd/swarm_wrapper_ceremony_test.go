package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwarmWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "swarm.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "swarm.md"),
	}

	required := []string{
		"## Watch Mode",
		"AETHER_OUTPUT_MODE=visual aether swarm --watch",
		"## Swarm Manifest",
		"AETHER_OUTPUT_MODE=json aether swarm --plan-only $ARGUMENTS",
		"result.swarm_manifest",
		"dispatch_mode: agent-delegate",
		"## Wave Execution",
		"Wave 1 investigation workers may run together.",
		"Wave 2 builder waits for wave 1 summaries.",
		"Wave 3 watcher waits for builder completion.",
		`subagent_type="{agent_name}"`,
		"AETHER_OUTPUT_MODE=json aether spawn-log",
		"AETHER_OUTPUT_MODE=json aether spawn-complete",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file",
		"## After Swarm",
		"Do NOT run nested subprocess swarm workers",
		"Do NOT run `aether swarm` without `--plan-only`",
	}

	inOrder := []string{
		"## Watch Mode",
		"AETHER_OUTPUT_MODE=visual aether swarm --watch",
		"## Swarm Manifest",
		"AETHER_OUTPUT_MODE=json aether swarm --plan-only $ARGUMENTS",
		"## Wave Execution",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file",
		"## After Swarm",
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
			"To launch the stubborn bug-destroyer flow, execute `AETHER_OUTPUT_MODE=visual aether swarm \"$ARGUMENTS\"` directly.",
			"AETHER_OUTPUT_MODE=visual aether swarm \"$ARGUMENTS\"",
			"manually reconstruct swarm artifacts",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old swarm pass-through contract %q", wrapperPath, forbidden)
			}
		}
	}
}
