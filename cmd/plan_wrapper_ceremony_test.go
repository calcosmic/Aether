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
		"Use the Go `aether` CLI as the source of truth.",
		"## Approved Specification Preflight",
		"AETHER_OUTPUT_MODE=json aether spec --inspect",
		"current revision is `APPROVED`",
		"Specification approval never counts as plan acceptance.",
		"## Choose Planning Preset",
		"preset_required",
		"preset_options",
		"aether host plan --preset <fast|balanced|deep|exhaustive>",
		planWrapperHostSpineCompatibility,
		"temporary manifest file outside `.aether/data/`",
		"result.plan_manifest",
		"result.planning_manifest",
		"orchestrator_boundary_guidance",
		"after_discuss_next",
		"aether discuss",
		"unresolved_clarifications",
		"/ant-discuss",
		"## Scout Stage",
		"Exactly one visible Scout",
		"plan_manifest.stage_manifest",
		"scout_result",
		"stage_receipt",
		"## First-Pass Owner Decision Boundary",
		"decision_batch",
		"decision_cards",
		"decision_resume_token",
		"route_stage_manifest",
		"## Route-Setter Stage",
		"Exactly one visible Route-Setter",
		"route_result",
		"route_stage_receipt",
		"## Iteration Card and Timeline",
		"iteration_card",
		"planning_projection",
		"## Continue, Pause, or Stop",
		"scout_stage_manifest",
		"plan_candidate",
		"## Candidate Review",
		"AETHER_OUTPUT_MODE=json aether plan --candidate",
		"[NOT ACTIVE]",
		"## Exact Candidate Acceptance",
		"acceptance_command",
		"aether plan --accept-candidate <candidate-id>",
		"acceptance_receipt",
		"AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <completion_file>",
		"AETHER_OUTPUT_MODE=json aether spawn-log",
		"AETHER_OUTPUT_MODE=json aether spawn-complete",
		"Do not widen permissions, set background execution, or add owner steering.",
		"**Purpose:**",
		"**Reads:**",
		"**Spawns:**",
		"**Stop conditions:**",
		"<success_criteria>",
		"<failure_modes>",
		"<read_only>",
		"aether command-guide plan --platform codex",
	}

	inOrder := []string{
		"## Approved Specification Preflight",
		"AETHER_OUTPUT_MODE=json aether spec --inspect",
		"## Choose Planning Preset",
		"aether host plan",
		"## Scout Stage",
		"scout_result",
		"## First-Pass Owner Decision Boundary",
		"route_stage_manifest",
		"## Route-Setter Stage",
		"route_result",
		"## Iteration Card and Timeline",
		"iteration_card",
		"## Continue, Pause, or Stop",
		"## Candidate Review",
		"aether plan --candidate",
		"## Exact Candidate Acceptance",
		"acceptance_command",
		"aether plan --accept-candidate",
		"acceptance_receipt",
		"## Guardrails",
	}

	retired := []string{
		"## Decision Moment",
		"depth_proposal_card",
		"research_proposal_card",
		"research_awaiting_approval",
		"research_warning",
		"plan-research-approve",
		"aether host plan --depth <choice> --planning-depth <choice>",
		"--verification-depth",
		"result.requires_next_iteration",
		"scout_report",
		"phase_plan",
		"Execute `AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS` directly.",
		"AETHER_OUTPUT_MODE=visual aether plan $ARGUMENTS",
		"Update watch files for tmux visibility",
		"Write COLONY_STATE.json",
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
		for _, forbidden := range retired {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains retired or unsafe plan contract %q", wrapperPath, forbidden)
			}
		}
	}
}

// TestPlanWrapperStageSkeleton protects the stage-based, thin-host shape:
// every non-exempt section explains its purpose, structured result blocks stay
// present, all public stop reasons remain explicit, and method prose outweighs
// low-level envelope bookkeeping.
func TestPlanWrapperStageSkeleton(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "plan.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant-plan.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"),
	}
	exemptHeadings := []string{
		"## Required Cross-Stage State",
		"## Cross-Platform Drift Guard",
		"## Guardrails",
	}

	t.Run("stage_skeleton_density", func(t *testing.T) {
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertStageSkeletonDensity(t, path, string(content), exemptHeadings)
		}
	})

	t.Run("structured_blocks_present", func(t *testing.T) {
		required := []string{"<success_criteria>", "<failure_modes>", "<read_only>", "## Required Cross-Stage State"}
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, want := range required {
				if !strings.Contains(string(content), want) {
					t.Errorf("%s missing structured block %q", path, want)
				}
			}
		}
	})

	t.Run("reasoned_stop_conditions_documented", func(t *testing.T) {
		concepts := []string{"Target sufficiency", "diminishing returns", "stall detected", "max iteration cap", "NOT ACTIVE"}
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, concept := range concepts {
				if !strings.Contains(string(content), concept) {
					t.Errorf("%s missing stop/candidate concept %q", path, concept)
				}
			}
		}
	})

	t.Run("method_outweighs_envelope_mechanics", func(t *testing.T) {
		methodMarkers := stageSkeletonMarkers()
		envelopeMarkers := envelopeMechanicsMarkers()
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			methodCount := countMarkerLines(text, methodMarkers)
			envelopeCount := countMarkerLines(text, envelopeMarkers)
			if methodCount < envelopeCount*3 {
				t.Errorf("%s: stage-skeleton marker lines (%d) must be at least 3x envelope-mechanics marker lines (%d)", path, methodCount, envelopeCount)
			}
		}
	})
}
