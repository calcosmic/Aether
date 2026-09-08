package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var planWrapperStageRuntimeKeys = []string{
	"result.plan_manifest",
	"result.planning_manifest",
	"stage_manifest",
	"stage_receipt",
	"route_stage_manifest",
	"route_stage_receipt",
	"decision_batch",
	"decision_cards",
	"decision_resume_token",
	"iteration_card",
	"scout_stage_manifest",
	"plan_candidate",
	"acceptance_command",
	"acceptance_receipt",
}

var planWrapperPresetLines = []string{
	"Fast        Target 80   Up to 4 passes",
	"Balanced    Target 90   Up to 6 passes",
	"Deep        Target 95   Up to 8 passes",
	"Exhaustive  Target 99   Up to 12 passes",
}

// stripCommentLines removes HTML-comment lines (the wrappers open with an
// Aether-managed banner) so a comment cannot satisfy a live contract check.
func stripCommentLines(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func level2Headings(text string) []string {
	var headings []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	return headings
}

func TestPlanWrapperCardsParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	paths := map[string]string{
		"Claude canonical": filepath.Join(repoRoot, ".claude", "commands", "ant", "plan.md"),
		"Claude flat":      filepath.Join(repoRoot, ".claude", "commands", "ant-plan.md"),
		"OpenCode":         filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"),
	}
	wrappers := make(map[string]string, len(paths))
	for name, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s plan wrapper: %v", name, err)
		}
		wrappers[name] = stripCommentLines(string(content))
	}
	yamlRaw, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "commands", "plan.yaml"))
	if err != nil {
		t.Fatalf("read plan.yaml: %v", err)
	}
	yamlText := string(yamlRaw)

	t.Run("ordered_heading_sets_and_claude_projections_match", func(t *testing.T) {
		canonical := wrappers["Claude canonical"]
		if canonical != wrappers["Claude flat"] {
			t.Fatal("Claude flat plan projection drifted from its canonical wrapper")
		}
		canonicalHeadings := level2Headings(canonical)
		for name, text := range wrappers {
			if headings := level2Headings(text); !reflect.DeepEqual(canonicalHeadings, headings) {
				t.Errorf("%s level-2 heading order differs:\ncanonical: %v\nactual:    %v", name, canonicalHeadings, headings)
			}
		}
		if len(canonicalHeadings) == 0 {
			t.Fatal("expected planning wrappers to contain staged level-2 headings")
		}
	})

	t.Run("four_exact_unbiased_presets", func(t *testing.T) {
		for name, text := range wrappers {
			for _, line := range planWrapperPresetLines {
				if count := strings.Count(text, line); count != 1 {
					t.Errorf("%s contains preset line %q %d times, want exactly 1", name, line, count)
				}
			}
			for _, want := range []string{
				"No option is preselected, recommended, or silently chosen.",
				"Choose the planning preset: Fast, Balanced, Deep, or Exhaustive.",
				"Planning did not start. State: unchanged.",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing unbiased preset contract %q", name, want)
				}
			}
		}
		for _, line := range planWrapperPresetLines {
			if !strings.Contains(yamlText, line) {
				t.Errorf("plan.yaml missing exact preset line %q", line)
			}
		}
		for _, forbidden := range []string{"defaults to Deep", "default to Deep", "Queen recommends Deep", "smart default"} {
			for name, text := range wrappers {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s silently prefers a preset via %q", name, forbidden)
				}
			}
		}
	})

	t.Run("runtime_authorizes_each_stage_and_card", func(t *testing.T) {
		for name, text := range wrappers {
			for _, key := range planWrapperStageRuntimeKeys {
				if !strings.Contains(text, key) {
					t.Errorf("%s missing structured planning key %q", name, key)
				}
			}
			assertSubstringsInOrder(t, name, text, []string{
				"## Scout Stage",
				"scout_result",
				"stage_receipt",
				"## First-Pass Owner Decision Boundary",
				"route_stage_manifest",
				"## Route-Setter Stage",
				"route_result",
				"route_stage_receipt",
				"## Iteration Card and Timeline",
				"## Continue, Pause, or Stop",
			})
		}
	})

	t.Run("cards_preserve_causality_and_autonomous_research", func(t *testing.T) {
		for name, text := range wrappers {
			for _, want := range []string{
				"Fresh evidence",
				"Knowledge, Requirements, Risks, Dependencies, and Effort",
				"weakest gap",
				"Evidence that would change it",
				"Semantic additions, changes, removals",
				"separate authority impact",
				"Target sufficiency",
				"diminishing returns",
				"stall detected",
				"selected max iteration cap",
				"Routine research proceeds automatically inside the selected preset.",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing iteration-causality contract %q", name, want)
				}
			}
		}
	})

	t.Run("candidate_is_reviewed_before_exact_acceptance", func(t *testing.T) {
		for name, text := range wrappers {
			assertSubstringsInOrder(t, name, text, []string{
				"## Candidate Review",
				"AETHER_OUTPUT_MODE=json aether plan --candidate",
				"[NOT ACTIVE]",
				"complete chronological iteration timeline/digest",
				"This candidate is not active and cannot be built yet.",
				"## Exact Candidate Acceptance",
				"acceptance_command",
				"aether plan --accept-candidate <candidate-id>",
				"acceptance_receipt",
				"/ant-build 1",
				"/ant-run",
			})
			for _, flag := range []string{
				"--spec-revision", "--spec-hash", "--base-plan-revision", "--timeline-digest", "--proposal-hash", "--acceptance-token",
			} {
				if !strings.Contains(text, flag) {
					t.Errorf("%s missing exact acceptance binding %q", name, flag)
				}
			}
		}
	})

	t.Run("retired_prompt_owned_contract_is_absent", func(t *testing.T) {
		retired := []string{
			"Decision Moment",
			"depth_proposal_card",
			"research_proposal_card",
			"research_awaiting_approval",
			"research_warning",
			"plan-research-approve",
			"--planning-depth",
			"--verification-depth",
		}
		all := map[string]string{"plan.yaml": yamlText}
		for name, text := range wrappers {
			all[name] = text
		}
		bareAccept := regexp.MustCompile("(^|[[:space:]`])--accept([[:space:]`]|$)")
		for name, text := range all {
			for _, token := range retired {
				if strings.Contains(text, token) {
					t.Errorf("%s still contains retired planning token %q", name, token)
				}
			}
			if bareAccept.MatchString(text) {
				t.Errorf("%s still contains the retired bare acceptance flag", name)
			}
		}
	})

	t.Run("yaml_owns_current_projection_contract", func(t *testing.T) {
		for _, want := range []string{
			"approved_specification_preflight:",
			"preset_selection:",
			"staged_scout_route_loop:",
			"material_decision_boundaries:",
			"iteration_card_and_timeline:",
			"candidate_review_and_acceptance:",
			"aether host plan --preset <fast|balanced|deep|exhaustive>",
			"aether plan --accept-candidate <candidate-id>",
		} {
			if !strings.Contains(yamlText, want) {
				t.Errorf("plan.yaml missing current projection contract %q", want)
			}
		}
	})
}
