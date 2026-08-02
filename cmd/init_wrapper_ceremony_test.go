package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestInitWrapperCeremonyContract is init.md's first dedicated ceremony
// contract test. init.md is the richest ceremony source at v5.4.0 (516
// lines, self-contained) and also the origin of the CRITICAL D-2
// Frankenstein-state regression (see 165-ARCHAEOLOGY.md §D-2), so this test
// both requires the restored ceremony markers and fences the forbidden
// mutation vocabulary in one place.
func TestInitWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := canonicalWrapperPaths(repoRoot, "init")

	required := []string{
		"## Required Cross-Stage State",
		"**Purpose:**",
		"**Reads:**",
		"**Stop conditions:**",
		"<success_criteria>",
		"<failure_modes>",
		"<read_only>",
		"Queen has set the colony's intention",
		"aether init-research",
		"aether init --colony-mode",
		"--charter-json",
		"selected_colony_mode",
		"aether pheromone-write",
		"aether shelf-list",
	}

	inOrder := []string{
		"## Codebase Summary",
		"## Prior Context",
		"## Intent Refinement",
		"## Colony Charter",
		"## Colony Mode",
		"## Pheromone Suggestions",
		"## Shelf Backlog",
		"## Cross-Platform Drift Guard",
		"## Approval",
	}

	// forbidden fences the CRITICAL D-2 Frankenstein-state regression
	// (165-ARCHAEOLOGY.md §D-2): v5.4.0 init.md reconstructed
	// COLONY_STATE.json with jq and wrote pheromones/constraints/midden/
	// learning-observations by hand. `aether state-write` is fenced here
	// even though no current test names it, because init.md is the plan
	// most likely to reintroduce that mining pattern.
	forbidden := []string{
		"queen-init",
		"Write COLONY_STATE.json",
		"Read `.aether/data/COLONY_STATE.json`.",
		".aether/aether-utils.sh",
		"aether state-write",
		"colony_depth",
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

		for _, forbid := range forbidden {
			if strings.Contains(text, forbid) {
				t.Errorf("%s still contains forbidden text %q", wrapperPath, forbid)
			}
		}
	}
}

// TestInitWrapperStageSkeletonAndParity covers the D-05 stage-skeleton
// proportion assertion, ordered-heading parity between platforms, the
// CMD-02 zero-envelope-parsing-prose invariant, and the single sanctioned
// platform delta (the AskUserQuestion vs. plain Ask approval wording) that
// makes init.md the one lifecycle wrapper NOT byte-identical across
// platforms.
func TestInitWrapperStageSkeletonAndParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := canonicalWrapperPaths(repoRoot, "init")
	exemptHeadings := []string{"## Required Cross-Stage State", "## Cross-Platform Drift Guard"}

	t.Run("stage_skeleton_density", func(t *testing.T) {
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertStageSkeletonDensity(t, path, string(content), exemptHeadings)
		}
	})

	t.Run("ordered_heading_parity", func(t *testing.T) {
		claudeContent, err := os.ReadFile(wrapperPaths[0])
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPaths[0], err)
		}
		opencodeContent, err := os.ReadFile(wrapperPaths[1])
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPaths[1], err)
		}
		claudeText := stripCommentLines(string(claudeContent))
		opencodeText := stripCommentLines(string(opencodeContent))
		assertOrderedHeadingParity(t, "init", claudeText, opencodeText)
	})

	t.Run("no_envelope_parsing_prose", func(t *testing.T) {
		markers := envelopeMechanicsMarkers()
		for _, path := range wrapperPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if count := countMarkerLines(string(content), markers); count != 0 {
				t.Errorf("%s: expected 0 envelope-parsing marker lines, got %d — init.md has no host-manifest step and needs none", path, count)
			}
		}
	})

	t.Run("platform_difference_is_the_sanctioned_one", func(t *testing.T) {
		claudeContent, err := os.ReadFile(wrapperPaths[0])
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPaths[0], err)
		}
		opencodeContent, err := os.ReadFile(wrapperPaths[1])
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPaths[1], err)
		}

		claudeLines := strings.Split(string(claudeContent), "\n")
		opencodeLines := strings.Split(string(opencodeContent), "\n")

		if len(claudeLines) != len(opencodeLines) {
			t.Fatalf("init.md is the one wrapper permitted a single sanctioned platform delta (the ask-tool wording), but .claude (%d lines) and .opencode (%d lines) have different line counts — any structural delta needs a deliberate decision, not a silent edit", len(claudeLines), len(opencodeLines))
		}

		var diffIdx []int
		for i := range claudeLines {
			if claudeLines[i] != opencodeLines[i] {
				diffIdx = append(diffIdx, i)
			}
		}

		if len(diffIdx) != 1 {
			t.Fatalf("init.md is the one wrapper permitted exactly one sanctioned platform delta (the ask-tool wording), but found %d differing line(s) at index/indices %v — any second delta needs a deliberate decision, not a silent edit", len(diffIdx), diffIdx)
		}

		idx := diffIdx[0]
		if !strings.Contains(claudeLines[idx], "AskUserQuestion") {
			t.Errorf("the one sanctioned differing line in .claude/commands/ant/init.md must contain %q, got: %q", "AskUserQuestion", claudeLines[idx])
		}
		if strings.Contains(opencodeLines[idx], "AskUserQuestion") {
			t.Errorf("the .opencode side of the sanctioned differing line must not contain %q, got: %q", "AskUserQuestion", opencodeLines[idx])
		}
	})
}
