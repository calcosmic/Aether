package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// planWrapperCardRuntimeKeys are the four runtime result keys the two
// decision-moment cards depend on. Both platform wrappers must reference
// all four so neither wrapper composes its own recommendation text.
var planWrapperCardRuntimeKeys = []string{
	"depth_proposal_card",
	"research_proposal_card",
	"research_awaiting_approval",
	"research_warning",
}

// stripCommentLines removes HTML-comment lines (the wrappers open with an
// `<!-- Aether-managed ... -->` banner) so a comment mentioning a token
// cannot satisfy an assertion meant to catch an actual instruction.
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

// level2Headings returns every line beginning with "## ", in order, from a
// wrapper body (comment lines already stripped).
func level2Headings(text string) []string {
	var headings []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	return headings
}

// TestPlanWrapperCardsParity is the ordered-heading-set invariant for
// Plan 09 Task 3: a one-sided wrapper edit (rename, addition, drop) fails
// this test instead of only being caught by named-section-exists checks
// that a rename could slip past.
func TestPlanWrapperCardsParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeRaw, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "commands", "ant", "plan.md"))
	if err != nil {
		t.Fatalf("read Claude plan.md: %v", err)
	}
	opencodeRaw, err := os.ReadFile(filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"))
	if err != nil {
		t.Fatalf("read OpenCode plan.md: %v", err)
	}
	yamlRaw, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "commands", "plan.yaml"))
	if err != nil {
		t.Fatalf("read plan.yaml: %v", err)
	}

	claudeText := stripCommentLines(string(claudeRaw))
	opencodeText := stripCommentLines(string(opencodeRaw))
	yamlText := string(yamlRaw)

	t.Run("ordered_heading_sets_are_identical", func(t *testing.T) {
		claudeHeadings := level2Headings(claudeText)
		opencodeHeadings := level2Headings(opencodeText)
		if !reflect.DeepEqual(claudeHeadings, opencodeHeadings) {
			t.Fatalf("wrapper level-2 heading order differs:\nClaude:   %v\nOpenCode: %v", claudeHeadings, opencodeHeadings)
		}
		if len(claudeHeadings) == 0 {
			t.Fatal("expected at least one level-2 heading in the plan wrappers")
		}
	})

	t.Run("both_wrappers_reference_all_four_runtime_keys", func(t *testing.T) {
		for _, key := range planWrapperCardRuntimeKeys {
			if !strings.Contains(claudeText, key) {
				t.Errorf("Claude wrapper missing runtime key %q", key)
			}
			if !strings.Contains(opencodeText, key) {
				t.Errorf("OpenCode wrapper missing runtime key %q", key)
			}
		}
	})

	t.Run("neither_wrapper_contains_retired_headings", func(t *testing.T) {
		for _, retired := range []string{"## Depth Ceremony", "## Planning Depth"} {
			if strings.Contains(claudeText, retired) {
				t.Errorf("Claude wrapper still contains retired heading %q", retired)
			}
			if strings.Contains(opencodeText, retired) {
				t.Errorf("OpenCode wrapper still contains retired heading %q", retired)
			}
		}
	})

	t.Run("exactly_one_approve_all_and_one_auto_instruction", func(t *testing.T) {
		for name, text := range map[string]string{"Claude": claudeText, "OpenCode": opencodeText} {
			approveAllCount := strings.Count(text, "plan-research-approve --approve-all")
			if approveAllCount != 1 {
				t.Errorf("%s wrapper has %d occurrences of 'plan-research-approve --approve-all', want exactly 1", name, approveAllCount)
			}
			autoCount := strings.Count(text, "plan-research-approve --auto")
			if autoCount != 1 {
				t.Errorf("%s wrapper has %d occurrences of 'plan-research-approve --auto', want exactly 1", name, autoCount)
			}
		}
	})

	t.Run("plan_yaml_carries_both_card_entries", func(t *testing.T) {
		for _, entry := range []string{"depth_proposal_card:", "research_batch_card:"} {
			if !strings.Contains(yamlText, entry) {
				t.Errorf("plan.yaml wrapper_additions missing card entry %q", entry)
			}
		}
	})

	t.Run("a_one_sided_heading_addition_would_fail", func(t *testing.T) {
		claudeHeadings := level2Headings(claudeText)
		opencodeHeadings := level2Headings(opencodeText)
		mutated := append(append([]string{}, opencodeHeadings...), "## Injected Only In OpenCode")
		if reflect.DeepEqual(claudeHeadings, mutated) {
			t.Fatal("expected a one-sided heading addition to break equality, but it did not")
		}
	})
}
