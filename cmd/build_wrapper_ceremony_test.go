package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"),
	}

	required := []string{
		"Use the Go `aether` CLI as the source of truth.",
		"AETHER_OUTPUT_MODE=visual aether status",
		"## Active Signals",
		"REDIRECT",
		"FOCUS",
		"FEEDBACK",
		"strength or remaining-life context",
		"## Phase Framing",
		"Phase N of M -- Name",
		"## Dispatch Manifest",
		"aether build $ARGUMENTS --plan-only",
		"temporary manifest file outside `.aether/data/`",
		"result.dispatch_manifest",
		"## Runtime Spawn Ceremony",
		"AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>",
		"Do not set `run_in_background`",
		"Do NOT run `aether host build` from this wrapper",
		"Do NOT run `aether build --synthetic` after real",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
		"/ant-continue",
		"dispatch_manifest.context_capsule",
		"brief_path",
		"## Required Cross-Stage State",
		"**Purpose:**",
		"**Reads:**",
		"**Stop conditions:**",
		"<success_criteria>",
		"<failure_modes>",
		"<read_only>",
		"You DIRECTLY spawn multiple workers",
		"Why this matters",
		"## Blocker Heads-Up",
		"result.blocker_advisory",
		"result.blocker_advisory_question",
		"Carry on with the build, or stop and deal with this first?",
		"a forced reviewer still waiting on the owner's check-in decision",
		"the last check-and-advance (`aether continue`) on this phase ending blocked",
	}

	inOrder := []string{
		"## Ownership Split",
		"## Colony Context",
		"## Active Signals",
		"## Phase Framing",
		"## Dispatch Manifest",
		"aether build $ARGUMENTS --plan-only",
		"## Runtime Spawn Ceremony",
		"## Blocker Heads-Up",
		"## Team Check-In",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
		"## After the Build",
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
			"Do NOT load playbooks",
			"aether host build --dry-run",
			"result.manifest.dispatch_manifest",
			"Do NOT run direct `aether build` from this wrapper for manifest generation",
			// D-10 regression-fence item 9: no wrapper-driven git stash or
			// commit. Checkpointing, if it ever returns, is runtime work.
			// The mining source for this rewrite (build-prep.md:292-300)
			// carries a `git stash push` checkpoint and `git stash pop`
			// rollback verbatim next to a colony beat, so this is the one
			// D-10 item most exposed by an executor mining that text.
			"git stash",
			"git add -A",
			"git commit",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old pass-through contract %q", wrapperPath, forbidden)
			}
		}
	}
}

// briefPathContractSurfaces is the exact four-surface list named by
// .aether/commands/build.yaml:16's drift_guard ("Update this YAML,
// Claude/OpenCode wrappers, the Codex skill, and cmd/command_guide.go
// together."). A change touching fewer than all four breaks that guard's own
// promise, so TestBriefPathReferencedAcrossAllFourSurfaces asserts len == 4
// to catch a future edit that quietly drops a surface.
var briefPathContractSurfaces = []string{
	".claude/commands/ant/build.md",
	".opencode/commands/ant/build.md",
	".aether/skills/colony/aether-colony-build-cycle/SKILL.md",
	"cmd/command_guide.go",
}

// TestBriefPathReferencedAcrossAllFourSurfaces proves the D-12 brief_path
// contract is sanctioned on every one of the four drift-guarded surfaces:
// each must mention both `brief_path` and (case-insensitively) "verbatim".
// It fails if brief_path is deleted from any one of the four files.
func TestBriefPathReferencedAcrossAllFourSurfaces(t *testing.T) {
	if len(briefPathContractSurfaces) != 4 {
		t.Fatalf("briefPathContractSurfaces has %d entries, want exactly 4 (per .aether/commands/build.yaml:16 drift_guard)", len(briefPathContractSurfaces))
	}

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range briefPathContractSurfaces {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		if !strings.Contains(text, "brief_path") {
			t.Errorf("%s missing brief_path", path)
		}
		if !strings.Contains(strings.ToLower(text), "verbatim") {
			t.Errorf("%s missing a case-insensitive mention of verbatim", path)
		}
	}
}

