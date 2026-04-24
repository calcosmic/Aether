package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleCommandDocsPreferRuntimeCLI(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	testCases := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: ".claude/commands/ant/status.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write upgraded state:",
				"Use the Read tool",
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/focus.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"aether focus \"$ARGUMENTS\"",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write constraints.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/feedback.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"aether feedback \"$ARGUMENTS\"",
			},
			forbidden: []string{
				"append to `memory.instincts`",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/redirect.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"aether redirect \"$ARGUMENTS\"",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write constraints.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/init.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth, but do not skip the init",
				"AETHER_OUTPUT_MODE=json aether init-research --goal",
				"AETHER_OUTPUT_MODE=visual aether init",
				"AskUserQuestion with 3 options: proceed, revise goal, cancel.",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				"queen-init",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/plan.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"## Depth Ceremony",
				"AETHER_OUTPUT_MODE=visual aether status",
				"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
				"result.plan_manifest",
				"result.planning_manifest",
				"## Clarification Gate",
				"/ant-discuss",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
			},
			forbidden: []string{
				"AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Update watch files for tmux visibility",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/discuss.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether discuss",
				"/ant-plan",
				"/ant-council",
				"lightweight pre-plan clarification gate",
			},
			forbidden: []string{
				"Write COLONY_STATE.json",
				"Write pheromones.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/phase.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether phase",
			},
			forbidden: []string{
				"Use the Read tool to read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/oracle.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether oracle",
			},
			forbidden: []string{
				"tmux",
				".aether/oracle/.stop",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/patrol.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether patrol",
			},
			forbidden: []string{
				"Read these files in parallel using the Read tool:",
				"completion-report.md is written",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/resume.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether resume",
			},
			forbidden: []string{
				"Use the Read tool to read `.aether/data/COLONY_STATE.json`.",
				"State file missing or corrupted.",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/resume-colony.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether resume-colony",
			},
			forbidden: []string{
				"Run using the Bash tool with description \"Restoring colony session...\": `aether load-state`",
				"Use Write tool to update COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/pause-colony.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether pause-colony",
			},
			forbidden: []string{
				"Use the Read tool to read `.aether/data/COLONY_STATE.json`.",
				"Use Write tool to update COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/build.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
				"AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only",
				".aether/docs/command-playbooks/build-wave.md",
				"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
			},
			forbidden: []string{
				"Briefly name the castes the colony is likely to send",
				"such as Builder, Watcher, Scout, Architect, Oracle, or Chaos",
				"AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS",
				"Read the file with the Read tool.",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/continue.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
				"## Verification Gates",
				"## Learning Extraction",
				"AETHER_OUTPUT_MODE=json aether continue --plan-only $ARGUMENTS",
				"result.continue_manifest",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
			},
			forbidden: []string{
				"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS",
				"It's safe to clear your context now.",
				"/ant-resume",
				"continue-verify.md",
				"continue-gates.md",
				"Read build packet files:",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/pheromones.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether pheromones",
				"aether signal-housekeeping",
				"aether pheromone-expire --id",
				"aether focus",
				"aether redirect",
				"aether feedback",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				".aether/data/pheromones.json",
				"jq",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/colonize.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether colonize $ARGUMENTS",
			},
			forbidden: []string{
				"Write a minimal COLONY_STATE.json",
				"Queen dispatching Surveyor Ants",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/seal.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS",
			},
			forbidden: []string{
				"CROWNED-ANTHILL.md",
				"AskUserQuestion",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/entomb.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS",
			},
			forbidden: []string{
				"Archive ALL colony data",
				"AskUserQuestion",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/run.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether run $ARGUMENTS",
			},
			forbidden: []string{
				"build-prep.md",
				"pending-decision-add",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/update.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether update $ARGUMENTS",
			},
			forbidden: []string{
				"Checking Aether hub",
				"rm -f .aether/data/.version-check-cache",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/watch.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether watch",
			},
			forbidden: []string{
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/help.md",
			required: []string{
				"runtime CLI and current slash-command surface as the source of truth.",
				"aether --help",
			},
			forbidden: []string{
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".claude/commands/ant/council.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"aether council-deliberate --topic",
			},
			forbidden: []string{
				"Write constraints.json",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/status.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write upgraded state:",
				"Use the Read tool",
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/init.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth, but do not skip the init",
				"AETHER_OUTPUT_MODE=json aether init-research --goal",
				"AETHER_OUTPUT_MODE=visual aether init",
				"Ask with 3 options: proceed, revise goal, cancel.",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				"queen-init",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/plan.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"## Depth Ceremony",
				"AETHER_OUTPUT_MODE=visual aether status",
				"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
				"result.plan_manifest",
				"result.planning_manifest",
				"## Clarification Gate",
				"/ant-discuss",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file",
			},
			forbidden: []string{
				"AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Update watch files for tmux visibility",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/discuss.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether discuss",
				"/ant-plan",
				"/ant-council",
				"lightweight pre-plan clarification gate",
			},
			forbidden: []string{
				"Write COLONY_STATE.json",
				"Write pheromones.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/phase.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether phase",
			},
			forbidden: []string{
				"Use the Read tool to read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/oracle.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether oracle",
			},
			forbidden: []string{
				"tmux",
				".aether/oracle/.stop",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/patrol.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether patrol",
			},
			forbidden: []string{
				"Read these files in parallel using the Read tool:",
				"completion-report.md is written",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/resume.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether resume",
			},
			forbidden: []string{
				"Use the Read tool to read `.aether/data/COLONY_STATE.json`.",
				"State file missing or corrupted.",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/resume-colony.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether resume-colony",
			},
			forbidden: []string{
				"Use Write tool to update COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/pause-colony.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether pause-colony",
			},
			forbidden: []string{
				"Use Write tool to update COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/build.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
				"AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only",
				".aether/docs/command-playbooks/build-wave.md",
				"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
			},
			forbidden: []string{
				"AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS",
				"Read the file with the Read tool.",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/continue.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether status",
				"## Verification Gates",
				"## Learning Extraction",
				"AETHER_OUTPUT_MODE=json aether continue --plan-only $ARGUMENTS",
				"result.continue_manifest",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
			},
			forbidden: []string{
				"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS",
				"It's safe to clear your context now.",
				"/ant-resume",
				"continue-verify.md",
				"continue-gates.md",
				"Read `.aether/data/COLONY_STATE.json`.",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/pheromones.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether pheromones",
				"aether signal-housekeeping",
				"aether pheromone-expire --id",
				"aether focus",
				"aether redirect",
				"aether feedback",
			},
			forbidden: []string{
				"Read `.aether/data/COLONY_STATE.json`.",
				".aether/data/pheromones.json",
				"jq",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/colonize.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether colonize $ARGUMENTS",
			},
			forbidden: []string{
				"Write a minimal COLONY_STATE.json",
				"Spawn 4 Surveyor Ants in parallel",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/seal.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether seal $ARGUMENTS",
			},
			forbidden: []string{
				"archive_dir",
				"manually update milestone via COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/entomb.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS",
			},
			forbidden: []string{
				"manifest.json (pheromone trails)",
				"command -v xmllint",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/run.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether run $ARGUMENTS",
			},
			forbidden: []string{
				"build-prep.md",
				"pending-decision-add",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/update.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether update $ARGUMENTS",
			},
			forbidden: []string{
				"test -f ~/.aether/version.json",
				"rm -f .aether/data/.version-check-cache",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/watch.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"AETHER_OUTPUT_MODE=visual aether watch",
			},
			forbidden: []string{
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/help.md",
			required: []string{
				"Use the runtime CLI as the source of truth.",
				"aether --help",
			},
			forbidden: []string{
				"tmux",
				".aether/aether-utils.sh",
			},
		},
		{
			path: ".opencode/commands/ant/council.md",
			required: []string{
				"Use the Go `aether` CLI as the source of truth.",
				"aether council-deliberate --topic",
			},
			forbidden: []string{
				"Write constraints.json",
				"Write COLONY_STATE.json",
				".aether/aether-utils.sh",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, tc.path))
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			text := string(data)
			for _, needle := range tc.required {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s missing required text %q", tc.path, needle)
				}
			}
			for _, needle := range tc.forbidden {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains forbidden text %q", tc.path, needle)
				}
			}
		})
	}
}

