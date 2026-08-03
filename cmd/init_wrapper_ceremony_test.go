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
		"--promote-shelf",
		"promoted_shelf_ids",
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
		// Phase 165 gap CR-01: init.md's Shelf Backlog stage instructed a
		// hand-append of a shelf-derived entry into protected runtime state —
		// the same D-2 Frankenstein-state class the entries above fence, just
		// against a different field the original list happened not to match.
		// The field named below is runtime-owned; the wrapper has no reason
		// to name it, or to name the write target it used to describe.
		"append `[shelf:",
		"to `active_todos`",
		"active_todos",
		"in the session file or colony state",
		// Phase 165 gap CR-01's second instance (review CR-01 / plan 165-10):
		// promoting or dismissing a shelf entry via the standalone batch
		// commands is an ORDERING hazard, not a vocabulary one -- both mutate
		// shelf.json the instant they run, while the wrapper's own
		// <failure_modes> promises a cancel or a failed init persists
		// nothing. Promotion must happen only inside the same `aether init`
		// call the user has already consented to, via `--promote-shelf` /
		// `--dismiss-shelf`.
		"aether shelf-promote-batch",
		"aether shelf-dismiss-batch",
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

	t.Run("approval_writes_pheromones_only_after_init_succeeds", func(t *testing.T) {
		// Phase 165 review WR-05: init.md's own <failure_modes> promises a
		// cancel or failure writes nothing, so `aether pheromone-write` must not
		// run before `aether init --colony-mode` has returned success.
		for _, wrapperPath := range wrapperPaths {
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			section := initApprovalSection(string(content))
			if section == "" {
				t.Fatalf("%s: missing \"## Approval\" section", wrapperPath)
			}
			initIdx := strings.Index(section, "aether init --colony-mode")
			pheromoneIdx := strings.Index(section, "aether pheromone-write")
			if initIdx == -1 {
				t.Fatalf("%s: Approval section missing %q", wrapperPath, "aether init --colony-mode")
			}
			if pheromoneIdx == -1 {
				t.Fatalf("%s: Approval section missing %q", wrapperPath, "aether pheromone-write")
			}
			if !(initIdx < pheromoneIdx) {
				t.Errorf("%s: Approval section runs `aether pheromone-write` before `aether init --colony-mode` has returned success — init.md's own <failure_modes> promises a cancel or failure writes nothing, so pheromones must be gated on init success (Phase 165 review WR-05)", wrapperPath)
			}
		}
	})

	t.Run("shelf_ids_are_spent_only_inside_the_approval_init_call", func(t *testing.T) {
		// Phase 165 gap CR-01 (plan 165-10): a shelf write placed before the
		// Approval consent gate contradicts init.md's <failure_modes> promise
		// and strands a promoted entry on cancel, goal revision, or failed
		// init. Checked directly against all three surfaces -- the two
		// canonical wrappers plus the flat installed mirror -- not only
		// transitively through TestLifecycleFlatMirrorsMatchCanonical.
		allPaths := append(append([]string{}, wrapperPaths...), flatMirrorPath(repoRoot, "init"))
		for _, wrapperPath := range allPaths {
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)

			// (a) no occurrence of "aether shelf-" other than "aether shelf-list".
			searchFrom := 0
			for {
				idx := strings.Index(text[searchFrom:], "aether shelf-")
				if idx == -1 {
					break
				}
				absIdx := searchFrom + idx
				rest := text[absIdx+len("aether shelf-"):]
				if !strings.HasPrefix(rest, "list") {
					end := absIdx + len("aether shelf-")
					limit := end + 20
					if limit > len(text) {
						limit = len(text)
					}
					t.Errorf("%s: found %q, a shelf write other than the read-only `aether shelf-list` -- a shelf write placed before the Approval consent gate contradicts init.md's <failure_modes> promise and strands a promoted entry on cancel, goal revision, or failed init (Phase 165 gap CR-01)", wrapperPath, text[absIdx:limit])
				}
				searchFrom = absIdx + len("aether shelf-")
			}

			// (b) --promote-shelf appears only after "## Approval".
			approvalIdx := strings.Index(text, "## Approval")
			if approvalIdx == -1 {
				t.Fatalf("%s: missing \"## Approval\" heading", wrapperPath)
			}
			promoteIdx := strings.Index(text, "--promote-shelf")
			if promoteIdx == -1 {
				t.Errorf("%s: missing %q -- shelf promotion must be spent inside the Approval init call (Phase 165 gap CR-01)", wrapperPath, "--promote-shelf")
			} else if promoteIdx <= approvalIdx {
				t.Errorf("%s: %q appears at index %d, at or before \"## Approval\" at index %d -- shelf IDs must be spent only inside the Approval stage's init call, after the user has consented (Phase 165 gap CR-01)", wrapperPath, "--promote-shelf", promoteIdx, approvalIdx)
			}

			// (c) the "## Shelf Backlog" section contains "aether shelf-list"
			// and no other "aether shelf-" occurrence.
			section := initShelfBacklogSection(text)
			if section == "" {
				t.Fatalf("%s: missing \"## Shelf Backlog\" heading", wrapperPath)
			}
			if !strings.Contains(section, "aether shelf-list") {
				t.Errorf("%s: \"## Shelf Backlog\" section is missing %q", wrapperPath, "aether shelf-list")
			}
			sectionSearchFrom := 0
			for {
				idx := strings.Index(section[sectionSearchFrom:], "aether shelf-")
				if idx == -1 {
					break
				}
				absIdx := sectionSearchFrom + idx
				rest := section[absIdx+len("aether shelf-"):]
				if !strings.HasPrefix(rest, "list") {
					end := absIdx + len("aether shelf-")
					limit := end + 20
					if limit > len(section) {
						limit = len(section)
					}
					t.Errorf("%s: \"## Shelf Backlog\" section contains %q, a shelf write other than the read-only `aether shelf-list` -- the Shelf Backlog stage must only collect IDs, never mutate the shelf (Phase 165 gap CR-01)", wrapperPath, section[absIdx:limit])
				}
				sectionSearchFrom = absIdx + len("aether shelf-")
			}
		}
	})
}

// initShelfBacklogSection returns the substring from the first index of
// "## Shelf Backlog" to the next "\n## " after it (end of file if none),
// modelled on initApprovalSection's slicing above. Fails the calling test via
// t.Fatalf when the heading is absent.
func initShelfBacklogSection(text string) string {
	const heading = "## Shelf Backlog"
	idx := strings.Index(text, heading)
	if idx == -1 {
		return ""
	}
	rest := text[idx+len(heading):]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx == -1 {
		return text[idx:]
	}
	return text[idx : idx+len(heading)+nextIdx]
}

// initApprovalSection returns the substring from the first index of
// "## Approval" to the end of the file, modelled on extractStateCarrySection's
// slicing approach in cmd/lifecycle_wrapper_contract_test.go.
func initApprovalSection(text string) string {
	const heading = "## Approval"
	idx := strings.Index(text, heading)
	if idx == -1 {
		return ""
	}
	return text[idx:]
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