// TestBuildWrapperVerbatimBriefBulletsStayMirrored proves the .claude and
// .opencode build.md wrappers' VERBATIM-brief bullet (worker spawning
// section) and prompt-recipe bullet (worker prompt composition) remain
// textually identical -- catching drift where one wrapper gains brief_path
// wording the other loses.
func TestBuildWrapperVerbatimBriefBulletsStayMirrored(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeContent, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"))
	if err != nil {
		t.Fatalf("read claude build.md: %v", err)
	}
	opencodeContent, err := os.ReadFile(filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"))
	if err != nil {
		t.Fatalf("read opencode build.md: %v", err)
	}

	claudeText := string(claudeContent)
	opencodeText := string(opencodeContent)

	claudeBriefBullet := mustExtractLineContaining(t, claudeText, "Each dispatch carries `brief`")
	opencodeBriefBullet := mustExtractLineContaining(t, opencodeText, "Each dispatch carries `brief`")
	if claudeBriefBullet != opencodeBriefBullet {
		t.Errorf("VERBATIM-brief bullet drifted between wrappers:\nclaude:   %s\nopencode: %s", claudeBriefBullet, opencodeBriefBullet)
	}

	claudeRecipeBullet := mustExtractLineContaining(t, claudeText, "The worker's prompt =")
	opencodeRecipeBullet := mustExtractLineContaining(t, opencodeText, "The worker's prompt =")
	if claudeRecipeBullet != opencodeRecipeBullet {
		t.Errorf("prompt-recipe bullet drifted between wrappers:\nclaude:   %s\nopencode: %s", claudeRecipeBullet, opencodeRecipeBullet)
	}
}

func mustExtractLineContaining(t *testing.T, content, marker string) string {
	t.Helper()
	line := extractLineContaining(content, marker)
	if line == "" {
		t.Fatalf("no line containing %q found", marker)
	}
	return line
}

func assertSubstringsInOrder(t *testing.T, path, content string, ordered []string) {
	t.Helper()

	cursor := 0
	for _, needle := range ordered {
		idx := strings.Index(content[cursor:], needle)
		if idx < 0 {
			t.Fatalf("%s missing ordered marker %q", path, needle)
		}
		cursor += idx + len(needle)
	}
}

// TestBuildMdOwnershipHandshake pins the D-08/CMD-05 ownership chain record:
// Phase 160 (Fail Loudly) merged first and made only narrow call-argument
// fixes; Phase 165 is the sole structural owner of build.md for milestone
// v1.25; Phase 168 (Live Visibility) appends a visual-guidance trailer
// afterward. Both canonical build.md files must carry an HTML comment
// recording the PHASE-160 fix and the exact reserved PHASE-168 trailer
// marker, in that order, with the PHASE-168 marker as the last non-empty
// line of the file -- an unambiguous insertion point for Phase 168.
func TestBuildMdOwnershipHandshake(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	const phase168Marker = "<!-- PHASE-168: visual-guidance trailer appends below this line -->"

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"),
	}

	for _, wrapperPath := range wrapperPaths {
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}
		text := string(content)

		if !strings.Contains(text, "PHASE-160:") {
			t.Errorf("%s missing a PHASE-160: ownership comment", wrapperPath)
		}
		if !strings.Contains(text, phase168Marker) {
			t.Errorf("%s missing the exact PHASE-168 trailer marker %q", wrapperPath, phase168Marker)
		}

		assertSubstringsInOrder(t, wrapperPath, text, []string{"PHASE-160:", phase168Marker})

		lines := strings.Split(text, "\n")
		lastNonEmpty := ""
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				lastNonEmpty = strings.TrimSpace(lines[i])
				break
			}
		}
		if lastNonEmpty != phase168Marker {
			t.Errorf("%s: last non-empty line is %q, want exactly %q", wrapperPath, lastNonEmpty, phase168Marker)
		}
	}
}

// TestBuildWrapperStageSkeletonAndParity is the D-05/D-09 stage-skeleton
// invariant for build.md: every non-exempt stage carries a Purpose marker
// (a proportion assertion that survives a stage rename, unlike a
// named-section grep), .claude and .opencode stay in ordered heading parity,
// the D-06 method-asset blocks are present, and method prose outweighs
// envelope-parsing mechanics per the CMD-02 relocation to
// wrapper-host-contract.md.
func TestBuildWrapperStageSkeletonAndParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudePath := filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md")
	opencodePath := filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md")

	claudeRaw, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("read %s: %v", claudePath, err)
	}
	opencodeRaw, err := os.ReadFile(opencodePath)
	if err != nil {
		t.Fatalf("read %s: %v", opencodePath, err)
	}

	claudeText := string(claudeRaw)
	opencodeText := string(opencodeRaw)

	exemptHeadings := []string{
		"## Ownership Split",
		"## Required Cross-Stage State",
		"## Cross-Platform Drift Guard",
		"## Guardrails",
	}

	t.Run("stage_skeleton_density", func(t *testing.T) {
		assertStageSkeletonDensity(t, claudePath, claudeText, exemptHeadings)
		assertStageSkeletonDensity(t, opencodePath, opencodeText, exemptHeadings)
	})

	t.Run("ordered_heading_parity", func(t *testing.T) {
		assertOrderedHeadingParity(t, "build", stripCommentLines(claudeText), stripCommentLines(opencodeText))
	})

	t.Run("structured_blocks_present", func(t *testing.T) {
		for _, path := range []string{claudePath, opencodePath} {
			text := claudeText
			if path == opencodePath {
				text = opencodeText
			}
			for _, marker := range []string{"<success_criteria>", "<failure_modes>", "<read_only>", "## Required Cross-Stage State"} {
				if !strings.Contains(text, marker) {
					t.Errorf("%s missing %q", path, marker)
				}
			}
		}
	})

	t.Run("method_outweighs_envelope_mechanics", func(t *testing.T) {
		for _, path := range []string{claudePath, opencodePath} {
			text := claudeText
			if path == opencodePath {
				text = opencodeText
			}
			methodCount := countMarkerLines(text, stageSkeletonMarkers())
			envelopeCount := countMarkerLines(text, envelopeMechanicsMarkers())
			if methodCount < 3*envelopeCount {
				t.Errorf("%s: method markers (%d) do not outweigh envelope-mechanics markers (%d) by at least 3x", path, methodCount, envelopeCount)
			}
		}
	})
}