func TestInterpretDocsStayReadOnly(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".claude/commands/ant/interpret.md",
		".opencode/commands/ant/interpret.md",
	} {
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, rel))
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			text := string(data)
			required := []string{
				"This command is read-only.",
				"Do not auto-inject pheromones or create action items.",
			}
			forbidden := []string{
				"append to `constraints.json`",
				"Write constraints.json",
				"append to `TO-DOS.md`",
				"Use **AskUserQuestion**",
			}
			for _, needle := range required {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s missing required text %q", rel, needle)
				}
			}
			for _, needle := range forbidden {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains forbidden text %q", rel, needle)
				}
			}
		})
	}
}

func TestOpenCodeAgentDocsAvoidLegacyShellHelpers(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	agentPaths, err := filepath.Glob(filepath.Join(repoRoot, ".opencode/agents/*.md"))
	if err != nil {
		t.Fatalf("glob agent docs: %v", err)
	}
	if len(agentPaths) == 0 {
		t.Fatal("no OpenCode agent docs found")
	}

	for _, abs := range agentPaths {
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			t.Fatalf("relative path for %s: %v", abs, err)
		}
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(abs)
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			text := string(data)
			if strings.Contains(text, ".aether/aether-utils.sh") {
				t.Fatalf("%s still references legacy shell helper path", rel)
			}
		})
	}
}
