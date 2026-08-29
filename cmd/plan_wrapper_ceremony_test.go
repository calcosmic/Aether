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
		"the exact same screen a direct `aether plan-finalize` run at the",
		"## After Planning",
		"/ant-build 1",
		"Do NOT run direct `aether plan` from this wrapper for manifest generation; use `aether host plan`.",
		"Do NOT run `aether plan --synthetic` after real agent workers complete.",
		"Do NOT add a third decision moment",
		"## Required Cross-Stage State",
		"**Purpose:**",
		"**Reads:**",
		"**Spawns:**",
		"**Stop conditions:**",
		"<success_criteria>",
		"<failure_modes>",
		"<read_only>",
	}

	// The Clarification Gate deliberately precedes Decision Moment 2 — Phase
	// 165 review WR-04: aether plan-research-approve mutates approval state,
	// and the gate may route to aether discuss, which discards the very
	// manifest that state was approved against. A future edit must not
	// quietly restore the hazardous order and call it a cleanup.
	inOrder := []string{
		"## Decision Moment 1 — Depth Proposal",
		"## Planning Manifest",
		"aether host plan",
		"## Clarification Gate",
		"## Decision Moment 2 — Research Batch",
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

// TestPlanWrapperStageSkeleton is the D-05/D-06 proportion and structure
// contract for plan.md: every non-exempt stage carries a Purpose line, the
// wrapper carries the D-06 structured blocks and cross-stage state manifest,
// termination conditions are documented in prose, and stage-skeleton method
// outweighs envelope-parsing mechanics by at least 3x.
//
// Note: ordered-heading parity between .claude and .opencode plan.md is
// already owned by TestPlanWrapperCardsParity's ordered_heading_sets_are_identical
// subtest in cmd/plan_wrapper_cards_test.go — do not add a second
// ordered-heading-parity subtest here.
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
		required := []string{
			"<success_criteria>",
			"<failure_modes>",
			"<read_only>",
			"## Required Cross-Stage State",
		}
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, want := range required {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing structured block %q", path, want)
				}
			}
		}
	})

	t.Run("termination_conditions_documented", func(t *testing.T) {
		// These four literal substrings name the loop's stop-condition
		// concepts (target confidence, stall detection, iteration cap,
		// escape hatch) without asserting a specific numeric threshold —
		// the wrapper describes what the runtime enforces, it does not
		// carry the threshold arithmetic itself.
		concepts := []string{
			"target confidence",
			"stall",
			"max iteration",
			"escape hatch",
		}
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, concept := range concepts {
				if !strings.Contains(text, concept) {
					t.Errorf("%s missing termination-condition concept %q", path, concept)
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
